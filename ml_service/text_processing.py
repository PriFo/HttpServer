from __future__ import annotations

import re
import unicodedata
from datetime import datetime
from typing import Iterable, List, Tuple

from dateutil import parser as date_parser
from unidecode import unidecode

from .schemas import NomenclatureItem, NormalizationRequest, NormalizationResponse, NormalizationResult


class TextNormalizer:
    """Правило-ориентированная нормализация текстов и служебных полей."""

    DATE_PATTERN = re.compile(r"\b\d{1,4}[./\-]\d{1,2}[./\-]\d{1,4}\b")
    NUMBER_PATTERN = re.compile(r"-?\d+[.,]?\d*")

    def normalize(self, payload: NormalizationRequest) -> NormalizationResponse:
        results: List[NormalizationResult] = []
        for item in payload.items:
            normalized_name, name_corrections = self._normalize_text(item.name or "")
            normalized_full_name, full_name_corrections = self._normalize_text(
                item.full_name or ""
            )
            corrections = [*name_corrections, *full_name_corrections]
            dates = self._extract_dates(item)
            numbers = self._extract_numbers(item)
            translation = self._transliterate(
                item, target=payload.transliterate_to or "latin"
            )
            results.append(
                NormalizationResult(
                    name=item.name,
                    normalized_name=normalized_name,
                    normalized_full_name=normalized_full_name or None,
                    corrections=corrections,
                    detected_dates=dates,
                    detected_numbers=numbers,
                    transliteration=translation,
                )
            )
        return NormalizationResponse(results=results)

    @staticmethod
    def _normalize_text(text: str) -> Tuple[str, List[str]]:
        corrections: List[str] = []
        cleaned = unicodedata.normalize("NFKC", text)
        if cleaned != text:
            corrections.append("приведена нормализация Unicode (NFKC)")
        collapsed = re.sub(r"\s+", " ", cleaned).strip()
        if collapsed != cleaned:
            corrections.append("лишние пробелы удалены")
        capitalized = collapsed.capitalize()
        if capitalized != collapsed:
            corrections.append("приведено к предложному регистру")
        return capitalized, corrections

    def _extract_dates(self, item: NomenclatureItem) -> List[str]:
        text = " ".join(filter(None, [item.name, item.full_name]))
        matches = []
        for candidate in self.DATE_PATTERN.findall(text):
            try:
                parsed = date_parser.parse(candidate, dayfirst=True, yearfirst=False)
            except (ValueError, OverflowError):
                continue
            matches.append(parsed.strftime("%Y-%m-%d"))
        return matches[:5]

    def _extract_numbers(self, item: NomenclatureItem) -> List[str]:
        text = " ".join(filter(None, [item.name, item.full_name]))
        matches = []
        for number in self.NUMBER_PATTERN.findall(text):
            normalized = number.replace(",", ".")
            matches.append(normalized)
        return matches[:5]

    def _transliterate(self, item: NomenclatureItem, target: str) -> str | None:
        text = " ".join(filter(None, [item.name, item.full_name])).strip()
        if not text:
            return None
        if target == "latin":
            return unidecode(text)
        if target == "ascii":
            return unidecode(text)
        return text


text_normalizer = TextNormalizer()

