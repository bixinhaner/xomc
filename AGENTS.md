# AGENTS.md — OMC 项目根级指导

本文件每次 AI 会话都会加载，只保留高频约束和导航。详细规则不要写在这里，按需打开对应文档。

## 项目定位

OMC（Operations, Management and Control）是面向小基站 / 皮基站 / 微基站的无线操作维护中心系统，核心协议是 TR069/CWMP，面向运营商级规模和商用品质。

## 仓库边界

- 仓库根目录是 `goomc/`，所有 git 操作在根目录执行。
- `omcgo/` 是 Go 后端源码目录，不是独立 git 仓库。
- `omcmb/` 是前端源码目录，不是独立 git 仓库。
- `AGENTS.md`、`.codex/`、`.agents/` 等本机 AI 配置默认视为本地文件，不加入 git、不提交、不进 PR，除非用户明确要求。

## 重要入口

- 后端详细规则：`omcgo/AGENTS.md`
- 人类上手说明：`README.md`
- 文档总索引：`docs/README.md`
- 领域术语：`CONTEXT.md`
- Docker / compose 部署：`deployments/docker/README.md`
- 专家审查视角：`docs/expert-personas.md`
- Codex 本地插件市场：`.agents/plugins/marketplace.json`

## 项目内 Codex Skills

本仓 vendored 了一组项目本地 Codex skills：

```text
.agents/plugins/plugins/mattpocock-skills/skills/
```

如果用户显式提到 `$diagnose`、`$tdd`、`$triage`、`$to-prd`、`$to-issues`、`$grill-me`、`$grill-with-docs`、`$zoom-out`、`$handoff`，或请求明显匹配这些 skill 的用途，即使本机尚未注册 marketplace，也应直接读取对应目录下的 `SKILL.md` 并按其流程执行。

团队成员如希望这些 skills 出现在 Codex 插件/技能 UI 中，可在仓库根目录执行：

```bash
codex plugin marketplace add .agents/plugins
```

## 代码归属

### 后端

- 后端入口和模块在 `omcgo/`。
- 模块分层遵循 `handler -> service -> repository/model`。
- SQL 使用 Squirrel + pgx，禁止 ORM 和字符串拼接 SQL。
- 错误要包装上下文：`fmt.Errorf("context: %w", err)`。
- 运营商差异走 `internal/core/carrier/`，禁止散落 `if carrier == "cmcc"` 之类硬编码。

### 前端

- 共享业务层放在 `omcmb/frontend-core/`：API、Hook、Store、Types、i18n、Mock。
- UI 壳放在 `omcmb/webcode*`：页面、组件、路由、Provider。
- 跨包引用走 `@core/...`，不要用相对路径越级访问 shared code。
- 用户可见文案走国际化，不要直接散落硬编码。

## 前端三皮肤铁律

前端是单底层 `frontend-core` + 三套皮肤：

- `webcode`：v1 主皮肤，唯一标准。
- `webcode-v2`：第二套皮肤。
- `webcode-v3`：第三套皮肤。

涉及前端 bug 或需求时：

- 优先改 `frontend-core/`，让三皮肤共享受益。
- 如果改页面、路由、菜单或交互，三套皮肤都要补齐。
- v2/v3 的可路由 path 集合和可见菜单项集合必须严格等于 v1。
- 浏览器操作或验收时，不管源码显示什么，都必须以浏览器中的实际页面为准；先读取运行中 DOM，确认目标控件、按钮和保存行为后再操作。
- 验收至少跑 `cd omcmb && npm run skin-parity` 和对应 typecheck。

## 运行与部署边界

- 重启 / 重建容器化后端服务优先走 `docker compose -f deployments/docker/docker-compose.yml`。
- 不要用 `run/scripts/restart-all.sh` 去重启容器化服务；`run/scripts/*` 是裸进程跑法，容易和 compose 栈冲突。
- 端口、容器名、健康检查以 `deployments/docker/README.md` 和 compose 文件为准，现查不要硬记。

## 本机权限经验

以下场景通常会被 sandbox 限制；需要执行时默认直接按权限规则提权，不先故意试错再重试。

- GitHub 相关 `gh issue` / `gh pr`、`git push` 需要网络，直接提权执行。
- Docker socket、Compose 部署、可见 Chrome / CDP、访问本机部署端口通常需要提权，直接提权执行。
- `go test ./...` 中涉及 `miniredis` / `httptest` 本地监听时，普通 sandbox 可能报 `bind: operation not permitted`；直接提权复跑以区分环境问题和真实测试失败。

## 常用验证

按改动范围选择验证，不能假装跑过。

- 后端：`cd omcgo && go build ./... && go test ./...`
- 前端：`cd omcmb && npm run typecheck`
- 前端三皮肤一致性：`cd omcmb && npm run skin-parity`
- Docker 状态：`docker compose -f deployments/docker/docker-compose.yml ps`

如果验证因环境、依赖、网络或权限失败，回复中明确说明失败命令和原因。

## Git 边界

- 不主动 `git fetch` / `git pull` / `git push`，除非用户明确要求。
- 不执行 `git reset --hard`、`git clean -f`、强推、`--no-verify`，除非用户明确要求且风险已说明。
- 不回滚用户已有改动；遇到无关脏文件直接忽略。
- 提交信息使用 Conventional Commits，中文描述。

## 工作方式

- 先读现有代码和文档，再动手。
- 小步修改，尽量保持每一步可验证。
- 保持项目既有模式，避免无关重构。
- 涉及高风险共享逻辑、跨模块契约或用户可见流程时补测试。
- 复杂问题最多尝试三种方案；仍不通时停下来记录现象、假设和下一步。
