# Security Review — T-0178 自定义 paramModel XML 链路

| 项 | 值 |
|---|---|
| 评审范围 | T-0178 Upload / Delete / Worker BackupCleanup 链路 |
| 评审日期 | 2026-05-29 |
| 评审人 | Claude(代行 `/security-review`)|
| 关联 PRD | `docs/project/prd/F02-param-model-custom-xml.md` |
| 关联 commits | 58640fb3 / 944c6d4c / d1349c3c / d1d6415c / ecefcfc6 / 105aec0a / 4e868236 / dff96d27 / 8f1cd905 / 77a3b83a |
| 评审范围 commit 顶 | `77a3b83a` |
| 修补 commit | (本报告同 commit) |

---

## 1. Executive Summary

| 严重度 | 总数 | 已 Mitigated | 已修补 | Open |
|--------|------|-------------|--------|------|
| CRITICAL | 0 | — | — | — |
| **HIGH** | **1** | 0 | **1** | 0 |
| MEDIUM | 2 | 1 | 0 | 1 |
| LOW | 4 | 4 | 0 | 0 |

**结论**:**可发布**。修补 HIGH-1 后,Upload/Delete 链路通过 OWASP Top 10 + Go-specific 检查清单;1 项 MEDIUM 列入运维注意事项,不阻塞 release。

---

## 2. 威胁建模 — 12 攻击面

按 OWASP Top 10 + Go-specific + 业务语义分类:

| ID | 攻击面 | 现有防御 | 残留风险 | 评级 |
|----|--------|---------|---------|------|
| T1 | **路径遍历**(file.Filename 含 `../`)| `filepath.Base()` 剥目录 + `uploadFilenamePattern` 正则白名单 + `pathContainedIn` Abs+Rel 二次验证 | 三层防御,极低 | ✅ LOW |
| T2 | **XXE**(XML 外部实体注入)| Go `encoding/xml` 默认不处理外部实体(stdlib safe-by-default)| 零 | ✅ LOW |
| T3 | **认证绕过**(无 token 即可上传)| v1 路由组 `admin.RequireAuthWithAPIKey` Bearer/API Key 强制 | 零 | ✅ LOW |
| T4 | **RBAC 不足**(非管理员上传)| `superAdminGroup.Use(admin.RequireSuperAdmin())` 仅 super_admin 可触 | 零 | ✅ LOW |
| T5 | **Multipart 内存炸弹**(并发上传耗内存)| 单文件硬上限 `MaxUploadXMLSize=1MiB` + `io.LimitReader(MaxUploadXMLSize+1)` | **engine.MaxMultipartMemory 未显式设置** → Gin 默认 32MiB,N 并发耗内存 | ❌ **HIGH-1 → 已修补** |
| T6 | **Race condition**(同名 Upload 并发)| `acquireFileLock(basename)` per-filename sync.Map mutex | 单实例假设,横扩需 PG advisory lock(已记 R-NEW-T0178-6)| ⚠️ MED-2 |
| T7 | **Symlink 攻击**(攻击者 UID 10001 创建符号链接逃逸 customDir)| `os.Rename` 不跟随 symlink;前置条件是攻击者已有容器内 shell | 攻击者一旦有 shell 已 game over;补 `Lstat` 防御为 over-engineering | ✅ LOW |
| T8 | **Slowloris**(慢速 HTTP 体)| `http.Server.ReadTimeout=30s` / `WriteTimeout=30s` / `IdleTimeout=120s` | `ReadHeaderTimeout` 未独立设置(可选,ReadTimeout 已覆盖)| ⚠️ MED-1 |
| T9 | **保留名拦截**(覆盖 standard-model.xml)| `reservedUploadFilenames` map 严格匹配 + case-insensitive | 零 | ✅ LOW |
| T10 | **审计日志注入**(`file.Filename` 写日志)| zap 结构化字段自动转义,无格式串拼接 | 零(Go 日志库安全)| ✅ LOW |
| T11 | **超大 XML 解析炸弹**(LimitReader 后仍可能慢)| 1 MiB 上限 + `xml.Decoder` 流式 + Token API 早 break | 极低 | ✅ LOW |
| T12 | **Rename ↔ Cron 竞态**(worker 清扫与 DELETE 同步)| .deleted/.bak 用文件名 ts 不是 mtime,新备份 ts=now 永不立即过期 | 零 | ✅ LOW |

