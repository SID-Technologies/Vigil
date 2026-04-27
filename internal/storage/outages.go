package storage

import (
	"context"

	"github.com/sid-technologies/vigil/db/ent"
	"github.com/sid-technologies/vigil/db/ent/outage"
)

// Outage is the IPC-shape projection of an Outage row. Mirrors the
// outage:start / outage:end event payload exactly so the frontend has one
// type definition.
type Outage struct {
	ID                  string         `json:"id"`
	Scope               string         `json:"scope"`
	StartTsUnixMs       int64          `json:"start_ts_unix_ms"`
	EndTsUnixMs         *int64         `json:"end_ts_unix_ms,omitempty"`
	ConsecutiveFailures int            `json:"consecutive_failures"`
	Errors              map[string]int `json:"errors,omitempty"`
}

// QueryOutagesParams scopes an outages.list call.
type QueryOutagesParams struct {
	FromMs    int64
	ToMs      int64
	Scope     string // optional — if non-empty, filter to this scope
	OnlyOpen  bool   // if true, only return outages with end_ts_unix_ms = null
}

// QueryOutages returns outages overlapping [fromMs, toMs]. Definition of
// "overlap": the outage either started in the window OR was still active
// at fromMs. This way a long outage that started outside the window is
// still surfaced — important for the dashboard's "outage in progress" badge.
func QueryOutages(ctx context.Context, client *ent.Client, p QueryOutagesParams) ([]Outage, error) {
	q := client.Outage.Query().
		Order(ent.Desc(outage.FieldStartTsUnixMs))

	// Started within window OR ongoing at window start (end_ts >= fromMs OR null).
	q = q.Where(
		outage.Or(
			outage.And(
				outage.StartTsUnixMsGTE(p.FromMs),
				outage.StartTsUnixMsLTE(p.ToMs),
			),
			outage.And(
				outage.StartTsUnixMsLT(p.FromMs),
				outage.Or(
					outage.EndTsUnixMsIsNil(),
					outage.EndTsUnixMsGTE(p.FromMs),
				),
			),
		),
	)

	if p.Scope != "" {
		q = q.Where(outage.ScopeEQ(p.Scope))
	}
	if p.OnlyOpen {
		q = q.Where(outage.EndTsUnixMsIsNil())
	}

	rows, err := q.All(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	out := make([]Outage, 0, len(rows))
	for _, r := range rows {
		o := Outage{
			ID:                  r.ID,
			Scope:               r.Scope,
			StartTsUnixMs:       r.StartTsUnixMs,
			ConsecutiveFailures: r.ConsecutiveFailures,
			Errors:              r.Errors,
		}
		if r.EndTsUnixMs != nil {
			o.EndTsUnixMs = r.EndTsUnixMs
		}
		out = append(out, o)
	}
	return out, nil
}
