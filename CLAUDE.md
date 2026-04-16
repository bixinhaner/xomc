# CLAUDE.md — OMC 项目全局指导

> 本文件为 AI 编码助手提供项目全局上下文。后端详细指导参见 `omcgo/CLAUDE.md`。

---

## 1. 项目简介

**OMC**（Operations, Management and Control）是面向小基站/皮基站/微基站的无线操作维护中心系统。

| 维度 | 说明 |
|------|------|
| 核心协议 | TR069/CWMP（SOAP/XML over HTTP） |
| 运营商 | 中国移动（cmcc）、中国电信（ctcc）、中国联通（cucc） |
| 制式 | LTE (4G)、5G NR (SA) |
| 规模 | 10 万基站起步，预留 100 万级扩展 |
| 功能域 | 10 个域（F01-F10），43 项子功能 |

---

## 2. 开发理念

### 核心信念

- **商用级品质** — 这是面向运营商的商用网管系统，任何设计与实现不能偷工减料
- **增量进展胜过激进变革** — 小步修改，确保每次编译通过并通过测试
- **从现有代码中学习** — 实施前先研究项目中已有的模式和约定
- **实用主义胜过教条主义** — 适应项目现实，不为理论纯洁性牺牲交付
- **清晰意图胜过巧妙代码** — 选择平凡明显的方案，而非精巧难懂的实现

### 简单性原则

- 每个函数/模块/组件单一职责
- 避免过早抽象 — 三处相似代码才考虑提取公共逻辑
- 不玩巧妙技巧 — 选择平凡的解决方案
- 如果需要额外注释来解释逻辑，说明代码本身太复杂了

---

## 3. 仓库结构

```
omc/                                # 根仓库
├── CLAUDE.md                       # 本文件 — 全局指导
├── omcgo/                          # Go 后端（独立 git 仓库，根 .gitignore 忽略）
│   ├── CLAUDE.md                   # 后端详细指导
│   ├── cmd/                        # 入口：app / acs / worker / migrate / omcctl
│   ├── internal/                   # 私有代码（按功能域组织）
│   ├── migrations/                 # 数据库迁移
│   └── scripts/                    # E2E 测试、压测工具
├── omcmb/                          # React 前端（独立 git 仓库，根 .gitignore 忽略）
│   └── webcode/                    # 前端源码
│       ├── src/services/api/       # API 服务层（24 文件）
│       ├── src/hooks/api/          # React Query Hooks（21 文件）
│       ├── src/pages/              # 页面组件（18 模块）
│       └── src/components/         # 可复用组件
├── .claude/commands/               # Claude Code Skills
├── docs/                           # 文档与审查报告
└── *.md                            # 分析报告、开发计划等
```

**子仓库规则**：`omcgo/` 和 `omcmb/` 各自拥有独立 `.git`，根目录 `.gitignore` 忽略它们。后端/前端代码必须在各自子目录内执行 git 操作。

---

## 4. 技术栈总览

### 后端 (omcgo/)

| 组件 | 选型 |
|------|------|
| 语言 | Go 1.25 |
| HTTP (管理面) | Gin |
| HTTP (ACS) | net/http stdlib |
| RPC | gRPC (ACS ↔ App) |
| SQL 构建 | Squirrel（不使用 ORM） |
| DB 驱动 | pgx/v5 + pgxpool |
| 缓存 | Redis (go-redis/v9) |
| 消息 | NATS JetStream |
| 对象存储 | MinIO (S3 兼容) |
| 日志 | Zap |
| 指标 | Prometheus |
| 链路追踪 | OpenTelemetry |

### 前端 (omcmb/webcode/)

| 组件 | 选型 |
|------|------|
| 框架 | React 19 + TypeScript |
| 构建 | Vite |
| UI 库 | Ant Design 5 |
| 状态管理 | Zustand |
| 数据请求 | React Query (TanStack) |
| HTTP 客户端 | Axios |
| 图表 | ECharts |
| 国际化 | react-intl |
| 路由 | react-router-dom |

---

## 5. 架构概览

### 模块化单体 + 独立 ACS 引擎

