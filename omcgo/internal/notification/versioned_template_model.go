package notification

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type ManagedTemplate struct {
	ID                        uuid.UUID               `json:"id"`
	Name                      string                  `json:"name"`
	Channel                   string                  `json:"channel"`
	Language                  string                  `json:"language"`
	Revision                  int64                   `json:"revision"`
	CurrentDraftVersionID     *uuid.UUID              `json:"current_draft_version_id,omitempty"`
	CurrentPublishedVersionID *uuid.UUID              `json:"current_published_version_id,omitempty"`
	Archived                  bool                    `json:"archived"`
	CreatedBy                 string                  `json:"created_by"`
	CreatedAt                 time.Time               `json:"created_at"`
	UpdatedAt                 time.Time               `json:"updated_at"`
	Draft                     *ManagedTemplateVersion `json:"draft,omitempty"`
	Published                 *ManagedTemplateVersion `json:"published,omitempty"`
}

type ManagedTemplateVersion struct {
	ID           uuid.UUID  `json:"id"`
	TemplateID   uuid.UUID  `json:"template_id"`
	VersionNo    int64      `json:"version_no"`
	Channel      string     `json:"channel"`
	Language     string     `json:"language"`
	Subject      string     `json:"subject"`
	TextBody     string     `json:"text_body"`
	HTMLBody     *string    `json:"html_body,omitempty"`
	Variables    []string   `json:"variables"`
	CreatedBy    string     `json:"created_by"`
	ChangeReason string     `json:"change_reason"`
	CreatedAt    time.Time  `json:"created_at"`
	PublishedAt  *time.Time `json:"published_at,omitempty"`
}

type ManagedTemplateInput struct {
	Name         string   `json:"name"`
	Channel      string   `json:"channel"`
	Language     string   `json:"language"`
	Subject      string   `json:"subject"`
	TextBody     string   `json:"text_body"`
	HTMLBody     *string  `json:"html_body,omitempty"`
	Variables    []string `json:"variables"`
	ChangeReason string   `json:"change_reason"`
}

type ManagedTemplatePreview struct {
	Subject     string  `json:"subject"`
	TextBody    string  `json:"text_body"`
	HTMLBody    *string `json:"html_body,omitempty"`
	SMSSegments int     `json:"sms_segments,omitempty"`
}

type TemplateManagementRepository interface {
	List(context.Context) ([]ManagedTemplate, error)
	Get(context.Context, uuid.UUID) (*ManagedTemplate, error)
	Create(context.Context, ManagedTemplateInput, string) (*ManagedTemplate, error)
	UpdateDraft(context.Context, uuid.UUID, int64, ManagedTemplateInput, string) (*ManagedTemplate, error)
	Publish(context.Context, uuid.UUID, int64, string) (*ManagedTemplate, error)
}
