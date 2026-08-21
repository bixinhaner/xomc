package device

import (
	"github.com/google/uuid"
)

// UpdateDeviceInfoRequest 定义操作员可手动更新的字段。
// 这些字段是 device_info 表中的运维标识字段，不包含 TR069 同步的状态字段。
type UpdateDeviceInfoRequest struct {
	// DeviceName 设备名称
	DeviceName *string `json:"device_name"`

	// SiteID 站点 ID。UPS 设备写入 device_ups_info，基站仍走 devices.site_id 更新接口。
	SiteID *string `json:"site_id"`

	// Address 安装地址
	Address *string `json:"address"`

	// Remark 备注信息
	Remark *string `json:"remark"`

	// ProjectStatus 项目状态
	// 示例值: "在建", "调试", "商用", "退网"
	ProjectStatus *string `json:"project_status"`

	// Height 安装高度（米）
	Height *float64 `json:"height"`
}

// DeviceInfoSyncUpdate 保存 TR069 参数同步时需要更新的字段。
// 在 Inform 处理流程中，从 device_parameters 计算出需要更新的字段，
// 通过此结构批量更新到 device_info 表。
type DeviceInfoSyncUpdate struct {
	// DeviceID 目标设备 ID
	DeviceID uuid.UUID

	// Fields 需要更新的字段映射
	// key: 数据库列名, value: 字段值
	// 示例: {"rf_status": "on", "cell_status": "normal", "num_of_cells": 2}
	Fields map[string]interface{}
}
