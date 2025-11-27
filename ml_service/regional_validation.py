from __future__ import annotations

import string
from typing import Iterable

from fastapi import HTTPException

from .schemas import NomenclatureItem


class RegionalValidator:
    """Performs lightweight regional / encoding compliance checks."""

    SUPPORTED_ENCODINGS = {"utf-8", "utf8", None}
    ISO_ALPHA2 = {
        "RU",
        "BY",
        "KZ",
        "AM",
        "CN",
        "TR",
        "KG",
        "UA",
        "DE",
        "US",
        "AE",
    }

    def validate_batch(self, items: Iterable[NomenclatureItem]) -> None:
        for idx, item in enumerate(items):
            self._validate_encoding(item, idx)
            self._validate_country(item, idx)
            self._validate_codes(item, idx)

    def _validate_encoding(self, item: NomenclatureItem, position: int) -> None:
        if item.encoding_hint and item.encoding_hint.lower() not in self.SUPPORTED_ENCODINGS:
            raise HTTPException(
                status_code=415,
                detail=f"Item #{position} uses unsupported encoding {item.encoding_hint}.",
            )

    def _validate_country(self, item: NomenclatureItem, position: int) -> None:
        if item.country_code and item.country_code not in self.ISO_ALPHA2:
            raise HTTPException(
                status_code=422,
                detail=f"Item #{position} has unsupported country code {item.country_code}.",
            )

    def _validate_codes(self, item: NomenclatureItem, position: int) -> None:
        if item.hs_code and not self._is_digit_like(item.hs_code):
            raise HTTPException(
                status_code=422,
                detail=f"Item #{position} has invalid HS/TN VED code.",
            )
        if item.okved_code and not self._is_okved_like(item.okved_code):
            raise HTTPException(
                status_code=422,
                detail=f"Item #{position} has invalid OKVED code structure.",
            )

    @staticmethod
    def _is_digit_like(value: str) -> bool:
        allowed = set(string.digits)
        sanitized = value.replace(".", "").replace(" ", "")
        return all(ch in allowed for ch in sanitized)

    @staticmethod
    def _is_okved_like(value: str) -> bool:
        sanitized = value.replace(".", "")
        return sanitized.isdigit() and 2 <= len(sanitized) <= 6

