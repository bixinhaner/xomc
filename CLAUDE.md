# CLAUDE.md — OMC 项目全局指导

> 本文件为 AI 编码助手提供项目全局上下文（每次会话加载，保持精简）。
> - 后端详细指导 → `omcgo/CLAUDE.md`
> - 人类上手 / 仓库地图 → `README.md`
> - 文档总索引（活文档 vs 归档）→ `docs/README.md`
> - 领域术语表 → `CONTEXT.md`
> - 专家角色（审查视角）→ `docs/expert-personas.md`

---

## 1. 项目简介

**OMC**（Operations, Management and Control）是面向小基站/皮基站/微基站的无线操作维护中心系统。

| 维度 | 说明 |
|------|------|
| 核心协议 | TR069/CWMP（SOAP/XML over HTTP） |
| 运营商 | 中国移动（cmcc）、中国电信（ctcc）、中国联通（cucc） |
| 制式 | LTE (4G)、5G NR (SA) |
| 规模 | 10 万基站起步，预留 100 万级扩展 |
| 功能域 | 10 个域（F01-F10） |

---

## 2. 开发理念

- **商用级品质** — 面向运营商的商用网管系统，不偷工减料
- **增量胜过激进** — 小步修改，每次编译通过并通过测试
- **从现有代码学习** — 实施前先研究项目已有模式与约定
- **实用胜过教条** — 适应项目现实，不为理论纯洁牺牲交付
- **清晰胜过巧妙** — 选平凡明显的方案；需要额外注释才能解释，说明代码太复杂
- **单一职责 + 避免过早抽象** — 三处相似才考虑提取公共逻辑

---

## 3. 仓库结构

```
omc/                                # 根仓库（单一 git）
├── CLAUDE.md / README.md / CONTEXT.md   # AI 指导 / 人类上手 / 术语表
├── omcgo/                          # Go 后端（子目录，随主仓库追踪）
│   ├── CLAUDE.md                   #   后端详细指导
│   ├── cmd/                        #   入口：app / acs / worker / migrate / omcctl / tools / backup-reencrypt
│   ├── internal/                   #   业务功能域 + 跨域基础设施（35 个模块，清单见 §6）
│   │   └── core/                   #     进程级基础设施（含 core/carrier 运营商适配）
│   ├── data/                       #   活跃字典 XML/JSON 资产（param-mappings / indicator-library / alarm-definitions / quicksettings / mml-catalog），dictloader 启动期加载
│   ├── datamodels/                 #   仅剩 mml-catalog/（JSON）+ templates/；TR069 参数已迁 data/ 与 product/
│   ├── migrations/                 #   数据库迁移（goose；2026-05-31 consolidated baseline，见 migrations/README.md）
│   ├── scripts/                    #   E2E、压测、CPE 模拟器、诊断
│   └── 规范/                        #   三大运营商技术规范原件（运营商规范的权威源）
├── omcmb/                          # 前端（子目录，随主仓库追踪）
│   ├── frontend-core/              #   共享业务层（services/api · hooks/api · store · types · i18n · mock），vite alias `@core`
│   ├── webcode/                    #   主皮肤 UI 壳（pages/components/router/providers）
│   ├── webcode-v2/ webcode-v3/     #   多皮肤候选（详见 docs/project/frontend-multi-skin-plan-20260422.md）
│   └── （OMC/ jx/ 广研院/ original-omc/ design/ analysis/ — 历史/参考资产，非构建产物，见 docs/README.md）
├── deployments/                    # 部署清单（docker compose、监控）；deployments/docker/README.md 有端口表
├── run/                            # 本地一键启停脚本（裸进程跑法）
├── .claude/                        # Skills（commands/）+ settings.json + skills-marketplace/
└── docs/                           # 设计 / 评审 / 流程 / PRD / Runbook（总索引见 docs/README.md）
```

**仓库组成**：**单一 git 仓库**（根 `goomc/.git`），`omcgo/` 和 `omcmb/` 是源码子目录，**不含独立 `.git`**。所有 git 操作在 `goomc/` 根目录执行。

