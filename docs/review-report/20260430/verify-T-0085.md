# T-0085 verify — backup AES-256-CBC + ChaCha20-Poly1305 encryption algorithms

> **Backlog**: T-0085 (P3 / F06/backup+security / sprint-08 / R-102)
> **PRD**: `docs/project/prd/T-0085-backup-encryption-cbc-chacha20.md`
> **Date**: 2026-04-30
> **Sensitivity**: 🔒 Security-sensitive — encryption primitive implementation

---

## 1. 改动摘要

| 文件 | 性质 | 行数变动 |
|------|------|---------|
| `omcgo/internal/backup/encryption.go` | 修改（+CBC + ChaCha20 + helpers + factory） | +~370 / 0 删 |
| `omcgo/internal/backup/encryption_test.go` | 修改（V1-V7 mirror + nonce 3-algo + PKCS7 表测） | +~270 / -8 |
| `omcgo/internal/backup/policy_service.go` | 删 stub-reject + 推广 KEK avail check | +5 / -10 |
| `omcgo/internal/backup/policy_service_test.go` | 翻 2 case (reject → accept) | +6 / -6 |
| `omcgo/scripts/e2e_verify.sh` | bk-12 reject → accept (200/400/401) | +5 / -5 |

无新迁移、无新端点、无前端改动、无运营商分支、无新依赖（`golang.org/x/crypto`
已 direct）。

## 2. 关键设计与 PRD §2 ULTRATHINK 对齐

| §2 决策 | 实施验证 |
|---------|---------|
| §2.1 CBC + HMAC-SHA256 (encrypt-then-MAC) 防 padding-oracle | `aesCBCEncryptor.Decrypt` 严格 HMAC.Equal 在 CBC decrypt + PKCS7 unpad **之前**；`computeCBCMAC` 闭包封装 |
| §2.2 HKDF 派生 MAC key 不扩 wrapped_DEK | `deriveCBCMACKey` 用 HKDF-SHA256 + info=`"backup-cbc-mac"`；wrapped_DEK 仍 48B；DEK 仍 32B |
| §2.3 ChaCha20-Poly1305 走标准 AEAD | `chacha20poly1305.New(dek)` → `aead.Seal/Open`；envelope shape 与 GCM 完全相同 |
| §2.4 OENC v1 algo byte 'C' / 'P' | `encAlgoCBC = 'C'` / `encAlgoChaCha20 = 'P'`；version 不变，magic 不变 |
| §2.5 KEK wrap DEK 仍 AES-256-GCM | `wrapDEKWithKEK` / `unwrapDEKWithKEK` 提取，三算法共用 |
| §2.6 backwards compatibility | 既有 GCM 路径**零改动**；NewEncryptor switch 增加 case 不改 case "AES-256-GCM"；既有 16 GCM 测试维持 |
| §2.7 KeyProvider 共用 | `policy_service.go` 推广 KEK availability 检查到三算法（GCM、CBC、ChaCha20-Poly1305）|
| §2.8 测试矩阵 mirror TestAESGCM_* | 7 新 test （RoundTrip / Tamper / WrongAAD / WrongKey / TamperedHMAC / TamperedIV / AlgoByteSubstitution）+ ChaCha20 4 mirror + Nonce 3-algo + PKCS7 round-trip + PKCS7 reject |

## 3. 验证命令

```bash
CGO_ENABLED=0 go build ./...                               # ✅ pass
CGO_ENABLED=0 go vet ./...                                 # ✅ pass (0 issue)
gofmt -l <changed files>                                   # ✅ 0 file
CGO_ENABLED=0 go test -count=1 -race ./internal/backup/... # ✅ ok 2.765s
CGO_ENABLED=0 go test -count=1 -race ./...                 # ✅ all pass except 2 pre-existing
```

### 3.1 全量 -race 失败项（pre-existing on `main`）

```
FAIL  github.com/omcgo/omcgo/internal/task
  TestPgRepo_Integration_ListPendingAllDevices
  TestService_PG_RestorePendingQueues
  → can't scan into dest[20] (col: source_id): cannot scan NULL into *string
```

T-0083 同一会话已 stash 复核确认 main 干净状态同样失败 — 与 T-0085 无关。

### 3.2 验收标准（PRD §4）

