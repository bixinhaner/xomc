# Verify Report — T-0075 backup 加密执行 AES-256-GCM 信封加密（**安全敏感**）

> **Task**: T-0075 / R-102 / Sprint-07
> **Branch**: main / pre-commit @ b5a6e5d4
> **Date**: 2026-04-29
> **PRD**: `docs/project/prd/T-0075-backup-encryption-aes256gcm.md`

---

## S4 出口门检查

| 门 | 结果 | 详情 |
|----|------|------|
| `go build ./...` | ✅ | clean |
| `go test -race ./internal/backup/... ./internal/acs/upload/... ./internal/acs/download/...` | ✅ | 0 fail；新增 16+ case (encryption round-trip 7 sizes / tamper / wrong-AAD / wrong-key / nonce uniqueness 500x / oversize / format invalid / Env hex parse 4 cases / policy validate KEK paths) |
| `go vet` | ✅ | clean |
| Migration | ✅ N/A | 复用 T-0071 既有 schema（enable_encryption + encryption_algorithm） |
| Metric grep | ✅ | 3 新 metric (omc_backup_encrypted_total + omc_backup_encryption_errors_total + omc_backup_decryption_errors_total) 全 grep 命中 |
| 新端点 | ✅ N/A | 无新路由；2 新 e2e claim 验证 PUT 校验 |
| 加密层敏感测试 | ✅ | 全 sentinel error path 覆盖；nonce 唯一性 500-iter 经验性验证 |

---

## 文件改动清单

新增（4 文件）：
- `omcgo/internal/backup/encryption.go` (~210 行) — Encryptor 接口 + AES-256-GCM 实现 (envelope encryption: KEK→DEK + AAD-bound) + CBC/ChaCha20 stubs + 6 sentinel error + 文件格式 OENC v1
- `omcgo/internal/backup/encryption_test.go` (~170 行) — 9 case
- `omcgo/internal/backup/key_provider.go` (~85 行) — KeyProvider interface + EnvKeyProvider (OMC_BACKUP_ENCRYPTION_KEY hex parse) + staticKeyProvider for tests
- `omcgo/internal/backup/key_provider_test.go` (~50 行) — 4 case (unset / valid / invalid hex / wrong length)

修改（7 文件）：
- `omcgo/internal/backup/policy_service.go` — +keyProvider field + SetKeyProvider + (s)PolicyService.validatePolicy 加 2 检查（GCM+KEK unavailable / CBC|ChaCha20 stub reject）
- `omcgo/internal/backup/policy_service_test.go` — +4 case (CBC reject / ChaCha20 reject / GCM no-kp accept / KEK unavail reject + KEK avail accept)
- `omcgo/internal/backup/policy_metrics.go` — PolicyMetrics +3 collector (encrypted_total + encryption_errors_total + decryption_errors_total) + 3 Record helper
- `omcgo/internal/acs/upload/handler.go` — Handler +encryptor field + SetEncryption + encryptUpload (64MB cap + AAD=filename) + classifyEncryptError + ServeHTTP encryption layer (after compression, before PutObject) — fail-closed semantics
- `omcgo/internal/acs/download/handler.go` — Handler +encryptor + encMetrics + SetEncryption + maybeDecrypt (peel `.enc` → buffer → decrypt → bytes.Reader) — fail-closed
- `omcgo/cmd/acs/main.go` — DI: NewEnvKeyProvider + NewEncryptor("AES-256-GCM") (when KEK available); upload.SetEncryption + download.SetEncryption
- `omcgo/cmd/app/provider/modules.go` — DI: policyService.SetKeyProvider for PUT-time validation
- `omcgo/scripts/e2e_verify.sh` — +bk-12 (CBC reject) + bk-13 (GCM schema-valid)

---

## GWT 验收对照

