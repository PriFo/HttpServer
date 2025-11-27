from __future__ import annotations

import asyncio
from datetime import datetime, timedelta
from pathlib import Path
from typing import Optional
import io
import secrets
import json

import httpx
import pandas as pd
import plotly.express as px
from plotly.utils import PlotlyJSONEncoder
from fastapi import (
    Depends,
    FastAPI,
    File,
    Form,
    HTTPException,
    Request,
    UploadFile,
    WebSocket,
    WebSocketDisconnect,
)
from fastapi.encoders import jsonable_encoder
from fastapi.responses import HTMLResponse, PlainTextResponse, Response, RedirectResponse
from fastapi.staticfiles import StaticFiles
from fastapi.templating import Jinja2Templates
from fastapi.security import HTTPBasic, HTTPBasicCredentials
from pydantic import ValidationError
from starlette.middleware.base import BaseHTTPMiddleware
from starlette.middleware.sessions import SessionMiddleware

from ..config import settings
from ..metadata import MetadataStore
from ..feature_store import FeatureBuilder
from ..repository import DatasetVersionExists, repository
from ..schemas import NomenclatureItem
from .store import MonitoringStore, get_monitoring_store
security = HTTPBasic()
def require_admin(credentials: HTTPBasicCredentials = Depends(security)):
    valid_user = secrets.compare_digest(credentials.username, settings.monitor_admin_user)
    valid_pwd = secrets.compare_digest(credentials.password, settings.monitor_admin_password)
    if not (valid_user and valid_pwd):
        raise HTTPException(status_code=401, detail="Unauthorized", headers={"WWW-Authenticate": "Basic"})
    return credentials

templates = Jinja2Templates(directory=str(Path(__file__).resolve().parent.parent / "templates"))


class RateLimitMiddleware(BaseHTTPMiddleware):
    def __init__(self, app: FastAPI, max_requests: int = 60):
        super().__init__(app)
        self.max_requests = max_requests
        self.hits: dict[str, list[datetime]] = {}

    async def dispatch(self, request, call_next):
        client_ip = request.client.host
        now = datetime.utcnow()
        self.hits.setdefault(client_ip, [])
        window = [ts for ts in self.hits[client_ip] if (now - ts).seconds < 60]
        window.append(now)
        self.hits[client_ip] = window
        if len(window) > self.max_requests:
            return PlainTextResponse("Rate limit exceeded", status_code=429)
        return await call_next(request)


app = FastAPI(
    title="ML Monitoring Dashboard",
    docs_url=None,
    redoc_url=None,
)
app.add_middleware(SessionMiddleware, secret_key="ml-monitor-secret")
app.add_middleware(RateLimitMiddleware, max_requests=120)

static_dir = Path(__file__).resolve().parent.parent / "static"
if static_dir.exists():
    app.mount("/static", StaticFiles(directory=str(static_dir)), name="static")


def get_metadata_store() -> MetadataStore:
    return MetadataStore()


def render_template(name: str, request: Request, **context):
    flash = request.session.pop("flash", None)
    ctx = {"request": request, "now": datetime.utcnow(), "flash": flash, **context}
    return templates.TemplateResponse(name, ctx)


def _safe_figure_dict(fig):
    return json.loads(json.dumps(fig, cls=PlotlyJSONEncoder))


@app.get("/", response_class=HTMLResponse)
async def overview(request: Request, store: MonitoringStore = Depends(get_monitoring_store)):
    snapshot = store.latest_worker_snapshot()
    stats = store.request_metrics()
    chart = store.requests_timeseries()
    recent_events = store.recent_events(limit=12)
    recent_requests = repository.recent_responses(limit=8)
    active_models = repository.list_models(active_only=True, limit=6)
    dataset_info = repository.latest_dataset_info(model_key="nomenclature_classifier")
    return render_template(
        "overview.html",
        request,
        snapshot=snapshot,
        stats=stats,
        chart=chart,
        recent_events=recent_events,
        recent_requests=recent_requests,
        models=active_models,
        dataset=dataset_info,
    )


@app.get("/workers", response_class=HTMLResponse)
async def workers_dashboard(request: Request, store: MonitoringStore = Depends(get_monitoring_store)):
    snapshot = store.latest_worker_snapshot()
    history = store.worker_history()
    chart = store.requests_timeseries()
    recent_errors = store.recent_events(limit=10)
    usage = store.worker_usage(limit=15)
    queued_requests = store.filter_requests(status="queued", limit=10)
    running_requests = store.filter_requests(status="running", limit=10)
    metrics = store.request_metrics()
    return render_template(
        "workers.html",
        request,
        snapshot=snapshot,
        history=history,
        chart=chart,
        recent_events=recent_errors,
        usage=usage,
        queued_requests=queued_requests,
        running_requests=running_requests,
        metrics=metrics,
    )