不使用微服务架构。当前阶段保持模块化单体，100 万规模时可按功能域渐进拆分。

### 三个部署单元

| 单元 | 用途 |
|------|------|
| `omcgo-app` | 主应用（F02-F10 模块化单体）REST API :8080 |
| `omcgo-acs` | TR069 ACS 引擎（独立进程，水平可扩展）:7547 |
| `omcgo-worker` | 后台工作进程（PM/MR 文件处理、KPI 计算）|

### 前端 SPA

Vite dev server :3000 通过 `/api` 代理到后端 :8080。

---

## 6. 功能域速览

| 编号 | 功能域 | 核心职责 | 后端模块 |
|------|--------|---------|---------|
| F01 | 南向接口 (TR069) | ACS 引擎，SOAP/XML 协议处理 | `acs/` |
| F02 | 数据模型与配置 | TR069 参数树，三级回退，配置模板/基线 | `config/` |
| F03 | 性能管理 (PM/KPI) | 计数器采集，KPI 计算，时序存储 | `pm/` |
| F04 | 告警管理 | 告警接收、去重、关联、生命周期 | `alarm/` |
| F05 | 测量报告 (MR) | MRO/MRS/MRE 文件采集与解析 | `mr/` |
| F06 | OMC-R 核心 | 设备、用户、拓扑、固件、备份、仪表盘、运维、报表、MML、文件、日志、许可 | `device/` `admin/` `topology/` `software/` `backup/` `dashboard/` `ops/` `report/` `mml/` `filemanager/` `syslog/` `license/` |
| F07 | 网元直连 | 网元与网管直连通道 | `nedirect/` |
| F08 | 北向/OSS 接口 | 向上游 OSS 开放数据 | `northbound/` |
| F09 | 自动开站 | 设备自动发现、模板匹配、配置下发 | `provision/` |
| F10 | 互操作测试 | 设备联调与一致性验证 | `interop/` |

---

## 7. 关键文件路径

### 后端

| 路径 | 说明 |
|------|------|
| `omcgo/cmd/app/main.go` | 主应用入口 |
| `omcgo/cmd/app/router/router.go` | 路由注册 + DI |
| `omcgo/internal/` | 全部业务模块（按功能域扁平组织） |
| `omcgo/global/errors.go` | 63 个错误码 |
| `omcgo/migrations/` | 数据库迁移 (`000NNN_desc.up/down.sql`) |
| `omcgo/scripts/e2e_verify.sh` | E2E 测试脚本 (~452 cases) |
| `omcgo/scripts/seed_e2e_testdata.sql` | 测试数据 |
| `omcgo/CLAUDE.md` | **后端详细指导** |

### 前端

| 路径 | 说明 |
|------|------|
| `omcmb/webcode/src/services/api/*.ts` | API 服务（24 文件，一模块一文件） |
| `omcmb/webcode/src/services/http.ts` | Axios 客户端（拦截器、camelCase↔snake_case） |
| `omcmb/webcode/src/services/apiSwitch.ts` | Mock/Real 切换 (`useMock`) |
| `omcmb/webcode/src/hooks/api/*.ts` | React Query Hooks（21 文件） |
| `omcmb/webcode/src/pages/` | 页面组件（18 功能模块） |
| `omcmb/webcode/src/store/` | Zustand 状态管理 |
| `omcmb/webcode/src/types/` | TypeScript 类型定义 |

---

## 8. 开发规范

### 8.1 Git 工作流

**提交格式**: Conventional Commits，中文描述

```
<type>(<scope>): <中文简要描述>

<body>

<footer>
```

**Type**: `feat` | `fix` | `refactor` | `docs` | `test` | `chore` | `perf` | `build` | `ci` | `style`

**Scope** (与功能域对应):
`acs` `config` `pm` `alarm` `mr` `device` `admin` `topology` `software` `backup` `dashboard` `ops` `report` `mml` `filemanager` `syslog` `license` `nedirect` `northbound` `provision` `interop` `carrier` `components` `api` `deploy`

