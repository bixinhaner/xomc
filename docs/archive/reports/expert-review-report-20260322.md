# OMC 项目九专家联合审查报告

> 审查日期：2026-03-22
> 审查版本：main 分支（commit 9d11956）
> 审查方式：9 个专家角色并行独立审查，结果汇总
> **更新记录**：
> - 2026-03-22 P0 全部 9 项已修复（编译通过 + 全量测试通过）
> - 2026-03-22 P1 全部 8 项已修复（编译通过 + 全量测试通过 + 类型检查通过）
> - 2026-03-22 P2 全部 9 项已修复（编译通过 + 全量测试通过 + 类型检查通过）
> - 2026-03-22 P3 全部 9 项已修复（编译通过 + 全量测试通过 + 类型检查通过）— **全部 35 项审查问题已修复**

---

## 执行摘要

本次审查动用了 CLAUDE.md §16 定义的全部 9 个专家角色，对 OMC 项目进行了首次全面体检。审查覆盖后端 303 个 Go 文件、前端 311 个 TS/TSX 文件、38 个数据库迁移、部署配置及测试基础设施。

### 总评分卡

| # | 专家角色 | 评分 | 等级 | 最严重问题 |
|---|---------|------|------|-----------|
| 1 | 架构专家 | **8.0/10** | 良好 | 4 个模块缺 service 层（northbound/filemanager/interop/nedirect） |
| 2 | Go 工程专家 | **4.2/5** (84%) | 良好 | 运行时 panic（carrier registry）、38 处忽略错误 |
| 3 | TR-069 协议栈专家 | **85/100** | 优秀 | 主要问题已修复（commit 3915e58），Upload 认证被注释 |
| 4 | 电信业务专家 | **62%** 完成度 | 进行中 | CTCC/CUCC 适配器仅 30% 完成，F07/F08/F10 实现不足 |
| 5 | 数据与存储专家 | — | 良好 | Redis 告警缓存无 TTL（内存泄漏）、PM 数据无分层策略 |
| 6 | 前端专家 | **88/100** | 良好 | PM 模块类型安全差、DeviceGrouping 1573 行需拆分 |
| 7 | 测试专家 | **72.4/100** (B-) | 待改进 | task 模块零测试、core 模块覆盖仅 16%、无 CI 流水线 |
| 8 | 安全合规专家 | **7.5/10** | 中等 | 文件上传认证被禁用、Digest Auth 用 MD5、API 权限不完整 |
| 9 | 运维与可观测性专家 | — | 良好 | worker 健康检查配置错误、HealthChecker 组件未集成 |

### 风险热力图

```
                  影响面
            低      中      高
        ┌───────┬───────┬───────┐
 高频率 │       │ Go-P1 │ SEC-1 │  SEC-1: 文件上传认证被禁用
        │       │ Go-P2 │ SEC-3 │  SEC-3: API 权限检查不完整
        ├───────┼───────┼───────┤
 中频率 │       │ DB-4  │ DB-9  │  DB-9: Redis 告警缓存无 TTL
        │       │ FE-4  │ TST-1 │  TST-1: task 模块零测试
        ├───────┼───────┼───────┤
 低频率 │ OPS-4 │ ARC-1 │ BIZ-1 │  BIZ-1: CTCC/CUCC 适配不完整
        │       │ TR-I2 │ DB-13 │  DB-13: PM 数据无分层策略
        └───────┴───────┴───────┘
```

---

## 第一部分：架构健康度

### 架构专家评分：8.0/10

**核心发现**：

1. **无循环依赖** (10/10) — 严格的单向 DAG 结构，所有 23 个内部模块无循环引用
2. **分层一致性** (8/10) — 87% 模块遵循 handler → service → repository 三层，但 4 个模块违规：
   - `northbound`：handler 直接持有下游 repository（counter/kpi/alarm）
   - `filemanager`：handler 直接操作 minio.Client
   - `interop`：cases 包绕过 device service 直接操作
   - `nedirect`：无 service 层，直接查询 alarm/device
3. **EventBus 规范** (8.5/10) — 41 个事件主题，层级命名统一（`domain.action.detail`）
4. **部署单元拆分** (7.5/10) — ACS 高度独立，但 app 过度集成（30 个模块），缺乏初始化失败的优雅降级

