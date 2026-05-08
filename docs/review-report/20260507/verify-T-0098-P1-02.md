# S4 Verify Report — T-0098-P1-02

| 字段 | 值 |
|------|-----|
| Task | T-0098-P1-02（迁移 `000057_products.sql` — products + product_class_patterns + devices.product_id） |
| 分支 | `feature/T-0098-P1-data-dict` |
| 章程 | `docs/methodology/AI承诺对峙清单.md` W3 / `docs/project/参数-KPI-告警-整合-实施计划.md` Phase 1 |
| 设计 | `docs/design/参数-KPI-告警-整合设计方案.md` §4.2.1 / §4.2.2 / §4.2.3 |
| 模式 | wave-batched（Skip S0/S1，per dev-pipeline §C.1） |
| 时间 | 2026-05-07 |

## 改动文件清单

| 文件 | 性质 | 行数 |
|------|------|------|
| `omcgo/migrations/000057_products.sql` | 新增 | ~95 |

## 出口门核销

| 门 | 命令 | 结果 |
|----|------|------|
| 编号连续性 | `bash omcgo/scripts/check-migrations.sh` | ✅ 57 个文件 / 编号 000001→000057 连续 / up-down 标记齐 / 命名合规 |
| go build | `go build ./...` | ✅ 无输出 |
| migrate up（干净环境） | `omcgo-migrate up` 跑全量 56→57 | ✅ 17.27:15 全部 OK，最终 version=57 |
| schema 验证 | `\d products` / `\d product_class_patterns` / `devices.product_id` 存在 | ✅ 三对象 + 5 索引（含部分唯一 sort_order）+ 触发器全部就位 |
| migrate down | `omcgo-migrate down` 回 57→56 | ✅ 6.77ms；`to_regclass('products') IS NULL` / `to_regclass('product_class_patterns') IS NULL` / devices.product_id 列 count=0 |
| migrate up（再次） | 重跑 up 56→57 | ✅ 16.41ms，证明可重复 |
| 新端点 E/R | — | N/A（纯 schema，无路由） |
| metric/log grep | — | N/A（无新指标） |
| 累计型 deps | — | N/A（Deps=空） |

## 设计契约要点

1. **三字典软引用 vs 硬 FK 分层**：
   - `product_class_patterns.product_id → products.id` 用硬 FK CASCADE（同一域强一致）
   - `products.param_model_id` 本迁移仅 UUID 列预留 nullable，硬 FK 由 P1-03 创建 `param_models` 后追加（避免循环依赖）
   - `products.indicator_platform` / `alarm_ne_type` 用 VARCHAR 软引用（设计 §4.2.1 注：跨表字符串维度，硬 FK 反造成依赖混乱；引用完整性由 handler 层校验）
2. **devices.product_id 不加 FK**：`devices` 是 LIST PARTITION BY carrier 分区表（迁移 000003），既有 `data_model_id UUID`（无 FK）模式一致；分区表 FK 行为复杂（CLAUDE.md §5.5.3），路由完整性由 ProductRegistry + 应用层校验。
3. **sort_order 全局唯一**：`uniq_product_class_patterns_global_order` 部分唯一索引（`WHERE is_active`）保证活跃正则跨产品全局有序；is_active=false 行不参与唯一性约束（允许历史/草稿态共存）。
4. **三策略字段默认值与 XML lenient unmarshal 对齐**：
   - `enable_filetype11 BOOLEAN DEFAULT TRUE`、`device_attrs_override JSONB DEFAULT '{}'`、`enable_unknown_alarm BOOLEAN DEFAULT FALSE`
   - `products.xml` 未写 `enableUnknownAlarm` 属性时 Go XML unmarshal 解为 `false`，与 DEFAULT FALSE 一致 — P1-06 Loader 不需特殊处理（实施计划 §P1-06 Notes 已标注）。
5. **触发器 `update_updated_at_column()` 共享**：迁移 000001 已创建该函数；本迁移直接复用，符合 CLAUDE.md §5.5.8 共享函数惯例。
6. **Down 顺序反转**：先 `ALTER devices DROP product_id` → `DROP product_class_patterns`（含 FK）→ `DROP products`，避免 FK 阻塞。

## 不在本任务交付范围（后续接力）

- `products.param_model_id` FK → P1-03 在创建 `param_models` 后用 `ALTER TABLE products ADD CONSTRAINT ...` 追加
- `parameter_discovery_log.param_model_id` 列 → P1-03 ALTER ADD COLUMN（与 param_models 同迁移更聚合）
- `products` 业务数据写入 → P1-06 Loader 从 `data/param-mappings/products.xml` 加载
- handler / Registry / REST API → Phase 2/3

## 已识别后续观察点

| 项 | 描述 | 跟进 |
|----|------|------|
| L1 | `device_attrs_override.data_type=true` 业务上禁止 — 仅靠 handler 层 + UI 拒绝；DB 不强制（CHECK 约束写 JSONB 比较丑陋，不值得） | P3-01 handler 校验 |
| L2 | `indicator_platform` / `alarm_ne_type` 引用完整性靠 handler 校验；P1-06 加载 XML 时已校验存在性，运行时改产品时再校验 | P3-01 handler 实现 |

## 结论

**S4 出口门通过**。S5 wave-batched 模式以本 verify-md 为审查证据，可直接进入 S6 commit。
