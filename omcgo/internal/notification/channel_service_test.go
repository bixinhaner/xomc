package notification

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

type channelRepositoryStub struct{ item *ChannelConfig }

func (s *channelRepositoryStub) List(context.Context) ([]ChannelConfig, error) {
	return []ChannelConfig{*s.item}, nil
}
func (s *channelRepositoryStub) Get(context.Context, uuid.UUID) (*ChannelConfig, error) {
	return s.item, nil
}
func (s *channelRepositoryStub) Update(_ context.Context, _ uuid.UUID, _ int64, input ChannelConfigUpdate, _ string) (*ChannelConfig, error) {
	s.item.Name, s.item.Enabled, s.item.Parameters = input.Name, input.Enabled, input.Parameters
	s.item.SecretConfigured = input.SecretRef != nil && *input.SecretRef != ""
	return s.item, nil
}
func (s *channelRepositoryStub) GetHealth(context.Context, uuid.UUID) (*ChannelHealth, error) {
	return &ChannelHealth{}, nil
}

func TestChannelService_RejectsSecretsInParametersAndSMSEnable(t *testing.T) {
	email := &ChannelConfig{ID: uuid.New(), Channel: "email"}
	service := NewChannelService(&channelRepositoryStub{item: email}, nil)
	_, err := service.Update(context.Background(), email.ID, 1, ChannelConfigUpdate{
		Name: "email", Parameters: json.RawMessage(`{"host":"smtp.local","password":"plain"}`),
	}, "operator")
	require.ErrorIs(t, err, commonerrors.ErrInvalidInput)
	_, err = service.Update(context.Background(), email.ID, 1, ChannelConfigUpdate{
		Name: "email", Parameters: json.RawMessage(`{"auth":{"smtp_password":"plain"}}`),
	}, "operator")
	require.ErrorIs(t, err, commonerrors.ErrInvalidInput)
	sms := &ChannelConfig{ID: uuid.New(), Channel: "sms_kafka"}
	service = NewChannelService(&channelRepositoryStub{item: sms}, nil)
	_, err = service.Update(context.Background(), sms.ID, 1, ChannelConfigUpdate{Name: "sms", Enabled: true}, "operator")
	require.ErrorIs(t, err, commonerrors.ErrInvalidInput)
}

func TestChannelConfig_JSONNeverContainsSecretReference(t *testing.T) {
	config := ChannelConfig{ID: uuid.New(), Channel: "email", SecretConfigured: true}
	payload, err := json.Marshal(config)
	require.NoError(t, err)
	require.NotContains(t, string(payload), "secret_ref")
	require.NotContains(t, string(payload), "vault/")
}

func TestChannelService_VerifyDoesNotPretendWithoutAdapter(t *testing.T) {
	service := NewChannelService(&channelRepositoryStub{item: &ChannelConfig{}}, nil)
	require.ErrorIs(t, service.Verify(context.Background(), uuid.New()), ErrChannelVerificationUnavailable)
}
