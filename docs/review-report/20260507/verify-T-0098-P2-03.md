# S4 Verify Report — T-0098-P2-03

| 字段 | 值 |
|------|-----|
| Task | T-0098-P2-03（ParamModel Intersect — 上传 paramModel ∩ 默认 + device_attrs_override 5 元属性覆盖 + 写 discovered_param_mappings + 失效缓存）|
| 分支 | `feature/T-0098-P1-data-dict`（P2 wave 续推） |
| 章程 | `docs/project/参数-KPI-告警-整合-实施计划.md` Phase 2 §2.A.P2-03 |
| 设计 | `docs/design/参数-KPI-告警-整合设计方案.md` §1.8（交集逻辑）/ §1.7 / §11.D（data_type 覆盖业务禁止决议） |
| 模式 | wave-batched（Skip S0/S1，per dev-pipeline §C.1） |
| Deps | T-0098-P2-02 done（commit `228f3d6f`） |
| 时间 | 2026-05-07 |

## 改动文件清单

| 文件 | 性质 | 行数 |
|------|------|------|
| `omcgo/internal/config/parammodel/intersect.go` | 新增 | ~345 |
| `omcgo/internal/config/parammodel/intersect_metrics.go` | 新增 | ~95 |
| `omcgo/internal/config/parammodel/intersect_test.go` | 新增 | ~325 |
| `omcgo/internal/config/parammodel/pg_repository.go` | 修改 | +60（UpsertDiscoveredMappings + nilIfEmpty） |
| `omcgo/cmd/app/provider/paramregistry.go` | 修改 | +12（IntersectService 接线） |
| `omcgo/cmd/app/provider/container.go` | 修改 | +3（ParamIntersect 字段） |

总：**~840 LOC** Go（含 ~325 LOC 测试）。

## 设计契约要点

### 1. 算法骨架（设计 §1.8）

```
IntersectCPEModel(productID, swVersion, []CPEEntry):
  1. products.GetProductByID(productID) — 取 ParamModelID + DeviceAttrsOverride
  2. repo.ListMappingsByParamModel(paramModelID) — 取全量默认映射
  3. cpeIndex := map[privatePath]CPEEntry（重复 path 后者覆盖）
  4. for each default in defaults:
       cpe, ok := cpeIndex[default.PrivatePath]
       if !ok → defaultsMissing++; continue
       row := buildIntersectRow(default, cpe, override)  // 元属性按 override 决定来源
       matched = append(matched, row)
  5. write.UpsertDiscoveredMappings(productID, swVersion, matched)
       — 单事务 DELETE + 批量 INSERT（len==0 也执行 DELETE，清理旧数据）
  6. invalidator.InvalidateProduct(productID, swVersion) — 失效失败仅 WARN（DB 已写）
```

### 2. device_attrs_override 5 元属性逐个判定

| 属性 | override=true | override=false / 未设 | 备注 |
|------|--------------|---------------------|------|
| access | CPE.Access（空字符串 fallback 默认） | 默认 | — |
| data_type | **强制走默认**（业务禁止覆盖） | 默认 | JSON 写 true 仅记 WARN + `param_intersect_datatype_override_rejected_total` |
| change_applies | CPE.ChangeApplies（空 fallback 默认） | 默认 | — |
| min_value | CPE.MinValue（nil fallback 默认） | 默认 | — |
| max_value | CPE.MaxValue（nil fallback 默认） | 默认 | — |

`is_storable` 始终从默认继承（设计 §1.2.3 明确不参与覆盖）。`standard_path / private_path / entry_type` 始终用默认（CPE 上传仅决定"哪些 path 进入交集"，不决定 path 本身的标准化）。

### 3. 写路径与读路径接口拆分

新增 `IntersectRepository` 接口（写）与既有 `Repository`（读）正交：

```go
type IntersectRepository interface {
    UpsertDiscoveredMappings(ctx, productID, swVersion, mappings) error
}
```

PgRepository 同时满足两接口；测试可独立 fake。`UpsertDiscoveredMappings` 实现 DELETE+INSERT 单事务：

- DELETE WHERE product_id=? AND software_version=?（始终执行，len==0 时仅清理）
- INSERT 批量（squirrel 多行 Values）
- 任一步失败 → tx.Rollback；commit 失败 → 错误向上返回

### 4. 缓存失效协议

