// Package model 定义系统核心数据实体和通用类型。
// 包括设备、告警、参数、PM 计数器/KPI、分页请求等。
// 常量与枚举类型通过 constants.go 从 global 包再导出。
package model

import (
	"time"

	"github.com/google/uuid"
)

// Device 表示一台被管理的基站/小基站设备。
// 对应数据库 devices 表，是设备全生命周期管理的核心实体。
// 设备由 TR-069 Bootstrap Inform 自动注册入库，状态流转见 DeviceStatus。
type Device struct {
	ID                          uuid.UUID              `json:"id" db:"id"`
	SerialNumber                string                 `json:"serial_number" db:"serial_number"`
	OUI                         string                 `json:"oui" db:"oui"`
	ProductClass                string                 `json:"product_class" db:"product_class"`
	Manufacturer                string                 `json:"manufacturer" db:"manufacturer"`
	ModelName                   string                 `json:"model_name" db:"model_name"`
	Carrier                     CarrierCode            `json:"carrier" db:"carrier"`
	Technology                  Technology             `json:"technology" db:"technology"`
	// T-0098 P5-02：DataModelID 字段已删除（devices.data_model_id 列 DROP，路由改由 productClass + ProductRegistry）。
	Status                      DeviceStatus           `json:"status" db:"status"`
	// OpState 激活状态（前端展示用，'1'=激活 / '0'=未激活）。
	// 由 Status 派生，**不入库**：scan/Register/Update 后通过
	// model.DeriveOpState(Status) 统一回填，保持 status='active' ↔ op_state='1'
	// 的双向契约（与 device_info_pg_repository.go OpState filter 一致）。
	OpState                     string                 `json:"op_state" db:"-"`
	FirmwareVersion             string                 `json:"firmware_version" db:"firmware_version"`
	IPAddress                   string                 `json:"ip_address" db:"ip_address"`
	ConnectionRequestURL        string                 `json:"connection_request_url" db:"connection_request_url"`
	NatDetected                 bool                   `json:"nat_detected" db:"nat_detected"`
	UDPConnectionRequestAddress string                 `json:"udp_connection_request_address,omitempty" db:"udp_connection_request_address"`
	LastInformAt                *time.Time             `json:"last_inform_at,omitempty" db:"last_inform_at"`
	LastInformEvents            []string               `json:"last_inform_events,omitempty" db:"last_inform_events"`
	LastBootAt                  *time.Time             `json:"last_boot_at,omitempty" db:"last_boot_at"`
	BootCount                   int                    `json:"boot_count" db:"boot_count"`
	InformInterval              int                    `json:"inform_interval" db:"inform_interval"`
	SiteName                    string                 `json:"site_name" db:"site_name"`
	SiteID                      string                 `json:"site_id" db:"site_id"`
	Latitude                    float64                `json:"latitude" db:"latitude"`
	Longitude                   float64                `json:"longitude" db:"longitude"`
	ExtensionData               map[string]interface{} `json:"extension_data,omitempty" db:"extension_data"`
	CreatedAt                   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt                   time.Time              `json:"updated_at" db:"updated_at"`
	DeletedAt                   *time.Time             `json:"deleted_at,omitempty" db:"deleted_at"`
	DeletedBy                   string                 `json:"deleted_by,omitempty" db:"deleted_by"` // who moved to recycle bin

	// Group info (from device_groups table via device_group_members)
	GroupName                   string                 `json:"group_name,omitempty" db:"group_name"`
}

// DeriveOpState 把设备 Status 翻译为前端"激活状态"展示值。
//
// 契约（与 internal/device/device_info_pg_repository.go OpState filter 一致）：
//
//	DeviceActive → "1"（激活）
//	其它 status  → "0"（未激活，含 discovered/registered/provisioning/offline/maintenance/decommissioned）
//
// FE 渲染（omcmb/.../DeviceList/index.tsx）：opState ∈ {'1', 'active'} 显示 success，
// 否则显示 error；返 'unknown' 会被前端视作未激活。
func DeriveOpState(status DeviceStatus) string {
	if status == DeviceActive {
		return "1"
	}
	return "0"
}
