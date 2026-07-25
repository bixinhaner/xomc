package dashboard

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/omcgo/omcgo/internal/pm/indicator"
)

// kpi_layout.go —— Dashboard 首页 KPI 折线图区全局布局存储（issue #213 S1）。
//
// 与每用户 dashboard_widgets 不同：本表全局单套、按制式（lte/nr/gsm）各一行，
// 管理员配一次所有用户共享。读接口所有登录用户可读，存接口仅管理员（super_admin）。
// 取数与画图复用 Phase1 指标库接口 + 现成批量时序接口，本文件只管布局的读/存。

// 合法制式集合（与迁移 dashboard_kpi_layouts_tech_check 约束、前端 TechnologyType 对齐）。
const (
	techLTE = "lte"
	techNR  = "nr"
	techGSM = "gsm"
)

// ErrInvalidTech 表示传入了非 lte/nr/gsm 的制式。
var ErrInvalidTech = errors.New("invalid technology (must be lte / nr / gsm)")

// ErrDisabledKPILayoutMetric 表示首页 KPI 布局引用了当前未启用的指标。
var ErrDisabledKPILayoutMetric = fmt.Errorf("%w: dashboard KPI layout contains disabled indicators", commonerrors.ErrInvalidInput)

// isValidTech 校验制式取值。
func isValidTech(tech string) bool {
	return tech == techLTE || tech == techNR || tech == techGSM
}

func deviceTypeForTech(tech string) (indicator.DeviceType, error) {
	switch tech {
	case techLTE:
		return indicator.DeviceTypeENB, nil
	case techNR:
		return indicator.DeviceTypeGNB, nil
	case techGSM:
		return indicator.DeviceTypeGSM, nil
	default:
		return "", ErrInvalidTech
	}
}

func techForDeviceType(dt indicator.DeviceType) (string, error) {
	switch dt {
	case indicator.DeviceTypeENB:
		return techLTE, nil
	case indicator.DeviceTypeGNB:
		return techNR, nil
	case indicator.DeviceTypeGSM:
		return techGSM, nil
	default:
		return "", fmt.Errorf("unsupported dashboard KPI layout device type: %q", dt)
	}
}

// KPILayout 是单个制式的全局首页布局（dashboard_kpi_layouts 一行）。
//
// Layout 为布局体 JSONB（panels 数组，每图含 title / metrics / x,y / w,h / chartType），
// 不在后端解析其内部结构，原样透传给前端渲染（后端只管按制式存取整块）。
type KPILayout struct {
	// Tech 是制式主键：lte / nr / gsm。
	Tech string `json:"tech"`
	// Layout 是布局体 JSONB（panels 数组），原样透传。
	Layout json.RawMessage `json:"layout"`
	// UpdatedAt 是最近保存时间（timestamptz；pgx 二进制协议要求扫进 time.Time，不能用 string）。
	UpdatedAt time.Time `json:"updated_at"`
	// UpdatedBy 是最近保存的管理员用户 ID（seed 灌入的初始行为空）。
	UpdatedBy *uuid.UUID `json:"updated_by,omitempty"`
}

// KPILayoutRepository 抽象全局布局的读写（接口优先，便于测试 mock）。
type KPILayoutRepository interface {
	// GetByTech 读单个制式的布局；无行时返回 (nil, nil)（由 service 决定回退）。
	GetByTech(ctx context.Context, tech string) (*KPILayout, error)
	// Upsert 写入单个制式的布局（最后写入生效），返回写入后的行。
	Upsert(ctx context.Context, tech string, layout json.RawMessage, updatedBy uuid.UUID) (*KPILayout, error)
}

// layoutQuerier 是本仓库所需的最小读写能力（*pgxpool.Pool 满足），
// 抽成接口便于测试用 fake 替换，不强依赖具体连接池实现。
type layoutQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// pgKPILayoutRepository 是 KPILayoutRepository 的 PostgreSQL 实现（Squirrel + pgx）。
type pgKPILayoutRepository struct {
	pool layoutQuerier
}

// NewKPILayoutRepository 构造主库实现。pool 走主库（dashboard_kpi_layouts 在主库）。
func NewKPILayoutRepository(pool layoutQuerier) KPILayoutRepository {
	return &pgKPILayoutRepository{pool: pool}
}

type KPILayoutReferenceChecker struct {
	repo KPILayoutRepository
}

func NewKPILayoutReferenceChecker(pool layoutQuerier) *KPILayoutReferenceChecker {
	if pool == nil {
		return nil
	}
	return &KPILayoutReferenceChecker{repo: NewKPILayoutRepository(pool)}
}

func NewKPILayoutReferenceCheckerFromRepository(repo KPILayoutRepository) *KPILayoutReferenceChecker {
	if repo == nil {
		return nil
	}
	return &KPILayoutReferenceChecker{repo: repo}
}

func (c *KPILayoutReferenceChecker) ReferencedIndicators(ctx context.Context, dt indicator.DeviceType, indicatorIDs []string) ([]string, error) {
	if c == nil || c.repo == nil || len(indicatorIDs) == 0 {
		return nil, nil
	}
	tech, err := techForDeviceType(dt)
	if err != nil {
		return nil, err
	}
	layout, err := c.repo.GetByTech(ctx, tech)
	if err != nil {
		return nil, fmt.Errorf("query dashboard KPI layout references: %w", err)
	}
	if layout == nil {
		layout = defaultKPILayout(tech)
	}
	metrics, err := extractKPILayoutMetrics(layout.Layout)
	if err != nil {
		return nil, err
	}
	targets := make(map[string]struct{}, len(indicatorIDs))
	for _, id := range indicatorIDs {
		if id != "" {
			targets[id] = struct{}{}
		}
	}
	referenced := map[string]struct{}{}
	for _, metric := range metrics {
		if _, ok := targets[metric]; ok {
			referenced[metric] = struct{}{}
		}
	}
	out := make([]string, 0, len(referenced))
	for id := range referenced {
		out = append(out, id)
	}
	sort.Strings(out)
	return out, nil
}

