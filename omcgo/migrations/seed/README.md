# MML 种子数据 - 快速参考卡

## 📦 数据概览
```
文件: migrations/seed/000005_seed_mml_param_library.sql
大小: 4.0 MB (103,803 行)
数据: 1,919 分组 + 7,207 参数 = 9,126 条
格式: PostgreSQL Goose 种子文件
```

## 🚀 一键执行
```bash
cd omcgo
goose -dir migrations up           # 创建表
goose -dir migrations/seed up      # 插入数据
```

## ✅ 快速验证
```sql
-- 数据量检查
SELECT 
  (SELECT COUNT(*) FROM mml_param_groups) as groups,
  (SELECT COUNT(*) FROM mml_params) as params;

-- 版本分布
SELECT param_version, COUNT(*) as cnt 
FROM mml_params 
GROUP BY param_version 
ORDER BY cnt DESC LIMIT 5;

-- 示例数据
SELECT group_code, group_name_zh, param_version 
FROM mml_param_groups 
WHERE group_code = 'NTP' 
LIMIT 3;
```

## 🔑 关键特性
- ✅ **幂等性**: 可重复执行 (ON CONFLICT DO NOTHING)
- ✅ **完整性**: 99.8% 数据转换成功
- ✅ **追溯性**: UUID 格式保留老 ID (a/b + 31位填充, 标准 8-4-4-4-12 格式)
- ✅ **兼容性**: 老系统 V_TYPE → JSONB 自动转换

## 📊 数据版本 Top 5
| 版本 | 数量 | 说明 |
|------|------|------|
| MLQ1.0 | 707 | ML 系列 |
| BLX1.0 | 603 | BLX 系列 |
| BAIBLQ1.0 | 588 | BaiBLQ 系列 |
| MLN1.0 | 559 | MLN 系列 |
| CR4.0 | 546 | CR 系列 |

## ⚡ 备选方案
```bash
# 如果种子文件执行慢,使用数据库转换脚本
psql -U omcgo -d omcgo -f migrations/scripts/migrate_old_mml_data.sql
```

## 📚 相关文档
- [转换报告](scripts/CONVERSION_REPORT.md) - 详细转换逻辑
- [实施总结](scripts/SEED_CONVERSION_SUMMARY.md) - 完整总结
- [使用指南](scripts/README_MML_MIGRATION.md) - 迁移指南

---
**生成日期**: 2026-04-17 | **状态**: ✅ 可用
