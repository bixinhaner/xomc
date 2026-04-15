# Code Review: MML 命令种子数据与迁移修复

**日期**: 2026-04-15
**审查范围**: omcgo — dictionary_handler, migrations
**审查结论**: PASS_WITH_WARNINGS

---

## 审查文件

| # | 文件 | 变更类型 |
|---|------|---------|
| 1 | `omcgo/internal/admin/dictionary_handler.go` | 修改 — 响应信封对齐前端 |
| 2 | `omcgo/migrations/000013_mml_templates_and_audit.sql` | 修改 — 添加 goose StatementBegin/End |
| 3 | `omcgo/migrations/seed/000001_seed_data.sql` | 删除 — 移除重复种子文件 |
| 4 | `omcgo/migrations/seed/900004_mml_enhance.sql` | 修改 — 命令种子数据从 11 条扩展为 25 条 7 类 |

---

## 发现

### WARNING

**W1. 种子 DELETE 使用宽泛 UUID LIKE 模式**
- 文件: `omcgo/migrations/seed/900004_mml_enhance.sql:84`
- `DELETE FROM mml_commands WHERE id::text LIKE '00000000-0000-0000-%'` 可能误删用户自定义命令（如果 UUID 偶然匹配前缀）
- 建议: 缩小 DELETE 范围，添加 `AND category IN (...)` 条件

**W2. 缺少 `-- +goose Down` 部分**
- 文件: `omcgo/migrations/seed/900004_mml_enhance.sql`
- 包含主动 DELETE + INSERT 但没有 down 迁移，`goose down` 无法恢复
- 风险: 低（900xxx 种子迁移通常仅初始化时运行）

### INFO

**I1. 重复种子文件正确移除** — `000001_seed_data.sql` 与 `900001_seed_data.sql` 重复

**I2. Goose StatementBegin/End 修复正确** — `000013` 中的 PL/pgSQL 函数现在正确包裹

**I3. dictionary_handler 响应信封对齐** — `GetDictionaryDetailList` 返回 `{"list":..., "total":...}` 与前端 `adminApi.ts` 对齐

**I4. 种子文件添加 `-- +goose Up` 注解** — goose 现在能正确识别此迁移

**I5. 所有 UUID LIKE 使用 `::text` 转换** — 遵循项目约定

**I6. 25 条命令覆盖 7 个标准分类** — 使用 ON CONFLICT 确保幂等

---

## 审查清单

- [x] SQL 无字符串拼接，使用参数化
- [x] 无硬编码密钥/凭证
- [x] 迁移文件幂等（ON CONFLICT）
- [x] PL/pgSQL 使用 goose StatementBegin/End
- [x] UUID 类型比较使用 ::text 转换
- [x] 无 ORM 使用
- [x] 错误处理遵循项目规范
