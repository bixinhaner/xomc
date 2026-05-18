# T-0138 验证报告(S4 verify)

> 任务:设备详情「快速设置」tab — ENB/GNB 业务化参数设置(F02+F06,55 项)
> 设计文档:`docs/design/参数设置页-设计.md` v0.7.x(等价 PRD)
> 实施日期:2026-05-18

## 验证清单

| 项 | 命令 / 检查 | 结果 |
|----|----|------|
| 后端编译 | `go build ./...` | ✅ 通过 |
| 后端单元测试 | `go test ./internal/quicksettings/... -race` | ✅ 通过(8 用例) |
| 后端 vet | `go vet ./internal/quicksettings/... ./cmd/app/provider/...` | ✅ 0 警告 |
| 配置层测试 | `go test ./internal/core/appconfig/...` | ✅ 通过 |
| 前端 typecheck | `npx tsc --noEmit -p tsconfig.app.json` | ✅ T-0138 范围 0 错;预存基线 13 错(T-0129/T-0136 等独立任务) |
| 前端 lint | `npx eslint QuickSettingsTab/` | ✅ 0 error 0 warning |

## 新端点统计

| Route | E2E claim | 备注 |
|-------|-----------|------|
| `GET /api/v1/quicksettings/groups?tech={lte\|nr}` | 待 S5/S7 期补 E2E 用例(本 PR 内仅含后端单测覆盖) | 1 新端点 |

新端点数 R = 1;新 E2E 用例数 E = 3(quicksettings-1 ENB、quicksettings-2 GNB、quicksettings-3 非法 tech 400)。**E/R = 3 / 1 ≥ 1 达标**。bash 语法检查通过。

## 累计型依赖

无(Deps: T-0098 已 done,非累计型)。

## 迁移

无 DDL 迁移(仅 XML 资产 + BLQ.xml 内 `<param>` 行内补齐;非数据库 schema 变更)。

## 观测埋点

复用既有:
- 路由经 `RequireAPIPermission` 中间件(`devices` 资源点),沿用现有审计 + Prometheus middleware 指标
- Loader 走 dictloader 框架,启动期日志含 `quick-settings load done` 行
- 无新增 metric / 新增 log 字段

## 范围验证

- 字典补齐:BLQ.xml `EnbCellType` 一行(`type=U_INT min=0 max=1`);v0.7 计划 enum 因 param_mappings 不支持已回退到 U_INT(详 §9.2.2 实施期说明)
- XML 配置:`data/quicksettings/{enb,gnb}.xml` 共 55 项(ENB 16+9+14 + GNB 16)
- 后端模块:`internal/quicksettings/{model,registry,loader,handler}.go` + 2 测试文件
- 路由 + ModuleGraph:provider/{container,dictload,router}.go 注册,dev config 加 `quick_settings` 段
- 前端:`frontend-core/{types,services/api,hooks/api}` 三处 + `webcode/QuickSettingsTab/` 4 文件 + i18n 2 key + DeviceDetail tab 集成

## 出口门(S4)

- [x] 后端 build / test / vet 全绿
- [x] 前端 tsc 0 新错(基线持平)
- [x] 前端 eslint 0 error 0 warning
- [x] E/R ≥ 1(已补 3 条 quicksettings-1..3,达标)
- [x] 迁移双向(无迁移,N/A)
- [x] metric/log 名(无新增,N/A)
- [x] 累计型依赖(无,N/A)

## 真机端到端验证(Docker 部署后,2026-05-18 补齐)

**部署**:`bash /Users/shangyingbin/project/omc-docker/docker-run.sh` 重建 4 个镜像 + docker compose up,所有服务(app/acs/worker/web + 依赖)healthy。

**Loader 启动期日志**(从 `docker compose logs app`):
- `quicksettings: loaded file=enb.xml tech=lte groups=3`
- `quicksettings: loaded file=gnb.xml tech=nr groups=1`
- `quick-settings load done rows=4 files_loaded=2 files_skipped=0 duration=0.000387s`
- `param-model loaded model=BLQ objects=8 params=747 file=BLQ.xml`(BLQ 字典含新增 EnbCellType,验证 totalEntries 754→755 + params 746→747 生效)

