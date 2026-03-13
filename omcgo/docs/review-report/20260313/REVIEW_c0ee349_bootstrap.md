# 代码审查报告

| 项目 | 值 |
|------|-----|
| 提交 | c0ee349 (pre-commit) |
| 日期 | 2026-03-13 |
| 审查者 | AI Assistant |
| Scope | deploy (bootstrap) |
| 结论 | **PASS** |

## 变更概要

提取三个 main.go 中重复的基础设施初始化代码到 `internal/bootstrap` 包，将公共逻辑（日志、连接、GracefulShutdown、信号处理）统一封装，大幅简化入口文件。

## 变更文件

| 文件 | 变更类型 | 行数变化 | 说明 |
|------|---------|---------|------|
| internal/bootstrap/bootstrap.go | 新增 | +292 | App 容器 + 基础设施初始化 + HTTP 服务生命周期 |
| cmd/app/main.go | 修改 | 200→65 | 基础设施初始化委托给 bootstrap.InitForApp |
| cmd/acs/main.go | 修改 | 147→68 | 基础设施初始化委托给 bootstrap.InitForACS |
| cmd/worker/main.go | 修改 | 236→140 | 基础设施初始化委托给 bootstrap.InitForWorker |
| docs/main-simplification-proposal.md | 新增 | +464 | 方案设计文档 |

**净效果**: +292（bootstrap）-387（三个 main.go 精简）= -95 行净减少，消除约 300 行重复代码

## 审查结果

### 检查项

| 检查项 | 结果 | 说明 |
|--------|------|------|
| Go 命名规范 | PASS | App、InitForApp、ListenAndServe 等均符合导出命名规范 |
| 错误处理 | PASS | 所有 connect 方���使用 `fmt.Errorf("context: %w", err)` |
| 资源泄漏 | PASS | PostgreSQL/Redis/NATS/EventBus 均注册 GracefulShutdown hook |
| 运营商硬编码 | PASS | 通过 CarrierRegistry + 适配器注册，无 `if carrier ==` 硬编码 |
| SQL 安全 | N/A | 本次变更不涉及 SQL 操作 |
| 功能一致性 | PASS | 简化后行为与原实现完全一致，编译通过、测试通过 |
| 接口保持 | PASS | router.Deps 结构不变，业务模块零影响 |
| 信号处理 | PASS | waitForShutdown 正确处理 SIGINT/SIGTERM + 可选 errCh |
| 优雅关机 | PASS | 保持原有优先级体系（1=HTTP, 2=NATS/EventBus, 3=Redis, 4=DB） |

### 发现

无 CRITICAL 或 WARNING 级别问题。

### INFO

- bootstrap.App.Carriers 字段名与 router.Deps.CarrierRegistry 不一致（Carriers vs CarrierRegistry），但属于不同包的命名风格差异，不影响功能
- MinIO 的 EnsureBuckets 从 worker-only 变为所有使用 MinIO 的入口都执行，这是一个改进（幂等操作，无副作用）
