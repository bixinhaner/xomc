package alarm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/asyncjob"
	"github.com/omcgo/omcgo/internal/notification"
)

const AlarmEmailJobType = "alarm_email_notification"

type alarmEmailJobPayload struct {
	Recipients []string `json:"recipients"`
	Subject    string   `json:"subject"`
	Body       string   `json:"body"`
	DedupKey   string   `json:"dedup_key"`
	AlarmID    string   `json:"alarm_id,omitempty"`
	RuleID     string   `json:"rule_id,omitempty"`
	Lifecycle  string   `json:"lifecycle,omitempty"`
}

type alarmEmailJobEnqueuer interface {
	Insert(context.Context, asyncjob.InsertRequest) (uuid.UUID, error)
}

// AsyncEmailDispatcher 只冻结邮件快照并入队，不在告警接收链路执行 SMTP。
type AsyncEmailDispatcher struct {
	jobs    alarmEmailJobEnqueuer
	metrics *EmailMetrics
}

func NewAsyncEmailDispatcher(jobs alarmEmailJobEnqueuer, metrics *EmailMetrics) *AsyncEmailDispatcher {
	return &AsyncEmailDispatcher{jobs: jobs, metrics: metrics}
}

func (d *AsyncEmailDispatcher) Dispatch(ctx context.Context, recipients []string, subject, body string) error {
	return d.dispatch(ctx, AlarmEmailDispatchRequest{Recipients: recipients, Subject: subject, Body: body})
}

func (d *AsyncEmailDispatcher) DispatchAlarm(ctx context.Context, request AlarmEmailDispatchRequest) error {
	return d.dispatch(ctx, request)
}

func (d *AsyncEmailDispatcher) dispatch(ctx context.Context, request AlarmEmailDispatchRequest) error {
	if d.jobs == nil {
		d.record("failure")
		return fmt.Errorf("alarm email job repository is not configured")
	}
	normalized, err := notification.NormalizeRecipients(request.Recipients)
	if err != nil {
		d.record("failure")
		return err
	}
	if strings.TrimSpace(request.Subject) == "" {
		d.record("failure")
		return fmt.Errorf("alarm email subject is empty")
	}
	dedupKey := alarmEmailDedupKey(normalized, request.Subject, request.Body,
		request.AlarmID.String(), request.RuleID.String(), request.Lifecycle)
	payload, err := json.Marshal(alarmEmailJobPayload{
		Recipients: normalized, Subject: request.Subject, Body: request.Body, DedupKey: dedupKey,
		AlarmID: optionalUUIDString(request.AlarmID), RuleID: optionalUUIDString(request.RuleID), Lifecycle: request.Lifecycle,
	})
	if err != nil {
		d.record("failure")
		return fmt.Errorf("encode alarm email job: %w", err)
	}
	if _, err := d.jobs.Insert(ctx, asyncjob.InsertRequest{
		JobType: AlarmEmailJobType, ScheduledAt: time.Now(), Payload: payload,
	}); err != nil {
		d.record("failure")
		return fmt.Errorf("enqueue alarm email job: %w", err)
	}
	d.record("success")
	return nil
}

func (d *AsyncEmailDispatcher) record(result string) {
	if d.metrics != nil {
		d.metrics.DispatchTotal.WithLabelValues(result).Inc()
	}
}

func alarmEmailDedupKey(recipients []string, subject, body string, businessParts ...string) string {
	stableRecipients := append([]string(nil), recipients...)
	sort.Strings(stableRecipients)
	sum := sha256.Sum256([]byte(strings.Join(businessParts, "\x00") + "\x00" + strings.Join(stableRecipients, ";") + "\x00" + subject + "\x00" + body))
	return hex.EncodeToString(sum[:])
}

func optionalUUIDString(value uuid.UUID) string {
	if value == uuid.Nil {
		return ""
	}
	return value.String()
}

type EmailJobRunner struct {
	mailer *notification.Mailer
	logger *zap.Logger
}

func NewEmailJobRunner(mailer *notification.Mailer, logger *zap.Logger) *EmailJobRunner {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &EmailJobRunner{mailer: mailer, logger: logger.Named("alarm.email.runner")}
}

func (r *EmailJobRunner) JobType() string { return AlarmEmailJobType }

func (r *EmailJobRunner) Run(ctx context.Context, job *asyncjob.Job) (json.RawMessage, error) {
	if r.mailer == nil {
		return nil, fmt.Errorf("alarm email mailer is not configured")
	}
	var payload alarmEmailJobPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return nil, fmt.Errorf("decode alarm email job: %w", err)
	}
	if _, err := r.mailer.SendMessageBatch(ctx, notification.EmailMessage{
		To: payload.Recipients, Subject: payload.Subject, TextBody: payload.Body,
	}, notification.DeliveryMetadata{
		AlarmID:      parseOptionalUUID(payload.AlarmID),
		BusinessType: "alarm", BusinessID: alarmEmailBusinessID(payload),
		DedupKey:     "alarm-email:" + payload.DedupKey,
		FinalAttempt: job.MaxAttempts > 0 && job.Attempt >= job.MaxAttempts,
	}); err != nil {
		return nil, fmt.Errorf("send alarm email job: %w", err)
	}
	return json.Marshal(map[string]string{"status": "sent", "dedup_key": payload.DedupKey})
}

func alarmEmailBusinessID(payload alarmEmailJobPayload) string {
	parts := make([]string, 0, 3)
	for _, value := range []string{payload.AlarmID, payload.RuleID, payload.Lifecycle} {
		if value = strings.TrimSpace(value); value != "" {
			parts = append(parts, value)
		}
	}
	return strings.Join(parts, ":")
}

func parseOptionalUUID(value string) *uuid.UUID {
	id, err := uuid.Parse(value)
	if err != nil {
		return nil
	}
	return &id
}

var _ asyncjob.JobRunner = (*EmailJobRunner)(nil)
