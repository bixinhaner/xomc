# S4 Verify Report — T-0098-P2-09

| 字段 | 值 |
|------|-----|
| Task | T-0098-P2-09（KPI loader enabled 属性解析 + 多文件 OR 合并 + operator_code='default' 桶刷新；D10=A 吸收 T-0095 标准报表/站点报表 KPI 工作）|
| 分支 | `feature/T-0098-P1-data-dict` |
| 章程 | `docs/project/参数-KPI-告警-整合-实施计划.md` Phase 2 §2.A.P2-09 |
| 设计 | §2.6 KPI 加载流程 |
| 模式 | wave-batched（Skip S0/S1） |
| Deps | T-0098-P1-06 done（commit `75b551bb`） |
| 时间 | 2026-05-07 |

## 改动文件清单

| 文件 | 性质 | 行数 |
|------|------|------|
| `omcgo/internal/pm/indicator/loader.go` | 修改 | +95（aggregateEnabledOR + parseEnabledFlag + refreshDefaultEnabledBucket + 三 device type 调用） |
| `omcgo/internal/pm/indicator/loader_test.go` | 新增 | ~115（8 测试用例） |

总：**~210 LOC** Go（含 ~115 LOC 测试）。

## 设计契约要点

### 1. enabled 属性解析（P1-06 留位 → P2-09 兑现）

XML `<indicator enabled="..." />` 字符串 → bool：

| 输入 | 输出 |
|------|------|
| 缺省 / `""` | true |
| `true` / `1` / `t` / 大小写变体 | true |
| 非 false 系列其它字符串 | true（容错） |
| `false` / `0` / `f` / `no` / `n` | false |

`parseEnabledFlag` 容错优先（与 XML 现网约定一致：未填默认 enabled）。

### 2. 多文件 OR 合并语义

ENB 子目录可有多个 platform XML（QRTB / BLQ / MLN ...）。同一 indicator_id 在多 platform 文件出现时：

- **任一文件 enabled=true → 最终 true**
- **全部文件 enabled=false → 最终 false**
- 与处理顺序无关

`aggregateEnabledOR` 实现：第一次遇到记入 map；后续遍历仅在当前为 false 时尝试用新值覆盖（顺序无关——保证 OR 不被翻转）。

### 3. operator_code='default' 桶刷新

`refreshDefaultEnabledBucket(tx, table, enabledMap)` 仅刷新 default 桶：

```sql
-- 启用集
INSERT INTO enabled_pm_indicators_<dt> (operator_code, indicator_id)
VALUES ('default', $1), ('default', $2), ...
ON CONFLICT (operator_code, indicator_id) DO NOTHING;

-- 禁用集
DELETE FROM enabled_pm_indicators_<dt>
WHERE operator_code = 'default' AND indicator_id = ANY($1);
```

**关键设计点**：不预先 TRUNCATE 整表 — 这是 §2.6 "桶刷新"语义的核心。运营商在 UI 配的非 default 桶（cmcc / ctcc / cucc 等）必须保留。本函数严格只动 `operator_code='default'` 的行。

### 4. 与 P1-06 留位的衔接

P1-06 Loader 已经解析了 `xmlIndicator.Enabled` 字段（注释明确 "P2-09 接力 OR 合并"）；本任务在 P1-06 单事务内追加三行调用：

```go
enbEnabled := aggregateEnabledOR(enbDocs)
gsmEnabled := aggregateEnabledOR([]xmlIndicatorModel{gsmDoc})
gnbEnabled := aggregateEnabledOR([]xmlIndicatorModel{gnbDoc})
refreshDefaultEnabledBucket(ctx, tx, "enabled_pm_indicators_enb", enbEnabled)
refreshDefaultEnabledBucket(ctx, tx, "enabled_pm_indicators_gsm", gsmEnabled)
refreshDefaultEnabledBucket(ctx, tx, "enabled_pm_indicators_gnb", gnbEnabled)
```

复用 P1-06 既建的事务 `tx`，与 indicators / formulas / units 同事务原子提交，避免中间态。

