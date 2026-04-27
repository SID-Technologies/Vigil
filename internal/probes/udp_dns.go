package probes

import (
	"context"
	"crypto/rand"
	"errors"
	"net"
	"strconv"
	"time"
)

// UDPDNSProbe sends one DNS query for "example.com IN A" over UDP and waits
// for a reply with the matching transaction ID. The query itself is cheap
// because example.com is cached upstream essentially everywhere — this probe
// measures the round-trip, not actual DNS resolution work.
//
// Direct port of pingscraper.probes.DnsUdpProbe.
type UDPDNSProbe struct {
	target Target
}

func NewUDPDNSProbe(target Target) *UDPDNSProbe {
	return &UDPDNSProbe{target: target}
}

func (p *UDPDNSProbe) Target() Target { return p.target }

func (p *UDPDNSProbe) Run(ctx context.Context, timeoutMs int) Result {
	r := Result{
		TimestampMs: nowMs(),
		Target:      p.target,
	}

	port := 53
	if p.target.Port != nil {
		port = *p.target.Port
	}

	tid := make([]byte, 2)
	if _, err := rand.Read(tid); err != nil {
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
	defer conn.Close()

	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	_ = conn.SetDeadline(deadline)

	start := time.Now()
	if _, err := conn.Write(packet); err != nil {
		r.Error = errPtr("write_failed")
		return r
	}

	buf := make([]byte, 512)
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

	if n < 2 || buf[0] != tid[0] || buf[1] != tid[1] {
		r.Error = errPtr("tid_mismatch")
		return r
	}

	r.Success = true
	r.RTTMs = floatPtr(round2(float64(time.Since(start)) / float64(time.Millisecond)))
	return r
}

// dnsQueryPacket builds a minimal RFC 1035 query for "example.com IN A".
// 17 bytes header (12) + 5 octets QNAME + 4 trailing bytes for QTYPE+QCLASS.
func dnsQueryPacket(tid []byte) []byte {
	return append(append([]byte{}, tid...),
		// flags=0x0100 (standard query, recursion desired)
		0x01, 0x00,
		// counts: QD=1 AN=0 NS=0 AR=0
		0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		// QNAME: 7 "example" 3 "com" 0
		0x07, 'e', 'x', 'a', 'm', 'p', 'l', 'e',
		0x03, 'c', 'o', 'm',
		0x00,
		// QTYPE=A QCLASS=IN
		0x00, 0x01, 0x00, 0x01,
	)
}
