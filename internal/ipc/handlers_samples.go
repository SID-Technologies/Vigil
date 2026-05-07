package ipc

import (
	"context"
	"time"

	"github.com/sid-technologies/vigil/internal/storage"
)

// Granularity strings exchanged on the wire.
const (
	granularityAuto  = "auto"
	granularityRaw   = "raw"
	granularity1Min  = "1min"
	granularity5min  = "5min"
	granularity1Hour = "1h"
)

// SamplesQueryResponse tags rows with their granularity so the frontend knows the row shape.
// Rows is []storage.Sample when granularity=="raw", otherwise []storage.AggregatedRow.
type SamplesQueryResponse struct {
	Granularity string `json:"granularity"`
	Rows        any    `json:"rows"`
}

// RegisterSampleHandlers wires samples.query. See pickGranularity for auto-resolution rules.
func RegisterSampleHandlers(s *Server, store *storage.Client) {
	s.Register("samples.query", bind(func(ctx context.Context, p querySamplesParams) (SamplesQueryResponse, *Error) {
		var zero SamplesQueryResponse

		now := time.Now().UnixMilli()
		if p.ToMs == 0 {
			p.ToMs = now
		}

		if p.FromMs == 0 {
			p.FromMs = p.ToMs - 60*60*1000 // default: last hour
		}

		if p.Limit <= 0 || p.Limit > 50000 {
			p.Limit = 5000
		}

		gran := p.Granularity
		if gran == "" {
			gran = granularityAuto
		}

		if gran == granularityAuto {
			gran = pickGranularity(p.ToMs - p.FromMs)
		}

		switch gran {
		case granularityRaw:
			rows, err := store.Samples.Query(ctx, storage.QuerySamplesParams{
				FromMs:       p.FromMs,
				ToMs:         p.ToMs,
				TargetLabels: p.TargetLabels,
				Limit:        p.Limit,
			})
			if err != nil {
				return zero, internalErr(err)
			}

			return SamplesQueryResponse{Granularity: granularityRaw, Rows: rows}, nil

		case granularity1Min:
			rows, err := store.Samples.Query1Min(ctx, storage.QueryAggregatedParams{
				FromMs:       p.FromMs,
				ToMs:         p.ToMs,
				TargetLabels: p.TargetLabels,
			})
			if err != nil {
				return zero, internalErr(err)
			}

			return SamplesQueryResponse{Granularity: granularity1Min, Rows: rows}, nil

		case granularity5min:
			rows, err := store.Samples.Query5Min(ctx, storage.QueryAggregatedParams{
				FromMs:       p.FromMs,
				ToMs:         p.ToMs,
				TargetLabels: p.TargetLabels,
			})
			if err != nil {
				return zero, internalErr(err)
			}

			return SamplesQueryResponse{Granularity: granularity5min, Rows: rows}, nil

		case granularity1Hour:
			rows, err := store.Samples.Query1H(ctx, storage.QueryAggregatedParams{
				FromMs:       p.FromMs,
				ToMs:         p.ToMs,
				TargetLabels: p.TargetLabels,
			})
			if err != nil {
				return zero, internalErr(err)
			}

			return SamplesQueryResponse{Granularity: granularity1Hour, Rows: rows}, nil

		default:
			return zero, &Error{Code: "invalid_params", Message: "unknown granularity: " + gran}
		}
	}))
}

// pickGranularity targets ~60-600 points per series so recharts stays smooth.
//
// At raw 2.5s cadence × 13 default targets, a 1-hour window is already 1,440
// points per series — past the legibility sweet spot. Drop the raw cutoff to
// 30 minutes so anything ≥1h falls into 1-min aggregation (60 points/hour).
//
//	≤ 30m → raw, ≤ 6h → 1min, ≤ 7d → 5min, otherwise 1h.
func pickGranularity(windowMs int64) string {
	const (
		thirtyMin = int64(30 * 60 * 1000)
		sixHours  = int64(6 * 60 * 60 * 1000)
		sevenDays = int64(7 * 24 * 60 * 60 * 1000)
	)

	switch {
	case windowMs <= thirtyMin:
		return granularityRaw
	case windowMs <= sixHours:
		return granularity1Min
	case windowMs <= sevenDays:
		return granularity5min
	default:
		return granularity1Hour
	}
}

type querySamplesParams struct {
	FromMs       int64    `json:"from_ms,omitempty"`
	ToMs         int64    `json:"to_ms,omitempty"`
	TargetLabels []string `json:"target_labels,omitempty"`
	Limit        int      `json:"limit,omitempty"`
	Granularity  string   `json:"granularity,omitempty"`
}