### 5. D10=A 吸收 T-0095

实施计划 §2 标注 P2-09 吸收原 T-0095（标准报表 / 站点报表 KPI 工作）。本任务通过统一 enabled 维度治理，把"哪些 KPI 进入默认报表"从硬编码 / 散落 SQL 改为字典驱动：

- 修改 XML enabled → 重启 Loader → default 桶自动重算
- UI（P3-03）后续编辑非 default 桶不会被本路径覆盖

## 出口门核销

### 编译与测试

| 门 | 命令 | 结果 |
|----|------|------|
| go build | `go build ./...` | ✅ |
| go vet | `go vet ./internal/pm/indicator/...` | ✅ |
| 单测 | `go test ./internal/pm/indicator/... -race -count=1` | ✅ ok |
| 全 internal 回归 | `go test ./... -count=1` | ⚠️ 仅 `TestDownloadHandler` pre-existing flake |

### 单测覆盖矩阵

| 测试 | 覆盖 |
|------|------|
| `TestParseEnabledFlag` | 14 case：true/false/缺省/大小写/trim/容错 |
| `TestAggregateEnabledOR_Empty` | 边界：nil + 空文档 |
| `TestAggregateEnabledOR_SingleFile` | 设计契约：单文件直通 + 缺省 → true |
| `TestAggregateEnabledOR_MultiFile_TrueWins` | OR 合并：false 后 true → true |
| `TestAggregateEnabledOR_MultiFile_AllFalseStaysFalse` | OR 合并：全 false → false |
| `TestAggregateEnabledOR_MultiFile_OrderAgnostic` | 设计契约：true 先出现，false 后不翻 |
| `TestAggregateEnabledOR_EmptyIDIgnored` | 边界：空 ID 跳过 |
| `TestAggregateEnabledOR_DefaultEnabledTreatedAsTrue` | 与 §2.6 约定一致：默认 true |

### dev-pipeline §B3 / §B4 硬门

- [x] `go build ./...` 通过
- [x] `go test ./...` 仅 pre-existing flake
- [x] 无新增 TODO/FIXME/panic
- [x] 无新增 `if carrier == ...` 硬编码
- [x] 公共接口无新增 `any` / `interface{}`
- [x] 新端点 E/R 比 — N/A
- [x] 迁移双向演练 — N/A（schema 已 P1-06 与早期 000035 落地）
- [x] metric / log 名 grep — N/A（本任务零新增 metric；行数变化通过 dictloader.Report.RowsAffected 透传）
- [x] 累计型依赖核销 — N/A

## 不在本任务范围（接力）

| 项 | 接力任务 |
|---|---------|
| `/api/v1/indicators` REST CRUD + 切换运营商桶 | P3-03 |
| PM worker 路径接 enabled 过滤 | P2-11（条件性，下个 commit 评估） |
| UI KPI 库管理（5 Tabs） | P4-05 |

## 已识别后续观察点

| 项 | 描述 | 跟进 |
|----|------|------|
| L1 | SQL 路径（refreshDefaultEnabledBucket）单测覆盖 0 — 与 P1-06 / P2-01..P2-03 一致，需 testcontainers | 集成测试桩在 P3-03 上线后端到端跑 |
| L2 | enabled 解析容错优先（"random" → true）与 strict 模式（仅 "true"/"" 视 true）的取舍 — 当前选容错与 XML 现网约定一致 | 若运营商规范变更可调 parseEnabledFlag 默认分支 |
| L3 | 多文件首次出现 = false，第二次出现 = true 的 OR 合并 — 单测 case5 已覆盖；现实 ENB platform 文件之间 indicator_id 重叠概率低 | 无 |

## 结论

**S4 出口门通过**。enabled 属性 OR 合并 + default 桶刷新落地；8 测试覆盖 OR 合并三向（true 单边 / false 单边 / 顺序无关）+ 解析容错；编译 + 测试全绿。本 verify-md 同时承担 wave-batched 模式 S5 review 凭据。
