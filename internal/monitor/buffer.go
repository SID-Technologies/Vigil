package monitor

import (
	"sync"

	"github.com/sid-technologies/vigil/internal/probes"
)

// Sized for steady-state ~5 rows/sec × 60s flush; avoids reallocation.
const initialBufferCap = 256

// buffer is unbounded by design — a stalled flusher should surface in
// metrics, not silently drop data.
type buffer struct {
	mu      sync.Mutex
	results []probes.Result
}

func newBuffer() *buffer {
	return &buffer{results: make([]probes.Result, 0, initialBufferCap)}
}

func (b *buffer) pushMany(rs []probes.Result) {
	if len(rs) == 0 {
		return
	}

	b.mu.Lock()
	b.results = append(b.results, rs...)
	b.mu.Unlock()
}

// drain returns all buffered results and resets. Returns nil when empty.
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

// requeue prepends results so a failed bulk insert retries on the next flush.
func (b *buffer) requeue(rs []probes.Result) {
	if len(rs) == 0 {
		return
	}

	b.mu.Lock()
	b.results = append(rs, b.results...)
	b.mu.Unlock()
}

func (b *buffer) len() int {
	b.mu.Lock()
	defer b.mu.Unlock()

	return len(b.results)
}
