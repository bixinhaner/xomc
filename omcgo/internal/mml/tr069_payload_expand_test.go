package mml

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// 动态实例路径必须先绑定具体实例，不能折叠成宽泛 partial path。

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
	assert.Equal(t, in, out, "未绑定实例时保留占位符，由路径校验拒绝下发")
}

func TestExpandInstancePaths_NestedInstances(t *testing.T) {
	in := []string{
		"Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.PCI",
	}
	out := expandInstancePaths(in)
	assert.Equal(t, in, out, "多实例路径不能折叠为父对象")
}

func TestExpandInstancePaths_Deduplication(t *testing.T) {
	in := []string{
		"Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.PCI",
		"Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.CID",
		"Device.Services.FAPService.{i}.FAPControl.LTE.AdminState",
	}
	out := expandInstancePaths(in)
	assert.Equal(t, in, out, "不同叶子路径不能合并为父对象")
}

func TestExpandInstancePaths_MixedInstanceAndConcrete(t *testing.T) {
	in := []string{
		"Device.DeviceInfo.SerialNumber", // 无 {i}
		"Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.PCI", // 有 {i}
		"Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.5GCell.{i}.PCI",  // 有 {i}
	}
	out := expandInstancePaths(in)
	assert.Equal(t, in, out)
}

func TestExpandInstancePaths_DropsLeadingInstance(t *testing.T) {
	in := []string{"{i}.Bad.Path"}
	out := expandInstancePaths(in)
	assert.Equal(t, in, out, "以 {i} 开头也不能静默丢弃")
}

func TestExpandInstancePaths_PreservesOrder(t *testing.T) {
	in := []string{
		"Device.A.{i}.Foo",
		"Device.B.{i}.Bar",
		"Device.C.Concrete",
	}
	out := expandInstancePaths(in)
	assert.Equal(t, in, out, "保留路径及顺序")
}

// 端到端：含 {i} 的 paramRefs 未绑定实例时不得生成 partial path payload
func TestBuildParameterNames_WithInstancePaths(t *testing.T) {
	refs := []MMLParamRef{
		{Tr069Path: "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.PCI"},
		{Tr069Path: "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.CID"},
		{Tr069Path: "Device.DeviceInfo.SerialNumber"},
	}
	_, err := buildParameterNames(refs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "has_placeholder")
}
