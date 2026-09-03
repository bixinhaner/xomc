package agentconfig

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/admin"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

type ConfigReader interface {
	List(ctx context.Context, category string, publicOnly bool) ([]admin.SysConfig, error)
}

type ConfigWriter interface {
	BatchUpsert(ctx context.Context, req admin.BatchUpdateSysConfigRequest) (int, error)
}

type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type Service struct {
	reader ConfigReader
	writer ConfigWriter
	http   HTTPDoer
	now    func() time.Time
	logger *zap.Logger
}

func NewService(reader ConfigReader, writer ConfigWriter, httpClient HTTPDoer, logger *zap.Logger) *Service {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{
		reader: reader,
		writer: writer,
		http:   httpClient,
		now:    time.Now,
		logger: logger.Named("agentconfig"),
	}
}

func (s *Service) GetAdminConfig(ctx context.Context) (*AdminConfig, error) {
	values, err := s.loadValues(ctx)
	if err != nil {
		return nil, err
	}
	return adminConfigFromValues(values), nil
}

func (s *Service) GetRuntimeConfig(ctx context.Context) (*RuntimeConfig, error) {
	cfg, err := s.GetAdminConfig(ctx)
	if err != nil {
		return nil, err
	}
	enabled := cfg.Enabled && cfg.Status == StatusConnected && cfg.ConnectorID != "" && cfg.AgentStudioBaseURL != ""
	return &RuntimeConfig{
		Visible:          cfg.Enabled,
		Enabled:          enabled,
		Endpoint:         DefaultRuntimeStreamPath,
		ConnectorID:      cfg.ConnectorID,
		Status:           cfg.Status,
		LastValidatedAt:  cfg.LastValidatedAt,
		LastError:        cfg.LastError,
		ConfiguredSource: "server",
		Policy:           cfg.Policy,
	}, nil
}

func (s *Service) GetVisibilityConfig(ctx context.Context) (*VisibilityConfig, error) {
	cfg, err := s.GetAdminConfig(ctx)
	if err != nil {
		return nil, err
	}
	enabled := cfg.Enabled && cfg.Status == StatusConnected && cfg.ConnectorID != "" && cfg.AgentStudioBaseURL != ""
	return &VisibilityConfig{
		Visible:         cfg.Enabled,
		Enabled:         enabled,
		Status:          cfg.Status,
		LastValidatedAt: cfg.LastValidatedAt,
		LastError:       cfg.LastError,
	}, nil
}

func (s *Service) GetRuntimeTarget(ctx context.Context) (*RuntimeTarget, error) {
	values, err := s.loadValues(ctx)
	if err != nil {
		return nil, err
	}
	cfg := adminConfigFromValues(values)
	instanceName, instanceNameIsDefault := s.loadInstanceName(ctx)
	enabled := cfg.Enabled && cfg.Status == StatusConnected && cfg.AgentStudioBaseURL != "" && cfg.ConnectorID != ""
	return &RuntimeTarget{
		Enabled:                 enabled,
		AgentStudioBaseURL:      cfg.AgentStudioBaseURL,
		AgentStudioServiceToken: strings.TrimSpace(values[KeyAgentStudioServiceToken]),
		ConnectorSlug:           cfg.ConnectorSlug,
		ConnectorID:             cfg.ConnectorID,
		Status:                  cfg.Status,
		LastError:               cfg.LastError,
		InstanceName:            instanceName,
		InstanceNameIsDefault:   instanceNameIsDefault,
		Policy:                  cfg.Policy,
	}, nil
}

func (s *Service) loadInstanceName(ctx context.Context) (string, bool) {
	if s == nil || s.reader == nil {
		return DefaultOMCName, true
	}
	rows, err := s.reader.List(ctx, BasicCategory, false)
	if err != nil {
		s.logger.Warn("list basic config for agent runtime target", zap.Error(err))
		return DefaultOMCName, true
	}
	for _, row := range rows {
		if row.Key != OMCNameKey {
			continue
		}
		value := strings.TrimSpace(row.Value)
		if value == "" {
			return DefaultOMCName, true
		}
		return value, isDefaultOMCName(value)
	}
	return DefaultOMCName, true
}

func (s *Service) Save(ctx context.Context, req UpdateRequest) (*AdminConfig, error) {
	values, settings, err := s.mergeSettings(ctx, req)
	if err != nil {
		return nil, err
	}
	if err := s.persistSettings(ctx, settings, ""); err != nil {
		return nil, err
	}
	for key, value := range settings.toValues(false) {
		values[key] = value
	}
	return adminConfigFromValues(values), nil
}

