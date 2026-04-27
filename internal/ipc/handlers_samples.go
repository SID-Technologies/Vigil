package ipc

import (
	"context"
	"encoding/json"
	"time"

	"github.com/sid-technologies/vigil/db/ent"
	"github.com/sid-technologies/vigil/internal/storage"
)

// Granularity strings exchanged on the wire.
const (
	granularityAuto  = "auto"
	granularityRaw   = "raw"
	granularity5min  = "5min"
	granularity1Hour = "1h"
)

// SamplesQueryResponse wraps the result of samples.query so the frontend
// knows which row shape it received without inspecting field presence.
//
// When granularity == "raw", Rows is []storage.Sample.
// When granularity is "5min" or "1h", Rows is []storage.AggregatedRow.
type SamplesQueryResponse struct {
	Granularity string `json:"granularity"`
	Rows        any    `json:"rows"`
}

// RegisterSampleHandlers wires samples.query onto the IPC server.
//
// Granularity dispatch:
//   - "raw"  : always raw, regardless of window size. Beware: 7 days of raw
//              for 13 targets is ~3M rows. Frontend should not request raw
//              over wide windows.
//   - "5min" : always 5-min buckets.
//   - "1h"   : always 1-hour buckets.
//   - "auto" : window <= 2h → raw, <= 7d → 5min, > 7d → 1h.
func RegisterSampleHandlers(s *Server, client *ent.Client) {
	s.Register("samples.query", func(ctx context.Context, params json.RawMessage) (any, *Error) {
		var p querySamplesParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &Error{Code: "invalid_params", Message: err.Error()}
		}

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
			rows, err := storage.QuerySamples(ctx, client, storage.QuerySamplesParams{
				FromMs:       p.FromMs,
				ToMs:         p.ToMs,
				TargetLabels: p.TargetLabels,
				Limit:        p.Limit,
			})
			if err != nil {
				return nil, &Error{Code: "internal", Message: err.Error()}
			}
			return SamplesQueryResponse{Granularity: granularityRaw, Rows: rows}, nil

		case granularity5min:
			rows, err := storage.Query5minSamples(ctx, client, storage.QueryAggregatedParams{
				FromMs:       p.FromMs,
				ToMs:         p.ToMs,
				TargetLabels: p.TargetLabels,
			})
			if err != nil {
				return nil, &Error{Code: "internal", Message: err.Error()}
			}
			return SamplesQueryResponse{Granularity: granularity5min, Rows: rows}, nil

		case granularity1Hour:
			rows, err := storage.Query1hSamples(ctx, client, storage.QueryAggregatedParams{
				FromMs:       p.FromMs,
				ToMs:         p.ToMs,
				TargetLabels: p.TargetLabels,
			})
			if err != nil {
				return nil, &Error{Code: "internal", Message: err.Error()}
			}
			return SamplesQueryResponse{Granularity: granularity1Hour, Rows: rows}, nil

		default:
			return nil, &Error{Code: "invalid_params", Message: "unknown granularity: " + gran}
		}
	})
}

// pickGranularity is the auto-resolution heuristic. Mirrors the windowing
// design captured in CLAUDE.md.
func pickGranularity(windowMs int64) string {
	const twoHours = int64(2 * 60 * 60 * 1000)
	const sevenDays = int64(7 * 24 * 60 * 60 * 1000)
	switch {
	case windowMs <= twoHours:
		return granularityRaw
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
