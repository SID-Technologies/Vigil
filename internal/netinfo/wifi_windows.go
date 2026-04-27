//go:build windows

package netinfo

import (
	"context"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// SampleWifi parses `netsh wlan show interfaces` on Windows. Unchanged in
// behavior from the legacy Python implementation — same parser, same
// expected output format.
//
// Caveat: netsh's output is localized. On a German/French/etc. Windows, the
// "Signal" / "Receive rate" / "BSSID" labels are translated and this parser
// returns mostly empty fields. We don't try to handle that — Vigil's target
// audience runs English-locale Windows and the ROI on a multi-locale parser
// is low.
func SampleWifi(ctx context.Context) WifiSample {
	sample := WifiSample{Timestamp: time.Now()}

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	out, err := exec.CommandContext(cctx, "netsh", "wlan", "show", "interfaces").Output()
	if err != nil {
		return sample
	}

	parseNetshWlan(string(out), &sample)
	return sample
}

func grab(stdout, key string) string {
	re := regexp.MustCompile(`(?im)^\s*` + regexp.QuoteMeta(key) + `\s*:\s*(.+?)\s*$`)
	m := re.FindStringSubmatch(stdout)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

func parseNetshWlan(stdout string, sample *WifiSample) {
	if v := grab(stdout, "SSID"); v != "" {
		sample.SSID = strPtr(v)
	}
	if v := grab(stdout, "BSSID"); v != "" {
		sample.BSSID = strPtr(v)
	}
	if v := grab(stdout, "Channel"); v != "" {
		sample.Channel = strPtr(v)
	}
	if v := grab(stdout, "Signal"); v != "" {
		if pct, ok := parsePercent(v); ok {
			sample.SignalPercent = intPtr(pct)
			// Rough Windows convention: 100% ≈ -50 dBm, 0% ≈ -100 dBm.
			rssi := -100 + (pct / 2)
			sample.RSSIDbm = intPtr(rssi)
		}
	}
	if v := grab(stdout, "Receive rate (Mbps)"); v != "" {
		if rate, err := strconv.ParseFloat(v, 64); err == nil {
			sample.RxRateMbps = f64Ptr(rate)
		}
	}
	if v := grab(stdout, "Transmit rate (Mbps)"); v != "" {
		if rate, err := strconv.ParseFloat(v, 64); err == nil {
			sample.TxRateMbps = f64Ptr(rate)
		}
	}
}

func parsePercent(raw string) (int, bool) {
	if !strings.HasSuffix(raw, "%") {
		return 0, false
	}
	v, err := strconv.Atoi(strings.TrimSuffix(raw, "%"))
	if err != nil {
		return 0, false
	}
	return v, true
}
