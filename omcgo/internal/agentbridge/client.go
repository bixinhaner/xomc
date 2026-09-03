package agentbridge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/omcgo/omcgo/internal/agentconfig"
)

var ErrBridgeNotConnected = errors.New("agent bridge target is not connected or has no service credential")

type RuntimeTargetProvider interface {
	GetRuntimeTarget(context.Context) (*agentconfig.RuntimeTarget, error)
}

type Client struct {
	targets RuntimeTargetProvider
	http    *http.Client
}

func NewClient(targets RuntimeTargetProvider, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 35 * time.Second}
	} else if httpClient.Timeout == 0 {
		clone := *httpClient
		clone.Timeout = 35 * time.Second
		httpClient = &clone
	}
	return &Client{targets: targets, http: httpClient}
}

func (c *Client) PublishEvent(ctx context.Context, envelope ConnectorEventEnvelope) error {
	return c.doJSON(ctx, http.MethodPost, "/events", envelope, nil)
}

func (c *Client) LeaseTools(ctx context.Context, workerID string, maxItems int) ([]ToolInvocation, error) {
	var response struct {
		Items []ToolInvocation `json:"items"`
	}
	err := c.doJSON(ctx, http.MethodPost, "/tool-invocations/lease", map[string]any{
		"workerId": workerID, "maxItems": maxItems, "waitSeconds": 25,
		"leaseSeconds": 60, "contractVersions": []string{ContractVersion},
	}, &response)
	return response.Items, err
}

func (c *Client) SubmitToolResult(ctx context.Context, invocationID string, result ToolResult) error {
	return c.doJSON(ctx, http.MethodPost, "/tool-invocations/"+url.PathEscape(invocationID)+"/result", result, nil)
}

func (c *Client) LeaseFindings(ctx context.Context, workerID string, maxItems int) ([]FindingDelivery, error) {
	var response struct {
		Items []FindingDelivery `json:"items"`
	}
	err := c.doJSON(ctx, http.MethodPost, "/findings/lease", map[string]any{
		"workerId": workerID, "maxItems": maxItems, "waitSeconds": 25, "leaseSeconds": 60,
	}, &response)
	return response.Items, err
}

func (c *Client) AckFinding(ctx context.Context, deliveryID string, ack FindingDeliveryAck) error {
	return c.doJSON(ctx, http.MethodPost, "/findings/"+url.PathEscape(deliveryID)+"/ack", ack, nil)
}

func (c *Client) GetOverview(ctx context.Context) (ProactiveOverview, error) {
	var response ProactiveOverview
	err := c.doJSON(ctx, http.MethodGet, "/proactive/overview", nil, &response)
	return response, err
}

func (c *Client) UpdateScenario(ctx context.Context, scenarioKey string, update ScenarioUpdate) (ProactiveScenario, error) {
	var response ProactiveScenario
	err := c.doJSON(ctx, http.MethodPatch, "/proactive/scenarios/"+url.PathEscape(scenarioKey), update, &response)
	return response, err
}

func (c *Client) CancelRun(ctx context.Context, runID string) error {
	return c.doJSON(ctx, http.MethodPost, "/proactive/runs/"+url.PathEscape(runID)+"/cancel", map[string]any{}, nil)
}

func (c *Client) SendHeartbeat(ctx context.Context, heartbeat ConnectorHeartbeat) error {
	return c.doJSON(ctx, http.MethodPost, "/proactive/heartbeat", heartbeat, nil)
}

func (c *Client) doJSON(ctx context.Context, method, suffix string, input any, output any) error {
	if c == nil || c.targets == nil {
		return fmt.Errorf("agent bridge target provider is unavailable")
	}
	target, err := c.targets.GetRuntimeTarget(ctx)
	if err != nil {
		return fmt.Errorf("load agent bridge target: %w", err)
	}
	if !target.Enabled || strings.TrimSpace(target.AgentStudioServiceToken) == "" || strings.TrimSpace(target.ConnectorID) == "" {
		return ErrBridgeNotConnected
	}
	var body io.Reader
	if input != nil {
		raw, marshalErr := json.Marshal(input)
		if marshalErr != nil {
			return fmt.Errorf("marshal agent bridge request: %w", marshalErr)
		}
		body = bytes.NewReader(raw)
	}
	endpoint := strings.TrimRight(target.AgentStudioBaseURL, "/") + "/api/action-connectors/" + url.PathEscape(target.ConnectorID) + suffix
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return fmt.Errorf("build agent bridge request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+target.AgentStudioServiceToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("call agent bridge endpoint: %w", err)
	}
	defer resp.Body.Close()
	responseRaw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return fmt.Errorf("read agent bridge response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("agent bridge endpoint returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(responseRaw)))
	}
	if output != nil && len(responseRaw) > 0 {
		if err := json.Unmarshal(responseRaw, output); err != nil {
			return fmt.Errorf("decode agent bridge response: %w", err)
		}
	}
	return nil
}
