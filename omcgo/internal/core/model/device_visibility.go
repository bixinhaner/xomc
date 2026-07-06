package model

import "github.com/google/uuid"

// DeviceVisibilityGrant describes one device-data grant derived from a role_device_groups row.
// GroupIDs holds the expanded device-group subtree for that grant.
// Technologies is nil or empty when the grant does not restrict technology,
// and a non-empty slice lists allowed device technologies.
type DeviceVisibilityGrant struct {
	GroupIDs     []uuid.UUID `json:"group_ids,omitempty"`
	Technologies []Technology `json:"technologies,omitempty"`
}