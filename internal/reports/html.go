package reports

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"strings"

	"github.com/sid-technologies/vigil/pkg/errors"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

// writeHTML renders the embedded template into a self-contained file.
// Chart.js is loaded from CDN; offline recipients see tables only.
func writeHTML(path string, rep *report) (err error) {
	t, err := loadTemplate()
	if err != nil {
		return errors.Wrap(err, "load template")
	}

	f, err := os.Create(path) //nolint:gosec // path supplied by user via report export UI
	if err != nil {
		return err //nolint:wrapcheck // wrapped by caller in Generate
	}

	defer func() {
		cerr := f.Close()
		if err == nil && cerr != nil {
			err = cerr
		}
	}()

	// chartData is injected as a JS literal so Chart.js picks it up inline.
	chartData, err := json.Marshal(buildChartData(rep))
	if err != nil {
		return errors.Wrap(err, "marshal chart data")
	}

	return t.Execute(f, map[string]any{ //nolint:wrapcheck // wrapped by caller in Generate
		"Report":    rep,
		"ChartJSON": template.JS(string(chartData)), //nolint:gosec // chart JSON is generated server-side from local probe data, not user input
	})
}

func loadTemplate() (*template.Template, error) {
	return template.New("report").Funcs(funcMap).ParseFS(templateFS, "templates/*.tmpl") //nolint:wrapcheck // wrapped by caller in writeHTML
}

// chartPayload is consumed by the inlined Chart.js script.
type chartPayload struct {
	Hours     []string  `json:"hours"`
	Median    []float64 `json:"median"`
	P95       []float64 `json:"p95"`
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
		out.Median = append(out.Median, floatOrZero(b.MedianRTT))
		out.P95 = append(out.P95, floatOrZero(b.P95RTT))
		out.UptimePct = append(out.UptimePct, b.UptimePct)
	}

	return out
}

// floatOrZero unboxes a *float64 to 0 for missing values — recharts can't
// render a nil so we substitute the floor value instead of dropping the bucket.
func floatOrZero(p *float64) float64 {
	if p == nil {
		return 0
	}

	return *p
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
