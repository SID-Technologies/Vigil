package probes

import (
	"context"
	"crypto/rand"
)

// RFC 1035 DNS-over-UDP constants used by UDPDNSProbe.
const (
	defaultDNSPort      = 53
	dnsTransactionIDLen = 2

	// RFC 1035 ceiling on a UDP DNS message without EDNS(0).
	dnsUDPMaxMessageSize = 512
)

// dnsExampleQueryBody is the RFC 1035 query for "example.com IN A" without
// the leading 2-byte txn id (prepended per call):
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
// for a reply with the matching txn id. example.com is cached upstream
// everywhere, so this measures round-trip rather than resolution work.
type UDPDNSProbe struct {
	target Target
}

// NewUDPDNSProbe constructs a UDPDNSProbe for the given target.
func NewUDPDNSProbe(target Target) *UDPDNSProbe {
	return &UDPDNSProbe{target: target}
}

// Target returns the probe's bound target.
func (p *UDPDNSProbe) Target() Target { return p.target }

// Run executes the UDP DNS probe.
func (p *UDPDNSProbe) Run(ctx context.Context, timeoutMs int) Result {
	port := defaultDNSPort
	if p.target.Port != nil {
		port = *p.target.Port
	}

	return dialAndMeasure(ctx, p.target, port, timeoutMs, PacketSpec{
		Network:    "udp",
		ReadBuffer: dnsUDPMaxMessageSize,
		Request:    dnsRequest,
		InvalidErr: "tid_mismatch",
	})
}

// dnsRequest builds the query and a txn-id-matching validator.
func dnsRequest() ([]byte, func([]byte) bool, string) {
	tid := make([]byte, dnsTransactionIDLen)

	_, err := rand.Read(tid)
	if err != nil {
		return nil, nil, "rand_failed"
	}

	packet := dnsQueryPacket(tid)

	validate := func(reply []byte) bool {
		return len(reply) >= dnsTransactionIDLen && reply[0] == tid[0] && reply[1] == tid[1]
	}

	return packet, validate, ""
}

// dnsQueryPacket prepends the transaction id to the static query body.
func dnsQueryPacket(tid []byte) []byte {
	return append(append([]byte{}, tid...), dnsExampleQueryBody...)
}
