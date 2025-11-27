from __future__ import annotations

from pathlib import Path
from typing import Optional

from pydantic import Field, ConfigDict, field_validator
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    """Centralized configuration for the ML microservice."""

    model_config = SettingsConfigDict(
        env_prefix="ML_",
        case_sensitive=False,
        protected_namespaces=("settings_",),
    )

    artifacts_root: Path = Field(
        default=Path("ml_artifacts"),
        description="Root directory for all persisted ML artifacts.",
    )
    model_registry_dir: Path = Field(
        default=Path("ml_artifacts/models"),
        description="Directory that stores serialized pipelines.",
    )
    feature_store_dir: Path = Field(
        default=Path("ml_artifacts/feature_store"),
        description="Directory for persisted feature data & metadata.",
    )
    datasets_dir: Path = Field(
        default=Path("ml_service/datasets"),
        description="Directory containing canonical CSV datasets.",
    )
    metadata_store_path: Path = Field(
        default=Path("ml_artifacts/metadata.json"),
        description="JSON file for model metadata & governance info.",
    )
    drift_baseline_path: Path = Field(
        default=Path("ml_artifacts/drift_baseline.parquet"),
        description="Reference dataset for drift monitoring.",
    )
    reference_dataset_path: Path = Field(
        default=Path("ml_artifacts/reference_dataset.parquet"),
        description="Canonical labeled dataset used for mixing with client data.",
    )
    default_train_dataset: Path = Field(
        default=Path("ml_service/datasets/train_dataset.csv"),
        description="Path to base CSV used when /train receives no data block.",
    )
    monitoring_db_path: Path = Field(
        default=Path("ml_store.db"),
        description="SQLite database used by monitoring dashboard and trackers.",
    )
    shap_sample_size: int = Field(
        default=256,
        ge=32,
        description="Number of samples to use when building SHAP explainers.",
    )
    max_training_rows: Optional[int] = Field(
        default=None,
        description="Optional guardrail to cap in-memory training data size.",
    )
    api_host: str = Field(default="0.0.0.0")
    api_port: int = Field(default=8085)
    predict_timeout_seconds: int = Field(
        default=120,
        ge=5,
        description="Max time to wait for a queued prediction task.",
    )
    priority_workers: int = Field(
        default=4,
        ge=1,
        le=16,
        description="Number of background workers serving priority queue.",
    )
    monitor_admin_user: str = Field(default="admin")
    monitor_admin_password: str = Field(default="admin")

    @field_validator(
        "artifacts_root",
        "model_registry_dir",
        "feature_store_dir",
        "datasets_dir",
        mode="before",
    )
    def _ensure_path(cls, value: Path | str) -> Path:
        path = Path(value).expanduser()
        path.mkdir(parents=True, exist_ok=True)
        return path

    @field_validator(
        "metadata_store_path",
        "drift_baseline_path",
        "reference_dataset_path",
        "default_train_dataset",
        "monitoring_db_path",
        mode="before",
    )
    def _expand_file(cls, value: Path | str) -> Path:
        path = Path(value).expanduser()
        path.parent.mkdir(parents=True, exist_ok=True)
        return path


# Instantiate once for reuse across modules
settings = Settings()

