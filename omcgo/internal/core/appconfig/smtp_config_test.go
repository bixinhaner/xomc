package appconfig

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSMTPConfigEffectivePreservesLegacyYAMLSemantics(t *testing.T) {
	t.Parallel()

	effective := (SMTPConfig{
		Host: "smtp.example.test", Username: "legacy-user", StartTLS: true,
	}).Effective()

	require.Equal(t, "starttls", effective.SecurityMode)
	require.True(t, effective.AuthEnabled)
}

func TestSMTPConfigEffectiveKeepsExplicitSecurityMode(t *testing.T) {
	t.Parallel()

	effective := (SMTPConfig{
		SecurityMode: "implicit_tls", StartTLS: true,
	}).Effective()

	require.Equal(t, "implicit_tls", effective.SecurityMode)
}
