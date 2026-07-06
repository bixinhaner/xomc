package agentaction

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/alarm"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
)

type DeviceService interface {
	ListDevices(ctx context.Context, filter device.DeviceFilter) (*model.ListResponse[model.Device], error)
	GetDevice(ctx context.Context, id uuid.UUID) (*model.Device, error)
	GetDeviceParameters(ctx context.Context, deviceID uuid.UUID) ([]model.DeviceParameter, error)
	AuthorizeDeviceGroupAccess(ctx context.Context, deviceID uuid.UUID, visibleGroups []uuid.UUID) error
}

type AlarmStore interface {
	ListActive(ctx context.Context, filter alarm.AlarmFilter) (*model.ListResponse[model.Alarm], error)
	Statistics(ctx context.Context, filter alarm.AlarmFilter) (*alarm.AlarmStatistics, error)
}

type Service struct {
	deviceSvc  DeviceService
	alarmStore AlarmStore
	now        func() time.Time
	logger     *zap.Logger
}

func NewService(deviceSvc DeviceService, alarmStore AlarmStore, logger *zap.Logger) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{
		deviceSvc:  deviceSvc,
		alarmStore: alarmStore,
		now:        time.Now,
		logger:     logger.Named("agentaction"),
	}
}

func (s *Service) ListActions() []ActionDescriptor {
	return []ActionDescriptor{
		{
			ID:          ActionDeviceSearch,
			Title:       "Search visible devices",
			Description: "Search devices visible to the current user by serial number, name, manufacturer, model, IP, or site.",
			Risk:        RiskRead,
			Scopes:      []string{"agent-actions", "devices:read"},
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{},
				"properties": map[string]any{
					"query": map[string]any{"type": "string", "description": "Keyword or serial number."},
					"limit": map[string]any{"type": "integer", "minimum": 1, "maximum": 50, "default": 10},
				},
			},
		},
		{
			ID:          ActionDeviceSummary,
			Title:       "Read device summary",
			Description: "Read one visible device and a compact sample of its current parameters.",
			Risk:        RiskRead,
			Scopes:      []string{"agent-actions", "devices:read"},
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{},
				"properties": map[string]any{
					"id":           map[string]any{"type": "string", "description": "Device UUID."},
					"serialNumber": map[string]any{"type": "string", "description": "Device serial number."},
				},
				"anyOf": []map[string]any{
					{"required": []string{"id"}},
					{"required": []string{"serialNumber"}},
				},
			},
		},
		{
			ID:          ActionAlarmActiveSummary,
			Title:       "Read active alarm summary",
			Description: "Read active alarm counts and top rows visible to the current user.",
			Risk:        RiskRead,
			Scopes:      []string{"agent-actions", "alarms:read"},
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"deviceSerialNumber": map[string]any{"type": "string", "description": "Optional device serial number filter."},
					"limit":              map[string]any{"type": "integer", "minimum": 1, "maximum": 50, "default": 10},
				},
			},
		},
		{
			ID:          ActionSystemHealth,
			Title:       "Read system health",
			Description: "Return the agent action service health timestamp.",
			Risk:        RiskRead,
			Scopes:      []string{"agent-actions"},
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
	}
}

func (s *Service) SearchActions(query string) []ActionDescriptor {
	query = strings.TrimSpace(strings.ToLower(query))
	actions := s.ListActions()
	if query == "" {
		return actions
	}
	filtered := make([]ActionDescriptor, 0, len(actions))
	for _, action := range actions {
		text := strings.ToLower(action.ID + " " + action.Title + " " + action.Description)
		if strings.Contains(text, query) {
			filtered = append(filtered, action)
		}
	}
	return filtered
}

func (s *Service) DescribeAction(actionID string) (*ActionDescriptor, error) {
	for _, action := range s.ListActions() {
		if action.ID == actionID {
			return &action, nil
		}
	}
	return nil, commonerrors.ErrNotFound
}

func (s *Service) Preview(ctx context.Context, req ActionRequest, actor RequestContext) (*ActionPreview, error) {
	desc, err := s.DescribeAction(req.ActionID)
	if err != nil {
		return nil, err
	}
	if desc.Risk != RiskRead {
		return nil, commonerrors.ErrForbidden
	}
	summary, err := s.previewSummary(ctx, req, actor)
	if err != nil {
		return nil, err
	}
	return &ActionPreview{
		ActionID: desc.ID,
		Title:    desc.Title,
		Summary:  summary,
		Risk:     desc.Risk,
	}, nil
}

