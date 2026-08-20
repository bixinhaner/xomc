package notification

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubSMTPSettingsStore struct {
	values map[string]string
	err    error
}

func (s stubSMTPSettingsStore) LoadSMTPSettings(context.Context) (map[string]string, error) {
	return s.values, s.err
}

func completeSMTPSettings() map[string]string {
	return map[string]string{
		SMTPKeyEnabled:        "true",
		SMTPKeyHost:           "smtp.example.test",
		SMTPKeyPort:           "587",
		SMTPKeySecurityMode:   "starttls",
		SMTPKeyAuthEnabled:    "true",
		SMTPKeyUsername:       "omc@example.test",
		SMTPKeyPassword:       "secret",
		SMTPKeyFromAddress:    "omc@example.test",
		SMTPKeyFromName:       "Test OMC",
		SMTPKeyTimeoutSeconds: "15",
	}
}

func TestValidateSMTPSettings(t *testing.T) {
	t.Parallel()

	t.Run("complete authenticated config", func(t *testing.T) {
		require.NoError(t, ValidateSMTPSettings(completeSMTPSettings()))
	})

	t.Run("internal relay without auth", func(t *testing.T) {
		values := completeSMTPSettings()
		values[SMTPKeyPort] = "25"
		values[SMTPKeySecurityMode] = "none"
		values[SMTPKeyAuthEnabled] = "false"
		values[SMTPKeyUsername] = ""
		values[SMTPKeyPassword] = ""
		require.NoError(t, ValidateSMTPSettings(values))
	})

	t.Run("disabled channel may be incomplete", func(t *testing.T) {
		values := completeSMTPSettings()
		values[SMTPKeyEnabled] = "false"
		values[SMTPKeyHost] = ""
		values[SMTPKeyFromAddress] = ""
		values[SMTPKeyPassword] = ""
		require.NoError(t, ValidateSMTPSettings(values))
	})

	t.Run("auth requires password", func(t *testing.T) {
		values := completeSMTPSettings()
		values[SMTPKeyPassword] = ""
		err := ValidateSMTPSettings(values)
		require.ErrorContains(t, err, "password is required")
	})

	t.Run("plaintext authentication is rejected", func(t *testing.T) {
		values := completeSMTPSettings()
		values[SMTPKeySecurityMode] = "none"
		err := ValidateSMTPSettings(values)
		require.ErrorContains(t, err, "requires starttls or implicit_tls")
	})

	t.Run("unknown key rejected", func(t *testing.T) {
		values := completeSMTPSettings()
		values["provider"] = "vendor_specific"
		err := ValidateSMTPSettings(values)
		require.ErrorContains(t, err, "unsupported SMTP setting")
	})
}

func TestLoadSMTPOptions(t *testing.T) {
	t.Parallel()

	fallback := SMTPOptions{Password: "fallback-secret", Timeout: 10 * time.Second}
	opts, err := LoadSMTPOptions(context.Background(), stubSMTPSettingsStore{values: completeSMTPSettings()}, fallback)
	require.NoError(t, err)
	assert.True(t, opts.Enabled)
	assert.Equal(t, "smtp.example.test", opts.Host)
	assert.Equal(t, 587, opts.Port)
	assert.Equal(t, SMTPSecuritySTARTTLS, opts.SecurityMode)
	assert.True(t, opts.AuthEnabled)
	assert.Equal(t, "secret", opts.Password)
	assert.Equal(t, `"Test OMC" <omc@example.test>`, opts.From)
	assert.Equal(t, 15*time.Second, opts.Timeout)
}

func TestLoadSMTPOptions_EmptyStoreUsesFallback(t *testing.T) {
	t.Parallel()

	fallback := SMTPOptions{Enabled: true, Host: "fallback.test", Port: 25}
	opts, err := LoadSMTPOptions(context.Background(), stubSMTPSettingsStore{}, fallback)
	require.NoError(t, err)
	assert.Equal(t, fallback, opts)
}

func TestLoadSMTPOptions_StoreFailureWrapped(t *testing.T) {
	t.Parallel()

	_, err := LoadSMTPOptions(context.Background(), stubSMTPSettingsStore{err: errors.New("db unavailable")}, SMTPOptions{})
	require.ErrorContains(t, err, "load smtp settings")
}
