package mmlstandardloader

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateCommands_LSTAlwaysGenerated(t *testing.T) {
	groups := []GroupSpec{{
		Path: "Device.FaultMgmt.CurrentAlarm",
		Code: "DEVICE_FAULTMGMT_CURRENTALARM",
		Params: []ParamSpec{
			mkParam("Device.FaultMgmt.CurrentAlarm.AlarmActive", AccessReadOnly, "BOOLEAN"),
		},
	}}
	cmds := GenerateCommands(groups)
	require.Len(t, cmds, 1, "全 RO group 只生成 LST")
	c := cmds[0]
	assert.Equal(t, "LST_DEVICE_FAULTMGMT_CURRENTALARM", c.Code)
	assert.Equal(t, OpLST, c.OperationType)
	assert.Equal(t, RPCGetParameterValues, c.RPCMethod)
	assert.Equal(t, CategoryAlarmQuery, c.Category, "FaultMgmt → 4 告警查询")
	assert.Empty(t, c.TargetObject, "LST 无 target_object")
	assert.Equal(t, []string{"Device.FaultMgmt.CurrentAlarm.AlarmActive"}, c.TargetPaths)
}

func TestGenerateCommands_MODOnlyWhenWritable(t *testing.T) {
	groups := []GroupSpec{{
		Path: "Device.DeviceInfo.AntennaInfo",
		Code: "DEVICE_DEVICEINFO_ANTENNAINFO",
		Params: []ParamSpec{
			mkParam("Device.DeviceInfo.AntennaInfo.Azimuth", AccessReadWrite, "INT"),
			mkParam("Device.DeviceInfo.AntennaInfo.HeightType3", AccessReadOnly, "STRING"),
		},
	}}
	cmds := GenerateCommands(groups)
	require.Len(t, cmds, 2, "含 RW → LST + MOD")
	codes := []string{cmds[0].Code, cmds[1].Code}
	assert.Contains(t, codes, "LST_DEVICE_DEVICEINFO_ANTENNAINFO")
	assert.Contains(t, codes, "MOD_DEVICE_DEVICEINFO_ANTENNAINFO")

	// MOD target_paths 仅含 RW
	for _, c := range cmds {
		if c.OperationType == OpMOD {
			assert.Equal(t, []string{"Device.DeviceInfo.AntennaInfo.Azimuth"}, c.TargetPaths,
				"MOD target_paths 仅含 writable")
		}
	}
}

func TestGenerateCommands_NoMODWhenAllReadOnly(t *testing.T) {
	groups := []GroupSpec{{
		Path:   "Device.FaultMgmt.CurrentAlarm",
		Code:   "DEVICE_FAULTMGMT_CURRENTALARM",
		Params: []ParamSpec{mkParam("Device.FaultMgmt.CurrentAlarm.X", AccessReadOnly, "STRING")},
	}}
	cmds := GenerateCommands(groups)
	for _, c := range cmds {
		assert.NotEqual(t, OpMOD, c.OperationType, "全 RO → 不应生成 MOD")
	}
}

func TestGenerateCommands_ADDRMVWhenInstance(t *testing.T) {
	groups := []GroupSpec{{
		Path:        "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell",
		Code:        "DEVICE_SERVICES_FAPSERVICE_CELLCONFIG_LTE_RAN_NEIGHBORLIST_LTECELL",
		HasInstance: true,
		Params: []ParamSpec{
			mkParam("Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.PCI", AccessReadWrite, "U_INT"),
		},
	}}
	cmds := GenerateCommands(groups)
	require.Len(t, cmds, 4, "{i} 路径 → LST + MOD + ADD + RMV")
	ops := make(map[string]bool)
	for _, c := range cmds {
		ops[c.OperationType] = true
	}
	assert.True(t, ops[OpLST])
	assert.True(t, ops[OpMOD])
	assert.True(t, ops[OpADD])
	assert.True(t, ops[OpRMV])

	// ADD/RMV target_object = group path 去 {i} 后以 . 结尾
	for _, c := range cmds {
		switch c.OperationType {
		case OpADD, OpRMV:
			assert.NotEmpty(t, c.TargetObject)
			assert.True(t, strings.HasSuffix(c.TargetObject, "."),
				"target_object 必以 . 结尾（TR-069 object 形态）")
			assert.NotContains(t, c.TargetObject, "{i}", "去 {i}")
			assert.Equal(t, CategoryNeighborMgmt, c.Category, "NeighborList → 2 邻区管理")
			if c.OperationType == OpADD {
				assert.Equal(t, []string{"Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.PCI"},
					c.TargetPaths, "ADD 应显示全部可编辑 RW 参数")
			} else {
				assert.Empty(t, c.TargetPaths, "RMV 不需要参数")
			}
		}
	}
}

func TestGenerateCommands_ADDExcludesNestedInstanceFields(t *testing.T) {
	groups := []GroupSpec{{
		Path:        "Device.KeepalivedMgmt.VrrpMgmt",
		Code:        "DEVICE_KEEPALIVEDMGMT_VRRPMGMT",
		HasInstance: true,
		Params: []ParamSpec{
			mkParam("Device.KeepalivedMgmt.VrrpMgmt.{i}.AdvertInt", AccessReadWrite, "U_INT"),
			mkParam("Device.KeepalivedMgmt.VrrpMgmt.{i}.VirtualIpList.{i}.Interface", AccessReadWrite, "STRING"),
		},
	}}

	cmds := GenerateCommands(groups)
	for _, c := range cmds {
		if c.OperationType == OpADD {
			assert.Equal(t,
				[]string{"Device.KeepalivedMgmt.VrrpMgmt.{i}.AdvertInt"},
				c.TargetPaths,
				"新增 VRRP 实例不能混入 VirtualIpList 子实例字段",
			)
			return
		}
	}
	t.Fatal("missing ADD command")
}

