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

func TestNormalizedTechnologyAliases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "enb option maps to lte bucket",
			input:    "eNB",
			expected: []string{"enb", "lte", "enodeb"},
		},
		{
			name:     "lte raw value maps to same bucket",
			input:    "lte",
			expected: []string{"enb", "lte", "enodeb"},
		},
		{
			name:     "gnb option maps to nr bucket",
			input:    "gNB",
			expected: []string{"gnb", "nr", "5gnr", "gnodeb"},
		},
		{
			name:     "ups option maps to ups bucket",
			input:    "UPS",
			expected: []string{"ups"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, normalizedTechnologyAliases(tt.input))
		})
	}
}

func TestNormalizedTechnologyExprIncludesAliases(t *testing.T) {
	sqlizer := normalizedTechnologyExpr("technology", []string{"eNB"})
	sql, args, err := sqlizer.ToSql()
	require.NoError(t, err)
	assert.Contains(t, sql, "LOWER(COALESCE(technology, ''))")
	assert.Equal(t, []interface{}{"enb", "lte", "enodeb"}, args)
}

func TestAlarmTechnologyExprDerivesUPSFromProductClass(t *testing.T) {
	expr := alarmTechnologyExpr("alarms_active", "d")

	assert.Contains(t, expr, "d.product_class")
	assert.Contains(t, expr, "alarms_active.alarm_source")
	assert.Contains(t, expr, "LIKE 'UPS%'")
	assert.Contains(t, expr, "NULLIF(alarms_active.technology, '')")
	assert.Contains(t, expr, "NULLIF(d.technology, '')")
}
