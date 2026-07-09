package mml

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/product"
	"github.com/omcgo/omcgo/internal/task"
)

// DeviceTaskCreator creates device_tasks from an MML task.
// Implemented by task.TaskService; defined here to avoid circular dependency.
type DeviceTaskCreator interface {
	BatchCreateTasks(ctx context.Context, reqs []*task.CreateTaskRequest) ([]*task.Task, error)
}

// ProductMatcher routes a productClass string to a product (with ParamModelID).
//
// Implemented by *product.Registry. Defined here (consumer-side) to keep mml
// from a hard dependency on the concrete Registry constructor.
type ProductMatcher interface {
	MatchProductClass(ctx context.Context, productClass string) (*product.MatchResult, error)
}

// ParamModelTranslatorFactory builds a Translator for (productID, swVersion).
//
// Implemented by *parammodel.Registry (its Translator method) — the
// per-(product, sw) MappingSet is fetched & cached internally.
type ParamModelTranslatorFactory interface {
	Translator(ctx context.Context, productID uuid.UUID, swVersion string) (*parammodel.Translator, error)
}

// DeviceLookup resolves a device serial number to its routing metadata.
//
// Only ProductClass + FirmwareVersion are consumed by fanout; passing the
// full Device is convenient since callers (device.DeviceService) already
// have it cached.
type DeviceLookup interface {
	GetBySerialNumber(ctx context.Context, sn string) (*model.Device, error)
}

// Fanouter fans out an MML task into individual device_tasks.
//
// Sprint B Q-V3-3：sequentialMode=true 时，初次 fanout 仅入队
// **每个设备的 command_index=0** device_task；后续命令由 Sequencer
// 在前一行完成回调中追加入队，实现严格序列。
//
// Stage 1 (T-0123 v5): MML 命令的 sub_fields 自 migration 000113 起改挂
// standard_params（系统级标准 path 字典）。fanout 阶段需把 standardPath
// 翻译为 device 对应的 privatePath 后再入 device_tasks.params；翻译路径：
//
//	device.SerialNumber → DeviceLookup.GetBySerialNumber → Device.ProductClass
//	→ ProductMatcher.MatchProductClass → Product.ParamModelID
//	→ ParamModelTranslatorFactory.Translator(productID, swVersion)
//	→ Translator.ToPrivate(standardPath)
//
// 任何一步失败（设备未注册 / product 未匹配 / mapping 不存在 / path 未命中）
// 都 fallback：用 standardPath 直接当 privatePath 下发，并在 device_task 上
// 标记 has_path_translation_miss=true + path_translation_miss_count，前端
// 任务详情页据此显示"路径翻译警告"标签（用户决策 Q1 选项 B）。
type Fanouter struct {
	taskCreator       DeviceTaskCreator
	productMatcher    ProductMatcher              // 可空 → 所有路径走 fallback
	translatorFactory ParamModelTranslatorFactory // 可空 → 所有路径走 fallback
	deviceLookup      DeviceLookup                // 可空 → 所有路径走 fallback
	sequentialMode    bool
	metrics           *FanoutMetrics // 可空 → 跳过 metric 累加（测试场景）
	logger            *zap.Logger
}

// NewFanouter creates a new Fanouter.
//
// productMatcher / translatorFactory / deviceLookup 任一为 nil 时，fanout
// 跳过 standardPath ↔ privatePath 翻译，原 standardPath 直接下发；适合早期
// 集成 / 测试场景。生产部署必须三者齐全。
func NewFanouter(
	taskCreator DeviceTaskCreator,
	productMatcher ProductMatcher,
	translatorFactory ParamModelTranslatorFactory,
	deviceLookup DeviceLookup,
	logger *zap.Logger,
) *Fanouter {
	return &Fanouter{
		taskCreator:       taskCreator,
		productMatcher:    productMatcher,
		translatorFactory: translatorFactory,
		deviceLookup:      deviceLookup,
		logger:            logger.Named("mml-fanout"),
	}
}

// SetSequentialMode 启用脚本多行严格序列模式（Sprint B Q-V3-3 决议）。
// 启用后初次 fanout 只入队第一行；Sequencer 通过 completion callback 链式
// 入队后续行。需要在 DI 层把 Sequencer 注册到 CompletionRouter。
func (f *Fanouter) SetSequentialMode(enabled bool) { f.sequentialMode = enabled }

// SetMetrics 挂接 Prometheus 指标集合。可重复调用；nil 则跳过累加。
// Stage 1 整改方案 §3：每次 path 翻译 fallback 都需累加 mml_path_translation_miss_total
// 供 alert 监控（详见 deployments/monitoring/alerts/omc-rules.yml）。
func (f *Fanouter) SetMetrics(m *FanoutMetrics) { f.metrics = m }

