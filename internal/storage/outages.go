package storage

import (
	"context"

	"github.com/sid-technologies/vigil/db/ent"
	"github.com/sid-technologies/vigil/db/ent/outage"
)

// Outage matches the outage:start / outage:end event payload so the frontend
// has one type definition.
type Outage struct {
	ID                  string         `json:"id"`
	Scope               string         `json:"scope"`
	StartTSUnixMs       int64          `json:"start_ts_unix_ms"`
	EndTSUnixMs         *int64         `json:"end_ts_unix_ms,omitempty"`
	ConsecutiveFailures int            `json:"consecutive_failures"`
	Errors              map[string]int `json:"errors,omitempty"`
}

// QueryOutagesParams scopes a list-outages call.
type QueryOutagesParams struct {
	FromMs   int64
	ToMs     int64
	Scope    string
	OnlyOpen bool
}

// QueryOutages returns outages overlapping [fromMs, toMs] — either started in
// the window, or started before but still active at fromMs. Catches the
// in-progress outage that began before the window so the "outage in progress"
// badge stays accurate.
func (s *Store) QueryOutages(ctx context.Context, p QueryOutagesParams) ([]Outage, error) {
	q := s.client.Outage.Query().
		Order(ent.Desc(outage.FieldStartTsUnixMs))

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
		return nil, err //nolint:wrapcheck // wrapped at IPC boundary
	}

	out := make([]Outage, 0, len(rows))
	for _, r := range rows {
		o := Outage{
			ID:                  r.ID,
			Scope:               r.Scope,
			StartTSUnixMs:       r.StartTsUnixMs,
			ConsecutiveFailures: r.ConsecutiveFailures,
			Errors:              r.Errors,
		}
		if r.EndTsUnixMs != nil {
			o.EndTSUnixMs = r.EndTsUnixMs
		}

		out = append(out, o)
	}

	return out, nil
}
