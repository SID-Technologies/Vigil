package monitor

import (
	"sync"

	"github.com/sid-technologies/vigil/internal/probes"
)

// initialBufferCap is the starting slice capacity for a fresh buffer. Sized
// for the steady-state load (~5 rows/sec × 60s flush = 300 rows worst-case)
// with a little headroom — keeps the slice from reallocating during the
// common path.
const initialBufferCap = 256

// buffer is an unbounded in-memory queue of probe results waiting to be
// flushed to the database. The probe loop pushes; the flusher drains.
//
// Bounded would be safer in theory, but Vigil's steady-state write rate is
// ~5 rows/second and the flusher runs every 60s — worst case 300 rows in
// flight, ~50KB. If the flusher gets stuck, an unbounded buffer lets us see
// the problem in metrics rather than silently dropping data.
type buffer struct {
	mu      sync.Mutex
	results []probes.Result
}

func newBuffer() *buffer {
	return &buffer{results: make([]probes.Result, 0, initialBufferCap)}
}

// pushMany appends a batch.
func (b *buffer) pushMany(rs []probes.Result) {
	if len(rs) == 0 {
		return
	}

	b.mu.Lock()
	b.results = append(b.results, rs...)
	b.mu.Unlock()
}

// drain returns all currently buffered results and clears the buffer.
// Returns nil (not empty slice) when there's nothing to flush.
func (b *buffer) drain() []probes.Result {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.results) == 0 {
		return nil
	}

	out := b.results
	b.results = make([]probes.Result, 0, initialBufferCap)

	return out
}

// requeue puts results back at the front — used when the flusher's bulk
// insert fails so we don't drop data on transient DB errors.
func (b *buffer) requeue(rs []probes.Result) {
	if len(rs) == 0 {
		return
	}

	b.mu.Lock()
	b.results = append(rs, b.results...)
	b.mu.Unlock()
}

// len returns the current buffer depth (for diagnostics / metrics).
func (b *buffer) len() int {
	b.mu.Lock()
	defer b.mu.Unlock()

	return len(b.results)
}
