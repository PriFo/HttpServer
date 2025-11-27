package database

import (
	"database/sql"
	"fmt"
)

// InitGostsSchema initializes the GOST database schema with all required tables
func InitGostsSchema(db *sql.DB) error {
	schema := `
	-- Table for storing GOST standards metadata
	CREATE TABLE IF NOT EXISTS gosts (
		id INTEGER PRIMARY KEY,
		gost_number TEXT UNIQUE NOT NULL,        -- GOST number (e.g., "ГОСТ 12345-2020")
		title TEXT NOT NULL,                     -- Title of the standard
		adoption_date DATE,                      -- Adoption date
		effective_date DATE,                     -- Effective date
		status TEXT,                            -- Status (действующий, отменен, заменен)
		source_type TEXT,                       -- Source type (national, interstate, etc.)
		source_id INTEGER,                      -- Foreign key to gost_sources table
		source_url TEXT,                        -- URL to the source
		description TEXT,                       -- Description
		keywords TEXT,                          -- Keywords for search
		created_at TIMESTAMP,
		updated_at TIMESTAMP,
		FOREIGN KEY(source_id) REFERENCES gost_sources(id) ON DELETE SET NULL
	);

	-- Table for storing GOST standard documents
	CREATE TABLE IF NOT EXISTS gost_documents (
		id INTEGER PRIMARY KEY,
		gost_id INTEGER NOT NULL,                -- Foreign key to gosts table
		file_path TEXT,                         -- File path on disk
		file_type TEXT,                         -- File type (pdf, doc, docx)
		file_size INTEGER,                      -- File size
		uploaded_at TIMESTAMP,
		FOREIGN KEY(gost_id) REFERENCES gosts(id) ON DELETE CASCADE
	);

	-- Table for storing GOST sources information
	CREATE TABLE IF NOT EXISTS gost_sources (
		id INTEGER PRIMARY KEY,
		source_name TEXT UNIQUE NOT NULL,       -- Source name
		source_url TEXT,                        -- Source URL
		last_sync_date TIMESTAMP,               -- Last synchronization date
		records_count INTEGER,                  -- Number of records from this source
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Indexes for performance optimization
	CREATE INDEX IF NOT EXISTS idx_gosts_number ON gosts(gost_number);
	CREATE INDEX IF NOT EXISTS idx_gosts_title ON gosts(title);
	CREATE INDEX IF NOT EXISTS idx_gosts_status ON gosts(status);
	CREATE INDEX IF NOT EXISTS idx_gosts_source_type ON gosts(source_type);
	CREATE INDEX IF NOT EXISTS idx_gosts_keywords ON gosts(keywords);
	CREATE INDEX IF NOT EXISTS idx_gosts_adoption_date ON gosts(adoption_date);
	CREATE INDEX IF NOT EXISTS idx_gost_documents_gost_id ON gost_documents(gost_id);
	CREATE INDEX IF NOT EXISTS idx_gost_sources_name ON gost_sources(source_name);
	`

	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create gosts schema: %w", err)
	}

	return nil
}
