#!/usr/bin/env python3
# ============================================================
# convert_mml_to_seed.py
# 将老 MML MySQL 数据转换为 PostgreSQL Goose 种子文件
#
# 使用方式:
# cd omcgo/migrations/scripts
# python3 convert_mml_to_seed.py
#
# 输入:
# - ../../../files/db/mml/small_cell_param_group.sql
# - ../../../files/db/mml/small_cell_param.sql
#
# 输出:
# ../seed/000023_seed_mml_param_library.sql
# ============================================================

import re
import json
import sys

OLD_GROUP_FILE = "../../../files/db/mml/small_cell_param_group.sql"
OLD_PARAM_FILE = "../../../files/db/mml/small_cell_param.sql"
OUTPUT_FILE = "../seed/000023_seed_mml_param_library.sql"

# 合法的 value_type 值（对应 mml_params.chk_value_type 约束）
VALID_TYPES = {'string', 'enum', 'unsignedInt', 'unsignedIntList',
               'stringList', 'boolean', 'uniqueInt', 'int'}

def parse_v_type(v_type_str):
    """解析老 V_TYPE 字符串到 value_type 和 value_constraint"""
    if not v_type_str or v_type_str == '':
        return 'string', '{"type": "string"}'

    # enum 类型(双花括号): enum-{label1,label2}-{value1,value2}
    enum_match = re.match(r'^enum-\{([^}]*)\}-\{([^}]*)\}$', v_type_str)
    if enum_match:
        labels = enum_match.group(1).split(',')
        values = enum_match.group(2).split(',')
        constraint = {"type": "enum", "labels": labels, "values": values}
        return 'enum', json.dumps(constraint, ensure_ascii=False)

    # enum 类型(单花括号): enum-{val1,val2,...}
    enum_single = re.match(r'^enum-\{([^}]+)\}$', v_type_str)
    if enum_single:
        values = enum_single.group(1).split(',')
        constraint = {"type": "enum", "labels": values, "values": values}
        return 'enum', json.dumps(constraint, ensure_ascii=False)

    # string 带长度: string-[0:256]
    string_match = re.match(r'^string-\[(\d*):(\d*)\]$', v_type_str)
    if string_match:
        constraint = {"type": "string"}
        if string_match.group(1):
            constraint["min_length"] = int(string_match.group(1))
        if string_match.group(2):
            constraint["max_length"] = int(string_match.group(2))
        return 'string', json.dumps(constraint, ensure_ascii=False)

    # 数值类型带范围: unsignedInt-[1:65535]
    uint_match = re.match(r'^(unsignedInt|int|uniqueInt)-\[(\d*):(\d*)\]$', v_type_str)
    if uint_match:
        constraint = {"type": uint_match.group(1)}
        if uint_match.group(2):
            constraint["min"] = int(uint_match.group(2))
        if uint_match.group(3):
            constraint["max"] = int(uint_match.group(3))
        return uint_match.group(1), json.dumps(constraint, ensure_ascii=False)

    # 复杂数值类型: type-[min:max]-[default:tr069path]
    complex_uint = re.match(r'^(unsignedInt|int|uniqueInt)-\[([^:]*):([^\]]*)\]-\[.*\]$', v_type_str)
    if complex_uint:
        constraint = {"type": complex_uint.group(1), "raw": v_type_str}
        if complex_uint.group(2):
            constraint["min"] = int(complex_uint.group(2))
        if complex_uint.group(3):
            constraint["max"] = int(complex_uint.group(3))
        return complex_uint.group(1), json.dumps(constraint, ensure_ascii=False)

    # 列表类型带范围
    list_match = re.match(r'^(unsignedIntList|stringList)-\[(\d*):(\d*)\]$', v_type_str)
    if list_match:
        constraint = {"type": list_match.group(1)}
        if list_match.group(2):
            constraint["min"] = int(list_match.group(2))
        if list_match.group(3):
            constraint["max"] = int(list_match.group(3))
        return list_match.group(1), json.dumps(constraint, ensure_ascii=False)

    # 提取基础类型名（取第一个非字母字符之前的部分）
    base_type = re.match(r'^([a-zA-Z]+)', v_type_str)
    if base_type:
        type_name = base_type.group(1)
        if type_name in VALID_TYPES:
            return type_name, json.dumps({"type": type_name, "raw": v_type_str}, ensure_ascii=False)

    # 兜底: 使用 string 类型
    return 'string', json.dumps({"type": "string", "raw": v_type_str}, ensure_ascii=False)


