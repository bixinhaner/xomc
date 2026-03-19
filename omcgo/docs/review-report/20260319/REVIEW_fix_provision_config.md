# Code Review Report

| 项目 | 值 |
|------|------|
| 提交基础 | main |
| 审查时间 | 2026-03-19 |
| Scope | config |
| 审查结论 | **PASS** |

## 变更概要

修复 Provisioning 引擎数据模型查找失败问题：
1. 修改 `registry.go` 中的 `resolveFromDB` 函数正确处理 `ErrNotFound`
2. 新增默认数据模型迁移文件

## 变更文件

| 文件 | 变更 | 说明 |
|------|------|------|
| internal/config/datamodel/registry.go | +37/-6 | 修复 ErrNotFound 处理逻辑 |
| migrations/000046_add_default_data_models.up.sql | +109 | 默认数据模型迁移 |
| migrations/000046_add_default_data_models.down.sql | +8 | 回滚迁移 |
| migrations/000047_seed_data_models.up.sql | +185 | 数据模型种子数据 |

## 审查详情

### registry.go 修改

**修复前问题**：
```go
dm, err := r.repo.FindActive(ctx, carrier, tech, oui, productClass, model.ScopeProduct)
if err != nil {
    return nil, fmt.Errorf("find active product model: %w", err)  // ErrNotFound 直接返回错误
}
```

**修复后**：
```go
dm, err := r.repo.FindActive(ctx, carrier, tech, oui, productClass, model.ScopeProduct)
if err != nil {
    if !errors.Is(err, commonerrors.ErrNotFound) {
        return nil, fmt.Errorf("find active product model: %w", err)  // 仅系统错误返回
    }
    r.logger.Debug("no product-level data model found, trying oui level", ...)  // 记录并继续
}
```

**评估**: ✅ 正确实现了三级回退逻辑：
- product 级未找到 → 继续查 oui 级
- oui 级未找到 → 继续查 carrier_default 级
- carrier_default 级未找到 → 返回 nil（允许无数据模型）

### 迁移文件检查

- ✅ `ON CONFLICT DO NOTHING` 确保幂等性
- ✅ 包含三家运营商（cmcc/ctcc/cucc）默认模型
- ✅ 包含 BaiCells 厂商级和产品级模型

### 发现

| 级别 | 数量 |
|------|------|
| CRITICAL | 0 |
| WARNING | 0 |
| INFO | 0 |

## 关联问题

修复日志错误：
```
resolve data model: resolve data model: find active product model: resource not found
```
