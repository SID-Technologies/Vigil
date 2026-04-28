// Package probes contains the four probe implementations Vigil uses to
// test network reachability. Each probe is stateless, runs once per
// monitor cycle, and returns a Result. ICMP uses raw ICMP via pro-bing
// (no shelling out to the system `ping`); TCP/UDP probes are pure Go
// socket code, so cross-compilation stays clean.
package probes

import (
	"context"
	"time"
)

// Kind enumerates the supported probe types. Mirrors the schema enum on
// Target.kind in db/ent/schema/target.go.
type Kind string

// Probe Kind values, mirroring the Target.kind enum.
const (
	KindICMP    Kind = "icmp"
	KindTCP     Kind = "tcp"
	KindUDPDNS  Kind = "udp_dns"
	KindUDPSTUN Kind = "udp_stun"
)

// Target captures the information needed to construct and run a probe.
// Decoupled from the Ent entity so probes can be unit-tested without a DB.
//
// JSON tags are present so the wire format used in probe:cycle events is
// snake_case, consistent with the rest of the IPC schema.
type Target struct {
	Label string `json:"label"`
	Kind  Kind   `json:"kind"`
	Host  string `json:"host"`
	Port  *int   `json:"port,omitempty"` // nil for ICMP; required for the rest
}

// Result is the outcome of a single probe execution. RTTMs is set only when
// Success is true. Error is a short stable machine code (e.g. "timeout",
// "host_unreachable") that the UI can map to user-friendly text — never a
// raw OS error string.
type Result struct {
	TimestampMs int64    `json:"ts_unix_ms"`
	Target      Target   `json:"target"`
	Success     bool     `json:"success"`
	RTTMs       *float64 `json:"rtt_ms,omitempty"`
	Error       *string  `json:"error,omitempty"`
}

// Probe runs a single check against its bound target. Implementations must:
//   - Always return within `timeoutMs` (no surprise blocking).
//   - Never panic — return Result{Success:false, Error:"..."} instead.
//   - Be safe to call concurrently with other Probe.Run calls (the monitor
//     fires all probes in parallel).
type Probe interface {
	Run(ctx context.Context, timeoutMs int) Result
	Target() Target
}

// nowMs returns the current wall time in milliseconds since the unix epoch.
// Centralized so tests can monkeypatch via a build tag if ever needed.
func nowMs() int64 {
	return time.Now().UnixMilli()
}

// errPtr / floatPtr / intPtr — small allocation helpers, kept here so probe
// files don't repeat them.
func errPtr(s string) *string { return &s }
func floatPtr(f float64) *float64 {
	return &f
}
