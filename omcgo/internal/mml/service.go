package mml

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/global"
	acsrpc "github.com/omcgo/omcgo/internal/acs/rpc"
	appcontext "github.com/omcgo/omcgo/internal/core/context"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	taskpkg "github.com/omcgo/omcgo/internal/task"
)

// RoleQuerier 派生当前 admin 在 RBAC 体系下可见的 device group IDs（用于 T-0090-c
// 私有命令 group-share 过滤）。
//
// 消费者驱动小接口：admin.PgRoleRepository.GetUserVisibleGroupIDs 天然实现。
// 不直接依赖 admin 包，便于测试 mock + 避免反向依赖。
type RoleQuerier interface {
	GetUserVisibleGroupIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}

// Service provides business logic for the MML console module.
type Service struct {
	cmdRepo                      CommandRepository
	scriptRepo                   ScriptRepository
	taskRepo                     TaskRepository
	customCommandRepo            CustomCommandRepository
	customCommandPathRepo        CustomCommandPathRepository // issue #115 调整3：自定义命令 path 关联表；setter 注入，nil 时 path 端点返 503
	auditRepo                    AuditRepository
	cmdParamRepo                 CommandParamRepository
	fanouter                     *Fanouter
	hub                          SSEPublisher
	roleQuerier                  RoleQuerier                  // optional; nil 时 ListCustomCommands fallback creator-only 过滤
	deviceTaskPathMissAggregator DeviceTaskPathMissAggregator // Stage 3 路径翻译警告字段聚合
	deviceTaskResultLister       DeviceTaskResultLister       // 任务记录"查看"modal 的设备级结果（2026-05-23 修）
	deviceLookup                 DeviceLookup                 // 设备 product_class 反查（混类型 per-product 翻译用）；nil 时跳过
	pathTranslator               PathTranslator               // R-9.3 per-device standardPath → privatePath 翻译；nil 时跳过
	exporter                     *Exporter                    // 结果 CSV 导出（MinIO）；nil 时导出端点返回 503
	// 参数路径 → 友好名（standard_params.description）解析，CSV「参数名称」列用；nil 时回退 param_refs。
	pathNameResolver func(ctx context.Context, paths []string) (map[string]string, error)
	// customCommandSupportedPaths 按产品参数模型解析自定义命令可用 path。
	// 仅控制台携带 ProductID 时使用；管理端不传产品上下文，不裁剪模板。
	customCommandSupportedPaths func(ctx context.Context, productID uuid.UUID) (map[string]struct{}, error)
	scriptImportService         ScriptImportServiceAPI
	// scriptExecutionValidator is optional because older deployments may not
	// have the TXT validator wired yet. When present it performs the dynamic
	// device/command checks immediately before an execution is persisted and
	// again when a scheduled instance is dispatched.
	scriptExecutionValidator ScriptImportValidationRunner
	logger                   *zap.Logger

	// 按 product_class 缓存 standardPath→privatePath 翻译结果（TTL 1 分钟）。混类型任务
	// per-product 翻译一次、同 product_class 复用，避免逐设备重复翻译。
	pcTransCache   map[string]pcTransCacheEntry
	pcTransCacheMu sync.Mutex
}

// DeviceLookup 接口复用 fanout.go 已有定义（GetBySerialNumber → *model.Device，
// 含 ProductClass / FirmwareVersion 字段）。R-8.4 一致性校验通过该接口逐 SN 查。

// PathTranslator 让 mml.Service 在 fanout 前把 standardPath → privatePath。
//
// 消费者驱动小接口；实现在 cmd/app/provider/mml_adapters.go 提供薄包装：
//
//	type mmlPathTranslatorAdapter struct { products *product.Registry; params *parammodel.Registry }
//	func (a *mmlPathTranslatorAdapter) TranslateForDevice(ctx, productClass, swVer, paths) (*TranslationOutcome, error) {...}
//
// nil 时跳过翻译（向后兼容 / dev / 单测）。
//
// T-0168: 接口签名升级为返 *TranslationOutcome，把"产品解析"+"路径翻译"两步结果合一，
// 让 caller（translateTaskPaths）一次拿全任务级审计元数据（product_resolved /
// matched_product_id / aggregate_source）。
type PathTranslator interface {
	// TranslateForDevice 把 standardPaths 按 (productClass, swVersion) 翻译为 privatePath。
	//   - 内部链路：productClass → ProductRegistry → product.id → Translator.ToPrivate
	//   - 单条未命中（passthrough）时 Private = Standard，Source="passthrough"
	//   - **激进路线**（T-0168）：productClass 整批未识别时 **不**返 ErrProductClassUnresolved，
	//     而是构造 TranslationOutcome{ProductResolved=false, AggregateSource="orphan_passthrough"}，
	//     所有 Paths.Source="orphan_passthrough"，让上层不阻塞 fanout
	//   - 仅在 Registry IO / DI 错误时返 error
	TranslateForDevice(ctx context.Context, productClass, softwareVersion string, standardPaths []string) (*TranslationOutcome, error)
}

// TranslatedPath 是 Translator 的单条结果。
type TranslatedPath struct {
	Standard string `json:"standard"`
	Private  string `json:"private"`
	Source   string `json:"source"` // discovered / default / passthrough / orphan_passthrough
}

// TranslationOutcome 是 PathTranslator.TranslateForDevice 的完整返回结果（T-0168）。
//
// 它把"产品解析"与"路径翻译"两步合一返回，避免上层 caller 二次调用 ResolveProduct +
// TranslateForDevice 的冗余。translateTaskPaths 从这里抽出任务级审计元数据写到
// mml_tasks 表的 4 列（product_resolved / matched_product_id /
// matched_product_class / path_translation_source），见 migration 000171。
type TranslationOutcome struct {
	// Paths 是逐条 standardPath → privatePath 的翻译结果（与输入 standardPaths 顺序对齐）。
	Paths []TranslatedPath

	// ProductResolved 表示 productClass 是否成功匹配到 product。
	//   true  → MatchProductClass 命中，所有 Paths.Source ∈ {"discovered","default","passthrough"}
	//   false → MatchProductClass 返 ErrOrphan，所有 Paths.Source == "orphan_passthrough"
	ProductResolved bool

	// ProductID 是命中的 product.id；ProductResolved=false 时为 uuid.Nil。
	ProductID uuid.UUID

	// ProductClass 透传调用方传入的 productClass，便于上层写审计列（mml_tasks.matched_product_class）。
	ProductClass string

	// AggregateSource 是任务级翻译来源汇总（基于 Paths[].Source 推导）：
	//   - 全 discovered / 全 default / 全 passthrough / 全 orphan_passthrough → 取该单一值
	//   - 否则（任意混合）→ "mixed"
	AggregateSource string
}

// SSEPublisher defines the interface for publishing SSE events.
// Implemented by *events.MessageHub; nil means SSE is disabled.
type SSEPublisher interface {
	PublishSimple(userID, eventType string, data []byte)
}

// SSEGranularity 标识推送颗粒度。
// docs/design/mml-task-flow-design-20260424.md §6 Q6：
//   - task    —— mml_task 级推送，节流 ≤1/s（当前实现的唯一粒度）
//   - deviceTask —— device_task 级推送（粒度更细），P4 阶段先占位，实现延后
//
// Aggregator 发 SSE 时统一经 sseGranularityForExecutor(userID) 判定，
// 当前一律返回 SSEGranularityTask。未来加入 device_task 级推送时，在此处
// 改规则（例如按用户偏好查询缓存），下游 Aggregator 无需改动。
type SSEGranularity string

const (
	SSEGranularityTask       SSEGranularity = "task"
	SSEGranularityDeviceTask SSEGranularity = "device_task"
)

// NewService creates a new MML Service.
// hub may be nil if SSE is not configured.
// auditRepo may be nil if audit logging is not configured.
func NewService(
	cmdRepo CommandRepository,
	scriptRepo ScriptRepository,
	taskRepo TaskRepository,
	customCommandRepo CustomCommandRepository,
	hub SSEPublisher,
	logger *zap.Logger,
) *Service {
	return &Service{
		cmdRepo:           cmdRepo,
		scriptRepo:        scriptRepo,
		taskRepo:          taskRepo,
		customCommandRepo: customCommandRepo,
		hub:               hub,
		logger:            logger.Named("mml"),
	}
}

// SetAuditRepo sets the audit repository for command execution logging.
func (s *Service) SetAuditRepo(repo AuditRepository) {
	s.auditRepo = repo
}

// SetScriptImportService injects the server-authoritative TXT import service
// used by Handler's import routes. It is kept on Service as the module's
// composition boundary, while Handler receives the narrow API interface.
func (s *Service) SetScriptImportService(importService ScriptImportServiceAPI) {
	s.scriptImportService = importService
}

// SetScriptExecutionValidator wires the server-authoritative validator used
// by imported-script execution preflight. Keeping this as a narrow setter
// avoids changing the long-standing NewService constructor used by callers
// and tests.
func (s *Service) SetScriptExecutionValidator(validator ScriptImportValidationRunner) {
	s.scriptExecutionValidator = validator
}

// ScriptImportService returns the configured TXT import service for handler
// wiring. A nil value means the optional import capability is unavailable.
func (s *Service) ScriptImportService() ScriptImportServiceAPI {
	return s.scriptImportService
}

// SetFanouter sets the fan-out engine for creating device_tasks from MML tasks.
func (s *Service) SetFanouter(f *Fanouter) {
	s.fanouter = f
}

// SetCmdParamRepo sets the command-param relationship repository.
func (s *Service) SetCmdParamRepo(repo CommandParamRepository) {
	s.cmdParamRepo = repo
}

// SetRoleQuerier 注入 RBAC group 派生器；T-0090-c 用于 private 命令 group-share 过滤。
// 不注入时 ListCustomCommands fallback 仅 creator-self 可见（即 T-0090-c 之前的行为）。
func (s *Service) SetRoleQuerier(rq RoleQuerier) {
	s.roleQuerier = rq
}

// SetCustomCommandSupportedPathsResolver 注入产品参数模型支持 path 解析器。
// 解析失败时控制台列表直接报错，避免回退展示跨产品模板。
func (s *Service) SetCustomCommandSupportedPathsResolver(
	resolver func(ctx context.Context, productID uuid.UUID) (map[string]struct{}, error),
) {
	s.customCommandSupportedPaths = resolver
}

// SetDeviceLookup 注入设备查询适配器，启用 R-8.4 product_class 一致性校验。
// nil 时跳过（向后兼容）。
// SetPathNameResolver 注入按请求语言解析 standardPath → 友好名，供 CSV 导出使用。
func (s *Service) SetPathNameResolver(fn func(ctx context.Context, paths []string) (map[string]string, error)) {
	s.pathNameResolver = fn
}

func (s *Service) SetDeviceLookup(d DeviceLookup) {
	s.deviceLookup = d
}

// SetPathTranslator 注入 R-9.3 path 翻译适配器（per-device standardPath → privatePath）。
// nil 时跳过翻译（向后兼容 / dev / 单测）。
func (s *Service) SetPathTranslator(t PathTranslator) {
	s.pathTranslator = t
}

// publishTaskStatus pushes a task status change event via SSE.
func (s *Service) publishTaskStatus(executor, taskID, oldStatus, newStatus string) {
	data, _ := json.Marshal(map[string]string{
		"task_id":    taskID,
		"old_status": oldStatus,
		"new_status": newStatus,
		"executor":   executor,
	})
	s.hub.PublishSimple(executor, "mml_task_status", data)
}

// ---- Command operations (read-only) ----

// ListCommands returns a paginated list of predefined MML commands, enriched with params.
func (s *Service) ListCommands(ctx context.Context, filter CommandFilter) (*model.ListResponse[MMLCommand], error) {
	resp, err := s.cmdRepo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	if s.cmdParamRepo != nil && len(resp.Items) > 0 {
		ids := make([]uuid.UUID, len(resp.Items))
		for i, cmd := range resp.Items {
			ids[i] = cmd.ID
		}
		paramMap, err := s.cmdParamRepo.ListByCommandIDs(ctx, ids)
		if err != nil {
			s.logger.Warn("failed to load command params, skipping enrichment", zap.Error(err))
		} else {
			for i := range resp.Items {
				if params, ok := paramMap[resp.Items[i].ID]; ok {
					resp.Items[i].Params = params
				}
			}
		}
	}

	return resp, nil
}

// GetCommand retrieves a predefined MML command by ID, enriched with params.
func (s *Service) GetCommand(ctx context.Context, id uuid.UUID) (*MMLCommand, error) {
	cmd, err := s.cmdRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if s.cmdParamRepo != nil {
		paramMap, err := s.cmdParamRepo.ListByCommandIDs(ctx, []uuid.UUID{id})
		if err != nil {
			s.logger.Warn("failed to load command params, skipping enrichment", zap.Error(err))
		} else if params, ok := paramMap[id]; ok {
			cmd.Params = params
		}
	}

	return cmd, nil
}

// GetCommandByCode retrieves a predefined MML command by its command code, enriched with params.
func (s *Service) GetCommandByCode(ctx context.Context, code string) (*MMLCommand, error) {
	cmd, err := s.cmdRepo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}

	if s.cmdParamRepo != nil {
		paramMap, err := s.cmdParamRepo.ListByCommandIDs(ctx, []uuid.UUID{cmd.ID})
		if err != nil {
			s.logger.Warn("failed to load command params, skipping enrichment", zap.Error(err))
		} else if params, ok := paramMap[cmd.ID]; ok {
			cmd.Params = params
		}
	}

	return cmd, nil
}

// CommandParamPath represents a TR-069 parameter path bound to an MML command.
type CommandParamPath struct {
	Path     string `json:"path"`
	Label    string `json:"label"`
	Writable bool   `json:"writable"`
}

// CommandParamPathsResponse defines the response payload for command parameter paths.
type CommandParamPathsResponse struct {
	CommandCode         string             `json:"command_code"`
	OperationType       string             `json:"operation_type"`
	SupportedOperations []string           `json:"supported_operations"`
	ParamPaths          []CommandParamPath `json:"param_paths"`
}

// GetCommandParamPaths retrieves TR-069 parameter paths for a predefined MML command.
//
// 自 migration 000090 schema 重建后：
//   - 单条命令只绑定一个 operation_type（standard-model 已为每个操作发独立 code）
//   - 路径列表存在 mml_commands.target_paths（JSONB []string）
//   - 老 supported_operations 字段已下线 — response 中保留单元素数组以兼容 FE
func (s *Service) GetCommandParamPaths(ctx context.Context, id uuid.UUID) (*CommandParamPathsResponse, error) {
	cmd, err := s.GetCommand(ctx, id)
	if err != nil {
		return nil, err
	}

	writable := isWritableOperation(cmd.OperationType)
	paramPaths := make([]CommandParamPath, 0, len(cmd.TargetPaths))
	for _, p := range cmd.TargetPaths {
		if p == "" {
			continue
		}
		paramPaths = append(paramPaths, CommandParamPath{
			Path:     p,
			Label:    p,
			Writable: writable,
		})
	}

	supportedOperations := []string{cmd.OperationType}

	return &CommandParamPathsResponse{
		CommandCode:         cmd.CommandCode,
		OperationType:       cmd.OperationType,
		SupportedOperations: supportedOperations,
		ParamPaths:          paramPaths,
	}, nil
}

