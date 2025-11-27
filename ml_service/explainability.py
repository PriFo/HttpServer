from __future__ import annotations

from collections import Counter
from typing import Iterable, Optional

import pandas as pd
from sklearn.inspection import permutation_importance

from .schemas import NomenclatureItem, NomenclatureType


class ExplainabilityEngine:
    """Hybrid explanation helper combining lexical heuristics & permutation importances."""

    SERVICE_KWS = (
        "услуг",
        "service",
        "обслуж",
        "аренда",
        "ремонт",
        "консультац",
        "логист",
        "техподдерж",
    )
    PRODUCT_KWS = (
        "товар",
        "оборудован",
        "комплект",
        "материал",
        "деталь",
        "product",
        "item",
        "device",
    )

    def __init__(self) -> None:
        self.global_importance: list[tuple[str, float]] = []

    def update_global_importance(
        self,
        pipeline,
        frame: pd.DataFrame,
        labels: Iterable[str],
        n_features: int = 15,
    ) -> None:
        """Derive feature importance for monitoring dashboards."""
        try:
            result = permutation_importance(
                pipeline,
                frame,
                labels,
                n_repeats=5,
                random_state=17,
                scoring="accuracy",
            )
        except Exception:
            return

        importance = sorted(
            zip(frame.columns, result.importances_mean), key=lambda x: abs(x[1]), reverse=True
        )
        self.global_importance = importance[:n_features]

    def explain_instance(
        self,
        item: NomenclatureItem,
        predicted: str,
        probability: float,
    ) -> str:
        """Объясняет предсказание для произвольного класса."""
        tokens = self._tokenize(item)
        reasons: list[str] = []

        service_hits = self._match(tokens, self.SERVICE_KWS)
        product_hits = self._match(tokens, self.PRODUCT_KWS)
        
        predicted_lower = str(predicted).lower()

        # Универсальные маркеры для разных классов
        if service_hits and ("услуг" in predicted_lower or "service" in predicted_lower):
            reasons.append(
                f"найдены сервисные маркеры {service_hits} в названии, что усилило вероятность '{predicted}'"
            )
        if product_hits and ("товар" in predicted_lower or "product" in predicted_lower or "goods" in predicted_lower):
            reasons.append(
                f"найдены товарные маркеры {product_hits} в названии, что усилило вероятность '{predicted}'"
            )
        if item.okved_code:
            if item.okved_code.startswith(("45", "46", "47")):
                reasons.append("класс OKВЭД относится к торговле")
            if item.okved_code.startswith(("62", "63", "69")):
                reasons.append("класс OKВЭД относится к услугам (IT/профсервисы)")

        prob_statement = (
            f"вероятность класса '{predicted}' = {probability:.2%} согласно нейросети"
        )
        reasons.insert(0, prob_statement)

        if not service_hits and not product_hits:
            reasons.append(
                "ключевых маркеров не обнаружено, решение основано на статистических признаках (tf-idf + one-hot)"
            )

        if self.global_importance:
            top_feature = self.global_importance[0][0]
            reasons.append(f"текущая глобальная фича-лидер: {top_feature}")

        return "; ".join(reasons)

    @staticmethod
    def _tokenize(item: NomenclatureItem) -> Counter:
        text = " ".join(filter(None, [item.name, item.full_name or ""])).lower()
        tokens = [
            token.strip(".,;:()\"'[]{}")
            for token in text.split()
            if token.strip(".,;:()\"'[]{}")
        ]
        return Counter(tokens)

    @staticmethod
    def _match(tokens: Counter, vocabulary: tuple[str, ...]) -> list[str]:
        hits = []
        for kw in vocabulary:
            if kw in tokens:
                hits.append(f"{kw}×{tokens[kw]}")
        return hits[:5]

