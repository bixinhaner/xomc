package notification

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Mailer 把「模板渲染 + 邮件发送 + 历史记录」串成一次完整投递。
//
// 每次投递都会在 notification_history 落一条记录：先 pending，发送成功转 sent、
// 失败转 failed（带 error_message）。failed 行保留供运维排查；异步任务在
// 最后一次尝试仍失败时转 dead_letter，后续重复消费不得把它当作发送成功。
type Mailer struct {
	templates *TemplateService
	history   *HistoryService
	transport EmailTransport
	logger    *zap.Logger
}

// pendingRetryAfter fences concurrent duplicate sends but must remain shorter
// than asyncjob.ZombieThreshold (5 minutes). A worker that crashes after
// ClaimAttempt must be able to reclaim the delivery on its first zombie retry,
// before the generic three-attempt budget is exhausted.
const pendingRetryAfter = 4 * time.Minute

var (
	ErrDeliveryPending    = errors.New("email delivery is already pending")
	ErrDeliveryDeadLetter = errors.New("email delivery is dead-lettered")
)

// DeliveryMetadata 关联通知记录与业务任务，并为重复消费提供幂等键。
type DeliveryMetadata struct {
	AlarmID      *uuid.UUID
	BusinessType string
	BusinessID   string
	DedupKey     string
	FinalAttempt bool
}

// DeliveryResult identifies the per-recipient audit record produced by a
// batch delivery. Recipient is returned to the in-process caller only; logs
// and error messages deliberately avoid including the address.
type DeliveryResult struct {
	HistoryID uuid.UUID
	Recipient string
}

// NewMailer 创建 Mailer。
func NewMailer(templates *TemplateService, history *HistoryService, transport EmailTransport, logger *zap.Logger) *Mailer {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Mailer{
		templates: templates,
		history:   history,
		transport: transport,
		logger:    logger.Named("mailer"),
	}
}

// SendByTemplate 按模板名渲染并发送邮件，全程记入 notification_history。
// alarmID 可为 nil（非告警类通知）。
func (m *Mailer) SendByTemplate(ctx context.Context, templateName string, recipients []string, vars map[string]string, alarmID *uuid.UUID) error {
	tpl, err := m.templates.GetByName(ctx, templateName)
	if err != nil || tpl == nil {
		return fmt.Errorf("notification template %q unavailable: %w", templateName, err)
	}
	if tpl.Channel != TemplateChannelEmail {
		return fmt.Errorf("template %q channel is %q, expected email", templateName, tpl.Channel)
	}
	if !tpl.Enabled {
		return fmt.Errorf("template %q is disabled", templateName)
	}
	subject, body, err := RenderTemplate(tpl, vars)
	if err != nil {
		return fmt.Errorf("render template %q: %w", templateName, err)
	}
	return m.dispatch(ctx, EmailMessage{To: recipients, Subject: subject, TextBody: body}, &tpl.ID, DeliveryMetadata{AlarmID: alarmID})
}

// SendRaw 发送一封已渲染好的邮件（不经模板），全程记入 notification_history。
func (m *Mailer) SendRaw(ctx context.Context, recipients []string, subject, body string, alarmID *uuid.UUID) error {
	return m.dispatch(ctx, EmailMessage{To: recipients, Subject: subject, TextBody: body}, nil, DeliveryMetadata{AlarmID: alarmID})
}

// SendMessage 发送可带附件的邮件。业务 runner 应提供稳定 DedupKey。
func (m *Mailer) SendMessage(ctx context.Context, message EmailMessage, metadata DeliveryMetadata) error {
	return m.dispatch(ctx, message, nil, metadata)
}

// SendMessageBatch sends one SMTP envelope per recipient and records one
// notification_history row per address. A rejected address therefore does not
// prevent other recipients from receiving the message. With a stable base
// DedupKey, retrying the batch skips addresses already marked sent and retries
// only failed or stale-pending deliveries.
func (m *Mailer) SendMessageBatch(ctx context.Context, message EmailMessage, metadata DeliveryMetadata) ([]DeliveryResult, error) {
	recipients, err := normalizeRecipients(message.To)
	if err != nil {
		return nil, err
	}
	results := make([]DeliveryResult, 0, len(recipients))
	errs := make([]error, 0)
	for index, recipient := range recipients {
		individual := message
		individual.To = []string{recipient}
		individualMetadata := metadata
		if metadata.DedupKey != "" {
			individualMetadata.DedupKey = recipientDedupKey(metadata.DedupKey, recipient)
		}
		historyID, sendErr := m.dispatchRecord(ctx, individual, nil, individualMetadata)
		if sendErr != nil {
			errs = append(errs, fmt.Errorf("recipient %d: %w", index+1, sendErr))
			continue
		}
		results = append(results, DeliveryResult{HistoryID: historyID, Recipient: recipient})
	}
	return results, errors.Join(errs...)
}

func recipientDedupKey(base, recipient string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(recipient))))
	return base + ":recipient:" + hex.EncodeToString(sum[:])
}

