from __future__ import annotations

import asyncio
from concurrent.futures import TimeoutError
from contextlib import asynccontextmanager, suppress
from datetime import datetime
from typing import Any, Dict, List, Optional

from fastapi import Depends, FastAPI, HTTPException, Request
from fastapi.encoders import jsonable_encoder

from .repository import DatasetVersionExists, repository
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
    ModelActivationRequest,
    NomenclatureItem,
    NormalizationRequest,
    NormalizationResponse,
    PredictRequest,
    PredictResponse,
    QualityReport,
    TrainRequest,
)
from .text_processing import text_normalizer


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


def trim_payload(payload: Any, max_items: int = 5) -> Any:
    encoded = jsonable_encoder(payload)
    if isinstance(encoded, dict):
        trimmed = dict(encoded)
        for key in ("items", "data"):
            value = trimmed.get(key)
            if isinstance(value, list) and len(value) > max_items:
                trimmed[key] = value[:max_items]
        return trimmed
    if isinstance(encoded, list) and len(encoded) > max_items:
        return encoded[:max_items]
    return encoded


def summarize_payload(payload: Any) -> Dict[str, Any]:
    encoded = jsonable_encoder(payload)
    if not isinstance(encoded, dict):
        return {"type": type(encoded).__name__}
    summary: Dict[str, Any] = {}
    for key, value in encoded.items():
        if isinstance(value, list):
            summary[key] = {"type": "array", "count": len(value)}
        elif isinstance(value, dict):
            summary[key] = {"type": "object", "keys": list(value.keys())[:5]}
        else:
            summary[key] = {"type": type(value).__name__}
    return summary


def build_audit_meta(
    http_request: Request,
    payload_summary: Dict[str, Any],
    *,
    extra_meta: Optional[Dict[str, Any]] = None,
) -> Dict[str, Any]:
    envelope = {
        "client_ip": http_request.client.host if http_request.client else "unknown",
        "user_agent": http_request.headers.get("user-agent"),
        "meta": payload_summary.copy(),
    }
    if extra_meta:
        envelope["meta"].update(extra_meta)
    return envelope