---

## 3. 详细发现

### HIGH-1 [已修补] Gin engine.MaxMultipartMemory 未显式设置

**位置**:`omcgo/cmd/app/main.go:64`

**问题**:
- `gin.New()` 后未 `engine.MaxMultipartMemory = N`
- Gin 默认值 = 32 MiB(per request)
- 攻击者通过 N 并发 multipart 上传可消耗 N × 32 MiB 内存
- PRD §9.4 注明 "Gin engine MaxMultipartMemory 应配 4 MiB" 但实施时漏掉

**影响**:
- DoS 攻击成本极低(N=32 并发 → 1 GiB RAM 消耗)
- 当前 RBAC 拦在 super_admin,但 token 泄露后无量级限制

**修补**:
```go
// omcgo/cmd/app/main.go (本 commit)
engine := gin.New()
engine.MaxMultipartMemory = 4 << 20 // 4 MiB(T-0178 S5)
```

4 MiB = 单文件 1 MiB + 3 MiB form 字段冗余,远高于业务需求,远低于 32 MiB 默认。

**验证**:`go build ./...` clean。需 P5 E2E 补一个并发 multipart 上限测试(留作 follow-up)。

---

### MED-1 [Open] ReadHeaderTimeout 未独立设置

**位置**:`omcgo/internal/core/components/infra.go:210`

**问题**:
- `http.Server` 设 `ReadTimeout=30s` 覆盖整个请求读取
- Go 1.8+ 推荐独立设置 `ReadHeaderTimeout` 让 header 读取更严格
- 当前对 multipart upload 的 30s 已足够卡 slowloris,无紧迫修补需求

**建议**(不阻塞 release):
- 加 `ReadHeaderTimeout: 10s`,与 ReadTimeout 分离
- 留作运维优化任务,P0 风险登记不补条目

**当前状态**:Open(优先级低,留 P3 ops 任务)。

---

### MED-2 [Mitigated by R-NEW-T0178-6] 多实例并发同名上传

**位置**:`omcgo/internal/config/parammodel/handler.go::acquireFileLock`

**问题**:`sync.Map[basename]*sync.Mutex` 仅进程内互斥;多 app 实例横扩时同名 Upload 会丢更新。

**已 Mitigated**:
- 当前生产单 app 实例(`docker-compose.app.yml` 无 replicas)
- `omcgo/CLAUDE.md §5.3.1` 显式记单实例假设
- `risk-register.md R-NEW-T0178-6` Open 状态,横扩前 PR 必须补 PG advisory lock

**横扩前必做**(已记):
```go
lockKey := int64(crc32.ChecksumIEEE([]byte("parammodel:" + basename)))
_, _ = h.pool.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", lockKey)
```

---

### LOW-1 [Mitigated] 路径遍历(T1)

**当前防御** 3 层:
1. `filepath.Base(file.Filename)` 剥目录组件(`../../etc/passwd` → `passwd`)
2. `uploadFilenamePattern = ^[A-Za-z0-9_-]{1,64}\.xml$` 正则白名单 → `passwd` 无 `.xml` 后缀被拒
3. `pathContainedIn(customDir, targetPath)` Abs+Rel 二次验证

**E2E 覆盖**:`scripts/e2e_param_model_custom.sh GWT-9` 显式造 `filename=../../etc/passwd` 验证 400。

---

### LOW-2 [Mitigated] XXE

Go `encoding/xml.Decoder` **默认不处理外部实体**(stdlib doc 明示)。无需 `decoder.Entity = nil` 或类似配置。

测试:`scripts/e2e_param_model_custom.sh GWT-10` 上传 wrongRoot XML 已覆盖根元素校验;外部实体由 Go stdlib 拦在解析期。

---

### LOW-3 [Mitigated] Symlink 攻击

**攻击场景**:攻击者 UID 10001(容器用户)在 customDir 创建符号链接 `X.xml -> /etc/passwd`,Upload 覆盖 X.xml 同时覆盖目标。

**防御分析**:
- `os.WriteFile(tmpPath, ...)`:tmpPath 含 `.tmp.<uuid>` 后缀,攻击者无法预测 UUID 提前创建符号链
- `os.Rename(tmpPath, targetPath)`:Go `os.Rename` **不跟随** symlink,直接覆盖 symlink 本身(而非目标)
- `os.Rename(targetPath, backupPath)`:同理,移动 symlink 不跟随

