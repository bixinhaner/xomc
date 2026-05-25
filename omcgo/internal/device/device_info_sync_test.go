package device

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDeriveEnbID covers Phase 3 (设计文档 §4.2 Layer C) — eNodeB ID 派生。
// LTE 28-bit ECI = 20-bit eNB-ID + 8-bit Cell-ID，即 enb_id = eci >> 8。
func TestDeriveEnbID(t *testing.T) {
	tests := []struct {
		name string
		eci  string
		want string
		ok   bool
	}{
		{"valid ECI 654321", "654321", "2555", true}, // 654321 >> 8 = 2555 (0x9FBF1 >> 8 = 0x9FB)
		{"single cell ECI 256", "256", "1", true},
		{"empty input", "", "", false},
		{"non-numeric", "abc", "", false},
		{"zero", "0", "", false},
		{"negative literal", "-1", "", false},
		{"max LTE ECI", "268435455", "1048575", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := deriveEnbID(tt.eci)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.ok, ok)
		})
	}
}

// TestDeriveNetworkModel 验证 TDD / FDD 判定逻辑。
func TestDeriveNetworkModel(t *testing.T) {
	cases := []struct {
		name  string
		paths map[string]string
		want  string
		ok    bool
	}{
		{
			name: "TDD with SubFrameAssignment",
			paths: map[string]string{
				"Device.Services.FAPService.1.CellConfig.LTE.RAN.PHY.TDDFrame.SubFrameAssignment": "2",
			},
			want: "TDD", ok: true,
		},
		{
			name: "FDD subtree present",
			paths: map[string]string{
				"Device.Services.FAPService.1.CellConfig.LTE.RAN.PHY.FDDFrame.SubFrameAssignment": "0",
			},
			want: "FDD", ok: true,
		},
		{
			name:  "no PHY frame path",
			paths: map[string]string{"Device.DeviceInfo.SoftwareVersion": "1.0"},
			want:  "", ok: false,
		},
		{
			name:  "empty paramValues",
			paths: map[string]string{},
			want:  "", ok: false,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := deriveNetworkModel(tt.paths)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.ok, ok)
		})
	}
}

// TestLookupGPSHeight 验证 GPS 高度多 path 优先级（Baicells 拼写优先）。
func TestLookupGPSHeight(t *testing.T) {
	cases := []struct {
		name  string
		paths map[string]string
		want  string
		ok    bool
	}{
		{
			name:  "altidute (Baicells spelling) wins",
			paths: map[string]string{"Device.FAP.GPS.altidute": "170", "Device.FAP.GPS.Altitude": "999"},
			want:  "170", ok: true,
		},
		{
			name:  "fallback to Altitude when altidute missing",
			paths: map[string]string{"Device.FAP.GPS.Altitude": "200"},
			want:  "200", ok: true,
		},
		{
			name:  "fallback to Height when both missing",
			paths: map[string]string{"Device.FAP.GPS.Height": "150"},
			want:  "150", ok: true,
		},
		{
			name:  "empty value is skipped",
			paths: map[string]string{"Device.FAP.GPS.altidute": "", "Device.FAP.GPS.Altitude": "100"},
			want:  "100", ok: true,
		},
		{
			name:  "all missing",
			paths: map[string]string{},
			want:  "", ok: false,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := lookupGPSHeight(tt.paths)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.ok, ok)
		})
	}
}

func TestParseRunTimeToSeconds(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int64
	}{
		{
			name:  "full format with days hours minutes",
			input: "40d 4h 58m",
			want:  40*86400 + 4*3600 + 58*60, // 3482308
		},
		{
			name:  "hours and minutes only",
			input: "4h 58m",
			want:  4*3600 + 58*60, // 17880
		},
		{
			name:  "minutes only",
			input: "58m",
			want:  58 * 60, // 3480
		},
		{
			name:  "full format with seconds",
			input: "1d 2h 3m 4s",
			want:  86400 + 2*3600 + 3*60 + 4, // 93784
		},
		{
			name:  "days only",
			input: "10d",
			want:  10 * 86400, // 864000
		},
		{
			name:  "hours only",
			input: "24h",
			want:  24 * 3600, // 86400
		},
		{
			name:  "seconds only",
			input: "30s",
			want:  30,
		},
		{
			name:  "zero value",
			input: "0d 0h 0m",
			want:  0,
		},
		{
			name:  "empty string",
			input: "",
			want:  0,
		},
		{
			name:  "invalid format",
			input: "invalid",
			want:  0,
		},
		{
			name:  "large values",
			input: "365d 23h 59m 59s",
			want:  365*86400 + 23*3600 + 59*60 + 59, // 31622399
		},
		{
			name:  "with extra spaces",
			input: "  10d   5h  ",
			want:  10*86400 + 5*3600, // 903600
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseRunTimeToSeconds(tt.input)
			if got != tt.want {
				t.Errorf("parseRunTimeToSeconds(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}
