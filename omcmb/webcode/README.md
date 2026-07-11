# webcode — OMC 前端主皮肤

OMC（运营商小基站 TR-069/CWMP 无线网管）的前端主皮肤 UI 壳。技术栈：React 19 + TypeScript（严格模式）+ Vite 7 + Ant Design 5 + Pro-Components + Zustand 5 + React Query v5 + ECharts 6 + OpenLayers。

> 本文件是「前端接手页」。仓库全景见根 [`README.md`](../../README.md)，AI 与人工的硬约束见根 [`CLAUDE.md`](../../CLAUDE.md) §3（仓库结构）、§7（关键路径）、§8.3（前端规范）。历史架构见 [`docs/project/frontend-multi-skin-plan-20260422.md`](../../docs/project/frontend-multi-skin-plan-20260422.md)。

---

## 1. 三包结构

前端是「一份业务层 + 三套 UI 皮肤」。业务层全部在 `frontend-core/`，三个 UI 壳通过 Vite alias `@core → ../frontend-core/src` 共享同一份业务层。

```
omcmb/
├── frontend-core/        # 共享业务层：services/api · hooks/api · store · types · i18n · mock · utils
├── webcode/              # 主皮肤（本目录，Ant Design 5，日常开发以此为主）
```

`frontend-core/` 是 npm workspace 成员（包名 `@omc/frontend-core`），但日常通过路径别名 `@core/*` 直接消费源码，无需 build，HMR 跨包生效。

---

## 2. 硬约束（接手前必读）

| 规则 | 说明 |
|------|------|
| **业务层归属** | API 服务 / Hook / Store / Types / i18n / Mock **必须写在 `frontend-core/`**；页面 / 组件 / 路由 / 布局 / 主题写在 `webcode/`（或对应皮肤包）。 |
| **跨包引用走 `@core`** | 形如 `import { authApi } from '@core/services/api/authApi'`；**禁止相对路径越级**（如 `../../../frontend-core/...`）。本地 UI 代码用 `@/*`。 |
| **禁 `any`** | 后端响应定义 `BackendXxx` 接口 → `mapBackendXxx` 转换 → 前端 `Xxx` 接口，三段式落地类型安全。 |
| **API / Hook 模式** | 一模块一文件：`frontend-core/src/services/api/xxxApi.ts` 导出服务对象；`frontend-core/src/hooks/api/useXxx.ts` 内部 `useMock ? mockService : realApi`。 |
| **查询键层级化** | `['domain', 'action', params]`，如 `['devices', 'list', params]`。 |
| **i18n** | 用户可见文本走 `react-intl`，语料进 `frontend-core/src/i18n/`（zh-CN / en-US），禁止散落硬编码中文。 |

---

## 3. 常用命令

以 `package.json` 的 `scripts` 为准。dev server 跑在 `:3000`，`/api`（含 `/tiles`、`/tiles-metadata`）代理到后端 App `:8081`。

```bash
cd omcmb/webcode

npm run dev          # 开发服务器 :3000，代理到后端 App :8081
npm run dev:mock     # Mock 模式开发（--mode mock，VITE_USE_MOCK=true）
npm run build        # 生产构建（npm run build:mock 为 Mock 模式构建）
npm run typecheck    # tsc --noEmit
npm run lint         # ESLint
npm run test         # Vitest 单元测试（含 frontend-core 下 __tests__；test:watch / test:coverage 同族）
npm run test:e2e     # Playwright E2E（test:e2e:ui 为交互模式）
```

> workspace 级别命令在 `omcmb/` 根：`npm install` 安装 V1 + core；`omcmb/ npm run lint` 同时扫描 webcode 与 frontend-core。

---

## 4. Mock 开关

由环境变量 `VITE_USE_MOCK` 控制（`frontend-core/src/services/apiSwitch.ts` 读取，Hook 内 `useMock ? mockService : realApi` 分流）。

- `npm run dev:mock` / `npm run build:mock` 走 `--mode mock`，自动置 `VITE_USE_MOCK=true`，不依赖后端。
- 默认 `npm run dev` 走真实后端（`/api` 代理到 `:8081`）。
- Mock 数据与适配器在 `frontend-core/src/mock/`，V1 的环境变量位于 `webcode/.env*`。

---

## 5. 后端联通

前端不直连后端进程，统一经 Vite `/api` 代理。后端运行态是 docker compose 容器栈（详见根 `CLAUDE.md` §14），App 默认监听 `:8081`。离线地图瓦片代理目标可经 `VITE_TILES_PROXY_TARGET` 覆盖。

---

## 6. 延伸阅读

- 仓库全景与上手 — 根 [`README.md`](../../README.md)
- 前端规范（硬约束权威）— 根 [`CLAUDE.md`](../../CLAUDE.md) §3 / §7 / §8.3
- 已归档的历史多皮肤方案 — [`docs/project/frontend-multi-skin-plan-20260422.md`](../../docs/project/frontend-multi-skin-plan-20260422.md)
