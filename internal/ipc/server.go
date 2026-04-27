package ipc

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/sid-technologies/vigil/pkg/errors"
)

// Handler is the signature every IPC method implements. Returns a JSON-
// serializable result on success, or an *Error on failure. Returning a plain
// Go error is converted to Error{Code: "internal", Message: err.Error()}.
type Handler func(ctx context.Context, params json.RawMessage) (any, *Error)

// Server reads requests line-by-line from `in`, dispatches to registered
// Handlers, and writes responses + events line-by-line to `out`.
//
// The protocol is deliberately simple: one JSON object per line. There's no
// framing header, no batching, no streaming. If a method needs to push
// progress, it does so via Emit() between request handlers.
type Server struct {
	in       io.Reader
	out      io.Writer
	handlers map[string]Handler
	outMu    sync.Mutex // serializes writes to out (events from many goroutines)
}

// NewServer constructs an IPC server bound to the given streams.
// Typical wiring: NewServer(os.Stdin, os.Stdout).
func NewServer(in io.Reader, out io.Writer) *Server {
	return &Server{
		in:       in,
		out:      out,
		handlers: make(map[string]Handler),
	}
}

// Register binds a method name to a handler. Calling Register with the same
// name twice replaces the previous handler. Not thread-safe — call all
// Register()s before Run().
func (s *Server) Register(method string, handler Handler) {
	s.handlers[method] = handler
}

// Run blocks reading from `in` until ctx is cancelled or the input stream
// closes (e.g. parent Tauri shell exits). Each request is dispatched on a
// goroutine so a slow handler can't block subsequent requests.
//
// Returns nil on clean EOF, or a wrapped error on read failure.
func (s *Server) Run(ctx context.Context) error {
	scanner := bufio.NewScanner(s.in)
	// IPC payloads can be larger than the default 64K (e.g. samples.query
	// returning a few thousand rows). Bump the buffer ceiling.
	const maxIPCLine = 8 * 1024 * 1024
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, maxIPCLine)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		// Copy the line because the scanner reuses its buffer between Scan calls.
		payload := make([]byte, len(line))
		copy(payload, line)

		go s.dispatch(ctx, payload)
	}

	if err := scanner.Err(); err != nil {
		return errors.Wrap(err, "ipc scanner error")
	}
	return nil
}

func (s *Server) dispatch(ctx context.Context, payload []byte) {
	var req Request
	if err := json.Unmarshal(payload, &req); err != nil {
		log.Warn().Err(err).Msg("ipc: malformed request, dropping")
		return
	}

	handler, ok := s.handlers[req.Method]
	if !ok {
		s.writeResponse(Response{
			ID: req.ID,
			Error: &Error{
				Code:    "method_not_found",
				Message: "unknown method: " + req.Method,
			},
		})
		return
	}

	result, ipcErr := handler(ctx, req.Params)
	if ipcErr != nil {
		s.writeResponse(Response{ID: req.ID, Error: ipcErr})
		return
	}
	s.writeResponse(Response{ID: req.ID, Result: result})
}

func (s *Server) writeResponse(resp Response) {
	s.write(resp)
}

// Emit sends an unsolicited event to the Tauri shell. Safe to call from any
// goroutine. Errors writing to stdout are logged but not surfaced — if the
// Tauri shell has gone away, the next read in Run() will return EOF and the
// sidecar will shut down on its own.
func (s *Server) Emit(event string, data any) {
	s.write(Event{Event: event, Data: data})
}

func (s *Server) write(v any) {
	encoded, err := json.Marshal(v)
	if err != nil {
		log.Error().Err(err).Msg("ipc: marshal failed")
		return
	}

	s.outMu.Lock()
	defer s.outMu.Unlock()

	if _, err := s.out.Write(encoded); err != nil {
		log.Error().Err(err).Msg("ipc: write failed")
		return
	}
	if _, err := s.out.Write([]byte{'\n'}); err != nil {
		log.Error().Err(err).Msg("ipc: write newline failed")
	}
}