func normalizeCommandParamPaths(raw json.RawMessage, operationType string, supportedOperations []string) ([]CommandParamPath, error) {
	if len(raw) == 0 {
		return []CommandParamPath{}, nil
	}

	defaultWritable := isWritableOperation(operationType)
	if !defaultWritable {
		for _, op := range supportedOperations {
			if isWritableOperation(op) {
				defaultWritable = true
				break
			}
		}
	}

	type rawCommandParamPath struct {
		Path     string `json:"path"`
		Label    string `json:"label"`
		Writable *bool  `json:"writable"`
	}

	var objectPaths []rawCommandParamPath
	if err := json.Unmarshal(raw, &objectPaths); err == nil {
		paramPaths := make([]CommandParamPath, 0, len(objectPaths))
		for _, item := range objectPaths {
			if item.Path == "" {
				continue
			}

			label := item.Label
			if label == "" {
				label = item.Path
			}

			writable := defaultWritable
			if item.Writable != nil {
				writable = *item.Writable
			}

			paramPaths = append(paramPaths, CommandParamPath{
				Path:     item.Path,
				Label:    label,
				Writable: writable,
			})
		}
		return paramPaths, nil
	}

	var stringPaths []string
	if err := json.Unmarshal(raw, &stringPaths); err != nil {
		return nil, err
	}

	paramPaths := make([]CommandParamPath, 0, len(stringPaths))
	for _, path := range stringPaths {
		if path == "" {
			continue
		}
		paramPaths = append(paramPaths, CommandParamPath{
			Path:     path,
			Label:    path,
			Writable: defaultWritable,
		})
	}
	return paramPaths, nil
}

func isWritableOperation(operation string) bool {
	switch strings.ToUpper(operation) {
	case "MOD", "ADD", "RMV", "ACT", "DEA", "RST", "CLR", "UPG":
		return true
	default:
		return false
	}
}

// ---- Script operations (CRUD) ----

// CreateScript creates a new user-defined MML script.
func (s *Service) CreateScript(ctx context.Context, script *MMLScript) (*MMLScript, error) {
	if script.Tags == nil {
		script.Tags = []string{}
	}

	if err := s.scriptRepo.Create(ctx, script); err != nil {
		return nil, fmt.Errorf("create mml script: %w", err)
	}

	s.logger.Info("mml script created",
		zap.String("script_id", script.ID.String()),
		zap.String("script_name", script.ScriptName),
	)

	return script, nil
}

// GetScript retrieves an MML script by ID.
func (s *Service) GetScript(ctx context.Context, id uuid.UUID) (*MMLScript, error) {
	return s.scriptRepo.GetByID(ctx, id)
}

// UpdateScript updates an existing MML script.
func (s *Service) UpdateScript(ctx context.Context, id uuid.UUID, script *MMLScript) (*MMLScript, error) {
	existing, err := s.scriptRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get mml script: %w", err)
	}

	// Apply updates
	existing.ScriptName = script.ScriptName
	existing.Description = script.Description
	existing.Content = script.Content
	if script.Tags != nil {
		existing.Tags = script.Tags
	}

	if err := s.scriptRepo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("update mml script: %w", err)
	}

	s.logger.Info("mml script updated",
		zap.String("script_id", id.String()),
	)

	return existing, nil
}

// UpdateScriptMetadata changes only mutable presentation metadata. Imported
// TXT content and its server-generated plan remain immutable here; replacing
// content must go through ReplaceScriptFromImport.
func (s *Service) UpdateScriptMetadata(ctx context.Context, id uuid.UUID, name, description string, tags []string) (*MMLScript, error) {
	existing, err := s.scriptRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get mml script: %w", err)
	}
	if repo, ok := s.scriptRepo.(interface {
		UpdateMetadata(context.Context, uuid.UUID, string, string, []string) error
	}); ok {
		if err := repo.UpdateMetadata(ctx, id, name, description, tags); err != nil {
			return nil, fmt.Errorf("update mml script metadata: %w", err)
		}
	} else {
		existing.ScriptName = name
		existing.Description = description
		existing.Tags = tags
		if err := s.scriptRepo.Update(ctx, existing); err != nil {
			return nil, fmt.Errorf("update mml script metadata: %w", err)
		}
	}
	// UpdateMetadata writes updated_at in PostgreSQL. Reload the row so callers
	// receive the new optimistic-concurrency version before a replacement
	// import is attempted.
	updated, err := s.scriptRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("reload mml script metadata: %w", err)
	}
	return updated, nil
}

// DeleteScript deletes an MML script by ID.
func (s *Service) DeleteScript(ctx context.Context, id uuid.UUID) error {
	return s.scriptRepo.Delete(ctx, id)
}

// ---- Script lifecycle (status machine) ----
//
// 合法迁移（仅脚本级生命周期域，不涉及 mml_tasks 创建）：
//   active / archived / pending / paused / failed / cancelled  --Start-->  running
//   running                                                   --Pause-->  paused
//   pending / running / paused                                --Cancel-> cancelled
// 已是 completed 的脚本不允许再 Start/Pause/Cancel；调用方应视作终态。

func canTransition(from ScriptStatus, to ScriptStatus) bool {
	switch to {
	case ScriptRunning: // Start
		switch from {
		case ScriptActive, ScriptArchived, ScriptPending, ScriptPaused, ScriptFailed, ScriptCancelled:
			return true
		default: // running / completed 不能再 start
			return false
		}
	case ScriptPaused: // Pause
		return from == ScriptRunning
	case ScriptCancelled: // Cancel
		switch from {
		case ScriptPending, ScriptRunning, ScriptPaused:
			return true
		default:
			return false
		}
	}
	return false
}

// StartScript 将脚本置为 running，并刷新 start_time / end_time / progress。
func (s *Service) StartScript(ctx context.Context, id uuid.UUID) (*MMLScript, error) {
	script, err := s.scriptRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get mml script: %w", err)
	}
	if !canTransition(script.Status, ScriptRunning) {
		return nil, fmt.Errorf("cannot start script in status %q", script.Status)
	}
	now := time.Now()
	script.Status = ScriptRunning
	script.StartTime = &now
	script.EndTime = nil
	script.Progress = 0
	if err := s.scriptRepo.UpdateLifecycle(ctx, script); err != nil {
		return nil, fmt.Errorf("persist script start: %w", err)
	}
	s.logger.Info("mml script started", zap.String("script_id", id.String()))
	return script, nil
}

// PauseScript 将 running 脚本置为 paused。
func (s *Service) PauseScript(ctx context.Context, id uuid.UUID) (*MMLScript, error) {
	script, err := s.scriptRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get mml script: %w", err)
	}
	if !canTransition(script.Status, ScriptPaused) {
		return nil, fmt.Errorf("cannot pause script in status %q", script.Status)
	}
	script.Status = ScriptPaused
	if err := s.scriptRepo.UpdateLifecycle(ctx, script); err != nil {
		return nil, fmt.Errorf("persist script pause: %w", err)
	}
	s.logger.Info("mml script paused", zap.String("script_id", id.String()))
	return script, nil
}

// CancelScript 将 pending/running/paused 脚本置为 cancelled，落 end_time。
func (s *Service) CancelScript(ctx context.Context, id uuid.UUID) (*MMLScript, error) {
	script, err := s.scriptRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get mml script: %w", err)
	}
	if !canTransition(script.Status, ScriptCancelled) {
		return nil, fmt.Errorf("cannot cancel script in status %q", script.Status)
	}
	now := time.Now()
	script.Status = ScriptCancelled
	script.EndTime = &now
	if err := s.scriptRepo.UpdateLifecycle(ctx, script); err != nil {
		return nil, fmt.Errorf("persist script cancel: %w", err)
	}
	s.logger.Info("mml script cancelled", zap.String("script_id", id.String()))
	return script, nil
}

// ListScripts returns a paginated list of MML scripts.
func (s *Service) ListScripts(ctx context.Context, filter ScriptFilter) (*model.ListResponse[MMLScript], error) {
	return s.scriptRepo.List(ctx, filter)
}

// ---- Task / Execution operations ----

// ExecuteRequest defines the parameters for executing an MML command.
type ExecuteRequest struct {
	CommandCode string                   `json:"command_code"`
	CommandName string                   `json:"command_name,omitempty"`
	DeviceSNs   []string                 `json:"device_sns"`
	Parameters  map[string]interface{}   `json:"parameters"`
	TaskName    string                   `json:"task_name"`
	RequestID   string                   `json:"request_id,omitempty"`
	Creator     string                   `json:"creator"`
	Executor    string                   `json:"executor,omitempty"`
	Commands    []map[string]interface{} `json:"commands"`
	PlanItems   []MMLPlanItem            `json:"plan_items"`
	ScriptID    *string                  `json:"script_id,omitempty"`
	// Immutable metadata copied from an imported script into the task snapshot.
	// These fields are populated by CreateScriptExecution and are not accepted
	// from the legacy HTTP command endpoints.
	ScriptContentSHA256     string `json:"-"`
	ScriptValidationVersion string `json:"-"`
	PreservePlanSnapshot    bool   `json:"-"`

	// Parameter path command support
	ParamPaths    []string `json:"param_paths"`
	ParamValues   []string `json:"param_values"` // 与 ParamPaths 等长，仅 MOD 时使用
	OperationType string   `json:"operation_type"`
	// ExecuteMode: ""/"whole"=整体下发(一条 RPC 含全部 path)；"single_path"=逐 PATH
	// (LST/MOD 拆成每 path 一条 command→每 path 一个 device_task/RPC，path 级成败独立)。
	ExecuteMode string `json:"execute_mode"`

	// Scheduling
	ExecuteType ExecuteType `json:"execute_type"`
	ScheduledAt *string     `json:"scheduled_at"`
	PeriodStart *string     `json:"period_start"`
	PeriodEnd   *string     `json:"period_end"`
	PeriodTime  string      `json:"period_time"`

	// Retry strategy
	OfflineRetry        bool `json:"offline_retry"`
	OfflineRetryWait    int  `json:"offline_retry_wait"`
	FailedRetry         bool `json:"failed_retry"`
	FailedRetryCount    int  `json:"failed_retry_count"`
	FailedRetryInterval int  `json:"failed_retry_interval"`
}

const (
	maxMMLTaskDevices = 200
	maxMMLPlanItems   = 2000
)

func validateMMLTaskScale(req ExecuteRequest, mode TaskExecuteMode) error {
	if len(req.PlanItems) > maxMMLPlanItems {
		return fmt.Errorf("plan item count %d exceeds limit %d: %w", len(req.PlanItems), maxMMLPlanItems, commonerrors.ErrInvalidInput)
	}

	deviceSNs := req.DeviceSNs
	if mode == TaskExecuteModeDeviceBound {
		deviceSNs = make([]string, 0, len(req.PlanItems))
		for _, item := range req.PlanItems {
			deviceSNs = append(deviceSNs, item.DeviceSN)
		}
	}
	seen := make(map[string]struct{}, len(deviceSNs))
	for _, rawSN := range deviceSNs {
		sn := strings.TrimSpace(rawSN)
		if sn != "" {
			seen[sn] = struct{}{}
		}
	}
	if len(seen) > maxMMLTaskDevices {
		return fmt.Errorf("device count %d exceeds limit %d: %w", len(seen), maxMMLTaskDevices, commonerrors.ErrInvalidInput)
	}
	return nil
}

func taskExecuteModeFromRequest(raw string, hasPlanItems bool) (TaskExecuteMode, error) {
	mode := strings.ToLower(strings.TrimSpace(raw))
	switch mode {
	case "", "whole", "single_path":
		if hasPlanItems {
			return TaskExecuteModeDeviceBound, nil
		}
		return TaskExecuteModeCommon, nil
	case string(TaskExecuteModeCommon):
		if hasPlanItems {
			return "", fmt.Errorf("plan_items cannot be used with execute_mode=common: %w", commonerrors.ErrInvalidInput)
		}
		return TaskExecuteModeCommon, nil
	case string(TaskExecuteModeDeviceBound):
		return TaskExecuteModeDeviceBound, nil
	default:
		if hasPlanItems {
			return "", fmt.Errorf("unsupported execute_mode %q for plan_items: %w", raw, commonerrors.ErrInvalidInput)
		}
		return TaskExecuteModeCommon, nil
	}
}

func cloneCommandMap(in map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func commandString(entry map[string]interface{}, key string) string {
	v, _ := entry[key].(string)
	return strings.TrimSpace(v)
}

func commandParameters(entry map[string]interface{}) map[string]interface{} {
	raw, ok := entry["parameters"].(map[string]interface{})
	if ok {
		return raw
	}
	return nil
}

func formatMMLParameterValue(value interface{}) string {
	s := strings.TrimSpace(fmt.Sprint(value))
	if s == "" {
		return "{}"
	}
	if strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}") {
		return s
	}
	return "{" + s + "}"
}

func formatMMLParameterList(params map[string]interface{}) string {
	if len(params) == 0 {
		return ""
	}
	keys := make([]string, 0, len(params))
	for key := range params {
		key = strings.TrimSpace(key)
		if key != "" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, formatMMLParameterValue(params[key])))
	}
	return strings.Join(parts, ",")
}

func stripDeviceSuffixFromMML(rawLine, deviceSN string) string {
	line := strings.TrimSpace(rawLine)
	if line == "" {
		return ""
	}
	if indexes, err := topLevelIndexes(line, ';'); err == nil && len(indexes) > 0 {
		last := indexes[len(indexes)-1]
		commandPart := strings.TrimSpace(line[:last])
		devicePart := strings.TrimSpace(line[last+1:])
		for _, sn := range strings.Split(devicePart, ",") {
			if strings.TrimSpace(sn) == strings.TrimSpace(deviceSN) {
				return commandPart
			}
		}
	}
	line = strings.TrimSuffix(line, ";")
	return strings.TrimSpace(line)
}

func formatMMLScriptForResult(command map[string]interface{}, rawLine, deviceSN string) string {
	if script := stripDeviceSuffixFromMML(rawLine, deviceSN); script != "" {
		return script
	}
	code := commandString(command, "command_code")
	if code == "" {
		return ""
	}
	if params := formatMMLParameterList(commandParameters(command)); params != "" {
		return code + ":" + params
	}
	return code
}

func uniqueDeviceSNsFromPlanItems(items []MMLPlanItem) []string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		sn := strings.TrimSpace(item.DeviceSN)
		if sn == "" {
			continue
		}
		if _, ok := seen[sn]; ok {
			continue
		}
		seen[sn] = struct{}{}
		out = append(out, sn)
	}
	return out
}

