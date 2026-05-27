package devsweep

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/time/rate"

	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/product"
)

// Error codes 给 CLI 退出码/JSON output 使用。
const (
	ErrCodeDeviceNotFound       = "device_not_found"
	ErrCodeDeviceOffline        = "device_offline"
	ErrCodeProductClassEmpty    = "product_class_empty"
	ErrCodeOrphanProduct        = "orphan_product_class"
	ErrCodeParamModelUnset      = "param_model_unset"
	ErrCodeNoMappings           = "no_mappings"
	ErrCodeUnsupportedRateHigh  = "unsupported_rate_high"
	ErrCodeParamModelWideUnconf = "param_model_wide_unconfirmed"
)

// DeviceLookup 是 Service 反查设备的最小依赖。
//
// 生产环境由 *device.DeviceService 满足。测试可注入 stub。
type DeviceLookup interface {
	GetBySerialNumber(ctx context.Context, sn string) (*model.Device, error)
}

// ProductMatcher 是 Service 路由 productClass → product 装配件的最小依赖。
//
// 生产环境由 *product.Registry 满足。测试可注入 stub。
type ProductMatcher interface {
	MatchProductClass(ctx context.Context, productClass string) (*product.MatchResult, error)
}

// ParamLookup 是 Service 取 paramModel 默认映射集的最小依赖。
//
// 生产环境由 *parammodel.Registry 满足。测试可注入 stub。
type ParamLookup interface {
	GetByParamModel(ctx context.Context, paramModelID uuid.UUID) (*parammodel.MappingSet, error)
}

// Service 编排单设备 sweep 流程：解析 → 候选 → 探测 → 安全门 → 写库。
//
// 所有依赖通过构造函数注入；不做"读 viper 全局配置"等隐式行为。
type Service struct {
	devices  DeviceLookup
	products ProductMatcher
	params   ParamLookup
	prober   Prober
	repo     Repository
	logger   *zap.Logger
}

// NewService 构造 Service。所有参数必填；nil logger → zap.Nop()。
func NewService(devices DeviceLookup, products ProductMatcher, params ParamLookup, prober Prober, repo Repository, logger *zap.Logger) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{
		devices:  devices,
		products: products,
		params:   params,
		prober:   prober,
		repo:     repo,
		logger:   logger.Named("devsweep"),
	}
}

