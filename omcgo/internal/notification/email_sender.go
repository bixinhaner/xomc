package notification

import (
	"context"
	"crypto/tls"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

// SMTPOptions 是 EmailSender 的传输配置。由上层（cmd/app/provider）从 appconfig
// 映射而来 —— notification 包以此与 appconfig 解耦，便于单测构造。
type SMTPOptions struct {
	Enabled  bool          // false 时 Send 直接返回错误（未配置 SMTP）
	Host     string        // SMTP 服务器地址
	Port     int           // SMTP 端口（25 / 465 / 587）
	Username string        // 留空 = 不做 SMTP AUTH
	Password string        // SMTP AUTH 口令
	From     string        // 发件人地址
	StartTLS bool          // true 时在握手后升级 STARTTLS
	Timeout  time.Duration // 连接超时（<=0 时取默认 10s）
}

// EmailSender 通过 SMTP 发送纯文本（UTF-8）邮件。
//
// 不使用第三方邮件库：net/smtp 足以覆盖「可选 STARTTLS + 可选 AUTH + 纯文本」
// 这一需求，且不引入新依赖。HTML 邮件、附件等留待后续按需扩展。
type EmailSender struct {
	opts   SMTPOptions
	logger *zap.Logger
}

// NewEmailSender 创建 EmailSender。
func NewEmailSender(opts SMTPOptions, logger *zap.Logger) *EmailSender {
	if logger == nil {
		logger = zap.NewNop()
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 10 * time.Second
	}
	return &EmailSender{opts: opts, logger: logger.Named("email-sender")}
}

// Send 向所有收件人发送一封纯文本邮件。任一环节失败即返回错误。
func (s *EmailSender) Send(ctx context.Context, to []string, subject, body string) error {
	if !s.opts.Enabled {
		return fmt.Errorf("email sender disabled (notification.smtp.enabled=false)")
	}
	if s.opts.Host == "" {
		return fmt.Errorf("smtp host not configured")
	}
	if len(to) == 0 {
		return fmt.Errorf("no recipients")
	}

	addr := net.JoinHostPort(s.opts.Host, strconv.Itoa(s.opts.Port))
	dialer := net.Dialer{Timeout: s.opts.Timeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("dial smtp %s: %w", addr, err)
	}
	// 兜底超时，防止某个 SMTP 阶段无限挂起。
	_ = conn.SetDeadline(time.Now().Add(s.opts.Timeout))

	client, err := smtp.NewClient(conn, s.opts.Host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("smtp handshake: %w", err)
	}
	defer client.Close()

	if s.opts.StartTLS {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return fmt.Errorf("starttls=true but smtp server does not advertise STARTTLS")
		}
		if err := client.StartTLS(&tls.Config{ServerName: s.opts.Host}); err != nil {
			return fmt.Errorf("starttls: %w", err)
		}
	}
	if s.opts.Username != "" {
		auth := smtp.PlainAuth("", s.opts.Username, s.opts.Password, s.opts.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	if err := client.Mail(s.opts.From); err != nil {
		return fmt.Errorf("smtp MAIL FROM <%s>: %w", s.opts.From, err)
	}
	for _, rcpt := range to {
		if err := client.Rcpt(rcpt); err != nil {
			return fmt.Errorf("smtp RCPT TO <%s>: %w", rcpt, err)
		}
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp DATA: %w", err)
	}
	if _, err := w.Write([]byte(buildMessage(s.opts.From, to, subject, body))); err != nil {
		_ = w.Close()
		return fmt.Errorf("write message body: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("finalize message: %w", err)
	}
	return client.Quit()
}

// buildMessage 组装 RFC 5322 纯文本邮件报文。
// 主题用 RFC 2047 编码以支持中文；正文 UTF-8，统一 CRLF 换行。
func buildMessage(from string, to []string, subject, body string) string {
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + strings.Join(to, ", ") + "\r\n")
	b.WriteString("Subject: " + mime.QEncoding.Encode("utf-8", subject) + "\r\n")
	b.WriteString("Date: " + time.Now().Format(time.RFC1123Z) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
	b.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	b.WriteString("\r\n")
	b.WriteString(normalizeCRLF(body))
	return b.String()
}

// normalizeCRLF 把任意换行统一为 CRLF（SMTP DATA 要求）。
func normalizeCRLF(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(s, "\n", "\r\n")
}
