package task

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseDateFromKey(t *testing.T) {
	cases := []struct {
		name   string
		key    string
		wantOK bool
		want   time.Time
	}{
		{
			name:   "standard cmcc key",
			key:    "cmcc/2026/05/01/SN001/file.xml",
			wantOK: true,
			want:   time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:   "ctcc key",
			key:    "ctcc/2025/12/31/SN999/foo.xml",
			wantOK: true,
			want:   time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
		},
		{
			name:   "missing date segment",
			key:    "cmcc/SN001/foo.xml",
			wantOK: false,
		},
		{
			name:   "invalid month",
			key:    "cmcc/2026/13/01/SN001/x.xml",
			wantOK: false,
		},
		{
			name:   "empty key",
			key:    "",
			wantOK: false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := parseDateFromKey(c.key)
			assert.Equal(t, c.wantOK, ok)
			if c.wantOK {
				assert.True(t, got.Equal(c.want), "got %v, want %v", got, c.want)
			}
		})
	}
}

func TestCleanerConfigDefaults(t *testing.T) {
	got := CleanerConfig{}.Defaults()
	assert.Equal(t, 3, got.SaveDays)
	assert.Equal(t, "@daily", got.Schedule)

	got = CleanerConfig{SaveDays: 7, Schedule: "@hourly"}.Defaults()
	assert.Equal(t, 7, got.SaveDays)
	assert.Equal(t, "@hourly", got.Schedule)
}
