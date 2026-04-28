package ipc

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/sid-technologies/vigil/internal/reports"
	"github.com/sid-technologies/vigil/internal/storage"
)

// Cap report windows at 90 days; otherwise a "last 10 years" request scans ~30M raw rows.
const maxReportWindowMs int64 = 90 * 24 * 60 * 60 * 1000

// RegisterReportHandlers wires report.generate.
func RegisterReportHandlers(s *Server, store *storage.Client) {
	s.Register("report.generate", func(ctx context.Context, params json.RawMessage) (any, *Error) {
		var p reportGenerateParams

		err := json.Unmarshal(params, &p)
		if err != nil {
			return nil, &Error{Code: "invalid_params", Message: err.Error()}
		}

		if p.OutDir == "" {
			return nil, &Error{Code: "invalid_params", Message: "out_dir required"}
		}

		if p.FromMs == 0 || p.ToMs == 0 || p.ToMs <= p.FromMs {
			return nil, &Error{Code: "invalid_params", Message: "valid from_ms < to_ms required"}
		}

		if p.ToMs-p.FromMs > maxReportWindowMs {
			return nil, &Error{Code: "window_too_large", Message: "max 90 days"}
		}

		formats := parseFormats(p.Formats)
		if formats == 0 {
			return nil, &Error{Code: "invalid_params", Message: "at least one format required"}
		}

		res, err := reports.Generate(ctx, store, reports.GenerateParams{
			OutDir:   p.OutDir,
			FromMs:   p.FromMs,
			ToMs:     p.ToMs,
			Targets:  p.Targets,
			Formats:  formats,
			BaseName: p.BaseName,
		})
		if err != nil {
			return nil, &Error{Code: "internal", Message: err.Error()}
		}

		return res, nil
	})
}

func parseFormats(in []string) reports.Format {
	var f reports.Format

	for _, name := range in {
		switch strings.ToLower(name) {
		case "csv":
			f |= reports.FormatCSV
		case "json":
			f |= reports.FormatJSON
		case "html":
			f |= reports.FormatHTML
		default:
			// unknown formats silently dropped
		}
	}

	return f
}

type reportGenerateParams struct {
	OutDir   string   `json:"out_dir"`
	FromMs   int64    `json:"from_ms"`
	ToMs     int64    `json:"to_ms"`
	Targets  []string `json:"targets,omitempty"`
	Formats  []string `json:"formats"` // ["csv", "json", "html"] in any combination
	BaseName string   `json:"base_name,omitempty"`
}
