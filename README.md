# OMC — 小基站 TR-069/CWMP 无线网管

> **本文档给人读「怎么上手」**；AI 编码助手的工作指导见 [`CLAUDE.md`](./CLAUDE.md)（项目全局上下文与规范）、后端细则见 [`omcgo/CLAUDE.md`](./omcgo/CLAUDE.md)。两者分工：README = 人类入口，CLAUDE.md = AI 干活手册。

---

## ① 这是什么

OMC（Operations, Management and Control）是面向小基站 / 皮基站 / 微基站的**无线操作维护中心**，核心协议是 **TR-069/CWMP**（SOAP/XML over HTTP），同时适配**中国移动（cmcc）/ 中国电信（ctcc）/ 中国联通（cucc）**三大运营商的参数、告警、KPI、开站流程差异，支持 LTE(4G) 与 5G NR(SA) 两种制式。系统按**模块化单体 + 独立 ACS 引擎**架构组织，10 个功能域（F01-F10）43 项子功能，**10 万基站起步、预留 100 万级扩展**。

---

## ② 仓库地图

本仓库是**单一 git 仓库**（根 `goomc/.git`），`omcgo/` 与 `omcmb/` 是源码子目录，不含独立 `.git`，所有 git 操作在根目录执行。

| 目录 | 说明 | 入口文档 |
|------|------|---------|
| `omcgo/` | Go 1.25 后端（模块化单体）：cmd 7 个二进制（app/acs/worker/migrate/omcctl/tools/backup-reencrypt）+ internal 35 个模块 + 字典 XML(`data/`) + 迁移(`migrations/`) | [`omcgo/CLAUDE.md`](./omcgo/CLAUDE.md) |
| `omcmb/` | 前端：业务层 `frontend-core/`（API/Hook/Store/Types/i18n/Mock，vite alias `@core`）+ 主皮肤 `webcode/` + 候选皮肤 `webcode-v2/`·`webcode-v3/` | [`omcmb/webcode/README.md`](./omcmb/webcode/README.md) |
| `deployments/` | 部署清单（docker compose 容器栈 + 监控配置） | [`deployments/docker/README.md`](./deployments/docker/README.md) |
| `run/` | 本地一键启停脚本（裸进程跑法，备用） | `run/scripts/` |
| `docs/` | 设计 / 审查 / 流程 / PRD / Sprint / 运维手册 / 协议合规报告 | （见 §⑥） |
| `.claude/` | Claude Code Skills（`commands/` 内含 ship / commit / review / e2e / acs-stress-test）+ matt-pocock 标准套件 + settings.json | [`CLAUDE.md`](./CLAUDE.md) |

---

## ③ 5 分钟跑起来

**运行态是 docker compose 容器栈**（权威跑法）。所有 Dockerfile 的 build context 是仓库根（`context: ../..`），**必须在 `goomc/` 根目录执行**，用 `-f` 指定 compose 文件：

```bash
# 在仓库根 goomc/ 下执行（首次构建 + 后台启动全部服务）
docker compose -f deployments/docker/docker-compose.yml up -d --build

# 查看状态
docker compose -f deployments/docker/docker-compose.yml ps
```

前置：Docker 24.0+ / Compose v2.20+ / 内存 ≥ 4 GB / 磁盘 ≥ 10 GB。国内首次部署若镜像拉取超时，先按 [`deployments/docker/REGISTRY_SETUP.md`](./deployments/docker/REGISTRY_SETUP.md) 配镜像源。

启动后验证：

| 入口 | URL | 说明 |
|------|-----|------|
| 前端 + REST API（Nginx 网关） | `http://localhost:8081` | SPA 页面 + `/api/*` 代理到 app |
| ACS TR-069 设备接入（Nginx 网关） | `http://localhost:8080` | 代理到 acs:7557 |
| MinIO 控制台 | `http://localhost:9001` | 账号 `minioadmin`/`minioadmin`（dev 默认） |
| Grafana | `http://localhost:3030` | 账号 `admin`/`admin`（dev 默认，宿主 3030 避让 vite :3000） |
| Prometheus | `http://localhost:9090` | 指标存储与查询 |

> 重建 / 重启 / 排障的完整命令见 [`deployments/docker/README.md`](./deployments/docker/README.md) §4、§11、§13。

**前端独立 dev server（热更新开发用）**：

```bash
cd omcmb/webcode && npm install && npm run dev   # :3000，经 /api 代理到后端 :8081
```

**裸进程跑法（备用，未起容器栈时用，不要用于重启容器）**：

