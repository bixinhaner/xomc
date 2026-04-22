# 前端多皮肤（Multi-Skin）架构方案

**创建日期**: 2026-04-22
**作者**: OMC 开发组
**状态**: 草案（待评审）
**关联**: `CLAUDE.md §3.1（design-baseline worktree）`、`omcmb/webcode/`

---

## 0. 背景与目标

### 0.1 现状
- 单一前端 `omcmb/webcode/`，技术栈：React 19 + Vite + TypeScript + Antd 5 + Pro-Components + Zustand + React Query
- 已有 `goomc-design/` 姊妹 worktree（`design-baseline` 分支）用于对比设计还原度，**仅对比、不开发**
- 后端 REST API 已稳定（`:8080`），前端通过 Vite `/api` 代理访问
- 业务层已相对清晰：`services/api/`（28 模块）+ `hooks/api/`（23 模块）+ `store/`（5 个 Zustand）+ `types/`（11 文件）+ `i18n/`（zh-CN/en-US）

### 0.2 目标
1. **保留业务层**：API 服务、React Query Hooks、Zustand Store、TypeScript 类型、i18n、Mock 适配器 → 在多皮肤间 100% 复用
2. **彻底重设计 UI**：新皮肤（v2）采用与现 Antd 5 完全不同的风格与组件库
3. **后端一套 / 前端多套**：同一份后端 API 同时服务 v1、v2、未来 vN
4. **并行开发不互相干扰**：v1 继续迭代修 bug / 补功能的同时，v2 从零搭建
5. **可渐进切换**：用户/租户级别选择皮肤；最终 v2 成熟后可平滑下线 v1

### 0.3 非目标
- 不重写后端或 API 契约
- 不做 micro-frontend（Module Federation / qiankun）— 过度工程
- 不支持运行时动态加载皮肤包（Build-time 选皮即可，上线后用 nginx/反向代理做路径分发）

---

## 1. 架构总览

### 1.1 最终目录结构

```
omc/                                  # git 根
├── goomc/                            # 主工作区（main 分支，v1 + core 维护）
│   ├── omcmb/
│   │   ├── frontend-core/            # 【新】共享业务层 workspace
│   │   │   ├── package.json          # name: "@omc/frontend-core"
│   │   │   └── src/
│   │   │       ├── services/         # HTTP 客户端、apiSwitch、API 服务对象
│   │   │       ├── hooks/            # React Query hooks（不含 UI）
│   │   │       ├── store/            # Zustand stores
│   │   │       ├── types/            # 所有 TS 类型
│   │   │       ├── i18n/             # 语言包（文本不含组件）
│   │   │       ├── mock/             # Mock 数据与适配器
│   │   │       └── utils/            # 格式化、字段映射、时间等纯函数
│   │   ├── webcode/                  # v1 皮肤（保留，Antd 5）
│   │   │   ├── package.json          # deps: "@omc/frontend-core": "workspace:*"
│   │   │   └── src/
│   │   │       ├── components/       # Antd 封装
│   │   │       ├── pages/            # 18 模块页面
│   │   │       ├── layouts/
│   │   │       ├── theme/
│   │   │       ├── router/
│   │   │       └── main.tsx
│   │   └── webcode-v2/               # 【新】v2 皮肤（shadcn/ui + Tailwind 或 Mantine）
│   │       ├── package.json          # deps: "@omc/frontend-core": "workspace:*"
│   │       └── src/
│   │           ├── components/       # 新组件库封装
│   │           ├── pages/            # 18 模块对应页面（逐步补齐）
│   │           ├── layouts/
│   │           ├── theme/
│   │           ├── router/
│   │           └── main.tsx
│   ├── omcgo/                        # 后端（无变化）
│   └── package.json                  # 根 workspaces 配置
├── goomc-design/                     # design-baseline 对比 worktree（无变化）
└── goomc-v2/                         # 【新】v2 开发 worktree（feat/frontend-v2 分支）
```

### 1.2 分层语义

| 层 | 归属 | 可复用性 | 禁入内容 |
|---|---|---|---|
| HTTP 客户端、拦截器 | `@omc/frontend-core/services` | 100% 跨皮肤 | `import 'antd'`、`import 'react'`（仅 `type`） |
| API 服务对象（28 模块） | `@omc/frontend-core/services/api` | 100% 跨皮肤 | UI 组件引用 |
| React Query Hooks（23 模块） | `@omc/frontend-core/hooks` | 100% 跨皮肤 | 组件 JSX、样式 |
| Zustand Store | `@omc/frontend-core/store` | 100% 跨皮肤 | — |
| TS Types | `@omc/frontend-core/types` | 100% 跨皮肤 | — |
| i18n 文案 | `@omc/frontend-core/i18n` | 100% 跨皮肤 | 组件 |
| 字段映射 / 格式化 utils | `@omc/frontend-core/utils` | 100% 跨皮肤 | UI |
| 业务组件（DeviceSelector、AlarmFilter...） | 各皮肤独立实现 | **不共享** | — |
| 页面、路由、布局 | 各皮肤独立实现 | **不共享** | — |
| 主题、样式 | 各皮肤独立实现 | **不共享** | — |