func (s *Service) normalizePlanItems(ctx context.Context, items []MMLPlanItem) ([]MMLPlanItem, []map[string]interface{}, []string, error) {
	if len(items) == 0 {
		return nil, nil, nil, fmt.Errorf("plan_items required for device_bound task: %w", commonerrors.ErrInvalidInput)
	}

	normalized := make([]MMLPlanItem, 0, len(items))
	commands := make([]map[string]interface{}, 0, len(items))
	nextOrderBySN := make(map[string]int, len(items))
	ordersBySN := make(map[string]map[int]struct{}, len(items))

	for idx, item := range items {
		sn := strings.TrimSpace(item.DeviceSN)
		if sn == "" {
			return nil, nil, nil, fmt.Errorf("plan_items[%d].device_sn required: %w", idx, commonerrors.ErrInvalidInput)
		}

		lineNo := item.LineNo
		if lineNo <= 0 {
			lineNo = idx + 1
		}

		order := item.Order
		if order <= 0 {
			order = nextOrderBySN[sn] + 1
		}
		byOrder := ordersBySN[sn]
		if byOrder == nil {
			byOrder = map[int]struct{}{}
			ordersBySN[sn] = byOrder
		}
		if _, exists := byOrder[order]; exists {
			return nil, nil, nil, fmt.Errorf("plan_items[%d] duplicate order %d for device %s: %w", idx, order, sn, commonerrors.ErrInvalidInput)
		}
		byOrder[order] = struct{}{}
		if order > nextOrderBySN[sn] {
			nextOrderBySN[sn] = order
		}

		baseCommand := cloneCommandMap(item.Command)
		if item.CommandCode != "" && commandString(baseCommand, "command_code") == "" {
			baseCommand["command_code"] = strings.TrimSpace(item.CommandCode)
		}
		if item.OperationType != "" && commandString(baseCommand, "operation_type") == "" {
			baseCommand["operation_type"] = strings.TrimSpace(item.OperationType)
		}
		if item.Parameters != nil {
			if _, exists := baseCommand["parameters"]; !exists {
				baseCommand["parameters"] = item.Parameters
			}
		}
		if _, exists := baseCommand["parameters"]; !exists {
			baseCommand["parameters"] = map[string]interface{}{}
		}
		if commandString(baseCommand, "command_code") == "" && commandString(baseCommand, "rpc_method") == "" {
			return nil, nil, nil, fmt.Errorf("plan_items[%d].command.command_code required: %w", idx, commonerrors.ErrInvalidInput)
		}

		planCommands := []map[string]interface{}{baseCommand}
		if isRawPathCommandEntry(baseCommand) {
			var err error
			planCommands, err = normalizeRawPathCommandEntry(baseCommand)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("plan_items[%d].command: %w", idx, err)
			}
		} else {
			attachObjectNameParamFromTarget(baseCommand)
		}
		if len(planCommands) == 0 {
			return nil, nil, nil, fmt.Errorf("plan_items[%d].command produced no executable commands: %w", idx, commonerrors.ErrInvalidInput)
		}

		firstCommandIndex := len(commands)
		for subIdx, command := range planCommands {
			command["plan_line_no"] = lineNo
			command["plan_device_sn"] = sn
			command["plan_order"] = order
			command["plan_sort_order"] = order*planOrderScale + subIdx
			if raw := strings.TrimSpace(item.RawLine); raw != "" {
				command["plan_raw_line"] = raw
			}
			commands = append(commands, command)
		}

		normalized = append(normalized, MMLPlanItem{
			LineNo:   lineNo,
			DeviceSN: sn,
			Order:    order,
			RawLine:  item.RawLine,
			Command:  commands[firstCommandIndex],
		})
	}

	commands = s.resolveRPCMethods(ctx, commands)
	for i := range normalized {
		for cmdIdx, cmd := range commands {
			if commandInt(cmd, "plan_line_no") == normalized[i].LineNo &&
				commandString(cmd, "plan_device_sn") == normalized[i].DeviceSN &&
				commandInt(cmd, "plan_order") == normalized[i].Order {
				normalized[i].Command = commands[cmdIdx]
				break
			}
		}
	}
	return normalized, commands, uniqueDeviceSNsFromPlanItems(normalized), nil
}

func applyExecuteSchedule(task *MMLTask, req ExecuteRequest) error {
	scheduledAt, err := parseExecuteTime("scheduled_at", req.ScheduledAt)
	if err != nil {
		return err
	}
	periodStart, err := parseExecuteTime("period_start", req.PeriodStart)
	if err != nil {
		return err
	}
	periodEnd, err := parseExecuteTime("period_end", req.PeriodEnd)
	if err != nil {
		return err
	}

	task.ScheduledAt = scheduledAt
	task.PeriodStart = periodStart
	task.PeriodEnd = periodEnd
	task.PeriodTime = strings.TrimSpace(req.PeriodTime)

	switch req.ExecuteType {
	case ExecuteScheduled:
		if task.ScheduledAt == nil {
			return fmt.Errorf("scheduled_at required for scheduled task: %w", commonerrors.ErrInvalidInput)
		}
	case ExecutePeriodic:
		if task.PeriodStart == nil {
			return fmt.Errorf("period_start required for periodic task: %w", commonerrors.ErrInvalidInput)
		}
		if task.PeriodEnd == nil {
			return fmt.Errorf("period_end required for periodic task: %w", commonerrors.ErrInvalidInput)
		}
		if task.PeriodTime == "" {
			return fmt.Errorf("period_time required for periodic task: %w", commonerrors.ErrInvalidInput)
		}
		if _, _, _, err := parsePeriodTime(task.PeriodTime); err != nil {
			return fmt.Errorf("invalid period_time %q: %w", task.PeriodTime, commonerrors.ErrInvalidInput)
		}
		if task.PeriodEnd.Before(*task.PeriodStart) {
			return fmt.Errorf("period_end must be after period_start: %w", commonerrors.ErrInvalidInput)
		}
		if computeNextPeriodicTrigger(task, time.Now()) == nil {
			return fmt.Errorf("periodic schedule has no future trigger: %w", commonerrors.ErrInvalidInput)
		}
	}

	return nil
}

func parseExecuteTime(field string, raw *string) (*time.Time, error) {
	if raw == nil {
		return nil, nil
	}
	value := strings.TrimSpace(*raw)
	if value == "" {
		return nil, nil
	}

	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return &parsed, nil
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return &parsed, nil
	}

	for _, layout := range []string{"2006-01-02T15:04:05", "2006-01-02 15:04:05"} {
		if parsed, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return &parsed, nil
		}
	}

	return nil, fmt.Errorf("invalid %s %q: %w", field, value, commonerrors.ErrInvalidInput)
}

// ExecuteGroupRequest 批量执行某 mml_param_group 下所有命令的入参
// （Sprint B Q-V3-1 决议）。
type ExecuteGroupRequest struct {
	GroupID    uuid.UUID              `json:"group_id"`
	DeviceSNs  []string               `json:"device_sns"`
	Parameters map[string]interface{} `json:"parameters,omitempty"` // 可选，传入下游每条 command
	TaskName   string                 `json:"task_name,omitempty"`
	Creator    string                 `json:"creator,omitempty"`
	Executor   string                 `json:"executor,omitempty"`
	// 仅过滤特定 operation_type（如只跑 LST，跳过 MOD/ADD/RMV）；
	// 空数组 = 跑全部
	OperationFilter []string    `json:"operation_filter,omitempty"`
	ExecuteType     ExecuteType `json:"execute_type,omitempty"`
	ScheduledAt     *string     `json:"scheduled_at,omitempty"`
}

// ExecuteGroup 把 group 下的全部命令展开为一个 MML task 执行（Sprint B Q-V3-1）。
//
// 语义：选一个 group 等于一键发起该 group 下所有 mml_commands 各自的 RPC。
// 多命令在底层共享 mml_task.commands 数组，fanout + sequencer 自动串行展开。
//
// 失败语义：group 不存在或下无命令 → error；devices 为空 → error。
func (s *Service) ExecuteGroup(ctx context.Context, req ExecuteGroupRequest) (*MMLTask, error) {
	// 全零 UUID 是合法 UUID 格式但绝不对应任何真实 group —— 等同「group 不存在」，
	// 翻 404 而非误导性的「group_id required」（参数其实已由 URL path 提供）。
	if req.GroupID == uuid.Nil {
		return nil, fmt.Errorf("group %s not found: %w", req.GroupID, commonerrors.ErrNotFound)
	}
	if len(req.DeviceSNs) == 0 {
		return nil, fmt.Errorf("device_sns must be non-empty: %w", commonerrors.ErrInvalidInput)
	}

	cmds, err := s.cmdRepo.ListByGroupID(ctx, req.GroupID)
	if err != nil {
		return nil, fmt.Errorf("list commands by group %s: %w", req.GroupID, err)
	}
	// group 不存在或下无任何命令 —— Service 无独立 group 仓储,二者同归「group 不可执行」,
	// 翻 404 而非裸 500。真实 group 必带命令(catalog 字典/admin 建组语义)。
	if len(cmds) == 0 {
		return nil, fmt.Errorf("group %s not found or has no commands: %w", req.GroupID, commonerrors.ErrNotFound)
	}

	// 过滤 operation_type（如指定）
	filtered := cmds
	if len(req.OperationFilter) > 0 {
		allowed := make(map[string]struct{}, len(req.OperationFilter))
		for _, op := range req.OperationFilter {
			allowed[op] = struct{}{}
		}
		filtered = filtered[:0]
		for _, c := range cmds {
			if _, ok := allowed[c.OperationType]; ok {
				filtered = append(filtered, c)
			}
		}
		if len(filtered) == 0 {
			return nil, fmt.Errorf("group %s: no commands match operation_filter=%v", req.GroupID, req.OperationFilter)
		}
	}

	// 构造 ExecuteRequest commands 数组：每个 command 一个 entry
	commands := make([]map[string]interface{}, 0, len(filtered))
	for _, c := range filtered {
		entry := map[string]interface{}{
			"command_code":   c.CommandCode,
			"rpc_method":     c.RPCMethod,
			"operation_type": c.OperationType,
			"parameters":     req.Parameters,
		}
		s.attachParamRefs(ctx, entry, c.ID)
		commands = append(commands, entry)
	}

	// 透传到 ExecuteCommand 走标准 fanout 链路
	execReq := ExecuteRequest{
		Commands:    commands,
		DeviceSNs:   req.DeviceSNs,
		Parameters:  req.Parameters,
		TaskName:    req.TaskName,
		Creator:     req.Creator,
		Executor:    req.Executor,
		ExecuteType: req.ExecuteType,
		ScheduledAt: req.ScheduledAt,
	}
	if execReq.TaskName == "" {
		execReq.TaskName = fmt.Sprintf("group:%s (%d commands)", req.GroupID, len(filtered))
	}
	if execReq.ExecuteType == "" {
		execReq.ExecuteType = ExecuteImmediate
	}
	return s.ExecuteCommand(ctx, execReq)
}

