package agentconfig

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/admin"
)

type fakeConfigStore struct {
	values      map[string]string
	basicValues map[string]string
}

func newFakeConfigStore(values map[string]string) *fakeConfigStore {
	copy := make(map[string]string, len(values))
	for key, value := range values {
		copy[key] = value
	}
	return &fakeConfigStore{values: copy, basicValues: map[string]string{}}
}

func (s *fakeConfigStore) List(_ context.Context, category string, _ bool) ([]admin.SysConfig, error) {
	values := s.values
	if category == BasicCategory {
		values = s.basicValues
	} else if category != Category {
		return nil, nil
	}
	rows := make([]admin.SysConfig, 0, len(values))
	for key, value := range values {
		rows = append(rows, admin.SysConfig{
			ID:       uuid.New(),
			Category: category,
			Key:      key,
			Value:    value,
		})
	}
	return rows, nil
}

func (s *fakeConfigStore) BatchUpsert(_ context.Context, req admin.BatchUpdateSysConfigRequest) (int, error) {
	if req.Category != Category {
		return 0, nil
	}
	for _, item := range req.Items {
		s.values[item.Key] = item.Value
	}
	return len(req.Items), nil
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) Do(req *http.Request) (*http.Response, error) {
	return f(req)
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func stringPtr(value string) *string {
	return &value
}

func TestAdminConfigMasksServiceToken(t *testing.T) {
	store := newFakeConfigStore(map[string]string{
		KeyEnabled:                 "true",
		KeyAgentStudioServiceToken: "secret-token",
	})
	svc := NewService(store, store, nil, nil)

	cfg, err := svc.GetAdminConfig(context.Background())

	require.NoError(t, err)
	require.True(t, cfg.Enabled)
	require.True(t, cfg.ServiceTokenConfigured)
}

func TestAdminConfigUsesSafeConnectionDefaultsWithoutEmbeddedCredential(t *testing.T) {
	store := newFakeConfigStore(nil)
	svc := NewService(store, store, nil, nil)

	cfg, err := svc.GetAdminConfig(context.Background())

	require.NoError(t, err)
	require.Equal(t, DefaultAgentStudioBaseURL, cfg.AgentStudioBaseURL)
	require.False(t, cfg.ServiceTokenConfigured)
	require.Equal(t, DefaultConnectorSlug, cfg.ConnectorSlug)
	require.Equal(t, StatusDisabled, cfg.Status)
}

func TestSaveKeepsExistingServiceTokenWhenOmitted(t *testing.T) {
	store := newFakeConfigStore(map[string]string{
		KeyAgentStudioServiceToken: "old-token",
		KeyAgentStudioBaseURL:      "https://old-agent.example.com",
	})
	svc := NewService(store, store, nil, nil)

	enabled := true
	_, err := svc.Save(context.Background(), UpdateRequest{
		Enabled:            &enabled,
		AgentStudioBaseURL: stringPtr("https://agent.example.com/"),
	})

	require.NoError(t, err)
	require.Equal(t, "old-token", store.values[KeyAgentStudioServiceToken])
	require.Equal(t, "https://agent.example.com", store.values[KeyAgentStudioBaseURL])
}

func TestSavePatchDoesNotClearExistingValuesWhenFieldsAreOmitted(t *testing.T) {
	store := newFakeConfigStore(map[string]string{
		KeyEnabled:                 "true",
		KeyAgentStudioBaseURL:      "https://agent.example.com",
		KeyAgentStudioServiceToken: "old-token",
		KeyConnectorSlug:           "legacy-connector",
	})
	svc := NewService(store, store, nil, nil)

	enabled := false
	cfg, err := svc.Save(context.Background(), UpdateRequest{
		Enabled: &enabled,
	})

	require.NoError(t, err)
	require.False(t, cfg.Enabled)
	require.Equal(t, "https://agent.example.com", store.values[KeyAgentStudioBaseURL])
	require.Equal(t, "old-token", store.values[KeyAgentStudioServiceToken])
	require.Equal(t, DefaultConnectorSlug, store.values[KeyConnectorSlug])
}

func TestRuntimeVisibilitySeparatesSwitchFromConnectedRuntime(t *testing.T) {
	store := newFakeConfigStore(map[string]string{
		KeyEnabled:            "true",
		KeyAgentStudioBaseURL: "https://agent.example.com",
		KeyStatus:             StatusNotConfigured,
	})
	svc := NewService(store, store, nil, nil)

	visibility, err := svc.GetVisibilityConfig(context.Background())
	require.NoError(t, err)
	require.True(t, visibility.Visible)
	require.False(t, visibility.Enabled)

	runtime, err := svc.GetRuntimeConfig(context.Background())
	require.NoError(t, err)
	require.True(t, runtime.Visible)
	require.False(t, runtime.Enabled)
}

func TestRuntimeTargetIncludesInstanceDisplayMetadata(t *testing.T) {
	store := newFakeConfigStore(map[string]string{
		KeyEnabled:            "true",
		KeyAgentStudioBaseURL: "https://agent.example.com",
		KeyConnectorSlug:      "legacy-connector",
		KeyConnectorID:        "connector-1",
		KeyStatus:             StatusConnected,
	})
	store.basicValues[OMCNameKey] = "OMC 陕西"
	svc := NewService(store, store, nil, nil)

	target, err := svc.GetRuntimeTarget(context.Background())

	require.NoError(t, err)
	require.True(t, target.Enabled)
	require.Equal(t, DefaultConnectorSlug, target.ConnectorSlug)
	require.Equal(t, "connector-1", target.ConnectorID)
	require.Equal(t, "OMC 陕西", target.InstanceName)
	require.False(t, target.InstanceNameIsDefault)
}

func TestRuntimeTargetMarksDefaultInstanceName(t *testing.T) {
	store := newFakeConfigStore(map[string]string{
		KeyEnabled:            "true",
		KeyAgentStudioBaseURL: "https://agent.example.com",
		KeyConnectorID:        "connector-1",
		KeyStatus:             StatusConnected,
	})
	store.basicValues[OMCNameKey] = DefaultOMCName
	svc := NewService(store, store, nil, nil)

	target, err := svc.GetRuntimeTarget(context.Background())

	require.NoError(t, err)
	require.Equal(t, DefaultOMCName, target.InstanceName)
	require.True(t, target.InstanceNameIsDefault)
}

func TestSyncProvisionsConnectorAndPersistsRuntimeConfig(t *testing.T) {
	store := newFakeConfigStore(map[string]string{
		KeyAgentStudioServiceToken: "provision-token",
	})
	var capturedAuth string
	var capturedBody string
	httpClient := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		capturedAuth = req.Header.Get("Authorization")
		raw, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		capturedBody = string(raw)
		return jsonResponse(http.StatusOK, `{
			"connectorId":"connector-1",
			"slug":"external-agent-abc",
			"status":"connected",
			"runtimeStreamUrl":"https://agent.example.com/api/action-connectors/connector-1/chat/stream"
		}`), nil
	})
	svc := NewService(store, store, httpClient, nil)
	svc.now = func() time.Time { return time.Date(2026, 7, 7, 10, 0, 0, 0, time.UTC) }

	enabled := true
	cfg, err := svc.Sync(context.Background(), UpdateRequest{
		Enabled:            &enabled,
		AgentStudioBaseURL: stringPtr("https://agent.example.com/"),
	})

	require.NoError(t, err)
	require.Equal(t, "Bearer provision-token", capturedAuth)
	require.NotContains(t, capturedBody, "agent-actions")
	require.NotContains(t, capturedBody, `"runtimeInstruction"`)
	require.Contains(t, capturedBody, `"allowedMethods":["GET"]`)
	require.Equal(t, StatusConnected, cfg.Status)
	require.Equal(t, "connector-1", cfg.ConnectorID)
	require.Equal(t, DefaultRuntimeStreamPath, cfg.RuntimeStreamURL)
	require.Equal(t, StatusConnected, store.values[KeyStatus])
	require.Equal(t, "2026-07-07T10:00:00Z", store.values[KeyLastValidatedAt])
}

func TestSyncFailurePersistsErrorWithoutClearingExistingConnector(t *testing.T) {
	store := newFakeConfigStore(map[string]string{
		KeyAgentStudioServiceToken: "provision-token",
		KeyConnectorID:             "connector-old",
		KeyRuntimeStreamURL:        "https://agent.example.com/api/action-connectors/connector-old/chat/stream",
	})
	httpClient := roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusBadGateway, `{"detail":"Action connector is unreachable","connectorId":"connector-new"}`), nil
	})
	svc := NewService(store, store, httpClient, nil)
	svc.now = func() time.Time { return time.Date(2026, 7, 7, 10, 0, 0, 0, time.UTC) }

	enabled := true
	_, err := svc.Sync(context.Background(), UpdateRequest{
		Enabled:            &enabled,
		AgentStudioBaseURL: stringPtr("https://agent.example.com/"),
	})

	require.Error(t, err)
	require.Equal(t, StatusError, store.values[KeyStatus])
	require.Contains(t, store.values[KeyLastError], "Action connector is unreachable")
	require.Equal(t, "connector-old", store.values[KeyConnectorID])
	require.Equal(t, "https://agent.example.com/api/action-connectors/connector-old/chat/stream", store.values[KeyRuntimeStreamURL])
}
