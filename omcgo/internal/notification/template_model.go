package notification

import (
	"time"

	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/core/model"
)

// Template channel constants.
const (
	TemplateChannelEmail   = "email"
	TemplateChannelSMS     = "sms"
	TemplateChannelWebhook = "webhook"
)

// Template language constants (extend as needed).
const (
	TemplateLanguageZhCN = "zh-CN"
	TemplateLanguageEnUS = "en-US"
)

// NotificationTemplate represents a reusable notification template (email / sms / webhook).
// The Body field uses Go {{var}} placeholders that are rendered against a variable map
// at dispatch time. Variables lists the names of variables expected by the template, used
// by the UI to validate user input.
type NotificationTemplate struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Channel   string    `json:"channel" db:"channel"`
	Language  string    `json:"language" db:"language"`
	Subject   string    `json:"subject" db:"subject"`
	Body      string    `json:"body" db:"body"`
	Variables []string  `json:"variables" db:"variables"`
	Enabled   bool      `json:"enabled" db:"enabled"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// NotificationTemplateFilter describes list-query parameters for templates.
type NotificationTemplateFilter struct {
	Channel  *string
	Language *string
	Enabled  *bool
	model.ListRequest
}

// CreateTemplateRequest is the JSON body for POST /notifications/templates.
type CreateTemplateRequest struct {
	Name      string   `json:"name" binding:"required,max=128"`
	Channel   string   `json:"channel" binding:"required,oneof=email sms webhook"`
	Language  string   `json:"language" binding:"omitempty,max=16"`
	Subject   string   `json:"subject" binding:"required"`
	Body      string   `json:"body" binding:"required"`
	Variables []string `json:"variables"`
	Enabled   *bool    `json:"enabled"`
}

// UpdateTemplateRequest is the JSON body for PUT /notifications/templates/:id.
// All fields optional; nil means "leave unchanged".
type UpdateTemplateRequest struct {
	Name      *string  `json:"name" binding:"omitempty,max=128"`
	Channel   *string  `json:"channel" binding:"omitempty,oneof=email sms webhook"`
	Language  *string  `json:"language" binding:"omitempty,max=16"`
	Subject   *string  `json:"subject"`
	Body      *string  `json:"body"`
	Variables []string `json:"variables"`
	Enabled   *bool    `json:"enabled"`
}
