# Code Review: device_info 字段长度同步与 RF 状态容量

## 结论

PASS

未发现 CRITICAL 问题。本次变更限定在 `device_info` 快查列同步和主库 baseline schema，未包含即插即用菜单显示 seed 改动。

## 审查范围

- `internal/device/device_info_pg_repository.go`
- `internal/device/device_info_sync.go`
- `internal/device/device_info_sync_test.go`
- `migrations/000001_init_schema.sql`

## 重点检查

- SQL 安全：新增 `information_schema.columns` 查询为固定 SQL，无用户输入拼接。
- 更新可靠性：同步前按数据库 `varchar` 长度过滤字段，避免单个超长字段导致整条 `device_info` 更新失败。
- 数据完整性：CSV 字段只按完整条目裁剪，普通字符串超长时跳过更新，避免写入半截坏数据。
- 性能影响：字段长度只在 `InfoSyncer` 内按实例缓存一次，运行期仅做 map 查询和字符数检查。
- 迁移一致性：consolidated baseline 中 `rf_status` 扩为 `varchar(64)`，可容纳 9 个小区 RF 状态。

## 风险与建议

- INFO：`InfoSyncer` 字段长度缓存随进程生命周期生效；若运行中调整列宽，需要重启 app 让缓存重新加载。
- INFO：本次仅扩大 baseline schema。当前项目说明该 baseline 面向全新部署或清库重建，不提供既有库增量升级路径。

## 验证

- `go test ./internal/device ./cmd/migrate ./cmd/app/provider` — 通过
- `git diff --staged --check` — 通过
