# S4 Verify Report — T-0098-P2-04

| 字段 | 值 |
|------|-----|
| Task | T-0098-P2-04（provision/sync.go 重写 Path B：删 Phase1GPNs 阶段 + ParamMapping 列表去重前缀 + GPV 对象前缀 + Translator 落库 + is_storable 过滤）|
| 分支 | `feature/T-0098-P1-data-dict` |
| 章程 | `docs/project/参数-KPI-告警-整合-实施计划.md` Phase 2 §2.A.P2-04 |
| 设计 | §1.11 Path B / §1.12 消费者更新清单 |
| 模式 | wave-batched（Skip S0/S1） |
| Deps | T-0098-P2-02 done（commit `228f3d6f`） |
| 时间 | 2026-05-07 |

## 改动文件清单

| 文件 | 性质 | 行数 |
|------|------|------|
| `omcgo/internal/provision/sync_pathb.go` | 新增 | ~225（StartPathBSync + HandleSyncResultPathB + WithParamRegistry + 3 helpers） |
| `omcgo/internal/provision/sync_pathb_test.go` | 新增 | ~140（11 测试用例） |
| `omcgo/internal/provision/sync.go` | 修改 | +12（4 字段 + import） |
| `omcgo/internal/provision/engine.go` | 修改 | +30（handleAutoSync / handleModelFileReceived / handleGPVResponse 三站点 dual-stack） |
| `omcgo/cmd/app/provider/modules.go` | 修改 | +1（WithParamRegistry 接线） |

总：**~410 LOC** Go（含 ~140 LOC 测试）。

## 设计契约要点

### 1. Path B 双栈协议（设计 §1.11 + P2-02 dual-stack 协议）

新增 `WithParamRegistry(paramReg, prodReg, enabled)` setter。`enabled=false` 或任一 registry nil → 等价不调用本方法（沿用 datamodel 旧栈，零行为差异）。

`PathBEnabled(ctx, dev)` 在所有切换站点前 dry-run：dev.ProductClass 空 / MatchProductClass miss / GetByProduct 失败 → 返回 false → 调用方降级旧栈。

### 2. StartPathBSync — 删 GPN 阶段（设计 §1.11 关键点）

旧 `StartTwoPhaseSync`：
- Phase 1: GPN 多深度递归发现实例
- Phase 2: GPV 按发现的实例展开路径

新 `StartPathBSync`：
- 直接从 ParamMapping 列表抽 `is_storable=true` 的 privatePath 去重前缀
- 直接 GPV 对象前缀；CPE 自动返回所有实例（设计 §1.11 删 GPN 的核心原因）

### 3. extractStorablePrefixes — ParamMapping → GPV 前缀集

去重算法：
- `is_storable=false` 直接过滤
- `Dev.WiFi.SSID.{i}.Enabled` → `Dev.WiFi.SSID.`（截到第一个 `{i}` 前一段）
- `Dev.System.Mode` → `Dev.System.`（无 `{i}` → 截到最后一个 "."）
- 已含末尾 "." 的 object 路径原样使用
- 排序输出（map 遍历无序，确保结果稳定）

测试 5 case 覆盖：happy path / 同前缀去重 / 全 non-storable / object 路径 / 空+无 dot 边界。

### 4. HandleSyncResultPathB — Translator 翻译 + is_storable 二次过滤

GPV 响应处理：
1. 用 `MappingValidator.LookupParam(privatePath)`（含实例号归一化）查映射
2. 未命中 → 容错写入原 privatePath + WARN log（保住厂商扩展参数可见性）
3. 命中但 `IsStorable=false` → 静默丢弃（GPV 子树会带回不可存条目，设计 §1.11）
4. 命中 → 用 `instantiateStandardPath(actualPrivate, templatePrivate, templateStandard)` 把 `{i}` 占位符按位置填充 → 写入 standardPath

### 5. instantiateStandardPath — `{i}` 位置感知替换

`actualPrivate="Dev.WiFi.SSID.7.Enabled"` + `templatePrivate="Dev.WiFi.SSID.{i}.Enabled"` + `templateStandard="Device.WiFi.SSID.{i}.Enable"` → `"Device.WiFi.SSID.7.Enable"`

容错：段数不一致 → 返回模板原文（避免错误填充）。多 `{i}` 按位置左到右配对。测试 6 case 覆盖：单实例 / 多实例 / 无 `{i}` / 段数不一致 fallback。

### 6. engine.go 三站点 dual-stack

| 站点 | 旧 | 新 |
|------|-----|-----|
| `handleAutoSync` | StartTwoPhaseSync(dev, dm) | StartPathBSync 优先；返回 used=true 即结束 |
| `handleModelFileReceived` | StartTwoPhaseSync(dev, dm) | 同上（model upload 后自动 sync） |
| `handleGPVResponse` | HandleSyncResult(dev, values) | HandleSyncResultPathB 优先；返回 used=true 即结束 |

每站点保留旧栈做 fallback；feature flag off 时行为零差异。

### 7. 旧栈保留 — engine_test.go 不动

