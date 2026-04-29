package probes_test

import (
	"testing"

	"github.com/sid-technologies/vigil/internal/probes"
)

func TestBuild_dispatchesByKind(t *testing.T) {
	t.Parallel()

	port := 53

	cases := []struct {
		name   string
		target probes.Target
		typ    string
	}{
		{"icmp", probes.Target{Label: "a", Kind: probes.KindICMP, Host: "1.1.1.1"}, "*probes.ICMPProbe"},
		{"tcp", probes.Target{Label: "b", Kind: probes.KindTCP, Host: "example.com", Port: &port}, "*probes.TCPProbe"},
		{"udp_dns", probes.Target{Label: "c", Kind: probes.KindUDPDNS, Host: "1.1.1.1", Port: &port}, "*probes.UDPDNSProbe"},
		{"udp_stun", probes.Target{Label: "d", Kind: probes.KindUDPSTUN, Host: "stun.l.google.com", Port: &port}, "*probes.UDPSTUNProbe"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			probe, err := probes.Build(tc.target)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if probe == nil {
				t.Fatal("nil probe")
			}

			if probe.Target() != tc.target {
				t.Fatalf("target round-trip lost data: got %+v want %+v", probe.Target(), tc.target)
			}
		})
	}
}

func TestBuild_unknownKind(t *testing.T) {
	t.Parallel()

	_, err := probes.Build(probes.Target{Label: "x", Kind: "carrier_pigeon", Host: "h"})
	if err == nil {
		t.Fatal("expected error for unknown kind")
	}
}
