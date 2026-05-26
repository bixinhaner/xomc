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
// ============================================================
// data/mml-catalog/cmcc-tdlte-v2.3.json 的角色与边界（CRITICAL）
// ============================================================
// 该 JSON **只用于提取"命令分组结构"**：chapter / group_code /
// command_zh_name / has_instance / command_code / operation_type 等元数据。
//
// 它 **不是 path 的真值源**。
//   · 命令真正引用哪些 path：来自 `mml_command_sub_fields` 表，由该表的
//     `standard_path_id` FK → `standard_params`（系统级 standardPath 字典）。
//   · catalog JSON 里的 `paths[]` 与 `target_paths[]` 只是 catalog 生成时
//     冗余写出的"该 group 应包含哪些标准 path"建议；运行时不依据它做 GPV。
//   · 因此 **改 catalog.target_paths 不会改 sub_field 表**（catalogloader
//     UPSERT 写 mml_commands.target_paths jsonb 列，但 executor 拼 GPV 时
//     是从 sub_field 表 JOIN standard_params 拿 path，与 target_paths 列无关）。
//
// 历史教训：曾多次因把 catalog target_paths 当成 sub_field 源导致误判。
// 任何"删除 / 增加命令字段集"的操作，都应改 `mml_command_sub_fields` 行
// （或新增 seed migration），不应改 catalog JSON。
// ============================================================
//
// 行为契约（见方案 §6.7.6）：
//   - 启动期 LoadOnce 失败 → app 启动失败（fail-fast）
//   - 热重载 Reload 失败 → 事务回滚，DB 保持原状
//   - 删除策略：差集软删 deprecated_at，仅 source='standard' 范围
//
// 仅 v2 catalog（spec §R-1 一级分组 = 18 SA-SR 章节，groupCode = "chapter:<SA-SR>"）。
// 历史 v1 path（object 维度 groups[]、TargetPaths、SubFields）已下线。
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
	// stable order so logs / diff are deterministic
	sort.Strings(files)
	r.FilesScanned = len(files)

	for _, f := range files {
		select {
		case <-ctx.Done():
			return r, ctx.Err()
		default:
		}

		cat, err := ParseFile(f)
		if err != nil {
			r.AddError(f, "parse", err)
			return r, fmt.Errorf("catalogloader: parse %s: %w", f, err)
		}

		s, err := l.upsertCatalogV2(ctx, cat)
		if err != nil {
			r.AddError(f, "persist", err)
			return r, fmt.Errorf("catalogloader: upsert %s: %w", f, err)
		}
		r.FilesLoaded++
		r.RowsAffected += s.ChapterGroupCount + s.CommandLeafCount

		info := LoadInfo{
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
		l.logger.Info("mml-catalog loaded",
			zap.String("mode", mode),
			zap.String("file", f),
			zap.String("spec_version", cat.SpecVersion),
			zap.Int("chapters", s.ChapterGroupCount),
			zap.Int("commands", s.CommandLeafCount),
			zap.Int("link_failures", s.LinkHealthFailures),
			zap.Int("link_resolved", s.LinkHealthResolved),
		)

		l.mu.Lock()
		l.loadInfo[cat.SpecVersion] = info
		l.mu.Unlock()
	}

	return r, nil
}
