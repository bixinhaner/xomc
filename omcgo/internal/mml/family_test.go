package mml

import "testing"

// TestInferFamily 验证 §15.4 9 条 path-prefix-family 规则的命中与 fallback 行为。
//
// 覆盖：
//   - 每条规则至少一个正例（共 10 个正例，含 G-02 这种容易被规则 1 误捕的边界）
//   - 5 个 fallback 负例（不属于任何 family 的 group 应返回空字符串两个）
//   - 规则顺序敏感的边界：MU.SwUpgrade vs DeviceInfo.SwUpgrade 必须分到不同 family
func TestInferFamily(t *testing.T) {
	tests := []struct {
		name           string
		groupCode      string
		wantCode       string
		wantNameZh     string
		wantFamilyHit  bool // true 表示应命中某 family；false 表示应 fallback 到空
	}{
		// ============================================================
		// 正例（10 条）—— 覆盖 §15.4 9 类 family 各至少一例
		// ============================================================
		{
			name:          "G-01 设备信息根",
			groupCode:     "Device.DeviceInfo.*",
			wantCode:      "device_info",
			wantNameZh:    "设备信息",
			wantFamilyHit: true,
		},
		{
			name:          "G-02 设备版本升级（DeviceInfo.SwUpgrade，非 MU 子树）",
			groupCode:     "Device.DeviceInfo.SwUpgrade.*",
			wantCode:      "device_info",
			wantNameZh:    "设备信息",
			wantFamilyHit: true,
		},
		{
			name:          "G-64~G-68 硬件单元（MU 子树，非升级）",
			groupCode:     "Device.DeviceInfo.MU.{i}.*",
			wantCode:      "hardware_units",
			wantNameZh:    "硬件单元",
			wantFamilyHit: true,
		},
		{
			name:          "G-69~G-72 硬件升级（MU 子树的 SwUpgrade，优先级最高）",
			groupCode:     "Device.DeviceInfo.MU.{iα}.Slot.{iβ}.EU.{iγ}.RU.{iδ}.SwUpgrade.*",
			wantCode:      "hardware_upgrade",
			wantNameZh:    "硬件升级",
			wantFamilyHit: true,
		},
		{
			name:          "G-05~G-10 告警实例（FaultMgmt 子树）",
			groupCode:     "Device.FaultMgmt.CurrentAlarm.{i}.*",
			wantCode:      "alarm_instances",
			wantNameZh:    "告警实例",
			wantFamilyHit: true,
		},
		{
			name:          "G-18 FAPService 能力集",
			groupCode:     "Device.Services.FAPService.{i}.Capabilities.*",
			wantCode:      "capabilities",
			wantNameZh:    "能力集",
			wantFamilyHit: true,
		},
		{
			name:          "G-19 CellConfig 能力集（.Capabilities. 作为路径分量）",
			groupCode:     "Device.Services.FAPService.{i}.CellConfig.Capabilities.*",
			wantCode:      "capabilities",
			wantNameZh:    "能力集",
			wantFamilyHit: true,
		},
		{
			name:          "G-23 SCTP 配置",
			groupCode:     "Device.Services.FAPControl.Transport.SCTP.*",
			wantCode:      "sctp",
			wantNameZh:    "SCTP",
			wantFamilyHit: true,
		},
		{
			name:          "G-24 SCTP Assoc",
			groupCode:     "Device.Services.FAPControl.Transport.SCTP.Assoc.{i}.*",
			wantCode:      "sctp",
			wantNameZh:    "SCTP",
			wantFamilyHit: true,
		},
		{
			name:          "G-38 A3 测量控制（MeasureCtrl 子串）",
			groupCode:     "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.A3MeasureCtrl.{iβ}.*",
			wantCode:      "measure_ctrl",
			wantNameZh:    "测量控制",
			wantFamilyHit: true,
		},
		{
			name:          "G-52 Ethernet 接口 + IPv4 地址",
			groupCode:     "Device.Ethernet.Interface.{iα}.IPv4Address.{iβ}.*",
			wantCode:      "ethernet_ip",
			wantNameZh:    "以太网/IP",
			wantFamilyHit: true,
		},
		{
			name:          "G-58 Ethernet IP 路由",
			groupCode:     "Device.Ethernet.IpRoute.{i}.*",
			wantCode:      "ethernet_ip",
			wantNameZh:    "以太网/IP",
			wantFamilyHit: true,
		},
		{
			name:          "G-47 IdleMode IRAT GERAN 频组",
			groupCode:     "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.GERAN.GERANFreqGroup.{iβ}.*",
			wantCode:      "idle_mode",
			wantNameZh:    "空闲态移动性",
			wantFamilyHit: true,
		},

		// ============================================================
		// 负例（5 条）—— fallback 到空字符串，表示 group 自成一家
		// ============================================================
		{
			name:          "G-13 安全/接入网关（FAPControl.LTE.Gateway，不属于任何 family）",
			groupCode:     "Device.Services.FAPControl.LTE.Gateway.*",
			wantCode:      "",
			wantNameZh:    "",
			wantFamilyHit: false,
		},
		{
			name:          "G-03 软件控制（SoftwareCtrl 不属于 DeviceInfo / Ethernet / FaultMgmt）",
			groupCode:     "Device.SoftwareCtrl.*",
			wantCode:      "",
			wantNameZh:    "",
			wantFamilyHit: false,
		},
		{
			name:          "G-31~G-34 邻区族（暂保留独立，不并入 family）",
			groupCode:     "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.*",
			wantCode:      "",
			wantNameZh:    "",
			wantFamilyHit: false,
		},
		{
			name:          "G-26 MAC 层配置",
			groupCode:     "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.*",
			wantCode:      "",
			wantNameZh:    "",
			wantFamilyHit: false,
		},
		{
			name:          "G-60 时间配置（Time 不在任何 family 范围）",
			groupCode:     "Device.Time.*",
			wantCode:      "",
			wantNameZh:    "",
			wantFamilyHit: false,
		},
	}

	for _, tt := range tests {
		tt := tt // capture
		t.Run(tt.name, func(t *testing.T) {
			gotCode, gotNameZh := InferFamily(tt.groupCode)
			if gotCode != tt.wantCode {
				t.Errorf("InferFamily(%q) familyCode = %q, want %q",
					tt.groupCode, gotCode, tt.wantCode)
			}
			if gotNameZh != tt.wantNameZh {
				t.Errorf("InferFamily(%q) familyNameZh = %q, want %q",
					tt.groupCode, gotNameZh, tt.wantNameZh)
			}
			gotHit := gotCode != "" && gotNameZh != ""
			if gotHit != tt.wantFamilyHit {
				t.Errorf("InferFamily(%q) hit = %v, want %v",
					tt.groupCode, gotHit, tt.wantFamilyHit)
			}
		})
	}
}

