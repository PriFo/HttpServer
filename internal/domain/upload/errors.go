package upload

import "errors"

// Domain-specific errors для upload domain
var (
	ErrUploadNotFound      = errors.New("upload not found")
	ErrInvalidUUID         = errors.New("invalid upload UUID")
	ErrInvalidVersion      = errors.New("invalid 1C version")
	ErrInvalidConfigName   = errors.New("invalid config name")
	ErrUploadAlreadyExists = errors.New("upload already exists")
	ErrUploadCompleted     = errors.New("upload already completed")
	ErrUploadFailed        = errors.New("upload failed")
	ErrInvalidStatus       = errors.New("invalid upload status")
)

