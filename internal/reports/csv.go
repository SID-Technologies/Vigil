package reports

import (
	"encoding/csv"
	"os"
	"strconv"
	"time"

	"github.com/sid-technologies/vigil/internal/storage"
)

// writeCSV produces one row per probe — flat, spreadsheet-friendly. Same
// columns as the legacy Python tool's CSV output so existing downstream
// scripts (if any) keep working.
func writeCSV(path string, samples []storage.Sample) error {
	f, err := os.Create(path)
	if err != nil {
		return err //nolint:wrapcheck
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	if err := w.Write(csvHeader); err != nil {
		return err //nolint:wrapcheck
	}

	for _, s := range samples {
		ts := time.UnixMilli(s.TsUnixMs).UTC()
		port := ""
		if s.TargetPort != nil {
			port = strconv.Itoa(*s.TargetPort)
		}
		rtt := ""
		if s.RTTMs != nil {
			rtt = strconv.FormatFloat(*s.RTTMs, 'f', 2, 64)
		}
		errStr := ""
		if s.Error != nil {
			errStr = *s.Error
		}
		successStr := "no"
		if s.Success {
			successStr = "yes"
		}
		row := []string{
			ts.Format(time.RFC3339),
			ts.Local().Format("2006-01-02 15:04:05"),
			s.TargetLabel,
			s.TargetKind,
			s.TargetHost,
			port,
			successStr,
			rtt,
			errStr,
		}
		if err := w.Write(row); err != nil {
			return err //nolint:wrapcheck
		}
	}
	return nil
}

var csvHeader = []string{
	"timestamp_utc",
	"timestamp_local",
	"target_label",
	"target_kind",
	"target_host",
	"target_port",
	"success",
	"rtt_ms",
	"error",
}
