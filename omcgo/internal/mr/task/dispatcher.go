package task

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/acs/transfercfg"
	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/tracing"
	devicetask "github.com/omcgo/omcgo/internal/task"
)

// SPV 参数路径常量（文档 §4 给定，全部带 {i} 占位符表示实例号）。
// {i} 在调用时按设备的 MRMgmt.Config 实例号替换，无对应实例时默认 1（设备
// 首次开启 MR 时该路径必然只有 1 个实例）。
const (
	paramMrEnable            = "Device.FAP.MRMgmt.Config.{i}.MrEnable"
	paramVendor              = "Device.FAP.MRMgmt.Config.{i}.Vendor"
	paramOmcName             = "Device.FAP.MRMgmt.Config.{i}.OmcName"
	paramMrURL               = "Device.FAP.MRMgmt.Config.{i}.MrUrl"
	paramPeriodicReport      = "Device.FAP.MRMgmt.Config.{i}.PeriodicReportInterval"
	paramUploadPeriod        = "Device.FAP.MRMgmt.Config.{i}.UploadPeriod"
	defaultMRMgmtInstanceIdx = 1
	methodSetParameterValues = "SetParameterValues"
	commandKeyOpenPrefix     = "mr-open-"
	commandKeyClosePrefix    = "mr-close-"
)

// DeviceContext 抽象"设备 SN → *model.Device"查询，由 internal/device.DeviceRepository
// 满足。Dispatcher 需要的字段：ProductClass（平台支持判断）+ FirmwareVersion
// （Translator 解析 mapping set）。抽小接口让单测用 fake 替换。
type DeviceContext interface {
	GetDevice(ctx context.Context, deviceSN string) (*model.Device, error)
}

// TranslatorResolver 抽象"按设备的 productClass + swVersion 取 Translator"。
//
// 设计参考 internal/provision/sync_pathb.go SyncService.ResolveTranslator —
// MR 模块不直接拖 provision 包依赖，而是定义同形状的小接口，由 app provider
// 装配时把 productRegistry + paramRegistry 包成 adapter 注入。
//
// 返回 (nil, false) 时 dispatcher 跳过翻译，直接下发 standardPath（fail-soft）。
type TranslatorResolver interface {
	ResolveForDevice(ctx context.Context, dev *model.Device) (*parammodel.Translator, bool)
}

type uploadAddressResolver interface {
	Resolve(ctx context.Context, deviceID uuid.UUID, direction transfercfg.TransferDirection) (transfercfg.AddressDecision, error)
}

// Dispatcher 负责把"开启/关闭 MR"动作转换成实际下发的 SPV task。
// 它与 scheduler 的关系：scheduler 决定"何时"、对"哪些 cell"动手；
// dispatcher 负责"怎么拼参数"和"如何派发"。
//
// **路径翻译**：dispatcher 持有的是 standardPath 常量（如
// `Device.FAP.MRMgmt.Config.{i}.MrEnable`）。下发前对每条 path 调
// `Translator.ToPrivate()` 翻译为设备私有路径再塞进 SPV。Translator 缺失
// 或 mapping Found=false 时降级到 standardPath（与 provision orchestrator 一致）。
type Dispatcher struct {
	cfg       appconfig.MRConfig
	enqueuer  devicetask.Enqueuer
	devices   DeviceContext
	resolver  TranslatorResolver
	platforms PlatformResolver // 可 nil → 全部 unsupport（防呆）
	repo      Repository
	logger    *zap.Logger
	metrics   *Metrics // 可 nil；nil-safe 调用方法

	uploadResolver uploadAddressResolver
}

// SetMetrics 注入 Prometheus 指标（可选，nil 表示禁用）。
func (d *Dispatcher) SetMetrics(m *Metrics) { d.metrics = m }

// SetPlatformResolver 注入平台解析器（productClass → param_model.name 链路）。
// 不注入则所有设备都按 unsupport 处理（防呆：避免漏配 ProductRegistry 时静默乱发）。
func (d *Dispatcher) SetPlatformResolver(p PlatformResolver) { d.platforms = p }

// SetUploadAddressResolver 注入统一 ACS 传输地址策略。
func (d *Dispatcher) SetUploadAddressResolver(r uploadAddressResolver) { d.uploadResolver = r }

