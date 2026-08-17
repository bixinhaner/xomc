package admin

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

var publicSysConfigs = map[validatorKey]struct{}{
	{Category: "system", Key: "system_name"}:               {},
	{Category: "system", Key: "system_version"}:            {},
	{Category: "system", Key: "show_menu_icon"}:            {},
	{Category: "basic", Key: "mrOMCName"}:                  {},
	{Category: "security", Key: "isBrowserAutoRecordPass"}: {},
	{Category: "ui_custom", Key: "ui_login_background"}:    {},
	{Category: "ui_custom", Key: "ui_menu_logo_up"}:        {},
	{Category: "ui_custom", Key: "ui_menu_logo_down"}:      {},
}

var secretSysConfigs = map[validatorKey]struct{}{
	{Category: "security", Key: "defaultPasswd"}:           {},
	{Category: "agent", Key: "agent_studio_service_token"}: {},
	{Category: "acs_transfer", Key: "uploadPassword"}:      {},
	{Category: "acs_transfer", Key: "downloadPassword"}:    {},
	{Category: "notification.email", Key: "password"}:      {},
}

func isPublicSysConfig(category, key string) bool {
	_, ok := publicSysConfigs[validatorKey{Category: category, Key: key}]
	return ok
}

func isSecretSysConfig(category, key string) bool {
	_, ok := secretSysConfigs[validatorKey{Category: category, Key: key}]
	return ok
}

func isPreserveOnBlankSecret(category, key string) bool {
	return (category == "acs_transfer" && (key == "uploadPassword" || key == "downloadPassword")) ||
		(category == "notification.email" && key == "password")
}

func preserveBlankSecrets(category string, items []BatchItem) []BatchItem {
	filtered := make([]BatchItem, 0, len(items))
	for _, item := range items {
		if isPreserveOnBlankSecret(category, item.Key) && item.Value == "" {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

// validateGenericBatchWrite protects the generic system-config endpoint.
// The default password remains writable here for the security settings page,
// while Agent tokens must use the dedicated Agent configuration service.
func validateGenericBatchWrite(req BatchUpdateSysConfigRequest) error {
	for _, item := range req.Items {
		if !isSecretSysConfig(req.Category, item.Key) {
			continue
		}
		if req.Category == "security" && item.Key == "defaultPasswd" && item.Value != "" {
			continue
		}
		// Historical ACS credentials are write-only: a non-empty value rotates
		// the secret, while an empty value is filtered before persistence and
		// therefore preserves the existing credential.
		if isPreserveOnBlankSecret(req.Category, item.Key) {
			continue
		}
		return fmt.Errorf("%w: generic sys_config write is not allowed for %s.%s",
			commonerrors.ErrInvalidInput, req.Category, item.Key)
	}
	return nil
}

// SysConfigResponse is the only HTTP representation of a system configuration.
// Secret values are never copied to Value; callers only learn whether one is configured.
type SysConfigResponse struct {
	ID           uuid.UUID `json:"id"`
	Category     string    `json:"category"`
	Key          string    `json:"key"`
	Value        string    `json:"value"`
	ValueType    string    `json:"value_type"`
	Description  string    `json:"desc"`
	IsPublic     bool      `json:"is_public"`
	IsSecret     bool      `json:"is_secret"`
	IsConfigured bool      `json:"is_configured"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func toSysConfigResponse(cfg SysConfig) SysConfigResponse {
	isSecret := isSecretSysConfig(cfg.Category, cfg.Key)
	value := cfg.Value
	if isSecret {
		value = ""
	}

	return SysConfigResponse{
		ID:           cfg.ID,
		Category:     cfg.Category,
		Key:          cfg.Key,
		Value:        value,
		ValueType:    cfg.ValueType,
		Description:  cfg.Description,
		IsPublic:     isPublicSysConfig(cfg.Category, cfg.Key),
		IsSecret:     isSecret,
		IsConfigured: isSecret && cfg.Value != "",
		CreatedAt:    cfg.CreatedAt,
		UpdatedAt:    cfg.UpdatedAt,
	}
}

func toSysConfigResponses(configs []SysConfig) []SysConfigResponse {
	responses := make([]SysConfigResponse, 0, len(configs))
	for _, cfg := range configs {
		responses = append(responses, toSysConfigResponse(cfg))
	}
	return responses
}
