# 三皮肤前端对抗测试报告（v1 / v2 / v3）

> 日期：2026-06-14 ｜ 分支：`fix/frontend-tri-skin-audit-20260614` ｜ 修复 PR：见文末
> 执行：全自动浏览器对抗深测（Playwright 驱动真实 docker 栈，非 mock）
> 范围：v1(webcode/Antd5) · v2(webcode-v2/shadcn) · v3(webcode-v3/STARFORGE HUD) 三套皮肤**全部路由 + 关键业务页数据核验**

---

## 1. 结论速览

| 维度 | v1 (Antd5) | v2 (shadcn) | v3 (STARFORGE) |
|------|-----------|-------------|----------------|
| 路由总数（非参） | 123 | 137 | 136 |
| **渲染对抗扫描（修复后）** | **0 FAIL** / 45 可达 / 78 RBAC 门禁 | **137/137 PASS / 0 FAIL** | **136/136 PASS / 0 FAIL** |
| 关键业务页数据正确 | ✅ 5/5 | ✅ 5/5 | ✅ 4/5（GIS 地图见 §6） |
| 登录（RSA 真站） | ✅ | ✅ | ✅ |
| 业务模块对齐 | 基准 | ✅ 全模块对齐 | ✅ 全模块对齐 |

**总体判定：三套皮肤业务功能齐全、数据正确、渲染零错误。** 本轮修复了 1 处部署回归 + 3 处共享层代码 bug（全部落在 `frontend-core`，三皮肤一处生效）。剩余为 3 项设计层「对齐深度」差异（非崩溃，已记录待决策，§6）。

---

## 2. 测试环境

| 项 | 值 |
|----|----|
| 代码基线 | `main` @ `a338d131`（本轮已 ff 到最新） |
| 部署 | docker compose（project `omc`，端口偏移）；web 三皮肤同 nginx：`/`=v1 `/v2/`=v2 `/v3/`=v3 |
| 前端入口 | `http://localhost:18081`（真实后端 `/api` 代理到 app:8081，**非 mock**） |
| 部署版本 | v1 `index-C0b2bZZQ` · v2 `index-C8HVQ8n0` · v3 `index-ipojZ2FB`（本轮重建） |
| 登录 | admin / admin123（RSA 客户端加密，走真实 UI） |
| 灌入数据 | 设备 **551**（KPILT-LTE 400 / NR 120 / GSM 30）· 活跃告警 **80** · 历史告警 707 · pm_metrics **2,842,936** 行 · 地理坐标 550 · system_license 1 |

数据灌注：`kpiperf -synth -tsdb`（三制式 KPI）+ 告警注入脚本 + GIS 坐标回填（赞比亚区域，与本实例地图配置一致）+ system_license 种子。

---

## 3. 测试方法（对抗深测）

1. **全路由渲染扫描**：每皮肤登录一次后逐路由（123/137/136）`goto`，严格判失败口径——
   `console.error` / `pageerror` / 4xx-5xx（401 除外）/ requestfailed / AntD error-boundary /
   空白页 / 失败文案（页面不存在|加载失败|请求失败|系统错误…）任一命中即 **FAIL**；
   被 RBAC 弹到 `/403` 单独记为 **RBAC403**（区别于渲染错误）。噪声（AntD 弃用告警 /
   SSE 中断 / `/tiles` 瓦片 404）已过滤。
2. **关键业务页数据核验**：三个 agent（每皮肤一个）适配各自框架选择器，深测
   仪表盘 / 设备列表 / 当前告警 / 性能设备视图 / GIS 地图——**核对是否真展示已灌数据**
   （表格行数、分页总数、统计卡数字、图表 canvas、KPI 卡），区分「有数据已加载却空表」
   （bug）与「优雅空态」。
3. 失败页留截图；登录避开 `waitForURL`（仪表盘轮询致执行上下文 churn 假超时），
   改 `waitForResponse(/auth/login POST 200)` + 定时等 token 落地。

---

## 4. 修复后渲染扫描结果（权威数据）

```
v1  total=123  PASS=45   FAIL=0   RBAC403=78
v2  total=137  PASS=137  FAIL=0   RBAC403=0
v3  total=136  PASS=136  FAIL=0   RBAC403=0
```

**三皮肤渲染零 FAIL。** v1 的 78 项 RBAC403 是动态菜单门禁按设计拦截（§6.1）。

### 跨皮肤模块矩阵（pass / fail / rbac）

