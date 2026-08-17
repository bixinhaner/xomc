package notification

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"mime"
	"mime/multipart"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

// SMTPSecurityMode 描述 SMTP 连接建立方式。邮箱服务商只体现在配置值，
// 业务代码不识别任何邮箱厂商名称。
type SMTPSecurityMode string

const (
	SMTPSecurityNone        SMTPSecurityMode = "none"
	SMTPSecuritySTARTTLS    SMTPSecurityMode = "starttls"
	SMTPSecurityImplicitTLS SMTPSecurityMode = "implicit_tls"

	defaultSMTPTimeout = 10 * time.Second
)

// SMTPOptions 是统一 SMTP 传输配置。
//
// AuthEnabled=false 支持客户内网无认证 SMTP；Password 应由 write-only 配置或
// 部署 Secret 注入，调用方不得把它写入日志或 HTTP 响应。
type SMTPOptions struct {
	Enabled      bool
	Host         string
	Port         int
	SecurityMode SMTPSecurityMode
	AuthEnabled  bool
	Username     string
	Password     string
	From         string
	Timeout      time.Duration

	// tlsConfig 仅供同包测试注入受信任 CA；生产始终由 tlsConfigForHost 生成。
	tlsConfig *tls.Config
}

// Validate 在连接外部 SMTP 前完成确定性配置校验。
func (o SMTPOptions) Validate() error {
	if !o.Enabled {
		return fmt.Errorf("email sender disabled")
	}
	if strings.TrimSpace(o.Host) == "" {
		return fmt.Errorf("smtp host not configured")
	}
	if o.Port < 1 || o.Port > 65535 {
		return fmt.Errorf("smtp port must be between 1 and 65535, got %d", o.Port)
	}
	if _, err := mail.ParseAddress(strings.TrimSpace(o.From)); err != nil {
		return fmt.Errorf("invalid smtp from address: %w", err)
	}
	switch o.SecurityMode {
	case "", SMTPSecurityNone, SMTPSecuritySTARTTLS, SMTPSecurityImplicitTLS:
	default:
		return fmt.Errorf("unsupported smtp security mode %q", o.SecurityMode)
	}
	if o.AuthEnabled {
		if strings.TrimSpace(o.Username) == "" {
			return fmt.Errorf("smtp username is required when authentication is enabled")
		}
		if o.Password == "" {
			return fmt.Errorf("smtp password is required when authentication is enabled")
		}
		if o.SecurityMode == "" || o.SecurityMode == SMTPSecurityNone {
			return fmt.Errorf("smtp authentication requires starttls or implicit_tls")
		}
	}
	return nil
}

// EmailSender 是全系统唯一的 SMTP 协议实现。告警、KPI 和测试邮件均通过它发送。
type EmailSender struct {
	opts   SMTPOptions
	logger *zap.Logger
}

// EmailAttachment 是已从共享对象存储读取并限长校验后的邮件附件。
// 文件内容只在发送期间驻留内存，不写日志。
type EmailAttachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

// NewEmailSender 创建 EmailSender；未指定超时时使用安全默认值。
func NewEmailSender(opts SMTPOptions, logger *zap.Logger) *EmailSender {
	if logger == nil {
		logger = zap.NewNop()
	}
	if opts.Timeout <= 0 {
		opts.Timeout = defaultSMTPTimeout
	}
	if opts.SecurityMode == "" {
		opts.SecurityMode = SMTPSecurityNone
	}
	return &EmailSender{opts: opts, logger: logger.Named("email-sender")}
}

// Send 向所有收件人发送一封 UTF-8 纯文本邮件。任一环节失败即返回错误。
func (s *EmailSender) Send(ctx context.Context, to []string, subject, body string) error {
	return s.send(ctx, to, subject, buildMessage(s.opts.From, to, subject, body, "text/plain"))
}

// SendHTML sends an HTML body with the correct MIME type.
func (s *EmailSender) SendHTML(ctx context.Context, to []string, subject, body string) error {
	return s.send(ctx, to, subject, buildMessage(s.opts.From, to, subject, body, "text/html"))
}

func (s *EmailSender) SendHTMLWithMessageID(ctx context.Context, to []string, subject, body, messageID string) error {
	return s.send(ctx, to, subject, buildMessage(s.opts.From, to, subject, body, "text/html", messageID))
}

