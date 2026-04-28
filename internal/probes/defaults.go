package probes

// Well-known ports used by the default targets list.
const (
	HTTPSPort         = 443
	GoogleSTUNAltPort = 19302 // stun.l.google.com listens on 19302, not 3478
)

// DefaultTargets are the 12 builtin probe targets seeded on first run. A
// 13th (router_icmp) is added at runtime against the detected gateway.
// Mix is intentional: ICMP + TCP:443 to real video-call hostnames, UDP:53
// to public DNS, and STUN over UDP because that's the same protocol
// Teams/Zoom/Meet/Discord use at call setup.
func DefaultTargets() []Target {
	port443 := HTTPSPort
	port53 := defaultDNSPort
	port3478 := defaultSTUNPort
	portStunGoogle := GoogleSTUNAltPort

	return []Target{
		{Label: "google_dns_icmp", Kind: KindICMP, Host: "8.8.8.8"},
		{Label: "cloudflare_dns_icmp", Kind: KindICMP, Host: "1.1.1.1"},
		{Label: "teams_icmp", Kind: KindICMP, Host: "teams.microsoft.com"},
		{Label: "zoom_icmp", Kind: KindICMP, Host: "zoom.us"},
		{Label: "outlook_icmp", Kind: KindICMP, Host: "outlook.office.com"},

		{Label: "teams_tcp443", Kind: KindTCP, Host: "teams.microsoft.com", Port: &port443},
		{Label: "zoom_tcp443", Kind: KindTCP, Host: "zoom.us", Port: &port443},
		{Label: "outlook_tcp443", Kind: KindTCP, Host: "outlook.office.com", Port: &port443},

		{Label: "google_dns_udp", Kind: KindUDPDNS, Host: "8.8.8.8", Port: &port53},
		{Label: "cloudflare_dns_udp", Kind: KindUDPDNS, Host: "1.1.1.1", Port: &port53},

		{Label: "google_stun_udp", Kind: KindUDPSTUN, Host: "stun.l.google.com", Port: &portStunGoogle},
		{Label: "cloudflare_stun_udp", Kind: KindUDPSTUN, Host: "stun.cloudflare.com", Port: &port3478},
	}
}
