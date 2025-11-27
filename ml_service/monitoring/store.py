from __future__ import annotations

import asyncio
import json
from datetime import datetime, timedelta
from pathlib import Path
from typing import Any, Dict, List, Optional

import pandas as pd
import plotly.express as px
from plotly.utils import PlotlyJSONEncoder
import psutil
from sqlalchemy import (
    JSON,
    Column,
    DateTime,
    Float,
    Integer,
    MetaData,
    String,
    Table,
    Text,
    create_engine,
    func,
    insert,
    delete,
    select,
    update,
    text,
)
from sqlalchemy.engine import Engine
from sqlalchemy.exc import OperationalError
from sqlalchemy.sql import and_

from ..config import settings


class MonitoringStore:
    """Lightweight persistence layer for monitoring metrics."""

    def __init__(self, db_path: Path | None = None):
        self.db_path = db_path or settings.monitoring_db_path
        self.db_path.parent.mkdir(parents=True, exist_ok=True)
        self.engine: Engine = create_engine(
            f"sqlite:///{self.db_path}", future=True, echo=False
        )
        self.metadata = MetaData()
        self.requests = Table(
            "monitoring_requests",
            self.metadata,
            Column("id", Integer, primary_key=True),
            Column("kind", String, nullable=False),
            Column("status", String, nullable=False),
            Column("priority", Float, default=0.0),
            Column("progress", Float, default=0.0),
            Column("started_at", DateTime, default=datetime.utcnow),
            Column("updated_at", DateTime, default=datetime.utcnow),
            Column("completed_at", DateTime),
            Column("detail", JSON, default={}),
            Column("error", Text),
        )
        self.worker_snapshots = Table(
            "worker_snapshots",
            self.metadata,
            Column("id", Integer, primary_key=True),
            Column("created_at", DateTime, default=datetime.utcnow),
            Column("active_workers", Integer, default=0),
            Column("queued_tasks", Integer, default=0),
            Column("completed_tasks", Integer, default=0),
            Column("error_tasks", Integer, default=0),
            Column("cpu_usage", Float, default=0.0),
            Column("ram_usage", Float, default=0.0),
        )
        self.admin_logs = Table(
            "monitoring_admin_logs",
            self.metadata,
            Column("id", Integer, primary_key=True),
            Column("action", String, nullable=False),
            Column("actor", String, default="system"),
            Column("created_at", DateTime, default=datetime.utcnow),
            Column("payload", JSON, default={}),
        )
        self.system_events = Table(
            "monitoring_events",
            self.metadata,
            Column("id", Integer, primary_key=True),
            Column("created_at", DateTime, default=datetime.utcnow),
            Column("level", String, default="info"),
            Column("source", String, nullable=False),
            Column("message", Text, nullable=False),
            Column("payload", JSON, default={}),
        )
        try:
            self.metadata.create_all(self.engine)
        except OperationalError as exc:
            if "already exists" not in str(exc).lower():
                raise

    def record_admin_action(self, action: str, actor: str, payload: dict | None = None):
        with self.engine.begin() as conn:
            conn.execute(
                insert(self.admin_logs),
                {
                    "action": action,
                    "actor": actor,
                    "payload": payload or {},
                },
            )

    def record_event(
        self,
        *,
        level: str,
        source: str,
        message: str,
        payload: Optional[dict] = None,
    ) -> None:
        with self.engine.begin() as conn:
            conn.execute(
                insert(self.system_events),
                {
                    "level": level,
                    "source": source,
                    "message": message,
                    "payload": payload or {},
                },
            )

    def recent_events(self, limit: int = 10) -> list[dict]:
        query = (
            select(self.system_events)
            .order_by(self.system_events.c.created_at.desc())
            .limit(limit)
        )
        with self.engine.begin() as conn:
            rows = conn.execute(query).mappings().all()
            return [dict(row) for row in rows]

    def start_request(
        self, kind: str, priority: float, detail: Optional[dict] = None
    ) -> int:
        payload = {
            "kind": kind,
            "status": "queued",
            "priority": priority,
            "detail": detail or {},
        }
        with self.engine.begin() as conn:
            result = conn.execute(insert(self.requests), payload)
            return int(result.inserted_primary_key[0])

    def update_request(
        self,
        request_id: int,
        *,
        status: Optional[str] = None,
        progress: Optional[float] = None,
        detail: Optional[dict] = None,
        error: Optional[str] = None,
    ) -> None:
        values: Dict[str, Any] = {"updated_at": datetime.utcnow()}
        if status:
            values["status"] = status
        if progress is not None:
            values["progress"] = progress
        if detail is not None:
            values["detail"] = detail
        if error is not None:
            values["error"] = error
        if status in {"completed", "error"}:
            values["completed_at"] = datetime.utcnow()
            values["progress"] = progress or 100.0
        with self.engine.begin() as conn:
            conn.execute(
                update(self.requests).where(self.requests.c.id == request_id), values
            )

    def active_requests(self, limit: int = 50) -> list[dict]:
        query = (
            select(self.requests)
            .order_by(self.requests.c.started_at.desc())
            .limit(limit)
        )
        with self.engine.begin() as conn:
            rows = conn.execute(query).mappings().all()
            return [dict(row) for row in rows]

    def filter_requests(
        self,
        *,
        status: Optional[str] = None,
        kind: Optional[str] = None,
        start: Optional[datetime] = None,
        end: Optional[datetime] = None,
        limit: int = 200,
    ) -> list[dict]:
        query = select(self.requests)
        conditions = []
        if status:
            conditions.append(self.requests.c.status == status)
        if kind:
            conditions.append(self.requests.c.kind == kind)
        if start:
            conditions.append(self.requests.c.started_at >= start)
        if end:
            conditions.append(self.requests.c.started_at <= end)
        if conditions:
            query = query.where(and_(*conditions))
        query = query.order_by(self.requests.c.started_at.desc()).limit(limit)
        with self.engine.begin() as conn:
            rows = conn.execute(query).mappings().all()
            return [dict(row) for row in rows]

    def worker_usage(self, limit: int = 25) -> list[dict]:
        query = text(
            """
            SELECT prediction_id, request_kind, status, client_ip, user_agent,
                   created_at, meta, workers_allocated
            FROM predictions_log
            ORDER BY prediction_id DESC
            LIMIT :limit
            """
        )
        with self.engine.begin() as conn:
            rows = conn.execute(query, {"limit": limit}).mappings().all()
        usage: list[dict] = []
        for row in rows:
            meta = row.get("meta") or {}
            if isinstance(meta, str):
                try:
                    meta = json.loads(meta)
                except json.JSONDecodeError:
                    meta = {}
            usage.append(
                {
                    "id": row["prediction_id"],
                    "kind": row["request_kind"],
                    "status": row["status"],
                    "client_ip": row["client_ip"],
                    "user_agent": row["user_agent"],
                    "created_at": row["created_at"],
                    "meta": meta,
                    "workers_allocated": row["workers_allocated"],
                }
            )
        return usage

    def record_worker_snapshot(
        self,
        *,
        active_workers: int,
        queued_tasks: int,
        completed_tasks: int,
        error_tasks: int,
    ) -> None:
        try:
            cpu = psutil.cpu_percent(interval=None)
            ram = psutil.virtual_memory().percent
        except Exception:
            cpu = ram = 0.0
        payload = {
            "active_workers": active_workers,
            "queued_tasks": queued_tasks,
            "completed_tasks": completed_tasks,
            "error_tasks": error_tasks,
            "cpu_usage": cpu,
            "ram_usage": ram,
        }
        with self.engine.begin() as conn:
            conn.execute(insert(self.worker_snapshots), payload)

    def prune_worker_snapshots(self, days: int = 7) -> int:
        threshold = datetime.utcnow() - timedelta(days=days)
        stmt = delete(self.worker_snapshots).where(
            self.worker_snapshots.c.created_at < threshold
        )
        with self.engine.begin() as conn:
            result = conn.execute(stmt)
            return result.rowcount or 0

    def latest_worker_snapshot(self) -> dict:
        query = (
            select(self.worker_snapshots)
            .order_by(self.worker_snapshots.c.created_at.desc())
            .limit(1)
        )
        with self.engine.begin() as conn:
            row = conn.execute(query).mappings().first()
            return dict(row) if row else {}

    def worker_history(self, hours: int = 24) -> list[dict]:
        since = datetime.utcnow() - timedelta(hours=hours)
        query = (
            select(self.worker_snapshots)
            .where(self.worker_snapshots.c.created_at >= since)
            .order_by(self.worker_snapshots.c.created_at)
        )
        with self.engine.begin() as conn:
            rows = conn.execute(query).mappings().all()
            return [dict(row) for row in rows]

    def request_metrics(self, hours: int = 24) -> dict:
        since = datetime.utcnow() - timedelta(hours=hours)
        query = select(self.requests).where(self.requests.c.started_at >= since)
        with self.engine.begin() as conn:
            rows = conn.execute(query).mappings().all()
        total = len(rows)
        completed = sum(1 for row in rows if row["status"] == "completed")
        errored = sum(1 for row in rows if row["status"] == "error")
        running = sum(1 for row in rows if row["status"] == "running")
        queued = sum(1 for row in rows if row["status"] == "queued")
        retrain_needed = sum(
            1 for row in rows if row["detail"] and row["detail"].get("kind") == "train"
        )
        return {
            "window_hours": hours,
            "total": total,
            "completed": completed,
            "errored": errored,
            "running": running,
            "queued": queued,
            "retrain_candidates": retrain_needed,
        }

    def requests_timeseries(self, hours: int = 24) -> dict:
        data = self.filter_requests(
            start=datetime.utcnow() - timedelta(hours=hours), limit=5000
        )
        if not data:
            fig = px.line(title="No requests yet")
            return json.loads(json.dumps(fig, cls=PlotlyJSONEncoder))
        frame = pd.DataFrame(data)
        frame["started_at"] = pd.to_datetime(frame["started_at"])
        frame["bucket"] = frame["started_at"].dt.strftime("%Y-%m-%d %H:00")
        grouped = frame.groupby(["bucket", "status"]).size().reset_index(name="count")
        fig = px.bar(grouped, x="bucket", y="count", color="status", title="Requests")
        fig.update_layout(xaxis_title="Hour", yaxis_title="Count")
        return json.loads(json.dumps(fig, cls=PlotlyJSONEncoder))

    def db_summary(self) -> dict:
        stats = {
            "tables": 0,
            "size_mb": 0.0,
            "last_updated": None,
            "tables_detail": [],
        }
        if not self.db_path.exists():
            return stats

        import sqlite3

        conn = sqlite3.connect(self.db_path)
        try:
            cursor = conn.execute(
                "SELECT name FROM sqlite_master WHERE type='table' ORDER BY name"
            )
            tables = [row[0] for row in cursor.fetchall()]
            stats["tables"] = len(tables)
            for table in tables:
                count = conn.execute(f"SELECT COUNT(*) FROM {table}").fetchone()[0]
                column_count = conn.execute(
                    f"PRAGMA table_info({table})"
                ).fetchall()
                column_total = len(column_count)
                try:
                    last_ts = conn.execute(
                        f"SELECT MAX(created_at) FROM {table}"
                    ).fetchone()[0]
                except sqlite3.OperationalError:
                    last_ts = None
                stats["tables_detail"].append(
                    {
                        "name": table,
                        "column_count": column_total,
                        "count": count,
                        "last_entry": last_ts,
                    }
                )

            page_count = conn.execute("PRAGMA page_count").fetchone()[0]
            page_size = conn.execute("PRAGMA page_size").fetchone()[0]
            stats["size_mb"] = round((page_count * page_size) / 1_000_000, 2)
            stats["last_updated"] = max(
                (detail["last_entry"] for detail in stats["tables_detail"] if detail["last_entry"]),
                default=None,
            )
        finally:
            conn.close()
        return stats

    def fetch_table_preview(self, table: str, limit: int = 100) -> dict:
        import sqlite3

        if not self.db_path.exists():
            return {"columns": [], "rows": [], "error": "Database file missing"}

        conn = sqlite3.connect(self.db_path)
        conn.row_factory = sqlite3.Row
        try:
            exists = conn.execute(
                "SELECT name FROM sqlite_master WHERE type='table' AND name=?", (table,)
            ).fetchone()
            if not exists:
                return {"columns": [], "rows": [], "error": f"Table {table} not found"}
            cursor = conn.execute(f"SELECT * FROM {table} ORDER BY ROWID DESC LIMIT ?", (limit,))
            rows = cursor.fetchall()
            columns = rows[0].keys() if rows else []
            return {"columns": list(columns), "rows": [dict(row) for row in rows], "error": None}
        except sqlite3.OperationalError as exc:
            return {"columns": [], "rows": [], "error": str(exc)}
        finally:
            conn.close()

    def initialize_database(self) -> None:
        schema_file = Path(__file__).resolve().parent.parent / "sql" / "sqlite_schema.sql"
        if not schema_file.exists():
            raise FileNotFoundError(schema_file)
        import sqlite3

        conn = sqlite3.connect(self.db_path)
        try:
            with schema_file.open("r", encoding="utf-8") as fh:
                conn.executescript(fh.read())
        finally:
            conn.close()

    def export_statistics(self, fmt: str = "json") -> str:
        payload = {
            "requests": self.filter_requests(limit=1000),
            "worker_history": self.worker_history(),
            "db": self.db_summary(),
        }
        if fmt == "json":
            return json.dumps(payload, default=str, indent=2)
        frame = pd.DataFrame(payload["requests"])
        if frame.empty:
            return ""
        if fmt == "csv":
            return frame.to_csv(index=False)
        raise ValueError("Unsupported export format")


_store = MonitoringStore()


def get_monitoring_store() -> MonitoringStore:
    return _store

