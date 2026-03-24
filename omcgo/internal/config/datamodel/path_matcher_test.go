package datamodel

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContainsPlaceholder(t *testing.T) {
	assert.True(t, ContainsPlaceholder("Device.Services.FAPService.{12}."))
	assert.True(t, ContainsPlaceholder("Device.Services.FAPService.{12}.CellConfig.LTE.EPC.PLMNList.{6}."))
	assert.False(t, ContainsPlaceholder("Device.DeviceInfo.Manufacturer"))
	assert.False(t, ContainsPlaceholder("Device.Services.FAPService.1."))
}

func TestTemplateToBasePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Device.Services.FAPService.{12}.", "Device.Services.FAPService."},
		{"Device.Services.FAPService.{12}.CellConfig.LTE.EPC.PLMNList.{6}.", "Device.Services.FAPService."},
		{"Device.DeviceInfo.", "Device.DeviceInfo."},
		{"Device.", "Device."},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, TemplateToBasePath(tt.input), "input: %s", tt.input)
	}
}

func TestTemplateToRegex(t *testing.T) {
	re := TemplateToRegex("Device.Services.FAPService.{12}.CellConfig.LTE.RAN.Common.CellIdentity")
	assert.True(t, re.MatchString("Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellIdentity"))
	assert.True(t, re.MatchString("Device.Services.FAPService.99.CellConfig.LTE.RAN.Common.CellIdentity"))
	assert.False(t, re.MatchString("Device.Services.FAPService.CellConfig.LTE.RAN.Common.CellIdentity"))
	assert.False(t, re.MatchString("Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.Other"))

	// Nested placeholders
	re2 := TemplateToRegex("Device.Services.FAPService.{12}.CellConfig.LTE.EPC.PLMNList.{6}.PLMNID")
	assert.True(t, re2.MatchString("Device.Services.FAPService.1.CellConfig.LTE.EPC.PLMNList.3.PLMNID"))
	assert.True(t, re2.MatchString("Device.Services.FAPService.2.CellConfig.LTE.EPC.PLMNList.1.PLMNID"))
	assert.False(t, re2.MatchString("Device.Services.FAPService.1.CellConfig.LTE.EPC.PLMNList.PLMNID"))
}

func TestExtractInstanceNumbers(t *testing.T) {
	refs := ExtractInstanceNumbers("Device.Services.FAPService.2.CellConfig.LTE.EPC.PLMNList.3.PLMNID")
	assert.Len(t, refs, 2)
	assert.Equal(t, "FAPService", refs[0].Segment)
	assert.Equal(t, 2, refs[0].Instance)
	assert.Equal(t, "PLMNList", refs[1].Segment)
	assert.Equal(t, 3, refs[1].Instance)

	// No instances
	refs = ExtractInstanceNumbers("Device.DeviceInfo.Manufacturer")
	assert.Len(t, refs, 0)
}

func TestReplaceInstance(t *testing.T) {
	result := ReplaceInstance("Device.Services.FAPService.{12}.CellConfig.LTE.EPC.PLMNList.{6}.", 0, 1)
	assert.Equal(t, "Device.Services.FAPService.1.CellConfig.LTE.EPC.PLMNList.{6}.", result)

	result = ReplaceInstance("Device.Services.FAPService.{12}.CellConfig.LTE.EPC.PLMNList.{6}.", 1, 3)
	assert.Equal(t, "Device.Services.FAPService.{12}.CellConfig.LTE.EPC.PLMNList.3.", result)
}

func TestCountPlaceholders(t *testing.T) {
	assert.Equal(t, 0, CountPlaceholders("Device.DeviceInfo."))
	assert.Equal(t, 1, CountPlaceholders("Device.Services.FAPService.{12}."))
	assert.Equal(t, 2, CountPlaceholders("Device.Services.FAPService.{12}.CellConfig.LTE.EPC.PLMNList.{6}."))
}
