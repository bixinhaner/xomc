# 614.xlsx 测试问题单核查与建单映射（批次 qa-614）

> 源文件：`docs/test-report-humen/614.xlsx`（V0.5.0，报告人 党晓萍 / 刘高杰，2026-06-11~14）
> 核查方式：58-agent 双重（核查+对抗复核）静态代码核查工作流，2026-06-15
> GitHub 批次标签：[`qa-614`](https://github.com/569423176-sketch/goomc/labels/qa-614)

## 汇总

- 原始 36 行 → 去重合并为 **29 条**问题单
- **25 条建为 GitHub issue**（#356–#380，均 `ready-for-agent`，可并行开发；其中 #373 标 `needs-triage` 待 2G 产品决策）
- **4 条经核查确认代码已修复**，不再建单（见下表「已修复」）

| 状态分布 | 数量 |
|---|---|
| 仍存在 STILL_BUG | 13 |
| 疑似(需运行复现) | 7 |
| 部分存在(残留) | 3 |
| 设计建议 | 1 |
| 已修复(不建单) | 5* |

*5 条 LIKELY_FIXED 中 #25(License) 因 prod/k8s 配置仍缺 public_endpoint 残留运维隐患，单独建为 #377；其余 4 条纯已修。

## 建单明细

| qa-614# | GitHub | 状态 | 优先级 | 层 | 标题 |
|---|---|---|---|---|---|
| #1 | [#356](https://github.com/569423176-sketch/goomc/issues/356) | 设计建议 | P3 | fullstack | design(pm): KPI 视图导出改直接下载，任务面板就地下载消除切页摩擦 |
| #2 | [#357](https://github.com/569423176-sketch/goomc/issues/357) | 部分存在 | P3 | frontend | fix(frontend): 三皮肤系统配置「设备 Inform 周期」对齐——v1 已修文案/分组，v2/v3 缺结构化心跳表单且裸 KV 无说明 |
| #4 | [#358](https://github.com/569423176-sketch/goomc/issues/358) | 疑似(需复现) | P3 | backend | fix(alarm): 离线设备活动告警清理阈值硬编码 1h 且统计口径不区分离线——窗口内离线设备告警仍计入活动统计 |
| #6 | [#359](https://github.com/569423176-sketch/goomc/issues/359) | 疑似(需复现) | P1 | fullstack | fix(dashboard): 首页 4/5G KPI 折线图空白——取数只读全网预聚合表(pm_adhoc_aggregation_results/hourly)，而性能仪表板可回退原始 pm_metrics |
| #7 | [#360](https://github.com/569423176-sketch/goomc/issues/360) | 疑似(需复现) | P2 | fullstack | fix(dashboard): 首页设备状态分布柱状图默认不显示在线/告警数据（v1/v2 取决于数据态空+v3 缺真实图用写死假数） |
| #8 | [#361](https://github.com/569423176-sketch/goomc/issues/361) | 仍存在 | P1 | fullstack | fix(device): 设备列表『告警级别』列恒显示『无』——前端 mapBackendDevice 硬编码 none + 后端列表未聚合 alarms_active（di.alarm_severity 仅 Radisys 自报、无人回写） |
| #9 | [#362](https://github.com/569423176-sketch/goomc/issues/362) | 仍存在 | P2 | backend | fix(device): 4G 设备列表"发射功率"被 universalInformMapping 的 MaxTxPower 覆盖 carrier 的 ReferenceSignalPower，与 LMT 口径不一致 |
| #10 | [#363](https://github.com/569423176-sketch/goomc/issues/363) | 仍存在 | P2 | fullstack | fix(pm): 自定义聚合任务 15min×设备组组合无前置校验，创建成功运行期才失败 |
| #11 | [#364](https://github.com/569423176-sketch/goomc/issues/364) | 疑似(需复现) | P1 | backend | fix(pm): BSC/BM 站 KPI 文件上报但解析不出数据——SN 提取方言缺失 + product_class 精确正则未命中致 KPI 路由空 |
| #12 | [#365](https://github.com/569423176-sketch/goomc/issues/365) | 仍存在 | P2 | fullstack | feat(ufte): 统一文件传输任务/模板管理补齐 2G(GSM) 升级分类与制式匹配链路 |
| #14 | [#366](https://github.com/569423176-sketch/goomc/issues/366) | 部分存在 | P2 | frontend | fix(frontend): v2/v3 皮肤 KPI 查询/模板页未国际化，英文环境下仍显示中文（v1 已正确） |
| #15 | [#367](https://github.com/569423176-sketch/goomc/issues/367) | 仍存在 | P3 | frontend | style(ufte): 文件传输任务详情 Descriptions 字段加 nowrap/ellipsis 防换行并将详情区字号下调至 12 |
| #16 | [#368](https://github.com/569423176-sketch/goomc/issues/368) | 仍存在 | P3 | frontend | refactor(transfer): v1 文件传输中心合并 4G/5G 升级为「设备升级」页签并删除与底部下拉框重复的模板子页签 |
| #17 | [#369](https://github.com/569423176-sketch/goomc/issues/369) | 仍存在 | P2 | fullstack | fix(software): IMAGE 版本上传「产品类型标识」收敛到共同标识，隐藏 SC/DC/CA 载波变体级别 |
| #18 | [#370](https://github.com/569423176-sketch/goomc/issues/370) | 仍存在 | P3 | frontend | fix(dashboard): 修复 v1 设备数量统计刷新时闪现硬编码占位数字（1284/1137/43/7）+ KPICard loading 死参数 |
| #19 | [#371](https://github.com/569423176-sketch/goomc/issues/371) | 仍存在 | P1 | fullstack | fix(software): 5G 基站版本回退子任务起始时间未记录、目标版本列空且 v1 列标签错配 |
| #20 | [#372](https://github.com/569423176-sketch/goomc/issues/372) | 部分存在 | P2 | fullstack | fix(software): 固件 IMAGE 导入失败时 v1 toast 透出真实失败原因 + 后端 firmware Create 映射 23505→409 友好文案 |
| #21 | [#373](https://github.com/569423176-sketch/goomc/issues/373) | 仍存在 | P2 | fullstack | feat(ufte): 任务创建中心补 2G/GSM 升级任务入口（含 gsm_upgrade 分类与软件升级引擎 GSM 分支，先做产品决策） |
| #22 | [#374](https://github.com/569423176-sketch/goomc/issues/374) | 疑似(需复现) | P2 | fullstack | fix(device): BM(3GMS+6LTE) 设备概览/快速设置漏显第 6 个 LTE 小区（前端 InUse 过滤无兜底） |
| #23 | [#375](https://github.com/569423176-sketch/goomc/issues/375) | 仍存在 | P3 | frontend | fix(frontend): 设备列表日志收集跳转 /transfer/center 未 openTab，v1 激活页签标题误留在「设备列表」 |
| #24 | [#376](https://github.com/569423176-sketch/goomc/issues/376) | 仍存在 | P1 | backend | fix(backup): BM 站(FAP/BU1810)配置备份永远走 XML 兜底，NV 路径不可达（附带 MLN 前缀 underscore 漂移） |
| #25 | [#377](https://github.com/569423176-sketch/goomc/issues/377) | 已修复 | P2 | fullstack | fix(backup): License 下载预签名 URL 走 minio.public_endpoint，避免泄漏 minio:9000 内网主机名（prod/k8s 配置补缺） |
| #27 | [#378](https://github.com/569423176-sketch/goomc/issues/378) | 疑似(需复现) | P1 | backend | fix(device): 回收站批量恢复因 serial_number 部分唯一索引冲突致整批 UPDATE 回滚（单台偶发/批量必现 500） |
| #28 | [#379](https://github.com/569423176-sketch/goomc/issues/379) | 疑似(需复现) | P1 | fullstack | fix(software): 升级文件管理 IMAGE/BM 固件导入失败——重复版本回 500 应改 409 + 产品类下拉缺 BM 且不可手填 + v2 皮肤无导入入口 |
| #29 | [#380](https://github.com/569423176-sketch/goomc/issues/380) | 仍存在 | P3 | frontend | fix(alarm): 当前告警确认/清除弹窗 showCount 字数计数器与底部确认按钮重叠 |

## 已确认修复（不建单，留档）

| qa-614# | 模块 | 现象 | 核查结论 |
|---|---|---|---|
| #3 | 设备列表/离线 | 100s无心跳应离线但5min仍在线 | 设备离线判定按配置阈值生效（#203/PR #282 已修，含 text*interval 42883 修复） |
| #5 | PM文件/目录 | KPI文件平铺当日目录,建议SN子目录+文件数 | PM 文件已按 SN 分组聚合并显示文件数（GET /pm/files/devices） |
| #13 | 产品管理/KPI指标 | 导入新产品KPI后下拉不刷新需重登 | 导入 KPI XML 后下拉绑定字典源+导入即刷新免重登（#241） |
| #26 | 当前告警/导出 | 告警导出表头底色=字体色文字不可见 | 当前告警导出已统一为纯 CSV(无单元格样式)，底色=字体色症状结构上无法复现 |

---
_自动生成自 QA 核查工作流；问题单全文与逐条根因/修复方案/验收标准见各 GitHub issue。_
