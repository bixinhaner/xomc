package mmlstandardloader

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/dictloader"
)

// LoaderName 用于 dictloader.Registry 注册名 + 日志域。
const LoaderName = "mml-standard"

// VersionCode 唯一的 STANDARD 版本号；所有 path/group 都绑定到这条。
const VersionCode = "STANDARD"

// XML 文件路径默认值（相对 dictload 根目录）。
const (
	DefaultDirectory         = "param-mappings"
	DefaultStandardModelFile = "standard-model.xml"
)

// Loader 实现 dictloader.Loader。
//
// 加载语义：
//
//	parse standard-model.xml
//	  → trie 切分 group（≤50 阈值）
//	  → 生成 LST/MOD/ADD/RMV 命令
//	  → 单事务 UPSERT mml_param_versions / mml_param_groups / mml_params / mml_commands /
//	    mml_group_param_rel
//
// 二次跑：UPSERT 幂等，无副作用。
type Loader struct {
	pool    *pgxpool.Pool
	baseDir string
	dirName string
	file    string
	logger  *zap.Logger
	metrics *Metrics
}

// NewLoader 构造 Loader。
// baseDir = DictLoaderConfig.XMLBaseDir；dir/file 为空使用默认值。
// reg 可为 nil（metric 注册自动跳过）。
func NewLoader(pool *pgxpool.Pool, baseDir, dir, file string, logger *zap.Logger, reg prometheus.Registerer) *Loader {
	if dir == "" {
		dir = DefaultDirectory
	}
	if file == "" {
		file = DefaultStandardModelFile
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Loader{
		pool:    pool,
		baseDir: baseDir,
		dirName: dir,
		file:    file,
		logger:  logger.Named(LoaderName),
		metrics: NewMetrics(reg),
	}
}

// Name implements dictloader.Loader.
func (l *Loader) Name() string { return LoaderName }

// Directory implements dictloader.Loader.
func (l *Loader) Directory() string { return l.dirName }

// LoadOnce 启动期入口。
func (l *Loader) LoadOnce(ctx context.Context) (dictloader.Report, error) {
	return l.run(ctx)
}

// Reload 与 LoadOnce 等价（UPSERT 幂等）。
func (l *Loader) Reload(ctx context.Context) (dictloader.Report, error) {
	return l.run(ctx)
}

// computeFileHash 算 XML 文件 sha256 hex；失败返空串让上游退化为全量 UPSERT。
func computeFileHash(path string) (string, error) {
	bytes, err := os.ReadFile(path) //nolint:gosec
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(bytes)
	return hex.EncodeToString(sum[:]), nil
}

// readStoredHash 取 STANDARD version 行已存的 content_hash；row 不存在 / 列为 NULL
// 都返空串（首次跑路径）。
func readStoredHash(ctx context.Context, pool *pgxpool.Pool) (string, error) {
	var stored *string
	err := pool.QueryRow(ctx,
		`SELECT content_hash FROM mml_param_versions WHERE version_code = $1`,
		VersionCode,
	).Scan(&stored)
	if err == pgx.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if stored == nil {
		return "", nil
	}
	return *stored, nil
}

func (l *Loader) run(ctx context.Context) (dictloader.Report, error) {
	rep := dictloader.NewReport(LoaderName)
	defer rep.Finish()

	start := time.Now()
	xmlPath := filepath.Join(l.baseDir, l.dirName, l.file)

	// Sprint B / F-3 增量重载：先算文件 sha256，对比上次入库值；相同直接 skip
	// 全量 UPSERT。Hash 算 + DB 一次 SELECT 通常 <10ms vs 全量 ~500ms。
	currentHash, hashErr := computeFileHash(xmlPath)
	if hashErr != nil {
		// 读不到文件就让下面 ParseStandardXMLFile 走原路径报错；不在这里返
		l.logger.Warn("mml standard loader: hash precompute failed (fallback to full UPSERT)",
			zap.String("xml_path", xmlPath), zap.Error(hashErr))
	} else {
		storedHash, storedErr := readStoredHash(ctx, l.pool)
		if storedErr != nil {
			l.logger.Warn("mml standard loader: read stored hash failed (fallback to full UPSERT)",
				zap.Error(storedErr))
		} else if storedHash != "" && storedHash == currentHash {
			elapsed := time.Since(start)
			rep.RowsAffected = 0
			rep.FilesLoaded = 1
			l.metrics.recordSuccess(elapsed, 0, 0, 0)
			l.logger.Info("mml standard loader: skipped (content unchanged)",
				zap.String("xml_path", xmlPath),
				zap.String("hash", currentHash[:12]+"…"),
				zap.Duration("elapsed", elapsed))
			return rep, nil
		}
	}

	l.logger.Info("mml standard loader: phase start",
		zap.String("phase", "parse_xml"),
		zap.String("xml_path", xmlPath))

	params, _, err := ParseStandardXMLFile(xmlPath)
	if err != nil {
		l.metrics.recordFailure("parse_xml")
		l.logger.Error("mml standard loader: parse xml failed",
			zap.String("phase", "parse_xml"),
			zap.String("xml_path", xmlPath),
			zap.Error(err))
		return rep, fmt.Errorf("parse xml: %w", err)
	}
	l.logger.Info("mml standard loader: parse_xml done",
		zap.String("xml_path", xmlPath),
		zap.Int("params_parsed", len(params)))

	groups := BuildGroups(params)
	commands := GenerateCommands(groups)
	l.logger.Info("mml standard loader: build done",
		zap.Int("groups", len(groups)),
		zap.Int("commands", len(commands)))

	tx, err := l.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		l.metrics.recordFailure("begin_tx")
		return rep, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := upsertVersion(ctx, tx, currentHash); err != nil {
		l.metrics.recordFailure("upsert_version")
		l.logger.Error("mml standard loader: upsert_version failed",
			zap.String("phase", "upsert_version"), zap.Error(err))
		return rep, fmt.Errorf("upsert version: %w", err)
	}

	if err := upsertParams(ctx, tx, params); err != nil {
		l.metrics.recordFailure("upsert_params")
		l.logger.Error("mml standard loader: upsert_params failed",
			zap.String("phase", "upsert_params"),
			zap.Int("attempted_rows", len(params)),
			zap.Error(err))
		return rep, fmt.Errorf("upsert params: %w", err)
	}

	groupIDs, err := upsertGroups(ctx, tx, groups)
	if err != nil {
		l.metrics.recordFailure("upsert_groups")
		l.logger.Error("mml standard loader: upsert_groups failed",
			zap.String("phase", "upsert_groups"),
			zap.Int("attempted_rows", len(groups)),
			zap.Error(err))
		return rep, fmt.Errorf("upsert groups: %w", err)
	}

	if err := upsertCommands(ctx, tx, commands, groupIDs); err != nil {
		l.metrics.recordFailure("upsert_commands")
		l.logger.Error("mml standard loader: upsert_commands failed",
			zap.String("phase", "upsert_commands"),
			zap.Int("attempted_rows", len(commands)),
			zap.Error(err))
		return rep, fmt.Errorf("upsert commands: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		l.metrics.recordFailure("commit")
		return rep, fmt.Errorf("commit: %w", err)
	}

	elapsed := time.Since(start)
	rep.RowsAffected = len(params) + len(groups) + len(commands)
	rep.FilesLoaded = 1

	l.metrics.recordSuccess(elapsed, len(params), len(groups), len(commands))
	l.logger.Info("mml standard loader: completed",
		zap.Duration("elapsed", elapsed),
		zap.Int("params_upserted", len(params)),
		zap.Int("groups_upserted", len(groups)),
		zap.Int("commands_generated", len(commands)),
		zap.String("xml_path", xmlPath))

	return rep, nil
}

// ============================================================
// UPSERT helpers
// ============================================================

// upsertVersion 维护 STANDARD version 行。content_hash 由 caller 提供（来自当前
// XML 文件 sha256），若为空字符串则不写入此列保留之前的值（兜底）。
func upsertVersion(ctx context.Context, tx pgx.Tx, contentHash string) error {
	var hashArg interface{}
	if contentHash == "" {
		hashArg = nil
	} else {
		hashArg = contentHash
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO mml_param_versions (id, version_code, version_name, description, source, is_active, is_deprecated, content_hash)
		VALUES (gen_random_uuid(), $1, $2, $3, 'standard', true, false, $4)
		ON CONFLICT (version_code) DO UPDATE
		SET version_name  = EXCLUDED.version_name,
		    description   = EXCLUDED.description,
		    source        = EXCLUDED.source,
		    is_active     = true,
		    is_deprecated = false,
		    content_hash  = COALESCE(EXCLUDED.content_hash, mml_param_versions.content_hash),
		    updated_at    = NOW()
	`, VersionCode, "TR-069 Standard Model", "由 standard-model.xml 派生，loader 自动维护", hashArg)
	return err
}

// upsertParams 批量 UPSERT 1988 行 mml_params。
//
// 注意：当前 mml_params 的 UNIQUE 约束是 (param_version, tr069_path)；
// 单 STANDARD version 下这等价于 tr069_path 唯一。
func upsertParams(ctx context.Context, tx pgx.Tx, params []ParamSpec) error {
	if len(params) == 0 {
		return nil
	}
	// 一条一条 UPSERT — 1988 行可接受（~3-5s）；可改为 batch 后续优化。
	for _, p := range params {
		valueType, constraint := mapValueType(p)
		writable := p.IsWritable()

		// param_code = 末段（去 {i}）
		segs := strings.Split(StripInstanceIndex(p.StandardPath), ".")
		paramCode := segs[len(segs)-1]

		_, err := tx.Exec(ctx, `
			INSERT INTO mml_params (
				id, param_code, param_name_zh, param_name_en, tr069_path,
				value_type, value_constraint, is_writable, is_leaf,
				param_version, name_i18n, explanation_i18n,
				display_order, is_active
			) VALUES (
				gen_random_uuid(), $1, $2, $2, $3,
				$4, $5, $6, true,
				$7, $8, '{}'::jsonb,
				0, true
			)
			ON CONFLICT (param_version, tr069_path) DO UPDATE
			SET param_code    = EXCLUDED.param_code,
			    value_type    = EXCLUDED.value_type,
			    value_constraint = EXCLUDED.value_constraint,
			    is_writable   = EXCLUDED.is_writable,
			    name_i18n     = EXCLUDED.name_i18n,
			    is_active     = true,
			    updated_at    = NOW()
		`, paramCode, paramCode, p.StandardPath,
			valueType, constraint, writable,
			VersionCode, nameI18nJSON(paramCode, paramCode))
		if err != nil {
			return fmt.Errorf("upsert param %s: %w", p.StandardPath, err)
		}
	}
	return nil
}

// upsertGroups 批量 UPSERT mml_param_groups，返回 path→id 映射用于后续 commands.group_id 解析。
func upsertGroups(ctx context.Context, tx pgx.Tx, groups []GroupSpec) (map[string]string, error) {
	ids := make(map[string]string, len(groups))
	for _, g := range groups {
		nameZh, nameEn := groupDisplayName(g)
		row := tx.QueryRow(ctx, `
			INSERT INTO mml_param_groups (
				id, group_code, group_name_zh, group_name_en,
				path, name_i18n,
				param_version, display_order, is_active
			) VALUES (
				gen_random_uuid(), $1, $2, $3,
				$4::ltree, $5,
				$6, 0, true
			)
			ON CONFLICT (param_version, group_code) DO UPDATE
			SET group_name_zh = EXCLUDED.group_name_zh,
			    group_name_en = EXCLUDED.group_name_en,
			    name_i18n     = EXCLUDED.name_i18n,
			    path          = EXCLUDED.path,
			    is_active     = true,
			    updated_at    = NOW()
			RETURNING id::text
		`, g.Code, nameZh, nameEn,
			pathToLtree(g.Path), nameI18nJSON(nameZh, nameEn),
			VersionCode)
		var id string
		if err := row.Scan(&id); err != nil {
			return nil, fmt.Errorf("upsert group %s: %w", g.Code, err)
		}
		ids[g.Path] = id
	}
	return ids, nil
}

// upsertCommands 批量 UPSERT mml_commands；按 command_code UNIQUE 幂等。
func upsertCommands(ctx context.Context, tx pgx.Tx, cmds []CommandSpec, groupIDs map[string]string) error {
	for _, c := range cmds {
		targetPathsJSON, err := json.Marshal(c.TargetPaths)
		if err != nil {
			return fmt.Errorf("marshal target_paths for %s: %w", c.Code, err)
		}
		nameJSON := nameI18nJSON(c.Name, c.NameEn)

		var groupID interface{}
		if id, ok := groupIDs[c.GroupPath]; ok {
			groupID = id
		} else {
			groupID = nil
		}

		var targetObject interface{}
		if c.TargetObject != "" {
			targetObject = c.TargetObject
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO mml_commands (
				id, command_name, command_code, category, description, rpc_method,
				operation_type, target_paths, target_object,
				group_id, command_name_i18n,
				help_doc
			) VALUES (
				gen_random_uuid(), $1, $2, $3, $4, $5,
				$6, $7::jsonb, $8,
				$9, $10::jsonb,
				''
			)
			ON CONFLICT (command_code) DO UPDATE
			SET command_name      = EXCLUDED.command_name,
			    category          = EXCLUDED.category,
			    rpc_method        = EXCLUDED.rpc_method,
			    operation_type    = EXCLUDED.operation_type,
			    target_paths      = EXCLUDED.target_paths,
			    target_object     = EXCLUDED.target_object,
			    group_id          = EXCLUDED.group_id,
			    command_name_i18n = EXCLUDED.command_name_i18n
		`, c.Name, c.Code, c.Category, c.Name, c.RPCMethod,
			c.OperationType, targetPathsJSON, targetObject,
			groupID, nameJSON)
		if err != nil {
			return fmt.Errorf("upsert command %s: %w", c.Code, err)
		}
	}
	return nil
}

// ============================================================
// helpers
// ============================================================

// mapValueType 把 XML 的 type 字符串翻译为 mml_params.value_type 的 CHECK 约束允许值，
// 同时构造 value_constraint JSONB。
//
// mml_params.value_type CHECK: 'string' / 'enum' / 'unsignedInt' / 'unsignedIntList' /
// 'stringList' / 'boolean' / 'uniqueInt' / 'int'
func mapValueType(p ParamSpec) (string, []byte) {
	constraint := map[string]interface{}{}
	var vt string

	switch p.Type {
	case "INT":
		vt = "int"
		if p.Min != nil {
			constraint["min"] = *p.Min
		}
		if p.Max != nil {
			constraint["max"] = *p.Max
		}
	case "U_INT":
		vt = "unsignedInt"
		if p.Min != nil {
			constraint["min"] = *p.Min
		}
		if p.Max != nil {
			constraint["max"] = *p.Max
		}
	case "BOOLEAN":
		vt = "boolean"
	case "DATE_TIME":
		// CHECK 不含 datetime，先归 string + 标 subtype
		vt = "string"
		constraint["subtype"] = "datetime"
	case "STRING", "":
		vt = "string"
		if p.MaxLen != nil {
			constraint["max_length"] = *p.MaxLen
		}
	default:
		// 未知类型 fail-safe 归 string
		vt = "string"
		constraint["raw_type"] = p.Type
	}

	b, _ := json.Marshal(constraint)
	return vt, b
}

// nameI18nJSON 构造 {"zh":"...","en":"..."} 的 JSONB 字符串。
func nameI18nJSON(zh, en string) []byte {
	m := map[string]string{"zh": zh, "en": en}
	b, _ := json.Marshal(m)
	return b
}

// pathToLtree 把 TR-069 path 转成 PostgreSQL ltree 兼容形式。
// ltree label 只允许 [A-Za-z0-9_]；TR-069 path 含 `{i}` 不合规，去掉 + 全转下划线。
//
// 输入: "Device.Services.FAPService.{i}.CellConfig.LTE"
// 输出: "Device.Services.FAPService.CellConfig.LTE"
func pathToLtree(path string) string {
	s := StripInstanceIndex(path)
	// 替换其它非法字符（极少出现，但 X_COM 这种含下划线本身合规）
	return s
}
