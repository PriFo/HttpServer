from __future__ import annotations

import argparse
import csv
import json
import logging
from dataclasses import asdict, dataclass
from pathlib import Path
from typing import Dict, Iterable, Iterator, List, Optional

import pandas as pd

from ..config import settings

logger = logging.getLogger("dataset_builder")


COLUMN_ALIASES = {
    "name": "name",
    "название": "name",
    "наименование": "name",
    "item_name": "name",
    "full_name": "full_name",
    "полное наименование": "full_name",
    "описание": "full_name",
    "kind": "kind",
    "категория": "kind",
    "вид": "kind",
    "unit": "unit",
    "единица": "unit",
    "единица измерения": "unit",
    "type": "type_hint",
    "тип": "type_hint",
    "label": "label",
    "класс": "kind",
    "оквэд": "okved_code",
    "оквэд2": "okved_code",
    "okved": "okved_code",
    "hs": "hs_code",
    "тнвэд": "hs_code",
    "country": "country_code",
    "страна": "country_code",
    "jurisdiction": "jurisdiction",
    "юрисдикция": "jurisdiction",
    "region": "location_hint",
    "регион": "location_hint",
    "encoding": "encoding_hint",
    "кодировка": "encoding_hint",
}

TARGET_COLUMNS = [
    "name",
    "full_name",
    "kind",
    "unit",
    "type_hint",
    "okved_code",
    "hs_code",
    "country_code",
    "jurisdiction",
    "location_hint",
    "encoding_hint",
    "label",
]

LABEL_MAP = {
    "товар": "product",
    "товары": "product",
    "product": "product",
    "goods": "product",
    "услуга": "service",
    "услуги": "service",
    "service": "service",
    "services": "service",
}

ENCODING_CANDIDATES = (
    "utf-8-sig",
    "utf-8",
    "cp1251",
    "windows-1251",
    "utf-16",
    "utf-16-le",
    "utf-16-be",
)
DELIMITER_FALLBACK = (",", ";", "\t", "|")


@dataclass
class FileStats:
    name: str
    rows_included: int = 0
    rows_dropped: int = 0

    def asdict(self) -> dict:
        return asdict(self)