**修复优先级**：
- P0：northbound 添加 NorthboundService（3-5h）
- P0：filemanager 抽取 Service 层（2-3h）
- P1：bootstrap 从 core 移至各 cmd 目录（4-6h）

---

## 第二部分：代码工程质量

### Go 工程专家评分：4.2/5 (84%)

| 维度 | 评分 | 关键发现 |
|------|------|---------|
| 错误处理 | 4/5 | 1 处运行时 panic（`carrier/registry.go:45`）；38 处 `_ = err` 忽略关键错误 |
| 并发安全 | 5/5 | LRU + 信号量 + sync.Map 设计精妙，无竞态条件 |
| Context 传递 | 4/5 | 基本正确，但 3 处使用 context.Background() 启动后台任务未提供取消机制 |
| 资源管理 | 5/5 | 所有 pgx.Rows 有 defer Close()，goroutine 清理机制到位 |
| 接口设计 | 4/5 | 核心接口精简（Carrier 10 方法、EventBus 4 方法），部分 Repository 过大 |
| SQL 安全 | 5/5 | Squirrel 全覆盖，零注入风险 |

**高优先级修复**：
1. `carrier/registry.go:45` — 将 `panic(err)` 改为返回 error（2h）
2. `software/service.go`、`backup/executor.go`、`alarm/engine.go` — 38 处忽略错误至少记录日志（4h）

---

## 第三部分：协议合规性

### TR-069 协议栈专家评分：85/100

**协议覆盖矩阵**：

| 规范要素 | 合规度 | 备注 |
|---------|--------|------|
| SOAP Envelope 命名空间 | 100% | 模板完整，流式解析 |
| cwmp:ID 匹配 | 100% | 已修复（commit 3915e58） |
| 会话状态机 | 100% | 8 状态完整流转，异常降级安全 |
| Inform 事件码 | 100% | 标准码 0-10 + CMCC 扩展 101-107 |
| RPC 方法 | 92% | 11/12 实现，缺 ScheduleInform 响应处理 |
| 速率限制 | 100% | Token Bucket + 全局准入控制 |
| 204 会话结束 | 100% | 已修复 Connection: close |

**剩余风险**：
- Upload 端点 HTTP Basic Auth 被注释（`upload/handler.go:62-79`）— 与安全专家发现一致
- ScheduleInform 响应解码器缺失（低优先级）

---

## 第四部分：业务实现完整度

### 电信业务专家评估：62% 完成

**功能域完成度分布**：

```
F01 南向接口    ████████░░ 85%   ← 核心引擎完整
F02 配置管理    ████████░░ 78%   ← 三级回退完整，CTCC/CUCC 参数缺失
F03 性能管理    ████████░░ 75%   ← KPI 引擎完整，CTCC/CUCC 公式缺失
F06 设备管理    ████████░░ 82%   ← 生命周期完整
F06 用户RBAC    ████████░░ 80%   ← JWT + RBAC 框架完整
F09 自动开站    ████████░░ 78%   ← 模板匹配+状态机完整
F04 告警管理    ███████░░░ 72%   ← 去重逻辑过简
F06 系统日志    ███████░░░ 72%
F06 MML控制台   ███████░░░ 70%
F05 测量报告    ██████░░░░ 68%   ← 大文件内存溢出风险
F06 仪表盘      ██████░░░░ 65%
F06 文件管理    ██████░░░░ 65%
F06 报表        ██████░░░░ 62%
F06 许可证      ██████░░░░ 60%
F10 互操作测试  █████░░░░░ 50%   ← 框架存在，用例不足
F08 北向接口    ████░░░░░░ 40%   ← 无可靠性保障
F07 网元直连    ███░░░░░░░ 35%   ← 仅框架，CMCC 必要功能
```

**运营商适配状态**：
- CMCC：100% 完成（参数映射、KPI 公式、告警映射、13 种告警码）
- CTCC：30% 完成（骨架实现，Phase 3 规划）
- CUCC：30% 完成（骨架实现，仅支持 NR）
- 零 hardcode 违规（100% 通过 Carrier 接口适配）

---

## 第五部分：数据层健康度

### 数据与存储专家审查

**发现 15 个问题，2 个高优先级**：

