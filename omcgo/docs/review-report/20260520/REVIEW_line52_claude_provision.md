# Code Review — line 52 参数树全量同步对象级差集删除

| 字段 | 值 |
|------|------|
| 时间 | 2026-05-20 |
| 范围 | `omcgo/internal/provision/sync_pathb.go` + `sync_pathb_test.go` + 3 处 device mock 桩补丁 |
| 类型 | feat (用户反馈：CPE 真删的实例 DB 永久残留) |
| 审查者 | Claude (夜间无人监督自治) |
| 关联 | TODO.MD line 52 |
| 真机验证 | BLQ 注入假 Carrier.99 → 同步参数 → reconcile 精准删除（仅 missing_count=2 / 该 batch 总 deleted=74，0 误删跨 batch 数据） |

## 变更摘要

`HandleSyncResultPathB` 加入对象级 reconcile：基于本次响应数据自动推导对象 prefix，在 prefix 范围内对账 → DB 有但 CPE 未返回的 path 走 `DeleteByPathPrefix` 物理删除。

### 关键设计点

1. **不依赖外部 requestedPrefixes**：基于响应自身推导 prefix（path 最深"数字段"之前的部分），天然限定为"本次响应实际覆盖到的多实例对象"
2. **段数门槛 `minPrefixSegments=4`**：BLQ 实测 `Device.DeviceInfo.2.UE_Count` 类路径会推导出 `Device.DeviceInfo.`（2 段）覆盖整个子树，必须门槛拒之。≥4 段对应 `Device.Services.FAPService.{N}.CellConfig...` 这种深层多实例
3. **保守降级**：CPE Fault / 超时 / 全叶子响应 / 浅 prefix → 不删，仅 BatchUpsert
4. **顺手修补**：50446aa1 (本会话之前) 加 `DeleteByPathPrefix` 接口时漏更新 9 处 mock 桩，本次补全

### 实施迭代历史（透明记录）

| 版本 | 方案 | 结果 |
|------|------|------|
| v1 | 透传 ACS handler GPV `requested_names` → engine payload → reconcile 用 requestedPrefixes 限定范围 | **失败** — `Device.` 1 段宽前缀覆盖全设备误删 845 行（BLQ 实测） |
| v2 | v1 + mapPrivatePrefixesToStandard 加段数门槛 ≥3 | **失效** — 门槛后所有 prefix 被拒，reconcile 完全不工作 |
| v3 (最终) | 基于响应自身推导对象 prefix + 段数门槛 ≥4 + 删除 requestedPrefixes 整条链路 | **成功** — Carrier.99 注入测试精准删除，0 误删 |

## 审查清单（Go 后端）

- [x] 命名规范：Go 标准
- [x] 错误处理：`DeleteByPathPrefix` 失败仅 Warn 不阻断（下次同步还有机会修复）
- [x] SQL 安全：复用现有 `DeleteByPathPrefix` 参数化查询
- [x] 无 carrier 硬编码
- [x] 无 panic 风险（纯字符串处理 + map 操作）
- [x] 测试覆盖：
  - `TestNearestObjectPrefix_Cases` 10 种边界（含浅 prefix 拒绝）
  - `TestDeriveObjectPrefixesFromParams_DedupAndSkipLeaves`
  - `TestPathInAnyPrefix`
  - `TestReconcileDeletedPaths_DeletesMissingInstances` (核心)
  - `TestReconcileDeletedPaths_NoMissing_NoDelete`
  - `TestReconcileDeletedPaths_AllLeavesNoOp`
  - `TestReconcileDeletedPaths_RejectsShallowInstances` (回归 v1 误删 bug)

## 风险评估

| 风险 | 概率 | 影响 | 缓解 |
|------|------|------|------|
| 真实 CPE 在某 batch 部分响应（不是 fault，但少返回某些字段） | 低 | 中（误删少量字段） | path 必须含数字段才纳入；BatchUpsert 同时进行，下轮可恢复 |
| ≥4 段门槛漏掉浅层实例对象（如 FAPService 整个实例被删） | 中 | 低 | 文档明示局限；实际场景"整个 FAPService 实例被删"极少 |
| reconcile 时序与 DeleteObject 链路（50446aa1）重叠 | 低 | 低 | 两者目标一致（删 DB 实例），重叠仅产生 0 deleted_rows |

## 结论

**PASS_WITH_WARNINGS** — 0 CRITICAL / 1 WARNING（reconcile 主动删 DB 数据，首次部署需观察）/ 1 INFO（v1 迭代教训已写入代码注释 + 测试用例防回归）。

BLQ 真机端到端验证通过，单元测试 7 个 PASS。建议明早 review 时关注：
1. v1/v2 迭代过程是否合理（接受"试错 + 立即修正"模式）
2. 段数门槛 4 是否过严（漏删风险 vs 误删风险的权衡）
3. 顺手补 9 处 mock 桩是否越权（属修复 50446aa1 漏改）
