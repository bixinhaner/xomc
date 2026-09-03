package agentbridge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/agentconfig"
	"github.com/omcgo/omcgo/internal/agentruntime"
	"go.uber.org/zap"
)

type LocalToolExecutor interface {
	Execute(context.Context, *admin.Claims, agentruntime.ToolRequest, agentconfig.RuntimePolicy) agentruntime.ToolResult
	HandbookDigest() (string, error)
}

type ToolWorker struct {
	client   *Client
	pool     *pgxpool.Pool
	executor LocalToolExecutor
	workerID string
	policy   agentconfig.RuntimePolicy
	logger   *zap.Logger
	cancel   context.CancelFunc
	done     chan struct{}
	once     sync.Once
}

func NewToolWorker(client *Client, pool *pgxpool.Pool, executor LocalToolExecutor, workerID string, logger *zap.Logger) *ToolWorker {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &ToolWorker{
		client: client, pool: pool, executor: executor, workerID: workerID,
		policy: agentconfig.RuntimePolicy{
			AllowedMethods:      []string{http.MethodGet},
			BlockedPathPrefixes: []string{"/api/v1/auth/*", "/api/v1/agent/*", "/api/v1/admin/*"},
			ToolTimeoutSeconds:  30, MaxResponseBytes: agentconfig.DefaultMaxResponseBytes,
		},
		logger: logger.Named("agent-tool-worker"), done: make(chan struct{}),
	}
}

func (w *ToolWorker) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel
	go w.run(ctx)
}

func (w *ToolWorker) Close() error {
	w.once.Do(func() {
		if w.cancel != nil {
			w.cancel()
		}
		<-w.done
	})
	return nil
}

func (w *ToolWorker) run(ctx context.Context) {
	defer close(w.done)
	for ctx.Err() == nil {
		items, err := w.client.LeaseTools(ctx, w.workerID, 8)
		if err != nil {
			if ctx.Err() == nil {
				if errors.Is(err, ErrBridgeNotConnected) {
					w.logger.Debug("agent tool worker waiting for connector configuration")
				} else {
					w.logger.Warn("lease agent tool invocations", zap.Error(err))
				}
				select {
				case <-ctx.Done():
					return
				case <-time.After(5 * time.Second):
				}
			}
			continue
		}
		if len(items) == 0 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
			continue
		}
		for _, invocation := range items {
			w.execute(ctx, invocation)
		}
	}
}

func (w *ToolWorker) execute(parent context.Context, invocation ToolInvocation) {
	result := ToolResult{
		ContractVersion: ContractVersion, LeaseToken: invocation.LeaseToken, WorkerID: w.workerID,
		Status: "failed", CompletedAt: time.Now().UTC(), TraceID: invocation.TraceID,
	}
	deadline := invocation.DeadlineAt
	if deadline.IsZero() || deadline.After(time.Now().Add(2*time.Minute)) {
		deadline = time.Now().Add(2 * time.Minute)
	}
	ctx, cancel := context.WithDeadline(parent, deadline)
	defer cancel()
	claims, path, err := w.authorize(ctx, invocation)
	if err != nil {
		result.Error = &ToolError{Code: "OPERATION_NOT_ALLOWED", Message: err.Error(), Retryable: false}
	} else {
		local := w.executor.Execute(ctx, claims, agentruntime.ToolRequest{
			RunID: invocation.RunID, ToolCallID: invocation.InvocationID, Tool: "rest.request",
			Input: agentruntime.ToolRequestBody{
				OperationID: invocation.OperationID, Method: http.MethodGet, Path: path,
				Query: invocation.Arguments.Query, Body: nil, Reason: "background agent read-only investigation",
			},
		}, w.policy)
		if local.Status == "ok" {
			result.Status = "succeeded"
			result.HTTPStatus = http.StatusOK
			result.Output = local.Output
			raw, _ := json.Marshal(local.Output)
			result.ResultBytes = len(raw)
		} else {
			result.Error = &ToolError{Code: local.Error.Code, Message: local.Error.Message, Retryable: local.Error.Retryable}
		}
	}
	result.CompletedAt = time.Now().UTC()
	if err := w.client.SubmitToolResult(parent, invocation.InvocationID, result); err != nil {
		w.logger.Error("submit agent tool result", zap.String("invocation_id", invocation.InvocationID), zap.Error(err))
	}
}

