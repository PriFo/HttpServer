from __future__ import annotations

import json
from dataclasses import dataclass
from pathlib import Path
from typing import Iterable, Optional

import numpy as np
import pandas as pd
from scipy.spatial.distance import jensenshannon

from .config import settings
from .feature_store import FeatureBuilder
from .schemas import DriftReport, NomenclatureItem


@dataclass
class BaselineMeta:
    version: str
    row_count: int


class DriftMonitor:
    """Simple data drift watchdog built around PSI + Jensen-Shannon metrics."""

    NUMERIC_FEATURES = ("token_count", "name_len", "full_name_len")
    CATEGORICAL_FEATURES = ("kind", "unit", "type_hint")

    def __init__(self, baseline_path: Path | None = None):
        self.baseline_path = baseline_path or settings.drift_baseline_path
        self.meta_path = self.baseline_path.with_suffix(".meta.json")
        self.builder = FeatureBuilder()

    def _ensure_baseline(self) -> pd.DataFrame:
        if not self.baseline_path.exists():
            raise FileNotFoundError(
                "Drift baseline is missing. Train with refresh_baseline=True first."
            )
        return pd.read_parquet(self.baseline_path)

    def update_baseline(
        self, items: Iterable[NomenclatureItem], version: str
    ) -> BaselineMeta:
        features = self.builder.build(items)
        features.to_parquet(self.baseline_path, index=False)
        meta = BaselineMeta(version=version, row_count=len(features))
        self.meta_path.write_text(
            json.dumps(meta.__dict__, indent=2, ensure_ascii=False)
        )
        return meta

    def has_baseline(self) -> bool:
        return self.baseline_path.exists() and self.meta_path.exists()

    def _load_meta(self) -> Optional[BaselineMeta]:
        if not self.meta_path.exists():
            return None
        payload = json.loads(self.meta_path.read_text())
        return BaselineMeta(**payload)

    @staticmethod
    def _psi(expected: pd.Series, actual: pd.Series, buckets: int = 10) -> float:
        # Create quantile bins on expected distribution
        quantiles = np.linspace(0, 1, buckets + 1)
        bins = expected.quantile(quantiles).drop_duplicates().to_numpy()
        if len(bins) <= 1:
            return 0.0
        bins[0] = bins[0] - 1e-9
        bins[-1] = bins[-1] + 1e-9
        expected_counts = np.histogram(expected, bins=bins)[0] / len(expected)
        actual_counts = np.histogram(actual, bins=bins)[0] / len(actual)
        expected_counts = np.clip(expected_counts, 1e-6, None)
        actual_counts = np.clip(actual_counts, 1e-6, None)
        psi = np.sum((expected_counts - actual_counts) * np.log(expected_counts / actual_counts))
        return float(psi)

    def _categorical_js(self, base: pd.Series, current: pd.Series) -> float:
        union = sorted(set(base.unique()).union(current.unique()))
        base_probs = base.value_counts(normalize=True).reindex(union, fill_value=0)
        current_probs = current.value_counts(normalize=True).reindex(union, fill_value=0)
        return float(jensenshannon(base_probs, current_probs))

    def build_report(self, items: Iterable[NomenclatureItem]) -> DriftReport:
        baseline = self._ensure_baseline()
        current = self.builder.build(items)

        numeric_scores = {}
        for feature in self.NUMERIC_FEATURES:
            numeric_scores[feature] = self._psi(
                baseline[feature].astype(float), current[feature].astype(float)
            )

        categorical_scores = {}
        for feature in self.CATEGORICAL_FEATURES:
            categorical_scores[feature] = self._categorical_js(
                baseline[feature].astype(str), current[feature].astype(str)
            )

        combined = {**numeric_scores, **categorical_scores}
        psi_mean = float(np.mean(list(numeric_scores.values()))) if numeric_scores else 0.0
        triggered = psi_mean > 0.2 or any(score > 0.3 for score in categorical_scores.values())

        meta = self._load_meta()

        return DriftReport(
            feature_drift_scores=combined,
            population_stability_index=psi_mean,
            triggered=triggered,
            baseline_version=meta.version if meta else None,
        )

