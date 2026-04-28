//go:build linux

package netinfo

import (
	"context"
	"strconv"
	"time"

	"github.com/mdlayher/wifi"
)

// SampleWifi reads Wi-Fi state from netlink via mdlayher/wifi. Pure Go, no
// shell-out, no `iw`/`nmcli` dependency. Works on any modern Linux desktop
// distro (kernel 3.0+).
//
// Returns a near-empty sample if the host has no Wi-Fi interface or the
// netlink call fails — never errors back to the caller.
func SampleWifi(_ context.Context) WifiSample {
	sample := WifiSample{Timestamp: time.Now()}

	c, err := wifi.New()
	if err != nil {
		return sample
	}

	defer func() { _ = c.Close() }()

	ifaces, err := c.Interfaces()
	if err != nil {
		return sample
	}

	// Pick the first interface that's actually associated with a BSS.
	for _, iface := range ifaces {
		if iface.Type == wifi.InterfaceTypeMonitor {
			continue
		}

		bss, err := c.BSS(iface)
		if err != nil || bss == nil {
			continue
		}

		if bss.SSID != "" {
			sample.SSID = strPtr(bss.SSID)
		}

		if len(bss.BSSID) > 0 {
			sample.BSSID = strPtr(bss.BSSID.String())
		}

		// mdlayher/wifi doesn't expose RSSI via the BSS struct; ask
		// StationInfo on the connected interface for signal strength.
		stations, err := c.StationInfo(iface)
		if err == nil {
			for _, s := range stations {
				if s.Signal != 0 {
					sample.RSSIDbm = intPtr(s.Signal)

					break
				}
			}
		}

		// Channel is encoded in BSS.Frequency (MHz). Round-trip to channel
		// number is a piecewise function — skip the math, store MHz.
		if bss.Frequency != 0 {
			ch := strconv.Itoa(bss.Frequency) + " MHz"
			sample.Channel = strPtr(ch)
		}

		break
	}

	return sample
}
