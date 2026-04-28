package reports

import (
	"sort"
	"time"

	"github.com/sid-technologies/vigil/internal/constants"
	"github.com/sid-technologies/vigil/internal/stats"
	"github.com/sid-technologies/vigil/internal/storage"
)

// targetScopePrefix marks per-target outage scopes (e.g. "target:foo");
// network-wide outages use scope "network" and are excluded from per-target
// burst counts.
const targetScopePrefix = "target:"

// minSamplesForBucketP95 guards against p95-as-max on tiny hourly windows.
const minSamplesForBucketP95 = 20

// report is the payload written to JSON and passed to the HTML template.
// Timestamps appear in both unix-ms and ISO8601 for script + human readers.
type report struct {
	GeneratedAt   string               `json:"generated_at"`
	WindowStart   string               `json:"window_start_utc"`
	WindowEnd     string               `json:"window_end_utc"`
	WindowHours   float64              `json:"window_hours"`
	TotalSamples  int                  `json:"total_samples"`
	Summary       summaryStats         `json:"summary"`
	PerTarget     []targetStats        `json:"per_target"`
	HourlyBuckets []hourlyBucket       `json:"hourly"`
	Outages       []storage.Outage     `json:"outages"`
	WifiSamples   []storage.WifiSample `json:"wifi_samples,omitempty"`
	// Raw probes only emitted to JSON; HTML doesn't inline them.
	Samples []storage.Sample `json:"samples,omitempty"`
}

type summaryStats struct {
	UptimePct     float64  `json:"uptime_pct"`
	TotalSuccess  int      `json:"successful_probes"`
	TotalFail     int      `json:"failed_probes"`
	OutageCount   int      `json:"outage_count"`
	OutageNetwork int      `json:"network_outage_count"`
	OutageTargets int      `json:"target_outage_count"`
	MeanRTTMs     *float64 `json:"mean_rtt_ms,omitempty"`
	P95RTTMs      *float64 `json:"p95_rtt_ms,omitempty"`
	P99RTTMs      *float64 `json:"p99_rtt_ms,omitempty"`
}

type targetStats struct {
	Label        string   `json:"target_label"`
	Kind         string   `json:"target_kind"`
	Host         string   `json:"target_host"`
	Port         *int     `json:"target_port,omitempty"`
	Total        int      `json:"total_probes"`
	Successful   int      `json:"successful"`
	Failed       int      `json:"failed"`
	UptimePct    float64  `json:"uptime_pct"`
	P50Ms        *float64 `json:"p50_ms,omitempty"`
	P95Ms        *float64 `json:"p95_ms,omitempty"`
	P99Ms        *float64 `json:"p99_ms,omitempty"`
	MaxMs        *float64 `json:"max_ms,omitempty"`
	MeanMs       *float64 `json:"mean_ms,omitempty"`
	JitterMs     *float64 `json:"jitter_ms,omitempty"`
	OutageBursts int      `json:"target_outage_bursts"`
}

type hourlyBucket struct {
	Hour      string   `json:"hour_local"`
	Total     int      `json:"total"`
	Failed    int      `json:"failed"`
	UptimePct float64  `json:"uptime_pct"`
	MedianRTT *float64 `json:"median_rtt_ms,omitempty"`
	P95RTT    *float64 `json:"p95_rtt_ms,omitempty"`
}

// buildReport folds raw samples + wifi + outages into the report payload.
// Percentile math goes through internal/stats so values match the dashboard.
func buildReport(fromMs, toMs int64, samples []storage.Sample, wifi []storage.WifiSample, outages []storage.Outage) *report {
	now := time.Now().UTC()
	r := &report{
		GeneratedAt:  now.Format(time.RFC3339),
		WindowStart:  time.UnixMilli(fromMs).UTC().Format(time.RFC3339),
		WindowEnd:    time.UnixMilli(toMs).UTC().Format(time.RFC3339),
		WindowHours:  float64(toMs-fromMs) / float64(time.Hour/time.Millisecond),
		TotalSamples: len(samples),
		Outages:      outages,
		WifiSamples:  wifi,
		Samples:      samples,
	}

	r.Summary = computeSummary(samples, outages)
	r.PerTarget = computePerTarget(samples, outages)
	r.HourlyBuckets = computeHourlyBuckets(samples)

	return r
}

func computeSummary(samples []storage.Sample, outages []storage.Outage) summaryStats {
	out := summaryStats{}
	if len(samples) == 0 {
		return out
	}

	rtts := make([]float64, 0, len(samples))
	for _, s := range samples {
		if s.Success {
			out.TotalSuccess++

			if s.RTTMs != nil {
				rtts = append(rtts, *s.RTTMs)
			}
		} else {
			out.TotalFail++
		}
	}

	total := out.TotalSuccess + out.TotalFail
	if total > 0 {
		out.UptimePct = stats.Round2(float64(out.TotalSuccess) / float64(total) * constants.PercentMultiplier)
	}

	out.OutageCount = len(outages)
	for _, o := range outages {
		if o.Scope == "network" {
			out.OutageNetwork++
		} else {
			out.OutageTargets++
		}
	}

	if len(rtts) > 0 {
		sort.Float64s(rtts)

		if v, ok := stats.Mean(rtts); ok {
			r := stats.Round2(v)
			out.MeanRTTMs = &r
		}

		if v, ok := stats.Percentile(rtts, constants.P95Quantile); ok {
			r := stats.Round2(v)
			out.P95RTTMs = &r
		}

		if v, ok := stats.Percentile(rtts, constants.P99Quantile); ok {
			r := stats.Round2(v)
			out.P99RTTMs = &r
		}
	}

	return out
}