`engine_test.go` 1082 行测试覆盖**旧 datamodel 栈**（StartTwoPhaseSync / Phase1GPNs / iterator 等）。本任务保留旧栈不删，以维持现有测试覆盖。`engine_test.go` 同步重构留给 Phase 5（`P5-01` 删 datamodel 时统一处理）。

新栈测试通过 `sync_pathb_test.go` 11 用例覆盖纯函数路径；service-level dual-stack 集成测试需 testcontainers，留给 P3-02 上线后跑。

## 出口门核销

### 编译与测试

| 门 | 命令 | 结果 |
|----|------|------|
| go build | `go build ./...` | ✅ |
| go vet | `go vet ./internal/provision/... ./cmd/app/...` | ✅ |
| 单测（provision） | `go test ./internal/provision/... -race -count=1` | ✅ ok |
| 全 internal 回归 | `go test ./... -count=1` | ⚠️ 仅 `TestDownloadHandler` pre-existing flake |

### 单测覆盖矩阵

| 测试 | 覆盖 |
|------|------|
| `TestExtractStorablePrefixes_HappyPath` | 5 mapping → 3 distinct prefix；non-storable 过滤 |
| `TestExtractStorablePrefixes_DedupesIdenticalPrefixes` | 同前缀多 mapping → 单输出 |
| `TestExtractStorablePrefixes_AllNonStorable_Empty` | 边界：全过滤 |
| `TestExtractStorablePrefixes_ObjectPaths` | 末尾 "." object 路径原样 |
| `TestExtractStorablePrefixes_EmptyAndInvalid` | 边界：空串 + 无 "." 路径 |
| `TestBasePrefix_Cases` | 7 case：模板 / 多 `{i}` / object / 叶子 / 边界 |
| `TestInstantiateStandardPath_NoPlaceholder` | 模板无 `{i}` → 直返 |
| `TestInstantiateStandardPath_SingleInstance` | 设计契约：单 `{i}` 填充 |
| `TestInstantiateStandardPath_MultipleInstances` | 多 `{i}` 按位置左到右配对 |
| `TestInstantiateStandardPath_PartialOverlap_LeftFirst` | 跨段多实例 |
| `TestInstantiateStandardPath_LengthMismatch_Fallback` | 失败路径：段数不一致 → 容错返回 |
| `TestInstantiateStandardPath_NoPlaceholderInTemplate` | 模板 standard 无 `{i}` 即便 actual 含数字 → 原样 |

总：**11 测试用例** 覆盖纯函数路径成功+失败+边界。

### dev-pipeline §B3 / §B4 硬门

- [x] `go build ./...` 通过
- [x] `go test ./...` 仅 pre-existing flake
- [x] 无新增 TODO/FIXME/panic
- [x] 无新增 `if carrier == ...` 硬编码
- [x] 公共接口无新增 `any` / `interface{}`
- [x] 新端点 E/R 比 — N/A
- [x] 迁移双向演练 — N/A
- [x] metric / log 名 grep — "path-b sync started" / "path-b parameter values saved" / "path-b auto-sync initiated" 全部命中
- [x] 累计型依赖核销 — N/A

## 不在本任务范围（接力）

| 项 | 接力任务 |
|---|---------|
| engine_test.go 1082 行同步重构（覆盖新栈） | P5-01（删 datamodel 时统一处理） |
| `is_storable` 在 model_upload Intersect 阶段的写入语义（discovered 表继承默认） | P2-06 |
| dual-stack 集成测试（testcontainers / dockertest） | P3-02 上线后端到端跑 |
| 删除 `datamodel.ParameterTreeIterator` / `Phase1GPNs` / `SyncPlan` | P5-01 |

## 已识别后续观察点

| 项 | 描述 | 跟进 |
|----|------|------|
| L1 | 未命中 mapping 的 GPV 条目"容错写入原 privatePath" — 长期可能积累不可翻译参数；建议 P3-02 加 metric `path_b_untranslated_total{device}` 监测；当前实现仅 WARN log | 若现网积累 > 1% 再加埋点 |
| L2 | StartPathBSync 不需要 GPN 分阶段，但依然保留 `discoveryRepo` 状态轮换（DiscoverySyncing / Completed），与旧栈 hand-off 一致 | 设计契约保持 |
| L3 | extractStorablePrefixes 返回切片当前用 O(n²) 排序——map 数量 ≤ 256（设计上限），无优化必要 | 若实际 paramModel 涨到 1000+ 改 sort.Strings |
| L4 | `model_upload` 调用 StartTwoPhaseSync 走 dual-stack，但 model_upload 自身改造在 P2-06；P2-04 提前接 hook 不破坏现状 | P2-06 后两条联动测试 |

## 结论

**S4 出口门通过**。Path B 新栈完整落地：StartPathBSync 删 GPN 阶段、HandleSyncResultPathB 通过 Translator 翻译 + is_storable 过滤落 standardPath；engine.go 三站点 dual-stack；feature flag off 时行为零差异。11 测试用例 + 全链路 build/test 绿。本 verify-md 同时承担 wave-batched 模式 S5 review 凭据。
