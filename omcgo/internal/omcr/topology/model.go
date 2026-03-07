package topology

import (
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/common/model"
)

// DeviceGroup represents a hierarchical group for organizing devices.
type DeviceGroup struct {
	ID          uuid.UUID          `json:"id"`
	Name        string             `json:"name"`
	ParentID    *uuid.UUID         `json:"parent_id,omitempty"`
	Carrier     model.CarrierCode  `json:"carrier,omitempty"`
	Description string             `json:"description,omitempty"`
	SortOrder   int                `json:"sort_order"`
	Children    []DeviceGroup      `json:"children,omitempty"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

// DeviceGroupMember represents a device's membership in a group.
type DeviceGroupMember struct {
	GroupID  uuid.UUID `json:"group_id"`
	DeviceID uuid.UUID `json:"device_id"`
	AddedAt  time.Time `json:"added_at"`
}