// ExecuteCommand creates an MML task with pending status.
// Real execution through cmdQueue to ACS is a future integration.
func (s *Service) ExecuteCommand(ctx context.Context, req ExecuteRequest) (*MMLTask, error) {
	taskExecuteMode, err := taskExecuteModeFromRequest(req.ExecuteMode, len(req.PlanItems) > 0)
	if err != nil {
		return nil, err
	}
	if err := validateMMLTaskScale(req, taskExecuteMode); err != nil {
		return nil, err
	}

	// Build the commands list from the request
	commands := req.Commands
	if commands == nil {
		commands = []map[string]interface{}{}
	}

	// If a script_id is provided, save it in the task and optionally parse content.
	// BUG-06 fix (#706)：前端在脚本「执行」Drawer 中同时发送 script_id 和
	// 前端本地解析好的 commands 数组时，只需保存 script_id 引用即可，
	// 不应再次解析脚本内容（否则会产生重复命令）。
	// 当 commands 为空时（仅传 script_id 无 commands），才从库读取脚本内容展开。
	var scriptID *uuid.UUID
	if req.ScriptID != nil && *req.ScriptID != "" {
		sid, err := uuid.Parse(*req.ScriptID)
		if err != nil {
			return nil, fmt.Errorf("parse script_id %q: %w", *req.ScriptID, err)
		}
		scriptID = &sid
		// 只在调用方未提供 commands 时，才从库解析脚本内容（向后兼容 /mml/execute 直传 script_id）
		if taskExecuteMode == TaskExecuteModeCommon && len(commands) == 0 {
			script, err := s.scriptRepo.GetByID(ctx, sid)
			if err != nil {
				return nil, fmt.Errorf("resolve script %s: %w", sid, err)
			}
			// Parse script content into structured command entries (one command per line).
			lines, err := ParseScriptContent(script.Content)
			if err != nil {
				return nil, fmt.Errorf("parse script %s content: %w", sid, err)
			}
			for _, line := range lines {
				commands = append(commands, map[string]interface{}{
					"command_code":   line.CommandCode,
					"parameters":     scriptLineParameters(line),
					"operation_type": deriveOperationType(line.CommandCode),
					"source":         "script",
					"script_name":    script.ScriptName,
				})
			}
		}
	}
	if taskExecuteMode == TaskExecuteModeCommon && len(commands) > maxMMLPlanItems {
		return nil, fmt.Errorf("command row count %d exceeds limit %d: %w", len(commands), maxMMLPlanItems, commonerrors.ErrInvalidInput)
	}

	var planItems []MMLPlanItem
	deviceSNs := req.DeviceSNs
	if taskExecuteMode == TaskExecuteModeDeviceBound {
		var err error
		planItems, commands, deviceSNs, err = s.normalizePlanItems(ctx, req.PlanItems)
		if err != nil {
			return nil, err
		}
	} else {
		// If a command_code is provided, resolve it and build the command entry.
		if req.CommandCode != "" {
			cmd, err := s.cmdRepo.GetByCode(ctx, req.CommandCode)
			if err != nil {
				// Sprint B-6 fallback：standard-model 重建后部分老 command_code 已下线，
				// FE Console 保存的 mml_custom_command 可能仍引用孤儿码。此时不阻塞任务
				// 创建——退化为透传 entry，下游 resolveRPCMethods / fanout 会再次尝试
				// 解析；若彻底无法识别，Fanouter 内部会跳过对应 device 任务并在审计
				// 日志中留下痕迹，比直接 500 对用户友好得多。
				if errors.Is(err, commonerrors.ErrNotFound) {
					s.logger.Warn("mml execute: command_code not in mml_commands, degrading to raw passthrough",
						zap.String("command_code", req.CommandCode),
						zap.String("operation_type", req.OperationType),
					)
					entry := map[string]interface{}{
						"command_code": req.CommandCode,
						"parameters":   req.Parameters,
						"orphan":       true, // 标记孤儿，便于审计 / FE 提示
					}
					if name := strings.TrimSpace(req.CommandName); name != "" {
						entry["command_name"] = name
					}
					if len(req.ParamPaths) > 0 {
						entry["param_paths"] = req.ParamPaths
					}
					if req.OperationType != "" {
						entry["operation_type"] = req.OperationType
					}
					commands = append(commands, entry)
				} else {
					return nil, fmt.Errorf("resolve command code %q: %w", req.CommandCode, err)
				}
			} else {
				entry := map[string]interface{}{
					"command_code": cmd.CommandCode,
					"rpc_method":   cmd.RPCMethod,
					"parameters":   req.Parameters,
					"command_name": localizedCommandName(ctx, cmd),
				}
				if len(req.ParamPaths) > 0 {
					entry["param_paths"] = req.ParamPaths
				}
				if req.OperationType != "" {
					entry["operation_type"] = req.OperationType
				}
				s.attachParamRefs(ctx, entry, cmd.ID)
				commands = append(commands, entry)
			}
		} else if len(req.ParamPaths) > 0 {
			// 裸路径模式：用户没在命令树选命令，直接在"参数路径指定"面板敲了 N 个 TR-069 路径。
			// 支持 LST/DSP/MOD/ADD/RMV 五种操作类型；param_refs 在此处按 op 合成
			// 最小集合，下游 BuildTR069Params 据此构造 wire 格式 SOAP body。
			op := strings.ToUpper(strings.TrimSpace(req.OperationType))
			if op == "" {
				op = "LST"
			}

			// 过滤空路径并对齐 param_values（按下标平行）。
			paths := make([]string, 0, len(req.ParamPaths))
			values := make([]string, 0, len(req.ParamPaths))
			for i, p := range req.ParamPaths {
				t := strings.TrimSpace(p)
				if t == "" {
					continue
				}
				paths = append(paths, t)
				if i < len(req.ParamValues) {
					values = append(values, req.ParamValues[i])
				} else {
					values = append(values, "")
				}
			}
			if len(paths) == 0 {
				return nil, fmt.Errorf("raw param_paths mode: all paths empty")
			}

			switch op {
			case "LST", "DSP":
				if req.ExecuteMode == "single_path" {
					// 逐 PATH：每 path 一条 GetParameterValues command → 每 path 一个 device_task/RPC，
					// path 级成败独立（某 path 9005 不连累其它 path）。
					for _, p := range paths {
						commands = append(commands, map[string]interface{}{
							"command_code":   "RAW " + op,
							"rpc_method":     "GetParameterValues",
							"operation_type": op,
							"param_paths":    []string{p},
							"param_refs":     []MMLParamRef{{Tr069Path: p, ValueType: "string"}},
							"command_name":   strings.TrimSpace(req.CommandName),
						})
					}
					break
				}
				synthRefs := make([]MMLParamRef, len(paths))
				for i, p := range paths {
					synthRefs[i] = MMLParamRef{Tr069Path: p, ValueType: "string"}
				}
				commands = append(commands, map[string]interface{}{
					"command_code":   "RAW " + op,
					"rpc_method":     "GetParameterValues",
					"operation_type": op,
					"param_paths":    paths,
					"param_refs":     synthRefs,
					"command_name":   strings.TrimSpace(req.CommandName),
				})
			case "MOD":
				// SetParameterValues 一定要有非空值。让 ParamCode == Tr069Path，
				// 这样 buildParameterValues 可以按 ParamCode 索引到 Tr069Path。
				for i, v := range values {
					if strings.TrimSpace(v) == "" {
						return nil, fmt.Errorf("raw param_paths MOD: param_values[%d] is empty for path %q", i, paths[i])
					}
				}
				if req.ExecuteMode == "single_path" {
					// 逐 PATH：每 path 一条 SetParameterValues command → path 级成败独立。
					for i, p := range paths {
						commands = append(commands, map[string]interface{}{
							"command_code":   "RAW MOD",
							"rpc_method":     "SetParameterValues",
							"operation_type": op,
							"param_paths":    []string{p},
							"param_refs":     []MMLParamRef{{ParamCode: p, Tr069Path: p, ValueType: "string"}},
							"parameters":     map[string]interface{}{p: values[i]},
							"command_name":   strings.TrimSpace(req.CommandName),
						})
					}
				} else {
					synthRefs := make([]MMLParamRef, len(paths))
					formValues := make(map[string]interface{}, len(paths))
					for i, p := range paths {
						synthRefs[i] = MMLParamRef{ParamCode: p, Tr069Path: p, ValueType: "string"}
						formValues[p] = values[i]
					}
					commands = append(commands, map[string]interface{}{
						"command_code":   "RAW MOD",
						"rpc_method":     "SetParameterValues",
						"operation_type": op,
						"param_paths":    paths,
						"param_refs":     synthRefs,
						"parameters":     formValues,
						"command_name":   strings.TrimSpace(req.CommandName),
					})
				}
				// #196：MOD 后自动追加一条 LST 回读，核实基站是否真的改成功（自定义 / 指定参数 PATH
				// 通道，与结构化通道 buildStatementCommandEntries 的 MOD 回读对齐）。回读经 Sequencer
				// 在 SPV 完成后顺序执行（下方 fanout 检测到 lst_after_mod 即启用 sequential 模式）。
				commands = append(commands, buildRawReadbackLSTCommand(paths))
			case "ADD":
				// TR-069 AddObject 单次仅作用于 ONE object_name，多 path 在协议层
				// 没有"批量"语义。前端已锁单行，这里再做一次防御以拒绝来自脚本/
				// 直接 API 调用的异常输入。buildObjectName 会自动补尾点。
				if len(paths) > 1 {
					return nil, fmt.Errorf("raw param_paths ADD: TR-069 AddObject only accepts one object path per call, got %d", len(paths))
				}
				commands = append(commands, map[string]interface{}{
					"command_code":   "RAW ADD",
					"rpc_method":     "AddObject",
					"operation_type": op,
					"param_paths":    paths,
					"parameters":     map[string]interface{}{"object_name": paths[0]},
					"command_name":   strings.TrimSpace(req.CommandName),
				})
			case "RMV", "DEL":
				if len(paths) > 1 {
					return nil, fmt.Errorf("raw param_paths %s: TR-069 DeleteObject only accepts one object path per call, got %d", op, len(paths))
				}
				commands = append(commands, map[string]interface{}{
					"command_code":   "RAW " + op,
					"rpc_method":     "DeleteObject",
					"operation_type": op,
					"param_paths":    paths,
					"parameters":     map[string]interface{}{"object_name": paths[0]},
					"command_name":   strings.TrimSpace(req.CommandName),
				})
			default:
				return nil, fmt.Errorf("raw param_paths mode: unsupported operation_type %q (allowed: LST/DSP/MOD/ADD/RMV)", req.OperationType)
			}
		}

		// rpc_method 补齐：前端或脚本入口的 commands 可能只带 command_code，
		// 没有 rpc_method。Fanouter 依据 rpc_method 决定 device_task 的 Method 字段，
		// 缺失则该条 command 在 fanout 时被跳过，整个 task 对那些设备没有实际效果。
		// 此处按 command_code 逐条查库补齐，无法解析的（比如纯原始 MML 行）至少
		// 保留 command_code 供后续手工诊断（to-do-list #4）。
		commands = s.resolveRPCMethods(ctx, commands)
	}

	// Service 层兜底：ExecuteType 为空时默认立即执行（handler 已有默认值，
	// 这里防其他内部调用方漏传）。否则下面扇出判据
	// `ExecuteType == ExecuteImmediate` 失败 → device_tasks 无法派生。
	if req.ExecuteType == "" {
		req.ExecuteType = ExecuteImmediate
	}

	task := &MMLTask{
		TaskName:                req.TaskName,
		RequestID:               strings.TrimSpace(req.RequestID),
		ScriptID:                scriptID,
		DeviceSNs:               deviceSNs,
		Commands:                commands,
		ExecuteMode:             taskExecuteMode,
		PlanItems:               planItems,
		ScriptContentSHA256:     req.ScriptContentSHA256,
		ScriptValidationVersion: req.ScriptValidationVersion,
		Status:                  TaskPending,
		Results:                 []map[string]interface{}{},
		Creator:                 req.Creator,
		Executor:                req.Executor,

		ExecuteType:         req.ExecuteType,
		OfflineRetry:        req.OfflineRetry,
		OfflineRetryWait:    req.OfflineRetryWait,
		FailedRetry:         req.FailedRetry,
		FailedRetryCount:    req.FailedRetryCount,
		FailedRetryInterval: req.FailedRetryInterval,
		TotalDevices:        len(deviceSNs),
	}
	if req.PreservePlanSnapshot {
		task.PlanItems = cloneScriptPlanItems(req.PlanItems)
	}

	if err := applyExecuteSchedule(task, req); err != nil {
		return nil, err
	}

	// Map execute_type to initial status + next_trigger_at.
	// P2/P3（docs/design/mml-task-flow-design-20260424.md）：scheduled/periodic
	// 不再在 CreateTask 里立刻 fanout，而是留在 pending + 写 next_trigger_at，
	// 等 Scheduler 按时唤醒。suspended 保持 paused，不 fanout。
	switch req.ExecuteType {
	case ExecuteSuspended:
		task.Status = TaskPaused
	case ExecuteScheduled:
		task.Status = TaskPending
		if task.ScheduledAt != nil {
			task.NextTriggerAt = task.ScheduledAt
		}
	case ExecutePeriodic:
		task.Status = TaskPending
		if next := computeNextPeriodicTrigger(task, time.Now()); next != nil {
			task.NextTriggerAt = next
		}
	default:
		task.Status = TaskPending
	}

	if err := s.taskRepo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("create mml task: %w", err)
	}

	s.logger.Info("mml task created",
		zap.String("task_id", task.ID.String()),
		zap.String("task_name", task.TaskName),
		zap.String("execute_type", string(task.ExecuteType)),
		zap.Int("device_count", len(task.DeviceSNs)),
		zap.String("execute_mode", string(task.ExecuteMode)),
		zap.Int("plan_item_count", len(task.PlanItems)),
	)

	// Write audit log entries for each command+device combination
	s.writeAuditLogs(ctx, task)

	// Fan-out to device_tasks only for immediate execution.
	// scheduled/periodic 由 Scheduler 唤醒；suspended 需用户显式 StartTask。
	if task.Status == TaskPending && task.ExecuteType == ExecuteImmediate && s.fanouter != nil {
		// #196：含 MOD 回读复合(lst_after_mod)时必须顺序执行，确保 LST 在 SPV 之后回读到新值；
		// 否则并发扇出可能让 LST 早于 SPV 生效，读回旧值。save/set/restore 与 CreateAndFanoutTask 一致。
		sequential := task.ExecuteMode == TaskExecuteModeDeviceBound || commandsNeedSequential(commands)
		prev := s.fanouter.sequentialMode
		s.fanouter.SetSequentialMode(sequential)
		created, err := s.fanouter.Fanout(ctx, task)
		s.fanouter.SetSequentialMode(prev)
		if err != nil {
			s.logger.Error("fanout mml task failed", zap.Error(err))
			if errors.Is(err, taskpkg.ErrTaskAdmissionDenied) {
				return nil, s.failAdmissionDeniedTask(ctx, task, err)
			}
		} else if created > 0 {
			if err := s.taskRepo.UpdateStatus(ctx, task.ID, TaskRunning); err != nil {
				s.logger.Error("update mml task to running", zap.Error(err))
			}
			task.Status = TaskRunning
		}
	}

	// Push SSE event to executor
	if s.hub != nil && task.Executor != "" {
		s.publishTaskStatus(task.Executor, task.ID.String(), "", string(task.Status))
	}

	return task, nil
}

// buildRawReadbackLSTCommand 构造 #196 raw/自定义命令通道里 MOD 之后的「回读 LST」命令
// （GetParameterValues），读回刚下发的 paths，核实基站是否真的改成功。compound_phase
// 标记 lst_after_mod，供前端关联展示 + fanout 据此启用顺序执行。
func buildRawReadbackLSTCommand(paths []string) map[string]interface{} {
	refs := make([]MMLParamRef, len(paths))
	for i, p := range paths {
		refs[i] = MMLParamRef{Tr069Path: p, ValueType: "string"}
	}
	return map[string]interface{}{
		"command_code":   "RAW LST",
		"rpc_method":     "GetParameterValues",
		"operation_type": "LST",
		"param_paths":    paths,
		"param_refs":     refs,
		"compound_phase": "lst_after_mod",
	}
}

// commandsNeedSequential 判断 commands 是否含需顺序执行的复合。
// lst_after_mod 保证回读 LST 在 SPV 之后执行；spv_after_add 保证 AddObject 返回新实例号后再配置对象内参数。
func commandsNeedSequential(commands []map[string]interface{}) bool {
	for _, c := range commands {
		ph, _ := c["compound_phase"].(string)
		if ph == "lst_after_mod" || ph == "spv_after_add" {
			return true
		}
	}
	return false
}

// CreateAndFanoutTask 是 ConsoleService.ExecuteStatements 的桥接入口（T-0123-P1 S3-D2）。
// 持久化 *MMLTask 并按 sequential 选择 fanouter 模式：
//   - sequential=true → SetSequentialMode 仅入队 cmd_idx=0 device_task，
//     Sequencer 在前一条完成后追加入队，实现多 statement 严格序列
//   - sequential=false → 单 statement 默认并发扇出到 N 设备
//
// 与 ExecuteCommand 区别：调用方已编译完 commands[] entry，跳过 command_code 解析、
// resolveRPCMethods 兜底、param_paths 合成等老入口的兼容逻辑——直接 Create + Fanout。
func (s *Service) CreateAndFanoutTask(ctx context.Context, task *MMLTask, sequential bool) error {
	if task == nil {
		return fmt.Errorf("CreateAndFanoutTask: nil task")
	}
	if task.ExecuteType == "" {
		task.ExecuteType = ExecuteImmediate
	}
	if task.ExecuteMode == "" {
		task.ExecuteMode = TaskExecuteModeCommon
	}
	if task.Status == "" {
		task.Status = TaskPending
	}
	if task.Results == nil {
		task.Results = []map[string]interface{}{}
	}
	if task.TotalDevices == 0 {
		task.TotalDevices = len(task.DeviceSNs)
	}

	// 混类型（不同 product_class）不再拒绝（原 R-8.4 已解除）：每个 product_class 翻译一次、
	// 同类复用（缓存 1 分钟），逐设备实际翻译在 ACS 出队时按各自 product_class 完成（含 fallback）。
	if err := s.translateTaskPaths(ctx, task); err != nil {
		return err
	}

	if err := s.taskRepo.Create(ctx, task); err != nil {
		return fmt.Errorf("create mml task: %w", err)
	}

	s.logger.Info("mml console task created",
		zap.String("task_id", task.ID.String()),
		zap.String("task_name", task.TaskName),
		zap.Int("statement_count", len(task.Commands)),
		zap.Int("device_count", len(task.DeviceSNs)),
		zap.String("execute_mode", string(task.ExecuteMode)),
		zap.Int("plan_item_count", len(task.PlanItems)),
		zap.Bool("sequential", sequential),
	)

	s.writeAuditLogs(ctx, task)

	if task.Status == TaskPending && task.ExecuteType == ExecuteImmediate && s.fanouter != nil {
		prev := s.fanouter.sequentialMode
		s.fanouter.SetSequentialMode(sequential)
		created, err := s.fanouter.Fanout(ctx, task)
		s.fanouter.SetSequentialMode(prev)
		if err != nil {
			s.logger.Error("fanout console task failed", zap.Error(err))
			if errors.Is(err, taskpkg.ErrTaskAdmissionDenied) {
				return s.failAdmissionDeniedTask(ctx, task, err)
			}
		} else if created > 0 {
			if err := s.taskRepo.UpdateStatus(ctx, task.ID, TaskRunning); err != nil {
				s.logger.Error("update console task to running", zap.Error(err))
			}
			task.Status = TaskRunning
		}
	}

	if s.hub != nil && task.Executor != "" {
		s.publishTaskStatus(task.Executor, task.ID.String(), "", string(task.Status))
	}
	return nil
}