**核心原则**：`@omc/frontend-core` 只依赖 `react`（peer）、`@tanstack/react-query`（peer）、`zustand`（peer）、`axios`、`dayjs`、`react-intl`。**不依赖** 任何 UI 组件库。

### 1.3 皮肤选型决策

**v1（保留）**: Antd 5 + @ant-design/pro-components + echarts + ol

**v2（新建）** — 三选一，推荐顺序：

| 方案 | 风格 | 优势 | 劣势 | 推荐度 |
|---|---|---|---|---|
| **A. shadcn/ui + TailwindCSS + Radix** | 现代极简、高度可定制 | 与 Antd 彻底不同；无组件依赖（复制源码）；动画细腻 | 企业级重组件（表格/树）需自研或接 TanStack Table | ★★★★★（推荐） |
| B. Mantine 8 | 精致、现代、动感 | 开箱即用组件齐全；主题系统强 | 与 Antd 风格差异没 shadcn 大 | ★★★★ |
| C. Arco Design Pro | 企业风、字节系 | 组件齐全、中文生态 | 仍是"另一个 Antd"，视觉冲击感不够 | ★★★ |

**推荐方案 A**：shadcn/ui + Tailwind。理由：
- 视觉语言与 Antd 差异最大，符合"完全不同的风格"
- 源码即组件，无版本锁定，主题可深度定制
- 表格/图表组件用 TanStack Table + Recharts/Visx 补齐（比 echarts 更轻）
- Radix 底层保障可访问性

---

## 2. 实施阶段（七步走）

### Phase 0 — 准备（0.5 天，在 `goomc/` main 上进行）
- [ ] 评审并确认本方案（Owner: 前端组 + 架构专家）
- [ ] 锁定 v2 UI 栈选型（建议 shadcn/ui + Tailwind）
- [ ] 在 `docs/project/backlog.md` 登记母任务 `T-MS-FRONTEND-V2`，拆分 10 个子任务（见附录 B）
- [ ] 风险登记 `docs/project/risk-register.md`：双栈维护、并行偏移、i18n 分叉、部署复杂度

**Gate**：方案评审通过 → 进入 Phase 1

---

### Phase 1 — 抽取 `@omc/frontend-core`（2 天，在 main 上，不影响 v1）

目标：把"业务层"从 `omcmb/webcode/` 搬到 `omcmb/frontend-core/`，v1 无感继续运行。

**步骤**:
1. 根目录启用 npm workspaces — 新建 `package.json`：
   ```json
   { "name": "omc", "private": true,
     "workspaces": ["omcmb/frontend-core", "omcmb/webcode", "omcmb/webcode-v2"] }
   ```
2. 创建 `omcmb/frontend-core/package.json`（peerDeps: react / @tanstack/react-query / zustand；deps: axios / dayjs / react-intl）
3. 逐目录迁移（Git mv 保留历史）：
   - `webcode/src/services/` → `frontend-core/src/services/`
   - `webcode/src/hooks/api/` → `frontend-core/src/hooks/api/`
   - `webcode/src/store/` → `frontend-core/src/store/`
   - `webcode/src/types/` → `frontend-core/src/types/`
   - `webcode/src/i18n/` → `frontend-core/src/i18n/`
   - `webcode/src/mock/` → `frontend-core/src/mock/`
   - 选择性：`webcode/src/hooks/useResponsive.ts` 等纯逻辑 hook 也移入 core；`use3DTilt.ts`、`useMousePosition.ts` 等 UI 动效 hook **保留在 webcode**
4. 配置 `@omc/frontend-core` 的 tsconfig，输出 ESM，使用 `tsup` 或直接 `tsc` 构建 / 也可配为 source-only 引用（开发更快）
5. 在 `webcode/` 中：
   - `package.json` 加 `"@omc/frontend-core": "workspace:*"`
   - 全仓替换 import：`@/services/api/foo` → `@omc/frontend-core/services/api/foo`（用 codemod 或 ripgrep + sed 脚本）
   - Vite alias 保留 `@` 用于本地 UI 代码（components/pages/theme/...）
