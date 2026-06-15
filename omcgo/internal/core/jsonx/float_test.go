package jsonx

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFloat_MarshalJSON_Finite 有限值按普通 JSON number 输出，且与 encoding/json
// 对原生 float64 的口径一致。
func TestFloat_MarshalJSON_Finite(t *testing.T) {
	cases := []struct {
		name string
		in   float64
	}{
		{"零", 0},
		{"正整数", 42},
		{"负数", -3.5},
		{"小数", 12.625},
		{"大数", 1234567.89},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := json.Marshal(Float(tc.in))
			require.NoError(t, err)
			want, err := json.Marshal(tc.in)
			require.NoError(t, err)
			assert.Equal(t, string(want), string(got), "应与原生 float64 输出一致")
		})
	}
}

// TestFloat_MarshalJSON_NonFinite 非有限值（NaN/+Inf/-Inf）序列化为 JSON null，
// 不报错（这是 issue #387 的核心：避免单个 NaN 致整批编码失败、返回空 body）。
func TestFloat_MarshalJSON_NonFinite(t *testing.T) {
	cases := []struct {
		name string
		in   float64
	}{
		{"NaN", math.NaN()},
		{"正无穷", math.Inf(1)},
		{"负无穷", math.Inf(-1)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := json.Marshal(Float(tc.in))
			require.NoError(t, err, "非有限值不得报错")
			assert.Equal(t, "null", string(got))
		})
	}
}

// TestFloat_InStruct_MixedRows 模拟 PM 结果集：含 NaN 的整批结构体能被完整序列化，
// NaN 字段呈现为 null，其余有限值正常（修复前 encoding/json 会整体报错、Gin 流式编码
// 已写 200 头后中断 → 空 body）。
func TestFloat_InStruct_MixedRows(t *testing.T) {
	type row struct {
		Path  string `json:"metric_path"`
		Value Float  `json:"metric_value"`
	}
	rows := []row{
		{Path: "C1", Value: Float(100)},
		{Path: "C2", Value: Float(math.NaN())}, // 平均型分母为 0 → NaN
		{Path: "C3", Value: Float(math.Inf(1))},
		{Path: "C4", Value: Float(3.5)},
	}

	b, err := json.Marshal(rows)
	require.NoError(t, err, "含 NaN/Inf 行不得致整批编码失败")
	require.NotEmpty(t, b, "body 不得为空")

	var out []map[string]any
	require.NoError(t, json.Unmarshal(b, &out))
	require.Len(t, out, 4)

	assert.EqualValues(t, 100, out[0]["metric_value"])
	assert.Nil(t, out[1]["metric_value"], "NaN 行 metric_value 应为 null")
	_, hasKey := out[1]["metric_value"]
	assert.True(t, hasKey, "字段应存在且为 null（不是缺字段）")
	assert.Nil(t, out[2]["metric_value"], "Inf 行 metric_value 应为 null")
	assert.EqualValues(t, 3.5, out[3]["metric_value"])
}
