package alarm

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/notification"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// EmailDispatcher 保留告警过滤引擎的窄接口；SMTP/MIME 统一由 notification 实现。
type EmailDispatcher interface {
	Dispatch(ctx context.Context, to []string, subject, body string) error
}

const (
	AlarmEmailLifecycleRaised  = "raised"
	AlarmEmailLifecycleCleared = "cleared"
)

// AlarmEmailDispatchRequest 为异步派发器补充可追踪和可幂等的业务上下文。
type AlarmEmailDispatchRequest struct {
	AlarmID    uuid.UUID
	RuleID     uuid.UUID
	Lifecycle  string
	Recipients []string
	Subject    string
	Body       string
}

// AlarmContextEmailDispatcher 由生产异步派发器实现；窄接口 EmailDispatcher
// 继续保留以兼容现有同步适配器和单测桩。
type AlarmContextEmailDispatcher interface {
	DispatchAlarm(context.Context, AlarmEmailDispatchRequest) error
}

// EmailMetrics 维度：result = success | failure | timeout。
type EmailMetrics struct {
	DispatchTotal *prometheus.CounterVec
}

func NewEmailMetrics(reg prometheus.Registerer) *EmailMetrics {
	m := &EmailMetrics{DispatchTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "alarm_email_dispatches_total",
		Help: "Total alarm email dispatches by result (success/failure/timeout)",
	}, []string{"result"})}
	if reg != nil {
		reg.MustRegister(m.DispatchTotal)
	}
	return m
}

// NotificationEmailDispatcher 把告警旧接口适配到共享 EmailTransport。
type NotificationEmailDispatcher struct {
	transport notification.EmailTransport
	logger    *zap.Logger
	metrics   *EmailMetrics
}

func NewNotificationEmailDispatcher(transport notification.EmailTransport, logger *zap.Logger, metrics *EmailMetrics) *NotificationEmailDispatcher {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &NotificationEmailDispatcher{transport: transport, logger: logger.Named("alarm-email"), metrics: metrics}
}

func (d *NotificationEmailDispatcher) Dispatch(ctx context.Context, to []string, subject, body string) error {
	if d.transport == nil {
		d.record("failure")
		return fmt.Errorf("alarm email transport is not configured")
	}
	if strings.TrimSpace(subject) == "" {
		d.record("failure")
		return fmt.Errorf("alarm email subject is empty")
	}
	err := d.transport.Send(ctx, notification.EmailMessage{To: to, Subject: subject, TextBody: body})
	if err == nil {
		d.record("success")
		return nil
	}
	result := "failure"
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		result = "timeout"
	}
	d.record(result)
	d.logger.Warn("alarm email dispatch failed", zap.Int("recipient_count", len(to)), zap.Error(err))
	return fmt.Errorf("dispatch alarm email: %w", err)
}

func (d *NotificationEmailDispatcher) record(result string) {
	if d.metrics != nil {
		d.metrics.DispatchTotal.WithLabelValues(result).Inc()
	}
}

type noopEmailDispatcher struct{}

func (noopEmailDispatcher) Dispatch(_ context.Context, _ []string, _ string, _ string) error {
	return nil
}
