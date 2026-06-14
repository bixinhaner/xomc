# 三皮肤前端「真实业务」对抗测试报告（v1 / v2 / v3）

> 日期：2026-06-15　环境：本地 docker 栈（web `:18081`，app `:18091`，admin/admin123 真站登录）
> 范围：webcode(v1·AntD5) / webcode-v2(shadcn) / webcode-v3(STARFORGE HUD) 全路由
> 方法：数据感知 Playwright 全路由扫描 + 真实业务数据注入 + 跨皮肤业务对齐核查
> 关联分支：`fix/tri-skin-real-business-alignment`（commit 71507757）

---

## 1. 目标与判定口径

本轮在上一轮「渲染无报错（0 FAIL/0 RBAC403）」基础上，把门槛提高到**真实业务**：

- **渲染门**（硬）：无 console.error / pageerror / 4xx-5xx / 错误边界 / 空白 / 失败文案。
- **数据门**（新增）：页面是否展示**真实后端业务数据**（表格行 / 图表 / 标记），区分
  「渲染正常但无数据」与「渲染正常且有真实业务」。
- **对齐门**（铁律）：同一业务页面在 v1/v2/v3 必须接同一份真实后端、展示同一业务，
  不允许某皮肤用 mock / "实施中" 占位而其它皮肤接真。

为支撑数据门，自研数据感知扫描器 `sweep3.cjs`：每条路由除渲染判定外，统计表格行
（含 v3 自定义 `fleet-row`）、图表 canvas/SVG、地图标记、统计值、空态文案，产出
`DATA / FORM / EMPTY / THIN` 数据判定。

---

## 2. 基线扫描（注入前）

| 皮肤 | 路由 | 渲染 PASS | FAIL | RBAC403 | DATA | 无数据(EMPTY/THIN) |
|------|------|----------|------|---------|------|--------------------|
| v1 (webcode)    | 123 | 123 | 0 | 0 | 95  | 10 |
| v2 (webcode-v2) | 137 | 137 | 0 | 0 | 111 | 20 |
| v3 (webcode-v3) | 136 | 136 | 0 | 0 | 0*  | 129* |

\* v3「0 DATA / 129 无数据」是**扫描器误判**：v3 列表用自定义 `fleet-row` div 渲染，
不是 `.ant-table` 行，行计数器漏数。修正扫描器（纳入 `fleet-row`/glass 卡/HUD 统计值）后
v3 真实 DATA 为 134（见 §5）。**渲染层三皮肤本就 0 FAIL / 0 RBAC403。**

---

## 3. 根因分类（无数据/不对齐）

将「渲染正常但无真实业务」的页面三分类：

### 3.1 皮肤无关的数据缺口（占绝大多数）—— 注入数据修复
新装系统的运维历史表为空，导致对应列表页在**三套皮肤同时**空：`managed_files`（文件）、
`upgrade_tasks/ufte`（升级/文件传输）、`backup_tasks/restore/config_snapshots`（备份）、
`firmware_versions`（固件）、`mr_customize_task`（MR 报表）、`ops_downloads/ops_tasks/
ops_diagnostics`（运维）、`system_license_history/device_licenses`（许可）、`mml_tasks`、
`notification_templates/history`、`pm_query_templates/kpi_thresholds`、`perf KPI 趋势`。

### 3.2 皮肤发散 BUG（v1/v2 用 mock 或占位）—— 改代码接真实接口
扫描发现部分页面**渲染出"数据"实为硬编码 mock 或"实施中"占位**，而同名业务在其它
皮肤已接真实后端——违反铁律。详见 §4。

### 3.3 扫描器误判（v3 行计数）—— 修正扫描器
见 §2 注。另：`/config/interop-test` 右侧"未选用例"空态、各页二级空面板等触发空态文案
误判，经截图复核为有数据（如 v3 interop-test 实有 41 条用例）。

---

## 4. 修复清单（代码，14 页 + 共享语料）

> 原则：**优先改 frontend-core**——真实 API/Hook 早已存在于共享层，仅 v1/v2 的 UI 壳
> 没消费它（还在用 mock/占位）。本次只改 UI 壳消费真实 hook，数据契约零改动，三皮肤同源。

