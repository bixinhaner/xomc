package mmlstandardloader

// seed_importer.go — 把 cmcc_tdlte_v23.json seed 一次性 UPSERT 到 4 张表：
//
//	mml_param_versions        — 版本锚点（version_code = "cmcc-tdlte-v2.3"）
//	mml_command_groups          — 18 个 chapter 一级分组（SA…SR）
//	mml_commands              — 派生后的 LST/MOD/ADD/RMV 命令
//	mml_command_sub_fields    — 命令 ↔ standard_params M:N 关联
//
// standard_params 表为只读引用：其数据由产品/参数模型管理流程预先维护，
// importer 仅按 standard_path 查询已有 ID 用于建立 sub_fields 关联，不写入新行。
//
// 单事务执行；UPSERT 幂等；二次跑无副作用。
//
// 表关系（M:N 通过 mml_command_sub_fields.standard_path_id → standard_params.id；
//        mml_params 表不参与该链路，seed 不再向其写入数据）：
//
//	standard_params(id, standard_path)   ← 只读引用（不写入）；standard_path UNIQUE
//	  ▲
//	  │  standard_path_id FK（按 path 查询所得 id）
//	mml_command_sub_fields(command_id, standard_path_id, mml_code, ...)
//	  ▼
//	mml_commands(id, command_code, group_id, ...)
//	  ▲
//	  │  group_id FK
//	mml_command_groups(id, group_code, chapter_code, param_version, ...)
//	  ▲
//	  │  param_version FK (VARCHAR)
//	mml_param_versions(version_code, ...)

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// SeedImportSource 标识 mml_commands.source / mml_command_groups.source
// 是 standard 来源（vs admin 手动新增）。
const SeedImportSource = "standard"

// ImportSeedFile 入口：读取 seed JSON 并落库。
//
// 调用方负责传入 *pgxpool.Pool + 文件绝对路径。失败返回 error，事务自动回滚。
//
// 性能预算：71 commands + 616 params + 18 groups
//
//	典型耗时 1-3s（每行单独 UPSERT；后续可改 batch）。
//	standard_params 不写入，仅一次批量 SELECT 解析 path → id 映射。
func ImportSeedFile(ctx context.Context, pool *pgxpool.Pool, seedPath string, logger *zap.Logger) error {
	if logger == nil {
		logger = zap.NewNop()
	}
	logger = logger.Named("seed-importer")

	start := time.Now()

	seed, err := ParseSeedFile(seedPath)
	if err != nil {
		return fmt.Errorf("parse seed: %w", err)
	}

	hash, hashErr := computeSeedFileHash(seedPath)
	if hashErr != nil {
		logger.Warn("seed hash precompute failed; will fall back to full upsert without hash gating",
			zap.Error(hashErr))
	}

	derived := DeriveCommandsFromSeed(seed)

	logger.Info("seed parsed",
		zap.String("version_code", seed.VersionCode()),
		zap.Int("groups", len(seed.Groups)),
		zap.Int("commands_seed", seed.CountCommands()),
		zap.Int("params", seed.CountParams()),
		zap.Int("commands_derived", len(derived)))

	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := upsertSeedVersion(ctx, tx, seed, hash); err != nil {
		return fmt.Errorf("upsert version: %w", err)
	}

	stdPathIDs, missingPaths, err := lookupStandardParamIDs(ctx, tx, derived)
	if err != nil {
		return fmt.Errorf("lookup standard_params: %w", err)
	}
	logger.Info("standard_params lookup",
		zap.Int("matched", len(stdPathIDs)),
		zap.Int("missing", len(missingPaths)))
	if len(missingPaths) > 0 {
		preview := missingPaths
		if len(preview) > 10 {
			preview = preview[:10]
		}
		logger.Warn("some seed paths not found in standard_params; sub_fields for these paths will be skipped",
			zap.Int("missing_total", len(missingPaths)),
			zap.Strings("missing_sample", preview))
	}

	chapterIDs, err := upsertChapterGroups(ctx, tx, seed)
	if err != nil {
		return fmt.Errorf("upsert mml_command_groups: %w", err)
	}
	logger.Info("mml_command_groups upserted", zap.Int("rows", len(chapterIDs)))

	cmdIDs, err := upsertDerivedCommands(ctx, tx, derived, chapterIDs)
	if err != nil {
		return fmt.Errorf("upsert mml_commands: %w", err)
	}
	logger.Info("mml_commands upserted", zap.Int("rows", len(cmdIDs)))

	subFieldRows, skipped, err := upsertCommandSubFields(ctx, tx, derived, cmdIDs, stdPathIDs, logger)
	if err != nil {
		return fmt.Errorf("upsert mml_command_sub_fields: %w", err)
	}
	logger.Info("mml_command_sub_fields upserted",
		zap.Int("rows", subFieldRows),
		zap.Int("skipped_missing_path", skipped))

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	logger.Info("seed import completed",
		zap.Duration("elapsed", time.Since(start)),
		zap.String("version_code", seed.VersionCode()))
	return nil
}

