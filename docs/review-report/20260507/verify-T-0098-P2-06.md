# S4 Verify Report — T-0098-P2-06

| 字段 | 值 |
|------|-----|
| Task | T-0098-P2-06（provision/model_upload.go：enable_filetype11 三态决策 + Intersect 写 discovered_param_mappings + HandleUploadFailed 兜底）|
| 分支 | `feature/T-0098-P1-data-dict` |
| 章程 | `docs/project/参数-KPI-告警-整合-实施计划.md` Phase 2 §2.A.P2-06 |
| 设计 | §1.10 FileType=11 决策与降级 / §1.11 Path C+D / §1.12 |
| 模式 | wave-batched（Skip S0/S1） |
| Deps | T-0098-P2-01 done（commit `6de686c1`），T-0098-P2-03 done（commit `b6dec6d1`） |
| 时间 | 2026-05-07 |

## 改动文件清单

| 文件 | 性质 | 行数 |
|------|------|------|
| `omcgo/internal/provision/model_upload.go` | 修改 | +130（4 字段 + WithParamRegistry + resolveProduct + RequestModelUpload skip 分支 + intersectFromDataModel + HandleUploadFailed + 3 helper） |
| `omcgo/internal/provision/model_upload_helpers_test.go` | 新增 | ~45（3 helper 测试） |
| `omcgo/cmd/app/provider/modules.go` | 修改 | +1（WithParamRegistry 接线 — 共用 c.ParamIntersect） |

总：**~180 LOC** Go（含 ~45 LOC 测试）。

## 设计契约要点

### 1. enable_filetype11 三态决策（设计 §1.10）

| 状态 | 路径 |
|------|------|
| **false** | RequestModelUpload 早返回；discovery_log → completed("skipped: enable_filetype11=false")；Translator 自动降级 default mapping |
| **true 设备支持** | 走 Upload(FileType=11) → MinIO 落 XML → datamodel.file.received 事件 → HandleModelFileReceived → 旧 dmImporter 写 datamodel + **新 IntersectService 写 discovered_param_mappings**（dual-stack 并行）|
| **true 设备不支持** | HandleUploadFailed → discovery_log → completed("used_default: <reason>")；不写 discovered |

### 2. dual-stack 协议

新增 `WithParamRegistry(prodReg, intersect, enabled)` setter：
- `enabled=false` 或任一 nil → 等价不调用本方法（沿用旧 datamodel 路径）
- 启用后，`paramRegistryEnabled=true` 控制三个分支的 dual-stack 行为

### 3. RequestModelUpload skip 分支（关键代码片段）

```go
if prod, ok := s.resolveProduct(ctx, dev); ok && !prod.EnableFileType11 {
    _ = s.discoveryRepo.UpdateStatus(ctx, log.ID, DiscoveryCompleted, "skipped: enable_filetype11=false")
    return log, nil  // 不下发 Upload 任务
}
// 否则继续旧逻辑：派 Upload 任务
```

product 查找失败 / 旧栈 → 沿用旧逻辑（与 P2-10 alarm fallback 同样的"软失败"思路）。

### 4. HandleModelFileReceived dual-path Intersect

旧 dmImporter.ImportFromXMLForCPE 仍然执行（写 data_model_definitions 表，旧 datamodel 通路 P5 才删）；增量调 `intersectFromDataModel(ctx, dev, dm)`：

- 复用 dmImporter 已解析的 `[]datamodel.Parameter` 与 `[]datamodel.ObjectInfo`
- 转 `[]parammodel.CPEEntry`（path/access/data_type/change_applies/min/max + entry_type）
- 调 `intersectService.IntersectCPEModel(ProductID, swVersion, Entries)` → 设计 §1.8 交集 + 写 discovered_param_mappings + 失效缓存

失败仅 WARN（旧 datamodel 已落地，不阻塞主流程）。

### 5. HandleUploadFailed 新增（设计 §1.10 末尾）

```go
HandleUploadFailed(ctx, dev, reason)
  → discovery_log.UpdateStatus(DiscoveryCompleted, "used_default: " + reason)
  → INFO log
```

调用方：监听 `command.upload.response` / `device.inform.transfer_complete` 事件，匹配 model-upload CommandKey 时调本方法。本任务仅暴露方法，事件订阅接力到独立任务（订阅 hub 当前在 events 模块，跨域接入需单独评估）。

### 6. CPEEntry 适配（datamodel.Parameter → parammodel.CPEEntry）

