package global

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_CarrierCode_IsValid(t *testing.T) {
	tests := []struct {
		name    string
		carrier CarrierCode
		want    bool
	}{
		{"cmcc is valid", CarrierCMCC, true},
		{"ctcc is valid", CarrierCTCC, true},
		{"cucc is valid", CarrierCUCC, true},
		{"empty is invalid", CarrierCode(""), false},
		{"unknown is invalid", CarrierCode("unknown"), false},
		{"uppercase CMCC is invalid", CarrierCode("CMCC"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.carrier.IsValid())
		})
	}
}

func Test_Technology_IsValid(t *testing.T) {
	tests := []struct {
		name string
		tech Technology
		want bool
	}{
		{"lte is valid", TechLTE, true},
		{"nr is valid", TechNR, true},
		{"empty is invalid", Technology(""), false},
		{"unknown is invalid", Technology("wifi"), false},
		{"uppercase LTE is invalid", Technology("LTE"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.tech.IsValid())
		})
	}
}

func Test_ValidCarriers(t *testing.T) {
	carriers := ValidCarriers()

	assert.Len(t, carriers, 3)
	assert.Contains(t, carriers, CarrierCMCC)
	assert.Contains(t, carriers, CarrierCTCC)
	assert.Contains(t, carriers, CarrierCUCC)
}

func Test_CarrierCode_Values(t *testing.T) {
	assert.Equal(t, CarrierCode("cmcc"), CarrierCMCC)
	assert.Equal(t, CarrierCode("ctcc"), CarrierCTCC)
	assert.Equal(t, CarrierCode("cucc"), CarrierCUCC)
}

func Test_Technology_Values(t *testing.T) {
	assert.Equal(t, Technology("lte"), TechLTE)
	assert.Equal(t, Technology("nr"), TechNR)
}

func Test_AlarmSeverity_Ordering(t *testing.T) {
	// Critical should be most severe (lowest number)
	assert.Less(t, int(AlarmCritical), int(AlarmMajor))
	assert.Less(t, int(AlarmMajor), int(AlarmMinor))
	assert.Less(t, int(AlarmMinor), int(AlarmWarning))
}

func Test_ParameterType_Values(t *testing.T) {
	tests := []struct {
		name string
		pt   ParameterType
		want string
	}{
		{"string", ParamString, "string"},
		{"int", ParamInt, "int"},
		{"uint", ParamUint, "unsignedInt"},
		{"bool", ParamBool, "boolean"},
		{"dateTime", ParamDateTime, "dateTime"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, string(tt.pt))
		})
	}
}

func Test_DataModelScope_Values(t *testing.T) {
	assert.Equal(t, DataModelScope("product"), ScopeProduct)
	assert.Equal(t, DataModelScope("oui"), ScopeOUI)
	assert.Equal(t, DataModelScope("carrier_default"), ScopeCarrierDefault)
}

func Test_DeviceStatus_Values(t *testing.T) {
	statuses := []DeviceStatus{
		DeviceDiscovered,
		DeviceRegistered,
		DeviceProvisioning,
		DeviceActive,
		DeviceMaintenance,
		DeviceOffline,
		DeviceDecommissioned,
	}

	assert.Len(t, statuses, 7)

	// Ensure all are distinct
	seen := make(map[DeviceStatus]bool)
	for _, s := range statuses {
		assert.False(t, seen[s], "duplicate status: %s", s)
		seen[s] = true
	}
}

func Test_AlarmStatus_Values(t *testing.T) {
	assert.Equal(t, AlarmStatus("active"), AlarmActive)
	assert.Equal(t, AlarmStatus("acknowledged"), AlarmAcknowledged)
	assert.Equal(t, AlarmStatus("cleared"), AlarmCleared)
}

// Test_CarrierCode_UnmarshalJSON_CaseInsensitive 锁住 case-normalize 行为：
// 任何外部 JSON 入参（北向 OSS / 手工 API / 批量导入）的 carrier 字段，
// 不管大写小写混合，都规范成内部 canonical 小写。
func Test_CarrierCode_UnmarshalJSON_CaseInsensitive(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  CarrierCode
	}{
		{"lowercase pass-through", `"cmcc"`, CarrierCMCC},
		{"uppercase normalized", `"CMCC"`, CarrierCMCC},
		{"mixed case normalized", `"Cmcc"`, CarrierCMCC},
		{"trims whitespace", `"  ctcc  "`, CarrierCTCC},
		{"uppercase ctcc", `"CTCC"`, CarrierCTCC},
		{"mixed cucc", `"cUcC"`, CarrierCUCC},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var c CarrierCode
			require.NoError(t, json.Unmarshal([]byte(tt.input), &c))
			assert.Equal(t, tt.want, c)
		})
	}
}

// Test_Technology_UnmarshalJSON_CaseInsensitive 锁住 case-normalize：
// 3GPP/TR-181 标准约定 LTE/NR 大写，我们内部 canonical 小写。任何外部
// 大小写都规范掉。
func Test_Technology_UnmarshalJSON_CaseInsensitive(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Technology
	}{
		{"lowercase pass-through", `"lte"`, TechLTE},
		{"uppercase per TR-181", `"LTE"`, TechLTE},
		{"mixed case", `"Lte"`, TechLTE},
		{"trims whitespace", `"  nr  "`, TechNR},
		{"uppercase NR", `"NR"`, TechNR},
		{"mixed NR", `"Nr"`, TechNR},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tech Technology
			require.NoError(t, json.Unmarshal([]byte(tt.input), &tech))
			assert.Equal(t, tt.want, tech)
		})
	}
}

// Test_Technology_UnmarshalJSON_InStruct 验证嵌套字段（最接近 BatchImport /
// CreateDeviceRequest 真实使用场景）正确触发 UnmarshalJSON。
func Test_Technology_UnmarshalJSON_InStruct(t *testing.T) {
	type req struct {
		Carrier CarrierCode `json:"carrier"`
		Tech    Technology  `json:"technology"`
	}
	var r req
	require.NoError(t, json.Unmarshal([]byte(`{"carrier":"CMCC","technology":"LTE"}`), &r))
	assert.Equal(t, CarrierCMCC, r.Carrier)
	assert.Equal(t, TechLTE, r.Tech)
	// 规范化后 IsValid 也能通过（之前大写会被 oneof binding 拒掉）
	assert.True(t, r.Carrier.IsValid())
	assert.True(t, r.Tech.IsValid())
}