**前端三包关系**：业务层（API / Hook / Store / Types / i18n / Mock）全部在 `omcmb/frontend-core/`，三个 UI 包通过 `@core → ../frontend-core/src` 共享同一份业务层。日常以 `webcode/` 为主皮肤；**新增 API / Hook / Store / Types 必须写在 `frontend-core/`**，UI 壳里只放页面与组件。

### 3.1 设计基线对比环境（worktree）

本仓库用 `design-baseline` 分支 + `goomc-design` 姊妹 worktree 做 UI 还原度对比。**初次 clone / pull 后一次性 setup**（否则 `start-all.sh` 跳过基线启动）：

```bash
cd <path>/omc/goomc
git fetch origin design-baseline:design-baseline     # 拉取基线分支到本地
git worktree add ../goomc-design design-baseline     # 建立姊妹 worktree
ln -s "$(pwd)/omcmb/webcode/node_modules" \
      ../goomc-design/omcmb/webcode/node_modules    # 共享 node_modules
```

之后 `bash run/scripts/restart-all.sh` 同时启动 `:3000`（当前开发版）与 `:3001`（设计基线），浏览器并排对比。基线由维护者把 `design-baseline` ff 到新 commit 后 push，其他人 pull 对应 worktree 同步。

---

## 4. 技术栈总览

### 后端 (omcgo/)

| 组件 | 选型 |
|------|------|
| 语言 | Go 1.25 |
| HTTP（管理面）| Gin |
| HTTP（ACS）| net/http stdlib |
| RPC | gRPC（ACS ↔ App）|
| SQL 构建 | Squirrel（不使用 ORM）|
| DB 驱动 | pgx/v5 + pgxpool |
| 缓存 | Redis（go-redis/v9）|
| 消息 | NATS JetStream |
| 对象存储 | MinIO（S3 兼容）|
| 日志 | Zap |
| 指标 | Prometheus |
| 可观测性 | OpenTelemetry Collector → Tempo（trace）/ Loki（log）/ Prometheus（metric）；trace↔log 经 `logger.L(ctx)` 关联。落地细节见 `docs/operations/OMC可观测性使用手册.md` |

### 前端 (omcmb/ — frontend-core 业务层 + webcode UI 壳)

| 组件 | 选型 |
|------|------|
| 框架 | React 19 + TypeScript（严格模式）|
| 构建 | Vite 7 |
| UI 库 | Ant Design 5 + @ant-design/pro-components |
| 状态管理 | Zustand 5（`frontend-core/src/store/`）|
| 数据请求 | React Query v5（`frontend-core/src/hooks/api/`）|
| HTTP 客户端 | Axios（`frontend-core/src/services/http.ts`，自动 camelCase ↔ snake_case）|
| 图表 / 地图 | ECharts 6 + echarts-for-react / OpenLayers |
| 国际化 / 路由 | react-intl 8（`frontend-core/src/i18n/`）/ react-router-dom 6 |
| 测试 | Vitest + Testing Library + Playwright |

---

## 5. 架构概览

**模块化单体 + 独立 ACS 引擎**：不使用微服务。当前保持模块化单体，100 万规模时可按功能域渐进拆分。

| 部署单元 | 用途 | dev 端口（`cmd/*/etc/config.dev.yaml`）| 生产 |
|------|------|----|----|
| `omcgo-app` | 主应用（F02-F10）REST + gRPC | HTTP `:8081` / TLS `:8444` / gRPC `:50051` / metrics `:9091` | 同左 |
| `omcgo-acs` | TR069 ACS 引擎（独立进程，水平扩展）| HTTP `:7557` / TLS `:7558` / STUN `:3478` / metrics `:9090`（容器内）| CWMP 标准 `:7547` |
| `omcgo-worker` | 后台进程（PM/MR 文件处理、KPI 计算）| metrics `:9092` | 同左 |

> 端口以 `config.dev.yaml` 为准。容器栈宿主映射（含 nginx `:8080`/`:8081`、acs metrics 宿主映射 `:9095`）见 §14 与 `deployments/docker/docker-compose.yml`。
> **前端 SPA**：Vite dev server `:3000` 经 `/api` 代理到 App `:8081`；设计基线跑 `:3001`（§3.1）。