@app.get("/db", response_class=HTMLResponse)
async def database_dashboard(
    request: Request,
    store: MonitoringStore = Depends(get_monitoring_store),
    table: Optional[str] = None,
    limit: int = 100,
):
    summary = store.db_summary()
    table_preview = None
    if table:
        table_preview = store.fetch_table_preview(table, limit=limit)
    return render_template(
        "database.html",
        request,
        summary=summary,
        table_preview=table_preview,
        selected_table=table,
        limit=limit,
    )


@app.get("/requests", response_class=HTMLResponse)
async def requests_dashboard(
    request: Request,
    store: MonitoringStore = Depends(get_monitoring_store),
    status: Optional[str] = None,
    kind: Optional[str] = None,
    limit: int = 200,
):
    items = store.filter_requests(status=status, kind=kind, limit=limit)
    metrics = store.request_metrics()
    chart = store.requests_timeseries()
    return render_template(
        "requests.html",
        request,
        items=items,
        status=status,
        kind=kind,
        metrics=metrics,
        chart=chart,
    )


@app.get("/models", response_class=HTMLResponse)
async def models_dashboard(
    request: Request,
    meta_store: MetadataStore = Depends(get_metadata_store),
):
    envelope = meta_store.get_envelope()
    history = envelope.history[::-1]
    fig_data = []
    if history:
        frame = []
        for record in history:
            row = {
                "model_version": record.model_version,
                "created_at": record.created_at,
                "accuracy": record.metrics.get("accuracy"),
                "f1_macro": record.metrics.get("f1_macro"),
                "avg_confidence": record.metrics.get("avg_confidence"),
                "data_volume": record.data_volume,
            }
            frame.append(row)
        df = pd.DataFrame(frame)
        fig = px.line(df, x="created_at", y=["accuracy", "f1_macro"], markers=True, title="Model Metrics")
        fig_data = _safe_figure_dict(fig)
    else:
        fig_data = _safe_figure_dict(px.line(title="No models yet"))
    active_models = repository.list_models(active_only=True, limit=10)
    model_history = repository.list_models(
        model_key="nomenclature_classifier", limit=25
    )
    recent_responses = repository.recent_responses(
        limit=12, model_key="nomenclature_classifier"
    )
    dataset_info = repository.latest_dataset_info(model_key="nomenclature_classifier")
    feature_list = FeatureBuilder.describe_features()
    return render_template(
        "models.html",
        request,
        envelope=envelope,
        history=history,
        chart=fig_data,
        active_models=active_models,
        model_history=model_history,
        recent_responses=recent_responses,
        dataset=dataset_info,
        features=feature_list,
    )


# --- API endpoints ---


@app.get("/monitoring/workers/stats")
async def workers_stats(store: MonitoringStore = Depends(get_monitoring_store)):
    return {"snapshot": store.latest_worker_snapshot(), "history": store.worker_history()}


@app.get("/monitoring/db/status")
async def db_status(store: MonitoringStore = Depends(get_monitoring_store)):
    return store.db_summary()


@app.get("/monitoring/requests/active")
async def active_requests(
    status: Optional[str] = None,
    kind: Optional[str] = None,
    store: MonitoringStore = Depends(get_monitoring_store),
):
    return store.filter_requests(status=status, kind=kind)


@app.post("/monitoring/db/init")
async def init_database(
    store: MonitoringStore = Depends(get_monitoring_store),
    _: HTTPBasicCredentials = Depends(require_admin),
):
    store.initialize_database()
    store.record_admin_action("db_init", "api")
    return {"status": "ok"}


@app.post("/monitoring/actions/stop-request/{request_id}")
async def stop_request(
    request_id: int,
    store: MonitoringStore = Depends(get_monitoring_store),
    _: HTTPBasicCredentials = Depends(require_admin),
):
    store.update_request(request_id, status="stopped", progress=100.0)
    store.record_admin_action("stop_request", "api", {"request_id": request_id})
    return {"status": "stopped"}


@app.post("/monitoring/actions/cleanup-logs")
async def cleanup_logs(
    store: MonitoringStore = Depends(get_monitoring_store),
    _: HTTPBasicCredentials = Depends(require_admin),
):
    cutoff = datetime.utcnow() - timedelta(days=30)
    with store.engine.begin() as conn:
        conn.execute(
            store.admin_logs.delete().where(store.admin_logs.c.created_at < cutoff)
        )
    store.record_admin_action("cleanup_logs", "api", {"before": cutoff.isoformat()})
    return {"status": "ok"}


