# PRD: backup FTP 连接测试端点实现（T-0032）

> **关联**: Backlog T-0032 / Sprint-08 / Domain=F06/backup / Type=feat / Prio=P2
> **作者**: Claude（代 Owner=运维）
> **创建**: 2026-04-30
> **状态**: 草案 → 实施
> **关键决策**: 两层测试 — TCP reachability（所有协议）+ FTP USER/PASS auth probe（protocol="ftp" only）；SFTP/FTPS 深度测试 carve T-0093

---

## 1. 业务背景

T-0016 已实施 FTPConfig CRUD 端点 + 前端 BackupTasks 页面"Test Connection"按钮。
当前 `POST /api/v1/backup/ftp-configs/:id/test` 端点（handler.go:476-494）返回硬编码：

```json
{ "success": true, "message": "Connection test not implemented" }
```

运维痛点：
- 无法确认 FTP 配置正确（host typo / port 防火墙拦截 / username/password 错误）
- 必须先 Save 然后等 backup 任务真正跑才知道配置错没错（反馈闭环 ≥ minutes）
- 排查 FTP 配置变成"反复保存 + 跑测试任务"循环

T-0032 让"Test"按钮真正测试，把反馈闭环缩到秒级。

---

## 2. ULTRATHINK 决策

### 2.1 测试深度选择 — TCP-only vs 协议层 auth probe

候选方案：

| 方案 | 优 | 劣 |
|------|---|---|
| A. TCP reachability only | 0 新依赖；S size；快 | 不验证 username/password；价值有限 |
| B. FTP USER/PASS auth probe（stdlib net.textproto）| 真实测试凭据；FTP 可用 stdlib 实现 | 仅 FTP；SFTP/FTPS 需第三方 lib（jlaffaye/ftp / pkg/sftp） |
| C. 全协议（FTP/SFTP/FTPS）auth probe | 完整 | 2 新依赖；scope 膨胀到 M+ |

**采纳 A+B 混合**：
- TCP reachability 对所有协议可用（DNS + 端口可达性，操作员立即得"网络层通"反馈）
- FTP 协议层 USER/PASS 走 stdlib `net.textproto`（0 新 dep）
- SFTP / FTPS / 其他 protocol：仅做 TCP probe + 返回明确"deeper auth probe deferred to T-XXXX"
- 满足 P2/S size 边界

T-0093 carve：SFTP/FTPS 真 auth probe（`pkg/sftp` + `crypto/tls` 集成）。

### 2.2 narrow Dialer 接口 testability

`Dialer` 接口仅 1 方法：`DialContext(ctx, network, addr) (net.Conn, error)`。

`net.Dialer` 自动满足；测试用 `fakeDialer` 注入决定性 conn 或 error。

### 2.3 timeout 默认 5 秒

ops 工具语义：fast feedback。5s 覆盖：
- DNS resolve
- TCP 3-way handshake
- FTP USER + PASS 命令往返

可由 env / flag 调（未来）；MVP 不做。

### 2.4 PasswordEncrypted 字段当前实际是明文

`FTPConfig.PasswordEncrypted *string` 字段命名误导 — 当前 schema **不**对内容加密
存储（T-0016 留 TODO）。本任务直接读 `*PasswordEncrypted` 作为明文 password。

未来加密层落地（独立任务）后，本测试器调用方需先 decrypt 再传入。本任务不引入
该层 — scope 收紧。

### 2.5 安全考虑

- **不 log password**：所有 zap 字段拒绝 password；error message 不含 password
- **不 cache password**：测试器 receive password 立即用，不存储
- **rate limit 防滥用**：MVP 不实施（端点已有 admin auth；rate limit 拆 future）
- **fail-closed**：连接失败返结构化 error 不泄露内部 server 信息

### 2.6 测试结果 JSON 形态

```json
{
  "success": true|false,
  "tcp_reachable": true|false,
  "auth_probe_supported": true|false,  // 仅 protocol=ftp 时 true
  "auth_probe_passed": true|false,     // null 当 not supported
  "latency_ms": 123,
  "message": "Connected and authenticated successfully" | "DNS resolve failed: ..." | "FTP 530 auth rejected" | ...,
  "protocol": "ftp"|"sftp"|"ftps",
  "host": "ftp.example.com",
  "port": 21
}
```

`success` = TCP reachable AND（auth probe 不支持 OR auth probe 通过）。

### 2.7 不实施 ListObjects / 写测试文件

避免在测试端点里产生副作用（写 / 删文件可能触发服务器端 audit / 配额消耗）。
仅做 reachability + auth；后续 backup 任务真正跑才会写。

### 2.8 测试矩阵

V1: TCP 可达 + FTP auth 通过（protocol=ftp）
V2: TCP 可达 + FTP auth 失败（530 response）
V3: TCP 不可达（dial timeout）
V4: DNS resolve 失败
V5: protocol=sftp → TCP probe only + auth_probe_supported=false
V6: protocol=ftps → 同 V5
V7: ctx timeout（默认 5s）
V8: handler 端到端（mock service → JSON response）