// NewDispatcher 创建 dispatcher 实例。
//   - cfg：MR yaml 配置，经 Defaults() 兜底
//   - enqueuer：task.TaskService 满足
//   - devices：device.DeviceRepository 经 adapter 满足
//   - resolver：可为 nil（禁用翻译，全部下发 standardPath；仅适用于厂商私有
//     path 与标准 path 一致的场景，长期不推荐）
//   - repo：mr/task 自己的 Repository（用于失败时回写 progress）
func NewDispatcher(cfg appconfig.MRConfig, enqueuer devicetask.Enqueuer, devices DeviceContext, resolver TranslatorResolver, repo Repository, logger *zap.Logger) *Dispatcher {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Dispatcher{
		cfg:      cfg.Defaults(),
		enqueuer: enqueuer,
		devices:  devices,
		resolver: resolver,
		repo:     repo,
		logger:   logger.Named("mr-dispatcher"),
	}
}

// Open 对一个 cell 触发"开启 MR 上报"流程：
//  1. 查设备 productClass 判断平台是否支持
//  2. 平台不支持 → 直接标 unsupport
//  3. 支持 → 派发 SPV（6 参数）入队
//
// 注意：本方法只负责"派发任务"，**不等设备应答**。设备应答到达后由
// CompletionCallback（registerCompletionCallback 注册）回写 progress_status。
// 这种异步设计与 provision/orchestrator 的 EnqueueSteps 模式一致。
func (d *Dispatcher) Open(ctx context.Context, task *Task, cell Progress) error {
	ctx, span := tracing.StartSpan(ctx, tracing.MRTaskTracerName, "mr.task.open",
		attribute.String("mr.task_id", task.TaskID.String()),
		attribute.String("mr.cell", cell.SmallCellCode),
		attribute.String("mr.device_sn", cell.SerialNumber),
	)
	defer span.End()

	dev, err := d.devices.GetDevice(ctx, cell.SerialNumber)
	if err != nil || dev == nil {
		// 设备查询失败不算"开启失败"，而是上游 device 模块的数据问题；
		// 标记 unsupport 让用户看到（最常见原因：设备未注册到 OMC）。
		d.logger.Warn("device lookup failed for MR open",
			zap.String("task_id", task.TaskID.String()),
			zap.String("cell", cell.SmallCellCode),
			zap.String("sn", cell.SerialNumber),
			zap.Error(err),
		)
		return d.repo.UpdateProgressDispatch(ctx, task.TaskID, cell.SmallCellCode, ProgressUnsupport, strPtr("device_not_found"))
	}
	// 平台判断走 ProductRegistry：productClass → product → param_model.name → 允许列表
	var platformName string
	if d.platforms != nil {
		var perr error
		platformName, perr = d.platforms.ResolvePlatform(ctx, dev.ProductClass)
		if perr != nil {
			d.logger.Warn("platform resolver failed; treating as unsupported",
				zap.String("sn", cell.SerialNumber),
				zap.String("product_class", dev.ProductClass),
				zap.Error(perr))
		}
	}
	if !IsSupportedPlatform(platformName) {
		d.logger.Info("device platform unsupported for MR",
			zap.String("sn", cell.SerialNumber),
			zap.String("product_class", dev.ProductClass),
			zap.String("resolved_platform", platformName),
			zap.Strings("allowed", SupportedPlatforms()),
		)
		d.metrics.IncDispatched("open", "unsupport")
		return d.repo.UpdateProgressDispatch(ctx, task.TaskID, cell.SmallCellCode, ProgressUnsupport, strPtr("platform_unsupport"))
	}

	// 解析 Translator：失败 / 字典未收录 → 降级（dispatcher 仍下发 standardPath）。
	// 与 internal/provision/orchestrator.go BuildProvisioningStepsTranslated 行为一致。
	translator := d.resolveTranslator(ctx, dev)
	values, transferDecision, err := d.buildOpenParameters(ctx, task, cell, dev, translator)
	if err != nil {
		_ = d.repo.UpdateProgressDispatch(ctx, task.TaskID, cell.SmallCellCode, ProgressOpenFailure, strPtr("invalid_mr_upload_url"))
		d.metrics.IncDispatched("open", "invalid_url")
		return fmt.Errorf("build MR open SPV parameters for %s: %w", cell.SerialNumber, err)
	}
	cmdKey := commandKeyOpenPrefix + shortID(task.TaskID) + "-" + cell.SmallCellCode

	if err := d.enqueueSPV(ctx, cell.SerialNumber, values, cmdKey, task.TaskID); err != nil {
		// 派发失败 → 标 openFailure 让用户立刻能看到（区别于设备应答失败的 openFailure）
		_ = d.repo.UpdateProgressDispatch(ctx, task.TaskID, cell.SmallCellCode, ProgressOpenFailure, strPtr("enqueue_failed"))
		d.metrics.IncDispatched("open", "enqueue_failed")
		return fmt.Errorf("enqueue MR open SPV for %s: %w", cell.SerialNumber, err)
	}
	d.metrics.IncDispatched("open", "success")
	d.logger.Info("MR open SPV enqueued",
		zap.String("task_id", task.TaskID.String()),
		zap.String("cell", cell.SmallCellCode),
		zap.String("sn", cell.SerialNumber),
		zap.String("command_key", cmdKey),
		zap.Bool("translator", translator != nil),
		zap.String("transfer_protocol", string(transferDecision.Protocol)),
		zap.String("transfer_reason", string(transferDecision.Reason)),
		zap.String("https_capability", string(transferDecision.Capability)),
	)
	return nil
}

