package catalogloader

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

// upsertSummaryV2 是 v2 catalog 单次 upsert 的统计结果。
type upsertSummaryV2 struct {
	ChapterGroupCount  int // §R-1 一级分组，预期 18
	CommandLeafCount   int // op leaves，预期 ~190
	LinkHealthFailures int // §R-2.5.2 tree_node_refs 命中 standard_params 失败的次数
	LinkHealthResolved int // 之前记录但本次命中成功的失败项被标 resolved_at=NOW()
}

// upsertCatalogV2 在单事务内完成 v2 catalog 的 UPSERT + link_health 维护。
//
// 与 v1 upsertCatalog 的关键差异：
//   - 一级分组是 18 个 SA-SR 章节（§R-1，group_code = "chapter:SA"），不再是 71 个
//     object 组；object_path_template 列也用 "chapter:SA" 写入，与 v1 的
//     "Device.X.*" 共享同一列但 namespace 不冲撞。
//   - 每条命令把 spec MD 派生的 standardPath 集合写入新增的 tree_node_refs JSONB
//     列（§R-2.5），同时保留写入旧 target_paths 列做兼容窗口；P3 cutover 后下线
//     target_paths。
//   - command 显示名 / logical_name_i18n 直接采用 §R-2.4 命令中文名权威表（catalog
//     文件已嵌入 LogicalNameI18n），不再走 commandDisplayName(group.name + op) 拼。
//   - InstanceRangeMeta 写入新列（§R-4.1.1），前端 Control Panel InstancePicker
//     运行时按此校验 {i}。
//   - 启动期对每条 treeNodeRef 做 standard_params 命中检查，未命中入
//     mml_catalog_link_health（§R-2.5.2 失败清单）；ADD/RMV 父级实例路径
//     `Device.X.{i}.` 不算 standard_params 行，跳过检查。
//
// 兼容期策略：v1 standard 行不被本函数触动。P2.c cutover 时由 Loader 主入口
// 决定是调用 v1 还是 v2 upsert，并在 v2 启用后单独清理 v1 standard 行。
//
// 事务语义：任何错误回滚整个事务，DB 保持原状（无半成品）。
func (l *Loader) upsertCatalogV2(ctx context.Context, cat *Catalog) (*upsertSummaryV2, error) {
	if !cat.IsV2() {
		return nil, fmt.Errorf("upsertCatalogV2: catalog schemaVersion=%q, expected 'v2'", cat.SchemaVersion)
	}
	s := &upsertSummaryV2{}

	tx, err := l.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := ensureParamVersion(ctx, tx, cat.Carrier, cat.Tech, cat.SpecVersion); err != nil {
		return nil, fmt.Errorf("ensure param_version: %w", err)
	}

	// 预加载 standard_params 全集到 set，避免命令循环里每 path 一次 SELECT。
	knownPaths, err := preloadStandardParams(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("preload standard_params: %w", err)
	}

	for gi := range cat.Groups {
		g := &cat.Groups[gi]
		gid, err := upsertChapterGroupV2(ctx, tx, cat.Carrier, cat.Tech, g)
		if err != nil {
			return nil, fmt.Errorf("upsert chapter group %s: %w", g.GroupCode, err)
		}
		s.ChapterGroupCount++

		for ci := range g.Commands {
			cmd := &g.Commands[ci]
			if err := upsertCommandV2(ctx, tx, gid, cmd); err != nil {
				return nil, fmt.Errorf("upsert command %s: %w", cmd.CommandCode, err)
			}
			s.CommandLeafCount++

			// §R-2.5.2 link health 命中检查
			if cmd.OperationType == "ADD" || cmd.OperationType == "RMV" {
				// ADD/RMV 的 treeNodeRefs 是父级实例路径（如 Device.X.{i}.），
				// 它本身不是 standard_params 行，跳过检查。
				continue
			}
			for _, ref := range cmd.TreeNodeRefs {
				if _, ok := knownPaths[ref]; ok {
					continue
				}
				if err := recordLinkHealthFailure(
					ctx, tx, cat.SpecVersion, ref, cmd.GroupCodeObject, cmd.OperationType,
				); err != nil {
					return nil, fmt.Errorf("record link_health failure for %s: %w", ref, err)
				}
				s.LinkHealthFailures++
			}
		}
	}

	// 之前记录但本次 standard_params 已有的失败项 → 标 resolved
	resolved, err := resolveLinkHealthSucceeded(ctx, tx, cat.SpecVersion)
	if err != nil {
		return nil, fmt.Errorf("resolve link_health: %w", err)
	}
	s.LinkHealthResolved = resolved

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	l.logger.Info("mml-catalog v2 upsert done",
		zap.String("spec_version", cat.SpecVersion),
		zap.Int("chapters", s.ChapterGroupCount),
		zap.Int("commands", s.CommandLeafCount),
		zap.Int("link_failures", s.LinkHealthFailures),
		zap.Int("link_resolved", s.LinkHealthResolved),
	)
	return s, nil
}