---

## 6. 功能域速览

| 编号 | 功能域 | 核心职责 | 后端模块 |
|------|--------|---------|---------|
| F01 | 南向接口 (TR069) | ACS 引擎，SOAP/XML 协议处理；TR069 报文跟踪 | `acs/` `trace/` |
| F02 | 数据模型与配置 | ParamModel 字典（Translator 双向翻译 + Intersect 写 discovered_param_mappings）+ 产品装配件（ProductRegistry 路由 productClass）+ 配置模板/基线/审计/备份；参数快速设置、单设备 sweep（T-0098 后旧 datamodel 包已下线）| `product/` `config/{parammodel,template,baseline,audit,backup}/` `quicksettings/` `devsweep/` |
| F03 | 性能管理 (PM/KPI) | 计数器采集，KPI 多级聚合，时序存储 | `pm/` |
| F04 | 告警管理 | 告警接收、去重、关联、生命周期；设备事件日志 | `alarm/` `eventlog/` |
| F05 | 测量报告 (MR) | MRO/MRS/MRE 文件采集与解析 | `mr/` |
| F06 | OMC-R 核心 | 设备/用户/拓扑/固件/备份/仪表盘/运维/报表/MML/文件/日志/许可；批量下载、统一文件任务、基站日志、重启记录 | `device/` `admin/` `topology/` `software/` `backup/` `dashboard/` `ops/` `report/` `mml/` `filemanager/` `syslog/` `license/` `bundle/` `ufte/` `stationlog/` `rebootrecord/` |
| F07 | 网元直连 | 网元与网管直连通道 | `nedirect/` |
| F08 | 北向/OSS 接口 | 向上游 OSS 开放数据 | `northbound/` |
| F09 | 自动开站 | 设备自动发现、模板匹配、配置下发 | `provision/` |
| F10 | 互操作测试 | 设备联调与一致性验证 | `interop/` |

### 跨域基础设施模块（不属 F0X，承载多个功能域）

| 模块 | 用途 |
|------|------|
| `internal/task/` | 统一任务队列（Redis 双写 PG），CWMP ID ↔ Task 映射、reboot closer、completion router |
| `internal/notification/` | 通知中心（ExpeditedEvent 实时告警、设备列表批量同步、邮件/SMS/Webhook）|
| `internal/transfer/` | 文件传输桥（Download/Upload 与 ACS / app 的中介）|
| `internal/events/` | 事件 hub / handler / store（基于 `internal/core/event` 的 EventBus 上层）|
| `internal/core/` | 进程级基础设施：`carrier`（运营商适配）/ appconfig / asyncjob / components / dictloader / errors / event / health / middleware / model / redact / reliability / response / storage / tracing / utils |

> **运营商适配在 `internal/core/carrier/`**（接口 `carrier.go` + registry + cmcc/ctcc/cucc 适配器）。所有运营商差异在此封装，禁止 `if carrier == "cmcc"` 硬编码。

---

## 7. 关键文件路径

> 计数（错误码数 / 文件数 / 断言数等）一律**现查**（`ls` / `grep -c` / 跑脚本看输出），不在本文件硬编码，避免持续腐烂。

### 后端

| 路径 | 说明 |
|------|------|
| `omcgo/cmd/app/main.go` · `cmd/app/provider/` | 主应用入口；路由注册 + DI 容器（`router.go`/`container.go`/`modules.go`/`bootstrap.go`）|
| `omcgo/internal/` | 业务模块（功能域）+ 跨域基础设施（§6）|
| `omcgo/internal/core/` | 进程级基础设施（含 `core/carrier`）|
| `omcgo/global/errors.go` | 全局错误码（数量现查 `grep -c ErrCode`）|
| `omcgo/migrations/` | 数据库迁移（goose；consolidated baseline，详见 `omcgo/migrations/README.md` 与 `omcgo/CLAUDE.md`）|
| `omcgo/scripts/e2e_verify.sh` | E2E 脚本（默认 `:8081`；断言数以运行输出 `Results:` 为准）|
| `omcgo/scripts/cpe_simulator.py` | CPE TR-069 模拟器 |
| `omcgo/CLAUDE.md` | **后端详细指导** |

