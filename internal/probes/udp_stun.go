package probes

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"net"
	"strconv"
	"time"

	"github.com/sid-technologies/vigil/internal/stats"
	"github.com/sid-technologies/vigil/pkg/errors"
)

// RFC 5389 STUN constants used by UDPSTUNProbe.
const (
	defaultSTUNPort      = 3478
	stunTransactionIDLen = 12 // RFC 5389 §6: 96-bit transaction id

	// stunMessageHeaderLen is the fixed Binding Request/Response header:
	// 2 bytes type + 2 bytes length + 4 bytes magic cookie + 12 bytes
	// transaction id = 20 bytes.
	stunMessageHeaderLen = 20

	// stunPreambleLen is the type+length+cookie portion (8 bytes); we
	// allocate it separately and append the transaction id.
	stunPreambleLen = 8

	// stunCookieOffset / stunTransIDOffset locate those fields inside an
	// incoming response, used by validSTUNResponse.
	stunCookieOffset  = 4
	stunTransIDOffset = 8

	// stunReadBufferSize is plenty for any plausible Binding Response;
	// real responses are well under 100 bytes.
	stunReadBufferSize = 1024

	stunMessageTypeBindingRequest uint16 = 0x0001
)

// UDPSTUNProbe sends a minimal STUN Binding Request (RFC 5389) and waits
// for a Binding Response. This is the exact UDP call-plane protocol every
// WebRTC client (Teams, Zoom, Meet, Discord) uses at call setup, so it's
// the closest DIY proxy for "will my Zoom call connect" — far better than
// a generic ICMP or HTTPS probe.
type UDPSTUNProbe struct {
	target Target
}

//nolint:revive // self-evident from signature
func NewUDPSTUNProbe(target Target) *UDPSTUNProbe {
	return &UDPSTUNProbe{target: target}
}

//nolint:revive // self-evident from signature
func (p *UDPSTUNProbe) Target() Target { return p.target }

// StunMagicCookie is the fixed RFC 5389 §6 magic cookie carried in every STUN message.
var StunMagicCookie = []byte{0x21, 0x12, 0xa4, 0x42}

// Run executes the UDP STUN probe.
func (p *UDPSTUNProbe) Run(ctx context.Context, timeoutMs int) Result {
	r := Result{
		TimestampMs: nowMs(),
		Target:      p.target,
	}

	port := defaultSTUNPort
	if p.target.Port != nil {
		port = *p.target.Port
	}

	transID := make([]byte, stunTransactionIDLen)

	_, err := rand.Read(transID)
	if err != nil {
		r.Error = errPtr("rand_failed")
		return r
	}

	packet := StunBindingRequest(transID)

	address := net.JoinHostPort(p.target.Host, strconv.Itoa(port))

	dialCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	conn, err := (&net.Dialer{}).DialContext(dialCtx, "udp", address)
	if err != nil {
		r.Error = errPtr(classifyDialError(err))
		return r
	}

	defer func() { _ = conn.Close() }()

	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	_ = conn.SetDeadline(deadline)

	start := time.Now()

	_, err = conn.Write(packet)
	if err != nil {
		r.Error = errPtr("write_failed")
		return r
	}

	buf := make([]byte, stunReadBufferSize)

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

	if !ValidSTUNResponse(buf[:n], transID) {
		r.Error = errPtr("malformed_response")
		return r
	}

	r.Success = true
	r.RTTMs = floatPtr(stats.Round2(float64(time.Since(start)) / float64(time.Millisecond)))

	return r
}

// StunBindingRequest builds an RFC 5389 Binding Request: type=0x0001,
// length=0, magic cookie, 96-bit transaction id.
func StunBindingRequest(transID []byte) []byte {
	hdr := make([]byte, stunPreambleLen, stunMessageHeaderLen)
	binary.BigEndian.PutUint16(hdr[0:2], stunMessageTypeBindingRequest)
	binary.BigEndian.PutUint16(hdr[2:4], 0) // length = 0 (no attributes)
	copy(hdr[stunCookieOffset:stunPreambleLen], StunMagicCookie)

	return append(hdr, transID...)
}

// ValidSTUNResponse reports whether `data` is a well-formed Binding
// Response carrying our transaction id. We don't parse attributes — for
// reachability we just need the matching txn id back.
func ValidSTUNResponse(data, transID []byte) bool {
	if len(data) < stunMessageHeaderLen {
		return false
	}

	for i := range StunMagicCookie {
		if data[stunCookieOffset+i] != StunMagicCookie[i] {
			return false
		}
	}

	for i := range transID {
		if data[stunTransIDOffset+i] != transID[i] {
			return false
		}
	}

	return true
}