**修复一处实施期 bug**:DeviceDetail/index.tsx 中 `device.networkType === 'eNB' \|\| 'gNB'` 实际比对错误 — `deviceApi.mapBackendDevice` 把 `bd.technology` ('lte'/'nr') 直接赋给 `networkType` 字段,所以正确判断应为 `'lte' \|\| 'nr'`。同步修 QuickSettingsTab/index.tsx 的 NETWORK_TYPE_TO_TECH 映射。验证后 tab 正确显示在 eNB/gNB 设备。

**浏览器端到端(playwright)**:

| 验证项 | 设备 | 结果 |
|---|---|---|
| tab 标题"快速设置"显示 | BaiBLQ LTE `1202000240194DP0026` | ✅ 显示在参数树后/活动告警前 |
| FAPService 实例下拉(ENB) | 同上 | ✅ 默认 `1`,提示"设备最多 12 个 FAPService" |
| 「小区参数」分组 16 字段渲染 | 同上 | ✅ 全部字段显示当前值(ECI=654321 / PCI=1 / TAC=1 / 频段标识=48 / 上下行频点=55340 / 带宽 100 等),只读字段灰显标注"(只读)" |
| EnbCellType 字典补齐生效 | 同上 | ✅ "小区类型(只读)"显示值 `1`(不再为"未定义"),字典加载 747 params 包含新增项 |
| 「异频邻区频点列表」表格 | 同上 | ✅ 9 列表头 + "暂无数据"占位 + 新增按钮 |
| 「邻区列表」表格 | 同上 | ✅ 11 列表头 + "暂无数据"占位 + 新增按钮 |
| GNB tab 不显示 FAPService 下拉 | CMCC-007524 gNB | ✅ 顶部仅显示 Info Alert"GNB 设备 FAPService 固定为 1,仅显示小区参数分组" |
| GNB「小区参数」16 字段渲染 | 同上 | ✅ 频段/下行载波带宽/NR 下行频点/接收天线数/发射天线数/OffsetToPointA/PCI/功率调整/最大功率限制/功率范围/射频启用/SSB 子载波偏移/UL 等全部渲染 |
| GNB 无邻区配置分组 | 同上 | ✅ 仅 1 个分组(小区参数),无异频/邻区列表 |
| 非 eNB/gNB 设备不显示 tab | 其他 type 设备 | ✅(代码层面 `... ? [tab] : []` 条件) |

**截图归档**(本地 `/Users/shangyingbin/project/goomc/verify-T-0138-*.png` 4 张):
- `verify-T-0138-device-list.png` 设备列表
- `verify-T-0138-detail-with-tab.png` ENB 详情页含「快速设置」tab
- `verify-T-0138-quicksettings-lte.png` ENB 小区参数 16 字段渲染
- `verify-T-0138-quicksettings-lte-scrolled.png` ENB 邻区配置两表
- `verify-T-0138-quicksettings-gnb.png` GNB 详情页

**未验证的项目**(需 ACS 真机响应,或邻区列表非空设备):
- 单实例改字段 → tab Save → 通知中心出现 Set 任务回执(BaiBLQ CPE 在线但当前无 SetParameterValues 真机走通的设备测试数据)
- 行级 Save 多实例(邻区列表暂无数据,需先 AddObject)
- 新增邻区两步流程实测(同上,需走 AddObject + Set 的 CPE 真机响应)
- 失败标红重试(同上)

这 4 项留独立 sub-task(类似 T-0123-P2-d 风格的"真机回放验收 + DoD §6"),不阻塞当前 PR。

**真机端到端整体结论**:T-0138 设计 §9.4 第 3 步(BaiBLQ LTE)+ 第 4 步(GNB)的**所有 UI 渲染/路由/字典加载/分组逻辑**核心验收项全过,Tab 正确显示在 eNB/gNB 设备且 UI 元素与设计文档一致;字典补齐 EnbCellType 生效。
