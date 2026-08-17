package notification

import (
	"context"
	"fmt"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

const (
	SMTPConfigCategory = "notification.email"

	SMTPKeyEnabled        = "enabled"
	SMTPKeyHost           = "host"
	SMTPKeyPort           = "port"
	SMTPKeySecurityMode   = "security_mode"
	SMTPKeyAuthEnabled    = "auth_enabled"
	SMTPKeyUsername       = "username"
	SMTPKeyPassword       = "password"
	SMTPKeyFromAddress    = "from_address"
	SMTPKeyFromName       = "from_name"
	SMTPKeyTimeoutSeconds = "timeout_seconds"
)

var smtpSettingKeys = map[string]struct{}{
	SMTPKeyEnabled:        {},
	SMTPKeyHost:           {},
	SMTPKeyPort:           {},
	SMTPKeySecurityMode:   {},
	SMTPKeyAuthEnabled:    {},
	SMTPKeyUsername:       {},
	SMTPKeyPassword:       {},
	SMTPKeyFromAddress:    {},
	SMTPKeyFromName:       {},
	SMTPKeyTimeoutSeconds: {},
}

// SMTPSettingsStore is the narrow persistence boundary used by the mail
// transport. Implementations return one logical OMC's shared settings.
type SMTPSettingsStore interface {
	LoadSMTPSettings(ctx context.Context) (map[string]string, error)
}

// DynamicEmailSender loads the shared SMTP settings for every delivery. Email
// volume is low and this deliberately avoids stale per-process configuration in
// multi-node deployments.
type DynamicEmailSender struct {
	store    SMTPSettingsStore
	fallback SMTPOptions
	logger   *zap.Logger
}

func NewDynamicEmailSender(store SMTPSettingsStore, fallback SMTPOptions, logger *zap.Logger) *DynamicEmailSender {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &DynamicEmailSender{store: store, fallback: fallback, logger: logger}
}

func (s *DynamicEmailSender) Send(ctx context.Context, to []string, subject, body string) error {
	opts, err := LoadSMTPOptions(ctx, s.store, s.fallback)
	if err != nil {
		return err
	}
	return NewEmailSender(opts, s.logger).Send(ctx, to, subject, body)
}

func (s *DynamicEmailSender) SendHTML(ctx context.Context, to []string, subject, body string) error {
	opts, err := LoadSMTPOptions(ctx, s.store, s.fallback)
	if err != nil {
		return err
	}
	return NewEmailSender(opts, s.logger).SendHTML(ctx, to, subject, body)
}

func (s *DynamicEmailSender) SendHTMLWithMessageID(ctx context.Context, to []string, subject, body, messageID string) error {
	opts, err := LoadSMTPOptions(ctx, s.store, s.fallback)
	if err != nil {
		return err
	}
	return NewEmailSender(opts, s.logger).SendHTMLWithMessageID(ctx, to, subject, body, messageID)
}

func (s *DynamicEmailSender) SendWithAttachments(
	ctx context.Context,
	to []string,
	subject, body string,
	attachments []EmailAttachment,
) error {
	opts, err := LoadSMTPOptions(ctx, s.store, s.fallback)
	if err != nil {
		return err
	}
	return NewEmailSender(opts, s.logger).SendWithAttachments(ctx, to, subject, body, attachments)
}

func (s *DynamicEmailSender) SendWithAttachmentsMessageID(
	ctx context.Context,
	to []string,
	subject, body string,
	attachments []EmailAttachment,
	messageID string,
) error {
	opts, err := LoadSMTPOptions(ctx, s.store, s.fallback)
	if err != nil {
		return err
	}
	return NewEmailSender(opts, s.logger).SendWithAttachmentsMessageID(ctx, to, subject, body, attachments, messageID)
}

// LoadSMTPOptions merges persisted settings over the process fallback. The
// fallback keeps existing deployments bootable; once the category exists in
// sys_configs it is the canonical source shared by app and worker nodes.
func LoadSMTPOptions(ctx context.Context, store SMTPSettingsStore, fallback SMTPOptions) (SMTPOptions, error) {
	if store == nil {
		return fallback, nil
	}
	values, err := store.LoadSMTPSettings(ctx)
	if err != nil {
		return SMTPOptions{}, fmt.Errorf("load smtp settings: %w", err)
	}
	if len(values) == 0 {
		return fallback, nil
	}

	opts := fallback
	if raw, ok := values[SMTPKeyEnabled]; ok {
		opts.Enabled, err = strconv.ParseBool(raw)
		if err != nil {
			return SMTPOptions{}, fmt.Errorf("parse smtp enabled: %w", err)
		}
	}
	if raw, ok := values[SMTPKeyHost]; ok {
		opts.Host = strings.TrimSpace(raw)
	}
	if raw, ok := values[SMTPKeyPort]; ok {
		opts.Port, err = strconv.Atoi(raw)
		if err != nil {
			return SMTPOptions{}, fmt.Errorf("parse smtp port: %w", err)
		}
	}
	if raw, ok := values[SMTPKeySecurityMode]; ok {
		opts.SecurityMode = SMTPSecurityMode(raw)
	}
	if raw, ok := values[SMTPKeyAuthEnabled]; ok {
		opts.AuthEnabled, err = strconv.ParseBool(raw)
		if err != nil {
			return SMTPOptions{}, fmt.Errorf("parse smtp auth enabled: %w", err)
		}
	}
	if raw, ok := values[SMTPKeyUsername]; ok {
		opts.Username = strings.TrimSpace(raw)
	}
	// A blank persisted password means "not configured" for a fresh install.
	// During rotation the generic sys_config endpoint removes blank password
	// updates before persistence, so an existing secret is retained.
	if raw, ok := values[SMTPKeyPassword]; ok && raw != "" {
		opts.Password = raw
	}
	if raw, ok := values[SMTPKeyFromAddress]; ok {
		fromAddress := strings.TrimSpace(raw)
		fromName := strings.TrimSpace(values[SMTPKeyFromName])
		if fromName == "" {
			opts.From = fromAddress
		} else {
			opts.From = (&mail.Address{Name: fromName, Address: fromAddress}).String()
		}
	}
	if raw, ok := values[SMTPKeyTimeoutSeconds]; ok {
		seconds, parseErr := strconv.Atoi(raw)
		if parseErr != nil {
			return SMTPOptions{}, fmt.Errorf("parse smtp timeout: %w", parseErr)
		}
		opts.Timeout = time.Duration(seconds) * time.Second
	}
	return opts, nil
}

// ValidateSMTPSettings validates the complete merged sys_config category.
func ValidateSMTPSettings(values map[string]string) error {
	for key := range values {
		if _, ok := smtpSettingKeys[key]; !ok {
			return fmt.Errorf("unsupported SMTP setting %q", key)
		}
	}

	enabled, err := parseRequiredBool(values, SMTPKeyEnabled)
	if err != nil {
		return err
	}
	authEnabled, err := parseRequiredBool(values, SMTPKeyAuthEnabled)
	if err != nil {
		return err
	}
	securityMode := SMTPSecurityMode(values[SMTPKeySecurityMode])
	switch securityMode {
	case SMTPSecurityNone, SMTPSecuritySTARTTLS, SMTPSecurityImplicitTLS:
	default:
		return fmt.Errorf("security_mode must be none, starttls, or implicit_tls")
	}
	port, err := parseBoundedInt(values, SMTPKeyPort, 1, 65535)
	if err != nil {
		return err
	}
	_ = port
	if _, err := parseBoundedInt(values, SMTPKeyTimeoutSeconds, 1, 120); err != nil {
		return err
	}
	if !enabled {
		return nil
	}
	if strings.TrimSpace(values[SMTPKeyHost]) == "" {
		return fmt.Errorf("host is required when email is enabled")
	}
	if _, err := mail.ParseAddress(strings.TrimSpace(values[SMTPKeyFromAddress])); err != nil {
		return fmt.Errorf("from_address is invalid: %w", err)
	}
	if authEnabled {
		if securityMode == SMTPSecurityNone {
			return fmt.Errorf("authentication requires starttls or implicit_tls")
		}
		if strings.TrimSpace(values[SMTPKeyUsername]) == "" {
			return fmt.Errorf("username is required when authentication is enabled")
		}
		if values[SMTPKeyPassword] == "" {
			return fmt.Errorf("password is required when authentication is enabled")
		}
	}
	return nil
}

func parseRequiredBool(values map[string]string, key string) (bool, error) {
	raw, ok := values[key]
	if !ok {
		return false, fmt.Errorf("%s is required", key)
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%s must be true or false", key)
	}
	return value, nil
}

func parseBoundedInt(values map[string]string, key string, minValue, maxValue int) (int, error) {
	raw, ok := values[key]
	if !ok {
		return 0, fmt.Errorf("%s is required", key)
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < minValue || value > maxValue {
		return 0, fmt.Errorf("%s must be between %d and %d", key, minValue, maxValue)
	}
	return value, nil
}
