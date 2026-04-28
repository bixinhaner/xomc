package alarm

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// EmailDispatcher 告警邮件派发接口。
// W2 Block A.1（T-0007）冒烟版本只实现：to 列表 + Subject + 纯文本 body 的单次 SMTP 投递。
// HTML body / 多语言模板 / 附件 / 重试 / 死信留给 T-0043 全量实现。
type EmailDispatcher interface {
	// Dispatch 把 subject+body 以 RFC 822 文本邮件发到 to 列表里所有地址。
	// to / subject 为空时返回参数错。失败应返回 error，调用方决定是否上报。
	Dispatch(ctx context.Context, to []string, subject, body string) error
}

// EmailConfig SMTP 客户端配置。
//
//   - UseTLS    : implicit TLS（一般 465 端口），先 tls.Dial 再握手 SMTP
//   - UseSTARTTLS: 标准 SMTP 上 STARTTLS 升级（一般 587 端口）
//   - 两者都为 false：明文 SMTP（仅限本地内网/测试）
//   - Timeout   : 拨号 + 命令读写整体超时，默认 5s（NewSMTPEmailDispatcher 兜底）
type EmailConfig struct {
	Host        string
	Port        int
	Username    string
	Password    string
	From        string
	UseTLS      bool
	UseSTARTTLS bool
	Timeout     time.Duration
}

const (
	defaultEmailTimeout = 5 * time.Second
	emailUserAgent      = "omcgo-alarm-email/1.0"
)

// EmailMetrics 维度：result = success | failure | timeout。
type EmailMetrics struct {
	DispatchTotal *prometheus.CounterVec
}

// NewEmailMetrics 创建并注册邮件派发指标。
// reg 为 nil 时仅返回未注册的指标对象（便于单测共享独立 registry）。
func NewEmailMetrics(reg prometheus.Registerer) *EmailMetrics {
	m := &EmailMetrics{
		DispatchTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "alarm_email_dispatches_total",
			Help: "Total alarm email dispatches by result (success/failure/timeout)",
		}, []string{"result"}),
	}
	if reg != nil {
		reg.MustRegister(m.DispatchTotal)
	}
	return m
}

// SMTPEmailDispatcher 基于标准库 net/smtp 的邮件派发实现。
type SMTPEmailDispatcher struct {
	cfg     EmailConfig
	logger  *zap.Logger
	metrics *EmailMetrics
	// nowFn 注入便于测试 Date 头与超时计算（默认 time.Now）。
	nowFn func() time.Time
}

// NewSMTPEmailDispatcher 构造 SMTP 派发器。
// logger / metrics 任一为 nil 时使用 NoOp 替身，避免调用方 nil-check。
func NewSMTPEmailDispatcher(cfg EmailConfig, logger *zap.Logger, metrics *EmailMetrics) *SMTPEmailDispatcher {
	if logger == nil {
		logger = zap.NewNop()
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultEmailTimeout
	}
	return &SMTPEmailDispatcher{
		cfg:     cfg,
		logger:  logger,
		metrics: metrics,
		nowFn:   time.Now,
	}
}

// Dispatch 实现 EmailDispatcher 接口。
//
// 流程：
//  1. 入参验证（to 非空、subject 非空、From 非空）
//  2. 拼接 RFC 822 消息（From / To / Subject / Date / MIME-Version / Content-Type）
//  3. 起 goroutine 执行 SMTP 投递；主 goroutine select 监听 ctx.Done() 与投递结果
//  4. ctx 取消优先于 SMTP 错误返回（结果维度区分 timeout / failure / success）
func (d *SMTPEmailDispatcher) Dispatch(ctx context.Context, to []string, subject, body string) error {
	if err := validateEmailParams(d.cfg.From, to, subject); err != nil {
		d.recordFailure()
		return err
	}

	msg := buildEmailMessage(d.cfg.From, to, subject, body, d.nowFn())

	// 用带超时的子 context 包一层；select 同时监听 ctx 与投递完成。
	dialCtx, cancel := context.WithTimeout(ctx, d.cfg.Timeout)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- d.sendSMTP(to, msg)
	}()

	select {
	case <-dialCtx.Done():
		// 区分调用方主动取消 vs 自身超时
		if errors.Is(dialCtx.Err(), context.DeadlineExceeded) {
			d.logger.Warn("email dispatch timeout",
				zap.String("smtp_host", d.cfg.Host),
				zap.Int("smtp_port", d.cfg.Port),
				zap.Duration("timeout", d.cfg.Timeout))
			d.recordTimeout()
			return fmt.Errorf("smtp dispatch: %w", dialCtx.Err())
		}
		d.recordFailure()
		return fmt.Errorf("smtp dispatch: %w", dialCtx.Err())
	case err := <-errCh:
		if err != nil {
			d.logger.Warn("email dispatch failed",
				zap.String("smtp_host", d.cfg.Host),
				zap.Int("smtp_port", d.cfg.Port),
				zap.Strings("to", to),
				zap.Error(err))
			d.recordFailure()
			return fmt.Errorf("smtp dispatch: %w", err)
		}
		d.recordSuccess()
		return nil
	}
}

