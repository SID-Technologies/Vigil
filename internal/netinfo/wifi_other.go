//go:build !darwin && !linux && !windows

package netinfo

import (
	"context"
	"time"
)

// SampleWifi is a no-op stub for unsupported platforms (BSD, Plan 9, etc).
// All fields stay nil; the timestamp is still populated so flushing works.
func SampleWifi(_ context.Context) WifiSample {
	return WifiSample{Timestamp: time.Now()}
}
