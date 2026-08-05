package notification

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
)

const (
	emailWorkerLease       = time.Minute
	emailWorkerBatchSize   = 20
	emailWorkerMaxAttempts = 5
)

type EmailDeliveryClaimRequest struct {
	WorkerID      string
	Now           time.Time
	LeaseDuration time.Duration
	Limit         int
}

type EmailDeliveryContent struct {
	Template              DomainTemplateVersion
	Payload               event.AlarmLifecyclePayload
	DigestEventCount      int
	DigestWindowStartedAt *time.Time
	DigestWindowEndsAt    *time.Time
}

type EmailAttemptCompletion struct {
	DeliveryID     uuid.UUID
	AttemptID      uuid.UUID
	WorkerID       string
	FinishedAt     time.Time
	AttemptResult  string
	ErrorCategory  *string
	StatusSummary  *string
	NextRetryAt    *time.Time
	FlowState      string
	DeliveryResult string
	FailureReason  *string
}

type EmailWorkerRepository interface {
	ClaimEmailDeliveries(context.Context, EmailDeliveryClaimRequest) ([]DomainDelivery, error)
	AuthorizeSend(context.Context, uuid.UUID, time.Time) (AuthorizedDelivery, error)
	StartEmailAttempt(context.Context, uuid.UUID, string, time.Time) (DomainDeliveryAttempt, error)
	LoadEmailDeliveryContent(context.Context, AuthorizedDelivery) (EmailDeliveryContent, error)
	FinishEmailAttempt(context.Context, EmailAttemptCompletion) error
}

type RecipientUnprotector interface {
	Unprotect(channel string, ciphertext []byte, keyVersion int) (string, error)
}

type SingleEmailTransport interface {
	SendOne(context.Context, string, string, string) error
}

type EmailWorker struct {
	repository  EmailWorkerRepository
	protector   RecipientUnprotector
	transport   SingleEmailTransport
	workerID    string
	now         func() time.Time
	logger      *zap.Logger
	maxAttempts int
}

func NewEmailWorker(
	repository EmailWorkerRepository,
	protector RecipientUnprotector,
	transport SingleEmailTransport,
	workerID string,
	logger *zap.Logger,
) *EmailWorker {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &EmailWorker{
		repository: repository, protector: protector, transport: transport, workerID: workerID,
		now: func() time.Time { return time.Now().UTC() }, logger: logger.Named("email-worker"),
		maxAttempts: emailWorkerMaxAttempts,
	}
}

func (w *EmailWorker) RunOnce(ctx context.Context) (int, error) {
	if w == nil || w.repository == nil || w.protector == nil || w.transport == nil {
		return 0, fmt.Errorf("run notification email worker: dependencies are required")
	}
	if w.workerID == "" {
		return 0, fmt.Errorf("run notification email worker: worker ID is required")
	}
	now := w.now()
	deliveries, err := w.repository.ClaimEmailDeliveries(ctx, EmailDeliveryClaimRequest{
		WorkerID: w.workerID, Now: now, LeaseDuration: emailWorkerLease, Limit: emailWorkerBatchSize,
	})
	if err != nil {
		return 0, fmt.Errorf("claim notification email deliveries: %w", err)
	}
	var runErrors []error
	for _, delivery := range deliveries {
		if err := w.processClaimed(ctx, delivery); err != nil {
			runErrors = append(runErrors, fmt.Errorf("process notification email delivery %s: %w", delivery.ID, err))
		}
	}
	return len(deliveries), errors.Join(runErrors...)
}

