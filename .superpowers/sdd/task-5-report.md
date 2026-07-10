# Task 5 实现报告：事务化 TXT 脚本导入服务

## 实现内容

- 新增 `ScriptImportService`，提供 `ValidateScriptImport`、`CreateScriptFromImport` 和 `ReplaceScriptFromImport`。
- 校验阶段复用纯 TXT Parser 与权威 `ScriptImportValidator`；语法/命令/设备错误只返回预览，不生成保存令牌；离线等 warning 可生成令牌并在会话中保存完整权威结果。
- 保存阶段只接受验证令牌和脚本元数据，服务端从 Redis 会话恢复规范化内容、SHA-256、校验版本、计划项和校验摘要。
- 保存严格执行 Claim → PostgreSQL 写入 → Finalize；数据库失败或校验错误执行 Release，不消费令牌。
- 通过 `import_session_id` 查询和 Redis consumed tombstone 支持网络重试幂等，不重复创建脚本。
- 新增 `ImportedScriptRepository` 及 PostgreSQL `CreateImported`、`GetByImportSessionID`、`ReplaceImported`、`UpdateMetadata`；替换使用 `id + updated_at` 乐观锁，版本冲突返回 `ErrScriptVersionConflict`。
- consumed tombstone 保留会话快照，使数据库已提交但客户端未收到响应时可安全恢复同一脚本。

## 测试覆盖

- 解析/校验错误阻止令牌创建和脚本创建。
- warning 允许创建令牌并保存完整校验结果。
- 空脚本名称、数据库失败 Release、不 Finalize。
- 服务端权威字段一致性（客户端无法提供 content/commands/plan_items）。
- 重复确认（活动会话和已 consumed 会话）只返回同一脚本。
- PG repository 导入查询、元数据更新、替换及 stale `updated_at` 冲突。

## 验证命令

```text
cd omcgo && /usr/local/go/bin/go test ./internal/mml -run 'TestScriptImportService|TestPgScriptRepository_Imported' -count=1 -v
cd omcgo && /usr/local/go/bin/go test ./internal/mml -count=1
cd omcgo && /usr/local/go/bin/go build ./cmd/app
```

结果：全部通过。未修改用户提供的未跟踪参考文档，也未修改 unrelated carrier OUI 失败。

