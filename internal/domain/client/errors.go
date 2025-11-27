package client

import "errors"

// Доменные ошибки для Client Domain
var (
	ErrInvalidClientID   = errors.New("invalid client ID")
	ErrClientNotFound     = errors.New("client not found")
	ErrClientNameRequired = errors.New("client name is required")
	ErrClientExists       = errors.New("client already exists")
	ErrInvalidProjectID   = errors.New("invalid project ID")
	ErrProjectNotFound    = errors.New("project not found")
	ErrProjectNameRequired = errors.New("project name is required")
	ErrProjectExists      = errors.New("project already exists")
)

