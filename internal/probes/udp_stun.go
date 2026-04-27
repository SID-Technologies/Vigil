package probes

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"net"
	"strconv"
	"time"
)

// UDPSTUNProbe sends a minimal STUN Binding Request (RFC 5389) and waits for
// a Binding Response. This is the exact UDP call-plane protocol every WebRTC
// client (Teams, Zoom, Meet, Discord) uses at call setup, so it's the
// closest DIY proxy for "will my Zoom call connect" — far better than a
// generic ICMP or HTTPS probe.
//
// Direct port of pingscraper.probes.StunUdpProbe.
type UDPSTUNProbe struct {
	target Target
}

func NewUDPSTUNProbe(target Target) *UDPSTUNProbe {
	return &UDPSTUNProbe{target: target}
}

func (p *UDPSTUNProbe) Target() Target { return p.target }

var stunMagicCookie = []byte{0x21, 0x12, 0xa4, 0x42}

func (p *UDPSTUNProbe) Run(ctx context.Context, timeoutMs int) Result {
	r := Result{
		TimestampMs: nowMs(),
		Target:      p.target,
	}

	port := 3478
	if p.target.Port != nil {
		port = *p.target.Port
	}

	transID := make([]byte, 12)
	if _, err := rand.Read(transID); err != nil {
		r.Error = errPtr("rand_failed")
		return r
	}
	packet := stunBindingRequest(transID)

	address := net.JoinHostPort(p.target.Host, strconv.Itoa(port))
	dialCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	conn, err := (&net.Dialer{}).DialContext(dialCtx, "udp", address)
	if err != nil {
		r.Error = errPtr(classifyDialError(err))
		return r
	}
	defer conn.Close()

	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	_ = conn.SetDeadline(deadline)

	start := time.Now()
	if _, err := conn.Write(packet); err != nil {
		r.Error = errPtr("write_failed")
		return r
	}

	buf := make([]byte, 1024)
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

	if !validSTUNResponse(buf[:n], transID) {
		r.Error = errPtr("malformed_response")
		return r
	}

	r.Success = true
	r.RTTMs = floatPtr(round2(float64(time.Since(start)) / float64(time.Millisecond)))
	return r
}

// stunBindingRequest builds an RFC 5389 Binding Request:
// type=0x0001, length=0, magic cookie, 96-bit transaction id.
func stunBindingRequest(transID []byte) []byte {
	hdr := make([]byte, 8, 20)
	binary.BigEndian.PutUint16(hdr[0:2], 0x0001) // Binding Request
	binary.BigEndian.PutUint16(hdr[2:4], 0)      // length = 0 (no attributes)
	copy(hdr[4:8], stunMagicCookie)
	return append(hdr, transID...)
}

// validSTUNResponse checks for a well-formed Binding Response with our
// transaction id. Doesn't parse attributes — for reachability we just need
// the matching txn id back.
func validSTUNResponse(data, transID []byte) bool {
	if len(data) < 20 {
		return false
	}
	for i := range stunMagicCookie {
		if data[4+i] != stunMagicCookie[i] {
			return false
		}
	}
	for i := range transID {
		if data[8+i] != transID[i] {
			return false
		}
	}
	return true
}