func failedRetryMaxRetries(mmlTask *MMLTask) *int {
	retries := 0
	if mmlTask != nil && mmlTask.FailedRetry {
		retries = mmlTask.FailedRetryCount
		if retries < 0 {
			retries = 0
		}
	}
	return &retries
}

func offlineRetryExpiresIn(mmlTask *MMLTask) int {
	if mmlTask == nil || !mmlTask.OfflineRetry || mmlTask.OfflineRetryWait <= 0 {
		return 0
	}
	return mmlTask.OfflineRetryWait * 60
}

func failedRetryIntervalSeconds(mmlTask *MMLTask) int {
	if mmlTask == nil || !mmlTask.FailedRetry || mmlTask.FailedRetryInterval <= 0 {
		return 0
	}
	return mmlTask.FailedRetryInterval * 60
}

// Fanout creates device_tasks for each (command, device) pair in the MML task.
// Only called for immediate execution; scheduled/periodic tasks are fan-outed when started.
func (f *Fanouter) Fanout(ctx context.Context, mmlTask *MMLTask) (int, error) {
	if len(mmlTask.DeviceSNs) == 0 || len(mmlTask.Commands) == 0 {
		return 0, nil
	}

	reqs := f.buildDeviceTaskRequests(ctx, mmlTask)
	if len(reqs) == 0 {
		return 0, nil
	}

	// Sprint B Q-V3-3: sequentialMode 下只入队 command_index=0 那批
	if f.sequentialMode && len(mmlTask.Commands) > 1 {
		if mmlTask.ExecuteMode == TaskExecuteModeDeviceBound {
			reqs = firstDeviceBoundRequests(mmlTask, reqs)
		} else {
			first := make([]*task.CreateTaskRequest, 0, len(mmlTask.DeviceSNs))
			for _, r := range reqs {
				if r.CommandIndex == 0 {
					first = append(first, r)
				}
			}
			reqs = first
		}
		f.logger.Info("mml fanout sequential mode: enqueue first line only",
			zap.String("mml_task_id", mmlTask.ID.String()),
			zap.Int("first_line_tasks", len(reqs)),
			zap.Int("total_lines", len(mmlTask.Commands)),
		)
	}

	created, err := f.taskCreator.BatchCreateTasks(ctx, reqs)
	if err != nil {
		return 0, fmt.Errorf("fanout mml task %s: %w", mmlTask.ID, err)
	}

	f.logger.Info("mml task fan-out completed",
		zap.String("mml_task_id", mmlTask.ID.String()),
		zap.Int("device_count", len(mmlTask.DeviceSNs)),
		zap.Int("command_count", len(mmlTask.Commands)),
		zap.Int("device_tasks_created", len(created)),
		zap.Bool("sequential_mode", f.sequentialMode),
	)

	return len(created), nil
}

// buildDeviceTaskRequests converts an MML task into per-device CreateTaskRequest.
//
// 与早期版本相比，本函数已变为 **per-device** 渲染（先按设备解析 Translator，
// 再为该设备单独构造 TR-069 params）。原因：mml_command_sub_fields 自
// migration 000113 起携带 standardPath；fanout 阶段必须按 device 把
// standardPath → privatePath 翻译后再入 device_tasks，否则 CPE 收到的将是
// IETF 标准 path 而非厂商私有 path，参数操作会返回 Fault 9005。
//
// Translator 解析任一阶段失败均不阻塞下发：fallback 用 standardPath 兜底，
// device_task 打 miss 标记，前端任务详情可见警告（用户决策 Q1=B）。
func (f *Fanouter) buildDeviceTaskRequests(ctx context.Context, mmlTask *MMLTask) []*task.CreateTaskRequest {
	var reqs []*task.CreateTaskRequest

	if mmlTask.ExecuteMode == TaskExecuteModeDeviceBound {
		return f.buildDeviceBoundTaskRequests(ctx, mmlTask)
	}

	for cmdIdx, cmd := range mmlTask.Commands {
		for devIdx, sn := range mmlTask.DeviceSNs {
			req := f.buildDeviceTaskRequest(ctx, mmlTask, cmd, cmdIdx, sn, devIdx, devIdx == 0)
			if req == nil {
				continue
			}
			reqs = append(reqs, req)
		}
	}

	return reqs
}

