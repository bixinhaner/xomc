package alarm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/asyncjob"
)

const (
	AlarmEmailRunPending       = "pending"
	AlarmEmailRunProcessing    = "processing"
	AlarmEmailRunSent          = "sent"
	AlarmEmailRunPartialFailed = "partial_failed"
	AlarmEmailRunFailed        = "failed"

	AlarmEmailDeliveryPending    = "pending"
	AlarmEmailDeliveryProcessing = "processing"
	AlarmEmailDeliverySent       = "sent"
	AlarmEmailDeliveryFailed     = "failed"
)

type AlarmEmailRun struct {
	ID                   uuid.UUID
	SubscriptionID       uuid.UUID
	AsyncJobID           uuid.UUID
	Window               AlarmEmailWindow
	Status               string
	SubscriptionSnapshot AlarmEmailSubscription
	RecipientsSnapshot   []string
}

type AlarmEmailDelivery struct {
	ID        uuid.UUID
	RunID     uuid.UUID
	Recipient string
	Status    string
	Attempt   int
}

type AlarmEmailRunRepository interface {
	GetRun(ctx context.Context, runID uuid.UUID) (*AlarmEmailRun, error)
	GetSubscription(ctx context.Context, subscriptionID uuid.UUID) (*AlarmEmailSubscription, error)
	GetGlobalSetting(ctx context.Context) (*AlarmEmailGlobalSetting, error)
	MarkRunProcessing(ctx context.Context, runID uuid.UUID) error
	MarkRunResult(ctx context.Context, runID uuid.UUID, status, subject, bodySummary, lastError string) error
	EnsureDeliveries(ctx context.Context, runID uuid.UUID, recipients []string) error
	ListUnsentDeliveries(ctx context.Context, runID uuid.UUID) ([]AlarmEmailDelivery, error)
	MarkDeliveryProcessing(ctx context.Context, deliveryID uuid.UUID) error
	MarkDeliverySent(ctx context.Context, deliveryID uuid.UUID, sentAt time.Time) error
	MarkDeliveryFailed(ctx context.Context, deliveryID uuid.UUID, sendError string) error
}

type AlarmEmailAlarmReader interface {
	ListForEmailWindow(ctx context.Context, subscription *AlarmEmailSubscription, window AlarmEmailWindow) ([]AlarmEmailItem, error)
}

type alarmEmailTransport interface {
	SendHTMLWithMessageID(ctx context.Context, recipients []string, subject, body, messageID string) error
}

type alarmEmailScopeAuthorizer interface {
	AuthorizeDeviceScope(ctx context.Context, userID uuid.UUID, deviceIDs, deviceGroupIDs []uuid.UUID) error
}

type alarmEmailJobPayload struct {
	RunID uuid.UUID `json:"run_id"`
}

type AlarmEmailJobRunner struct {
	repository AlarmEmailRunRepository
	alarms     AlarmEmailAlarmReader
	transport  alarmEmailTransport
	omcName    string
	location   *time.Location
	locationFn func() *time.Location
	scope      alarmEmailScopeAuthorizer
	logger     *zap.Logger
}

func (r *AlarmEmailJobRunner) SetLocationProvider(provider func() *time.Location) {
	r.locationFn = provider
}

func (r *AlarmEmailJobRunner) SetScopeAuthorizer(authorizer alarmEmailScopeAuthorizer) {
	r.scope = authorizer
}

