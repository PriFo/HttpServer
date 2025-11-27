from __future__ import annotations

from pathlib import Path
from typing import Iterable, List, Optional

import joblib
import numpy as np
import pandas as pd
from sklearn.compose import ColumnTransformer
from sklearn.metrics import accuracy_score, f1_score
from sklearn.model_selection import train_test_split
from sklearn.neural_network import MLPClassifier
from sklearn.pipeline import Pipeline
from sklearn.preprocessing import MaxAbsScaler, OneHotEncoder
from sklearn.feature_extraction.text import TfidfVectorizer

from .config import settings
from .explainability import ExplainabilityEngine
from .schemas import NomenclatureItem, NomenclatureType, PredictResult


class NomenclatureClassifier:
    """Wraps an sklearn MLP pipeline for training + inference."""

    def __init__(self, registry_dir: Path | None = None):
        self.registry_dir = registry_dir or settings.model_registry_dir
        self.registry_dir.mkdir(parents=True, exist_ok=True)
        self.pipeline: Optional[Pipeline] = None
        self.model_version: Optional[str] = None
        self.explainer = ExplainabilityEngine()

    @staticmethod
    def _frame_from_items(items: Iterable[NomenclatureItem]) -> pd.DataFrame:
        frame = pd.DataFrame([item.dict() for item in items])
        # Заполняем пропуски для основных полей
        frame["full_name"] = frame["full_name"].fillna("")
        frame["kind"] = frame["kind"].fillna("unknown")
        frame["unit"] = frame["unit"].fillna("unit_na")
        frame["type_hint"] = frame["type_hint"].fillna("unassigned")
        frame["okved_code"] = frame["okved_code"].fillna("0000")
        frame["hs_code"] = frame["hs_code"].fillna("0000")
        # Создаем дополнительные признаки
        frame["text_joined"] = (frame["name"].fillna("") + " " + frame["full_name"]).str.lower()
        frame["name_len"] = frame["name"].str.len().fillna(0)
        frame["full_name_len"] = frame["full_name"].str.len().fillna(0)
        frame["token_count"] = frame["text_joined"].str.split().apply(len)
        # Опциональные флаги (не используются, если не в feature_fields)
        frame["service_kw_flag"] = frame["text_joined"].str.contains("услуг|service", regex=True)
        frame["product_kw_flag"] = frame["text_joined"].str.contains("товар|product|item", regex=True)
        return frame

    def _build_pipeline(self, feature_fields: Optional[List[str]] = None, X_columns: Optional[List[str]] = None) -> Pipeline:
        """Строит pipeline с учетом указанных полей признаков."""
        # Определяем доступные поля
        available_categorical = ["kind", "unit", "type_hint", "okved_code", "hs_code", "country_code", "jurisdiction"]
        available_numeric = ["name_len", "full_name_len", "token_count"]
        available_text = ["text_joined"]
        available_bool = ["service_kw_flag", "product_kw_flag"]
        
        # Используем реальные колонки из X, если они переданы
        columns_to_use = X_columns if X_columns is not None else (feature_fields if feature_fields else None)
        
        if columns_to_use:
            # Фильтруем только те поля, которые реально есть в X и могут быть обработаны
            categorical_cols = [col for col in columns_to_use if col in available_categorical]
            numeric_cols = [col for col in columns_to_use if col in available_numeric]
            text_cols = [col for col in columns_to_use if col in available_text]
            bool_cols = [col for col in columns_to_use if col in available_bool]
            
            # Проверяем, что хотя бы некоторые поля были найдены
            if not (categorical_cols or numeric_cols or text_cols or bool_cols):
                # Если ни одно поле не найдено, но columns_to_use указаны, это ошибка
                unused_fields = [col for col in columns_to_use if col not in (available_categorical + available_numeric + available_text + available_bool)]
                raise ValueError(
                    f"Указанные поля признаков не могут быть обработаны pipeline. "
                    f"Необработанные поля: {unused_fields}. "
                    f"Доступные категориальные: {available_categorical}, "
                    f"числовые: {available_numeric}, "
                    f"текстовые: {available_text}, "
                    f"булевы: {available_bool}"
                )
        else:
            categorical_cols = available_categorical
            numeric_cols = available_numeric
            text_cols = available_text
            bool_cols = available_bool
        
        transformers = []
        
        # Текстовые поля
        if text_cols:
            for text_col in text_cols:
                transformers.append((
                    f"text_{text_col}",
                    TfidfVectorizer(
                        max_features=20000,
                        ngram_range=(1, 2),
                        min_df=2,
                        sublinear_tf=True,
                    ),
                    text_col,
                ))
        
        # Категориальные поля
        if categorical_cols:
            transformers.append((
                "categorical",
                OneHotEncoder(handle_unknown="ignore"),
                categorical_cols,
            ))
        
        # Числовые поля
        if numeric_cols:
            transformers.append((
                "numeric",
                MaxAbsScaler(),
                numeric_cols,
            ))
        
        # Булевы поля (преобразуем в числовые)
        if bool_cols:
            transformers.append((
                "bool",
                MaxAbsScaler(),
                bool_cols,
            ))
        
        if not transformers:
            raise ValueError("Не указано ни одного поля признаков для обучения.")
        
        preprocessor = ColumnTransformer(
            transformers=transformers,
            sparse_threshold=0.3,
            remainder="drop",  # Игнорируем поля, которые не обрабатываются
        )

        mlp = MLPClassifier(
            hidden_layer_sizes=(512, 256, 128),
            activation="relu",
            solver="adam",
            alpha=1e-4,
            batch_size=1024,
            learning_rate="adaptive",
            learning_rate_init=1e-3,
            max_iter=50,
            early_stopping=True,
            n_iter_no_change=5,
            warm_start=True,
            verbose=False,
        )

        pipeline = Pipeline(
            steps=[
                ("preprocessor", preprocessor),
                ("mlp", mlp),
            ]
        )
        return pipeline

    def train(
        self,
        items: List[NomenclatureItem],
        version: Optional[str] = None,
        test_size: float = 0.2,
        target_field: str = "label",
        feature_fields: Optional[List[str]] = None,
    ) -> dict:
        frame = self._frame_from_items(items)
        
        if target_field not in frame.columns:
            raise ValueError(f"Целевое поле '{target_field}' отсутствует в данных.")
        
        if frame[target_field].isna().all():
            raise ValueError(f"Training requires labeled data (поле '{target_field}' пустое).")

        def _normalize_target(value):
            if value is None:
                return None
            if isinstance(value, str):
                normalized = value.strip().lower().replace("ё", "е")
                mapped = NomenclatureItem.LABEL_ALIASES.get(normalized)
                if mapped:
                    return mapped
                return value.strip()
            return str(value).strip() if value else None

        frame[target_field] = frame[target_field].apply(_normalize_target)
        if frame[target_field].isna().any() or (frame[target_field] == "").any():
            raise ValueError(f"Датасет содержит пустые значения в поле '{target_field}' после нормализации.")

        y = frame[target_field]
        
        if feature_fields:
            missing_fields = [f for f in feature_fields if f not in frame.columns]
            if missing_fields:
                raise ValueError(f"Указанные поля признаков отсутствуют в данных: {missing_fields}")
            X = frame[feature_fields].copy()
            # Используем только те поля, которые реально есть в X
            actual_feature_fields = list(X.columns)
        else:
            exclude_cols = [target_field, "text_joined", "service_kw_flag", "product_kw_flag"]
            X = frame.drop(columns=[col for col in exclude_cols if col in frame.columns]).copy()
            actual_feature_fields = None  # Используем все доступные поля

        if y.nunique() < 2:
            raise ValueError("Need at least two classes to train the classifier.")

        X_train, X_valid, y_train, y_valid = train_test_split(
            X, y, test_size=test_size, stratify=y, random_state=42
        )

        if settings.max_training_rows and len(X_train) > settings.max_training_rows:
            raise ValueError(
                f"Training dataset ({len(X_train)}) exceeds hard cap {settings.max_training_rows}."
            )

        # Передаем реальные колонки из X_train, чтобы pipeline знал, какие поля обрабатывать
        # Важно: передаем именно колонки из X_train, чтобы избежать ошибок "column not found"
        self.pipeline = self._build_pipeline(feature_fields=actual_feature_fields, X_columns=X_train.columns.tolist())
        
        # Проверяем, что все поля, которые будут использоваться в pipeline, есть в X_train
        try:
            self.pipeline.fit(X_train, y_train)
        except KeyError as e:
            raise ValueError(
                f"Ошибка при обучении: поле отсутствует в данных. "
                f"Доступные колонки: {list(X_train.columns)}, "
                f"Ошибка: {e}"
            ) from e

        y_pred = self.pipeline.predict(X_valid)
        y_proba = self.pipeline.predict_proba(X_valid)
        metrics = {
            "accuracy": float(accuracy_score(y_valid, y_pred)),
            "f1_macro": float(f1_score(y_valid, y_pred, average="macro")),
        }
        metrics["avg_confidence"] = float(np.mean(np.max(y_proba, axis=1)))

        self.explainer.update_global_importance(self.pipeline, X_valid, y_valid)

        self.model_version = version or pd.Timestamp.utcnow().strftime("mlp_%Y%m%d%H%M%S")
        self._persist_pipeline()
        return metrics

    def _persist_pipeline(self) -> None:
        if not self.pipeline or not self.model_version:
            raise RuntimeError("No trained pipeline to persist.")
        out_path = self.registry_dir / f"{self.model_version}.joblib"
        joblib.dump(self.pipeline, out_path)

    def load_latest(self) -> Optional[str]:
        candidates = sorted(self.registry_dir.glob("*.joblib"))
        if not candidates:
            return None
        latest = candidates[-1]
        self.pipeline = joblib.load(latest)
        self.model_version = latest.stem
        return self.model_version

    def predict(
        self, items: List[NomenclatureItem], top_k: int = 1, explain: bool = False
    ) -> List[PredictResult]:
        if not self.pipeline:
            if not self.load_latest():
                raise RuntimeError("Model is not trained yet.")

        frame = self._frame_from_items(items)
        scores = self.pipeline.predict_proba(frame)
        classes = self.pipeline.classes_

        results: List[PredictResult] = []
        for idx, row in frame.iterrows():
            probs = scores[idx]
            order = np.argsort(probs)[::-1]
            predicted_idx = order[0]
            predicted_class = str(classes[predicted_idx])  # Произвольная строка
            alternatives = [
                str(classes[i])
                for i in order[1 : min(len(classes), top_k)]
            ]
            explanation = None
            if explain:
                explanation = self.explainer.explain_instance(
                    items[idx], predicted_class, float(probs[predicted_idx])
                )
            results.append(
                PredictResult(
                    name=items[idx].name,
                    predicted_type=predicted_class,
                    probability=float(probs[predicted_idx]),
                    alternatives=alternatives,
                    explanation=explanation,
                )
            )
        return results

