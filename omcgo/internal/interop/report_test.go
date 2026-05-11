package interop

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReportFormat_IsValid covers the supported format enum.
func TestReportFormat_IsValid(t *testing.T) {
	tests := []struct {
		f     ReportFormat
		valid bool
	}{
		{ReportFormatCSV, true},
		{ReportFormat("pdf"), false},
		{ReportFormat("markdown"), false},
		{ReportFormat(""), false},
	}
	for _, tt := range tests {
		t.Run(string(tt.f), func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.f.IsValid())
		})
	}
}

// TestWriteCSVReport_HeaderAndRows verifies the canonical CSV column order and
// row payload format. Operator QA tooling depends on this column order being
// stable.
func TestWriteCSVReport_HeaderAndRows(t *testing.T) {
	results := []TestResult{
		{
			TestCaseID: "PROTO-001",
			TestName:   "Inform Required Fields",
			Category:   CategoryProtocol,
			Passed:     true,
			Duration:   12 * time.Millisecond,
			Details:    "All steps passed",
		},
		{
			TestCaseID:      "INF-004",
			TestName:        "Negative — Unknown Inform Field Probe",
			Category:        CategoryInform,
			Passed:          true,
			Duration:        3 * time.Millisecond,
			Details:         "Expected failure observed: field \"some_unknown_inform_field\" is empty",
			ExpectedOutcome: "fail",
		},
	}

	var buf bytes.Buffer
	err := WriteCSVReport(&buf, "AB123", results)
	require.NoError(t, err)

	r := csv.NewReader(strings.NewReader(buf.String()))
	rows, err := r.ReadAll()
	require.NoError(t, err)
	require.Len(t, rows, 3, "expect header + 2 rows")

	// Header — column order is contract.
	assert.Equal(t, csvReportColumns, rows[0])

	// Row 1: PROTO-001 default outcome (empty ExpectedOutcome column).
	assert.Equal(t, "AB123", rows[1][0])
	assert.Equal(t, "PROTO-001", rows[1][1])
	assert.Equal(t, "Inform Required Fields", rows[1][2])
	assert.Equal(t, "protocol", rows[1][3])
	assert.Equal(t, "true", rows[1][4])
	assert.Equal(t, "", rows[1][5], "default case has empty expected_outcome")
	assert.Equal(t, "12", rows[1][6], "duration_ms in milliseconds")

	// Row 2: INF-004 negative case with ExpectedOutcome="fail".
	assert.Equal(t, "AB123", rows[2][0])
	assert.Equal(t, "INF-004", rows[2][1])
	assert.Equal(t, "inform", rows[2][3])
	assert.Equal(t, "true", rows[2][4], "negative case marked passed when failure observed")
	assert.Equal(t, "fail", rows[2][5])
	assert.Contains(t, rows[2][7], "Expected failure observed")
}

// TestWriteCSVReport_EmptyResults still emits a valid header row.
func TestWriteCSVReport_EmptyResults(t *testing.T) {
	var buf bytes.Buffer
	err := WriteCSVReport(&buf, "EMPTY-DEV", nil)
	require.NoError(t, err)

	r := csv.NewReader(strings.NewReader(buf.String()))
	rows, err := r.ReadAll()
	require.NoError(t, err)
	require.Len(t, rows, 1, "header-only output for empty results")
	assert.Equal(t, csvReportColumns, rows[0])
}

// TestReportFilename_ReplacesUnsafeChars guards Content-Disposition safety.
func TestReportFilename_ReplacesUnsafeChars(t *testing.T) {
	now := time.Date(2026, 5, 11, 23, 45, 7, 0, time.UTC)

	tests := []struct {
		name     string
		deviceSN string
		want     string
	}{
		{"alnum", "AB123", "interop-report-AB123-20260511-234507.csv"},
		{"with-dash", "AB-123_X", "interop-report-AB-123_X-20260511-234507.csv"},
		{"special-chars", "AB/123:foo bar*", "interop-report-AB_123_foo_bar_-20260511-234507.csv"},
		{"empty", "", "interop-report-device-20260511-234507.csv"},
		{"unicode", "设备X", "interop-report-__X-20260511-234507.csv"}, // 2 CJK runes → 2 underscores + X
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ReportFilename(tt.deviceSN, ReportFormatCSV, now)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestSupportedReportFormats sanity — locks the supported list to today's MVP.
func TestSupportedReportFormats(t *testing.T) {
	formats := SupportedReportFormats()
	require.Len(t, formats, 1, "MVP supports CSV only; expanding to PDF/markdown must update this test")
	assert.Equal(t, ReportFormatCSV, formats[0])
}
