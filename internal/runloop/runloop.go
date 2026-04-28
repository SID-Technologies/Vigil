// Package runloop provides the loop-with-ticker scaffolding shared by
// Vigil's periodic goroutines.
package runloop

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
)

// Every runs runOnce immediately, then on every interval tick until ctx is canceled.
func Every(ctx context.Context, name string, interval time.Duration, runOnce func(context.Context)) {
	runOnce(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info().Msgf("%s: shutting down", name)
			return
		case <-ticker.C:
			runOnce(ctx)
		}
	}
}
