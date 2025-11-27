from __future__ import annotations

from datetime import datetime
from enum import Enum
from typing import Any, ClassVar, Dict, List, Optional

from pydantic import BaseModel, Field, ConfigDict, validator


class NomenclatureType(str, Enum):
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
    label: Optional[NomenclatureType] = Field(
        default=None, description="Ground-truth class for supervised tasks."
    )

    LABEL_ALIASES: ClassVar[dict[str, NomenclatureType]] = {
        "товар": NomenclatureType.PRODUCT,
        "товары": NomenclatureType.PRODUCT,
        "product": NomenclatureType.PRODUCT,
        "goods": NomenclatureType.PRODUCT,
        "услуга": NomenclatureType.SERVICE,
        "услуги": NomenclatureType.SERVICE,
        "service": NomenclatureType.SERVICE,
        "services": NomenclatureType.SERVICE,
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
        if value is None or isinstance(value, NomenclatureType):
            return value
        if isinstance(value, str):
            normalized = value.strip().lower().replace("ё", "е")
            return cls.LABEL_ALIASES.get(normalized, value)
        return value


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
        description="Optional semantic version for the resulting model/feature set.",
    )
    refresh_baseline: bool = Field(
        default=False,
        description="If true, replace the drift baseline with this dataset.",
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


class PredictResult(BaseModel):
    name: str
    predicted_type: NomenclatureType
    probability: float
    alternatives: List[NomenclatureType] = Field(default_factory=list)
    explanation: Optional[str] = None


class UnexpectedItem(BaseModel):
    item: Dict[str, Any]
    issues: List[str] = Field(default_factory=list)


class PredictResponse(BaseModel):
    model_config = ConfigDict(protected_namespaces=())
    results: List[PredictResult]
    model_version: str
    unexpected_items: List[UnexpectedItem] = Field(default_factory=list)


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

