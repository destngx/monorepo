// Package domain holds the types and rules of Health Watcher. It does no I/O.
package domain

import (
	"errors"
	"fmt"
)

// Sentinel errors. Adapters map them to HTTP statuses; wrap them with
// context using the helpers below so callers can still match them.
var (
	ErrValidation   = errors.New("validation failed")
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")
	ErrUnauthorized = errors.New("not signed in")
	ErrForbidden    = errors.New("not allowed")
	ErrRateLimited  = errors.New("too many attempts")
)

// Invalid returns a validation error with a message fit for the user.
func Invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrValidation, fmt.Sprintf(format, args...))
}

// NotFound returns a not-found error naming the missing thing.
func NotFound(what string) error {
	return fmt.Errorf("%w: %s", ErrNotFound, what)
}

// Conflict returns a conflict error with a message fit for the user.
func Conflict(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrConflict, fmt.Sprintf(format, args...))
}

// Forbidden returns a forbidden error naming the action.
func Forbidden(action string) error {
	return fmt.Errorf("%w: %s", ErrForbidden, action)
}
