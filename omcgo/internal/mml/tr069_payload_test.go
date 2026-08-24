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

func TestBuildTR069Params_GetParameterValues_KeepsStandardPathForACSTranslation(t *testing.T) {
	refs := []MMLParamRef{
		{
			ParamCode:         "LICENSE_AUTHOR",
			Tr069Path:         "Device.Services.FAPService.1.FAPControl.LTE.LICENSE.Author",
			PrivatePath:       "Device.FAP.License.Author",
			TranslationSource: "translated",
			ValueType:         "string",
		},
		{
			ParamCode:         "LICENSE_CAPACITY_VALUE",
			Tr069Path:         "Device.Services.FAPService.1.FAPControl.LTE.LICENSE.Capacity.1.Value",
			PrivatePath:       "Device.FAP.License.LicenseItem.1.Value",
			TranslationSource: "translated",
			ValueType:         "string",
		},
	}

	payload, err := BuildTR069Params("GetParameterValues", refs, nil, "LST")

	require.NoError(t, err)
	assert.JSONEq(t, `{"names":["Device.Services.FAPService.1.FAPControl.LTE.LICENSE.Author","Device.Services.FAPService.1.FAPControl.LTE.LICENSE.Capacity.1.Value"]}`, string(payload))
	assert.NotContains(t, string(payload), "path_mode")
}

func TestBuildTR069Params_GetParameterValues_TreatsStandardInstanceRefAsCollectionPartialPath(t *testing.T) {
	refs := []MMLParamRef{{
		ParamCode:         "CAPACITY_INSTANCE",
		Tr069Path:         "Device.Services.FAPService.1.FAPControl.LTE.LICENSE.Capacity.1",
		PrivatePath:       "Device.FAP.License.LicenseItem.1",
		TranslationSource: "translated",
		ValueType:         "string",
	}}

	payload, err := BuildTR069Params("GetParameterValues", refs, nil, "LST")

	require.NoError(t, err)
	assert.JSONEq(t, `{"names":["Device.Services.FAPService.1.FAPControl.LTE.LICENSE.Capacity."]}`, string(payload))
}

func TestBuildTR069Params_PrivatePathModeBypassesKnownRootFilter(t *testing.T) {
	refs := []MMLParamRef{{
		ParamCode: "P",
		Tr069Path: "VendorRoot.DeviceInfo.X_PRIVATE_NotRegistered",
		ValueType: "string",
		PathMode:  rawPathModePrivate,
	}}

	payload, err := BuildTR069Params("GetParameterValues", refs, nil, "LST")

	require.NoError(t, err)
	assert.JSONEq(t, `{"path_mode":"private","names":["VendorRoot.DeviceInfo.X_PRIVATE_NotRegistered"]}`, string(payload))
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

func TestBuildTR069Params_RFTxStatusUsesU32WireType(t *testing.T) {
	const path = "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus"
	refs := []MMLParamRef{{ParamCode: "RFTX_STATUS", Tr069Path: path, ValueType: "U_INT"}}

	payload, err := BuildTR069Params("SetParameterValues", refs,
		map[string]interface{}{"RFTX_STATUS": "1"}, "MOD")
	require.NoError(t, err)

	var got struct {
		Values []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
			Type  string `json:"type"`
		} `json:"values"`
	}
	require.NoError(t, json.Unmarshal(payload, &got))
	require.Len(t, got.Values, 1)
	assert.Equal(t, path, got.Values[0].Name)
	assert.Equal(t, "1", got.Values[0].Value)
	assert.Equal(t, "xsd:unsignedInt", got.Values[0].Type)
}

func TestBuildTR069Params_SetParameterValues_NormalizesBooleanToNumericWireValue(t *testing.T) {
	refs := []MMLParamRef{
		{ParamCode: "ENABLE", Tr069Path: "Device.DeviceInfo.SignallingTrace.Enable", ValueType: "boolean"},
	}

	for _, tc := range []struct {
		name string
		form interface{}
		want string
	}{
		{name: "string true", form: "true", want: "1"},
		{name: "string false", form: "false", want: "0"},
		{name: "json true", form: true, want: "1"},
		{name: "json false", form: false, want: "0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			payload, err := BuildTR069Params("SetParameterValues", refs,
				map[string]interface{}{"ENABLE": tc.form}, "MOD")
			require.NoError(t, err)
			var got struct {
				Values []struct {
					Name  string `json:"name"`
					Value string `json:"value"`
					Type  string `json:"type"`
				} `json:"values"`
			}
			require.NoError(t, json.Unmarshal(payload, &got))
			require.Len(t, got.Values, 1)
			assert.Equal(t, "Device.DeviceInfo.SignallingTrace.Enable", got.Values[0].Name)
			assert.Equal(t, tc.want, got.Values[0].Value)
			assert.Equal(t, "xsd:string", got.Values[0].Type)
		})
	}
}

