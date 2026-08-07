package notification

import (
	"bytes"
	"context"
	"fmt"
	htmltemplate "html/template"
	"strings"
	texttemplate "text/template"
	"unicode/utf8"

	"github.com/google/uuid"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

var allowedNotificationTemplateVariables = map[string]struct{}{
	"alarm_name": {}, "alarm_identifier": {}, "severity": {}, "device_sn": {},
	"device_id": {}, "device_type": {}, "carrier": {}, "technology": {},
	"raised_at": {}, "occurred_at": {}, "cleared_at": {}, "probable_cause": {},
	"specific_problem": {}, "handling_suggestion": {}, "omc_url": {}, "status": {},
	"event_count": {}, "window_started_at": {}, "window_ends_at": {},
}

type TemplateManagementService struct{ repository TemplateManagementRepository }

func NewTemplateManagementService(repository TemplateManagementRepository) *TemplateManagementService {
	return &TemplateManagementService{repository: repository}
}

func (s *TemplateManagementService) List(ctx context.Context) ([]ManagedTemplate, error) {
	items, err := s.repository.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list versioned notification templates: %w", err)
	}
	return items, nil
}

func (s *TemplateManagementService) Get(ctx context.Context, id uuid.UUID) (*ManagedTemplate, error) {
	item, err := s.repository.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get versioned notification template: %w", err)
	}
	return item, nil
}

func (s *TemplateManagementService) Create(ctx context.Context, input ManagedTemplateInput, actor string) (*ManagedTemplate, error) {
	input, err := normalizeManagedTemplateInput(input)
	if err != nil {
		return nil, err
	}
	item, err := s.repository.Create(ctx, input, normalizeActor(actor))
	if err != nil {
		return nil, fmt.Errorf("create versioned notification template: %w", err)
	}
	return item, nil
}

func (s *TemplateManagementService) UpdateDraft(ctx context.Context, id uuid.UUID, revision int64, input ManagedTemplateInput, actor string) (*ManagedTemplate, error) {
	input, err := normalizeManagedTemplateInput(input)
	if err != nil {
		return nil, err
	}
	item, err := s.repository.UpdateDraft(ctx, id, revision, input, normalizeActor(actor))
	if err != nil {
		return nil, fmt.Errorf("update versioned notification template draft: %w", err)
	}
	return item, nil
}

func (s *TemplateManagementService) Publish(ctx context.Context, id uuid.UUID, revision int64, actor string) (*ManagedTemplate, error) {
	item, err := s.repository.Publish(ctx, id, revision, normalizeActor(actor))
	if err != nil {
		return nil, fmt.Errorf("publish notification template version: %w", err)
	}
	return item, nil
}

func (s *TemplateManagementService) Preview(input ManagedTemplateInput, values map[string]string) (ManagedTemplatePreview, error) {
	input, err := normalizeManagedTemplateInput(input)
	if err != nil {
		return ManagedTemplatePreview{}, err
	}
	subject, err := renderStrictText("subject", input.Subject, values)
	if err != nil {
		return ManagedTemplatePreview{}, fmt.Errorf("render notification template subject: %w", err)
	}
	body, err := renderStrictText("text_body", input.TextBody, values)
	if err != nil {
		return ManagedTemplatePreview{}, fmt.Errorf("render notification template text body: %w", err)
	}
	preview := ManagedTemplatePreview{Subject: subject, TextBody: body}
	if input.HTMLBody != nil {
		htmlBody, renderErr := renderStrictHTML("html_body", *input.HTMLBody, values)
		if renderErr != nil {
			return ManagedTemplatePreview{}, fmt.Errorf("render notification template HTML body: %w", renderErr)
		}
		preview.HTMLBody = &htmlBody
	}
	if input.Channel == TemplateChannelSMS {
		preview.SMSSegments = estimateSMSSegments(body)
	}
	return preview, nil
}

func normalizeManagedTemplateInput(input ManagedTemplateInput) (ManagedTemplateInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Channel = strings.TrimSpace(input.Channel)
	input.Language = defaultLanguage(strings.TrimSpace(input.Language))
	input.ChangeReason = strings.TrimSpace(input.ChangeReason)
	input.Variables = normalizeVariables(input.Variables)
	if input.Name == "" || len(input.Name) > 128 || input.TextBody == "" {
		return input, commonerrors.ErrInvalidInput
	}
	if input.Channel != TemplateChannelEmail && input.Channel != TemplateChannelSMS {
		return input, commonerrors.ErrInvalidInput
	}
	if input.Channel == TemplateChannelEmail && strings.TrimSpace(input.Subject) == "" {
		return input, commonerrors.ErrInvalidInput
	}
	if input.Channel == TemplateChannelSMS && (input.Subject != "" || input.HTMLBody != nil) {
		return input, commonerrors.ErrInvalidInput
	}
	values := make(map[string]string, len(input.Variables))
	for _, variable := range input.Variables {
		if _, allowed := allowedNotificationTemplateVariables[variable]; !allowed {
			return input, commonerrors.ErrInvalidInput
		}
		values[variable] = "sample"
	}
	if _, err := renderStrictText("subject", input.Subject, values); err != nil {
		return input, commonerrors.ErrInvalidInput
	}
	if _, err := renderStrictText("text_body", input.TextBody, values); err != nil {
		return input, commonerrors.ErrInvalidInput
	}
	if input.HTMLBody != nil {
		if _, err := renderStrictHTML("html_body", *input.HTMLBody, values); err != nil {
			return input, commonerrors.ErrInvalidInput
		}
	}
	return input, nil
}

func renderStrictText(name, source string, values map[string]string) (string, error) {
	tpl, err := texttemplate.New(name).Option("missingkey=error").Parse(source)
	if err != nil {
		return "", err
	}
	var output bytes.Buffer
	if err := tpl.Execute(&output, values); err != nil {
		return "", err
	}
	return output.String(), nil
}

func renderStrictHTML(name, source string, values map[string]string) (string, error) {
	tpl, err := htmltemplate.New(name).Option("missingkey=error").Parse(source)
	if err != nil {
		return "", err
	}
	var output bytes.Buffer
	if err := tpl.Execute(&output, values); err != nil {
		return "", err
	}
	return output.String(), nil
}

func estimateSMSSegments(body string) int {
	if body == "" {
		return 0
	}
	ascii := true
	for _, value := range body {
		if value > 127 {
			ascii = false
			break
		}
	}
	length, single, multipart := utf8.RuneCountInString(body), 70, 67
	if ascii {
		single, multipart = 160, 153
	}
	if length <= single {
		return 1
	}
	return (length + multipart - 1) / multipart
}
