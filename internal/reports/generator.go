// Package reports renders shareable artifacts (CSV / JSON / HTML) from raw probe samples.
package reports

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/sid-technologies/vigil/internal/storage"
	"github.com/sid-technologies/vigil/pkg/errors"
)

// Format selects report outputs. Combine with bitwise OR.
type Format int

// Format flags, OR'd together to select report outputs.
const (
	FormatCSV Format = 1 << iota
	FormatJSON
	FormatHTML
)

const outDirPerm = 0o750

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

// Generate writes the requested formats and returns the paths actually
// written. On partial failure, already-written paths are kept and an error
// is returned so the caller can surface both.
func Generate(ctx context.Context, store *storage.Client, p GenerateParams) (Result, error) {
	if p.OutDir == "" {
		return Result{}, errors.New("out_dir is required")
	}

	if p.FromMs == 0 || p.ToMs == 0 || p.ToMs <= p.FromMs {
		return Result{}, errors.New("invalid time window")
	}

	if p.Formats == 0 {
		return Result{}, errors.New("at least one format must be selected")
	}

	err := os.MkdirAll(p.OutDir, outDirPerm)
	if err != nil {
		return Result{}, errors.Wrap(err, "create out dir")
	}

	base := p.BaseName
	if base == "" {
		base = "vigil-report-" + time.Now().Format("2006-01-02T15-04")
	}

	// Reports pull raw samples; the IPC handler validates window size first.
	samples, err := store.Samples.Query(ctx, storage.QuerySamplesParams{
		FromMs:       p.FromMs,
		ToMs:         p.ToMs,
		TargetLabels: p.Targets,
		Limit:        1_000_000,
	})
	if err != nil {
		return Result{}, errors.Wrap(err, "load samples")
	}

	wifi, err := store.Wifi.Query(ctx, p.FromMs, p.ToMs)
	if err != nil {
		return Result{}, errors.Wrap(err, "load wifi samples")
	}

	outages, err := store.Outages.Query(ctx, storage.QueryOutagesParams{
		FromMs: p.FromMs,
		ToMs:   p.ToMs,
	})
	if err != nil {
		return Result{}, errors.Wrap(err, "load outages")
	}

	rep := buildReport(p.FromMs, p.ToMs, samples, wifi, outages)

	var paths []string

	if p.Formats&FormatCSV != 0 {
		path := filepath.Join(p.OutDir, base+".csv")

		err := writeCSV(path, samples)
		if err != nil {
			return Result{Paths: paths}, errors.Wrap(err, "csv")
		}

		paths = append(paths, path)
	}

	if p.Formats&FormatJSON != 0 {
		path := filepath.Join(p.OutDir, base+".json")

		err := writeJSON(path, rep)
		if err != nil {
			return Result{Paths: paths}, errors.Wrap(err, "json")
		}

		paths = append(paths, path)
	}

	if p.Formats&FormatHTML != 0 {
		path := filepath.Join(p.OutDir, base+".html")

		err := writeHTML(path, rep)
		if err != nil {
			return Result{Paths: paths}, errors.Wrap(err, "html")
		}

		paths = append(paths, path)
	}

	return Result{Paths: paths}, nil
}

func writeJSON(path string, rep *report) (err error) {
	f, err := os.Create(path) //nolint:gosec // path supplied by user via report export UI
	if err != nil {
		return err //nolint:wrapcheck // wrapped by caller in Generate
	}

	defer func() {
		cerr := f.Close()
		if err == nil && cerr != nil {
			err = cerr
		}
	}()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")

	return enc.Encode(rep) //nolint:wrapcheck // wrapped by caller in Generate
}
