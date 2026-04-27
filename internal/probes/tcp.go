package probes

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"syscall"
	"time"
)

// TCPProbe times a TCP three-way handshake to host:port. Proves the actual
// service is reachable, not just the host — some ISPs drop HTTPS while
// leaving ICMP alone (or vice versa), which is why ICMP-only monitors miss
// real call/web outages.
type TCPProbe struct {
	target Target
}

func NewTCPProbe(target Target) *TCPProbe {
	return &TCPProbe{target: target}
}

func (p *TCPProbe) Target() Target { return p.target }

func (p *TCPProbe) Run(ctx context.Context, timeoutMs int) Result {
	r := Result{
		TimestampMs: nowMs(),
		Target:      p.target,
	}

	if p.target.Port == nil {
		r.Error = errPtr("missing_port")
		return r
	}

	dialCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	d := net.Dialer{}
	address := net.JoinHostPort(p.target.Host, strconv.Itoa(*p.target.Port))

	start := time.Now()
	conn, err := d.DialContext(dialCtx, "tcp", address)
	if err != nil {
		r.Error = errPtr(classifyDialError(err))
		return r
	}
	_ = conn.Close()

	r.Success = true
	r.RTTMs = floatPtr(round2(float64(time.Since(start)) / float64(time.Millisecond)))
	return r
}

// classifyDialError maps Go's network errors to stable codes the UI can
// translate. Matches the Python tool's classification labels so existing
// reports remain comparable.
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