func (w *ToolWorker) authorize(ctx context.Context, invocation ToolInvocation) (*admin.Claims, string, error) {
	if w.executor == nil {
		return nil, "", fmt.Errorf("local tool executor is unavailable")
	}
	if !strings.EqualFold(invocation.Method, http.MethodGet) {
		return nil, "", fmt.Errorf("background method %s is forbidden", invocation.Method)
	}
	if invocation.PackageDigest != XOMCPackageDigest {
		return nil, "", fmt.Errorf("package digest does not match the locally installed policy")
	}
	digest, err := w.executor.HandbookDigest()
	if err != nil {
		return nil, "", err
	}
	if invocation.HandbookDigest != digest {
		return nil, "", fmt.Errorf("handbook digest does not match the running API contract")
	}
	allowed := map[string]map[string]struct{}{
		"task-failure-analysis": {
			"get.devices.by_id": {}, "get.devices.tasks.by_task_id": {}, "get.devices": {}, "get.alarms.active": {},
		},
		"access-review-assistant": {
			"get.device_access.candidates": {}, "get.devices.by_id": {}, "get.devices": {}, "get.topology.nodes": {},
		},
		"severe-alarm-explanation": {
			"get.alarms.active": {}, "get.devices.by_id": {}, "get.devices": {}, "get.pm.counters.aggregated": {},
		},
		"daily-operations-summary": {
			"get.alarms.active": {}, "get.devices": {}, "get.device_access.candidates": {}, "get.pm.counters.aggregated": {},
		},
	}
	operations := allowed[invocation.ScenarioKey]
	_, scenarioOperation := operations[invocation.OperationID]
	if !scenarioOperation && !isHandbookOperation(invocation.OperationID) {
		return nil, "", fmt.Errorf("operation %s is outside the local scenario policy", invocation.OperationID)
	}
	// The local connector owns the operation-to-route mapping. Never execute a
	// model-supplied literal path directly: bind the route from authoritative
	// resource scope so a forged path cannot escape the triggering task/device.
	path, err := bindScopedOperationPath(invocation.OperationID, invocation.Path, invocation.ResourceScope)
	if err != nil {
		return nil, "", err
	}
	claims, err := w.resolveBackgroundPrincipal(ctx, invocation)
	if err != nil {
		return nil, "", err
	}
	return claims, path, nil
}

func isHandbookOperation(operationID string) bool {
	return operationID == "get.agent.handbook.manifest" || operationID == "get.agent.handbook.chunks.by_index"
}

