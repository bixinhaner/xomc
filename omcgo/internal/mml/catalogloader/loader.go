package catalogloader

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/dictloader"
)

// LoaderName 是注册到 dictloader.Registry 的唯一名称。
const LoaderName = "mml-catalog"

// DefaultDirectory 是相对 DictLoaderConfig.XMLBaseDir 的子目录。
// 与现有 4 个 dictloader 同级（param-mappings / products / indicators / alarms）。
const DefaultDirectory = "mml-catalog"

// Loader 实现 dictloader.Loader 接口；在启动期由 Registry 编排调用。
//
// 行为契约（见方案 §6.7.6）：
//   - 启动期 LoadOnce 失败 → app 启动失败（fail-fast）
//   - 热重载 Reload 失败 → 事务回滚，DB 保持原状
//   - 删除策略：差集软删 deprecated_at，仅 source='standard' 范围
type Loader struct {
	db        *pgxpool.Pool
	directory string
	fileGlob  string
	logger    *zap.Logger

	mu       sync.RWMutex
	loadInfo map[string]LoadInfo // key = specVersion
}

// NewLoader constructs a Loader. directory is the absolute path scanned
// (typically derived from DictLoaderConfig.XMLBaseDir / DefaultDirectory).
func NewLoader(db *pgxpool.Pool, directory string, logger *zap.Logger) *Loader {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Loader{
		db:        db,
		directory: directory,
		fileGlob:  "*.json",
		logger:    logger,
		loadInfo:  make(map[string]LoadInfo),
	}
}

// Name implements dictloader.Loader.
func (l *Loader) Name() string { return LoaderName }

// Directory implements dictloader.Loader.
func (l *Loader) Directory() string { return l.directory }

// LoadOnce implements dictloader.Loader. fail-fast on any error.
func (l *Loader) LoadOnce(ctx context.Context) (dictloader.Report, error) {
	return l.run(ctx, "load-once")
}

// Reload implements dictloader.Loader. fail-fast on any error.
func (l *Loader) Reload(ctx context.Context) (dictloader.Report, error) {
	return l.run(ctx, "reload")
}

// Info returns a snapshot of currently loaded catalogs (by specVersion).
// Used by admin API GET /api/v1/mml/catalog/info.
func (l *Loader) Info() []LoadInfo {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]LoadInfo, 0, len(l.loadInfo))
	for _, v := range l.loadInfo {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SpecVersion < out[j].SpecVersion })
	return out
}

// run scans, parses and upserts all JSON catalogs under l.directory.
func (l *Loader) run(ctx context.Context, mode string) (dictloader.Report, error) {
	r := dictloader.NewReport(LoaderName)
	defer r.Finish()

	files, err := filepath.Glob(filepath.Join(l.directory, l.fileGlob))
	if err != nil {
		return r, fmt.Errorf("catalogloader: glob %s: %w", l.directory, err)
	}
	r.FilesScanned = len(files)

	// stable order so logs / diff are deterministic
	sort.Strings(files)

	for _, f := range files {
		select {
		case <-ctx.Done():
			return r, ctx.Err()
		default:
		}

		cat, err := ParseFile(f)
		if err != nil {
			r.AddError(f, "parse", err)
			// fail-fast: solitary file failure aborts the load
			return r, fmt.Errorf("catalogloader: parse %s: %w", f, err)
		}

		summary, err := l.upsertCatalog(ctx, cat)
		if err != nil {
			r.AddError(f, "persist", err)
			return r, fmt.Errorf("catalogloader: upsert %s: %w", f, err)
		}
		r.FilesLoaded++
		r.RowsAffected += summary.RowsAffected

		l.mu.Lock()
		l.loadInfo[cat.SpecVersion] = LoadInfo{
			SpecVersion:     cat.SpecVersion,
			Carrier:         cat.Carrier,
			Tech:            cat.Tech,
			SourceDocSha256: cat.SourceDocSha256,
			GeneratedAt:     cat.GeneratedAt,
			LoadedAt:        time.Now(),
			GroupCount:      summary.GroupCount,
			CommandCount:    summary.CommandCount,
			SubFieldCount:   summary.SubFieldCount,
			DeprecatedCount: summary.DeprecatedCount,
		}
		l.mu.Unlock()

		l.logger.Info("mml-catalog loaded",
			zap.String("mode", mode),
			zap.String("file", f),
			zap.String("spec_version", cat.SpecVersion),
			zap.Int("groups", summary.GroupCount),
			zap.Int("commands", summary.CommandCount),
			zap.Int("sub_fields", summary.SubFieldCount),
			zap.Int("deprecated", summary.DeprecatedCount),
		)
	}

	return r, nil
}