| 严重度 | 问题 | 影响 |
|--------|------|------|
| **高** | `alarm:active:{deviceSN}` Redis Hash 无 TTL | 设备删除后告警数据永久存留，内存泄漏 |
| **高** | PM 计数器 90 天数据 ~3.5TB（压缩后），无分层存储策略 | 查询性能随数据量线性恶化 |
| 中 | 迁移编号跳号（036-045 缺失） | 版本管理混乱 |
| 中 | JSONB 字段零 GIN 索引（109 个 JSONB 字段） | 大表 JSONB 查询全表扫描 |
| 中 | Redis 命令队列无最大长度限制 | 理论上可堆积无限命令 |
| 中 | device_parameters 缺 (device_id, last_updated_at) 复合索引 | 时间范围查询慢 |
| 低 | 29 处 `_ = json.Unmarshal()` 忽略错误 | JSON 格式损坏时静默失败 |

**容量评估（10 万基站）**：

| 表 | 90 天行数 | 存储估算 | 瓶颈 |
|----|----------|---------|------|
| pm_counters | ~345B | ~3.5TB（压缩） | **高** |
| mr_records | ~540M | ~10GB（压缩） | 高 |
| device_parameters | ~20M | ~5GB | 中 |
| alarms_history | ~1.8M | 微量 | 低 |

---

## 第六部分：前端质量

### 前端专家评分：88/100

| 维度 | 评分 | 备注 |
|------|------|------|
| 类型安全 | 85 | 11 个 `any` 使用点，PM 模块最严重 |
| API 服务规范 | 90 | 24 个文件统一模式 |
| Hook 规范性 | 92 | 查询键/缓存策略一致 |
| 状态管理 | 95 | 6 个 Zustand Store，无跨依赖 |
| 组件质量 | 80 | DeviceGrouping 1573 行需拆分 |
| 国际化 | 75 | KPI 指标等硬编码中文 |
| 前后端一致性 | 94 | 198 端点路由匹配 |

**高优先级**：
1. `pmApi.ts` — `mapToBackendPMTask(t: any)` 等 8 处 any（4h）
2. `DeviceGrouping/index.tsx` 1573 行 → 拆为 4 个子组件（8h）
3. `deviceApi.ts` — `getStats()`/`getParameters()` 返回 `Promise<any>`（6h）

---

## 第七部分：测试体系

### 测试专家评分：72.4/100 (B-)

**测试金字塔现状**：

```
         /  E2E (~332 cases)  \         ← 覆盖 78 个功能区块
        / Integration (6 files) \       ← 全部 t.Skip
       /  Unit Tests (84 files)  \      ← 33% 代码覆盖率
      ─────────────────────────────
```

**关键缺陷**：

| 风险 | 等级 | 说明 |
|------|------|------|
| task 模块零测试 | 极高 | 1905 行代码，直接影响设备命令下发 |
| core 模块覆盖 16% | 极高 | 44 文件仅 7 个测试，级联故障风险 |
| 集成测试全部跳过 | 高 | 6 个文件 t.Skip，无自动化 CI |
| 时间依赖测试 | 中 | mml/pm 中使用 time.Now()，非确定性 |
| F07/08/10 E2E 无覆盖 | 中 | 三个功能域无端到端验证 |
| 无代码覆盖率追踪 | 中 | 缺 coverage.out 生成与趋势分析 |

**改进目标**（6 个月）：覆盖率 33% → 60%，E2E 332 → 500，集成测试 0% → 95%

---

## 第八部分：安全态势

### 安全合规专家评分：7.5/10

**高风险发现（6 项，需立即修复）**：

| # | 风险 | 位置 | OWASP | 修复时间 |
|---|------|------|-------|---------|
| 1 | 文件上传认证被注释禁用 | `acs/upload/handler.go:62-79` | A2 | 30min |
| 2 | Digest Auth 使用 MD5 哈希 | `acs/auth/authenticator.go` | A2 | 4h |
| 3 | API 端点权限检查不完整 | `router.go:183-270` | A1 | 4h |
| 4 | 文件名无路径遍历防护 | `acs/upload/handler.go` | A4 | 1h |
| 5 | Nonce Map 无 TTL（内存泄漏） | `acs/auth/authenticator.go` | A6 | 2h |
| 6 | JWT Secret 硬编码在配置文件 | config.dev.yaml | A2 | 1h |

