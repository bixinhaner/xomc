package deviceaccess

import (
	"encoding/json"
	"net"
	"time"
)

type IdentityStatus string

const (
	IdentityStatusResolved   IdentityStatus = "resolved"
	IdentityStatusUnresolved IdentityStatus = "unresolved"
	IdentityStatusConflict   IdentityStatus = "conflict"
	IdentityStatusAmbiguous  IdentityStatus = "ambiguous"
)

type IdentitySnapshot struct {
	ID                 string         `json:"id"`
	RequestID          string         `json:"request_id"`
	DecisionID         string         `json:"decision_id,omitempty"`
	DeviceID           string         `json:"device_id,omitempty"`
	CandidateID        string         `json:"candidate_id,omitempty"`
	Carrier            string         `json:"carrier"`
	SerialNumber       string         `json:"serial_number"`
	DeviceCode         string         `json:"device_code,omitempty"`
	CloudKey           string         `json:"cloud_key,omitempty"`
	OUI                string         `json:"oui,omitempty"`
	ProductClass       string         `json:"product_class,omitempty"`
	RawRemoteIP        net.IP         `json:"raw_remote_ip,omitempty"`
	ObservedRemoteIP   net.IP         `json:"observed_remote_ip,omitempty"`
	InformEvent        string         `json:"inform_event,omitempty"`
	InformTime         time.Time      `json:"inform_time"`
	IdentitySource     string         `json:"identity_source"`
	IdentityStatus     IdentityStatus `json:"identity_status"`
	IdentityReasonCode ReasonCode     `json:"identity_reason_code,omitempty"`
	CreatedAt          time.Time      `json:"created_at"`
}

type DecisionArchive struct {
	ID         string     `json:"id"`
	DecisionID string     `json:"decision_id"`
	ArchivedBy string     `json:"archived_by"`
	Reason     string     `json:"reason"`
	ArchivedAt time.Time  `json:"archived_at"`
	RestoredBy string     `json:"restored_by,omitempty"`
	RestoredAt *time.Time `json:"restored_at,omitempty"`
}

type AccessManualOperationItem struct {
	ID         string          `json:"id"`
	UserID     string          `json:"user_id,omitempty"`
	Username   string          `json:"username"`
	Operation  string          `json:"operation"`
	ResourceID string          `json:"resource_id"`
	Details    json.RawMessage `json:"details"`
	CreatedAt  time.Time       `json:"created_at"`
}
