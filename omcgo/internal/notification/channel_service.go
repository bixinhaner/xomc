package notification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

var ErrChannelVerificationUnavailable = errors.New("notification channel verification adapter is unavailable")

type ChannelConfig struct {
	ID               uuid.UUID       `json:"id"`
	Channel          string          `json:"channel"`
	Name             string          `json:"name"`
	Enabled          bool            `json:"enabled"`
	Parameters       json.RawMessage `json:"parameters"`
	SecretConfigured bool            `json:"secret_configured"`
	Revision         int64           `json:"revision"`
	CreatedBy        string          `json:"created_by"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type ChannelConfigUpdate struct {
	Name       string          `json:"name"`
	Enabled    bool            `json:"enabled"`
	Parameters json.RawMessage `json:"parameters"`
	SecretRef  *string         `json:"secret_ref,omitempty"`
}

type ChannelHealth struct {
	ChannelConfigID      uuid.UUID  `json:"channel_config_id"`
	CircuitState         string     `json:"circuit_state"`
	ConsecutiveSuccesses int        `json:"consecutive_successes"`
	ConsecutiveFailures  int        `json:"consecutive_failures"`
	LastSuccessAt        *time.Time `json:"last_success_at,omitempty"`
	LastFailureAt        *time.Time `json:"last_failure_at,omitempty"`
	LastVerifiedAt       *time.Time `json:"last_verified_at,omitempty"`
	LastErrorCategory    *string    `json:"last_error_category,omitempty"`
	LastErrorSummary     *string    `json:"last_error_summary,omitempty"`
	CircuitOpenedAt      *time.Time `json:"circuit_opened_at,omitempty"`
	NextProbeAt          *time.Time `json:"next_probe_at,omitempty"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

type ChannelConfigRepository interface {
	List(context.Context) ([]ChannelConfig, error)
	Get(context.Context, uuid.UUID) (*ChannelConfig, error)
	Update(context.Context, uuid.UUID, int64, ChannelConfigUpdate, string) (*ChannelConfig, error)
	GetHealth(context.Context, uuid.UUID) (*ChannelHealth, error)
}

type ChannelVerifier interface {
	Verify(context.Context, ChannelConfig) error
}

type ChannelService struct {
	repository ChannelConfigRepository
	verifier   ChannelVerifier
}

func NewChannelService(repository ChannelConfigRepository, verifier ChannelVerifier) *ChannelService {
	return &ChannelService{repository: repository, verifier: verifier}
}

func (s *ChannelService) List(ctx context.Context) ([]ChannelConfig, error) {
	items, err := s.repository.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list notification channels: %w", err)
	}
	return items, nil
}

func (s *ChannelService) Update(ctx context.Context, id uuid.UUID, revision int64, input ChannelConfigUpdate, actor string) (*ChannelConfig, error) {
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get notification channel for update: %w", err)
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" || len(input.Name) > 128 {
		return nil, commonerrors.ErrInvalidInput
	}
	if input.Enabled && current.Channel != TemplateChannelEmail {
		return nil, commonerrors.ErrInvalidInput
	}
	if len(input.Parameters) == 0 {
		input.Parameters = json.RawMessage(`{}`)
	}
	if err := validateNonSensitiveChannelParameters(input.Parameters); err != nil {
		return nil, err
	}
	if input.SecretRef != nil {
		trimmed := strings.TrimSpace(*input.SecretRef)
		input.SecretRef = &trimmed
	}
	updated, err := s.repository.Update(ctx, id, revision, input, normalizeActor(actor))
	if err != nil {
		return nil, fmt.Errorf("update notification channel: %w", err)
	}
	return updated, nil
}

func (s *ChannelService) Verify(ctx context.Context, id uuid.UUID) error {
	if s.verifier == nil {
		return ErrChannelVerificationUnavailable
	}
	config, err := s.repository.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("get notification channel for verification: %w", err)
	}
	if err := s.verifier.Verify(ctx, *config); err != nil {
		return fmt.Errorf("verify notification channel: %w", err)
	}
	return nil
}

func (s *ChannelService) Health(ctx context.Context, id uuid.UUID) (*ChannelHealth, error) {
	health, err := s.repository.GetHealth(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get notification channel health: %w", err)
	}
	return health, nil
}

func validateNonSensitiveChannelParameters(parameters json.RawMessage) error {
	var object map[string]any
	if !json.Valid(parameters) || json.Unmarshal(parameters, &object) != nil || object == nil {
		return commonerrors.ErrInvalidInput
	}
	for key := range object {
		if channelParameterContainsSecret(key, object[key]) {
			return commonerrors.ErrInvalidInput
		}
	}
	return nil
}

func channelParameterContainsSecret(key string, value any) bool {
	normalized := strings.ToLower(strings.ReplaceAll(key, "-", "_"))
	for _, forbidden := range []string{"password", "token", "secret", "private_key", "api_key", "access_key", "credential"} {
		if normalized == forbidden || strings.HasSuffix(normalized, "_"+forbidden) {
			return true
		}
	}
	switch nested := value.(type) {
	case map[string]any:
		for nestedKey, nestedValue := range nested {
			if channelParameterContainsSecret(nestedKey, nestedValue) {
				return true
			}
		}
	case []any:
		for _, nestedValue := range nested {
			if object, ok := nestedValue.(map[string]any); ok {
				for nestedKey, objectValue := range object {
					if channelParameterContainsSecret(nestedKey, objectValue) {
						return true
					}
				}
			}
		}
	}
	return false
}
