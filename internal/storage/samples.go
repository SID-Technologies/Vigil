package storage

import (
	"context"

	"github.com/sid-technologies/vigil/db/ent"
	"github.com/sid-technologies/vigil/db/ent/sample"
)

// Sample is the storage-layer view of a probe result. JSON-serialized
// directly into IPC responses.
type Sample struct {
	TsUnixMs    int64    `json:"ts_unix_ms"`
	TargetLabel string   `json:"target_label"`
	TargetKind  string   `json:"target_kind"`
	TargetHost  string   `json:"target_host"`
	TargetPort  *int     `json:"target_port,omitempty"`
	Success     bool     `json:"success"`
	RTTMs       *float64 `json:"rtt_ms,omitempty"`
	Error       *string  `json:"error,omitempty"`
}

// QuerySamplesParams scopes a samples.query call. fromMs/toMs are inclusive
// unix-ms bounds. If targetLabels is non-empty, results are filtered to that
// set; otherwise all targets are included. Limit caps the result count to
// keep the IPC payload bounded — frontend typically requests <= 5000 rows.
type QuerySamplesParams struct {
	FromMs       int64
	ToMs         int64
	TargetLabels []string
	Limit        int
}

// QuerySamples runs a time-windowed (and optionally target-filtered) read.
// Results are ordered by ts_unix_ms ascending — that's what charts expect.
func QuerySamples(ctx context.Context, client *ent.Client, p QuerySamplesParams) ([]Sample, error) {
	q := client.Sample.Query().
		Where(
			sample.TsUnixMsGTE(p.FromMs),
			sample.TsUnixMsLTE(p.ToMs),
		).
		Order(ent.Asc(sample.FieldTsUnixMs))

	if len(p.TargetLabels) > 0 {
		q = q.Where(sample.TargetLabelIn(p.TargetLabels...))
	}
	if p.Limit > 0 {
		q = q.Limit(p.Limit)
	}

	rows, err := q.All(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	out := make([]Sample, 0, len(rows))
	for _, r := range rows {
		s := Sample{
			TsUnixMs:    r.TsUnixMs,
			TargetLabel: r.TargetLabel,
			TargetKind:  r.TargetKind,
			TargetHost:  r.TargetHost,
			Success:     r.Success,
		}
		if r.TargetPort != nil {
			s.TargetPort = r.TargetPort
		}
		if r.RttMs != nil {
			s.RTTMs = r.RttMs
		}
		if r.Error != nil {
			s.Error = r.Error
		}
		out = append(out, s)
	}
	return out, nil
}
