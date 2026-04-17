# MML 参数库迁移指南

## 📋 概述

本次迁移重新设计了 MML 参数管理的数据库表结构,主要改进包括:

- ✅ 显式外键关联(替代隐式路径匹配)
- ✅ PostgreSQL 高级特性(JSONB、数组、LTREE、生成列)
- ✅ 完整的审计追踪
- ✅ 兼容老系统查询

## 📁 文件说明

```
omcgo/migrations/
├── 000022_mml_param_library.sql          # Goose 迁移文件(自动执行)
└── scripts/
    ├── migrate_old_mml_data.sql          # 老数据迁移脚本(手动执行)
    └── legacy_compatibility_views.sql    # 兼容视图(手动执行)
```

## 🚀 迁移步骤

### 推荐方式: 使用数据库层面转换 (适合大数据量)

对于 7000+ 条参数数据,**推荐使用数据库层面的转换脚本**,而不是 Goose 种子文件:

```bash
cd omcgo

# 步骤 1: 执行 Goose 迁移(创建新表)
goose -dir migrations up

# 步骤 2: 执行数据转换(从老表转换到新表)
psql -U omcgo -d omcgo -f migrations/scripts/migrate_old_mml_data.sql

# 步骤 3: 创建兼容视图(可选)
psql -U omcgo -d omcgo -f migrations/scripts/legacy_compatibility_views.sql

# 步骤 4: 验证
psql -U omcgo -d omcgo -f migrations/scripts/verify_mml_migration.sql
```

**优势**:
- ✅ 自动处理所有 7000+ 条数据
- ✅ 自动建立关联关系
- ✅ 数据完整性验证
- ✅ 无需手动转换格式

### 备选方式: 使用 Goose 种子文件 (适合小量数据)

如果你只需要**部分示例数据**用于开发测试:

```bash
cd omcgo

# 执行包含示例数据的种子文件
goose -dir migrations/seed up

# 或使用 migrate 工具
./bin/omcgo-migrate --dsn "postgres://..." --path migrations/seed up
```

**说明**:
- `000023_seed_mml_param_library.sql` - 包含 14 条分组 + 9 条参数(示例数据)
- 适合开发环境快速测试
- 不适合生产环境(数据不完整)

### 方式 3: 自动生成完整种子文件 (不推荐)

理论上可以使用脚本自动生成完整种子文件,但由于数据量太大(7000+ 条),会导致:
- SQL 文件超过 10MB
- 执行时间很长
- 难以维护

```bash
# 生成完整种子文件(不推荐)
cd omcgo/migrations/scripts
chmod +x generate_mml_seed.sh
./generate_mml_seed.sh
```

**建议**: 生产环境使用方式 1,开发测试使用方式 2。

## 📊 新表结构说明

### mml_param_versions (版本管理)

管理不同基站型号的参数版本。

```sql
-- 查询所有版本
SELECT version_code, version_name, product_models 
FROM mml_param_versions;

-- 结果示例:
-- QB1.0   | Qcells B1.0       | {Qcells-B100}
-- CA2.0   | Celleagle A2.0    | {Celleagle-A200}
-- MLN1.0  | Multi-mode LTE    | {Baicells-Neo}
```

### mml_param_groups (参数分组)

树形结构的参数分组。

```sql
-- 查询 QB1.0 版本的分组树
SELECT 
    id, group_code, group_name_zh, 
    parent_id, level, display_order
FROM mml_param_groups
WHERE param_version = 'QB1.0'
  AND is_active = true
ORDER BY display_order;

-- 查询某个分组的所有子分组(使用 LTREE)
SELECT * FROM mml_param_groups
WHERE path <@ '基站配置.设备信息'::ltree;
```

### mml_params (参数定义)

具体的参数定义,包含 TR-069 路径和类型约束。

```sql
-- 查询 NTP 相关参数
SELECT 
    param_code, param_name_zh, tr069_path,
    value_type, value_constraint, default_value
FROM mml_params
WHERE tr069_path LIKE '%NTP%'
  AND param_version = 'QB1.0';

-- 查询 enum 类型的参数
SELECT 
    param_name_zh, tr069_path,
    value_constraint->>'labels' AS enum_labels
FROM mml_params
WHERE value_type = 'enum'
  AND param_version = 'QB1.0'
LIMIT 5;

-- 使用生成列查询特定层级
SELECT * FROM mml_params
WHERE tr069_path_parts[2] = 'Time';  -- Device.Time.*
```

### mml_group_param_rel (关联表)

显式定义分组和参数的多对多关系。

```sql
-- 查询某个分组下的所有参数
SELECT 
    p.param_code, p.param_name_zh, p.tr069_path,
    r.sort_order, r.matched_by
FROM mml_group_param_rel r
JOIN mml_params p ON p.id = r.param_id
WHERE r.group_id = (
    SELECT id FROM mml_param_groups 
    WHERE group_code = 'NTP' AND param_version = 'QB1.0'
)
ORDER BY r.sort_order;

-- 查看关联方式统计
SELECT 
    matched_by, 
    COUNT(*) AS count
FROM mml_group_param_rel
GROUP BY matched_by;
```

## 🔄 新旧字段映射

### 分组表映射

| 老字段 | 新字段 | 说明 |
|--------|--------|------|
| `keyword` | `group_code` | 分组编码 |
| `param_name` | `group_name_zh` | 中文名称 |
| `param_name_en` | `group_name_en` | 英文名称 |
| `v_lst` | `is_listable` | Y/N → boolean |
| `v_mod` | `is_modifiable` | Y/N → boolean |
| `v_add` | `is_addable` | Y/N → boolean |
| `v_rmv` | `is_removable` | Y/N → boolean |
| `add_path` | `add_object_path` | AddObject 路径 |
| `dis_order` | `display_order` | 显示顺序 |
| `stop_sign` | `!is_active` | 0/1 → boolean |

