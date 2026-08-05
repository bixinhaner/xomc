package notification

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

type templateManagementRepositoryStub struct {
	item  *ManagedTemplate
	input ManagedTemplateInput
}

func (s *templateManagementRepositoryStub) List(context.Context) ([]ManagedTemplate, error) {
	return []ManagedTemplate{*s.item}, nil
}
func (s *templateManagementRepositoryStub) Get(context.Context, uuid.UUID) (*ManagedTemplate, error) {
	return s.item, nil
}
func (s *templateManagementRepositoryStub) Create(_ context.Context, input ManagedTemplateInput, _ string) (*ManagedTemplate, error) {
	s.input = input
	return s.item, nil
}
func (s *templateManagementRepositoryStub) UpdateDraft(_ context.Context, _ uuid.UUID, _ int64, input ManagedTemplateInput, _ string) (*ManagedTemplate, error) {
	s.input = input
	return s.item, nil
}
func (s *templateManagementRepositoryStub) Publish(context.Context, uuid.UUID, int64, string) (*ManagedTemplate, error) {
	return s.item, nil
}

func TestTemplateManagementService_RejectsUnknownOrMissingVariables(t *testing.T) {
	service := NewTemplateManagementService(&templateManagementRepositoryStub{item: &ManagedTemplate{}})
	_, err := service.Create(context.Background(), ManagedTemplateInput{
		Name: "alarm", Channel: "email", Subject: "{{.password}}", TextBody: "body", Variables: []string{"password"},
	}, "operator")
	require.ErrorIs(t, err, commonerrors.ErrInvalidInput)
	_, err = service.Create(context.Background(), ManagedTemplateInput{
		Name: "alarm", Channel: "email", Subject: "{{.device_sn}}", TextBody: "body",
	}, "operator")
	require.ErrorIs(t, err, commonerrors.ErrInvalidInput)
}

func TestTemplateManagementService_PreviewIsStrictAndReportsSMSSegments(t *testing.T) {
	service := NewTemplateManagementService(&templateManagementRepositoryStub{item: &ManagedTemplate{}})
	preview, err := service.Preview(ManagedTemplateInput{
		Name: "sms", Channel: "sms", TextBody: "基站{{.device_sn}}" + strings.Repeat("告", 70), Variables: []string{"device_sn"},
	}, map[string]string{"device_sn": "SN001"})
	require.NoError(t, err)
	require.GreaterOrEqual(t, preview.SMSSegments, 2)
	_, err = service.Preview(ManagedTemplateInput{
		Name: "sms", Channel: "sms", TextBody: "{{.device_sn}}", Variables: []string{"device_sn"},
	}, nil)
	require.Error(t, err)
}

func TestTemplateManagementService_SMSRejectsSubjectAndHTML(t *testing.T) {
	service := NewTemplateManagementService(&templateManagementRepositoryStub{item: &ManagedTemplate{}})
	html := "<p>body</p>"
	_, err := service.Create(context.Background(), ManagedTemplateInput{
		Name: "sms", Channel: "sms", Subject: "subject", TextBody: "body", HTMLBody: &html,
	}, "operator")
	require.ErrorIs(t, err, commonerrors.ErrInvalidInput)
}