func bindScopedOperationPath(operationID, requestedPath string, resources []ResourceRef) (string, error) {
	resourceID := func(resourceType string) string {
		for _, ref := range resources {
			if ref.Type == resourceType {
				return strings.TrimSpace(ref.ID)
			}
		}
		return ""
	}
	switch operationID {
	case "get.agent.handbook.manifest":
		return "/api/v1/agent/handbook/manifest", nil
	case "get.agent.handbook.chunks.by_index":
		const prefix = "/api/v1/agent/handbook/chunks/"
		index := strings.TrimPrefix(strings.TrimSpace(requestedPath), prefix)
		if index == "" || strings.TrimLeft(index, "0123456789") != "" || prefix+index != strings.TrimSpace(requestedPath) {
			return "", fmt.Errorf("handbook chunk path must contain a numeric index")
		}
		return prefix + index, nil
	case "get.devices.by_id":
		id := resourceID("device")
		if id == "" {
			return "", fmt.Errorf("device resource is missing from invocation scope")
		}
		return "/api/v1/devices/" + id, nil
	case "get.devices.tasks.by_task_id":
		id := resourceID("task")
		if id == "" {
			return "", fmt.Errorf("task resource is missing from invocation scope")
		}
		return "/api/v1/devices/tasks/" + id, nil
	case "get.devices":
		return "/api/v1/devices", nil
	case "get.alarms.active":
		return "/api/v1/alarms/active", nil
	case "get.pm.counters.aggregated":
		return "/api/v1/pm/counters/aggregated", nil
	case "get.topology.nodes":
		return "/api/v1/topology/nodes", nil
	case "get.device_access.candidates":
		if resourceID("candidate") == "" && resourceID("ne_group") == "" {
			return "", fmt.Errorf("candidate or network-group resource is missing from invocation scope")
		}
		return "/api/v1/device-access/candidates", nil
	default:
		return "", fmt.Errorf("operation %s has no local route binding", operationID)
	}
}

func (w *ToolWorker) resolveBackgroundPrincipal(ctx context.Context, invocation ToolInvocation) (*admin.Claims, error) {
	claims, err := w.resolveTaskCreator(ctx, invocation.ResourceScope)
	if err != nil && !strings.Contains(err.Error(), "without a task resource") {
		return nil, err
	}
	if claims == nil {
		// Automatic alarm/access/summary events have no interactive actor. Their
		// service identity is intentionally powerful at the JWT layer but remains
		// constrained to one minute, one scenario allowlist and the invocation's
		// authoritative resource scope before this point.
		claims = &admin.Claims{
			UserID:       uuid.NewSHA1(uuid.NameSpaceOID, []byte("xomc-agent-background")),
			Username:     "agent-background",
			IsSuperAdmin: true,
			Roles:        []string{"agent-background"},
		}
	}
	claims.Scopes = []string{
		"agent-background:run:" + invocation.RunID,
		"agent-background:scenario:" + invocation.ScenarioKey,
		"agent-background:package:" + invocation.PackageDigest,
		"agent-background:handbook:" + invocation.HandbookDigest,
		"agent-background:operation:" + invocation.OperationID,
	}
	for _, resource := range invocation.ResourceScope {
		claims.Scopes = append(claims.Scopes, "agent-background:resource:"+resource.Type+":"+resource.ID)
	}
	return claims, nil
}

func (w *ToolWorker) resolveTaskCreator(ctx context.Context, resources []ResourceRef) (*admin.Claims, error) {
	var taskID string
	for _, resource := range resources {
		if resource.Type == "task" {
			taskID = resource.ID
			break
		}
	}
	if taskID == "" {
		return nil, fmt.Errorf("background principal cannot be resolved without a task resource")
	}
	query, args, err := psql.Select("u.id", "u.username", "u.source", "ur.role_id", "r.name").
		From("device_tasks t").Join("users u ON u.id::text = t.creator_id AND u.status = 'active'").
		Join("user_roles ur ON ur.user_id = u.id AND ur.is_default = true").Join("roles r ON r.id = ur.role_id").
		Where(sq.Eq{"t.id": taskID}).Limit(1).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build background principal lookup: %w", err)
	}
	var userID, roleID uuid.UUID
	var username, source, roleName string
	if err := w.pool.QueryRow(ctx, query, args...).Scan(&userID, &username, &source, &roleID, &roleName); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("task creator is unavailable or has no active default role")
		}
		return nil, fmt.Errorf("resolve background principal: %w", err)
	}
	if source == string(admin.UserSourceBuiltIn) {
		return nil, fmt.Errorf("built-in super administrator cannot be used as a background principal")
	}
	return &admin.Claims{UserID: userID, Username: username, Roles: []string{roleName}, CurrentRoleID: &roleID}, nil
}
