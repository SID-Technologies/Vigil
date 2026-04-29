package storage

import (
	"context"

	"github.com/sid-technologies/vigil/db/ent"
	"github.com/sid-technologies/vigil/db/ent/sample"
	"github.com/sid-technologies/vigil/db/ent/sample1h"
	"github.com/sid-technologies/vigil/db/ent/sample1min"
	"github.com/sid-technologies/vigil/db/ent/sample5min"
	"github.com/sid-technologies/vigil/pkg/errors"
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

// QuerySamplesParams — FromMs/ToMs are inclusive unix-ms bounds; Limit > 0
// caps result count to keep IPC payloads bounded.
type QuerySamplesParams struct {
	FromMs       int64
	ToMs         int64
	TargetLabels []string
	Limit        int
}

// SampleClient owns reads against the raw + rollup sample tables.
type SampleClient struct {
	client *ent.Client
}

// NewSampleClient wraps an Ent client.
func NewSampleClient(client *ent.Client) *SampleClient {
	return &SampleClient{client: client}
}

// Query returns raw samples in [FromMs, ToMs] ordered ascending by timestamp.
func (c *SampleClient) Query(ctx context.Context, p QuerySamplesParams) ([]Sample, error) {
	q := c.client.Sample.Query().
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
		return nil, errors.Wrap(err, "failed to query samples")
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

// Query1Min returns 1-minute rollup buckets in the window.
func (c *SampleClient) Query1Min(ctx context.Context, p QueryAggregatedParams) ([]AggregatedRow, error) {
	q := c.client.Sample1min.Query().
		Where(sample1min.BucketStartUnixMsGTE(p.FromMs), sample1min.BucketStartUnixMsLTE(p.ToMs)).
		Order(ent.Asc(sample1min.FieldBucketStartUnixMs))
	if len(p.TargetLabels) > 0 {
		q = q.Where(sample1min.TargetLabelIn(p.TargetLabels...))
	}

	rows, err := q.All(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query 1min samples")
	}

	return projectAggRows(rows), nil
}

// Query5Min returns 5-minute rollup buckets in the window.
func (c *SampleClient) Query5Min(ctx context.Context, p QueryAggregatedParams) ([]AggregatedRow, error) {
	q := c.client.Sample5min.Query().
		Where(sample5min.BucketStartUnixMsGTE(p.FromMs), sample5min.BucketStartUnixMsLTE(p.ToMs)).
		Order(ent.Asc(sample5min.FieldBucketStartUnixMs))
	if len(p.TargetLabels) > 0 {
		q = q.Where(sample5min.TargetLabelIn(p.TargetLabels...))
	}

	rows, err := q.All(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query 5min samples")
	}

	return projectAggRows(rows), nil
}

// Query1H returns 1-hour rollup buckets in the window.
func (c *SampleClient) Query1H(ctx context.Context, p QueryAggregatedParams) ([]AggregatedRow, error) {
	q := c.client.Sample1h.Query().
		Where(sample1h.BucketStartUnixMsGTE(p.FromMs), sample1h.BucketStartUnixMsLTE(p.ToMs)).
		Order(ent.Asc(sample1h.FieldBucketStartUnixMs))
	if len(p.TargetLabels) > 0 {
		q = q.Where(sample1h.TargetLabelIn(p.TargetLabels...))
	}

	rows, err := q.All(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query 1h samples")
	}

	return projectAggRows(rows), nil
}
