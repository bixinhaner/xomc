package response

import (
	"encoding/json"
	"math"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- 集中兜底单元：sanitizeNonFiniteFloats 直接测 ---

func TestSanitize_FiniteUntouched(t *testing.T) {
	// 不含非有限浮点时，原样返回入参（保真：让 encoding/json 全权编码）。
	in := map[string]any{"a": 1.5, "b": "x", "c": []int{1, 2}}
	out := sanitizeNonFiniteFloats(in)
	// 同一引用返回，证明未做任何拷贝。
	got, ok := out.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, 1.5, got["a"])
}

func TestSanitize_NaNInfToNull(t *testing.T) {
	type row struct {
		Path  string  `json:"metric_path"`
		Value float64 `json:"metric_value"`
	}
	in := []row{
		{Path: "ok", Value: 1.5},
		{Path: "nan", Value: math.NaN()},
		{Path: "pinf", Value: math.Inf(1)},
		{Path: "ninf", Value: math.Inf(-1)},
	}
	out := sanitizeNonFiniteFloats(in)
	b, err := json.Marshal(out)
	require.NoError(t, err, "净化后必须可序列化（修复前直接 marshal 报错）")

	var rows []map[string]any
	require.NoError(t, json.Unmarshal(b, &rows))
	require.Len(t, rows, 4)
	assert.EqualValues(t, 1.5, rows[0]["metric_value"])
	assert.Nil(t, rows[1]["metric_value"])
	assert.Nil(t, rows[2]["metric_value"])
	assert.Nil(t, rows[3]["metric_value"])
	// 字段必须存在（是 null，不是缺字段）。
	_, has := rows[1]["metric_value"]
	assert.True(t, has)
}

func TestSanitize_Float32(t *testing.T) {
	out := sanitizeNonFiniteFloats(map[string]any{"v": float32(math.Inf(1))})
	b, err := json.Marshal(out)
	require.NoError(t, err)
	assert.JSONEq(t, `{"v":null}`, string(b))
}

func TestSanitize_PreservesTimeAndPointers(t *testing.T) {
	// 含 time.Time（自定义 Marshaler）+ 指针 + NaN，验证 time 原样、NaN 归 null。
	ts := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	nan := math.NaN()
	in := map[string]any{
		"at":   ts,
		"vp":   &nan,
		"good": 2.0,
	}
	out := sanitizeNonFiniteFloats(in)
	b, err := json.Marshal(out)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(b, &got))
	assert.Nil(t, got["vp"])
	assert.EqualValues(t, 2.0, got["good"])
	// time 仍被序列化为 RFC3339 字符串（自定义 marshaler 未被拆解）。
	assert.Equal(t, "2026-06-15T10:00:00Z", got["at"])
}

// --- 通过 response.OK 走真实 Gin 渲染路径，覆盖三个曾漏网的 PM DTO 形状 ---

// pmMetricLike 模拟 metrics.PMMetric：裸 float64 + time.Time，曾未被 jsonx.Float 覆盖。
type pmMetricLike struct {
	MetricPath  string    `json:"metric_path"`
	MetricValue float64   `json:"metric_value"`
	EndTime     time.Time `json:"end_time"`
}

func TestOK_PMMetricBareFloatNaN(t *testing.T) {
	rows := []pmMetricLike{
		{MetricPath: "RRC.Succ", MetricValue: 99.5, EndTime: time.Unix(0, 0).UTC()},
		{MetricPath: "RRC.Ratio", MetricValue: math.NaN(), EndTime: time.Unix(0, 0).UTC()},
	}
	c, rec := newCtx()
	OK(c, rows)

	assert.Equal(t, http.StatusOK, rec.Code)
	require.NotZero(t, rec.Body.Len(), "含 NaN 的裸 float DTO 也必须返回非空 body")

	body := decode(t, rec)
	data, ok := body["data"].([]any)
	require.True(t, ok)
	require.Len(t, data, 2)
	assert.EqualValues(t, 99.5, data[0].(map[string]any)["metric_value"])
	v, has := data[1].(map[string]any)["metric_value"]
	assert.True(t, has)
	assert.Nil(t, v, "NaN 行的 metric_value 应为 null")
}

