package storage

import (
	"context"

	"github.com/sid-technologies/vigil/db/ent"
	"github.com/sid-technologies/vigil/db/ent/sample"
)

// Sample is the IPC-shape view of a single probe result.
type Sample struct {
	TSUnixMs    int64    `json:"ts_unix_ms"`
	TargetLabel string   `json:"target_label"`
	TargetKind  string   `json:"target_kind"`
	TargetHost  string   `json:"target_host"`
	TargetPort  *int     `json:"target_port,omitempty"`
	Success     bool     `json:"success"`
	RTTMs       *float64 `json:"rtt_ms,omitempty"`
	Error       *string  `json:"error,omitempty"`
}

// QuerySamplesParams — FromMs/ToMs are inclusive unix-ms bounds; Limit > 0 caps
// result count to keep IPC payloads bounded.
type QuerySamplesParams struct {
	FromMs       int64
	ToMs         int64
	TargetLabels []string
	Limit        int
}

// QuerySamples returns samples in [FromMs, ToMs] ordered ascending by timestamp.
func (s *Store) QuerySamples(ctx context.Context, p QuerySamplesParams) ([]Sample, error) {
	q := s.client.Sample.Query().
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
		return nil, err //nolint:wrapcheck // wrapped at IPC boundary
	}

	out := make([]Sample, 0, len(rows))
	for _, r := range rows {
		row := Sample{
			TSUnixMs:    r.TsUnixMs,
			TargetLabel: r.TargetLabel,
			TargetKind:  r.TargetKind,
			TargetHost:  r.TargetHost,
			Success:     r.Success,
		}
		if r.TargetPort != nil {
			row.TargetPort = r.TargetPort
		}

		if r.RttMs != nil {
			row.RTTMs = r.RttMs
		}

		if r.Error != nil {
			row.Error = r.Error
		}

		out = append(out, row)
	}

	return out, nil
}
