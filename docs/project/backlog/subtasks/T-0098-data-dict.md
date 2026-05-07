# T-0098 拆分子任务（参数 / KPI / 告警 数据字典平台化）

> 从 `docs/project/backlog.md` §4.1 拆出（2026-05-07，纯搬运）。
> umbrella 任务 T-0098 的子任务清单；行 schema 与 backlog.md §4 Triaged 主表对齐。
> umbrella 行仍在主 backlog §4，状态联动通过本表 sub-task 的 sprint planning 推进。

### 4.1 T-0098 拆分子任务（36 条，2026-05-07 D1-D10 全采纳推荐后批量登记）

> **来源**：`docs/project/参数-KPI-告警-整合-实施计划.md` §2 任务清单
> **决策记录**：D1=A 保留 KPI 表名 / D2=B DROP+CREATE alarm_definitions / D3=A 保留自实现公式引擎 / D4=A 后端架构产 products.xml / D5=B P1-P3 准入 W3、P4/P5 推迟 / D6 默认分工 F02 主轨+F03/F04 分轨+frontend-core+frontend / D7=A 仅评估 webcode-v2/v3 / D8=A 单 PR 合 P1 迁移 / D9=B feature flag 保留至 P5 / D10=A 吸收 T-0095（详见 §10 changelog 2026-05-07 行）
> **数量说明**：实施计划标题写"33 子任务"，按 §2 实际行枚举 36 条（差额 3 在 P2-11 PM worker 条件性 / P4-08 webcode-v2/v3 评估 / P1-05 D1=A 后保留 docs 任务）
> **Wave 3 准入**：P1-P3（22 条）按 D5=B 视为 W3 内"实现收敛"准入 sprint planning；P4-P5（14 条）标 **D5=B Wave 3 后启动**