func (w *EmailWorker) processClaimed(ctx context.Context, claimed DomainDelivery) error {
	authorized, err := w.repository.AuthorizeSend(ctx, claimed.ID, w.now())
	if err != nil {
		if errors.Is(err, ErrDeliveryLeaseInvalid) || errors.Is(err, ErrDeliveryOccurrenceFenced) ||
			errors.Is(err, ErrDeliveryChannelUnavailable) {
			w.logger.Info("email delivery rejected by final authorization",
				zap.String("delivery_id", claimed.ID.String()), zap.Error(err))
			return nil
		}
		return fmt.Errorf("authorize email delivery: %w", err)
	}
	recipient, err := w.protector.Unprotect(
		authorized.Channel, authorized.AddressCiphertext, authorized.AddressKeyVersion,
	)
	if err != nil {
		return w.recordLocalFailure(ctx, authorized, "recipient_data_error", err)
	}
	content, err := w.repository.LoadEmailDeliveryContent(ctx, authorized)
	if err != nil {
		return w.recordLocalFailure(ctx, authorized, "template_error", err)
	}
	subject, body, err := renderEmailDelivery(content)
	if err != nil {
		return w.recordLocalFailure(ctx, authorized, "template_error", err)
	}
	attempt, err := w.repository.StartEmailAttempt(ctx, authorized.ID, w.workerID, w.now())
	if err != nil {
		return fmt.Errorf("start email delivery attempt: %w", err)
	}
	if err := w.transport.SendOne(ctx, recipient, subject, body); err != nil {
		return w.finishTransportFailure(ctx, authorized, attempt, err)
	}

	finishedAt := w.now()
	if err := w.repository.FinishEmailAttempt(ctx, EmailAttemptCompletion{
		DeliveryID: authorized.ID, AttemptID: attempt.ID, WorkerID: w.workerID, FinishedAt: finishedAt,
		AttemptResult: "accepted", FlowState: "completed", DeliveryResult: "accepted",
	}); err != nil {
		return fmt.Errorf("complete accepted email delivery attempt: %w", err)
	}
	w.logger.Info("email accepted by smtp server", zap.String("delivery_id", authorized.ID.String()))
	return nil
}

func (w *EmailWorker) recordLocalFailure(
	ctx context.Context,
	delivery AuthorizedDelivery,
	category string,
	cause error,
) error {
	attempt, err := w.repository.StartEmailAttempt(ctx, delivery.ID, w.workerID, w.now())
	if err != nil {
		return errors.Join(fmt.Errorf("prepare email delivery: %w", cause), fmt.Errorf("start failed email attempt: %w", err))
	}
	return w.finishLocalFailure(ctx, delivery, attempt, category, cause)
}

func (w *EmailWorker) finishLocalFailure(
	ctx context.Context,
	delivery AuthorizedDelivery,
	attempt DomainDeliveryAttempt,
	category string,
	cause error,
) error {
	summary := "email delivery data validation failed"
	reason := category
	completion := EmailAttemptCompletion{
		DeliveryID: delivery.ID, AttemptID: attempt.ID, WorkerID: w.workerID, FinishedAt: w.now(),
		AttemptResult: "failed", ErrorCategory: &category, StatusSummary: &summary,
		FlowState: "dead_letter", DeliveryResult: "failed", FailureReason: &reason,
	}
	if err := w.repository.FinishEmailAttempt(ctx, completion); err != nil {
		return errors.Join(fmt.Errorf("prepare email delivery: %w", cause), fmt.Errorf("finish failed email attempt: %w", err))
	}
	return fmt.Errorf("prepare email delivery: %w", cause)
}

func (w *EmailWorker) finishTransportFailure(
	ctx context.Context,
	delivery AuthorizedDelivery,
	attempt DomainDeliveryAttempt,
	cause error,
) error {
	category, retryable, unknown, stage := classifyEmailTransportFailure(cause)
	summary := "smtp " + stage + " failed"
	completion := EmailAttemptCompletion{
		DeliveryID: delivery.ID, AttemptID: attempt.ID, WorkerID: w.workerID, FinishedAt: w.now(),
		AttemptResult: "failed", ErrorCategory: &category, StatusSummary: &summary,
		DeliveryResult: "failed",
	}
	if unknown {
		reason := EmailErrorUnknown
		completion.AttemptResult = "unknown"
		completion.FlowState = "completed"
		completion.DeliveryResult = "unknown"
		completion.FailureReason = &reason
	} else if retryable && attempt.AttemptNo < w.maxAttempts {
		next := completion.FinishedAt.Add(emailRetryDelay(delivery.ID, attempt.AttemptNo))
		completion.FlowState = "retry_wait"
		completion.NextRetryAt = &next
	} else {
		reason := category
		completion.FlowState = "dead_letter"
		completion.FailureReason = &reason
	}
	if err := w.repository.FinishEmailAttempt(ctx, completion); err != nil {
		return errors.Join(cause, fmt.Errorf("finish smtp email attempt: %w", err))
	}
	return cause
}

