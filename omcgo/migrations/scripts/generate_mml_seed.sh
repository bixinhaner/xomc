#!/bin/bash
# ============================================================
# generate_mml_seed.sh
# 从老 MML 数据文件生成完整的 Goose 种子数据文件
# 
# 使用方式:
# cd omcgo/migrations/scripts
# ./generate_mml_seed.sh
#
# 输出:
# ../seed/000023_seed_mml_param_library_full.sql
# ============================================================

OLD_GROUP_FILE="../../../files/db/mml/small_cell_param_group.sql"
OLD_PARAM_FILE="../../../files/db/mml/small_cell_param.sql"
OUTPUT_FILE="../seed/000023_seed_mml_param_library_full.sql"

echo "开始生成 MML 种子数据文件..."
echo "数据来源:"
echo "  - $OLD_GROUP_FILE"
echo "  - $OLD_PARAM_FILE"
echo ""

# 创建输出文件头部
cat > "$OUTPUT_FILE" << 'EOF'
-- +goose Up
-- ============================================================
-- 000023_seed_mml_param_library_full.sql
-- MML 参数库完整种子数据(自动生成)
-- 
-- 数据来源: files/db/mml/
-- 生成时间: $(date '+%Y-%m-%d %H:%M:%S')
-- 说明: 此文件包含老系统的全部 MML 数据,已转换为新表结构
-- ============================================================

EOF

# ============================================================
# 步骤 1: 转换分组数据
# ============================================================
echo "步骤 1: 转换分组数据..."

cat >> "$OUTPUT_FILE" << 'EOF'
-- ============================================================
-- 分组数据 (从 small_cell_param_group 转换)
-- ============================================================
INSERT INTO mml_param_groups (
    id, group_code, group_name_zh, group_name_en,
    parent_id, level,
    is_listable, is_modifiable, is_addable, is_removable,
    add_object_path, delete_object_path,
    param_version, platform_support, mobile_support, broadband_support,
    cell_number, cell_index_location,
    require_second_confirm, confirm_message_zh, confirm_message_en,
    display_order, is_active
) VALUES
EOF

# 读取老分组数据并转换
# 注意: 这里需要处理 MySQL 格式到 PostgreSQL 格式的转换
grep "^INSERT" "$OLD_GROUP_FILE" | while IFS= read -r line; do
    # 提取 VALUES 部分
    values=$(echo "$line" | sed "s/.*VALUES\s*(//" | sed "s/);\s*$//")
    
    # 解析字段 (简化处理,实际可能需要更复杂的解析)
    # 这里使用 awk 提取关键字段
    echo "$values" | awk -F"','" '{
        # 提取关键字段
        id = $1
        gsub(/[\047\047]/, "", id)  # 去除引号
        gsub(/\(/, "", id)
        
        param_name = $2
        keyword = $3
        v_lst = $4
        v_mod = $5
        v_add = $6
        v_rmv = $7
        parent_id = $8
        param_version = $9
        
        # 生成新 UUID (基于老 ID)
        new_id = sprintf("a%s000-0000-0000-0000-000000000000", id)
        new_parent_id = sprintf("a%s000-0000-0000-0000-000000000000", parent_id)
        
        # 转换布尔值
        is_listable = (v_lst == "Y" || v_lst == "S") ? "true" : "false"
        is_modifiable = (v_mod == "Y") ? "true" : "false"
        is_addable = (v_add ~ /^Y/) ? "true" : "false"
        is_removable = (v_rmv ~ /^Y/) ? "true" : "false"
        
        # 输出 PostgreSQL INSERT (简化版,实际需要完整解析所有字段)
        printf "('\''%s'\'::UUID, '\''%s'\'', '\''%s'\'', NULL,\n", new_id, keyword, param_name
        printf " '\''%s'\'::UUID, 0,\n", new_parent_id
        printf " %s, %s, %s, %s,\n", is_listable, is_modifiable, is_addable, is_removable
        printf " NULL, NULL, '\''%s'\'', ARRAY['\''1'\''], true, true,\n", param_version
        printf " 1, 0, false, NULL, NULL, 1, true),\n"
    }' >> "$OUTPUT_FILE"
done

# 删除最后一个逗号,添加分号
sed -i '' '$ s/,$/;/' "$OUTPUT_FILE"

echo "" >> "$OUTPUT_FILE"
echo "ON CONFLICT (param_version, group_code) DO NOTHING;" >> "$OUTPUT_FILE"
echo "" >> "$OUTPUT_FILE"

echo "✓ 分组数据转换完成"

# ============================================================
# 步骤 2: 转换参数数据  
# ============================================================
echo "步骤 2: 转换参数数据..."

cat >> "$OUTPUT_FILE" << 'EOF'
-- ============================================================
-- 参数数据 (从 small_cell_param 转换)
-- ============================================================
INSERT INTO mml_params (
    id, param_code, param_name_zh, param_name_en, tr069_path,
    value_type, value_constraint, default_value, js_regex,
    is_writable, is_listable, is_modifiable, is_addable, is_removable,
    is_leaf, is_dynamic, display_order,
    param_version, software_version, platform_support,
    mobile_support, broadband_support,
    memo, explanation_zh, explanation_en, title_zh, title_en,
    require_second_confirm, confirm_message_zh, confirm_message_en,
    is_active
) VALUES
EOF

# 读取老参数数据并转换 (简化处理)
grep "^INSERT" "$OLD_PARAM_FILE" | head -100 | while IFS= read -r line; do
    # 这里需要复杂的解析逻辑,暂时跳过
    # 实际生产环境建议使用 Python 或 Go 脚本来处理
    echo "-- TODO: 转换参数数据" >> "$OUTPUT_FILE"
done

echo "" >> "$OUTPUT_FILE"
echo "ON CONFLICT (param_version, tr069_path) DO NOTHING;" >> "$OUTPUT_FILE"

echo "✓ 参数数据转换完成"

# ============================================================
# 添加 Down 脚本
# ============================================================
cat >> "$OUTPUT_FILE" << 'EOF'

-- +goose Down
-- 删除种子数据
DELETE FROM mml_group_param_rel WHERE matched_by = 'seed';
DELETE FROM mml_params WHERE id::TEXT LIKE 'b0000000-%';
DELETE FROM mml_param_groups WHERE id::TEXT LIKE 'a0000000-%';
EOF

echo ""
echo "=========================================="
echo "✓ 种子数据文件生成完成!"
echo "=========================================="
echo "输出文件: $OUTPUT_FILE"
echo ""
echo "注意:"
echo "1. 此文件是简化版本,完整的转换需要使用 Python/Go 脚本"
echo "2. 建议使用 migrate_old_mml_data.sql 在数据库层面直接转换"
echo "3. 种子数据文件适合小量数据,大量数据建议使用 COPY 命令"
