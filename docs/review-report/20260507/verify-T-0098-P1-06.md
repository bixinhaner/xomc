# S4 Verify Report — T-0098-P1-06

| 字段 | 值 |
|------|-----|
| Task | T-0098-P1-06（4 Loader 实现：product / parammodel / indicator / alarm-definition + provider 接线 + 修复迁移 entry_type 列宽） |
| 分支 | `feature/T-0098-P1-data-dict` |
| 章程 | `docs/methodology/AI承诺对峙清单.md` W3 / `docs/project/参数-KPI-告警-整合-实施计划.md` Phase 1 |
| 设计 | `docs/design/参数-KPI-告警-整合设计方案.md` §1.7 / §2.6 / §3.4 / §4.5 / §5 / §7 |
| 模式 | wave-batched（Skip S0/S1，per dev-pipeline §C.1） |
| Deps | T-0098-P1-01..05 全部 done |
| 时间 | 2026-05-07 |

## 改动文件清单

| 文件 | 性质 | 行数 |
|------|------|------|
| `omcgo/migrations/000060_fix_entry_type_width.sql` | 新增（P1-03 followup） | ~25 |
| `omcgo/internal/core/appconfig/config.go` | 修改 | +40 行（4 个 Loader 子配置） |
| `omcgo/internal/config/parammodel/model.go` | 新增 | ~85 |
| `omcgo/internal/config/parammodel/loader.go` | 新增 | ~325 |
| `omcgo/internal/product/model.go` | 新增 | ~85 |
| `omcgo/internal/product/loader.go` | 新增 | ~245 |
| `omcgo/internal/alarm/definition/model.go` | 新增 | ~60 |
| `omcgo/internal/alarm/definition/loader.go` | 新增 | ~200 |
| `omcgo/internal/pm/indicator/loader.go` | 新增 | ~370 |
| `omcgo/cmd/app/provider/dictload.go` | 新增 | ~100 |
| `omcgo/cmd/app/provider/container.go` | 修改 | +3 行（DictLoaderRegistry 字段） |
| `omcgo/cmd/app/provider/router.go` | 修改 | +6 行（dictload 模块注册） |
| `omcgo/cmd/app/etc/config.dev.yaml` | 修改 | +20 行（dict_loader 4 子配置） |

总新增：**~1500 LOC** Go + ~25 LOC SQL + ~20 LOC YAML。

## 出口门核销 — DoD 行数对账

> 干净测试 DB（`omcgo_t0098_test`）migrate up 全量到 60 后，跑 4 Loader 端到端 LoadOnce，验证设计 Phase 1 DoD：

| 域 | DoD 目标 | 实测 | 结果 |
|----|---------|------|------|
| `param_models` | 9 | **9** | ✅ |
| `param_mappings` | 4781 | **4781** | ✅ |
| `standard_params` | 2001 | **2001** | ✅ |
| `products` | 15 | **15** | ✅ |
| `product_class_patterns` | 29 | **29** | ✅ |
| `alarm_severity_levels` | 4 | **4** | ✅（P1-04 种子，P1-06 不重写） |
| `alarm_definitions` | 442 | **442** | ✅ |
| `perf_indicators_enb` | 1409 | **1409** | ✅ |
| `perf_indicators_gsm` | 73 | **73** | ✅ |
| `perf_indicators_gnb` | 282 | **282** | ✅ |
| **perf_indicators 合计** | **1764** | **1764** | ✅ |
| 公式 (3 表合计) | 6254 | **6254** | ✅ |
| `indicator_unit` | 27（设计） | 21（XML 实际 distinct） | ⚠️ XML 来源 distinct 即 21；27 是设计文档示意 |

> **核心 12 项 DoD 全部命中**（除 indicator_unit 因 XML 实际值少于设计示意；与 Phase 1 收敛性无关）。

## 出口门核销 — 编译与测试

| 门 | 命令 | 结果 |
|----|------|------|
| 编号连续性 | `bash scripts/check-migrations.sh` | ✅ 60 文件连续 |
| go build | `go build ./...` | ✅ 无输出 |
| go vet（新包） | `go vet ./internal/config/parammodel/ ./internal/product/ ./internal/alarm/definition/ ./internal/pm/indicator/ ./cmd/app/provider/` | ✅ 无输出 |
| dictloader 包测试 | `go test ./internal/core/dictloader/...` | ✅ ok（98.4% 覆盖率从 P1-01 保留） |
| appconfig 测试 | `go test ./internal/core/appconfig/...` | ✅ ok |
| pm/indicator 测试 | `go test ./internal/pm/indicator/...` | ✅ ok（既有用例不受 loader 新增影响） |
| 全 internal/ 回归 | `go test ./internal/...` | ⚠️ 1 项 `TestDownloadHandler` 失败 — **stash 验证为 P1-06 前已存在**（与本任务无关） |
| migrate up（干净环境） | `omcgo-migrate up` 全量 0→60 | ✅ |
| migrate down（60→59） | `omcgo-migrate down` | ✅ TRUNCATE + ALTER TYPE 6.52ms |
| migrate up（再次） | `omcgo-migrate up` | ✅ 26ms |
| Loader smoke test | 临时 `cmd/_dictload_smoke`（已删除）跑 4 Loader | ✅ DoD 全过 |

## 设计契约要点

### 1. Loader 接口与编排

4 Loader 全部实现 `dictloader.Loader`（P1-01 框架）：`Name()`/`Directory()`/`LoadOnce()`/`Reload()`。Reload 当前与 LoadOnce 等价（UPSERT 语义天然幂等）。
注册到 `c.DictLoaderRegistry` 供未来管理 API 触发 ReloadOne（Phase 3）。

