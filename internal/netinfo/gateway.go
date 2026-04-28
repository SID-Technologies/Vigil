// Package netinfo wraps OS-specific network introspection (default gateway,
// Wi-Fi state) behind a small cross-platform interface.
//
// Gateway detection is uniform across OSes via jackpal/gateway (which uses
// netlink on Linux and PF_ROUTE on macOS/BSD/Windows internally), so it
// lives in this single file with no build tags.
//
// Wi-Fi state varies wildly per platform — see wifi_*.go for the
// build-tagged implementations.
package netinfo

import (
	"net"

	"github.com/jackpal/gateway"
)

// DetectDefaultGateway returns the IP address of the system's default route,
// or ("", false) if it can't be determined.
//
// Used by the monitor to dynamically add a `router_icmp` probe at startup —
// the legacy Python tool inserted this probe so users always have direct
// "is my router reachable?" data alongside the public-internet probes.
func DetectDefaultGateway() (string, bool) {
	ip, err := gateway.DiscoverGateway()
	if err != nil {
		return "", false
	}

	if ip == nil || ip.Equal(net.IPv4zero) {
		return "", false
	}

	return ip.String(), true
}
