package database

import "errors"

// Доменные ошибки для Database Domain
var (
	ErrInvalidDatabaseID   = errors.New("invalid database ID")
	ErrDatabaseNotFound    = errors.New("database not found")
	ErrDatabaseNameRequired = errors.New("database name is required")
	ErrDatabasePathRequired = errors.New("database file path is required")
	ErrDatabaseExists      = errors.New("database already exists")
	ErrInvalidProjectID    = errors.New("invalid project ID")
	ErrProjectNotFound     = errors.New("project not found")
	ErrConnectionFailed    = errors.New("database connection failed")
)