def make_deterministic_uuid(prefix: str, old_id: str) -> str:
    """Generate a valid UUID from prefix + old numeric ID (8-4-4-4-12 format)."""
    padded = old_id.zfill(31)
    return f"{prefix}{padded[:7]}-{padded[7:11]}-{padded[11:15]}-{padded[15:19]}-{padded[19:]}"


def parse_insert_values(line):
    """解析 INSERT VALUES 语句,提取字段值"""
    match = re.search(r'VALUES\s*\((.*)\)\s*;?\s*$', line, re.DOTALL)
    if not match:
        return None

    values_str = match.group(1)
    values = []
    current = ''
    in_quotes = False
    i = 0

    while i < len(values_str):
        char = values_str[i]

        if char == "'" and (i == 0 or values_str[i-1] != '\\'):
            in_quotes = not in_quotes
            current += char
        elif char == ',' and not in_quotes:
            values.append(current.strip())
            current = ''
        else:
            current += char

        i += 1

    if current.strip():
        values.append(current.strip())

    return values


def convert_groups():
    """转换分组数据"""
    print("步骤 1: 转换分组数据...")

    inserts = []
    with open(OLD_GROUP_FILE, 'r', encoding='utf-8') as f:
        for line in f:
            if line.startswith('INSERT'):
                inserts.append(line)

    print(f"  找到 {len(inserts)} 条分组数据")

    # 第一遍: 收集所有存在的 old_id
    existing_ids: set[str] = set()
    parsed_rows = []
    for line in inserts:
        values = parse_insert_values(line)
        if not values or len(values) < 20:
            continue
        old_id = values[0].strip("'()")
        existing_ids.add(old_id)
        parsed_rows.append(values)

    print(f"  有效分组: {len(parsed_rows)} 条, 唯一 ID: {len(existing_ids)} 个")

    sql_values = []

    for values in parsed_rows:
        old_id = values[0].strip("'()")
        param_name = values[1].strip("'")
        keyword = values[2].strip("'") if values[2] != 'NULL' else ''
        if not keyword:
            keyword = f'_g_{old_id}'
        v_lst = values[3].strip("'")
        v_mod = values[4].strip("'")
        v_add = values[5].strip("'")
        v_rmv = values[6].strip("'")
        parent_id = values[7].strip("'()")
        param_version = values[8].strip("'")
        add_path = values[9].strip("'") if values[9] != 'NULL' else 'NULL'
        param_name_en = values[10].strip("'")
        mobile_support = values[11].strip("'")
        broadband_support = values[12].strip("'")
        platform_support = values[13].strip("'") if values[13] != 'NULL' else '1'
        cell_number = values[14].strip("'")
        cell_index_location = values[15].strip("'")
        second_confirm = values[16].strip("'")
        confirm_en = values[17].strip("'") if values[17] != 'NULL' else 'NULL'
        confirm_cn = values[18].strip("'") if values[18] != 'NULL' else 'NULL'
        dis_order = values[19].strip("'")

        new_id = make_deterministic_uuid('a', old_id)
        new_parent_id = make_deterministic_uuid('a', parent_id) if parent_id in existing_ids else None

        is_listable = 'true' if v_lst in ['Y', 'S'] else 'false'
        is_modifiable = 'true' if v_mod == 'Y' else 'false'
        is_addable = 'true' if v_add.startswith('Y') else 'false'
        is_removable = 'true' if v_rmv.startswith('Y') else 'false'

        if add_path == 'NULL' or add_path == '':
            add_path_sql = 'NULL'
        else:
            add_path_sql = f"'{add_path}'"

        require_second_confirm = 'true' if second_confirm not in ['NULL', ''] else 'false'

        param_name = param_name.replace("'", "''")
        param_name_en = param_name_en.replace("'", "''") if param_name_en != 'NULL' else 'NULL'
        confirm_cn = confirm_cn.replace("'", "''") if confirm_cn != 'NULL' else 'NULL'
        confirm_en = confirm_en.replace("'", "''") if confirm_en != 'NULL' else 'NULL'

        parent_sql = f"'{new_parent_id}'::UUID" if new_parent_id else 'NULL'
        sql = f"""('{new_id}'::UUID, '{keyword}', '{param_name}', {f"'{param_name_en}'" if param_name_en != 'NULL' else 'NULL'},
 {parent_sql}, 0,
 {is_listable}, {is_modifiable}, {is_addable}, {is_removable},
 {add_path_sql}, {add_path_sql},
 '{param_version}', ARRAY['{platform_support}'], {'true' if mobile_support == 'Y' else 'false'}, {'true' if broadband_support == 'Y' else 'false'},
 {cell_number}, {cell_index_location},
 {require_second_confirm}, {f"'{confirm_cn}'" if confirm_cn != 'NULL' else 'NULL'}, {f"'{confirm_en}'" if confirm_en != 'NULL' else 'NULL'},
 {dis_order}, true)"""

        sql_values.append(sql)

    print(f"  ✓ 转换 {len(sql_values)} 条分组数据")
    return sql_values