### 前端（业务层 vs UI 壳分离）

API / Hook / Store / Types / i18n / Mock 全在 `omcmb/frontend-core/`（多皮肤共享）；页面 / 组件 / 路由在 `omcmb/webcode/`（及 v2/v3）。import 形如 `import { authApi } from '@core/services/api/authApi'`。

| 路径 | 说明 |
|------|------|
| `frontend-core/src/services/api/*.ts` | API 服务（一模块一文件）|
| `frontend-core/src/services/http.ts` | Axios 客户端（拦截器、camelCase ↔ snake_case、Token 续期）|
| `frontend-core/src/services/apiSwitch.ts` | Mock / Real 切换（`useMock`，受 `VITE_USE_MOCK` 控制）|
| `frontend-core/src/hooks/api/*.ts` · `store/*.ts` · `types/` · `i18n/` · `mock/` | Hook / Zustand store / 类型 / 语料 / Mock |
| `webcode/src/{pages,components,router,providers}/` · `vite.config.ts` | 主皮肤页面 / 组件 / 路由 / Provider / Vite 配置（`@core` alias + `:8081` 代理）|

---

## 8. 开发规范

### 8.1 Git 工作流

**提交格式**：Conventional Commits，中文描述 —— `<type>(<scope>): <中文简要描述>` + body + footer。

**Type**：`feat` `fix` `refactor` `docs` `test` `chore` `perf` `build` `ci` `style`

**Scope**（与模块对应）：`acs` `config` `pm` `alarm` `mr` `device` `admin` `topology` `software` `backup` `dashboard` `ops` `report` `mml` `filemanager` `syslog` `license` `product` `trace` `bundle` `ufte` `quicksettings` `devsweep` `stationlog` `eventlog` `rebootrecord` `nedirect` `northbound` `provision` `interop` `carrier` `notification` `task` `transfer` `events` `core` `migration` `frontend` `frontend-core` `components` `api` `deploy` `docs`

**Claude AI 的 git 操作边界**（软约束 = 本节行为；硬约束 = `.claude/settings.json` permissions）：
1. 完成任务后默认**只 commit 不 push**，回复中注明"本地已提交（hash xxx），未推送"。
2. `git fetch/pull/push` **严禁自主触发**，仅在用户明确说"推送/同步远端"时执行。
3. `git pull` 与 `git push` **严禁同一条命令**，必须分步（先 `pull --rebase` 处理干净再 push）。
4. `--force` 推送 / `--no-verify` / `rm -rf` / `git config --global` —— **永不执行**（settings.json 已 deny）；`reset --hard` / `branch -D` / `clean -f` 需用户明确指令。

完整软/硬约束矩阵见 `docs/agents/git-boundaries.md`（如缺失，settings.json 为唯一硬约束真值源）。

### 8.2 Go 后端规范（完整见 `omcgo/CLAUDE.md`）

接口优先（核心概念先定义接口）· 错误包装 `fmt.Errorf("context: %w", err)`，禁裸 panic · SQL 用 Squirrel + pgx，禁 ORM、禁字符串拼接 · 运营商差异走 `Carrier` 接口，禁 `if carrier == "cmcc"` · 模块分层 `handler → service → pg_repository + model` · DB 主键 UUID，表含 `created_at`/`updated_at`。

### 8.3 React/TypeScript 前端规范

代码归属：API/Hook/Store/Types/i18n/Mock → `frontend-core/`，页面/组件/路由 → `webcode*/` · 禁 `any`（后端响应 `BackendXxx` → `mapBackendXxx` → `Xxx`）· 跨包引用走 `@core/...`，禁相对路径越级 · 用户可见文本走 `react-intl` · 改 `frontend-core/` 的类型/Mock/Store 形态时评估对 v2/v3 三皮肤的编译影响。

---

## 9. 开发流程

