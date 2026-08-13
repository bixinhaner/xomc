package device

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/product"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/omcgo/omcgo/pkg/tr069"
)

// RPCResponseSubscriber 订阅 ACS 进程发布的 RPC 响应事件（command.*.response），
// 把 CPE 响应里的 privatePath 经 ParamModel Translator 翻译为 standardPath 后
// 持久化到 device_parameters 表。
//
// 设计依据：用户决策 2026-05-16 — 整改方案 Stage 2（方案 D 事件驱动）。
// ACS 独立进程，不装配 ParamRegistry / ProductRegistry，跨进程 path 翻译
// 通过 EventBus 转交本进程订阅者处理；本订阅者复用 App 进程已有的 Registry
// 缓存（L1 sync.Map + L2 Redis），翻译路径与 mml/fanout.go Stage 1 下行链路
// 对称：
//
//	下行 (Stage 1) standardPath → privatePath (in mml/fanout.go)
//	上行 (Stage 2) privatePath → standardPath (本文件)
//
// 翻译失败的 path（设备未注册 / mapping 未覆盖）退化为按原 privatePath 写入，
// 与 Path B 同步 (provision/sync_pathb.go) 的 fallback 行为对齐；后续可通过
// admin UI 补充 discovered_param_mappings 后由用户 retry 命令获得正确翻译。
type RPCResponseSubscriber struct {
	bus               event.EventBus
	productMatcher    ProductMatcher
	translatorFactory ParamModelTranslatorFactory
	deviceLookup      RPCDeviceLookup
	paramRepo         DeviceParameterRepository
	infoRefresher     rpcResponseDeviceInfoRefresher
	deviceWriter      DeviceSyncFailureWriter // migration 000146: 写 last_param_sync_failed_at + error
	taskEnqueuer      task.Enqueuer
	logger            *zap.Logger
	gpvConsumer       appconfig.GPVResponseConsumerConfig

	subscriptions []event.Subscription
}

type keyedQueueEventBus interface {
	KeyedQueueSubscribe(
		subject string,
		config event.KeyedQueueConfig,
		keyFunc event.EventKeyFunc,
		handler event.EventHandler,
	) (event.Subscription, error)
}

// DeviceSyncFailureWriter 窄接口:仅负责把 Path B GPV task 失败回写到 devices 表
// (migration 000146 新增 last_param_sync_failed_at + last_param_sync_error 两列)。
// 实现由 *PgDeviceRepository 提供;nil 时 handleSyncTaskFailed 静默跳过(测试场景)。
type DeviceSyncFailureWriter interface {
	UpdateLastParamSyncFailed(ctx context.Context, id uuid.UUID, failedAt time.Time, errMsg string) error
}

// ProductMatcher 通过 productClass 路由出 product (含 ParamModelID)。
// 接口在消费者侧定义（device 包内），实现由 *product.Registry 提供。
type ProductMatcher interface {
	MatchProductClass(ctx context.Context, productClass string) (*product.MatchResult, error)
}

// ParamModelTranslatorFactory 按 (productID, swVersion) 构造 Translator。
// 实现由 *parammodel.Registry 提供。
type ParamModelTranslatorFactory interface {
	Translator(ctx context.Context, productID uuid.UUID, swVersion string) (*parammodel.Translator, error)
}

// RPCDeviceLookup 按 SN 查询设备路由元数据（ProductClass / FirmwareVersion / ID）。
// 实现由 *DeviceService 提供（GetBySerialNumber 已存在）。
type RPCDeviceLookup interface {
	GetBySerialNumber(ctx context.Context, sn string) (*model.Device, error)
}

type rpcResponseDeviceInfoRefresher interface {
	SyncFromParameters(ctx context.Context, deviceID uuid.UUID, carrierCode model.CarrierCode, tech model.Technology, productClass string) ([]string, error)
}

