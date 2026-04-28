package probes

// Well-known ports used by the default targets list.
const (
	HTTPSPort           = 443
	GoogleSTUNAltPort   = 19302 // stun.l.google.com listens on 19302, not 3478
)

// DefaultTargets are the 12 builtin probe targets seeded into the database
// on first run. The monitor adds a 13th at runtime (`router_icmp`, pointed
// at the detected default gateway), which isn't seeded because the gateway
// can change across networks and DHCP renewals.
//
// The selection is deliberately stakeholder-proof: hostile counterparties
// (ISPs, property managers) can't wave away the evidence as "Google's
// problem" because the probes hit:
//
//   - Real video-call hostnames (teams.microsoft.com, zoom.us,
//     outlook.office.com) over both ICMP and TCP:443.
//   - Public anycast DNS (8.8.8.8, 1.1.1.1) over UDP:53 — actual DNS
//     traffic, not just reachability.
//   - Public STUN servers — the exact UDP protocol Teams / Zoom / Meet /
//     Discord use at call setup.
//
// All 13 are inserted with is_builtin=true. Users can disable individual
// builtins or add custom targets through the UI.
func DefaultTargets() []Target {
	port443 := HTTPSPort
	port53 := defaultDNSPort
	port3478 := defaultSTUNPort
	portStunGoogle := GoogleSTUNAltPort

	return []Target{
		// ICMP — network-layer reachability to anycast and the real video-call hostnames.
		{Label: "google_dns_icmp", Kind: KindICMP, Host: "8.8.8.8"},
		{Label: "cloudflare_dns_icmp", Kind: KindICMP, Host: "1.1.1.1"},
		{Label: "teams_icmp", Kind: KindICMP, Host: "teams.microsoft.com"},
		{Label: "zoom_icmp", Kind: KindICMP, Host: "zoom.us"},
		{Label: "outlook_icmp", Kind: KindICMP, Host: "outlook.office.com"},

		// TCP :443 — some ISPs drop HTTPS while leaving ICMP alone, or vice versa.
		{Label: "teams_tcp443", Kind: KindTCP, Host: "teams.microsoft.com", Port: &port443},
		{Label: "zoom_tcp443", Kind: KindTCP, Host: "zoom.us", Port: &port443},
		{Label: "outlook_tcp443", Kind: KindTCP, Host: "outlook.office.com", Port: &port443},

		// UDP DNS — real UDP traffic to well-known public resolvers.
		{Label: "google_dns_udp", Kind: KindUDPDNS, Host: "8.8.8.8", Port: &port53},
		{Label: "cloudflare_dns_udp", Kind: KindUDPDNS, Host: "1.1.1.1", Port: &port53},

		// UDP STUN — WebRTC / Teams / Zoom call-plane protocol.
		{Label: "google_stun_udp", Kind: KindUDPSTUN, Host: "stun.l.google.com", Port: &portStunGoogle},
		{Label: "cloudflare_stun_udp", Kind: KindUDPSTUN, Host: "stun.cloudflare.com", Port: &port3478},
	}
}