**子仓库 git 操作**:
- 后端: `cd omcgo && git add ... && git commit ...`
- 前端: `cd omcmb && git add ... && git commit ...`
- 禁止从根目录对子仓库文件执行 git 操作

**推送规范**:
- **严禁** 将 `git pull` 和 `git push` 放在同一条命令中执行
- 必须分步：先 `git pull --rebase`，确认无冲突后再 `git push`
- 遇到冲突时，先解决冲突完成 rebase，再推送

**提交示例**:
```
feat(device): 实现设备列表分页查询与批量操作

What: 新增 device handler 的 List/BatchDelete 接口，支持按 carrier/status 过滤
Why: Sprint 1 设备管理基础功能需求
Impact: 新增 GET /api/v1/devices 和 DELETE /api/v1/devices/batch 端点

Review: docs/review-report/20260312/REVIEW_abc1234_author_device.md
Related: F06
```

### 8.2 Go 后端规范

> 完整规范详见 `omcgo/CLAUDE.md`

- **命名**: 导出 `PascalCase`，未导出 `camelCase`，包名小写单数
- **接口优先**: 核心概念均定义接口（`DeviceRepository`、`Carrier`、`EventBus`）
- **错误处理**: `fmt.Errorf("context: %w", err)`，禁止裸 panic
- **SQL**: Squirrel 构建 + pgx 执行，禁止 ORM，禁止字符串拼接
- **运营商**: 全部通过 `Carrier` 接口适配器，禁止 `if carrier == "cmcc"` 硬编码
- **模块结构**: `handler.go` → `service.go` → `pg_repository.go` + `model.go`
- **DB 主键**: UUID (`gen_random_uuid()`)，所有表含 `created_at`/`updated_at`

### 8.3 React/TypeScript 前端规范

- **API 服务模式**: 每模块一个 `xxxApi.ts`，导出服务对象
- **Hook 模式**: 每模块一个 `useXxx.ts`，内部使用 `useMock ? mockService : realApi`
- **HTTP 客户端**: Axios 拦截器自动 camelCase ↔ snake_case 转换，Bearer token 注入
- **类型安全**: 禁止 `any`，Backend 响应用 `BackendXxx` 接口 → `mapBackendXxx` 转换 → 前端 `Xxx` 接口
- **状态管理**: Zustand（`userStore`、`deviceStore`、`uiStore`）
- **查询键**: 层级式 `['devices', 'list', params]`
- **Mock 开关**: `VITE_USE_MOCK` 环境变量控制

---

## 9. 开发流程

### 实施步骤

1. **理解** — 研究代码库中的现有模式，阅读相关模块的 handler/service/repository
2. **规划** — 复杂功能分解为 3-5 个阶段，明确每阶段的可交付成果
3. **测试** — 先编写测试用例（后端 `_test.go`，E2E 脚本）
4. **实现** — 最少代码使测试通过
5. **重构** — 在测试通过的前提下清理代码
6. **提交** — 使用 `/commit` 自动审查并提交

### 遇到困难时（3 次尝试规则）

每个问题最多尝试 3 次不同方案，然后**停止**并：

1. 记录失败情况 — 尝试了什么、具体错误、失败原因推测
2. 研究 2-3 个替代方案
3. 质疑基础 — 抽象层次对吗？能拆解为更小问题吗？有更简单的方法吗？
4. 尝试不同角度 — 换一种架构模式？移除抽象而非添加？

---

## 10. 质量关卡

### 每次提交必须

- [ ] 后端 `go build ./...` 编译通过
- [ ] 后端 `go test ./...` 测试通过
- [ ] 前端 `npx tsc --noEmit` 类型检查通过（如涉及前端）
- [ ] 无编译器/linter 警��
- [ ] 新功能包含对应测试
- [ ] 提交消息清晰说明"为什么"

### 绝不要

- 使用 `--no-verify` 绕过提交钩子
- 提交无法编译的代码
- 使用 ORM（GORM 等）替代 Squirrel
- 使用 `any` / `interface{}` 替代明确的接口定义
- 使用 `if carrier == "cmcc"` 硬编码运营商逻辑
- 字符串拼接 SQL 查询
- 忽略编译器警告或 linter 报告
- 做假设 — 用现有代码验证
- 禁用已有测试来让构建通过

