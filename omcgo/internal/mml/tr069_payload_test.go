package mml

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 命令模拟：DEVICE_INFO（LST/MOD），param_refs 来自 mml_params 真实形态
func deviceInfoRefs() []MMLParamRef {
	mk := func(code, path, vt string, writable bool) MMLParamRef {
		return MMLParamRef{
			ID:         uuid.New(),
			ParamCode:  code,
			Tr069Path:  path,
			ValueType:  vt,
			IsWritable: writable,
		}
	}
	return []MMLParamRef{
		mk("LTE_BTSNUM", "Device.X_BAICELLS_LTE.BtsNum", "unsignedInt", false),
		mk("LTE_GSM_IP", "Device.X_BAICELLS_LTE.GsmIP", "string", false),
		mk("DEVICEGSM_MCC", "Device.X_BAICELLS_LTE.GsmMcc", "string", true),
		mk("DEVICEGSM_NRIBITLEN", "Device.X_BAICELLS_LTE.NriBitLen", "int", true),
		mk("LTE_DEVICE_HARDWARE_VERSION", "Device.DeviceInfo.HardwareVersion", "string", false),
	}
}

func TestBuildTR069Params_GetParameterValues(t *testing.T) {
	refs := deviceInfoRefs()

	// LST 操作：用户没填表单，formValues 为前端按 param_code 列空值
	formValues := map[string]interface{}{
		"LTE_BTSNUM":                  "",
		"LTE_GSM_IP":                  "",
		"DEVICEGSM_MCC":               "",
		"DEVICEGSM_NRIBITLEN":         "",
		"LTE_DEVICE_HARDWARE_VERSION": "",
	}

	payload, err := BuildTR069Params("GetParameterValues", refs, formValues, "LST")
	require.NoError(t, err)

	var got struct {
		Names []string `json:"names"`
	}
	require.NoError(t, json.Unmarshal(payload, &got))
	assert.Len(t, got.Names, len(refs), "names count must equal param_refs count")
	assert.Contains(t, got.Names, "Device.X_BAICELLS_LTE.BtsNum")
	assert.Contains(t, got.Names, "Device.DeviceInfo.HardwareVersion")
}

func TestBuildTR069Params_GetParameterValues_DedupesPaths(t *testing.T) {
	dup := MMLParamRef{ParamCode: "DUP", Tr069Path: "Device.X.Y", ValueType: "string"}
	refs := []MMLParamRef{dup, dup, {ParamCode: "OK", Tr069Path: "Device.X.Z", ValueType: "string"}}

	payload, err := BuildTR069Params("GetParameterValues", refs, nil, "LST")
	require.NoError(t, err)
	var got struct {
		Names []string `json:"names"`
	}
	require.NoError(t, json.Unmarshal(payload, &got))
	assert.Equal(t, []string{"Device.X.Y", "Device.X.Z"}, got.Names)
}

func TestBuildTR069Params_GetParameterValues_NoRefs(t *testing.T) {
	_, err := BuildTR069Params("GetParameterValues", nil, nil, "LST")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNoUsableParams))
}

func TestBuildTR069Params_SetParameterValues(t *testing.T) {
	refs := deviceInfoRefs()

	// MOD 操作：仅填了两个字段，其余空字符串应被过滤
	formValues := map[string]interface{}{
		"DEVICEGSM_MCC":       "460",
		"DEVICEGSM_NRIBITLEN": float64(8), // JSON 数字反序列化默认 float64
		"LTE_BTSNUM":          "",         // 空 → 跳过
		"LTE_GSM_IP":          "  ",       // 空白 → 跳过
	}

	payload, err := BuildTR069Params("SetParameterValues", refs, formValues, "MOD")
	require.NoError(t, err)

	var got struct {
		Values []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
			Type  string `json:"type"`
		} `json:"values"`
	}
	require.NoError(t, json.Unmarshal(payload, &got))
	require.Len(t, got.Values, 2)

	byName := make(map[string]struct{ Value, Type string })
	for _, v := range got.Values {
		byName[v.Name] = struct{ Value, Type string }{v.Value, v.Type}
	}
	assert.Equal(t, "460", byName["Device.X_BAICELLS_LTE.GsmMcc"].Value)
	assert.Equal(t, "xsd:string", byName["Device.X_BAICELLS_LTE.GsmMcc"].Type)
	assert.Equal(t, "8", byName["Device.X_BAICELLS_LTE.NriBitLen"].Value)
	assert.Equal(t, "xsd:int", byName["Device.X_BAICELLS_LTE.NriBitLen"].Type)
}

func TestBuildTR069Params_SetParameterValues_AllEmpty(t *testing.T) {
	refs := deviceInfoRefs()
	formValues := map[string]interface{}{
		"DEVICEGSM_MCC": "",
		"LTE_BTSNUM":    "",
	}
	_, err := BuildTR069Params("SetParameterValues", refs, formValues, "MOD")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNoUsableParams))
}

