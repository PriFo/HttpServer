package database

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestDetectDatabaseType(t *testing.T) {
	// Создаем временную директорию для тестов
	tmpDir, err := os.MkdirTemp("", "test_db_analytics")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name     string
		setup    func() string
		wantType string
		wantErr  bool
	}{
		{
			name: "non-existent file",
			setup: func() string {
				return filepath.Join(tmpDir, "non_existent.db")
			},
			wantType: "unknown",
			wantErr:  true,
		},
		{
			name: "empty database",
			setup: func() string {
				dbPath := filepath.Join(tmpDir, "empty.db")
				// Создаем пустой файл БД, чтобы DetectDatabaseType не падал на os.Stat
				file, err := os.Create(dbPath)
				if err != nil {
					t.Fatalf("Failed to create empty DB file: %v", err)
				}
				file.Close()
				return dbPath
			},
			wantType: "unknown",
			wantErr:  false,
		},
		{
			name: "service database",
			setup: func() string {
				dbPath := filepath.Join(tmpDir, "service.db")
				db, err := sql.Open("sqlite3", dbPath)
				if err != nil {
					t.Fatalf("Failed to create test DB: %v", err)
				}
				defer db.Close()

				// Создаем таблицу clients
				_, err = db.Exec(`
					CREATE TABLE IF NOT EXISTS clients (
						id INTEGER PRIMARY KEY,
						name TEXT
					)
				`)
				if err != nil {
					t.Fatalf("Failed to create table: %v", err)
				}
				return dbPath
			},
			wantType: "service",
			wantErr:  false,
		},
		{
			name: "uploads database",
			setup: func() string {
				dbPath := filepath.Join(tmpDir, "uploads.db")
				db, err := sql.Open("sqlite3", dbPath)
				if err != nil {
					t.Fatalf("Failed to create test DB: %v", err)
				}
				defer db.Close()

				// Создаем таблицу uploads
				_, err = db.Exec(`
					CREATE TABLE IF NOT EXISTS uploads (
						id INTEGER PRIMARY KEY,
						name TEXT
					)
				`)
				if err != nil {
					t.Fatalf("Failed to create table: %v", err)
				}
				return dbPath
			},
			wantType: "uploads",
			wantErr:  false,
		},
		{
			name: "benchmarks database",
			setup: func() string {
				dbPath := filepath.Join(tmpDir, "benchmarks.db")
				db, err := sql.Open("sqlite3", dbPath)
				if err != nil {
					t.Fatalf("Failed to create test DB: %v", err)
				}
				defer db.Close()

				// Создаем таблицу client_benchmarks
				_, err = db.Exec(`
					CREATE TABLE IF NOT EXISTS client_benchmarks (
						id INTEGER PRIMARY KEY,
						name TEXT
					)
				`)
				if err != nil {
					t.Fatalf("Failed to create table: %v", err)
				}
				return dbPath
			},
			wantType: "benchmarks",
			wantErr:  false,
		},
		{
			name: "combined database",
			setup: func() string {
				dbPath := filepath.Join(tmpDir, "combined.db")
				db, err := sql.Open("sqlite3", dbPath)
				if err != nil {
					t.Fatalf("Failed to create test DB: %v", err)
				}
				defer db.Close()

				// Создаем обе таблицы
				_, err = db.Exec(`
					CREATE TABLE IF NOT EXISTS clients (
						id INTEGER PRIMARY KEY,
						name TEXT
					);
					CREATE TABLE IF NOT EXISTS uploads (
						id INTEGER PRIMARY KEY,
						name TEXT
					)
				`)
				if err != nil {
					t.Fatalf("Failed to create tables: %v", err)
				}
				return dbPath
			},
			wantType: "combined",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbPath := tt.setup()
			got, err := DetectDatabaseType(dbPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("DetectDatabaseType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantType {
				t.Errorf("DetectDatabaseType() = %v, want %v", got, tt.wantType)
			}
		})
	}
}

