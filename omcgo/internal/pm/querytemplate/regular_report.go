package querytemplate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

type RegularReportPeriod string

const (
	RegularReport15Min RegularReportPeriod = "15min"
	RegularReportHour  RegularReportPeriod = "hour"
	RegularReportDay   RegularReportPeriod = "day"
)

var (
	ErrReportRevisionMismatch      = errors.New("querytemplate: regular report revision mismatch")
	ErrRegularReportRunnerNotReady = errors.New("querytemplate: regular report delivery runner is not available")
)

type RegularReport struct {
	TemplateID uuid.UUID           `json:"template_id"`
	Enabled    bool                `json:"enabled"`
	SendTime   string              `json:"send_time"`
	Period     RegularReportPeriod `json:"period"`
	Recipients []string            `json:"recipients"`
	Revision   int64               `json:"revision"`
	NextRunAt  *time.Time          `json:"next_run_at,omitempty"`
}

type RegularReportInput struct {
	Enabled    bool                `json:"enabled"`
	SendTime   string              `json:"send_time"`
	Period     RegularReportPeriod `json:"period"`
	Recipients []string            `json:"recipients"`
}

func BuildRegularReportExportParams(payload []byte, period RegularReportPeriod, start, end time.Time) ([]byte, error) {
	params := make(map[string]any)
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &params); err != nil {
			return nil, fmt.Errorf("decode KPI query template payload: %w", err)
		}
	}
	params["start_time"] = start.UTC().Format(time.RFC3339)
	params["end_time"] = end.UTC().Format(time.RFC3339)
	if _, ok := params["granularity"]; !ok {
		switch period {
		case RegularReport15Min:
			params["granularity"] = "15min"
		case RegularReportHour:
			params["granularity"] = "hourly"
		case RegularReportDay:
			params["granularity"] = "daily"
		default:
			return nil, fmt.Errorf("unsupported KPI regular report period %q", period)
		}
	}
	return json.Marshal(params)
}

type ProtectedReportRecipient struct {
	Ciphertext  []byte
	KeyVersion  int
	Fingerprint []byte
}

type ClaimedRegularReport struct {
	RunID        uuid.UUID
	TemplateID   uuid.UUID
	TemplateName string
	Payload      []byte
	Period       RegularReportPeriod
	Recipients   []ProtectedReportRecipient
	WindowStart  time.Time
	WindowEnd    time.Time
}

type RegularReportRunState string

const (
	RegularReportRunExporting RegularReportRunState = "exporting"
	RegularReportRunSending   RegularReportRunState = "sending"
	RegularReportRunRetryWait RegularReportRunState = "retry_wait"
	RegularReportRunSucceeded RegularReportRunState = "succeeded"
	RegularReportRunFailed    RegularReportRunState = "failed"
)

type RegularReportRun struct {
	ID           uuid.UUID
	TemplateID   uuid.UUID
	WindowStart  time.Time
	WindowEnd    time.Time
	Payload      []byte
	ExportTaskID *uuid.UUID
	State        RegularReportRunState
}

type RegularReportRunRecipient struct {
	ID          uuid.UUID
	Ciphertext  []byte
	KeyVersion  int
	Fingerprint []byte
	State       string
}

type regularReportRecord struct {
	TemplateID uuid.UUID
	Enabled    bool
	SendTime   string
	Period     RegularReportPeriod
	Revision   int64
	NextRunAt  *time.Time
	Recipients []ProtectedReportRecipient
}

type RegularReportRepository interface {
	GetRegularReport(context.Context, uuid.UUID) (*regularReportRecord, error)
	UpdateRegularReport(context.Context, uuid.UUID, int64, RegularReportInput, []ProtectedReportRecipient) (*regularReportRecord, error)
	ClaimDueRegularReports(context.Context, time.Time, *time.Location, int) ([]ClaimedRegularReport, error)
	ClaimRegularReportRuns(context.Context, time.Time, int) ([]RegularReportRun, error)
	SetRegularReportExportTask(context.Context, uuid.UUID, uuid.UUID) error
	MarkRegularReportRunSending(context.Context, uuid.UUID) error
	ListRegularReportRunRecipients(context.Context, uuid.UUID) ([]RegularReportRunRecipient, error)
	ClaimRegularReportRecipient(context.Context, uuid.UUID, uuid.UUID) (bool, error)
	RequeueStaleRegularReportRecipients(context.Context, uuid.UUID, time.Time) error
	MarkRegularReportRecipientSucceeded(context.Context, uuid.UUID) error
	MarkRegularReportRecipientFailed(context.Context, uuid.UUID, string) error
	MarkRegularReportRunRetry(context.Context, uuid.UUID, string, time.Time) error
	MarkRegularReportRunSucceeded(context.Context, uuid.UUID) error
	ClearRegularReportExportTask(context.Context, uuid.UUID) error
}

