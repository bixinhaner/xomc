# S4 Verify Report — T-0098-P2-05

| 字段 | 值 |
|------|-----|
| Task | T-0098-P2-05（provision/orchestrator.go 改造 Path A：模板 Parameters JSON key 视为 standardPath；SPV/GPV 步骤生成时翻译为 privatePath）|
| 分支 | `feature/T-0098-P1-data-dict` |
| 章程 | `docs/project/参数-KPI-告警-整合-实施计划.md` Phase 2 §2.A.P2-05 |
| 设计 | §1.11 Path A / §1.12 消费者更新清单 |
| 模式 | wave-batched（Skip S0/S1） |
| Deps | T-0098-P2-02 done（commit `228f3d6f`） |
| 时间 | 2026-05-07 |

## 改动文件清单

| 文件 | 性质 | 行数 |
|------|------|------|
| `omcgo/internal/provision/orchestrator.go` | 修改 | +55（BuildProvisioningStepsTranslated + translateTemplateParameters 容错） |
| `omcgo/internal/provision/orchestrator_pathA_test.go` | 新增 | ~150（10 测试用例） |
| `omcgo/internal/provision/sync_pathb.go` | 修改 | +18（ResolveTranslator 公共方法） |
| `omcgo/internal/provision/engine.go` | 修改 | +14（handleTemplateProvisioning dual-stack）|

总：**~240 LOC** Go（含 ~150 LOC 测试）。

## 设计契约要点

### 1. Path A 双栈（设计 §1.11）

新增 `BuildProvisioningStepsTranslated(tmpl, translator)`：
- `translator == nil` → 等价于 `BuildProvisioningSteps(tmpl)`，零行为差异
- `translator != nil` → Parameters JSON key 翻译 standardPath → privatePath，构建步骤后下发

`extractParameterNames` 从已翻译的 Parameters JSON 提取，故 GPV `parameter_names` 也是 privatePath。

### 2. translateTemplateParameters — 容错优先

- `len(params) == 0 || translator == nil` → 原样返回
- Unmarshal 失败（如非对象 JSON）→ 容错保留原 RawMessage（不报错）
- key 翻译命中 → 用 privatePath
- key 翻译未命中 → 保留原 key（可能是模板存了 privatePath 老数据）

### 3. SyncService.ResolveTranslator — 桥接 SyncService 与 Path A

复用 `resolveMappingSet` 内部逻辑，封装为公共方法 `ResolveTranslator(ctx, dev) (*Translator, bool)`。Engine 调用统一通过此方法获取 translator；任一前置条件失败 → 返回 (nil, false) → 调用方降级旧路径。

### 4. engine.go handleTemplateProvisioning 双栈

```go
if e.syncService != nil {
    if translator, ok := e.syncService.ResolveTranslator(ctx, dev); ok {
        steps, err = BuildProvisioningStepsTranslated(tmpl, translator)
    } else {
        steps, err = BuildProvisioningSteps(tmpl)
    }
} else {
    steps, err = BuildProvisioningSteps(tmpl)
}
```

3 fallthrough 兜底：syncService nil / ResolveTranslator 未解出 / 任意路径出错。feature flag off 时行为零差异。

### 5. 测试矩阵（10 测试用例）

| 测试 | 覆盖 |
|------|------|
| `BuildProvisioningStepsTranslated_NilTranslator_FallbackToOldPath` | translator nil → SPV Params 原样 |
| `BuildProvisioningStepsTranslated_WithTranslator_KeyTranslated` | 命中：SPV key 翻译为 privatePath |
| `BuildProvisioningStepsTranslated_GPVStep_AlsoTranslated` | GPV `parameter_names` 也翻译 |
| `BuildProvisioningStepsTranslated_UnknownKey_PreservedAsIs` | 未命中：保留原 standardPath |
| `TranslateTemplateParameters_EmptyJSON_PassThrough` | 边界：nil + `{}` |
| `TranslateTemplateParameters_InvalidJSON_FallthroughOriginal` | 容错：非对象 JSON 原样 |
| `TranslateTemplateParameters_NilTranslator_PassThrough` | 边界：nil translator |
| `TranslateTemplateParameters_PreservesValues` | 设计契约：翻译 key 不动 value |

## 出口门核销

### 编译与测试

| 门 | 命令 | 结果 |
|----|------|------|
| go build | `go build ./...` | ✅ |
| go vet | `go vet ./internal/provision/...` | ✅ |
| 单测（provision） | `go test ./internal/provision/... -race -count=1` | ✅ ok |
| 全 internal 回归 | `go test ./... -count=1` | ⚠️ 仅 `TestDownloadHandler` pre-existing flake |

### dev-pipeline §B3 / §B4 硬门

- [x] `go build ./...` 通过
- [x] `go test ./...` 仅 pre-existing flake
- [x] 无新增 TODO/FIXME/panic
- [x] 无新增 `if carrier == ...` 硬编码
- [x] 公共接口无新增 `any` / `interface{}` — `map[string]any` 仅本地变量解析 JSON
- [x] 新端点 E/R 比 — N/A
- [x] 迁移双向演练 — N/A
- [x] metric / log 名 grep — 无新增 metric / log，依赖 P2-04 既有
- [x] 累计型依赖核销 — N/A

## 不在本任务范围（接力）

| 项 | 接力任务 |
|---|---------|
| 模板存储已经写过的 privatePath 数据迁移到 standardPath | P3-02（template handler 上线后批量迁移工具） |
| Translator 实例化（`Foo.{i}.Bar` → `Foo.7.Bar`）当模板已含具体实例号 | 现实模板已存具体实例号；template/instances.go 预留 |
| webcode 编辑模板 UI 切换 | P4-04 |

## 已识别后续观察点

| 项 | 描述 | 跟进 |
|----|------|------|
| L1 | 模板 Parameters 形态：本任务假设是 `{"path": value}` 形式；如出现嵌套对象 `{"groups": {...}}`，translateTemplateParameters 容错保留原文（不翻译嵌套） | template handler P3-02 review 时核对所有现网模板格式 |
| L2 | Translator 一次构造对应一个 MappingSet 快照——长任务中如 mapping 变更不会感知；适合 SPV 这种短时间下发 | 设计契约（translator.go 注释明确） |
| L3 | 当 ResolveTranslator 返回 false（productClass miss / GetByProduct 错误）但模板已是 standardPath 形态——会按旧栈下发 standardPath 给设备，设备拒绝。可观测点：SPV 失败率 | 现网 flag off 时不触发；P3 上线后看监控 |

## 结论

**S4 出口门通过**。Path A 模板下发 dual-stack 完成：BuildProvisioningStepsTranslated 翻译 standardPath → privatePath；engine.handleTemplateProvisioning 三 fallthrough 兜底；feature flag off 时零行为差异。10 测试覆盖成功+失败+边界。本 verify-md 同时承担 wave-batched 模式 S5 review 凭据。