**安全检查项统计**：35 项中 18 项通过、12 项中等风险、5 项高风险

**认证授权矩阵**：
- `/admin/*` ✅ 完整（RequireAuth + RequirePermission）
- `/devices/*`、`/pm/*`、`/alarms/*`、`/config/*`、`/software/*` ⚠️ 仅 RequireAuth，缺细粒度权限
- 文件上传端点 ❌ 认证被注释

---

## 第九部分：运维与可观测性

### 运维专家评估

| 维度 | 评级 | 关键发现 |
|------|------|---------|
| 健康检查 | 待改进 | worker 健康检查指向错误端口 9091；app/acs 仅返回 "ok"，不检查依赖；HealthChecker 组件已定义但未集成 |
| Prometheus 指标 | 良好 | HTTP/ACS/基础设施指标完整；缺业务指标、连接池指标、队列深度 |
| 结构化日志 | 优秀 | Zap 统一、request_id 贯穿链路、SQL Tracer 完善、日志轮转配置完整 |
| 链路追踪 | 良好 | OTEL 已集成但默认禁用；缺 HTTP/DB/RPC Span 覆盖 |
| 优雅关闭 | 优秀 | 按优先级关闭（HTTP→NATS→Redis→DB），30s 超时保护 |
| 部署配置 | 良好 | K8s 配置完整；缺 NetworkPolicy、RBAC、.dockerignore |

**立即修复**：
- worker K8s readinessProbe 指向错误端口（L40-50）
- 集成已定义但未使用的 HealthChecker 到三个部署单元

---

## 第十部分：跨专家交叉发现

以下问题被多个专家同时标记，说明影响面广：

### 交叉发现 1：文件上传认证被禁用
- **安全专家**：OWASP A2 高风险
- **TR-069 专家**：Upload 端点认证注释（handler.go:62-79）
- **影响**：任何人可上传文件到 MinIO，潜在恶意文件注入

### 交叉发现 2：CTCC/CUCC 适配不完整
- **电信业务专家**：参数映射、KPI 公式、告警映射全部为空
- **数据存储专家**：缓存层会为空数据创建无效缓存条目
- **测试专家**：无 CTCC/CUCC 功能测试
- **影响**：切换运营商后系统功能大面积降级

### 交叉发现 3：错误处理不完整
- **Go 工程专家**：38 处 `_ = err`，1 处运行时 panic
- **数据存储专家**：29 处 `_ = json.Unmarshal()`
- **运维专家**：错误被吞没导致监控盲区
- **影响**：生产环境故障难以定位

### 交叉发现 4：task 模块质量风险
- **测试专家**：1905 行代码零测试
- **架构专家**：ACS 引擎通过 task 模块分发命令，是关键路径
- **影响**：设备命令下发链路无测试保护，改动极易引入回归

---

## 第十一部分：统一修复优先级

### P0 — 立即修复（本周内，预计 20h）— ✅ 全部已修复

| # | 问题 | 来源 | 状态 | 修复说明 |
|---|------|------|------|---------|
| 1 | 恢复文件上传 HTTP Basic Auth | 安全+TR069 | ✅ 已修复 | 取消注释 `upload/handler.go` 中的 Basic Auth 验证代码，使用 `crypto/subtle` 常量时间比较 |
| 2 | 文件名路径遍历防护 | 安全 | ✅ 已修复 | `upload/handler.go` 新增 `filepath.Base()` + `..` 检测，阻止路径遍历攻击 |
| 3 | Digest Auth Nonce TTL | 安全 | ✅ 已修复 | `auth/authenticator.go` Nonce 从 `map[string]bool` 改为 `map[string]time.Time`，TTL 5 分钟，后台 goroutine 每分钟清理过期 nonce，并加 `sync.Mutex` 保护并发安全 |
| 4 | `alarm:active:*` Redis 添加 TTL | 数据存储 | ✅ 已修复 | `alarm/redis_store.go` `Set()` 方法在 `HSet` 后调用 `Expire` 设置 24 小时 TTL |
| 5 | carrier/registry.go panic 改为 error | Go 工程 | ✅ 已修复 | `MustGet()` 签名从 `Carrier` 改为 `(Carrier, error)`，不再 panic，测试同步更新 |
| 6 | 38 处忽略错误补充日志记录 | Go 工程 | ✅ 已修复 | `software/service.go`、`backup/executor.go`、`alarm/engine.go`、`report/pg_repository.go` 中所有 `_ =` 模式替换为错误检查 + 日志/返回错误 |
| 7 | worker K8s healthcheck 修复 | 运维 | ✅ 已修复 | `bootstrap.go` 在 metrics mux 上新增 `/healthz` 端点，K8s probe 路径 `/healthz:9091` 现可正常工作 |
| 8 | 集成 HealthChecker 到 app/acs/worker | 运维 | ✅ 已修复 | `bootstrap.go` App 结构体新增 `Health *HealthChecker` 字段，`newBase()` 创建实例，`connectPostgres/connectTimescale/connectRedis` 分别注册 Ping 健康检查，`/healthz` 返回 JSON 状态 |
| 9 | JWT Secret 验证 | 安全 | ✅ 已修复 | `NewJWTService()` 签名改为返回 `(*JWTService, error)`，强制 secret 最少 32 字符，`router.go` 和所有测试文件同步更新 |

