package probes

import (
	"context"
	"errors"
	"net"
	"runtime"
	"time"

	probing "github.com/prometheus-community/pro-bing"
)

// ICMPProbe sends one ICMP echo request and waits for a reply. Uses
// pro-bing, which:
//   - On Linux: uses unprivileged UDP-mode ICMP (no root needed when
//     net.ipv4.ping_group_range is configured, the default on Ubuntu/Fedora).
//   - On Windows: uses real ICMP sockets (no admin needed).
//   - On macOS: uses unprivileged UDP-mode ICMP via the kernel's special
//     socket type — works without sudo.
//
// We deliberately do not shell out to `ping`. The legacy Python tool did,
// which broke on localized Windows (`ping` output is translated) and added
// ~100ms of subprocess startup per probe.
type ICMPProbe struct {
	target Target
}

func NewICMPProbe(target Target) *ICMPProbe {
	return &ICMPProbe{target: target}
}

func (p *ICMPProbe) Target() Target { return p.target }

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

	// Windows requires privileged mode for raw ICMP sockets; macOS/Linux
	// work unprivileged via the UDP-ICMP kernel feature.
	pinger.SetPrivileged(runtime.GOOS == "windows")

	pinger.Count = 1
	pinger.Timeout = time.Duration(timeoutMs) * time.Millisecond

	// Run blocks until the count is met or timeout fires.
	if err := pinger.RunWithContext(ctx); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			r.Error = errPtr("timeout")
			return r
		}
		r.Error = errPtr("send_failed")
		return r
	}

	stats := pinger.Statistics()
	if stats.PacketsRecv == 0 {
		r.Error = errPtr("timeout")
		return r
	}

	r.Success = true
	rttMs := float64(stats.AvgRtt) / float64(time.Millisecond)
	r.RTTMs = floatPtr(round2(rttMs))
	return r
}

// round2 rounds to 2 decimal places — matches the Python tool's output so
// existing dashboards / CSV consumers see identical values.
func round2(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100.0
}
