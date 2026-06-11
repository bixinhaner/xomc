package rebootrecord

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// #63 设备组可见性 fail-closed 三态在 buildUnion 的 SQL 体现。

func TestBuildUnion_Superadmin_NoVisibilityFilter(t *testing.T) {
	sql, args := buildUnion(Filter{VisibleGroups: nil})
	assert.NotContains(t, sql, "device_group_members", "超管不应注入可见性子查询")
	assert.NotContains(t, sql, "FALSE")
	assert.Empty(t, args)
}

func TestBuildUnion_EmptyGroups_FailClosed(t *testing.T) {
	sql, args := buildUnion(Filter{VisibleGroups: []uuid.UUID{}})
	// 两半（event + fault）各自被加 FALSE → 整体空集。
	assert.Equal(t, 2, strings.Count(sql, "FALSE"), "空可见组应在两半各加 FALSE")
	assert.Empty(t, args)
}

func TestBuildUnion_WithGroups_SubqueryBothHalves(t *testing.T) {
	g1, g2 := uuid.New(), uuid.New()
	sql, args := buildUnion(Filter{VisibleGroups: []uuid.UUID{g1, g2}})
	// 两半各引用一次 device_group_members 子查询。
	assert.Equal(t, 2, strings.Count(sql, "device_id IN (SELECT device_id FROM device_group_members WHERE group_id = ANY("),
		"两半都应按 device_id 关联可见组子查询")
	require.Len(t, args, 1, "可见组作为单个数组参数")
	assert.ElementsMatch(t, []uuid.UUID{g1, g2}, args[0])
}

// 可见组过滤与其它条件共存时占位符连续不撞号（device_sn 先占 $1，可见组占 $2）。
func TestBuildUnion_PlaceholdersSequentialWithOtherFilters(t *testing.T) {
	g1 := uuid.New()
	sql, args := buildUnion(Filter{DeviceSN: "SN-1", VisibleGroups: []uuid.UUID{g1}})
	require.Len(t, args, 2)
	assert.Contains(t, sql, "$1", "device_sn 占 $1")
	assert.Contains(t, sql, "ANY($2)", "可见组占 $2，紧随其后不撞号")
}
