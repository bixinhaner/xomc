package main

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/notification"
)

func TestSMTPOptionsFromWorkerConfigUsesLegacyEnvironmentFallback(t *testing.T) {
	t.Setenv("OMC_SMTP_HOST", "smtp.legacy.example")
	t.Setenv("OMC_SMTP_PORT", "587")
	t.Setenv("OMC_SMTP_USERNAME", "omc@example.test")
	t.Setenv("OMC_SMTP_PASSWORD", "secret")
	t.Setenv("OMC_SMTP_FROM", "omc@example.test")
	t.Setenv("OMC_SMTP_USE_STARTTLS", "true")

	options := smtpOptionsFromWorkerConfig(appconfig.SMTPConfig{})
	require.True(t, options.Enabled)
	require.Equal(t, "smtp.legacy.example", options.Host)
	require.Equal(t, 587, options.Port)
	require.True(t, options.AuthEnabled)
	require.Equal(t, notification.SMTPSecuritySTARTTLS, options.SecurityMode)
}

func TestSMTPOptionsFromWorkerConfigPreservesLegacyYAML(t *testing.T) {
	options := smtpOptionsFromWorkerConfig(appconfig.SMTPConfig{
		Enabled: true, Host: "smtp.yaml.example", Port: 587,
		Username: "legacy-user", Password: "secret", From: "omc@example.test", StartTLS: true,
	})

	require.True(t, options.AuthEnabled)
	require.Equal(t, notification.SMTPSecuritySTARTTLS, options.SecurityMode)
}
