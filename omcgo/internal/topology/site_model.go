package topology

import (
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// SiteStatus represents the status of a site.
type SiteStatus string

const (
	SiteActive      SiteStatus = "active"
	SiteInactive    SiteStatus = "inactive"
	SiteMaintenance SiteStatus = "maintenance"
)

// NodeStatus represents the status of a topology node.
type NodeStatus string

const (
	NodeOnline      NodeStatus = "online"
	NodeOffline     NodeStatus = "offline"
	NodeAlarm       NodeStatus = "alarm"
	NodeMaintenance NodeStatus = "maintenance"
)

// EdgeStatus represents the status of a topology edge.
type EdgeStatus string

const (
	EdgeActive   EdgeStatus = "active"
	EdgeInactive EdgeStatus = "inactive"
	EdgeDegraded EdgeStatus = "degraded"
)

// Site represents a physical site with geographic coordinates.
type Site struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	DomainID    *uuid.UUID `json:"domain_id,omitempty"`
	Address     string     `json:"address,omitempty"`
	Longitude   *float64   `json:"longitude,omitempty"`
	Latitude    *float64   `json:"latitude,omitempty"`
	DeviceCount int        `json:"device_count"`
	Status      SiteStatus `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// TopoNode represents a node in the topology graph.
type TopoNode struct {
	ID        uuid.UUID  `json:"id"`
	Label     string     `json:"label"`
	NodeType  string     `json:"node_type"`
	X         float64    `json:"x"`
	Y         float64    `json:"y"`
	Status    NodeStatus `json:"status"`
	DeviceSN  string     `json:"device_sn,omitempty"`
	SiteID    *uuid.UUID `json:"site_id,omitempty"`
	DomainID  *uuid.UUID `json:"domain_id,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// TopoEdge represents an edge (link) between two topology nodes.
type TopoEdge struct {
	ID        uuid.UUID  `json:"id"`
	SourceID  uuid.UUID  `json:"source_id"`
	TargetID  uuid.UUID  `json:"target_id"`
	Label     string     `json:"label,omitempty"`
	Status    EdgeStatus `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
}

// TopoGraph combines nodes and edges for the graph endpoint.
type TopoGraph struct {
	Nodes []TopoNode `json:"nodes"`
	Edges []TopoEdge `json:"edges"`
}

// GeoData combines sites and nodes that have coordinates for the GIS endpoint.
type GeoData struct {
	Sites []Site     `json:"sites"`
	Nodes []TopoNode `json:"nodes"`
}

// SiteFilter specifies criteria for listing sites.
type SiteFilter struct {
	DomainID *uuid.UUID
	Status   *SiteStatus
	model.ListRequest
}

// TopoNodeFilter specifies criteria for listing topology nodes.
type TopoNodeFilter struct {
	DomainID *uuid.UUID
	NodeType *string
	model.ListRequest
}

// TopoEdgeFilter specifies criteria for listing topology edges.
type TopoEdgeFilter struct {
	model.ListRequest
}