func (s *Service) Test(ctx context.Context, req UpdateRequest) (*ProvisionResult, error) {
	_, settings, err := s.mergeSettings(ctx, req)
	if err != nil {
		return nil, err
	}
	if err := settings.validateForProvision(); err != nil {
		return nil, err
	}
	return s.provision(ctx, settings)
}

func (s *Service) Sync(ctx context.Context, req UpdateRequest) (*AdminConfig, error) {
	values, settings, err := s.mergeSettings(ctx, req)
	if err != nil {
		return nil, err
	}
	if err := s.persistSettings(ctx, settings, ""); err != nil {
		return nil, err
	}
	for key, value := range settings.toValues(false) {
		values[key] = value
	}

	if !settings.Enabled {
		if err := s.persistStatus(ctx, "", "", StatusDisabled, ""); err != nil {
			return nil, err
		}
		values[KeyStatus] = StatusDisabled
		values[KeyLastError] = ""
		return adminConfigFromValues(values), nil
	}

	if err := settings.validateForProvision(); err != nil {
		if saveErr := s.persistStatus(ctx, "", "", StatusError, err.Error()); saveErr != nil {
			s.logger.Warn("persist agent config validation failure", zap.Error(saveErr))
		}
		return nil, err
	}

	result, err := s.provision(ctx, settings)
	if err != nil {
		if saveErr := s.persistStatus(ctx, "", "", StatusError, err.Error()); saveErr != nil {
			s.logger.Warn("persist agent config provision failure", zap.Error(saveErr))
		}
		return nil, err
	}

	runtimeURL := DefaultRuntimeStreamPath
	if err := s.persistStatus(ctx, result.ConnectorID, runtimeURL, StatusConnected, ""); err != nil {
		return nil, err
	}
	values[KeyConnectorID] = result.ConnectorID
	values[KeyRuntimeStreamURL] = runtimeURL
	values[KeyStatus] = StatusConnected
	values[KeyLastValidatedAt] = s.now().UTC().Format(time.RFC3339)
	values[KeyLastError] = ""
	return adminConfigFromValues(values), nil
}

func (s *Service) loadValues(ctx context.Context) (map[string]string, error) {
	rows, err := s.reader.List(ctx, Category, false)
	if err != nil {
		return nil, fmt.Errorf("list agent config: %w", err)
	}
	values := make(map[string]string, len(rows))
	for _, row := range rows {
		values[row.Key] = row.Value
	}
	if strings.TrimSpace(values[KeyAgentStudioBaseURL]) == "" {
		values[KeyAgentStudioBaseURL] = DefaultAgentStudioBaseURL
	}
	// Connector provisioning is shared infrastructure, not an instance-level user choice.
	values[KeyConnectorSlug] = DefaultConnectorSlug
	return values, nil
}

type mergedSettings struct {
	Enabled                 bool
	AgentStudioBaseURL      string
	AgentStudioServiceToken string
	ConnectorSlug           string
	Policy                  RuntimePolicy
}

func (s *Service) mergeSettings(ctx context.Context, req UpdateRequest) (map[string]string, mergedSettings, error) {
	values, err := s.loadValues(ctx)
	if err != nil {
		return nil, mergedSettings{}, err
	}

	settings := mergedSettings{
		Enabled:                 parseBool(values[KeyEnabled]),
		AgentStudioBaseURL:      values[KeyAgentStudioBaseURL],
		AgentStudioServiceToken: values[KeyAgentStudioServiceToken],
		ConnectorSlug:           values[KeyConnectorSlug],
		Policy:                  policyFromValues(values),
	}
	if req.Enabled != nil {
		settings.Enabled = *req.Enabled
	}
	if req.AgentStudioBaseURL != nil {
		raw := strings.TrimSpace(*req.AgentStudioBaseURL)
		if raw != "" {
			base, err := normalizeHTTPURL(raw)
			if err != nil {
				return nil, mergedSettings{}, fmt.Errorf("%w: invalid agentStudioBaseUrl: %v", commonerrors.ErrInvalidInput, err)
			}
			settings.AgentStudioBaseURL = base
		} else {
			settings.AgentStudioBaseURL = ""
		}
	}
	if req.AgentStudioServiceToken != nil && strings.TrimSpace(*req.AgentStudioServiceToken) != "" {
		settings.AgentStudioServiceToken = strings.TrimSpace(*req.AgentStudioServiceToken)
	}
	if req.AllowedMethods != nil {
		settings.Policy.AllowedMethods = normalizeMethods(*req.AllowedMethods)
	}
	if req.BlockedPathPrefixes != nil {
		settings.Policy.BlockedPathPrefixes = normalizePrefixes(*req.BlockedPathPrefixes)
	}
	if req.ToolTimeoutSeconds != nil {
		settings.Policy.ToolTimeoutSeconds = normalizePositiveInt(*req.ToolTimeoutSeconds, DefaultToolTimeoutSeconds, 300)
	}
	if req.MaxResponseBytes != nil {
		settings.Policy.MaxResponseBytes = normalizePositiveInt(*req.MaxResponseBytes, DefaultMaxResponseBytes, 4<<20)
	}
	return values, settings, nil
}

