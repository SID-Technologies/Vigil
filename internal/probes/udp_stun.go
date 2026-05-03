package probes

import (
	"context"
	"crypto/rand"
	"encoding/binary"
)

// RFC 5389 STUN constants. Header layout: 2 type + 2 length + 4 magic cookie
// + 12 transaction id = 20 bytes.
const (
	defaultSTUNPort      = 3478
	stunTransactionIDLen = 12
	stunMessageHeaderLen = 20
	stunPreambleLen      = 8 // type+length+cookie, txn id appended separately
	stunCookieOffset     = 4
	stunTransIDOffset    = 8
	stunReadBufferSize   = 1024

	stunMessageTypeBindingRequest uint16 = 0x0001
)

// UDPSTUNProbe sends a STUN Binding Request (RFC 5389) and waits for a
// Binding Response. STUN over UDP is the same call-setup protocol Teams,
// Zoom, Meet, and Discord use, making it a far better proxy for "will my
// call connect" than ICMP or HTTPS.
type UDPSTUNProbe struct {
	target Target
}

// NewUDPSTUNProbe constructs a UDPSTUNProbe for the given target.
func NewUDPSTUNProbe(target Target) *UDPSTUNProbe {
	return &UDPSTUNProbe{target: target}
}

// Target returns the probe's bound target.
func (p *UDPSTUNProbe) Target() Target { return p.target }

// StunMagicCookie is the RFC 5389 §6 magic cookie carried in every STUN message.
var StunMagicCookie = []byte{0x21, 0x12, 0xa4, 0x42}

// Run executes the UDP STUN probe.
func (p *UDPSTUNProbe) Run(ctx context.Context, timeoutMs int) Result {
	port := defaultSTUNPort
	if p.target.Port != nil {
		port = *p.target.Port
	}

	return dialAndMeasure(ctx, p.target, port, timeoutMs, PacketSpec{
		Network:    "udp",
		ReadBuffer: stunReadBufferSize,
		Request:    stunRequest,
		InvalidErr: "malformed_response",
	})
}

// stunRequest builds the binding request and a txn-id-matching validator.
func stunRequest() ([]byte, func([]byte) bool, string) {
	transID := make([]byte, stunTransactionIDLen)

	_, err := rand.Read(transID)
	if err != nil {
		return nil, nil, "rand_failed"
	}

	packet := StunBindingRequest(transID)

	validate := func(reply []byte) bool {
		return ValidSTUNResponse(reply, transID)
	}

	return packet, validate, ""
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

// ValidSTUNResponse reports whether data is a Binding Response carrying our
// txn id. Attributes are not parsed; reachability only needs txn id match.
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