### 总是要

- 接口优先 — 核心概念先定义接口再实现
- 错误包装 — `fmt.Errorf("context: %w", err)`
- 运营商适配 — 通过 `Carrier` 接口适配差异
- 增量提交 — 每次提交都是可工作的代码
- 3 次失败后停止并重新评估
- 从现有实现中学习，保持风格一致
- 使用中文交互和中文提交消息

### 测试原则

- 测试行为，不测试实现细节
- 测试成功和失败两种路径
- 测试应当是确定性的（不依赖时间、随机数）
- 绝不禁用失败的测试 — 修复它们

---

## 11. 决策框架

当存在多个有效方法时，按以下优先级选择：

| 优先级 | 维度 | 判断标准 |
|--------|------|---------|
| 1 | **可测试性** | 能轻松写测试吗？ |
| 2 | **可读性** | 6 个月后还能理解吗？ |
| 3 | **一致性** | 符合项目现有模式吗？ |
| 4 | **简单性** | 是最简单的有效方案吗？ |
| 5 | **可逆性** | 以后更改有多困难？ |
| 6 | **可扩展性** | 满足 10 万→100 万基站的扩展要求吗？ |

---

## 12. 数据库

| 类型 | 存储 |
|------|------|
| 业务数据 | PostgreSQL 16 |
| 时序数据 (PM/KPI/告警历史) | TimescaleDB 扩展 |
| 会话/缓存/队列 | Redis 7 |
| 文件 (PM/MR/固件/备份) | MinIO |
| 消息 | NATS JetStream |

**迁移**: `omcgo/migrations/000NNN_description.sql`（goose 格式，版本号严格连续递增，详见 `omcgo/CLAUDE.md` 5.5 节）

---

## 13. 测试

| 类型 | 工具/方法 |
|------|----------|
| 后端单元测试 | `go test ./...` + testify |
| 后端 E2E | `bash omcgo/scripts/e2e_verify.sh http://localhost:8080` (~452 cases) |
| 后端 lint | `golangci-lint run` (`.golangci.yml` 配置) |
| 前端类型检查 | `cd omcmb/webcode && npx tsc --noEmit` |
| 前端 lint | `cd omcmb/webcode && npm run lint` |
| 压力测试 | `omcgo/bin/loadtest` (acs/kpi/mr/all 模式) |

---

## 14. 常用命令

### 后端 (omcgo/)

```bash
cd omcgo
make build          # 编译所有二进制
make test           # 运行测试
make lint           # golangci-lint
make migrate-up     # 执行数据库迁移
go build ./...      # 快速编译检查
```

### 前端 (omcmb/webcode/)

```bash
cd omcmb/webcode
npm run dev         # 开发服务器 (代理到 :8080)
npm run dev:mock    # Mock 模式开发
npm run build       # 生产构建
npx tsc --noEmit    # 类型检查
npm run lint        # ESLint
```

---

## 15. 子系统文档索引

| 文档 | 路径 |
|------|------|
| 后端详细指导 | `omcgo/CLAUDE.md` |
| E2E 测试 Skill | `.claude/commands/e2e.md` |
| 压力测试 Skill | `.claude/commands/acs-stress-test.md` |
| 代码审查 Skill | `.claude/commands/review.md` |
| 智能提交 Skill | `.claude/commands/commit.md` |
| 前后端整合方案 | `前后端整合方案.md` |

---

## 16. 专家角色（Expert Personas）

AI 在处理不同领域的代码变更时，应自动激活对应专家视角进行审查。每个专家角色包含：领域知识、审查清单、决策原则。

### 激活规则

根据修改文件路径自动激活对应专家：

