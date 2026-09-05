package agentbridge

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/omcgo/omcgo/internal/agentassistant"
	"github.com/omcgo/omcgo/internal/agentconfig"
)

type assistantTarget struct{ value *agentconfig.RuntimeTarget }

func (t assistantTarget) GetRuntimeTarget(context.Context) (*agentconfig.RuntimeTarget, error) {
	return t.value, nil
}

// Freeze the target for one request: a concurrent connector reconfiguration
// must not send an old run's payload using a different connector's credentials.
func (c *Client) assistantClient(ctx context.Context, expected string, timeout time.Duration) (*Client, string, error) {
	if c == nil || c.targets == nil {
		return nil, "", fmt.Errorf("ASSISTANT_NOT_CONNECTED")
	}
	target, err := c.targets.GetRuntimeTarget(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("ASSISTANT_NOT_CONNECTED: %w", err)
	}
	if target == nil || !target.Enabled || strings.TrimSpace(target.ConnectorID) == "" || strings.TrimSpace(target.AgentStudioServiceToken) == "" {
		return nil, "", fmt.Errorf("ASSISTANT_NOT_CONNECTED")
	}
	if expected != "" && target.ConnectorID != expected {
		return nil, "", fmt.Errorf("ASSISTANT_CONNECTOR_CHANGED")
	}
	snapshot := *target
	client := *c.http
	client.Timeout = timeout
	return &Client{targets: assistantTarget{value: &snapshot}, http: &client}, snapshot.ConnectorID, nil
}
func (c *Client) AssistantConnection(ctx context.Context) (string, error) {
	_, id, err := c.assistantClient(ctx, "", 35*time.Second)
	return id, err
}
func (c *Client) PlanAssistant(ctx context.Context, input agentassistant.PlanRequest) (agentassistant.PlanResponse, error) {
	var out agentassistant.PlanResponse
	client, _, err := c.assistantClient(ctx, "", 145*time.Second)
	if err != nil {
		return out, err
	}
	err = client.doJSON(ctx, http.MethodPost, "/assistant-builder/plan", input, &out)
	return out, err
}
func (c *Client) SubmitAssistant(ctx context.Context, connector string, input agentassistant.ExecutionRequest) (agentassistant.RemoteRun, error) {
	var out agentassistant.RemoteRun
	client, _, err := c.assistantClient(ctx, connector, 35*time.Second)
	if err != nil {
		return out, err
	}
	err = client.doJSON(ctx, http.MethodPost, "/assistant-runs", input, &out)
	return out, err
}
func (c *Client) GetAssistantRun(ctx context.Context, connector, id string) (agentassistant.RemoteRun, error) {
	var out agentassistant.RemoteRun
	client, _, err := c.assistantClient(ctx, connector, 35*time.Second)
	if err != nil {
		return out, err
	}
	err = client.doJSON(ctx, http.MethodGet, "/assistant-runs/"+url.PathEscape(id), nil, &out)
	return out, err
}
func (c *Client) CancelAssistantRun(ctx context.Context, connector, id string) error {
	client, _, err := c.assistantClient(ctx, connector, 35*time.Second)
	if err != nil {
		return err
	}
	return client.CancelRun(ctx, id)
}