### 参数表映射

| 老字段 | 新字段 | 说明 |
|--------|--------|------|
| `PARAM_ID` | `id` | bigint → UUID |
| `PARAM_NAME` | `param_name_zh` | 中文名称 |
| `NAME_PATH` | `tr069_path` | TR-069 路径 |
| `MIB_DN` | `param_code` | MIB 编码 |
| `V_TYPE` | `value_type` + `value_constraint` | 字符串 → JSONB |
| `V_WRITABLE` | `is_writable` | W/- → boolean |
| `V_DYNAMIC` | `is_dynamic` | 1/0 → boolean |
| `DFT_VALUE` | `default_value` | 默认值 |
| `MEMO` | `memo` | 描述 |

## 🎯 使用示例

### 示例 1: 查询参数树

```sql
-- 查询完整的 QB1.0 参数树(前2级)
SELECT 
    group_code, group_name_zh,
    param_name, tr069_path,
    value_type, default_value
FROM v_param_tree
WHERE param_version = 'QB1.0'
  AND level <= 2
ORDER BY group_path, param_order;
```

### 示例 2: 查询带类型约束的参数

```sql
-- 查询所有 enum 类型参数及其选项
SELECT 
    param_name_zh,
    tr069_path,
    value_constraint->>'labels' AS labels,
    value_constraint->>'values' AS values
FROM mml_params
WHERE value_type = 'enum'
  AND param_version = 'QB1.0'
  AND is_active = true
LIMIT 10;
```

### 示例 3: 查询某层级的所有参数

```sql
-- 查询 Device.Time.* 下的所有参数
SELECT 
    param_code, param_name_zh, tr069_path,
    value_type, default_value
FROM mml_params
WHERE tr069_path_parts[1] = 'Device'
  AND tr069_path_parts[2] = 'Time'
  AND param_version = 'QB1.0'
ORDER BY display_order;
```

## ⚠️ 注意事项

1. **数据迁移前务必备份**
   ```bash
   pg_dump -U omcgo -d omcgo -f backup_before_mml_migration.sql
   ```

2. **在测试环境验证**
   - 先在小范围数据上测试迁移脚本
   - 验证数据完整性(记录数、外键关系)
   - 测试兼容视图的查询结果

3. **过渡期策略**
   - 保留老表不动
   - 新代码使用新表
   - 老代码通过兼容视图查询
   - 逐步迁移老代码

4. **性能优化**
   - 新增的索引已包含在迁移文件中
   - JSONB 字段使用 GIN 索引
   - 数组字段使用 GIN 索引
   - LTREE 使用 GiST 索引

## 🔧 回滚方案

如果需要回滚:

```sql
-- 方式 1: 使用 goose down
goose -dir migrations -dsn "postgres://..." down 20220222_mml_param_library

-- 方式 2: 手动删除
DROP VIEW IF EXISTS v_param_version_stats CASCADE;
DROP VIEW IF EXISTS v_param_tree CASCADE;
DROP VIEW IF EXISTS v_params_legacy CASCADE;
DROP VIEW IF EXISTS v_param_groups_legacy CASCADE;
DROP TABLE IF EXISTS mml_group_param_rel CASCADE;
DROP TABLE IF EXISTS mml_params CASCADE;
DROP TABLE IF EXISTS mml_param_groups CASCADE;
DROP TABLE IF EXISTS mml_param_versions CASCADE;
DROP FUNCTION IF EXISTS parse_v_type CASCADE;
DROP FUNCTION IF EXISTS build_v_type_string CASCADE;
```

## 📞 问题排查

### 问题 1: 迁移脚本报错 "function parse_v_type does not exist"

**原因**: Goose 迁移文件中函数定义有问题

**解决**: 检查 `000022_mml_param_library.sql` 中的函数定义,确保在迁移数据脚本之前执行

### 问题 2: 兼容视图查询结果为空

**原因**: 数据未迁移或 `is_active = false`

**解决**:
```sql
-- 检查数据是否存在
SELECT COUNT(*) FROM mml_params;
SELECT COUNT(*) FROM mml_params WHERE is_active = true;

-- 如果数据存在但未激活
UPDATE mml_params SET is_active = true WHERE is_active = false;
```

### 问题 3: 关联关系数量不符合预期

**原因**: 自动匹配规则可能有遗漏

**解决**:
```sql
-- 查看未关联的参数
SELECT param_name_zh, tr069_path
FROM mml_params
WHERE NOT EXISTS (
    SELECT 1 FROM mml_group_param_rel WHERE param_id = mml_params.id
)
AND is_active = true;

-- 手动添加关联
INSERT INTO mml_group_param_rel (group_id, param_id, matched_by)
VALUES (
    (SELECT id FROM mml_param_groups WHERE group_code = 'NTP'),
    (SELECT id FROM mml_params WHERE tr069_path = 'Device.Time.NTPServer1'),
    'manual'
);
```

## 📚 相关文档

- [MML 参数库表结构重构设计](../../../docs/design/mml-param-library-redesign.md)
- [老 OMC MML 数据库结构分析](../../../files/db/mml/)
- [PostgreSQL JSONB 文档](https://www.postgresql.org/docs/current/datatype-json.html)
- [PostgreSQL LTREE 文档](https://www.postgresql.org/docs/current/ltree.html)
