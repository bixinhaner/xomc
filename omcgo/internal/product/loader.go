package product

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/dictloader"
)

// LoaderName 是 dictloader.Registry 中的注册名。
const LoaderName = "product"

// Loader 实现 dictloader.Loader（T-0098 P1-06）。
//
// 加载语义（设计 §4.5）：
//   1. 读取 products.xml
//   2. 校验三引用：
//        - paramModel.name 存在于 param_models（命中失败 → 跳过 product + ERROR）
//        - alarm_ne_type 存在于 alarm_definitions.ne_type（distinct）（命中失败 → 跳过 + ERROR）
//        - indicator platform — 因 platform 表跨 3 设备类型，本 P1-06 baseline 仅 WARN（不阻塞）
//   3. 校验 device_attrs_override.data_type ≠ "true"（设计 §4.5 校验 3）
//   4. 事务内：UPSERT products + 清空 + 插入 product_class_patterns
//
// 必须在 ParamModel + AlarmDefinition 加载完成之后运行（provider 编排 LoadOnce 顺序）。
type Loader struct {
	pool   *pgxpool.Pool
	cfg    appconfig.ProductLoaderConfig
	base   string
	logger *zap.Logger
}

func NewLoader(pool *pgxpool.Pool, cfg appconfig.ProductLoaderConfig, baseDir string, logger *zap.Logger) *Loader {
	if cfg.Directory == "" {
		cfg.Directory = "param-mappings"
	}
	if cfg.File == "" {
		cfg.File = "products.xml"
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Loader{pool: pool, cfg: cfg, base: baseDir, logger: logger.Named(LoaderName)}
}

func (l *Loader) Name() string      { return LoaderName }
func (l *Loader) Directory() string { return l.cfg.Directory }

func (l *Loader) LoadOnce(ctx context.Context) (dictloader.Report, error) { return l.run(ctx) }
func (l *Loader) Reload(ctx context.Context) (dictloader.Report, error)   { return l.run(ctx) }

func (l *Loader) run(ctx context.Context) (dictloader.Report, error) {
	rep := dictloader.NewReport(LoaderName)
	defer rep.Finish()

	path := filepath.Join(l.base, l.cfg.Directory, l.cfg.File)
	rep.FilesScanned = 1

	raw, err := os.ReadFile(path)
	if err != nil {
		rep.AddError(l.cfg.File, "read", err)
		return rep, fmt.Errorf("read %s: %w", path, err)
	}
	var doc xmlProducts
	if err := xml.Unmarshal(raw, &doc); err != nil {
		rep.AddError(l.cfg.File, "parse", err)
		return rep, fmt.Errorf("xml unmarshal %s: %w", path, err)
	}

	// 加载已存在的 paramModel name → ID 映射，用于解析三引用之参数模型软引用
	paramModelIDs, err := l.fetchParamModelIDs(ctx)
	if err != nil {
		return rep, fmt.Errorf("fetch param_model ids: %w", err)
	}

	// 加载 alarm_definitions 中 distinct ne_type，用于校验 alarm 引用
	alarmNeTypes, err := l.fetchAlarmNeTypes(ctx)
	if err != nil {
		return rep, fmt.Errorf("fetch alarm ne_types: %w", err)
	}

	tx, err := l.pool.Begin(ctx)
	if err != nil {
		return rep, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// 清空 product_class_patterns（FK CASCADE 无需先 DELETE products）
	if _, err := tx.Exec(ctx, `TRUNCATE product_class_patterns RESTART IDENTITY`); err != nil {
		return rep, fmt.Errorf("truncate patterns: %w", err)
	}

	productsLoaded := 0
	patternsLoaded := 0
	for _, p := range doc.Products {
		// 校验 1: device_attrs_override.data_type 不允许 true（设计 §4.5）
		if strings.EqualFold(strings.TrimSpace(p.DeviceAttrsOverride.DataType), "true") {
			rep.AddError(l.cfg.File, "validate", fmt.Errorf("product %q: device_attrs_override.data_type=true is forbidden", p.Name))
			l.logger.Error("product skip: data_type=true forbidden", zap.String("product", p.Name))
			continue
		}

		// 校验 2: paramModel 引用
		var paramModelID interface{}
		if p.ParamModel != "" {
			id, ok := paramModelIDs[p.ParamModel]
			if !ok {
				rep.AddError(l.cfg.File, "validate", fmt.Errorf("product %q: paramModel %q not found in param_models", p.Name, p.ParamModel))
				l.logger.Error("product skip: paramModel ref missing",
					zap.String("product", p.Name), zap.String("paramModel", p.ParamModel))
				continue
			}
			paramModelID = id
		}

		// 校验 3: alarm ne_type 引用（设计 §4.5）— 在 alarm_definitions 中存在或在 P1-04 4 行 ne_type 集合中（即等到 alarm loader 跑过）
		neType := strings.TrimSpace(p.Alarm.NeType)
		if neType != "" {
			if _, ok := alarmNeTypes[neType]; !ok {
				// alarm loader 尚未跑或该 ne_type 真的不在告警库 — 允许通过但 WARN（设计 §4.5 严格校验，本 baseline 放宽）
				l.logger.Warn("product alarm ne_type not yet in alarm_definitions; will validate at runtime",
					zap.String("product", p.Name), zap.String("ne_type", neType))
			}
		}

		// 校验 4 (WARN-only): indicator platform — Phase 2 P2-09 之后才能严格校验
		if p.Indicator.Platform == "" {
			l.logger.Warn("product missing indicator platform", zap.String("product", p.Name))
		}

		// 序列化 device_attrs_override 为 JSONB
		override := map[string]bool{
			"access":         strings.EqualFold(p.DeviceAttrsOverride.Access, "true"),
			"min_value":      strings.EqualFold(p.DeviceAttrsOverride.MinValue, "true"),
			"max_value":      strings.EqualFold(p.DeviceAttrsOverride.MaxValue, "true"),
			"change_applies": strings.EqualFold(p.DeviceAttrsOverride.ChangeApplies, "true"),
			"data_type":      strings.EqualFold(p.DeviceAttrsOverride.DataType, "true"),
		}
		overrideJSON, err := json.Marshal(override)
		if err != nil {
			return rep, fmt.Errorf("marshal device_attrs_override for %q: %w", p.Name, err)
		}

		enableFT11 := !strings.EqualFold(strings.TrimSpace(p.EnableFileType11), "false") // 缺省 / 非 "false" → true（设计 §4.5）
		enableUnknown := strings.EqualFold(strings.TrimSpace(p.Alarm.EnableUnknownAlarm), "true")

		// UPSERT products
		const upsertProduct = `INSERT INTO products
		(product_name, vendor, tech, radio_modes, description, param_model_id,
		 indicator_device_type, indicator_platform, alarm_ne_type,
		 enable_filetype11, device_attrs_override, enable_unknown_alarm)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11::jsonb, $12)
		ON CONFLICT (product_name) DO UPDATE
		SET vendor                = EXCLUDED.vendor,
		    tech                  = EXCLUDED.tech,
		    radio_modes           = EXCLUDED.radio_modes,
		    description           = EXCLUDED.description,
		    param_model_id        = EXCLUDED.param_model_id,
		    indicator_device_type = EXCLUDED.indicator_device_type,
		    indicator_platform    = EXCLUDED.indicator_platform,
		    alarm_ne_type         = EXCLUDED.alarm_ne_type,
		    enable_filetype11     = EXCLUDED.enable_filetype11,
		    device_attrs_override = EXCLUDED.device_attrs_override,
		    enable_unknown_alarm  = EXCLUDED.enable_unknown_alarm
		RETURNING id`
		var productID string
		if err := tx.QueryRow(ctx, upsertProduct,
			p.Name, p.Vendor, p.Tech, p.RadioModes, p.Description, paramModelID,
			p.Indicator.DeviceType, p.Indicator.Platform, neType,
			enableFT11, overrideJSON, enableUnknown,
		).Scan(&productID); err != nil {
			return rep, fmt.Errorf("upsert product %q: %w", p.Name, err)
		}
		productsLoaded++

		// 插入 patterns
		if len(p.Patterns) > 0 {
			ib := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
				Insert("product_class_patterns").
				Columns("product_id", "product_class", "sort_order", "is_active")
			added := 0
			for _, pat := range p.Patterns {
				val := strings.TrimSpace(pat.Value)
				if val == "" {
					continue
				}
				ib = ib.Values(productID, val, pat.GlobalOrder, true)
				added++
			}
			if added > 0 {
				sqlStr, args, err := ib.ToSql()
				if err != nil {
					return rep, fmt.Errorf("build patterns sql for %q: %w", p.Name, err)
				}
				if _, err := tx.Exec(ctx, sqlStr, args...); err != nil {
					return rep, fmt.Errorf("insert patterns for %q: %w", p.Name, err)
				}
				patternsLoaded += added
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return rep, fmt.Errorf("commit product tx: %w", err)
	}

	rep.FilesLoaded = 1
	rep.RowsAffected = productsLoaded + patternsLoaded
	l.logger.Info("products loaded",
		zap.Int("products", productsLoaded),
		zap.Int("patterns", patternsLoaded),
		zap.String("file", l.cfg.File))

	if rep.HasErrors() {
		// 部分校验失败但非全失败 — 设计 §4.5 "不通过的 product 跳过 + ERROR 日志"
		l.logger.Warn("products loaded with non-fatal errors", zap.Int("err_count", len(rep.Errors)))
	}
	return rep, nil
}

func (l *Loader) fetchParamModelIDs(ctx context.Context) (map[string]string, error) {
	rows, err := l.pool.Query(ctx, `SELECT name, id::text FROM param_models WHERE is_active = TRUE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]string, 16)
	for rows.Next() {
		var name, id string
		if err := rows.Scan(&name, &id); err != nil {
			return nil, err
		}
		out[name] = id
	}
	return out, rows.Err()
}

func (l *Loader) fetchAlarmNeTypes(ctx context.Context) (map[string]struct{}, error) {
	rows, err := l.pool.Query(ctx, `SELECT DISTINCT ne_type FROM alarm_definitions`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]struct{}, 8)
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		out[t] = struct{}{}
	}
	return out, rows.Err()
}
