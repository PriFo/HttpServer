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
        frame["full_name"] = frame["full_name"].fillna("")
        frame["kind"] = frame["kind"].fillna("unknown")
        frame["unit"] = frame["unit"].fillna("unit_na")
        frame["type_hint"] = frame["type_hint"].fillna("unassigned")
        frame["okved_code"] = frame["okved_code"].fillna("0000")
        frame["hs_code"] = frame["hs_code"].fillna("0000")
        frame["text_joined"] = (frame["name"].fillna("") + " " + frame["full_name"]).str.lower()
        frame["name_len"] = frame["name"].str.len().fillna(0)
        frame["full_name_len"] = frame["full_name"].str.len().fillna(0)
        frame["token_count"] = frame["text_joined"].str.split().apply(len)
        frame["service_kw_flag"] = frame["text_joined"].str.contains("услуг|service", regex=True)
        frame["product_kw_flag"] = frame["text_joined"].str.contains("товар|product|item", regex=True)
        return frame

    def _build_pipeline(self) -> Pipeline:
        categorical_cols = ["kind", "unit", "type_hint", "okved_code", "hs_code"]
        numeric_cols = ["name_len", "full_name_len", "token_count"]

        preprocessor = ColumnTransformer(
            transformers=[
                (
                    "text",
                    TfidfVectorizer(
                        max_features=20000,
                        ngram_range=(1, 2),
                        min_df=2,
                        sublinear_tf=True,
                    ),
                    "text_joined",
                ),
                (
                    "categorical",
                    OneHotEncoder(handle_unknown="ignore"),
                    categorical_cols,
                ),
                ("numeric", MaxAbsScaler(), numeric_cols),
            ],
            sparse_threshold=0.3,
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
    ) -> dict:
        frame = self._frame_from_items(items)
        if "label" not in frame.columns or frame["label"].isna().all():
            raise ValueError("Training requires labeled data (label field is empty).")

        def _normalize_label(value):
            if value is None:
                return None
            if isinstance(value, NomenclatureType):
                return value.value
            if isinstance(value, str):
                normalized = value.strip().lower().replace("ё", "е")
                return NomenclatureItem.LABEL_ALIASES.get(normalized, normalized)
            return str(value).strip().lower()

        frame["label"] = frame["label"].apply(_normalize_label)
        if frame["label"].isna().any() or (frame["label"] == "").any():
            raise ValueError("Training dataset contains unlabeled rows after normalization.")

        y = frame["label"]
        X = frame.drop(columns=["label"])

        if y.nunique() < 2:
            raise ValueError("Need at least two classes to train the classifier.")

        X_train, X_valid, y_train, y_valid = train_test_split(
            X, y, test_size=test_size, stratify=y, random_state=42
        )

        if settings.max_training_rows and len(X_train) > settings.max_training_rows:
            raise ValueError(
                f"Training dataset ({len(X_train)}) exceeds hard cap {settings.max_training_rows}."
            )

        self.pipeline = self._build_pipeline()
        self.pipeline.fit(X_train, y_train)

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
            predicted_class = NomenclatureType(classes[predicted_idx])
            alternatives = [
                NomenclatureType(classes[i])
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