def convert_params():
    """转换参数数据"""
    print("步骤 2: 转换参数数据...")

    inserts = []
    with open(OLD_PARAM_FILE, 'r', encoding='utf-8') as f:
        for line in f:
            if line.startswith('INSERT'):
                inserts.append(line)

    print(f"  找到 {len(inserts)} 条参数数据")

    sql_values = []

    for idx, line in enumerate(inserts):
        values = parse_insert_values(line)
        if not values or len(values) < 30:
            continue

        param_id = values[0].strip("'()")
        param_name = values[1].strip("'")
        name_path = values[2].strip("'")
        dft_value = values[3].strip("'") if values[3] != 'NULL' else 'NULL'
        v_type = values[4].strip("'")
        v_writable = values[5].strip("'")
        v_lst = values[6].strip("'")
        v_mod = values[7].strip("'")
        v_add = values[8].strip("'")
        v_rmv = values[9].strip("'")
        is_leaf = values[10].strip("'")
        disp_ord = values[11].strip("'")
        stop_sign = values[12].strip("'")
        memo = values[13].strip("'") if values[13] != 'NULL' else 'NULL'
        v_dynamic = values[14].strip("'")
        param_version = values[15].strip("'")
        mib_dn = values[16].strip("'") if values[16] != 'NULL' else ''
        param_name_en = values[17].strip("'")
        mobile_support = values[18].strip("'")
        broadband_support = values[19].strip("'")
        js_regex = values[20].strip("'") if values[20] != 'NULL' else 'NULL'
        title_cn = values[21].strip("'") if values[21] != 'NULL' else 'NULL'
        title_en = values[22].strip("'") if values[22] != 'NULL' else 'NULL'
        platform_support = values[23].strip("'") if values[23] != 'NULL' else '1'
        cn_explanation = values[24].strip("'") if values[24] != 'NULL' else 'NULL'
        en_explanation = values[25].strip("'") if values[25] != 'NULL' else 'NULL'
        software_version = values[26].strip("'") if values[26] != 'NULL' else 'NULL'
        second_confirm = values[27].strip("'") if len(values) > 27 else 'NULL'
        confirm_en = values[28].strip("'") if len(values) > 28 and values[28] != 'NULL' else 'NULL'
        confirm_cn = values[29].strip("'") if len(values) > 29 and values[29] != 'NULL' else 'NULL'

        new_id = make_deterministic_uuid('b', param_id)

        value_type, value_constraint = parse_v_type(v_type)

        is_writable = 'true' if v_writable == 'W' else 'false'
        is_listable = 'true' if v_lst == 'Y' else 'false'
        is_modifiable = 'true' if v_mod == 'Y' else 'false'
        is_addable = 'true' if v_add.startswith('Y') else 'false'
        is_removable = 'true' if v_rmv.startswith('Y') else 'false'
        is_leaf_bool = 'true' if is_leaf == 'Y' else 'false'
        is_dynamic = 'true' if v_dynamic == '1' else 'false'

        param_name = param_name.replace("'", "''")
        param_name_en = param_name_en.replace("'", "''") if param_name_en else ''
        cn_explanation = cn_explanation.replace("'", "''") if cn_explanation != 'NULL' else 'NULL'
        en_explanation = en_explanation.replace("'", "''") if en_explanation != 'NULL' else 'NULL'
        memo = memo.replace("'", "''") if memo != 'NULL' else 'NULL'
        confirm_cn = confirm_cn.replace("'", "''") if confirm_cn != 'NULL' else 'NULL'
        confirm_en = confirm_en.replace("'", "''") if confirm_en != 'NULL' else 'NULL'

        dft_value_sql = f"'{dft_value}'" if dft_value != 'NULL' else 'NULL'
        js_regex_sql = f"'{js_regex}'" if js_regex != 'NULL' else 'NULL'
        memo_sql = f"'{memo}'" if memo != 'NULL' else 'NULL'
        cn_exp_sql = f"'{cn_explanation}'" if cn_explanation != 'NULL' else 'NULL'
        en_exp_sql = f"'{en_explanation}'" if en_explanation != 'NULL' else 'NULL'
        title_cn_sql = f"'{title_cn}'" if title_cn != 'NULL' else 'NULL'
        title_en_sql = f"'{title_en}'" if title_en != 'NULL' else 'NULL'
        software_sql = f"'{software_version}'" if software_version != 'NULL' else 'NULL'
        require_second_confirm = 'true' if second_confirm not in ['NULL', ''] else 'false'
        confirm_cn_sql = f"'{confirm_cn}'" if confirm_cn != 'NULL' else 'NULL'
        confirm_en_sql = f"'{confirm_en}'" if confirm_en != 'NULL' else 'NULL'

        sql = f"""('{new_id}'::UUID, '{mib_dn}', '{param_name}', {f"'{param_name_en}'" if param_name_en else 'NULL'},
 '{name_path}',
 '{value_type}', '{value_constraint.replace("'", "''")}',
 {dft_value_sql}, {js_regex_sql},
 {is_writable}, {is_listable}, {is_modifiable}, {is_addable}, {is_removable},
 {is_leaf_bool}, {is_dynamic}, {disp_ord},
 '{param_version}', {software_sql}, ARRAY['{platform_support}'],
 {'true' if mobile_support == 'Y' else 'false'}, {'true' if broadband_support == 'Y' else 'false'},
 {memo_sql}, {cn_exp_sql}, {en_exp_sql}, {title_cn_sql}, {title_en_sql},
 {require_second_confirm}, {confirm_cn_sql}, {confirm_en_sql},
 true)"""

        sql_values.append(sql)

        if (idx + 1) % 1000 == 0:
            print(f"  已转换 {idx + 1}/{len(inserts)} 条...")

    print(f"  ✓ 转换 {len(sql_values)} 条参数数据")
    return sql_values


