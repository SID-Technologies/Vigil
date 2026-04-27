// Package ipc implements the stdio JSON-RPC protocol used between the Tauri
// shell and the Vigil sidecar.
//
// Wire format: one JSON object per line on stdin (sidecar reads requests) and
// stdout (sidecar writes responses + events). The Tauri Rust bridge reads
// stdout line by line, dispatches by message kind, and forwards events to the
// frontend via tauri.emit.
package ipc

import "encoding/json"

// Request is an inbound method call from the Tauri shell.
//
// `id` is opaque to the sidecar; we echo it back on the response so the Tauri
// side can route to the right pending promise. `method` is dotted ("samples.query").
// `params` is method-specific JSON; handlers unmarshal to their own types.
type Request struct {
	ID     string          `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

// Response is the reply to a Request. Exactly one of Result or Error is set.
type Response struct {
	ID     string `json:"id"`
	Result any    `json:"result,omitempty"`
	Error  *Error `json:"error,omitempty"`
}

// Event is an unsolicited message from the sidecar. The Tauri shell forwards
// these to the frontend as Tauri events under the same name.
//
// Distinguished from Response by the absence of an `id` field.
//
// IMPORTANT: Tauri 2 validates event names — only alphanumerics + the
// separators `-`, `/`, `:`, `_` are allowed. Use `:` for namespacing
// (e.g. `probe:cycle`, `outage:start`). NEVER use `.` — it will silently
// fail to deliver to frontend listeners.
type Event struct {
	Event string `json:"event"`
	Data  any    `json:"data,omitempty"`
}

// Error is an RPC-style structured error. `code` is a stable machine-readable
// string (e.g. "not_found", "invalid_params"). `message` is human-readable.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// HealthCheckResult is the payload returned by the health.check method.
// Used by the Tauri shell on startup to verify the sidecar is alive.
type HealthCheckResult struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	Commit  string `json:"commit,omitempty"`
}
