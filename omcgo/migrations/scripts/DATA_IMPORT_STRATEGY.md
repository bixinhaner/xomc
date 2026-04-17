# MML 参数库数据导入方案总结

## 📋 方案对比

| 方案 | 适用场景 | 数据量 | 文件大小 | 优点 | 缺点 |
|------|---------|--------|----------|------|------|
| **方案 1: Goose 种子文件(完整)** ⭐ | 开发/测试 | 9,126 条 | 4.0 MB | 完整数据、符合 Goose 规范 | 文件较大 |
| **方案 2: Goose 种子文件(示例)** | 快速演示 | 23 条 | 15 KB | 快速启动、文件小 | 数据不完整 |
| **方案 3: 数据库转换脚本** | 生产环境 | 7000+ 条 | 10 KB | 自动处理、完整性验证、快速 | 需要老表存在 |

## 📁 文件清单

### 核心迁移文件

```
omcgo/migrations/
└── 000022_mml_param_library.sql          # 创建新表结构 (必须执行)
```

### 数据导入文件

```
omcgo/migrations/scripts/
├── migrate_old_mml_data.sql              # ⭐ 方案1: 数据库转换脚本 (推荐)
├── legacy_compatibility_views.sql        # 兼容视图
├── verify_mml_migration.sql              # 验证脚本
├── generate_mml_seed.sh                  # 方案3: 自动生成脚本 (不推荐)
└── README_MML_MIGRATION.md              # 完整使用指南

omcgo/migrations/seed/
├── 000023_seed_mml_param_library.sql     # 方案1: Goose 种子文件 (完整数据 9126条) ⭐
└── 000023_seed_mml_param_library_example.sql  # 方案2: Goose 种子文件 (示例数据 23条)
```

## 🚀 快速开始

### 生产环境 (推荐方案 1)

```bash
# 1. 创建新表
cd omcgo
goose -dir migrations up

# 2. 转换老数据 (自动处理 7000+ 条)
psql -U omcgo -d omcgo -f migrations/scripts/migrate_old_mml_data.sql

# 3. 创建兼容视图 (可选)
psql -U omcgo -d omcgo -f migrations/scripts/legacy_compatibility_views.sql

# 4. 验证
psql -U omcgo -d omcgo -f migrations/scripts/verify_mml_migration.sql
```

### 开发测试 (方案 2)

```bash
# 1. 创建新表 + 示例数据
cd omcgo
goose -dir migrations up
goose -dir migrations/seed up

# 2. 验证 (只有 23 条示例数据)
psql -U omcgo -d omcgo -c "SELECT COUNT(*) FROM mml_param_groups;"
psql -U omcgo -d omcgo -c "SELECT COUNT(*) FROM mml_params;"
```

## ⚠️ 重要说明

### 为什么不推荐完整的 Goose 种子文件?

1. **文件体积过大**
   - 7000+ 条数据 → SQL 文件 > 10MB
   - 违反 Goose 种子文件的设计初衷(小量初始数据)

2. **执行效率低**
   - 7000+ 条 INSERT 语句
   - 每次 goose up 都要解析整个文件
   - 难以调试和维护

3. **格式转换复杂**
   - 老数据是 MySQL 格式(反引号)
   - 需要转换为 PostgreSQL 格式
   - V_TYPE 字符串需要解析为 JSONB
   - 自动脚本难以处理所有边界情况

4. **数据完整性**
   - 数据库转换脚本可以:
     - ✅ 自动建立树形结构
     - ✅ 自动匹配关联关系
     - ✅ 验证数据完整性
     - ✅ 处理边界情况
   - 静态 SQL 文件无法做到

### 为什么数据库转换脚本更好?

```sql
-- migrate_old_mml_data.sql 的优势:

-- 1. 使用 parse_v_type() 函数自动解析 V_TYPE
INSERT INTO mml_params (..., value_type, value_constraint)
SELECT ..., (parse_v_type(V_TYPE)).value_type, (parse_v_type(V_TYPE)).value_constraint
FROM small_cell_param;

-- 2. 自动建立关联关系
INSERT INTO mml_group_param_rel (group_id, param_id, matched_by)
SELECT g.id, p.id, 'path_prefix'
FROM mml_param_groups g
JOIN mml_params p ON p.param_version = g.param_version
WHERE p.tr069_path LIKE (g.add_object_path || '%');

-- 3. 数据完整性验证
DO $$
DECLARE
    v_old_count INT;
    v_new_count INT;
BEGIN
    SELECT COUNT(*) INTO v_old_count FROM small_cell_param;
    SELECT COUNT(*) INTO v_new_count FROM mml_params;
    
    IF v_old_count != v_new_count THEN
        RAISE WARNING '数据不一致!';
    END IF;
END $$;
```

## 📊 数据量统计

| 数据源 | 分组数 | 参数数 | 文件大小 |
|--------|--------|--------|----------|
| `small_cell_param_group.sql` | 1,919 条 | - | 842 KB |
| `small_cell_param.sql` | - | 7,226 条 | 5.3 MB |
| **总计** | **1,919** | **7,226** | **6.1 MB** |

## 🎯 最佳实践

### 开发环境
```bash
# 使用示例数据快速启动
goose -dir migrations up
goose -dir migrations/seed up
```

### 测试环境
```bash
# 使用完整数据测试
goose -dir migrations up
psql -U omcgo -d omcgo_test -f migrations/scripts/migrate_old_mml_data.sql
```

### 生产环境
```bash
# 1. 备份
pg_dump -U omcgo -d omcgo -f backup_$(date +%Y%m%d).sql

# 2. 执行迁移
goose -dir migrations up
psql -U omcgo -d omcgo -f migrations/scripts/migrate_old_mml_data.sql

# 3. 验证
psql -U omcgo -d omcgo -f migrations/scripts/verify_mml_migration.sql

# 4. 创建兼容视图 (如果需要)
psql -U omcgo -d omcgo -f migrations/scripts/legacy_compatibility_views.sql
```

## 🔧 常见问题

### Q1: 老表不存在怎么办?

**A**: `migrate_old_mml_data.sql` 需要老表 `small_cell_param_group` 和 `small_cell_param` 存在。

如果没有老表:
- **开发环境**: 使用 Goose 种子文件(示例数据)
- **生产环境**: 需要从备份恢复老表,或手动导入数据

### Q2: 只想要部分数据怎么办?

**A**: 修改 `migrate_old_mml_data.sql`,添加 WHERE 条件:

```sql
-- 只迁移 QB1.0 版本
FROM small_cell_param
WHERE PARAM_VERSION = 'QB1.0';

-- 只迁移特定分组
FROM small_cell_param_group
WHERE keyword IN ('NTP', 'SYNC', 'IPSEC');
```

### Q3: 如何验证数据迁移成功?

**A**: 执行验证脚本:

```bash
psql -U omcgo -d omcgo -f migrations/scripts/verify_mml_migration.sql
```

会输出:
- ✅ 表结构检查
- ✅ 数据统计
- ✅ 关联关系验证
- ✅ 函数测试

## 📚 相关文档

- [完整迁移指南](README_MML_MIGRATION.md)
- [实施总结](IMPLEMENTATION_SUMMARY.md)
- [老数据分析](../../../files/db/mml/)

---

**更新日期**: 2026-04-17  
**推荐方案**: 数据库转换脚本 (migrate_old_mml_data.sql)
