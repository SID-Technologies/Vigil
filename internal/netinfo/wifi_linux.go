//go:build linux

package netinfo

import (
	"context"
	"strconv"
	"time"

	"github.com/mdlayher/wifi"
)

// SampleWifi reads Wi-Fi state from netlink via mdlayher/wifi. Pure Go, no
// shell-out. Returns a near-empty sample on failure rather than erroring.
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

		// mdlayher/wifi doesn't expose RSSI on BSS; StationInfo carries it.
		stations, err := c.StationInfo(iface)
		if err == nil {
			for _, s := range stations {
				if s.Signal != 0 {
					sample.RSSIDbm = intPtr(s.Signal)

					break
				}
			}
		}

		// Frequency is in MHz; channel-number conversion is piecewise so we store MHz.
		if bss.Frequency != 0 {
			ch := strconv.Itoa(bss.Frequency) + " MHz"
			sample.Channel = strPtr(ch)
		}

		break
	}

	return sample
}
