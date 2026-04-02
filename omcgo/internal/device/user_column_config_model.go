package device

import (
	"time"

	"github.com/google/uuid"
)

// ColumnConfig stores a user's customized column layout for a page.
type ColumnConfig struct {
	UserID    uuid.UUID    `json:"user_id"`
	PageKey   string       `json:"page_key"`
	Columns   []ColumnItem `json:"columns"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

// ColumnItem describes a single column in the user's layout.
type ColumnItem struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	Visible bool   `json:"visible"`
	Width   int    `json:"width,omitempty"`
	Order   int    `json:"order"`
}

// DefaultDeviceColumns returns the default column layout for the device list page.
func DefaultDeviceColumns() []ColumnItem {
	return []ColumnItem{
		{Key: "serial_number", Label: "序列号", Visible: true, Order: 1},
		{Key: "device_name", Label: "设备名称", Visible: true, Order: 2},
		{Key: "status", Label: "状态", Visible: true, Order: 3},
		{Key: "carrier", Label: "运营商", Visible: true, Order: 4},
		{Key: "technology", Label: "制式", Visible: true, Order: 5},
		{Key: "model_name", Label: "型号", Visible: true, Order: 6},
		{Key: "manufacturer", Label: "厂商", Visible: true, Order: 7},
		{Key: "rf_status", Label: "射频状态", Visible: true, Order: 8},
		{Key: "cell_status", Label: "小区状态", Visible: true, Order: 9},
		{Key: "alarm_severity", Label: "告警级别", Visible: true, Order: 10},
		{Key: "site_name", Label: "站点", Visible: false, Order: 11},
		{Key: "firmware_version", Label: "固件版本", Visible: false, Order: 12},
		{Key: "ip_address", Label: "IP 地址", Visible: false, Order: 13},
		{Key: "last_inform_at", Label: "最后心跳", Visible: false, Order: 14},
		{Key: "gps_status", Label: "GPS 状态", Visible: false, Order: 15},
		{Key: "license_status", Label: "License 状态", Visible: false, Order: 16},
		{Key: "bandwidth", Label: "带宽", Visible: false, Order: 17},
		{Key: "transmit_power", Label: "发射功率", Visible: false, Order: 18},
	}
}