// Run 执行单设备 sweep。Result 字段在不同阶段填写；
// Aborted=true 时跳过后续步骤并保留已填字段。
//
// 用户决策：dry-run 默认（opts.Apply=false）；--apply 才写 DB。
func (s *Service) Run(ctx context.Context, opts Options) (*Result, error) {
	opts = normalizeOptions(opts)

	res := &Result{
		DeviceSN:  opts.DeviceSN,
		DryRun:    !opts.Apply,
		StartedAt: time.Now(),
	}

	// Step 1: 解析设备
	dev, err := s.devices.GetBySerialNumber(ctx, opts.DeviceSN)
	if err != nil {
		return res, fmt.Errorf("lookup device %q: %w", opts.DeviceSN, err)
	}
	if dev == nil {
		res.Aborted = true
		res.ErrorCode = ErrCodeDeviceNotFound
		return res, fmt.Errorf("device not found: %s", opts.DeviceSN)
	}
	res.ProductClass = dev.ProductClass
	res.FirmwareVersion = dev.FirmwareVersion
	res.IsOnline = dev.IsOnline

	if strings.TrimSpace(dev.ProductClass) == "" {
		res.Aborted = true
		res.ErrorCode = ErrCodeProductClassEmpty
		return res, fmt.Errorf("device %s has empty product_class", opts.DeviceSN)
	}
	if !dev.IsOnline {
		res.Aborted = true
		res.ErrorCode = ErrCodeDeviceOffline
		return res, fmt.Errorf("device %s is offline", opts.DeviceSN)
	}

	// Step 2: 路由 product
	match, err := s.products.MatchProductClass(ctx, dev.ProductClass)
	if err != nil {
		if errors.Is(err, product.ErrOrphan) {
			res.Aborted = true
			res.ErrorCode = ErrCodeOrphanProduct
			return res, fmt.Errorf("product_class %q is orphan (no pattern matched)", dev.ProductClass)
		}
		return res, fmt.Errorf("match product_class %q: %w", dev.ProductClass, err)
	}
	if match == nil || match.Product == nil {
		res.Aborted = true
		res.ErrorCode = ErrCodeOrphanProduct
		return res, fmt.Errorf("product_class %q returned nil product", dev.ProductClass)
	}
	res.ProductID = match.Product.ID
	res.ProductName = match.Product.Name
	if match.Product.ParamModelID == nil {
		res.Aborted = true
		res.ErrorCode = ErrCodeParamModelUnset
		return res, fmt.Errorf("product %s has no param_model_id", match.Product.ID)
	}
	res.ParamModelID = *match.Product.ParamModelID

	// Step 3: 候选 path
	set, err := s.params.GetByParamModel(ctx, res.ParamModelID)
	if err != nil {
		if errors.Is(err, parammodel.ErrNoMapping) {
			res.Aborted = true
			res.ErrorCode = ErrCodeNoMappings
			return res, fmt.Errorf("param_model %s has no mappings", res.ParamModelID)
		}
		return res, fmt.Errorf("get by param_model %s: %w", res.ParamModelID, err)
	}
	res.ParamModelName = "" // Registry 未直接返回 name；不阻塞流程
	candidates := filterCandidates(set.Mappings, opts.Prefix)
	res.CandidateCount = len(candidates)
	if res.CandidateCount == 0 {
		res.Aborted = true
		res.ErrorCode = ErrCodeNoMappings
		return res, fmt.Errorf("no active mappings (prefix=%q) for param_model %s", opts.Prefix, res.ParamModelID)
	}

	// Step 4: 分批探测
	records := s.probeAll(ctx, opts, candidates)
	res.PerPath = records

	unsupportedPaths := make([]string, 0, len(records))
	unknownPaths := make([]string, 0)
	for _, r := range records {
		switch r.Outcome {
		case OutcomeSupported:
			res.SupportedCount++
		case OutcomeUnsupported:
			res.UnsupportedCount++
			unsupportedPaths = append(unsupportedPaths, r.StandardPath)
		case OutcomeUnknown:
			res.UnknownCount++
			unknownPaths = append(unknownPaths, r.StandardPath)
		}
	}
	res.UnsupportedPaths = unsupportedPaths
	res.UnknownPaths = unknownPaths

	// Step 5: 安全门 — 仅在有 unsupported 需要写时才检查
	devCount, err := s.repo.CountDevicesByParamModel(ctx, res.ParamModelID)
	if err != nil {
		s.logger.Warn("count devices by param_model failed", zap.Error(err))
	}
	res.ParamModelDevicesAffected = int(devCount)

	if len(unsupportedPaths) > 0 {
		// 比例阈值
		fraction := float64(res.UnsupportedCount) / float64(res.CandidateCount)
		if fraction > opts.Safety.MaxUnsupportedFraction && !opts.Force {
			res.Aborted = true
			res.ErrorCode = ErrCodeUnsupportedRateHigh
			s.finalize(res)
			return res, fmt.Errorf(
				"unsupported rate %.2f > %.2f (use --force to override)",
				fraction, opts.Safety.MaxUnsupportedFraction)
		}
		// paramModel-wide 阈值
		if int(devCount) > opts.Safety.MaxParamModelDevices && !opts.ConfirmParamModelWide {
			res.Aborted = true
			res.ErrorCode = ErrCodeParamModelWideUnconf
			s.finalize(res)
			return res, fmt.Errorf(
				"param_model %s affects %d devices > %d (use --confirm-paramodel-wide to acknowledge)",
				res.ParamModelID, devCount, opts.Safety.MaxParamModelDevices)
		}
	}

	// Step 6: 写库（仅 --apply）
	if opts.Apply && len(unsupportedPaths) > 0 {
		affected, err := s.repo.MarkUnsupportedBatch(ctx, res.ParamModelID, unsupportedPaths)
		if err != nil {
			s.finalize(res)
			return res, fmt.Errorf("mark unsupported batch: %w", err)
		}
		res.MarkedCount = int(affected)
		s.logger.Info("devsweep applied",
			zap.String("device_sn", opts.DeviceSN),
			zap.String("operator", opts.Operator),
			zap.String("param_model_id", res.ParamModelID.String()),
			zap.Int("candidates", res.CandidateCount),
			zap.Int("supported", res.SupportedCount),
			zap.Int("unsupported", res.UnsupportedCount),
			zap.Int("unknown", res.UnknownCount),
			zap.Int("marked", res.MarkedCount),
			zap.Int("paramodel_devices", int(devCount)),
		)
	} else {
		s.logger.Info("devsweep dry-run",
			zap.String("device_sn", opts.DeviceSN),
			zap.String("operator", opts.Operator),
			zap.Int("candidates", res.CandidateCount),
			zap.Int("supported", res.SupportedCount),
			zap.Int("unsupported", res.UnsupportedCount),
			zap.Int("unknown", res.UnknownCount),
			zap.Int("paramodel_devices", int(devCount)),
		)
	}

	s.finalize(res)
	return res, nil
}