> **任务源（活）= GitHub Issues**（`github.com/569423176-sketch/goomc`，`gh` CLI）。新需求/缺陷登记到 GitHub Issues，经 triage（`needs-triage` → `ready-for-agent` 等）后拉起开发。
> `docs/project/backlog.md` 已**冻结为历史审计归档**（保留 dev-pipeline 的 closing-evidence 追溯链，不再新增任务）。
> ⚠️ 迁移进行中：`/dev-pipeline` 与 BaiBM `issue:*` skill 仍部分依赖 backlog.md，重接到 GitHub Issues 是在途项，详见 `docs/agents/issue-tracker.md`。

**实施步骤**（流水线 S2–S6 操作性摘要，小改/hotfix 快速通道）：
1. **理解** — 研究现有模式，读相关模块 handler/service/repository
2. **规划** — 复杂功能分解为 3-5 阶段，明确每阶段可交付物
3. **测试** — 先写测试（后端 `_test.go`、E2E 脚本）
4. **实现** — 最少代码使测试通过
5. **重构** — 测试通过前提下清理
6. **提交** — `/commit` 自动审查并提交

**3 次尝试规则**：每个问题最多尝试 3 次不同方案，然后停止 → 记录失败 → 研究 2-3 个替代 → 质疑抽象层次 → 换角度。

---

## 10. 质量关卡

**每次提交必须**：
- [ ] 后端 `go build ./...` 编译通过、`go test ./...` 测试通过
- [ ] 前端（如涉及）`cd omcmb/webcode && npm run typecheck` 通过
- [ ] 无编译器 / linter 警告
- [ ] 新功能含成功 + 失败两条路径的测试
- [ ] 提交消息说清"为什么"

**绝不**：`--no-verify` 绕过钩子 · 提交不编译的代码 · 用 ORM 替代 Squirrel · 用 `any`/`interface{}` 替代明确接口 · `if carrier == "cmcc"` 硬编码 · 字符串拼接 SQL · 忽略警告 · 禁用失败测试。

**总是**：接口优先 · 错误包装 · Carrier 接口适配运营商差异 · 增量提交（每次都是可工作代码）· 从现有实现学习保持风格 · 中文交互与提交消息。

---

## 11. 决策框架

存在多个有效方法时按优先级选择：

| 优先级 | 维度 | 判断标准 |
|--------|------|---------|
| 1 | 可测试性 | 能轻松写测试吗？ |
| 2 | 可读性 | 6 个月后还能理解吗？ |
| 3 | 一致性 | 符合项目现有模式吗？ |
| 4 | 简单性 | 是最简单的有效方案吗？ |
| 5 | 可逆性 | 以后更改有多难？ |
| 6 | 可扩展性 | 满足 10 万→100 万基站扩展吗？ |

---

## 12. 数据库

| 类型 | 存储 |
|------|------|
| 业务数据 | PostgreSQL 16 |
| 时序（PM/KPI/告警历史）| TimescaleDB 扩展 |
| 会话/缓存/队列 | Redis 7 |
| 文件（PM/MR/固件/备份）| MinIO |
| 消息 | NATS JetStream |

**迁移**：`omcgo/migrations/`（goose；2026-05-31 已做 consolidated baseline，schema `000001` + 增量，非简单连续递增）。唯一事实源 = `migrations/README.md` + `omcgo/CLAUDE.md`。

---

## 13. 测试

| 类型 | 工具 / 方法 |
|------|----------|
| 后端单元 | `go test ./...` + testify |
| 后端 E2E | `bash omcgo/scripts/e2e_verify.sh http://localhost:8081` |
| 后端 lint / 集成 | `golangci-lint run` / `scripts/integration_test.sh`、`test_acs_inform.sh` |
| CPE 模拟 | `python3 omcgo/scripts/cpe_simulator.py` |
| 前端 | `cd omcmb/webcode && npm run {typecheck,lint,test,test:e2e}` |
| 压力测试 | `omcgo/bin/loadtest`（acs/kpi/mr/all 模式）|

> 测试/断言数量随用例补齐变化，以 CI / 脚本实际输出为准，不在文档硬编码。

---

## 14. 常用命令

### 服务重启规则（CRITICAL — 统一用 docker compose）

