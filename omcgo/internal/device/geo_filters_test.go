package device

import (
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/global"
	"github.com/omcgo/omcgo/internal/core/model"
)

// 默认 L2 组是一个真实分组 ID；只有空字符串用于兼容历史无归属数据兜底。
// 本测试防止 repository 收到 "" 走 dg.id IN (”) 的退化语义。
func TestSplitGeoGroupIDs(t *testing.T) {
	tests := []struct {
		name             string
		in               []string
		wantRealIDs      []string
		wantIncludeUngro bool
	}{
		{
			name:             "nil 输入：无过滤",
			in:               nil,
			wantRealIDs:      nil,
			wantIncludeUngro: false,
		},
		{
			name:             "空切片：无过滤",
			in:               []string{},
			wantRealIDs:      nil,
			wantIncludeUngro: false,
		},
		{
			name:             "默认组 ID 保留为真实分组",
			in:               []string{global.DefaultLevel2GroupID},
			wantRealIDs:      []string{global.DefaultLevel2GroupID},
			wantIncludeUngro: false,
		},
		{
			name:             "仅真实分组",
			in:               []string{"4e04dc11-aaaa-bbbb-cccc-000000000001"},
			wantRealIDs:      []string{"4e04dc11-aaaa-bbbb-cccc-000000000001"},
			wantIncludeUngro: false,
		},
		{
			name: "真实分组 + 默认组混合：都保留为真实分组",
			in: []string{
				"4e04dc11-aaaa-bbbb-cccc-000000000001",
				global.DefaultLevel2GroupID,
				"4e04dc11-aaaa-bbbb-cccc-000000000002",
			},
			wantRealIDs: []string{
				"4e04dc11-aaaa-bbbb-cccc-000000000001",
				global.DefaultLevel2GroupID,
				"4e04dc11-aaaa-bbbb-cccc-000000000002",
			},
			wantIncludeUngro: false,
		},
		{
			name:             "空字符串转历史无归属兜底：避免 dg.id IN ('') 退化",
			in:               []string{"", "4e04dc11-aaaa-bbbb-cccc-000000000001", ""},
			wantRealIDs:      []string{"4e04dc11-aaaa-bbbb-cccc-000000000001"},
			wantIncludeUngro: true,
		},
		{
			name:             "全是空字符串：历史无归属兜底",
			in:               []string{"", ""},
			wantRealIDs:      nil,
			wantIncludeUngro: true,
		},
		{
			name:             "重复的默认组 ID：仍按真实分组传入",
			in:               []string{global.DefaultLevel2GroupID, global.DefaultLevel2GroupID},
			wantRealIDs:      []string{global.DefaultLevel2GroupID, global.DefaultLevel2GroupID},
			wantIncludeUngro: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotIDs, gotInc := splitGeoGroupIDs(tc.in)
			assert.Equal(t, tc.wantRealIDs, gotIDs, "realIDs 不一致")
			assert.Equal(t, tc.wantIncludeUngro, gotInc, "includeUngrouped 不一致")
		})
	}
}

// Issue #490 回归：前端三档 status（onlineActive/onlineInactive/offline）必须翻
// 译为 DB 层 lifecycle_state + is_online 组合。空串不施加过滤；未知值兜底透传
// 保持向后兼容（老调用方可能直接传 "active" 这种 DB 值）。ListGeo 与
// GetGeoStats 共享此 helper，断言两接口翻译规则不分叉。
func TestParseGeoStatusFilter(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []model.DeviceStatus
	}{
		{
			name: "空串：不施加过滤",
			in:   "",
			want: nil,
		},
		{
			name: "onlineActive 单档",
			in:   "onlineActive",
			want: []model.DeviceStatus{model.DeviceActive},
		},
		{
			name: "onlineInactive 单档展开为 registered + provisioning",
			in:   "onlineInactive",
			want: []model.DeviceStatus{model.DeviceRegistered, model.DeviceProvisioning},
		},
		{
			name: "offline 单档展开为 offline+maintenance+discovered+decommissioned",
			in:   "offline",
			want: []model.DeviceStatus{
				model.DeviceOffline,
				model.DeviceMaintenance,
				model.DeviceDiscovered,
				model.DeviceDecommissioned,
			},
		},
		{
			name: "多档逗号拼接",
			in:   "onlineActive,offline",
			want: []model.DeviceStatus{
				model.DeviceActive,
				model.DeviceOffline,
				model.DeviceMaintenance,
				model.DeviceDiscovered,
				model.DeviceDecommissioned,
			},
		},
		{
			name: "未知值兜底透传（向后兼容直接传 DB 字面量的老调用方）",
			in:   "weird_legacy_value",
			want: []model.DeviceStatus{model.DeviceStatus("weird_legacy_value")},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseGeoStatusFilter(tc.in)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestParseGeoBounds(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want *GeoBounds
	}{
		{
			name: "合法视口",
			raw:  "73.5,135.1,3.8,53.6",
			want: &GeoBounds{MinLng: 73.5, MaxLng: 135.1, MinLat: 3.8, MaxLat: 53.6},
		},
		{name: "空串不施加过滤", raw: "", want: nil},
		{name: "字段数量错误", raw: "73.5,135.1,3.8", want: nil},
		{name: "非数字不能退化为零坐标", raw: "bad,135.1,3.8,53.6", want: nil},
		{name: "拒绝非有限数", raw: "NaN,135.1,3.8,53.6", want: nil},
		{name: "拒绝反向经度边界", raw: "135.1,73.5,3.8,53.6", want: nil},
		{name: "拒绝越界纬度", raw: "73.5,135.1,-91,53.6", want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, parseGeoBounds(tt.raw))
		})
	}
}