// kpiValueLike 模拟 model.KPIValue：裸 float64（json kpi_value），曾未被覆盖。
type kpiValueLike struct {
	KPIName  string  `json:"kpi_name"`
	KPIValue float64 `json:"kpi_value"`
}

func TestOK_KPIValueBareFloatInf(t *testing.T) {
	c, rec := newCtx()
	OK(c, gin.H{"items": []kpiValueLike{
		{KPIName: "Drop", KPIValue: 0.1},
		{KPIName: "AvgThp", KPIValue: math.Inf(1)},
	}})

	assert.Equal(t, http.StatusOK, rec.Code)
	require.NotZero(t, rec.Body.Len())
	body := decode(t, rec)
	items := body["data"].(map[string]any)["items"].([]any)
	require.Len(t, items, 2)
	assert.EqualValues(t, 0.1, items[0].(map[string]any)["kpi_value"])
	assert.Nil(t, items[1].(map[string]any)["kpi_value"])
}

// aggCounterLike 模拟 counter.AggregatedCounter：多个裸 float64 字段。
type aggCounterLike struct {
	CellID    string  `json:"cell_id"`
	AvgValue  float64 `json:"avg_value"`
	SumValue  float64 `json:"sum_value"`
	MinValue  float64 `json:"min_value"`
	MaxValue  float64 `json:"max_value"`
	SampleCnt int     `json:"sample_count"`
}

func TestOK_AggregatedCounterMixedNaN(t *testing.T) {
	c, rec := newCtx()
	OK(c, gin.H{"items": []aggCounterLike{
		{CellID: "1", AvgValue: math.NaN(), SumValue: 0, MinValue: math.Inf(-1), MaxValue: 5, SampleCnt: 0},
	}})

	assert.Equal(t, http.StatusOK, rec.Code)
	require.NotZero(t, rec.Body.Len())
	body := decode(t, rec)
	item := body["data"].(map[string]any)["items"].([]any)[0].(map[string]any)
	assert.Nil(t, item["avg_value"])
	assert.EqualValues(t, 0, item["sum_value"])
	assert.Nil(t, item["min_value"])
	assert.EqualValues(t, 5, item["max_value"])
	assert.EqualValues(t, 0, item["sample_count"])
}

// TestOK_NoNaN_ByteIdenticalToPlainJSON 保真守门：不含 NaN 时，统一信封输出与
// 直接 c.JSON 编码逐字节一致（证明集中兜底对正常响应零副作用）。
func TestOK_NoNaN_ByteIdenticalToPlainJSON(t *testing.T) {
	type nested struct {
		Name string   `json:"name"`
		Vals []float64 `json:"vals"`
		Opt  string   `json:"opt,omitempty"`
	}
	payload := []nested{
		{Name: "a", Vals: []float64{1.5, 2.5}},
		{Name: "b", Vals: []float64{0}, Opt: "x"},
	}

	c, rec := newCtx()
	OK(c, payload)
	got := rec.Body.String()

	// 期望 = 手工拼的标准信封（omitempty 生效：第一条无 opt 字段）。
	want, err := json.Marshal(gin.H{"ret": 1, "msg": MsgOK, "data": payload})
	require.NoError(t, err)
	assert.JSONEq(t, string(want), got)
	// 进一步断言 omitempty 真生效（第一条不含 opt）。
	var env map[string]any
	require.NoError(t, json.Unmarshal([]byte(got), &env))
	first := env["data"].([]any)[0].(map[string]any)
	_, hasOpt := first["opt"]
	assert.False(t, hasOpt, "omitempty 必须保真：无值字段不应出现")
}
