package tr069

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestXSDType(t *testing.T) {
	tests := map[string]string{
		"BOOLEAN":   "xsd:boolean",
		"U_INT":     "xsd:unsignedInt",
		"INT":       "xsd:int",
		"DATE_TIME": "xsd:dateTime",
		"U_LONG":    "xsd:unsignedLong",
		"STRING":    "xsd:string",
		"":          "xsd:string",
	}
	for dataType, want := range tests {
		assert.Equal(t, want, XSDType(dataType), dataType)
	}
}

func TestNormalizeValueForPath(t *testing.T) {
	assert.Equal(t, "1", NormalizeValueForPath("Device.X.Enable", "true", "BOOLEAN"))
	assert.Equal(t, "0", NormalizeValueForPath("Device.X.Enable", "false", "BOOLEAN"))
	assert.Equal(t, "text", NormalizeValueForPath("Device.X.Name", "text", "STRING"))
	assert.Equal(t, "1", NormalizeValueForPath(signallingTraceEnablePath, "true", "STRING"))
	assert.Equal(t, "xsd:string", XSDTypeForPath(signallingTraceEnablePath, "BOOLEAN"))
}