```bash
bash run/scripts/start-all.sh    # 依赖 → 后端三进程 → 前端 :3000 → 设计基线 :3001
bash run/scripts/status.sh       # 查看 PID / 端口
bash run/scripts/stop-all.sh     # 停止全部
```

**设计基线对比（UI 还原度并排对比，:3001）** 依赖 `design-baseline` 分支 + `goomc-design` 姊妹 worktree 的一次性 setup，步骤见 [`CLAUDE.md` §3.1](./CLAUDE.md)。

---

## ④ 我该读什么（按角色分流）

| 你是 | 先读 |
|------|------|
| **后端工程师** | [`omcgo/CLAUDE.md`](./omcgo/CLAUDE.md)（架构决策 / TR-069 ACS / 参数字典 / 迁移铁律 / 事件驱动）+ 根 [`CLAUDE.md` §6-§14](./CLAUDE.md) |
| **前端工程师** | [`omcmb/webcode/README.md`](./omcmb/webcode/README.md)（三包结构 + 硬约束 + 命令）+ [`docs/project/frontend-multi-skin-plan-20260422.md`](./docs/project/frontend-multi-skin-plan-20260422.md) + 根 [`CLAUDE.md` §8.3 / §16.6](./CLAUDE.md)。业务层一律写 `frontend-core/`，UI 壳只放页面/组件 |
| **协议 / ACS** | [`docs/tr069-protocol-compliance-report.md`](./docs/tr069-protocol-compliance-report.md)、[`docs/acs-verification-plan.md`](./docs/acs-verification-plan.md)，代码在 `omcgo/internal/acs/`；术语见 [`CONTEXT.md`](./CONTEXT.md) |
| **部署 / 运维** | [`deployments/docker/README.md`](./deployments/docker/README.md)（服务清单 / 端口 / 环境变量 / 排障）+ `docs/operations/`、`docs/runbook/` |
| **术语 / 领域语言** | [`CONTEXT.md`](./CONTEXT.md)（统一语言术语表，探索代码前先读）；功能域 F01-F10 速览见根 [`CLAUDE.md` §6](./CLAUDE.md) |

---

## ⑤ 怎么贡献

- **任务源**：活任务走 **GitHub Issues**（`github.com/569423176-sketch/goomc`，`gh` CLI），约定见 [`docs/agents/issue-tracker.md`](./docs/agents/issue-tracker.md)。`docs/project/backlog.md` 为历史审计归档，不再新增任务。
- **开发流程**：一键 `/ship` 贯穿全流程（对齐 → 立项 → 切片 → 分诊 → 实现 → 验证 → 审查 → 提交 → 收尾），它只调度、按阶段委派 matt-pocock 标准套件（grill-with-docs / to-prd / to-issues / triage / tdd…）与项目自有 review / commit / e2e，Skill 在 [`.claude/commands/ship.md`](./.claude/commands/ship.md)。小改 / hotfix 走快速通道。
- **提交规范**：Conventional Commits + 中文描述，scope 对应功能域 / 模块，git 操作边界（不自动 push、`--force`/`--no-verify` 硬禁）见根 [`CLAUDE.md` §8.1](./CLAUDE.md)。提交前过质量关卡：后端 `go build ./... && go test ./...`，前端 `cd omcmb/webcode && npm run typecheck`。

---

## ⑥ 文档地图

**完整文档索引（活文档 vs 归档区分）见 [`docs/README.md`](./docs/README.md)。** 下表为高频主题入口：

| 主题 | 位置 |
|------|------|
| 📑 文档总索引 | [`docs/README.md`](./docs/README.md) |
| 统一语言术语表 | [`CONTEXT.md`](./CONTEXT.md) |
| 设计方案 | `docs/design/` |
| 项目管理（PRD / Sprint / milestone / backlog / risk-register / dod / release-gate） | `docs/project/` |
| 运维与可观测性 | `docs/operations/`、`docs/runbook/` |
| 后端参考（迁移踩坑 / 三库导入历史 / ADR） | `docs/ref/`、`docs/adr/` |
| 协议合规与验证报告 | `docs/tr069-protocol-compliance-report.md`、`docs/acs-verification-plan.md` |
| 消息队列全流程 | `docs/消息队列全流程流转说明书.md` |
| AI 协作约定（issue tracker / triage / domain） | `docs/agents/` |
| 审查专家角色 | `docs/expert-personas.md` + 根 [`CLAUDE.md` §16](./CLAUDE.md) |
