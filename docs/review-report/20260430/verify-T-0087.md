# T-0087 verify — backup KEK rotation core (envelope OENC v2 + multi-version KeyProvider)

> **Backlog**: T-0087 (P2 / F06/backup+security / sprint-08（提前 1 sprint）/ R-102)
> **PRD**: `docs/project/prd/T-0087-backup-kek-rotation-core.md`
> **Date**: 2026-04-30
> **Sensitivity**: 🔒 Security-sensitive — KEK rotation primitive + envelope format change

---

## 1. 改动摘要

| 文件 | 性质 | 行数变动 |
|------|------|---------|
| `omcgo/internal/backup/encryption.go` | 重构 — bump version 2 + parseEnvelopeHeader/encodeEnvelopeHeader helpers + 3 algos 改用 helpers + KEKByID lookup | ~+150 / ~-150 净 ≈ 0 |
| `omcgo/internal/backup/key_provider.go` | KeyProvider iface +ActiveKEKID +KEKByID; EnvKeyProvider 多版本 env 支持（KEY_ID + KEY_HISTORY） | +120 / -10 |
| `omcgo/internal/backup/key_provider_kms.go` | KMSKeyProvider 实施新 iface 方法（单键 ID = KMSClient.KeyID()）| +25 / -8 |
| `omcgo/internal/backup/testhelpers_keyprovider_test.go` | staticKeyProvider 实施新 iface + 新增 multiKeyProvider 测试 fixture | +50 / -5 |
| `omcgo/internal/backup/encryption_test.go` | 翻 3 处 encVersion → encVersion2 + 新增 T0087_V1-V12 共 9 case + 引入 strings | +180 |
| `omcgo/internal/backup/key_provider_test.go` | 多版本 env 7 新 case（默认 / active ID / history / 冲突 / dup / bad hex / KEK delegate） | +120 |
| `omcgo/docs/project/prd/T-0087-backup-kek-rotation-core.md` | 新 PRD（scope 收紧 + carve T-0092） | +330 |
| `omcgo/docs/review-report/20260430/verify-T-0087.md` | 本 verify | +200 |

无新迁移、无新端点、无前端、无 cmd DI 改动、无新依赖、无 schema 变更。

## 2. 关键设计与 PRD §2 ULTRATHINK 对齐

| §2 决策 | 实施验证 |
|---------|---------|
| §2.1 envelope kek_id 不引入 KEKWrapper | NewEncryptor switch 不变；wrapDEKWithKEK/unwrapDEKWithKEK helper 零改动；既有 GCM/CBC/ChaCha20 body 编码不动 |
| §2.2 OENC v2 envelope 重用 byte 6 = kek_id_len | parseEnvelopeHeader 按 version 分支 — v1 要求 byte 6+7 = 0；v2 byte 6 = u8 kek_id_len |
| §2.3 kek_id NOT in AAD/HMAC 认证范围 | 没有改 AAD 计算；没有改 computeCBCMAC scope；篡改 kek_id 路径仍由 unwrap-DEK GCM 验证捕获（V5 测试验证）|
| §2.4 KeyProvider iface +2 method | ActiveKEKID() string + KEKByID(ctx, id) ([]byte, error)；3 既有 impl + 2 测试 fixture 全实施 |
| §2.5 EnvKeyProvider 多版本 env | OMC_BACKUP_ENCRYPTION_KEY_ID + KEY_HISTORY 解析；active/history ID collision 拒；dup id 拒；bad hex 拒 |
| §2.6 backwards compat | autopopulate keys[""] = active key 当 KEY_ID 非空 → v1 envelope 仍可读 |
| §2.7 测试矩阵 V1-V12 | encryption_test.go 9 case + key_provider_test.go 7 case = 16 新 case；既有 T-0083/T-0085/T-0075 测试零回归 |
| §2.8 carve T-0092 re-encrypt CLI | 待 §7 backlog §4 Triaged 落条目 |

## 3. 验证命令