func (f *Fanouter) buildDeviceBoundTaskRequests(ctx context.Context, mmlTask *MMLTask) []*task.CreateTaskRequest {
	reqs := make([]*task.CreateTaskRequest, 0, len(mmlTask.PlanItems))
	deviceIndexes := make(map[string]int, len(mmlTask.DeviceSNs))
	for idx, sn := range mmlTask.DeviceSNs {
		if _, exists := deviceIndexes[sn]; !exists {
			deviceIndexes[sn] = idx
		}
	}

	for planIdx, item := range mmlTask.PlanItems {
		if planIdx >= len(mmlTask.Commands) {
			f.logger.Warn("skip plan item without matching command",
				zap.String("mml_task_id", mmlTask.ID.String()),
				zap.Int("plan_idx", planIdx),
				zap.Int("line_no", item.LineNo),
				zap.String("device_sn", item.DeviceSN),
			)
			continue
		}
		devIdx, ok := deviceIndexes[item.DeviceSN]
		if !ok {
			devIdx = len(deviceIndexes)
			deviceIndexes[item.DeviceSN] = devIdx
		}
		req := f.buildDeviceTaskRequest(ctx, mmlTask, mmlTask.Commands[planIdx], planIdx, item.DeviceSN, devIdx, true)
		if req == nil {
			continue
		}
		reqs = append(reqs, req)
	}
	return reqs
}

func (f *Fanouter) buildDeviceTaskRequest(
	ctx context.Context,
	mmlTask *MMLTask,
	cmd map[string]interface{},
	cmdIdx int,
	sn string,
	devIdx int,
	logSchema bool,
) *task.CreateTaskRequest {
	parentID := mmlTask.ID.String()
	rpcMethod, _ := cmd["rpc_method"].(string)
	if rpcMethod == "" {
		f.logger.Warn("skip command without rpc_method",
			zap.String("mml_task_id", parentID),
			zap.Int("cmd_idx", cmdIdx),
			zap.String("device_sn", sn),
			zap.Any("command_code", cmd["command_code"]),
		)
		return nil
	}

	paramRefs := paramRefsFromEntry(cmd)
	formValues, _ := cmd["parameters"].(map[string]interface{})
	operationType, _ := cmd["operation_type"].(string)
	commandCode, _ := cmd["command_code"].(string)

	description := fmt.Sprintf("MML %s", commandCode)
	if mmlTask.TaskName != "" {
		description = fmt.Sprintf("MML %s: %s", commandCode, mmlTask.TaskName)
	}

	translated, missCount, translator := f.translateParamRefs(ctx, sn, paramRefs)

	// v1.2 测试报告 §3 P0：ADD/RMV 的 object_name 走独立翻译。
	// translateParamRefs 仅处理 param_refs[].Tr069Path；ADD/RMV 命令的
	// parameters.object_name 是父对象路径（如 .../Carrier.），同样需要
	// standardPath → privatePath；否则 BaiBLQ 等私有 path 设备必返 9005。
	perDeviceFormValues := f.translateObjectName(ctx, sn, formValues)

	params, err := BuildTR069Params(rpcMethod, translated, perDeviceFormValues, operationType)
	if err != nil {
		f.logger.Warn("build tr069 params failed, skip device",
			zap.String("mml_task_id", parentID),
			zap.Int("cmd_idx", cmdIdx),
			zap.String("device_sn", sn),
			zap.String("command_code", commandCode),
			zap.String("rpc_method", rpcMethod),
			zap.String("operation_type", operationType),
			zap.Int("param_refs_count", len(translated)),
			zap.Int("form_values_count", len(formValues)),
			zap.Error(err),
		)
		return nil
	}

	if logSchema {
		// 一条 command 打一次 schema 摘要日志（per-cmd 而非 per-device）。
		summary := SummarizeSchema(params)
		f.logger.Info("device_task params built",
			zap.String("mml_task_id", parentID),
			zap.Int("cmd_idx", cmdIdx),
			zap.String("command_code", commandCode),
			zap.String("rpc_method", rpcMethod),
			zap.String("operation_type", operationType),
			zap.String("translator_source", translatorSourceLabel(translator)),
			zap.Int("payload_size", summary.PayloadSize),
			zap.Bool("has_names", summary.HasNames),
			zap.Int("names_count", summary.NamesCount),
			zap.Bool("has_values", summary.HasValues),
			zap.Int("values_count", summary.ValuesCount),
		)
	}

	return &task.CreateTaskRequest{
		DeviceSN:             sn,
		Method:               rpcMethod,
		Params:               params,
		Priority:             10,
		ExpiresIn:            offlineRetryExpiresIn(mmlTask),
		Source:               task.TaskSourceMML,
		CreatorID:            mmlTask.Creator,
		Description:          description,
		MaxRetries:           failedRetryMaxRetries(mmlTask),
		RetryIntervalSeconds: failedRetryIntervalSeconds(mmlTask),

		SourceID:     parentID,
		CommandIndex: cmdIdx,
		DeviceIndex:  devIdx,

		HasPathTranslationMiss:   missCount > 0,
		PathTranslationMissCount: missCount,
		// T-0168: per-device 翻译来源继承 task 维度（D2 决策：R-8.4 保证一致）。
		PathTranslationSource: mmlTask.PathTranslationSource,
	}
}

