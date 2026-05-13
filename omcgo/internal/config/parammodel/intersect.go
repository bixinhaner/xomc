package parammodel

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ── 错误 ──────────────────────────────────────────────────────────────

// ErrProductNotFound 在 IntersectCPEModel 时 product 不存在。
var ErrProductNotFound = errors.New("parammodel: product not found")

// ── 输入 / 输出 类型 ──────────────────────────────────────────────────

// CPEEntry 表示设备 FileType=11 上传 XML 中的一条 privatePath 记录。
//
// 字段语义对应 datamodel/xml_parser.go XMLParam（privatePath=Name attr，
// 元属性按设备实际上报取值）。entry_type 由调用方在解析阶段判定（object | parameter）。
type CPEEntry struct {
	PrivatePath   string
	EntryType     string // "object" | "parameter"
	Access        string
	DataType      string
	ChangeApplies string
	MinValue      *int64
	MaxValue      *int64
}

// IntersectInput 是 IntersectCPEModel 的最小入参集合。
//
// Entries 中每条 PrivatePath 必须唯一；重复以最后一条为准（设计 §1.8 未规定，从严按"覆盖"处理）。
type IntersectInput struct {
	ProductID       uuid.UUID
	SoftwareVersion string
	Entries         []CPEEntry
}

// IntersectResult 汇总一次交集写入的统计。
//
// Matched + SkippedNoStorable + Discarded = DefaultCount 中能在 CPE Entries 找到的部分。
// 不在 CPE 集合中的默认条目计入 DefaultsMissing。
type IntersectResult struct {
	ProductID         uuid.UUID
	SoftwareVersion   string
	ParamModelID      uuid.UUID
	DefaultCount      int   // param_mappings 该 paramModel 的总条目
	UploadedCount     int   // CPE Entries 长度（去重前）
	Matched           int   // 实际写入 discovered_param_mappings 的行数
	DefaultsMissing   int   // CPE 中找不到的默认条目数（保留默认，不写 discovered）
	UploadedExtras    int   // CPE 中没有对应默认的私有条目数（丢弃）
	OverrideDataType  bool  // 是否检测到 data_type 覆盖请求（true 视为非法并降级为 false）
	DurationMs        int64
}

// ── 写路径仓库 ────────────────────────────────────────────────────────

// IntersectRepository 抽象 IntersectService 的写路径。
//
// 与 Repository（读）拆分：读写关注点正交；测试也可独立 fake。
type IntersectRepository interface {
	// UpsertDiscoveredMappings 以 (productID, swVersion) 为键替换 discovered_param_mappings：
	//   1. DELETE WHERE product_id=? AND software_version=?
	//   2. INSERT 给定 mappings（每条带 standardPath/privatePath/entry_type 等完整字段）
	// 实现必须在单事务内完成；len(mappings)==0 时仍执行 DELETE（清空既有数据）。
	UpsertDiscoveredMappings(ctx context.Context, productID uuid.UUID, swVersion string, mappings []ParamMapping) error
}

// ── 失效器 ────────────────────────────────────────────────────────────

// IntersectInvalidator 抽象写完后的缓存失效操作。
// 生产环境由 *Registry 满足；测试可注入 stub 验证调用。
type IntersectInvalidator interface {
	InvalidateProduct(ctx context.Context, productID uuid.UUID, swVersion string) error
}

// ── 服务主体 ──────────────────────────────────────────────────────────

// IntersectService 实现设计 §1.8 交集逻辑：
//
//   1. 从 productGetter 取 product → ParamModelID 与 DeviceAttrsOverride
//   2. 从 Repository 取该 paramModel 的全量默认 mappings
//   3. 与 CPE Entries 按 privatePath 取交集
//   4. 按 device_attrs_override 决定每条交集结果的 5 个元属性来源
//   5. 通过 IntersectRepository 写入 discovered_param_mappings（DELETE+INSERT 事务）
//   6. 通过 IntersectInvalidator 失效该 (productID, swVersion) 缓存
//
// 注意：本服务不解析 CPE XML——XML 解析由 P2-06 model_upload 阶段在调用前完成。
type IntersectService struct {
	read        Repository
	write       IntersectRepository
	products    productGetter
	invalidator IntersectInvalidator
	logger      *zap.Logger
	metrics     *intersectMetrics
}

