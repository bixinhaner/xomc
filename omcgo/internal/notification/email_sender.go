package notification

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
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

// SMTPOptions 是 EmailSender 的传输配置。由上层（cmd/app/provider）从 appconfig
// 映射而来 —— notification 包以此与 appconfig 解耦，便于单测构造。
type SMTPOptions struct {
	Enabled  bool          // false 时 Send 直接返回错误（未配置 SMTP）
	Host     string        // SMTP 服务器地址
	Port     int           // SMTP 端口（25 / 465 / 587）
	Username string        // 留空 = 不做 SMTP AUTH
	Password string        // SMTP AUTH 口令
	From     string        // 发件人地址
	TLSMode  string        // none / starttls / implicit；空值兼容旧 StartTLS
	StartTLS bool          // Deprecated: TLSMode 为空时兼容旧配置
	Timeout  time.Duration // 连接超时（<=0 时取默认 10s）
}

const (
	SMTPTLSNone        = "none"
	SMTPTLSStartTLS    = "starttls"
	SMTPTLSImplicit    = "implicit"
	maxAttachmentBytes = 20 << 20
)

type EmailAttachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

const (
	EmailErrorConfiguration  = "configuration_error"
	EmailErrorAuthentication = "authentication_error"
	EmailErrorTLS            = "tls_error"
	EmailErrorConnection     = "connection_error"
	EmailErrorTemporary      = "provider_temporary"
	EmailErrorRecipient      = "recipient_rejected"
	EmailErrorMessage        = "message_rejected"
	EmailErrorUnknown        = "outcome_unknown"
)

type EmailSendError struct {
	Category       string
	Retryable      bool
	OutcomeUnknown bool
	Stage          string
	Err            error
}

func (e *EmailSendError) Error() string {
	return fmt.Sprintf("smtp %s: %v", e.Stage, e.Err)
}

func (e *EmailSendError) Unwrap() error { return e.Err }

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

// Send 为兼容既有调用逐个发送邮件。每个收件人使用独立 SMTP 信封，避免地址泄露和
// 批量投递中无法区分单个结果；可靠 Worker 直接调用 SendOne。
func (s *EmailSender) Send(ctx context.Context, to []string, subject, body string) error {
	if len(to) == 0 {
		return fmt.Errorf("no recipients")
	}
	for _, recipient := range to {
		if err := s.SendOne(ctx, recipient, subject, body); err != nil {
			return err
		}
	}
	return nil
}

// SendOne 在一个只含单个 RCPT TO 的 SMTP 信封中发送邮件。DATA 最终响应成功即视为
// accepted；之后的 QUIT 失败不改变服务端已经受理的事实。
func (s *EmailSender) SendOne(ctx context.Context, recipient, subject, body string) error {
	return s.sendOne(ctx, recipient, subject, body, nil)
}

// SendOneWithAttachments 使用单收件人 SMTP 信封发送带附件邮件。当前业务只允许一个
// KPI 报表附件，避免把发送器扩展成通用邮件客户端。
func (s *EmailSender) SendOneWithAttachments(
	ctx context.Context,
	recipient, subject, body string,
	attachments []EmailAttachment,
) error {
	if len(attachments) > 1 {
		return emailSendError(EmailErrorMessage, false, false, "message", errors.New("at most one attachment is supported"))
	}
	for _, attachment := range attachments {
		if attachment.Filename == "" || strings.ContainsAny(attachment.Filename, "\r\n") {
			return emailSendError(EmailErrorMessage, false, false, "message", errors.New("invalid attachment filename"))
		}
		if len(attachment.Data) > maxAttachmentBytes {
			return emailSendError(EmailErrorMessage, false, false, "message", fmt.Errorf("attachment exceeds %d bytes", maxAttachmentBytes))
		}
	}
	return s.sendOne(ctx, recipient, subject, body, attachments)
}

