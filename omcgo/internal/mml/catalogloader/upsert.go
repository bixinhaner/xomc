package catalogloader

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// upsertSummary 是单个 catalog 文件的 UPSERT 结果统计。
type upsertSummary struct {
	GroupCount      int
	CommandCount    int
	SubFieldCount   int
	DeprecatedCount int
	RowsAffected    int
}

// upsertCatalog 在单事务内完成一份 catalog 的 UPSERT + 差集软删。
//
// 失败语义：事务回滚，DB 保持原状（无半成品中间态）。
// 软删边界：仅 source='standard' AND carrier/tech 匹配的 group / command。
// admin / Customized 行 (source='admin') 不被触碰。
func (l *Loader) upsertCatalog(ctx context.Context, cat *Catalog) (*upsertSummary, error) {
	s := &upsertSummary{}

	tx, err := l.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// 确保 param_version 行存在（FK 约束依赖）。catalog 维度的版本由 carrier+tech+spec
	// 派生（与 upsertGroup 中的 paramVersion 派生一致），首次启动时自动 upsert。
	if err := ensureParamVersion(ctx, tx, cat.Carrier, cat.Tech, cat.SpecVersion); err != nil {
		return nil, fmt.Errorf("ensure param_version: %w", err)
	}

	incomingGroupCodes := make([]string, 0, len(cat.Groups))
	for _, g := range cat.Groups {
		gid, err := upsertGroup(ctx, tx, cat.Carrier, cat.Tech, &g)
		if err != nil {
			return nil, fmt.Errorf("upsert group %s: %w", g.GroupCode, err)
		}
		incomingGroupCodes = append(incomingGroupCodes, g.GroupCode)
		s.GroupCount++
		s.RowsAffected++

		// 命令：per-op_type 一行
		incomingOps := make([]string, 0, len(g.Commands))
		for _, cmd := range g.Commands {
			if err := upsertCommand(ctx, tx, gid, &g, &cmd); err != nil {
				return nil, fmt.Errorf("upsert command (%s, %s): %w", g.GroupCode, cmd.OperationType, err)
			}
			incomingOps = append(incomingOps, cmd.OperationType)
			s.CommandCount++
			s.RowsAffected++
		}
		// 差集软删 commands（仅 source='standard' 同 group_id 下 op 不在 incoming 的）
		n, err := softDeleteOrphanCommands(ctx, tx, gid, incomingOps)
		if err != nil {
			return nil, fmt.Errorf("soft delete orphan commands for %s: %w", g.GroupCode, err)
		}
		s.DeprecatedCount += n

		// 子字段：group 维度持久化（按 mml_code 在 group 内唯一）
		// 注意：现有 mml_command_sub_fields FK 是 command_id（保留兼容期，D24）；
		// Loader 把 group.subFields 写入：以本 group 内 op_type=LST 的 command 作为 owner
		// （sub_fields 实际跨 op 共享，运行期 BuildTree 按 group 聚合返回）。
		// 后续 P3 切换到 group_id FK 时移除 owner cmd 的过渡逻辑。
		incomingMMLCodes := make([]string, 0, len(g.SubFields))
		ownerCmdID, err := lookupOwnerCommandForSubFields(ctx, tx, gid)
		if err != nil {
			return nil, fmt.Errorf("lookup owner command for sub_fields of %s: %w", g.GroupCode, err)
		}
		if ownerCmdID != uuid.Nil {
			for _, sf := range g.SubFields {
				if err := upsertSubField(ctx, tx, ownerCmdID, &sf); err != nil {
					return nil, fmt.Errorf("upsert sub_field (%s, %s): %w", g.GroupCode, sf.MMLCode, err)
				}
				incomingMMLCodes = append(incomingMMLCodes, sf.MMLCode)
				s.SubFieldCount++
				s.RowsAffected++
			}
			n2, err := softDeleteOrphanSubFields(ctx, tx, ownerCmdID, incomingMMLCodes)
			if err != nil {
				return nil, fmt.Errorf("soft delete orphan sub_fields for %s: %w", g.GroupCode, err)
			}
			s.DeprecatedCount += n2
		}
	}

	// 差集软删 groups（仅 source='standard' AND carrier/tech 匹配 AND object_path_template 不在 incoming）
	n, err := softDeleteOrphanGroups(ctx, tx, cat.Carrier, cat.Tech, incomingGroupCodes)
	if err != nil {
		return nil, fmt.Errorf("soft delete orphan groups: %w", err)
	}
	s.DeprecatedCount += n

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	return s, nil
}

