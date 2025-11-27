from __future__ import annotations

from datetime import datetime
from enum import Enum
from typing import Any, ClassVar, Dict, List, Optional

from pydantic import BaseModel, Field, ConfigDict, validator


class NomenclatureType(str, Enum):
    """Устаревшее перечисление для обратной совместимости. Используйте строковые значения в label."""
    PRODUCT = "product"
    SERVICE = "service"


class NomenclatureItem(BaseModel):
    """Canonical representation of a nomenclature record."""

    name: str = Field(..., description="Short name as provided by source system.")
    full_name: Optional[str] = Field(
        default=None, description="Extended name or description."
    )
    kind: Optional[str] = Field(
        default=None, description="Original nomenclature kind (may be noisy)."
    )
    unit: Optional[str] = Field(default=None, description="Unit of measure.")
    type_hint: Optional[str] = Field(
        default=None,
        description="Raw type indicator coming from upstream (may be empty).",
    )
    okved_code: Optional[str] = Field(default=None, description="OKVED-2 code.")
    hs_code: Optional[str] = Field(
        default=None, description="HS / TN VED customs code."
    )
    country_code: Optional[str] = Field(
        default=None,
        description="ISO3166-1 alpha-2 country code where the record originates.",
    )
    jurisdiction: Optional[str] = Field(
        default=None, description="Regulatory jurisdiction identifier."
    )
    location_hint: Optional[str] = Field(
        default=None,
        description="Free-form location string (city/region) for compliance checks.",
    )
    encoding_hint: Optional[str] = Field(
        default=None,
        description="Original character encoding reported by upstream system.",
    )
    label: Optional[str] = Field(
        default=None, description="Ground-truth class for supervised tasks. Может быть любым строковым значением (Товар, Услуга, Тара, Набор, Работа и т.д.)."
    )

    LABEL_ALIASES: ClassVar[dict[str, str]] = {
        "товар": "Товар",
        "товары": "Товар",
        "product": "Товар",
        "goods": "Товар",
        "услуга": "Услуга",
        "услуги": "Услуга",
        "service": "Услуга",
        "services": "Услуга",
        "тара": "Тара",
        "набор": "Набор",
        "работа": "Работа",
    }

    @validator("name", "full_name", "kind", "unit", "type_hint", "okved_code", "hs_code")
    def _strip_whitespace(cls, value: Optional[str]) -> Optional[str]:
        if value is None:
            return value
        normalized = value.strip()
        return normalized or None

    @validator("country_code")
    def _normalize_country(cls, value: Optional[str]) -> Optional[str]:
        if not value:
            return value
        return value.strip().upper()

    @validator("jurisdiction")
    def _normalize_jur(cls, value: Optional[str]) -> Optional[str]:
        if not value:
            return value
        return value.strip().lower()

    @validator("label", pre=True)
    def _normalize_label(cls, value):
        if value is None:
            return value
        if isinstance(value, NomenclatureType):
            return value.value
        if isinstance(value, str):
            normalized = value.strip().lower().replace("ё", "е")
            mapped = cls.LABEL_ALIASES.get(normalized)
            if mapped:
                return mapped
            return value.strip()
        return str(value).strip() if value else None


class NomenclatureBatch(BaseModel):
    items: List[NomenclatureItem]

    def has_labels(self) -> bool:
        return any(item.label is not None for item in self.items)


class TrainRequest(NomenclatureBatch):
    items: List[NomenclatureItem] = Field(
        default_factory=list,
        description="(deprecated) inline dataset; use `data` for new uploads.",
    )
    data: Optional[List[NomenclatureItem]] = Field(
        default=None,
        description="Optional dataset appended к базовому train_dataset.csv.",
    )
    version: Optional[str] = Field(
        default=None,
        description="Версия датасета (и связанной модели) для текущего запуска.",
    )
    refresh_baseline: bool = Field(
        default=False,
        description="If true, replace the drift baseline with this dataset.",
    )
    model_key: str = Field(
        default="nomenclature_classifier",
        description="Базовая модель, к которой относится датасет/обучение.",
    )
    dataset_name: Optional[str] = Field(
        default=None,
        description="Человекочитаемое имя датасета для мониторинга.",
    )
    rewrite_dataset: bool = Field(
        default=False,
        description="Разрешить перезапись датасета с той же версией.",
    )
    task_type: str = Field(
        default="classification",
        description="Тип ML-задачи (classification/normalization/etc).",
    )
    target_field: Optional[str] = Field(
        default="label",
        description="Поле в NomenclatureItem, которое используется как целевая переменная для обучения. По умолчанию 'label'. Может быть любым строковым полем (label, type_hint, kind и т.д.).",
    )
    feature_fields: Optional[List[str]] = Field(
        default=None,
        description="Список полей, используемых как признаки для обучения. Если не указан, используются все доступные поля кроме target_field. Пример: ['name', 'full_name', 'kind', 'unit', 'okved_code', 'hs_code'].",
    )


class PredictRequest(NomenclatureBatch):
    top_k: int = Field(
        default=1,
        ge=1,
        le=2,
        description="Return the top-K classes in scoring responses.",
    )
    explain: bool = Field(
        default=False, description="If true, include SHAP-driven explanations."
    )
    model_key: str = Field(
        default="nomenclature_classifier",
        description="Ключ модели, которую следует использовать для инференса.",
    )


class PredictResult(BaseModel):
    name: str
    predicted_type: str  # Теперь произвольная строка, не только NomenclatureType
    probability: float
    alternatives: List[str] = Field(default_factory=list)  # Список строк
    explanation: Optional[str] = None


class UnexpectedItem(BaseModel):
    item: Dict[str, Any]
    issues: List[str] = Field(default_factory=list)


class PredictResponse(BaseModel):
    model_config = ConfigDict(protected_namespaces=())
    results: List[PredictResult]
    model_version: str
    unexpected_items: List[UnexpectedItem] = Field(default_factory=list)


class NormalizationRequest(BaseModel):
    items: List[NomenclatureItem]
    transliterate_to: str = Field(
        default="latin",
        description="Целевой алфавит для транслитерации (latin/cyrillic).",
    )
    locale: str = Field(default="ru", description="Локаль для форматирования.")


class NormalizationResult(BaseModel):
    name: str
    normalized_name: str
    normalized_full_name: Optional[str]
    corrections: List[str] = Field(default_factory=list)
    detected_dates: List[str] = Field(default_factory=list)
    detected_numbers: List[str] = Field(default_factory=list)
    transliteration: Optional[str] = None


class NormalizationResponse(BaseModel):
    results: List[NormalizationResult]


class QualityWarning(BaseModel):
    field: str
    issue: str
    affected_rows: int
    samples: List[dict] | None = None


class QualityReport(BaseModel):
    issues: List[QualityWarning] = Field(default_factory=list)
    passed: bool
    valid_items: List[Dict[str, Any]] = Field(default_factory=list, exclude=True)
    unexpected_items: List[UnexpectedItem] = Field(default_factory=list)


class DriftReport(BaseModel):
    feature_drift_scores: dict
    population_stability_index: float
    triggered: bool
    baseline_version: Optional[str]
    generated_at: datetime = Field(default_factory=datetime.utcnow)


class MetadataRecord(BaseModel):
    model_config = ConfigDict(protected_namespaces=())
    model_version: str
    created_at: datetime
    data_volume: int
    metrics: dict
    notes: Optional[str] = None


class MetadataEnvelope(BaseModel):
    current: Optional[MetadataRecord]
    history: List[MetadataRecord] = Field(default_factory=list)


class ModelActivationRequest(BaseModel):
    version: str
    model_key: Optional[str] = None