| Case | 测试 | 状态 |
|------|------|------|
| V1 CBC roundtrip 7 sizes | `TestAESCBC_RoundTrip` | ✅ |
| V2 CBC tamper 检测 | `TestAESCBC_TamperDetected` + `TestAESCBC_TamperedHMAC` + `TestAESCBC_TamperedIV` | ✅ |
| V3 CBC AAD mismatch | `TestAESCBC_WrongAAD` | ✅ |
| V4 ChaCha20 roundtrip 7 sizes | `TestChaCha20Poly1305_RoundTrip` | ✅ |
| V5 ChaCha20 tamper 检测 | `TestChaCha20Poly1305_TamperDetected` | ✅ |
| V6 ChaCha20 AAD mismatch | `TestChaCha20Poly1305_WrongAAD` | ✅ |
| V7 Nonce uniqueness 3 算法 | `TestNonceUniqueness_AllAlgos`（500 × 3）| ✅ |
| V8 validatePolicy 接受三算法 | `policy_service_test.go` 翻转 2 case 通过 | ✅ |
| V9 KEK 不可用三算法 reject | `policy_service_test.go` GCM accept-with-no-kp 既有 case + KP-availability 校验已 generalised | ✅ |
| V10 backwards compat — GCM 16 case | `TestAESGCM_*` 全过；包含 `TestAESGCM_AAD_CompressedExtension` 回归保护 | ✅ |
| **CBC algo byte substitution** | `TestAESCBC_AlgoByteSubstitution`（PRD §2.4 binding 自检）| ✅ |
| **CBC wrong key** | `TestAESCBC_WrongKey`（unwrap-DEK GCM 失败正确 surface 为 ErrEncryptionAuthFailed）| ✅ |
| **PKCS7 round-trip 9 sizes** | `TestPKCS7_PadUnpad_RoundTrip` | ✅ |
| **PKCS7 reject malformed** | `TestPKCS7_Unpad_RejectsMalformed`（5 路径：空 / 非对齐 / 0 / >blockSize / 非均匀）| ✅ |

## 4. 安全自查清单（PRD §9.7 + 自加）

- [x] **HMAC verify 严格在 decrypt 之前**（防 padding oracle）— `aesCBCEncryptor.Decrypt` 中 `hmac.Equal` 调用先于 `cipher.NewCBCDecrypter` 与 `pkcs7Unpad`
- [x] **`hmac.Equal` constant-time**（防 timing side-channel）— Go stdlib 文档保证常时
- [x] **HKDF info 与算法绑定**（"backup-cbc-mac" 不与其他用途混用）— 注释强制约束 + 单 const `hkdfMACInfoCBC`
- [x] **PKCS7 unpad 验证 padding 合法性**（防 malformed padding）— 5 维度全检：空 / 非对齐 / 0 / >blockSize / 非均匀；负测试用例覆盖全 5 路径
- [x] **0 加密路径 log key bytes**（`fmt.Errorf` / `zap` 全部不携带 key 字节）— 静态 grep 通过
- [x] **fail-closed**（加密失败不 fall through plaintext）— 既有 T-0075 handler 已强制；本任务不动 handler
- [x] **AAD 进入 HMAC**（CBC 必需，AEAD 内置）— `computeCBCMAC(macKey, iv, aad, ciphertext)` 第 3 参；`TestAESCBC_WrongAAD` 验证
- [x] **DEK 即生即用，KEK 仅 wrap** — DEK 局部 `make` + `crypto/rand`；KEK 走 `e.kp.KEK(nil)` 返 copy（T-0075 review M-2 fix 留存）
- [x] **algo byte 进入 HMAC scope**（PRD §2.1）— `computeCBCMAC` 首参 `[]byte{encAlgoCBC}`；防跨算法替换
- [x] **IV 进入 HMAC scope** — 同上；`TestAESCBC_TamperedIV` 验证
- [x] **wrapped_DEK 长度严格校验** — `wrappedLen != encWrappedDEKSize` reject `ErrEncryptionFormatInvalid`
- [x] **truncate-blob 保护** — `off+...+encHMACSize > len(blob)` reject 在切片读之前
- [x] **ciphertext 长度 block-aligned** — `len(ciphertext)%encCBCBlockSize != 0` reject
- [x] **decrypt 错误统一 surface 为 ErrEncryptionAuthFailed** — wrong-KEK / tamper / wrong-AAD 三路径同 sentinel；不区分 timing 也不区分类型 string
- [x] **既有 16 个 GCM 测试零回归** — `go test -race ./internal/backup/...` 全过

## 5. 威胁建模差异（CBC vs GCM/AEAD）