// upsertGroup UPSERT mml_param_groups 一行，返回 group.id。
//
// 唯一键：object_path_template（部分唯一索引，仅 deleted_at IS NULL 时生效）。
// 注意 mml_param_groups 表存在 (param_version, group_code) UNIQUE 约束，但
// 本 Loader 写入的是 v2.3 catalog 维度，不绑定 param_version；通过新加的
// object_path_template 列作为本 catalog 范围唯一标识。
func upsertGroup(ctx context.Context, tx pgx.Tx, carrier, tech string, g *Group) (uuid.UUID, error) {
	nameI18nJSON, _ := json.Marshal(g.NameI18n)
	instLevelsJSON, _ := json.Marshal(g.InstanceLevels)

	// group_code（旧字段，业务大类）继承 ChapterCode；object_path_template 是新 catalog 标识
	// param_version 用占位 'v2.3-cmcc-tdlte'；group_name_zh / group_name_en 取 nameI18n
	nameZH := g.NameI18n["zh-CN"]
	nameEN := g.NameI18n["en-US"]

	// mml_param_groups schema after 000090 + 000129:
	//   id, group_code, group_name_zh, group_name_en, path, param_version,
	//   display_order, is_active, name_i18n, source, catalog_protected,
	//   object_path_template, chapter_code, instance_arity, instance_levels,
	//   deprecated_at, deleted_at, created_*, updated_*
	//
	// 注：ON CONFLICT 子句不能直接引用部分唯一索引；这里用 WHERE NOT EXISTS 子查询
	// 实现 UPSERT 语义（先 UPDATE，无效 → INSERT）。
	//
	// 简化方案：先 SELECT 看是否存在 → UPDATE 或 INSERT。
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
			instance_arity = $7,
			instance_levels = $8::text[],
			deprecated_at = NULL,
			updated_at = NOW()
		WHERE id = $1;
	`
	const insertSQL = `
		INSERT INTO mml_param_groups (
			id, group_code, group_name_zh, group_name_en, name_i18n,
			param_version, display_order, source, catalog_protected,
			object_path_template, chapter_code, instance_arity, instance_levels,
			deprecated_at, created_at, updated_at
		) VALUES (
			gen_random_uuid(), $1, $2, $3, $4::jsonb,
			$5, $6, 'standard', true,
			$7, $8, $9, $10::text[],
			NULL, NOW(), NOW()
		) RETURNING id;
	`
	_ = instLevelsJSON

	var levels any
	if len(g.InstanceLevels) > 0 {
		levels = g.InstanceLevels
	} else {
		levels = []string{}
	}
	paramVersion := fmt.Sprintf("%s-%s-v2.3", carrier, tech)

	// 1. 尝试 UPDATE（基于 object_path_template 部分唯一索引）
	var id uuid.UUID
	err := tx.QueryRow(ctx, selectSQL, g.GroupCode).Scan(&id)
	if err == nil {
		// 已存在 → UPDATE
		if _, err := tx.Exec(ctx, updateSQL,
			id, nameZH, nameEN, nameI18nJSON,
			g.ChapterCode, g.DisplayOrder, g.InstanceArity, levels,
		); err != nil {
			return uuid.Nil, fmt.Errorf("update group %s: %w", g.GroupCode, err)
		}
		return id, nil
	}
	if err != pgx.ErrNoRows {
		return uuid.Nil, fmt.Errorf("select group: %w", err)
	}
	// 2. INSERT 新行
	// 注意：旧 group_code 列保留兼容，与新 object_path_template 同值
	err = tx.QueryRow(ctx, insertSQL,
		g.GroupCode, nameZH, nameEN, nameI18nJSON,
		paramVersion, g.DisplayOrder,
		g.GroupCode, g.ChapterCode, g.InstanceArity, levels,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("insert group %s: %w", g.GroupCode, err)
	}
	return id, nil
}

// upsertCommand UPSERT mml_commands 一行（per-op_type）。
//
// 唯一性靠 (command_code) 全局唯一 + (logical_code, operation_type) 业务唯一
// (现有 schema 中 command_code 是全表 UNIQUE)。本 Loader 用
// command_code = "<OP>_<groupCode>" 模式确保 deterministic。
func upsertCommand(ctx context.Context, tx pgx.Tx, groupID uuid.UUID, g *Group, cmd *Command) error {
	commandCode := fmt.Sprintf("%s_%s", cmd.OperationType, normalizeForCommandCode(g.GroupCode))
	logicalCode := normalizeForCommandCode(g.GroupCode)

	nameI18nJSON, _ := json.Marshal(g.NameI18n)
	targetPathsJSON, _ := json.Marshal(cmd.TargetPaths)

	rpcMethod := rpcMethodFor(cmd.OperationType)
	commandName := commandDisplayName(g.NameI18n, cmd.OperationType)

	const sql = `
		INSERT INTO mml_commands (
			id, command_name, command_code,
			rpc_method, operation_type,
			target_paths,
			group_id, logical_code, logical_name_i18n,
			source, catalog_protected,
			deprecated_at, created_at
		) VALUES (
			gen_random_uuid(), $1, $2,
			$3, $4,
			$5::jsonb,
			$6, $7, $8::jsonb,
			'standard', true,
			NULL, NOW()
		)
		ON CONFLICT (command_code) DO UPDATE SET
			command_name = EXCLUDED.command_name,
			rpc_method = EXCLUDED.rpc_method,
			operation_type = EXCLUDED.operation_type,
			target_paths = EXCLUDED.target_paths,
			group_id = EXCLUDED.group_id,
			logical_code = EXCLUDED.logical_code,
			logical_name_i18n = EXCLUDED.logical_name_i18n,
			source = 'standard',
			catalog_protected = true,
			deprecated_at = NULL;
	`
	_, err := tx.Exec(ctx, sql,
		commandName, commandCode,
		rpcMethod, cmd.OperationType,
		targetPathsJSON,
		groupID, logicalCode, nameI18nJSON,
	)
	if err != nil {
		return fmt.Errorf("insert/update command %s: %w", commandCode, err)
	}
	return nil
}

// lookupOwnerCommandForSubFields 找一条 group 下的 standard command 作为 sub_fields 的
// owner（兼容期方案：sub_fields FK 仍是 command_id）。优先 LST，找不到回落任意一条。
func lookupOwnerCommandForSubFields(ctx context.Context, tx pgx.Tx, groupID uuid.UUID) (uuid.UUID, error) {
	const sql = `
		SELECT id FROM mml_commands
		WHERE group_id = $1 AND source = 'standard' AND deprecated_at IS NULL
		ORDER BY CASE operation_type WHEN 'LST' THEN 0 ELSE 1 END, operation_type
		LIMIT 1;
	`
	var id uuid.UUID
	err := tx.QueryRow(ctx, sql, groupID).Scan(&id)
	if err == pgx.ErrNoRows {
		return uuid.Nil, nil
	}
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

// upsertSubField UPSERT mml_command_sub_fields 一行。
//
// 注意：现有 sub_fields 表的 param_id 列 FK 指向 standard_params（migration 000113）；
// 当 standard_path 在 standard_params 表中尚未注册时，先 INSERT 占位行（最小可工作）。
func upsertSubField(ctx context.Context, tx pgx.Tx, commandID uuid.UUID, sf *SubField) error {
	paramID, err := ensureStandardParam(ctx, tx, sf)
	if err != nil {
		return fmt.Errorf("ensure standard_param for %s: %w", sf.StandardPath, err)
	}

	labelI18nJSON, _ := json.Marshal(sf.LabelI18n)

	// DB 列名为 standard_path_id（migration 000113 重命名 param_id → standard_path_id）；
	// 上层 Go 模型保留 soft alias ParamID。
	const sql = `
		INSERT INTO mml_command_sub_fields (
			id, command_id, standard_path_id,
			mml_code, label_i18n,
			default_selected, is_required, sort_order,
			access_type,
			created_at, updated_at
		) VALUES (
			gen_random_uuid(), $1, $2,
			$3, $4::jsonb,
			$5, false, $6,
			$7,
			NOW(), NOW()
		)
		ON CONFLICT (command_id, mml_code) DO UPDATE SET
			standard_path_id = EXCLUDED.standard_path_id,
			label_i18n = EXCLUDED.label_i18n,
			sort_order = EXCLUDED.sort_order,
			access_type = EXCLUDED.access_type,
			deprecated_at = NULL,
			updated_at = NOW();
	`
	_, err = tx.Exec(ctx, sql,
		commandID, paramID,
		sf.MMLCode, labelI18nJSON,
		sf.DefaultSelected, sf.SortOrder,
		sf.AccessType,
	)
	if err != nil {
		return fmt.Errorf("insert/update sub_field %s: %w", sf.MMLCode, err)
	}
	return nil
}

// ensureStandardParam ensures a row exists in standard_params for sf.StandardPath
// and returns its id. If it does not exist, an inline INSERT is performed.
//
// 该函数避免 Loader 启动期因 standard_params 表内容缺失而失败。
func ensureStandardParam(ctx context.Context, tx pgx.Tx, sf *SubField) (uuid.UUID, error) {
	const selectSQL = `SELECT id FROM standard_params WHERE standard_path = $1 LIMIT 1;`
	var id uuid.UUID
	err := tx.QueryRow(ctx, selectSQL, sf.StandardPath).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != pgx.ErrNoRows {
		return uuid.Nil, fmt.Errorf("select standard_param: %w", err)
	}
	// 创建占位行（entry_type / access / data_type 与 standard_params 表 schema 对齐）
	const insertSQL = `
		INSERT INTO standard_params (id, standard_path, entry_type, access, data_type, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, 'parameter', $2, $3, NOW(), NOW())
		RETURNING id;
	`
	dataType := sf.ValueType
	if dataType == "" {
		dataType = "string"
	}
	access := sf.AccessType
	if access == "" {
		access = "RW"
	}
	err = tx.QueryRow(ctx, insertSQL, sf.StandardPath, access, dataType).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("insert placeholder standard_param: %w", err)
	}
	return id, nil
}

// softDeleteOrphanCommands 把 group 内 source='standard' 但不在 incomingOps 的 command 标 deprecated_at。
func softDeleteOrphanCommands(ctx context.Context, tx pgx.Tx, groupID uuid.UUID, incomingOps []string) (int, error) {
	const sql = `
		UPDATE mml_commands
		SET deprecated_at = NOW()
		WHERE group_id = $1
		  AND source = 'standard'
		  AND deprecated_at IS NULL
		  AND NOT (operation_type = ANY($2::text[]));
	`
	ct, err := tx.Exec(ctx, sql, groupID, incomingOps)
	if err != nil {
		return 0, err
	}
	return int(ct.RowsAffected()), nil
}

// softDeleteOrphanSubFields 把同 owner command 下、不在 incomingMMLCodes 的 sub_field 标 deprecated_at。
func softDeleteOrphanSubFields(ctx context.Context, tx pgx.Tx, ownerCmdID uuid.UUID, incomingMMLCodes []string) (int, error) {
	const sql = `
		UPDATE mml_command_sub_fields
		SET deprecated_at = NOW()
		WHERE command_id = $1
		  AND deprecated_at IS NULL
		  AND NOT (mml_code = ANY($2::text[]));
	`
	ct, err := tx.Exec(ctx, sql, ownerCmdID, incomingMMLCodes)
	if err != nil {
		return 0, err
	}
	return int(ct.RowsAffected()), nil
}

// softDeleteOrphanGroups 把同 carrier/tech 下、不在 incomingGroupCodes 的 standard group 标 deprecated_at。
func softDeleteOrphanGroups(ctx context.Context, tx pgx.Tx, carrier, tech string, incomingGroupCodes []string) (int, error) {
	paramVersion := fmt.Sprintf("%s-%s-v2.3", carrier, tech)
	const sql = `
		UPDATE mml_param_groups
		SET deprecated_at = NOW()
		WHERE param_version = $1
		  AND object_path_template IS NOT NULL
		  AND deprecated_at IS NULL
		  AND deleted_at IS NULL
		  AND NOT (object_path_template = ANY($2::text[]));
	`
	ct, err := tx.Exec(ctx, sql, paramVersion, incomingGroupCodes)
	if err != nil {
		return 0, err
	}
	return int(ct.RowsAffected()), nil
}

// rpcMethodFor 映射 operationType → RPC 方法名。
func rpcMethodFor(op string) string {
	switch op {
	case "LST":
		return "GetParameterValues"
	case "MOD":
		return "SetParameterValues"
	case "ADD":
		return "AddObject"
	case "RMV":
		return "DeleteObject"
	}
	return ""
}

// commandDisplayName 生成命令展示名（中文优先），如 "LST 设备信息"。
func commandDisplayName(nameI18n map[string]string, op string) string {
	for _, k := range []string{"zh-CN", "en-US"} {
		if v := nameI18n[k]; v != "" {
			return fmt.Sprintf("%s %s", op, v)
		}
	}
	return op
}

// normalizeForCommandCode 把对象路径模板 → SQL/MML 友好的标识符。
//
//	"Device.DeviceInfo.*"                  → "Device_DeviceInfo"
//	"Device.FaultMgmt.CurrentAlarm.{i}.*"  → "Device_FaultMgmt_CurrentAlarm_i"
func normalizeForCommandCode(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		switch {
		case r == '.', r == '{', r == '}', r == '*', r == ' ':
			if len(out) > 0 && out[len(out)-1] != '_' {
				out = append(out, '_')
			}
		default:
			out = append(out, r)
		}
	}
	// trim trailing underscore
	for len(out) > 0 && out[len(out)-1] == '_' {
		out = out[:len(out)-1]
	}
	return string(out)
}

// ensureParamVersion 确保 catalog 对应的 mml_param_versions 行存在。
// FK：mml_param_groups.param_version → mml_param_versions.version_code。
// 派生规则与 upsertGroup 保持一致：version_code = "{carrier}-{tech}-v2.3"。
// 注：specVersion 在文件中是 "cmcc-tdlte-v2.3" 类自描述串，不直接做 version_code。
func ensureParamVersion(ctx context.Context, tx pgx.Tx, carrier, tech, specVersion string) error {
	versionCode := fmt.Sprintf("%s-%s-v2.3", carrier, tech)
	versionName := fmt.Sprintf("%s %s %s", carrier, tech, specVersion)
	_, err := tx.Exec(ctx, `
		INSERT INTO mml_param_versions (
			id, version_code, version_name, description, is_active, source, created_at, updated_at
		) VALUES (
			gen_random_uuid(), $1, $2, $3, true, 'standard', NOW(), NOW()
		)
		ON CONFLICT (version_code) DO UPDATE SET
			version_name = EXCLUDED.version_name,
			updated_at = NOW();
	`, versionCode, versionName, "Auto-created by mml-catalog loader for spec "+specVersion)
	if err != nil {
		return fmt.Errorf("upsert param_version %s: %w", versionCode, err)
	}
	return nil
}
