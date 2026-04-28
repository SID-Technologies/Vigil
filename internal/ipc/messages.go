// Package ipc implements the stdio JSON-RPC protocol between the Tauri shell and the sidecar.
package ipc

import "encoding/json"

// Request is an inbound method call from the Tauri shell.
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

// Event is an unsolicited message from the sidecar (no id, distinguishes from Response).
//
// Tauri 2 event names: alphanumerics + `-/:_` only. Use `:` for namespacing; `.` silently drops.
type Event struct {
	Event string `json:"event"`
	Data  any    `json:"data,omitempty"`
}

// Error is an RPC-style structured error.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// HealthCheckResult is the payload returned by health.check.
type HealthCheckResult struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	Commit  string `json:"commit,omitempty"`
}
