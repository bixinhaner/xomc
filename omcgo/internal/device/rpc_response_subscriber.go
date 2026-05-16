package device

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/product"
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
//   下行 (Stage 1) standardPath → privatePath (in mml/fanout.go)
//   上行 (Stage 2) privatePath → standardPath (本文件)
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
	logger            *zap.Logger

	subscriptions []event.Subscription
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

// NewRPCResponseSubscriber 构造订阅者。Start() 时才真正订阅 bus。
// 任一依赖为 nil 时订阅者会跳过翻译，CPE 响应仍按 privatePath 入库（兼容
// 启动早期 / 测试场景）。
func NewRPCResponseSubscriber(
	bus event.EventBus,
	productMatcher ProductMatcher,
	translatorFactory ParamModelTranslatorFactory,
	deviceLookup RPCDeviceLookup,
	paramRepo DeviceParameterRepository,
	logger *zap.Logger,
) *RPCResponseSubscriber {
	return &RPCResponseSubscriber{
		bus:               bus,
		productMatcher:    productMatcher,
		translatorFactory: translatorFactory,
		deviceLookup:      deviceLookup,
		paramRepo:         paramRepo,
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
	sub, err := s.bus.Subscribe(event.SubjectCommandGetParamsResponse, s.handleGPVResponse)
	if err != nil {
		return fmt.Errorf("subscribe %s: %w", event.SubjectCommandGetParamsResponse, err)
	}
	s.subscriptions = append(s.subscriptions, sub)
	s.logger.Info("RPC response subscriber started",
		zap.String("subject", event.SubjectCommandGetParamsResponse),
	)
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

	rawParams, ok := payload["parameter_values"]
	if !ok {
		return nil
	}
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
	if err != nil || device == nil {
		s.logger.Warn("device lookup failed; persist raw privatePath",
			zap.String("device_sn", deviceSN),
			zap.Error(err),
		)
		return s.persist(ctx, uuid.Nil, paramValues, false /*translated*/)
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

	return s.persist(ctx, device.ID, persistParams, translated)
}

// resolveTranslator 重复 mml/fanout.go translateParamRefs 的路由解析；
// 返回 nil 表示走 fallback（原 privatePath 入库）。
func (s *RPCResponseSubscriber) resolveTranslator(ctx context.Context, device *model.Device) *parammodel.Translator {
	if s.productMatcher == nil || s.translatorFactory == nil {
		return nil
	}
	matchRes, err := s.productMatcher.MatchProductClass(ctx, device.ProductClass)
	if err != nil || matchRes == nil || matchRes.Product == nil || matchRes.Product.ParamModelID == nil {
		return nil
	}
	translator, err := s.translatorFactory.Translator(ctx, *matchRes.Product.ParamModelID, device.FirmwareVersion)
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
	ctx context.Context, deviceID uuid.UUID, params []tr069.ParameterValueStruct, translated bool,
) error {
	if deviceID == uuid.Nil || len(params) == 0 {
		return nil
	}
	now := time.Now()
	rows := make([]model.DeviceParameter, 0, len(params))
	for _, p := range params {
		if p.Name == "" {
			continue
		}
		rows = append(rows, model.DeviceParameter{
			DeviceID:       deviceID,
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
	if err := s.paramRepo.BatchUpsert(ctx, deviceID, rows); err != nil {
		s.logger.Error("batch upsert device parameters from rpc response",
			zap.String("device_id", deviceID.String()),
			zap.Bool("translated", translated),
			zap.Int("rows", len(rows)),
			zap.Error(err),
		)
		return err
	}
	s.logger.Info("rpc response parameters persisted",
		zap.String("device_id", deviceID.String()),
		zap.Bool("translated_to_standard", translated),
		zap.Int("rows", len(rows)),
	)
	return nil
}

// decodeParameterValues 容忍多种 payload 形态（rawParams 可能是
// []tr069.ParameterValueStruct 原型，也可能 NATS 反序列化为 []interface{}
// of map[string]interface{}）。
func decodeParameterValues(raw interface{}) ([]tr069.ParameterValueStruct, error) {
	switch v := raw.(type) {
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