// Close 对一个 cell 触发"关闭 MR 上报"：仅下发 MrEnable=false 单参数。
// 只对当前 openSuccess 的 cell 关闭有意义（unsupport / openFailure 跳过）。
//
// 注意 Close 也走翻译路径（standardPath → privatePath），保证 open / close
// 用的 path 一致 — 否则厂商可能开了一个路径却关到另一个路径。
// 关闭流程允许设备查询失败：失败时仍下发 standardPath，让设备自己回 Fault。
func (d *Dispatcher) Close(ctx context.Context, task *Task, cell Progress) error {
	ctx, span := tracing.StartSpan(ctx, tracing.MRTaskTracerName, "mr.task.close",
		attribute.String("mr.task_id", task.TaskID.String()),
		attribute.String("mr.cell", cell.SmallCellCode),
		attribute.String("mr.device_sn", cell.SerialNumber),
	)
	defer span.End()

	if cell.ProgressStatus != ProgressOpenSuccess {
		// 未开启的 cell 不需要关闭；scheduler 决定整体任务是否进 off 时另算。
		d.logger.Debug("skip MR close: cell not in openSuccess",
			zap.String("cell", cell.SmallCellCode),
			zap.String("status", string(cell.ProgressStatus)),
		)
		return nil
	}

	var translator *parammodel.Translator
	if dev, err := d.devices.GetDevice(ctx, cell.SerialNumber); err == nil && dev != nil {
		translator = d.resolveTranslator(ctx, dev)
	}

	values := []spvValue{
		{Name: d.translatePath(paramMrEnable, translator), Value: "false", Type: "xsd:string"},
	}
	cmdKey := commandKeyClosePrefix + shortID(task.TaskID) + "-" + cell.SmallCellCode

	if err := d.enqueueSPV(ctx, cell.SerialNumber, values, cmdKey, task.TaskID); err != nil {
		_ = d.repo.UpdateProgressDispatch(ctx, task.TaskID, cell.SmallCellCode, ProgressCloseFailure, strPtr("enqueue_failed"))
		d.metrics.IncDispatched("close", "enqueue_failed")
		return fmt.Errorf("enqueue MR close SPV for %s: %w", cell.SerialNumber, err)
	}
	d.metrics.IncDispatched("close", "success")
	d.logger.Info("MR close SPV enqueued",
		zap.String("task_id", task.TaskID.String()),
		zap.String("cell", cell.SmallCellCode),
		zap.String("command_key", cmdKey),
	)
	return nil
}

// ----------- 参数拼装 -----------

type spvValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Type  string `json:"type"`
}

// buildOpenParameters 构造开启 SPV 的 6 个参数。
// path 命名严格按文档 §4 / §5 报文示例（standardPath），经 translatePath
// 翻译为 privatePath 后塞入 SPV。
func (d *Dispatcher) buildOpenParameters(
	ctx context.Context,
	task *Task,
	cell Progress,
	dev *model.Device,
	translator *parammodel.Translator,
) ([]spvValue, transfercfg.AddressDecision, error) {
	uploadPeriodSec, _ := UploadPeriodSeconds(task.ReportPeriod) // service 已校验过合法
	mrURL, transferDecision, err := d.buildMrURL(ctx, dev.ID, cell.SmallCellCode)
	if err != nil {
		return nil, transfercfg.AddressDecision{}, err
	}

	return []spvValue{
		{Name: d.translatePath(paramMrEnable, translator), Value: "true", Type: "xsd:string"},
		{Name: d.translatePath(paramVendor, translator), Value: d.cfg.Vendor, Type: "xsd:string"},
		{Name: d.translatePath(paramOmcName, translator), Value: d.cfg.OmcName, Type: "xsd:string"},
		{Name: d.translatePath(paramMrURL, translator), Value: mrURL, Type: "xsd:string"},
		{Name: d.translatePath(paramPeriodicReport, translator), Value: task.StatisPeriod, Type: "xsd:string"},
		{Name: d.translatePath(paramUploadPeriod, translator), Value: fmt.Sprintf("%d", uploadPeriodSec), Type: "xsd:string"},
	}, transferDecision, nil
}

