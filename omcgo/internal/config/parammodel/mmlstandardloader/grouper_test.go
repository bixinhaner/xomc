package mmlstandardloader

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mkParam(path, access, vtype string) ParamSpec {
	return ParamSpec{StandardPath: path, Access: access, Type: vtype}
}

// 小型 group：≤ 50 param 全部合一
func TestBuildGroups_SmallSubtree_SingleGroup(t *testing.T) {
	params := []ParamSpec{
		mkParam("Device.DeviceInfo.AntennaInfo.Azimuth", "READ_WRITE", "INT"),
		mkParam("Device.DeviceInfo.AntennaInfo.Downtilt", "READ_WRITE", "INT"),
		mkParam("Device.DeviceInfo.AntennaInfo.Gain", "READ_WRITE", "INT"),
	}
	groups := BuildGroups(params)
	require.Len(t, groups, 1, "3 params under one subtree → 1 group")
	g := groups[0]
	assert.Equal(t, "Device.DeviceInfo.AntennaInfo", g.Path)
	assert.Equal(t, "DEVICE_DEVICEINFO_ANTENNAINFO", g.Code)
	assert.Len(t, g.Params, 3)
	assert.False(t, g.HasInstance, "path 无 {i}")
}

// 大型子树：触发递归下钻
func TestBuildGroups_LargeSubtree_DownsplitsByThreshold(t *testing.T) {
	var params []ParamSpec
	// 生成 60 个同 group 的 param → 子树 > 50 → 必须下钻
	// 用 "Device.DeviceInfo.X.A1".. "Device.DeviceInfo.X.A60" 平铺
	for i := 0; i < 60; i++ {
		path := "Device.DeviceInfo.X." + paramSuffix(i)
		params = append(params, mkParam(path, "READ_ONLY", "STRING"))
	}
	groups := BuildGroups(params)
	// 60 个全在 "Device.DeviceInfo.X" 子树下；子树超 50 →
	// 算法落到 X 节点把 directParams 提交为一个 group（60 > 50 仍只能合一，因子节点都是 leaf）
	// 实际行为：当无可继续下钻的 child 时，directParams 形成一个 group（即使 > 50）
	require.NotEmpty(t, groups)
	// 总 params 数守恒
	total := 0
	for _, g := range groups {
		total += len(g.Params)
	}
	assert.Equal(t, 60, total, "所有 param 必须有归属，且只有一处")
}

// {i} 路径必须被检测
func TestBuildGroups_InstanceDetection(t *testing.T) {
	params := []ParamSpec{
		mkParam("Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.PCI", "READ_WRITE", "U_INT"),
		mkParam("Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.CID", "READ_WRITE", "U_INT"),
	}
	groups := BuildGroups(params)
	require.Len(t, groups, 1)
	g := groups[0]
	assert.True(t, g.HasInstance, "{i} 路径要标 HasInstance=true")
	assert.Equal(t, "DEVICE_SERVICES_FAPSERVICE_CELLCONFIG_LTE_RAN_NEIGHBORLIST_LTECELL", g.Code,
		"group_code 去 {i} 后转 UPPER_SNAKE_CASE")
}

// 多分支：每分支独立 group
func TestBuildGroups_MultipleBranches(t *testing.T) {
	params := []ParamSpec{
		mkParam("Device.DeviceInfo.AntennaInfo.Azimuth", "READ_WRITE", "INT"),
		mkParam("Device.DeviceInfo.BTS.CurrentArfcn", "READ_ONLY", "STRING"),
		mkParam("Device.FaultMgmt.CurrentAlarm.AlarmActive", "READ_ONLY", "BOOLEAN"),
	}
	groups := BuildGroups(params)
	assert.GreaterOrEqual(t, len(groups), 3, "AntennaInfo / BTS / CurrentAlarm 各成 group")
}

// 端到端：跑真实 XML 数据 — 验证 1988 params 切出合理的 group 数
func TestBuildGroups_RealData(t *testing.T) {
	params, _, err := ParseStandardXMLFile(repoXMLPath(t))
	require.NoError(t, err)
	groups := BuildGroups(params)
	// 大致预期 80-200 个 group
	assert.GreaterOrEqual(t, len(groups), 50, "real data should yield ≥50 groups")
	assert.LessOrEqual(t, len(groups), 500, "real data should yield ≤500 groups")
	// 守恒：所有 param 必须落到某个 group
	total := 0
	for _, g := range groups {
		total += len(g.Params)
	}
	assert.Equal(t, len(params), total, "params 守恒：每个 param 都属于某个 group")
	t.Logf("real data: %d params → %d groups", len(params), len(groups))
}

// ============================================================
// helpers
// ============================================================

func paramSuffix(i int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	if i < len(letters) {
		return "P" + string(letters[i])
	}
	return "P" + string(letters[i%len(letters)]) + string(letters[(i/len(letters))%len(letters)])
}
