package provider

import (
	"github.com/omcgo/omcgo/internal/notification"
	"go.uber.org/zap"
)

// initNotificationEmailTransport 在业务模块之前构造唯一的 SMTP/MIME 传输实例。
func initNotificationEmailTransport(c *Container) error {
	cfg := c.Cfg.Notification.SMTP
	c.EmailTransport = notification.NewEmailSender(notification.SMTPOptions{
		Enabled:            cfg.Enabled,
		Host:               cfg.Host,
		Port:               cfg.Port,
		Username:           cfg.Username,
		Password:           cfg.Password,
		From:               cfg.From,
		TLSMode:            notification.SMTPTLSMode(cfg.TLSMode),
		Timeout:            cfg.Timeout,
		MaxAttachmentBytes: cfg.MaxAttachmentBytes,
	}, c.Logger)
	c.Logger.Info("email transport initialized",
		zap.Bool("smtp_enabled", cfg.Enabled),
		zap.String("smtp_host", cfg.Host),
		zap.Int("smtp_port", cfg.Port),
		zap.String("smtp_tls_mode", cfg.TLSMode))
	return nil
}
