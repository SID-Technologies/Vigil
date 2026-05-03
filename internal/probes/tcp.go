package probes

import (
	"context"
	"fmt"
	"net"
	"syscall"

	"github.com/sid-technologies/vigil/pkg/errors"
)

// TCPProbe times a TCP three-way handshake. Proves service reachability,
// not just host — some ISPs drop HTTPS while leaving ICMP alone.
type TCPProbe struct {
	target Target
}

// NewTCPProbe constructs a TCPProbe for the given target.
func NewTCPProbe(target Target) *TCPProbe {
	return &TCPProbe{target: target}
}

// Target returns the probe's bound target.
func (p *TCPProbe) Target() Target { return p.target }

// Run executes the TCP probe.
func (p *TCPProbe) Run(ctx context.Context, timeoutMs int) Result {
	if p.target.Port == nil {
		return Result{
			TimestampMs: nowMs(),
			Target:      p.target,
			Error:       errPtr("missing_port"),
		}
	}

	return dialAndMeasure(ctx, p.target, *p.target.Port, timeoutMs, PacketSpec{Network: "tcp"})
}

// classifyDialError maps Go network errors to stable codes the UI can translate.
func classifyDialError(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return "dns"
	}

	if errors.Is(err, syscall.ECONNREFUSED) {
		return "conn_refused"
	}

	if errors.Is(err, syscall.EHOSTUNREACH) {
		return "host_unreachable"
	}

	if errors.Is(err, syscall.ENETUNREACH) {
		return "net_unreachable"
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "timeout"
	}

	return fmt.Sprintf("unknown:%T", err)
}
