erver/migrations/001_create_benchmarks.sql</path>
<content-- Create benchmarks table for storing reference data that will be prioritized over external AI services during normalization
-- This table stores the main reference data for entities like counterparties and nomenclature items

CREATE TABLE IF NOT EXISTS benchmarks (
    id TEXT PRIMARY KEY, -- UUID identifier for the benchmark record
    entity_type TEXT NOT NULL, -- Type of entity ('counterparty', 'nomenclature')
    name TEXT NOT NULL, -- Canonical name for the benchmark
    data TEXT NOT NULL, -- JSON data containing fields like inn, address, article, brand, etc.
    source_upload_id TEXT NOT NULL, -- Foreign key to uploads table (which upload this benchmark came from)
    source_client_id INTEGER NOT NULL, -- Foreign key to clients table (which client owns this benchmark)
    is_active BOOLEAN NOT NULL DEFAULT 1, -- Whether this benchmark is active for use
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP, -- When the record was created
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP -- When the record was last updated
);

-- Create benchmark_variations table for storing name variations of benchmarks
-- This helps with fuzzy matching and normalization by providing alternative names

CREATE TABLE IF NOT EXISTS benchmark_variations (
    id INTEGER PRIMARY KEY AUTOINCREMENT, -- Auto-incrementing ID for each variation
    benchmark_id TEXT NOT NULL, -- Foreign key to benchmarks table
    variation TEXT NOT NULL, -- Alternative name/variation for the benchmark
    FOREIGN KEY (benchmark_id) REFERENCES benchmarks(id) ON DELETE CASCADE -- Delete variations when benchmark is deleted
);

-- Create indexes for performance optimization

-- Index on benchmarks.entity_type for faster filtering by entity type
CREATE INDEX IF NOT EXISTS idx_benchmarks_entity_type ON benchmarks(entity_type);

-- Index on benchmarks.name for faster searching by name
CREATE INDEX IF NOT EXISTS idx_benchmarks_name ON benchmarks(name);

-- Index on benchmark_variations.variation for faster searching by variation text
CREATE INDEX IF NOT EXISTS idx_benchmark_variations_variation ON benchmark_variations(variation);