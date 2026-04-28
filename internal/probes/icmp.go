package probes

import (
	"context"
	"net"
	"runtime"
	"time"

	"github.com/sid-technologies/vigil/internal/stats"
	"github.com/sid-technologies/vigil/pkg/errors"

	probing "github.com/prometheus-community/pro-bing"
)

// ICMPProbe sends one ICMP echo request via pro-bing. Unprivileged on
// Linux/macOS (UDP-mode ICMP) and Windows (real ICMP socket); never shells
// out to `ping`, which broke on localized Windows in the predecessor tool.
type ICMPProbe struct {
	target Target
}

//nolint:revive // self-evident from signature
func NewICMPProbe(target Target) *ICMPProbe {
	return &ICMPProbe{target: target}
}

//nolint:revive // self-evident from signature
func (p *ICMPProbe) Target() Target { return p.target }

// Run executes the ICMP probe.
func (p *ICMPProbe) Run(ctx context.Context, timeoutMs int) Result {
	r := Result{
		TimestampMs: nowMs(),
		Target:      p.target,
	}

	pinger, err := probing.NewPinger(p.target.Host)
	if err != nil {
		var dnsErr *net.DNSError
		if errors.As(err, &dnsErr) {
			r.Error = errPtr("dns")
			return r
		}

		r.Error = errPtr("init_failed")

		return r
	}

	// Windows raw ICMP needs privileged mode; macOS/Linux do not.
	pinger.SetPrivileged(runtime.GOOS == "windows")

	pinger.Count = 1
	pinger.Timeout = time.Duration(timeoutMs) * time.Millisecond

	err = pinger.RunWithContext(ctx)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			r.Error = errPtr("timeout")
			return r
		}

		r.Error = errPtr("send_failed")

		return r
	}

	pingStats := pinger.Statistics()
	if pingStats.PacketsRecv == 0 {
		r.Error = errPtr("timeout")
		return r
	}

	r.Success = true
	rttMs := float64(pingStats.AvgRtt) / float64(time.Millisecond)
	r.RTTMs = floatPtr(stats.Round2(rttMs))

	return r
}
