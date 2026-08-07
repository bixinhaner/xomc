package notification

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

var statusSummaryConfigID = uuid.MustParse("aaaa000c-4000-0000-0000-000000000001")

func StatusSummaryConfigID() uuid.UUID { return statusSummaryConfigID }

var ErrStatusSummaryNotReady = errors.New("Zed Mobile status summary is not ready for enablement")

type StatusSummaryConfig struct {
	ID         uuid.UUID `json:"id"`
	Enabled    bool      `json:"enabled"`
	SendTime   string    `json:"send_time"`
	TimeZone   string    `json:"time_zone"`
	Recipients []string  `json:"recipients"`
	Revision   int64     `json:"revision"`
	UpdatedBy  string    `json:"updated_by,omitempty"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type StatusSummaryConfigInput struct {
	Enabled    bool     `json:"enabled"`
	SendTime   string   `json:"send_time"`
	TimeZone   string   `json:"time_zone"`
	Recipients []string `json:"recipients"`
}

type StatusSummaryRuntimeConfig struct {
	ID         uuid.UUID
	Enabled    bool
	SendTime   string
	TimeZone   string
	Recipients []ProtectedStatusSummaryRecipient
	Revision   int64
	UpdatedAt  time.Time
}

type StatusSummaryConfigRepository interface {
	Get(context.Context, uuid.UUID) (*StatusSummaryRuntimeConfig, error)
	Update(context.Context, uuid.UUID, int64, StatusSummaryConfigInput, []ProtectedStatusSummaryRecipient, string) (*StatusSummaryRuntimeConfig, error)
}

type statusSummaryRecipientProtector interface {
	Protect(channel, address string) ([]byte, int, []byte, error)
	Unprotect(channel string, ciphertext []byte, keyVersion int) (string, error)
}

type StatusSummaryConfigService struct {
	repository StatusSummaryConfigRepository
	protector  statusSummaryRecipientProtector
	smtpReady  bool
}

func NewStatusSummaryConfigService(repository StatusSummaryConfigRepository, protector statusSummaryRecipientProtector, smtpReady bool) *StatusSummaryConfigService {
	return &StatusSummaryConfigService{repository: repository, protector: protector, smtpReady: smtpReady}
}

func (s *StatusSummaryConfigService) Get(ctx context.Context) (*StatusSummaryConfig, error) {
	record, err := s.repository.Get(ctx, statusSummaryConfigID)
	if err != nil {
		return nil, fmt.Errorf("get Zed status summary config: %w", err)
	}
	return s.toConfig(ctx, record)
}

func (s *StatusSummaryConfigService) Update(ctx context.Context, revision int64, input StatusSummaryConfigInput, actor string) (*StatusSummaryConfig, error) {
	if s == nil || s.repository == nil || s.protector == nil {
		return nil, fmt.Errorf("update Zed status summary config: dependencies are required")
	}
	addresses, err := normalizeStatusSummaryRecipients(input.Recipients)
	if err != nil {
		return nil, err
	}
	input.SendTime = strings.TrimSpace(input.SendTime)
	if _, err := time.Parse("15:04", input.SendTime); err != nil {
		return nil, fmt.Errorf("validate Zed status summary send time: %w", commonerrors.ErrInvalidInput)
	}
	input.TimeZone = strings.TrimSpace(input.TimeZone)
	if input.TimeZone == "" {
		return nil, commonerrors.ErrInvalidInput
	}
	if _, err := time.LoadLocation(input.TimeZone); err != nil {
		return nil, fmt.Errorf("validate Zed status summary time zone: %w", commonerrors.ErrInvalidInput)
	}
	if input.Enabled {
		if !s.smtpReady || len(addresses) == 0 {
			return nil, ErrStatusSummaryNotReady
		}
	}
	protected := make([]ProtectedStatusSummaryRecipient, 0, len(addresses))
	for _, address := range addresses {
		ciphertext, keyVersion, fingerprint, err := s.protector.Protect("email", address)
		if err != nil {
			return nil, fmt.Errorf("protect Zed status summary recipient: %w", err)
		}
		protected = append(protected, ProtectedStatusSummaryRecipient{Ciphertext: ciphertext, KeyVersion: keyVersion, Fingerprint: fingerprint})
	}
	record, err := s.repository.Update(ctx, statusSummaryConfigID, revision, input, protected, actor)
	if err != nil {
		return nil, fmt.Errorf("update Zed status summary config: %w", err)
	}
	return s.toConfig(ctx, record)
}

func (s *StatusSummaryConfigService) toConfig(ctx context.Context, record *StatusSummaryRuntimeConfig) (*StatusSummaryConfig, error) {
	if record == nil {
		return nil, commonerrors.ErrNotFound
	}
	addresses := make([]string, 0, len(record.Recipients))
	for _, recipient := range record.Recipients {
		address, err := s.protector.Unprotect("email", recipient.Ciphertext, recipient.KeyVersion)
		if err != nil {
			return nil, fmt.Errorf("decrypt Zed status summary recipient: %w", err)
		}
		addresses = append(addresses, address)
	}
	sort.Strings(addresses)
	return &StatusSummaryConfig{ID: record.ID, Enabled: record.Enabled, SendTime: record.SendTime, TimeZone: record.TimeZone, Recipients: addresses, Revision: record.Revision, UpdatedAt: record.UpdatedAt}, nil
}

func normalizeStatusSummaryRecipients(values []string) ([]string, error) {
	seen := make(map[string]struct{}, len(values))
	addresses := make([]string, 0, len(values))
	for _, raw := range values {
		for _, value := range strings.FieldsFunc(raw, func(r rune) bool { return r == ';' || r == ',' || r == '\n' }) {
			value = strings.ToLower(strings.TrimSpace(value))
			parsed, err := mail.ParseAddress(value)
			if err != nil || parsed.Address != value {
				return nil, commonerrors.ErrInvalidInput
			}
			if _, ok := seen[value]; ok {
				continue
			}
			seen[value] = struct{}{}
			addresses = append(addresses, value)
		}
	}
	sort.Strings(addresses)
	return addresses, nil
}
