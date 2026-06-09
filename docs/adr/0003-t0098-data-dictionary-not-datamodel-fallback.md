# 0003 — T-0098 数据字典重构：下线旧 datamodel 三级回退，改走 product 装配件 + ParamModel 字典 + Translator

## Status

Accepted（追认 retroactive）。决策于 2026-05-07 ~ 05-08 落地（T-0098 wave-3），本目录成立时补记。

## Context

T-0098 前，F02 配置域用一套 `internal/config/datamodel/`（约 5166 LOC，19 个 `.go`）+ `data_model_definitions` 表来解析设备参数，按 **product → oui → carrier_default 三级回退**（`ScopeProduct/ScopeOUI/ScopeCarrierDefault`）选定数据模型。该方案的问题：

- **三级回退语义模糊**：同一设备命中哪级 scope 取决于多张表的覆盖关系，路由结果难预测、难治理。
- **标准路径与厂商私有路径混杂**：模板/下发/校验各处直接面对 `Device.X_VENDOR.` 这类厂商专有路径，运营商差异与厂商差异耦合进业务逻辑。
- **设备实际能力无沉淀**：CPE 实际支持的参数集（FileType=11 上传的 paramModel）没有结构化落库，每次都重新发现。
- 参数（F02）、KPI（F03）、告警（F04）三套字典各自为政，缺统一的"装配件 + 字典 + 启动期加载"范式。

需要一套可预测、可治理、能沉淀设备实际能力、且把"标准路径 ↔ 厂商私有路径"翻译收敛到一处的新模型。

## Decision

以 product **装配件**为中心，重构为字典 + 翻译的三层模型，整建制替换旧 datamodel：

1. **装配件（product）**：`products` 表聚合产品类 / 参数模型 / KPI 平台 / 告警 ne_type / 上传开关；`product_class_patterns` 存 productClass 正则。配 `ParamModel` 字典：`param_models` + `param_mappings`（默认映射）+ `discovered_param_mappings`（设备 FileType=11 上传 XML 后由 IntersectService 按 `product_id + sw_version` 写入）+ `standard_params`（standardPath 元属性参考）。
2. **正则路由替代三级回退**：设备上报 productClass → `ProductRegistry.MatchProductClass`（全局正则，L1 `sync.Map` → L2 Redis → PG）→ product → `param_model_id` → `ParamRegistry.GetByProduct`（优先 discovered 精确匹配 swVersion，退化 default，双源合并为带 `Source` 标记的 MappingSet）。**不再有 oui / carrier_default 回退层**。
3. **双向翻译收敛到 Translator**：standardPath（标准化）↔ privatePath（厂商专有）O(1) 互译；模板/SPV/GPV 输入侧统一用 standardPath，下发/持久化用 privatePath；`{i}` 占位符与运行时实例号 `.N.` 折叠为同一索引键。
4. **统一启动期加载**：`internal/core/dictloader/` 框架编排 product / parammodel / indicator / alarm-definition 四个 Loader，文件白名单 → 单事务幂等 UPSERT，ModuleGraph 编排依赖（dict 并行 → product 串行后置）。
5. **整建制下线旧体系**：删除 `internal/config/datamodel/` 全包 + 改造 9 个消费者（device param handler、interop runner/validator、provision sync/engine/model_upload、provider 接线）；DROP `data_model_definitions` / `data_model_import_log` / `oui_registry` 及 `devices.data_model_id` 等列；前端下线 `datamodelApi.ts` 与 DataModelManagement 页。治理端点（products / param-models / indicators / alarm-definitions）由 `RequireSuperAdmin` 中间件收口。

> `consts.go` 仍保留 `ScopeProduct/ScopeOUI/ScopeCarrierDefault` 仅为向后兼容，**新代码勿用**（见 `omcgo/CLAUDE.md §4.1`）。

## Consequences

**正向**：

- 路由可预测、可治理：productClass 正则 → product 一条链路，配 `/api/v1/products` 全套 CRUD + 匹配顺序 + 孤儿设备 + unknown-stats 治理闭环。
- 设备实际能力沉淀进 `discovered_param_mappings`，按 `product_id + sw_version` 索引，避免重复发现。
- 厂商私有路径只在 Translator 一处出现，业务层只见 standardPath，运营商/厂商差异解耦。
- 参数 / KPI / 告警三库统一为"装配件 + 字典 + dictloader 启动期加载"范式，三库后续演进同构（为 ADR 0004 的三库导入统一打基础）。
- 一次性删掉约 5166 LOC 旧代码 + 4 张废表，技术债大幅收敛。

**代价 / 约束**：

- 一次性大改面（迁移 + 9 消费者重写 + 前端页面下线 + 三库 Loader），靠 P1-P5 分波（22 + 14 条子任务）控制风险。
- discovered 优先、default 退化的双源合并语义需团队理解到位，否则容易误判某参数映射来源。
- 多实例横扩前，字典 Upload/Delete/Reload 走进程内文件锁，需补 PG advisory lock（见 ADR 0004）。

> 后端速记见 `omcgo/CLAUDE.md §4.3`。子任务清单见 `docs/project/backlog/subtasks/T-0098-data-dict.md`，整体设计见 `docs/design/参数-KPI-告警-整合设计方案.md`。
