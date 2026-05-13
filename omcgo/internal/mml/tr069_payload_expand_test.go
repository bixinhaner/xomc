package mml

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Sprint B Q-V3-2 决议：LST 含 {i} 路径透明展开为 TR-069 partial path。
// 测试 expandInstancePaths 函数的展开规则 + 边界情况。

func TestExpandInstancePaths_NoInstance_PassThrough(t *testing.T) {
	in := []string{
		"Device.DeviceInfo.AntennaInfo.Azimuth",
		"Device.FaultMgmt.CurrentAlarm.AlarmActive",
	}
	out := expandInstancePaths(in)
	assert.Equal(t, in, out, "无 {i} 路径原样保留")
}

func TestExpandInstancePaths_SingleInstance(t *testing.T) {
	in := []string{
		"Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.PCI",
	}
	out := expandInstancePaths(in)
	assert.Equal(t, []string{"Device.Services.FAPService."}, out,
		"单 {i} → 首个 {i} 之前 + 末尾点（保守 partial path）")
}

func TestExpandInstancePaths_NestedInstances(t *testing.T) {
	in := []string{
		"Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.PCI",
	}
	out := expandInstancePaths(in)
	assert.Equal(t, []string{"Device.Services.FAPService."}, out,
		"多 {i} → 取首个 {i} 之前的最短前缀（最稳）")
}

func TestExpandInstancePaths_Deduplication(t *testing.T) {
	in := []string{
		"Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.PCI",
		"Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.CID",
		"Device.Services.FAPService.{i}.FAPControl.LTE.AdminState",
	}
	out := expandInstancePaths(in)
	assert.Equal(t, []string{"Device.Services.FAPService."}, out,
		"3 个不同 {i} 路径共享同前缀 → 合并为 1 个 partial path")
}

func TestExpandInstancePaths_MixedInstanceAndConcrete(t *testing.T) {
	in := []string{
		"Device.DeviceInfo.SerialNumber",                                                              // 无 {i}
		"Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.PCI",              // 有 {i}
		"Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.5GCell.{i}.PCI",               // 有 {i}
	}
	out := expandInstancePaths(in)
	// 顺序：无 {i} 的先保留；{i} 的合并为 1 个 partial（首个 {i} 同位置）
	assert.Equal(t, []string{
		"Device.DeviceInfo.SerialNumber",
		"Device.Services.FAPService.",
	}, out)
}

func TestExpandInstancePaths_DropsLeadingInstance(t *testing.T) {
	in := []string{"{i}.Bad.Path"}
	out := expandInstancePaths(in)
	assert.Empty(t, out, "以 {i} 开头无法构造合规 partial → 跳过")
}

func TestExpandInstancePaths_PreservesOrder(t *testing.T) {
	in := []string{
		"Device.A.{i}.Foo",
		"Device.B.{i}.Bar",
		"Device.C.Concrete",
	}
	out := expandInstancePaths(in)
	assert.Equal(t, []string{
		"Device.A.",
		"Device.B.",
		"Device.C.Concrete",
	}, out, "保留首次出现顺序")
}

// 端到端：含 {i} 的 paramRefs 经 buildParameterNames 后产 partial path payload
func TestBuildParameterNames_WithInstancePaths(t *testing.T) {
	refs := []MMLParamRef{
		{Tr069Path: "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.PCI"},
		{Tr069Path: "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.CID"},
		{Tr069Path: "Device.DeviceInfo.SerialNumber"},
	}
	raw, err := buildParameterNames(refs)
	assert.NoError(t, err)
	assert.Contains(t, string(raw), `"Device.Services.FAPService."`,
		"含 {i} 的 2 条 path 合并为 1 个 partial path")
	assert.Contains(t, string(raw), `"Device.DeviceInfo.SerialNumber"`,
		"无 {i} 的 path 原样保留")
}