// dispatch 落历史(pending) → 记录尝试 → 发送 → 落终态(sent / failed)。
func (m *Mailer) dispatch(ctx context.Context, message EmailMessage, templateID *uuid.UUID, metadata DeliveryMetadata) error {
	_, err := m.dispatchRecord(ctx, message, templateID, metadata)
	return err
}

func (m *Mailer) dispatchRecord(ctx context.Context, message EmailMessage, templateID *uuid.UUID, metadata DeliveryMetadata) (uuid.UUID, error) {
	recipients, err := normalizeRecipients(message.To)
	if err != nil {
		return uuid.Nil, err
	}
	message.To = recipients
	businessType := optionalString(metadata.BusinessType)
	businessID := optionalString(metadata.BusinessID)
	dedupKey := optionalString(metadata.DedupKey)

	h := &NotificationHistory{
		TemplateID:   templateID,
		Channel:      TemplateChannelEmail,
		Recipients:   message.To,
		Subject:      message.Subject,
		Body:         message.TextBody,
		Status:       HistoryStatusPending,
		AlarmID:      metadata.AlarmID,
		BusinessType: businessType,
		BusinessID:   businessID,
		DedupKey:     dedupKey,
	}
	// 无幂等键的旧调用保持“审计失败不阻断邮件”语义；带幂等键的
	// 异步任务必须先成功占位，否则无法保证不重复投递。
	historyOK := true
	record, created, err := m.history.InsertIfAbsent(ctx, h)
	if err != nil {
		historyOK = false
		m.logger.Error("insert notification history failed", zap.Error(err))
		if metadata.DedupKey != "" {
			return uuid.Nil, fmt.Errorf("create idempotent notification history: %w", err)
		}
	} else {
		h = record
		attemptedAt := time.Now()
		if !created {
			switch h.Status {
			case HistoryStatusSent:
				m.logger.Info("skip duplicate email delivery", zap.String("history_id", h.ID.String()), zap.String("status", h.Status))
				return h.ID, nil
			case HistoryStatusDeadLetter:
				return h.ID, fmt.Errorf("%w: history_id=%s", ErrDeliveryDeadLetter, h.ID)
			case HistoryStatusPending:
				if h.AttemptedAt == nil || h.AttemptedAt.After(attemptedAt.Add(-pendingRetryAfter)) {
					m.logger.Info("skip duplicate email delivery", zap.String("history_id", h.ID.String()), zap.String("status", h.Status))
					return h.ID, fmt.Errorf("%w: history_id=%s", ErrDeliveryPending, h.ID)
				}
				m.logger.Warn("retry stale pending email delivery", zap.String("history_id", h.ID.String()), zap.Time("attempted_at", *h.AttemptedAt))
			}
		}
		claimed, claimErr := m.history.ClaimAttempt(ctx, h.ID, attemptedAt, attemptedAt.Add(-pendingRetryAfter))
		if claimErr != nil {
			historyOK = false
			m.logger.Error("claim notification attempt failed", zap.String("id", h.ID.String()), zap.Error(claimErr))
			if metadata.DedupKey != "" {
				return h.ID, fmt.Errorf("record idempotent email attempt: %w", claimErr)
			}
		} else if !claimed {
			latest, loadErr := m.history.GetByID(ctx, h.ID)
			if loadErr == nil {
				h = latest
				if h.Status == HistoryStatusSent {
					m.logger.Info("skip duplicate email delivery", zap.String("history_id", h.ID.String()), zap.String("status", h.Status))
					return h.ID, nil
				}
				if h.Status == HistoryStatusDeadLetter {
					return h.ID, fmt.Errorf("%w: history_id=%s", ErrDeliveryDeadLetter, h.ID)
				}
			}
			historyOK = false
			if metadata.DedupKey != "" {
				if loadErr != nil {
					return h.ID, fmt.Errorf("load unclaimed idempotent email attempt: %w", loadErr)
				}
				return h.ID, fmt.Errorf("%w: history_id=%s", ErrDeliveryPending, h.ID)
			}
		}
	}

	if sendErr := m.transport.Send(ctx, message); sendErr != nil {
		if historyOK {
			markStatus := m.history.MarkFailed
			if metadata.FinalAttempt {
				markStatus = m.history.MarkDeadLetter
			}
			if err := markStatus(ctx, h.ID, sendErr.Error()); err != nil {
				m.logger.Error("mark history failed", zap.String("id", h.ID.String()), zap.Error(err))
			}
		}
		return h.ID, fmt.Errorf("send email: %w", sendErr)
	}

	if historyOK {
		if err := m.history.MarkSent(ctx, h.ID, time.Now()); err != nil {
			m.logger.Error("mark history sent", zap.String("id", h.ID.String()), zap.Error(err))
			if metadata.DedupKey != "" {
				return h.ID, fmt.Errorf("record idempotent email sent status: %w", err)
			}
		}
	}
	m.logger.Info("email sent",
		zap.Int("recipient_count", len(message.To)),
		zap.String("subject", message.Subject))
	return h.ID, nil
}

func optionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}