// Issue #490 回归：applyGeoGroupFilter 必须按 (realIDs, includeUngrouped) 四象
// 限拼出正确的 WHERE 子句。任何一个分支漏拼或 OR/AND 错位都会导致地图统计与列表
// 不一致。直接断言生成的 SQL 字符串里包含/不包含关键片段。
func TestApplyGeoGroupFilter(t *testing.T) {
	const ungroupedSnippet = "NOT EXISTS (SELECT 1 FROM device_group_members"

	t.Run("两路都为空：不加 WHERE", func(t *testing.T) {
		b := sq.Select("d.id").From("devices d").PlaceholderFormat(sq.Dollar)
		got := applyGeoGroupFilter(b, nil, false)
		q, args, err := got.ToSql()
		require.NoError(t, err)
		assert.NotContains(t, q, "dg.id IN")
		assert.NotContains(t, q, ungroupedSnippet)
		assert.Empty(t, args)
	})

	t.Run("仅真实 ID：dg.id IN (...)", func(t *testing.T) {
		b := sq.Select("d.id").From("devices d").PlaceholderFormat(sq.Dollar)
		got := applyGeoGroupFilter(b, []string{"g1", "g2"}, false)
		q, args, err := got.ToSql()
		require.NoError(t, err)
		assert.Contains(t, q, "dg.id IN")
		assert.NotContains(t, q, ungroupedSnippet)
		assert.ElementsMatch(t, []any{"g1", "g2"}, args)
	})

	t.Run("仅未分组：NOT EXISTS 子查询", func(t *testing.T) {
		b := sq.Select("d.id").From("devices d").PlaceholderFormat(sq.Dollar)
		got := applyGeoGroupFilter(b, nil, true)
		q, args, err := got.ToSql()
		require.NoError(t, err)
		assert.NotContains(t, q, "dg.id IN")
		assert.Contains(t, q, ungroupedSnippet)
		assert.Empty(t, args)
	})

	t.Run("混合：OR(dg.id IN, NOT EXISTS)", func(t *testing.T) {
		b := sq.Select("d.id").From("devices d").PlaceholderFormat(sq.Dollar)
		got := applyGeoGroupFilter(b, []string{"g1"}, true)
		q, args, err := got.ToSql()
		require.NoError(t, err)
		assert.Contains(t, q, "dg.id IN")
		assert.Contains(t, q, ungroupedSnippet)
		assert.Contains(t, q, " OR ", "真实分组与未分组之间必须是 OR")
		assert.ElementsMatch(t, []any{"g1"}, args)
	})
}

// Issue #490 回归：applyGeoStatusFilter 必须把多档 DeviceStatus 翻译成
// OR 拼接的 (lifecycle_state, is_online) 组合。空切片不施加 WHERE；Active /
// Offline 必须带 is_online 判定，其它档位只断 lifecycle_state。
func TestApplyGeoStatusFilter(t *testing.T) {
	t.Run("空切片：不加 WHERE", func(t *testing.T) {
		b := sq.Select("d.id").From("devices d").PlaceholderFormat(sq.Dollar)
		got := applyGeoStatusFilter(b, nil)
		q, _, err := got.ToSql()
		require.NoError(t, err)
		assert.NotContains(t, q, "lifecycle_state")
		assert.NotContains(t, q, "is_online")
	})

	t.Run("Active：lifecycle=commissioned AND is_online=true", func(t *testing.T) {
		b := sq.Select("d.id").From("devices d").PlaceholderFormat(sq.Dollar)
		got := applyGeoStatusFilter(b, []model.DeviceStatus{model.DeviceActive})
		q, args, err := got.ToSql()
		require.NoError(t, err)
		assert.Contains(t, q, "d.lifecycle_state")
		assert.Contains(t, q, "d.is_online")
		// 期望 2 个参数：lifecycle 值 + is_online 值
		assert.Len(t, args, 2)
	})

	t.Run("Registered：仅断 lifecycle 不带 is_online", func(t *testing.T) {
		b := sq.Select("d.id").From("devices d").PlaceholderFormat(sq.Dollar)
		got := applyGeoStatusFilter(b, []model.DeviceStatus{model.DeviceRegistered})
		q, args, err := got.ToSql()
		require.NoError(t, err)
		assert.Contains(t, q, "d.lifecycle_state")
		assert.NotContains(t, q, "d.is_online")
		assert.Len(t, args, 1)
	})

	t.Run("Active+Offline 多档：OR 拼接", func(t *testing.T) {
		b := sq.Select("d.id").From("devices d").PlaceholderFormat(sq.Dollar)
		got := applyGeoStatusFilter(b, []model.DeviceStatus{
			model.DeviceActive,
			model.DeviceOffline,
		})
		q, args, err := got.ToSql()
		require.NoError(t, err)
		assert.Contains(t, q, " OR ")
		// Active(2) + Offline(2) = 4 个参数
		assert.Len(t, args, 4)
	})
}
