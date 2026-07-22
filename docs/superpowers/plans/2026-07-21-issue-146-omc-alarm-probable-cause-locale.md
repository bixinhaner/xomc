# Issue #146 OMC 告警可能原因国际化修复实施计划

> **供代理执行者：** 必须使用 `superpowers:test-driven-development` 按任务逐项实施；先用数据库查询层测试证明语言选择规则，再修改 SQL。不得覆盖设备上报的原始可能原因。

**目标：** 告警源为 OMC 的告警在中英文环境下按告警库显示对应语言的“可能原因”，并覆盖活动告警、历史告警、详情和导出；设备源告警继续保留设备上报原文。

**架构：** OMC 告警的可能原因在查询时基于请求 locale 和告警定义库动态选择，避免持久化时所处语言锁死显示语言；非 OMC 告警仍读取告警记录自身的 `probable_cause`。

**技术栈：** Go、Squirrel、PostgreSQL/TimescaleDB、React API 契约。

---

## 一、问题拆分与原因结论

### 测试服务器现场证据

在 `http://172.17.9.239:8081` 将界面从中文切换到英文后核对当前告警：当前仅有一条设备源 GSM 告警 `60004`，可能原因从中文环境下的展示切换为设备英文原文 `MASTER A link error`。这验证了请求语言会触发重新取数，也验证了非 OMC 告警必须保留设备原文。当前测试数据暂时没有 OMC 源活动告警，因此 OMC 截图问题由 SQL 契约测试覆盖，待新版本部署后再做页面回归。

### 1. 请求语言已经正确传到后端

**结论：已确认，不是前端漏传语言。**

`omcmb/frontend-core/src/services/http.ts` 已根据当前语言设置 `Accept-Language`。活动告警页面的其他字段可以随语言变化，说明请求 locale 链路可用。

### 2. OMC 告警创建时把中文可能原因持久化

**结论：已确认。**

- `omcgo/internal/device/status_reconciler.go` 创建 OMC 断连告警时不直接携带 probable cause。
- 告警过滤/补全逻辑在写入时根据当时 context locale 从告警库补齐 probable cause。
- 后台任务没有用户请求语言，默认中文，因此活动告警表中保存的是中文原因。

写入中文本身不是唯一问题；真正的问题是读取时没有重新按当前请求语言投影。

### 3. 查询层只国际化 description，没有国际化 probable_cause

**结论：已确认，且能闭环解释截图。**

- `omcgo/internal/alarm/pg_store.go` 已通过 `localizedAlarmNameExpr` 对 `description` 按 locale 选择告警定义中的中英文名称。
- 同一查询仍直接选择 `alarms_active.probable_cause` 或 `alarms_history.probable_cause`。
- 因而英语页面中名称可以是英文，可能原因仍是后台任务当初保存的中文。
- 告警定义 id 4、7、23 均存在英文可能原因，数据源并不缺失。

**根因：** OMC 告警查询投影遗漏了 probable cause 的 locale 化，而且持久化文本被错误当成最终展示文本。

## 二、修复方案

新增 `localizedProbableCauseExpr(locale, table)`，查询规则如下：

### OMC 告警

- 英文：`en_probable_cause -> cn_probable_cause -> 告警记录 probable_cause -> ''`。
- 中文：`cn_probable_cause -> en_probable_cause -> 告警记录 probable_cause -> ''`。
- 告警源匹配应对大小写和首尾空格做规范化。

### 非 OMC 告警

- 始终返回告警记录自身的 `probable_cause`，因为这是设备上报原文，不能用 OMC 字典覆盖。

查询时动态选择可以让历史数据立即生效，也能反映告警库后续编辑，不需要数据库迁移或批量回填。

## 三、实施任务

### 任务 1：为本地化 SQL 表达式建立测试

**文件：**

- 修改：`omcgo/internal/alarm/pg_store.go`
- 修改：`omcgo/internal/alarm/pg_store_test.go`

- [x] 先添加英文 OMC 告警优先英文原因的失败测试。
- [x] 添加中文 OMC 告警优先中文原因的测试。
- [x] 添加目标语言缺失时跨语言及持久化值回退测试。
- [x] 添加非 OMC 告警保留原文测试。
- [x] 添加 `omc` 大小写和首尾空格测试。
- [x] 实现 `localizedProbableCauseExpr`，使用固定 SQL 表达式；locale 仅选择预定义列，不拼接用户输入。

建议表达式语义：

```sql
CASE
  WHEN lower(trim(<table>.alarm_source)) = 'omc' THEN
    COALESCE(NULLIF(d.<locale_probable_cause>, ''),
             NULLIF(d.<fallback_probable_cause>, ''),
             NULLIF(<table>.probable_cause, ''), '')
  ELSE COALESCE(<table>.probable_cause, '')
END AS probable_cause
```

### 任务 2：覆盖活动、历史和详情查询

**文件：**

- 修改：`omcgo/internal/alarm/pg_store.go`
- 修改/新增：`omcgo/internal/alarm/pg_store*_test.go`

- [x] 在 `ListActive` 的列投影中替换原始 probable cause。
- [x] 在 `ListHistory` 的列投影中使用历史告警定义维表的中英文原因。
- [x] `GetActiveByID`、`GetHistoryByID` 的用户可见详情使用同一规则。
- [x] 规则仅按 `alarm_source=OMC` 投影；设备源告警与写入模型保持原文。
- [x] 活动告警与历史告警的 scan 列顺序保持不变。
- [x] 实现按 alarm source 通用匹配，不写死告警标识。

定向测试：

```bash
cd omcgo && go test ./internal/alarm -run 'TestLocalizedProbableCause|TestPgAlarmStore.*Locale' -count=1 -v
```

### 任务 3：验证列表、详情和导出共用结果

**文件：**

- 如 API 契约无需改变，前端不修改业务映射。
- 仅在发现导出绕过列表 API 时，修改相应导出服务并补测试。

- [x] `alarmApi.ts` 仍直接消费后端 `probable_cause`，不在前端二次翻译自由文本。
- [x] 导出使用列表返回的 `probableCause`，与列表共用本地化查询结果。
- [x] 测试服务器语言切换后列表立即重新请求并更新展示。

### 任务 4：完整验证

- [ ] `cd omcgo && go build ./... && go test ./internal/alarm -count=1`
- [ ] 若前端有改动：`cd omcmb && npm run typecheck`
- [ ] 英文环境核对所有 `alarm_source=OMC` 的活动告警，可能原因应为英文或按定义回退。
- [ ] 中文环境核对同一批告警，可能原因应为中文。
- [ ] 核对历史告警、告警详情和导出。
- [ ] 核对至少一条设备源告警，其设备上报可能原因在中英文切换时均不被覆盖。
- [ ] 修改告警库英文原因后重新查询旧告警，确认无需回填即可显示新内容。

## 四、不接受的表面修复

- 前端维护中文到英文的文本映射：无法覆盖用户编辑的告警库，也会产生双份字典。
- 批量把活动告警表更新成英文：切回中文仍然错误，并破坏历史原文。
- 对所有告警都使用 OMC 告警库覆盖 probable cause：会丢失设备上报的诊断信息。
- 只修 id 4：issue 明确要求检查所有 OMC 源告警，必须按 alarm source 建立通用规则。
