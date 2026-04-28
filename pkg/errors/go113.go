package errors

import (
	stderrors "errors"
)

// This file ensures this package is compatible with the stdlib error package.

// Is reports whether any error in err's chain matches target.
func Is(err, target error) bool { return stderrors.Is(err, target) }

// As finds the first error in err's chain that matches target, and if so, sets
// target to that error value and returns true.
func As(err error, target any) bool { return stderrors.As(err, target) }

// Unwrap returns the result of calling the Unwrap method on err, if err's
// type contains an Unwrap method returning error.
func Unwrap(err error) error {
	return stderrors.Unwrap(err)
}