// NewRPCResponseSubscriber 构造订阅者。Start() 时才真正订阅 bus。
// 任一依赖为 nil 时订阅者会跳过翻译，CPE 响应仍按 privatePath 入库（兼容
// 启动早期 / 测试场景）。
func NewRPCResponseSubscriber(
	bus event.EventBus,
	productMatcher ProductMatcher,
	translatorFactory ParamModelTranslatorFactory,
	deviceLookup RPCDeviceLookup,
	paramRepo DeviceParameterRepository,
	infoRefresher rpcResponseDeviceInfoRefresher,
	deviceWriter DeviceSyncFailureWriter,
	taskEnqueuer task.Enqueuer,
	gpvConsumer appconfig.GPVResponseConsumerConfig,
	logger *zap.Logger,
) *RPCResponseSubscriber {
	return &RPCResponseSubscriber{
		bus:               bus,
		productMatcher:    productMatcher,
		translatorFactory: translatorFactory,
		deviceLookup:      deviceLookup,
		paramRepo:         paramRepo,
		infoRefresher:     infoRefresher,
		deviceWriter:      deviceWriter,
		taskEnqueuer:      taskEnqueuer,
		gpvConsumer:       gpvConsumer,
		logger:            logger.Named("device-rpc-resp-sub"),
	}
}

// Start 订阅 RPC 响应事件并启动 handler。当前仅订阅
// command.get_parameters.response（GPV）——SetParameterValuesResponse 不
// 返回 path/value（仅 Status），Inform 走另一条链路（device_service.go），
// 都不需要本订阅者处理。
func (s *RPCResponseSubscriber) Start() error {
	if s.bus == nil {
		s.logger.Warn("event bus is nil; RPC response subscriber disabled")
		return nil
	}
	config := s.gpvConsumer.Defaults()
	var (
		sub event.Subscription
		err error
	)
	if keyedBus, ok := s.bus.(keyedQueueEventBus); ok {
		sub, err = keyedBus.KeyedQueueSubscribe(
			event.SubjectCommandGetParamsResponse,
			event.KeyedQueueConfig{
				Durable:       config.RPCDurable,
				StartSequence: config.RPCStartSequence,
				Concurrency:   config.RPCConcurrency,
				QueueDepth:    config.RPCQueueDepth,
				AckWait:       config.AckWait,
				MaxDeliver:    config.MaxDeliver,
				MaxAckPending: config.MaxAckPending,
			},
			gpvResponseDeviceKey,
			s.handleGPVResponse,
		)
	} else {
		if config.RPCStartSequence > 0 {
			return fmt.Errorf(
				"subscribe %s: event bus does not support lossless start sequence %d",
				event.SubjectCommandGetParamsResponse,
				config.RPCStartSequence,
			)
		}
		sub, err = s.bus.QueueSubscribe(
			event.SubjectCommandGetParamsResponse,
			config.RPCDurable,
			s.handleGPVResponse,
		)
	}
	if err != nil {
		return fmt.Errorf(
			"queue subscribe %s (%s): %w",
			event.SubjectCommandGetParamsResponse,
			config.RPCDurable,
			err,
		)
	}
	s.subscriptions = append(s.subscriptions, sub)
	delSub, err := s.bus.Subscribe(event.SubjectCommandDeleteObjectResponse, s.handleDeleteObjectResponse)
	if err != nil {
		return fmt.Errorf("subscribe %s: %w", event.SubjectCommandDeleteObjectResponse, err)
	}
	s.subscriptions = append(s.subscriptions, delSub)
	// migration 000146: 订阅 task.failed,识别 Path B GPV task(command_key 前缀
	// "sync-gpv-")失败后回写 last_param_sync_failed_at + error,供前端"上次同步失败"展示。
	failSub, err := s.bus.Subscribe(event.SubjectTaskFailed, s.handleSyncTaskFailed)
	if err != nil {
		return fmt.Errorf("subscribe %s: %w", event.SubjectTaskFailed, err)
	}
	s.subscriptions = append(s.subscriptions, failSub)
	s.logger.Info("RPC response subscriber started",
		zap.Strings("subjects", []string{
			event.SubjectCommandGetParamsResponse,
			event.SubjectCommandDeleteObjectResponse,
			event.SubjectTaskFailed,
		}),
	)
	return nil
}