@app.get("/monitoring/export/{fmt}")
async def export_stats(
    fmt: str,
    store: MonitoringStore = Depends(get_monitoring_store),
    _: HTTPBasicCredentials = Depends(require_admin),
):
    content = store.export_statistics(fmt)
    media_type = "application/json" if fmt == "json" else "text/csv"
    return Response(content=content, media_type=media_type)


@app.websocket("/ws/workers")
async def workers_ws(websocket: WebSocket, store: MonitoringStore = Depends(get_monitoring_store)):
    await websocket.accept()
    try:
        while True:
            data = {
                "snapshot": store.latest_worker_snapshot(),
                "history": store.worker_history(),
            }
            await websocket.send_json(data)
            await asyncio.sleep(10)
    except WebSocketDisconnect:
        return


@app.post("/datasets/upload")
async def upload_dataset(
    request: Request,
    model_key: str = Form("nomenclature_classifier"),
    task_type: str = Form("classification"),
    version_label: str = Form(...),
    dataset_name: str = Form("Интерактивная загрузка"),
    rewrite_dataset: bool = Form(False),
    trigger_training: bool = Form(False),
    target_field: str = Form("label"),
    feature_fields: str = Form(""),
    file: UploadFile = File(...),
):
    raw = await file.read()
    filename = file.filename or ""
    is_json = filename.lower().endswith(".json")
    
    items: list[NomenclatureItem] = []
    metadata = {}
    
    try:
        if is_json:
            payload = json.loads(raw.decode("utf-8"))
            if isinstance(payload, dict) and "data" in payload:
                metadata = {
                    "version": payload.get("version"),
                    "description": payload.get("description"),
                    "created_date": payload.get("created_date"),
                    "statistics": payload.get("statistics", {}),
                }
                records = payload["data"]
            elif isinstance(payload, list):
                records = payload
            else:
                raise ValueError("JSON должен содержать массив объектов или объект с полем 'data'")
        else:
            frame = pd.read_csv(io.BytesIO(raw))
            records = frame.to_dict("records")
    except json.JSONDecodeError as exc:
        raise HTTPException(status_code=400, detail=f"Неверный формат JSON: {exc}") from exc
    except Exception as exc:
        raise HTTPException(status_code=400, detail=f"Не удалось прочитать файл: {exc}") from exc
    
    for row in records:
        try:
            items.append(NomenclatureItem(**row))
        except ValidationError as exc:
            raise HTTPException(status_code=422, detail=f"Ошибка в данных: {exc}") from exc
    
    if not items:
        raise HTTPException(status_code=400, detail="Файл пуст или не содержит валидных строк.")
    
    try:
        dataset_record = repository.persist_dataset(
            model_key=model_key,
            task_type=task_type,
            version_label=version_label,
            dataset_name=dataset_name,
            items=items,
            rewrite=rewrite_dataset,
            source="dashboard",
        )
    except DatasetVersionExists:
        raise HTTPException(
            status_code=409,
            detail="Версия датасета уже существует. Укажите rewrite_dataset для перезаписи.",
        )

    stats_info = ""
    if metadata.get("statistics") and "total_items" in metadata["statistics"]:
        stats_info = f" (всего записей: {metadata['statistics']['total_items']})"
    
    message = f"Датасет {version_label} сохранён ({dataset_record.row_count} записей{stats_info})."
    if trigger_training:
        train_payload = {
            "model_key": model_key,
            "task_type": task_type,
            "dataset_name": dataset_name,
            "version": version_label,
            "rewrite_dataset": True,
            "target_field": target_field,
            "data": [item.dict() for item in items],
        }
        if feature_fields.strip():
            train_payload["feature_fields"] = [f.strip() for f in feature_fields.split(",") if f.strip()]
        try:
            async with httpx.AsyncClient(
                base_url=f"http://localhost:{settings.api_port}", timeout=120
            ) as client:
                response = await client.post("/train", json=jsonable_encoder(train_payload))
                response.raise_for_status()
            message += " Обучение запущено."
        except httpx.HTTPError as exc:
            message += f" Не удалось запустить обучение: {exc}"
    request.session["flash"] = message
    return RedirectResponse("/models", status_code=303)


@app.post("/models/activate")
async def activate_model_ui(
    request: Request,
    version: str = Form(...),
    model_key: str = Form("nomenclature_classifier"),
):
    repository.activate_model(version=version, model_key=model_key)
    request.session["flash"] = f"Модель {version} активирована."
    return RedirectResponse("/models", status_code=303)