// fanoutClaimed 在 Scheduler 认领并把 status 改为 running 之后被调用，
// 只负责写 device_tasks。不做状态机转换（认领阶段已完成）。
// 失败时记 error 日志，不改回 pending——Scheduler 再次循环时会重试，且
// RecoverPendingTasks / RebootCloser 会兜底残留。
func (s *Service) fanoutClaimed(ctx context.Context, task *MMLTask) error {
	if s.fanouter == nil {
		return fmt.Errorf("fanouter not wired")
	}
	sequential := task.ExecuteMode == TaskExecuteModeDeviceBound || commandsNeedSequential(task.Commands)
	prev := s.fanouter.sequentialMode
	s.fanouter.SetSequentialMode(sequential)
	created, err := s.fanouter.Fanout(ctx, task)
	s.fanouter.SetSequentialMode(prev)
	if err != nil {
		return fmt.Errorf("fanout mml task: %w", err)
	}
	s.logger.Info("scheduler fanned out mml task",
		zap.String("task_id", task.ID.String()),
		zap.String("execute_type", string(task.ExecuteType)),
		zap.Int("device_tasks", created),
	)
	if s.hub != nil && task.Executor != "" {
		s.publishTaskStatus(task.Executor, task.ID.String(), string(TaskPending), string(task.Status))
	}
	return nil
}

// failAdmissionDeniedTask 将已落库、但被设备接入门禁拒绝创建 device_tasks 的
// MML 父任务收口为失败。其他 fanout 错误保持原有处理语义。
func (s *Service) failAdmissionDeniedTask(ctx context.Context, task *MMLTask, fanoutErr error) error {
	if task == nil {
		return fanoutErr
	}

	oldStatus := string(task.Status)
	now := time.Now().UTC()
	task.Status = TaskFailed
	task.Result = resultFailedPtr()
	task.FinishedAt = &now
	task.NextTriggerAt = nil
	task.FailedCount = task.TotalDevices
	task.Results = append(task.Results, map[string]interface{}{
		"code":    "MML_TASK_ADMISSION_DENIED",
		"message": fanoutErr.Error(),
	})

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return errors.Join(
			fmt.Errorf("fanout mml task: %w", fanoutErr),
			fmt.Errorf("mark mml task failed: %w", err),
		)
	}
	if s.hub != nil && task.Executor != "" {
		s.publishTaskStatus(task.Executor, task.ID.String(), oldStatus, string(TaskFailed))
	}
	return fmt.Errorf("fanout mml task: %w", fanoutErr)
}

// resolveRPCMethods 为缺少 rpc_method 的 command 条目按 command_code 查库补齐。
// 调用方保证 commands 为 []map[string]interface{}。查不到或 command_code 是
// 裸 MML 行（例如 "LST DEVICE_INFO:lstId={ver};"）时会先尝试提取首 token 再查，
// 都失败时保留条目原样，由 Fanouter 决定是否跳过。
func (s *Service) resolveRPCMethods(ctx context.Context, commands []map[string]interface{}) []map[string]interface{} {
	for _, entry := range commands {
		if method, _ := entry["rpc_method"].(string); method != "" {
			continue
		}
		rawCode, _ := entry["command_code"].(string)
		// BUG-01 fix (#706)：用户在脚本 textarea 里写 "LST DEVICE_INFO;" 带尾部分号，
		// 前端直接提交 command_code 字符串时分号会跟进来；mml_commands 字典存的是无分号形式，
		// 需要在查库前 trim，否则两次候选查找都 miss，rpc_method 留空，fanout 跳过整条命令。
		code := strings.TrimRight(strings.TrimSpace(rawCode), ";")
		code = strings.TrimSpace(code)
		if code == "" {
			continue
		}

		// 先用完整字符串查；查不到再取首段（分隔符 `:`/空格）再试。
		candidates := []string{code}
		if idx := strings.IndexAny(code, ": "); idx > 0 {
			candidates = append(candidates, strings.TrimSpace(code[:idx]))
		}
		for _, c := range candidates {
			cmd, err := s.cmdRepo.GetByCode(ctx, c)
			if err != nil || cmd == nil {
				continue
			}
			entry["rpc_method"] = cmd.RPCMethod
			// 命中后把 command_code 统一为数据库里的规范形式，方便后续审计。
			entry["command_code"] = cmd.CommandCode
			if _, hasCat := entry["category"]; !hasCat && cmd.Category != "" {
				entry["category"] = cmd.Category
			}
			s.attachObjectNameParam(entry, cmd)
			s.attachParamRefs(ctx, entry, cmd.ID)
			break
		}
	}
	return commands
}

// attachObjectNameParam 让脚本/直接 API 入口的 ADD/RMV 标准命令与控制台结构化执行保持一致：
// 用户写 "ADD FOO;" 时通常不会手填 object_name，需从命令字典 target_object 补齐。
func (s *Service) attachObjectNameParam(entry map[string]interface{}, cmd *MMLCommand) {
	if entry == nil || cmd == nil || strings.TrimSpace(cmd.TargetObject) == "" {
		return
	}
	method := strings.TrimSpace(cmd.RPCMethod)
	op := strings.ToUpper(strings.TrimSpace(cmd.OperationType))
	if op == "" {
		op = deriveOperationType(cmd.CommandCode)
	}
	if method != "AddObject" && method != "DeleteObject" && op != "ADD" && op != "RMV" && op != "DEL" {
		return
	}

	params := map[string]interface{}{}
	switch raw := entry["parameters"].(type) {
	case map[string]interface{}:
		params = raw
	case map[string]string:
		for k, v := range raw {
			params[k] = v
		}
	case nil:
	default:
		return
	}
	if existing, ok := params["object_name"].(string); ok && strings.TrimSpace(existing) != "" {
		return
	}

	targetObject := strings.TrimSpace(cmd.TargetObject)
	if !strings.HasSuffix(targetObject, ".") {
		targetObject += "."
	}
	params["object_name"] = targetObject
	entry["parameters"] = params
}

func attachObjectNameParamFromTarget(entry map[string]interface{}) {
	if entry == nil {
		return
	}
	targetObject := strings.TrimSpace(commandString(entry, "target_object"))
	if targetObject == "" {
		return
	}
	method := strings.TrimSpace(commandString(entry, "rpc_method"))
	op := strings.ToUpper(strings.TrimSpace(commandString(entry, "operation_type")))
	if method != "AddObject" && method != "DeleteObject" && op != "ADD" && op != "RMV" && op != "DEL" {
		return
	}

	params := commandParameters(entry)
	if params == nil {
		params = map[string]interface{}{}
	}
	if existing, ok := params["object_name"].(string); ok && strings.TrimSpace(existing) != "" {
		return
	}
	if !strings.HasSuffix(targetObject, ".") {
		targetObject += "."
	}
	params["object_name"] = targetObject
	entry["parameters"] = params
}

// attachParamRefs 把 mml_command_sub_fields JOIN standard_params 的结果挂到 entry 上。
// Fanouter 后续会调用 BuildTR069Params(rpcMethod, paramRefs, ...) 翻译为 TR-069
// wire 格式。失败仅记 warn，让 Fanouter 走兜底路径（透传 formValues）。
func (s *Service) attachParamRefs(ctx context.Context, entry map[string]interface{}, cmdID uuid.UUID) {
	if s.cmdParamRepo == nil {
		return
	}
	if _, exists := entry["param_refs"]; exists {
		return
	}
	refs, err := s.cmdParamRepo.ListByCommandID(ctx, cmdID)
	if err != nil {
		s.logger.Warn("attach param_refs failed",
			zap.String("command_id", cmdID.String()),
			zap.Error(err),
		)
		return
	}
	if len(refs) == 0 {
		return
	}
	entry["param_refs"] = refs
}

// writeAuditLogs creates audit log entries for a newly created task.
// Errors are logged but do not fail the task creation.
func (s *Service) writeAuditLogs(ctx context.Context, task *MMLTask) {
	if s.auditRepo == nil {
		return
	}

	creator := task.Creator
	if creator == "" {
		creator = task.Executor
	}

	var entries []*MMLAuditLog
	if task.ExecuteMode == TaskExecuteModeDeviceBound {
		for idx, cmd := range task.Commands {
			deviceSN := planDeviceSNForCommand(task, idx)
			if deviceSN == "" && idx < len(task.PlanItems) {
				deviceSN = task.PlanItems[idx].DeviceSN
			}
			if deviceSN == "" {
				continue
			}
			commandCode, _ := cmd["command_code"].(string)
			operationType, _ := cmd["operation_type"].(string)

			var params map[string]interface{}
			if p, ok := cmd["parameters"]; ok {
				if pm, ok := p.(map[string]interface{}); ok {
					params = pm
				}
			}

			var paramPaths []string
			if pp, ok := cmd["param_paths"]; ok {
				if ppSlice, ok := pp.([]string); ok {
					paramPaths = ppSlice
				}
			}

			entries = append(entries, &MMLAuditLog{
				TaskID:        &task.ID,
				CommandCode:   commandCode,
				OperationType: operationType,
				DeviceSN:      deviceSN,
				Parameters:    params,
				ParamPaths:    paramPaths,
				ResultStatus:  string(task.Status),
				Creator:       creator,
			})
		}
		if len(entries) == 0 {
			return
		}
		if err := s.auditRepo.CreateBatch(ctx, entries); err != nil {
			s.logger.Error("failed to write MML audit logs",
				zap.String("task_id", task.ID.String()),
				zap.Int("entry_count", len(entries)),
				zap.Error(err),
			)
		}
		return
	}

	for _, cmd := range task.Commands {
		commandCode, _ := cmd["command_code"].(string)
		operationType, _ := cmd["operation_type"].(string)

		var params map[string]interface{}
		if p, ok := cmd["parameters"]; ok {
			if pm, ok := p.(map[string]interface{}); ok {
				params = pm
			}
		}

		var paramPaths []string
		if pp, ok := cmd["param_paths"]; ok {
			if ppSlice, ok := pp.([]string); ok {
				paramPaths = ppSlice
			}
		}

		for _, sn := range task.DeviceSNs {
			entries = append(entries, &MMLAuditLog{
				TaskID:        &task.ID,
				CommandCode:   commandCode,
				OperationType: operationType,
				DeviceSN:      sn,
				Parameters:    params,
				ParamPaths:    paramPaths,
				ResultStatus:  string(task.Status),
				Creator:       creator,
			})
		}
	}

	if len(entries) == 0 {
		return
	}

	if err := s.auditRepo.CreateBatch(ctx, entries); err != nil {
		s.logger.Error("failed to write MML audit logs",
			zap.String("task_id", task.ID.String()),
			zap.Int("entry_count", len(entries)),
			zap.Error(err),
		)
	}
}

// GetTask retrieves an MML task by ID.
//
// Stage 3 — 聚合 device_tasks.has_path_translation_miss 填充
// PathTranslationWarning 字段供前端任务详情页显示警告标签。aggregator 未注入
// 或聚合失败均不阻塞主流程，task 仍正常返回。
func (s *Service) GetTask(ctx context.Context, id uuid.UUID) (*MMLTask, error) {
	t, err := s.taskRepo.GetByID(ctx, id)
	if err != nil || t == nil {
		return t, err
	}
	if s.deviceTaskPathMissAggregator != nil {
		stats, aggErr := s.deviceTaskPathMissAggregator.AggregatePathTranslationMissBySourceID(ctx, t.ID.String())
		if aggErr != nil {
			s.logger.Warn("aggregate path translation miss",
				zap.String("mml_task_id", t.ID.String()),
				zap.Error(aggErr))
		} else if stats.AnyMiss {
			t.PathTranslationWarning = &PathTranslationWarning{
				AnyMiss:     true,
				DeviceCount: stats.DeviceCount,
				PathCount:   stats.PathCount,
			}
		}
	}
	s.enrichCommandNames(ctx, t.Commands)
	return t, nil
}

// enrichCommandNames 给任务 commands JSONB 的每条注入友好命令名（command_name，按 command_code
// 查 mml_commands，同码缓存）。命令记录 / 执行结果重建时显示「列出 设备基本信息」而非 command_code
// 「LST DEVICE_INFO」。RAW 命令（command_code 不在 mml_commands）查不到→留空，前端回退 path 命名。
func (s *Service) enrichCommandNames(ctx context.Context, commands []map[string]interface{}) {
	if len(commands) == 0 {
		return
	}
	cache := make(map[string]string)
	for _, cmd := range commands {
		code, _ := cmd["command_code"].(string)
		// 自定义/裸路径命令的名称由执行入口写入任务快照；不能被 command_code
		// 查询失败或后续重建覆盖，否则详情页又会退回内部 RAW 命令码。
		if existing := commandString(cmd, "command_name"); existing != "" &&
			(strings.HasPrefix(strings.ToUpper(strings.TrimSpace(code)), "RAW ") || cmd["orphan"] == true) {
			continue
		}
		if code == "" {
			continue
		}
		name, cached := cache[code]
		if !cached {
			if c, err := s.cmdRepo.GetByCode(ctx, code); err == nil && c != nil {
				name = localizedCommandName(ctx, c)
			}
			cache[code] = name
		}
		if name != "" {
			cmd["command_name"] = name
		}
	}
}

// localizedCommandName 返回当前请求语言下的命令名；老数据缺少对应翻译时回退默认名称。
func localizedCommandName(ctx context.Context, command *MMLCommand) string {
	if command == nil {
		return ""
	}
	if name := strings.TrimSpace(command.CommandNameI18n[string(appcontext.GetLocale(ctx))]); name != "" {
		return name
	}
	return command.CommandName
}

// DeviceTaskPathMissAggregator 提供按 MML task ID 聚合 device_tasks 的
// has_path_translation_miss / path_translation_miss_count 接口。
// 由 task.PgTaskRepository.AggregatePathTranslationMissBySourceID 实现。
type DeviceTaskPathMissAggregator interface {
	AggregatePathTranslationMissBySourceID(ctx context.Context, sourceID string) (PathTranslationMissStatsView, error)
}

