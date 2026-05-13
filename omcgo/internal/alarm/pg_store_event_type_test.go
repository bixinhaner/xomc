package alarm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizedEventTypeAliases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "current page numeric code maps to communication bucket",
			input:    "30000",
			expected: []string{"30000", "communication", "communications", "communicationalarm", "communicationsalarm"},
		},
		{
			name:     "historical page enum maps to device bucket",
			input:    "device",
			expected: []string{"30003", "device", "equipment", "devicealarm", "equipmentalarm"},
		},
		{
			name:     "legacy tr069 string maps to qos bucket",
			input:    "Quality Of Service Alarm",
			expected: []string{"30001", "qualityofservice", "qualityofservicealarm"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, normalizedEventTypeAliases(tt.input))
		})
	}
}

func TestNormalizeEventTypeToken(t *testing.T) {
	assert.Equal(t, "qualityofservicealarm", normalizeEventTypeToken(" Quality Of Service Alarm "))
	assert.Equal(t, "processingerroralarm", normalizeEventTypeToken("processing-error_alarm"))
}

func TestNormalizedEventTypeExprIncludesAliases(t *testing.T) {
	sqlizer := normalizedEventTypeExpr("event_type", "device")
	sql, args, err := sqlizer.ToSql()
	require.NoError(t, err)
	assert.Contains(t, sql, "LOWER(COALESCE(event_type, ''))")
	assert.Equal(t, []interface{}{"30003", "device", "equipment", "devicealarm", "equipmentalarm"}, args)
}