func computePerTarget(samples []storage.Sample, outages []storage.Outage) []targetStats {
	groups := make(map[string][]storage.Sample)
	heads := make(map[string]storage.Sample)

	for _, s := range samples {
		groups[s.TargetLabel] = append(groups[s.TargetLabel], s)
		if _, ok := heads[s.TargetLabel]; !ok {
			heads[s.TargetLabel] = s
		}
	}

	bursts := make(map[string]int)

	for _, o := range outages {
		if len(o.Scope) > len(targetScopePrefix) && o.Scope[:len(targetScopePrefix)] == targetScopePrefix {
			bursts[o.Scope[len(targetScopePrefix):]]++
		}
	}

	out := make([]targetStats, 0, len(groups))
	for label, rows := range groups {
		head := heads[label]
		statsRow := targetStats{
			Label:        label,
			Kind:         head.TargetKind,
			Host:         head.TargetHost,
			Port:         head.TargetPort,
			Total:        len(rows),
			OutageBursts: bursts[label],
		}

		rtts := make([]float64, 0, len(rows))

		rttsTimeOrder := make([]float64, 0, len(rows))
		for _, r := range rows {
			if r.Success {
				statsRow.Successful++

				if r.RTTMs != nil {
					rtts = append(rtts, *r.RTTMs)
					rttsTimeOrder = append(rttsTimeOrder, *r.RTTMs)
				}
			} else {
				statsRow.Failed++
			}
		}

		if statsRow.Total > 0 {
			statsRow.UptimePct = stats.Round2(
				float64(statsRow.Successful) / float64(statsRow.Total) * constants.PercentMultiplier,
			)
		}

		FillTargetRTTStats(&statsRow, rtts, rttsTimeOrder)

		out = append(out, statsRow)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Label < out[j].Label
	})

	return out
}

// FillTargetRTTStats populates percentile/max/mean/jitter pointer fields.
// rtts is sorted in place; rttsTimeOrder must be in time order. No-op if empty.
func FillTargetRTTStats(row *targetStats, rtts, rttsTimeOrder []float64) {
	if len(rtts) == 0 {
		return
	}

	sort.Float64s(rtts)

	if v, ok := stats.Percentile(rtts, constants.P50Quantile); ok {
		r := stats.Round2(v)
		row.P50Ms = &r
	}

	if v, ok := stats.Percentile(rtts, constants.P95Quantile); ok {
		r := stats.Round2(v)
		row.P95Ms = &r
	}

	if v, ok := stats.Percentile(rtts, constants.P99Quantile); ok {
		r := stats.Round2(v)
		row.P99Ms = &r
	}

	maxMs := stats.Round2(rtts[len(rtts)-1])
	row.MaxMs = &maxMs

	if v, ok := stats.Mean(rtts); ok {
		r := stats.Round2(v)
		row.MeanMs = &r
	}

	if v, ok := stats.JitterMs(rttsTimeOrder); ok {
		r := stats.Round2(v)
		row.JitterMs = &r
	}
}

// computeHourlyBuckets bins samples by local-time hour for the time-series chart.
func computeHourlyBuckets(samples []storage.Sample) []hourlyBucket {
	type acc struct {
		total int
		fail  int
		rtts  []float64
	}

	buckets := make(map[string]*acc)

	for _, s := range samples {
		hr := time.UnixMilli(s.TSUnixMs).Local().Format("2006-01-02 15:00") //nolint:gosmopolitan // local time is intentional for user-facing reports

		b := buckets[hr]
		if b == nil {
			b = &acc{}
			buckets[hr] = b
		}

		b.total++
		if !s.Success {
			b.fail++
		} else if s.RTTMs != nil {
			b.rtts = append(b.rtts, *s.RTTMs)
		}
	}

	keys := make([]string, 0, len(buckets))
	for k := range buckets {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	out := make([]hourlyBucket, 0, len(buckets))
	for _, k := range keys {
		b := buckets[k]

		row := hourlyBucket{
			Hour:   k,
			Total:  b.total,
			Failed: b.fail,
		}
		if b.total > 0 {
			row.UptimePct = stats.Round2(
				float64(b.total-b.fail) / float64(b.total) * constants.PercentMultiplier,
			)
		}

		if len(b.rtts) > 0 {
			sort.Float64s(b.rtts)
			med := stats.Round2(b.rtts[len(b.rtts)/2])
			row.MedianRTT = &med

			if len(b.rtts) > minSamplesForBucketP95 {
				idx := int(float64(len(b.rtts)) * constants.P95Quantile)
				if idx >= len(b.rtts) {
					idx = len(b.rtts) - 1
				}

				p95 := stats.Round2(b.rtts[idx])
				row.P95RTT = &p95
			}
		}

		out = append(out, row)
	}

	return out
}

