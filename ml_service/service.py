from __future__ import annotations

import asyncio
from concurrent.futures import TimeoutError
from contextlib import asynccontextmanager, suppress
from typing import Optional, List

from fastapi import Depends, FastAPI, HTTPException
from fastapi.encoders import jsonable_encoder

from .config import Settings, settings
from .data_quality import DataQualityGuard
from .dataset_manager import ReferenceDatasetManager
from .data.training_data import load_training_dataset, TrainingDatasetError
from .drift_monitor import DriftMonitor
from .feature_store import FeatureStore
from .metadata import MetadataStore
from .model import NomenclatureClassifier
from .priority_scheduler import PriorityScheduler
from .regional_validation import RegionalValidator
from .monitoring.store import MonitoringStore, get_monitoring_store
from .schemas import (
    DriftReport,
    MetadataEnvelope,
    NomenclatureItem,
    PredictRequest,
    PredictResponse,
    QualityReport,
    TrainRequest,
)


def get_settings() -> Settings:
    return settings


PRIORITY_FIELDS = (
    "full_name",
    "kind",
    "unit",
    "type_hint",
    "okved_code",
    "hs_code",
)


def compute_information_score(batch) -> float:
    total = 0.0
    for item in batch:
        score = 0.0
        if item.name:
            score += min(len(item.name), 120) / 60  # length bonus up to 2 points
        for field in PRIORITY_FIELDS:
            value = getattr(item, field)
            if value:
                score += 1.0
        total += score
    return total


class RequestTracker:
    def __init__(self, store: MonitoringStore, kind: str, priority: float, detail: Optional[dict] = None):
        self.store = store
        self.kind = kind
        self.priority = priority
        self.detail = detail or {}
        self.request_id: Optional[int] = None
        self._completed = False

    def __enter__(self):
        self.request_id = self.store.start_request(self.kind, self.priority, self.detail)
        self.store.update_request(self.request_id, status="running")
        return self

    def update(self, **kwargs) -> None:
        if self.request_id is None:
            return
        self.store.update_request(self.request_id, **kwargs)

    def complete(self, detail: Optional[dict] = None) -> None:
        if self.request_id is None:
            return
        self.store.update_request(
            self.request_id,
            status="completed",
            detail=detail or self.detail,
            progress=100.0,
        )
        self._completed = True

    def __exit__(self, exc_type, exc, tb):
        if self.request_id is None:
            return False
        if exc_type:
            self.store.update_request(
                self.request_id, status="error", error=str(exc), progress=100.0
            )
        elif not self._completed:
            self.complete()
        return False