// upsertChapterGroupV2 UPSERT 一个 chapter 分组到 mml_param_groups。
//
// group_code / object_path_template 都用 "chapter:SA" 形式（§R-1 namespace），
// 与 v1 的 "Device.X.*" 共存于同一列但无 key 冲撞。
// path LTREE depth=1（即只有 chapter 一段），符合 §R-1 多级分组禁令。
func upsertChapterGroupV2(ctx context.Context, tx pgx.Tx, carrier, tech string, g *Group) (uuid.UUID, error) {
	nameI18nJSON, _ := json.Marshal(g.NameI18n)
	nameZH := g.NameI18n["zh-CN"]
	nameEN := g.NameI18n["en-US"]

	const selectSQL = `
		SELECT id FROM mml_param_groups
		WHERE object_path_template = $1 AND deleted_at IS NULL
		LIMIT 1;
	`
	const updateSQL = `
		UPDATE mml_param_groups SET
			group_name_zh = $2,
			group_name_en = $3,
			name_i18n = $4::jsonb,
			chapter_code = $5,
			display_order = $6,
			instance_arity = 0,
			instance_levels = '{}'::text[],
			deprecated_at = NULL,
			updated_at = NOW()
		WHERE id = $1;
	`
	// path LTREE：v2 chapter 单段，例 "chapter_SA"。归一化 ":" → "_"。
	const insertSQL = `
		INSERT INTO mml_param_groups (
			id, group_code, group_name_zh, group_name_en, name_i18n,
			param_version, display_order, source, catalog_protected,
			object_path_template, chapter_code,
			instance_arity, instance_levels,
			path,
			deprecated_at, created_at, updated_at
		) VALUES (
			gen_random_uuid(), $1, $2, $3, $4::jsonb,
			$5, $6, 'standard', true,
			$7, $8,
			0, '{}'::text[],
			$9::ltree,
			NULL, NOW(), NOW()
		) RETURNING id;
	`

	var id uuid.UUID
	err := tx.QueryRow(ctx, selectSQL, g.GroupCode).Scan(&id)
	if err == nil {
		if _, err := tx.Exec(ctx, updateSQL,
			id, nameZH, nameEN, nameI18nJSON, g.ChapterCode, g.DisplayOrder,
		); err != nil {
			return uuid.Nil, fmt.Errorf("update chapter group: %w", err)
		}
		return id, nil
	}
	if err != pgx.ErrNoRows {
		return uuid.Nil, fmt.Errorf("select chapter group: %w", err)
	}

	paramVersion := fmt.Sprintf("%s-%s-v2.3", carrier, tech)
	ltreePath := normalizeChapterLTreePath(g.GroupCode)
	if err := tx.QueryRow(ctx, insertSQL,
		g.GroupCode, nameZH, nameEN, nameI18nJSON,
		paramVersion, g.DisplayOrder,
		g.GroupCode, g.ChapterCode,
		ltreePath,
	).Scan(&id); err != nil {
		return uuid.Nil, fmt.Errorf("insert chapter group: %w", err)
	}
	return id, nil
}

