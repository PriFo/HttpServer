from __future__ import annotations

import csv
from pathlib import Path
from typing import Iterable, List, Optional

from pydantic import ValidationError

from ..schemas import NomenclatureItem, NomenclatureType


class TrainingDatasetError(Exception):
    """Raised when the primary training dataset cannot be parsed."""


NAME_FIELDS = ("name", "название", "наименование")
FULL_NAME_FIELDS = ("full_name", "описание", "полное наименование")
LABEL_FIELDS = ("label", "тип", "class")
KIND_FIELDS = ("kind", "категория", "вид")
UNIT_FIELDS = ("unit", "единица", "единица измерения")
TYPE_HINT_FIELDS = ("type_hint", "type", "тип_подсказки")

LABEL_ALIASES = {
    "product": NomenclatureType.PRODUCT,
    "products": NomenclatureType.PRODUCT,
    "товар": NomenclatureType.PRODUCT,
    "товары": NomenclatureType.PRODUCT,
    "service": NomenclatureType.SERVICE,
    "services": NomenclatureType.SERVICE,
    "услуга": NomenclatureType.SERVICE,
    "услуги": NomenclatureType.SERVICE,
}

FILE_ENCODINGS = ("utf-8-sig", "utf-8", "cp1251")


def load_training_dataset(path: Path) -> List[NomenclatureItem]:
    """Load canonical training records from CSV located in datasets/."""

    if not path.exists():
        return []

    errors: List[str] = []
    records: List[NomenclatureItem] = []

    for encoding in FILE_ENCODINGS:
        try:
            with path.open("r", encoding=encoding) as fh:
                sample = fh.read(2048)
                fh.seek(0)
                try:
                    delimiter = csv.Sniffer().sniff(sample).delimiter
                except csv.Error:
                    delimiter = ","
                reader = csv.DictReader(fh, delimiter=delimiter)
                for idx, raw in enumerate(reader, start=2):
                    try:
                        item = _row_to_item(raw)
                    except TrainingDatasetError as exc:
                        errors.append(f"строка {idx}: {exc}")
                        continue
                    if item is not None:
                        records.append(item)
                break
        except UnicodeDecodeError:
            continue
    else:
        raise TrainingDatasetError(
            f"Не удалось прочитать {path.name}: неизвестная кодировка."
        )

    if errors:
        sample = "; ".join(errors[:5])
        raise TrainingDatasetError(
            f"train_dataset.csv содержит ошибки: {sample}"
        )
    return records


def _first_non_empty(row: dict, keys: Iterable[str]) -> Optional[str]:
    for key in keys:
        value = row.get(key)
        if value is None:
            continue
        text = str(value).strip()
        if text:
            return text
    return None


def _normalize_label(value: Optional[str]) -> Optional[NomenclatureType]:
    if not value:
        return None
    normalized = value.strip().lower().replace("ё", "е")
    return LABEL_ALIASES.get(normalized)


def _row_to_item(row: dict) -> Optional[NomenclatureItem]:
    name = _first_non_empty(row, NAME_FIELDS)
    if not name:
        raise TrainingDatasetError("не заполнено поле названия")

    label_value = _normalize_label(_first_non_empty(row, LABEL_FIELDS))
    if not label_value:
        raise TrainingDatasetError("не заполнено или неизвестно поле типа (label)")

    payload = {
        "name": name,
        "full_name": _first_non_empty(row, FULL_NAME_FIELDS) or name,
        "kind": _first_non_empty(row, KIND_FIELDS),
        "unit": _first_non_empty(row, UNIT_FIELDS),
        "type_hint": _first_non_empty(row, TYPE_HINT_FIELDS) or label_value.value,
        "label": label_value,
    }
    try:
        return NomenclatureItem(**payload)
    except ValidationError as exc:
        raise TrainingDatasetError(str(exc)) from exc