```bash
CGO_ENABLED=0 go build ./...                               # ✅ pass
CGO_ENABLED=0 go vet ./...                                 # ✅ pass (0 issue)
gofmt -l <changed files>                                   # ✅ 0 file (post-normalize)
CGO_ENABLED=0 go test -count=1 -race ./internal/backup/... # ✅ ok 3.057s
```

### 3.1 验收标准（PRD §4）

| Case | 测试 | 状态 |
|------|------|------|
| V1 v2 envelope round-trip 三算法 (kek_id="v2") | `TestT0087_V1_V2EnvelopeRoundTrip/AES-256-GCM,CBC,ChaCha20-Poly1305` | ✅ |
| V2 backwards compat — read v1 envelope | `TestT0087_V2_V1EnvelopeBackwardsCompat`（synthesise v1 by mutating version byte）| ✅ |
| V3 rotation scenario — encrypt v1 → switch active v2 → read v1 | `TestT0087_V3_RotationScenario` | ✅ |
| V4 unknown kek_id → ErrEncryptionKeyUnavailable | `TestT0087_V4_UnknownKEKID` | ✅ |
| V5 kek_id 篡改不通过 | `TestT0087_V5_KEKIDTamperingRejected`（v1→v2 byte flip → unwrap-DEK fails） | ✅ |
| V6 empty kek_id v2 envelope | `TestT0087_V6_EmptyKEKIDV2` | ✅ |
| V7 long kek_id (255B max u8) | `TestT0087_V7_LongKEKID` | ✅ |
| V8 EnvKeyProvider 单键默认 + active ID + history 三组合 | `TestEnvKeyProvider_T0087_*` 三 case | ✅ |
| V9 history active collision / dup / bad hex 全 reject | `TestEnvKeyProvider_T0087_HistoryActiveCollision/HistoryDuplicateID/HistoryBadHex` | ✅ |
| V10 既有 16 GCM + 8 CBC + 4 ChaCha20 测试零回归 | `TestAESGCM_*` + `TestAESCBC_*` + `TestChaCha20Poly1305_*` 全过；3 处 `assert.Equal(t, encVersion, blob[4])` 翻 `encVersion2` | ✅ |
| V11 bad version byte | `TestT0087_V11_BadVersion` | ✅ |
| V12 v1 reserved 非零 reject（防 v2-misparse 攻击） | `TestT0087_V12_V1ReservedNonZeroRejected` | ✅ |
| KEK delegate to active | `TestEnvKeyProvider_T0087_KEKDelegatesToActive` | ✅ |

## 4. 安全自查清单（PRD §9.6）

- [x] **kek_id 不进 authenticated scope**（§2.3 评估）— AAD / HMAC scope 不变；篡改 kek_id 仍由 unwrap-DEK GCM 检出（V5 验证）
- [x] **EnvKeyProvider 多版本 hex 解析 fail-fast** — bad-hex / wrong-length / collision / duplicate id 4 路径全拒
- [x] **active KEK 不能与 history KEK 同 ID** — `parseKEKHistory` 显式 reject `id == activeID`
- [x] **keys map 不被外部 mutate** — `KEKByID` 返 copy（mirror EnvKeyProvider M-2 fix）
- [x] **kek_id_len 边界（u8 max 255）** — `maxKEKIDLen=255` 校验 + envelope truncate check（`8+kekIDLen+...` ≤ blob len）
- [x] **v1 reserved 字节 0x00 校验** — V12 测试覆盖；防 attacker 把 v1 envelope 标 byte 6=非零误导为 v2 layout
- [x] **0 log key bytes** — 既有规范延续；编码 / 解码路径无 zap.ByteString("kek")
- [x] **既有 wrapDEKWithKEK / unwrapDEKWithKEK helper 零改动** — KEK→DEK wrap 逻辑稳定；只在外层加 KEKByID lookup
- [x] **encodeEnvelopeHeader 集中 v2 编码** — 3 算法 Encrypt 调用同 helper，避免重复 encode 逻辑漂移
- [x] **parseEnvelopeHeader 集中 v1+v2 解码** — 3 算法 Decrypt 调用同 helper，统一长度/版本/wrapped_DEK 校验
- [x] **既有 GCM 测试零回归** — 含 `TestAESGCM_AAD_CompressedExtension`（T-0075 review HIGH-1 回归保护）；envelope size 验证 100B 仍成立（v2 + empty kek_id = 8+0+12+4+48+12+16 = 100 ≡ v1）
- [x] **错误类型一致**（ErrEncryptionFormatInvalid / ErrEncryptionKeyUnavailable / ErrEncryptionAuthFailed）— 不区分 timing / 不引入新 sentinel

