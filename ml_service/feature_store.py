from __future__ import annotations

import json
from dataclasses import dataclass
from datetime import datetime
from pathlib import Path
from typing import Iterable, Optional

import pandas as pd

from .config import settings
from .schemas import NomenclatureItem


@dataclass
class FeatureSetMeta:
    version: str
    row_count: int
    created_at: datetime
    comment: str | None = None

    def to_dict(self) -> dict:
        return {
            "version": self.version,
            "row_count": self.row_count,
            "created_at": self.created_at.isoformat(),
            "comment": self.comment,
        }


class FeatureBuilder:
    """Utility that converts normalized nomenclature rows into ML-ready features."""

    @staticmethod
    def _base_frame(items: Iterable[NomenclatureItem]) -> pd.DataFrame:
        payload = []
        for item in items:
            payload.append(item.dict())
        return pd.DataFrame(payload)

    def build(self, items: Iterable[NomenclatureItem]) -> pd.DataFrame:
        df = self._base_frame(items)

        if df.empty:
            return df

        df["name"] = df["name"].fillna("").str.lower()
        df["full_name"] = df["full_name"].fillna("").str.lower()
        df["kind"] = df["kind"].fillna("unknown").str.lower()
        df["unit"] = df["unit"].fillna("unit_na").str.lower()
        df["type_hint"] = df["type_hint"].fillna("unassigned").str.lower()
        df["okved_code"] = df["okved_code"].fillna("0000")
        df["hs_code"] = df["hs_code"].fillna("0000")

        df["text_joined"] = (df["name"] + " " + df["full_name"]).str.strip()
        df["token_count"] = df["text_joined"].str.split().apply(len)
        df["name_len"] = df["name"].str.len()
        df["full_name_len"] = df["full_name"].str.len()
        df["contains_service_kw"] = df["text_joined"].str.contains(
            "услуг|service", regex=True
        )
        df["contains_goods_kw"] = df["text_joined"].str.contains(
            "товар|product|item", regex=True
        )

        return df


class FeatureStore:
    """Lightweight feature registry w/ versioning and derived feature materialization."""

    def __init__(self, root_dir: Path | None = None):
        self.root_dir = root_dir or settings.feature_store_dir
        self.root_dir.mkdir(parents=True, exist_ok=True)
        self._index_path = self.root_dir / "index.json"
        self.builder = FeatureBuilder()

    def _load_index(self) -> dict:
        if not self._index_path.exists():
            return {"versions": []}
        return json.loads(self._index_path.read_text(encoding="utf-8"))

    def _persist_index(self, index: dict) -> None:
        self._index_path.write_text(json.dumps(index, indent=2, ensure_ascii=False))

    def ingest(
        self,
        items: Iterable[NomenclatureItem],
        version: Optional[str] = None,
        comment: Optional[str] = None,
    ) -> FeatureSetMeta:
        features = self.builder.build(items)

        if features.empty:
            raise ValueError("Cannot ingest an empty feature set.")

        version = version or datetime.utcnow().strftime("v%Y%m%d%H%M%S")
        out_path = self.root_dir / f"{version}.parquet"
        features.to_parquet(out_path, index=False)

        meta = FeatureSetMeta(
            version=version,
            row_count=len(features),
            created_at=datetime.utcnow(),
            comment=comment,
        )

        index = self._load_index()
        index["versions"].append(meta.to_dict())
        index["versions"] = sorted(index["versions"], key=lambda x: x["created_at"])
        self._persist_index(index)

        return meta

    def load(self, version: Optional[str] = None) -> pd.DataFrame:
        if version:
            path = self.root_dir / f"{version}.parquet"
            if not path.exists():
                raise FileNotFoundError(f"Feature version {version} not found.")
            return pd.read_parquet(path)

        index = self._load_index()
        if not index["versions"]:
            raise FileNotFoundError("Feature store is empty.")
        latest_version = index["versions"][-1]["version"]
        return self.load(latest_version)

    def list_versions(self) -> list[FeatureSetMeta]:
        index = self._load_index()
        return [
            FeatureSetMeta(
                version=entry["version"],
                row_count=entry["row_count"],
                created_at=datetime.fromisoformat(entry["created_at"]),
                comment=entry.get("comment"),
            )
            for entry in index.get("versions", [])
        ]

