# 审查报告：北向 job/result 参数值返回与载荷精简（issue #344）

- 日期：2026-08-20
- 分支：`fix/344-northbound-task-result`（基线 867c6640e）
- 范围：`omcgo/internal/northbound/`（legacy_facade、param_sync_facade、pageconfig 契约、测试）、`omcgo/internal/paramsync/`（model、pg_repository、service）、`omcmb/webcode/.../NorthboundPageConfig/index.tsx`
- 关联：GitLab Issue #344（任务结果查询返回路径元数据而非基站实际参数值）

## 变更概要

1. `job/result/{jobId}` 参数同步分支新增 `parameters`（路径→值）与 `total`：
   - 主路径：新增 `ListRunValues` 只读查询 `parameter_sync_staging_values`，按 requested_paths + coverage 冻结映射匹配（标准↔私有换算、`{i}` 实例化、实例隔离）；
   - 回退路径：暂存无值且 run/request 已成功时，取设备当前参数（`GetDeviceParameters`）过滤后返回；
   - full-sync（无 requested_paths）前置短路，避免拉取全量暂存值/设备参数。
2. 响应载荷精简为最小集：`jobId/name/status/sn/errorMessage/createTime/completeTime/parameters/total`；移除 `coverage`、`items`、`values`、重复 ID（task_id/run_id/request_id/source_id）、重复状态（legacy_status/request_status/result_code/error_code）、内部元数据（mapping_source/mapping_version/sync_scope/trigger_reason）与 5 个计数字段。
3. 页面配置契约（default_configs.go）与前端接口文档示例（index.tsx）同步更新。

## 审查发现

### CRITICAL：无

### WARNING

1. **`parameters` 键冲突即覆盖**：两条暂存值换算到同一输出键时后者覆盖前者（map 语义）。当前数据模型下同 run 同路径唯一，实际不可触发；如未来暂存表放宽唯一性需改为聚合。已接受。
2. **typecheck 门禁未执行**：本 worktree 未安装 `omcmb/node_modules`（`tsc` exit 127）。tsx 改动为纯字符串字面量替换（responseExample/responseFields），与既有条目同构，无类型面变化；合入前建议在有依赖的环境补跑 `npm run typecheck --workspace webcode`。

### INFO

1. full-sync 成功任务现在返回空 `parameters`（原行为同样不含值，无回退）；接口定位是路径级查询（#344 场景），符合预期。
2. `legacy_status` 数字码及映射函数已删除；对接方改用 `status` 新词汇表（succeeded/failed/cancelled 为终态）。属破坏性变更，但该字段为本分支前一版本新增、未发布，且 #344 明确要求返回实际值而非元数据。
3. paramsync 集成测试 `TestCompletionProjectorContinuesAfterOneRunFails` 直连本地共享 PG（`TEST_PG_URL` 缺省 localhost:5432），在本地栈运行时会被线上写入的 run 干扰而偶发失败；与本变更无关（本变更未触碰 projector）。

## 验证记录

| 项 | 结果 |
|---|---|
| `go build ./...` | 通过 |
| `go vet ./internal/northbound/` | 通过 |
| `go test ./internal/northbound/`（全部北向 handler 用例） | 通过 |
| `go test ./...` 全量 | 唯一失败 `alarm/definition` GSM 告警库计数，main 既有（GSM.xml 由已提交 83e3f7cdf 引入，测试未同步），与本变更无关 |
| 本地栈实测（worktree 构建 app 容器） | 成功（10 字段）/staging 主路径/设备参数回退/UUID 与 source_id 两查询/错误格式（400 invalid task_id、401 token 缺失与过期）全部符合预期；实例隔离（FAPService.2 不泄露）验证通过 |
| 影响面实测 | job/result UFTE 旧任务分支返回原完整格式；device/query、user/users、log/page、job/result/page、device/status、device/group、device/register/page 均 200/ret=1 |

## 结论

**PASS**（含 2 项 WARNING，均为可接受/环境性）。代码作用域精确限定在 job/result 参数同步分支；SQL 走 squirrel 参数化、错误均带上下文包装、资源关闭（rows.Close/rows.Err）完整；新增只读查询无写路径风险；测试覆盖成功/等待/映射换算/实例隔离路径。