// NewIntersectService 构造一个 IntersectService。
//
// 参数 invalidator 可为 nil（典型测试场景）；生产 provider 必须注入 Registry。
// products 为 nil → IntersectCPEModel 返回 ErrProductGetterUnset。
func NewIntersectService(
	read Repository,
	write IntersectRepository,
	products productGetter,
	invalidator IntersectInvalidator,
	metrics *intersectMetrics,
	logger *zap.Logger,
) *IntersectService {
	if metrics == nil {
		metrics = NewIntersectMetrics(nil)
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &IntersectService{
		read:        read,
		write:       write,
		products:    products,
		invalidator: invalidator,
		logger:      logger.Named("parammodel.intersect"),
		metrics:     metrics,
	}
}

// IntersectCPEModel 执行一次基站上传参数模型 ∩ 默认模型 的交集计算并落库。
//
// 边界：
//   - SoftwareVersion 为空 → ErrNoMapping（不允许写入空 swVersion 行，与 PG 列定义一致）
//   - product 不存在 → ErrProductNotFound
//   - product.ParamModelID == nil → ErrNoParamModel
//   - 默认 mappings 空 → 仍尝试 DELETE + INSERT(0)，确保旧数据被清理
//
// data_type 覆盖语义：业务禁止覆盖 data_type，JSON 即便写 true 也强制视为 false（设计 §1.7 / §11.D 决议）。
// 检测到时仅 WARN 一次，不报错。
func (s *IntersectService) IntersectCPEModel(ctx context.Context, input IntersectInput) (*IntersectResult, error) {
	t0 := time.Now()
	defer func() {
		s.metrics.duration.Observe(time.Since(t0).Seconds())
	}()

	if s.products == nil {
		s.metrics.outcome("err_no_product_getter")
		return nil, ErrProductGetterUnset
	}
	if input.SoftwareVersion == "" {
		s.metrics.outcome("err_no_swversion")
		return nil, fmt.Errorf("intersect: software_version is empty")
	}

	prod, err := s.products.GetProductByID(ctx, input.ProductID)
	if err != nil {
		s.metrics.outcome("err_product_lookup")
		return nil, fmt.Errorf("lookup product %s: %w", input.ProductID, err)
	}
	if prod == nil {
		s.metrics.outcome("err_product_missing")
		return nil, ErrProductNotFound
	}
	if prod.ParamModelID == nil {
		s.metrics.outcome("err_no_param_model")
		return nil, ErrNoParamModel
	}

	defaults, err := s.read.ListMappingsByParamModel(ctx, *prod.ParamModelID)
	if err != nil {
		s.metrics.outcome("err_list_default")
		return nil, fmt.Errorf("list default mappings %s: %w", *prod.ParamModelID, err)
	}

	override, dtOverrideRequested := normalizeOverride(prod.DeviceAttrsOverride)
	if dtOverrideRequested {
		s.logger.Warn("device_attrs_override.data_type=true rejected at intersect layer; using default data_type",
			zap.String("product_id", prod.ID.String()),
			zap.String("product_name", prod.Name))
		s.metrics.dataTypeOverrideRejected()
	}

	cpeIndex := make(map[string]CPEEntry, len(input.Entries))
	for _, e := range input.Entries {
		cpeIndex[e.PrivatePath] = e
	}

	matched := make([]ParamMapping, 0, len(defaults))
	defaultsMissing := 0
	for _, dm := range defaults {
		cpe, ok := cpeIndex[dm.PrivatePath]
		if !ok {
			defaultsMissing++
			continue
		}
		row := buildIntersectRow(prod.ID, input.SoftwareVersion, dm, cpe, override)
		matched = append(matched, row)
	}

	uploadedExtras := len(cpeIndex) - len(matched)
	if uploadedExtras < 0 {
		uploadedExtras = 0 // 重复 privatePath 已被 map 折叠，不可能为负，仅防御
	}

	if err := s.write.UpsertDiscoveredMappings(ctx, prod.ID, input.SoftwareVersion, matched); err != nil {
		s.metrics.outcome("err_write")
		return nil, fmt.Errorf("upsert discovered mappings %s/%s: %w", prod.ID, input.SoftwareVersion, err)
	}

	if s.invalidator != nil {
		if err := s.invalidator.InvalidateProduct(ctx, prod.ID, input.SoftwareVersion); err != nil {
			// 失效失败不致命：DB 已写，下次 cache TTL 到期或 Refresh 后必然修正。
			s.logger.Warn("invalidate cache after intersect failed (non-fatal)",
				zap.String("product_id", prod.ID.String()),
				zap.String("software_version", input.SoftwareVersion),
				zap.Error(err))
			s.metrics.invalidateErr()
		}
	}

	s.metrics.outcome("ok")
	s.metrics.matched.Add(float64(len(matched)))
	s.metrics.defaultsMissing.Add(float64(defaultsMissing))
	s.metrics.uploadedExtras.Add(float64(uploadedExtras))

	s.logger.Info("ParamIntersect computed",
		zap.String("product_id", prod.ID.String()),
		zap.String("software_version", input.SoftwareVersion),
		zap.String("param_model_id", prod.ParamModelID.String()),
		zap.Int("default_count", len(defaults)),
		zap.Int("uploaded_count", len(input.Entries)),
		zap.Int("matched", len(matched)),
		zap.Int("defaults_missing", defaultsMissing),
		zap.Int("uploaded_extras", uploadedExtras),
	)

	return &IntersectResult{
		ProductID:        prod.ID,
		SoftwareVersion:  input.SoftwareVersion,
		ParamModelID:     *prod.ParamModelID,
		DefaultCount:     len(defaults),
		UploadedCount:    len(input.Entries),
		Matched:          len(matched),
		DefaultsMissing:  defaultsMissing,
		UploadedExtras:   uploadedExtras,
		OverrideDataType: dtOverrideRequested,
		DurationMs:       time.Since(t0).Milliseconds(),
	}, nil
}

// ── 内部辅助 ──────────────────────────────────────────────────────────

// overrideFlags 持有 5 个属性的覆盖决策（true = 用 CPE 值，false = 用默认值）。
//
// 注意：data_type 永远 false（业务强制），即使 JSON 中写了 true。
type overrideFlags struct {
	access        bool
	dataType      bool // 永远 false
	changeApplies bool
	minValue      bool
	maxValue      bool
}

// normalizeOverride 解析 product.DeviceAttrsOverride（map[string]any，JSONB 反序列化）。
// 返回 (flags, dataTypeRequested) — dataTypeRequested 为 true 表示 JSON 中显式开启 data_type
// 覆盖，但 flags.dataType 仍强制为 false。
func normalizeOverride(m map[string]any) (overrideFlags, bool) {
	var f overrideFlags
	dtReq := false
	if m == nil {
		return f, false
	}
	if v, ok := boolFromAny(m["access"]); ok {
		f.access = v
	}
	if v, ok := boolFromAny(m["change_applies"]); ok {
		f.changeApplies = v
	}
	if v, ok := boolFromAny(m["min_value"]); ok {
		f.minValue = v
	}
	if v, ok := boolFromAny(m["max_value"]); ok {
		f.maxValue = v
	}
	if v, ok := boolFromAny(m["data_type"]); ok && v {
		dtReq = true // 业务禁止；flags.dataType 保持 false
	}
	return f, dtReq
}

func boolFromAny(v any) (bool, bool) {
	switch b := v.(type) {
	case bool:
		return b, true
	case nil:
		return false, false
	default:
		return false, false
	}
}

// buildIntersectRow 根据默认条目 dm + CPE 上传条目 cpe + override flags 生成 discovered 行。
//
// id 留零值，由 PG `gen_random_uuid()` 默认值赋值；ParamModelID 留零值——discovered 表无该列。
// SoftwareVersion 透传；is_storable / is_active 始终从默认继承。
// standard_path / private_path / entry_type 总是用默认（设计 §1.8 第 1 段）。
func buildIntersectRow(productID uuid.UUID, swVersion string, dm ParamMapping, cpe CPEEntry, of overrideFlags) ParamMapping {
	row := ParamMapping{
		StandardPath:    dm.StandardPath,
		PrivatePath:     dm.PrivatePath,
		EntryType:       dm.EntryType,
		Access:          dm.Access,
		DataType:        dm.DataType,
		ChangeApplies:   dm.ChangeApplies,
		MinValue:        dm.MinValue,
		MaxValue:        dm.MaxValue,
		IsStorable:      dm.IsStorable,
		IsActive:        true,
		IsSupported:     dm.IsSupported, // T-0103 从默认映射继承
		SoftwareVersion: ptrStr(swVersion),
	}
	// productID 通过 write repo 显式传，row 内不冗余存储，但 PG INSERT 时按 productID 写入。
	_ = productID

	if of.access && cpe.Access != "" {
		row.Access = cpe.Access
	}
	// dataType 覆盖永远跳过（业务禁止）
	if of.changeApplies && cpe.ChangeApplies != "" {
		row.ChangeApplies = cpe.ChangeApplies
	}
	if of.minValue && cpe.MinValue != nil {
		v := *cpe.MinValue
		row.MinValue = &v
	}
	if of.maxValue && cpe.MaxValue != nil {
		v := *cpe.MaxValue
		row.MaxValue = &v
	}
	return row
}

func ptrStr(s string) *string {
	v := s
	return &v
}