```
internal/acs/**              → TR-069 协议栈专家 + Go 工程专家
internal/config/**           → 电信业务专家 + 数据与存储专家
internal/pm/**               → 电信业务专家 + 数据与存储专家
internal/alarm/**            → 电信业务专家 + 安全合规专家
internal/admin/**            → 安全合规专家 + Go 工程专家
internal/core/**             → 架构专家 + Go 工程专家
migrations/**                → 数据与存储专家
deployments/**               → 运维与可观测性专家
omcmb/webcode/**             → 前端专家
scripts/e2e* | *_test.go     → 测试专家
scripts/loadtest*            → 测试专家 + 运维专家
cmd/**/main.go | router.go   → 架构专家 + 运维专家
新增模块目录                  → 架构专家 + Go 工程专家
```

---

### 16.1 架构专家（Architecture Expert）

**职责**：守护系统整体架构一致性和演进方向。

**核心知识**：
- 模块化单体 — 当前不拆微服务，按功能域内聚
- 三个部署单元：app（:8080）、acs（:7547）、worker（后台）
- 模块间通信：同进程直接调用 + EventBus（NATS JetStream）
- 扩展路线：10 万 → 100 万基站时按功能域渐进拆分

**审查清单**：
- [ ] 新模块是否遵循 handler → service → repository 分层
- [ ] 跨模块依赖是否通过接口而非直接引用
- [ ] 是否引入了循环依赖
- [ ] EventBus 事件主题是否遵循层级命名（`domain.action.detail`）
- [ ] 是否不必要地耦合了 app/acs/worker 三个部署单元
- [ ] 公共组件位置：`internal/core/`（业务基础设施）vs `pkg/`（可独立复用库）

**决策原则**：
- 优先内聚（一个模块完成一个业务域），而非方便（跨模块共享代码）
- 模块间耦合通过 EventBus 事件解耦，不通过直接依赖
- `internal/core/` 放业务无关的基础设施，`pkg/` 放可独立复用的库

---

### 16.2 Go 工程专家（Go Engineering Expert）

**职责**：确保 Go 代码质量、并发安全性和性能。

**核心知识**：
- Go 1.25 特性与最佳实践
- 并发模型：goroutine + channel + sync 原语
- 接口设计：小接口、消费者定义接口、「接受接口，返回具体类型」

**审查清单**：
- [ ] 错误处理：`fmt.Errorf("context: %w", err)`，不裸 panic
- [ ] 并发安全：共享状态有 mutex/channel 保护
- [ ] Context 传递：正确传播 ctx，长操作响应取消
- [ ] 资源释放：defer Close()、连接池归还、goroutine 泄漏检查
- [ ] 接口设计：「接受接口，返回具体类型」
- [ ] 命名：导出 PascalCase，未导出 camelCase，包名小写单数
- [ ] SQL：Squirrel 构建，禁止字符串拼接，禁止 ORM

**性能关注点**：
- ACS 热路径：避免反射、减少内存分配、使用预编译模板
- 批量操作：Squirrel 批量 INSERT/UPDATE
- 连接池：pgxpool 合理配置 MaxConns/MinConns

---

### 16.3 TR-069 协议栈专家（TR-069 Protocol Stack Expert）

**职责**：保障 TR-069/CWMP 协议实现的正确性和合规性。

**核心知识**：
- TR-069 Amendment 6 核心规范
- TR-098/TR-181 数据模型标准
- SOAP 1.1 + CWMP 命名空间
- 会话状态机：Inform → 事务循环 → Empty Response → 结束
- 12 种 RPC 方法：Get/SetParameterValues、Get/SetParameterNames、AddObject、DeleteObject、Download、Upload、Reboot、FactoryReset、GetRPCMethods、Inform、TransferComplete

**审查清单**：
- [ ] SOAP 信封：命名空间声明完整（`cwmp:`, `soap:`, `xsd:`, `xsi:`）
- [ ] cwmpID：请求/响应的 `<cwmp:ID>` 必须匹配
- [ ] 会话状态转换：覆盖所有合法路径，异常路径安全降级
- [ ] Inform 事件码：正确处理 0-BOOTSTRAP / 1-BOOT / 2-PERIODIC / 6-CONNECTION REQUEST
- [ ] 参数路径：`Device.X_VENDOR.` vs `InternetGatewayDevice.` 适配
- [ ] 文件传输：Download/Upload URL 生成、认证、超时
- [ ] 会话超时：Redis TTL 合理（默认 5 分钟）
- [ ] 速率限制：per-device 限流 + 全局准入控制协同
- [ ] Empty HTTP Response（204/空 body）：正确释放 CPE 会话

