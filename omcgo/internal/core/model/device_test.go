package model

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDeriveOpState — 派生函数语义反退化：仅 DeviceActive → "1"，其余皆 "0"。
// 契约同 internal/device/device_info_pg_repository.go OpState filter
// (status='active' ↔ op_state='1')。
func TestDeriveOpState(t *testing.T) {
	cases := []struct {
		status DeviceStatus
		want   string
	}{
		{DeviceActive, "1"},
		{DeviceDiscovered, "0"},
		{DeviceRegistered, "0"},
		{DeviceProvisioning, "0"},
		{DeviceOffline, "0"},
		{DeviceMaintenance, "0"},
		{DeviceDecommissioned, "0"},
		{DeviceStatus(""), "0"},          // empty string → 未激活
		{DeviceStatus("garbage"), "0"},   // 未知值 → 未激活（fail-safe）
	}
	for _, c := range cases {
		c := c
		t.Run(string(c.status), func(t *testing.T) {
			assert.Equal(t, c.want, DeriveOpState(c.status))
		})
	}
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