func TestBuildTR069Params_SetParameterValues_KeepsBooleanWireTypeForOtherPaths(t *testing.T) {
	refs := []MMLParamRef{{ParamCode: "ENABLE", Tr069Path: "Device.X.Enable", ValueType: "boolean"}}

	payload, err := BuildTR069Params("SetParameterValues", refs,
		map[string]interface{}{"ENABLE": true}, "MOD")
	require.NoError(t, err)

	var got struct {
		Values []struct {
			Type string `json:"type"`
		} `json:"values"`
	}
	require.NoError(t, json.Unmarshal(payload, &got))
	require.Len(t, got.Values, 1)
	assert.Equal(t, "xsd:boolean", got.Values[0].Type)
}

func TestBuildTR069Params_SetParameterValues_NormalizesRawSignallingTraceEnable(t *testing.T) {
	refs := []MMLParamRef{{
		ParamCode: "Device.DeviceInfo.SignallingTrace.Enable",
		Tr069Path: "Device.DeviceInfo.SignallingTrace.Enable",
		ValueType: "string",
	}}

	payload, err := BuildTR069Params("SetParameterValues", refs,
		map[string]interface{}{"Device.DeviceInfo.SignallingTrace.Enable": "true"}, "MOD")
	require.NoError(t, err)

	var got struct {
		Values []struct {
			Value string `json:"value"`
			Type  string `json:"type"`
		} `json:"values"`
	}
	require.NoError(t, json.Unmarshal(payload, &got))
	require.Len(t, got.Values, 1)
	assert.Equal(t, "1", got.Values[0].Value)
	assert.Equal(t, "xsd:string", got.Values[0].Type)
}

func TestBuildTR069Params_SetParameterValues_KeepsStandardPathForACSTranslation(t *testing.T) {
	refs := []MMLParamRef{{
		ParamCode:         "LICENSE_CODE",
		Tr069Path:         "Device.Services.FAPService.1.FAPControl.LTE.LICENSE.Code",
		PrivatePath:       "Device.FAP.License.Code",
		TranslationSource: "translated",
		ValueType:         "string",
	}}

	payload, err := BuildTR069Params("SetParameterValues", refs, map[string]interface{}{"LICENSE_CODE": "abc"}, "MOD")

	require.NoError(t, err)
	assert.JSONEq(t, `{"values":[{"name":"Device.Services.FAPService.1.FAPControl.LTE.LICENSE.Code","value":"abc","type":"xsd:string"}]}`, string(payload))
	assert.NotContains(t, string(payload), "path_mode")
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
		"NOT_A_REAL_CODE": "xxx", // 没绑定，跳过
		"DEVICEGSM_MCC":   "460", // OK
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

// 回归 baicell 实测报文："DeviceGSM.*"（前缀非法）+ "{i}"（占位符未替换）+
// "X_COM_*"（合规字符集）混在一起。校验后只保留合法路径，全非法时报错。
func TestValidatePath_AllReasons(t *testing.T) {
	cases := []struct {
		name string
		path string
		want PathSkipReason
	}{
		{"empty", "", PathSkipBadPrefix},
		{"whitespace", "   ", PathSkipBadPrefix},
		{"missing_root", "BogusRoot.Mcc", PathSkipBadPrefix},
		{"internal_namespace", "Internal.X.Y", PathSkipBadPrefix},
		{"legal_devicegsm", "DeviceGSM.Bts.1.X", ""}, // 百怡 GSM 私有根已加入白名单
		{"legal_boardconf", "boardconf.HALOD.HALOD_PORT", ""},
		{"placeholder_i", "Device.IP.Interface.{i}.IPv4Address.{i}.IPAddress", PathSkipPlaceholder},
		{"placeholder_n", "Device.WiFi.SSID.{n}.SSID", PathSkipPlaceholder},
		{"placeholder_idx", "Device.X.Y.{idx}.Z", PathSkipPlaceholder},
		{"chinese", "Device.设备信息", PathSkipBadChars},
		{"space", "Device.Info Foo", PathSkipBadChars},
		{"legal_tr181", "Device.DeviceInfo.HardwareVersion", ""},
		{"legal_igd", "InternetGatewayDevice.DeviceInfo.HardwareVersion", ""},
		{"legal_xvendor", "Device.DeviceInfo.X_COM_MACAddress", ""},
		{"legal_indexed", "Device.IP.Interface.1.IPv4Address.1.IPAddress", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, validatePath(c.path), c.path)
		})
	}
}

