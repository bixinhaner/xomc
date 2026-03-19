# Code Review Report

**Date**: 2026-03-19
**Reviewer**: Claude Code
**Scope**: acs
**Files**: 11 files changed, +3670 lines, -78 lines

---

## Summary

实现 ACS 文件上传代理功能，支持 CPE 设备通过 ACS 服务器上传文件到 MinIO。

### 变更范围

| 组件 | 文件 | 变更类型 |
|------|------|----------|
| Upload 组件 | `internal/acs/upload/*.go` | 新增 (4 files) |
| ACS Server | `internal/acs/server.go` | 修改 |
| ACS Handler | `internal/acs/handler.go` | 修改 |
| 配置 | `internal/core/appconfig/config.go` | 修改 |
| Bootstrap | `internal/core/bootstrap/bootstrap.go` | 修改 |
| 设计文档 | `docs/design/*.md` | 修改/新增 |

---

## Review Findings

### CRITICAL (0)

无严重问题。

### WARNING (2)

#### 1. 代码重复：bucketForFileType 和 objectPath

**文件**: `internal/acs/upload/handler.go:117-128`, `internal/acs/upload/uploader.go:66-82`

**问题**: `bucketForFileType` 和 `objectPath` 方法在 `Handler` 和 `Uploader` 两个结构体中重复定义。

**建议**: 考虑提取到公共位置或让 Handler 持有 Uploader 引用。

```go
// 当前：重复定义
// handler.go
func (h *Handler) bucketForFileType(fileType string) string { ... }
func (h *Handler) objectPath(deviceSN, filename string) string { ... }

// uploader.go
func (u *Uploader) bucketForFileType(fileType string) string { ... }
func (u *Uploader) objectPath(deviceSN, filename string) string { ... }
```

**影响**: 低，不影响功能，仅影响可维护性。

#### 2. 缺少 MinIOConfig 定义确认

**文件**: `internal/core/appconfig/config.go`

**问题**: 新增了对 `MinIOConfig` 的引用，但 diff 中未显示该结构体定义。需确认是否已存在于代码库中。

**验证**: 经检查，`MinIOConfig` 已存在于同一文件中，无问题。

### INFO (3)

#### 1. Session TTL 硬编码

**文件**: `internal/acs/upload/session.go`

**观察**: Session TTL 通过构造函数参数传入，设计良好。

#### 2. JWT 安全实践

**文件**: `internal/acs/upload/token.go`

**观察**:
- ✅ 使用 HMAC-SHA256 签名
- ✅ 包含过期时间
- ✅ 验证签名方法类型防止算法切换攻击

#### 3. 文件上传流式处理

**文件**: `internal/acs/upload/handler.go:65-77`

**观察**:
- ✅ 直接将 `r.Body` 流式传递给 MinIO
- ✅ 不将整个文件加载到内存
- ✅ 适合大文件上传场景

---

## Security Review

| 检查项 | 状态 | 说明 |
|--------|------|------|
| JWT 签名验证 | ✅ | 使用 HMAC-SHA256，验证签名方法 |
| Token 过期 | ✅ | 有 ExpiresAt 设置 |
| 文件大小限制 | ✅ | 检查 ContentLength |
| 路径注入 | ✅ | 使用设备 SN 和时间戳生成路径 |
| 资源泄漏 | ✅ | 使用 context 管理生命周期 |

---

## Code Quality

| 维度 | 评分 | 说明 |
|------|------|------|
| 可读性 | ⭐⭐⭐⭐ | 代码清晰，注释充分 |
| 可维护性 | ⭐⭐⭐ | 存在少量代码重复 |
| 安全性 | ⭐⭐⭐⭐⭐ | JWT 验证完善，流式处理 |
| 测试覆盖 | ⭐⭐ | 缺少单元测试 |
| 文档 | ⭐⭐⭐⭐⭐ | 设计文档详尽 |

---

## Conclusion

**PASS_WITH_WARNINGS**

代码实现正确，安全措施完善。存在少量代码重复，建议后续重构优化。可以提交。

---

## Action Items

1. **[可选]** 后续重构：提取 `bucketForFileType` 和 `objectPath` 到公共位置
2. **[推荐]** 添加单元测试覆盖 upload 包