def build_app() -> FastAPI:
    scheduler = PriorityScheduler(max_workers=settings.priority_workers)
    classifier = NomenclatureClassifier()
    quality_guard = DataQualityGuard()
    feature_store = FeatureStore()
    drift_monitor = DriftMonitor()
    metadata_store = MetadataStore()
    dataset_manager = ReferenceDatasetManager()
    regional_validator = RegionalValidator()
    monitoring_store = get_monitoring_store()

    @asynccontextmanager
    async def lifespan(_: FastAPI):
        classifier.load_latest()
        async def poll_workers():
            try:
                while True:
                    snapshot = scheduler.snapshot()
                    if (
                        snapshot["active_workers"]
                        or snapshot["queued_tasks"]
                        or snapshot["error_tasks"]
                    ):
                        monitoring_store.record_worker_snapshot(
                            active_workers=snapshot["active_workers"],
                            queued_tasks=snapshot["queued_tasks"],
                            completed_tasks=snapshot["completed_tasks"],
                            error_tasks=snapshot["error_tasks"],
                        )
                        monitoring_store.prune_worker_snapshots(days=7)
                    await asyncio.sleep(10)
            except asyncio.CancelledError:
                pass

        poller = asyncio.create_task(poll_workers())
        try:
            yield
        finally:
            poller.cancel()
            with suppress(asyncio.CancelledError):
                await poller
        scheduler.shutdown(wait=True)

    app = FastAPI(
        title="Nomenclature ML Service",
        version="0.1.0",
        description="API gateway for nomenclature classification & governance.",
        lifespan=lifespan,
    )

    @app.get("/health")
    def health(settings: Settings = Depends(get_settings)) -> dict:
        return {
            "status": "ok",
            "model_version": classifier.model_version,
            "artifacts_dir": str(settings.artifacts_root),
        }

    @app.post("/quality", response_model=QualityReport)
    def validate_payload(request: TrainRequest) -> QualityReport:
        with RequestTracker(
            monitoring_store, kind="quality", priority=0.5, detail={"items": len(request.items)}
        ) as tracker:
            items = request.data or request.items
            if not items:
                tracker.update(status="error", error="empty_payload")
                raise HTTPException(
                    status_code=422, detail="Не переданы данные для проверки качества."
                )
            tracker.update(detail={"items": len(items)})
            report = quality_guard.evaluate(items)
            tracker.complete(
                detail={
                    "passed": report.passed,
                    "issues": [issue.dict() for issue in report.issues],
                    "flagged_rows": len(report.unexpected_items),
                }
            )
            return report

    @app.post("/train")
    def train(
        request: TrainRequest,
        settings: Settings = Depends(get_settings),
    ) -> dict:
        with RequestTracker(
            monitoring_store,
            kind="train",
            priority=1.0,
            detail={"version": request.version},
        ) as tracker:
            provided_items: List[NomenclatureItem] = []
            if request.items:
                provided_items.extend(request.items)
            if request.data:
                provided_items.extend(request.data)
            try:
                base_items = load_training_dataset(settings.default_train_dataset)
            except TrainingDatasetError as exc:
                tracker.update(status="error", error="base_dataset_invalid")
                monitoring_store.record_event(
                    level="error",
                    source="train",
                    message=str(exc),
                    payload={"version": request.version},
                )
                raise HTTPException(status_code=422, detail=str(exc))

            combined_items: List[NomenclatureItem] = []
            if base_items:
                combined_items.extend(base_items)
            if provided_items:
                combined_items.extend(provided_items)

            if not combined_items:
                message = (
                    "Нет данных для обучения, перед первым использованием необходимо "
                    "загрузить в блок data датасет для обучения"
                )
                tracker.update(status="error", error="missing_training_data")
                raise HTTPException(status_code=422, detail=message)

            quality = quality_guard.evaluate(combined_items)
            cleaned_items = [NomenclatureItem(**row) for row in (quality.valid_items or [])]
            if not cleaned_items:
                tracker.update(status="error", error="no_valid_records")
                raise HTTPException(
                    status_code=422,
                    detail="Не осталось валидных строк для обучения после контроля качества.",
                )
            tracker.update(
                detail={
                    "quality_passed": quality.passed,
                    "quality_issues": [issue.dict() for issue in quality.issues],
                    "flagged_rows": len(quality.unexpected_items),
                }
            )

            regional_validator.validate_batch(cleaned_items)
            try:
                mixed_frame, mixing_report = dataset_manager.mix_for_training(cleaned_items)
                mixed_items = [
                    NomenclatureItem(**row) for row in mixed_frame.to_dict(orient="records")
                ]
                metrics = classifier.train(mixed_items, version=request.version)
                if provided_items:
                    dataset_manager.append_reference(provided_items)
                feature_meta = feature_store.ingest(
                    mixed_items, version=request.version, comment="training dataset"
                )
                if request.refresh_baseline:
                    drift_monitor.update_baseline(cleaned_items, feature_meta.version)

                metadata_record = metadata_store.add_record(
                    version=classifier.model_version,
                    data_volume=feature_meta.row_count,
                    metrics=metrics,
                )
            except Exception as exc:
                tracker.update(status="error", error=str(exc))
                monitoring_store.record_event(
                    level="error",
                    source="train",
                    message=str(exc),
                    payload={"version": request.version, "items": len(request.items)},
                )
                raise HTTPException(
                    status_code=500, detail=f"Training failed: {exc}"
                ) from exc
            metadata_payload = jsonable_encoder(metadata_record)
            tracker.complete(
                detail={
                    "metrics": metrics,
                    "feature_version": feature_meta.version,
                    "mixing": mixing_report.__dict__,
                    "training_rows": len(cleaned_items),
                    "quality_issues": [issue.dict() for issue in quality.issues],
                    "flagged_rows": len(quality.unexpected_items),
                    "metadata": metadata_payload,
                }
            )

            return {
                "model_version": classifier.model_version,
                "metrics": metrics,
                "feature_version": feature_meta.version,
                "metadata": metadata_payload,
                "mixing": mixing_report.__dict__,
            }

    @app.post("/predict", response_model=PredictResponse)
    def predict(request: PredictRequest) -> PredictResponse:
        priority = compute_information_score(request.items)
        with RequestTracker(
            monitoring_store,
            kind="predict",
            priority=priority,
            detail={"top_k": request.top_k, "explain": request.explain},
        ) as tracker:
            quality = quality_guard.evaluate(request.items)
            clean_items = [NomenclatureItem(**row) for row in (quality.valid_items or [])]
            if not clean_items:
                tracker.update(status="error", error="no_valid_records")
                raise HTTPException(
                    status_code=422,
                    detail="Нет валидных записей для предсказания после контроля качества.",
                )
            tracker.update(
                detail={
                    "quality_issues": [issue.dict() for issue in quality.issues],
                    "flagged_rows": len(quality.unexpected_items),
                }
            )
            regional_validator.validate_batch(clean_items)

            future = scheduler.submit(
                priority,
                classifier.predict,
                clean_items,
                request.top_k,
                request.explain,
            )
            try:
                results = future.result(timeout=settings.predict_timeout_seconds)
            except TimeoutError as exc:
                tracker.update(status="error", error="timeout")
                raise HTTPException(
                    status_code=504, detail="Prediction timed out in priority queue."
                ) from exc
            except Exception as exc:  # pragma: no cover - bubbled to HTTP layer
                tracker.update(status="error", error=str(exc))
                monitoring_store.record_event(
                    level="error",
                    source="predict",
                    message=str(exc),
                    payload={"items": len(request.items)},
                )
                raise HTTPException(status_code=500, detail=str(exc)) from exc

            version = classifier.model_version or classifier.load_latest()
            tracker.complete(
                detail={
                    "model_version": version,
                    "flagged_rows": len(quality.unexpected_items),
                }
            )
            return PredictResponse(
                results=results,
                model_version=version or "unknown",
                unexpected_items=quality.unexpected_items,
            )

    @app.post("/drift/check", response_model=DriftReport)
    def drift_check(request: TrainRequest) -> DriftReport:
        with RequestTracker(
            monitoring_store,
            kind="drift",
            priority=0.6,
            detail={"items": len(request.items)},
        ) as tracker:
            if not drift_monitor.has_baseline():
                tracker.update(status="error", error="missing_baseline")
                raise HTTPException(
                    status_code=412, detail="Baseline missing. Train first."
                )
            report = drift_monitor.build_report(request.items)
            tracker.complete(detail={"psi": report.population_stability_index, "triggered": report.triggered})
            return report

    @app.post("/drift/baseline")
    def refresh_baseline(request: TrainRequest) -> dict:
        with RequestTracker(
            monitoring_store,
            kind="baseline_refresh",
            priority=0.4,
            detail={"items": len(request.items)},
        ) as tracker:
            meta = drift_monitor.update_baseline(
                request.items, version=request.version or "baseline"
            )
            tracker.complete(detail={"baseline_version": meta.version, "rows": meta.row_count})
            return {"baseline_version": meta.version, "rows": meta.row_count}

    @app.get("/features")
    def feature_versions() -> list[dict]:
        return [meta.to_dict() for meta in feature_store.list_versions()]

    @app.get("/metadata", response_model=MetadataEnvelope)
    def get_metadata() -> MetadataEnvelope:
        return metadata_store.get_envelope()

    return app


app = build_app()


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(
        "ml_service.service:app",
        host=settings.api_host,
        port=settings.api_port,
        reload=False,
        log_level="info",
    )