func TestBuildTR069Params_GetParameterValues_FiltersIllegalPaths(t *testing.T) {
	// 复刻 baicell 现场报文形态：7 条完全未知前缀（非法）、1 条带 {i} 占位符（未绑定）、
	// 其余合法。校验后应只剩 6 条合法路径 + 1 条 {i} 展开 = 7。
	//
	// 注：DeviceGSM 已被加入合法根白名单（百怡 GSM 设备子树），
	// 故本测试用 BogusRoot.* 表示真实非法前缀场景。
	refs := []MMLParamRef{
		{ParamCode: "BAD1", Tr069Path: "BogusRoot.Encryption", ValueType: "int"},
		{ParamCode: "BAD2", Tr069Path: "BogusRoot.Mcc", ValueType: "string"},
		{ParamCode: "BAD3", Tr069Path: "BogusRoot.Mnc", ValueType: "string"},
		{ParamCode: "BAD4", Tr069Path: "BogusRoot.NriBitLen", ValueType: "int"},
		{ParamCode: "BAD5", Tr069Path: "BogusRoot.NriNullAdd", ValueType: "string"},
		{ParamCode: "BAD6", Tr069Path: "BogusRoot.TimerNetT3212", ValueType: "int"},
		{ParamCode: "BAD7", Tr069Path: "BogusRoot.BtsNum", ValueType: "int"},
		{ParamCode: "DEV_HW", Tr069Path: "Device.DeviceInfo.HardwareVersion", ValueType: "string"},
		{ParamCode: "IP_ADDR", Tr069Path: "Device.IP.Interface.{i}.IPv4Address.{i}.IPAddress", ValueType: "string"},
		{ParamCode: "DEV_MAC", Tr069Path: "Device.DeviceInfo.X_COM_MACAddress", ValueType: "string"},
		{ParamCode: "DEV_MME", Tr069Path: "Device.DeviceInfo.X_COM_MME_Status", ValueType: "string"},
		{ParamCode: "DEV_MOD", Tr069Path: "Device.DeviceInfo.X_COM_MODULE_TYPE", ValueType: "string"},
		{ParamCode: "DEV_SW", Tr069Path: "Device.DeviceInfo.SoftwareVersion", ValueType: "string"},
		{ParamCode: "DEV_RUN", Tr069Path: "Device.DeviceInfo.X_COM_STATION_RUN_Time", ValueType: "string"},
	}
	payload, err := BuildTR069Params("GetParameterValues", refs, nil, "LST")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "has_placeholder")
}

func TestBuildTR069Params_GetParameterValues_AllIllegalReturnsError(t *testing.T) {
	// 未绑定的动态路径和非法前缀路径都应验证 ErrNoUsableParams。
	refs := []MMLParamRef{
		{ParamCode: "A", Tr069Path: "Internal.Mcc"},   // 非法前缀
		{ParamCode: "B", Tr069Path: "BogusRoot.X"},    // 非法前缀
		{ParamCode: "C", Tr069Path: "Bad@chars#here"}, // 非法字符
	}
	_, err := BuildTR069Params("GetParameterValues", refs, nil, "LST")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNoUsableParams))
	assert.Contains(t, err.Error(), "failed validation")
}

func TestBuildTR069Params_SetParameterValues_FiltersIllegalPaths(t *testing.T) {
	refs := []MMLParamRef{
		{ParamCode: "BAD", Tr069Path: "Internal.Mcc", ValueType: "string"},
		{ParamCode: "PLACEHOLDER", Tr069Path: "Device.IP.Interface.{i}.X", ValueType: "string"},
		{ParamCode: "OK", Tr069Path: "Device.DeviceInfo.X_BAICELLS_GsmMcc", ValueType: "string"},
	}
	formValues := map[string]interface{}{
		"BAD":         "460",
		"PLACEHOLDER": "1.2.3.4",
		"OK":          "460",
	}
	payload, err := BuildTR069Params("SetParameterValues", refs, formValues, "MOD")
	require.NoError(t, err)
	var got struct {
		Values []struct{ Name string } `json:"values"`
	}
	require.NoError(t, json.Unmarshal(payload, &got))
	require.Len(t, got.Values, 1, "只剩 OK 一条")
	assert.Equal(t, "Device.DeviceInfo.X_BAICELLS_GsmMcc", got.Values[0].Name)
}

func TestBuildTR069Params_GetParameterNames_RejectsIllegalPath(t *testing.T) {
	cases := []struct {
		name string
		path string
	}{
		{"bad_prefix", "Internal."},
		{"placeholder", "Device.WiFi.{i}."},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := BuildTR069Params("GetParameterNames", nil,
				map[string]interface{}{"path": c.path}, "LST")
			require.Error(t, err)
			assert.True(t, errors.Is(err, ErrNoUsableParams))
		})
	}
}

func TestBuildTR069Params_AddObject_RejectsIllegalPath(t *testing.T) {
	cases := []string{"Internal.Foo", "Device.IP.{i}"}
	for _, p := range cases {
		t.Run(p, func(t *testing.T) {
			_, err := BuildTR069Params("AddObject", nil,
				map[string]interface{}{"object_name": p}, "ADD")
			require.Error(t, err)
			assert.True(t, errors.Is(err, ErrNoUsableParams))
		})
	}
}

func TestXSDType(t *testing.T) {
	cases := map[string]string{
		"boolean":         "xsd:boolean",
		"unsignedInt":     "xsd:unsignedInt",
		"U_INT":           "xsd:unsignedInt",
		"u_int":           "xsd:unsignedInt",
		"int":             "xsd:int",
		"uniqueInt":       "xsd:int",
		"DATE_TIME":       "xsd:dateTime",
		"U_LONG":          "xsd:unsignedLong",
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
