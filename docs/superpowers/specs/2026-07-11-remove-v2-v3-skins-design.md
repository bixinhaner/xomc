# 删除 V2/V3 前端皮肤设计

## 目标

项目只保留 V1 主前端 `omcmb/webcode` 与共享业务层 `omcmb/frontend-core`，整体移除
`omcmb/webcode-v2`、`omcmb/webcode-v3`，同时清理构建、部署、验证和 AI 提示词中的
多皮肤约束。

该变更独立于 #36。#36 已先通过 MR !57 合入，本设计不再修改 #36 的业务行为。

## 删除范围

### 源码与依赖

- 删除 `omcmb/webcode-v2/` 全目录。
- 删除 `omcmb/webcode-v3/` 全目录。
- 从 `omcmb/package.json` workspaces、lint、typecheck 中移除两套皮肤。
- 删除 `omcmb/scripts/skin-parity.mjs` 及只服务于三皮肤一致性的脚本/测试。
- 用 `npm install --package-lock-only --legacy-peer-deps` 重新生成根 lockfile，移除两个
  workspace 及其专属依赖；保留 V1/共享层仍需要的依赖。

### Docker 与 Nginx

- `deployments/docker/Dockerfile.web` 只复制、构建、校验和发布 V1。
- `deployments/docker/default.conf` 与 `default.local.conf` 删除 V2/V3 SPA location。
- 增加精确兼容入口：`/v2`、`/v2/`、`/v3`、`/v3/` 返回 302 到 `/`。
- 不对 `/v2/<任意旧页面>` 做静态映射；使用前缀 location 将旧深链统一重定向到 `/`，
  避免被 V1 SPA fallback 当成未知业务路由。
- Docker web 镜像中不得再包含 `/usr/share/nginx/html/v2`、`v3`。

## 当前文档与提示词

更新当前有效文件，使后续开发只维护 V1：

- 根 `AGENTS.md` 删除“三皮肤铁律”，改为“V1 单皮肤约束”。
- 根 `CLAUDE.md` 删除三皮肤同步实现、skin-parity 和三套冒烟要求。
- `README.md`、`docs/系统说明文档.md`、Docker 访问说明与部署 README 只描述 V1。
- 搜索项目中的现行命令/CI/脚本引用，确保没有命令继续访问已删除 workspace。

`.codex/`、`.agents/` 等本机配置仍不提交。若提示词文件已经被 git 跟踪，则作为本次
产品维护规则变更提交；未跟踪的个人配置不纳入。

## 历史文档策略

历史设计、审查报告、任务报告不重写正文，因为它们描述当时真实实现。对仍可能被当作
当前架构入口的文档（例如 `docs/project/frontend-multi-skin-plan-20260422.md`）在文件顶部
增加醒目的归档说明：V2/V3 已于本次变更移除，当前实现以 V1 为准。

普通历史记录中出现的 `webcode-v2`、`webcode-v3` 路径允许保留；删除验收不会要求全仓
文本零命中，而要求“活动代码、活动配置、当前指南和可执行命令零引用”。

## 旧路径行为

采用兼容重定向而非 404/410：

```text
/v2/* ─┐
       ├─ 302 Location: /
/v3/* ─┘
```

这样旧书签不会落入 V1 的 NotFound 页面，也不会长期承诺保留多皮肤 URL。302 便于后续
调整；不使用 301，避免浏览器永久缓存迁移策略。

## 验证

1. `npm ci --legacy-peer-deps` 成功，lockfile 不再含两个 workspace 节点。
2. `npm run typecheck` 只执行 V1 typecheck 且通过。
3. `npm run lint` 的扫描路径只包含 V1 与 frontend-core；按仓库既有 lint 基线记录结果。
4. `rg` 检查活动配置、脚本、提示词中无 V2/V3 或 skin-parity 引用。
5. Docker web 构建成功，容器只含 V1 产物。
6. Nginx 配置测试通过；`/` 返回 V1，`/v2/*`、`/v3/*` 返回 302 且 Location 为 `/`。
7. 后端不修改，无需全量 Go 回归；Docker 栈状态需正常。
8. 删除量和 lockfile 变化经独立代码审阅，确认没有误删 `frontend-core` 或 V1 消费代码。

## 提交与合入

使用独立分支、独立 GitLab MR。MR 描述必须明确这是架构收敛，不与 #36 混合；合入后
同步本地 main，删除本地/远端删除分支。
