//nolint:testpackage // whitebox tests for unexported buffer helpers
package monitor

import (
	"sync"
	"testing"

	"github.com/sid-technologies/vigil/internal/probes"
)

func sample(label string) probes.Result {
	return probes.Result{
		Target: probes.Target{Label: label, Kind: probes.KindICMP, Host: "1.1.1.1"},
	}
}

func TestBuffer_pushDrain(t *testing.T) {
	t.Parallel()

	b := newBuffer()

	if b.len() != 0 {
		t.Fatal("new buffer should be empty")
	}

	b.pushMany([]probes.Result{sample("a"), sample("b")})

	if b.len() != 2 {
		t.Fatalf("len=%d, want 2", b.len())
	}

	out := b.drain()
	if len(out) != 2 {
		t.Fatalf("drain returned %d, want 2", len(out))
	}

	if b.len() != 0 {
		t.Fatal("drain should reset the buffer")
	}

	if got := b.drain(); got != nil {
		t.Fatalf("empty drain should return nil, got %v", got)
	}
}

func TestBuffer_pushManyEmpty(t *testing.T) {
	t.Parallel()

	b := newBuffer()
	b.pushMany(nil)
	b.pushMany([]probes.Result{})

	if b.len() != 0 {
		t.Fatal("empty pushMany should be a no-op")
	}
}

// requeue prepends so a failed flush retries oldest-first on the next sweep.
func TestBuffer_requeuePrepends(t *testing.T) {
	t.Parallel()

	b := newBuffer()
	b.pushMany([]probes.Result{sample("new1"), sample("new2")})
	b.requeue([]probes.Result{sample("old1"), sample("old2")})

	out := b.drain()
	if len(out) != 4 {
		t.Fatalf("len=%d", len(out))
	}

	if out[0].Target.Label != "old1" || out[1].Target.Label != "old2" {
		t.Fatalf("requeue did not prepend: got order %s,%s,%s,%s",
			out[0].Target.Label, out[1].Target.Label, out[2].Target.Label, out[3].Target.Label)
	}
}

// Concurrent pushes shouldn't lose data or race the detector.
func TestBuffer_concurrentPush(t *testing.T) {
	t.Parallel()

	b := newBuffer()

	const (
		writers   = 8
		perWriter = 100
	)

	var wg sync.WaitGroup

	wg.Add(writers)

	for range writers {
		go func() {
			defer wg.Done()

			for range perWriter {
				b.pushMany([]probes.Result{sample("x")})
			}
		}()
	}

	wg.Wait()

	if b.len() != writers*perWriter {
		t.Fatalf("len=%d, want %d", b.len(), writers*perWriter)
	}
}
