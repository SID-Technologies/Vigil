// Package errors is a drop-in replacement for the stdlib errors package
// that carries slog-style structured attributes alongside each error.
package errors

import (
	"fmt"

	pkgerrors "github.com/pkg/errors"
)

// New returns an error formatted via fmt.Sprintf with the attached attrs.
func New(msg string, attrs ...any) error {
	formatted := fmt.Sprintf(msg, attrs...)

	return structured{
		err:   pkgerrors.New(formatted),
		attrs: attrs,
	}
}

// Wrap returns err wrapped with msg and additional structured attrs.
//
//nolint:inamedparam // attrs... is variadic; named return is intentional.
func Wrap(err error, msg string, attrs ...any) error {
	if err == nil {
		panic("wrap nil error")
	}

	if wrapper, ok := err.(interface{ Wrap(string, ...any) error }); ok {
		return wrapper.Wrap(msg, attrs...)
	}

	var inner structured
	if As(err, &inner) {
		attrs = append(attrs, inner.attrs...)
	}

	return structured{
		err:   pkgerrors.Wrap(err, msg),
		attrs: attrs,
	}
}
