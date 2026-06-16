package adhoc

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// ctxWithQuery 构造一个携带原始 query string 的 gin.Context，便于离线测两个 query 解析器。
func ctxWithQuery(rawQuery string) *gin.Context {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/r?"+rawQuery, nil)
	return c
}

// issue #401 核心：object_ldns 的设备组值合法含一个逗号
// （'DeviceGroup=<uuid>,Tech=<tech>'），parseRepeatedQuery 必须整值保留不拆，
// 否则被切成两段、object_ldn = ANY(...) 都匹配不上 → 0 行。
func Test_parseRepeatedQuery_DeviceGroupValueWithCommaKeptIntact(t *testing.T) {
	full := "DeviceGroup=00000000-0000-0000-0000-000000000001,Tech=lte"
	c := ctxWithQuery("object_ldns=" + url.QueryEscape(full))

	got := parseRepeatedQuery(c, "object_ldns")

	// 单值整体保留，绝不被逗号切成两段
	assert.Equal(t, []string{full}, got)
	assert.Len(t, got, 1, "含逗号的设备组值不应被拆分")
}

// 多个 object_ldns 重复参数取并集：每个整值各自保留。
func Test_parseRepeatedQuery_MultipleRepeatedParamsUnion(t *testing.T) {
	v1 := "DeviceGroup=00000000-0000-0000-0000-000000000001,Tech=lte"
	v2 := "DeviceGroup=00000000-0000-0000-0000-000000000002,Tech=lte"
	c := ctxWithQuery("object_ldns=" + url.QueryEscape(v1) + "&object_ldns=" + url.QueryEscape(v2))

	got := parseRepeatedQuery(c, "object_ldns")

	assert.Equal(t, []string{v1, v2}, got)
}

// 无值返回 nil（= ANY 子句不进 SQL = 不过滤），向后兼容不传过滤的旧行为。
func Test_parseRepeatedQuery_EmptyReturnsNil(t *testing.T) {
	c := ctxWithQuery("granularity=hourly")
	assert.Nil(t, parseRepeatedQuery(c, "object_ldns"))
}

// 空白值被去除，全空白返回 nil。
func Test_parseRepeatedQuery_BlankTrimmed(t *testing.T) {
	c := ctxWithQuery("object_ldns=%20%20&object_ldns=Band%3D42")
	got := parseRepeatedQuery(c, "object_ldns")
	assert.Equal(t, []string{"Band=42"}, got)
}

// 对照：parseCSVQuery 仍按逗号拆（product_ids 纯 UUID 不含逗号，行为不变）。
// 若误把含逗号值喂给它会被拆——正说明 object_ldns 必须改走 parseRepeatedQuery。
func Test_parseCSVQuery_StillSplitsOnComma(t *testing.T) {
	c := ctxWithQuery("product_ids=a,b&product_ids=c")
	got := parseCSVQuery(c, "product_ids")
	assert.Equal(t, []string{"a", "b", "c"}, got)
}

// 端到端落到 SQL：含逗号的设备组值经 parseRepeatedQuery → SubsetLDNs，
// buildResultsQuery 把完整整值原样放进 object_ldn = ANY 的占位参数（不被切碎）。
func Test_buildResultsQuery_SubsetLDNWithCommaIntact(t *testing.T) {
	full := "DeviceGroup=00000000-0000-0000-0000-000000000001,Tech=lte"
	c := ctxWithQuery("object_ldns=" + url.QueryEscape(full))
	ldns := parseRepeatedQuery(c, "object_ldns")

	q, args := buildResultsQuery(uuid.New(), resultsFilter{SubsetLDNs: ldns}, 100, 0)

	assert.Contains(t, q, "AND r.object_ldn = ANY($2)")
	// 占位参数里是完整含逗号整值（[]string 单元素），不是被拆开的两段
	gotArr, ok := args[1].([]string)
	assert.True(t, ok, "SubsetLDNs 应作为 []string 占位参数入 args")
	assert.Equal(t, []string{full}, gotArr)
}