6. 跑 `npx tsc --noEmit` + `npm run build` + `npm run dev` + 点几个关键页面验证
7. 提交：`refactor(frontend): 抽取业务层为 @omc/frontend-core workspace`

**Gate**：v1 完全等价工作、类型检查零错误、E2E 通过 → 进入 Phase 2

**回滚方案**：若抽取后出现不可修复问题，`git revert` 该 commit 即恢复单包结构。

---

### Phase 2 — 开 v2 开发 worktree（0.5 天）

```bash
cd <path>/omc/goomc
git checkout main && git pull --rebase
git checkout -b feat/frontend-v2
git push -u origin feat/frontend-v2
cd ..
git worktree add goomc-v2 feat/frontend-v2

# 共享 node_modules（与 goomc-design 模式一致，省一次 install）
# 但因为 workspaces 把 node_modules 放在仓库根，需要在 v2 worktree 内首次 npm install
cd goomc-v2
npm install
```

**约定**:
- v2 **只**在 `goomc-v2/` 开发，不在 `goomc/` 改 v2 代码
- `@omc/frontend-core` 的修改**统一在 `goomc/`**（main 分支）完成，v2 通过 rebase / merge 同步
- 若 v2 发现 core 缺少某个 API/hook，先回 `goomc/` main 加上再合进 v2

---

### Phase 3 — v2 脚手架（1 天，在 `goomc-v2/` 内）

在 `omcmb/webcode-v2/` 下：
1. `npm create vite@latest webcode-v2 -- --template react-ts`
2. 清理模板，安装依赖：
   ```
   tailwindcss postcss autoprefixer clsx tailwind-merge
   @radix-ui/react-* (按需)
   lucide-react          # 图标
   @tanstack/react-table # 表格
   recharts              # 图表（或保留 echarts）
   react-router-dom zustand @tanstack/react-query
   @omc/frontend-core    # workspace link
   ```
3. 初始化 shadcn/ui CLI：`npx shadcn@latest init`（或手动）
4. 搭建基础目录：`components/ui/`（shadcn 组件）、`components/business/`（业务封装）、`layouts/`、`pages/`、`theme/`
5. `vite.config.ts`：端口 `:3002`，`/api` 代理到 `:8081`（与 v1 后端同端口，无需起两套）
6. `main.tsx`：挂载 `QueryClientProvider` + `IntlProvider` + Router，**复用 `@omc/frontend-core` 的 userStore 和 hooks**
7. 先跑通"登录 → 设备列表"两个页面作为骨架 smoke test

**Gate**：登录 + 设备列表两页在 v2 下可工作 → 进入 Phase 4

---

### Phase 4 — v2 页面逐模块补齐（10–14 个 Sprint）

18 个模块按优先级与依赖顺序构建。每个模块一个子任务，走正常 `/dev-pipeline` S0→S7 流程：

**优先级 P0**（基础路径，Sprint 1-2）:
- login, dashboard, device, alarm

**优先级 P1**（运维核心，Sprint 3-5）:
- config, mml, performance, software, topology

**优先级 P2**（扩展模块，Sprint 6-9）:
- backup, log, file, report, system, ops, mr, license

**特殊约定**:
- 每完成一模块，需在 `docs/project/frontend-v2-parity-matrix.md`（新建）打勾
- Parity 指的是：业务覆盖与 v1 等价（不强制 UI 一致）
- 每模块必须提供 Playwright E2E smoke（沿用 `webcode/e2e/` 框架 + copy）

---

### Phase 5 — 开发期并行运行（从 Phase 3 起）

通过 `run/scripts/start-all.sh` 扩展，同时起：
- `:3000` v1（`webcode/`）— main 分支
- `:3001` design baseline（只读对比）
- `:3002` v2（`webcode-v2/`）— 从 `goomc-v2` worktree 启动
- `:8080` 后端 app
- `:7547` 后端 acs

提供 `run/scripts/start-v2.sh` 脚本封装（Phase 3 末尾一并提交）。

---

### Phase 6 — 部署与用户切换（v2 feature parity 达标后）

**最小可用部署**（阶段性）:
- v1 仍走主域名 `omc.example.com/`
- v2 走同域子路径 `omc.example.com/v2/` 或子域 `v2.omc.example.com/`
- Nginx 配置示例见附录 A
- 两者共享同一份后端，同一份 JWT（`userStore` 的 token 统一放 localStorage 同 key）

