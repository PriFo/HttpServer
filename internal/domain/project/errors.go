package project

import "errors"

// Доменные ошибки для Project Domain
var (
	ErrInvalidProjectID   = errors.New("invalid project ID")
	ErrProjectNotFound    = errors.New("project not found")
	ErrProjectNameRequired = errors.New("project name is required")
	ErrProjectExists      = errors.New("project already exists")
	ErrInvalidClientID    = errors.New("invalid client ID")
	ErrClientNotFound     = errors.New("client not found")
)