def main():
    print("=" * 60)
    print("MML 数据转换为 Goose 种子文件")
    print("=" * 60)
    print()

    group_values = convert_groups()
    print()
    param_values = convert_params()
    print()

    print(f"步骤 3: 写入种子文件...")

    with open(OUTPUT_FILE, 'w', encoding='utf-8') as f:
        f.write("""-- +goose Up
-- ============================================================
-- 000023_seed_mml_param_library.sql
-- MML 参数库完整种子数据(自动生成)
--
-- 数据来源: files/db/mml/
-- - small_cell_param_group.sql (1919 条)
-- - small_cell_param.sql (7226 条)
--
-- 说明: 此文件包含老系统的全部 MML 数据,已转换为新表结构
-- ============================================================

""")

        # 参数版本
        f.write("-- ============================================================\n")
        f.write("-- 参数版本数据 (确保所有版本存在)\n")
        f.write("-- ============================================================\n")
        f.write("INSERT INTO mml_param_versions (version_code, version_name, description, product_models) VALUES\n")
        versions = [
            ('QB1.0', 'Qcells B1.0', '早期小站版本', 'Qcells-B100'),
            ('CA2.0', 'Celleagle A2.0', '中期版本', 'Celleagle-A200'),
            ('436Q1.0', '436 Q1.0', '特定型号版本', 'Baicells-436'),
            ('MLN1.0', 'Multi-mode LTE N1.0', '最新多模版本', 'Baicells-Neo'),
            ('MLQ1.0', 'Multi-mode LTE Q1.0', '多模LTE版本', 'Baicells-Neo'),
            ('BLX1.0', 'Baicells LTE X1.0', 'LTE版本', 'Baicells-BLX'),
            ('BAIBLQ1.0', 'Baicells BL Q1.0', 'Baicells BL版本', 'Baicells-BLQ'),
            ('BaiBNX1.0', 'Baicells N X1.0', 'NR版本', 'Baicells-NX'),
            ('CR4.0', 'CR 4.0', 'CR版本', 'Baicells-CR'),
            ('EA4.0', 'EA 4.0', 'EA版本', 'Baicells-EA'),
            ('EA4.0DUAL', 'EA 4.0 Dual', 'EA双模版本', 'Baicells-EA-Dual'),
            ('QC3.1', 'Qcells C3.1', 'Qcells版本', 'Qcells-C300'),
            ('QC4.2', 'Qcells C4.2', 'Qcells版本', 'Qcells-C400'),
            ('QC4.2T', 'Qcells C4.2T', 'Qcells T版本', 'Qcells-C400T'),
            ('BSC1.0', 'BSC 1.0', 'BSC版本', 'Baicells-BSC'),
            ('BTS1.0', 'BTS 1.0', 'BTS版本', 'Baicells-BTS'),
            ('DXDF1.0', 'DXDF 1.0', 'DXDF版本', 'Baicells-DXDF'),
            ('NBIOT1.0', 'NB-IoT 1.0', '物联网版本', 'Baicells-NB'),
            ('Nova430', 'Nova 430', 'Nova 430版本', 'Baicells-Nova430'),
            ('Nova430i', 'Nova 430i', 'Nova 430i版本', 'Baicells-Nova430i'),
            ('BaiBLN_3.0.3', 'Baicells BLN 3.0.3', 'BLN版本', 'Baicells-BLN'),
            ('BaiBLQ_3.0.2', 'Baicells BLQ 3.0.2', 'BLQ版本', 'Baicells-BLQ3'),
            ('ENB_DEFAULT_098', 'eNB Default 098', 'TR098默认', 'Baicells-eNB'),
            ('ENB_DEFAULT_181', 'eNB Default 181', 'TR181默认', 'Baicells-eNB'),
        ]
        for i, (code, name, desc, model) in enumerate(versions):
            comma = ',' if i < len(versions) - 1 else ''
            f.write(f"('{code}', '{name}', '{desc}', ARRAY['{model}']){comma}\n")
        f.write("ON CONFLICT (version_code) DO NOTHING;\n\n")

        # 分组数据
        f.write("-- ============================================================\n")
        f.write("-- 分组数据 (从 small_cell_param_group 转换)\n")
        f.write("-- ============================================================\n")
        f.write("INSERT INTO mml_param_groups (\n")
        f.write("    id, group_code, group_name_zh, group_name_en,\n")
        f.write("    parent_id, level,\n")
        f.write("    is_listable, is_modifiable, is_addable, is_removable,\n")
        f.write("    add_object_path, delete_object_path,\n")
        f.write("    param_version, platform_support, mobile_support, broadband_support,\n")
        f.write("    cell_number, cell_index_location,\n")
        f.write("    require_second_confirm, confirm_message_zh, confirm_message_en,\n")
        f.write("    display_order, is_active\n")
        f.write(") VALUES\n")

        for i, sql in enumerate(group_values):
            if i < len(group_values) - 1:
                f.write(sql + ",\n\n")
            else:
                f.write(sql + "\n\n")

        f.write("ON CONFLICT (param_version, group_code) DO NOTHING;\n\n")

        # 参数数据
        f.write("-- ============================================================\n")
        f.write("-- 参数数据 (从 small_cell_param 转换)\n")
        f.write("-- ============================================================\n")
        f.write("INSERT INTO mml_params (\n")
        f.write("    id, param_code, param_name_zh, param_name_en, tr069_path,\n")
        f.write("    value_type, value_constraint, default_value, js_regex,\n")
        f.write("    is_writable, is_listable, is_modifiable, is_addable, is_removable,\n")
        f.write("    is_leaf, is_dynamic, display_order,\n")
        f.write("    param_version, software_version, platform_support,\n")
        f.write("    mobile_support, broadband_support,\n")
        f.write("    memo, explanation_zh, explanation_en, title_zh, title_en,\n")
        f.write("    require_second_confirm, confirm_message_zh, confirm_message_en,\n")
        f.write("    is_active\n")
        f.write(") VALUES\n")

        for i, sql in enumerate(param_values):
            if i < len(param_values) - 1:
                f.write(sql + ",\n\n")
            else:
                f.write(sql + "\n\n")

        f.write("ON CONFLICT (param_version, tr069_path) DO NOTHING;\n\n")

        f.write("-- +goose Down\n")
        f.write("-- 删除种子数据\n")
        f.write("DELETE FROM mml_group_param_rel WHERE matched_by = 'seed';\n")
        f.write("DELETE FROM mml_params WHERE id::TEXT LIKE 'b%';\n")
        f.write("DELETE FROM mml_param_groups WHERE id::TEXT LIKE 'a%';\n")

    print(f"  ✓ 种子文件已生成: {OUTPUT_FILE}")
    print()
    print("=" * 60)
    print("✓ 转换完成!")
    print("=" * 60)
    print(f"分组数据: {len(group_values)} 条")
    print(f"参数数据: {len(param_values)} 条")
    print(f"输出文件: {OUTPUT_FILE}")


if __name__ == "__main__":
    try:
        main()
    except Exception as e:
        print(f"错误: {e}", file=sys.stderr)
        import traceback
        traceback.print_exc()
        sys.exit(1)
