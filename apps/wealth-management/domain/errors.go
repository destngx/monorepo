package domain

import "errors"

// Sentinel errors that adapters map to transport status codes
// (HTTP 400/404/409, MCP isError results).
var (
	ErrValidation = errors.New("validation failed")
	ErrNotFound   = errors.New("not found")
	ErrConflict   = errors.New("conflict")
)
