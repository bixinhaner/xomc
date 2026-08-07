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
// 设备由 TR-069 Bootstrap Inform 自动注册入库。
//
// T-0162 解耦：状态流转见 DeviceLifecycle（业务进度）+ IsOnline（实时连接），
// 不再用单字段 Status（DB 列已 DROP，Go 字段过渡期保留供 P3 渐进清理）。
type Device struct {
	ID           uuid.UUID   `json:"id" db:"id"`
	SerialNumber string      `json:"serial_number" db:"serial_number"`
	OUI          string      `json:"oui" db:"oui"`
	ProductClass string      `json:"product_class" db:"product_class"`
	ProductName  string      `json:"product_name,omitempty" db:"-"`
	Manufacturer string      `json:"manufacturer" db:"manufacturer"`
	ModelName    string      `json:"model_name" db:"model_name"`
	Carrier      CarrierCode `json:"carrier" db:"carrier"`
	Technology   Technology  `json:"technology" db:"technology"`
	// T-0098 P5-02：DataModelID 字段已删除（devices.data_model_id 列 DROP，路由改由 productClass + ProductRegistry）。
	// ProductID / ParamModelID：productClass → product 装配件路由结果。Inform 路径
	// applyProductMetadata 按 productClass 命中后回填，让设备所属产品随 product_class 变更实时重算。
	// nil = 未命中（孤儿）/ 未在本路径计算，持久化时不覆盖既有值。
	ProductID    *uuid.UUID `json:"product_id,omitempty" db:"product_id"`
	ParamModelID *uuid.UUID `json:"param_model_id,omitempty" db:"param_model_id"`

	// T-0162 新字段：业务生命周期（lifecycle_state 列）+ 实时在线（is_online 列）
	LifecycleState DeviceLifecycle `json:"lifecycle_state" db:"lifecycle_state"`
	IsOnline       bool            `json:"is_online" db:"is_online"`

	// DEPRECATED (T-0162): Status DB 列已删除，Go 字段过渡期保留兼容老调用方，
	// db tag "-" 让 ScanStruct 跳过；P3 完成所有调用方迁移后整体删除此字段。
	Status DeviceStatus `json:"status,omitempty" db:"-"`
	// OpState 激活状态（前端展示用，'1'=激活 / '0'=未激活）。
	// 由 Status 派生，**不入库**：scan/Register/Update 后通过
	// model.DeriveOpState(Status) 统一回填，保持 status='active' ↔ op_state='1'
	// 的双向契约（与 device_info_pg_repository.go OpState filter 一致）。
	OpState                     string     `json:"op_state" db:"-"`
	FirmwareVersion             string     `json:"firmware_version" db:"firmware_version"`
	IPAddress                   string     `json:"ip_address" db:"ip_address"`
	ConnectionRequestURL        string     `json:"connection_request_url" db:"connection_request_url"`
	NatDetected                 bool       `json:"nat_detected" db:"nat_detected"`
	UDPConnectionRequestAddress string     `json:"udp_connection_request_address,omitempty" db:"udp_connection_request_address"`
	LastInformAt                *time.Time `json:"last_inform_at,omitempty" db:"last_inform_at"`
	LastInformEvents            []string   `json:"last_inform_events,omitempty" db:"last_inform_events"`
	// T-0124: 上一次 Path B 全量参数同步完成时刻（由 HandleSyncResultPathB 在 BatchUpsert 成功后回写，
	// 不区分触发源 device_online / periodic / firmware_changed / manual）；PeriodicSyncer 据此找过期设备。
	LastParamSyncAt *time.Time `json:"last_param_sync_at,omitempty" db:"last_param_sync_at"`
	// 上次参数同步失败时刻 + 错误信息 (migration 000142)。与 LastParamSyncAt 互斥:
	// 成功路径写 succeeded 并清失败列;失败路径写 failed 并清 succeeded。任一非空即"上次终态",
	// 都为空表示从未同步过。前端 GetSyncStatus 据此渲染 idle vs failed 子态。
	LastParamSyncFailedAt *time.Time `json:"last_param_sync_failed_at,omitempty" db:"last_param_sync_failed_at"`
	LastParamSyncError    *string    `json:"last_param_sync_error,omitempty" db:"last_param_sync_error"`
	// T-0173: 最近一次离线原因（heartbeat_timeout / manual / reboot / NULL）。
	// 由 DeviceStatusReconciler 或管理面操作写入；从未离线为 nil。诊断字段。
	LastOfflineReason *string    `json:"last_offline_reason,omitempty" db:"last_offline_reason"`
	LastBootAt        *time.Time `json:"last_boot_at,omitempty" db:"last_boot_at"`
	BootCount         int        `json:"boot_count" db:"boot_count"`
	InformInterval    int        `json:"inform_interval" db:"inform_interval"`
	// DeviceName 设备名称。DB 物理列名仍为 site_name（历史原因，未做物理迁移），
	// 故 db tag 与字段名/JSON 不一致——这是有意为之，业务/API 层统一用 device_name。
	DeviceName      string                 `json:"device_name" db:"site_name"`
	SiteID          string                 `json:"site_id" db:"site_id"`
	Latitude        *float64               `json:"latitude,omitempty" db:"latitude"`
	Longitude       *float64               `json:"longitude,omitempty" db:"longitude"`
	ExtensionData   map[string]interface{} `json:"extension_data,omitempty" db:"extension_data"`
	CreatedAt       time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at" db:"updated_at"`
	DeletedAt       *time.Time             `json:"deleted_at,omitempty" db:"deleted_at"`
	DeletedBy       string                 `json:"deleted_by,omitempty" db:"deleted_by"` // who moved to recycle bin
	RecycleType     string                 `json:"recycle_type,omitempty" db:"recycle_type"`
	RecycleExecutor string                 `json:"recycle_executor,omitempty" db:"recycle_executor"`

	// Group info (from device_groups table via device_group_members)
	GroupName string `json:"group_name,omitempty" db:"group_name"`
}

// DeriveOpStateActivated 由"激活时间"派生前端"激活状态"展示值。
//
// 这是激活状态的**权威口径**：激活是一次性、单调的持久事实 —— 设备首次上线
// （inform）即激活，之后离线/维护/生命周期流转都不再改变它。
// activatedAt（= device_info.first_online_time）非空 → "1"（已激活），否则 "0"。
//
// 相比 DeriveOpState(status)，本函数不从混合语义的 status 反推，因此不会出现
// "在线却未激活 / 离线却激活" 之类自相矛盾。设备列表（DeviceWithInfo，已 JOIN
// device_info 拿得到 first_online_time）一律用本函数。
//
// FE 渲染（omcmb/.../DeviceList/index.tsx）：opState ∈ {'1', 'active'} 显示 success，
// 否则显示 error。
func DeriveOpStateActivated(activatedAt *time.Time) string {
	if activatedAt != nil {
		return "1"
	}
	return "0"
}

// DeriveOpState 由 Status 派生"激活状态"——**回退口径**。
//
// 仅供拿不到 device_info.first_online_time 的裸 Device 扫描（设备详情 / 回收站）
// 使用。优先用 DeriveOpStateActivated。规则：
//
//	active / offline / maintenance     → "1"（已激活，含临时失联或维护中）
//	discovered / registered / provisioning / decommissioned / 空 / 未知 → "0"（未激活）
func DeriveOpState(status DeviceStatus) string {
	switch status {
	case DeviceActive, DeviceOffline, DeviceMaintenance:
		return "1"
	default:
		return "0"
	}
}