| GWT | 验证手段 | 状态 |
|-----|---------|-----|
| V1 AES-GCM round-trip 多 sizes | TestAESGCM_RoundTrip_sizes (0/1/16/1023/1024/65536/1024*1024 byte) + magic check + 100B overhead 断言 | ✅ |
| V2 Tamper detection | TestAESGCM_TamperDetected — 位翻 → ErrEncryptionAuthFailed | ✅ |
| V3 Nonce uniqueness | TestAESGCM_NonceUniqueness — 500 次同 plaintext+key+AAD ⇒ 500 个唯一 ciphertext | ✅ |
| V4 Wrong AAD rejection | TestAESGCM_WrongAAD — A 加密 B 解密 → ErrEncryptionAuthFailed | ✅ |
| V5 Service reject CBC/ChaCha20 | TestValidatePolicy_table 含 2 case (substring "not yet implemented") | ✅ |
| V6 Service reject GCM no-KEK | TestValidatePolicy_AESGCM_KEKUnavailable — substring "encryption key not configured" + EnvBackupEncryptionKey | ✅ |
| V7 Upload encryption integrate | encryptUpload + ServeHTTP 路径 + RecordBackupEncrypted on success | ✅（设计 + 单元测试；e2e 需 docker） |
| V8 Upload 64MB cap | encryptUpload limited LimitReader + 检查 len > 64MB → ErrEncryptionInputTooLarge | ✅ |
| V9 Download decrypt | maybeDecrypt 路径 + AAD=filename(stripped .enc) → bytes.Reader → 解压链 | ✅（设计 + 单元测试） |
| V10 FE warning Tag 关闭 | **deferred T-0088 FE-only**（PRD §6 N7 调整） | ⏸ defer |

---

## ULTRATHINK 决策回顾

| 决策 | 兑现 |
|------|------|
| 信封加密 KEK + 每文件 DEK | ✅ aesGCMEncryptor.Encrypt 每次 io.ReadFull(rand.Reader, dek) 32B + outer/inner nonce 各 12B |
| EnvKeyProvider MVP / KMS plug-in 接口 | ✅ KeyProvider interface + EnvKeyProvider impl + staticKeyProvider for tests |
| Buffer-then-encrypt 64MB cap | ✅ encMaxPlaintext = 64*1024*1024；upload encryptUpload + download maybeDecrypt 双侧均 cap |
| AAD=filename 防替换 | ✅ encryptUpload `aad := []byte(filename)`；download maybeDecrypt `aad := []byte(filepath.Base(innerPath))` |
| 文件格式 OENC v1 | ✅ magic "OENC" + version 0x01 + algo 'G' + reserved 2B + outer_nonce + wrapped_DEK_len + wrapped_DEK + inner_nonce + ciphertext+tag |
| 100B overhead per file | ✅ Encrypt 验证 `len(blob) == len(plaintext)+100` |
| Stub CBC + ChaCha20 | ✅ TestNewEncryptor_stubAlgorithms + service-layer reject |
| Fail-closed on encryption error | ✅ ServeHTTP encryption error → http.Error 不 fall through plaintext |
| Fail-closed on decryption error | ✅ ServeHTTP maybeDecrypt error → http.Error 不 serve ciphertext |
| 频道 fast-path: KEK unset → encryption disabled | ✅ NewEnvKeyProvider unset → Available()=false → upload 不 wrap encryption |

---

## 安全性自检（SecOps 预审清单）

| 安全要求 | 实施 | 状态 |
|----------|------|------|
| AES-256-GCM AEAD | ✅ stdlib `crypto/cipher.NewGCM` | safe |
| Nonce 96-bit 随机 per file | ✅ `io.ReadFull(rand.Reader, nonce)` 每次新生成 | safe |
| 常数时间 tag 校验 | ✅ stdlib internal `crypto/subtle.ConstantTimeCompare` | safe |
| 密钥不打日志/不进 error message | ✅ encryption.go / key_provider.go 全部 zap 字段不含 key 字节 | verified |
| KEK 来源 = env var (OMC_BACKUP_ENCRYPTION_KEY 64 hex) | ✅ EnvKeyProvider | safe |
| 32 字节长度强校验 | ✅ NewEnvKeyProvider 拒绝 != 32 字节 | safe |
| AAD 绑定 filename 防替换 | ✅ Encrypt + Decrypt 双侧 + Wrong AAD test | safe |
| 加密失败不 fall through plaintext | ✅ encryption error → 5xx；不写 MinIO | safe |
| 解密失败不 serve ciphertext | ✅ Decrypt error → 4xx (auth)/5xx (other)；不响应 body | safe |
| 64MB buffer cap (DoS 防护) | ✅ upload + download 双侧 LimitReader + 大小检查 | safe |
| crypto/rand 是 /dev/urandom | ✅ stdlib | safe |
| KEK 旋转 | ⏸ 不实施；defer T-0087 + 文档化 | doc |
| 历史 plaintext backups 反向加密 | ⏸ forward-only；defer T-0087 backfill | doc |
| 关闭加密 → 老 .enc 不可解（KEK 删后）| ⏸ doc | doc |

