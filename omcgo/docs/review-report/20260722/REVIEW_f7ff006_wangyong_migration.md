# Review: MML 配置分组 seed 与部署 SQL 更新

日期：2026-07-22

## 结论

PASS

未发现 CRITICAL 问题，可以提交。改动集中在 MML 初始化 seed 与部署更新脚本，属于数据目录配置调整。

## 审查范围

- `omcgo/migrations/seed/000001_init_seed.sql`
- `omcgo/scripts/mml_apply_config_updates_20260721.sql`

## 重点检查

- SQL 幂等性：新增/更新命令和字段使用 `ON CONFLICT` 或软删除更新，重复执行不会产生重复命令。
- 命令归一：IPsec 旧 `LST/MOD MML350_DEVICE_FAP__IPSEC` 在合并到 `SM_SUB_01` 后增加最终兜底软删除，避免界面重复显示。
- 读写规则：LICENSE 最终仅保留查询命令；IPsec MultiIpsecConfigParam 按读写属性加入 `LST/MOD SM_SUB_01`。
- 分组位置：HALOB 与小区服务参数管理（总体）同级，日志命令直接挂在日志参数管理下。
- 路径覆盖：新增 DNS、LICENSE、移动性、日志、IPsec 等标准 TRPath 绑定，并刷新命令 `target_paths/tree_node_refs`。

## 发现

无 CRITICAL。

INFO：
- 本次 SQL 块较大，后续如果继续扩展 MML 分组，建议抽出可复用的标准路径绑定生成脚本，减少 seed 与部署 SQL 双份维护成本。
- 审计 CSV/Markdown 位于仓库外 `/Users/wangyong/OBJECT/Codex/outputs/mml-trpath-audit-20260722/`，已随本地验证刷新，但不属于本次 Git 提交范围。

## 验证

- `go build ./...`：通过
- `go test ./test/integration -run TestSeedBaselineHasOnConflict -v`：通过
- `go test ./internal/mml`：通过
- 本地部署验证：`http://localhost:8081/` 返回 `HTTP/1.1 200 OK`
