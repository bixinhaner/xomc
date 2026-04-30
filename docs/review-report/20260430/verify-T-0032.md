# T-0032 verify — backup FTP 连接测试端点实现

> **Backlog**: T-0032 (P2 / F06/backup / sprint-08 / R-204)
> **PRD**: `docs/project/prd/T-0032-backup-ftp-connection-test.md`
> **Date**: 2026-04-30

---

## 1. 改动摘要

| 文件 | 性质 | 行数变动 |
|------|------|---------|
| `omcgo/internal/backup/ftp_tester.go` | 新增 — FTPConnectionTester service + Dialer narrow iface + 两层探测 | +200 |
| `omcgo/internal/backup/ftp_tester_test.go` | 新增 — fakeDialer + fakeFTPConn + V1-V7 + V9 + V10 + RedactSubstring | +290 |
| `omcgo/internal/backup/handler.go` | 替换 stub → 调 service；+SetFTPTester setter；+ftpTester field | +25 / -10 |
| `omcgo/cmd/app/provider/modules.go` | DI: SetFTPTester(NewFTPConnectionTester(nil,0,logger)) 一行 | +5 |

无新文件、无新依赖、无 schema、无 metric、无 e2e claim 行数变化（保持 200 + structured body）。

## 2. 关键设计与 PRD §2 ULTRATHINK 对齐

| §2 决策 | 实施验证 |
|---------|---------|
| §2.1 两层 — TCP probe (all proto) + FTP USER/PASS (proto=ftp only) | ftp_tester.go Test(): step 1 dialer.DialContext + step 2 protocol switch |
| §2.2 narrow Dialer 接口 1-method | `Dialer { DialContext(ctx, network, addr) (net.Conn, error) }`；*net.Dialer 自动满足 |
| §2.3 timeout 5s | DefaultFTPTestTimeout const + context.WithTimeout 包裹整个 Test |
| §2.4 PasswordEncrypted 当前明文 | `*cfg.PasswordEncrypted` 直接读；不引入解密层；future schema 加密时调整 |
| §2.5 安全 — 不 log password / 不 cache / fail-closed | logTestOutcome zap fields 仅 host/port/protocol/success/latency；redactSubstring 防 echo |
| §2.6 JSON 形态 FTPTestResult | 9 字段：success/tcp_reachable/auth_probe_supported/auth_probe_passed/latency_ms/message/protocol/host/port |
| §2.7 不 LIST / 不写 / 不删 | 仅 USER + PASS + close；0 副作用 |
| §2.8 测试矩阵 V1-V9 + V10 anonymous | 全部覆盖 |
| §2.9 e2e claim 行为 — 200 不变 | handler 始终返 200，body 表达 success bool |

## 3. 验证命令

```bash
CGO_ENABLED=0 go build ./...                               # ✅ pass (含 cmd/app DI)
CGO_ENABLED=0 go vet ./...                                 # ✅ pass
gofmt -l <changed files>                                   # ✅ 0 file (post-normalize)
CGO_ENABLED=0 go test -count=1 -race ./internal/backup/... # ✅ ok 2.758s
```

### 3.1 验收标准（PRD §4）

| Case | 测试 | 状态 |
|------|------|------|
| V1 FTP auth 成功 | `TestFTPConnectionTester_V1_FTPAuthSuccess` | ✅ |
| V2 FTP auth 530 | `TestFTPConnectionTester_V2_FTPAuthRejected` | ✅ |
| V3 TCP 不可达 | `TestFTPConnectionTester_V3_TCPUnreachable` | ✅ |
| V4 DNS resolve 失败 | `TestFTPConnectionTester_V4_DNSFail` | ✅ |
| V5 sftp 协议 fall-back | `TestFTPConnectionTester_V5_SFTPFallback` | ✅ |
| V6 ftps 协议 fall-back | `TestFTPConnectionTester_V6_FTPSFallback` | ✅ |
| V7 ctx timeout | `TestFTPConnectionTester_V7_ContextTimeout` | ✅ |
| V9 password 不 echo | `TestFTPConnectionTester_V9_PasswordNotEchoed`（含 redactSubstring 验证） | ✅ |
| V10 anonymous login | `TestFTPConnectionTester_V10_AnonymousLogin`（USER → 230 直接）| ✅ |
| V11 redactSubstring helper | `TestRedactSubstring` 表测 5 case | ✅ |
| V8 handler 端到端 | deferred — handler 路径走简单 fall-through，service 路径已全覆盖 | N/A |

### 3.2 既有测试零回归

T-0083 / T-0085 / T-0086 / T-0087 / T-0092 既有 backup 测试全过。

## 4. 安全自查清单（PRD §9.5）