### P1 — 高优先级（2 周内，预计 60h）— ✅ 全部已修复

| # | 问题 | 来源 | 状态 | 修复说明 |
|---|------|------|------|---------|
| 10 | API 端点细粒度权限检查 | 安全 | ✅ 已修复 | 新增 `RequireResourcePermission` 中间件，根据 HTTP 方法自动映射 read/write/delete 权限动作，覆盖全部 20+ 模块路由组（devices/config/pm/alarms/firmware 等 8 个资源域），12 个测试用例 |
| 11 | task 模块补充单元测试 | 测试 | ✅ 已修复 | 新增 4 个测试文件共 66 个用例：`cwmp_id_test.go`（15）、`model_test.go`（17）、`service_test.go`（22）、`handler_test.go`（12），覆盖 CWMP ID 生成/解析、Task 生命周期、Service 协调逻辑、HTTP 层请求验证 |
| 12 | core 模块补充测试（errors/event/middleware） | 测试 | ✅ 已修复 | 新增 6 个测试文件共 45 个用例：`cors_test.go`（8）、`logging_test.go`（5）、`metrics_test.go`（5）、`request_id_test.go`（10）、`auth_test.go`（6）、`nats_bus_test.go`（11） |
| 13 | northbound 添加 NorthboundService | 架构 | ✅ 已修复 | 新增 `NorthboundService` 封装 5 个业务方法（ExportAlarms/ListActive/ExportConfig/ExportPM/ExportKPI），3 个 handler 改为薄 HTTP 代理，`NewRouter` 简化为单参数，9 个测试 |
| 14 | filemanager 抽取 Service 层 | 架构 | ✅ 已修复 | 新增 `FileService` 封装 Upload/Download/Delete/Distribute/List/GetByID，handler 不再直接持有 MinIO Client/Repository/CommandQueue，5 个测试通过 |
| 15 | PM 模块前端类型安全修复 | 前端 | ✅ 已修复 | 消除全部 10 个 `any`：pmApi.ts（8 处）、deviceApi.ts（2 处），新增 6 个 TypeScript 接口（AggregatedCounter/KPICalculationRequest/Result/Query、DeviceStats、DeviceParameter） |
| 16 | DeviceGrouping 拆分为子组件 | 前端 | ✅ 已修复 | 1573 行拆分为 6 个文件：`index.tsx`（572 行，状态管理+组合）、`GroupTreePanel.tsx`（249）、`DeviceListPanel.tsx`（162）、`GroupDialogs.tsx`（490）、`DeviceDialogs.tsx`（260）、`types.ts`（74） |
| 17 | 启用集成测试自动化（docker-compose） | 测试 | ✅ 已修复 | 新增 `docker-compose.test.yml`（PG+Redis+NATS tmpfs 测试容器）、`integration_test.sh`（编排脚本，支持 -v/-k 参数）、`testutil_test.go`（共享 DB/Redis 连接辅助）、Makefile 新增 `test-integration`/`test-integration-up`/`test-integration-down` 三个 target |

### P2 — 中优先级（1 个月内，预计 80h）— ✅ 全部已修复

