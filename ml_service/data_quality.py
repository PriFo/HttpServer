from __future__ import annotations

from collections import defaultdict
from typing import Any, Dict, Iterable, List, Set

import pandas as pd
from pydantic import ValidationError

from .schemas import (
    NomenclatureItem,
    QualityReport,
    QualityWarning,
    UnexpectedItem,
)


class DataQualityGuard:
    """Runs lightweight validation rules before data hits the model."""

    REQUIRED_FIELDS = ("name", "full_name")

    def evaluate(self, items: Iterable[NomenclatureItem]) -> QualityReport:
        rows = []
        for idx, item in enumerate(items):
            try:
                rows.append(item.dict())
            except ValidationError as exc:
                raise ValueError(f"Invalid payload at index {idx}: {exc}") from exc

        if not rows:
            raise ValueError("Quality guard received an empty payload.")

        frame = pd.DataFrame(rows)
        issues: List[QualityWarning] = []
        unexpected_flags: Dict[int, Set[str]] = defaultdict(set)

        def _register_issue(
            *,
            field: str,
            issue: str,
            mask: pd.Series,
            sample_columns: List[str] | None = None,
        ) -> None:
            affected = int(mask.sum())
            if affected <= 0:
                return
            columns = [col for col in (sample_columns or frame.columns.tolist()) if col in frame.columns]
            sample_df = frame.loc[mask, columns] if columns else frame.loc[mask]
            issues.append(
                QualityWarning(
                    field=field,
                    issue=issue,
                    affected_rows=affected,
                    samples=sample_df.head(5).to_dict("records"),
                )
            )
            for idx in frame.index[mask]:
                unexpected_flags[idx].add(issue)

        for field in self.REQUIRED_FIELDS:
            if field not in frame.columns:
                continue
            values = frame[field].fillna("").astype(str).str.strip()
            missing_mask = values == ""
            _register_issue(
                field=field,
                issue="missing_values",
                mask=missing_mask,
                sample_columns=[field],
            )

        if {"name", "full_name"}.issubset(frame.columns):
            duplicates_mask = frame.duplicated(subset=["name", "full_name"], keep=False)
            _register_issue(
                field="name/full_name",
                issue="duplicates",
                mask=duplicates_mask,
                sample_columns=["name", "full_name"],
            )

        if "kind" in frame.columns:
            noisy_kind = frame["kind"].fillna("").astype(str).str.len() > 128
            _register_issue(
                field="kind",
                issue="suspicious_length",
                mask=noisy_kind,
                sample_columns=["name", "kind"],
            )

        if "okved_code" in frame.columns:
            okved_invalid = frame["okved_code"].fillna("").astype(str).str.contains(
                r"[^0-9\.]", regex=True
            )
            _register_issue(
                field="okved_code",
                issue="invalid_characters",
                mask=okved_invalid,
                sample_columns=["name", "okved_code"],
            )

        unexpected_items = [
            UnexpectedItem(item=rows[idx], issues=sorted(tags))
            for idx, tags in unexpected_flags.items()
        ]

        return QualityReport(
            issues=issues,
            passed=len(issues) == 0,
            valid_items=rows,
            unexpected_items=unexpected_items,
        )