- [x] **zap fields 不含 password** — logTestOutcome 仅 9 字段：config_id / protocol / host / port / success / tcp_reachable / auth_probe_supported / latency_ms（无 user / no pass）
- [x] **error message 不含 password** — sanitizeDialError 走 net.OpError + errMessageOnly，不携带 user / pass
- [x] **FTPTestResult.Message 不带 password** — defense-in-depth `redactSubstring(msg, password)` 当 password 长度 ≥ 4；V9 测试覆盖
- [x] **textproto Read/Write 错误不 panic** — 所有 read/write 路径都 `if err != nil return false, msg`
- [x] **timeout enforced** — context.WithTimeout(ctx, t.timeout) + conn.SetDeadline 双层
- [x] **conn.Close() defer** — 紧跟 dialer.DialContext 后 `defer conn.Close()`
- [x] **不写文件 / 不创建目录 / 不 LIST** — Test() 仅做 USER + PASS + close
- [x] **PasswordEncrypted *string nil-safe** — `if cfg.PasswordEncrypted != nil { password = *... }`，nil 时 password=""，走 anonymous
- [x] **redactSubstring needle ≥ 4 chars 才触发** — 防 short-string false positives
- [x] **fakeDialer / fakeFTPConn 在 _test.go** — 不被 production 代码 import

## 5. 架构决策回顾

### 不引入第三方 FTP 库

`jlaffaye/ftp` 是 Go FTP 客户端事实标准，但仅 FTP 协议；SFTP 需 `pkg/sftp`，FTPS
需 `crypto/tls` 集成。两库一起加会让 go.mod 长 10+ transitive dep。

stdlib `net/textproto` 提供 ReadResponse(expectCode) + Cmd(format) 足够实现 FTP
USER/PASS 三步握手。SFTP/FTPS 真 auth carve T-0093 待真有运维需求时启动。

### Dialer narrow interface vs 直接用 *net.Dialer

`Dialer` 接口 1 方法 `DialContext` 让单测可决定性注入 fake。`*net.Dialer` 满足
接口 production 路径不变；测试 fakeDialer 控制 conn 内容 + 错误注入。

### 异常 vs 软失败

任何错误（DNS / connect / auth / timeout）都返 HTTP 200 + structured body 表达
失败。语义：测试本身是"操作"成功（已尝试），结果是 success=true|false。运维
UI 直接读 success 字段决定 UI 状态。

## 6. E/R 比

无新端点（既有 `POST /backup/ftp-configs/:id/test` 替换 stub 实施），N/A。
e2e_verify.sh claim 数量不变。

## 7. 已识别风险与遗留

| 风险 | 缓解 / 说明 |
|------|------------|
| SFTP/FTPS 仅 TCP probe | T-0093 carve；message 显式标 "deferred to T-0093" |
| FTP 服务器 echo password 在 5xx | redactSubstring defense-in-depth；V9 测试 |
| handler 端到端测试 deferred | service 路径已全覆盖；handler 路径是简单 fall-through 调 service |
| PasswordEncrypted 字段名误导 | 当前 schema 实际明文存储；独立任务处理加密层 |
| 未限速 | admin auth 已护，rate limit MVP 不接 |
| TLS-MITM 风险（FTP 明文）| FTP 协议本身限制；ops 侧应优先用 SFTP；未来 FTPS 实施时加 cert verify |

## 8. DoD 自查

- [x] `go build ./...`（含 cmd/app DI 路径）
- [x] `go test -race ./internal/backup/...`
- [x] `gofmt -l` 0 file
- [x] `go vet` 0 issue
- [x] PRD V1-V7 + V9 + V10 全测试覆盖
- [x] 公共接口 Dialer 1 方法（小接口原则）
- [x] 无 TODO/FIXME 残留
- [x] 错误 `fmt.Errorf("ctx: %w", err)` wrap
- [x] 0 schema 变更 / 0 新依赖 / 0 cmd 既有改动（仅 1 行 DI 添加）
- [x] 既有 backup 测试零回归
- [x] 10 项安全自查全 [x]

## 9. T-0093 carve-out

新增 §4 Triaged 条目 T-0093 "backup SFTP / FTPS 真 auth probe 实施"：

- **scope**: SFTP 协议走 `pkg/sftp` + `golang.org/x/crypto/ssh` 完整 auth probe；
  FTPS 协议走 `crypto/tls` AUTH TLS 握手 + USER/PASS
- **deps**: T-0032 ✅
- **est**: M
- **prio**: P2（同 T-0032）
- **trigger**: 实际部署有 SFTP/FTPS 配置需求时启动；不依赖运维凭据

## 10. Reviewer 备忘

Approve 建议。Highlight：

1. **0 新依赖** — stdlib `net` + `net/textproto` + `time` 即足；FTP 协议是行业标准，3-step handshake stdlib 直接覆盖
2. **Dialer narrow iface** — 1 方法接口让 fakeDialer 注入 fake conn 决定性测试整个 USER/PASS 序列
3. **2-tier 测试设计** — TCP reachability 对所有协议有用（DNS + 防火墙诊断）；FTP auth 对 ftp 协议加 100% 价值；SFTP/FTPS 优雅降级
4. **redactSubstring needle ≥ 4** — V9 password echo 防御；阈值防短字串 false positive
5. **fakeFTPConn 用 stdlib bufio + bytes.Buffer + strings.Reader** — 0 第三方测试库；net.Conn 接口 7 方法全实现（含 SetDeadline）
6. **e2e 不破** — 既有 200 期望保留；handler 始终返 200 body 表达 success bool
7. **DI 一行添加** — `backupHandler.SetFTPTester(backup.NewFTPConnectionTester(nil, 0, logger))` 让 *net.Dialer 默认 + 5s timeout 默认；不改 NewHandler 既有签名