func gpvResponseDeviceKey(evt event.Event) (string, error) {
	var payload struct {
		DeviceSN string `json:"device_sn"`
	}
	if err := evt.DecodePayload(&payload); err != nil {
		return "", fmt.Errorf("decode GPV device key: %w", err)
	}
	return payload.DeviceSN, nil
}

// handleSyncTaskFailed 处理 task.failed (包含 failed/expired/cancelled)。
// 过滤条件:Path B 同步入队的 GPV task (Method=GetParameterValues + CommandKey 前缀
// "sync-gpv-",对应 provision/sync.go StartSync / enqueueGPVPrefixes 入队点)。
// 命中即回写 devices.last_param_sync_failed_at + error。
func (s *RPCResponseSubscriber) handleSyncTaskFailed(ctx context.Context, evt event.Event) error {
	var t task.Task
	if err := evt.DecodePayload(&t); err != nil {
		s.logger.Warn("task.failed: decode payload", zap.Error(err))
		return nil
	}
	if t.Method == "DeleteObject" && isDeleteObjectAlreadyGone(t.ErrorMessage) {
		return s.handleDeleteObjectAlreadyGone(ctx, &t)
	}
	if err := s.handlePasswordResetFallback(ctx, &t); err != nil {
		return err
	}
	if s.deviceWriter == nil {
		return nil
	}
	if t.Method != "GetParameterValues" || !strings.HasPrefix(t.CommandKey, "sync-gpv-") {
		return nil
	}
	dev, err := s.deviceLookup.GetBySerialNumber(ctx, t.DeviceSN)
	if err != nil || dev == nil {
		s.logger.Warn("task.failed sync: device lookup failed",
			zap.String("device_sn", t.DeviceSN), zap.Error(err))
		return nil
	}
	errMsg := t.ErrorMessage
	if errMsg == "" {
		// expired/cancelled task 通常 ErrorMessage 为空,给一个能看的兜底文案
		errMsg = fmt.Sprintf("task %s", t.Status)
	}
	if err := s.deviceWriter.UpdateLastParamSyncFailed(ctx, dev.ID, time.Now(), errMsg); err != nil {
		s.logger.Error("task.failed sync: write last_param_sync_failed",
			zap.String("device_sn", t.DeviceSN),
			zap.String("task_id", t.ID),
			zap.Error(err))
		return err
	}
	s.logger.Info("param sync failure recorded",
		zap.String("device_sn", t.DeviceSN),
		zap.String("task_id", t.ID),
		zap.String("command_key", t.CommandKey),
		zap.String("error", errMsg))
	return nil
}

func (s *RPCResponseSubscriber) handlePasswordResetFallback(ctx context.Context, t *task.Task) error {
	if t == nil || s.taskEnqueuer == nil {
		return nil
	}
	if t.Method != resetLMTPasswordMethod {
		return nil
	}
	if strings.HasPrefix(t.CommandKey, resetLMTPasswordFallbackPrefix) {
		return nil
	}
	paramsJSON, err := json.Marshal(map[string]string{
		"message_type": resetLMTPasswordFallbackMethod,
		"fallback_of":  t.ID,
	})
	if err != nil {
		return fmt.Errorf("marshal password reset fallback params: %w", err)
	}
	commandKey := fmt.Sprintf("%s%s", resetLMTPasswordFallbackPrefix, uuid.New().String()[:8])
	created, err := s.taskEnqueuer.CreateTask(ctx, &task.CreateTaskRequest{
		DeviceSN:    t.DeviceSN,
		Method:      resetLMTPasswordFallbackMethod,
		Params:      paramsJSON,
		Priority:    t.Priority,
		CommandKey:  commandKey,
		Source:      t.Source,
		CreatorID:   t.CreatorID,
		Description: "reset LMT password fallback",
	})
	if err != nil {
		s.logger.Warn("enqueue password reset fallback failed",
			zap.String("device_sn", t.DeviceSN),
			zap.String("task_id", t.ID),
			zap.Error(err))
		return err
	}
	s.logger.Info("password reset fallback queued",
		zap.String("device_sn", t.DeviceSN),
		zap.String("failed_task_id", t.ID),
		zap.String("fallback_task_id", created.ID),
		zap.String("fallback_method", resetLMTPasswordFallbackMethod))
	return nil
}

