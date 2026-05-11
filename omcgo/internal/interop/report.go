package interop

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// ValidationReport summarizes the comparison between a device's actual parameters
// and the expected parameters from its resolved data model definition.
type ValidationReport struct {
	DeviceID       string          `json:"device_id"`
	DeviceSN       string          `json:"device_sn"`
	ModelVersion   string          `json:"model_version"`
	TotalParams    int             `json:"total_params"`
	MatchedParams  int             `json:"matched_params"`
	MismatchParams []ParamMismatch `json:"mismatch_params"`
	MissingParams  []string        `json:"missing_params"`
	ExtraParams    []string        `json:"extra_params"`
	Score          float64         `json:"score"`
	CreatedAt      time.Time       `json:"created_at"`
}

// ParamMismatch records a single parameter whose actual value or type on the
// device differs from what the data model definition expects.
type ParamMismatch struct {
	Path     string `json:"path"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Type     string `json:"type"` // "type_mismatch" or "writable_mismatch"
}

// --- T-0115 conformance run report export -----------------------------------

// ReportFormat enumerates supported export formats for /interop/run/report.
// CSV is the MVP (T-0115 Phase 1); PDF / markdown are reserved for future
// iterations and intentionally rejected by the handler today.
type ReportFormat string

const (
	ReportFormatCSV ReportFormat = "csv"
)

// SupportedReportFormats lists every format the runner can currently emit.
func SupportedReportFormats() []ReportFormat {
	return []ReportFormat{ReportFormatCSV}
}

// IsValid reports whether the format is one the runner can emit today.
func (f ReportFormat) IsValid() bool {
	for _, v := range SupportedReportFormats() {
		if f == v {
			return true
		}
	}
	return false
}

// csvReportColumns is the canonical CSV column order for interop run reports.
// Stable for downstream consumers (operator QA tooling that ingests these
// CSVs); append new columns at the end only — never reorder or drop.
var csvReportColumns = []string{
	"device_sn",
	"test_case_id",
	"test_name",
	"category",
	"passed",
	"expected_outcome",
	"duration_ms",
	"details",
	"error",
}

// WriteCSVReport serialises a slice of TestResult as a CSV report to w. The
// first row is the header row (csvReportColumns); subsequent rows contain one
// test result each. deviceSN is included on every row so reports for multiple
// devices can be concatenated downstream.
func WriteCSVReport(w io.Writer, deviceSN string, results []TestResult) error {
	cw := csv.NewWriter(w)
	if err := cw.Write(csvReportColumns); err != nil {
		return fmt.Errorf("write csv header: %w", err)
	}
	for _, r := range results {
		row := []string{
			deviceSN,
			r.TestCaseID,
			r.TestName,
			string(r.Category),
			strconv.FormatBool(r.Passed),
			r.ExpectedOutcome,
			strconv.FormatInt(r.Duration.Milliseconds(), 10),
			r.Details,
			r.Error,
		}
		if err := cw.Write(row); err != nil {
			return fmt.Errorf("write csv row for %s: %w", r.TestCaseID, err)
		}
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		return fmt.Errorf("csv flush: %w", err)
	}
	return nil
}

// ReportFilename builds a safe attachment filename for a downloaded report.
// Non-alphanumeric runes in deviceSN are replaced with underscores so the
// filename never contains characters that complicate Content-Disposition
// header parsing across browsers.
func ReportFilename(deviceSN string, format ReportFormat, now time.Time) string {
	sn := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '-', r == '_':
			return r
		default:
			return '_'
		}
	}, deviceSN)
	if sn == "" {
		sn = "device"
	}
	return fmt.Sprintf("interop-report-%s-%s.%s", sn, now.UTC().Format("20060102-150405"), string(format))
}