// resolveTranslator 拿设备对应的 Translator。resolver 未注入或解析失败时返回 nil
// （调用方会降级到 standardPath）。
func (d *Dispatcher) resolveTranslator(ctx context.Context, dev *model.Device) *parammodel.Translator {
	if d.resolver == nil || dev == nil {
		return nil
	}
	t, ok := d.resolver.ResolveForDevice(ctx, dev)
	if !ok {
		return nil
	}
	return t
}

// translatePath 把 standardPath（带 {i} 占位符）翻译为厂商 privatePath。
//
// 步骤：
//  1. {i} 替换为 defaultMRMgmtInstanceIdx（当前固定 1，多任务并存时再扩展）
//  2. 若 translator 为 nil → 直接返回 substituted standardPath（fail-soft）
//  3. translator.ToPrivate：
//     - Found=true → 返回 privatePath
//     - Found=false → 返回 standardPath（与 provision orchestrator 一致策略）
//
// metrics 上的 translator 命中/未命中已由 parammodel 包内置 Prometheus 计数器统计。
func (d *Dispatcher) translatePath(standardPath string, translator *parammodel.Translator) string {
	concrete := substInstance(standardPath)
	if translator == nil {
		return concrete
	}
	return translator.ToPrivate(concrete).Translated
}

// buildMrURL 拼装设备 HTTP POST 上传地址（文档 §6）：
//
//	{URLBase}/smallcell/FileUploadService?fileType=MR&cellCode={cellCode}&filename=
//
// filename 段保留为空字符串是规范要求 —— 让设备自行决定上传文件名。
func (d *Dispatcher) buildMrURL(ctx context.Context, deviceID uuid.UUID, cellCode string) (string, transfercfg.AddressDecision, error) {
	if d.uploadResolver == nil {
		return "", transfercfg.AddressDecision{}, fmt.Errorf("MR upload address resolver is not configured")
	}

	decision, err := d.uploadResolver.Resolve(ctx, deviceID, transfercfg.TransferDirectionUpload)
	if err != nil {
		return "", transfercfg.AddressDecision{}, fmt.Errorf("resolve MR upload address: %w", err)
	}
	relativeReference := fmt.Sprintf("/smallcell/FileUploadService?fileType=MR&cellCode=%s&filename=",
		url.QueryEscape(cellCode))
	mrURL, err := transfercfg.BuildTemplateURL(decision.BaseURL, relativeReference)
	if err != nil {
		return "", transfercfg.AddressDecision{}, err
	}
	return mrURL, decision, nil
}

// substInstance 替换路径里的 {i} 为默认实例号 1。
// 未来按实例号动态分配（多 MR 任务并存）时，把此函数改为接收 idx int 参数即可。
func substInstance(path string) string {
	return strings.Replace(path, "{i}", fmt.Sprintf("%d", defaultMRMgmtInstanceIdx), 1)
}

// enqueueSPV 把构造好的 6 / 1 参数派发到 task 队列。
//
//   - Source = TaskSourceSystem：与 provision 一致（非 API 触发，非 MML 扇出）
//   - SourceID = MR taskID：completion callback 据此反查 mr_customize_task
//   - CommandKey = "mr-open-{taskID8}-{cellCode}" / "mr-close-..."
//     completion 回调会按 prefix 区分是开启还是关闭流程
func (d *Dispatcher) enqueueSPV(ctx context.Context, deviceSN string, values []spvValue, cmdKey string, taskID uuid.UUID) error {
	body := struct {
		Values []spvValue `json:"values"`
	}{Values: values}
	paramsJSON, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal SPV body: %w", err)
	}
	if _, err := d.enqueuer.CreateTask(ctx, &devicetask.CreateTaskRequest{
		DeviceSN:    deviceSN,
		Method:      methodSetParameterValues,
		Params:      paramsJSON,
		CommandKey:  cmdKey,
		Source:      devicetask.TaskSourceSystem,
		SourceID:    taskID.String(),
		Description: "MR measurement task SPV",
	}); err != nil {
		return fmt.Errorf("enqueue device task: %w", err)
	}
	return nil
}

// ----------- 设备应答回调 -----------

// CompletionCallback 实现 task.TaskCompletionCallback，挂到 worker 端
// CompletionRouter 上（source=TaskSourceSystem 全局监听；按 CommandKey 前缀
// 过滤是不是 MR 的）。设备 SPV 成功/失败到达后，根据 prefix 决定 progress 转移。
type CompletionCallback struct {
	repo   Repository
	logger *zap.Logger
}

