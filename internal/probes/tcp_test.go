//nolint:testpackage // whitebox test for unexported classifyDialError
package probes

import (
	"context"
	"net"
	"syscall"
	"testing"

	"github.com/sid-technologies/vigil/pkg/errors"
)

// Tests live in-package so they can exercise the unexported classifyDialError.
func TestClassifyDialError(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		want string
	}{
		{"deadline", context.DeadlineExceeded, "timeout"},
		{"dns", &net.DNSError{Err: "no such host", IsNotFound: true}, "dns"},
		{"refused", syscall.ECONNREFUSED, "conn_refused"},
		{"host_unreachable", syscall.EHOSTUNREACH, "host_unreachable"},
		{"net_unreachable", syscall.ENETUNREACH, "net_unreachable"},
		{"timeout_via_net_error", &timeoutError{}, "timeout"},
		{"unknown", errors.New("kaboom"), "unknown:errors.structured"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := classifyDialError(tc.err)
			if got != tc.want {
				t.Fatalf("classifyDialError(%v) = %q, want %q", tc.err, got, tc.want)
			}
		})
	}
}

// TCP probe with a missing port should fail fast with missing_port, not panic.
func TestTCPProbe_missingPort(t *testing.T) {
	t.Parallel()

	probe := NewTCPProbe(Target{Label: "x", Kind: KindTCP, Host: "127.0.0.1"})
	res := probe.Run(context.Background(), 100)

	if res.Success {
		t.Fatal("expected failure")
	}

	if res.Error == nil || *res.Error != "missing_port" {
		t.Fatalf("expected missing_port, got %v", res.Error)
	}
}

// timeoutError is a minimal net.Error that reports Timeout()=true.
type timeoutError struct{}

func (*timeoutError) Error() string   { return "i/o timeout" }
func (*timeoutError) Timeout() bool   { return true }
func (*timeoutError) Temporary() bool { return true }
