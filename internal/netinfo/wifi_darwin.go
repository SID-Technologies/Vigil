//go:build darwin

package netinfo

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// SampleWifi parses `system_profiler SPAirPortDataType` on macOS.
// system_profiler is the only sanctioned permissionless API — `airport -I`
// was removed in Sonoma 14.4, `wdutil info` needs Location Services or
// sudo, and CoreWLAN would require cgo. It's slow (~500ms) but runs at the
// 60s flush interval so the cost is invisible. SignalPercent and rx/tx
// rates aren't exposed and stay nil.
func SampleWifi(ctx context.Context) WifiSample {
	sample := WifiSample{Timestamp: time.Now()}

	cctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()

	out, err := exec.CommandContext(cctx, "system_profiler", "SPAirPortDataType", "-detailLevel", "basic").Output()
	if err != nil {
		return sample
	}

	parseSystemProfilerAirport(string(out), &sample)

	return sample
}

// parseSystemProfilerAirport pulls SSID/BSSID/channel/RSSI from the
// "Current Network Information" block of system_profiler output. SSID is
// the indented dictionary key under that header; we walk indentation depth
// to find it. Fragile to Apple format changes but stable across recent
// macOS versions.
//
// Output shape (abbreviated):
//
//	Current Network Information:
//	    MyNetworkName:
//	        BSSID: aa:bb:cc:dd:ee:ff
//	        Channel: 36 (5GHz, 80MHz)
//	        Signal / Noise: -52 dBm / -90 dBm
func parseSystemProfilerAirport(out string, sample *WifiSample) {
	lines := strings.Split(out, "\n")
	inCurrent := false
	currentIndent := -1

	for i, raw := range lines {
		line := strings.TrimRight(raw, " \t\r")
		if line == "" {
			continue
		}

		trimmed := strings.TrimLeft(line, " \t")
		indent := len(line) - len(trimmed)

		if strings.HasPrefix(trimmed, "Current Network Information:") {
			inCurrent = true
			currentIndent = indent
			// The SSID is the next non-empty line, indented further.
			for j := i + 1; j < len(lines); j++ {
				next := strings.TrimRight(lines[j], " \t\r")
				if next == "" {
					continue
				}

				nextTrim := strings.TrimLeft(next, " \t")

				nextIndent := len(next) - len(nextTrim)
				if nextIndent > currentIndent && strings.HasSuffix(nextTrim, ":") {
					ssid := strings.TrimSuffix(nextTrim, ":")
					sample.SSID = strPtr(ssid)
				}

				break
			}

			continue
		}

		if !inCurrent {
			continue
		}
		// Exit the block when indentation drops back.
		if indent <= currentIndent {
			break
		}

		switch {
		case strings.HasPrefix(trimmed, "BSSID:"):
			sample.BSSID = strPtr(strings.TrimSpace(strings.TrimPrefix(trimmed, "BSSID:")))
		case strings.HasPrefix(trimmed, "Channel:"):
			sample.Channel = strPtr(strings.TrimSpace(strings.TrimPrefix(trimmed, "Channel:")))
		case strings.HasPrefix(trimmed, "Signal / Noise:"):
			rssi := parseSignalDbm(trimmed)
			if rssi != nil {
				sample.RSSIDbm = rssi
			}
		default:
		}
	}
}

// parseSignalDbm extracts the signal value from "Signal / Noise: -52 dBm / -90 dBm".
func parseSignalDbm(line string) *int {
	_, after, ok := strings.Cut(line, ":")
	if !ok {
		return nil
	}

	rest := strings.TrimSpace(after) // e.g. "-52 dBm / -90 dBm"

	parts := strings.SplitN(rest, "/", 2)
	if len(parts) == 0 {
		return nil
	}

	first := strings.TrimSpace(parts[0])
	first = strings.TrimSuffix(first, "dBm")
	first = strings.TrimSpace(first)

	v, err := strconv.Atoi(first)
	if err != nil {
		return nil
	}

	return intPtr(v)
}