// computeSeedFileHash sha256(seed file) hex；失败返空串。
func computeSeedFileHash(path string) (string, error) {
	bytes, err := os.ReadFile(path) //nolint:gosec
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(bytes)
	return hex.EncodeToString(sum[:]), nil
}

// upsertSeedVersion 写 mml_param_versions 一行。
//
// version_code 是 VARCHAR PK/UNIQUE，由其它表 param_version FK 引用 — 保持稳定字符串。
func upsertSeedVersion(ctx context.Context, tx pgx.Tx, seed *SeedRoot, hash string) error {
	var hashArg interface{}
	if hash != "" {
		hashArg = hash
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO mml_param_versions (
			id, version_code, version_name, description,
			source, is_active, is_deprecated, content_hash
		) VALUES (
			gen_random_uuid(), $1, $2, $3,
			$4, true, false, $5
		)
		ON CONFLICT (version_code) DO UPDATE
		SET version_name  = EXCLUDED.version_name,
		    description   = EXCLUDED.description,
		    source        = EXCLUDED.source,
		    is_active     = true,
		    is_deprecated = false,
		    content_hash  = COALESCE(EXCLUDED.content_hash, mml_param_versions.content_hash),
		    updated_at    = NOW()
	`,
		seed.VersionCode(),
		seed.VersionName(),
		fmt.Sprintf("由 cmcc_tdlte_v%s.json 派生（mmlstandardloader.ImportSeedFile）", seed.Version),
		SeedImportSource,
		hashArg,
	)
	return err
}

// lookupStandardParamIDs 按 derived commands 中涉及的 leaf path 批量查询
// standard_params 已有记录的 id，返回 path → id 映射。
//
// standard_params 表为只读引用：其数据由产品/参数模型管理页面预先维护，
// importer 不写入新行。仅查询 sub_fields 关联所需的 leaf path（object_path
// 行不参与 M:N 关联，跳过查询）。
//
// 第二个返回值是 seed 中存在但 standard_params 缺失的 path 列表（去重并
// 排序），调用方可据此打印 warning；这些 path 对应的 sub_fields 将被跳过。
func lookupStandardParamIDs(
	ctx context.Context, tx pgx.Tx, cmds []DerivedCommand,
) (map[string]string, []string, error) {
	// 收集所有 sub_field 关联用 leaf path（去重）。
	pathSet := make(map[string]struct{})
	for _, c := range cmds {
		if !c.HasSubFields() {
			continue
		}
		for _, p := range c.SubFields {
			pathSet[p.Path] = struct{}{}
		}
	}
	if len(pathSet) == 0 {
		return map[string]string{}, nil, nil
	}

	paths := make([]string, 0, len(pathSet))
	for p := range pathSet {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	rows, err := tx.Query(ctx, `
		SELECT id::text, standard_path
		FROM standard_params
		WHERE standard_path = ANY($1)
	`, paths)
	if err != nil {
		return nil, nil, fmt.Errorf("query standard_params: %w", err)
	}
	defer rows.Close()

	ids := make(map[string]string, len(paths))
	for rows.Next() {
		var id, path string
		if err := rows.Scan(&id, &path); err != nil {
			return nil, nil, fmt.Errorf("scan standard_params row: %w", err)
		}
		ids[path] = id
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate standard_params rows: %w", err)
	}

	var missing []string
	for _, p := range paths {
		if _, ok := ids[p]; !ok {
			missing = append(missing, p)
		}
	}
	return ids, missing, nil
}

// upsertChapterGroups 写 18 个一级分组（SA…SR）到 mml_command_groups。
//
// 字段决策：
//   - group_code = chapter code（SA / SB / …）
//   - chapter_code = group_code（同步冗余，便于按 chapter 过滤）
//   - parent_id 不存在（migration 000090 已 DROP；用 path ltree 替代）
//   - path = chapter code（LTREE，单 label）
//   - source='standard', catalog_protected=true
//
// UNIQUE (param_version, group_code) 用于 ON CONFLICT。
func upsertChapterGroups(ctx context.Context, tx pgx.Tx, seed *SeedRoot) (map[string]string, error) {
	versionCode := seed.VersionCode()
	out := make(map[string]string, len(seed.Groups))
	for i, g := range seed.Groups {
		nameJSON := jsonI18n(g.Name, g.Name)
		var id string
		row := tx.QueryRow(ctx, `
			INSERT INTO mml_command_groups (
				id, group_code, group_name_zh, group_name_en,
				path, name_i18n, param_version,
				display_order, is_active,
				source, catalog_protected, chapter_code
			) VALUES (
				gen_random_uuid(), $1, $2, $2,
				$3::ltree, $4::jsonb, $5,
				$6, true,
				$7, true, $1
			)
			ON CONFLICT (param_version, group_code) DO UPDATE
			SET group_name_zh     = EXCLUDED.group_name_zh,
			    group_name_en     = EXCLUDED.group_name_en,
			    path              = EXCLUDED.path,
			    name_i18n         = EXCLUDED.name_i18n,
			    display_order     = EXCLUDED.display_order,
			    is_active         = true,
			    source            = EXCLUDED.source,
			    catalog_protected = EXCLUDED.catalog_protected,
			    chapter_code      = EXCLUDED.chapter_code,
			    deprecated_at     = NULL,
			    updated_at        = NOW()
			RETURNING id::text
		`,
			g.Code, g.Name,
			g.Code, nameJSON, versionCode,
			i,
			SeedImportSource,
		)
		if err := row.Scan(&id); err != nil {
			return nil, fmt.Errorf("upsert chapter group %s: %w", g.Code, err)
		}
		out[g.Code] = id
	}
	return out, nil
}

// upsertDerivedCommands 写 derived commands 到 mml_commands。
//
// command_code 全表 UNIQUE — UPSERT 幂等。
//
// 字段决策：
//   - command_code = "<OP> <LOGICAL_CODE>"（任务 #3 §4 明确，空格分隔）
//   - logical_code = LOGICAL_CODE（去 OP 前缀）
//   - source='standard', catalog_protected=true（不可被 admin UI 删除/改）
//   - visibility = NULL（standard 行 visibility 无意义；migration 000132 的 CHECK 允许）
//   - target_paths 由本函数直接计算（不依赖 refresh trigger） — 每个 sub_field 的 path 即 target_path
//   - target_object 仅 ADD/RMV 填；LST/MOD 留 NULL
func upsertDerivedCommands(
	ctx context.Context, tx pgx.Tx,
	cmds []DerivedCommand, chapterIDs map[string]string,
) (map[string]string, error) {
	out := make(map[string]string, len(cmds))
	for i, c := range cmds {
		// target_paths：LST 为全部字段，MOD/ADD 为 sub_fields 的可写字段，RMV 为空
		var paths []string
		for _, p := range c.SubFields {
			paths = append(paths, p.Path)
		}
		if paths == nil {
			paths = []string{}
		}
		targetPathsJSON, err := json.Marshal(paths)
		if err != nil {
			return nil, fmt.Errorf("marshal target_paths %s: %w", c.CommandCode, err)
		}

		var groupID interface{}
		if id, ok := chapterIDs[c.GroupCode]; ok {
			groupID = id
		}

		var targetObject interface{}
		if c.TargetObject != "" {
			targetObject = c.TargetObject
		}

		category := CategorizePath(c.ObjectPath)
		nameJSON := jsonI18n(c.NameZh, c.NameEn)
		logicalNameJSON := jsonI18n(c.LogicalNameZh, c.LogicalNameEn)

		var id string
		// logical_code 列已 DROP（读时从 command_code 派生 "OP "前缀）；不再写入。
		row := tx.QueryRow(ctx, `
			INSERT INTO mml_commands (
				id, command_name, command_code, category, description, rpc_method,
				operation_type, target_paths, target_object,
				group_id, command_name_i18n,
				logical_name_i18n,
				source, catalog_protected,
				help_doc
			) VALUES (
				gen_random_uuid(), $1, $2, $3, $4, $5,
				$6, $7::jsonb, $8,
				$9, $10::jsonb,
				$11::jsonb,
				$12, true,
				''
			)
			ON CONFLICT (command_code) DO UPDATE
			SET command_name      = EXCLUDED.command_name,
			    category          = EXCLUDED.category,
			    description       = EXCLUDED.description,
			    rpc_method        = EXCLUDED.rpc_method,
			    operation_type    = EXCLUDED.operation_type,
			    target_paths      = EXCLUDED.target_paths,
			    target_object     = EXCLUDED.target_object,
			    group_id          = EXCLUDED.group_id,
			    command_name_i18n = EXCLUDED.command_name_i18n,
			    logical_name_i18n = EXCLUDED.logical_name_i18n,
			    source            = EXCLUDED.source,
			    catalog_protected = true,
			    deprecated_at     = NULL,
			    updated_at        = NOW()
			RETURNING id::text
		`,
			c.NameZh, c.CommandCode, category, c.NameZh, c.RPCMethod,
			c.OperationType, targetPathsJSON, targetObject,
			groupID, nameJSON,
			logicalNameJSON,
			SeedImportSource,
		)
		if err := row.Scan(&id); err != nil {
			return nil, fmt.Errorf("upsert command %s: %w", c.CommandCode, err)
		}
		out[c.CommandCode] = id
		_ = i
	}
	return out, nil
}

// upsertCommandSubFields 写 mml_command_sub_fields。
//
// LST 写全部字段，MOD/ADD 写 RW 字段，RMV 不写 sub_field。
//
// 关键点（migration 000113）：standard_path_id FK → standard_params(id)。
// 走 (command_id, standard_path_id) UNIQUE 做 UPSERT；同时填 mml_code（path 末段）—
// 后者还有 (command_id, mml_code) UNIQUE 约束需保证不重复。
//
// access_type 来自 SeedParam.Access：R → "RO"；RW → "RW"（migration 000132 的 CHECK 约束）。
//
// 如果某个 path 在 standard_params 中不存在（可能尚未被参数模型管理员录入），
// 该 sub_field 将被跳过且由调用方打印 warning；不中断导入。
//
// 返回插入/更新的总行数及因缺失 path 被跳过的行数。
func upsertCommandSubFields(
	ctx context.Context, tx pgx.Tx,
	cmds []DerivedCommand,
	cmdIDs map[string]string,
	stdPathIDs map[string]string,
	logger *zap.Logger,
) (int, int, error) {
	count := 0
	skipped := 0
	for _, c := range cmds {
		if !c.HasSubFields() || len(c.SubFields) == 0 {
			continue
		}
		cmdID, ok := cmdIDs[c.CommandCode]
		if !ok {
			return count, skipped, fmt.Errorf("missing command id for %s", c.CommandCode)
		}

		// 同 command 内 mml_code 必须 UNIQUE — 用计数后缀防重（极少出现，TR-181 leaf 段名通常已唯一）。
		mmlCodeSeen := make(map[string]int, len(c.SubFields))

		for sortOrder, p := range c.SubFields {
			stdPathID, ok := stdPathIDs[p.Path]
			if !ok {
				logger.Warn("sub_field skipped: standard_params row not found",
					zap.String("command_code", c.CommandCode),
					zap.String("path", p.Path))
				skipped++
				continue
			}

			mmlCode := lastSegmentForCode(p.Path)
			if mmlCodeSeen[mmlCode] > 0 {
				mmlCode = fmt.Sprintf("%s_%d", mmlCode, mmlCodeSeen[mmlCode]+1)
			}
			mmlCodeSeen[lastSegmentForCode(p.Path)]++

			labelJSON := jsonI18n(p.Name, p.Name)
			accessType := subFieldAccessType(p.Access)
			isRequired := p.IsWritable() // MOD 时 RW 字段默认必填

			_, err := tx.Exec(ctx, `
				INSERT INTO mml_command_sub_fields (
					id, command_id, standard_path_id, mml_code, label_i18n,
					default_selected, is_required, sort_order, access_type
				) VALUES (
					gen_random_uuid(), $1, $2, $3, $4::jsonb,
					true, $5, $6, $7
				)
				ON CONFLICT (command_id, standard_path_id) DO UPDATE
				SET mml_code         = EXCLUDED.mml_code,
				    label_i18n       = EXCLUDED.label_i18n,
				    is_required      = EXCLUDED.is_required,
				    sort_order       = EXCLUDED.sort_order,
				    access_type      = EXCLUDED.access_type,
				    deprecated_at    = NULL,
				    updated_at       = NOW()
			`,
				cmdID, stdPathID, mmlCode, labelJSON,
				isRequired, sortOrder, accessType,
			)
			if err != nil {
				return count, skipped, fmt.Errorf(
					"upsert sub_field cmd=%s path=%s: %w", c.CommandCode, p.Path, err)
			}
			count++
		}
	}
	return count, skipped, nil
}

// ============================================================
// 类型 / 字段值映射 helpers
// ============================================================

// subFieldAccessType seed access → mml_command_sub_fields.access_type CHECK 词表（"RO"/"RW"）。
func subFieldAccessType(a string) string {
	if a == AccessRW {
		return "RW"
	}
	return "RO"
}

// jsonI18n {"zh": zh, "en": en} JSONB 字符串。
func jsonI18n(zh, en string) []byte {
	m := map[string]string{"zh": zh, "en": en}
	b, _ := json.Marshal(m)
	return b
}

// lastSegmentForCode 取 path 末段（去 {i}）作为 mml_code / param_code。
//
// "Device.DeviceInfo.UserLabel" → "UserLabel"
// "Device.X.{i}.Y"              → "Y"
func lastSegmentForCode(p string) string {
	clean := stripInstanceIndex(p)
	clean = strings.TrimSuffix(clean, ".")
	if clean == "" {
		return clean
	}
	idx := strings.LastIndex(clean, ".")
	if idx == -1 {
		return clean
	}
	return clean[idx+1:]
}
