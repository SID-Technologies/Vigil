package probes

import (
	"context"
	"crypto/rand"
	"net"
	"strconv"
	"time"

	"github.com/sid-technologies/vigil/internal/stats"
	"github.com/sid-technologies/vigil/pkg/errors"
)

// RFC 1035 DNS-over-UDP constants used by UDPDNSProbe.
const (
	defaultDNSPort      = 53
	dnsTransactionIDLen = 2

	// dnsUDPMaxMessageSize is the RFC 1035 ceiling on a UDP DNS message
	// without EDNS(0). 512 bytes is plenty for our reply (we only send one
	// query for example.com).
	dnsUDPMaxMessageSize = 512
)

// dnsExampleQueryBody is the RFC 1035-encoded query for "example.com IN A"
// without the leading 2-byte transaction id. The probe prepends a fresh tid
// per call.
//
//   - 0x01 0x00         flags = standard query, recursion desired
//   - 0x00 0x01         QDCOUNT = 1
//   - 0x00 0x00 ×3      ANCOUNT/NSCOUNT/ARCOUNT = 0
//   - QNAME             length-prefixed labels for "example", "com", null
//   - 0x00 0x01         QTYPE = A
//   - 0x00 0x01         QCLASS = IN
var dnsExampleQueryBody = []byte{
	0x01, 0x00,
	0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x07, 'e', 'x', 'a', 'm', 'p', 'l', 'e',
	0x03, 'c', 'o', 'm',
	0x00,
	0x00, 0x01, 0x00, 0x01,
}

// UDPDNSProbe sends one DNS query for "example.com IN A" over UDP and waits
// for a reply with the matching transaction ID. The query itself is cheap
// because example.com is cached upstream essentially everywhere — this
// probe measures the round-trip, not actual DNS resolution work.
type UDPDNSProbe struct {
	target Target
}

//nolint:revive // self-evident from signature
func NewUDPDNSProbe(target Target) *UDPDNSProbe {
	return &UDPDNSProbe{target: target}
}

//nolint:revive // self-evident from signature
func (p *UDPDNSProbe) Target() Target { return p.target }

// Run executes the UDP DNS probe.
func (p *UDPDNSProbe) Run(ctx context.Context, timeoutMs int) Result {
	r := Result{
		TimestampMs: nowMs(),
		Target:      p.target,
	}

	port := defaultDNSPort
	if p.target.Port != nil {
		port = *p.target.Port
	}

	tid := make([]byte, dnsTransactionIDLen)

	_, err := rand.Read(tid)
	if err != nil {
		r.Error = errPtr("rand_failed")
		return r
	}

	packet := dnsQueryPacket(tid)

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

	buf := make([]byte, dnsUDPMaxMessageSize)

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

	if n < dnsTransactionIDLen || buf[0] != tid[0] || buf[1] != tid[1] {
		r.Error = errPtr("tid_mismatch")
		return r
	}

	r.Success = true
	r.RTTMs = floatPtr(stats.Round2(float64(time.Since(start)) / float64(time.Millisecond)))

	return r
}

// dnsQueryPacket builds a minimal RFC 1035 query for "example.com IN A"
// by prepending the caller-supplied transaction id to the static query body.
func dnsQueryPacket(tid []byte) []byte {
	return append(append([]byte{}, tid...), dnsExampleQueryBody...)
}
