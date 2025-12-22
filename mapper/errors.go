package mapper

import "errors"

var (
	// ErrNilDB is returned when the provided database handle is nil.
	ErrNilDB = errors.New("db is nil")
)
