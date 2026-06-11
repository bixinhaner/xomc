package kpi

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/pm/counter"
	"github.com/omcgo/omcgo/internal/pm/kpi/router"
)

// KPIRouter 抽象 router.Router，方便单元测试 mock。
type KPIRouter interface {
	LookupByDevice(ctx context.Context, deviceSN string) (*router.KPIRoute, error)
}

// KPIEngine 按设备 → 产品 → 平台公式路由计算 KPI（T-0164-P1 / G1）。
//
// 旧实现（按 carrier+technology 过滤代码硬编码 KPIDefinitions）已下线，
// 现在所有 KPI 公式由 DB 元数据声明：
//   - 来源：indicator.PlatformFormulaRepository / IndicatorRepository
//   - 路由：router.Router.LookupByDevice — 决定"哪台设备算哪些 KPI"
//   - 调用方语义：carrier/tech 只作为 KPIValue 行的"分类标签"，不再用于过滤
//     公式集合；保留参数是为了多运营商环境下的存储分桶 + 历史兼容。
type KPIEngine struct {
	counterRepo counter.CounterRepository
	kpiRepo     KPIRepository
	router      KPIRouter
	logger      *zap.Logger
}

// NewKPIEngine 构造 Engine。router 必须非 nil（旧实现允许 carrier registry 为空，
// 现在路由是核心依赖）。logger nil → NewNop。
func NewKPIEngine(
	counterRepo counter.CounterRepository,
	kpiRepo KPIRepository,
	kpiRouter KPIRouter,
	logger *zap.Logger,
) *KPIEngine {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &KPIEngine{
		counterRepo: counterRepo,
		kpiRepo:     kpiRepo,
		router:      kpiRouter,
		logger:      logger.Named("kpi.engine"),
	}
}

// Calculate computes KPI values for a device/cell in a time range.
//
// 与旧签名相比 carrier+tech 仍保留 — 不用作公式集合的过滤键，而是作为
// 落库标签（KPIValue.Carrier / Technology）传入。Router 路由失败的设备
// 静默 skip（log warn）— 不应让一台 orphan 设备阻塞整批文件的 KPI 计算。
func (e *KPIEngine) Calculate(
	ctx context.Context,
	deviceID uuid.UUID,
	oui, deviceSN string,
	cellID string,
	startTime, endTime time.Time,
	carrierCode model.CarrierCode,
	tech model.Technology,
) ([]model.KPIValue, error) {
	if e.router == nil {
		return nil, errors.New("kpi.Engine.Calculate: router not configured")
	}

	route, err := e.router.LookupByDevice(ctx, deviceSN)
	if err != nil {
		if errors.Is(err, router.ErrProductNotMatched) || errors.Is(err, router.ErrInvalidProductMetadata) {
			e.logger.Warn("kpi route unavailable; skip device",
				zap.String("device_sn", deviceSN),
				zap.Error(err))
			return nil, nil
		}
		return nil, fmt.Errorf("kpi route lookup for %q: %w", deviceSN, err)
	}
	if route == nil || len(route.KPIs) == 0 {
		return nil, nil
	}

	counterNames := uniqueCounterDeps(route.KPIs)
	if len(counterNames) == 0 {
		return nil, nil
	}

	counterValues, err := e.counterRepo.QueryForKPI(ctx, deviceID, cellID, counterNames, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("query counters for kpi: %w", err)
	}

	return e.evaluateRoute(route, counterValues, deviceID, oui, deviceSN, cellID, endTime, carrierCode, tech), nil
}

// evaluateRoute 把一组 counter 值代入 route 的每条 KPI 公式，产出 KPIValue 切片。
// 从 Calculate 抽出，供单 cell（Calculate）与批量多 cell（CalculateCellsAndStore）两条
// 路径复用，保证两者公式求值/跳过/落库字段语义逐字一致。
//   - 公式解析失败：log warn 跳过该 KPI（不影响其他 KPI）。
//   - counter 缺失 / 除零（Evaluate 报错）：静默跳过该 KPI，不向上抛错。
func (e *KPIEngine) evaluateRoute(
	route *router.KPIRoute,
	counterValues map[string]float64,
	deviceID uuid.UUID,
	oui, deviceSN, cellID string,
	endTime time.Time,
	carrierCode model.CarrierCode,
	tech model.Technology,
) []model.KPIValue {
	results := make([]model.KPIValue, 0, len(route.KPIs))
	for _, k := range route.KPIs {
		parsed, err := ParseFormula(k.Formula)
		if err != nil {
			e.logger.Warn("skip kpi with invalid formula",
				zap.String("kpi", k.Name),
				zap.String("formula", k.Formula),
				zap.Error(err))
			continue
		}
		value, err := parsed.Evaluate(counterValues)
		if err != nil {
			// counter 缺失 / 除零 → 跳过这条 KPI；不向上抛错。
			continue
		}
		results = append(results, model.KPIValue{
			Time:        endTime,
			DeviceID:    deviceID,
			OUI:         oui,
			DeviceSN:    deviceSN,
			CellID:      cellID,
			IndicatorID: k.IndicatorID,
			KPIName:     k.Name,
			KPIValue:    value,
			Carrier:     carrierCode,
			Technology:  tech,
		})
	}
	return results
}