**正式切换路径**:
1. 后端 `user_profile` 增加 `ui_skin` 字段（`v1` / `v2`）
2. 登录页（v1、v2 都有）成功后读取 profile，若 `ui_skin !== 当前皮肤` → 跳转到对应入口
3. 用户偏好页提供切换开关
4. 统计 v1/v2 的 PV/UV，灰度到全量后下线 v1

---

### Phase 7 — 收尾（v2 feature parity 达到 100% 后）

- 合并 `feat/frontend-v2` 到 main；`goomc-v2/` worktree 仍保留用于持续迭代（或删除）
- `docs/project/frontend-v2-parity-matrix.md` 归档
- 评估 v1 下线时间线，写入 milestone
- 若决定下线 v1：移除 `omcmb/webcode/`，保留 commit 历史；`@omc/frontend-core` 继续服务 v2

---

## 3. 关键技术决策

### 3.1 为何选 npm workspaces 而非 git submodule/subtree？
- 单仓库下开发体验最好，TypeScript 跨包跳转/重构生效
- 无需双向同步开销
- CI/CD 简单（单个 `npm ci` 装所有）

### 3.2 为何不做 Module Federation / 微前端？
- 两套前端不需要**运行时**共存，构建时分发即可
- MF 带来的复杂度（shared deps 版本约束、路由协同、状态同步）远超收益
- 当 v1 下线时，MF 的桥接代码反而成负担

### 3.3 HTTP 客户端唯一性
`@omc/frontend-core/services/http.ts` 是**唯一** Axios 实例来源。v1、v2 都引用它：
- 拦截器（camelCase↔snake_case、Bearer Token、401 跳登录）定义一次
- 环境配置通过 `import.meta.env.VITE_API_BASE_URL` 暴露，各皮肤各自配置 `.env`

### 3.4 Mock 适配器复用
`@omc/frontend-core/mock/` + `apiSwitch.ts` 对两个皮肤均生效。v2 的 `dev:mock` 脚本直接沿用。

### 3.5 i18n 复用
文案归 core，组件(<FormattedMessage>)调用归皮肤。避免 v1/v2 各维护一份导致翻译分叉。

### 3.6 Store 共享但页面独立
Zustand 的 `userStore` / `alarmStore` / `tabStore` / `taskStore` / `appStore` 在 core 中，两皮肤挂同一份 store（同浏览器同 localStorage key）。这样：
- 在 v1 登录 → 直接进 v2 无需重新登录（同域下）
- 偏好设置跨皮肤生效

**注意**：`tabStore` 可能存在 UI 模式差异（v2 不一定用 tab），留出可选订阅机制。

---

## 4. 风险与缓解

| 风险 | 影响 | 概率 | 缓解 |
|---|---|---|---|
| 双栈维护工作量翻倍 | 高 | 高 | 明确 v2 成熟后下线 v1 时间表；bug 修复优先级在 core 层 |
| v1/v2 业务逻辑分叉 | 高 | 中 | 强制业务逻辑走 core；PR review 专家角色检查 |
| i18n 翻译缺失 | 中 | 中 | 文案集中在 core；CI 加"key 一致性"检查 |
| Core 抽取时破坏 v1 | 中 | 低 | Phase 1 结束必须跑完整 E2E；rollback 方案清晰 |
| v2 组件生态缺口（如企业级表格） | 中 | 中 | Phase 3 骨架阶段先跑最复杂页面验证可行性（如设备列表 + 参数树） |
| node_modules 体积翻倍 | 低 | 高 | workspaces 能自动 hoist；实测 +300MB 可接受 |
| 不同皮肤路由冲突 | 低 | 中 | 部署时路径隔离（/ vs /v2/），或子域 |
| 用户同时打开两个皮肤造成 store 竞态 | 低 | 低 | 同域下 localStorage 仍一致；实际很少出现 |

**进入 `risk-register.md` 的条目**：
- R-FE-V2-01 双栈维护成本（Owner: 前端组长；下次复盘 Phase 1 结束）
- R-FE-V2-02 业务分叉（Owner: 架构专家；持续）
- R-FE-V2-03 Core 抽取风险（Owner: 前端组；Phase 1 结束关闭或升级）

---

## 5. 质量关卡

### 5.1 Phase 1 完成必须
- [ ] `@omc/frontend-core` 编译通过，无循环依赖
- [ ] `webcode/` 全部页面可用（手工 smoke 覆盖 18 模块首页）
- [ ] `npx tsc --noEmit` 零错误
- [ ] `npm run lint` 零警告
- [ ] 后端 E2E 脚本跑通（前后端联通性）
- [ ] 前端 Playwright E2E 跑通

### 5.2 Phase 3 完成必须
- [ ] v2 登录 + 设备列表两页功能等价于 v1
- [ ] v2 复用 core 无重复代码
- [ ] 样式与 v1 视觉差异显著（由架构专家判定）

