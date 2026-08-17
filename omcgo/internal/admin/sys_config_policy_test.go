package admin

import (
	"testing"

	"github.com/stretchr/testify/assert"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

func TestIsPublicSysConfig_UsesExactAllowlist(t *testing.T) {
	tests := []struct {
		name     string
		category string
		key      string
		want     bool
	}{
		{name: "system name", category: "system", key: "system_name", want: true},
		{name: "system version", category: "system", key: "system_version", want: true},
		{name: "menu icon switch", category: "system", key: "show_menu_icon", want: true},
		{name: "OMC name", category: "basic", key: "mrOMCName", want: true},
		{name: "browser password recording policy", category: "security", key: "isBrowserAutoRecordPass", want: true},
		{name: "login background", category: "ui_custom", key: "ui_login_background", want: true},
		{name: "expanded menu logo", category: "ui_custom", key: "ui_menu_logo_up", want: true},
		{name: "collapsed menu logo", category: "ui_custom", key: "ui_menu_logo_down", want: true},
		{name: "same key wrong category", category: "security", key: "system_name", want: false},
		{name: "unknown key", category: "system", key: "rogue_public", want: false},
		{name: "default password", category: "security", key: "defaultPasswd", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isPublicSysConfig(tt.category, tt.key))
		})
	}
}

func TestIsSecretSysConfig_UsesExactRegistry(t *testing.T) {
	tests := []struct {
		name     string
		category string
		key      string
		want     bool
	}{
		{name: "default password", category: "security", key: "defaultPasswd", want: true},
		{name: "Agent Studio token", category: "agent", key: "agent_studio_service_token", want: true},
		{name: "ACS upload password", category: "acs_transfer", key: "uploadPassword", want: true},
		{name: "ACS download password", category: "acs_transfer", key: "downloadPassword", want: true},
		{name: "SMTP password", category: "notification.email", key: "password", want: true},
		{name: "same password key wrong category", category: "basic", key: "defaultPasswd", want: false},
		{name: "ordinary security setting", category: "security", key: "passwordMinLength", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isSecretSysConfig(tt.category, tt.key))
		})
	}
}

func TestToSysConfigResponse_RedactsConfiguredSecret(t *testing.T) {
	got := toSysConfigResponse(SysConfig{
		Category: "security",
		Key:      "defaultPasswd",
		Value:    "OMC@123456",
	})

	assert.Empty(t, got.Value)
	assert.True(t, got.IsSecret)
	assert.True(t, got.IsConfigured)
	assert.False(t, got.IsPublic)
}

func TestToSysConfigResponse_RedactsUnconfiguredSecret(t *testing.T) {
	got := toSysConfigResponse(SysConfig{
		Category: "agent",
		Key:      "agent_studio_service_token",
	})

	assert.Empty(t, got.Value)
	assert.True(t, got.IsSecret)
	assert.False(t, got.IsConfigured)
}

func TestToSysConfigResponse_DerivesPublicFlagAndPreservesOrdinaryValue(t *testing.T) {
	public := toSysConfigResponse(SysConfig{
		Category: "system",
		Key:      "system_name",
		Value:    "OMC",
		IsPublic: false,
	})
	assert.Equal(t, "OMC", public.Value)
	assert.True(t, public.IsPublic)
	assert.False(t, public.IsSecret)

	rogue := toSysConfigResponse(SysConfig{
		Category: "system",
		Key:      "rogue_public",
		Value:    "must stay private",
		IsPublic: true,
	})
	assert.False(t, rogue.IsPublic)
}

func TestValidateGenericBatchWrite_RejectsAgentToken(t *testing.T) {
	err := validateGenericBatchWrite(BatchUpdateSysConfigRequest{
		Category: "agent",
		Items: []BatchItem{
			{Key: "agent_studio_service_token", Value: "secret"},
		},
	})

	assert.ErrorIs(t, err, commonerrors.ErrInvalidInput)
}

func TestValidateGenericBatchWrite_AllowsWriteOnlyACSPasswordRotation(t *testing.T) {
	for _, value := range []string{"new-secret", ""} {
		err := validateGenericBatchWrite(BatchUpdateSysConfigRequest{
			Category: "acs_transfer",
			Items: []BatchItem{
				{Key: "uploadPassword", Value: value},
				{Key: "downloadPassword", Value: value},
			},
		})
		assert.NoError(t, err)
	}
}

func TestPreserveBlankSecretsKeepsNonEmptyRotationAndDropsBlankPlaceholder(t *testing.T) {
	items := preserveBlankSecrets("acs_transfer", []BatchItem{
		{Key: "uploadPassword", Value: ""},
		{Key: "downloadPassword", Value: "rotated"},
		{Key: "uploadBaseURL", Value: "https://acs.example.com"},
	})

	assert.Equal(t, []BatchItem{
		{Key: "downloadPassword", Value: "rotated"},
		{Key: "uploadBaseURL", Value: "https://acs.example.com"},
	}, items)
}

func TestSMTPPasswordIsWriteOnlyAndBlankPreservesExistingValue(t *testing.T) {
	configured := toSysConfigResponse(SysConfig{
		Category: "notification.email",
		Key:      "password",
		Value:    "smtp-secret",
	})
	assert.Empty(t, configured.Value)
	assert.True(t, configured.IsSecret)
	assert.True(t, configured.IsConfigured)

	items := preserveBlankSecrets("notification.email", []BatchItem{
		{Key: "password", Value: ""},
		{Key: "host", Value: "smtp.example.test"},
	})
	assert.Equal(t, []BatchItem{{Key: "host", Value: "smtp.example.test"}}, items)
}

func TestValidateGenericBatchWrite_RejectsEmptyDefaultPassword(t *testing.T) {
	err := validateGenericBatchWrite(BatchUpdateSysConfigRequest{
		Category: "security",
		Items: []BatchItem{
			{Key: "defaultPasswd", Value: ""},
		},
	})

	assert.ErrorIs(t, err, commonerrors.ErrInvalidInput)
}

func TestValidateGenericBatchWrite_AllowsNonEmptyDefaultPassword(t *testing.T) {
	err := validateGenericBatchWrite(BatchUpdateSysConfigRequest{
		Category: "security",
		Items: []BatchItem{
			{Key: "defaultPasswd", Value: "new secret"},
		},
	})

	assert.NoError(t, err)
}