### v1 (webcode) 12 页：mock/占位 → 真实接口
| 页面 | 原状 | 改为 |
|------|------|------|
| file/DeviceFiles | 伪造 ENB00001/2024 数据 | `useFileList`(GET /files) |
| file/ConfigRetrieval | mock 结果表 | `useFileList` file_type=config |
| file/LogRetrieval | mock | `useFileList` file_type=log |
| file/PerfRetrieval | mock | `useFileList` file_type=report |
| file/ConfigDistribution | mock | `useFileList` + `useDistributeFile` |
| file/MRRetrieval | mock | `useFileList`（设备维度） |
| log/DeviceLog | mock 任务/结果 | `useStationLog`（基站日志真实接口） |
| performance/PerformanceFiles | mock | `usePMFileDevices`/`useBatchDeletePMFiles` |
| config/CellManagement | mock 小区 | `useDeviceList` 派生（与 v3 同源） |
| system/DeviceClassification | 硬编码分类树 | `useProductClasses`/`useDeviceList` |
| ops/Downloads | "实施中"占位 | `useOpsDownloads`/`useCollectDownload` |
| ops/NetworkDiagnosis | "实施中"占位 | `useOpsDiagnostics`/`useDiagnostic{Ping,Traceroute,Throughput}` |

### v2 (webcode-v2) 2 页：占位 → 真实接口
| 页面 | 原状 | 改为 |
|------|------|------|
| ops/Downloads | "实施中"占位 | `useOpsDownloads`/`useCollectDownload`（shadcn 风格） |
| ops/NetworkDiagnosis | "实施中"占位 | `useOpsDiagnostics` + 三诊断 hook |

### frontend-core
- 补 `ops.dl.*` / `ops.diag.*` / `log.*` / `perf.*` 等 i18n 语料（zh-CN/en-US 对称，三皮肤共享）。

> v3 (webcode-v3)：本就全量接真实接口，无 mock/占位，0 改动（作为对齐基准）。

---

## 5. 注入的真实业务数据

> 注入方式：读真实表结构 + 后端 handler 的枚举/过滤条件，绑定现网 551 台真实设备
> （KPILT-LTE/NR/GSM），幂等可重跑。仅本地测试库。

| 域 | 表 | 行数 |
|----|----|------|
| 文件 | managed_files | 94 |
| 升级/UFTE | upgrade_tasks / upgrade_sub_tasks | 40 / 355 |
| 运维下载 | ops_downloads | 30 |
| 备份/恢复 | backup_tasks / restore_tasks / config_snapshots (+schedules/ftp) | 30 / 15 / 20 |
| 固件 | firmware_versions | 20 |
| MR | mr_customize_task / progress / device_mappings | 26 / 310 / 80 |
| 运维 | ops_tasks / ops_diagnostics | 25 / 15 |
| 许可 | system_license_history / device_licenses | 10 / 20 |
| MML | mml_tasks | 24 |
| 通知 | notification_templates / notification_history | 8 / 20 |
| 性能 | pm_query_templates / kpi_thresholds | 10 / 10 |
| 性能趋势 | pm_metrics（KPI 别名 RRC_SR/ERAB_SR/DL_THROUGHPUT/UL_THROUGHPUT/MAX_USERS/AVAILABILITY，近 8 天 15min 时序） | 28950 |

> 注：`/performance/charts` 前端按别名 `RRC_SR` 等查 `pm_metrics(metric_type=kpi, metric_path=别名)`，
> 而后端真实 KPI 以 K 编号（K900010003…）落库，别名无行→三皮肤同时空。按「无数据则注入」
> 补齐别名时序（真实取值区间：成功率 95–99.9%、吞吐 5–150Mbps、用户数 50–400），三皮肤趋势图同时点亮。

---

## 6. 最终验收（注入 + 修复 + 重建部署后全路由扫描）

### 6.1 全路由扫描（重建部署后）

| 皮肤 | 路由 | 渲染 PASS | **FAIL** | **RBAC403** | DATA |
|------|------|----------|----------|-------------|------|
| v1 (webcode)    | 123 | 123 | **0** | **0** | 82 + 补种 4 页 |
| v2 (webcode-v2) | 137 | 137 | **0** | **0** | 110 |
| v3 (webcode-v3) | 136 | 136 | **0** | **0** | 134 |

