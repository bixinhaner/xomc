package geofence

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// RuleType identifies the geometry and safety semantics of one geofence.
type RuleType string

const (
	RuleTypePolygonAllowZone RuleType = "polygon_allow_zone"
	RuleTypeBaselineRadius   RuleType = "baseline_radius"
)

// DefinitionStatus is the lifecycle state of a stable geofence definition.
type DefinitionStatus string

const (
	DefinitionStatusDraft    DefinitionStatus = "draft"
	DefinitionStatusEnabled  DefinitionStatus = "enabled"
	DefinitionStatusDisabled DefinitionStatus = "disabled"
	DefinitionStatusArchived DefinitionStatus = "archived"
)

// VersionStatus is the lifecycle state of an immutable geofence version.
type VersionStatus string

const (
	VersionStatusDraft      VersionStatus = "draft"
	VersionStatusPublished  VersionStatus = "published"
	VersionStatusSuperseded VersionStatus = "superseded"
)

// BindingStatus is the lifecycle state of a device/geofence relationship.
type BindingStatus string

const (
	BindingStatusPending   BindingStatus = "pending"
	BindingStatusActive    BindingStatus = "active"
	BindingStatusSuspended BindingStatus = "suspended"
	BindingStatusRemoved   BindingStatus = "removed"
)

// Definition is the stable identity and lifecycle of a geofence.
type Definition struct {
	ID               uuid.UUID        `json:"id"`
	Name             string           `json:"name"`
	Carrier          string           `json:"carrier"`
	RuleType         RuleType         `json:"rule_type"`
	OwnerDeviceID    *uuid.UUID       `json:"owner_device_id,omitempty"`
	Status           DefinitionStatus `json:"status"`
	CurrentVersionID *uuid.UUID       `json:"current_version_id,omitempty"`
	CreatedBy        uuid.UUID        `json:"created_by"`
	UpdatedBy        uuid.UUID        `json:"updated_by"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

// Version is one immutable geometry and policy snapshot.
type Version struct {
	ID           uuid.UUID       `json:"id"`
	GeofenceID   uuid.UUID       `json:"geofence_id"`
	Version      int64           `json:"version"`
	Status       VersionStatus   `json:"status"`
	GeometryJSON json.RawMessage `json:"geometry"`
	BoundingBox  BoundingBox     `json:"bounding_box"`
	PolicyJSON   json.RawMessage `json:"policy"`
	CreatedBy    uuid.UUID       `json:"created_by"`
	PublishedBy  *uuid.UUID      `json:"published_by,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	PublishedAt  *time.Time      `json:"published_at,omitempty"`
}

// Binding keeps device supervision separate from current evaluation state.
type Binding struct {
	ID           uuid.UUID     `json:"id"`
	DeviceID     uuid.UUID     `json:"device_id"`
	GeofenceID   uuid.UUID     `json:"geofence_id"`
	RuleType     RuleType      `json:"rule_type"`
	Status       BindingStatus `json:"status"`
	BindSource   string        `json:"bind_source"`
	BoundBy      uuid.UUID     `json:"bound_by"`
	BoundAt      time.Time     `json:"bound_at"`
	RemovedBy    *uuid.UUID    `json:"removed_by,omitempty"`
	RemovedAt    *time.Time    `json:"removed_at,omitempty"`
	RemoveReason string        `json:"remove_reason,omitempty"`
}

type DefinitionFilter struct {
	Carrier       string
	Status        DefinitionStatus
	Name          string
	VisibleGroups []uuid.UUID
}
