package entity

import (
	"errors"
	"fmt"
)

// ErrValidation is the sentinel returned for malformed search requests. The
// delivery layer maps it to an HTTP 400 response.
var ErrValidation = errors.New("validation error")

// ValidationError is a user friendly validation failure that unwraps to
// ErrValidation.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

func (e *ValidationError) Unwrap() error { return ErrValidation }

// NewValidationError builds a ValidationError with a formatted message.
func NewValidationError(format string, args ...any) error {
	return &ValidationError{Message: fmt.Sprintf(format, args...)}
}
