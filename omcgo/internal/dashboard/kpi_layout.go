package dashboard

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/omcgo/omcgo/internal/core/storage"
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

// isValidTech 校验制式取值。
func isValidTech(tech string) bool {
	return tech == techLTE || tech == techNR || tech == techGSM
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
	return s.layoutRepo.Upsert(ctx, tech, layout, updatedBy)
}