| 模块 | v1 | v2 | v3 |
|------|----|----|----|
| dashboard | 1/0/0 | 1/0/0 | 1/0/0 |
| device | 4/0/9 | 13/0/0 | 13/0/0 |
| alarm | 4/0/2 | 6/0/0 | 6/0/0 |
| performance | 12/0/0 | 12/0/0 | 12/0/0 |
| config | 0/0/13 | 14/0/0 | **14/0/0**（修复前 10/4/0） |
| topology | 1/0/5 | 7/0/0 | 7/0/0 |
| mml | 4/0/3 | 8/0/0 | 8/0/0 |
| backup | 0/0/6 | 7/0/0 | 7/0/0 |
| file | 0/0/7 | 8/0/0 | 8/0/0 |
| mr | 0/0/5 | 6/0/0 | 6/0/0 |
| ops | 1/0/6 | 8/0/0 | 8/0/0 |
| report | 0/0/4 | 5/0/0 | 5/0/0 |
| software | 0/0/5 | 6/0/0 | 6/0/0 |
| license | 0/0/2 | **2/0/0**（修复前 1/1/0） | **2/0/0** |
| log | 0/0/6 | 7/0/0 | 7/0/0 |
| system | 8/0/5 | 14/0/0 | 14/0/0 |
| product | 6/0/0 | 6/0/0 | 6/0/0 |
| transfer | 3/0/0 | 3/0/0 | 3/0/0 |
| notifications | 1/0/0 | 4/0/0 | 3/0/0 |

> v1 整模块 `0/0/N`（backup/config/file/log/mr/report/software）= 该模块路由全不在 admin 的
> 36 项菜单内 → 全被门禁拦（§6.1）。v2/v3 无此门禁，故全可达。

---

## 5. 关键业务页数据核验（真站）

| 页面 | v1 | v2 | v3 |
|------|----|----|----|
| 仪表盘 | ✅ 总设备 551 / 活跃告警 80，10 个 KPI 端点全 200，8 canvas | ✅ 551/80/license 1，35 SVG 图 | ✅ BRIDGE：设备 551 / 告警 80 / 紧急 8，TOP 告警榜 |
| 设备列表 | ✅ 分页「共 551 条」 | ✅ 表格 20 行/页，总数 ~551 | ✅ FLEET TOTAL 551，真实 SN/厂商 |
| 当前告警 | ✅ 「共 80 条」分级正确 | ✅ 80 条 active | ✅ CRIT8/MAJ16/MIN24/WARN32=80 |
| 性能-设备视图 | ✅ 选设备弹窗→出图 14 canvas | ✅ 选设备→查询 200，71 KPI 卡（呼叫成功 87896 等） | ✅ 取数流程通，空窗口优雅空态 |
| GIS 地图 | ✅ OpenLayers + 标记（`/devices/geo` 550） | ✅ 地理分布表 550 行带坐标 | ⚠️ 0 站点（读 `/topology/geo` 空，§6.2） |

**数据正确性：v1/v2/v3 核心业务页均真实拉取后端数据、无「有数据却空表」缺陷**（网络层核对 `/api/v1/*` 全 200）。唯一数据缺口是 v3 GIS 地图（§6.2）。

---

## 6. 发现的问题与处置

### 已修复（本 PR）

| # | 问题 | 影响皮肤 | 根因 | 修复 |
|---|------|---------|------|------|
| D0 | **部署回归**：`/v2/dashboard` 深路由加载到 v1 根包 → 弹 /403 | v2/v3 全部深路由 | overlay 重建继承陈旧 `omc-base-web` 的 nginx 配置（缺 `/v2//v3` location 块）→ 深路由 fallthrough 到 `location /` 的根 `index.html` | overlay 改为同时 COPY 仓库当前 `default.conf`/`nginx.conf`（仓库配置本就正确，纯 harness 坑） |
| F1 | **硬跳登录丢皮肤上下文**：401/闲置登出 `window.location.href='/login'` 忽略 base | v2/v3 | 写死 `/login`，未带 `import.meta.env.BASE_URL` | 新增 `frontend-core/utils/appBase.ts`（`loginUrl()`），http.ts ×2 + useIdleLogout 改用之。实测 v2→`/v2/login`、v3→`/v3/login`、v1→`/login` ✅ |
| F2 | **空设备 id 打 400**：`/devices//parameters`（空段） | v3 config 4 页（实时参数/参数列表/参数同步/批量分类） | 共享 hook `useConfigParams` 无 `enabled` 守卫，设备未加载时 deviceId='' 仍发请求 | 加 `enabled: useMock||Boolean(deviceId||deviceSn)`；v2 用的 `useDeviceParameters` 本有守卫，此修复让 v3 对齐 |
| F3 | license 页 404 | v2/v3（v1 被门禁未测到） | `system_license` 表空时后端返 404 | 灌 1 条种子 license（数据条件，非代码 bug；建议后端空态返 200+null，见下） |

