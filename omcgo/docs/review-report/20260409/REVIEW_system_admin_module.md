# 代码审查报告 — 系统管理模块

| 项目 | 值 |
|------|-----|
| 日期 | 2026-04-09 |
| 范围 | admin 模块（字典/配置/日志/Casbin RBAC/多角色切换） |
| 变更文件 | 35 files, +3139/-42 lines |
| 审查结论 | **PASS_WITH_WARNINGS** |
| CRITICAL | 0 |
| WARNING | 8 |
| INFO | 9 |

---

## WARNING 发现

### W-01: handler.go 超过 800 行限制
- **文件**: `internal/admin/handler.go` (988 行)
- **描述**: 文件包含 auth/user/role/menu/device-group/audit/switch-role/menu-by-role 等全部 handler
- **建议**: 按功能拆分为 `auth_handler.go`、`user_handler.go`、`role_handler.go`、`menu_handler.go`

### W-02: Casbin 域派生逻辑存在跨运营商风险
- **文件**: `internal/admin/casbin.go:36-47`
- **描述**: LATERAL JOIN 从任意用户推断 domain，跨运营商角色可能映射错误
- **建议**: 权限域应作为角色或权限实体的显式列

### W-03: API Key 认证绕过 RBAC
- **文件**: `internal/admin/middleware.go:57-63`
- **描述**: X-API-Key 认证用户无角色，仅依赖 RequireAuth 的路由有越权风险
- **建议**: API Key 应携带显式范围或加载用户角色

### W-04: DELETE 端点使用 JSON body
- **文件**: `dictionary_handler.go:61-66, 150-155`
- **描述**: DeleteDictionary/DeleteDictionaryDetail 使用 ShouldBindJSON，与项目 DELETE /:id 模式不一致
- **建议**: 改为 DELETE /sysDictionary/:id 路径参数模式

### W-05: ILIKE 通配符未转义
- **文件**: `pg_dictionary_repository.go:286-289`, `sys_log.go:172,208,253`, `pg_role_repository.go:206`
- **描述**: 用户输入中的 `%` 和 `_` 作为 LIKE 通配符可能导致意外匹配
- **建议**: 添加 `escapeLike()` 辅助函数

### W-06: 字典删除缺少事务保护
- **文件**: `pg_dictionary_repository.go:161-185`
- **描述**: 先软删 details 再软删 dictionary，第二步失败时数据不一致
- **建议**: 包装在 pgx 事务中

### W-07: Update 方法修改入参（违反不可变性原则）
- **文件**: `pg_dictionary_repository.go:129-130, 341`, `sys_config.go:155`
- **描述**: `dict.UpdatedAt = time.Now()` 直接修改输入指针
- **建议**: 在 service 层计算时间戳或返回新实体

### W-08: CreateUser 角色分配失败不回滚
- **文件**: `internal/admin/service.go:170-178`
- **描述**: 角色分配失败仅 log warn，用户创建成功但无角色
- **建议**: 在事务中包装用户创建和角色分配

---

## INFO 发现

### I-01: LogHandler 绕过 service 层直接访问 repository
- **文件**: `sys_log_handler.go:13`
- **说明**: 只读查询无业务逻辑，可接受

### I-02: 分页默认值在 3 个 handler 中重复设置
- **文件**: `sys_log_handler.go:37-42, 57-62, 77-82`
- **建议**: 提取 `ensurePagination()` 辅助方法

### I-03: Dictionary 主键使用 int64 而非 UUID
- **文件**: `dictionary_model.go:11`
- **说明**: 与项目其他 model (User/Role/Menu) 的 UUID 主键不一致

### I-04: 日志时间过滤使用 string 而非 time.Time
- **文件**: `sys_log.go:69-70, 80-81, 90-91`
- **建议**: 使用 *time.Time 或添加格式验证

### I-05: SysConfig 删除为硬删除，Dictionary 为软删除
- **文件**: `sys_config.go:178` vs `pg_dictionary_repository.go:161-185`
- **说明**: 策略不一致

### I-06: 审计日志 goroutine 使用 context.Background()
- **文件**: `handler.go:317`
- **说明**: fire-and-forget 写入，丢失 OTEL 追踪

### I-07: 响应格式不完全统一
- **文件**: 多个 handler 文件
- **建议**: 定义标准 API response helper

### I-08: 新模块缺少单元测试
- **说明**: dictionary/sys_config/sys_log 模块无 _test.go 文件

### I-09: 冗余布尔默认逻辑
- **文件**: `pg_dictionary_repository.go:29-32, 231-234`
- **建议**: 简化为 `status := dict.Status`

---

## 审查结论

**PASS_WITH_WARNINGS** — 无 CRITICAL 问题，WARNING 问题均为非阻塞项，可在后续迭代中修复。
