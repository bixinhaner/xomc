package attention

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type SectionStatus string

const (
	SectionOK      SectionStatus = "ok"
	SectionPartial SectionStatus = "partial"
	SectionError   SectionStatus = "error"
)

type Section string

const (
	SectionAbnormalities Section = "abnormalities"
	SectionTodos         Section = "todos"
)

type Kind string

const (
	KindActiveAlarm        Kind = "active_alarm"
	KindDeviceAccessReview Kind = "device_access_review"
)

type Action string

const (
	ActionViewAlarm       Action = "view_alarm"
	ActionReviewCandidate Action = "review_device_candidate"
)

type Response struct {
	Abnormalities AttentionSection `json:"abnormalities"`
	Todos         AttentionSection `json:"todos"`
	GeneratedAt   time.Time        `json:"generated_at"`
}

type AttentionSection struct {
	Status SectionStatus `json:"status"`
	Total  int64         `json:"total"`
	Items  []Item        `json:"items"`
}

type Page struct {
	Status   SectionStatus `json:"status"`
	Total    int64         `json:"total"`
	Items    []Item        `json:"items"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
}

type Item struct {
	ID             string     `json:"id"`
	Kind           Kind       `json:"kind"`
	Source         string     `json:"source"`
	SourceID       string     `json:"source_id"`
	Title          string     `json:"title"`
	Summary        string     `json:"summary"`
	Severity       string     `json:"severity,omitempty"`
	Priority       string     `json:"priority"`
	Risk           string     `json:"risk,omitempty"`
	Target         *Target    `json:"target,omitempty"`
	OccurredAt     *time.Time `json:"occurred_at,omitempty"`
	CreatedAt      *time.Time `json:"created_at,omitempty"`
	UpdatedAt      *time.Time `json:"updated_at,omitempty"`
	DetailRoute    string     `json:"detail_route"`
	AllowedActions []Action   `json:"allowed_actions"`
}

type Target struct {
	Type         string `json:"type"`
	ID           string `json:"id,omitempty"`
	Name         string `json:"name,omitempty"`
	SerialNumber string `json:"serial_number,omitempty"`
}

type Scope struct {
	UserID        uuid.UUID
	Username      string
	IsSuperAdmin  bool
	VisibleGroups []uuid.UUID
}

type Permission struct {
	Resource string
	Action   string
}

type SourceResult struct {
	Total int64
	Items []Item
}

type RankedSource interface {
	Name() string
	Rank() int
	Permission() Permission
	ListPrefix(ctx context.Context, scope Scope, limit int) (SourceResult, error)
	ListWindow(ctx context.Context, scope Scope, offset, limit int) (SourceResult, error)
}
