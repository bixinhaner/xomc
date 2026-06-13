# webcode-v2/v3 皮肤恢复与业务适配路线图

**日期**: 2026-06-13
**分支**: `feat/restore-webcode-v2-v3`
**关联**: `docs/project/frontend-multi-skin-plan-20260422.md`（多皮肤总方案）、移除提交 `961c1984`(#102)

---

## 0. 背景

2026-06-10 提交 #102 以"暂不需要"为由移除了多皮肤候选 `webcode-v2`/`webcode-v3`。
本轮按需求恢复二者，并打通"**单底层（frontend-core）+ 三套页面（webcode/v2/v3）**"的
构建与部署。

### 前端架构现状（恢复后）

```
omcmb/
├── frontend-core/   单底层：API / Hooks / Store / Types / i18n / Mock（@core，三套共享）
├── webcode/         v1 主皮肤  · Antd 5 + ProComponents      · ~296 页面文件（成熟商用）
├── webcode-v2/      v2 皮肤    · shadcn/ui + Tailwind + Radix · 每模块单页骨架（17 模块）
└── webcode-v3/      v3 皮肤    · STARFORGE 沉浸式 HUD          · 少数实页 + 多为占位（_stub）
```

- **底层只有一套**：业务逻辑全在 `frontend-core`，皮肤包只放页面/组件/router/theme。
- **部署一套底层三套页面**：同一 nginx 下 `/`=v1、`/v2/`=v2、`/v3/`=v3，共享同一后端 `/api`。
  皮肤靠各自 vite `base` + 路由 `basename` 隔离子路径；产物分目录到 `html/`、`html/v2`、`html/v3`。

---

## 1. 本轮已完成（地基增量）

| 项 | 内容 |
|----|------|
| 恢复 | 从 `961c1984^` 恢复 v2/v3 源码（99 文件），重新纳入 `omcmb` workspaces |
| 框架对齐 | v2/v3 升 react-router 7 / react-intl 10，补 `node-forge`(RSA 登录)；ts/vite 用 hoist 的 5.9.3/7.3.5 与 lock 一致 |
| v2/v3 漂移修复 | `Alarm.alarmCode→alarmIdentifier`、`MMLScript.deviceType→tags`、DeviceStats 可空、**topology 漏导入 useRef/useEffect（运行期崩溃）**、经纬度空值守卫 |
| frontend-core 类型债 | 清理 v1 typecheck 空跑漏检的 9 处（dashboard 未用导入/KPI 时序类型/useDashboard 转型、deviceApi `cellStatus`、mmlConsole op 收窄、mock device/stats 字段补齐）→ v1 潜伏错误 99→89 |
| 部署 | `Dockerfile.web` 构建三套并分目录；`default.conf` 增 `/v2/`、`/v3/` SPA location |
| 验证 | 三套 `typecheck` 0 错、`build` 绿；重建 `omc-local-web` 起容器；浏览器冒烟 **登录 3/3 · 18/18 页面渲染真实后端数据** |

> ⚠️ 已知遗留：v1 的 `typecheck` 脚本（`tsc --noEmit` 跑 `files:[]` 的 solution config）实际**空跑不校验**，
> frontend-core 仍有约 89 处潜伏类型错未被门禁拦截。建议另立任务：清 frontend-core 类型债 +
> 改用 `tsc -b` 做真实门禁（三套皮肤统一）。

---

## 2. 业务深度差距（v2/v3 → v1 适配的真实工作量）

v1 有 22 模块 / ~296 页面文件（详情页、下钻、向导）。v2/v3 当前只是**广度骨架 / demo**：

| 模块组 | v1 深度（文件数） | v2 现状 | v3 现状 | 适配工作 |
|--------|------------------|---------|---------|----------|
| device | 49（列表+详情+下钻+小区/PLMN 向导） | 单页列表 | 占位 | 详情页/下钻/向导全套 |
| mml | 57（命令树 console + 任务） | 单页脚本列表 | 实页(部分) | 命令树交互、任务执行流 |
| performance | 28（KPI 多级下钻） | 单页 | 实页(部分) | KPI 库/下钻/图表联动 |
| system | 28（用户/角色/字典/策略） | 单页 | 占位 | RBAC/字典/安全策略页 |
| product | 21 | 缺失 | 缺失 | 整模块新建 |
| config | 13（模板/基线/审计/备份） | 单页 | 占位 | 模板/基线/审计/备份 |
| alarm | 16 | 单页 | 实页 | 规则/关联/生命周期 |
| 其余(backup/mr/ops/topology/file/log/software/report/license/notifications/transfer) | 4–10 各 | 多为单页 | 多为占位 | 逐模块补详情与操作 |

**结论**：把 v2/v3 做到 v1 业务水平 = 在两套不同 UI 体系里各重建 ~22 模块的完整页面，
属多 PR 的大工程，需按模块优先级分批推进，不能一次到位（否则是 22 模块的浅层半成品）。

---

## 3. 后续路线（建议按模块切 issue，每个 = 一个独立 PR）

按业务价值优先级，建议从两套皮肤各自最常用的模块起步：

1. **P0 核心闭环**（每皮肤）：device 列表→详情→下钻、alarm 生命周期、performance KPI 下钻
2. **P1 配置与运维**：config 模板/基线、mml 命令树、software 升级、backup
3. **P2 长尾**：product、system(RBAC/字典)、report、file/log/mr/ops/topology 深化、license、notifications/transfer
4. **横切**：v2/v3 i18n 接入（当前部分硬编码中文）、暗色/主题 token、空态/错误边界对齐 v1、
   v3 外链字体改本地（CSP `font-src 'self'` + 离线）

切片建议套用 `/ship` + `to-issues`（tracer-bullet 纵切），每模块"列表→详情→操作"一条竖线打穿。

---

## 4. 复现实验环境

```bash
# 三套已部署在运行栈 web 容器：
#   http://localhost:18081/      v1
#   http://localhost:18081/v2/   v2
#   http://localhost:18081/v3/   v3
# 登录 admin/admin123（RSA，仅 UI 可登）。
# 重建 web： docker compose -p omc -f deployments/docker/docker-compose.yml \
#            -f /tmp/omc-ports-override.yml build web && ... up -d --no-deps --no-build web
```