func (s mergedSettings) toValues(includeSecret bool) map[string]string {
	values := map[string]string{
		KeyEnabled:             fmt.Sprintf("%t", s.Enabled),
		KeyAgentStudioBaseURL:  s.AgentStudioBaseURL,
		KeyConnectorSlug:       s.ConnectorSlug,
		KeyAllowedMethods:      strings.Join(s.Policy.AllowedMethods, ","),
		KeyBlockedPathPrefixes: strings.Join(s.Policy.BlockedPathPrefixes, "\n"),
		KeyToolTimeoutSeconds:  fmt.Sprintf("%d", s.Policy.ToolTimeoutSeconds),
		KeyMaxResponseBytes:    fmt.Sprintf("%d", s.Policy.MaxResponseBytes),
	}
	if includeSecret && s.AgentStudioServiceToken != "" {
		values[KeyAgentStudioServiceToken] = s.AgentStudioServiceToken
	}
	return values
}

func (s mergedSettings) validateForProvision() error {
	if s.AgentStudioBaseURL == "" {
		return fmt.Errorf("%w: agentStudioBaseUrl is required", commonerrors.ErrInvalidInput)
	}
	if s.AgentStudioServiceToken == "" {
		return fmt.Errorf("%w: agentStudioServiceToken is required", commonerrors.ErrInvalidInput)
	}
	if s.ConnectorSlug == "" {
		return fmt.Errorf("%w: connectorSlug is required", commonerrors.ErrInvalidInput)
	}
	return nil
}

func (s *Service) persistSettings(ctx context.Context, settings mergedSettings, status string) error {
	values := settings.toValues(settings.AgentStudioServiceToken != "")
	if status != "" {
		values[KeyStatus] = status
	}
	items := make([]admin.BatchItem, 0, len(values))
	for key, value := range values {
		items = append(items, admin.BatchItem{Key: key, Value: value, ValueType: valueTypeForKey(key)})
	}
	_, err := s.writer.BatchUpsert(ctx, admin.BatchUpdateSysConfigRequest{Category: Category, Items: items})
	if err != nil {
		return fmt.Errorf("save agent config: %w", err)
	}
	return nil
}

func (s *Service) persistStatus(ctx context.Context, connectorID, runtimeURL, status, lastError string) error {
	now := s.now().UTC().Format(time.RFC3339)
	items := []admin.BatchItem{
		{Key: KeyStatus, Value: status, ValueType: "string"},
		{Key: KeyLastValidatedAt, Value: now, ValueType: "string"},
		{Key: KeyLastError, Value: lastError, ValueType: "string"},
	}
	if connectorID != "" {
		items = append(items, admin.BatchItem{Key: KeyConnectorID, Value: connectorID, ValueType: "string"})
	}
	if runtimeURL != "" {
		items = append(items, admin.BatchItem{Key: KeyRuntimeStreamURL, Value: runtimeURL, ValueType: "string"})
	}
	_, err := s.writer.BatchUpsert(ctx, admin.BatchUpdateSysConfigRequest{Category: Category, Items: items})
	if err != nil {
		return fmt.Errorf("save agent config status: %w", err)
	}
	return nil
}

func (s *Service) provision(ctx context.Context, settings mergedSettings) (*ProvisionResult, error) {
	payload := map[string]any{
		"slug":           settings.ConnectorSlug,
		"name":           "External Operations",
		"description":    "Generic business system action connector",
		"status":         "active",
		"runtimeBaseUrl": settings.AgentStudioBaseURL,
		"config": map[string]any{
			"displayName":      "External Operations",
			"delegationHeader": "Authorization",
			"policy": map[string]any{
				"allowReadActions":     methodAllowed(settings.Policy, http.MethodGet),
				"allowLowRiskActions":  hasWriteMethod(settings.Policy),
				"allowHighRiskActions": hasWriteMethod(settings.Policy),
				"allowedMethods":       settings.Policy.AllowedMethods,
				"blockedPathPrefixes":  settings.Policy.BlockedPathPrefixes,
				"toolTimeoutSeconds":   settings.Policy.ToolTimeoutSeconds,
				"maxResponseBytes":     settings.Policy.MaxResponseBytes,
			},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal action connector provision payload: %w", err)
	}
	endpoint := settings.AgentStudioBaseURL + "/api/integrations/action-connectors/provision"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build action connector provision request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+settings.AgentStudioServiceToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := s.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call action connector provision: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read action connector provision response: %w", err)
	}
	var result ProvisionResult
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &result)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		detail := result.Detail
		if detail == "" {
			detail = strings.TrimSpace(string(raw))
		}
		if detail == "" {
			detail = fmt.Sprintf("agent studio provision failed with HTTP %d", resp.StatusCode)
		}
		return &result, fmt.Errorf("%w: %s", commonerrors.ErrUnavailable, detail)
	}
	if result.ConnectorID == "" {
		return nil, fmt.Errorf("%w: action connector provision response missing connectorId", commonerrors.ErrUnavailable)
	}
	return &result, nil
}

