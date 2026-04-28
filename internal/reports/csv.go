package reports

import (
	"encoding/csv"
	"os"
	"strconv"
	"time"

	"github.com/sid-technologies/vigil/internal/storage"
)

const float64BitSize = 64

// writeCSV emits one row per probe in spreadsheet-friendly columns.
func writeCSV(path string, samples []storage.Sample) (err error) {
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

	w := csv.NewWriter(f)
	defer w.Flush()

	err = w.Write(csvHeader)
	if err != nil {
		return err //nolint:wrapcheck // wrapped by caller in Generate
	}

	for _, s := range samples {
		ts := time.UnixMilli(s.TSUnixMs).UTC()

		port := ""
		if s.TargetPort != nil {
			port = strconv.Itoa(*s.TargetPort)
		}

		rtt := ""
		if s.RTTMs != nil {
			rtt = strconv.FormatFloat(*s.RTTMs, 'f', 2, float64BitSize)
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
			ts.Local().Format("2006-01-02 15:04:05"), //nolint:gosmopolitan // local time is intentional for user-facing reports
			s.TargetLabel,
			s.TargetKind,
			s.TargetHost,
			port,
			successStr,
			rtt,
			errStr,
		}

		err := w.Write(row)
		if err != nil {
			return err //nolint:wrapcheck // wrapped by caller in Generate
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