// probeAll 把 candidates 拆批 + rate-limit 后逐批 prober.Probe。
//
// T-0180 起 BatchSize 默认 16: ACS handler 对 GPV failure 写结构化 param_faults[]
// (与 SPV schema 对齐),prober 内部 retry 循环把 unknown 子集递归重发,O(N_bad)
// 收敛。BatchSize=1 仍可用(精准归因每条 path,但 765 path 全 sweep ~25min);
// BatchSize=16 在含 11 bad path 的典型场景下 ~30s 完成。
func (s *Service) probeAll(ctx context.Context, opts Options, candidates []parammodel.ParamMapping) []ProbeRecord {
	out := make([]ProbeRecord, 0, len(candidates))
	if len(candidates) == 0 {
		return out
	}

	limiter := rate.NewLimiter(rate.Limit(opts.RPCRate), 1)
	batchSize := opts.BatchSize
	if batchSize <= 0 {
		batchSize = 16
	}

	for batchIdx, start := 0, 0; start < len(candidates); start += batchSize {
		end := start + batchSize
		if end > len(candidates) {
			end = len(candidates)
		}
		batch := candidates[start:end]
		standardPaths := make([]string, 0, len(batch))
		probePaths := make([]string, 0, len(batch))
		for _, m := range batch {
			standardPaths = append(standardPaths, m.StandardPath)
			probePaths = append(probePaths, NormalizeForProbe(m.StandardPath))
		}

		if err := limiter.Wait(ctx); err != nil {
			// ctx 被取消 — 把剩余 candidates 标 unknown 后返回
			for _, m := range candidates[start:] {
				out = append(out, ProbeRecord{
					StandardPath: m.StandardPath, ProbePath: NormalizeForProbe(m.StandardPath),
					Outcome: OutcomeUnknown, Batch: batchIdx,
					FaultMessage: "rate-limit wait: " + err.Error(),
				})
			}
			return out
		}

		records := s.prober.Probe(ctx, opts.DeviceSN, probePaths, batchIdx)
		// 把 prober 返的 ProbePath 对回 StandardPath（prober 不知道 standardPath）
		for i, r := range records {
			if i < len(standardPaths) {
				r.StandardPath = standardPaths[i]
			}
			out = append(out, r)
		}
		batchIdx++
	}
	return out
}

// finalize 填末态字段：FinishedAt / DurationMS。
func (s *Service) finalize(res *Result) {
	res.FinishedAt = time.Now()
	res.DurationMS = res.FinishedAt.Sub(res.StartedAt).Milliseconds()
}

// filterCandidates 应用 prefix 过滤 + entry_type=parameter 过滤 + is_active=true。
//
// Registry.GetByParamModel 已用 is_active=true 过滤；这里只补 prefix 与
// entry_type — 对象前缀（entry_type=object）发 GPV 通常返回多实例参数列表，
// 不是探测单 path 是否存在的好载荷，跳过。
func filterCandidates(mappings []parammodel.ParamMapping, prefix string) []parammodel.ParamMapping {
	out := make([]parammodel.ParamMapping, 0, len(mappings))
	for _, m := range mappings {
		if !m.IsActive {
			continue
		}
		if m.EntryType != "" && m.EntryType != "parameter" {
			continue
		}
		if prefix != "" && !strings.HasPrefix(m.StandardPath, prefix) {
			continue
		}
		out = append(out, m)
	}
	return out
}

// normalizeOptions 设置 Options 的默认值（仅当字段为零值）。
func normalizeOptions(opts Options) Options {
	if opts.BatchSize <= 0 {
		// T-0180: ACS handler 现已对 GPV failure 写结构化 param_faults[],prober
		// 用 retry 循环把 unknown 子集重发 (O(N_bad) 收敛),BatchSize>1 可靠。
		// 默认 16 平衡:速度(BLQ 765 path / 16 ≈ 48 batches) vs RPC 单包上限
		// (主流 CPE 支持 32-100 names per GPV,16 留 safety margin)。
		opts.BatchSize = 16
	}
	if opts.RPCTimeout <= 0 {
		opts.RPCTimeout = 30 * time.Second
	}
	if opts.RPCRate <= 0 {
		opts.RPCRate = 5
	}
	if opts.Safety.MaxUnsupportedFraction <= 0 {
		opts.Safety.MaxUnsupportedFraction = SafetyDefaultMaxUnsupportedFraction
	}
	if opts.Safety.MaxParamModelDevices <= 0 {
		opts.Safety.MaxParamModelDevices = SafetyDefaultMaxParamModelDevices
	}
	return opts
}