func adminConfigFromValues(values map[string]string) *AdminConfig {
	enabled := parseBool(values[KeyEnabled])
	status := values[KeyStatus]
	if status == "" {
		if enabled {
			status = StatusNotConfigured
		} else {
			status = StatusDisabled
		}
	}
	return &AdminConfig{
		Enabled:                enabled,
		AgentStudioBaseURL:     values[KeyAgentStudioBaseURL],
		ServiceTokenConfigured: strings.TrimSpace(values[KeyAgentStudioServiceToken]) != "",
		ConnectorSlug:          values[KeyConnectorSlug],
		ConnectorID:            values[KeyConnectorID],
		RuntimeStreamURL:       values[KeyRuntimeStreamURL],
		Status:                 status,
		LastValidatedAt:        values[KeyLastValidatedAt],
		LastError:              values[KeyLastError],
		Policy:                 policyFromValues(values),
	}
}

func parseBool(value string) bool {
	v := strings.TrimSpace(strings.ToLower(value))
	return v == "true" || v == "1" || v == "yes" || v == "on"
}

func policyFromValues(values map[string]string) RuntimePolicy {
	return RuntimePolicy{
		AllowedMethods:      normalizeMethods(splitConfigList(defaultString(values[KeyAllowedMethods], DefaultAllowedMethods))),
		BlockedPathPrefixes: normalizePrefixes(splitConfigList(defaultString(values[KeyBlockedPathPrefixes], DefaultBlockedPathPrefixes))),
		ToolTimeoutSeconds:  normalizePositiveInt(parseInt(values[KeyToolTimeoutSeconds]), DefaultToolTimeoutSeconds, 300),
		MaxResponseBytes:    normalizePositiveInt(parseInt(values[KeyMaxResponseBytes]), DefaultMaxResponseBytes, 4<<20),
	}
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func splitConfigList(value string) []string {
	fields := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r' || r == '\t'
	})
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		if trimmed := strings.TrimSpace(field); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func normalizeMethods(input []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(input))
	for _, method := range input {
		normalized := strings.ToUpper(strings.TrimSpace(method))
		switch normalized {
		case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			if !seen[normalized] {
				seen[normalized] = true
				out = append(out, normalized)
			}
		}
	}
	if len(out) == 0 {
		return []string{http.MethodGet}
	}
	return out
}

func normalizePrefixes(input []string) []string {
	out := make([]string, 0, len(input))
	seen := map[string]bool{}
	for _, prefix := range input {
		trimmed := strings.TrimSpace(prefix)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		out = append(out, trimmed)
	}
	return out
}

func parseInt(value string) int {
	var parsed int
	_, _ = fmt.Sscanf(strings.TrimSpace(value), "%d", &parsed)
	return parsed
}

func normalizePositiveInt(value, fallback, max int) int {
	if value <= 0 {
		return fallback
	}
	if value > max {
		return max
	}
	return value
}

func methodAllowed(policy RuntimePolicy, method string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(method))
	for _, allowed := range policy.AllowedMethods {
		if allowed == normalized {
			return true
		}
	}
	return false
}

func hasWriteMethod(policy RuntimePolicy) bool {
	for _, method := range policy.AllowedMethods {
		if method != http.MethodGet {
			return true
		}
	}
	return false
}

func normalizeHTTPURL(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}
	u, err := url.Parse(trimmed)
	if err != nil {
		return "", err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("scheme must be http or https")
	}
	if u.Host == "" {
		return "", fmt.Errorf("host is required")
	}
	u.RawQuery = ""
	u.Fragment = ""
	return strings.TrimRight(u.String(), "/"), nil
}

func isDefaultOMCName(value string) bool {
	return strings.TrimSpace(value) == DefaultOMCName
}

func valueTypeForKey(key string) string {
	if key == KeyEnabled {
		return "bool"
	}
	if key == KeyToolTimeoutSeconds || key == KeyMaxResponseBytes {
		return "int"
	}
	return "string"
}
