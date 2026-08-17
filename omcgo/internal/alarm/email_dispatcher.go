package alarm

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/notification"
)

// EmailDispatcher 是告警模块使用的最小邮件传输接口。
// 最终的告警订阅会改为异步任务；保留本接口只为当前主线兼容。
type EmailDispatcher interface {
	Dispatch(ctx context.Context, to []string, subject, body string) error
}

type emailTransport interface {
	Send(ctx context.Context, to []string, subject, body string) error
}

// EmailConfig 是历史告警装配层的兼容配置。SMTP 协议实现已经统一迁移到
// internal/notification.EmailSender；这里不再维护第二套 net/smtp 客户端。
type EmailConfig struct {
	Enabled     bool
	Host        string
	Port        int
	AuthEnabled bool
	Username    string
	Password    string
	From        string
	UseTLS      bool
	UseSTARTTLS bool
	Timeout     time.Duration
}

const defaultEmailTimeout = 5 * time.Second

type EmailMetrics struct {
	DispatchTotal *prometheus.CounterVec
}

func NewEmailMetrics(reg prometheus.Registerer) *EmailMetrics {
	metrics := &EmailMetrics{
		DispatchTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "alarm_email_dispatches_total",
			Help: "Total alarm email dispatches by result (success/failure/timeout)",
		}, []string{"result"}),
	}
	if reg != nil {
		reg.MustRegister(metrics.DispatchTotal)
	}
	return metrics
}

// SMTPEmailDispatcher 是告警接口到统一 notification EmailSender 的薄适配器。
type SMTPEmailDispatcher struct {
	sender  emailTransport
	timeout time.Duration
	logger  *zap.Logger
	metrics *EmailMetrics
}

func NewSMTPEmailDispatcher(cfg EmailConfig, logger *zap.Logger, metrics *EmailMetrics) *SMTPEmailDispatcher {
	if logger == nil {
		logger = zap.NewNop()
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultEmailTimeout
	}
	securityMode := notification.SMTPSecurityNone
	if cfg.UseSTARTTLS {
		securityMode = notification.SMTPSecuritySTARTTLS
	} else if cfg.UseTLS {
		securityMode = notification.SMTPSecurityImplicitTLS
	}
	sender := notification.NewEmailSender(notification.SMTPOptions{
		Enabled:      cfg.Enabled,
		Host:         cfg.Host,
		Port:         cfg.Port,
		SecurityMode: securityMode,
		AuthEnabled:  cfg.AuthEnabled,
		Username:     cfg.Username,
		Password:     cfg.Password,
		From:         cfg.From,
		Timeout:      cfg.Timeout,
	}, logger)
	return NewEmailDispatcher(sender, cfg.Timeout, logger, metrics)
}

// NewEmailDispatcher adapts the shared notification transport to the legacy
// alarm interface while the synchronous notify_email action is being retired.
func NewEmailDispatcher(sender emailTransport, timeout time.Duration, logger *zap.Logger, metrics *EmailMetrics) *SMTPEmailDispatcher {
	if logger == nil {
		logger = zap.NewNop()
	}
	if timeout <= 0 {
		timeout = defaultEmailTimeout
	}
	return &SMTPEmailDispatcher{
		sender:  sender,
		timeout: timeout,
		logger:  logger,
		metrics: metrics,
	}
}

func (d *SMTPEmailDispatcher) Dispatch(ctx context.Context, to []string, subject, body string) error {
	dispatchCtx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	result := make(chan error, 1)
	go func() {
		result <- d.sender.Send(dispatchCtx, to, subject, body)
	}()

	select {
	case <-dispatchCtx.Done():
		d.record("timeout")
		return fmt.Errorf("smtp dispatch: %w", dispatchCtx.Err())
	case err := <-result:
		if err != nil {
			if isEmailTimeout(err) {
				d.record("timeout")
				d.logger.Warn("email dispatch timeout", zap.Error(err))
				return fmt.Errorf("smtp dispatch timeout: %v: %w", err, context.DeadlineExceeded)
			} else {
				d.record("failure")
			}
			d.logger.Warn("email dispatch failed", zap.Error(err))
			return fmt.Errorf("smtp dispatch: %w", err)
		}
		d.record("success")
		return nil
	}
}

func isEmailTimeout(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func (d *SMTPEmailDispatcher) record(result string) {
	if d.metrics != nil {
		d.metrics.DispatchTotal.WithLabelValues(result).Inc()
	}
}

type noopEmailDispatcher struct{}

func (noopEmailDispatcher) Dispatch(_ context.Context, _ []string, _ string, _ string) error {
	return nil
}