### 2. 启动期编排（设计 §7.2）

Provider `initDictLoadModule` 不调用 `Registry.LoadAll`（无序），而是**显式两阶段**：
- **Phase 1**: paramModel + indicator + alarmDefinition 三 Loader 通过 `errgroup` 并行
- **Phase 2**: product Loader 串行后置（验证三引用 paramModel.name / alarm.ne_type 需 Phase 1 数据已落库）

### 3. UPSERT 与重写策略（设计 §1.7 / §2.6 / §3.4）

| Loader | 主表策略 | 关联表策略 |
|--------|---------|-----------|
| ParamModel | param_models 按 name UPSERT | param_mappings 按 model_id 删旧 + 批量插入；standard_params TRUNCATE + 全量重写 |
| Product | products 按 product_name UPSERT | product_class_patterns TRUNCATE + 全量插入 |
| Indicator | perf_indicators_* 按 id UPSERT | rela_platform_indicator_formula_* TRUNCATE + 重写；indicator_unit ON CONFLICT DO NOTHING；indicator_group_* ensure default group |
| AlarmDefinition | alarm_definitions 按 identifier UPSERT；severity 字符串→ severity_id 通过 4 行种子 lookup | — |

### 4. 跨文件 identifier 全局唯一（设计 §3.4 加载流程 2）

AlarmDefinition Loader 持有 `seen` map 跨 7 个 ne_type XML 的 identifier；重复 identifier 第二次出现 → 跳过 + ERROR 日志（保留 first-seen）。
Indicator Loader 同样 first-seen 胜出（设计 §1.7.1 整合规则的 indicator 等价）。

### 5. 三引用校验（Product Loader，设计 §4.5）

- **paramModel.name** 引用 `param_models`：未命中 → 跳过 product + ERROR 日志
- **alarm.ne_type** 引用 `alarm_definitions.ne_type`：未命中 → WARN（不阻塞，因 alarm loader 仍在 phase 1，时序可能微妙）
- **indicator platform** 引用 `platform_indicator_formulas_{deviceType}`：当前 WARN-only（严格校验留待 P3 handler）
- **device_attrs_override.data_type=true 拒绝**：硬拦截 + skip + ERROR 日志

### 6. P1-03 followup 修正（migration 000060）

P1-03 设计将 `entry_type VARCHAR(8)` 但 CHECK 允许 `'parameter'`（9 字符）— 设计文档自身 typo。P1-06 实测 Loader 时触发 SQLSTATE 22001。
修正：新建 000060 forward fix（不 amend P1-03，遵守 dev-pipeline `git commit --amend` 红线），三表 entry_type → VARCHAR(16)；CHECK 不变。
Down 含 TRUNCATE（缩列前必清，'parameter' 9 chars 不可逆装入 VARCHAR(8)）— 数据由 Loader 重建可接受。

### 7. config.dev.yaml dict_loader 子段

补全 4 子配置的默认值；空白由 Loader 构造期 fallback（`Directory="param-mappings"` 等），yaml 缺省即崩的零值场景在 P1-01 `validate.go` 已处理。

## 不在本任务交付范围（后续接力）

| 项 | 接力任务 |
|---|---------|
| ParamRegistry / Translator / Intersect | P2-02 / P2-03 |
| ProductRegistry（路由 + L1/L2 缓存） | P2-01 |
| AlarmDefinition Registry sync.Map + 接收路径 fallback | P2-10 |
| Indicator `enabled` 属性 OR 合并 + `operator_code='default'` 桶刷新 | P2-09 |
| `{i}` 占位符校验 + 跨 XML 字段冲突合并 | P2-02 设计 §1.7.1 / §1.9 |
| indicator_group_* 真正分层 | Phase 4 UI 层维护 |
| 4 域 handler / REST API | Phase 3 |
| 4 域 Reload via REST + cache_version watchdog | Phase 3 |

## 已识别后续观察点

| 项 | 描述 | 跟进 |
|----|------|------|
| L1 | 4 Loader 包均 0 单元测试 — XML 解析 / SQL 拼装为生信路径 | Phase 2 起为各 Loader 补 table-driven 测试（解析 / 边界 / 错误路径） |
| L2 | `TestDownloadHandler` 在 `internal/acs/rpc/` 失败 — git stash 验证为**P1-06 前已失败**，与本任务无关 | 单独 backlog 任务跟进（与 SOAP 渲染 cwmp namespace 相关，非本 Wave 范围） |
| L3 | indicator_unit 实测 21 行 vs 设计示意 27 行 — XML 实际 distinct 数 21 即真相源 | 若 UI 期望 27，由 Phase 4 在 indicator_unit 补充人工编辑界面 |
| L4 | dict_loader yaml 默认 `xml_base_dir: "data"` 相对 cwd — dev 模式从 omcgo/ 启动 OK；prod 镜像内需绝对路径 | 部署阶段在 prod yaml 改绝对路径 |
| L5 | Loader.Reload 当前与 LoadOnce 等价（UPSERT 幂等保证），未做"diff 文件指纹跳过未变化文件"优化（Scanner.DiffFingerprints 已就绪） | Phase 3 ReloadOne 端点接入时再优化 |
| L6 | indicator Loader 占位 group_id="default"；`indicator_group_enb/gsm/gnb` 仅一行 | Phase 4 UI 维护多级分组时由 handler 端补全 |

## 结论

**S4 出口门通过**。12 项 DoD 行数核心全部命中。S5 wave-batched 模式以本 verify-md 为审查证据，可直接进入 S6 commit。