func NewAlarmEmailJobRunner(
	repository AlarmEmailRunRepository,
	alarms AlarmEmailAlarmReader,
	transport alarmEmailTransport,
	omcName string,
	location *time.Location,
	logger *zap.Logger,
) *AlarmEmailJobRunner {
	if location == nil {
		location = time.UTC
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &AlarmEmailJobRunner{
		repository: repository,
		alarms:     alarms,
		transport:  transport,
		omcName:    omcName,
		location:   location,
		logger:     logger.Named("alarm-email-job"),
	}
}

func (r *AlarmEmailJobRunner) JobType() string { return AlarmEmailJobType }

func (r *AlarmEmailJobRunner) Run(ctx context.Context, job *asyncjob.Job) (json.RawMessage, error) {
	if r.repository == nil || r.alarms == nil || r.transport == nil {
		return nil, fmt.Errorf("alarm email runner dependencies are incomplete")
	}
	var payload alarmEmailJobPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return nil, fmt.Errorf("decode alarm email job payload: %w", err)
	}
	if payload.RunID == uuid.Nil {
		return nil, fmt.Errorf("decode alarm email job payload: run_id is required")
	}
	run, err := r.repository.GetRun(ctx, payload.RunID)
	if err != nil {
		return nil, fmt.Errorf("load alarm email run: %w", err)
	}
	currentSubscription, err := r.repository.GetSubscription(ctx, run.SubscriptionID)
	if err != nil {
		return nil, fmt.Errorf("load alarm email subscription: %w", err)
	}
	setting, err := r.repository.GetGlobalSetting(ctx)
	if err != nil {
		return nil, fmt.Errorf("load alarm email global setting: %w", err)
	}
	if !setting.Enabled {
		_ = r.repository.MarkRunResult(ctx, run.ID, AlarmEmailRunFailed, "", "", "alarm email notification is disabled")
		return nil, fmt.Errorf("alarm email notification is disabled")
	}
	if !currentSubscription.Enabled {
		_ = r.repository.MarkRunResult(ctx, run.ID, AlarmEmailRunFailed, "", "", "alarm email subscription is disabled")
		return nil, fmt.Errorf("alarm email subscription is disabled")
	}
	subscription := run.SubscriptionSnapshot
	if subscription.ID == uuid.Nil {
		// Compatibility for jobs created before immutable snapshots were added.
		subscription = *currentSubscription
	}
	if r.scope != nil {
		if subscription.CreatedBy == nil {
			err := fmt.Errorf("alarm email subscription creator is missing")
			_ = r.repository.MarkRunResult(ctx, run.ID, AlarmEmailRunFailed, "", "", err.Error())
			return nil, err
		}
		if err := r.scope.AuthorizeDeviceScope(ctx, *subscription.CreatedBy, subscription.DeviceIDs, subscription.DeviceGroupIDs); err != nil {
			_ = r.repository.MarkRunResult(ctx, run.ID, AlarmEmailRunFailed, "", "", err.Error())
			return nil, fmt.Errorf("revalidate alarm email device scope: %w", err)
		}
	}
	recipients := append([]string(nil), run.RecipientsSnapshot...)
	if len(recipients) == 0 {
		// Compatibility for jobs created before immutable snapshots were added.
		recipients, err = ResolveAlarmEmailRecipients(
			subscription.Recipients,
			setting.DefaultRecipients,
			subscription.IncludeDefaultRecipients,
		)
		if err != nil {
			_ = r.repository.MarkRunResult(ctx, run.ID, AlarmEmailRunFailed, "", "", err.Error())
			return nil, fmt.Errorf("resolve alarm email recipients: %w", err)
		}
	}
	if err := r.repository.MarkRunProcessing(ctx, run.ID); err != nil {
		return nil, fmt.Errorf("mark alarm email run processing: %w", err)
	}

	items, err := r.alarms.ListForEmailWindow(ctx, &subscription, run.Window)
	if err != nil {
		_ = r.repository.MarkRunResult(ctx, run.ID, AlarmEmailRunFailed, "", "", err.Error())
		return nil, fmt.Errorf("query alarm email window: %w", err)
	}
	if len(items) == 0 {
		if err := r.repository.MarkRunResult(ctx, run.ID, AlarmEmailRunSent, "", "no matching alarms", ""); err != nil {
			return nil, fmt.Errorf("mark empty alarm email run: %w", err)
		}
		return json.RawMessage(`{"status":"empty"}`), nil
	}

	location := r.location
	if r.locationFn != nil {
		if current := r.locationFn(); current != nil {
			location = current
		}
	}
	subject, body, err := RenderAlarmEmail(r.omcName, location, run.Window, items)
	if err != nil {
		_ = r.repository.MarkRunResult(ctx, run.ID, AlarmEmailRunFailed, "", "", err.Error())
		return nil, err
	}
	if err := r.repository.EnsureDeliveries(ctx, run.ID, recipients); err != nil {
		return nil, fmt.Errorf("ensure alarm email deliveries: %w", err)
	}
	deliveries, err := r.repository.ListUnsentDeliveries(ctx, run.ID)
	if err != nil {
		return nil, fmt.Errorf("list alarm email deliveries: %w", err)
	}

	failed := 0
	for _, delivery := range deliveries {
		if err := r.repository.MarkDeliveryProcessing(ctx, delivery.ID); err != nil {
			failed++
			continue
		}
		messageID := fmt.Sprintf("<alarm-%s@omc.local>", delivery.ID)
		if sendErr := r.transport.SendHTMLWithMessageID(ctx, []string{delivery.Recipient}, subject, body, messageID); sendErr != nil {
			failed++
			_ = r.repository.MarkDeliveryFailed(ctx, delivery.ID, sendErr.Error())
			continue
		}
		if err := r.repository.MarkDeliverySent(ctx, delivery.ID, time.Now()); err != nil {
			failed++
		}
	}

	bodySummary := fmt.Sprintf("active=%d cleared=%d recipients=%d", countActiveEmailItems(items), countClearedEmailItems(items), len(recipients))
	if failed > 0 {
		status := AlarmEmailRunPartialFailed
		if failed == len(deliveries) && len(deliveries) == len(recipients) {
			status = AlarmEmailRunFailed
		}
		lastError := fmt.Sprintf("%d recipient deliveries failed", failed)
		_ = r.repository.MarkRunResult(ctx, run.ID, status, subject, bodySummary, lastError)
		return nil, fmt.Errorf("alarm email delivery: %s", lastError)
	}
	if err := r.repository.MarkRunResult(ctx, run.ID, AlarmEmailRunSent, subject, bodySummary, ""); err != nil {
		return nil, fmt.Errorf("mark alarm email run sent: %w", err)
	}
	r.logger.Info("alarm email run sent",
		zap.String("run_id", run.ID.String()),
		zap.Int("recipients", len(recipients)),
		zap.Int("alarms", len(items)))
	return json.RawMessage(`{"status":"sent"}`), nil
}

func countActiveEmailItems(items []AlarmEmailItem) int {
	count := 0
	for _, item := range items {
		if item.ClearedAt == nil {
			count++
		}
	}
	return count
}

func countClearedEmailItems(items []AlarmEmailItem) int {
	return len(items) - countActiveEmailItems(items)
}