// NewCompletionCallback 创建 SPV 完成回调。
func NewCompletionCallback(repo Repository, logger *zap.Logger) *CompletionCallback {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &CompletionCallback{repo: repo, logger: logger.Named("mr-completion")}
}

// OnTaskCompleted 实现 task.TaskCompletionCallback。
//
// 仅处理 CommandKey 前缀以 "mr-open-" 或 "mr-close-" 开头的 task；其它一律 no-op。
// 之所以这样过滤而不是按 Source 字段：TaskSourceSystem 是多模块共用值，按
// CommandKey 区分更稳妥（不会"偷"了 provision 等模块的应答）。
func (c *CompletionCallback) OnTaskCompleted(ctx context.Context, t *devicetask.Task) {
	if t == nil {
		return
	}
	key := t.CommandKey
	isOpen := strings.HasPrefix(key, commandKeyOpenPrefix)
	isClose := strings.HasPrefix(key, commandKeyClosePrefix)
	if !isOpen && !isClose {
		return
	}

	// 从 CommandKey 解析出 cellCode：mr-{open|close}-{taskID8}-{cellCode}
	cellCode, ok := parseCellFromCommandKey(key)
	if !ok || t.SourceID == "" {
		c.logger.Warn("malformed MR command key", zap.String("key", key), zap.String("source_id", t.SourceID))
		return
	}
	taskID, err := uuid.Parse(t.SourceID)
	if err != nil {
		c.logger.Warn("invalid source_id for MR task", zap.String("source_id", t.SourceID))
		return
	}

	// 根据 task.Status / FaultCode 决定 progress 转移
	success := isTaskSuccess(t)
	var (
		newStatus ProgressStatus
		fault     *string
	)
	switch {
	case isOpen && success:
		newStatus = ProgressOpenSuccess
	case isOpen && !success:
		newStatus = ProgressOpenFailure
		fault = strPtr(t.ErrorMessage)
	case isClose && success:
		newStatus = ProgressCloseSuccess
	case isClose && !success:
		newStatus = ProgressCloseFailure
		fault = strPtr(t.ErrorMessage)
	}

	if err := c.repo.UpdateProgressDispatch(ctx, taskID, cellCode, newStatus, fault); err != nil {
		c.logger.Error("update MR progress on task completion failed",
			zap.String("task_id", taskID.String()),
			zap.String("cell", cellCode),
			zap.String("new_status", string(newStatus)),
			zap.Error(err),
		)
		return
	}
	c.logger.Info("MR progress updated by SPV completion",
		zap.String("task_id", taskID.String()),
		zap.String("cell", cellCode),
		zap.String("status", string(newStatus)),
	)
}

// ----------- 辅助 -----------

// parseCellFromCommandKey 从 "mr-open-abc12345-CELL001" 取出 "CELL001"。
// 失败返回 ("", false)。允许 cellCode 内含连字符（如 "AREA-01-CELL"）—— 取第
// 三段及之后所有字符。
func parseCellFromCommandKey(key string) (string, bool) {
	// 去除已知前缀
	var rest string
	switch {
	case strings.HasPrefix(key, commandKeyOpenPrefix):
		rest = strings.TrimPrefix(key, commandKeyOpenPrefix)
	case strings.HasPrefix(key, commandKeyClosePrefix):
		rest = strings.TrimPrefix(key, commandKeyClosePrefix)
	default:
		return "", false
	}
	// rest = "{taskID8}-{cellCode...}"
	idx := strings.Index(rest, "-")
	if idx < 0 || idx == len(rest)-1 {
		return "", false
	}
	return rest[idx+1:], true
}

// isTaskSuccess 判断 task.Task 是否被设备成功应答。
// task 模块未导出 status 常量做枚举，按字符串比较；
// ErrorCode > 0 一律视为失败（CWMP FaultCode 通过 task.MarkTaskCompleted 落到此字段）。
func isTaskSuccess(t *devicetask.Task) bool {
	if t == nil {
		return false
	}
	status := strings.ToLower(string(t.Status))
	if status == "failed" || status == "timeout" || status == "expired" {
		return false
	}
	if t.ErrorCode > 0 {
		return false
	}
	return true
}

// shortID 返回 uuid 的前 8 个十六进制字符（无连字符），用于 CommandKey 紧凑命名。
func shortID(id uuid.UUID) string {
	s := strings.ReplaceAll(id.String(), "-", "")
	if len(s) < 8 {
		return s
	}
	return s[:8]
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