// CalculateAndStore calculates KPIs and persists them.
func (e *KPIEngine) CalculateAndStore(
	ctx context.Context,
	deviceID uuid.UUID,
	oui, deviceSN string,
	cellID string,
	collectTime time.Time,
	carrierCode model.CarrierCode,
	tech model.Technology,
) ([]model.KPIValue, error) {
	startTime := collectTime.Add(-15 * time.Minute)
	results, err := e.Calculate(ctx, deviceID, oui, deviceSN, cellID, startTime, collectTime, carrierCode, tech)
	if err != nil {
		return nil, err
	}
	if len(results) > 0 {
		if err := e.kpiRepo.BatchInsert(ctx, results); err != nil {
			return nil, fmt.Errorf("store kpi values: %w", err)
		}
	}
	return results, nil
}

// CalculateCellsAndStore 批量计算单个 PM 文件内多个 cell 的 KPI 并一次性落库。
//
// 替代 collector 旧热路径"对每个 cellID 各调一次 CalculateAndStore"的 N 次扇出
// （GSM 真机 256 cell/文件 → 256×(route 查 + 全设备 counter 扫描 + KPI 写)）。本方法：
//   - route 只查一次（同一 deviceSN，所有 cell 共享路由结果）；
//   - counter 值一次查全（QueryForKPICells 单查询按 cell 分桶），替代每 cell 一次全设备扫描；
//   - 所有 cell 的 KPIValue 累积后一次 BatchInsert（大批量自动走 COPY）。
//
// 语义与逐 cell CalculateAndStore 等价（QueryForKPICells 复刻 QueryForKPI 的 cell 过滤 /
// "" 跨全 cell 求和 / period_seconds 注入；evaluateRoute 复用同一求值逻辑）。
// 返回写入的 KPIValue 行数。route 不可用（产品未匹配等）时静默 skip 返回 0。
func (e *KPIEngine) CalculateCellsAndStore(
	ctx context.Context,
	deviceID uuid.UUID,
	oui, deviceSN string,
	cellIDs []string,
	collectTime time.Time,
	carrierCode model.CarrierCode,
	tech model.Technology,
) (int, error) {
	if e.router == nil {
		return 0, errors.New("kpi.Engine.CalculateCellsAndStore: router not configured")
	}
	if len(cellIDs) == 0 {
		return 0, nil
	}

	route, err := e.router.LookupByDevice(ctx, deviceSN)
	if err != nil {
		if errors.Is(err, router.ErrProductNotMatched) || errors.Is(err, router.ErrInvalidProductMetadata) {
			e.logger.Warn("kpi route unavailable; skip device",
				zap.String("device_sn", deviceSN),
				zap.Error(err))
			return 0, nil
		}
		return 0, fmt.Errorf("kpi route lookup for %q: %w", deviceSN, err)
	}
	if route == nil || len(route.KPIs) == 0 {
		return 0, nil
	}

	counterNames := uniqueCounterDeps(route.KPIs)
	if len(counterNames) == 0 {
		return 0, nil
	}

	startTime := collectTime.Add(-15 * time.Minute)
	perCell, err := e.counterRepo.QueryForKPICells(ctx, deviceID, cellIDs, counterNames, startTime, collectTime)
	if err != nil {
		return 0, fmt.Errorf("query counters for kpi (cells): %w", err)
	}

	all := make([]model.KPIValue, 0, len(cellIDs)*len(route.KPIs))
	for _, cellID := range cellIDs {
		counterValues := perCell[cellID]
		// counterValues 可能为 nil（该 cell 无任何 counter 行且 period<=0）；evaluateRoute 对
		// nil map 读取返回 0/缺失 → 公式 Evaluate 跳过，行为与旧逐 cell 路径一致。
		all = append(all, e.evaluateRoute(route, counterValues, deviceID, oui, deviceSN, cellID, collectTime, carrierCode, tech)...)
	}
	if len(all) == 0 {
		return 0, nil
	}
	if err := e.kpiRepo.BatchInsert(ctx, all); err != nil {
		return 0, fmt.Errorf("store kpi values: %w", err)
	}
	return len(all), nil
}

// uniqueCounterDeps 去重收集 route.KPIs 中所有公式依赖的 counter 名。
func uniqueCounterDeps(kpis []router.KPIDef) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0)
	for _, k := range kpis {
		for _, name := range k.Dependencies {
			if _, dup := seen[name]; dup {
				continue
			}
			seen[name] = struct{}{}
			out = append(out, name)
		}
	}
	return out
}