// SendWithAttachments 发送 UTF-8 纯文本正文及附件。附件名会按 RFC 2231 编码，
// 内容使用 base64 和 76 字符折行，兼容常见企业邮箱网关。
func (s *EmailSender) SendWithAttachments(
	ctx context.Context,
	to []string,
	subject, body string,
	attachments []EmailAttachment,
) error {
	if len(attachments) == 0 {
		return s.Send(ctx, to, subject, body)
	}
	recipients, err := normalizeRecipients(to)
	if err != nil {
		return err
	}
	message, err := buildMessageWithAttachments(s.opts.From, recipients, subject, body, attachments)
	if err != nil {
		return err
	}
	return s.send(ctx, recipients, subject, message)
}

func (s *EmailSender) SendWithAttachmentsMessageID(
	ctx context.Context,
	to []string,
	subject, body string,
	attachments []EmailAttachment,
	messageID string,
) error {
	if len(attachments) == 0 {
		return s.send(ctx, to, subject, buildMessage(s.opts.From, to, subject, body, "text/plain", messageID))
	}
	recipients, err := normalizeRecipients(to)
	if err != nil {
		return err
	}
	message, err := buildMessageWithAttachments(s.opts.From, recipients, subject, body, attachments, messageID)
	if err != nil {
		return err
	}
	return s.send(ctx, recipients, subject, message)
}

func (s *EmailSender) send(ctx context.Context, to []string, subject, message string) error {
	if err := s.opts.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(subject) == "" {
		return fmt.Errorf("email subject is required")
	}
	recipients, err := normalizeRecipients(to)
	if err != nil {
		return err
	}

	client, conn, err := s.openClient(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = client.Close()
		_ = conn.Close()
	}()

	if s.opts.SecurityMode == SMTPSecuritySTARTTLS {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return fmt.Errorf("smtp server does not advertise STARTTLS")
		}
		if err := client.StartTLS(s.tlsConfigForHost()); err != nil {
			return fmt.Errorf("smtp starttls: %w", err)
		}
	}
	if s.opts.AuthEnabled {
		if ok, _ := client.Extension("AUTH"); !ok {
			return fmt.Errorf("smtp server does not advertise AUTH")
		}
		auth := smtp.PlainAuth("", s.opts.Username, s.opts.Password, s.opts.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	from, _ := mail.ParseAddress(strings.TrimSpace(s.opts.From))
	if err := client.Mail(from.Address); err != nil {
		return fmt.Errorf("smtp MAIL FROM: %w", err)
	}
	for _, rcpt := range recipients {
		if err := client.Rcpt(rcpt); err != nil {
			return fmt.Errorf("smtp RCPT TO: %w", err)
		}
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp DATA: %w", err)
	}
	if _, err := w.Write([]byte(message)); err != nil {
		_ = w.Close()
		return fmt.Errorf("write smtp message: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("finalize smtp message: %w", err)
	}
	if err := client.Quit(); err != nil {
		return fmt.Errorf("smtp quit: %w", err)
	}
	return nil
}

func (s *EmailSender) openClient(ctx context.Context) (*smtp.Client, net.Conn, error) {
	addr := net.JoinHostPort(s.opts.Host, strconv.Itoa(s.opts.Port))
	dialer := &net.Dialer{Timeout: s.opts.Timeout}
	var conn net.Conn
	var err error
	if s.opts.SecurityMode == SMTPSecurityImplicitTLS {
		tlsDialer := &tls.Dialer{NetDialer: dialer, Config: s.tlsConfigForHost()}
		conn, err = tlsDialer.DialContext(ctx, "tcp", addr)
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("dial smtp %s: %w", addr, err)
	}
	if err := conn.SetDeadline(time.Now().Add(s.opts.Timeout)); err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("set smtp deadline: %w", err)
	}
	client, err := smtp.NewClient(conn, s.opts.Host)
	if err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("smtp handshake: %w", err)
	}
	return client, conn, nil
}

func (s *EmailSender) tlsConfigForHost() *tls.Config {
	if s.opts.tlsConfig != nil {
		return s.opts.tlsConfig.Clone()
	}
	return &tls.Config{ServerName: s.opts.Host, MinVersion: tls.VersionTLS12}
}

