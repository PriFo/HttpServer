from __future__ import annotations

import json
from dataclasses import dataclass
from pathlib import Path
from typing import Iterable, List, Sequence

import numpy as np
import pandas as pd

from .config import settings
from .schemas import NomenclatureItem


def _items_to_frame(items: Sequence[NomenclatureItem]) -> pd.DataFrame:
    return pd.DataFrame([item.dict() for item in items])


@dataclass
class MixingReport:
    total_rows: int
    reference_rows: int
    client_rows: int
    reference_ratio: float
    oversampled: bool


class ReferenceDatasetManager:
    """Handles canonical labeled dataset storage and controlled data mixing."""

    def __init__(self, path: Path | None = None, min_reference_ratio: float = 0.1):
        self.path = path or settings.reference_dataset_path
        self.path.parent.mkdir(parents=True, exist_ok=True)
        self.min_reference_ratio = min_reference_ratio

    def has_reference(self) -> bool:
        return self.path.exists()

    def load_reference(self) -> pd.DataFrame:
        if not self.has_reference():
            raise FileNotFoundError("Reference dataset is missing.")
        return pd.read_parquet(self.path)

    def save_reference(self, frame: pd.DataFrame) -> None:
        frame.to_parquet(self.path, index=False)

    def append_reference(self, items: Iterable[NomenclatureItem]) -> None:
        frame = _items_to_frame(list(items))
        if frame.empty:
            return
        if self.has_reference():
            existing = self.load_reference()
            combined = pd.concat([existing, frame], ignore_index=True)
        else:
            combined = frame
        deduped = combined.drop_duplicates(subset=["name", "full_name", "kind"])
        self.save_reference(deduped)

    def mix_for_training(
        self, client_items: Sequence[NomenclatureItem]
    ) -> tuple[pd.DataFrame, MixingReport]:
        client_frame = _items_to_frame(client_items)
        oversampled = False

        if not self.has_reference():
            mixed = client_frame.copy()
            report = MixingReport(
                total_rows=len(mixed),
                reference_rows=0,
                client_rows=len(client_frame),
                reference_ratio=0.0,
                oversampled=False,
            )
            return self._shuffle_frame(mixed), report

        reference_frame = self.load_reference()
        if reference_frame.empty:
            self.path.unlink(missing_ok=True)
            return self.mix_for_training(client_items)

        total_rows = len(reference_frame) + len(client_frame)
        current_ratio = len(reference_frame) / max(total_rows, 1)

        mixed = pd.concat([reference_frame, client_frame], ignore_index=True)

        if current_ratio < self.min_reference_ratio and len(reference_frame) > 0:
            needed_reference = int(
                np.ceil(self.min_reference_ratio * len(mixed)) - len(reference_frame)
            )
            reps = max(1, int(np.ceil(needed_reference / len(reference_frame))))
            reference_augmented = pd.concat(
                [reference_frame] * reps, ignore_index=True
            )
            mixed = pd.concat([reference_augmented, client_frame], ignore_index=True)
            oversampled = True
            total_rows = len(mixed)
            current_ratio = len(reference_augmented) / total_rows

        mixed = self._shuffle_frame(mixed)
        report = MixingReport(
            total_rows=len(mixed),
            reference_rows=len(reference_frame),
            client_rows=len(client_frame),
            reference_ratio=round(current_ratio, 4),
            oversampled=oversampled,
        )
        return mixed, report

    @staticmethod
    def _shuffle_frame(frame: pd.DataFrame) -> pd.DataFrame:
        if frame.empty:
            return frame
        rng = np.random.default_rng(seed=42)
        indices = np.arange(len(frame))
        rng.shuffle(indices)
        return frame.iloc[indices].reset_index(drop=True)

