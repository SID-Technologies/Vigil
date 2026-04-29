//nolint:testpackage // whitebox test for unexported parseFormats
package ipc

import (
	"testing"

	"github.com/sid-technologies/vigil/internal/reports"
)

func TestParseFormats(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   []string
		want reports.Format
	}{
		{"empty", nil, 0},
		{"only_unknown", []string{"pdf", "xlsx"}, 0},
		{"single_csv", []string{"csv"}, reports.FormatCSV},
		{"case_insensitive", []string{"CSV", "Json"}, reports.FormatCSV | reports.FormatJSON},
		{"all_three", []string{"csv", "json", "html"}, reports.FormatCSV | reports.FormatJSON | reports.FormatHTML},
		{"unknown_silently_dropped", []string{"csv", "pdf"}, reports.FormatCSV},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := parseFormats(tc.in)
			if got != tc.want {
				t.Fatalf("parseFormats(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}
