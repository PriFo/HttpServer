from __future__ import annotations

import hashlib
import json
import sqlite3
from contextlib import contextmanager
from dataclasses import dataclass
from datetime import datetime
from pathlib import Path
from typing import Any, Iterable, Optional

from fastapi.encoders import jsonable_encoder

from .config import settings
from .schemas import NomenclatureItem


class DatasetVersionExists(Exception):
    """Raised when attempting to create a dataset version that already exists."""


class DatasetNotFound(Exception):
    """Raised when the requested dataset or version is missing."""


@dataclass
class DatasetRecord:
    dataset_id: int
    version_id: int
    version_label: str
    row_count: int


class MlRepository:
    """Thin SQLite-based repository for datasets, models и журналов запросов."""

    def __init__(self, db_path: Path | None = None) -> None:
        self.db_path = Path(db_path or settings.monitoring_db_path)
        self.db_path.parent.mkdir(parents=True, exist_ok=True)
        self._conn_kwargs = {"check_same_thread": False, "timeout": 30}
        self._ensure_schema()

    @contextmanager
    def _connect(self):
        conn = sqlite3.connect(self.db_path, **self._conn_kwargs)
        conn.row_factory = sqlite3.Row
        conn.execute("PRAGMA foreign_keys = ON")
        try:
            yield conn
        finally:
            conn.close()

    def _ensure_schema(self) -> None:
        with self._connect() as conn:
            self._ensure_column(
                conn,
                "datasets",
                "model_key",
                "TEXT NOT NULL DEFAULT 'nomenclature_classifier'",
            )
            self._ensure_column(
                conn,
                "datasets",
                "task_type",
                "TEXT NOT NULL DEFAULT 'classification'",
            )
            self._ensure_column(
                conn,
                "datasets",
                "latest_version_label",
                "TEXT",
            )
            self._ensure_column(
                conn,
                "dataset_versions",
                "row_count",
                "INTEGER DEFAULT 0",
            )
            self._ensure_column(
                conn,
                "dataset_versions",
                "task_type",
                "TEXT NOT NULL DEFAULT 'classification'",
            )
            self._ensure_column(
                conn,
                "dataset_versions",
                "dataset_hash",
                "TEXT",
            )
            self._ensure_column(
                conn,
                "training_jobs",
                "model_key",
                "TEXT NOT NULL DEFAULT 'nomenclature_classifier'",
            )
            self._ensure_column(
                conn,
                "training_jobs",
                "task_type",
                "TEXT NOT NULL DEFAULT 'classification'",
            )
            self._ensure_column(
                conn,
                "models",
                "task_type",
                "TEXT NOT NULL DEFAULT 'classification'",
            )
            self._ensure_column(
                conn,
                "models",
                "model_key",
                "TEXT NOT NULL DEFAULT 'nomenclature_classifier'",
            )
            self._ensure_column(
                conn,
                "models",
                "status",
                "TEXT NOT NULL DEFAULT 'draft'",
            )
            self._ensure_column(conn, "models", "accuracy", "REAL")
            self._ensure_column(conn, "models", "f1_macro", "REAL")
            self._ensure_column(conn, "models", "confidence", "REAL")
            self._ensure_column(
                conn,
                "predictions_log",
                "request_kind",
                "TEXT NOT NULL DEFAULT 'predict'",
            )
            self._ensure_column(
                conn,
                "predictions_log",
                "client_ip",
                "TEXT",
            )
            self._ensure_column(
                conn,
                "predictions_log",
                "user_agent",
                "TEXT",
            )
            self._ensure_column(
                conn,
                "predictions_log",
                "workers_allocated",
                "INTEGER NOT NULL DEFAULT 1",
            )
            self._ensure_column(
                conn,
                "predictions_log",
                "meta",
                "JSON",
            )

    @staticmethod
    def _ensure_column(conn: sqlite3.Connection, table: str, column: str, ddl: str) -> None:
        cursor = conn.execute(f"PRAGMA table_info({table})")
        columns = {row[1] for row in cursor.fetchall()}
        if column not in columns:
            conn.execute(f"ALTER TABLE {table} ADD COLUMN {column} {ddl}")
            conn.commit()

    def persist_dataset(
        self,
        *,
        model_key: str,
        task_type: str,
        version_label: str,
        dataset_name: str,
        items: list[NomenclatureItem],
        source: str = "api",
        rewrite: bool = False,
    ) -> DatasetRecord:
        if not items:
            raise ValueError("Нельзя создать пустой датасет.")

        normalized_name = dataset_name.strip() or model_key
        payload = [jsonable_encoder(item.dict()) for item in items]
        row_count = len(payload)
        checksum = hashlib.sha256(
            json.dumps(payload, ensure_ascii=False, sort_keys=True).encode("utf-8")
        ).hexdigest()

        with self._connect() as conn:
            dataset_id = self._get_or_create_dataset(
                conn, normalized_name, model_key, task_type, source
            )
            version_id = self._create_dataset_version(
                conn,
                dataset_id=dataset_id,
                version_label=version_label,
                task_type=task_type,
                rewrite=rewrite,
                checksum=checksum,
                row_count=row_count,
            )

            rows = []
            for row in payload:
                rows.append(
                    {
                        "dataset_id": dataset_id,
                        "version_id": version_id,
                        "name": row.get("name"),
                        "full_name": row.get("full_name"),
                        "kind": row.get("kind"),
                        "unit": row.get("unit"),
                        "type_hint": row.get("type_hint"),
                        "okved_code": row.get("okved_code"),
                        "hs_code": row.get("hs_code"),
                        "label": (row.get("label") or None),
                        "source_payload": json.dumps(row, ensure_ascii=False),
                    }
                )

            conn.executemany(
                """
                INSERT INTO dataset_items (
                    dataset_id, version_id, name, full_name, kind, unit,
                    type_hint, okved_code, hs_code, label, source_payload
                )
                VALUES (
                    :dataset_id, :version_id, :name, :full_name, :kind, :unit,
                    :type_hint, :okved_code, :hs_code, :label, :source_payload
                )
                """,
                rows,
            )
            conn.execute(
                """
                UPDATE datasets
                SET row_count = CASE WHEN :rewrite = 1 THEN :row_count ELSE COALESCE(row_count, 0) + :row_count END,
                    status = 'ready',
                    latest_version_label = :version_label
                WHERE dataset_id = :dataset_id
                """,
                {
                    "row_count": row_count,
                    "version_label": version_label,
                    "dataset_id": dataset_id,
                    "rewrite": 1 if rewrite else 0,
                },
            )
            conn.commit()
            return DatasetRecord(
                dataset_id=dataset_id,
                version_id=version_id,
                version_label=version_label,
                row_count=row_count,
            )

    def _get_or_create_dataset(
        self,
        conn: sqlite3.Connection,
        name: str,
        model_key: str,
        task_type: str,
        source: str,
    ) -> int:
        row = conn.execute(
            """
            SELECT dataset_id FROM datasets
            WHERE name = ? AND model_key = ?
            """,
            (name, model_key),
        ).fetchone()
        if row:
            return int(row["dataset_id"])
        cursor = conn.execute(
            """
            INSERT INTO datasets (name, source, description, model_key, task_type, status)
            VALUES (?, ?, ?, ?, ?, 'pending')
            """,
            (name, source, f"Автосбор для модели {model_key}", model_key, task_type),
        )
        return int(cursor.lastrowid)

    def _create_dataset_version(
        self,
        conn: sqlite3.Connection,
        *,
        dataset_id: int,
        version_label: str,
        task_type: str,
        rewrite: bool,
        checksum: str,
        row_count: int,
    ) -> int:
        existing = conn.execute(
            """
            SELECT version_id FROM dataset_versions
            WHERE dataset_id = ? AND version_label = ?
            """,
            (dataset_id, version_label),
        ).fetchone()
        if existing and not rewrite:
            raise DatasetVersionExists

        if existing and rewrite:
            version_id = int(existing["version_id"])
            conn.execute(
                "DELETE FROM dataset_items WHERE version_id = ?", (version_id,)
            )
            conn.execute(
                """
                UPDATE dataset_versions
                SET created_at = CURRENT_TIMESTAMP,
                    row_count = ?,
                    dataset_hash = ?,
                    task_type = ?
                WHERE version_id = ?
                """,
                (row_count, checksum, task_type, version_id),
            )
            return version_id

        cursor = conn.execute(
            """
            INSERT INTO dataset_versions (
                dataset_id, version_label, task_type, row_count, dataset_hash
            )
            VALUES (?, ?, ?, ?, ?)
            """,
            (dataset_id, version_label, task_type, row_count, checksum),
        )
        return int(cursor.lastrowid)

    def schedule_training_job(
        self,
        *,
        dataset_id: int,
        dataset_version: int,
        model_key: str,
        task_type: str,
        params: dict,
    ) -> int:
        with self._connect() as conn:
            cursor = conn.execute(
                """
                INSERT INTO training_jobs (
                    dataset_id, dataset_version, status, params, model_key, task_type, created_at
                )
                VALUES (?, ?, 'queued', ?, ?, ?, CURRENT_TIMESTAMP)
                """,
                (
                    dataset_id,
                    dataset_version,
                    json.dumps(params, ensure_ascii=False),
                    model_key,
                    task_type,
                ),
            )
            conn.commit()
            return int(cursor.lastrowid)

    def save_model_version(
        self,
        *,
        version: str,
        model_key: str,
        task_type: str,
        dataset_id: Optional[int],
        dataset_version: Optional[int],
        metrics: dict,
        artifact_path: Optional[str],
        activate: bool,
    ) -> int:
        metrics_json = json.dumps(metrics, ensure_ascii=False)
        accuracy = metrics.get("accuracy")
        f1_macro = metrics.get("f1_macro")
        confidence = metrics.get("avg_confidence")
        with self._connect() as conn:
            cursor = conn.execute(
                """
                INSERT INTO models (
                    version, dataset_id, dataset_version, metrics, artifact_path,
                    task_type, model_key, status, accuracy, f1_macro, confidence
                )
                VALUES (?, ?, ?, ?, ?, ?, ?, 'draft', ?, ?, ?)
                """,
                (
                    version,
                    dataset_id,
                    dataset_version,
                    metrics_json,
                    artifact_path,
                    task_type,
                    model_key,
                    accuracy,
                    f1_macro,
                    confidence,
                ),
            )
            model_id = int(cursor.lastrowid)
            if activate:
                self.activate_model(version=version, model_key=model_key)
            conn.commit()
            return model_id

    def activate_model(self, *, version: str, model_key: str) -> None:
        with self._connect() as conn:
            conn.execute(
                """
                UPDATE models SET status = 'archived' WHERE model_key = ?
                """,
                (model_key,),
            )
            conn.execute(
                """
                UPDATE models
                SET status = 'active', activated_at = CURRENT_TIMESTAMP
                WHERE version = ?
                """,
                (version,),
            )
            conn.commit()

    def log_request_start(
        self,
        *,
        kind: str,
        model_version: str,
        payload: Any,
        client_ip: Optional[str],
        user_agent: Optional[str],
        meta: Optional[dict],
    ) -> int:
        payload_str = json.dumps(jsonable_encoder(payload), ensure_ascii=False)
        meta_str = json.dumps(meta or {}, ensure_ascii=False)
        with self._connect() as conn:
            cursor = conn.execute(
                """
                INSERT INTO predictions_log (
                    model_version, request_payload, status, request_kind,
                    client_ip, user_agent, meta
                )
                VALUES (?, ?, 'pending', ?, ?, ?, ?)
                """,
                (model_version, payload_str, kind, client_ip, user_agent, meta_str),
            )
            conn.commit()
            return int(cursor.lastrowid)

    def finalize_request_log(
        self,
        log_id: int,
        *,
        status: str,
        response_payload: Any | None = None,
        error_message: Optional[str] = None,
        workers_allocated: int = 1,
        model_version: Optional[str] = None,
    ) -> None:
        response_str = (
            json.dumps(jsonable_encoder(response_payload), ensure_ascii=False)
            if response_payload is not None
            else None
        )
        with self._connect() as conn:
            conn.execute(
                """
                UPDATE predictions_log
                SET status = ?, response_payload = COALESCE(?, response_payload),
                    error_message = COALESCE(?, error_message),
                    completed_at = CURRENT_TIMESTAMP,
                    workers_allocated = ?,
                    model_version = COALESCE(?, model_version)
                WHERE prediction_id = ?
                """,
                (
                    status,
                    response_str,
                    error_message,
                    workers_allocated,
                    model_version,
                    log_id,
                ),
            )
            conn.commit()

    def attach_model_version_to_log(self, log_id: int, model_version: str) -> None:
        with self._connect() as conn:
            conn.execute(
                """
                UPDATE predictions_log
                SET model_version = ?
                WHERE prediction_id = ?
                """,
                (model_version, log_id),
            )
            conn.commit()

    def list_models(
        self,
        *,
        task_type: Optional[str] = None,
        model_key: Optional[str] = None,
        active_only: bool = False,
        limit: int = 50,
    ) -> list[dict]:
        query = """
            SELECT version, task_type, model_key, status, metrics, activated_at,
                   dataset_id, dataset_version, accuracy, f1_macro, confidence, created_at
            FROM models
            ORDER BY created_at DESC
            LIMIT ?
        """
        rows: list[sqlite3.Row]
        with self._connect() as conn:
            rows = conn.execute(query, (limit,)).fetchall()

        result = []
        for row in rows:
            if task_type and row["task_type"] != task_type:
                continue
            if model_key and row["model_key"] != model_key:
                continue
            if active_only and row["status"] != "active":
                continue
            metrics = {}
            if row["metrics"]:
                try:
                    metrics = json.loads(row["metrics"])
                except json.JSONDecodeError:
                    metrics = {}
            result.append(
                {
                    "version": row["version"],
                    "task_type": row["task_type"],
                    "model_key": row["model_key"],
                    "status": row["status"],
                    "metrics": metrics,
                    "activated_at": row["activated_at"],
                    "dataset_id": row["dataset_id"],
                    "dataset_version": row["dataset_version"],
                    "accuracy": row["accuracy"],
                    "f1_macro": row["f1_macro"],
                    "confidence": row["confidence"],
                    "created_at": row["created_at"],
                }
            )
        return result

    def latest_dataset_info(self, model_key: Optional[str] = None) -> Optional[dict]:
        query = """
            SELECT d.dataset_id, d.name, d.row_count, d.latest_version_label,
                   dv.created_at, dv.version_label, dv.row_count AS version_rows
            FROM datasets d
            LEFT JOIN dataset_versions dv
                ON d.dataset_id = dv.dataset_id
            WHERE d.status = 'ready'
        """
        params: tuple = ()
        if model_key:
            query += " AND d.model_key = ?"
            params = (model_key,)
        query += " ORDER BY dv.created_at DESC LIMIT 1"
        with self._connect() as conn:
            row = conn.execute(query, params).fetchone()
            if not row:
                return None
            return {
                "dataset_id": row["dataset_id"],
                "name": row["name"],
                "row_count": row["row_count"],
                "version_label": row["version_label"] or row["latest_version_label"],
                "version_rows": row["version_rows"],
                "last_used": row["created_at"],
            }

    def recent_responses(
        self,
        *,
        limit: int = 50,
        status: Optional[str] = None,
        model_key: Optional[str] = None,
    ) -> list[dict]:
        query = """
            SELECT prediction_id, model_version, status, request_kind,
                   request_payload, response_payload, client_ip, user_agent,
                   created_at, completed_at, error_message, meta, workers_allocated
            FROM predictions_log
            ORDER BY prediction_id DESC
            LIMIT ?
        """
        with self._connect() as conn:
            rows = conn.execute(query, (limit,)).fetchall()

        results = []
        for row in rows:
            if status and row["status"] != status:
                continue
            meta = {}
            if row["meta"]:
                try:
                    meta = json.loads(row["meta"])
                except json.JSONDecodeError:
                    meta = {}
            if model_key and meta.get("model_key") != model_key:
                continue
            results.append(
                {
                    "id": row["prediction_id"],
                    "model_version": row["model_version"],
                    "status": row["status"],
                    "request_kind": row["request_kind"],
                    "client_ip": row["client_ip"],
                    "user_agent": row["user_agent"],
                    "created_at": row["created_at"],
                    "completed_at": row["completed_at"],
                    "error_message": row["error_message"],
                    "meta": meta,
                    "workers_allocated": row["workers_allocated"],
                }
            )
        return results


repository = MlRepository()


