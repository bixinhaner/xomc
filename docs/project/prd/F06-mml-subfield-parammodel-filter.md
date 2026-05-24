# PRD — MML 控制台按 paramModel 过滤 sub_field + fanout 硬拒（T-0170）

| 项 | 值 |
|---|---|
| Backlog | T-0170 |
| Type / Prio | fix / P1 |
| Sprint / Owner | sprint-12 / Claude |
| Est | M (~1d) |
| Deps | T-0098 ✅ (standard_params + param_mappings schema) + T-0123 ✅ (sub_field 关联) |

---

## 1. 业务背景

### 用户真机复现（2026-05-24）
MML 控制台对 BLQ 设备 `1202000240194DP0026` 执行 `LST DEVICE_INFO_SW_UPGRADE`：
```
[Server] Invalid Parameter Names [3], including: Device.DeviceInfo.SwUpgrade.FailureCause
```
即 BAICELLS BLQ CPE 在 SOAP Fault 9005 中明确告知：「我不实现这 3 个节点」。

### 根因 — 设计哲学未落地到代码

用户主张（正确）：
- `standard_params` = 全局 IETF 标准字典，**不应**有 `is_supported` 字段
- `param_mappings` 按 paramModel 存"该 product 实际支持的 path"
- **缺映射 = product 不支持**（语义内在）

代码现状（**bug**）：
- `internal/mml/admin_repository.go:251 ListEnrichedByCommand` SQL 只 `JOIN standard_params`，**未 JOIN param_mappings** → 返全集
- handler `GET /mml/commands/:id/sub-fields` **不接 device_id/param_model_id 参数**
- `cmd/app/provider/mml_adapters.go:96` `Translator.ToPrivate` Found=false 时 **静默 passthrough** → CPE 9005

后果：BLQ paramModel 实际没有 SwUpgrade 映射（mapping_count=0），但前端仍向用户展示"可勾选"+ 后端仍下发原 standardPath → CPE 拒。

## 2. 用户故事

### 2.1 网管运维 — 不勾选"必失败"的命令
> 作为运维，**当我选 BLQ 设备并打开 LST DEVICE_INFO_SW_UPGRADE 命令时，前端应该显示空 sub_field 或灰显**（因为 BLQ paramModel 没这些映射），而不是让我误以为能勾选、提交后 30 秒等到 CPE 9005。

### 2.2 SRE — 后端硬拒不让"必失败"任务下发
> 作为 SRE，**当用户绕过前端（或 admin 手工调 API）提交了 BLQ 设备 + SwUpgrade 命令时，后端 fanout 应该返 422 拒绝**，明确告知"该 path 在 paramModel 不支持"，而不是默默 passthrough 让 CPE 拒。

## 3. 验收标准（Given-When-Then）

### GWT-1（sub_field 端点按 paramModel 过滤）
- **Given** BLQ paramModel 无 `Device.DeviceInfo.SwUpgrade.*` 映射
- **When** `GET /api/v1/mml/commands/<LST DEVICE_INFO_SW_UPGRADE_id>/sub-fields?device_id=<BLQ_device_uuid>`
- **Then** 响应 `items: []`（0 条 sub_field）

### GWT-2（不传 device_id 兼容现有 admin 视图）
- **Given** admin 想看命令的"全部 sub_field 模板"（无设备过滤）
- **When** `GET /api/v1/mml/commands/<cmd_id>/sub-fields`（不传 device_id）
- **Then** 响应全集（向后兼容）

### GWT-3（fanout 硬拒 mapping miss）
- **Given** 用户绕过前端直接 POST execute-statements，含 BLQ 不支持的 path
- **When** translateTaskPaths 检测 mapping miss
- **Then** 整 task **拒绝** 入 fanout，返 HTTP 422 + 错误信息含具体不支持的 path 列表

### GWT-4（前端命令树 + sub_field 联动）
- **Given** 用户选了 BLQ 设备，打开 LST DEVICE_INFO_SW_UPGRADE
- **When** 前端调 useCommandSubFields(commandId, deviceId)
- **Then** sub_field 列表空 → CommandTree 的 RightPanel 显示"该命令在当前设备上不可用（无任何参数在 paramModel 中支持）"，"执行"按钮禁用

### GWT-5（admin 视图不受影响）
- **Given** admin Tab 3 编辑命令的 sub_field 关联
- **When** admin 端点 `GET /admin/commands/:cid/sub-fields`
- **Then** 仍返全集（不按 paramModel 过滤；admin 维护 catalog 模板）

## 4. 运营商差异矩阵

**无运营商差异**。三家共用 paramModel 字典体系。

## 5. 非目标

- **不做命令树叶子隐藏**（Bug 2）：可选 UX 优化，本期只做 sub_field + fanout（核心修复）；命令树用户仍能看到，但点开后 sub_field 空 + 执行按钮禁用 = 等价于隐藏
- **不动 standard_params schema**（不加 is_supported；按用户哲学）
- **不动 BLQ.xml**（不补 SwUpgrade 映射；因为 BLQ 物理不支持）
- **不修 patterns**（T-0142 独立 task；本期与之正交）

## 6. 依赖

无未结依赖。

## 7. 风险

| ID | 风险 | 缓解 |
|---|---|---|
| R-NEW-T0170-1 | sub-fields 端点签名变化破坏前端 | device_id 设为**可选** query param；不传走老 SQL 全集兼容 |
| R-NEW-T0170-2 | fanout 硬拒可能误伤"暂时无 mapping 但其实支持"的真实场景 | 硬拒前先看 product_resolved；只有 product 命中 + mapping miss 才拒；orphan 走 T-0168 现状 |
| R-NEW-T0170-3 | param_mappings 大表 JOIN 性能 | param_model_id + standard_path 已有索引；SQL EXPLAIN 验证 |

## 8. 实施清单

### 后端（5 处）
1. `internal/mml/admin_repository.go::ListEnrichedByCommand` 加 `paramModelID *uuid.UUID` 参数 + SQL 条件 JOIN
2. `internal/mml/console_service.go::GetCommandSubFields` 加 paramModelID 形参 + device→paramModel 反查
3. `internal/mml/console_handler.go::GetCommandSubFields` 端点接 `?device_id=<uuid>` query
4. `cmd/app/provider/mml_adapters.go::TranslateForDevice` mapping miss 返 `ErrPathUnsupported`
5. `internal/mml/console_validate.go::translateTaskPaths` 捕获 ErrPathUnsupported → 整 task 拒绝

### 前端（3 处）
6. `omcmb/frontend-core/src/services/api/mmlApi.ts::getCommandSubFields` 加 deviceId 形参
7. `omcmb/frontend-core/src/hooks/api/useMmlConsole.ts::useCommandSubFields` 同步
8. `omcmb/webcode/src/pages/mml/Console/components/CommandTree.tsx:600` 调用点传 deviceId

### 测试
9. `admin_repository_test.go` 新增 paramModel 过滤用例
10. `console_validate_test.go` 新增 ErrPathUnsupported 用例

### 验证
11. 真机重跑 BLQ LST DEVICE_INFO_SW_UPGRADE → 前端 sub_field 空 + 按钮禁用；后端 hot-path 422
12. 真机重跑 BLQ LST DEVICE_INFO（有完整映射）→ 正常工作

---

**版本历史**: 2026-05-24 v1.0 PRD + S2 设计备忘合并起草。