// PathTranslationMissStatsView 屏蔽 task 包的具体类型，让 mml 包不直接 import
// task 包的内部 stats 结构（消费者驱动接口）。字段语义与 task 包一致。
type PathTranslationMissStatsView struct {
	DeviceCount int
	PathCount   int64
	AnyMiss     bool
}

// SetDeviceTaskPathMissAggregator 装配 device_tasks 聚合查询（DI Setter）。
func (s *Service) SetDeviceTaskPathMissAggregator(agg DeviceTaskPathMissAggregator) {
	s.deviceTaskPathMissAggregator = agg
}

// DeviceTaskResultLister 按 mml_task.id 拉 device_tasks 执行结果列表，供任务记录
// 页"查看"modal 渲染设备级输出（device_sn / status / 错误码 / SOAP raw_response）。
//
// 消费者驱动小接口；实现端在 cmd/app/provider/modules.go 用 task.TaskService 包装。
// nil 时 GetTaskResults 回退读 mml_tasks.results JSONB（兼容老数据）。
type DeviceTaskResultLister interface {
	ListResultsBySourceID(
		ctx context.Context, sourceID string, page, pageSize int,
	) ([]DeviceTaskResultRowView, int64, error)
}

type TaskResultStatsRepository interface {
	GetResultStatsByID(ctx context.Context, id uuid.UUID) (*MMLTask, error)
}

type PeriodicChildTaskRepository interface {
	GetLatestPeriodicChild(ctx context.Context, parentID uuid.UUID) (*MMLTask, error)
}

// DeviceTaskResultRowView 屏蔽 task 包内部 struct，让 mml 包不反向 import task 包。
// 字段语义对齐 task.DeviceTaskResultRow；Result 是 device_tasks.result JSONB 原始字节。
type DeviceTaskResultRowView struct {
	DeviceTaskID string // device_tasks.id（CSV 导出「子任务ID」）
	DeviceSN     string
	Method       string
	Params       json.RawMessage
	CommandKey   string
	CWMPID       string
	Status       string
	ErrorCode    int
	ErrorMessage string
	Result       json.RawMessage
	SentAt       *time.Time
	CompletedAt  *time.Time
	CreatedAt    time.Time
	CommandIndex int
	DeviceIndex  int
}

// SetDeviceTaskResultLister 装配 device_tasks → mml task results 适配器（DI Setter）。
func (s *Service) SetDeviceTaskResultLister(l DeviceTaskResultLister) {
	s.deviceTaskResultLister = l
}

// ListRunsByScript 返回脚本关联的全部执行实例（模板 + 子实例），分页倒序。
// P4 C11：脚本详情页"历史执行"tab 的后端入口。
func (s *Service) ListRunsByScript(ctx context.Context, scriptID uuid.UUID, req model.ListRequest) (*model.ListResponse[MMLTask], error) {
	return s.taskRepo.ListByScriptID(ctx, scriptID, req)
}

// ListTasks returns a paginated list of MML tasks.
func (s *Service) ListTasks(ctx context.Context, filter TaskFilter) (*model.ListResponse[MMLTask], error) {
	return s.taskRepo.List(ctx, filter)
}

// ---- Task control operations (Phase 2) ----

var (
	// ErrInvalidTransition is returned when a task status transition is not allowed.
	ErrInvalidTransition = errors.New("invalid task status transition")
	// ErrCannotDeleteRunning is returned when trying to delete a running task.
	ErrCannotDeleteRunning = errors.New("cannot delete a running task")
)

// StartTask transitions a task from pending/paused to running.
func (s *Service) StartTask(ctx context.Context, id uuid.UUID) (*MMLTask, error) {
	task, err := s.taskRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get mml task: %w", err)
	}

	if task.Status != TaskPending && task.Status != TaskPaused {
		return nil, fmt.Errorf("start task: status %s cannot transition to running: %w", task.Status, ErrInvalidTransition)
	}

	oldStatus := string(task.Status)

	if err := s.taskRepo.UpdateStatus(ctx, id, TaskRunning); err != nil {
		return nil, fmt.Errorf("update mml task status: %w", err)
	}

	task.Status = TaskRunning

	// Fan-out to device_tasks when starting from paused
	if s.fanouter != nil {
		created, err := s.fanouter.Fanout(ctx, task)
		if err != nil {
			s.logger.Error("fanout on start failed", zap.Error(err))
			if errors.Is(err, taskpkg.ErrTaskAdmissionDenied) {
				return nil, s.failAdmissionDeniedTask(ctx, task, err)
			}
		} else {
			s.logger.Info("mml task fanned out on start",
				zap.String("task_id", id.String()),
				zap.Int("device_tasks", created))
		}
	}

	s.logger.Info("mml task started", zap.String("task_id", id.String()))

	if s.hub != nil && task.Executor != "" {
		s.publishTaskStatus(task.Executor, task.ID.String(), oldStatus, string(TaskRunning))
	}

	return task, nil
}

// PauseTask transitions a running task to paused.
func (s *Service) PauseTask(ctx context.Context, id uuid.UUID) (*MMLTask, error) {
	task, err := s.taskRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get mml task: %w", err)
	}

	if task.Status != TaskRunning {
		return nil, fmt.Errorf("pause task: status %s cannot transition to paused: %w", task.Status, ErrInvalidTransition)
	}

	oldStatus := string(task.Status)

	if err := s.taskRepo.UpdateStatus(ctx, id, TaskPaused); err != nil {
		return nil, fmt.Errorf("update mml task status: %w", err)
	}

	task.Status = TaskPaused
	s.logger.Info("mml task paused", zap.String("task_id", id.String()))

	if s.hub != nil && task.Executor != "" {
		s.publishTaskStatus(task.Executor, task.ID.String(), oldStatus, string(TaskPaused))
	}

	return task, nil
}

// CancelTask transitions any task to cancelled.
func (s *Service) CancelTask(ctx context.Context, id uuid.UUID) (*MMLTask, error) {
	task, err := s.taskRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get mml task: %w", err)
	}

	if task.Status == TaskCancelled || task.Status == TaskCompleted || task.Status == TaskFailed {
		return nil, fmt.Errorf("cancel task: status %s cannot transition to cancelled: %w", task.Status, ErrInvalidTransition)
	}

	oldStatus := string(task.Status)

	if err := s.taskRepo.UpdateStatus(ctx, id, TaskCancelled); err != nil {
		return nil, fmt.Errorf("update mml task status: %w", err)
	}

	task.Status = TaskCancelled
	s.logger.Info("mml task cancelled", zap.String("task_id", id.String()))

	if s.hub != nil && task.Executor != "" {
		s.publishTaskStatus(task.Executor, task.ID.String(), oldStatus, string(TaskCancelled))
	}

	return task, nil
}

// DeleteTask removes a non-running task.
func (s *Service) DeleteTask(ctx context.Context, id uuid.UUID) error {
	task, err := s.taskRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get mml task: %w", err)
	}

	if task.Status == TaskRunning {
		return commonerrors.NewBusinessError(
			global.ErrCodeMMLTaskRunningCannotDelete,
			ErrCannotDeleteRunning.Error(),
			fmt.Errorf("%w: %w", commonerrors.ErrAlreadyExists, ErrCannotDeleteRunning),
		)
	}

	if err := s.taskRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete mml task: %w", err)
	}

	s.logger.Info("mml task deleted", zap.String("task_id", id.String()))
	return nil
}

// ScriptLine 一条解析后的脚本行（Sprint B Q-V3-4 决议）。
//
// 支持语法：
//
//	# 注释                            ← 跳过
//	// 注释                           ← 跳过
//	LST_DEVICE_DEVICEINFO              ← 无参 command_code（兼容老脚本）
//	MOD_DEVICE_DEVICEINFO Azimuth=180  ← 带 K=V 参数
//	MOD_DEVICE_DEVICEINFO Azimuth=180 Downtilt=5   ← 多个 K=V 空格分隔
//	LST_FOO; # trailing comment ok    ← 行尾分号 + 注释保留兼容
type ScriptLine struct {
	CommandCode string            // 'LST_DEVICE_DEVICEINFO'
	Parameters  map[string]string // {"Azimuth":"180","Downtilt":"5"}; 无参时空 map
	LineNumber  int               // 原始行号（1-based），错误定位用
}

// ScriptParseError 详细的脚本解析错误，含行号 + 原因，让 FE 能精确定位坏行。
type ScriptParseError struct {
	LineNumber int
	Raw        string
	Reason     string
}

func (e *ScriptParseError) Error() string {
	return fmt.Sprintf("script line %d: %s (raw=%q)", e.LineNumber, e.Reason, e.Raw)
}

// ParseScriptContent 把脚本文本解析为结构化 []ScriptLine。
//
// Sprint B Q-V3-4 决议 fail-fast：任一行语法错误 / 重复参数 key →
// 返回 *ScriptParseError，**不返回部分结果**。整脚本要么全过要么拒收。
//
// 兼容老脚本：不带 K=V 的纯 command_code 行仍合法（Parameters 为 nil）。
func ParseScriptContent(content string) ([]ScriptLine, error) {
	var out []ScriptLine
	for idx, raw := range strings.Split(content, "\n") {
		lineNum := idx + 1
		line := strings.TrimSpace(raw)

		// strip trailing inline comments after #/// — 兼容 "LST_FOO; # comment"
		if i := indexOfComment(line); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		// trailing semicolons
		line = strings.TrimSuffix(line, ";")
		line = strings.TrimSpace(line)

		// 空行 / 纯注释 → skip
		if line == "" {
			continue
		}

		code, paramTokens := splitScriptCommandAndParams(line)
		if !isValidCommandCode(code) {
			return nil, &ScriptParseError{
				LineNumber: lineNum, Raw: raw,
				Reason: fmt.Sprintf("invalid command_code %q (allowed: uppercase letters/digits/_ with optional spaces)", code),
			}
		}

		params := make(map[string]string)
		for _, t := range paramTokens {
			eq := strings.IndexByte(t, '=')
			if eq < 0 {
				return nil, &ScriptParseError{
					LineNumber: lineNum, Raw: raw,
					Reason: fmt.Sprintf("parameter %q missing '=' (expected key=value)", t),
				}
			}
			key := strings.TrimSpace(t[:eq])
			val := strings.TrimSpace(t[eq+1:])
			if key == "" {
				return nil, &ScriptParseError{
					LineNumber: lineNum, Raw: raw,
					Reason: fmt.Sprintf("parameter %q has empty key", t),
				}
			}
			if _, dup := params[key]; dup {
				return nil, &ScriptParseError{
					LineNumber: lineNum, Raw: raw,
					Reason: fmt.Sprintf("duplicate parameter key %q in same line", key),
				}
			}
			params[key] = val
		}
		out = append(out, ScriptLine{
			CommandCode: code,
			Parameters:  params,
			LineNumber:  lineNum,
		})
	}
	return out, nil
}

func splitScriptCommandAndParams(line string) (string, []string) {
	if idx := strings.IndexByte(line, ':'); idx >= 0 {
		return strings.TrimSpace(line[:idx]), splitScriptParamTokens(line[idx+1:])
	}

	tokens := strings.Fields(line)
	firstParam := -1
	for i, token := range tokens {
		if strings.Contains(token, "=") {
			firstParam = i
			break
		}
	}
	if firstParam < 0 {
		return strings.TrimSpace(line), nil
	}
	if firstParam > 1 && !isOperationToken(tokens[0]) {
		return tokens[0], splitScriptParamTokens(strings.Join(tokens[1:], " "))
	}
	return strings.Join(tokens[:firstParam], " "), splitScriptParamTokens(strings.Join(tokens[firstParam:], " "))
}

func isOperationToken(token string) bool {
	switch strings.ToUpper(strings.TrimSpace(token)) {
	case "LST", "MOD", "ADD", "RMV", "DSP", "ACT", "DEA", "RST", "CLR", "UPG", "REBOOT", "RESET":
		return true
	default:
		return false
	}
}

func splitScriptParamTokens(input string) []string {
	var out []string
	var b strings.Builder
	braceDepth := 0
	flush := func() {
		token := strings.TrimSpace(b.String())
		if token != "" {
			out = append(out, token)
		}
		b.Reset()
	}
	for _, r := range input {
		switch r {
		case '{':
			braceDepth++
		case '}':
			if braceDepth > 0 {
				braceDepth--
			}
		}
		if braceDepth == 0 && (r == ',' || r == ' ' || r == '\t' || r == '\r' || r == '\n') {
			flush()
			continue
		}
		b.WriteRune(r)
	}
	flush()
	return out
}

func scriptLineParameters(line ScriptLine) map[string]interface{} {
	if len(line.Parameters) == 0 {
		return map[string]interface{}{}
	}
	params := make(map[string]interface{}, len(line.Parameters))
	for k, v := range line.Parameters {
		params[k] = v
	}
	return params
}

func deriveOperationType(commandCode string) string {
	op := strings.ToUpper(strings.TrimSpace(commandCode))
	if idx := strings.IndexAny(op, " _"); idx > 0 {
		op = op[:idx]
	}
	return op
}

// indexOfComment 返回该行内首个 # 或 // 的位置；-1 表示无注释。
// 简单实现：不解析引号包裹（脚本本身不会含字符串字面量），首个出现就算。
func indexOfComment(s string) int {
	hash := strings.Index(s, "#")
	slash := strings.Index(s, "//")
	switch {
	case hash < 0 && slash < 0:
		return -1
	case hash < 0:
		return slash
	case slash < 0:
		return hash
	case hash < slash:
		return hash
	default:
		return slash
	}
}

// isValidCommandCode command_code 字符集约束：大写字母/数字/_，允许空格分隔逻辑码。
// 兼容新标准命令码（如 "MOD DEVICE_INFO"）与老命令码（如 "MOD_DEVICE_INFO"）。
func isValidCommandCode(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || strings.Contains(s, "  ") {
		return false
	}
	previousSpace := false
	for _, r := range s {
		isSpace := r == ' '
		if isSpace && previousSpace {
			return false
		}
		if !((r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || isSpace) {
			return false
		}
		previousSpace = isSpace
	}
	return !previousSpace
}

// splitScriptLines 兼容老接口：调 ParseScriptContent 取 command_code 列表。
// fail-fast 错误时丢错给老调用方（panic 不合适，老 caller 也得显式处理）。
//
// 新代码请直接用 ParseScriptContent 拿到结构化 [ScriptLine] + 参数。
func splitScriptLines(content string) []string {
	lines, err := ParseScriptContent(content)
	if err != nil {
		// 老调用方（service.go:499 ExecuteCommand）没法返错；当前 fallback 行为：
		// 走 fail-fast 时 ParseScriptContent 已经拒了，外层应该用 ParseScriptContent
		// 直接返 error。本兼容函数下次清理时删除。
		return nil
	}
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		out = append(out, l.CommandCode)
	}
	return out
}

