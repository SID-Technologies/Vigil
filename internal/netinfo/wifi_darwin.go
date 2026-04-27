//go:build darwin

package netinfo

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// SampleWifi parses `system_profiler SPAirPortDataType` on macOS. We chose
// system_profiler over the alternatives because:
//
//   - `airport -I` was removed in macOS Sonoma 14.4 (2024). Dead.
//   - `wdutil info` requires Location Services permission or sudo. Awkward.
//   - CoreWLAN via cgo gives the richest data but requires Objective-C
//     bindings — overkill for v1.
//
// system_profiler is slow (~500ms) but our flush interval is 60s so the cost
// is invisible. It runs without permissions and reliably exposes SSID, BSSID,
// channel, and RSSI.
//
// Fields not exposed by system_profiler (signal_percent, rx/tx rates) remain
// nil. Cross-platform code must treat them as "n/a" rather than 0.
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

// parseSystemProfilerAirport finds the "Current Network Information" block
// in system_profiler's plain-text output and pulls SSID/BSSID/channel/RSSI.
//
// system_profiler's output structure (abbreviated):
//
//	Wi-Fi:
//	    Interfaces:
//	        en0:
//	            Status: Connected
//	            Current Network Information:
//	                MyNetworkName:
//	                    PHY Mode: 802.11ax
//	                    BSSID: aa:bb:cc:dd:ee:ff
//	                    Channel: 36 (5GHz, 80MHz)
//	                    Country Code: US
//	                    Network Type: Infrastructure
//	                    Security: WPA2 Personal
//	                    Signal / Noise: -52 dBm / -90 dBm
//
// The SSID is the dictionary key one indent past "Current Network Information".
// We extract by walking lines and tracking indentation depth — fragile to
// Apple changing the format, but stable across recent macOS versions.
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
		}
	}
}

// parseSignalDbm extracts the signal value from "Signal / Noise: -52 dBm / -90 dBm".
func parseSignalDbm(line string) *int {
	idx := strings.Index(line, ":")
	if idx == -1 {
		return nil
	}
	rest := strings.TrimSpace(line[idx+1:])
	// rest = "-52 dBm / -90 dBm"
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) == 0 {
		return nil
	}
	first := strings.TrimSpace(parts[0]) // "-52 dBm"
	first = strings.TrimSuffix(first, "dBm")
	first = strings.TrimSpace(first)
	v, err := strconv.Atoi(first)
	if err != nil {
		return nil
	}
	return intPtr(v)
}