func firstDeviceBoundRequests(mmlTask *MMLTask, reqs []*task.CreateTaskRequest) []*task.CreateTaskRequest {
	bestByDevice := make(map[string]*task.CreateTaskRequest, len(reqs))
	for _, req := range reqs {
		current := bestByDevice[req.DeviceSN]
		if current == nil || planOrderForCommand(mmlTask, req.CommandIndex) < planOrderForCommand(mmlTask, current.CommandIndex) {
			bestByDevice[req.DeviceSN] = req
		}
	}
	out := make([]*task.CreateTaskRequest, 0, len(bestByDevice))
	for _, sn := range mmlTask.DeviceSNs {
		if req := bestByDevice[sn]; req != nil {
			out = append(out, req)
			delete(bestByDevice, sn)
		}
	}
	for _, req := range bestByDevice {
		out = append(out, req)
	}
	return out
}

func planOrderForCommand(mmlTask *MMLTask, cmdIdx int) int {
	if cmdIdx >= 0 && cmdIdx < len(mmlTask.PlanItems) && mmlTask.PlanItems[cmdIdx].Order > 0 {
		return mmlTask.PlanItems[cmdIdx].Order
	}
	return cmdIdx + 1
}

// translateParamRefs 已退化为 noop（T-XXX 改造）。
//
// 历史：本方法曾在 fanout 阶段把 standardPath 翻译为 privatePath 写入 device_task.params。
// 改造后：翻译职责完全迁移到 ACS 端（PopTask 后、BuildRequest 前），队列里存的是
// standardPath（用户的核心诉求 — 字典更新后 pending 任务自动使用新映射，任务审计稳定）。
//
// 保留方法签名是为了让 buildDeviceTaskRequests 现有调用站点继续编译；ProductMatcher /
// TranslatorFactory / DeviceLookup 依赖保留在 Fanouter struct 中但不再被消费，
// 是为了让 provider DI 接线层保持不变（向后兼容）。
//
// T-0170 即时 422 校验仍由 service.translateTaskPaths 在 fanout 前执行；本方法之后
// 即可移除——但本期保持最小改动面。
func (f *Fanouter) translateParamRefs(
	_ context.Context, _ string, refs []MMLParamRef,
) ([]MMLParamRef, int, *parammodel.Translator) {
	return refs, 0, nil
}

// translatorSourceLabel 给日志生成一个简短的 Translator 状态字串。
// nil → "fallback"（fanout 跳过翻译，原样下发 standardPath）。
func translatorSourceLabel(t *parammodel.Translator) string {
	if t == nil {
		return "fallback"
	}
	return "param_model"
}

// translateObjectName 已退化为 noop（T-XXX 改造）。
//
// 历史：曾在 fanout 阶段翻译 ADD/RMV 的 object_name 字段（避免 BaiBLQ 等私有 path 设备
// 收到 standardPath 后返 9005）。改造后：ACS 端的 PathTranslationService.TranslateTaskParams
// 在 BuildRequest 前统一翻译 object_name（含 AddObject / DeleteObject 路径）。
//
// 保留方法签名仅为兼容现有调用站点；formValues 原样返回（standardPath 落入 device_task.params）。
func (f *Fanouter) translateObjectName(
	_ context.Context, _ string, formValues map[string]interface{},
) map[string]interface{} {
	return formValues
}

// paramRefsFromEntry pulls param_refs out of a commands[] entry, tolerating
// the JSON-roundtrip form where the slice arrives as []interface{} of map[string]interface{}.
func paramRefsFromEntry(cmd map[string]interface{}) []MMLParamRef {
	raw, ok := cmd["param_refs"]
	if !ok {
		return nil
	}
	switch v := raw.(type) {
	case []MMLParamRef:
		return v
	case []interface{}:
		out := make([]MMLParamRef, 0, len(v))
		for _, item := range v {
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			b, err := json.Marshal(itemMap)
			if err != nil {
				continue
			}
			var ref MMLParamRef
			if err := json.Unmarshal(b, &ref); err != nil {
				continue
			}
			out = append(out, ref)
		}
		return out
	default:
		return nil
	}
}