// GetByTech 读单制式布局；无行返回 (nil, nil)。
func (r *pgKPILayoutRepository) GetByTech(ctx context.Context, tech string) (*KPILayout, error) {
	query, args, err := storage.Psql.
		Select("tech", "layout", "updated_at", "updated_by").
		From("dashboard_kpi_layouts").
		Where(sq.Eq{"tech": tech}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build kpi layout query: %w", err)
	}

	var l KPILayout
	err = r.pool.QueryRow(ctx, query, args...).Scan(&l.Tech, &l.Layout, &l.UpdatedAt, &l.UpdatedBy)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query kpi layout: %w", err)
	}
	return &l, nil
}

// Upsert 写单制式布局（ON CONFLICT (tech) DO UPDATE，最后写入生效）。
func (r *pgKPILayoutRepository) Upsert(ctx context.Context, tech string, layout json.RawMessage, updatedBy uuid.UUID) (*KPILayout, error) {
	query, args, err := storage.Psql.
		Insert("dashboard_kpi_layouts").
		Columns("tech", "layout", "updated_by", "updated_at").
		Values(tech, layout, updatedBy, sq.Expr("now()")).
		Suffix("ON CONFLICT (tech) DO UPDATE SET layout = EXCLUDED.layout, updated_by = EXCLUDED.updated_by, updated_at = now() RETURNING tech, layout, updated_at, updated_by").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build kpi layout upsert: %w", err)
	}

	var l KPILayout
	err = r.pool.QueryRow(ctx, query, args...).Scan(&l.Tech, &l.Layout, &l.UpdatedAt, &l.UpdatedBy)
	if err != nil {
		return nil, fmt.Errorf("upsert kpi layout: %w", err)
	}
	return &l, nil
}

// GetKPILayout 读单制式全局布局。
//
// 无配置（DB 无行）时回退内置默认布局（defaultKPILayout，= seed 灌入的同一套），
// 保证读接口永不返回空（首页永不空白的后端底座）。
func (s *Service) GetKPILayout(ctx context.Context, tech string) (*KPILayout, error) {
	if !isValidTech(tech) {
		return nil, ErrInvalidTech
	}
	if s.layoutRepo == nil {
		return defaultKPILayout(tech), nil
	}
	l, err := s.layoutRepo.GetByTech(ctx, tech)
	if err != nil {
		return nil, err
	}
	if l == nil {
		// 无配置 → 回退内置默认（与 seed 等价），首页永不空白。
		return defaultKPILayout(tech), nil
	}
	return l, nil
}

// SaveKPILayout 存单制式全局布局（最后写入生效，不做版本锁）。
//
// 仅在 handler 完成管理员身份校验后调用（service 不再重复鉴权，鉴权属 handler 职责）。
func (s *Service) SaveKPILayout(ctx context.Context, tech string, layout json.RawMessage, updatedBy uuid.UUID) (*KPILayout, error) {
	if !isValidTech(tech) {
		return nil, ErrInvalidTech
	}
	if s.layoutRepo == nil {
		return nil, fmt.Errorf("kpi layout repository not configured")
	}
	if len(layout) == 0 {
		return nil, fmt.Errorf("layout is required")
	}
	if !json.Valid(layout) {
		return nil, fmt.Errorf("layout is not valid JSON")
	}
	if err := s.validateKPILayoutEnabledMetrics(ctx, tech, layout); err != nil {
		return nil, err
	}
	return s.layoutRepo.Upsert(ctx, tech, layout, updatedBy)
}

type kpiLayoutPayload struct {
	Panels []struct {
		Metrics []string `json:"metrics"`
	} `json:"panels"`
}

func (s *Service) validateKPILayoutEnabledMetrics(ctx context.Context, tech string, layout json.RawMessage) error {
	if s.enabledIndicatorRepo == nil {
		return nil
	}
	dt, err := deviceTypeForTech(tech)
	if err != nil {
		return err
	}
	metrics, err := extractKPILayoutMetrics(layout)
	if err != nil {
		return err
	}
	if len(metrics) == 0 {
		return nil
	}
	enabledIDs, err := s.enabledIndicatorRepo.List(ctx, dt, "default")
	if err != nil {
		return fmt.Errorf("list enabled dashboard KPI indicators: %w", err)
	}
	enabled := make(map[string]struct{}, len(enabledIDs))
	for _, id := range enabledIDs {
		enabled[id] = struct{}{}
	}
	var disabled []string
	for _, id := range metrics {
		if _, ok := enabled[id]; !ok {
			disabled = append(disabled, id)
		}
	}
	if len(disabled) > 0 {
		return fmt.Errorf("%w: %v", ErrDisabledKPILayoutMetric, disabled)
	}
	return nil
}

func extractKPILayoutMetrics(layout json.RawMessage) ([]string, error) {
	var payload kpiLayoutPayload
	if err := json.Unmarshal(layout, &payload); err != nil {
		return nil, fmt.Errorf("parse dashboard KPI layout: %w", err)
	}
	seen := map[string]struct{}{}
	out := make([]string, 0)
	for _, panel := range payload.Panels {
		for _, metric := range panel.Metrics {
			if metric == "" {
				continue
			}
			if _, ok := seen[metric]; ok {
				continue
			}
			seen[metric] = struct{}{}
			out = append(out, metric)
		}
	}
	return out, nil
}
