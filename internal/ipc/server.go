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

// Handler is the signature every IPC method implements.
type Handler func(ctx context.Context, params json.RawMessage) (any, *Error)

// Wire format — one JSON object per line in each direction:
//
//	→ {"id":"...","method":"foo.bar","params":{...}}
//	← {"id":"...","result":{...}}            // success
//	← {"id":"...","error":{"code","message"}} // failure
//	← {"event":"foo:bar","data":{...}}        // unsolicited event (no id)

// Server dispatches JSON-RPC requests read from in and writes responses + events to out.
type Server struct {
	in       io.Reader
	out      io.Writer
	handlers map[string]Handler
	outMu    sync.Mutex // serializes writes from many goroutines
}

// NewServer constructs an IPC server bound to the given streams.
func NewServer(in io.Reader, out io.Writer) *Server {
	return &Server{
		in:       in,
		out:      out,
		handlers: make(map[string]Handler),
	}
}

// Register binds method to handler. Not thread-safe — call before Run.
func (s *Server) Register(method string, handler Handler) {
	s.handlers[method] = handler
}

// Run reads requests until ctx is canceled or in hits EOF. Each request is dispatched on its own goroutine.
func (s *Server) Run(ctx context.Context) error {
	scanner := bufio.NewScanner(s.in)
	// samples.query can return a few thousand rows; default 64K is too small.
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

		// scanner reuses its buffer between Scan calls; copy before goroutine handoff.
		payload := make([]byte, len(line))
		copy(payload, line)

		go s.dispatch(ctx, payload)
	}

	err := scanner.Err()
	if err != nil {
		return errors.Wrap(err, "ipc scanner error")
	}

	return nil
}

// Emit sends an unsolicited event to the Tauri shell. Safe from any goroutine.
func (s *Server) Emit(event string, data any) {
	s.write(Event{Event: event, Data: data})
}

func (s *Server) dispatch(ctx context.Context, payload []byte) {
	var req Request

	err := json.Unmarshal(payload, &req)
	if err != nil {
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

func (s *Server) write(v any) {
	encoded, err := json.Marshal(v)
	if err != nil {
		log.Error().Err(err).Msg("ipc: marshal failed")
		return
	}

	s.outMu.Lock()
	defer s.outMu.Unlock()

	_, err = s.out.Write(encoded)
	if err != nil {
		log.Error().Err(err).Msg("ipc: write failed")
		return
	}

	_, err = s.out.Write([]byte{'\n'})
	if err != nil {
		log.Error().Err(err).Msg("ipc: write newline failed")
	}
}
