package errors

import (
	pkgerrors "github.com/pkg/errors"
)

type structured struct {
	err   error
	attrs []any
}

// StackTrace implements pkgerrors.StackTracer.
func (s structured) StackTrace() pkgerrors.StackTrace {
	type stackTracer interface {
		StackTrace() pkgerrors.StackTrace
	}

	tracer, ok := s.err.(stackTracer)
	if !ok {
		return nil
	}

	trace := tracer.StackTrace()

	// Trim this package's frame and the runtime frame from the trace.
	return trace[1 : len(trace)-1]
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
	if !pkgerrors.As(err, &other) {
		return false
	}

	return pkgerrors.Is(s.err, other.err)
}