// TestInferFamily_RuleOrderingPriority 验证规则顺序敏感的关键边界：
//   - Device.DeviceInfo.MU.{i}.SwUpgrade.* 同时匹配规则 1（hardware_upgrade）
//     和规则 2（hardware_units）和规则 3（device_info），但应被规则 1 截获
//   - Device.DeviceInfo.SwUpgrade.* 不含 .MU.，应跳过规则 1 走规则 3
func TestInferFamily_RuleOrderingPriority(t *testing.T) {
	// MU.SwUpgrade 路径优先匹配 hardware_upgrade
	gotCode, _ := InferFamily("Device.DeviceInfo.MU.{i}.SwUpgrade.UpgradeImage")
	if gotCode != "hardware_upgrade" {
		t.Errorf("MU+SwUpgrade should match hardware_upgrade, got %q", gotCode)
	}

	// DeviceInfo.SwUpgrade（无 MU）走 device_info
	gotCode, _ = InferFamily("Device.DeviceInfo.SwUpgrade.UpgradeImage")
	if gotCode != "device_info" {
		t.Errorf("DeviceInfo.SwUpgrade without MU should match device_info, got %q", gotCode)
	}

	// 纯 MU 子树（无 SwUpgrade）走 hardware_units
	gotCode, _ = InferFamily("Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.*")
	if gotCode != "hardware_units" {
		t.Errorf("MU subtree without SwUpgrade should match hardware_units, got %q", gotCode)
	}
}

// TestInferFamily_EmptyInput 边界：空字符串不应 panic 且应 fallback 到空。
func TestInferFamily_EmptyInput(t *testing.T) {
	gotCode, gotNameZh := InferFamily("")
	if gotCode != "" || gotNameZh != "" {
		t.Errorf("InferFamily(\"\") = (%q, %q), want (\"\", \"\")", gotCode, gotNameZh)
	}
}