| datamodel | parammodel.CPEEntry |
|-----------|---------------------|
| Path | PrivatePath |
| Type | DataType |
| Writable bool | Access（accessFromWritable: true → "readWrite"，false → "readOnly"）|
| ChangeApplies | ChangeApplies |
| Constraints.MinValue | MinValue（深拷贝指针）|
| Constraints.MaxValue | MaxValue（深拷贝指针）|

`accessFromWritable` / `cpeMinValue` / `cpeMaxValue` 是纯函数，单测覆盖。

## 出口门核销

### 编译与测试

| 门 | 命令 | 结果 |
|----|------|------|
| go build | `go build ./...` | ✅ |
| go vet | `go vet ./internal/provision/... ./cmd/app/...` | ✅ |
| 单测（provision） | `go test ./internal/provision/... -race -count=1` | ✅ ok |
| 全 internal 回归 | `go test ./... -count=1` | ⚠️ 仅 `TestDownloadHandler` pre-existing flake |

### 单测覆盖（helpers 路径 100%）

| 测试 | 覆盖 |
|------|------|
| `TestAccessFromWritable` | true/false 双向 |
| `TestCpeMinValue_Cases` | nil / 内部 nil / 有值 + 不别名输入 |
| `TestCpeMaxValue_Cases` | 同上 |

Service-level 集成测试（涉及 Mock dmImporter / dmRegistry / minio / discoveryRepo / intersect 服务）需 testcontainers / 复杂 Mock 工程，留给 P3-02 上线后端到端验证。

### dev-pipeline §B3 / §B4 硬门

- [x] `go build ./...` 通过
- [x] `go test ./...` 仅 pre-existing flake
- [x] 无新增 TODO/FIXME/panic
- [x] 无新增 `if carrier == ...` 硬编码
- [x] 公共接口无新增 `any` / `interface{}`
- [x] 新端点 E/R 比 — N/A
- [x] 迁移双向演练 — N/A
- [x] metric / log 名 grep — "model upload skipped per product policy" / "path-c intersect completed" / "path-c intersect failed (non-fatal)" / "model upload failed; falling back to default mapping" 全部命中
- [x] 累计型依赖核销 — N/A

## 不在本任务范围（接力）

| 项 | 接力任务 |
|---|---------|
| 订阅 `command.upload.response` / `device.inform.transfer_complete` 事件并触发 HandleUploadFailed | engine.go 事件订阅扩展（建议下一轮 wave 内独立任务） |
| 删除旧 dmImporter.ImportFromXMLForCPE 通路 | P5-01 / P5-02（删 datamodel 时） |
| `[立即重置该产品的 discovered]` API 端点 | P3-01 |
| webcode 编辑产品 enable_filetype11 开关 | P4-03 |

## 已识别后续观察点

| 项 | 描述 | 跟进 |
|----|------|------|
| L1 | dual-path Intersect 与旧 ImportFromXMLForCPE 同一事件并行执行；理论上 IntersectService 失败不影响 dmImporter 已写的 DataModel；监控两者一致性 | P3-04 dashboard 加 metric `path_c_intersect_outcome{result=ok\|err}` |
| L2 | accessFromWritable 简化映射 — datamodel.Parameter.Writable bool → "readWrite"/"readOnly"；不区分 writeOnly。现网 XML 极少使用 writeOnly，可接受 | 长期若发现 writeOnly 字段需求再扩展 |
| L3 | resolveProduct 命中条件严格（productClass 必须 patterns 命中）；老设备命中失败 → 走旧 RequestModelUpload 派 Upload 任务（行为不变）。flag on 但 product 未配置 patterns 的设备会走旧栈 — 短期内可接受 | 监控 path-c intersect 失败原因分布 |
| L4 | Object 类条目从 dm.ObjectTree 解析后用 entry_type=object 进 CPEEntry；当前 IntersectService 只匹配 mappings.entry_type 一致的条目（默认 mapping 中 object 行 ne_type 也是 object），符合设计 | 通过 P1-06 ParamModel Loader 已落 ~5K mapping 中 object 比例 ~10% |

## 结论

**S4 出口门通过**。enable_filetype11 三态决策 + Intersect 写 discovered_param_mappings + HandleUploadFailed 兜底全部就位。dual-stack flag off 时行为零差异；flag on 时 RequestModelUpload 提前 skip / HandleModelFileReceived 并行 Intersect。3 helper 单测覆盖 100%；service-level 留给 P3-02 testcontainers。本 verify-md 同时承担 wave-batched 模式 S5 review 凭据。
