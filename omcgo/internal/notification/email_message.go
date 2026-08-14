package notification

import (
	"context"
	"fmt"
	"io"
	"net/mail"
	"strings"
)

// EmailMessage 是业务层可提交给共享邮件传输的完整消息。
// 发件人由部署配置统一控制，调用方不能覆盖 From/Bcc 等敏感邮件头。
type EmailMessage struct {
	To          []string
	Subject     string
	TextBody    string
	Attachments []Attachment
}

// Attachment 描述延迟打开、可流式读取的附件。
// Size 小于 0 表示未知；发送时仍会按 SMTPOptions.MaxAttachmentBytes 限流。
type Attachment struct {
	Filename    string
	ContentType string
	Size        int64
	Open        func(context.Context) (io.ReadCloser, error)
}

// EmailTransport 是告警、KPI 报表和 Alertmanager 共用的邮件传输边界。
type EmailTransport interface {
	Send(context.Context, EmailMessage) error
}

func normalizeRecipients(recipients []string) ([]string, error) {
	out := make([]string, 0, len(recipients))
	seen := make(map[string]struct{}, len(recipients))
	for _, raw := range recipients {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		if hasHeaderInjection(raw) {
			return nil, fmt.Errorf("invalid recipient address")
		}
		addr, err := mail.ParseAddress(raw)
		if err != nil || addr.Address != raw {
			return nil, fmt.Errorf("invalid recipient address %q", raw)
		}
		key := strings.ToLower(addr.Address)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, addr.Address)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no recipients")
	}
	return out, nil
}

// NormalizeRecipients 校验、规范化并按地址大小写不敏感去重，供订阅 API 复用。
func NormalizeRecipients(recipients []string) ([]string, error) {
	return normalizeRecipients(recipients)
}

func validateMailbox(field, value string) error {
	if hasHeaderInjection(value) {
		return fmt.Errorf("invalid %s address", field)
	}
	addr, err := mail.ParseAddress(value)
	if err != nil || addr.Address != value {
		return fmt.Errorf("invalid %s address", field)
	}
	return nil
}

func hasHeaderInjection(value string) bool {
	return strings.ContainsAny(value, "\r\n")
}