// upsertCommandV2 UPSERT 一条 v2 命令到 mml_commands。
//
// command_code 直接用 v2 schema 给的 "<OP>:<groupCodeObject>" 形式，与 v1 的
// "<OP>_<normalized>" 形式区分；mml_commands.command_code UNIQUE，新旧并存不冲突。
// tree_node_refs 写入新列；target_paths 兼容期同步写一份。
func upsertCommandV2(ctx context.Context, tx pgx.Tx, groupID uuid.UUID, cmd *Command) error {
	nameI18nJSON, _ := json.Marshal(cmd.LogicalNameI18n)
	treeNodeRefsJSON, _ := json.Marshal(cmd.TreeNodeRefs)
	targetPathsJSON, _ := json.Marshal(cmd.TreeNodeRefs) // 兼容窗口：双写
	rangeMetaJSON, _ := json.Marshal(cmd.InstanceRangeMeta)

	zhName := cmd.LogicalNameI18n["zh-CN"]
	if zhName == "" {
		zhName = cmd.CommandCode
	}
	displayName := fmt.Sprintf("%s %s", cmd.OperationType, zhName)

	logicalCode := normalizeForCommandCode(cmd.GroupCodeObject)
	rpcMethod := cmd.RPCMethod
	if rpcMethod == "" {
		rpcMethod = rpcMethodFor(cmd.OperationType)
	}

	const sql = `
		INSERT INTO mml_commands (
			id, command_name, command_code,
			rpc_method, operation_type,
			target_paths, tree_node_refs, instance_range_meta,
			group_id, logical_code, logical_name_i18n,
			source, catalog_protected,
			deprecated_at, created_at
		) VALUES (
			gen_random_uuid(), $1, $2,
			$3, $4,
			$5::jsonb, $6::jsonb, $7::jsonb,
			$8, $9, $10::jsonb,
			'standard', true,
			NULL, NOW()
		)
		ON CONFLICT (command_code) DO UPDATE SET
			command_name = EXCLUDED.command_name,
			rpc_method = EXCLUDED.rpc_method,
			operation_type = EXCLUDED.operation_type,
			target_paths = EXCLUDED.target_paths,
			tree_node_refs = EXCLUDED.tree_node_refs,
			instance_range_meta = EXCLUDED.instance_range_meta,
			group_id = EXCLUDED.group_id,
			logical_code = EXCLUDED.logical_code,
			logical_name_i18n = EXCLUDED.logical_name_i18n,
			source = 'standard',
			catalog_protected = true,
			deprecated_at = NULL;
	`
	if _, err := tx.Exec(ctx, sql,
		displayName, cmd.CommandCode,
		rpcMethod, cmd.OperationType,
		targetPathsJSON, treeNodeRefsJSON, rangeMetaJSON,
		groupID, logicalCode, nameI18nJSON,
	); err != nil {
		return fmt.Errorf("insert/update v2 command %s: %w", cmd.CommandCode, err)
	}
	return nil
}

// preloadStandardParams 读出 standard_params 全表的 standard_path 列，构造 O(1) 查询集。
// 启动期一次性，避免每条 treeNodeRef 一次 SELECT。
func preloadStandardParams(ctx context.Context, tx pgx.Tx) (map[string]struct{}, error) {
	rows, err := tx.Query(ctx, `SELECT standard_path FROM standard_params;`)
	if err != nil {
		return nil, fmt.Errorf("query standard_params: %w", err)
	}
	defer rows.Close()
	set := make(map[string]struct{}, 1024)
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, fmt.Errorf("scan standard_path: %w", err)
		}
		set[p] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate standard_params: %w", err)
	}
	return set, nil
}

// recordLinkHealthFailure UPSERT 一条失败到 mml_catalog_link_health。
// 自然键 = (spec_version, standard_path, group_code_object)；重复触发时刷新 detected_at。
// failure_reason 默认 'A'（spec 合法但 standard_params 未覆盖）；运维若手工细分原因
// 可后续 UPDATE 该列为 B/C/D。
func recordLinkHealthFailure(ctx context.Context, tx pgx.Tx, specVersion, standardPath, groupCodeObject, operationType string) error {
	const sql = `
		INSERT INTO mml_catalog_link_health (
			id, spec_version, standard_path, group_code_object,
			operation_type, failure_reason, detected_at,
			created_at, updated_at
		) VALUES (
			gen_random_uuid(), $1, $2, $3,
			$4, 'A', NOW(),
			NOW(), NOW()
		)
		ON CONFLICT (spec_version, standard_path, group_code_object) DO UPDATE SET
			operation_type = EXCLUDED.operation_type,
			detected_at = NOW(),
			resolved_at = NULL,
			updated_at = NOW();
	`
	if _, err := tx.Exec(ctx, sql, specVersion, standardPath, groupCodeObject, operationType); err != nil {
		return fmt.Errorf("upsert link_health: %w", err)
	}
	return nil
}

// resolveLinkHealthSucceeded 把"之前记录失败、但现在 standard_params 已包含"的行
// 标记 resolved_at=NOW()。运行在每次 Loader 完成上述 upsert 后，作为收尾步骤。
func resolveLinkHealthSucceeded(ctx context.Context, tx pgx.Tx, specVersion string) (int, error) {
	const sql = `
		UPDATE mml_catalog_link_health
		SET resolved_at = NOW(), updated_at = NOW()
		WHERE spec_version = $1
		  AND resolved_at IS NULL
		  AND standard_path IN (SELECT standard_path FROM standard_params);
	`
	ct, err := tx.Exec(ctx, sql, specVersion)
	if err != nil {
		return 0, fmt.Errorf("resolve link_health: %w", err)
	}
	return int(ct.RowsAffected()), nil
}

// normalizeChapterLTreePath 把 group_code "chapter:SA" 归一化为合法 ltree label "chapter_SA"。
// ltree label 字符集：[a-zA-Z0-9_]，所以把 ":" 替换为 "_"。
func normalizeChapterLTreePath(groupCode string) string {
	return strings.ReplaceAll(groupCode, ":", "_")
}
