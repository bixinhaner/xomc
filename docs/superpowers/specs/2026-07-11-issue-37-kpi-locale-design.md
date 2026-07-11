# #37 KPI 名称本地化设计

## 目标

英文界面及英文导出文件中的 KPI 名称应优先显示指标库 `en_name`；中文保持 `cn_name` 优先。缺失目标语言时回退另一语言，再回退指标编号。CSV 固定字段和维度字段也必须随导出语言切换。

## 根因

1. `GET /pm/kpi/definitions` 固定把 `cn_name` 写入 `display_name`，忽略请求语言。
2. 前端 KPI 目录和候选指标的 React Query 键未包含 locale，切换语言会复用旧缓存。
3. KPI 导出由 Worker 异步执行，创建请求的 `Accept-Language` 没有保存；Worker 无 HTTP 上下文时默认中文。
4. CSV 写入器把开始时间、结束时间、制式、测量对象和维度首列表头硬编码成中文。

## 方案

### 同步 KPI 目录

`ListKPIDefinitions` 根据 `appcontext.GetLocale` 选择显示名：英文 `en_name -> cn_name`，中文 `cn_name -> en_name`。指标 ID 和 `name` 保持不变，避免影响指标查询和模板保存的编号契约。

### 前端缓存

`frontend-core` 的 `useKPIList`、`useAllKPIs` 和 `useIndicatorCandidates` 将当前 app locale 纳入 query key。三套皮肤都通过共享 Hook 自动在语言切换时重新请求本地化目录。

### 异步 CSV 导出

创建导出任务时，Handler 将标准化 locale 写入 task `params` JSON。Runner 从该 JSON 解析 locale 后传给指标名解析器及 CSV 布局；英文导出同时本地化固定字段与设备/产品/频段等维度字段。旧任务、空值或未知值继续回退中文。只扩展已有 JSON 参数，不增加表字段或迁移。

## 验证

1. 后端单测：英文 KPI 定义接口返回 `en_name`，中文路径不回归。
2. 导出单测：带 `locale=en-US` 的任务将英文 locale 传入名称解析，固定字段与维度字段使用英文；旧参数默认中文。
3. 前端测试：KPI 目录/候选 Hook 的 query key 随 locale 变化。
4. Docker 真实复测：使用同时拥有中英文名称的指标，以英文请求创建导出，确认 CSV 固定字段、维度字段和指标字段均为英文；同时验证英文 API 目录返回英文。

## 非目标

- 不修改指标库原始中英文数据。
- 不改变指标编号、公式、单位或任务保存格式之外的业务语义。