// validateEmailParams 校验入参；错误以 sentinel 风格返回，便于上层 errors.Is 识别。
func validateEmailParams(from string, to []string, subject string) error {
	if strings.TrimSpace(from) == "" {
		return errors.New("smtp dispatch: from address empty")
	}
	if len(to) == 0 {
		return errors.New("smtp dispatch: recipient list empty")
	}
	for _, addr := range to {
		if strings.TrimSpace(addr) == "" {
			return errors.New("smtp dispatch: empty recipient in list")
		}
	}
	if strings.TrimSpace(subject) == "" {
		return errors.New("smtp dispatch: subject empty")
	}
	return nil
}

// buildEmailMessage 拼接最小可用 RFC 822 / MIME 邮件正文。
// 头字段顺序：Date / From / To / Subject / MIME-Version / Content-Type / User-Agent。
func buildEmailMessage(from string, to []string, subject, body string, now time.Time) []byte {
	var sb strings.Builder
	sb.WriteString("Date: ")
	sb.WriteString(now.UTC().Format(time.RFC1123Z))
	sb.WriteString("\r\n")
	sb.WriteString("From: ")
	sb.WriteString(from)
	sb.WriteString("\r\n")
	sb.WriteString("To: ")
	sb.WriteString(strings.Join(to, ", "))
	sb.WriteString("\r\n")
	sb.WriteString("Subject: ")
	sb.WriteString(subject)
	sb.WriteString("\r\n")
	sb.WriteString("MIME-Version: 1.0\r\n")
	sb.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	sb.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	sb.WriteString("User-Agent: ")
	sb.WriteString(emailUserAgent)
	sb.WriteString("\r\n")
	sb.WriteString("\r\n")
	sb.WriteString(body)
	return []byte(sb.String())
}

// sendSMTP 走标准 net/smtp 流程：Dial → Hello → (StartTLS) → (Auth) → Mail → Rcpt → Data → Quit。
// implicit TLS 走 tls.Dial 后再 smtp.NewClient。
func (d *SMTPEmailDispatcher) sendSMTP(to []string, msg []byte) error {
	addr := net.JoinHostPort(d.cfg.Host, strconv.Itoa(d.cfg.Port))
	dialer := &net.Dialer{Timeout: d.cfg.Timeout}

	var client *smtp.Client
	var err error
	if d.cfg.UseTLS {
		// implicit TLS
		tlsConn, errDial := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{ServerName: d.cfg.Host, MinVersion: tls.VersionTLS12})
		if errDial != nil {
			return fmt.Errorf("dial tls smtp %s: %w", addr, errDial)
		}
		client, err = smtp.NewClient(tlsConn, d.cfg.Host)
		if err != nil {
			_ = tlsConn.Close()
			return fmt.Errorf("smtp new client: %w", err)
		}
	} else {
		conn, errDial := dialer.Dial("tcp", addr)
		if errDial != nil {
			return fmt.Errorf("dial smtp %s: %w", addr, errDial)
		}
		client, err = smtp.NewClient(conn, d.cfg.Host)
		if err != nil {
			_ = conn.Close()
			return fmt.Errorf("smtp new client: %w", err)
		}
	}
	defer func() {
		_ = client.Quit()
		_ = client.Close()
	}()

	if err := client.Hello("omcgo"); err != nil {
		return fmt.Errorf("smtp hello: %w", err)
	}

	if d.cfg.UseSTARTTLS {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return errors.New("smtp dispatch: server does not support STARTTLS")
		}
		if err := client.StartTLS(&tls.Config{ServerName: d.cfg.Host, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("smtp starttls: %w", err)
		}
	}

	if d.cfg.Username != "" || d.cfg.Password != "" {
		if ok, _ := client.Extension("AUTH"); ok {
			auth := smtp.PlainAuth("", d.cfg.Username, d.cfg.Password, d.cfg.Host)
			if err := client.Auth(auth); err != nil {
				return fmt.Errorf("smtp auth: %w", err)
			}
		}
	}

	if err := client.Mail(d.cfg.From); err != nil {
		return fmt.Errorf("smtp mail-from: %w", err)
	}
	for _, rcpt := range to {
		if err := client.Rcpt(rcpt); err != nil {
			return fmt.Errorf("smtp rcpt-to %s: %w", rcpt, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("smtp data write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp data close: %w", err)
	}
	return nil
}

func (d *SMTPEmailDispatcher) recordSuccess() {
	if d.metrics != nil {
		d.metrics.DispatchTotal.WithLabelValues("success").Inc()
	}
}

func (d *SMTPEmailDispatcher) recordFailure() {
	if d.metrics != nil {
		d.metrics.DispatchTotal.WithLabelValues("failure").Inc()
	}
}

func (d *SMTPEmailDispatcher) recordTimeout() {
	if d.metrics != nil {
		d.metrics.DispatchTotal.WithLabelValues("timeout").Inc()
	}
}

// noopEmailDispatcher 占位实现：未注入 EmailDispatcher 时 FilterEngine 回退使用，避免散落 nil-check。
// （filter_engine.go 接入由主会话整合，本任务不修改 filter_engine.go。）
type noopEmailDispatcher struct{}

func (noopEmailDispatcher) Dispatch(_ context.Context, _ []string, _ string, _ string) error {
	return nil
}
