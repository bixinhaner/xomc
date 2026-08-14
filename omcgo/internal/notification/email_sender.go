package notification

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/smtp"
	"net/textproto"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"go.uber.org/zap"
)

const defaultMaxAttachmentBytes int64 = 20 << 20

// SMTPTLSMode 明确 SMTP 建连方式，避免用两个布尔值组合出非法状态。
type SMTPTLSMode string

const (
	SMTPTLSImplicit SMTPTLSMode = "implicit"
	SMTPTLSStartTLS SMTPTLSMode = "starttls"
	SMTPTLSNone     SMTPTLSMode = "none"
)

// SMTPOptions 是 EmailSender 的传输配置。由上层从 appconfig 映射而来，
// notification 包不依赖启动配置包。
type SMTPOptions struct {
	Enabled            bool
	Host               string
	Port               int
	Username           string
	Password           string
	From               string
	TLSMode            SMTPTLSMode
	Timeout            time.Duration
	MaxAttachmentBytes int64
}

// EmailSender 是告警与 KPI 报表共用的 SMTP/MIME 传输实现。
type EmailSender struct {
	opts   SMTPOptions
	logger *zap.Logger
}

func NewEmailSender(opts SMTPOptions, logger *zap.Logger) *EmailSender {
	if logger == nil {
		logger = zap.NewNop()
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 10 * time.Second
	}
	if opts.MaxAttachmentBytes <= 0 {
		opts.MaxAttachmentBytes = defaultMaxAttachmentBytes
	}
	return &EmailSender{opts: opts, logger: logger.Named("email-sender")}
}

// Send 发送消息。密码、正文和附件内容不会写入错误或日志。
func (s *EmailSender) Send(ctx context.Context, message EmailMessage) error {
	if !s.opts.Enabled {
		return fmt.Errorf("email sender disabled (notification.smtp.enabled=false)")
	}
	if s.opts.Host == "" {
		return fmt.Errorf("smtp host not configured")
	}
	if err := validateMailbox("from", s.opts.From); err != nil {
		return err
	}
	if hasHeaderInjection(message.Subject) {
		return fmt.Errorf("invalid subject: header injection is not allowed")
	}
	recipients, err := normalizeRecipients(message.To)
	if err != nil {
		return err
	}
	message.To = recipients
	if err := s.validateAttachments(message.Attachments); err != nil {
		return err
	}

	addr := net.JoinHostPort(s.opts.Host, strconv.Itoa(s.opts.Port))
	conn, err := s.dial(ctx, addr)
	if err != nil {
		return fmt.Errorf("dial smtp %s: %w", addr, err)
	}
	_ = conn.SetDeadline(time.Now().Add(s.opts.Timeout))

	client, err := smtp.NewClient(conn, s.opts.Host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("smtp handshake: %w", err)
	}
	defer client.Close()

	if s.opts.TLSMode == SMTPTLSStartTLS {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return fmt.Errorf("smtp server does not advertise STARTTLS")
		}
		if err := client.StartTLS(&tls.Config{MinVersion: tls.VersionTLS12, ServerName: s.opts.Host}); err != nil {
			return fmt.Errorf("smtp STARTTLS: %w", err)
		}
	}
	if s.opts.Username != "" {
		auth := smtp.PlainAuth("", s.opts.Username, s.opts.Password, s.opts.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp AUTH: %w", err)
		}
	}
	if err := client.Mail(s.opts.From); err != nil {
		return fmt.Errorf("smtp MAIL FROM: %w", err)
	}
	for _, recipient := range message.To {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("smtp RCPT TO: %w", err)
		}
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp DATA: %w", err)
	}
	if err := writeMessage(ctx, w, s.opts.From, message, s.opts.MaxAttachmentBytes); err != nil {
		_ = w.Close()
		return fmt.Errorf("write MIME message: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("finalize smtp DATA: %w", err)
	}
	if err := client.Quit(); err != nil {
		// A successful DATA final response means the server accepted the
		// message. QUIT is connection cleanup only; reporting its failure as a
		// delivery failure would make the async job resend an accepted message.
		s.logger.Warn("smtp QUIT failed after message was accepted", zap.Error(err))
	}
	return nil
}

func (s *EmailSender) dial(ctx context.Context, addr string) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: s.opts.Timeout}
	switch s.opts.TLSMode {
	case SMTPTLSImplicit:
		tlsDialer := &tls.Dialer{
			NetDialer: dialer,
			Config:    &tls.Config{MinVersion: tls.VersionTLS12, ServerName: s.opts.Host},
		}
		return tlsDialer.DialContext(ctx, "tcp", addr)
	case SMTPTLSStartTLS, SMTPTLSNone, "":
		return dialer.DialContext(ctx, "tcp", addr)
	default:
		return nil, fmt.Errorf("unsupported tls mode %q", s.opts.TLSMode)
	}
}

func (s *EmailSender) validateAttachments(attachments []Attachment) error {
	var declared int64
	for _, attachment := range attachments {
		if attachment.Open == nil {
			return fmt.Errorf("attachment %q has no content opener", attachment.Filename)
		}
		if attachment.Filename == "" || filepath.Base(attachment.Filename) != attachment.Filename || hasHeaderInjection(attachment.Filename) {
			return fmt.Errorf("invalid attachment filename")
		}
		if attachment.Size < -1 {
			return fmt.Errorf("attachment %q has invalid size", attachment.Filename)
		}
		if attachment.Size > 0 {
			declared += attachment.Size
			if declared > s.opts.MaxAttachmentBytes {
				return fmt.Errorf("attachments exceed configured size limit of %d bytes", s.opts.MaxAttachmentBytes)
			}
		}
		if attachment.ContentType != "" {
			if _, _, err := mime.ParseMediaType(attachment.ContentType); err != nil {
				return fmt.Errorf("attachment %q has invalid content type", attachment.Filename)
			}
		}
	}
	return nil
}

