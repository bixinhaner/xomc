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

## 2. 仓库结构

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

## 3. 技术栈总览

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

## 4. 架构概览

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

## 5. 功能域速览

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

## 6. 关键文件路径

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

## 7. 开发规范

### 7.1 Git 工作流

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

### 7.2 Go 后端规范

> 完整规范详见 `omcgo/CLAUDE.md`

- **命名**: 导出 `PascalCase`，未导出 `camelCase`，包名小写单数
- **接口优先**: 核心概念均定义接口（`DeviceRepository`、`Carrier`、`EventBus`）
- **错误处理**: `fmt.Errorf("context: %w", err)`，禁止裸 panic
- **SQL**: Squirrel 构建 + pgx 执行，禁止 ORM，禁止字符串拼接
- **运营商**: 全部通过 `Carrier` 接口适配器，禁止 `if carrier == "cmcc"` 硬编码
- **模块结构**: `handler.go` → `service.go` → `pg_repository.go` + `model.go`
- **DB 主键**: UUID (`gen_random_uuid()`)，所有表含 `created_at`/`updated_at`

### 7.3 React/TypeScript 前端规范

- **API 服务模式**: 每模块一个 `xxxApi.ts`，导出服务对象
- **Hook 模式**: 每模块一个 `useXxx.ts`，内部使用 `useMock ? mockService : realApi`
- **HTTP 客户端**: Axios 拦截器自动 camelCase ↔ snake_case 转换，Bearer token 注入
- **类型安全**: 禁止 `any`，Backend 响应用 `BackendXxx` 接口 → `mapBackendXxx` 转换 → 前端 `Xxx` 接口
- **状态管理**: Zustand（`userStore`、`deviceStore`、`uiStore`）
- **查询键**: 层级式 `['devices', 'list', params]`
- **Mock 开关**: `VITE_USE_MOCK` 环境变量控制

---

## 8. 数据库

| 类型 | 存储 |
|------|------|
| 业务数据 | PostgreSQL 16 |
| 时序数据 (PM/KPI/告警历史) | TimescaleDB 扩展 |
| 会话/缓存/队列 | Redis 7 |
| 文件 (PM/MR/固件/备份) | MinIO |
| 消息 | NATS JetStream |

**迁移**: `omcgo/migrations/000NNN_description.up.sql` / `.down.sql`

---

## 9. 测试

| 类型 | 工具/方法 |
|------|----------|
| 后端单元测试 | `go test ./...` + testify |
| 后端 E2E | `bash omcgo/scripts/e2e_verify.sh http://localhost:8080` (~452 cases) |
| 后端 lint | `golangci-lint run` (`.golangci.yml` 配置) |
| 前端类型检查 | `cd omcmb/webcode && npx tsc --noEmit` |
| 前端 lint | `cd omcmb/webcode && npm run lint` |
| 压力测试 | `omcgo/bin/loadtest` (acs/kpi/mr/all 模式) |

---

## 10. 常用命令

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

## 11. 子系统文档索引

| 文档 | 路径 |
|------|------|
| 后端详细指导 | `omcgo/CLAUDE.md` |
| E2E 测试 Skill | `.claude/commands/e2e.md` |
| 压力测试 Skill | `.claude/commands/acs-stress-test.md` |
| 代码审查 Skill | `.claude/commands/review.md` |
| 智能提交 Skill | `.claude/commands/commit.md` |
| 前后端整合方案 | `前后端整合方案.md` |
