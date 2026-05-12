# T-0093 Verify Report — SFTP / FTPS Real Auth Probe

**Date**: 2026-05-12
**Branch**: main
**Author**: Claude
**Type**: feat (F06 / backup)
**Sub-task**: T-0093 (carve-out from T-0032)

## Scope

Extend `internal/backup/ftp_tester.go` to replace the SFTP / FTPS
"deferred to T-0093" stubs with real authentication probes:

- **SFTP**: `golang.org/x/crypto/ssh` client handshake + password auth
- **FTPS** (implicit, port 990 style): `crypto/tls` handshake + FTP
  USER / PASS over the TLS conn (reusing `probeFTPAuth`)

## Files Changed

| File | Change | Lines |
|------|--------|-------|
| `omcgo/internal/backup/ftp_tester.go` | +SFTP/FTPS probe methods, +sshProbeFn/tlsProbeFn extension points, +sshDialDefault/tlsDialDefault production wiring, switch-case in `Test()` | +93 / -10 |
| `omcgo/internal/backup/ftp_tester_test.go` | V5 → V5+V5b+V5c (SFTP), V6 → V6+V6b+V6c+V6d (FTPS + unknown), +stub probe injection | +110 / -25 |
| `omcgo/cmd/app/provider/modules.go` | Update DI comment from "carve T-0093" → reflects implemented protocol coverage | +4 / -4 |

## Design Notes

- **Extension points**: `sshProbeFn` / `tlsProbeFn` function types stored as unexported fields on `FTPConnectionTester`. Production wiring uses `sshDialDefault` / `tlsDialDefault`; tests in the same package override the fields directly. Avoids changing the public constructor signature and avoids interface bloat.
- **SSH handshake bounded by context**: stdlib `net.Dialer.DialContext` honors `ctx.Deadline()`; the resulting conn's deadline is then set so `ssh.NewClientConn` fails-fast on the supplied timeout.
- **`InsecureIgnoreHostKey` for SSH + `InsecureSkipVerify` for TLS**: documented in package doc + `nolint:gosec` annotation. Operator-managed self-signed certs are the norm in network ops; trust signal is the USER/PASS exchange. Future T-0094 territory can add cert / host-key pinning when FTPConfig gains those fields.
- **Defense-in-depth password redaction**: `redactSubstring` already used for FTP server echo; now also applied to SSH error messages (V5c covers the case where an SSH server echoes the literal password back in its diagnostic).
- **Implicit FTPS only**: explicit FTPS (port 21 + `AUTH TLS` upgrade) is rare in modern deployments; if a real customer needs it, carve a future task.

## Test Plan

| Test | Path | Asserts |
|------|------|---------|
| V5 SFTPAuthSuccess | sftp + sshProbe returns nil | success=true, auth=true, message contains "SFTP authenticated successfully" |
| V5b SFTPAuthRejected | sftp + sshProbe returns ssh-auth-fail err | success=false, auth=false, message contains "SFTP auth failed" |
| V5c SFTPPasswordNotEchoed | sftp + sshProbe err echoes password | message redacted to `[REDACTED]`, never contains literal pwd |
| V6 FTPSAuthSuccess | ftps + tlsProbe returns scripted FTP server | success=true, ServerName=host, MinVersion≥TLS1.2, message contains "230" |
| V6b FTPSTLSHandshakeFail | ftps + tlsProbe returns err | success=false, message contains "FTPS TLS handshake failed" |
| V6c FTPSAuthRejected | ftps + tlsProbe returns 530 server | success=false, message contains "530" |
| V6d UnknownProtocol | "https" (anything not in [ftp,sftp,ftps]) | success=true (TCP only), auth-probe-supported=false |
| V1..V4, V7, V9, V10, RedactSubstring | existing — unchanged | regression baseline |

**Race**: `go test -race ./internal/backup/...` passes (2.240s).

## Commands

```bash
$ go build ./...                                                   # ✓
$ go test -race ./internal/backup/ -run FTPConnectionTester -v     # 13 PASS / 0 FAIL
$ go test -race ./internal/backup/... -count=1                     # ok 2.240s
```

## Out of Scope (Honest Carve-outs)

- **Explicit FTPS (AUTH TLS upgrade over port 21)**: file a follow-up if a customer config requires it.
- **SSH host key pinning / known_hosts**: needs FTPConfig schema extension; future T-0094.
- **TLS CA bundle / cert pinning**: same FTPConfig schema extension; future T-0094.
- **SFTP file-list / chdir probe**: the auth probe is enough for a "click test → see result" UX; deeper file-system probes would alter server state and grow the probe budget.

## DoD Checklist

- [x] `go build ./...` 通过
- [x] `go test -race ./internal/backup/...` 全绿（13 testcase 含 7 个新增）
- [x] 无新增 TODO/FIXME/panic
- [x] 无新增 `if carrier == "..."` 硬编码
- [x] 无新增 `any`/`interface{}`
- [x] 密码与会话凭据不会进入日志（zap.Info fields 不含 password/username 字段）
- [x] 密码不会出现在 result.Message（V5c / V9 双覆盖：FTP 错误回显 + SSH 错误回显）
- [x] context.Deadline 在 SSH 握手期间生效（sshDialDefault 设置 conn.SetDeadline）
- [x] 公共构造函数签名保持向后兼容（`NewFTPConnectionTester(dialer, timeout, logger)` 不变）
