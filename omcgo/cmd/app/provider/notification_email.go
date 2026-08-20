package provider

import (
	"context"

	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/notification"
)

type notificationSMTPStore struct {
	repo admin.SysConfigRepository
}

func (s notificationSMTPStore) LoadSMTPSettings(ctx context.Context) (map[string]string, error) {
	rows, err := s.repo.List(ctx, notification.SMTPConfigCategory, false)
	if err != nil {
		return nil, err
	}
	values := make(map[string]string, len(rows))
	for _, row := range rows {
		values[row.Key] = row.Value
	}
	return values, nil
}

func smtpOptionsFromAppConfig(cfg appconfig.SMTPConfig) notification.SMTPOptions {
	cfg = cfg.Effective()
	options := notification.SMTPOptions{
		Enabled:      cfg.Enabled,
		Host:         cfg.Host,
		Port:         cfg.Port,
		SecurityMode: notification.SMTPSecurityMode(cfg.SecurityMode),
		AuthEnabled:  cfg.AuthEnabled,
		Username:     cfg.Username,
		Password:     cfg.Password,
		From:         cfg.From,
		Timeout:      cfg.Timeout,
	}
	return notification.WithLegacySMTPEnvironment(options)
}