func isDeleteObjectAlreadyGone(errMsg string) bool {
	normalized := strings.ToLower(errMsg)
	return strings.Contains(normalized, "object instance not found") ||
		strings.Contains(normalized, "instance not found")
}

func (s *RPCResponseSubscriber) handleDeleteObjectAlreadyGone(ctx context.Context, t *task.Task) error {
	if t == nil || s.paramRepo == nil {
		return nil
	}
	var params struct {
		ObjectName string `json:"object_name"`
	}
	if err := json.Unmarshal(t.Params, &params); err != nil || params.ObjectName == "" {
		s.logger.Warn("delete_object.failed_not_found: parse object_name failed",
			zap.String("device_sn", t.DeviceSN),
			zap.String("task_id", t.ID),
			zap.Error(err))
		return nil
	}
	device, err := s.deviceLookup.GetBySerialNumber(ctx, t.DeviceSN)
	if err != nil || device == nil {
		s.logger.Warn("delete_object.failed_not_found: device lookup failed",
			zap.String("device_sn", t.DeviceSN),
			zap.String("task_id", t.ID),
			zap.Error(err))
		return nil
	}
	deleted, err := s.paramRepo.DeleteByPathPrefix(ctx, device.ID, params.ObjectName)
	if err != nil {
		s.logger.Error("delete_object.failed_not_found: clean stale device_parameters failed",
			zap.String("device_sn", t.DeviceSN),
			zap.String("task_id", t.ID),
			zap.String("object_name", params.ObjectName),
			zap.Error(err))
		return err
	}
	s.logger.Info("delete_object.failed_not_found: stale device_parameters cleaned",
		zap.String("device_sn", t.DeviceSN),
		zap.String("task_id", t.ID),
		zap.String("object_name", params.ObjectName),
		zap.Int64("deleted_rows", deleted))
	return nil
}

// handleDeleteObjectResponse 处理 CPE 对 DeleteObject 的成功应答。
//
// ACS publish payload: { device_sn, method, object_name } (object_name 为
// 已删除对象的路径，含末尾 ".")。本 handler 据此清理 device_parameters 表中
// 所有路径以 object_name 开头的叶子参数 —— 否则前端 schema 查询仍会读到被
// 删实例的旧数据，UI 显示与设备真实状态偏离。
func (s *RPCResponseSubscriber) handleDeleteObjectResponse(ctx context.Context, evt event.Event) error {
	var payload map[string]interface{}
	if err := evt.DecodePayload(&payload); err != nil {
		s.logger.Warn("delete_object.response: decode payload", zap.Error(err))
		return nil
	}
	deviceSN, _ := payload["device_sn"].(string)
	objectName, _ := payload["object_name"].(string)
	if deviceSN == "" || objectName == "" {
		s.logger.Warn("delete_object.response: missing device_sn or object_name",
			zap.String("device_sn", deviceSN), zap.String("object_name", objectName))
		return nil
	}
	device, err := s.deviceLookup.GetBySerialNumber(ctx, deviceSN)
	if err != nil || device == nil {
		s.logger.Warn("delete_object.response: device lookup failed",
			zap.String("device_sn", deviceSN), zap.Error(err))
		return nil
	}
	deleted, err := s.paramRepo.DeleteByPathPrefix(ctx, device.ID, objectName)
	if err != nil {
		s.logger.Error("delete_object.response: clean device_parameters failed",
			zap.String("device_sn", deviceSN),
			zap.String("object_name", objectName),
			zap.Error(err))
		return err
	}
	s.logger.Info("delete_object.response: device_parameters cleaned",
		zap.String("device_sn", deviceSN),
		zap.String("object_name", objectName),
		zap.Int64("deleted_rows", deleted))
	return nil
}

