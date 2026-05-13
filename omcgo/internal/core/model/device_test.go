package model

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDeriveOpState — 派生函数语义反退化：op_state 表达生命周期"已激活"状态，
// 与 status 的"当前在线"语义解耦。
//
// 已激活集合: {active, offline, maintenance}
//   - active: 在线运行中
//   - offline: 已激活但当前离线（OfflineDetector 标记，不应翻成未激活）
//   - maintenance: 已激活但维护中
// 未激活集合: {discovered, registered, provisioning, decommissioned, 空, 未知}
func TestDeriveOpState(t *testing.T) {
	cases := []struct {
		status DeviceStatus
		want   string
		note   string
	}{
		{DeviceActive, "1", "在线运行 → 激活"},
		{DeviceOffline, "1", "已激活但离线 → 仍激活（关键：心跳超时不翻转激活态）"},
		{DeviceMaintenance, "1", "已激活但维护中 → 仍激活"},
		{DeviceDiscovered, "0", "刚发现未注册 → 未激活"},
		{DeviceRegistered, "0", "API 预登记未来电 → 未激活"},
		{DeviceProvisioning, "0", "配置中 → 未激活"},
		{DeviceDecommissioned, "0", "已退役 → 未激活"},
		{DeviceStatus(""), "0", "空串 fail-safe → 未激活"},
		{DeviceStatus("garbage"), "0", "未知值 fail-safe → 未激活"},
	}
	for _, c := range cases {
		c := c
		t.Run(string(c.status), func(t *testing.T) {
			assert.Equal(t, c.want, DeriveOpState(c.status), c.note)
		})
	}
}

// TestDeriveOpState_OfflineDoesNotDeactivate — 反退化核心场景：设备 inform 后
// active → 离线检测器标 offline → op_state 不应翻转为未激活。
// 现网设备 120200024719AAB0039 离线后激活状态变未激活的 bug 由本测试守护。
func TestDeriveOpState_OfflineDoesNotDeactivate(t *testing.T) {
	// 初次 inform 后
	assert.Equal(t, "1", DeriveOpState(DeviceActive))
	// OfflineDetector.markOffline 把 status 改为 offline 后
	assert.Equal(t, "1", DeriveOpState(DeviceOffline),
		"离线设备 op_state 必须保持 '1'：在线/离线和激活状态是两个独立维度")
}

// TestDevice_OpState_JSONRoundTrip — Device 结构 JSON 序列化后必含 op_state 字段，
// 不带 db tag 不影响 squirrel 列扫描（依赖位置 Scan，不依赖 tag）。
func TestDevice_OpState_JSONRoundTrip(t *testing.T) {
	d := Device{
		SerialNumber: "SN-001",
		Status:       DeviceActive,
		OpState:      DeriveOpState(DeviceActive),
	}
	b, err := json.Marshal(d)
	require.NoError(t, err)
	var got map[string]interface{}
	require.NoError(t, json.Unmarshal(b, &got))
	assert.Equal(t, "active", got["status"], "status 字段保留")
	assert.Equal(t, "1", got["op_state"], "op_state 应序列化到 JSON 供前端消费")
}

// TestDevice_OpState_DerivedFromStatus — 验证一个未派生 OpState 的设备
// JSON 输出为空串（提醒调用方必须显式 DeriveOpState）。
func TestDevice_OpState_DerivedFromStatus(t *testing.T) {
	d := Device{Status: DeviceActive}
	// 未派生时 OpState 为零值 ""，FE 会 fallback 到 'unknown' → "未激活"。
	// 这是为什么 scan/Register/Update 三类入口都必须显式 DeriveOpState。
	assert.Equal(t, "", d.OpState)

	d.OpState = DeriveOpState(d.Status)
	assert.Equal(t, "1", d.OpState)
}