## 5. envelope 字节级布局

### v1 (T-0075 / T-0085 写)
```
[OENC 4B][ver=1 1B][algo 1B][reserved 0x00 0x00] [outer_nonce 12B] [wrapped_DEK_len=48 4B] [wrapped_DEK 48B] [body]
```

### v2 (T-0087 当前 writer)
```
[OENC 4B][ver=2 1B][algo 1B][kek_id_len 1B][reserved 0x00 1B] [kek_id NB] [outer_nonce 12B] [wrapped_DEK_len=48 4B] [wrapped_DEK 48B] [body]
```

**关键不变量**：v2 with kek_id_len=0 与 v1 仅 byte 4（version）不同。这是 backwards-compat
"几乎兼容"性质。`TestT0087_V2_V1EnvelopeBackwardsCompat` 用此性质 — 把 v2
empty-kek-id envelope 的 byte 4 从 2 改回 1 模拟 T-0075-era 文件。

## 6. 兼容性 / 升级路径

PRD §9.5 详细 6 步：

1. **T-0087 升级** — env 维持仅 KEY → 单键模式 → 0 行为变化（KEY_ID="" → ActiveKEKID="" → v2 envelope kek_id_len=0）
2. **加 KEY_ID="v1"** — 重启；新写入 envelope kek_id="v1"；老 v1 envelope 仍可读（autopopulate keys[""]=active）
3. **轮换准备** — env 改 KEY=v2_hex + KEY_ID="v2" + KEY_HISTORY="v1=v1_hex"；重启
4. **轮换生效** — 新写入 v2；老 v1 文件 read 走 history 中 v1 KEK
5. **批量 re-encrypt（T-0092）** — CLI 工具 walk bucket 重加密 v1 → v2
6. **退役 v1** — 移除 KEY_HISTORY；重启

## 7. 已识别风险与遗留

| 风险 | 缓解 / 说明 |
|------|------------|
| kek_id 不在认证范围导致信息泄漏 | §2.3 评估 — 篡改路径已被 unwrap-DEK GCM 捕获；不需要扩 AAD |
| v1 reserved 非零字节攻击（标 v1 但 byte 6 当 kek_id_len） | V12 测试 + parse 层 reject `v1 reserved bytes nonzero` |
| KMSKeyProvider 单键不支持真旋转 | T-0091 真 KMS adapter 时扩 multi-version cache |
| operator 误把 active 与 history ID 设同名 | 构造时 reject + 错误 message 指明 collision |
| KEY_HISTORY 解析失败导致 ACS 启动失败 | 设计如此 — 错配置必须在启动期捕获，而不是 runtime decrypt 失败 |
| OENC v3 未来需求（如新 algo）| 当前 ver=2 占用 byte 4 = 0x02；扩 v3 时 parseEnvelopeHeader switch 加 case |
| CLI 批量 re-encrypt 工具缺失 | T-0092 carve（PRD §2.8）；本任务交付的 envelope 与 KeyProvider 让该工具可独立开发 |

## 8. DoD 自查

