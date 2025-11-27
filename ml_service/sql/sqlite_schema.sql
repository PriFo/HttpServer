PRAGMA foreign_keys = ON;
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;

CREATE TABLE IF NOT EXISTS datasets (
    dataset_id      INTEGER PRIMARY KEY AUTOINCREMENT,
    name            TEXT NOT NULL,
    source          TEXT,
    description     TEXT,
    row_count       INTEGER DEFAULT 0,
    status          TEXT NOT NULL DEFAULT 'pending', -- pending | ready | archived
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS dataset_versions (
    version_id      INTEGER PRIMARY KEY AUTOINCREMENT,
    dataset_id      INTEGER NOT NULL REFERENCES datasets(dataset_id) ON DELETE CASCADE,
    version_label   TEXT NOT NULL,
    ingested_by     TEXT,
    checksum        TEXT,
    feature_version TEXT,
    notes           TEXT,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(dataset_id, version_label)
);

CREATE TABLE IF NOT EXISTS dataset_items (
    item_id         INTEGER PRIMARY KEY AUTOINCREMENT,
    dataset_id      INTEGER NOT NULL REFERENCES datasets(dataset_id) ON DELETE CASCADE,
    version_id      INTEGER REFERENCES dataset_versions(version_id) ON DELETE SET NULL,
    name            TEXT NOT NULL,
    full_name       TEXT,
    kind            TEXT,
    unit            TEXT,
    type_hint       TEXT,
    okved_code      TEXT,
    hs_code         TEXT,
    label           TEXT,
    source_payload  JSON,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS user_updates (
    update_id       INTEGER PRIMARY KEY AUTOINCREMENT,
    dataset_id      INTEGER NOT NULL REFERENCES datasets(dataset_id) ON DELETE CASCADE,
    item_id         INTEGER REFERENCES dataset_items(item_id) ON DELETE CASCADE,
    actor           TEXT,
    field_name      TEXT NOT NULL,
    previous_value  TEXT,
    new_value       TEXT,
    comment         TEXT,
    applied         INTEGER NOT NULL DEFAULT 0,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    applied_at      DATETIME
);

CREATE TABLE IF NOT EXISTS training_jobs (
    job_id          INTEGER PRIMARY KEY AUTOINCREMENT,
    dataset_id      INTEGER NOT NULL REFERENCES datasets(dataset_id) ON DELETE CASCADE,
    dataset_version INTEGER REFERENCES dataset_versions(version_id),
    status          TEXT NOT NULL DEFAULT 'queued', -- queued | running | failed | completed
    params          JSON,
    metrics         JSON,
    artifact_path   TEXT,
    initiated_by    TEXT,
    started_at      DATETIME,
    finished_at     DATETIME,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS models (
    model_id        INTEGER PRIMARY KEY AUTOINCREMENT,
    version         TEXT NOT NULL UNIQUE,
    dataset_id      INTEGER REFERENCES datasets(dataset_id),
    dataset_version INTEGER REFERENCES dataset_versions(version_id),
    training_job_id INTEGER REFERENCES training_jobs(job_id),
    artifact_path   TEXT,
    artifact_blob   BLOB,
    metrics         JSON,
    notes           TEXT,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    activated_at    DATETIME
);

CREATE TABLE IF NOT EXISTS predictions_log (
    prediction_id   INTEGER PRIMARY KEY AUTOINCREMENT,
    model_version   TEXT NOT NULL,
    request_payload JSON NOT NULL,
    response_payload JSON,
    status          TEXT DEFAULT 'pending', -- pending | success | failed
    error_message   TEXT,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at    DATETIME
);

CREATE INDEX IF NOT EXISTS idx_dataset_items_dataset ON dataset_items(dataset_id);
CREATE INDEX IF NOT EXISTS idx_dataset_items_kind ON dataset_items(kind);
CREATE INDEX IF NOT EXISTS idx_user_updates_dataset ON user_updates(dataset_id);
CREATE INDEX IF NOT EXISTS idx_training_jobs_dataset ON training_jobs(dataset_id);
CREATE INDEX IF NOT EXISTS idx_models_dataset ON models(dataset_id);

CREATE TRIGGER IF NOT EXISTS trg_datasets_updated
AFTER UPDATE ON datasets
FOR EACH ROW
BEGIN
    UPDATE datasets SET updated_at = CURRENT_TIMESTAMP WHERE dataset_id = NEW.dataset_id;
END;

CREATE TRIGGER IF NOT EXISTS trg_dataset_items_updated
AFTER UPDATE ON dataset_items
FOR EACH ROW
BEGIN
    UPDATE dataset_items SET updated_at = CURRENT_TIMESTAMP WHERE item_id = NEW.item_id;
END;