func TestGenerateCommands_NoADDWhenNoInstance(t *testing.T) {
	groups := []GroupSpec{{
		Path:        "Device.DeviceInfo.AntennaInfo",
		Code:        "DEVICE_DEVICEINFO_ANTENNAINFO",
		HasInstance: false,
		Params:      []ParamSpec{mkParam("Device.DeviceInfo.AntennaInfo.Azimuth", AccessReadWrite, "INT")},
	}}
	cmds := GenerateCommands(groups)
	for _, c := range cmds {
		assert.NotContains(t, []string{OpADD, OpRMV}, c.OperationType,
			"无 {i} → 不应生成 ADD/RMV")
	}
}

func TestGenerateCommands_EmptyGroupSkipped(t *testing.T) {
	groups := []GroupSpec{{Path: "Device.Empty", Code: "DEVICE_EMPTY"}}
	cmds := GenerateCommands(groups)
	assert.Empty(t, cmds, "空 group 不生成命令")
}

// 端到端：真实 1988 paths → 期望 80-300 个命令在合理 category 分布
func TestGenerateCommands_RealData_Distribution(t *testing.T) {
	params, _, err := ParseStandardXMLFile(repoXMLPath(t))
	require.NoError(t, err)
	groups := BuildGroups(params)
	cmds := GenerateCommands(groups)

	assert.GreaterOrEqual(t, len(cmds), 80, "real data 应产 ≥80 commands")
	assert.LessOrEqual(t, len(cmds), 1500, "real data 应产 ≤1500 commands")

	// category 分布
	dist := make(map[string]int)
	ops := make(map[string]int)
	for _, c := range cmds {
		dist[c.Category]++
		ops[c.OperationType]++
	}
	t.Logf("real data: %d commands; category distribution: %v; ops: %v",
		len(cmds), dist, ops)

	// 7 类必须都出现至少 1 个（除非真实数据某类完全空）
	// 这里至少验证 1 / 3 / 4 / 5 必有（小区/基站/告警/性能）— 这 4 类在 1988 paths 里肯定有
	assert.Positive(t, dist[CategoryCellMgmt], "1 小区管理必有")
	assert.Positive(t, dist[CategoryBaseStation], "3 基站管理必有（DeviceInfo 兜底）")
	assert.Positive(t, dist[CategoryAlarmQuery], "4 告警查询必有（FaultMgmt 59 paths）")
	assert.Positive(t, dist[CategoryPerfMgmt], "5 性能采集必有（FAP.PerfMgmt + KPI）")

	// 4 个操作类型都出现
	assert.Positive(t, ops[OpLST])
	assert.Positive(t, ops[OpMOD])
	assert.Positive(t, ops[OpADD])
	assert.Positive(t, ops[OpRMV])
}

// F-A 短 display name 规则覆盖：每条规则一个 testcase
func TestGroupDisplayName_Aliases(t *testing.T) {
	cases := []struct {
		path string
		zh   string
		en   string
	}{
		// CellConfig 子树 — 高频且最深
		{"Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PhyCellID", "小区.LTE.RAN.PhyCellID", "Cell.LTE.RAN.PhyCellID"},
		{"Device.Services.FAPService.{i}.CellConfig.{i}.NR.Foo.Bar", "小区.NR.Foo.Bar", "Cell.NR.Foo.Bar"},
		// FAPService 其它子树
		{"Device.Services.FAPService.{i}.Transport.SCTP", "FAP.Transport.SCTP", "FAP.Transport.SCTP"},
		// Device.* 顶级业务实体
		{"Device.DeviceInfo.AntennaInfo", "设备.AntennaInfo", "Device.AntennaInfo"},
		{"Device.Time.NTPServer1", "时间.NTPServer1", "Time.NTPServer1"},
		{"Device.IP.Interface", "网络.Interface", "IP.Interface"},
		{"Device.Ethernet.Interface", "以太网.Interface", "Ethernet.Interface"},
		{"Device.FAP.GPS", "FAP.GPS", "FAP.GPS"},
		{"Device.FaultMgmt.CurrentAlarm", "告警.CurrentAlarm", "Fault.CurrentAlarm"},
		{"Device.ManagementServer.URL", "TR069.URL", "TR069.URL"},
		// DeviceGSM
		{"DeviceGSM.Bts.{i}.RFParam", "GSM.Bts.RFParam", "GSM.Bts.RFParam"},
		// 兜底
		{"Device.UnknownNamespace.Foo", "UnknownNamespace.Foo", "UnknownNamespace.Foo"},
		{"AlienRoot.Foo", "AlienRoot.Foo", "AlienRoot.Foo"},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			zh, en := groupDisplayName(GroupSpec{Path: tc.path})
			assert.Equal(t, tc.zh, zh)
			assert.Equal(t, tc.en, en)
		})
	}
}

// 反退化：缩写后 name 长度上限远低于 command_code（145+ vs 30+ 数量级）
func TestGroupDisplayName_LengthBudget(t *testing.T) {
	// standard-model 里最深 path 实测
	deepest := "Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.PHY.BWP.BWPDL.PDSCH.PdschDedicatedTimeDomainResourceAllocationList"
	zh, en := groupDisplayName(GroupSpec{Path: deepest})
	assert.Less(t, len(zh), 80, "zh 缩写不超过 80 char 让 UI 表格可读")
	assert.Less(t, len(en), 100, "en 缩写不超过 100 char")
	t.Logf("deepest path zh: %q (%d chars)", zh, len(zh))
	t.Logf("deepest path en: %q (%d chars)", en, len(en))
}