func TestGetTableStats(t *testing.T) {
	// Создаем временную директорию для тестов
	tmpDir, err := os.MkdirTemp("", "test_db_analytics")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name      string
		setup     func() string
		wantCount int
		wantErr   bool
	}{
		{
			name: "empty database",
			setup: func() string {
				dbPath := filepath.Join(tmpDir, "empty.db")
				file, err := os.Create(dbPath)
				if err != nil {
					t.Fatalf("Failed to create empty DB file: %v", err)
				}
				file.Close()
				return dbPath
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "database with tables",
			setup: func() string {
				dbPath := filepath.Join(tmpDir, "with_tables.db")
				db, err := sql.Open("sqlite3", dbPath)
				if err != nil {
					t.Fatalf("Failed to create test DB: %v", err)
				}
				defer db.Close()

				// Создаем несколько таблиц
				_, err = db.Exec(`
					CREATE TABLE IF NOT EXISTS table1 (
						id INTEGER PRIMARY KEY,
						name TEXT
					);
					CREATE TABLE IF NOT EXISTS table2 (
						id INTEGER PRIMARY KEY,
						value TEXT
					);
					INSERT INTO table1 (name) VALUES ('test1'), ('test2');
					INSERT INTO table2 (value) VALUES ('value1')
				`)
				if err != nil {
					t.Fatalf("Failed to create tables: %v", err)
				}
				return dbPath
			},
			wantCount: 2,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbPath := tt.setup()
			stats, err := GetTableStats(dbPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTableStats() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(stats) != tt.wantCount {
				t.Errorf("GetTableStats() count = %v, want %v", len(stats), tt.wantCount)
			}
			// Проверяем структуру статистики
			for _, stat := range stats {
				if stat.Name == "" {
					t.Error("GetTableStats() stat.Name is empty")
				}
				if stat.RowCount < 0 {
					t.Errorf("GetTableStats() stat.RowCount = %v, want >= 0", stat.RowCount)
				}
				if stat.SizeBytes < 0 {
					t.Errorf("GetTableStats() stat.SizeBytes = %v, want >= 0", stat.SizeBytes)
				}
				if stat.SizeMB < 0 {
					t.Errorf("GetTableStats() stat.SizeMB = %v, want >= 0", stat.SizeMB)
				}
			}
		})
	}
}

// Note: GetDatabaseHistory and UpdateDatabaseHistory require ServiceDB which is tested separately
// This test focuses on core analytics functions

func TestGetDatabaseAnalytics(t *testing.T) {
	// Создаем временную директорию для тестов
	tmpDir, err := os.MkdirTemp("", "test_db_analytics")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name     string
		setup    func() string
		wantType string
		wantErr  bool
	}{
		{
			name: "service database",
			setup: func() string {
				dbPath := filepath.Join(tmpDir, "service.db")
				db, err := sql.Open("sqlite3", dbPath)
				if err != nil {
					t.Fatalf("Failed to create test DB: %v", err)
				}
				defer db.Close()

				_, err = db.Exec(`
					CREATE TABLE IF NOT EXISTS clients (
						id INTEGER PRIMARY KEY,
						name TEXT
					);
					INSERT INTO clients (name) VALUES ('client1'), ('client2')
				`)
				if err != nil {
					t.Fatalf("Failed to create table: %v", err)
				}
				return dbPath
			},
			wantType: "service",
			wantErr:  false,
		},
		{
			name: "non-existent file",
			setup: func() string {
				return filepath.Join(tmpDir, "non_existent.db")
			},
			wantType: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbPath := tt.setup()
			analytics, err := GetDatabaseAnalytics(dbPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetDatabaseAnalytics() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if analytics == nil {
				t.Fatal("GetDatabaseAnalytics() returned nil")
			}
			if analytics.DatabaseType != tt.wantType {
				t.Errorf("GetDatabaseAnalytics() DatabaseType = %v, want %v", analytics.DatabaseType, tt.wantType)
			}
			if analytics.FilePath != dbPath {
				t.Errorf("GetDatabaseAnalytics() FilePath = %v, want %v", analytics.FilePath, dbPath)
			}
			if analytics.TotalSize <= 0 {
				t.Errorf("GetDatabaseAnalytics() TotalSize = %v, want > 0", analytics.TotalSize)
			}
			if analytics.TableCount < 0 {
				t.Errorf("GetDatabaseAnalytics() TableCount = %v, want >= 0", analytics.TableCount)
			}
			if analytics.TotalRows < 0 {
				t.Errorf("GetDatabaseAnalytics() TotalRows = %v, want >= 0", analytics.TotalRows)
			}
		})
	}
}

func TestGetDatabaseName(t *testing.T) {
	tests := []struct {
		name     string
		dbPath   string
		wantName string
	}{
		{
			name:     "simple filename",
			dbPath:   "test.db",
			wantName: "test",
		},
		{
			name:     "path with filename",
			dbPath:   "/path/to/database.db",
			wantName: "database",
		},
		{
			name:     "Windows path",
			dbPath:   "C:\\data\\mydb.db",
			wantName: "mydb",
		},
		{
			name:     "filename without extension",
			dbPath:   "database",
			wantName: "database",
		},
		{
			name:     "empty path",
			dbPath:   "",
			wantName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetDatabaseName(tt.dbPath)
			if got != tt.wantName {
				t.Errorf("GetDatabaseName(%q) = %v, want %v", tt.dbPath, got, tt.wantName)
			}
		})
	}
}

