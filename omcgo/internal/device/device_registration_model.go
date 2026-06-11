package device

import (
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/global"
	"github.com/omcgo/omcgo/internal/core/model"
)

// DeviceRegistration represents a pre-registered device entry.
type DeviceRegistration struct {
	ID           uuid.UUID                 `json:"id"`
	SerialNumber string                    `json:"serial_number"`
	GroupID      *uuid.UUID                `json:"group_id,omitempty"`
	Carrier      model.CarrierCode         `json:"carrier"`
	SiteName     string                    `json:"site_name,omitempty"`
	Longitude    *float64                  `json:"longitude,omitempty"`
	Latitude     *float64                  `json:"latitude,omitempty"`
	Status       global.RegistrationStatus `json:"status"`
	Remark       string                    `json:"remark,omitempty"`
	CreatedBy    string                    `json:"created_by,omitempty"`
	CreatedAt    time.Time                 `json:"created_at"`
	UpdatedAt    time.Time                 `json:"updated_at"`
}

// CreateRegistrationRequest is the input for pre-registering a device.
// GroupID 必填：device_registrations.group_id 在 DB 层为 NOT NULL 约束，
// 缺参应在绑定阶段返回 400，而非落库触发 23502 炸 500（issue #126 第 2 项）。
type CreateRegistrationRequest struct {
	SerialNumber string   `json:"serial_number" binding:"required"`
	GroupID      string   `json:"group_id" binding:"required"`
	Carrier      string   `json:"carrier" binding:"required"`
	SiteName     string   `json:"site_name"`
	Longitude    *float64 `json:"longitude"`
	Latitude     *float64 `json:"latitude"`
	Remark       string   `json:"remark"`
}

// RegistrationFilter provides filtering options for listing registrations.
type RegistrationFilter struct {
	Status       *global.RegistrationStatus
	SerialNumber *string
	Carrier      *model.CarrierCode
	// VisibleGroups 是 #64 设备组数据权限的三态可见分组（nil=超管不过滤 / []=fail-closed 空集 /
	// [g...]=仅这些组）。预注册表自带 group_id 列，过滤经 authz.ApplyGroupVisibilityFilter
	// 直接对 group_id 取交；group_id IS NULL（待上线未分组）天然只对超管（nil）可见，对非超管
	// 不命中 IN 条件被排除，符合「未分组仅超管/默认组可见，不对所有人可见」的判定。
	VisibleGroups []uuid.UUID
	model.ListRequest
}

// ImportResult describes the outcome of a batch import.
type ImportResult struct {
	Total   int      `json:"total"`
	Success int      `json:"success"`
	Failed  int      `json:"failed"`
	Errors  []string `json:"errors,omitempty"`
}