**前置条件**:攻击者需先有容器内 shell 写权限。一旦满足,容器内身份已 = game over,Upload 不是主要攻击面。

**建议(不实施)**:可在 `pathContainedIn` 前加 `os.Lstat(parent).Mode()&os.ModeSymlink != 0` 检查父目录是否为 symlink — over-engineering,不补。

---

### LOW-4 [Mitigated] 审计日志注入

`file.Filename` 写日志使用 `zap.String("raw_filename", file.Filename)`。zap 结构化字段自动转义控制字符、换行符,无 `fmt.Sprintf` 拼接,不会被日志解析器误判为新 entry。

测试:无需新增,Go 日志库属系统级保证。

---

## 4. 检查清单(逐项核对)

### 4.1 OWASP Top 10 (2021)

- [✓] A01 Broken Access Control → `RequireSuperAdmin` 闭环
- [✓] A02 Cryptographic Failures → 无敏感数据加密需求,且走 HTTPS(部署侧)
- [✓] A03 Injection → XML 解析无字符串拼接 SQL,Squirrel 参数化
- [✓] A04 Insecure Design → PRD 5 轮 ULTRATHINK + risk-register 8 条已登记
- [✓] A05 Security Misconfiguration → **HIGH-1 修补后** Gin multipart 上限就位
- [✓] A06 Vulnerable & Outdated Components → 走 `go.sum` 版本锁
- [✓] A07 Identification & Authentication Failures → Bearer/API Key 双轨
- [✓] A08 Software & Data Integrity Failures → tmp + rename 原子写
- [✓] A09 Security Logging & Monitoring Failures → 7 个 audit_action 分支 + Prometheus 指标
- [✓] A10 SSRF → 业务无出站请求需求,N/A

### 4.2 文件处理专项

- [✓] 文件名白名单正则
- [✓] 保留名硬拦截
- [✓] 大小硬上限(单文件 + LimitReader 流式)
- [✓] 路径包含二次验证
- [✓] 原子写(tmp + rename)
- [✓] 失败回滚(反向 rename 还原 .bak)
- [✓] per-filename 互斥(单实例)
- [✓] 备份策略(30 天 + cron)

### 4.3 Go 安全规范

- [✓] `context.Context` 传递(handler 链路全程)
- [✓] 错误 wrap with context(`fmt.Errorf("...: %w", err)`)
- [✓] 不裸 panic(`commonerrors.AbortWithError` 统一)
- [✓] 不硬编码运营商字符串
- [✓] 不使用 ORM(squirrel + pgx)
- [✓] 测试覆盖(53 个 T-0178 单测 + 10 个 E2E 断言)

---

## 5. 修补摘要

| 类别 | 文件 | 改动 |
|------|------|------|
| HIGH-1 修补 | `omcgo/cmd/app/main.go` | +3 / -0(MaxMultipartMemory = 4 MiB + 注释) |

无前端 / yaml / 部署改动(本轮仅 1 处后端配置加固)。

---

## 6. 不阻塞 release 的运维注意事项

1. **ReadHeaderTimeout(MED-1)**:留 P3 ops 任务,加 `http.Server.ReadHeaderTimeout: 10s` 让 header 读取更严格(分离 ReadTimeout)。
2. **多实例横扩(R-NEW-T0178-6)**:横扩前必须先补 PG advisory lock;Wave 3 / GA 之前的单实例期内 OK。
3. **`.tmp.<uuid>` 残留监控**:Prometheus alert `parammodel_backup_cleanup_total{kind="tmp",result="swept"} rate > 0/h` 可触发"上传中断频发"告警。
4. **审计日志保留**:`parammodel.delete.aborted_backup_failed` 等 audit_action 7 个分支,Loki 查询模板可入运维手册。

---

## 7. Approval

| 角色 | 签字 | 结论 |
|------|------|------|
| Security Reviewer(Claude 代行) | ✓ | HIGH-1 修补后,可发布 |
| QA / 发布经理 | 待签 | 待 P5 E2E 跑通后会签 |
| 项目经理 | 待签 | 待 S7 收尾时会签 |

---

**版本历史**:
- 2026-05-29 v1.0 初版,12 攻击面建模 + HIGH-1 即修