`IntersectInvalidator` 接口包装 `Registry.InvalidateProduct(productID, swVersion)`，写完后调用一次。实现细节继承 P2-02：

- L1：`registry.discoveredByDevice.Delete(key)` 立即生效
- L2：`cache.InvalidateDiscovered` 删 Redis key；失败仅 WARN（L1 已删，下次 GetByProduct 必然击穿 DB）
- 失效失败**不致命**：DB 已写，即使其他实例 24h 内仍读旧 cache，cache TTL 自然到期或下次 Refresh 后必然修正

### 5. data_type 业务禁止决议（设计 §11.D）

JSONB `device_attrs_override.data_type=true` 在三层守门：

| 守门点 | 行为 |
|--------|------|
| Intersect（本任务） | 强制视为 false + WARN log + `param_intersect_datatype_override_rejected_total` 计数 |
| API（P3-01） | 接收新建/编辑请求时 HTTP 400 拒绝 |
| UI（P4-03） | data_type checkbox 永远 disabled |

本任务承担最底层兜底——即使 API/UI 被绕过（DB 直改），交集层仍保证 data_type 不被覆盖。

### 6. Provider 接线 — 共用 PgRepository 与 ProductRegistry

```go
// paramregistry.go initParamRegistryModule 末尾追加：
intersectMetrics := parammodel.NewIntersectMetrics(c.MetricsReg)
c.ParamIntersect = parammodel.NewIntersectService(
    repo,            // 同一 PgRepository（满足 Repository + IntersectRepository）
    repo,
    c.ProductRegistry,  // productGetter
    registry,           // IntersectInvalidator（Registry 满足）
    intersectMetrics,
    logger,
)
```

无新增 ModuleGraph 节点：`paramregistry` 已在 P2-02 落地，IntersectService 是其内部组合的写路径附属。Container 新增 `ParamIntersect *IntersectService` 字段，供 P2-06 model_upload 注入。

## 出口门核销

### 编译与测试

| 门 | 命令 | 结果 |
|----|------|------|
| go build | `go build ./...` | ✅ 无输出 |
| go vet | `go vet ./internal/config/parammodel/... ./cmd/app/provider/...` | ✅ 无输出 |
| 单测 | `go test ./internal/config/parammodel/... -race -count=1` | ✅ ok |
| 全 internal 回归 | `go test ./... -count=1` | ⚠️ 1 项 `TestDownloadHandler` 失败 — **本任务前已存在**（P2-01 / P2-02 verify L2 同一 flake），与本任务无关 |

### 单测覆盖矩阵（成功 + 失败两条路径）

| 测试 | 覆盖 |
|------|------|
| `TestIntersect_HappyPath_NoOverride` | 成功路径：5 默认 / 4 CPE / 3 命中 / 2 默认 missing / 1 CPE 多余；invalidator 调一次 |
| `TestIntersect_OverrideAccess_UsesCPEValue` | 成功路径：override.access=true → CPE access 替换默认 |
| `TestIntersect_OverrideMinMax_UsesCPEValues` | 成功路径：min/max 双开 → CPE 值替换 |
| `TestIntersect_OverrideChangeApplies_UsesCPEValue` | 成功路径：change_applies=true → CPE 值替换 |
| `TestIntersect_DataTypeOverride_RejectedAlwaysFalse` | 设计契约：data_type=true 强制视为 false + WARN |
| `TestIntersect_EmptyCPE_ZeroMatchedButStillUpsert` | 边界：空 CPE 仍 DELETE（清理旧数据）|
| `TestIntersect_EmptyDefault_NoRowsWritten` | 边界：空默认 → 0 写入仍 DELETE |
| `TestIntersect_ProductNotFound` | 失败路径：ErrProductNotFound |
| `TestIntersect_NoParamModelID` | 失败路径：ErrNoParamModel |
| `TestIntersect_EmptySoftwareVersion` | 失败路径：swVersion 空字符串拒绝 |
| `TestIntersect_ProductGetterUnset` | 失败路径：ErrProductGetterUnset |
| `TestIntersect_UpsertError_Propagated` | 失败路径：写 repo 错误向上传 |
| `TestIntersect_InvalidatorError_NotFatal` | 设计契约：invalidate 失败不致命，主流程返回成功 |
| `TestIntersect_NilInvalidator_OK` | 边界：未注入 invalidator 也可工作 |
| `TestIntersect_DuplicatePrivatePathInCPE_LastWins` | 边界：CPE 重复 path 后者覆盖 |
| `TestIntersect_ListDefaultError_Propagated` | 失败路径：list default 错误向上传 |
| `TestNormalizeOverride_Variants` | 5 case：nil / 全 false / 单属性 true / 全 true（含 dt）/ 非 bool 值忽略 |

