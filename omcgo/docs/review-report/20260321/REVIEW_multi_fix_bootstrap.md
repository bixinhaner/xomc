# 代码审查报告

| 字段 | 值 |
|------|-----|
| **日期** | 2026-03-21 |
| **审查者** | Claude Code |
| **Scope** | 多个 (bootstrap/deploy/migration) |
| **Type** | fix |
| **结论** | PASS_WITH_WARNINGS |

## 变更概述

修复多个独立问题：nginx 代理路径丢失、全局 logger 未设置、数据库迁移唯一约束冲突。

## 变更文件

| 文件 | 变更类型 | 说明 |
|------|----------|------|
| `internal/core/bootstrap/bootstrap.go` | 修复 | 添加 zap.ReplaceGlobals() |
| `deployments/docker/nginx.conf` | 修复 | 移除变量代理，直接 proxy_pass |
| `deployments/docker/Dockerfile.*` | 修改 | 配置文件改为 dev 版本 |
| `migrations/000047_seed_data_models.up.sql` | 重构 | 先删除后插入策略 |

## 详细审查

### 1. bootstrap.go - 全局 Logger 修复

**问题**: `logger.L(ctx)` 使用 `zap.L()` 获取全局 logger，但从未设置全局 logger。

**修复**:
```go
zap.ReplaceGlobals(logger)
```

**评估**: ✅ PASS - 正确修复，确保所有使用 `zap.L()` 的代码都能获取配置好的 logger

### 2. nginx.conf - 代理路径修复

**问题**: 使用 `set $backend` + `proxy_pass $backend` 时，nginx 不会自动追加请求路径。

**修复**:
```nginx
# Before
set $backend http://acs:7547;
proxy_pass $backend;

# After
proxy_pass http://acs:7547;
```

**评估**: ✅ PASS - 正确修复，Docker 内置 DNS 已处理服务发现

### 3. Dockerfile.* - 配置文件切换

**变更**: 将 `config.prod.yaml` 改为 `config.dev.yaml`

**评估**: ⚠️ WARNING - 开发调试用途，生产环境应使用 prod 配置

### 4. 000047 迁移 - 唯一约束冲突修复

**问题**: `ON CONFLICT (id)` 只处理主键冲突，不处理部分唯一索引冲突。

**修复**: 采用先删除后插入策略
- 删除 carrier_default/oui/product 级别可能冲突的记录
- 删除相同 ID 的记录
- 直接插入（无需 ON CONFLICT）

**评估**: ✅ PASS - 幂等性设计，可重复执行

## 检查清单

| 检查项 | 结果 |
|--------|------|
| Go 命名规范 | ✅ |
| 错误处理 | ✅ |
| SQL 幂等性 | ✅ |
| nginx 配置正确 | ✅ |
| 日志可观测性 | ✅ |

## 发现汇总

| 级别 | 数量 | 详情 |
|------|------|------|
| CRITICAL | 0 | - |
| WARNING | 1 | Dockerfile 使用 dev 配置 |
| INFO | 0 | - |

## 建议

1. **生产部署前**: 将 Dockerfile 配置文件改回 `config.prod.yaml`
2. **后续优化**: 考虑通过环境变量区分 dev/prod 配置

## 审查结论

**PASS_WITH_WARNINGS** - 核心修复正确，配置切换需注意生产环境。