class RequestTracker:
    def __init__(
        self,
        store: MonitoringStore,
        *,
        kind: str,
        priority: float,
        detail: Optional[dict] = None,
        repo=repository,
        audit_payload: Any | None = None,
        audit_meta: Optional[dict] = None,
    ):
        self.store = store
        self.kind = kind
        self.priority = priority
        self.detail = detail or {}
        self.request_id: Optional[int] = None
        self._completed = False
        self.repo = repo
        self.audit_payload = audit_payload
        self.audit_meta = audit_meta or {}
        self.audit_log_id: Optional[int] = None

    def __enter__(self):
        self.request_id = self.store.start_request(self.kind, self.priority, self.detail)
        self.store.update_request(self.request_id, status="running")
        if self.repo and self.audit_payload is not None:
            self.audit_log_id = self.repo.log_request_start(
                kind=self.kind,
                model_version=self.audit_meta.get("model_version", "n/a"),
                payload=self.audit_payload,
                client_ip=self.audit_meta.get("client_ip"),
                user_agent=self.audit_meta.get("user_agent"),
                meta=self.audit_meta.get("meta"),
            )
        return self

    def update(self, **kwargs) -> None:
        if self.request_id is None:
            return
        self.store.update_request(self.request_id, **kwargs)

    def bind_model_version(self, version: str) -> None:
        if self.audit_log_id and self.repo:
            self.repo.attach_model_version_to_log(self.audit_log_id, version)

    def complete(
        self,
        detail: Optional[dict] = None,
        *,
        response_payload: Any | None = None,
        workers_allocated: int = 1,
    ) -> None:
        if self.request_id is None:
            return
        self.store.update_request(
            self.request_id,
            status="completed",
            detail=detail or self.detail,
            progress=100.0,
        )
        if self.repo and self.audit_log_id:
            self.repo.finalize_request_log(
                self.audit_log_id,
                status="success",
                response_payload=response_payload or detail,
                workers_allocated=workers_allocated,
            )
        self._completed = True

    def __exit__(self, exc_type, exc, tb):
        if self.request_id is None:
            return False
        if exc_type:
            self.store.update_request(
                self.request_id, status="error", error=str(exc), progress=100.0
            )
            if self.repo and self.audit_log_id:
                self.repo.finalize_request_log(
                    self.audit_log_id,
                    status="error",
                    error_message=str(exc),
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
            last_snapshot: Optional[dict] = None
            try:
                while True:
                    snapshot = scheduler.snapshot()
                    changed = (
                        last_snapshot is None
                        or any(
                            snapshot[key] != last_snapshot.get(key)
                            for key in snapshot.keys()
                        )
                    )
                    if changed or snapshot["error_tasks"]:
                        monitoring_store.record_worker_snapshot(
                            active_workers=snapshot["active_workers"],
                            queued_tasks=snapshot["queued_tasks"],
                            completed_tasks=snapshot["completed_tasks"],
                            error_tasks=snapshot["error_tasks"],
                        )
                        monitoring_store.prune_worker_snapshots(days=7)
                        last_snapshot = snapshot
                    await asyncio.sleep(15)
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
    def validate_payload(payload: TrainRequest, http_request: Request) -> QualityReport:
        audit_payload = trim_payload(payload.dict())
        audit_meta = build_audit_meta(
            http_request,
            summarize_payload(payload.dict()),
            extra_meta={"model_key": payload.model_key},
        )
        with RequestTracker(
            monitoring_store,
            kind="quality",
            priority=0.5,
            detail={"items": len(payload.items)},
            audit_payload=audit_payload,
            audit_meta=audit_meta,
        ) as tracker:
            items = payload.data or payload.items
            if not items:
                tracker.update(status="error", error="empty_payload")
                raise HTTPException(
                    status_code=422, detail="Не переданы данные для проверки качества."
                )
            tracker.update(detail={"items": len(items)})
            report = quality_guard.evaluate(items)
            completion_detail = {
                "passed": report.passed,
                "issues": [issue.dict() for issue in report.issues],
                "flagged_rows": len(report.unexpected_items),
            }
            tracker.complete(detail=completion_detail, response_payload=report.dict())
            return report

    @app.post("/train")
    def train(
        payload: TrainRequest,
        http_request: Request,
        settings: Settings = Depends(get_settings),
    ) -> dict:
        if payload.task_type != "classification":
            raise HTTPException(
                status_code=400,
                detail="Поддерживается только обучение классификатора номенклатуры.",
            )
        audit_payload = trim_payload(payload.dict())
        audit_meta = build_audit_meta(
            http_request,
            summarize_payload(payload.dict()),
            extra_meta={"model_key": payload.model_key},
        )
        with RequestTracker(
            monitoring_store,
            kind="train",
            priority=1.0,
            detail={"version": payload.version},
            audit_payload=audit_payload,
            audit_meta=audit_meta,
        ) as tracker:
            provided_items: List[NomenclatureItem] = []
            if payload.items:
                provided_items.extend(payload.items)
            if payload.data:
                provided_items.extend(payload.data)
            try:
                base_items = load_training_dataset(settings.default_train_dataset)
            except TrainingDatasetError as exc:
                tracker.update(status="error", error="base_dataset_invalid")
                monitoring_store.record_event(
                    level="error",
                    source="train",
                    message=str(exc),
                    payload={"version": payload.version},
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
            dataset_version = payload.version or datetime.utcnow().strftime("ds_%Y%m%d%H%M%S")
            dataset_name = payload.dataset_name or payload.model_key
            try:
                dataset_record = repository.persist_dataset(
                    model_key=payload.model_key,
                    task_type=payload.task_type,
                    version_label=dataset_version,
                    dataset_name=dataset_name,
                    items=cleaned_items,
                    rewrite=payload.rewrite_dataset,
                )
            except DatasetVersionExists:
                tracker.update(status="error", error="dataset_version_exists")
                raise HTTPException(
                    status_code=409,
                    detail="Версия датасета уже существует. Используйте rewrite_dataset=true для перезаписи.",
                )

            try:
                mixed_frame, mixing_report = dataset_manager.mix_for_training(cleaned_items)
                mixed_items = [
                    NomenclatureItem(**row) for row in mixed_frame.to_dict(orient="records")
                ]
                model_version = f"{dataset_version}-model-{datetime.utcnow().strftime('%H%M%S')}"
                metrics = classifier.train(
                    mixed_items,
                    version=model_version,
                    target_field=payload.target_field or "label",
                    feature_fields=payload.feature_fields,
                )
                tracker.bind_model_version(model_version)
                if provided_items:
                    dataset_manager.append_reference(provided_items)
                feature_meta = feature_store.ingest(
                    mixed_items, version=model_version, comment=f"dataset {dataset_version}"
                )
                if payload.refresh_baseline:
                    drift_monitor.update_baseline(cleaned_items, feature_meta.version)

                metadata_record = metadata_store.add_record(
                    version=classifier.model_version,
                    data_volume=feature_meta.row_count,
                    metrics=metrics,
                    notes=f"dataset={dataset_version}",
                )
            except Exception as exc:
                tracker.update(status="error", error=str(exc))
                monitoring_store.record_event(
                    level="error",
                    source="train",
                    message=str(exc),
                    payload={"version": payload.version, "items": len(payload.items)},
                )
                raise HTTPException(
                    status_code=500, detail=f"Training failed: {exc}"
                ) from exc

            artifact_path = str(settings.model_registry_dir / f"{classifier.model_version}.joblib")
            should_persist = (
                metrics.get("accuracy", 0) >= 0.9 and metrics.get("avg_confidence", 0) >= 0.75
            )
            model_record_id = None
            if should_persist:
                model_record_id = repository.save_model_version(
                    version=classifier.model_version,
                    model_key=payload.model_key,
                    task_type=payload.task_type,
                    dataset_id=dataset_record.dataset_id,
                    dataset_version=dataset_record.version_id,
                    metrics=metrics,
                    artifact_path=artifact_path,
                    activate=True,
                )

            metadata_payload = jsonable_encoder(metadata_record)
            response_payload = {
                "model_version": classifier.model_version,
                "metrics": metrics,
                "feature_version": feature_meta.version,
                "metadata": metadata_payload,
                "mixing": mixing_report.__dict__,
                "dataset_version": dataset_version,
                "dataset_rows": dataset_record.row_count,
                "model_record_id": model_record_id,
                "model_persisted": bool(model_record_id),
            }
            tracker.complete(
                detail={
                    "metrics": metrics,
                    "feature_version": feature_meta.version,
                    "mixing": mixing_report.__dict__,
                    "training_rows": len(cleaned_items),
                    "quality_issues": [issue.dict() for issue in quality.issues],
                    "flagged_rows": len(quality.unexpected_items),
                    "metadata": metadata_payload,
                    "dataset_version": dataset_version,
                },
                response_payload=response_payload,
            )

            return response_payload

    @app.post("/predict", response_model=PredictResponse)
    def predict(payload: PredictRequest, http_request: Request) -> PredictResponse:
        priority = compute_information_score(payload.items)
        audit_payload = trim_payload(payload.dict())
        audit_meta = build_audit_meta(
            http_request,
            summarize_payload(payload.dict()),
            extra_meta={"model_key": payload.model_key},
        )
        with RequestTracker(
            monitoring_store,
            kind="predict",
            priority=priority,
            detail={
                "top_k": payload.top_k,
                "explain": payload.explain,
                "model_key": payload.model_key,
            },
            audit_payload=audit_payload,
            audit_meta=audit_meta,
        ) as tracker:
            if payload.model_key != "nomenclature_classifier":
                tracker.update(status="error", error="unsupported_model")
                raise HTTPException(
                    status_code=400,
                    detail=f"Модель {payload.model_key} недоступна для эндпоинта /predict.",
                )
            quality = quality_guard.evaluate(payload.items)
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
                payload.top_k,
                payload.explain,
            )
            try:
                results = future.result(timeout=settings.predict_timeout_seconds)
            except TimeoutError as exc:
                tracker.update(status="error", error="timeout")
                raise HTTPException(
                    status_code=504, detail="Prediction timed out in priority queue."
                ) from exc
            except Exception as exc:  # pragma: no cover
                tracker.update(status="error", error=str(exc))
                monitoring_store.record_event(
                    level="error",
                    source="predict",
                    message=str(exc),
                    payload={"items": len(payload.items)},
                )
                raise HTTPException(status_code=500, detail=str(exc)) from exc

            version = classifier.model_version or classifier.load_latest()
            if version:
                tracker.bind_model_version(version)
            response = PredictResponse(
                results=results,
                model_version=version or "unknown",
                unexpected_items=quality.unexpected_items,
            )
            tracker.complete(
                detail={
                    "model_version": version,
                    "flagged_rows": len(quality.unexpected_items),
                },
                response_payload=response.dict(),
            )
            return response

    @app.post("/normalize", response_model=NormalizationResponse)
    def normalize(payload: NormalizationRequest, http_request: Request) -> NormalizationResponse:
        audit_payload = trim_payload(payload.dict())
        audit_meta = build_audit_meta(
            http_request,
            summarize_payload(payload.dict()),
            extra_meta={"model_key": "text_normalizer"},
        )
        with RequestTracker(
            monitoring_store,
            kind="normalize",
            priority=0.3,
            detail={"items": len(payload.items), "mode": payload.transliterate_to},
            audit_payload=audit_payload,
            audit_meta=audit_meta,
        ) as tracker:
            response = text_normalizer.normalize(payload)
            tracker.bind_model_version("text_normalizer_v1")
            tracker.complete(
                detail={"normalized": len(response.results)},
                response_payload=response.dict(),
            )
            return response

    @app.post("/drift/check", response_model=DriftReport)
    def drift_check(payload: TrainRequest, http_request: Request) -> DriftReport:
        audit_payload = trim_payload(payload.dict())
        audit_meta = build_audit_meta(
            http_request,
            summarize_payload(payload.dict()),
            extra_meta={"model_key": payload.model_key},
        )
        with RequestTracker(
            monitoring_store,
            kind="drift",
            priority=0.6,
            detail={"items": len(payload.items), "model_key": payload.model_key},
            audit_payload=audit_payload,
            audit_meta=audit_meta,
        ) as tracker:
            if not drift_monitor.has_baseline():
                tracker.update(status="error", error="missing_baseline")
                raise HTTPException(
                    status_code=412, detail="Baseline missing. Train first."
                )
            report = drift_monitor.build_report(payload.items)
            tracker.complete(
                detail={
                    "psi": report.population_stability_index,
                    "triggered": report.triggered,
                },
                response_payload=report.dict(),
            )
            return report

    @app.post("/drift/baseline")
    def refresh_baseline(payload: TrainRequest, http_request: Request) -> dict:
        audit_payload = trim_payload(payload.dict())
        audit_meta = build_audit_meta(
            http_request,
            summarize_payload(payload.dict()),
            extra_meta={"model_key": payload.model_key},
        )
        with RequestTracker(
            monitoring_store,
            kind="baseline_refresh",
            priority=0.4,
            detail={"items": len(payload.items), "model_key": payload.model_key},
            audit_payload=audit_payload,
            audit_meta=audit_meta,
        ) as tracker:
            meta = drift_monitor.update_baseline(
                payload.items, version=payload.version or "baseline"
            )
            result = {"baseline_version": meta.version, "rows": meta.row_count}
            tracker.complete(
                detail={"baseline_version": meta.version, "rows": meta.row_count},
                response_payload=result,
            )
            return {"baseline_version": meta.version, "rows": meta.row_count}

    @app.post("/models/{model_key}/activate")
    def activate_model(model_key: str, body: ModelActivationRequest) -> dict:
        target_key = body.model_key or model_key
        repository.activate_model(version=body.version, model_key=target_key)
        monitoring_store.record_event(
            level="info",
            source="models",
            message="Активирована модель",
            payload={"version": body.version, "model_key": target_key},
        )
        return {"status": "activated", "model_key": target_key, "version": body.version}

    @app.get("/models/{model_key}/history")
    def model_history(model_key: str, limit: int = 50) -> dict:
        records = repository.list_models(model_key=model_key, limit=limit)
        return {"model_key": model_key, "items": records}

    @app.get("/models/{model_key}/responses")
    def model_responses(model_key: str, limit: int = 50) -> dict:
        records = repository.recent_responses(limit=limit, model_key=model_key)
        return {"model_key": model_key, "items": records}

    @app.get("/models/{model_key}/dataset/latest")
    def model_dataset_info(model_key: str):
        info = repository.latest_dataset_info(model_key=model_key)
        if not info:
            raise HTTPException(status_code=404, detail="Датасет для модели не найден.")
        return info

    @app.get("/models/{model_key}/features/active")
    def model_feature_overview(model_key: str) -> dict:
        return {
            "model_key": model_key,
            "features": feature_store.builder.describe_features(),
        }

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

