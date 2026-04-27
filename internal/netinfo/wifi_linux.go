//go:build linux

package netinfo

import (
	"context"
	"net"
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
	defer c.Close()

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
		if bss.BSSID != nil && len(bss.BSSID) > 0 {
			sample.BSSID = strPtr(net.HardwareAddr(bss.BSSID).String())
		}
		// mdlayher/wifi doesn't expose RSSI directly via the BSS struct;
		// use StationInfo for the connected interface to get signal.
		if stations, err := c.StationInfo(iface); err == nil {
			for _, s := range stations {
				if s.Signal != 0 {
					sample.RSSIDbm = intPtr(s.Signal)
					break
				}
			}
		}
		// Channel is encoded in BSS.Frequency (MHz). Round-trip to channel
		// number is a piecewise function — skip the math, just store MHz.
		if bss.Frequency != 0 {
			ch := strconv.Itoa(bss.Frequency) + " MHz"
			sample.Channel = strPtr(ch)
		}
		break
	}
	return sample
}
