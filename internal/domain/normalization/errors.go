package normalization

import "errors"

// Domain-specific errors для normalization domain
var (
	ErrProcessNotFound      = errors.New("normalization process not found")
	ErrProcessAlreadyRunning = errors.New("normalization process already running")
	ErrProcessNotRunning    = errors.New("normalization process is not running")
	ErrInvalidProcessID     = errors.New("invalid process ID")
	ErrInvalidUploadID      = errors.New("invalid upload ID")
	ErrSessionNotFound      = errors.New("normalization session not found")
	ErrInvalidSessionID     = errors.New("invalid session ID")
	ErrNormalizationFailed  = errors.New("normalization failed")
	ErrInvalidEntityType    = errors.New("invalid entity type")
)

