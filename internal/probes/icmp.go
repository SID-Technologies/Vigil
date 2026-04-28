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

	// Windows requires privileged mode for raw ICMP sockets; macOS/Linux
	// work unprivileged via the UDP-ICMP kernel feature.
	pinger.SetPrivileged(runtime.GOOS == "windows")

	pinger.Count = 1
	pinger.Timeout = time.Duration(timeoutMs) * time.Millisecond

	// Run blocks until the count is met or timeout fires.
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