// Stop 取消订阅，幂等。
func (s *RPCResponseSubscriber) Stop() {
	for _, sub := range s.subscriptions {
		if sub != nil {
			_ = sub.Unsubscribe()
		}
	}
	s.subscriptions = nil
}

// handleGPVResponse 处理 ACS 发布的 GetParameterValuesResponse 事件。
//
// Payload schema (来自 acs/handler.go:publishRPCResponseEvent)：
//
//	{
//	    "device_sn": "ABC123",
//	    "method": "GetParameterValuesResponse",
//	    "path": "<original request path, optional>",
//	    "parameter_values": [{"Name": "<privatePath>", "Value": "...", "Type": "..."}, ...]
//	}
//
// 处理流程：
//  1. 反序列化 parameter_values 为 []tr069.ParameterValueStruct
//  2. 按 device_sn 解析设备路由（DeviceLookup → ProductMatcher → Translator）
//  3. 对每个参数：ToStandard(privatePath) → standardPath（fallback 用原 path）
//  4. BatchUpsert device_parameters（parameter_path = standardPath）
func (s *RPCResponseSubscriber) handleGPVResponse(ctx context.Context, evt event.Event) error {
	// Payload schema 见 acs/handler.go:publishRPCResponseEvent —
	// map[string]interface{}{device_sn, method, path?, parameter_values}.
	var payload map[string]interface{}
	if err := evt.DecodePayload(&payload); err != nil {
		s.logger.Warn("decode payload", zap.Error(err))
		return nil
	}
	deviceSN, _ := payload["device_sn"].(string)
	if deviceSN == "" {
		return nil
	}
	if source, _ := payload["task_source"].(string); source == string(task.TaskSourceParamSync) {
		// paramsync.ResultConsumer is the sole writer for durable sync runs.
		return nil
	}

	rawParams, ok := payload["parameter_values"]
	if !ok {
		return nil
	}
	commandKey, _ := payload["command_key"].(string)
	paramValues, err := decodeParameterValues(rawParams)
	if err != nil {
		s.logger.Warn("decode parameter_values from event",
			zap.String("device_sn", deviceSN),
			zap.Error(err),
		)
		return nil
	}
	if len(paramValues) == 0 {
		return nil
	}

	device, err := s.deviceLookup.GetBySerialNumber(ctx, deviceSN)
	if err != nil {
		s.logger.Warn("device lookup failed; retry GPV response",
			zap.String("device_sn", deviceSN),
			zap.Error(err),
		)
		return fmt.Errorf("lookup GPV response device %s: %w", deviceSN, err)
	}
	if device == nil {
		s.logger.Warn("device lookup failed; persist raw privatePath",
			zap.String("device_sn", deviceSN),
		)
		return nil
	}

	translator := s.resolveTranslator(ctx, device)
	persistParams := make([]tr069.ParameterValueStruct, 0, len(paramValues))
	translated := translator != nil
	missCount := 0
	for _, p := range paramValues {
		if translator == nil {
			persistParams = append(persistParams, p)
			continue
		}
		res := translator.ToStandard(p.Name)
		if !res.Found {
			missCount++
		}
		out := p
		out.Name = res.Translated
		persistParams = append(persistParams, out)
	}

	if translated && missCount > 0 {
		s.logger.Info("rpc response path translation partial miss",
			zap.String("device_sn", deviceSN),
			zap.Int("total", len(paramValues)),
			zap.Int("miss", missCount),
		)
	}

	return s.persist(
		ctx,
		device,
		persistParams,
		translated,
		!isPathBSyncCommandKey(commandKey),
		strings.HasPrefix(commandKey, "geofence:"),
	)
}