func (s *EmailSender) sendOne(
	ctx context.Context,
	recipient, subject, body string,
	attachments []EmailAttachment,
) error {
	if err := s.validateConnectionOptions(); err != nil {
		return err
	}
	if s.opts.From == "" {
		return emailSendError(EmailErrorConfiguration, false, false, "configuration", errors.New("from is required"))
	}
	if !plainEmailAddress(recipient) {
		return emailSendError(EmailErrorRecipient, false, false, "recipient", errors.New("invalid recipient address"))
	}
	if !plainEmailAddress(s.opts.From) {
		return emailSendError(EmailErrorConfiguration, false, false, "configuration", errors.New("invalid from address"))
	}
	if strings.ContainsAny(subject, "\r\n") {
		return emailSendError(EmailErrorMessage, false, false, "message", errors.New("subject contains a line break"))
	}
	client, err := s.connect(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	if err := client.Mail(s.opts.From); err != nil {
		return classifySMTPError("mail_from", err, EmailErrorConfiguration)
	}
	if err := client.Rcpt(recipient); err != nil {
		return classifySMTPError("recipient", err, EmailErrorRecipient)
	}
	w, err := client.Data()
	if err != nil {
		return classifySMTPError("data", err, EmailErrorMessage)
	}
	message, err := buildEmailMessage(s.opts.From, recipient, subject, body, attachments)
	if err != nil {
		_ = w.Close()
		return emailSendError(EmailErrorMessage, false, false, "message", err)
	}
	if _, err := w.Write(message); err != nil {
		_ = w.Close()
		return emailSendError(EmailErrorConnection, true, false, "write_body", err)
	}
	if err := w.Close(); err != nil {
		return emailSendError(EmailErrorUnknown, false, true, "finalize_data", err)
	}
	_ = client.Quit()
	return nil
}

// Verify performs connection, greeting, optional STARTTLS and authentication
// only. It intentionally sends no MAIL FROM, RCPT TO or DATA command.
func (s *EmailSender) Verify(ctx context.Context) error {
	if s.opts.From == "" || !plainEmailAddress(s.opts.From) {
		return emailSendError(EmailErrorConfiguration, false, false, "configuration", errors.New("valid from address is required"))
	}
	client, err := s.connect(ctx)
	if err != nil {
		return err
	}
	defer client.Close()
	_ = client.Quit()
	return nil
}

func (s *EmailSender) connect(ctx context.Context) (*smtp.Client, error) {
	if err := s.validateConnectionOptions(); err != nil {
		return nil, err
	}
	addr := net.JoinHostPort(s.opts.Host, strconv.Itoa(s.opts.Port))
	dialer := net.Dialer{Timeout: s.opts.Timeout}
	var conn net.Conn
	var err error
	if s.tlsMode() == SMTPTLSImplicit {
		tlsDialer := tls.Dialer{NetDialer: &dialer, Config: s.tlsConfig()}
		conn, err = tlsDialer.DialContext(ctx, "tcp", addr)
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		category := EmailErrorConnection
		stage := "connect"
		if s.tlsMode() == SMTPTLSImplicit {
			category, stage = EmailErrorTLS, "implicit_tls"
		}
		return nil, emailSendError(category, true, false, stage, err)
	}
	// 兜底超时，防止某个 SMTP 阶段无限挂起。
	deadline := time.Now().Add(s.opts.Timeout)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}
	_ = conn.SetDeadline(deadline)

	client, err := smtp.NewClient(conn, s.opts.Host)
	if err != nil {
		_ = conn.Close()
		return nil, classifySMTPError("handshake", err, EmailErrorConnection)
	}

	if s.tlsMode() == SMTPTLSStartTLS {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			_ = client.Close()
			return nil, emailSendError(EmailErrorTLS, false, false, "starttls", errors.New("server does not advertise STARTTLS"))
		}
		if err := client.StartTLS(s.tlsConfig()); err != nil {
			_ = client.Close()
			return nil, emailSendError(EmailErrorTLS, false, false, "starttls", err)
		}
	}
	if s.opts.Username != "" {
		auth := smtp.PlainAuth("", s.opts.Username, s.opts.Password, s.opts.Host)
		if err := client.Auth(auth); err != nil {
			_ = client.Close()
			return nil, emailSendError(EmailErrorAuthentication, false, false, "auth", err)
		}
	}
	return client, nil
}

