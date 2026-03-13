package dashboard

import (
	"testing"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// parseKPINames
// ---------------------------------------------------------------------------

func TestParseKPINames_Empty(t *testing.T) {
	got := parseKPINames("")
	assert.Nil(t, got)
}

func TestParseKPINames_Single(t *testing.T) {
	got := parseKPINames("rrc_success_rate")
	assert.Equal(t, []string{"rrc_success_rate"}, got)
}

func TestParseKPINames_Multiple(t *testing.T) {
	got := parseKPINames("a,b,c")
	assert.Equal(t, []string{"a", "b", "c"}, got)
}

func TestParseKPINames_Whitespace(t *testing.T) {
	got := parseKPINames(" a , b , c ")
	assert.Equal(t, []string{"a", "b", "c"}, got)
}

func TestParseKPINames_EmptyParts(t *testing.T) {
	got := parseKPINames("a,,b")
	assert.Equal(t, []string{"a", "b"}, got)
}

// ---------------------------------------------------------------------------
// severityToLabel
// ---------------------------------------------------------------------------

func TestSeverityToLabel(t *testing.T) {
	tests := []struct {
		name     string
		severity model.AlarmSeverity
		want     string
	}{
		{"critical", model.AlarmCritical, "critical"},
		{"major", model.AlarmMajor, "major"},
		{"minor", model.AlarmMinor, "minor"},
		{"warning", model.AlarmWarning, "warning"},
		{"unknown value", model.AlarmSeverity(99), "unknown"},
		{"zero value", model.AlarmSeverity(0), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := severityToLabel(tt.severity)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// severityFromLabel
// ---------------------------------------------------------------------------

func TestSeverityFromLabel(t *testing.T) {
	tests := []struct {
		name  string
		label string
		want  int
	}{
		{"critical", "critical", 1},
		{"major", "major", 2},
		{"minor", "minor", 3},
		{"warning", "warning", 4},
		{"unknown label", "info", 5},
		{"empty label", "", 5},
		{"random string", "foobar", 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := severityFromLabel(tt.label)
			assert.Equal(t, tt.want, got)
		})
	}
}
