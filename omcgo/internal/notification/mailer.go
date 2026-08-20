package notification

import (
	"context"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// emailTransport 抽象邮件发送动作，便于 Mailer 单测注入 fake。
// *EmailSender 实现本接口。
type emailTransport interface {
	Send(ctx context.Context, to []string, subject, body string) error
}

// Mailer 把「模板渲染 + 邮件发送 + 历史记录」串成一次完整投递。
//
// 每次投递都会在 notification_history 落一条记录：先 pending，发送成功转 sent、
// 失败转 failed（带 error_message）。failed 行保留供运维排查；后续异步重试
// worker 可据 status=failed 复投并在彻底放弃时转 dead_letter（留作后续）。
type Mailer struct {
	templates *TemplateService
	history   *HistoryService
	transport emailTransport
	logger    *zap.Logger
}

// NewMailer 创建 Mailer。
func NewMailer(templates *TemplateService, history *HistoryService, transport emailTransport, logger *zap.Logger) *Mailer {
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
	return m.dispatch(ctx, recipients, subject, body, &tpl.ID, alarmID, false)
}

// SendRaw 发送一封已渲染好的邮件（不经模板），全程记入 notification_history。
func (m *Mailer) SendRaw(ctx context.Context, recipients []string, subject, body string, alarmID *uuid.UUID) error {
	return m.dispatch(ctx, recipients, subject, body, nil, alarmID, false)
}

// SendRawRecorded sends an already rendered email only after its pending
// history entry has been persisted. Administrative SMTP tests use this stricter
// contract so a successful test is always auditable.
func (m *Mailer) SendRawRecorded(ctx context.Context, recipients []string, subject, body string, alarmID *uuid.UUID) error {
	return m.dispatch(ctx, recipients, subject, body, nil, alarmID, true)
}

// dispatch 落历史(pending) → 发送 → 落终态(sent / failed)。
func (m *Mailer) dispatch(ctx context.Context, recipients []string, subject, body string, templateID, alarmID *uuid.UUID, requireHistory bool) error {
	if len(recipients) == 0 {
		return fmt.Errorf("no recipients")
	}

	h := &NotificationHistory{
		TemplateID: templateID,
		Channel:    TemplateChannelEmail,
		Recipients: recipients,
		Subject:    subject,
		Body:       body,
		Status:     HistoryStatusPending,
		AlarmID:    alarmID,
	}
	// 历史落库失败不阻断邮件发送 —— 邮件本身比审计记录优先。
	historyOK := true
	if err := m.history.Insert(ctx, h); err != nil {
		historyOK = false
		m.logger.Error("insert notification history failed", zap.Error(err))
		if requireHistory {
			return fmt.Errorf("record notification history: %w", err)
		}
	}

	if sendErr := m.transport.Send(ctx, recipients, subject, body); sendErr != nil {
		if historyOK {
			if err := m.history.MarkFailed(ctx, h.ID, sendErr.Error()); err != nil {
				m.logger.Error("mark history failed", zap.String("id", h.ID.String()), zap.Error(err))
			}
		}
		return fmt.Errorf("send email: %w", sendErr)
	}

	if historyOK {
		if err := m.history.MarkSent(ctx, h.ID, time.Now()); err != nil {
			m.logger.Error("mark history sent", zap.String("id", h.ID.String()), zap.Error(err))
		}
	}
	m.logger.Info("email sent",
		zap.Strings("recipients", maskEmailAddresses(recipients)),
		zap.String("subject", subject))
	return nil
}

func maskEmailAddresses(addresses []string) []string {
	masked := make([]string, 0, len(addresses))
	for _, raw := range addresses {
		parsed, err := mail.ParseAddress(strings.TrimSpace(raw))
		if err != nil {
			masked = append(masked, "***")
			continue
		}
		parts := strings.SplitN(parsed.Address, "@", 2)
		if len(parts) != 2 || parts[0] == "" {
			masked = append(masked, "***")
			continue
		}
		localRunes := []rune(parts[0])
		masked = append(masked, string(localRunes[0])+"***@"+parts[1])
	}
	return masked
}