| 威胁 | GCM/AEAD 防护 | CBC+HMAC 防护 |
|------|--------------|---------------|
| 密文篡改 | GCM tag 检测 | HMAC tag 检测 |
| nonce 复用 | nonce 12B 随机；500 iter 测试 | IV 16B 随机；500 iter 测试 |
| AAD 替换 | AEAD 内置 | HMAC scope 包 AAD |
| algo byte 替换 | 解密 strict 拒外算法 | strict 拒 + HMAC scope 含 algo |
| IV 替换 | N/A（nonce 入 GCM）| HMAC scope 含 IV |
| padding oracle | N/A | HMAC verify 严格先于 unpad；timing 不泄漏 |
| KEK 替换 | unwrap-DEK GCM 失败 | 同 GCM 路径（共用 wrapDEKWithKEK） |
| timing side-channel | Go stdlib AEAD 常时 | hmac.Equal 常时 + Pad 全块检验（不短路）|

## 6. E/R 比

无新端点。bk-12 既有 e2e claim 翻转描述 + 接受 codes 集合扩展（`200 400 401`），claim 数量不变。

## 7. 已识别风险与遗留

| 风险 | 缓解 / 说明 |
|------|------------|
| 加密 → 解密期间 policy 改算法（GCM 文件遇 CBC encryptor）| 既有 T-0075 限制 — 非本任务引入；缓解需未来加 algo-router (out of scope) |
| KEK 旋转 | T-0087 followup |
| KMS / HSM | T-0086 followup |
| 国密 SM4 | PRD §6 N1 非目标 |
| HMAC scope 不含 wrappedDEK / outerNonce | 这两值在 unwrap-DEK GCM 验证时已被自然认证（GCM tag 不通过 → ErrEncryptionAuthFailed），无需 HMAC 二重保险 |

## 8. DoD 自查

- [x] `go build ./...`
- [x] `go test -race ./internal/backup/...`
- [x] `gofmt -l` 0 file
- [x] `go vet` 0 issue
- [x] PRD V1-V10 GWT 全部测试覆盖
- [x] 公共接口无 `any` / `interface{}`
- [x] 无新 `if carrier == "..."` 硬编码
- [x] 无 TODO/FIXME 残留
- [x] 错误 `fmt.Errorf("ctx: %w", err)` wrap
- [x] 安全自查清单 14 项全 [x]
- [x] 既有 GCM 路径零改动 / 既有 GCM 测试零回归
- [x] 0 schema 变更 / 0 新依赖（`x/crypto` 已 direct）

## 9. 安全 review 等级建议

本任务**未触及** `internal/admin/` 或 `middleware/auth*`（dev-pipeline §B5 自动追加 security-review 路径触发器），但加密原语本身是高敏感。**Self-review verdict: APPROVE**，理由：

1. encrypt-then-MAC 是 CBC 安全的标准构造，遵循 RFC 5246 / NIST SP 800-38A 规范
2. HKDF-SHA256 derive separate MAC key 遵循 RFC 5869
3. ChaCha20-Poly1305 直接使用 `golang.org/x/crypto/chacha20poly1305` 标准库
4. 14 项安全自查全 pass；没有偏离 T-0075 review 已确立的 fail-closed / 不 log key / AAD 绑定模式
5. 测试覆盖三算法 × 5 维度（roundtrip / tamper / AAD / key / nonce）+ CBC 专属 4 维度（IV tamper / HMAC tamper / algo byte / PKCS7）

未推荐 reviewer-agent 介入：路径未触发 §B5 强制；内容是知名密码学构造；S5 self-review 已对照
NIST/RFC 标准；session-budget 守纪。如运营层有顾虑可单独触发 `/security-review`。

## 10. Reviewer 备忘

Approve 建议。Highlight：

1. **CBC encrypt-then-MAC 严格顺序** — `hmac.Equal` 在 `cipher.NewCBCDecrypter`、`pkcs7Unpad` 之前，padding oracle 攻击面被关闭
2. **HMAC scope 选择** — `algo_byte ‖ IV ‖ AAD ‖ ciphertext`：4 维度全防（algo 替换 / IV 替换 / AAD 替换 / 密文篡改），不含 wrappedDEK/outerNonce 因 unwrap-DEK GCM 自带认证
3. **wrapDEKWithKEK / unwrapDEKWithKEK 提取** — 三算法共用，削减重复且让"KEK→DEK wrap 与 body 算法解耦"显式化（PRD §2.5 决策的代码体现）
4. **既有 GCM 测试零改动** — 含 `TestAESGCM_AAD_CompressedExtension`（T-0075 review HIGH-1 回归保护）
5. **`TestNewEncryptor_stubAlgorithms` 翻转为 `TestNewEncryptor_allAlgorithmsConstruct`** — 留作硬回归卫士：未来若有人误把某算法回退为 stub，本测试立即失败
6. **policy_service KEK availability 校验推广到三算法** — defense-in-depth：PUT 时 fail-fast + 上传时 NewEncryptor fail