class DatasetBuilder:
    """Combines heterogenous CSV files into a canonical training dataset."""

    def __init__(
        self,
        dataset_dir: Path,
        output_parquet: Path,
        output_csv: Optional[Path] = None,
        chunk_size: int = 100_000,
        row_limit: Optional[int] = None,
    ) -> None:
        self.dataset_dir = dataset_dir
        self.output_parquet = output_parquet
        self.output_csv = output_csv
        self.chunk_size = chunk_size
        self.row_limit = row_limit
        self.processed: List[FileStats] = []
        self.failed: Dict[str, str] = {}

    def build(self) -> dict:
        frames: List[pd.DataFrame] = []
        logger.info("Scanning %s for CSV files", self.dataset_dir)
        for csv_path in sorted(self.dataset_dir.glob("*.csv")):
            stats = FileStats(name=csv_path.name)
            try:
                for chunk in self._iter_chunks(csv_path):
                    original_len = len(chunk)
                    normalized = self._normalize_chunk(chunk)
                    stats.rows_included += len(normalized)
                    stats.rows_dropped += original_len - len(normalized)
                    if not normalized.empty:
                        frames.append(normalized)
            except Exception as exc:  # noqa: BLE001
                logger.exception("Failed to process %s: %s", csv_path.name, exc)
                self.failed[csv_path.name] = str(exc)
                continue

            if stats.rows_included:
                self.processed.append(stats)
            elif stats.rows_dropped:
                self.failed[csv_path.name] = "no valid rows"
            else:
                self.failed[csv_path.name] = "empty file"

        if not frames:
            raise RuntimeError("No usable rows were collected from datasets.")

        combined = pd.concat(frames, ignore_index=True)
        combined = combined.drop_duplicates(subset=["name", "full_name", "label"])
        if self.row_limit:
            combined = combined.head(self.row_limit)

        logger.info("Final dataset shape: rows=%s", len(combined))

        self.output_parquet.parent.mkdir(parents=True, exist_ok=True)
        combined.to_parquet(self.output_parquet, index=False)
        logger.info("Saved parquet dataset to %s", self.output_parquet)

        if self.output_csv:
            self.output_csv.parent.mkdir(parents=True, exist_ok=True)
            combined.to_csv(self.output_csv, index=False)
            logger.info("Saved CSV dataset to %s", self.output_csv)

        label_counts = combined["label"].value_counts().to_dict()
        summary = {
            "rows_total": int(len(combined)),
            "label_distribution": label_counts,
            "processed_files": [entry.asdict() for entry in self.processed],
            "failed_files": self.failed,
            "output_parquet": str(self.output_parquet),
            "output_csv": str(self.output_csv) if self.output_csv else None,
        }
        return summary

    def _iter_chunks(self, path: Path) -> Iterator[pd.DataFrame]:
        delimiter = self._detect_delimiter(path)
        last_error: Optional[Exception] = None
        for encoding in ENCODING_CANDIDATES:
            try:
                reader = pd.read_csv(
                    path,
                    delimiter=delimiter,
                    encoding=encoding,
                    dtype=str,
                    chunksize=self.chunk_size,
                )
                for chunk in reader:
                    yield chunk
                return
            except UnicodeDecodeError as exc:
                last_error = exc
                continue
        raise RuntimeError(f"Unable to decode {path.name}: {last_error}")

    def _normalize_chunk(self, chunk: pd.DataFrame) -> pd.DataFrame:
        rename_map = {}
        for column in chunk.columns:
            alias = self._normalize_column_name(column)
            if alias:
                rename_map[column] = alias

        if "name" not in rename_map.values() and "name" not in chunk.columns:
            return pd.DataFrame(columns=TARGET_COLUMNS)

        normalized = chunk.rename(columns=rename_map)
        normalized = normalized[[col for col in normalized.columns if col in TARGET_COLUMNS]].copy()

        for col in TARGET_COLUMNS:
            if col not in normalized.columns:
                normalized[col] = None

        normalized = normalized[TARGET_COLUMNS]

        for column in ["name", "full_name", "kind", "unit", "type_hint", "okved_code", "hs_code"]:
            normalized[column] = normalized[column].apply(self._clean_str)

        normalized["name"] = normalized["name"].fillna("")
        normalized["name"] = normalized["name"].apply(self._clean_str)

        normalized["full_name"] = normalized["full_name"].fillna("")
        normalized["full_name"] = normalized.apply(
            lambda row: row["full_name"] or row["name"], axis=1
        )

        normalized["kind"] = normalized["kind"].apply(self._clean_str).fillna("unknown")
        normalized["unit"] = normalized["unit"].apply(self._clean_str).fillna("unit_na")

        normalized["label"] = normalized["label"].apply(self._clean_str)
        normalized["type_hint"] = normalized["type_hint"].apply(self._clean_str)

        normalized["label"] = normalized["label"].combine_first(normalized["type_hint"])
        normalized["label"] = normalized["label"].apply(self._normalize_label)
        normalized["type_hint"] = normalized["type_hint"].apply(self._normalize_type_hint)
        normalized["type_hint"] = normalized["type_hint"].fillna(normalized["label"]).fillna(
            "unassigned"
        )

        normalized["label"] = normalized["label"].where(
            normalized["label"].isin({"product", "service"})
        )

        normalized = normalized.dropna(subset=["name", "label"])
        normalized["name"] = normalized["name"].apply(self._clean_str)
        normalized = normalized[normalized["name"].notna()]

        normalized["country_code"] = normalized["country_code"].apply(self._clean_country)
        normalized["jurisdiction"] = normalized["jurisdiction"].apply(self._clean_lower)
        normalized["location_hint"] = normalized["location_hint"].apply(self._clean_str)
        normalized["encoding_hint"] = normalized["encoding_hint"].apply(self._clean_encoding)

        return normalized.reset_index(drop=True)

    @staticmethod
    def _normalize_column_name(column: str) -> Optional[str]:
        slug = column.strip().lower()
        return COLUMN_ALIASES.get(slug)

    @staticmethod
    def _clean_str(value: Optional[str]) -> Optional[str]:
        if value is None:
            return None
        if isinstance(value, float) and pd.isna(value):
            return None
        text = str(value).strip()
        return text or None

    @staticmethod
    def _clean_lower(value: Optional[str]) -> Optional[str]:
        cleaned = DatasetBuilder._clean_str(value)
        if cleaned:
            return cleaned.lower()
        return None

    @staticmethod
    def _clean_country(value: Optional[str]) -> Optional[str]:
        cleaned = DatasetBuilder._clean_str(value)
        if cleaned:
            return cleaned.upper()
        return None

    @staticmethod
    def _clean_encoding(value: Optional[str]) -> Optional[str]:
        cleaned = DatasetBuilder._clean_str(value)
        if cleaned:
            return cleaned.lower()
        return None

    @staticmethod
    def _normalize_label(value: Optional[str]) -> Optional[str]:
        cleaned = DatasetBuilder._clean_str(value)
        if not cleaned:
            return None
        normalized = cleaned.replace("ё", "е").lower()
        normalized = normalized.replace(" ", "")
        return LABEL_MAP.get(normalized)

    @staticmethod
    def _normalize_type_hint(value: Optional[str]) -> Optional[str]:
        normalized = DatasetBuilder._normalize_label(value)
        if normalized:
            return normalized
        return DatasetBuilder._clean_lower(value)

    def _detect_delimiter(self, path: Path) -> str:
        sample = path.read_bytes()[:20_000]
        try:
            decoded = sample.decode("utf-8", errors="ignore")
            dialect = csv.Sniffer().sniff(decoded)
            if dialect.delimiter in DELIMITER_FALLBACK:
                return dialect.delimiter
        except csv.Error:
            pass

        counts = {delim: sample.count(delim.encode()) for delim in DELIMITER_FALLBACK}
        return max(counts, key=counts.get)


