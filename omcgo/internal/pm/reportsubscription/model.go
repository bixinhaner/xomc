package reportsubscription

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

type Period string

const (
	Period15Min  Period = "15min"
	PeriodHourly Period = "hourly"
	PeriodDaily  Period = "daily"
)

type Subscription struct {
	ID              uuid.UUID  `json:"id"`
	QueryTemplateID uuid.UUID  `json:"query_template_id"`
	Enabled         bool       `json:"enabled"`
	Period          Period     `json:"period"`
	SendTimes       []string   `json:"send_times"`
	Recipients      []string   `json:"recipients"`
	TimezoneName    string     `json:"timezone_name"`
	NextRunAt       *time.Time `json:"next_run_at,omitempty"`
	LastRunAt       *time.Time `json:"last_run_at,omitempty"`
	LastStatus      *string    `json:"last_status,omitempty"`
	LastError       *string    `json:"last_error,omitempty"`
	CreatedBy       uuid.UUID  `json:"created_by"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// SubscriptionView is the HTTP representation. Recipients and the last
// internal error are present only for callers allowed to manage the template.
type SubscriptionView struct {
	ID              uuid.UUID  `json:"id"`
	QueryTemplateID uuid.UUID  `json:"query_template_id"`
	Enabled         bool       `json:"enabled"`
	Period          Period     `json:"period"`
	SendTimes       []string   `json:"send_times"`
	Recipients      []string   `json:"recipients,omitempty"`
	TimezoneName    string     `json:"timezone_name"`
	NextRunAt       *time.Time `json:"next_run_at,omitempty"`
	LastRunAt       *time.Time `json:"last_run_at,omitempty"`
	LastStatus      *string    `json:"last_status,omitempty"`
	LastError       *string    `json:"last_error,omitempty"`
}

type UpsertInput struct {
	Enabled      bool
	Period       Period
	SendTimes    []string
	Recipients   []string
	TimezoneName string
	CreatedBy    uuid.UUID
	NextRunAt    *time.Time
}

type Run struct {
	ID                  uuid.UUID       `json:"id"`
	SubscriptionID      *uuid.UUID      `json:"subscription_id,omitempty"`
	QueryTemplateID     uuid.UUID       `json:"query_template_id"`
	QueryTemplateName   string          `json:"query_template_name"`
	QueryPayload        json.RawMessage `json:"query_payload"`
	Period              Period          `json:"period"`
	Recipients          []string        `json:"recipients"`
	WindowStart         time.Time       `json:"window_start"`
	WindowEnd           time.Time       `json:"window_end"`
	JobID               *uuid.UUID      `json:"job_id,omitempty"`
	ExportTaskID        *uuid.UUID      `json:"export_task_id,omitempty"`
	NotificationHistory *uuid.UUID      `json:"notification_history_id,omitempty"`
	Status              string          `json:"status"`
	AttachmentName      *string         `json:"attachment_name,omitempty"`
	ErrorMessage        *string         `json:"error_message,omitempty"`
	StartedAt           *time.Time      `json:"started_at,omitempty"`
	FinishedAt          *time.Time      `json:"finished_at,omitempty"`
	CreatedAt           time.Time       `json:"created_at"`
}

// RunView deliberately excludes the query snapshot and internal job/history
// identifiers. Recipients and detailed errors are limited to template writers.
type RunView struct {
	ID                uuid.UUID  `json:"id"`
	QueryTemplateID   uuid.UUID  `json:"query_template_id"`
	QueryTemplateName string     `json:"query_template_name"`
	Period            Period     `json:"period"`
	Recipients        []string   `json:"recipients,omitempty"`
	WindowStart       time.Time  `json:"window_start"`
	WindowEnd         time.Time  `json:"window_end"`
	Status            string     `json:"status"`
	AttachmentName    *string    `json:"attachment_name,omitempty"`
	ErrorMessage      *string    `json:"error_message,omitempty"`
	StartedAt         *time.Time `json:"started_at,omitempty"`
	FinishedAt        *time.Time `json:"finished_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

const (
	RunStatusPending        = "pending"
	RunStatusRunning        = "running"
	RunStatusSent           = "sent"
	RunStatusExportFailed   = "export_failed"
	RunStatusDeliveryFailed = "delivery_failed"
)

var (
	ErrNotFound  = errors.New("report subscription: not found")
	ErrForbidden = errors.New("report subscription: forbidden")
	ErrInvalid   = errors.New("report subscription: invalid input")
)

func ValidPeriod(period Period) bool {
	return period == Period15Min || period == PeriodHourly || period == PeriodDaily
}