> **重启/重建后端服务（app/acs/worker）一律走 docker compose，禁止用 `run/scripts/restart-all.sh`。** 运行态是 docker compose 编排的容器栈；`run/scripts/*` 是裸进程跑法，与容器栈并存会端口冲突、状态不一致。
> **所有 compose 命令在仓库根 `goomc/` 下执行**（build context 是 `../..` 仓库根，且读根目录 `.env`），用 `-f` 指定 compose 文件。完整命令见 `deployments/docker/README.md`。

```bash
# 在仓库根 goomc/ 下执行
export COMPOSE="docker compose -f deployments/docker/docker-compose.yml"
$COMPOSE up -d --build app worker     # 改了 Go 代码后重建并重启（最常用）
$COMPOSE up -d --build acs            # ACS 同理
$COMPOSE up -d --build migrate-seed && $COMPOSE up migrate-seed   # 新增迁移后
$COMPOSE restart app worker           # 仅重启（无代码变更）
$COMPOSE ps                           # 状态；logs -f --tail=100 app worker；down 停止
```

宿主端口：app metrics `:9091`、acs `:7557`/metrics `:9095`、worker metrics `:9092`、web(nginx) `:8080`/`:8081`、Grafana `:3030`、Prometheus `:9090`。

### 一键启停（裸进程跑法，**不要用于重启容器化服务**）

```bash
bash run/scripts/{start-all,restart-all,status,stop-all}.sh
```

### 后端 / 前端

```bash
cd omcgo && make {build,test,lint,migrate-up}      # 或 go build ./...；bash scripts/check-migrations.sh
cd omcmb/webcode && npm run {dev,dev:mock,build,typecheck,lint,test,test:e2e}
```

---

## 15. 子系统文档索引

> 完整索引（含活文档 vs 归档区分）见 `docs/README.md`。下表为高频入口：

| 文档 | 路径 |
|------|------|
| 后端详细指导 | `omcgo/CLAUDE.md` |
| 专家角色（审查视角）| `docs/expert-personas.md` |
| 领域术语表 | `CONTEXT.md` |
| 开发流水线 Skill / 设计 | `.claude/commands/dev-pipeline.md` · `docs/project/dev-pipeline-design-20260420.md` |
| 任务源（活）| GitHub Issues `github.com/569423176-sketch/goomc`（详见 `docs/agents/issue-tracker.md`）|
| 历史任务归档 | `docs/project/backlog.md`（已冻结）|
| 其它 Skill | `.claude/commands/{e2e,acs-stress-test,review,commit}.md` |
| 前端多皮肤架构 | `docs/project/frontend-multi-skin-plan-20260422.md` |
| 消息队列全流程 | `docs/消息队列全流程流转说明书.md` |
| DoD / Release Gate / 风险登记册 | `docs/project/{dod,release-gate,risk-register}.md` |
| 部署 / 可观测性 / Runbook | `deployments/docker/README.md` · `docs/operations/` · `docs/runbook/` |

---

## 16. 专家角色（Expert Personas）

AI 处理不同领域代码时应自动激活对应专家视角审查。**完整的领域知识 / 审查清单 / 决策原则 + 文件路径→专家激活映射表，见 `docs/expert-personas.md`。**

速记：领域专家（架构 / Go 工程 / TR-069 协议 / 电信业务 / 数据存储 / 前端 / 测试 / 安全合规 / 运维可观测）按**修改文件路径**激活；流程专家（PM / PgM / QA 发布经理）按**工作阶段/用户意图**激活；两类叠加使用。

---

## Agent skills（matt-pocock 工程技能套件）

> 与本仓库自有 `/dev-pipeline` + Skill 协同。约定文件在 `docs/agents/`。

- **Issue tracker（任务源）**：Issues / PRDs 走 GitHub Issues（`github.com/569423176-sketch/goomc`，`gh` CLI）——已采纳为活任务源（§9）。详见 `docs/agents/issue-tracker.md`。
- **Triage labels**：5 个标准 triage 角色标签（`needs-triage` / `needs-info` / `ready-for-agent` / `ready-for-human` / `wontfix`）。详见 `docs/agents/triage-labels.md`。
- **Domain docs（单上下文）**：根级 `CONTEXT.md`（术语表）+ `docs/adr/`（决策记录）。详见 `docs/agents/domain.md`。
