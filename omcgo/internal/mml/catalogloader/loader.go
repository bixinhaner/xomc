package catalogloader

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
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

// v2CatalogSuffix 标识 v2 schema 文件命名约定。
// 与旧 cmcc-tdlte-v2.3.json (v1) 区分，避免 glob 时双方互相误读。
const v2CatalogSuffix = ".v2.json"

// Loader 实现 dictloader.Loader 接口；在启动期由 Registry 编排调用。
//
// 行为契约（见方案 §6.7.6）：
//   - 启动期 LoadOnce 失败 → app 启动失败（fail-fast）
//   - 热重载 Reload 失败 → 事务回滚，DB 保持原状
//   - 删除策略：差集软删 deprecated_at，仅 source='standard' 范围
//
// v2 调度（P2.c 引入，2026-05-21）：
//   - 默认 useV2Schema=false：仅扫 *.json（非 .v2.json），调 v1 upsertCatalog；
//     行为与 P0 之前一致，新增的 .v2.json 文件被跳过，不影响任何已有 DB 行。
//   - 显式 WithV2Schema(true) 后：仅扫 *.v2.json，调 v2 upsertCatalogV2，写
//     18 chapter group + ~190 op leaf + link_health；旧 v1 standard 行不被
//     触动（cutover 由后续 P3 单独清理）。
//   - 翻转默认值的时机：P4 frontend 切完命令树渲染、BuildTree 兼容 chapter 顶层
//     group 后；当前阶段 dictload.go 仍调 NewLoader 不带 Option，flag 保持 false。
type Loader struct {
	db        *pgxpool.Pool
	directory string
	fileGlob  string
	logger    *zap.Logger

	// useV2Schema 控制 Loader 是走 v1 还是 v2 路径。默认 false（v1）。
	useV2Schema bool

	mu       sync.RWMutex
	loadInfo map[string]LoadInfo // key = specVersion
}

// Option 是 NewLoader 的功能选项，供 dictload provider 在构造时按需开启 v2。
type Option func(*Loader)

// WithV2Schema 设置 Loader 是否启用 v2 catalog 调度（默认 false）。
//
// 启用后 Loader.run() 只处理 *.v2.json，并调用 upsertCatalogV2 写 v2 schema 行。
// 关闭（默认）时跳过 *.v2.json 文件，仅处理 *.json，行为同 P0 前。
func WithV2Schema(enabled bool) Option {
	return func(l *Loader) { l.useV2Schema = enabled }
}

// NewLoader constructs a Loader. directory is the absolute path scanned
// (typically derived from DictLoaderConfig.XMLBaseDir / DefaultDirectory).
//
// opts 中可传 WithV2Schema(true) 翻到 v2 路径；缺省 v1。
func NewLoader(db *pgxpool.Pool, directory string, logger *zap.Logger, opts ...Option) *Loader {
	if logger == nil {
		logger = zap.NewNop()
	}
	l := &Loader{
		db:        db,
		directory: directory,
		fileGlob:  "*.json",
		logger:    logger,
		loadInfo:  make(map[string]LoadInfo),
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
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

// isV2CatalogFile 标识 ".v2.json" 后缀文件。
//
// glob "*.json" 会同时匹配 .json 与 .v2.json；本函数用来在 run() 里按 useV2Schema
// 二选一分流：useV2Schema=true 时只处理 .v2.json，false 时跳过 .v2.json。
func isV2CatalogFile(path string) bool {
	return strings.HasSuffix(path, v2CatalogSuffix)
}

// run scans, parses and upserts all JSON catalogs under l.directory.
func (l *Loader) run(ctx context.Context, mode string) (dictloader.Report, error) {
	r := dictloader.NewReport(LoaderName)
	defer r.Finish()

	files, err := filepath.Glob(filepath.Join(l.directory, l.fileGlob))
	if err != nil {
		return r, fmt.Errorf("catalogloader: glob %s: %w", l.directory, err)
	}

	// 按 useV2Schema 过滤：当前默认 v1 模式跳过 .v2.json 文件，
	// v2 模式只处理 .v2.json。stable order so logs / diff are deterministic。
	sort.Strings(files)
	filtered := make([]string, 0, len(files))
	for _, f := range files {
		if l.useV2Schema {
			if !isV2CatalogFile(f) {
				continue
			}
		} else {
			if isV2CatalogFile(f) {
				continue
			}
		}
		filtered = append(filtered, f)
	}
	r.FilesScanned = len(filtered)

	for _, f := range filtered {
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

		var info LoadInfo
		if l.useV2Schema {
			s, err := l.upsertCatalogV2(ctx, cat)
			if err != nil {
				r.AddError(f, "persist", err)
				return r, fmt.Errorf("catalogloader: upsert v2 %s: %w", f, err)
			}
			r.FilesLoaded++
			r.RowsAffected += s.ChapterGroupCount + s.CommandLeafCount
			info = LoadInfo{
				SpecVersion:        cat.SpecVersion,
				Carrier:            cat.Carrier,
				Tech:               cat.Tech,
				SourceDocSha256:    cat.SourceDocSha256,
				GeneratedAt:        cat.GeneratedAt,
				LoadedAt:           time.Now(),
				GroupCount:         s.ChapterGroupCount,
				CommandCount:       s.CommandLeafCount,
				LinkHealthFailures: s.LinkHealthFailures,
				LinkHealthResolved: s.LinkHealthResolved,
			}
			l.logger.Info("mml-catalog v2 loaded",
				zap.String("mode", mode),
				zap.String("file", f),
				zap.String("spec_version", cat.SpecVersion),
				zap.Int("chapters", s.ChapterGroupCount),
				zap.Int("commands", s.CommandLeafCount),
				zap.Int("link_failures", s.LinkHealthFailures),
				zap.Int("link_resolved", s.LinkHealthResolved),
			)
		} else {
			summary, err := l.upsertCatalog(ctx, cat)
			if err != nil {
				r.AddError(f, "persist", err)
				return r, fmt.Errorf("catalogloader: upsert %s: %w", f, err)
			}
			r.FilesLoaded++
			r.RowsAffected += summary.RowsAffected
			info = LoadInfo{
				SpecVersion:         cat.SpecVersion,
				Carrier:             cat.Carrier,
				Tech:                cat.Tech,
				SourceDocSha256:     cat.SourceDocSha256,
				GeneratedAt:         cat.GeneratedAt,
				LoadedAt:            time.Now(),
				GroupCount:          summary.GroupCount,
				CommandCount:        summary.CommandCount,
				SubFieldCount:       summary.SubFieldCount,
				DeprecatedCount:     summary.DeprecatedCount,
				SkippedCommandCount: summary.SkippedCommandCount,
			}
			l.logger.Info("mml-catalog loaded",
				zap.String("mode", mode),
				zap.String("file", f),
				zap.String("spec_version", cat.SpecVersion),
				zap.Int("groups", summary.GroupCount),
				zap.Int("commands", summary.CommandCount),
				zap.Int("sub_fields", summary.SubFieldCount),
				zap.Int("deprecated", summary.DeprecatedCount),
				zap.Int("skipped_commands", summary.SkippedCommandCount),
			)
		}

		l.mu.Lock()
		l.loadInfo[cat.SpecVersion] = info
		l.mu.Unlock()
	}

	return r, nil
}
