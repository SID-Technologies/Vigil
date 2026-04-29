//nolint:testpackage // whitebox test for unexported bind / internalErr
package ipc

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/sid-technologies/vigil/pkg/errors"
)

type echoParams struct {
	Name string `json:"name"`
}

type echoResult struct {
	Greeting string `json:"greeting"`
}

func TestBind_unmarshalsParams(t *testing.T) {
	t.Parallel()

	h := bind(func(_ context.Context, p echoParams) (echoResult, *Error) {
		return echoResult{Greeting: "hi " + p.Name}, nil
	})

	out, ipcErr := h(context.Background(), json.RawMessage(`{"name":"vigil"}`))
	if ipcErr != nil {
		t.Fatalf("unexpected ipc error: %+v", ipcErr)
	}

	got, ok := out.(echoResult)
	if !ok {
		t.Fatalf("wrong result type: %T", out)
	}

	if got.Greeting != "hi vigil" {
		t.Fatalf("greeting=%q", got.Greeting)
	}
}

func TestBind_emptyParamsKeepZeroValue(t *testing.T) {
	t.Parallel()

	// Handlers that ignore params take struct{} for P and never see junk.
	called := false
	h := bind(func(_ context.Context, _ struct{}) (string, *Error) {
		called = true
		return "ok", nil
	})

	out, ipcErr := h(context.Background(), nil)
	if ipcErr != nil {
		t.Fatalf("unexpected ipc error: %+v", ipcErr)
	}

	if !called {
		t.Fatal("handler not invoked")
	}

	if out != "ok" {
		t.Fatalf("out=%v", out)
	}
}

func TestBind_invalidJSONReturnsInvalidParams(t *testing.T) {
	t.Parallel()

	h := bind(func(_ context.Context, _ echoParams) (echoResult, *Error) {
		t.Fatal("inner handler should not run on bad json")
		return echoResult{}, nil
	})

	_, ipcErr := h(context.Background(), json.RawMessage(`{not json`))
	if ipcErr == nil {
		t.Fatal("expected ipc error")
	}

	if ipcErr.Code != "invalid_params" {
		t.Fatalf("code=%q, want invalid_params", ipcErr.Code)
	}
}

// internalErr should preserve the underlying error message verbatim so
// the frontend's logs contain the actual cause.
func TestInternalErr(t *testing.T) {
	t.Parallel()

	cause := errors.New("disk on fire")

	got := internalErr(cause)
	if got.Code != "internal" {
		t.Fatalf("code=%q", got.Code)
	}

	if got.Message != "disk on fire" {
		t.Fatalf("message=%q", got.Message)
	}
}
