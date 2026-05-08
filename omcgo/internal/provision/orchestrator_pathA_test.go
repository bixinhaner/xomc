package provision

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/config/template"
	"github.com/omcgo/omcgo/internal/core/model"
)

// 构造一个 Translator 含 2 条 standard ↔ private 映射。
func newTestTranslator(t *testing.T) *parammodel.Translator {
	t.Helper()
	set := &parammodel.MappingSet{
		Source: parammodel.MappingSourceDefault,
		Mappings: []parammodel.ParamMapping{
			{StandardPath: "Device.WiFi.SSID", PrivatePath: "Dev.WiFi.SSID", EntryType: "parameter", Access: "readWrite", IsActive: true},
			{StandardPath: "Device.WiFi.Channel", PrivatePath: "Dev.WiFi.Ch", EntryType: "parameter", Access: "readWrite", IsActive: true},
		},
	}
	return parammodel.NewTranslator(set, nil, nil)
}

func TestBuildProvisioningStepsTranslated_NilTranslator_FallbackToOldPath(t *testing.T) {
	tmpl := &template.ConfigTemplate{
		ID:           uuid.New(),
		Name:         "tmpl",
		Carrier:      model.CarrierCMCC,
		TemplateType: template.TemplateProvisioning,
		Parameters:   json.RawMessage(`{"Dev.WiFi.SSID": "x"}`),
	}
	steps, err := BuildProvisioningStepsTranslated(tmpl, nil)
	require.NoError(t, err)
	require.NotEmpty(t, steps)

	// 旧路径不翻译——SPV 步骤的 Params 应原样保留
	for _, s := range steps {
		if s.Method == MethodSetParameterValues {
			assert.JSONEq(t, `{"Dev.WiFi.SSID": "x"}`, string(s.Params))
		}
	}
}

func TestBuildProvisioningStepsTranslated_WithTranslator_KeyTranslated(t *testing.T) {
	tr := newTestTranslator(t)
	tmpl := &template.ConfigTemplate{
		ID:           uuid.New(),
		Name:         "tmpl",
		Carrier:      model.CarrierCMCC,
		TemplateType: template.TemplateProvisioning,
		Parameters:   json.RawMessage(`{"Device.WiFi.SSID": "OMC-Test", "Device.WiFi.Channel": 6}`),
	}
	steps, err := BuildProvisioningStepsTranslated(tmpl, tr)
	require.NoError(t, err)

	// SPV 步骤的 Params key 应翻译为 privatePath
	var spvFound bool
	for _, s := range steps {
		if s.Method == MethodSetParameterValues {
			spvFound = true
			var got map[string]any
			require.NoError(t, json.Unmarshal(s.Params, &got))
			assert.Contains(t, got, "Dev.WiFi.SSID")
			assert.Contains(t, got, "Dev.WiFi.Ch")
			assert.NotContains(t, got, "Device.WiFi.SSID")
			assert.NotContains(t, got, "Device.WiFi.Channel")
		}
	}
	assert.True(t, spvFound, "应包含 SPV 步骤")
}

func TestBuildProvisioningStepsTranslated_GPVStep_AlsoTranslated(t *testing.T) {
	tr := newTestTranslator(t)
	tmpl := &template.ConfigTemplate{
		ID:           uuid.New(),
		Name:         "tmpl",
		TemplateType: template.TemplateProvisioning,
		Parameters:   json.RawMessage(`{"Device.WiFi.SSID": "x"}`),
	}
	steps, err := BuildProvisioningStepsTranslated(tmpl, tr)
	require.NoError(t, err)

	// GPV 步骤的 parameter_names 是从已翻译 Parameters 提取的，因此应是 privatePath
	var gpvFound bool
	for _, s := range steps {
		if s.Method == MethodGetParameterValues {
			gpvFound = true
			var got map[string]any
			require.NoError(t, json.Unmarshal(s.Params, &got))
			names, _ := got["parameter_names"].([]any)
			require.Len(t, names, 1)
			assert.Equal(t, "Dev.WiFi.SSID", names[0])
		}
	}
	assert.True(t, gpvFound, "应包含 GPV 步骤")
}

func TestBuildProvisioningStepsTranslated_UnknownKey_PreservedAsIs(t *testing.T) {
	tr := newTestTranslator(t)
	tmpl := &template.ConfigTemplate{
		ID:           uuid.New(),
		Name:         "tmpl",
		TemplateType: template.TemplateProvisioning,
		Parameters:   json.RawMessage(`{"Device.Unknown.Param": "x"}`),
	}
	steps, err := BuildProvisioningStepsTranslated(tmpl, tr)
	require.NoError(t, err)

	for _, s := range steps {
		if s.Method == MethodSetParameterValues {
			var got map[string]any
			require.NoError(t, json.Unmarshal(s.Params, &got))
			// Translator 命中失败 → 保留原 standardPath
			assert.Contains(t, got, "Device.Unknown.Param")
		}
	}
}

func TestTranslateTemplateParameters_EmptyJSON_PassThrough(t *testing.T) {
	tr := newTestTranslator(t)

	got, err := translateTemplateParameters(nil, tr)
	require.NoError(t, err)
	assert.Empty(t, got)

	got, err = translateTemplateParameters(json.RawMessage(`{}`), tr)
	require.NoError(t, err)
	assert.JSONEq(t, "{}", string(got))
}

func TestTranslateTemplateParameters_InvalidJSON_FallthroughOriginal(t *testing.T) {
	tr := newTestTranslator(t)
	bad := json.RawMessage(`[1,2,3]`) // 不是对象
	got, err := translateTemplateParameters(bad, tr)
	require.NoError(t, err, "容错：非对象 JSON 应保留原文")
	assert.JSONEq(t, "[1,2,3]", string(got))
}

func TestTranslateTemplateParameters_NilTranslator_PassThrough(t *testing.T) {
	in := json.RawMessage(`{"Device.X": "y"}`)
	got, err := translateTemplateParameters(in, nil)
	require.NoError(t, err)
	assert.Equal(t, in, got)
}

func TestTranslateTemplateParameters_PreservesValues(t *testing.T) {
	tr := newTestTranslator(t)
	in := json.RawMessage(`{"Device.WiFi.SSID": "OMC", "Device.WiFi.Channel": 6}`)
	got, err := translateTemplateParameters(in, tr)
	require.NoError(t, err)
	var out map[string]any
	require.NoError(t, json.Unmarshal(got, &out))
	assert.Equal(t, "OMC", out["Dev.WiFi.SSID"])
	assert.EqualValues(t, 6, out["Dev.WiFi.Ch"])
}