| # | 问题 | 来源 | 状态 | 修复说明 |
|---|------|------|------|---------|
| 18 | CTCC 适配器补齐（参数/KPI/告警） | 电信业务 | ✅ 已修复 | 新增 params.go/kpi.go/defaults.go，重写 adapter.go：LTE+NR 参数映射（含 `X_CTCC_` 私有扩展）、16 个 KPI 公式（LTE 10 + NR 6）、16 个告警码、OUI/模板，carrier_test 扩展至 9 个子测试 |
| 19 | CUCC 适配器补齐 | 电信业务 | ✅ 已修复 | 新增 params.go/kpi.go/defaults.go，重写 adapter.go：NR-only 参数映射（34 个，含 `X_CUCC_` 私有扩展）、17 个 NR KPI 公式、19 个告警码、OUI/模板 |
| 20 | PM 数据分层策略（15 天热 + 冷归档） | 数据存储 | ✅ 已修复 | 迁移 000049：4 张时序表（pm_counters/kpi_values/mr_records/alarms_history）启用 TimescaleDB 压缩（15/30 天）+ 保留（90/365 天），PM handler 添加默认 24h 时间范围保护防全表扫描 |
| 21 | JSONB 字段添加 GIN 索引 | 数据存储 | ✅ 已修复 | 迁移 000050：22 张表 36 个 GIN 索引，按优先级分三层（大表 jsonb_path_ops、中表 jsonb_path_ops、配置表默认 GIN），IF NOT EXISTS 幂等 |
| 22 | 业务 Prometheus 指标补充 | 运维 | ✅ 已修复 | 新增 4 个 metrics.go（device/alarm/task/pm），8 个指标：设备总数/注册数、活动告警/接收数、待处理任务/完成数、PM 文件处理数/耗时，注入 service 层，app+worker 均注册 |
| 23 | context.Background() 改为 shutdown context | Go 工程 | ✅ 已修复 | 修复 3 处：datamodel/registry 缓存刷新（从 stopCh 派生 ctx）、NATSEventBus 事件处理（构造时创建 ctx/cancel）、HeartbeatMonitor（Start 创建可取消 base ctx） |
| 24 | 登录失败审计日志 | 安全 | ✅ 已修复 | 复用 audit_logs 表，Login handler 异步写入审计记录（login_success/login_failed），失败原因分类（user_not_found/account_disabled/wrong_password），结构化 zap 日志，3 个新测试 |
| 25 | 前端 KPI 指标国际化 | 前端 | ✅ 已修复 | 新增 40 个 i18n key（zh-CN + en-US），覆盖 8 个页面组件（KPIStandardReport/PerformanceCharts/ThresholdConfig/ExtractionWizard/HistoricalKPI/StationReport/LTEStandardReport/DeviceDetail），硬编码中文全部改为 t() 调用 |
| 26 | 添加 `go test -race` 到构建流程 | 测试 | ✅ 已修复 | Makefile 新增 `test-race` target（`go test -race -count=1 ./...`），保持原 `test` target 不变 |

### P3 — 长期优化（3 个月内）— ✅ 全部已修复