func (s *Service) Execute(ctx context.Context, req ActionRequest, actor RequestContext) (*ActionResponse, error) {
	desc, err := s.DescribeAction(req.ActionID)
	if err != nil {
		return nil, err
	}
	if desc.Risk != RiskRead {
		return nil, commonerrors.ErrForbidden
	}
	if req.DryRun {
		preview, err := s.Preview(ctx, req, actor)
		if err != nil {
			return nil, err
		}
		return &ActionResponse{ActionID: req.ActionID, Status: "preview", Result: preview}, nil
	}

	var result any
	switch req.ActionID {
	case ActionDeviceSearch:
		result, err = s.executeDeviceSearch(ctx, req.Input, actor)
	case ActionDeviceSummary:
		result, err = s.executeDeviceSummary(ctx, req.Input, actor)
	case ActionAlarmActiveSummary:
		result, err = s.executeAlarmActiveSummary(ctx, req.Input, actor)
	case ActionSystemHealth:
		result = map[string]any{
			"status":     "ok",
			"checked_at": s.now().UTC().Format(time.RFC3339Nano),
		}
	default:
		err = commonerrors.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &ActionResponse{ActionID: req.ActionID, Status: "ok", Result: result}, nil
}

func (s *Service) previewSummary(ctx context.Context, req ActionRequest, actor RequestContext) (string, error) {
	switch req.ActionID {
	case ActionDeviceSearch:
		query, _ := stringInput(req.Input, "query")
		limit := intInput(req.Input, "limit", 10, 50)
		return fmt.Sprintf("Read up to %d visible devices matching %q.", limit, query), nil
	case ActionDeviceSummary:
		dev, err := s.resolveDevice(ctx, req.Input, actor)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Read summary and parameter sample for device %s.", dev.SerialNumber), nil
	case ActionAlarmActiveSummary:
		limit := intInput(req.Input, "limit", 10, 50)
		if sn, ok := stringInput(req.Input, "deviceSerialNumber"); ok && sn != "" {
			return fmt.Sprintf("Read active alarm summary for visible device %s, limited to %d rows.", sn, limit), nil
		}
		return fmt.Sprintf("Read active alarm summary for visible devices, limited to %d rows.", limit), nil
	case ActionSystemHealth:
		return "Read agent action service health timestamp.", nil
	default:
		return "", commonerrors.ErrNotFound
	}
}

func (s *Service) executeDeviceSearch(ctx context.Context, input map[string]any, actor RequestContext) (any, error) {
	if s.deviceSvc == nil {
		return nil, commonerrors.ErrUnavailable
	}
	query, _ := stringInput(input, "query")
	limit := intInput(input, "limit", 10, 50)
	req := model.DefaultListRequest()
	req.PageSize = limit
	req.SortBy = "updated_at"
	req.SortDir = "desc"
	filter := device.DeviceFilter{
		Search:        optionalString(query),
		VisibleGroups: actor.VisibleGroups,
		ListRequest:   req,
	}
	result, err := s.deviceSvc.ListDevices(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("search devices: %w", err)
	}
	items := make([]DeviceSummary, 0, len(result.Items))
	for _, dev := range result.Items {
		items = append(items, summarizeDevice(dev, nil))
	}
	return map[string]any{
		"items":     items,
		"total":     result.Total,
		"page":      result.Page,
		"page_size": result.PageSize,
		"query":     query,
	}, nil
}

func (s *Service) executeDeviceSummary(ctx context.Context, input map[string]any, actor RequestContext) (any, error) {
	dev, err := s.resolveDevice(ctx, input, actor)
	if err != nil {
		return nil, err
	}
	params, err := s.deviceSvc.GetDeviceParameters(ctx, dev.ID)
	if err != nil {
		return nil, fmt.Errorf("get device parameters: %w", err)
	}
	return summarizeDevice(*dev, sampleParameters(params, 24)), nil
}

func (s *Service) resolveDevice(ctx context.Context, input map[string]any, actor RequestContext) (*model.Device, error) {
	if s.deviceSvc == nil {
		return nil, commonerrors.ErrUnavailable
	}
	if idText, ok := stringInput(input, "id"); ok && idText != "" {
		id, err := uuid.Parse(idText)
		if err != nil {
			return nil, commonerrors.ErrInvalidInput
		}
		dev, err := s.deviceSvc.GetDevice(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("get device: %w", err)
		}
		if dev == nil {
			return nil, commonerrors.ErrNotFound
		}
		if err := s.deviceSvc.AuthorizeDeviceGroupAccess(ctx, dev.ID, actor.VisibleGroups); err != nil {
			return nil, err
		}
		return dev, nil
	}
	sn, ok := stringInput(input, "serialNumber")
	if !ok || sn == "" {
		return nil, commonerrors.ErrInvalidInput
	}
	req := model.DefaultListRequest()
	req.PageSize = 1
	filter := device.DeviceFilter{
		SN:            &sn,
		VisibleGroups: actor.VisibleGroups,
		ListRequest:   req,
	}
	result, err := s.deviceSvc.ListDevices(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("find device by serial number: %w", err)
	}
	if len(result.Items) == 0 {
		return nil, commonerrors.ErrNotFound
	}
	return &result.Items[0], nil
}

func (s *Service) executeAlarmActiveSummary(ctx context.Context, input map[string]any, actor RequestContext) (any, error) {
	if s.alarmStore == nil {
		return nil, commonerrors.ErrUnavailable
	}
	limit := intInput(input, "limit", 10, 50)
	req := model.DefaultListRequest()
	req.PageSize = limit
	req.SortBy = "raised_at"
	req.SortDir = "desc"
	filter := alarm.AlarmFilter{
		ListRequest:   req,
		VisibleGroups: actor.VisibleGroups,
	}
	if sn, ok := stringInput(input, "deviceSerialNumber"); ok && sn != "" {
		filter.DeviceSN = &sn
	}
	stats, err := s.alarmStore.Statistics(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("read alarm statistics: %w", err)
	}
	list, err := s.alarmStore.ListActive(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list active alarms: %w", err)
	}
	rows := make([]AlarmRow, 0, len(list.Items))
	for _, item := range list.Items {
		rows = append(rows, AlarmRow{
			ID:              item.ID,
			DeviceID:        item.DeviceID,
			DeviceSN:        item.DeviceSN,
			Severity:        int(item.Severity),
			AlarmIdentifier: item.AlarmIdentifier,
			Description:     item.Description,
			RaisedAt:        item.RaisedAt.UTC().Format(time.RFC3339Nano),
			IsRead:          item.IsRead,
		})
	}
	return map[string]any{
		"statistics": stats,
		"items":      rows,
		"total":      list.Total,
	}, nil
}

func summarizeDevice(dev model.Device, params []ParameterValue) DeviceSummary {
	return DeviceSummary{
		ID:              dev.ID,
		SerialNumber:    dev.SerialNumber,
		DeviceName:      dev.DeviceName,
		ProductClass:    dev.ProductClass,
		ModelName:       dev.ModelName,
		Manufacturer:    dev.Manufacturer,
		Technology:      string(dev.Technology),
		LifecycleState:  string(dev.LifecycleState),
		IsOnline:        dev.IsOnline,
		IPAddress:       dev.IPAddress,
		GroupName:       dev.GroupName,
		LastInformAt:    dev.LastInformAt,
		ParameterSample: params,
	}
}

func sampleParameters(params []model.DeviceParameter, limit int) []ParameterValue {
	if len(params) == 0 || limit <= 0 {
		return nil
	}
	interesting := make([]model.DeviceParameter, 0, len(params))
	for _, param := range params {
		path := strings.ToLower(param.ParameterPath)
		if param.ParameterValue == "" {
			continue
		}
		if strings.Contains(path, "software") ||
			strings.Contains(path, "hardware") ||
			strings.Contains(path, "serial") ||
			strings.Contains(path, "cell") ||
			strings.Contains(path, "status") ||
			strings.Contains(path, "radio") ||
			strings.Contains(path, "ip") {
			interesting = append(interesting, param)
		}
	}
	if len(interesting) == 0 {
		interesting = params
	}
	sort.SliceStable(interesting, func(i, j int) bool {
		return interesting[i].ParameterPath < interesting[j].ParameterPath
	})
	if len(interesting) > limit {
		interesting = interesting[:limit]
	}
	out := make([]ParameterValue, 0, len(interesting))
	for _, param := range interesting {
		out = append(out, ParameterValue{
			Path:      param.ParameterPath,
			Value:     param.ParameterValue,
			Writable:  param.Writable,
			UpdatedAt: param.LastUpdatedAt.UTC().Format(time.RFC3339Nano),
		})
	}
	return out
}

func optionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func stringInput(input map[string]any, key string) (string, bool) {
	if input == nil {
		return "", false
	}
	value, ok := input[key]
	if !ok {
		return "", false
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed), true
	default:
		return strings.TrimSpace(fmt.Sprint(typed)), true
	}
}

func intInput(input map[string]any, key string, fallback, max int) int {
	if input == nil {
		return fallback
	}
	value, ok := input[key]
	if !ok {
		return fallback
	}
	var n int
	switch typed := value.(type) {
	case float64:
		n = int(typed)
	case int:
		n = typed
	case int64:
		n = int(typed)
	case string:
		if _, err := fmt.Sscanf(typed, "%d", &n); err != nil {
			n = fallback
		}
	default:
		n = fallback
	}
	if n < 1 {
		return fallback
	}
	if n > max {
		return max
	}
	return n
}

func actionErrorCode(err error) string {
	switch {
	case errors.Is(err, commonerrors.ErrUnauthorized):
		return "UNAUTHORIZED"
	case errors.Is(err, commonerrors.ErrForbidden):
		return "FORBIDDEN"
	case errors.Is(err, commonerrors.ErrNotFound):
		return "ACTION_NOT_FOUND"
	case errors.Is(err, commonerrors.ErrInvalidInput):
		return "VALIDATION_FAILED"
	case errors.Is(err, commonerrors.ErrUnavailable):
		return "UPSTREAM_ERROR"
	default:
		return "UPSTREAM_ERROR"
	}
}