> 修复均落在 **`frontend-core` 共享层**，符合三皮肤铁律（一处改三皮肤同时受益）。
> 附 `appBase` 单测 8 例（三皮肤 + 兜底）；三套 typecheck 真实归零；修复后三套全路由 0 FAIL。

### 待决策（设计层对齐差异，本轮未改）

| # | 差异 | 现状 | 建议 |
|---|------|------|------|
| P1 | **RBAC 门禁不对齐** | v1 启用动态菜单门禁（`VITE_DYNAMIC_MENU=true`），admin 仅 36 项菜单 routePaths，78 条路由弹 /403；**v2/v3 无门禁**（`VITE_DYNAMIC_MENU` 未设=false + 无 PrivateRoute），所有路由可达 | 这是 v2/v3 alpha 阶段未移植动态菜单 RBAC 层的已知架构缺口；且后端菜单返 v1 式路径（`/device/list`）与 v2/v3 路由 slug（`/devices`/`/fleet`）不一致，移植需加 slug 映射层——属**专项 feature**，不宜审计期顺手改。**建议立项专项对齐**（含菜单种子是否补全的产品决策） |
| P2 | **v3 GIS 地图数据源不对齐** | v1/v2 用 `useMapDevicesGeo`→`/devices/geo`（550 设备，有标记）；**v3 用 `useGeoData`→`/topology/geo`（topology 站点表空 → 0 标记）** | v3 的 `GISMapView`（477 行自绘 HUD）围绕 topology sites/nodes 构建，与 v1/v2 的设备地理散点是**两种业务概念**。建议产品决策 v3 GIS 应展示设备地理（对齐 v1/v2）还是 topology 站点；若对齐则改数据源（中等改动，有回归风险） |
| P3 | **v2 业务展示深度** | v2 性能-设备视图用 KPI 数值卡（最新值+桶数）而非 v1 的时序 canvas 折线图；v2 GIS 为地理分布**表格**（坐标+图钉）而非交互式地图 | 数据均正确呈现，属 UI 深度选择。建议补 v2 时序图 / 交互地图以对齐 v1（feature 级） |

### 已排除（误报/非 bug）

- v3 性能「制式 LTE/NR/GSM 切换不过滤设备列表」：**误报**。v3 DeviceView 列表筛选只有 ALL/在线/上报中，LTE/NR/GSM 是出图侧**指标库（ENB/GNB/BSC）选择器**，本就不过滤列表；首页全 GSM 是 SN 排序（`KPILT-GSM` < `LTE` < `NR`）所致。
- GIS 底图灰白 = `/tiles/*.png` 离线瓦片缺失（环境项，矢量标记照常渲染）。
- 仪表盘「在线设备 0 / 活跃 UE 0」= 合成数据设备均离线，优雅零值非 fetch bug。
- 地图中心在赞比亚（lat -14.5/lng 28）= 本实例地图基建即赞比亚区域（与坐标回填一致）。

---

## 7. 三皮肤铁律达成度

- ✅ 单底层 `frontend-core` + 三皮肤架构完好；本轮 bug 全在共享层一处修复。
- ✅ 三套 `npm run typecheck`（`-p tsconfig.app.json` 真校验）归零。
- ✅ 三套全路由浏览器冒烟修复后 0 FAIL。
- ⚠️ 业务**对齐深度**尚有 3 项设计差异（P1 RBAC / P2 v3 GIS / P3 v2 展示深度），均已记录待立项，不属本轮渲染/数据缺陷。

---

## 8. 复现资产

- 路由清单：`/tmp/omc-verify/routes-v{1,2,3}.json`
- 扫描 harness：`/tmp/omc-verify/sweep2.cjs`（严格判失败口径 + 三皮肤登录）
- 扫描结果：`/tmp/omc-verify/final-v{1,2,3}.json`（+ 失败页截图 `fshots-v*`）
- basename 修复验证：`/tmp/omc-verify/verify-basename.cjs`
- 业务数据深测：workflow `tri-skin-business-data-probe`（3 agent）
- 数据灌注：`kpiperf -synth -tsdb` + `/tmp/omc-verify/inject.sh` + GIS/license 种子 SQL

---

*报告由 Claude Code 全自动对抗测试生成；修复见 PR #338（`fix/frontend-tri-skin-audit-20260614` → `main`，待 review 合入）。*
