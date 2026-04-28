package reports

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"strings"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

// writeHTML renders the report payload through the embedded HTML template.
// Output is a single self-contained file the user can email or share —
// only external resource is Chart.js from CDN, which gracefully degrades
// to "tables only" when the recipient is offline.
func writeHTML(path string, rep *report) error {
	t, err := loadTemplate()
	if err != nil {
		return fmt.Errorf("load template: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return err //nolint:wrapcheck
	}
	defer f.Close()

	// Render with chartData injected as a JS literal so Chart.js can pick
	// it up without a network round-trip.
	chartData, err := json.Marshal(buildChartData(rep))
	if err != nil {
		return fmt.Errorf("marshal chart data: %w", err)
	}

	return t.Execute(f, map[string]any{ //nolint:wrapcheck
		"Report":    rep,
		"ChartJSON": template.JS(string(chartData)),
	})
}

func loadTemplate() (*template.Template, error) {
	return template.New("report").Funcs(funcMap).ParseFS(templateFS, "templates/*.tmpl") //nolint:wrapcheck
}

// chartPayload is the JSON shape consumed by the inlined Chart.js script.
type chartPayload struct {
	Hours    []string  `json:"hours"`
	Median   []float64 `json:"median"`
	P95      []float64 `json:"p95"`
	UptimePct []float64 `json:"uptime_pct"`
}

func buildChartData(rep *report) chartPayload {
	out := chartPayload{
		Hours:     make([]string, 0, len(rep.HourlyBuckets)),
		Median:    make([]float64, 0, len(rep.HourlyBuckets)),
		P95:       make([]float64, 0, len(rep.HourlyBuckets)),
		UptimePct: make([]float64, 0, len(rep.HourlyBuckets)),
	}
	for _, b := range rep.HourlyBuckets {
		out.Hours = append(out.Hours, b.Hour)
		if b.MedianRTT != nil {
			out.Median = append(out.Median, *b.MedianRTT)
		} else {
			out.Median = append(out.Median, 0)
		}
		if b.P95RTT != nil {
			out.P95 = append(out.P95, *b.P95RTT)
		} else {
			out.P95 = append(out.P95, 0)
		}
		out.UptimePct = append(out.UptimePct, b.UptimePct)
	}
	return out
}

// funcMap registers helpers usable inside templates/*.tmpl.
var funcMap = template.FuncMap{
	"fmtPct": func(v float64) string {
		return fmt.Sprintf("%.2f%%", v)
	},
	"fmtMs": func(v *float64) string {
		if v == nil {
			return "—"
		}
		return fmt.Sprintf("%.2f ms", *v)
	},
	"fmtFloat": func(v float64) string {
		return fmt.Sprintf("%.2f", v)
	},
	"replace": strings.ReplaceAll,
	"verdict": func(uptimePct float64, outages int) string {
		switch {
		case outages > 0:
			return "bad"
		case uptimePct < 99.0:
			return "warn"
		default:
			return "good"
		}
	},
	"verdictText": func(uptimePct float64, outages int) string {
		switch {
		case outages > 0:
			return fmt.Sprintf(
				"Network reliability incident: %d outage event(s) detected within the window. Uptime: %.2f%%.",
				outages, uptimePct,
			)
		case uptimePct < 99.0:
			return fmt.Sprintf(
				"Network reliability degraded. Uptime: %.2f%% — below the 99%% threshold.",
				uptimePct,
			)
		default:
			return fmt.Sprintf(
				"Network operating within healthy bounds. Uptime: %.2f%%.",
				uptimePct,
			)
		}
	},
}
