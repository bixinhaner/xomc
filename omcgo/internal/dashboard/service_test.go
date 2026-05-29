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

// ---------------------------------------------------------------------------
// coalesceDeviceName
// ---------------------------------------------------------------------------

func TestCoalesceDeviceName(t *testing.T) {
	tests := []struct {
		name     string
		inputName *string
		inputSN   string
		want     string
	}{
		{
			name:     "有设备名称时返回设备名称",
			inputName: strPtr("基站-A区"),
			inputSN:   "SN123456789012",
			want:     "基站-A区",
		},
		{
			name:     "设备名称为空字符串时使用SN",
			inputName: strPtr(""),
			inputSN:   "SN123456789012",
			want:     "...56789012",
		},
		{
			name:     "设备名称为nil时使用SN",
			inputName: nil,
			inputSN:   "SN123456789012",
			want:     "...56789012",
		},
		{
			name:     "SN长度大于阈值时截断",
			inputName: nil,
			inputSN:   "SN12345678901",  // 13 chars
			want:     "...45678901",
		},
		{
			name:     "SN长度小于阈值时完整返回",
			inputName: nil,
			inputSN:   "SN12345678",
			want:     "SN12345678",
		},
		{
			name:     "SN长度远大于阈值时截断",
			inputName: nil,
			inputSN:   "SN12345678901234",
			want:     "...78901234",
		},
		{
			name:     "SN为空字符串时返回空",
			inputName: nil,
			inputSN:   "",
			want:     "",
		},
		{
			name:     "SN刚好等于阈值长度时不截断",
			inputName: nil,
			inputSN:   "123456789012",  // exactly 12 chars
			want:     "123456789012",
		},
		{
			name:     "SN刚好超过阈值长度一位时截断",
			inputName: nil,
			inputSN:   "1234567890123",  // 13 chars
			want:     "...67890123",
		},
		{
			name:     "中文设备名称",
			inputName: strPtr("北京基站-001"),
			inputSN:   "BJ001SN1234567",
			want:     "北京基站-001",
		},
		{
			name:     "设备名称包含特殊字符",
			inputName: strPtr("基站-A区_测试"),
			inputSN:   "SN123456789012",
			want:     "基站-A区_测试",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := coalesceDeviceName(tt.inputName, tt.inputSN)
			assert.Equal(t, tt.want, got, "coalesceDeviceName() mismatch")
		})
	}
}

// strPtr is a helper function to create a string pointer.
func strPtr(s string) *string {
	return &s
}