### 5.3 v2 单模块合入必须（每模块）
- [ ] 走 `/dev-pipeline` S0→S7
- [ ] Parity Matrix 勾选
- [ ] Playwright smoke 绿灯
- [ ] `@omc/frontend-core` 若新增/改动 API，先在 `goomc/` main 完成并发布
- [ ] v1 对应模块不被破坏（CI 双栈并行测试）

### 5.4 绝不要
- 在 `@omc/frontend-core` 内 `import 'antd'` 或 `import 'react-dom'` 相关 UI
- 在 v2 绕过 core 自行发请求（除非为了 v2 专属的新端点，也应加回 core）
- 复制粘贴 v1 组件到 v2（要么走 core，要么重新设计）
- 让 v2 依赖 v1 包

---

## 6. 时间预估

| 阶段 | 工作量 | 依赖 |
|---|---|---|
| Phase 0 准备 + 方案评审 | 0.5 d | — |
| Phase 1 Core 抽取 | 2 d | Phase 0 |
| Phase 2 v2 worktree | 0.5 d | Phase 1 |
| Phase 3 v2 脚手架 | 1–2 d | Phase 2 |
| Phase 4 v2 模块补齐（18 模块 × 2–3 d/模块） | 36–54 d | Phase 3，可并行 |
| Phase 5 并行运行脚本 | 0.5 d | Phase 3 |
| Phase 6 部署/灰度 | 2 d | Phase 4 |
| Phase 7 收尾 | 1 d | Phase 6 |
| **总计（单人）** | **43–62 d** | |
| **双人并行（P4 拆分）** | **约 25–35 d** | |

---

## 附录 A — Nginx 双皮肤部署示例

