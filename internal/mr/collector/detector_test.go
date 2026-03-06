package collector

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectMRType(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected string
		wantErr  bool
	}{
		{name: "MRO lowercase", filename: "mr_mro_20240101.xml", expected: "mro"},
		{name: "MRS lowercase", filename: "mr_mrs_20240101.xml", expected: "mrs"},
		{name: "MRE lowercase", filename: "mr_mre_20240101.xml", expected: "mre"},
		{name: "MRO uppercase", filename: "FDD-LTE_MRO_YYYYMMDD.xml", expected: "mro"},
		{name: "MRS uppercase", filename: "FDD-LTE_MRS_YYYYMMDD.xml", expected: "mrs"},
		{name: "MRE uppercase", filename: "FDD-LTE_MRE_YYYYMMDD.xml", expected: "mre"},
		{name: "MRO with path", filename: "/data/mr/mro_report.xml", expected: "mro"},
		{name: "unknown type", filename: "data_report.xml", wantErr: true},
		{name: "empty filename", filename: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DetectMRType(tt.filename)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidMRTypes(t *testing.T) {
	types := ValidMRTypes()
	assert.Len(t, types, 3)
	assert.Contains(t, types, MRTypeMRO)
	assert.Contains(t, types, MRTypeMRS)
	assert.Contains(t, types, MRTypeMRE)
}