### 2.9 e2e claim 行为

scripts/e2e_verify.sh 63.4 当前期望 200。本任务保持 200，body 改为结构化结果。
不改 claim 行数（仍 1 claim）。E/R = 1/0 (无新端点) → N/A。

---

## 3. 用户故事

| 角色 | 故事 |
|------|------|
| 运维（首次配置 FTP）| 填完表 → 点"测试连接" → 5s 内得"连通且鉴权通过"或"DNS 失败 / 端口拒连 / 用户名密码错"的具体反馈 |
| 运维（FTP 配置漂移排查）| 既有 backup 任务突然失败，跑此测试隔离是网络问题还是凭据问题 |
| 后端开发 | TestFTPConnection handler 不再返硬编码；测试器单元可独立测 |
| QA | 测试覆盖 TCP-fail / auth-fail / sftp-protocol-fall-back 三大类 |
| SecOps | 端点不 log 密码；不在 error message 暴露内部服务器细节 |

---

## 4. 验收标准（GWT）

### V1 — FTP 协议 + auth 成功

- **Given** FTPConfig protocol=ftp，valid host/port/user/pass
- **When** POST /backup/ftp-configs/:id/test
- **Then** 200 + `{success:true, tcp_reachable:true, auth_probe_supported:true, auth_probe_passed:true, latency_ms:>0}`

### V2 — FTP auth 失败

- **Given** protocol=ftp，错误 user/pass
- **When** test
- **Then** 200 + `{success:false, tcp_reachable:true, auth_probe_passed:false, message:"FTP auth rejected: 530 ..."}`

### V3 — TCP 不可达

- **Given** host=10.0.0.1（防火墙拦截）
- **When** test
- **Then** 200 + `{success:false, tcp_reachable:false, message:"dial tcp ...: connection refused"}` 或类似

### V4 — DNS resolve 失败

- **Given** host="this-host-does-not-exist.invalid"
- **When** test
- **Then** 200 + `{success:false, tcp_reachable:false, message:"... no such host"}`

### V5 — protocol=sftp（TCP-only fall-back）

- **Given** protocol=sftp
- **When** test
- **Then** 200 + `{success:true|false, tcp_reachable:true|false, auth_probe_supported:false, auth_probe_passed:null, message:"sftp 协议深度鉴权测试待 T-XXXX 实施；当前仅 TCP 可达性已验证"}`

### V6 — protocol=ftps

- 同 V5（TCP only）

### V7 — ctx timeout

- **Given** dial latency > 5s
- **When** test
- **Then** 200 + `{success:false, tcp_reachable:false, message:"... context deadline exceeded"}`

### V8 — 不存在的 ftp-config id

- **Given** 随机 UUID
- **When** POST /backup/ftp-configs/:bad-uuid/test
- **Then** 404 (既有 GetByID 失败路径)

### V9 — 安全：无密码出现在 log / error message

- **Given** 任何测试场景
- **When** zap log dump + JSON response inspection
- **Then** 0 occurrences of password 字符串

---

## 5. 运营商差异矩阵

无差异。FTP 协议是行业标准，OMC 不绑定运营商。

---

## 6. 非目标

| # | 非目标 | 原因 |
|---|--------|------|
| N1 | SFTP 真 auth probe（pkg/sftp + crypto/ssh 集成）| 引入 2 新 deps；T-0093 carve |
| N2 | FTPS（FTP over TLS）真 auth probe | T-0093 carve |
| N3 | LIST / write test file | 副作用风险（audit / 配额）；T-0094 候选 |
| N4 | rate limit | MVP；admin auth 已有 |
| N5 | 测试结果持久化（DB 历史）| 无运维要求；future 加 |
| N6 | UI 端 fancy 进度展示 | 当前 5s 内同步返回；future TUI |
| N7 | password 解密层 | 当前 schema 字段名误导但 actually 明文；独立任务处理 |

---

## 7. 依赖

| 依赖 | 状态 |
|------|------|
| T-0016 FTPConfig CRUD | ✅ done |
| stdlib `net` + `net/textproto` | ✅ |

不引入第三方 dep。

---

## 8. 度量

无新 metric（端点是 ad-hoc ops 操作，不需 SLO 监控）。日志 zap.Info 含
host/port/protocol/success/latency 已足够。

---

## 9. 设计备忘（S2）

### 9.1 文件清单

新增：

- `omcgo/internal/backup/ftp_tester.go` — FTPConnectionTester service + Dialer narrow iface + 两层测试逻辑
- `omcgo/internal/backup/ftp_tester_test.go` — V1-V7 + V9 测试 with mock Dialer

修改：

- `omcgo/internal/backup/handler.go` — `TestFTPConnection` handler 替换 stub 调 service
- `omcgo/internal/backup/handler_test.go` — V8 端到端测试 + V9 安全自查（password not in response）

无新文件、无新 dep、无 schema、无 cmd DI 改动（Handler 通过 NewHandler 注入；
新增 SetFTPTester setter 保 backwards-compat 与既有 4-arg ctor）。

### 9.2 FTPConnectionTester 接口