func normalizeRecipients(to []string) ([]string, error) {
	if len(to) == 0 {
		return nil, fmt.Errorf("no recipients")
	}
	seen := make(map[string]struct{}, len(to))
	result := make([]string, 0, len(to))
	for _, raw := range to {
		parsed, err := mail.ParseAddress(strings.TrimSpace(raw))
		if err != nil {
			return nil, fmt.Errorf("invalid recipient address: %w", err)
		}
		key := strings.ToLower(parsed.Address)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, parsed.Address)
	}
	return result, nil
}

// buildMessage 组装 RFC 5322 邮件报文。
func buildMessage(from string, to []string, subject, body, contentType string, messageIDs ...string) string {
	if contentType == "" {
		contentType = "text/plain"
	}
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + strings.Join(to, ", ") + "\r\n")
	b.WriteString("Subject: " + mime.QEncoding.Encode("utf-8", subject) + "\r\n")
	b.WriteString("Date: " + time.Now().Format(time.RFC1123Z) + "\r\n")
	if len(messageIDs) > 0 && validMessageID(messageIDs[0]) {
		b.WriteString("Message-ID: " + messageIDs[0] + "\r\n")
	}
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: " + contentType + "; charset=\"utf-8\"\r\n")
	b.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	b.WriteString("\r\n")
	b.WriteString(normalizeCRLF(body))
	return b.String()
}

func buildMessageWithAttachments(
	from string,
	to []string,
	subject, body string,
	attachments []EmailAttachment,
	messageIDs ...string,
) (string, error) {
	if strings.TrimSpace(subject) == "" {
		return "", fmt.Errorf("email subject is required")
	}
	var content bytes.Buffer
	mixed := multipart.NewWriter(&content)

	writeHeaders(&content, from, to, subject, messageIDs...)
	content.WriteString("Content-Type: multipart/mixed; boundary=\"" + mixed.Boundary() + "\"\r\n\r\n")

	bodyHeader := make(textproto.MIMEHeader)
	bodyHeader.Set("Content-Type", `text/plain; charset="utf-8"`)
	bodyHeader.Set("Content-Transfer-Encoding", "8bit")
	bodyPart, err := mixed.CreatePart(bodyHeader)
	if err != nil {
		return "", fmt.Errorf("create email body part: %w", err)
	}
	if _, err := bodyPart.Write([]byte(normalizeCRLF(body))); err != nil {
		return "", fmt.Errorf("write email body part: %w", err)
	}

	for _, attachment := range attachments {
		filename := strings.TrimSpace(attachment.Filename)
		if filename == "" || strings.ContainsAny(filename, "\r\n") {
			return "", fmt.Errorf("attachment filename is required and must not contain newlines")
		}
		contentType := strings.TrimSpace(attachment.ContentType)
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		header := make(textproto.MIMEHeader)
		header.Set("Content-Type", contentType)
		header.Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
		header.Set("Content-Transfer-Encoding", "base64")
		part, err := mixed.CreatePart(header)
		if err != nil {
			return "", fmt.Errorf("create email attachment part: %w", err)
		}
		encoded := base64.StdEncoding.EncodeToString(attachment.Data)
		for len(encoded) > 76 {
			if _, err := part.Write([]byte(encoded[:76] + "\r\n")); err != nil {
				return "", fmt.Errorf("write email attachment: %w", err)
			}
			encoded = encoded[76:]
		}
		if _, err := part.Write([]byte(encoded + "\r\n")); err != nil {
			return "", fmt.Errorf("write email attachment: %w", err)
		}
	}
	if err := mixed.Close(); err != nil {
		return "", fmt.Errorf("finalize email attachments: %w", err)
	}
	return content.String(), nil
}

func writeHeaders(b *bytes.Buffer, from string, to []string, subject string, messageIDs ...string) {
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + strings.Join(to, ", ") + "\r\n")
	b.WriteString("Subject: " + mime.QEncoding.Encode("utf-8", subject) + "\r\n")
	b.WriteString("Date: " + time.Now().Format(time.RFC1123Z) + "\r\n")
	if len(messageIDs) > 0 && validMessageID(messageIDs[0]) {
		b.WriteString("Message-ID: " + messageIDs[0] + "\r\n")
	}
	b.WriteString("MIME-Version: 1.0\r\n")
}

func validMessageID(value string) bool {
	return strings.HasPrefix(value, "<") && strings.HasSuffix(value, ">") &&
		strings.Contains(value, "@") && !strings.ContainsAny(value, "\r\n")
}

func normalizeCRLF(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	return strings.ReplaceAll(value, "\n", "\r\n")
}
