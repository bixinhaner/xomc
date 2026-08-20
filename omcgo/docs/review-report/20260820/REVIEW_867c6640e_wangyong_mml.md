# 代码审查报告：MML 新增对象参数与 Keepalived/VRRP 命令修复

- 日期：2026-08-20
- 基线：`867c6640e`
- 范围：MML 命令派生、参数模型、基线种子、现网更新脚本与 V1 控制台命令选择弹窗
- 结论：PASS_WITH_WARNINGS

## 变更概述

- ADD 命令继承当前新增实例层的可写字段，同时排除更深层多实例子对象字段；RMV 继续仅使用 `target_object`。
- 命令派生器、运行期 seed importer 与 SQL 生成器统一写入 ADD 的 `sub_fields` 和可写 `target_paths`。
- 修复 BSC Keepalived/VRRP 对象及参数访问权限，恢复查询、修改、新增、删除命令与参数绑定。
- 更新 consolidated seed 和现网配置更新脚本，按 MOD 的有效可写字段补齐多实例 ADD 参数；无 MOD 对应命令才回退到标准可写字段。
- MML V1 命令选择弹窗同时展示 ADD 目标对象和可写参数，并过滤非可写 sub-field；RMV 保持无参数确认流程。

## 审查结果

### CRITICAL

无。

### WARNING

- `go test ./...` 未全绿：`internal/alarm/definition.TestBuiltinAlarmLibraries_GSMDefinitionsAreOwnedByGSM` 仍断言内置告警 XML 为 8 个，但 `origin/main` 已包含 9 个。该目录和测试文件相对 `origin/main` 无差异，判定为与本次 MML 改动无关的既有基线问题。
- Docker Compose 镜像重建因 Docker Hub 基础镜像下载长期停滞而中止；已使用本地通过的生产构建产物热更新并重启现有 web 容器。该方式适合本地验证，但不替代 CI 的干净镜像构建。

### INFO

- 本次没有可识别的关联 Issue；`issue_274_seed_test.go` 是既有回归测试文件名，近期同范围提交也记录为用户现场连续修复。
- 当前会话未提供 in-app Browser 控制接口，因此未自动点击登录后的 MML 页面；HTTP、生产构建、类型检查和组件交互测试均已覆盖。
- 参数目录变化折回 `migrations/seed/000001_init_seed.sql`，没有新增 `000002+` seed 迁移，符合当前未封版基线规则。

## 检查项

- ADD 实例层级：按 `{i}` 数量只绑定本次创建实例的直接可写字段，不把 `VirtualIpList.{i}` 等子表字段混入父对象 AddObject 后续写入。
- 权限口径：Keepalived/VRRP 的命令字段由 BSC 有效参数映射筛选；只读计数、源/目的 IP 与 Virtual IP 地址不会进入 MOD/ADD 可写集合。
- SQL 安全：更新脚本使用固定值表、JOIN 和 UPSERT，没有字符串拼接 SQL；重复执行路径由唯一键冲突处理保持幂等。
- 数据一致性：标准对象、产品映射、命令、sub-field、`target_paths` 和 `tree_node_refs` 在同一事务内更新。
- 前端行为：ADD 必须有可写参数才能确认并将过滤后的 sub-fields 映射为执行参数；RMV 继续允许仅凭目标对象确认。
- 安全与范围：未发现凭据、环境文件、构建输出或无关改动进入提交范围。

## 验证

- `cd omcgo && go build ./...`：通过。
- `cd omcgo && go test ./internal/config/parammodel/... ./internal/mml/... ./cmd/tools/gen_seed_sql/...`：通过。
- `cd omcgo && go test ./...`：除既有告警库数量断言外，其余执行包通过；失败与本次 diff 无关。
- `cd omcmb && npm run typecheck`：通过。
- `cd omcmb/webcode && npx eslint src/pages/mml/Console/components/CommandSelectModal.tsx src/pages/mml/Console/components/CommandSelectModal.test.tsx`：通过。
- `cd omcmb && npm test --workspace webcode -- --run src/pages/mml/Console/components/CommandSelectModal.test.tsx`：18 项通过。
- `cd omcmb/webcode && npm run build`：通过。
- `xmllint --noout` 校验 BSC 与 standard-model XML：通过。
- Keepalived/VRRP SQL 更新块在本地 PostgreSQL 中以 `BEGIN`/`ROLLBACK` 执行：通过，未保留数据变化。
- 本地 web 静态产物热更新并重启后 `http://localhost:8081/`：HTTP 200，nginx 启动日志正常。
- `git diff --check`：通过。
