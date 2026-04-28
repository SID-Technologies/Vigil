package probes

import (
	"context"
	"net"
	"strconv"
	"time"

	"github.com/sid-technologies/vigil/internal/stats"
	"github.com/sid-technologies/vigil/pkg/errors"
)

// PacketSpec configures dialAndMeasure for one probe type. Request nil = dial-only (TCP).
type PacketSpec struct {
	Network    string
	ReadBuffer int
	Request    func() (payload []byte, validate func(reply []byte) bool, errCode string)
	InvalidErr string
}

// dialAndMeasure runs the connect-write-read-time loop shared by TCP, UDP-DNS, and UDP-STUN probes.
func dialAndMeasure(ctx context.Context, target Target, port int, timeoutMs int, spec PacketSpec) Result {
	r := Result{TimestampMs: nowMs(), Target: target}

	var (
		payload  []byte
		validate func([]byte) bool
	)

	if spec.Request != nil {
		p, v, errCode := spec.Request()
		if errCode != "" {
			r.Error = errPtr(errCode)
			return r
		}

		payload, validate = p, v
	}

	address := net.JoinHostPort(target.Host, strconv.Itoa(port))

	dialCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	start := time.Now()

	conn, err := (&net.Dialer{}).DialContext(dialCtx, spec.Network, address)
	if err != nil {
		r.Error = errPtr(classifyDialError(err))
		return r
	}

	defer func() { _ = conn.Close() }()

	// Dial-only path (TCP): connect success is the measurement.
	if spec.Request == nil {
		r.Success = true
		r.RTTMs = floatPtr(stats.Round2(float64(time.Since(start)) / float64(time.Millisecond)))

		return r
	}

	_ = conn.SetDeadline(time.Now().Add(time.Duration(timeoutMs) * time.Millisecond))

	// RTT excludes dial setup for write-then-read probes; matches prior behavior.
	start = time.Now()

	_, err = conn.Write(payload)
	if err != nil {
		r.Error = errPtr("write_failed")
		return r
	}

	buf := make([]byte, spec.ReadBuffer)

	n, err := conn.Read(buf)
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			r.Error = errPtr("timeout")
			return r
		}

		r.Error = errPtr("read_failed")

		return r
	}

	if validate != nil && !validate(buf[:n]) {
		r.Error = errPtr(spec.InvalidErr)
		return r
	}

	r.Success = true
	r.RTTMs = floatPtr(stats.Round2(float64(time.Since(start)) / float64(time.Millisecond)))

	return r
}