func TestBuildTR069Params_SetParameterValues_UnknownCodes(t *testing.T) {
	refs := deviceInfoRefs()
	formValues := map[string]interface{}{
		"NOT_A_REAL_CODE": "xxx",          // 没绑定，跳过
		"DEVICEGSM_MCC":   "460",          // OK
	}
	payload, err := BuildTR069Params("SetParameterValues", refs, formValues, "MOD")
	require.NoError(t, err)
	assert.Contains(t, string(payload), "Device.X_BAICELLS_LTE.GsmMcc")
	assert.NotContains(t, string(payload), "NOT_A_REAL_CODE")
}

func TestBuildTR069Params_AddObject(t *testing.T) {
	refs := []MMLParamRef{{Tr069Path: "Device.IP.Interface"}}
	payload, err := BuildTR069Params("AddObject", refs, nil, "ADD")
	require.NoError(t, err)
	assert.Equal(t, `{"object_name":"Device.IP.Interface."}`, string(payload))
}

func TestBuildTR069Params_DeleteObject_FormOverride(t *testing.T) {
	refs := []MMLParamRef{{Tr069Path: "Device.IP.Interface"}}
	formValues := map[string]interface{}{"object_name": "Device.IP.Interface.3"}
	payload, err := BuildTR069Params("DeleteObject", refs, formValues, "RMV")
	require.NoError(t, err)
	assert.Equal(t, `{"object_name":"Device.IP.Interface.3."}`, string(payload))
}

func TestBuildTR069Params_RebootFactoryReset(t *testing.T) {
	for _, m := range []string{"Reboot", "FactoryReset"} {
		payload, err := BuildTR069Params(m, nil, nil, "RST")
		require.NoError(t, err, m)
		assert.JSONEq(t, `{}`, string(payload), m)
	}
}

func TestBuildTR069Params_GetParameterNames_Default(t *testing.T) {
	refs := []MMLParamRef{{Tr069Path: "Device.IP."}}
	payload, err := BuildTR069Params("GetParameterNames", refs, nil, "LST")
	require.NoError(t, err)
	assert.JSONEq(t, `{"path":"Device.IP.","next_level":false}`, string(payload))
}

func TestBuildTR069Params_GetParameterNames_FormOverride(t *testing.T) {
	formValues := map[string]interface{}{"path": "Device.WiFi.", "next_level": true}
	payload, err := BuildTR069Params("GetParameterNames", nil, formValues, "LST")
	require.NoError(t, err)
	assert.JSONEq(t, `{"path":"Device.WiFi.","next_level":true}`, string(payload))
}

func TestBuildTR069Params_UnknownMethodPassthrough(t *testing.T) {
	formValues := map[string]interface{}{"foo": "bar"}
	payload, err := BuildTR069Params("CustomMethod", nil, formValues, "")
	require.NoError(t, err)
	assert.JSONEq(t, `{"foo":"bar"}`, string(payload))
}

func TestXSDType(t *testing.T) {
	cases := map[string]string{
		"boolean":         "xsd:boolean",
		"unsignedInt":     "xsd:unsignedInt",
		"int":             "xsd:int",
		"uniqueInt":       "xsd:int",
		"string":          "xsd:string",
		"enum":            "xsd:string",
		"stringList":      "xsd:string",
		"unsignedIntList": "xsd:string",
		"":                "xsd:string", // 兜底
	}
	for in, want := range cases {
		assert.Equal(t, want, xsdType(in), in)
	}
}

func TestSummarizeSchema(t *testing.T) {
	cases := []struct {
		name    string
		payload string
		want    SchemaSummary // PayloadSize 不在表里，比较时由 len(payload) 注入
	}{
		{
			name:    "names",
			payload: `{"names":["a","b","c"]}`,
			want:    SchemaSummary{HasNames: true, NamesCount: 3},
		},
		{
			name:    "values",
			payload: `{"values":[{"name":"x","value":"1","type":"xsd:int"}]}`,
			want:    SchemaSummary{HasValues: true, ValuesCount: 1},
		},
		{
			name:    "object_name",
			payload: `{"object_name":"Device.X."}`,
			want:    SchemaSummary{HasObjectName: true},
		},
		{
			name:    "names empty (regression: schema-shape 对了但 ParameterNames 列表为空)",
			payload: `{"names":[]}`,
			want:    SchemaSummary{HasNames: true, NamesCount: 0},
		},
		{
			name:    "non-tr069 form (Q4 根因实际写入的形态)",
			payload: `{"LTE_BTSNUM":"","LTE_GSM_IP":""}`,
			want:    SchemaSummary{},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			c.want.PayloadSize = len(c.payload)
			got := SummarizeSchema(json.RawMessage(c.payload))
			assert.Equal(t, c.want, got)
		})
	}
}
