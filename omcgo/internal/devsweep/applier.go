package devsweep

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/config/parammodel"
)

// ParamModelNameLookup 抽象按 paramModel ID 查名字(用于定位 XML 文件)。
//
// 生产环境由 parammodel.PgRepository 满足。
type ParamModelNameLookup interface {
	GetParamModelByID(ctx context.Context, id uuid.UUID) (*parammodel.ParamModel, error)
}

// ApplyOutcome 是 Apply 的完整产出 — 给 CLI 渲染。
type ApplyOutcome struct {
	ParamModelName       string
	XML                  *XMLApplyOutcome
	DBUpdatedDefault     int64
	DBUpdatedDiscovered  int64
	CacheKeysCleared     int
	CacheVersionAfter    int64
}

// Applier 把 Result 的 unsupported 集合落到 XML + DB + Cache。
//
// 顺序设计:XML → DB → Cache。
// 即使 Cache 清除失败,DB 已是真相,跨实例 ParamRegistry 下次 load 自然走 DB
// 重建 L2;短暂数据漂移可接受。Apply 整体非事务 — XML 与 DB 互独立,失败时
// outcome 部分填好,返 error 但 outcome 仍可用作部分汇报。
type Applier struct {
	pool        *pgxpool.Pool
	redis       redis.UniversalClient
	xml         *XMLWriter
	pmRepo      ParamModelNameLookup
	dbRepo      Repository
	logger      *zap.Logger
}

// NewApplier 构造 Applier。
// xml 为 nil → 不写 XML(仅 DB + cache);用于 CI 测试或 schema-only 部署。
func NewApplier(pool *pgxpool.Pool, rdb redis.UniversalClient, xmlWriter *XMLWriter,
	pmRepo ParamModelNameLookup, dbRepo Repository, logger *zap.Logger) *Applier {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Applier{
		pool:   pool,
		redis:  rdb,
		xml:    xmlWriter,
		pmRepo: pmRepo,
		dbRepo: dbRepo,
		logger: logger.Named("applier"),
	}
}

// Apply 接受探测 Result,把 unsupported 集合写 XML + 写 DB + 清缓存。
//
// 行为:
//   1. 解析 paramModel 名(用于定位 <name>.xml)
//   2. XML 写(若 xml writer 注入):为每条 unsupported leaf 标 supported="false"
//   3. UPDATE param_mappings.is_supported=false WHERE (param_model_id, standard_path)
//   4. UPDATE discovered_param_mappings.is_supported=false 同步(SQL join 反查)
//   5. 清 Redis L2 + bump cache_version
//
// Result.Aborted=true 或 UnsupportedPaths 空 → no-op。
func (a *Applier) Apply(ctx context.Context, result *Result) (*ApplyOutcome, error) {
	out := &ApplyOutcome{}
	if result == nil || result.Aborted {
		return out, nil
	}
	if len(result.UnsupportedPaths) == 0 {
		return out, nil
	}

	// 1. paramModel 名
	pm, err := a.pmRepo.GetParamModelByID(ctx, result.ParamModelID)
	if err != nil {
		return out, fmt.Errorf("lookup param_model %s: %w", result.ParamModelID, err)
	}
	if pm == nil {
		return out, fmt.Errorf("param_model %s not found", result.ParamModelID)
	}
	out.ParamModelName = pm.Name
	result.ParamModelName = pm.Name

	// 2. XML(可选)
	if a.xml != nil {
		xmlOut, err := a.xml.Apply(pm.Name, result.UnsupportedPaths)
		out.XML = xmlOut
		if err != nil {
			return out, fmt.Errorf("write XML %s: %w", pm.Name, err)
		}
	}

	// 3. UPDATE param_mappings
	n, err := a.dbRepo.MarkUnsupportedBatch(ctx, result.ParamModelID, result.UnsupportedPaths)
	if err != nil {
		return out, fmt.Errorf("mark param_mappings unsupported: %w", err)
	}
	out.DBUpdatedDefault = n

	// 4. UPDATE discovered_param_mappings(同步,避免下次 Reload 之前漂移)
	nd, err := markDiscoveredUnsupported(ctx, a.pool, result.ParamModelID, result.UnsupportedPaths)
	if err != nil {
		return out, fmt.Errorf("mark discovered_param_mappings: %w", err)
	}
	out.DBUpdatedDiscovered = nd

	// 5. 清缓存
	if a.redis != nil {
		cacheResult, err := parammodel.InvalidateCache(ctx, a.redis, a.logger)
		if err != nil {
			a.logger.Warn("cache invalidation failed (DB still consistent)", zap.Error(err))
		} else {
			out.CacheKeysCleared = cacheResult.KeysCleared
			out.CacheVersionAfter = cacheResult.CacheVersionAfter
		}
	}

	a.logger.Info("devsweep applied",
		zap.String("param_model_name", pm.Name),
		zap.String("param_model_id", result.ParamModelID.String()),
		zap.Int("unsupported_paths", len(result.UnsupportedPaths)),
		zap.Int("xml_lines_modified", xmlModifiedLines(out.XML)),
		zap.Int64("db_default_rows", out.DBUpdatedDefault),
		zap.Int64("db_discovered_rows", out.DBUpdatedDiscovered),
		zap.Int("cache_keys_cleared", out.CacheKeysCleared),
	)

	// Result 也带个汇总 — CLI 显示用
	result.MarkedCount = int(out.DBUpdatedDefault)
	return out, nil
}

func xmlModifiedLines(x *XMLApplyOutcome) int {
	if x == nil {
		return 0
	}
	return x.LinesModified
}

// markDiscoveredUnsupported 把 discovered_param_mappings 中跟随同 paramModel 的
// 重叠 path 也设 is_supported=false。
//
// 关联条件:discovered.product_id → products.param_model_id = $1;
//          discovered.standard_path IN ($2);
//          仅当当前 is_supported=true 才动(防止反复 update updated_at)。
func markDiscoveredUnsupported(ctx context.Context, pool *pgxpool.Pool, paramModelID uuid.UUID, paths []string) (int64, error) {
	if len(paths) == 0 {
		return 0, nil
	}
	const sqlText = `
UPDATE discovered_param_mappings d
   SET is_supported = false,
       updated_at   = NOW()
  FROM products p
 WHERE d.product_id      = p.id
   AND p.param_model_id  = $1
   AND d.standard_path   = ANY($2::text[])
   AND d.is_supported    = true`
	tag, err := pool.Exec(ctx, sqlText, paramModelID, paths)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
