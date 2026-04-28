// Package probes contains the four probe implementations Vigil uses to test network reachability.
package probes

import (
	"context"
	"time"
)

// Kind enumerates the supported probe types.
type Kind string

// Probe Kind values, mirroring the Target.kind enum.
const (
	KindICMP    Kind = "icmp"
	KindTCP     Kind = "tcp"
	KindUDPDNS  Kind = "udp_dns"
	KindUDPSTUN Kind = "udp_stun"
)

// Target captures the information needed to construct and run a probe.
type Target struct {
	Label string `json:"label"`
	Kind  Kind   `json:"kind"`
	Host  string `json:"host"`
	Port  *int   `json:"port,omitempty"` // nil for ICMP; required for the rest
}

// Result is the outcome of a single probe execution. Error is a short stable
// machine code (e.g. "timeout", "host_unreachable"), never a raw OS string.
type Result struct {
	TimestampMs int64    `json:"ts_unix_ms"`
	Target      Target   `json:"target"`
	Success     bool     `json:"success"`
	RTTMs       *float64 `json:"rtt_ms,omitempty"`
	Error       *string  `json:"error,omitempty"`
}

// Probe runs a single check against its bound target. Implementations must
// return within timeoutMs, never panic, and be safe to call concurrently.
type Probe interface {
	Run(ctx context.Context, timeoutMs int) Result
	Target() Target
}

func nowMs() int64 {
	return time.Now().UnixMilli()
}

func errPtr(s string) *string { return &s }
func floatPtr(f float64) *float64 {
	return &f
}
