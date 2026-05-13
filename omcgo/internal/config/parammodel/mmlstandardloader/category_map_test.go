package mmlstandardloader

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCategorizePath 覆盖 §7.2 表的 7 类映射核心场景。
//
// 每个 case 选取最具代表性的 path 验证 category 归属。
func TestCategorizePath(t *testing.T) {
	cases := []struct {
		path     string
		expected string
		desc     string
	}{
		// 4 告警查询
		{"Device.FaultMgmt.CurrentAlarm.AlarmActive", CategoryAlarmQuery, "FaultMgmt 当前告警"},
		{"Device.FaultMgmt.ExpeditedEvent.EventTime", CategoryAlarmQuery, "FaultMgmt 加急事件"},

		// 5 性能采集
		{"Device.FAP.PerfMgmt.SOMETHING", CategoryPerfMgmt, "FAP.PerfMgmt 性能"},
		{"Device.Services.FAPService.{i}.FAPControl.NR.SelfConfig.SONConfigParam.ANR.KPI", CategoryPerfMgmt, "FAPControl SelfConfig KPI"},
		{"Device.KeepalivedMgmt.Counter.X", CategoryPerfMgmt, "KeepalivedMgmt Counter"},

		// 2 邻区管理
		{"Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell", CategoryNeighborMgmt, "LTE NeighborList"},
		{"Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.5GCell", CategoryNeighborMgmt, "5G NeighborList"},
		{"DeviceGSM.Bts.{i}.NeighborList", CategoryNeighborMgmt, "GSM NeighborList"},
		{"DeviceGSM.Bts.{i}.Si2quaterNeighborList", CategoryNeighborMgmt, "GSM Si2quater NeighborList"},

		// 7 版本管理（在 DeviceInfo 兜底前抢）
		{"Device.DeviceInfo.AdditionalSoftwareVersion", CategoryVersionMgmt, "Software 路径"},
		{"Device.RemoteDeviceList.1.SoftwareVersion", CategoryVersionMgmt, "RemoteDeviceList Software"},
		{"Device.Software", CategoryVersionMgmt, "顶层 Software"},
		{"Device.SoftwareCtrl.X", CategoryVersionMgmt, "SoftwareCtrl 控制"},

		// 1 小区管理
		{"Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Common.CellID", CategoryCellMgmt, "LTE CellConfig 普通"},
		{"Device.Services.FAPService.{i}.FAPControl.LTE.AdminState", CategoryCellMgmt, "FAPControl"},
		{"Device.Services.GsmBTSCellDT.1.Foo", CategoryCellMgmt, "GsmBTSCellDT"},

		// 6 传输管理
		{"Device.Services.FAPService.{i}.Transport.SCTPS.X", CategoryTransportMgmt, "Transport"},
		{"Device.Services.FAPService.{i}.Ipsec.X", CategoryTransportMgmt, "FAPService Ipsec"},
		{"Device.Ethernet.Interface.{i}.MACAddress", CategoryTransportMgmt, "Ethernet"},
		{"Device.IP.Interface.{i}.IPv4Address", CategoryTransportMgmt, "IP"},
		{"Device.ManagementServer.URL", CategoryTransportMgmt, "ACS URL"},

		// 3 基站管理（兜底 + 显式 DeviceInfo）
		{"Device.DeviceInfo.SerialNumber", CategoryBaseStation, "DeviceInfo 设备信息"},
		{"Device.DeviceInfo.AntennaInfo.Azimuth", CategoryBaseStation, "AntennaInfo"},
		{"Device.FAP.SomethingNotPerf", CategoryBaseStation, "FAP 非 PerfMgmt"},
		{"Device.Time.LocalTimeZone", CategoryBaseStation, "Time"},
		{"boardconf.HALOD.HALOD_IP", CategoryBaseStation, "boardconf"},
		{"InternetGatewayDevice.Time.NTPServer1", CategoryBaseStation, "InternetGatewayDevice"},
		{"DeviceGSM.Bts.{i}.Foo", CategoryBaseStation, "GSM 非 NeighborList"},

		// 边界 — 未知前缀
		{"Some.Random.Path", CategoryBaseStation, "未知前缀 fail-safe 兜底"},
	}

	for _, c := range cases {
		c := c
		t.Run(c.desc, func(t *testing.T) {
			assert.Equal(t, c.expected, CategorizePath(c.path),
				"path=%s expected=%s desc=%s", c.path, c.expected, c.desc)
		})
	}
}