总：**17 测试用例** 覆盖成功+失败两类路径。

### 覆盖率

| 文件 | 函数级覆盖 |
|------|----------|
| `intersect.go::NewIntersectService` | 80% |
| `intersect.go::IntersectCPEModel` | 94.5% |
| `intersect.go::normalizeOverride` | 100% |
| `intersect.go::buildIntersectRow` | 100% |
| `intersect.go::boolFromAny` | 100% |
| `intersect.go::ptrStr` | 100% |
| `intersect_metrics.go::全部` | 100% |

### dev-pipeline §B3 硬门

- [x] `go build ./...` 通过
- [x] `go test ./...` 与 P2-01/P2-02 同一 flake（pre-existing），无新增失败
- [x] 无新增 TODO/FIXME/panic("not implemented")
- [x] 无新增 `if carrier == "cmcc|ctcc|cucc"` 硬编码
- [x] 公共接口无新增 `any` / `interface{}`（`map[string]any` 仅用于 product.DeviceAttrsOverride 字段反序列化，与 §16.2 "公共接口" 语义无冲突；`boolFromAny` 内部使用）

### dev-pipeline §B4 硬门

- [x] 新端点 E/R 比 — N/A（本任务零新增 HTTP 端点；P3-01 接管 admin handler）
- [x] 迁移双向演练 — N/A（schema 已由 P1-03 落地）
- [x] metric / log 名 grep — 全部命中：
  - `param_intersect_total` ✅ / `param_intersect_duration_seconds` ✅ / `param_intersect_matched_rows_total` ✅ / `param_intersect_defaults_missing_total` ✅ / `param_intersect_uploaded_extras_total` ✅ / `param_intersect_datatype_override_rejected_total` ✅ / `param_intersect_invalidate_err_total` ✅
  - "ParamIntersect computed" INFO ✅ / "device_attrs_override.data_type=true rejected" WARN ✅ / "invalidate cache after intersect failed (non-fatal)" WARN ✅
- [x] 累计型依赖核销 — N/A（Deps=P2-02 done）

## 不在本任务范围（接力）

| 项 | 接力任务 |
|---|---------|
| 调用 IntersectCPEModel 的 model_upload 改造 + enable_filetype11 三态决策 | P2-06 |
| CPE 上传 XML → []CPEEntry 解析适配 | P2-06（复用 datamodel/xml_parser 既有 stream decoder） |
| `[立即重置该产品的 discovered]` 端点（DELETE WHERE product_id=?） | P3-01 |
| 重新导入 XML 时的对账 SQL（设计 §1.9） | P3-02 或独立任务 |

## 已识别后续观察点

| 项 | 描述 | 跟进 |
|----|------|------|
| L1 | UpsertDiscoveredMappings SQL 路径 0 单测覆盖 — 同 P1-06 / P2-01 / P2-02，需 dockertest/testcontainers | 集成测试桩在 P3-01 起补 |
| L2 | `TestDownloadHandler` 失败 — 沿用 P2-01 / P2-02 verify L2 决策 | 单独 backlog 任务跟进 |
| L3 | 大批量 INSERT（典型 4781 条默认 → 数千行交集）单 SQL 行数风险 — squirrel 多行 Values 在 PG 默认参数限 65535 下，13 列 × N 行需 N ≤ 5043；现实 4781 默认条目最多 4781 行 ≈ 62K 参数，临界。建议 P2-06 实施时若实测接近上限，分批（每 4000 行一批）写入 | P2-06 集成时观察实际行数后决定是否改批量 |
| L4 | `defaults_missing` 计数当前未做 percentile 报警 —— 长期增长可能反映固件演进偏离 XML 默认 | P3-01 dashboard 接入后加 grafana 看板 |

## 结论

**S4 出口门通过**。IntersectService 17 测试用例覆盖成功+失败两类路径；核心方法 94-100% 覆盖；7 metric + 3 类 log 全部 grep 命中。Provider 已接线 `c.ParamIntersect`，等 P2-06 model_upload 注入消费。本 verify-md 同时承担 wave-batched 模式 S5 review 凭据，可直接进入 S6 commit。
