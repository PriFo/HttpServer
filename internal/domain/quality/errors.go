package quality

import "errors"

// Domain-specific errors для quality domain
var (
	ErrReportNotFound      = errors.New("quality report not found")
	ErrUploadNotFound      = errors.New("upload not found")
	ErrInvalidUploadID     = errors.New("invalid upload ID")
	ErrInvalidDatabaseID   = errors.New("invalid database ID")
	ErrInvalidEntityID     = errors.New("invalid entity ID")
	ErrAnalysisFailed      = errors.New("quality analysis failed")
	ErrAnalyzerNotReady    = errors.New("quality analyzer not ready")
	ErrInvalidFilter       = errors.New("invalid quality filter")
)