// resolveTranslator 重复 mml/fanout.go translateParamRefs 的路由解析；
// 返回 nil 表示走 fallback（原 privatePath 入库）。
func (s *RPCResponseSubscriber) resolveTranslator(ctx context.Context, device *model.Device) *parammodel.Translator {
	if s.productMatcher == nil || s.translatorFactory == nil {
		return nil
	}
	matchRes, err := s.productMatcher.MatchProductClass(ctx, device.ProductClass)
	if err != nil || matchRes == nil || matchRes.Product == nil || matchRes.Product.ParamModelID == nil {
		// Orphan product/model is a controlled fallback: persist the original
		// privatePath so the RPC result is not lost, while keeping the condition
		// visible for product-model catalog cleanup.
		s.logger.Warn("uplink path translation fallback: product/param_model unresolved",
			zap.String("device_sn", device.SerialNumber),
			zap.String("product_class", device.ProductClass),
			zap.Error(err),
		)
		return nil
	}
	translator, err := s.translatorFactory.Translator(ctx, matchRes.Product.ID, device.FirmwareVersion)
	if err != nil {
		// ErrNoMapping 是合法业务态（产品未配 mapping），不算错误。
		if !errors.Is(err, parammodel.ErrNoMapping) {
			s.logger.Warn("build translator failed",
				zap.String("device_sn", device.SerialNumber),
				zap.String("product_id", matchRes.Product.ID.String()),
				zap.String("sw_version", device.FirmwareVersion),
				zap.Error(err),
			)
		}
		return nil
	}
	return translator
}

// persist 把翻译后的参数批量写 device_parameters。translated 参数仅供日志，
// 区分本次入库的 parameter_path 是 standardPath 还是 privatePath。
func (s *RPCResponseSubscriber) persist(
	ctx context.Context,
	device *model.Device,
	params []tr069.ParameterValueStruct,
	translated bool,
	refreshInfo bool,
	strictInfoRefresh bool,
) error {
	if device == nil || device.ID == uuid.Nil || len(params) == 0 {
		return nil
	}
	now := time.Now()
	rows := make([]model.DeviceParameter, 0, len(params))
	for _, p := range params {
		if p.Name == "" {
			continue
		}
		rows = append(rows, model.DeviceParameter{
			DeviceID:       device.ID,
			ParameterPath:  p.Name,
			ParameterValue: p.Value,
			ParameterType:  model.ParamString,
			Writable:       false,
			LastUpdatedAt:  now,
		})
	}
	if len(rows) == 0 {
		return nil
	}
	if err := s.paramRepo.BatchUpsert(ctx, device.ID, rows); err != nil {
		s.logger.Error("batch upsert device parameters from rpc response",
			zap.String("device_id", device.ID.String()),
			zap.Bool("translated", translated),
			zap.Int("rows", len(rows)),
			zap.Error(err),
		)
		return err
	}
	if refreshInfo && s.infoRefresher != nil {
		if _, err := s.infoRefresher.SyncFromParameters(ctx, device.ID, device.Carrier, device.Technology, device.ProductClass); err != nil {
			s.logger.Warn("refresh device_info snapshot after rpc response persist failed",
				zap.String("device_id", device.ID.String()),
				zap.String("device_sn", device.SerialNumber),
				zap.Error(err),
			)
			if strictInfoRefresh {
				return fmt.Errorf("refresh geofence device summary after GPV: %w", err)
			}
		}
	}
	s.logger.Info("rpc response parameters persisted",
		zap.String("device_id", device.ID.String()),
		zap.Bool("translated_to_standard", translated),
		zap.Int("rows", len(rows)),
	)
	return nil
}

func isPathBSyncCommandKey(commandKey string) bool {
	return strings.HasPrefix(commandKey, "sync-gpv-")
}

// decodeParameterValues 容忍多种 payload 形态（rawParams 可能是
// []tr069.ParameterValueStruct 原型，也可能 NATS 反序列化为 []interface{}
// of map[string]interface{}）。
func decodeParameterValues(raw interface{}) ([]tr069.ParameterValueStruct, error) {
	switch v := raw.(type) {
	case nil:
		return []tr069.ParameterValueStruct{}, nil
	case []tr069.ParameterValueStruct:
		return v, nil
	case []interface{}:
		b, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("marshal raw slice: %w", err)
		}
		var out []tr069.ParameterValueStruct
		if err := json.Unmarshal(b, &out); err != nil {
			return nil, fmt.Errorf("unmarshal into ParameterValueStruct: %w", err)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unexpected type %T", raw)
	}
}