func (s *EmailSender) validateConnectionOptions() error {
	if !s.opts.Enabled {
		return emailSendError(EmailErrorConfiguration, false, false, "configuration", errors.New("sender disabled"))
	}
	if s.opts.Host == "" {
		return emailSendError(EmailErrorConfiguration, false, false, "configuration", errors.New("host not configured"))
	}
	if s.opts.Port <= 0 {
		return emailSendError(EmailErrorConfiguration, false, false, "configuration", errors.New("port is required"))
	}
	if mode := s.tlsMode(); mode != SMTPTLSNone && mode != SMTPTLSStartTLS && mode != SMTPTLSImplicit {
		return emailSendError(EmailErrorConfiguration, false, false, "configuration", fmt.Errorf("unsupported tls mode %q", mode))
	}
	return nil
}

func (s *EmailSender) tlsMode() string {
	mode := strings.ToLower(strings.TrimSpace(s.opts.TLSMode))
	if mode == "" {
		if s.opts.StartTLS {
			return SMTPTLSStartTLS
		}
		return SMTPTLSNone
	}
	return mode
}

func (s *EmailSender) tlsConfig() *tls.Config {
	return &tls.Config{ServerName: s.opts.Host, MinVersion: tls.VersionTLS12}
}

func plainEmailAddress(value string) bool {
	parsed, err := mail.ParseAddress(value)
	return err == nil && parsed.Address == value
}

func classifySMTPError(stage string, err error, permanentCategory string) error {
	var protocolError *textproto.Error
	if errors.As(err, &protocolError) {
		if protocolError.Code >= 400 && protocolError.Code < 500 {
			return emailSendError(EmailErrorTemporary, true, false, stage, err)
		}
		return emailSendError(permanentCategory, false, false, stage, err)
	}
	return emailSendError(EmailErrorConnection, true, false, stage, err)
}

func emailSendError(category string, retryable, unknown bool, stage string, err error) error {
	return &EmailSendError{Category: category, Retryable: retryable, OutcomeUnknown: unknown, Stage: stage, Err: err}
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

func buildEmailMessage(from, to, subject, body string, attachments []EmailAttachment) ([]byte, error) {
	if len(attachments) == 0 {
		return []byte(buildMessage(from, []string{to}, subject, body)), nil
	}
	var result bytes.Buffer
	writer := multipart.NewWriter(&result)
	result.WriteString("From: " + from + "\r\n")
	result.WriteString("To: " + to + "\r\n")
	result.WriteString("Subject: " + mime.QEncoding.Encode("utf-8", subject) + "\r\n")
	result.WriteString("Date: " + time.Now().Format(time.RFC1123Z) + "\r\n")
	result.WriteString("MIME-Version: 1.0\r\n")
	result.WriteString("Content-Type: multipart/mixed; boundary=" + strconv.Quote(writer.Boundary()) + "\r\n\r\n")

	bodyHeader := textproto.MIMEHeader{}
	bodyHeader.Set("Content-Type", `text/plain; charset="utf-8"`)
	bodyHeader.Set("Content-Transfer-Encoding", "8bit")
	bodyPart, err := writer.CreatePart(bodyHeader)
	if err != nil {
		return nil, fmt.Errorf("create email body part: %w", err)
	}
	if _, err := bodyPart.Write([]byte(normalizeCRLF(body))); err != nil {
		return nil, fmt.Errorf("write email body part: %w", err)
	}

	attachment := attachments[0]
	contentType := strings.TrimSpace(attachment.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	attachmentHeader := textproto.MIMEHeader{}
	attachmentHeader.Set("Content-Type", mime.FormatMediaType(contentType, map[string]string{"name": attachment.Filename}))
	attachmentHeader.Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": attachment.Filename}))
	attachmentHeader.Set("Content-Transfer-Encoding", "base64")
	attachmentPart, err := writer.CreatePart(attachmentHeader)
	if err != nil {
		return nil, fmt.Errorf("create email attachment part: %w", err)
	}
	encoder := base64.NewEncoder(base64.StdEncoding, attachmentPart)
	if _, err := encoder.Write(attachment.Data); err != nil {
		return nil, fmt.Errorf("encode email attachment: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return nil, fmt.Errorf("finish email attachment: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("finish multipart email: %w", err)
	}
	return result.Bytes(), nil
}

// normalizeCRLF 把任意换行统一为 CRLF（SMTP DATA 要求）。
func normalizeCRLF(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(s, "\n", "\r\n")
}