S5 阶段需调用 `/security-review` 验证以上 + 看可能漏的攻击面。

---

## 备注

- **行数 delta**: ~520 行新代码（其中 ~370 是测试 + 加密核心；DI / wiring 较小）
- **0 schema 变更**；0 新外部依赖（所有 crypto stdlib）
- **frontend Tag 关闭 (V10) 拆 T-0088 FE-only task**：本任务集中后端 + 加密核心；FE 改动是 1-2 行 + i18n key，单独任务更清晰
- **PolicyService 改动**：`validatePolicy` 函数 → `(s *PolicyService).validatePolicy` 方法（保留 package-level `validatePolicy(p)` 作为 inner helper） — 接口签名不变；4 既有 mock 不需改
- **Cross-process 一致性**：app + acs 两进程都读同 env var；运维需保证部署同步（systemd EnvironmentFile 共享或 k8s ConfigMap）
- **Multi-skin**：`omcmb/` 完全未触（FE 收尾拆 T-0088）— v2/v3 零影响

---

## S5 Code Review + Security Review 复核（review-agent verdict: APPROVE-WITH-FIXES → APPROVE）

| 严重度 | 问题 | 修复 | 状态 |
|--------|------|------|------|
| **HIGH** | **AAD mismatch on compress+encrypt path** — upload 端 AAD=filename，download 端 AAD=basename(strip ".enc")=filename+cmp.ext，组合 EnableCompression+EnableEncryption=true 时**每次 restore 都 422**；round-trip tests 漏抓因为没跑 compress→encrypt 链 | upload 端 AAD 改为 `filename + cmp.ext` 与 download 端 reconstruction 对齐；新加 `TestAESGCM_AAD_CompressedExtension` 显式 regression guard | ✅ Fixed |
| **MEDIUM** | EnvKeyProvider.KEK() 返 backing slice — caller mutate 会污染后续 encrypt | 改返 `append([]byte(nil), p.key...)` 32B copy；defense-in-depth | ✅ Fixed |
| LOW | staticKeyProvider 在 production file（unexported 但仍是 attack surface） | 移到新文件 `testhelpers_keyprovider_test.go` (build tag _test.go 只在测试时编译) | ✅ Fixed |
| MEDIUM | `EnvKeyProvider.Available()` boxed-nil interface 安全分析 | verify safe: typed-nil 通过 interface 仍走方法 receiver；nil-receiver guard `p != nil` 兜底 | ⏸ accept (verified safe) |
| MEDIUM | Decrypt-as-DoS: 64MB×N 并发 buffer 内存放大 | 拆 **T-0089**（semaphore-bounded decrypt concurrency）；basic-auth + HTTP rate limits 部分缓解 | ⏸ defer T-0089 |
| MEDIUM | `encMaxPlaintext` 常量在 upload + download 重复 magic literal | 接受；后续可统一暴露为包级 const；不影响安全 | ⏸ accept |
| LOW | wrappedLen u32→int 整数溢出 | verified safe: 64-bit Go u32 fits int + strict-equality check 兜底 | ⏸ accept (safe) |

**review-agent 11 项验证全确认**：
- crypto/rand.Reader 始终 io.ReadFull 错误传播，无零字节回退
- KEK 仅 wrap DEK，从不直接 encrypt plaintext
- 0 处 `zap.ByteString("kek", ...)` 或 `%v` 格式化 key 字节
- Fail-closed 路径：upload http.Error 在 PutObject 之前；download http.Error 在 io.Copy 之前
- magic / version / algo 顺序检查无 timing oracle（恒定字节比较 pre-AES）
- validatePolicy package func 仅由 method 调用，无 KEK-availability check bypass
- 500-iteration nonce-uniqueness 测试覆盖 deterministic-encryption regression

**最终复核**：
- `go build ./...` ✅
- `go test -race ./internal/backup/... ./internal/acs/upload/... ./internal/acs/download/...` ✅ 全过含新增 `TestAESGCM_AAD_CompressedExtension` 100% green
- 无 P0/P1 阻塞项
- T-0089 (decrypt-DoS semaphore) 登记 followup