**协议陷阱**：
- CPE 可能在 Inform 中携带多个事件码，必须全部处理
- 某些 CPE 不支持 Connection Request，需降级为轮询模式
- SOAP Fault 必须保持正确 XML 结构，否则 CPE 可能进入异常循环
- 不同厂商 CPE 参数路径有私有扩展（`X_VENDOR_` 前缀）

---

### 16.4 电信业务专家（Telecom Domain Expert）

**职责**：确保功能实现符合运营商业务需求和行业标准。

**核心知识**：
- 运营商差异：CMCC/CTCC/CUCC 在参数映射、告警规范、性能指标上的差异
- 3GPP 标准：32.435（PM XML）、32.422（MR）、32.111（告警）
- 功能域 F01-F10 业务规则、数据流向、运营商定制点

**各功能域关键业务规则**：

| 功能域 | 核心业务规则 |
|--------|------------|
| F01 南向 | ACS 支持多厂商 CPE 同时在线；会话并发数受限于 Redis 和 goroutine 池 |
| F02 配置 | 三级回退（产品→OUI→运营商默认）；模板支持批量下发和回滚 |
| F03 PM | PM 文件遵循 3GPP 32.435；KPI 多级聚合（设备→站点→区域→网络）|
| F04 告警 | 去重窗口、关联规则、升级策略可配置；活动告警持久化 |
| F05 MR | MRO/MRS/MRE 三种类型；RSRP/RSRQ/SINR 测量值解析 |
| F06 核心 | 设备生命周期（发现→注册→激活→运行→退服）；固件灰度升级 |
| F09 开站 | 零接触部署：CPE 首次 Inform → 模板匹配 → 参数下发 → 激活确认 |

**运营商适配原则**：
- 所有差异通过 `Carrier` 接口适配器实现
- 禁止 `if carrier == "cmcc"` 硬编码
- 新增运营商只需实现 Carrier 接口，不修改核心逻辑

---

### 16.5 数据与存储专家（Data & Storage Expert）

**职责**：保障数据层的正确性、性能和可扩展性。

**核心知识**：
- PostgreSQL 16：业务数据主存储，UUID 主键，JSONB 扩展字段
- TimescaleDB：PM 计数器、KPI 时序、告警历史的超表（hypertable）
- Redis 7：会话（Hash+TTL）、命令队列（Sorted Set）、三级缓存 L1→L2→L3
- NATS JetStream：事件流、异步任务分发
- MinIO：PM/MR 文件、固件包、备份的对象存储

**审查清单**：
- [ ] 迁移：编号连续、up/down 配对、幂等性
- [ ] 索引：高频查询字段有索引，复合索引顺序正确
- [ ] 时序数据使用 TimescaleDB hypertable
- [ ] 连接池：pgxpool MaxConns = CPU 核数 × 2 + 磁盘数
- [ ] Redis 键命名：`domain:entity:{id}` 层级格式
- [ ] Redis TTL：所有缓存键必须设置 TTL
- [ ] 事务边界：写操作在事务内，正确处理回滚
- [ ] N+1 查询：列表接口使用 JOIN 或批量查询
- [ ] 大批量操作：分批处理（BATCH_SIZE），避免长事务锁表

**容量基线（10 万基站）**：
- PM 写入：~50,000 rows/min（高峰）
- 活动告警：~100,000 条常驻 Redis
- ACS 并发：~5,000 同时在线 CPE
- 文件存储：~500 GB/月（PM + MR）

---

### 16.6 前端专家（Frontend Expert）

**职责**：确保前端代码质量、用户体验和前后端一致性。

**核心知识**：
- React 19 + TypeScript 严格模式
- Ant Design 5 组件库规范
- Zustand 状态管理（userStore / deviceStore / uiStore）
- React Query 数据请求与缓存策略
- Axios 拦截器：自动 camelCase ↔ snake_case、Bearer Token 注入

