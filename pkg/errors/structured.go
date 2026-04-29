package errors

import (
	stderrors "errors"
)

type structured struct {
	err   error
	attrs []any
}

func (s structured) Error() string {
	return s.err.Error()
}

// Attrs returns the slog attributes attached to this error.
func (s structured) Attrs() []any {
	return s.attrs
}

func (s structured) Unwrap() error {
	return s.err
}

func (s structured) Is(err error) bool {
	var other structured
	if !stderrors.As(err, &other) {
		return false
	}

	return stderrors.Is(s.err, other.err)
}