func classifyEmailTransportFailure(err error) (category string, retryable, unknown bool, stage string) {
	var sendErr *EmailSendError
	if errors.As(err, &sendErr) {
		return sendErr.Category, sendErr.Retryable, sendErr.OutcomeUnknown, sendErr.Stage
	}
	return EmailErrorConnection, true, false, "transport"
}

func emailRetryDelay(deliveryID uuid.UUID, attemptNo int) time.Duration {
	if attemptNo < 1 {
		attemptNo = 1
	}
	delay := time.Minute << min(attemptNo-1, 6)
	if delay > time.Hour {
		delay = time.Hour
	}
	// Stable ±20% jitter avoids synchronized retries without introducing a
	// process-local random source that would make recovery tests nondeterministic.
	jitterPercent := int(deliveryID[0])%41 - 20
	return delay + time.Duration(int64(delay)*int64(jitterPercent)/100)
}

func renderEmailDelivery(content EmailDeliveryContent) (string, string, error) {
	if content.Template.Channel != TemplateChannelEmail {
		return "", "", fmt.Errorf("email delivery template channel is %q", content.Template.Channel)
	}
	values := emailTemplateValues(content.Payload)
	values["event_count"] = strconv.Itoa(content.DigestEventCount)
	values["window_started_at"] = timeValue(content.DigestWindowStartedAt)
	values["window_ends_at"] = timeValue(content.DigestWindowEndsAt)
	subject, err := renderStrictText("subject", content.Template.Subject, values)
	if err != nil {
		return "", "", fmt.Errorf("render email subject: %w", err)
	}
	body, err := renderStrictText("text_body", content.Template.TextBody, values)
	if err != nil {
		return "", "", fmt.Errorf("render email text body: %w", err)
	}
	return subject, body, nil
}

func emailTemplateValues(payload event.AlarmLifecyclePayload) map[string]string {
	snapshot := payload.Snapshot
	return map[string]string{
		"alarm_name":       stringValue(snapshot.AlarmName),
		"alarm_identifier": snapshot.AlarmIdentifier,
		"severity":         emailSeverity(snapshot.Severity),
		"device_sn":        snapshot.DeviceSN,
		"device_id":        snapshot.DeviceID.String(),
		"device_type":      stringValue(snapshot.NEType),
		"carrier":          string(snapshot.Carrier),
		"technology":       stringValue(snapshot.Technology),
		"raised_at":        formatEmailTime(snapshot.RaisedAt),
		"occurred_at":      formatEmailTime(payload.OccurredAt),
		"cleared_at":       timeValue(snapshot.ClearedAt),
		"probable_cause":   stringValue(snapshot.ProbableCause),
		"specific_problem": stringValue(snapshot.SpecificProblem),
		"omc_url":          "",
		"status":           string(snapshot.Status),
	}
}

func emailSeverity(severity model.AlarmSeverity) string {
	switch int(severity) {
	case int(model.AlarmCritical), 31001:
		return "critical"
	case int(model.AlarmMajor), 31002:
		return "major"
	case int(model.AlarmMinor), 31003:
		return "minor"
	case int(model.AlarmWarning), 31004:
		return "warning"
	default:
		return strconv.Itoa(int(severity))
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func timeValue(value *time.Time) string {
	if value == nil {
		return ""
	}
	return formatEmailTime(*value)
}

func formatEmailTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
