package db

import "errors"

var (
	// ErrNotFound indicates no rows matched the query.
	ErrNotFound = errors.New("not found")
	// ErrConflict indicates a uniqueness or integrity conflict.
	ErrConflict = errors.New("conflict")
	// ErrValidation indicates inputs failed validation.
	ErrValidation = errors.New("validation error")
)
