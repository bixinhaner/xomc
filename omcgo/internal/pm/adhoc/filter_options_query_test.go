package adhoc

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// ── buildFilterOptionsQuery：按维度分支 + DISTINCT/JOIN + 不含 LIMIT ──────────────

// product 维度：DISTINCT product_id + LEFT JOIN products，按 product_id 收口，无 LIMIT。
func Test_buildFilterOptionsQuery_Product(t *testing.T) {
	id := uuid.New()
	q, args, supported := buildFilterOptionsQuery(DimensionProduct, id, nil)

	assert.True(t, supported)
	assert.Contains(t, q, "SELECT DISTINCT r.dimension_key::uuid AS product_id, p.product_name")
	assert.Contains(t, q, "FROM pm_aggregation_results r")
	assert.NotContains(t, q, "pm_adhoc_aggregation_results")
	assert.Contains(t, q, "JOIN current_versions cv")
	assert.Contains(t, q, "JOIN pm_aggregation_publications published_window")
	assert.Contains(t, q, "published_window.status = 'published'")
	assert.Contains(t, q, "LEFT JOIN product_dim p ON p.id::text = r.dimension_key")
	assert.Contains(t, q, "WHERE r.task_id = $1")
	assert.Contains(t, q, "r.dimension = 'product'")
	assert.Contains(t, q, "r.dimension_key <> ''")
	// 验收 4：选项不被结果上限截断 —— SQL 不含 LIMIT/OFFSET
	assert.NotContains(t, q, "LIMIT")
	assert.NotContains(t, q, "OFFSET")
	assert.Equal(t, []any{id}, args)
}

// device_group 维度：DISTINCT object_ldn + LEFT JOIN device_groups（'DeviceGroup='||id 比对），无 LIMIT。
func Test_buildFilterOptionsQuery_DeviceGroup(t *testing.T) {
	id := uuid.New()
	q, args, supported := buildFilterOptionsQuery(DimensionDeviceGroup, id, nil)

	assert.True(t, supported)
	assert.Contains(t, q, "SELECT DISTINCT r.dimension_key AS object_ldn, g.name")
	// 设备组制式治本：object_ldn 带 ',Tech=<制式>' 后缀，取组名 JOIN 需 split_part 剥逗号前段。
	assert.Contains(t, q, "LEFT JOIN device_group_dim g ON ('DeviceGroup=' || g.id::text) = split_part(r.dimension_key, ',', 1)")
	assert.Contains(t, q, "r.dimension = 'device_group'")
	assert.Contains(t, q, "r.dimension_key LIKE 'DeviceGroup=%'")
	assert.Contains(t, q, "published_window.status = 'published'")
	assert.NotContains(t, q, "pm_adhoc_aggregation_results")
	assert.NotContains(t, q, "LIMIT")
	assert.NotContains(t, q, "OFFSET")
	assert.Equal(t, []any{id}, args)
}

// band 维度：DISTINCT object_ldn，按 Band= 前缀收口，无 JOIN、无 LIMIT。
func Test_buildFilterOptionsQuery_Band(t *testing.T) {
	id := uuid.New()
	q, args, supported := buildFilterOptionsQuery(DimensionBand, id, nil)

	assert.True(t, supported)
	assert.Contains(t, q, "SELECT DISTINCT r.dimension_key AS object_ldn")
	assert.Contains(t, q, "r.dimension = 'band'")
	assert.Contains(t, q, "r.dimension_key LIKE 'Band=%'")
	assert.Contains(t, q, "published_window.status = 'published'")
	assert.NotContains(t, q, "LEFT JOIN")
	assert.NotContains(t, q, "pm_adhoc_aggregation_results")
	assert.NotContains(t, q, "LIMIT")
	assert.NotContains(t, q, "OFFSET")
	assert.Equal(t, []any{id}, args)
}

// device/aggregate_group/network 维度：不支持筛选 → supported=false、无 SQL（前端不渲染筛选框）。
func Test_buildFilterOptionsQuery_UnsupportedDimensions(t *testing.T) {
	id := uuid.New()
	for _, dim := range []Dimension{DimensionDevice, DimensionAggregateGroup, DimensionNetwork} {
		q, args, supported := buildFilterOptionsQuery(dim, id, nil)
		assert.False(t, supported, "dimension %s should not be filterable", dim)
		assert.Empty(t, q, "dimension %s should produce no SQL", dim)
		assert.Nil(t, args, "dimension %s should produce no args", dim)
	}
}

func Test_buildFilterOptionsQuery_FilterByTaskMetricPaths(t *testing.T) {
	id := uuid.New()
	metrics := []string{"K1", "K2"}

	for _, dim := range []Dimension{DimensionProduct, DimensionDeviceGroup, DimensionBand} {
		q, args, supported := buildFilterOptionsQuery(dim, id, metrics)

		assert.True(t, supported, "dimension %s should be filterable", dim)
		assert.Contains(t, q, "AND r.metric_path = ANY($2)")
		assert.Equal(t, []any{id, metrics}, args)
	}
}

// ── parseCSVQuery 不便于离线测（依赖 gin.Context）；逗号/空白逻辑在 handler 层由 e2e 覆盖。──