| ID | Title | Type | Domain | Prio | Est | Deps | Risk | Notes |
|----|-------|------|--------|------|-----|------|------|-------|
| T-0098-P1-01 | 创建 internal/core/dictloader/（scanner+lifecycle+cache_version+report）+ DictLoaderConfig | feat | infra | P2 | M | — | R-T0098-04 | Phase 1 共享设施层；D8=A 单 PR 合 P1；出口=单测 + provider 框架可注册 Loader 接口 |
| T-0098-P1-02 | 迁移 000057_products.sql — products + product_class_patterns + devices.product_id 列 | feat | F02 | P2 | S | — | R-T0098-04 / ~~R-T0098-11~~ Closed | **2026-05-07 D4 自动满足**：products.xml 9714 bytes / 15 产品 / 29 正则已在 commit 50fa1a0c 入库（实施计划 §1.4 Gap 分析有误）；R-T0098-11 → Closed by reality；P1-02 仅写 schema 迁移 + up/down 配对；scripts/check-migrations.sh 通过 |
| T-0098-P1-03 | 迁移 000058_param_dictionary.sql — param_models + param_mappings(is_storable) + discovered_param_mappings(product_id+swVersion 主键) + standard_params + 既有表 param_model_id 列 | feat | F02 | P2 | M | T-0098-P1-02 | R-T0098-04 | D8=A 单 PR；索引校验；范式重写关键节点 |
| T-0098-P1-04 | 迁移 000059_alarm_dictionary.sql — alarm_severity_levels + 4 行种子 + alarm_definitions 新建 + alarms_active.is_unknown ADD COLUMN | feat | F04 | P2 | M | — | R-T0098-03 | D2=B DROP+CREATE 路径；4 行 severity 种子可查；旧 alarm_libraries 处置见 P5-06 |
| T-0098-P1-05 | KPI 表名对齐文档（保留现状） | docs | F03 | P2 | S | — | R-T0098-02 (closed) | **D1=A 决议保留 indicator_unit/rela_platform_indicator_formula_*/enabled_pm_indicators_*；本任务降级为 docs**：更新设计 footnote 说明命名差异；不再写 000060 迁移 |
| T-0098-P1-06 | 4 个 Loader 实现（product/loader.go + parammodel/loader.go + indicator/loader.go 增强 + alarm/definition/loader.go）+ provider 接线 | feat | infra+F02+F03+F04 | P2 | L | T-0098-P1-01,T-0098-P1-02,T-0098-P1-03,T-0098-P1-04,T-0098-P1-05 | R-T0098-04 / ~~R-T0098-11~~ Closed | 启动后 omcgo-app 各域行数 = 设计基线（参数 9+4781+2001 / 指标 1764+6254+27 / 告警 442+4 / 产品 15+29）；**products.xml lenient unmarshal**：缺 `enableUnknownAlarm` 属性时 Go bool 默认值 false（与 design schema DEFAULT false 一致）；可选一次性补 15 产品显式属性（cosmetic，P1-04/P1-06 同 PR） |
| T-0098-P2-01 | ProductRegistry — productClass 全局正则路由 + L1 sync.Map + L2 Redis 缓存 + 引用校验 | feat | F02 | P2 | M | T-0098-P1-06 | R-T0098-08 | FAP 兜底正则在最末；Phase 2 三轨并行起点 |
| T-0098-P2-02 | ParamModel Registry + Translator — 精简（去 productClass routing）+ O(1) 双向翻译（standardPath ↔ privatePath）+ discovered → default 降级 | feat | F02 | P2 | L | T-0098-P2-01 | R-T0098-05 / R-T0098-08 | D9=B feature flag param_registry.use_new 保护；关键路径节点 |
| T-0098-P2-03 | ParamModel Intersect — 基站上传 paramModel ∩ 默认；按 device_attrs_override 决定元属性来源；写 discovered_param_mappings | feat | F02 | P2 | M | T-0098-P2-02 | R-T0098-08 | 配合 P2-06 model_upload；元属性覆盖策略实现 |
| T-0098-P2-04 | provision/sync.go 重写 Path B — 删 Phase1GPNs 阶段 + 改用 ParamMapping 列表去重前缀 + GPV 对象前缀 + Translator 落库 + is_storable 过滤 | ref | F02 | P2 | L | T-0098-P2-02 | R-T0098-06 / R-T0098-08 | engine_test.go 1082 行同步重构（不留残留）；CPE simulator 兜底 |
| T-0098-P2-05 | provision/orchestrator.go 改造 Path A — 模板 Parameters JSON key 改 standardPath；SPV/GPV 步骤生成时翻译为 privatePath | ref | F02 | P2 | M | T-0098-P2-02 | R-T0098-08 | 模板兼容旧数据评估；可与 P2-04 同 PR |
| T-0098-P2-06 | provision/model_upload.go — enable_filetype11 决策（false 跳过 Upload；true 设备不支持 SOAP Fault → 降级默认映射；调用 Intersect 写 discovered） | ref | F02 | P2 | M | T-0098-P2-01,T-0098-P2-03 | R-T0098-08 | enable_filetype11 三态分支；运营商可配 |
| T-0098-P2-07 | device/device_param_handler.go — dmRegistry → paramRegistry，新 mapping-based validator | ref | F02 | P2 | S | T-0098-P2-02 | R-T0098-08 | 6 消费者改造 simplest case |
| T-0098-P2-08 | interop/runner.go + interop/validator.go — 切到新 ParamRegistry，改校验 ParamMapping 元属性 | ref | F10 | P2 | S | T-0098-P2-02 | R-T0098-08 | F10 模块独立切换 |
| T-0098-P2-09 | KPI loader enabled 属性解析 + 多文件 OR 合并 + operator_code='default' 行刷新语义（不动其他 operator） | feat | F03 | P2 | M | T-0098-P1-06 | — | **D10=A 吸收 T-0095 标准报表/站点报表 KPI 工作**；分桶刷新逻辑核心 |
| T-0098-P2-10 | AlarmDefinition Registry + 接收路径未命中 fallback — 查 device.product_id → product.enable_unknown_alarm；true 写 fallback severity=Warning is_unknown=true，false 丢弃 + INFO + Prom 指标 alarm_unknown_total | feat | F04 | P2 | M | T-0098-P2-01 | R-T0098-03 | enable_unknown_alarm 策略实现；治理闭环准备 |
| T-0098-P2-11 | PM worker 接 KPI Registry enabled 过滤（条件性，operator_code 桶生效） | feat | F03 | P2 | S | T-0098-P2-09 | — | **D10=A 吸收 T-0095**；如适用才做；非必需 |
| T-0098-P3-01 | /api/v1/products 全套（CRUD + patterns 上下移动 + 路由测试 + unknown-stats + 孤儿设备 + cache） | feat | F02 | P2 | L | T-0098-P2-01 | — | Phase 3 最复杂 handler；~20 端点 |
| T-0098-P3-02 | /api/v1/param-models 全套（CRUD + mappings + standard + translate + cache） | feat | F02 | P2 | M | T-0098-P2-02 | — | translate 端点是 Path A 模板用 |
| T-0098-P3-03 | /api/v1/indicators 全套（CRUD + groups + formulas + enabled + units + cache） | feat | F03 | P2 | M | T-0098-P2-09 | — | **D10=A 吸收 T-0095 KPI 管理 API 工作** |
| T-0098-P3-04 | /api/v1/alarm-definitions 全套（CRUD + severity-levels GET 只读 + unknown-stats + cache） | feat | F04 | P2 | S | T-0098-P2-10 | — | unknown-stats 治理闭环关键端点 |
| T-0098-P3-05 | RequireRole("super_admin") 中间件 + 4 路由组接入 + 现有 PrivateRoute 角色扩展 | feat | infra+admin | P2 | S | T-0098-P3-01,T-0098-P3-02,T-0098-P3-03,T-0098-P3-04 | R-T0098-12 | 启动前与 admin 模块 owner 对齐 super_admin vs admin 边界 |
| T-0098-P4-01 | frontend-core：拆出 paramModelApi.ts + 新建 productApi.ts + 拆出 alarmDefinitionApi.ts + 4 个 useXxx.ts Hook | feat | frontend-core | P2 | M | T-0098-P3-05 | — | **D5=B Wave 3 后启动**；P4-08 配套评估 v2/v3 |
| T-0098-P4-02 | navConfig.ts +nav.product 一级菜单 + 4 子项 + i18n 5 项（zh/en）+ super_admin 角色守卫 | feat | frontend | P2 | S | T-0098-P3-05 | R-T0098-12 | **D5=B Wave 3 后启动** |
| T-0098-P4-03 | /product/products 页（最复杂）— 列表 + 抽屉 4 段（基本/字典引用/上传策略/正则）+ 全局匹配顺序浮窗 + 测试匹配 | feat | frontend | P2 | L | T-0098-P4-01 | — | **D5=B Wave 3 后启动**；测试匹配交互核心 |
| T-0098-P4-04 | /product/param-model 页（3 Tabs：参数模型 / 默认映射 / 标准参数树）+ XML 导入 + storable 过滤 | feat | frontend | P2 | M | T-0098-P4-01 | — | **D5=B Wave 3 后启动**；{i} 占位符校验 |
| T-0098-P4-05 | /product/kpi-library 页（5 Tabs：ENB/GSM/GNB/启用/单位）+ 详情抽屉（含全平台公式 CRUD） | feat | frontend | P2 | M | T-0098-P4-01 | — | **D10=A 吸收 T-0095 前端 KPI 管理工作**；**D5=B Wave 3 后启动** |
| T-0098-P4-06 | /product/alarm-library 页（单页 + 详情抽屉）+ 未识别告警频次跳转端点 | feat | frontend | P2 | S | T-0098-P4-01 | — | **D5=B Wave 3 后启动**；is_unknown 过滤 |
| T-0098-P4-07 | /product/orphan-devices 页 — 列表 + 单台/批量绑定 + 触发重新匹配 | feat | frontend | P2 | S | T-0098-P4-01 | — | **D5=B Wave 3 后启动**；orphan 治理 v1 必做 |
| T-0098-P4-08 | webcode-v2 / webcode-v3 兼容性评估（API/Hook 共享，仅看是否因类型/Mock 形态变更崩溃） | td | frontend-core | P2 | S | T-0098-P4-01 | R-T0098-07 | **D7=A 仅评估编译，不强制 UI 完整**；**D5=B Wave 3 后启动** |
| T-0098-P5-01 | 删除 internal/config/datamodel/ 全包（21 文件 + 7 _test.go）+ 删 provision 中残留 iterator 引用 | ref | F02 | P2 | M | T-0098-P4-03,T-0098-P4-04,T-0098-P4-05,T-0098-P4-06,T-0098-P4-07 | R-T0098-01 | **D5=B Wave 3 后启动**；旧 datamodel 5166 LOC 清理 |
| T-0098-P5-02 | 迁移 000061_drop_datamodel.sql — DROP data_model_definitions + data_model_import_log + devices.data_model_id 列 | feat | infra | P2 | S | T-0098-P5-01 | R-T0098-01 | **D5=B Wave 3 后启动**；down section 验证可回滚 |
| T-0098-P5-03 | grep 验证：grep -r datamodel omcgo/internal/ 无业务引用 | td | infra | P2 | S | T-0098-P5-01 | — | **D5=B Wave 3 后启动**；仅历史文档命中视为通过 |
| T-0098-P5-04 | 旧 frontend datamodelApi.ts 删除 + 调整所有引用（拆分到 paramModelApi 完毕后） | ref | frontend-core | P2 | S | T-0098-P4-04 | — | **D5=B Wave 3 后启动**；与 P5-01 同 PR 或独立 PR |
| T-0098-P5-05 | 更新 CLAUDE.md + omcgo/CLAUDE.md 模块清单（datamodel 移除，新增 product/dictloader/parammodel） | docs | process | P2 | S | T-0098-P5-01 | — | **D5=B Wave 3 后启动**；架构文档收尾 |
| T-0098-P5-06 | 旧 alarm_libraries / alarm_library_i18n 处置（D2=B DROP） | ref | F04 | P2 | S | T-0098-P1-04 | R-T0098-03 | **D5=B Wave 3 后启动**；D2=B 决议 DROP；XML 重载即恢复 |

---

← 返回 [`docs/project/backlog.md`](../../backlog.md) §4 Triaged
