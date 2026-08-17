package provider

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/notification"
)

func TestSMTPOptionsFromAppConfigUsesLegacyEnvOnlyAsFallback(t *testing.T) {
	t.Setenv("OMC_SMTP_HOST", "smtp.legacy.example")
	t.Setenv("OMC_SMTP_PORT", "587")
	t.Setenv("OMC_SMTP_USERNAME", "omc@example.com")
	t.Setenv("OMC_SMTP_PASSWORD", "secret")
	t.Setenv("OMC_SMTP_FROM", "omc@example.com")
	t.Setenv("OMC_SMTP_USE_STARTTLS", "true")

	options := smtpOptionsFromAppConfig(appconfig.SMTPConfig{})
	require.True(t, options.Enabled)
	require.Equal(t, "smtp.legacy.example", options.Host)
	require.Equal(t, 587, options.Port)
	require.True(t, options.AuthEnabled)
	require.Equal(t, notification.SMTPSecuritySTARTTLS, options.SecurityMode)
}

func TestSMTPOptionsFromAppConfigWinsOverLegacyEnv(t *testing.T) {
	t.Setenv("OMC_SMTP_HOST", "smtp.legacy.example")
	options := smtpOptionsFromAppConfig(appconfig.SMTPConfig{
		Enabled:      true,
		Host:         "smtp.current.example",
		Port:         465,
		SecurityMode: string(notification.SMTPSecurityImplicitTLS),
	})
	require.Equal(t, "smtp.current.example", options.Host)
	require.Equal(t, 465, options.Port)
	require.Equal(t, notification.SMTPSecurityImplicitTLS, options.SecurityMode)
}