// ---- Custom Command operations (Phase 3) ----

// ListCustomCommands returns a paginated list of MML custom commands with RBAC
// visibility filtering applied (T-0090-c)：
//
//   - public 命令始终可见
//   - private 命令仅在 (creator==filter.Creator) OR (creator's group ∈ VisibleGroupIDs)
//     时可见；两条件均为空则 deny
//
// 若 filter.UserID 提供且 roleQuerier 已注入，则派生 filter.VisibleGroupIDs；
// roleQuerier 未注入时降级为仅 creator-self 可见（T-0090-c 之前的行为）。
func (s *Service) ListCustomCommands(ctx context.Context, filter CustomCommandFilter) (*model.ListResponse[MMLCustomCommand], error) {
	if s.roleQuerier != nil && filter.UserID != nil && *filter.UserID != uuid.Nil {
		groupIDs, err := s.roleQuerier.GetUserVisibleGroupIDs(ctx, *filter.UserID)
		if err != nil {
			// 派生失败不阻断查询；降级为 creator-self only（公有 + 自己创建的私有）。
			// 记录 warn 让运维可定位。
			s.logger.Warn("derive visible group IDs failed; fallback to creator-self only",
				zap.Stringer("user_id", *filter.UserID),
				zap.Error(err),
			)
		} else {
			filter.VisibleGroupIDs = groupIDs
		}
	}
	result, err := s.customCommandRepo.List(ctx, filter)
	if err != nil || filter.ProductID == nil {
		return result, err
	}
	if s.customCommandSupportedPaths == nil {
		return nil, fmt.Errorf("filter custom commands for product %s: supported paths resolver not configured", filter.ProductID)
	}
	supported, err := s.customCommandSupportedPaths(ctx, *filter.ProductID)
	if err != nil {
		return nil, fmt.Errorf("resolve supported paths for product %s: %w", filter.ProductID, err)
	}

	// 当前控制台一次拉取最多 1000 条模板；先按产品支持集合裁 path，再删除无可用
	// path 的模板，避免公有模板树残留“空节点”。
	filtered := make([]MMLCustomCommand, 0, len(result.Items))
	for _, command := range result.Items {
		paths := make([]string, 0, len(command.ParamPaths))
		for _, path := range command.ParamPaths {
			path = strings.TrimSpace(path)
			if path == "" {
				continue
			}
			if _, ok := supported[path]; ok {
				paths = append(paths, path)
			}
		}
		if len(paths) == 0 {
			continue
		}
		command.ParamPaths = paths
		filtered = append(filtered, command)
	}
	result.Items = filtered
	result.Total = int64(len(filtered))
	if result.PageSize > 0 {
		result.TotalPages = (len(filtered) + result.PageSize - 1) / result.PageSize
	} else {
		result.TotalPages = 0
	}
	return result, nil
}

// GetCustomCommand retrieves an MML custom command by ID.
func (s *Service) GetCustomCommand(ctx context.Context, id uuid.UUID) (*MMLCustomCommand, error) {
	return s.customCommandRepo.GetByID(ctx, id)
}

// isOwnerOrSuper 判定调用者是否对 cmd 有写权限。
//
// 鉴权语义（docs/design/mml-user-private-template-crud-20260520.md §4.3）：
//   - super_admin 始终放行
//   - 优先比对 owner_user_id (UUID)，是首选权威字段
//   - owner_user_id 为 NULL（migration 000134 backfill 未命中的历史脏数据）→ 回退
//     比对 creator (username)，避免老数据无法被任何人编辑/删除
func isOwnerOrSuper(cmd *MMLCustomCommand, currentUserID uuid.UUID, currentUsername string, isSuperAdmin bool) bool {
	if isSuperAdmin {
		return true
	}
	if cmd.OwnerUserID != nil && *cmd.OwnerUserID != uuid.Nil {
		return *cmd.OwnerUserID == currentUserID
	}
	return cmd.Creator != "" && cmd.Creator == currentUsername
}

// newTemplateNameDuplicatedErr 构造 17008 业务错误；wrap commonerrors.ErrAlreadyExists
// 让 HTTPStatusFromError 自动映射为 409 Conflict。
func newTemplateNameDuplicatedErr(name string) error {
	return commonerrors.NewBusinessError(
		global.ErrCodeTemplateNameDuplicated,
		"您已有同名的私有模板，请换个名字",
		fmt.Errorf("%w: %q", commonerrors.ErrAlreadyExists, name),
	)
}

// newPublicCommandNameDuplicatedErr 构造 17008 业务错误（→ 409）；公共命令对所有
// 用户可见，故 command_name 须全局唯一，文案区别于私有模板。
func newPublicCommandNameDuplicatedErr(name string) error {
	return commonerrors.NewBusinessError(
		global.ErrCodeTemplateNameDuplicated,
		"已存在同名的公共命令，请换个名字",
		fmt.Errorf("%w: %q", commonerrors.ErrAlreadyExists, name),
	)
}

// dupErrForScope 按 scope 选择重名业务错误文案（用于 DB 兜底索引 race 命中时翻译）。
func dupErrForScope(scope, name string) error {
	if scope == "public" {
		return newPublicCommandNameDuplicatedErr(name)
	}
	return newTemplateNameDuplicatedErr(name)
}

// CreateCustomCommand creates a new user-defined custom command.
//
// 私有模板用户级唯一性（docs/design/mml-user-private-template-crud-20260520.md §3 D1-D2）：
// 在 scope='private' 且 owner_user_id 已设的前提下，service pre-check 同 owner 是否
// 已有同名行；若有则返 17008（409 Conflict）。DB partial unique index 在 race 时兜底。
func (s *Service) CreateCustomCommand(ctx context.Context, cmd *MMLCustomCommand) (*MMLCustomCommand, error) {
	if cmd.Parameters == nil {
		cmd.Parameters = map[string]interface{}{}
	}
	if cmd.ParamPaths == nil {
		cmd.ParamPaths = []string{}
	}

	if cmd.CommandScope == "private" && cmd.OwnerUserID != nil && *cmd.OwnerUserID != uuid.Nil {
		exists, err := s.customCommandRepo.NameExistsForPrivate(ctx, *cmd.OwnerUserID, cmd.CommandName, nil)
		if err != nil {
			return nil, fmt.Errorf("check duplicate name: %w", err)
		}
		if exists {
			return nil, newTemplateNameDuplicatedErr(cmd.CommandName)
		}
	}

	// 公共命令全局唯一（跨所有用户）——纯查询防重，公共侧不加 DB 约束。
	if cmd.CommandScope == "public" {
		exists, err := s.customCommandRepo.NameExistsForPublic(ctx, cmd.CommandName, nil)
		if err != nil {
			return nil, fmt.Errorf("check duplicate public name: %w", err)
		}
		if exists {
			return nil, newPublicCommandNameDuplicatedErr(cmd.CommandName)
		}
	}

	if err := s.customCommandRepo.Create(ctx, cmd); err != nil {
		// 并发 race 穿过查询预检后，私有 DB 兜底索引可能抛 23505 → 翻 409。
		if isUniqueViolation(err) {
			return nil, dupErrForScope(cmd.CommandScope, cmd.CommandName)
		}
		return nil, fmt.Errorf("create mml custom command: %w", err)
	}

	s.logger.Info("mml custom command created",
		zap.String("command_id", cmd.ID.String()),
		zap.String("command_name", cmd.CommandName),
	)
	return cmd, nil
}

// UpdateCustomCommand updates an existing user-defined custom command.
//
// 鉴权（§4.3）：非创建者且非 super_admin → 403。
// 唯一性（§3）：scope='private' 且改名（或 owner 已设但本期名为新值）时，
// 检查同 owner 是否已有同名行；excludeID=id 避免误判自身。
func (s *Service) UpdateCustomCommand(
	ctx context.Context,
	id uuid.UUID,
	cmd *MMLCustomCommand,
	currentUserID uuid.UUID,
	currentUsername string,
	isSuperAdmin bool,
) (*MMLCustomCommand, error) {
	existing, err := s.customCommandRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get mml custom command: %w", err)
	}

	if !isOwnerOrSuper(existing, currentUserID, currentUsername, isSuperAdmin) {
		return nil, fmt.Errorf("only creator or super_admin can update template: %w", commonerrors.ErrForbidden)
	}
	paramPathsProvided := cmd.ParamPaths != nil

	// scope 在编辑模式下不允许变更（§2 D6）— 强制保留原值，防 UI 误传
	cmd.CommandScope = existing.CommandScope

	// 仅当 (scope='private' 且 owner 已绑定 且 name 实际改变) 才走唯一性查询，避免无意义查 DB
	if cmd.CommandScope == "private" &&
		existing.OwnerUserID != nil &&
		*existing.OwnerUserID != uuid.Nil &&
		cmd.CommandName != existing.CommandName {
		exists, dupErr := s.customCommandRepo.NameExistsForPrivate(ctx, *existing.OwnerUserID, cmd.CommandName, &id)
		if dupErr != nil {
			return nil, fmt.Errorf("check duplicate name: %w", dupErr)
		}
		if exists {
			return nil, newTemplateNameDuplicatedErr(cmd.CommandName)
		}
	}

	// 公共命令改名时校验全局唯一（排除自身）——纯查询防重。
	if cmd.CommandScope == "public" && cmd.CommandName != existing.CommandName {
		exists, dupErr := s.customCommandRepo.NameExistsForPublic(ctx, cmd.CommandName, &id)
		if dupErr != nil {
			return nil, fmt.Errorf("check duplicate public name: %w", dupErr)
		}
		if exists {
			return nil, newPublicCommandNameDuplicatedErr(cmd.CommandName)
		}
	}

	existing.CommandName = cmd.CommandName
	existing.CommandCode = cmd.CommandCode
	existing.OperationType = cmd.OperationType
	existing.CategoryGroup = cmd.CategoryGroup
	existing.Description = cmd.Description
	if cmd.Parameters != nil {
		existing.Parameters = cmd.Parameters
	}
	if cmd.ParamPaths != nil {
		existing.ParamPaths = cmd.ParamPaths
	}
	existing.ParamPathsProvided = paramPathsProvided

	if err := s.customCommandRepo.Update(ctx, existing); err != nil {
		// 并发 race 穿过查询预检后，私有 DB 兜底索引可能抛 23505 → 翻 409。
		if isUniqueViolation(err) {
			return nil, dupErrForScope(existing.CommandScope, existing.CommandName)
		}
		return nil, fmt.Errorf("update mml custom command: %w", err)
	}

	s.logger.Info("mml custom command updated",
		zap.String("command_id", id.String()),
		zap.String("actor", currentUserID.String()),
	)
	return existing, nil
}

// DeleteCustomCommand deletes an MML custom command.
//
// 鉴权（§4.3 / 修复历史 §6 G2）：
//   - 非创建者且非 super_admin → 403
//   - public / private 走同一鉴权规则；不再区分 scope（修复"删别人 private 放行"bug）
func (s *Service) DeleteCustomCommand(
	ctx context.Context,
	id uuid.UUID,
	currentUserID uuid.UUID,
	currentUsername string,
	isSuperAdmin bool,
) error {
	cmd, err := s.customCommandRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get mml custom command: %w", err)
	}

	if !isOwnerOrSuper(cmd, currentUserID, currentUsername, isSuperAdmin) {
		return fmt.Errorf("only creator or super_admin can delete template: %w", commonerrors.ErrForbidden)
	}

	if err := s.customCommandRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete mml custom command: %w", err)
	}

	s.logger.Info("mml custom command deleted",
		zap.String("command_id", id.String()),
		zap.String("actor", currentUserID.String()),
	)
	return nil
}

// CloneCustomCommand clones a public custom command as a private copy for the current user.
//
// ownerID 由 handler 从 gin context 提取（migration 000134 Phase 1 dual-write）。
// 副本名 = source.CommandName + " (副本)"；若与 owner 私有命名空间冲突 → 由 Create 走
// 唯一性检查返 17008，前端可提示用户改名重试。
func (s *Service) CloneCustomCommand(
	ctx context.Context,
	id uuid.UUID,
	currentUser string,
	ownerID *uuid.UUID,
) (*MMLCustomCommand, error) {
	source, err := s.customCommandRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get mml custom command: %w", err)
	}

	clone := &MMLCustomCommand{
		CommandName:   source.CommandName + " (副本)",
		CommandCode:   source.CommandCode,
		OperationType: source.OperationType,
		CommandScope:  "private",
		CategoryGroup: source.CategoryGroup,
		Parameters:    source.Parameters,
		ParamPaths:    source.ParamPaths,
		Description:   source.Description,
		Creator:       currentUser,
		OwnerUserID:   ownerID,
	}

	if _, err := s.CreateCustomCommand(ctx, clone); err != nil {
		return nil, err
	}

	s.logger.Info("mml custom command cloned",
		zap.String("source_id", id.String()),
		zap.String("clone_id", clone.ID.String()),
	)
	return clone, nil
}

// ErrForbidden 已下线（v1.0：统一走 commonerrors.ErrForbidden）。保留 sentinel 以
// 维持向后兼容，但新代码不应再使用。
//
// Deprecated: use commonerrors.ErrForbidden.
var ErrForbidden = errors.New("forbidden")

// ---- Dangerous command detection (Phase 2) ----

// DangerousCommand describes a command pattern that requires confirmation.
type DangerousCommand struct {
	Pattern *regexp.Regexp
	Name    string
	Desc    string
}

// DangerousCommands holds the list of command patterns requiring extra confirmation.
var DangerousCommands = []DangerousCommand{
	{regexp.MustCompile(`(?i)\bRST\b`), "重启", "此操作将重启设备，设备会暂时断开连接"},
	{regexp.MustCompile(`(?i)\bFACTORYRESET\b`), "恢复默认配置", "此操作将恢复设备出厂设置，所有配置将被清除"},
	{regexp.MustCompile(`(?i)\bCELLDEACTIVATE\b`), "小区去激活", "此操作将去激活小区，可能影响网络服务"},
	{regexp.MustCompile(`(?i)\bRFCTXOFF\b`), "关闭小区射频", "此操作将关闭小区射频发射，会影响无线信号"},
	{regexp.MustCompile(`(?i)\bCOLDREBOOT\b`), "冷重启", "此操作将执行设备冷重启，设备会完全断电重启"},
}

// ErrDangerousCommand is returned when a command matches a dangerous pattern.
var ErrDangerousCommand = errors.New("dangerous command requires confirmation")

// CheckDangerousCommand checks if a command code matches any dangerous pattern.
// Returns the matching DangerousCommand if found, or nil if safe.
func CheckDangerousCommand(commandCode string) *DangerousCommand {
	for _, dc := range DangerousCommands {
		if dc.Pattern.MatchString(commandCode) {
			return &dc
		}
	}
	return nil
}

