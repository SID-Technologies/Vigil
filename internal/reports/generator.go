// Package reports renders shareable artifacts (CSV / JSON / HTML) from raw
// probe samples.
//
// Used by the IPC `report.generate` method when the user clicks "Generate
// report" on the History page. Output goes to a user-chosen directory; we
// never write to anywhere the user didn't explicitly ask for.
//
// HTML format mirrors the legacy Python tool's report.html.j2 — same
// structure, ported to Go's html/template, restyled to match Vigil's
// Night Watch palette. Chart.js is loaded from the public CDN, so an
// offline reader sees a no-charts fallback (the tables remain readable).
package reports

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sid-technologies/vigil/db/ent"
	"github.com/sid-technologies/vigil/internal/storage"
)

// Format names accepted by the generator. Combine with bitwise OR.
type Format int

const (
	FormatCSV  Format = 1 << iota
	FormatJSON
	FormatHTML
)

// GenerateParams configures one report run.
type GenerateParams struct {
	OutDir   string
	FromMs   int64
	ToMs     int64
	Targets  []string // optional filter; empty = all
	Formats  Format
	BaseName string // optional, defaults to "vigil-report-<timestamp>"
}

// Result lists the paths of every artifact produced.
type Result struct {
	Paths []string `json:"paths"`
}

// Generate writes the requested formats. Always returns the absolute paths
// of every successfully written file so the IPC caller can show "wrote N
// files at /path/to/foo" toasts.
//
// Partial success is fine — if HTML rendering fails but CSV/JSON were
// already written, we keep them and surface a non-fatal error message.
func Generate(ctx context.Context, client *ent.Client, p GenerateParams) (Result, error) {
	if p.OutDir == "" {
		return Result{}, fmt.Errorf("out_dir is required")
	}
	if p.FromMs == 0 || p.ToMs == 0 || p.ToMs <= p.FromMs {
		return Result{}, fmt.Errorf("invalid time window")
	}
	if p.Formats == 0 {
		return Result{}, fmt.Errorf("at least one format must be selected")
	}

	if err := os.MkdirAll(p.OutDir, 0o755); err != nil {
		return Result{}, fmt.Errorf("create out dir: %w", err)
	}

	base := p.BaseName
	if base == "" {
		base = fmt.Sprintf("vigil-report-%s", time.Now().Format("2006-01-02T15-04"))
	}

	// Reports always pull RAW samples. They're meant for the "show the ISP
	// hard evidence" use case where every probe matters. If the window is
	// large the response will be too — but the IPC handler validates window
	// size before calling us.
	samples, err := storage.QuerySamples(ctx, client, storage.QuerySamplesParams{
		FromMs:       p.FromMs,
		ToMs:         p.ToMs,
		TargetLabels: p.Targets,
		Limit:        1_000_000,
	})
	if err != nil {
		return Result{}, fmt.Errorf("load samples: %w", err)
	}

	wifi, err := storage.QueryWifiSamples(ctx, client, p.FromMs, p.ToMs)
	if err != nil {
		return Result{}, fmt.Errorf("load wifi samples: %w", err)
	}

	outages, err := storage.QueryOutages(ctx, client, storage.QueryOutagesParams{
		FromMs: p.FromMs,
		ToMs:   p.ToMs,
	})
	if err != nil {
		return Result{}, fmt.Errorf("load outages: %w", err)
	}

	rep := buildReport(p.FromMs, p.ToMs, samples, wifi, outages)

	var paths []string
	if p.Formats&FormatCSV != 0 {
		path := filepath.Join(p.OutDir, base+".csv")
		if err := writeCSV(path, samples); err != nil {
			return Result{Paths: paths}, fmt.Errorf("csv: %w", err)
		}
		paths = append(paths, path)
	}
	if p.Formats&FormatJSON != 0 {
		path := filepath.Join(p.OutDir, base+".json")
		if err := writeJSON(path, rep); err != nil {
			return Result{Paths: paths}, fmt.Errorf("json: %w", err)
		}
		paths = append(paths, path)
	}
	if p.Formats&FormatHTML != 0 {
		path := filepath.Join(p.OutDir, base+".html")
		if err := writeHTML(path, rep); err != nil {
			return Result{Paths: paths}, fmt.Errorf("html: %w", err)
		}
		paths = append(paths, path)
	}

	return Result{Paths: paths}, nil
}

// writeJSON serializes the structured report payload (summary + raw samples
// + outages + wifi). Pretty-printed for human inspection.
func writeJSON(path string, rep *report) error {
	f, err := os.Create(path)
	if err != nil {
		return err //nolint:wrapcheck
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(rep) //nolint:wrapcheck
}