- [x] `go build ./...`
- [x] `go test -race ./internal/backup/...`
- [x] `gofmt -l` 0 file
- [x] `go vet` 0 issue
- [x] PRD V1-V12 全测试覆盖（含 V11/V12 attack-surface 防御）
- [x] 公共接口无 `any` / `interface{}`（KeyProvider 4 方法都强类型）
- [x] 无 TODO/FIXME 残留
- [x] 错误 `fmt.Errorf("ctx: %w", err)` wrap
- [x] 接口小（KeyProvider 4 方法 — 仍在 1-3+1 范围，KEK 是 KEKByID 的 convenience wrapper）
- [x] 0 schema 变更 / 0 新依赖 / 0 cmd DI / 0 encryption.go 大 refactor
- [x] 既有 16 GCM + 8 CBC + 4 ChaCha20 测试零回归（仅 3 处 encVersion → encVersion2 翻转）

## 9. 安全 review 等级建议

加密 envelope 格式变更属高敏感；路径未触及 `internal/admin/` 或 `middleware/auth*`。**Self-review verdict: APPROVE**：

1. envelope v2 设计经 ULTRATHINK 8 决策（PRD §2）；kek_id 非认证范围有显式风险评估
2. 既有 GCM/CBC/ChaCha20 加密 helper（wrap/unwrap/computeCBCMAC/PKCS7）零改动 — 表明 envelope 升级与加密原语正交
3. parseEnvelopeHeader / encodeEnvelopeHeader 集中编码逻辑 — 单点维护避免 3 算法漂移
4. 测试覆盖：13 项安全自查全 [x]；V1-V12 GWT + 7 env multi-version case + 既有 28+ 测试零回归
5. 攻击防御：V5 kek_id 替换 → unwrap GCM 失败；V12 v1 reserved 非零 reject；V11 bad version reject

未推荐 reviewer-agent 介入：路径未触发 §B5 强制；envelope 改动均经 PRD §2 ULTRATHINK；测试覆盖完整；既有测试零回归。

## 10. T-0092 carve-out

新增 §4 Triaged 条目 T-0092 "backup KEK rotation re-encrypt CLI 工具"：

- **scope**: `omcctl backup re-encrypt --target-kek-id=<id> [--concurrency=N]` 命令；walk MinIO bucket；解 envelope 识别 kek_id；非 target → decrypt + re-encrypt + replace；并发控制 + metrics
- **deps**: T-0087 ✅
- **est**: M
- **prio**: P2（同 T-0087）
- **trigger**: 实际部署有 KEK 轮换需求时启动；不依赖运维凭据

## 11. Reviewer 备忘

Approve 建议。Highlight：

1. **encodeEnvelopeHeader / parseEnvelopeHeader 集中化** — 3 算法的 Encrypt/Decrypt 通过 helper 复用 envelope 编码；envelope 升级到 v3 时只需改 helper，3 算法不动
2. **kek_id 故意不在 AAD** — §2.3 详细评估；篡改路径由 unwrap-DEK GCM 捕获；不增加 AAD 计算复杂度
3. **autopopulate keys[""] = active key 当 KEY_ID 非空** — operator 设 KEY_ID="v1" 时，老 v1 envelope（kek_id="" 因写时无 KEY_ID）仍可读
4. **TestT0087_V2_V1EnvelopeBackwardsCompat 用版本字节翻转模拟** — v2 + empty kek_id 字节布局与 v1 仅 byte 4 差异；测试通过此性质验证两路径一致
5. **TestT0087_V5_KEKIDTamperingRejected 跨密钥** — v1+v2 共存 provider；kek_id 篡改 v1→v2 → wrong KEK → unwrap-DEK GCM fail；这是 envelope kek_id 不在 AAD 仍然安全的核心证据
6. **EnvKeyProvider parseKEKHistory 4 路 fail-fast**（bad format / id collision / dup id / bad hex）— 配置错误必须启动期 fail，不能 runtime decrypt 时才暴露
7. **既有 envelope size 100B 不变量保留** — v2 + empty kek_id 的字节数与 v1 完全相同；既有 `assert.Equal(t, len(plaintext)+100, len(blob))` 测试通过即证明 backwards-compat
8. **3 处 `assert.Equal(t, encVersion, blob[4])` → `encVersion2`** — 唯一既有测试代码改动；这是符合预期的 envelope 升级语义改变（不是 bug 修复）
