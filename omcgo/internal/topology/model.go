package topology

import (
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// DeviceGroup represents a hierarchical group for organizing devices.
type DeviceGroup struct {
	ID          uuid.UUID         `json:"id"`
	Name        string            `json:"name"`
	ParentID    *uuid.UUID        `json:"parent_id,omitempty"`
	Carrier     model.CarrierCode `json:"carrier,omitempty"`
	Description string            `json:"description,omitempty"`
	SortOrder   int               `json:"sort_order"`
	Level       int               `json:"level"`
	Status      string            `json:"status"`
	IsDefault   bool              `json:"is_default"`
	Remark      string            `json:"remark,omitempty"`
	CreatedBy   string            `json:"created_by,omitempty"`
	UpdatedBy   string            `json:"updated_by,omitempty"`
	DeviceCount int               `json:"device_count"`
	Children    []DeviceGroup     `json:"children,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// DeviceGroupMember represents a device's membership in a group.
type DeviceGroupMember struct {
	GroupID  uuid.UUID `json:"group_id"`
	DeviceID uuid.UUID `json:"device_id"`
	AddedAt  time.Time `json:"added_at"`
}

// --- Request/Response DTOs ---

// CreateGroupRequest is the payload for creating a device group.
type CreateGroupRequest struct {
	Name      string          `json:"name" binding:"required"`
	ParentID  string          `json:"parent_id"`
	Carrier   string          `json:"carrier"`
	Remark    string          `json:"remark"`
	SortOrder int             `json:"sort_order"`
	SubGroups []SubGroupInput `json:"sub_groups,omitempty"`
}

// SubGroupInput defines a sub-group to create in batch.
type SubGroupInput struct {
	Name      string   `json:"name" binding:"required"`
	Remark    string   `json:"remark"`
	DeviceIDs []string `json:"device_ids,omitempty"`
}

// UpdateGroupRequest is the payload for updating a device group.
type UpdateGroupRequest struct {
	Name      *string `json:"name"`
	Remark    *string `json:"remark"`
	SortOrder *int    `json:"sort_order"`
}

// CheckDeleteResponse describes the impact of deleting a group.
type CheckDeleteResponse struct {
	CanDelete   bool   `json:"can_delete"`
	HasDevices  bool   `json:"has_devices"`
	DeviceCount int    `json:"device_count"`
	ChildCount  int    `json:"child_count"`
	Message     string `json:"message"`
}

// GroupStats provides aggregate statistics about device groups.
type GroupStats struct {
	TotalGroups      int `json:"total_groups"`
	GroupedDevices   int `json:"grouped_devices"`
	UngroupedDevices int `json:"ungrouped_devices"`
}

// MoveDevicesRequest is the payload for moving devices between groups.
type MoveDevicesRequest struct {
	DeviceIDs     []string `json:"device_ids" binding:"required"`
	TargetGroupID string   `json:"target_group_id" binding:"required"`
}

// BatchSortRequest is the payload for batch-updating sort orders.
type BatchSortRequest struct {
	Items []SortItem `json:"items" binding:"required"`
}

// SortItem specifies a sort order for a group.
type SortItem struct {
	ID        string `json:"id" binding:"required"`
	SortOrder int    `json:"sort_order"`
}

// BatchDevicesRequest is the payload for batch adding/removing devices.
type BatchDevicesRequest struct {
	DeviceIDs []string `json:"device_ids" binding:"required"`
}

// TreeResponse is the response for the device group tree endpoint.
type TreeResponse struct {
	Items []DeviceGroup `json:"items"`
	Stats *GroupStats   `json:"stats"`
}