// IsDangerousCommand checks if a command code is dangerous and requires confirmation.
func (s *Service) IsDangerousCommand(ctx context.Context, commandCode string) (*DangerousCommand, error) {
	return CheckDangerousCommand(commandCode), nil
}

// PathTranslationView 是任务路径翻译详情的单条视图（T-0168 PRD §S2.2.7）。
// 前端"任务记录列表行展开"读 GET /mml/tasks/{id}/results 响应的 stats.path_translations[]。
type PathTranslationView struct {
	StandardPath      string `json:"standard_path"`
	PrivatePath       string `json:"private_path"`
	TranslationSource string `json:"translation_source"` // discovered / default / passthrough / orphan_passthrough
	Translated        bool   `json:"translated"`         // source != "passthrough" && source != "orphan_passthrough"
}

// TaskResultsStats 是 GET /mml/tasks/{id}/results 响应的 stats 字段（T-0168）。
//
// 透出任务级翻译审计（mml_tasks 4 列）+ per-path 翻译详情（task.Commands JSONB）。
// 前端列表行展开 + product_resolved=false Banner 都从这里读。
type TaskResultsStats struct {
	PathTranslations      []PathTranslationView `json:"path_translations,omitempty"`
	ProductResolved       bool                  `json:"product_resolved"`
	MatchedProductClass   string                `json:"matched_product_class,omitempty"`
	PathTranslationSource string                `json:"path_translation_source,omitempty"`
}

// extractPathTranslations 从 task.Commands JSONB 抽出去重的 PathTranslationView 列表。
// 数据源：task.Commands[i].param_refs[] 与 task.Commands[i].translation_results。
// translateTaskPaths 已经把 standardPath / privatePath / translation_source 写入两者。
//
// MMLParamRef 的 json tag 是 "tr069_path"（不是 "param_path"）；in-memory 走 []MMLParamRef
// 直接读字段；DB JSONB 反序列化后走 []interface{} 读 "tr069_path"。
func extractPathTranslations(commands []map[string]interface{}) []PathTranslationView {
	seen := make(map[string]PathTranslationView)
	upsert := func(std, priv, src string) {
		if std == "" {
			return
		}
		if priv == "" {
			priv = std
		}
		if src == "" {
			src = "passthrough"
		}
		seen[std] = PathTranslationView{
			StandardPath:      std,
			PrivatePath:       priv,
			TranslationSource: src,
			Translated:        src != "passthrough" && src != "orphan_passthrough",
		}
	}
	for _, entry := range commands {
		// 优先读 param_refs[]（LST/MOD/ADD 选中路径）
		switch refs := entry["param_refs"].(type) {
		case []MMLParamRef:
			for _, ref := range refs {
				upsert(ref.Tr069Path, ref.PrivatePath, ref.TranslationSource)
			}
		case []interface{}:
			for _, r := range refs {
				m, ok := r.(map[string]interface{})
				if !ok {
					continue
				}
				std, _ := m["tr069_path"].(string)
				priv, _ := m["private_path"].(string)
				src, _ := m["translation_source"].(string)
				upsert(std, priv, src)
			}
		}
		// 兜底读 translation_results（部分 cmd 没 param_refs 但 translateTaskPaths 仍写了 results）
		if tr, ok := entry["translation_results"].([]interface{}); ok {
			for _, r := range tr {
				m, ok := r.(map[string]interface{})
				if !ok {
					continue
				}
				std, _ := m["standard"].(string)
				if std == "" || seen[std].StandardPath != "" {
					continue // 已被 param_refs 优先填充
				}
				priv, _ := m["private"].(string)
				src, _ := m["source"].(string)
				if priv == "" {
					priv = std
				}
				if src == "" {
					src = "passthrough"
				}
				seen[std] = PathTranslationView{
					StandardPath:      std,
					PrivatePath:       priv,
					TranslationSource: src,
					Translated:        src != "passthrough" && src != "orphan_passthrough",
				}
			}
		}
	}
	out := make([]PathTranslationView, 0, len(seen))
	for _, v := range seen {
		out = append(out, v)
	}
	// 稳定排序：按 standardPath 字母序，便于前端展示一致
	sort.Slice(out, func(i, j int) bool { return out[i].StandardPath < out[j].StandardPath })
	return out
}

// GetTaskResults returns paginated per-device execution results for a task.
//
// T-0168: 响应 stats 字段携带 TaskResultsStats（path_translations + product_resolved
// + matched_product_class + path_translation_source），供前端列表行展开使用。
func (s *Service) GetTaskResults(ctx context.Context, id uuid.UUID, page, pageSize int) (*model.ListResponse[map[string]interface{}], error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	taskMeta, resultSourceID, taskErr := s.resolveTaskResultSource(ctx, id)
	// taskErr 不阻塞主流程；找不到 task 让后续 device_tasks 查询自己处理
	stats := buildTaskResultsStats(taskMeta, taskErr)
	if taskErr == nil && taskMeta != nil {
		// 结果接口不经过 GetTask，必须在这里也注入本地化的命令名，
		// 否则任务详情页的结果行只能显示 command_code。
		s.enrichCommandNames(ctx, taskMeta.Commands)
	}

	// 优先路径：从 device_tasks 拉真实执行结果（2026-05-23 修；执行结果实际写在
	// device_tasks 表，mml_tasks.results 从未由 ACS 回写，老路径永远空）。
	if s.deviceTaskResultLister != nil {
		rows, total, err := s.deviceTaskResultLister.ListResultsBySourceID(ctx, resultSourceID.String(), page, pageSize)
		if err != nil {
			return nil, fmt.Errorf("list device task results: %w", err)
		}
		items := make([]map[string]interface{}, 0, len(rows))
		for _, row := range rows {
			items = append(items, deviceTaskRowToResultMap(row, taskMeta))
		}
		resp := model.NewListResponse(items, total, page, pageSize)
		resp.Stats = stats
		return resp, nil
	}

	// 兼容回退：装配器未注入时读老 JSONB（dev / 单测）。
	t, err := s.taskRepo.GetByID(ctx, resultSourceID)
	if err != nil {
		return nil, fmt.Errorf("get mml task: %w", err)
	}
	results := t.Results
	if results == nil {
		results = []map[string]interface{}{}
	}
	total := int64(len(results))
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > int(total) {
		start = int(total)
	}
	if end > int(total) {
		end = int(total)
	}
	items := results[start:end]
	if items == nil {
		items = []map[string]interface{}{}
	}
	resp := model.NewListResponse(items, total, page, pageSize)
	resp.Stats = stats
	return resp, nil
}

func (s *Service) resolveTaskResultSource(ctx context.Context, id uuid.UUID) (*MMLTask, uuid.UUID, error) {
	task, err := s.getTaskResultStats(ctx, id)
	if err != nil {
		return nil, id, err
	}
	if task == nil || task.ExecuteType != ExecutePeriodic || task.PeriodicParentID != nil {
		return task, id, nil
	}
	childRepo, ok := s.taskRepo.(PeriodicChildTaskRepository)
	if !ok {
		return task, id, nil
	}
	child, err := childRepo.GetLatestPeriodicChild(ctx, id)
	if err != nil {
		if errors.Is(err, commonerrors.ErrNotFound) {
			return task, id, nil
		}
		return nil, id, fmt.Errorf("get latest periodic child: %w", err)
	}
	if child == nil {
		return task, id, nil
	}
	return child, child.ID, nil
}

func (s *Service) getTaskResultStats(ctx context.Context, id uuid.UUID) (*MMLTask, error) {
	if repo, ok := s.taskRepo.(TaskResultStatsRepository); ok {
		return repo.GetResultStatsByID(ctx, id)
	}
	return s.taskRepo.GetByID(ctx, id)
}

// buildTaskResultsStats 装配 TaskResultsStats（T-0168）。taskErr 非 nil 时返回空 stats，
// 避免 GetByID 失败阻断主流程（让前端拿到 200 + 空翻译详情）。
func buildTaskResultsStats(task *MMLTask, taskErr error) *TaskResultsStats {
	if taskErr != nil || task == nil {
		return &TaskResultsStats{ProductResolved: true}
	}
	return &TaskResultsStats{
		PathTranslations:      extractPathTranslations(task.Commands),
		ProductResolved:       task.ProductResolved,
		MatchedProductClass:   task.MatchedProductClass,
		PathTranslationSource: task.PathTranslationSource,
	}
}

// deviceTaskRowToResultMap 把 device_tasks 行映射为前端 DeviceTaskResultItem 期望
// 的 snake_case map（mapBackendResult in mmlApi.ts）：
//   - device_sn / status / error_message / started_at / finished_at 直通
//   - success = (status=completed && error_code=0)
//   - raw_output = result->>'raw_response'
//   - mml_script = 用户下发的 MML 指令文本（来自 plan raw_line 或 commands[command_index]）
//   - execution_time = completed_at - sent_at（毫秒）
//
// 解析 result JSONB 失败时静默跳过该字段（不影响主流程），device_sn / status 等
// 主字段保持可用。
func deviceTaskRowToResultMap(row DeviceTaskResultRowView, task *MMLTask) map[string]interface{} {
	m := map[string]interface{}{
		"device_sn":      row.DeviceSN,
		"device_task_id": row.DeviceTaskID, // 子任务 ID（device_tasks.id），前端「PATH 列表」复制用
		"status":         row.Status,
		"error_code":     row.ErrorCode,
		"error_message":  row.ErrorMessage,
		"command_index":  row.CommandIndex,
		"device_index":   row.DeviceIndex,
		"success":        row.Status == "completed" && row.ErrorCode == 0,
	}
	if row.Method != "" {
		m["request_method"] = row.Method
	}
	if row.CommandKey != "" {
		m["request_command_key"] = row.CommandKey
	}
	if row.CWMPID != "" {
		m["request_cwmp_id"] = row.CWMPID
	}
	if len(row.Params) > 0 {
		if requestPayload, ok := decodeJSONRaw(row.Params); ok {
			m["request_payload"] = requestPayload
		}
	}
	if rawRequest := buildDeviceTaskRawRequest(row); rawRequest != "" {
		m["raw_request"] = rawRequest
	}
	var command map[string]interface{}
	var rawLine string
	if task != nil && row.CommandIndex >= 0 && row.CommandIndex < len(task.Commands) {
		command = task.Commands[row.CommandIndex]
		if commandCode := commandString(command, "command_code"); commandCode != "" {
			m["command_code"] = commandCode
		}
		if commandName := commandString(command, "command_name"); commandName != "" {
			m["command_name"] = commandName
		}
		if op := commandString(command, "operation_type"); op != "" {
			m["operation_type"] = op
		}
		if planRawLine := commandString(command, "plan_raw_line"); planRawLine != "" {
			rawLine = planRawLine
			m["plan_raw_line"] = planRawLine
		}
		if lineNo, ok := command["plan_line_no"].(float64); ok && lineNo > 0 {
			m["plan_line_no"] = int(lineNo)
		}
		if planDeviceSN := commandString(command, "plan_device_sn"); planDeviceSN != "" {
			m["plan_device_sn"] = planDeviceSN
		}
		if order, ok := command["plan_order"].(float64); ok && order > 0 {
			m["plan_order"] = int(order)
		}
	} else if task != nil && task.ExecuteMode == TaskExecuteModeDeviceBound &&
		row.CommandIndex >= 0 && row.CommandIndex < len(task.PlanItems) {
		plan := task.PlanItems[row.CommandIndex]
		command = plan.Command
		rawLine = plan.RawLine
		m["plan_line_no"] = plan.LineNo
		m["plan_device_sn"] = plan.DeviceSN
		m["plan_order"] = plan.Order
		if plan.RawLine != "" {
			m["plan_raw_line"] = plan.RawLine
		}
		if commandCode := commandString(plan.Command, "command_code"); commandCode != "" {
			m["command_code"] = commandCode
		}
		if commandName := commandString(plan.Command, "command_name"); commandName != "" {
			m["command_name"] = commandName
		}
		if op := commandString(plan.Command, "operation_type"); op != "" {
			m["operation_type"] = op
		}
	}
	// 老任务或结果查询时任务快照可能没有可用 command_index；仍从实际 RPC
	// 方法补齐最小的命令语义，让 MOD 结果不会在前端显示为空。
	if _, ok := m["operation_type"]; !ok {
		switch row.Method {
		case "SetParameterValues":
			m["operation_type"] = "MOD"
		case "GetParameterValues", "GetParameterAttributes", "GetParameterNames":
			m["operation_type"] = "LST"
		case "AddObject":
			m["operation_type"] = "ADD"
		case "DeleteObject":
			m["operation_type"] = "RMV"
		}
	}
	if _, ok := m["command_code"]; !ok {
		if op, ok := m["operation_type"].(string); ok && op != "" {
			m["command_code"] = op
		}
	}
	if script := formatMMLScriptForResult(command, rawLine, row.DeviceSN); script != "" {
		m["mml_script"] = script
	}
	if row.SentAt != nil {
		m["started_at"] = row.SentAt.Format(time.RFC3339)
	}
	if row.CompletedAt != nil {
		m["finished_at"] = row.CompletedAt.Format(time.RFC3339)
		m["timestamp"] = row.CompletedAt.Format(time.RFC3339)
	}
	if row.SentAt != nil && row.CompletedAt != nil {
		m["execution_time"] = row.CompletedAt.Sub(*row.SentAt).Milliseconds()
	}
	if len(row.Result) > 0 {
		var resObj map[string]interface{}
		if err := json.Unmarshal(row.Result, &resObj); err == nil {
			if rr, ok := resObj["raw_response"].(string); ok && rr != "" {
				m["raw_output"] = rr
			}
			// parsed_data：把整个 result JSON 透传给前端做兜底渲染（不阻塞主字段）
			m["parsed_data"] = resObj
		}
	}
	return m
}

func decodeJSONRaw(raw json.RawMessage) (interface{}, bool) {
	if len(raw) == 0 {
		return nil, false
	}
	var out interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return string(raw), true
	}
	return out, true
}

func buildDeviceTaskRawRequest(row DeviceTaskResultRowView) string {
	if row.Method == "" {
		return ""
	}
	cwmpID := row.CWMPID
	if cwmpID == "" {
		cwmpID = "<pending-cwmp-id>"
	}
	req, err := acsrpc.NewDispatcher().BuildRequest(&acsrpc.Command{
		ID:         row.DeviceTaskID,
		Method:     row.Method,
		Params:     row.Params,
		CommandKey: row.CommandKey,
	}, cwmpID)
	if err == nil && len(req) > 0 {
		return string(req)
	}
	fallback := map[string]interface{}{
		"method": row.Method,
	}
	if row.CommandKey != "" {
		fallback["command_key"] = row.CommandKey
	}
	if row.CWMPID != "" {
		fallback["cwmp_id"] = row.CWMPID
	}
	if payload, ok := decodeJSONRaw(row.Params); ok {
		fallback["params"] = payload
	}
	data, marshalErr := json.MarshalIndent(fallback, "", "  ")
	if marshalErr != nil {
		return row.Method
	}
	return string(data)
}