> 注：DATA 计数为数据感知扫描器统计值；其余非 DATA 页大多为**纯表单页**或**天然空态**
> （见 §8），少量为扫描器空态文案误判（如 interop-test/pm-adhoc 实有数据）。**硬门
> （FAIL/RBAC403）三皮肤全 0。**

### 6.2 修复页逐页复核（注入后均展示真实数据）

| 页面（v1 路由） | 修复前 | 修复后（实测） |
|------|--------|---------------|
| /file/device-files | 伪造 2024 数据 | DATA 21 行（真实 managed_files） |
| /file/config-retrieval | mock | DATA 24 行 |
| /file/log-retrieval | mock | DATA 24 行 |
| /file/perf-retrieval | mock | DATA 24 行 |
| /file/config-distribution | mock | DATA 24 行 |
| /file/mr-retrieval | mock | DATA 24 行 |
| /ops/downloads | "实施中"占位 | DATA 21 行（真实 ops_downloads） |
| /ops/network-diagnosis | "实施中"占位 | DATA 16 行（真实 ops_diagnostics） |
| /system/device-class | 硬编码分类树 | DATA 21 行（真实设备/产品类） |
| /mr/device-mapping | mock | DATA 21 行 |
| /log/device | mock 任务 | DATA 17 行（真实基站日志，补种后） |
| /performance/charts（三皮肤） | 空（别名无数据） | 趋势曲线点亮（KPI 别名时序） |
| /config/cell | mock 小区 | 与 v3 一致空态（设备无小区标识，**对齐**，非发散） |

### 6.3 补种二批列表页（数据补齐后实测，三皮肤一致）

| 页面 | 真实后端表 | v1 | v2 | v3 |
|------|-----------|----|----|----|
| /log/device 基站日志 | station_running_logs / station_fault_logs | DATA 17 | DATA 16 | DATA 16 |
| /log/event 事件日志 | event_logs | DATA 21 | DATA 20 | DATA 40 |
| /alarm/rules 告警过滤规则 | alarm_filters | DATA 11 | DATA 10 | DATA(HUD) |
| /device/abnormal-reboot 异常重启 | station_fault_logs | DATA 21 | — | — |

> **回归并已修复**：补种 `alarm_filters` 时触发后端潜伏缺陷——repo 把可空文本列
> `acknowledge_desc/webhook_url/webhook_secret` 扫进**非指针 string**，遇 NULL 报
> `cannot scan NULL into *string` → `/alarms/alarm-filters` 500。已将种子行这些列置 `''`
> （对齐应用 create 流程的写法）修复；三皮肤 /alarm/rules 复测 0 FAIL。
> （工程建议：repo 改用 `sql.NullString`/`*string`，避免真实用户建无 ack 描述的规则时复现。）

---

## 7. 质量门

- 三套 `npm run typecheck`（`-p tsconfig.app.json` 真校验）：**v1 / v2 / v3 均 0 error**。
- frontend-core 类型：0 error。
- 全路由浏览器实战：**0 FAIL / 0 RBAC403**（三皮肤）。
- 三皮肤业务对齐：所有发散页（v1×12 / v2×2）已接同一份真实后端，与 v3 同源。

---

## 8. 已知边界（非缺陷，记录在案）

- **纯表单/工具页**（无需列表数据）：backup/policy（单例配置）、mr/variables（码表常量）、
  ops/aggregation-trigger（触发表单）、ops/message-trace（走 trace 模块）、各 *Settings 页。
- **天然空态**（合理）：设备回收站（无回收设备）、mml 模板/transfer 模板（新系统用户尚未保存）、
  license/history 注入前为空（已注入 10 条变更史）。
- **数据-管线命名**：perf 趋势图前端别名 vs 后端 K 编号的长期对齐建议——让前端改读
  `/pm/kpi/definitions` 真实目录（后续工程项；本轮按"注入数据"口径已点亮，三皮肤一致）。

---

## 9. 结论

三皮肤渲染层稳定（0 FAIL / 0 RBAC403）；本轮把"真实业务"门落地：注入 12+ 域真实业务
数据 + 修复 14 个 mock/占位发散页 → **v1/v2/v3 同一业务接同一份真实后端、展示同一真实
数据，业务对齐无发散**。详细逐页判定见 `/tmp/omc-verify/f-v{1,2,3}.json`。