// ── buildResultsQuery：PM-DASH-DIMFILTER 子集过滤（ProductIDs / SubsetLDNs）────────

// 都不传 → 不新增维度过滤 WHERE 子句（向后兼容，现行行为不变）。
func Test_buildResultsQuery_NoDimFilter(t *testing.T) {
	id := uuid.New()
	q, args := buildResultsQuery(id, resultsFilter{}, 100, 0)

	assert.NotContains(t, q, "r.dimension_key = ANY")
	// object_ldn = ANY 不应出现（既无任务白名单也无子集过滤）
	assert.NotContains(t, q, "= ANY")
	assert.Equal(t, []any{id, 100, 0}, args)
}

// 传 ProductIDs → SQL 含 product 维度 dimension_key = ANY，占位号顺延入 args。
func Test_buildResultsQuery_ProductIDsSubset(t *testing.T) {
	id := uuid.New()
	pids := []string{uuid.NewString(), uuid.NewString()}
	q, args := buildResultsQuery(id, resultsFilter{ProductIDs: pids}, 100, 0)

	assert.Contains(t, q, "AND r.dimension = 'product' AND r.dimension_key = ANY($2)")
	assert.Contains(t, q, "LIMIT $3 OFFSET $4")
	if assert.Len(t, args, 4) {
		assert.Equal(t, id, args[0])
		assert.Equal(t, pids, args[1])
		assert.Equal(t, 100, args[2])
		assert.Equal(t, 0, args[3])
	}
}

// 传 SubsetLDNs → SQL 含投影 object_ldn = ANY（设备组/频段共用此子集参数）。
func Test_buildResultsQuery_SubsetLDNs(t *testing.T) {
	id := uuid.New()
	ldns := []string{"DeviceGroup=abc", "Band=42"}
	q, args := buildResultsQuery(id, resultsFilter{SubsetLDNs: ldns}, 100, 0)

	assert.Contains(t, q, "ELSE r.dimension_key END = ANY($2)")
	assert.Contains(t, q, "LIMIT $3 OFFSET $4")
	if assert.Len(t, args, 4) {
		assert.Equal(t, id, args[0])
		assert.Equal(t, ldns, args[1])
	}
}

// 任务白名单 ObjectLDNs + 维度子集 ProductIDs + SubsetLDNs 三者叠加：各自独立成子句，占位号连续递增。
func Test_buildResultsQuery_DimFilterStacksWithWhitelist(t *testing.T) {
	id := uuid.New()
	pids := []string{uuid.NewString()}
	ldns := []string{"Band=1"}
	whitelist := []string{"Cellid=1,PLMN=46000"}
	q, args := buildResultsQuery(id, resultsFilter{
		ObjectLDNs: whitelist,
		ProductIDs: pids,
		SubsetLDNs: ldns,
	}, 100, 0)

	// 顺序：任务白名单 object_ldn($2) → product_id($3) → 子集 object_ldn($4)
	assert.Contains(t, q, "ELSE r.dimension_key END = ANY($2)")
	assert.Contains(t, q, "AND r.dimension = 'product' AND r.dimension_key = ANY($3)")
	assert.Contains(t, q, "ELSE r.dimension_key END = ANY($4)")
	assert.Contains(t, q, "LIMIT $5 OFFSET $6")
	assert.Len(t, args, 6)
	// product_id 子句在第一个 object_ldn 子句之后、第二个之前
	first := strings.Index(q, "ANY($2)")
	prod := strings.Index(q, "r.dimension = 'product' AND r.dimension_key = ANY($3)")
	second := strings.Index(q, "ANY($4)")
	assert.True(t, first < prod && prod < second)
}

// ── results 分页截断：不再跑精确 COUNT(*) ────────────────────────────────────

// 都不传 → 主查询无新增维度 WHERE，也不生成 COUNT(*)。
func Test_buildResultsQuery_NoDimFilterAndNoExactCount(t *testing.T) {
	id := uuid.New()
	q, args := buildResultsQuery(id, resultsFilter{}, 100, 0)

	assert.NotContains(t, q, "COUNT(*)")
	assert.NotContains(t, q, "r.dimension_key = ANY")
	assert.NotContains(t, q, "= ANY")
	assert.Equal(t, []any{id, 100, 0}, args)
}

// 传 ProductIDs/SubsetLDNs → 主查询保留维度过滤；是否截断由 limit+1 返回行判断，不再额外 COUNT。
func Test_buildResultsQuery_DimFiltersWithoutExactCount(t *testing.T) {
	id := uuid.New()
	pids := []string{uuid.NewString()}
	ldns := []string{"DeviceGroup=g1"}
	f := resultsFilter{ProductIDs: pids, SubsetLDNs: ldns}

	dq, _ := buildResultsQuery(id, f, 100, 0)
	dqWhere := dq[:strings.LastIndex(dq, "ORDER BY")]
	assert.NotContains(t, dq, "COUNT(*)")
	assert.Contains(t, dqWhere, "AND r.dimension = 'product' AND r.dimension_key = ANY($2)")
	assert.Contains(t, dqWhere, "ELSE r.dimension_key END = ANY($3)")
}