type regularReportProtector interface {
	Protect(channel, address string) ([]byte, int, []byte, error)
	Unprotect(channel string, ciphertext []byte, keyVersion int) (string, error)
}

type RegularReportService struct {
	repository RegularReportRepository
	protector  regularReportProtector
}

func NewRegularReportService(repository RegularReportRepository, protector regularReportProtector) *RegularReportService {
	return &RegularReportService{repository: repository, protector: protector}
}

func (s *RegularReportService) Get(ctx context.Context, templateID uuid.UUID) (*RegularReport, error) {
	record, err := s.repository.GetRegularReport(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("get KPI regular report: %w", err)
	}
	return s.toReport(record)
}

func (s *RegularReportService) Update(ctx context.Context, templateID uuid.UUID, revision int64, input RegularReportInput) (*RegularReport, error) {
	if input.Enabled && s.protector == nil {
		return nil, ErrRegularReportRunnerNotReady
	}
	input.SendTime = strings.TrimSpace(input.SendTime)
	if _, err := time.Parse("15:04", input.SendTime); err != nil || !validRegularReportPeriod(input.Period) {
		return nil, fmt.Errorf("validate KPI regular report: invalid send time or period")
	}
	addresses, err := normalizeReportAddresses(input.Recipients)
	if err != nil || (input.Enabled && len(addresses) == 0) {
		return nil, fmt.Errorf("validate KPI regular report recipients")
	}
	protected := make([]ProtectedReportRecipient, 0, len(addresses))
	for _, address := range addresses {
		ciphertext, keyVersion, fingerprint, err := s.protector.Protect("email", address)
		if err != nil {
			return nil, fmt.Errorf("protect KPI report recipient: %w", err)
		}
		protected = append(protected, ProtectedReportRecipient{Ciphertext: ciphertext, KeyVersion: keyVersion, Fingerprint: fingerprint})
	}
	input.Recipients = addresses
	record, err := s.repository.UpdateRegularReport(ctx, templateID, revision, input, protected)
	if err != nil {
		return nil, fmt.Errorf("update KPI regular report: %w", err)
	}
	return s.toReport(record)
}

func (s *RegularReportService) toReport(record *regularReportRecord) (*RegularReport, error) {
	addresses := make([]string, 0, len(record.Recipients))
	for _, recipient := range record.Recipients {
		address, err := s.protector.Unprotect("email", recipient.Ciphertext, recipient.KeyVersion)
		if err != nil {
			return nil, fmt.Errorf("decrypt KPI report recipient: %w", err)
		}
		addresses = append(addresses, address)
	}
	sort.Strings(addresses)
	return &RegularReport{
		TemplateID: record.TemplateID, Enabled: record.Enabled, SendTime: record.SendTime,
		Period: record.Period, Recipients: addresses, Revision: record.Revision, NextRunAt: record.NextRunAt,
	}, nil
}

func validRegularReportPeriod(period RegularReportPeriod) bool {
	return period == RegularReport15Min || period == RegularReportHour || period == RegularReportDay
}

func normalizeReportAddresses(values []string) ([]string, error) {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, raw := range values {
		for _, value := range strings.FieldsFunc(raw, func(r rune) bool { return r == ';' || r == ',' || r == '\n' }) {
			value = strings.ToLower(strings.TrimSpace(value))
			parsed, err := mail.ParseAddress(value)
			if err != nil || parsed.Address != value {
				return nil, fmt.Errorf("invalid email address")
			}
			if _, ok := seen[value]; ok {
				continue
			}
			seen[value] = struct{}{}
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result, nil
}
