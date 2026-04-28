package errors

import (
	stderrors "errors"
)

// Is reports whether any error in err's chain matches target.
func Is(err, target error) bool { return stderrors.Is(err, target) }

// As unwraps err looking for a value matching target.
func As(err error, target any) bool { return stderrors.As(err, target) }

// Unwrap returns the result of calling the Unwrap method on err.
func Unwrap(err error) error {
	return stderrors.Unwrap(err)
}