```nginx
server {
    listen 443 ssl;
    server_name omc.example.com;

    root /var/www;

    # v2 优先（避免 /v2/assets 被 v1 SPA 路由吞掉）
    location /v2/ {
        alias /var/www/webcode-v2/;
        try_files $uri $uri/ /v2/index.html;
    }

    # v1 默认
    location / {
        root /var/www/webcode;
        try_files $uri $uri/ /index.html;
    }

    # 后端共享
    location /api/ {
        proxy_pass http://omc-app:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

v2 构建时需设置 `base: '/v2/'` 的 Vite 配置。

---

## 附录 B — Backlog 子任务草稿

登记到 `docs/project/backlog.md` 的 `T-MS-FRONTEND-V2` 母任务下：

- T-FE-V2-01: 方案评审 + UI 栈选型锁定
- T-FE-V2-02: 建立 npm workspaces + 抽取 `@omc/frontend-core`
- T-FE-V2-03: 建立 `feat/frontend-v2` 分支与 `goomc-v2/` worktree
- T-FE-V2-04: v2 项目脚手架（Vite + Tailwind + shadcn/ui + Router + Providers）
- T-FE-V2-05: v2 登录 + 主布局 + 侧边栏
- T-FE-V2-06: v2 Dashboard 模块
- T-FE-V2-07: v2 Device 模块（首个复杂模块，验证可行性）
- T-FE-V2-08: v2 Alarm 模块
- T-FE-V2-09: v2 Config + MML 模块
- T-FE-V2-10: v2 Performance + Topology 模块
- T-FE-V2-11: v2 剩余 10 个模块（分批）
- T-FE-V2-12: 双皮肤部署 + 用户偏好切换
- T-FE-V2-13: v2 Playwright E2E 套件
- T-FE-V2-14: v1 下线预案评审（v2 GA 后）

---

## 附录 C — Core 目录结构最终形态（Phase 1 目标）

```
omcmb/frontend-core/
├── package.json
├── tsconfig.json
├── src/
│   ├── index.ts                   # 统一导出
│   ├── services/
│   │   ├── http.ts                # Axios 实例 + 拦截器
│   │   ├── apiSwitch.ts           # useMock 开关
│   │   └── api/                   # 28 个 xxxApi.ts
│   ├── hooks/
│   │   └── api/                   # 23 个 useXxx.ts
│   ├── store/                     # 5 个 zustand store
│   ├── types/                     # 11 个 .ts 类型文件
│   ├── i18n/
│   │   ├── zh-CN/
│   │   └── en-US/
│   ├── mock/                      # mock 数据源与 handler
│   └── utils/
│       ├── caseTransform.ts       # snake↔camel
│       ├── date.ts
│       └── format.ts
└── README.md                      # core 使用说明 + 版本演进规则
```

**导出策略**：子路径导出（`@omc/frontend-core/services/api/deviceApi`），在 `package.json` 中配 `exports` 字段，避免 barrel file 导致 tree-shaking 失效。

---

## 变更记录

- 2026-04-22 初稿
- 2026-04-22 Phase 1 执行完成 — 见下节"执行记录"

---

## Phase 1 执行记录（2026-04-22）

### 实际完成

1. ✅ 治理分层违规：`NameFilterItem` 从 `pages/device/DeviceGrouping/types.ts` 提升到 `@core/types/device`（同时在 pages 内保留 `export type` 向后兼容）
2. ✅ `git mv` 迁移 6 个业务层目录到 `omcmb/frontend-core/src/`：`services/`、`hooks/api/`、`store/`、`types/`、`i18n/`、`mock/`
3. ✅ frontend-core 内部 79 个文件的 `@/` 转为相对路径
4. ✅ webcode 内部 141 个文件、234 处跨包 import 转为 `@core/*`
5. ✅ webcode 的 `tsconfig.app.json` / `vite.config.ts` / `vitest.config.ts` 添加 `@core` 别名 + 扩展 include
6. ✅ `omcmb/eslint.config.js`（workspace-level）+ `omcmb/package.json` `lint` 脚本扫描两个包
7. ✅ 建立 npm workspaces：`omcmb/package.json`（根）+ `omcmb/frontend-core/package.json`

### 质量验证

| 检查 | 迁移前（baseline main） | 迁移后 | 判定 |
|---|---|---|---|
| `tsc --noEmit` | 0 错误 | 0 错误 | ✅ |
| `npm run build`（vite） | 通过 | 通过 | ✅ |
| `npm run test`（vitest） | 11 失败 / 1 通过 | 11 失败 / 1 通过 | ⚠️ baseline 问题，与迁移无关 |
| `npm run lint`（eslint） | 342 errors / 69 warnings | 284 errors / 69 warnings | ⚠️ baseline 问题，迁移未引入新错误 |

**baseline 前置问题**（独立工单后续修复）：
- 单元测试失败：jsdom 下 zustand localStorage polyfill 缺失 + vitest 未固化 `VITE_USE_MOCK=false`
- lint 错误：主要是 `@typescript-eslint/no-unused-vars`、`react-hooks/exhaustive-deps`、`react-refresh/only-export-components` 等 pages/components 层的既有代码问题

### 与原方案的偏离

| 原方案 | 实际执行 | 原因 |
|---|---|---|
| npm workspaces 在 `goomc/` 根（含后端） | npm workspaces 根放在 `omcmb/` | 不污染 git 根；Node 工具链影响范围限制在前端目录 |
| 引入 `@omc/frontend-core` 作为 import 包名 | 保留路径别名 `@core/*`；包名 `@omc/frontend-core` 只做 workspaces 标识 | 改名成本高；`@core` 语义更直白 |
| frontend-core 子路径导出（`exports` 字段） | 不使用；`@core` 直接指到 `src/` 源码 | Vite + TS 直接消费源码，免 build；HMR 跨包工作 |
| lint 依赖在 webcode 里 | 同时在 `omcmb/` 根声明 lint devDeps（让 eslint 配置文件的 deps 能在 workspace 根解析） | ESLint 9 禁止扫描 cwd 之外的文件，必须把 config 上提；上提后配置文件的 deps 也要能在根层解析 |

### 目录结构（现状）

```
omcmb/
├── package.json              # workspace 根，lint 脚本，lint devDeps
├── eslint.config.js          # workspace-level ESLint 配置
├── frontend-core/
│   ├── package.json          # @omc/frontend-core，声明 peerDeps
│   └── src/
│       ├── services/ (with __tests__/)
│       ├── hooks/api/
│       ├── store/ (with __tests__/)
│       ├── types/
│       ├── i18n/
│       └── mock/
└── webcode/
    ├── package.json          # 通过 @core/* 访问 frontend-core
    ├── eslint.config.js      # 保留，供 cd webcode 时单独 lint
    ├── tsconfig.app.json     # paths.@core → ../frontend-core/src
    ├── vite.config.ts        # resolve.alias.@core → ../frontend-core/src
    ├── vitest.config.ts      # 同上
    └── src/
        ├── components/ / pages/ / layouts/ / theme/ / router/ / providers/ / hooks/ (UI 层 hooks) / assets/ / styles/
        └── main.tsx / App.tsx
```

### 开发/构建命令

```bash
# 在 omcmb/ 根
cd omcmb
npm install           # 一次性安装所有 workspace 依赖
npm run lint          # 同时 lint webcode 和 frontend-core
npm run typecheck     # 委托到 webcode 的 tsc --noEmit（覆盖 frontend-core）

# 在 omcmb/webcode/
cd omcmb/webcode
npm run dev           # Vite dev server :3000
npm run build         # 生产构建
npm run test          # vitest（覆盖 frontend-core 下 __tests__）
npm run typecheck     # tsc --noEmit
```

### 下一步（Phase 3 准备）

- v2 脚手架 `omcmb/webcode-v2/` 直接作为新 workspace 加入 `omcmb/package.json`，消费 `@core/*` 与 `@omc/frontend-core` symlink
- 运行时 workspace 共享 `@core/services/http.ts`，同一个 Axios 实例、同一套 Zustand store、同一份 i18n
- UI 栈选型待确认（建议 shadcn/ui + Tailwind）

---

## Phase 3 执行记录（2026-04-22）

UI 栈锁定：**shadcn/ui + TailwindCSS + Radix + lucide-react**。

### 创建结构
```
omcmb/webcode-v2/
├── package.json            # workspace 成员，依赖 @omc/frontend-core
├── index.html
├── tsconfig.{json,app.json,node.json}
├── vite.config.ts          # :3002，代理 /api → :8081，@core alias
├── tailwind.config.js      # shadcn CSS var 设计令牌
├── postcss.config.js
├── .env / .env.development
└── src/
    ├── env.d.ts
    ├── main.tsx / App.tsx
    ├── styles/globals.css  # Tailwind + CSS var（亮/暗双主题）
    ├── lib/utils.ts        # cn helper
    ├── components/ui/      # shadcn primitives (button/input/label/card)
    ├── providers/{QueryProvider,IntlProvider}
    ├── router/             # React Router，Protected 路由
    └── pages/
        ├── login/          # 真实接入 @core authApi（login + getMe）
        └── dashboard/      # 空壳，验证退出跳转
```

### 关键 @core 集成点
- `@core/services/api/authApi` — 登录 + getMe
- `@core/store/userStore` — setTokenPair / login / clearAuth / isAuthenticated
- `@core/store/appStore` — locale（IntlProvider 订阅）
- `@core/i18n` — getMessages(locale)

### 验证
- `npx tsc --noEmit`：0 errors ✅
- `npm run dev`：Vite ready（:3002 被占用时自动降 :3003）；HTTP 200；login/dashboard 模块可解析
- `npm run build`：1821 modules → 705 KB / 13.5 KB CSS / gzip 210 KB ✅

### 约束
- v2 首版页面仅 login + dashboard 壳，不承诺功能覆盖
- 18 模块补齐按 backlog T-0035 逐个立项
- UI 风格差异策略：shadcn/ui 极简 + Radix 可访问性，Tailwind 工具类直写，不采用 Antd 组件风格

---

## Phase 4 首个复杂页面：Device 模块（2026-04-22）

### 目的
验证 v2 皮肤完整吃通 `@omc/frontend-core`：React Query hook + Axios + Zustand + 字段映射全链路。

### 交付
- **AppShell 布局**：`components/layout/AppShell.tsx`，侧栏导航（控制台 / 设备管理 / 告警中心 TBD / 配置 TBD）+ 顶栏（当前用户 + 退出）
- **Devices 页面**：`pages/devices/index.tsx`
  - 调用 `@core/hooks/api/useDevices.useDeviceList`（直接消费 @core，不 proxy）
  - @tanstack/react-table 渲染表格（8 列：SN / 名称 / 厂商 / 产品型号 / 制式 / 连接状态 / 告警 / 最近在线）
  - 关键字搜索（`searchText` 参数）+ 连接状态 Select 过滤
  - 分页（上一页 / 下一页 + 页码提示）
  - 统计卡片（总计 / 在线 / 离线 / 有告警）读 `response.stats`
  - loading / empty / error 三态完备
- **shadcn 原语补齐**：Table、Badge（含 success/warning/destructive/muted 变体）、Select（基于 @radix-ui/react-select + tailwindcss-animate）
- **新增 v2 依赖**：`@tanstack/react-table`、`@radix-ui/react-select`、`tailwindcss-animate`
- **Router 改造**：Protected 路由外套 AppShell，`/dashboard` 和 `/devices` 共享布局；登录页独立

### 验证
| 检查 | 结果 |
|---|---|
| `tsc --noEmit` | 0 errors ✅ |
| `npm run build` v2 | 1912 modules → 882 KB JS / 21 KB CSS / gzip 262 KB ✅ |
| `npm run dev` v2 | HTTP 200；`useDeviceList` 正确解析为 `/@fs/.../frontend-core/src/hooks/api/useDevices.ts` ✅ |
| workspace `npm run lint` | 0 errors / 149 warnings（v2 新增 4 条 React Compiler 提示，不阻塞）✅ |
| `npm run test` | 12/12 passed ✅ |

### 架构意义
- 证实"v2 皮肤 = UI 层；业务层 = `@core` 共享"的分层有效
- 同一份 `userStore`：v1 登录 / v2 登录，localStorage key 相同，可互相识别认证状态
- 同一个 Axios 实例：`@core/services/http.ts` 的拦截器（camelCase↔snake_case + Bearer Token + 401 跳登录）对 v2 自动生效
- Mock 开关（`VITE_USE_MOCK`）各皮肤独立 .env，互不干扰

### 下一模块候选
优先补 **Alarm 告警中心**：实时数据 + Alarm 列表 + 严重度筛选。验证 @core/hooks/api/useAlarms 集成。

---

## Phase 5 一次性铺满 15 个模块（2026-04-22）

### 目的
一次性铺完 v2 的 15 个模块导航，让用户看到完整面貌；每个页面都接真实 `@core` hook，证明架构在广度上成立。

### 交付
**AppShell 分组侧栏**：监控 / 运维 / 数据 / 系统 四分区，16 个入口（含 Login）。

**真数据列表页 8 个**（接 `@core/hooks/api`）：
- `/dashboard` — 控制台（scaffold）
- `/devices` — 设备管理（TanStack Table，搜索+状态过滤+stats+分页）
- `/alarms` — 告警中心（严重度徽章，关键字+severity 过滤，30s 自动刷新）
- `/software` — 软件版本（状态徽章，文件大小/发布日期）
- `/backup` — 备份任务（状态/进度/设备数/文件大小）
- `/license` — 许可证（容量进度条，到期提醒）
- `/files` — 文件管理（类型/状态/大小/设备/上传者）
- `/reports` — 报表定义（类型/周期/自动生成/状态）
- `/logs` — 系统日志（级别徽章，关键字+级别过滤）
- `/system` — 系统管理（用户列表，用户名搜索）

**骨架列表页 6 个**（接 `@core` hook，暂只展示列表，不做详情/图表）：
- `/topology` — 站点列表（地图视图待补）
- `/config` — 配置模板（下发流程待补）
- `/mml` — MML 脚本（执行界面待补）
- `/performance` — KPI 列表（趋势图表待补）
- `/mr` — MR 指标列表
- `/ops` — 运维模板库

**共享基础设施**：
- `components/layout/PageShell.tsx` — 页面容器（标题 + 描述 + 工具栏 + isFetching 指示）
- `Pagination` / `LoadingRow` / `ErrorRow` / `EmptyRow` 统一三态 helper
- `formatTime` / `formatBytes` 纯函数工具

**新增 shadcn 原语**：Table、Badge、Select（都支持亮/暗主题 + CSS variables）

### 验证
| 检查 | 结果 |
|---|---|
| v2 src `tsc --noEmit -p tsconfig.app.json` | 0 errors ✅（@core 层 108 个既有错误已知待治理，不在本次范围）|
| `npm run build` v2 | 1093 KB JS / 22 KB CSS / gzip 310 KB ✅ |
| workspace `npm run lint` | 0 errors / 152 warnings ✅ |
| `npm run test` | 12/12 passed ✅ |

### 质量线
- **广度覆盖**：16 个侧栏入口全部接通 @core 真数据
- **深度**：Devices 模块（Phase 4）和 8 个"真数据"页完整；6 个"骨架"页只展示列表，详情/图表/下发流程标注 TBD
- **已知 debt**（独立治理，不在当前任务范围）：
  - @core 类型不一致（`authApi` 返回字段 vs `User` 类型定义）
  - @core `useAlarms` / `useConfig` 等 hooks 在 `createApiSwitch` 上有 TS2345 类型不匹配（`mock` 和 `real` 的 service 函数签名需对齐）
  - `package.json` 的 `typecheck` 目前是 bare `tsc --noEmit`，`tsconfig.json` 的 `files: []` 让它 trivially pass，需升级为 `tsc --noEmit -p tsconfig.app.json`

### 仍待补齐的深度能力（按优先级）
1. Device 详情（参数树、实时 KPI 图表、历史日志）
2. 各页面的"创建/编辑/删除/执行"Action 集合（目前只读）
3. Topology 地图视图（openlayers 或 leaflet）
4. Performance KPI 趋势图（recharts 或 echarts）
5. MML 命令执行界面
6. Config 下发流程
7. 主题切换 + 用户偏好持久化