```go
type Dialer interface {
    DialContext(ctx context.Context, network, addr string) (net.Conn, error)
}

type FTPConnectionTester struct {
    dialer  Dialer        // *net.Dialer satisfies; tests inject fake
    timeout time.Duration // default 5s
    logger  *zap.Logger
}

type FTPTestResult struct {
    Success            bool   `json:"success"`
    TCPReachable       bool   `json:"tcp_reachable"`
    AuthProbeSupported bool   `json:"auth_probe_supported"`
    AuthProbePassed    *bool  `json:"auth_probe_passed,omitempty"`
    LatencyMs          int64  `json:"latency_ms"`
    Message            string `json:"message"`
    Protocol           string `json:"protocol"`
    Host               string `json:"host"`
    Port               int    `json:"port"`
}

func NewFTPConnectionTester(dialer Dialer, timeout time.Duration, logger *zap.Logger) *FTPConnectionTester

func (t *FTPConnectionTester) Test(ctx context.Context, cfg *FTPConfig) FTPTestResult
```

### 9.3 测试流程

```
1. start := time.Now()
2. ctx, cancel := context.WithTimeout(ctx, t.timeout); defer cancel()
3. tcp probe:
   conn, err := t.dialer.DialContext(ctx, "tcp", JoinHostPort(host, port))
   if err: return result(success=false, tcp_reachable=false, message=err)
   defer conn.Close()
4. if protocol != "ftp": return result(success=true, tcp_reachable=true, auth_probe_supported=false, message="<protocol> 深度鉴权测试待 T-XXXX 实施；TCP 可达性已验证")
5. FTP USER/PASS probe via net/textproto:
   tc := textproto.NewConn(conn)
   _, _, _ := tc.ReadResponse(220)            // FTP banner; ignore tolerantly
   tc.Cmd("USER %s", username); _, _, _ = tc.ReadResponse(0)
   tc.Cmd("PASS %s", password); code, msg, _ = tc.ReadResponse(0)
   if 200 ≤ code ≤ 299: success
   else: success=false, message=trim "xxx <msg>"
6. latency = time.Since(start)
7. return result
```

### 9.4 handler 集成

```go
type Handler struct {
    ...
    ftpTester *FTPConnectionTester  // optional; nil = legacy stub message
}

func (h *Handler) SetFTPTester(t *FTPConnectionTester) {
    h.ftpTester = t
}

func (h *Handler) TestFTPConnection(c *gin.Context) {
    // existing id parse + GetByID
    cfg, err := h.ftpRepo.GetByID(...)
    if err: ...

    if h.ftpTester == nil {
        c.JSON(200, gin.H{"success": false, "message": "FTPConnectionTester not wired"})
        return
    }
    result := h.ftpTester.Test(c.Request.Context(), cfg)
    c.JSON(200, result)
}
```

DI 在 `cmd/app/provider/modules.go`：

```go
ftpTester := backup.NewFTPConnectionTester(&net.Dialer{}, 5*time.Second, logger)
backupHandler.SetFTPTester(ftpTester)
```

### 9.5 安全自查清单

- [ ] zap fields 不含 password — 仅 host/port/protocol/success/latency_ms
- [ ] error message 不含 password — `fmt.Errorf("dial %s: %w", addr, err)` 不带 user 不带 pass
- [ ] FTPTestResult.Message 不带 password — 只回 FTP server 返的 code+msg
- [ ] textproto Read/Write 错误不 panic
- [ ] timeout enforced（context.WithTimeout 包裹整个 Test）
- [ ] conn.Close() defer 正确（即使 textproto 占用 conn 也 defer）
- [ ] 不写文件 / 不创建目录 / 不 LIST（无副作用）
- [ ] PasswordEncrypted *string nil-safe（cfg 没 password → 跳 auth probe）

### 9.6 backwards compat

既有 e2e_verify.sh 63.4 期望 HTTP 200。新实施仍返 200（即使测试失败 — body 内
表达 success=false）。e2e claim 行数不变。

---

## 10. 实施要点

预计工作量：S（约 0.5 人日）— Tester 服务 ~120 行 + 测试 ~150 行 + handler 改 ~10 行

预计涉及模块：`omcgo/internal/backup/` 1 新文件 + 1 新测试文件 + 1 修改 + cmd/app/provider/modules.go +2 行 DI

预计新增端点：0；预计新迁移：0；FE 影响：0（仍 POST 同 URL，只是 body 变结构化）；DI 改动：+1 setter 调用

---

## 11. 审批

| 角色 | 占位 | 日期 | 备注 |
|------|------|------|------|
| 产品 / 架构 / 安全 / 运维 / QA | Claude | 2026-04-30 | 0 新依赖；FTP USER/PASS 走 stdlib net.textproto；SFTP/FTPS carve T-0093 |

---

## 12. 变更记录

| 日期 | 版本 | 摘要 | 作者 |
|------|------|------|------|
| 2026-04-30 | v1.0 | 初稿；ULTRATHINK 9 决策；scope = TCP probe + FTP auth；T-0093 carve | Claude |