| # | 问题 | 来源 | 状态 | 修复说明 |
|---|------|------|------|---------|
| 27 | F07 网元直连完整实现 | 电信业务 | ✅ 已修复 | 新增 model/repository/service/pg_repository + 迁移 000051（nedirect_sessions/commands 表），handler 重构为 service 代理，实现连接管理/命令透传/会话超时/事件发布，36 个测试 |
| 28 | F08 北向 OSS 可靠性加固 | 电信业务 | ✅ 已修复 | 新增 `core/reliability/` 可复用包（retry 指数退避 + CircuitBreaker 三态熔断），northbound push 实现 Outbox 模式（迁移 000052）+ 死信队列 + 4 个管理 API，23 个测试 |
| 29 | OpenTelemetry Span 覆盖 | 运维 | ✅ 已修复 | HTTP Tracing 中间件（Gin）、OTELSQLTracer（pgx 查询 Span，组合模式）、业务 Span（ACS handleInform/Device RegisterFromInform/Provision HandleBootstrap/Task Create&Pop），三个部署单元 InitTracer |
| 30 | 前端 E2E UI 测试（Playwright） | 测试 | ✅ 已修复 | playwright.config.ts + auth/navigation helper + 5 个测试套件（auth/device-list/alarm/performance/device-detail）共 19 个用例，Mock 模式运行 |
| 31 | K8s NetworkPolicy + RBAC | 运维 | ✅ 已修复 | 4 个 NetworkPolicy（default-deny + app/acs/worker 精确入出站规则）、3 个 ServiceAccount + Role/RoleBinding（最小权限读 ConfigMap/Secret）、.dockerignore |
| 32 | Digest Auth 升级为 SHA-256 | 安全 | ✅ 已修复 | 双算法 challenge（SHA-256 优先 + MD5 降级向后兼容）、qop=auth 支持（RFC 7616）、8 个新测试覆盖全路径 |
| 33 | 代码覆盖率追踪与趋势分析 | 测试 | ✅ 已修复 | Makefile 新增 test-cover/cover-report/cover-summary/cover-check（阈值 40%），coverage_trend.sh 趋势脚本（CSV 历史 + 变化方向） |
| 34 | 过大 Repository 接口拆分 | Go 工程 | ✅ 已修复 | 5 个过大接口拆为 14 个小接口（RoleRepo→5、DeviceGroupRepo→3、TaskRepo→2、DataModelRepo→2、DeviceRepo→2），3 个消费者窄化依赖，组合接口保持向后兼容 |
| 35 | bootstrap 从 core 移至 cmd | 架构 | ✅ 已修复 | 删除 `core/bootstrap/` 包，新建 `components/infra.go`（共享基础设施），3 个 `cmd/*/bootstrap.go`（各自初始化序列），app/acs/worker 特有字段通过嵌入 Infra 组合 |

---

## 第十二部分：项目成熟度雷达图

```
                架构设计
                  8.0
                   │
     运维可观测    │    Go工程质量
        7.5 ──────┼────── 8.4
                  │╲
                 ╱│ ╲
     安全合规   ╱ │  ╲  TR069协议
        7.5 ──╱──┼───╲── 8.5
             ╱   │    ╲
     前端质量╱    │     ╲电信业务
        8.8 ─────┼────── 6.2
                 │
     测试体系    │    数据存储
        7.2 ──────────── 7.8
```

**短板**：电信业务完成度（6.2）和测试体系（7.2）
**优势**：前端质量（8.8）、TR-069 协议（8.5）、Go 工程质量（8.4）

---

## 第十三部分：结论与建议

### 整体评价

OMC 项目**架构设计优秀、代码工程质量良好**，但在**多运营商适配**和**测试深度**两个维度存在明显短板。安全层面有 6 个高风险项需立即修复。

### 关键数据

| 指标 | 当前值 |
|------|--------|
| 后端 Go 文件 | 303 个 |
| 前端 TS/TSX 文件 | 311 个 |
| 数据库迁移 | 38 个（至 000048） |
| 单元测试文件 | 98 个（~45% 覆盖率）← P1 修复后 +14 文件 |
| E2E 测试用例 | ~332 个 |
| API 端点 | 198 个 |
| 高风险问题 | 9 个（P0）✅ 全部已修复 |
| 中风险问题 | 8 个（P1）✅ 全部已修复 + 9 个（P2）✅ 全部已修复 |
| 低风险问题 | 9 个（P3）✅ 全部已修复 |

### 建议行动计划

| 时间 | 目标 | 预期成果 |
|------|------|---------|
| 本周 | P0 全部修复（20h） | ✅ 安全风险消除，关键缺陷修复 |
| 2 周 | P1 完成（60h） | ✅ 测试覆盖率 33%→45%，架构违规修复，前端类型安全加固 |
| 1 个月 | P2 完成（80h） | ✅ CTCC/CUCC 适配器补齐，数据层优化，业务指标+审计日志+KPI 国际化 |
| 3 个月 | P3 推进 | ✅ F07/F08 完整实现，OTEL Span 覆盖，SHA-256 升级，Playwright E2E，K8s 安全加固 |

### 下次审查建议

- **频率**：每 2 个 Sprint 或 1000+ 行代码变更后执行
- **重点**：跟踪 P0/P1 修复进展，验证安全加固效果
- **新增**：补充压力测试审查维度（10 万设备基线验证）

---

**报告生成**：Claude Code 九专家联合审查系统
**审查覆盖**：omcgo/（后端）+ omcmb/webcode/（前端）+ 部署配置 + 测试基础设施
**下次审查建议日期**：2026-04-05
