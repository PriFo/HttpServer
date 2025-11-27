from __future__ import annotations

import json
from datetime import datetime
from pathlib import Path
from typing import List, Optional

from fastapi.encoders import jsonable_encoder

from .config import settings
from .schemas import MetadataEnvelope, MetadataRecord


class MetadataStore:
    """Simple JSON-backed registry for tracking model lineage & governance info."""

    def __init__(self, path: Path | None = None):
        self.path = path or settings.metadata_store_path
        self.path.parent.mkdir(parents=True, exist_ok=True)

    def _load(self) -> MetadataEnvelope:
        if not self.path.exists():
            return MetadataEnvelope(current=None, history=[])
        payload = json.loads(self.path.read_text(encoding="utf-8"))
        history = [
            MetadataRecord(
                model_version=entry["model_version"],
                created_at=datetime.fromisoformat(entry["created_at"]),
                data_volume=entry["data_volume"],
                metrics=entry["metrics"],
                notes=entry.get("notes"),
            )
            for entry in payload.get("history", [])
        ]
        current_payload = payload.get("current")
        current = (
            MetadataRecord(
                model_version=current_payload["model_version"],
                created_at=datetime.fromisoformat(current_payload["created_at"]),
                data_volume=current_payload["data_volume"],
                metrics=current_payload["metrics"],
                notes=current_payload.get("notes"),
            )
            if current_payload
            else None
        )
        return MetadataEnvelope(current=current, history=history)

    def _save(self, envelope: MetadataEnvelope) -> None:
        payload = {
            "current": envelope.current.dict() if envelope.current else None,
            "history": [record.dict() for record in envelope.history],
        }
        encoded = jsonable_encoder(payload)
        self.path.write_text(json.dumps(encoded, indent=2, ensure_ascii=False))

    def add_record(
        self,
        version: str,
        data_volume: int,
        metrics: dict,
        notes: Optional[str] = None,
    ) -> MetadataRecord:
        envelope = self._load()
        record = MetadataRecord(
            model_version=version,
            created_at=datetime.utcnow(),
            data_volume=data_volume,
            metrics=metrics,
            notes=notes,
        )
        if envelope.current:
            envelope.history.append(envelope.current)
        envelope.current = record
        self._save(envelope)
        return record

    def get_envelope(self) -> MetadataEnvelope:
        return self._load()