**审查清单**：
- [ ] 类型安全：禁止 `any`，后端响应定义 `BackendXxx` → `mapBackendXxx` → `Xxx`
- [ ] API 服务：一模块一文件（`xxxApi.ts`），导出服务对象
- [ ] Hook 模式：`useMock ? mockService : realApi`
- [ ] 查询键层级：`['domain', 'action', params]`
- [ ] 国际化：用户可见文本通过 `react-intl`
- [ ] 组件复用：优先使用 `src/components/` 现有组件
- [ ] 错误处理：Axios 拦截器统一处理 + 页面级 ErrorBoundary
- [ ] 字段映射：snake_case→camelCase 自动转换覆盖完整

---

### 16.7 测试专家（Testing Expert）

**职责**：保障测试覆盖率、测试质量和测试基础设施可靠性。

**核心知识**：
- 单元测试：Go table-driven tests + testify，Mock 接口实现
- E2E 测试：452 个 curl 用例（`e2e_verify.sh`），覆盖 10 个 Sprint
- CPE 模拟器：Python TR-069 会话模拟（`cpe_simulator.py`）
- 压力测试：递进式加压（200→500→1K→2K→5K 设备）

**审查清单**：
- [ ] 新功能包含成功和失败两条路径的测试
- [ ] 测试确定性：不依赖时间、随机数、外部服务状态
- [ ] Mock 隔离：单元测试用 in-memory Mock，不依赖真实 DB/Redis
- [ ] E2E 覆盖：新端点添加 seed 数据和 E2E 用例
- [ ] 测试命名：`Test_<Function>_<Scenario>`
- [ ] 绝不禁用失败测试 — 修复它们
- [ ] 性能变更需更新压测基线

**测试金字塔**：
```
         /  E2E (452 cases)  \         ← 端到端：验证完整业务流
        / Integration Tests   \        ← 集成：验证模块组合
       /   Unit Tests (84 files) \     ← 单元：验证函数/方法逻辑
      ─────────────────────────────
```

---

### 16.8 安全合规专家（Security & Compliance Expert）

**职责**：确保系统满足运营商级安全要求。

**核心知识**：
- 认证：CPE HTTP Basic/Digest、管理面 JWT Token
- 授权：RBAC 角色权限矩阵，操作级权限控制
- 运营商安全规范：三大运营商各自的网管安全要求

**审查清单**：
- [ ] API 鉴权：管理面端点经过 JWT 验证中间件
- [ ] RBAC：敏感操作（重启、升级、配置下发）校验角色权限
- [ ] 输入验证：HTTP 参数、SOAP XML 验证和清洗
- [ ] SQL 注入：Squirrel 参数化查询
- [ ] 密码安全：bcrypt 哈希，禁止日志打印密码/Token
- [ ] 审计日志：关键操作（登录、配置变更、升级）记录审计
- [ ] 敏感信息：日志/错误不泄露设备密钥、ACS 认证信息
- [ ] 文件上传：验证类型和大小，防路径遍历

---

### 16.9 运维与可观测性专家（Operations & Observability Expert）

**职责**：保障系统可运维性、可观测性和故障恢复能力。

**核心知识**：
- 三个部署单元的启停、健康检查、资源监控
- Prometheus 指标：ACS 会话数、RPC 延迟、队列深度、错误率
- Zap 结构化日志：级别、字段规范、轮转
- OpenTelemetry 链路追踪
- Docker Compose（开发）、Kubernetes（生产）

**审查清单**：
- [ ] 健康检查：新服务暴露 `/health` 端点
- [ ] 指标：关键操作注册 Prometheus 指标（计数器、直方图、仪表盘）
- [ ] 日志：`zap.String/Int/Error` 结构化字段，禁 `fmt.Sprintf` 拼接
- [ ] 错误追踪：携带 `request_id`，可关联链路追踪
- [ ] 优雅关闭：goroutine/监听器响应 SIGTERM
- [ ] 配置外置：环境相关配置不硬编码
- [ ] 容量告警：关键资源设告警阈值（连接池、队列积压、磁盘）
- [ ] 故障恢复：重启后自动恢复状态（Redis 会话、NATS 消费位点）
