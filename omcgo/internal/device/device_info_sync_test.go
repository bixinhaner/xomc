package device

import (
	"testing"
)

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