def run_cli(argv: Optional[Iterable[str]] = None) -> dict:
    parser = argparse.ArgumentParser(description="Build canonical nomenclature dataset.")
    default_dir = Path(__file__).resolve().parents[1] / "datasets"
    parser.add_argument(
        "--dataset-dir",
        type=Path,
        default=default_dir,
        help="Folder with raw CSV exports.",
    )
    parser.add_argument(
        "--output-parquet",
        type=Path,
        default=settings.reference_dataset_path,
        help="Destination for parquet dataset.",
    )
    parser.add_argument(
        "--output-csv",
        type=Path,
        default=Path("ml_service") / "datasets" / "nomenclature_master.csv",
        help="Optional CSV dump.",
    )
    parser.add_argument(
        "--chunk-size",
        type=int,
        default=100_000,
        help="Rows per chunk when streaming CSV files.",
    )
    parser.add_argument(
        "--limit",
        type=int,
        default=None,
        help="Keep only the first N rows of the combined dataset.",
    )
    parser.add_argument(
        "--log-level",
        default="INFO",
        choices=["DEBUG", "INFO", "WARNING", "ERROR"],
        help="Logging verbosity.",
    )
    args = parser.parse_args(list(argv) if argv is not None else None)

    logging.basicConfig(level=getattr(logging, args.log_level))
    builder = DatasetBuilder(
        dataset_dir=args.dataset_dir,
        output_parquet=args.output_parquet,
        output_csv=args.output_csv,
        chunk_size=args.chunk_size,
        row_limit=args.limit,
    )
    summary = builder.build()
    print(json.dumps(summary, ensure_ascii=False, indent=2))
    return summary


if __name__ == "__main__":
    run_cli()