func writeMessage(ctx context.Context, w io.Writer, from string, message EmailMessage, maxAttachmentBytes int64) error {
	if _, err := io.WriteString(w, messageHeaders(from, message.To, message.Subject)); err != nil {
		return fmt.Errorf("write headers: %w", err)
	}
	if len(message.Attachments) == 0 {
		_, err := io.WriteString(w, "Content-Type: text/plain; charset=\"utf-8\"\r\nContent-Transfer-Encoding: 8bit\r\n\r\n"+normalizeCRLF(message.TextBody))
		return err
	}

	mw := multipart.NewWriter(w)
	if _, err := io.WriteString(w, "Content-Type: multipart/mixed; boundary=\""+mw.Boundary()+"\"\r\n\r\n"); err != nil {
		return fmt.Errorf("write multipart header: %w", err)
	}
	textHeader := make(textproto.MIMEHeader)
	textHeader.Set("Content-Type", "text/plain; charset=\"utf-8\"")
	textHeader.Set("Content-Transfer-Encoding", "8bit")
	textPart, err := mw.CreatePart(textHeader)
	if err != nil {
		return fmt.Errorf("create text part: %w", err)
	}
	if _, err := io.WriteString(textPart, normalizeCRLF(message.TextBody)); err != nil {
		return fmt.Errorf("write text part: %w", err)
	}

	var total int64
	for _, attachment := range message.Attachments {
		contentType := attachment.ContentType
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		header := make(textproto.MIMEHeader)
		header.Set("Content-Type", contentType)
		header.Set("Content-Disposition", attachmentDisposition(attachment.Filename))
		header.Set("Content-Transfer-Encoding", "base64")
		part, err := mw.CreatePart(header)
		if err != nil {
			return fmt.Errorf("create attachment %q: %w", attachment.Filename, err)
		}
		r, err := attachment.Open(ctx)
		if err != nil {
			return fmt.Errorf("open attachment %q: %w", attachment.Filename, err)
		}
		remaining := maxAttachmentBytes - total
		encoded := base64.NewEncoder(base64.StdEncoding, &base64LineWriter{w: part, remaining: 76})
		n, copyErr := io.Copy(encoded, io.LimitReader(&contextReader{ctx: ctx, r: r}, remaining+1))
		closeErr := r.Close()
		encodeCloseErr := encoded.Close()
		total += n
		if copyErr != nil {
			return fmt.Errorf("read attachment %q: %w", attachment.Filename, copyErr)
		}
		if closeErr != nil {
			return fmt.Errorf("close attachment %q: %w", attachment.Filename, closeErr)
		}
		if encodeCloseErr != nil {
			return fmt.Errorf("encode attachment %q: %w", attachment.Filename, encodeCloseErr)
		}
		if total > maxAttachmentBytes {
			return fmt.Errorf("attachments exceed configured size limit of %d bytes", maxAttachmentBytes)
		}
		if _, err := io.WriteString(part, "\r\n"); err != nil {
			return fmt.Errorf("finish attachment %q: %w", attachment.Filename, err)
		}
	}
	if err := mw.Close(); err != nil {
		return fmt.Errorf("close multipart message: %w", err)
	}
	return nil
}

func messageHeaders(from string, to []string, subject string) string {
	return "From: " + from + "\r\n" +
		"To: " + strings.Join(to, ", ") + "\r\n" +
		"Subject: " + mime.QEncoding.Encode("utf-8", subject) + "\r\n" +
		"Date: " + time.Now().Format(time.RFC1123Z) + "\r\n" +
		"MIME-Version: 1.0\r\n"
}

// buildMessage 保留为纯文本消息构造辅助，供单测和无附件路径核对。
func buildMessage(from string, to []string, subject, body string) string {
	return messageHeaders(from, to, subject) +
		"Content-Type: text/plain; charset=\"utf-8\"\r\n" +
		"Content-Transfer-Encoding: 8bit\r\n\r\n" + normalizeCRLF(body)
}

func attachmentDisposition(filename string) string {
	fallback := asciiFilename(filename)
	encoded := strings.ReplaceAll(url.PathEscape(filename), "+", "%20")
	return fmt.Sprintf("attachment; filename=\"%s\"; filename*=UTF-8''%s", fallback, encoded)
}

func asciiFilename(filename string) string {
	var b strings.Builder
	for _, r := range filename {
		if r <= unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune("._-", r)) {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "attachment"
	}
	return b.String()
}

type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (r *contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.r.Read(p)
}

type base64LineWriter struct {
	w         io.Writer
	remaining int
}

func (w *base64LineWriter) Write(p []byte) (int, error) {
	written := 0
	for len(p) > 0 {
		if w.remaining == 0 {
			if _, err := io.WriteString(w.w, "\r\n"); err != nil {
				return written, err
			}
			w.remaining = 76
		}
		n := len(p)
		if n > w.remaining {
			n = w.remaining
		}
		count, err := w.w.Write(p[:n])
		written += count
		w.remaining -= count
		p = p[count:]
		if err != nil {
			return written, err
		}
		if count != n {
			return written, io.ErrShortWrite
		}
	}
	return written, nil
}

func normalizeCRLF(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	return strings.ReplaceAll(value, "\n", "\r\n")
}
