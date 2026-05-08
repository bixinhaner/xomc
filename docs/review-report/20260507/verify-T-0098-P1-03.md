# S4 Verify Report — T-0098-P1-03

| 字段 | 值 |
|------|-----|
| Task | T-0098-P1-03（迁移 `000058_param_dictionary.sql` — param_models + param_mappings + discovered_param_mappings + standard_params + 既有表 ALTER） |
| 分支 | `feature/T-0098-P1-data-dict` |
| 章程 | `docs/methodology/AI承诺对峙清单.md` W3 / `docs/project/参数-KPI-告警-整合-实施计划.md` Phase 1 |
| 设计 | `docs/design/参数-KPI-告警-整合设计方案.md` §1.2.1-§1.2.5 |
| 模式 | wave-batched（Skip S0/S1，per dev-pipeline §C.1） |
| Deps | T-0098-P1-02（products 表必须先存在以挂 FK） |
| 时间 | 2026-05-07 |

## 改动文件清单

| 文件 | 性质 | 行数 |
|------|------|------|
| `omcgo/migrations/000058_param_dictionary.sql` | 新增 | ~155 |

## 出口门核销

| 门 | 命令 | 结果 |
|----|------|------|
| 编号连续性 | `bash omcgo/scripts/check-migrations.sh` | ✅ 58 文件连续 |
| go build | `go build ./...` | ✅ 无输出 |
| migrate up（首次失败） | DO 块缺 `+goose StatementBegin/End` 注解 → SQLSTATE 42601 | ❌ 触发 CLAUDE.md §5.5.1 历史教训 |
| migrate up（修复后） | 加注解，重跑 → 27.93ms | ✅ version=58 |
| schema 验证 | 4 表 + 3 ALTER 列 + 2 跨域 FK | ✅ 详见 §"schema 摘要" |
| migrate down | down 11.73ms；4 表 NULL / fk_products + fk_pdl 全清 / `count=0` | ✅ |
| migrate up（再次） | 24ms 成功 | ✅ |
| 新端点 E/R | — | N/A |
| metric/log grep | — | N/A |

## schema 摘要（运行时验证）

| 对象 | 关键列 | 索引 / FK |
|------|-------|----------|
| `param_models` | id PK / name UNIQUE / total_entries/objects/params / loaded_from / is_active | 索引 1 个；被 param_mappings(CASCADE) / parameter_discovery_log(SET NULL) / products(SET NULL) 三表引用 |
| `param_mappings` | param_model_id FK CASCADE / standard_path / private_path / entry_type CHECK / **is_storable** / is_active | 唯一 (model_id,std_path)；反查 (model_id,priv_path) WHERE active；存储过滤 (model_id) WHERE is_storable |
| `discovered_param_mappings` | id PK / **product_id FK CASCADE** / software_version / standard_path / private_path / 9 元属性 / is_storable | 唯一 (product_id,sw_ver,std_path)；反查 (product_id,sw_ver,priv_path) WHERE active |
| `standard_params` | id PK / standard_path UNIQUE / entry_type CHECK / 元属性 | 索引 1 个 |
| `devices.param_model_id` | UUID nullable，无 FK（分区表沿用 data_model_id 模式） | — |
| `parameter_discovery_log.param_model_id` | UUID FK SET NULL → param_models | idx_pdl_param_model 部分索引 |
| `products.param_model_id` (跨域 FK 收口) | FK SET NULL → param_models | 由 P1-02 留位列 + 本迁移 ADD CONSTRAINT |

## 设计契约要点

1. **跨域 FK 收口**：P1-02 留下 `products.param_model_id UUID` 不加 FK；本迁移在 `param_models` 创建后用 DO 块条件性 ADD CONSTRAINT，规避了"先有鸡先有蛋"。
2. **PARTITION 表无 FK 一致性**：`devices.param_model_id` 与既有 `data_model_id` / 新增 `product_id` 全部不挂 FK；引用完整性由应用层 + ParamRegistry 校验，符合 CLAUDE.md §5.5.3。
3. **discovered 表用 UUID PK + 自然唯一约束**：设计 §1.2.3 描述自然主键为 (product_id, software_version, standard_path)，实施时统一 UUID PK + 唯一索引（与项目其他表风格一致），自然键查询走 UNIQUE INDEX。
4. **CHECK 约束选择性**：仅 `entry_type IN ('object', 'parameter')` 加 CHECK；access/data_type/change_applies 保持 VARCHAR 不约束，以容忍 XML 加载器后续可能扩展的取值（如 BOOLEAN/DATE_TIME 旁路）。
5. **is_storable 在 param_mappings 与 discovered 上语义一致但来源不同**：param_mappings.is_storable 来自 XML store 属性；discovered.is_storable 在交集时复制 default 当前值，**不参与** `device_attrs_override` 覆盖（设计 §1.2.3）。
6. **`+goose StatementBegin/End` 包裹 DO 块**：CLAUDE.md §5.5.1 历史教训复刻 — 第一次跑直接 SQLSTATE 42601；加注解后两次 DO 块均成功执行。
7. **Down 顺序**：先 drop 跨域 FK（products → param_models 与 pdl → param_models）→ 既有表 ALTER DROP COLUMN → 4 张新表反向 DROP（param_models 最后）。

## 不在本任务交付范围（后续接力）

- `param_models` / `param_mappings` 业务数据写入 → P1-06 Loader 从 9 个 paramModel XML 加载（BLQ/MLN/QRTB/MLQ/BM/BSC/BTS/BaiBNQ/ENB_DEFAULT_098/ENB_DEFAULT_181）+ standard-model.xml
- `discovered_param_mappings` 写入 → Phase 2 ParamModel.Intersect（设备 Bootstrap 后 FileType=11 流程）
- Translator handler / Registry / REST API → Phase 2/3

## 已识别后续观察点

| 项 | 描述 | 跟进 |
|----|------|------|
| L1 | 旧 `data_model_id` 列与新 `param_model_id` 在 devices 与 parameter_discovery_log 双轨共存 | 直至 Phase 5 P5-02 drop 旧列 |
| L2 | `is_storable WHERE is_storable` 部分索引稀疏 — 默认 true 多数命中 | 优化空间小，可观察 P2-04 sync 实际查询计划再调 |
| L3 | discovered.id UUID PK 与设计文档"主键改为(product_id,sw,std)"语义不同（实际是唯一约束） | 已在 §"设计契约要点" 3 解释；Phase 2 Translator 查询走 UNIQUE INDEX |

## 结论

**S4 出口门通过**。S5 wave-batched 模式以本 verify-md 为审查证据，可直接进入 S6 commit。
