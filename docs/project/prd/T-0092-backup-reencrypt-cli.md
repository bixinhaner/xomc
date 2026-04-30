# PRD: backup KEK rotation re-encrypt CLI（T-0092）

> **关联**: Backlog T-0092 / Sprint-08 / Domain=F06/backup+security+ops / Type=feat / Prio=P2
> **作者**: Claude（代 Owner=电信+SecOps+运维）
> **创建**: 2026-04-30
> **状态**: 草案 → 实施
> **关键决策**: 独立二进制 `cmd/backup-reencrypt/`（不并入 omcctl，因 omcctl 是 HTTP-only client；re-encrypt 是直访 MinIO + 本地 env KEK 的 data-plane 操作）

---

## 1. 业务背景

T-0087 落 envelope OENC v2 + 多版本 KeyProvider，让 KEK 旋转的"读"路径可工作（v1
envelope 走 history KEK）。但**写**路径要彻底退役旧 KEK 仍需把所有历史文件重加密
到新 active KEK 下 — 这就是本任务交付的工具。

运维场景：

- KEK 泄漏紧急轮换：`omcgo-backup-reencrypt --target-kek-id=v2` 走完整 bucket，
  迁移 v1 文件到 v2，迁移完成后 KEY_HISTORY 可移除
- 季度 / 年度合规轮换：同上，但有 days/weeks 的迁移窗口
- 部署初次迁移到 KEY_ID 模式：原本 KEY_ID="" 的环境改 KEY_ID="v1"，老 v1 envelope
  无 kek_id 仍可读（autopopulate），但运营商希望"显式"标记旧文件 — 同样跑此工具
  到 target=v1

不做：实时 / 触发式 re-encrypt（那是 T-0083 reaper / T-0073 cleanup 逻辑域）。

---

## 2. ULTRATHINK 决策

### 2.1 独立二进制 vs omcctl 子命令

候选：

| 方案 | 优 | 劣 |
|------|---|---|
| A. 独立 `cmd/backup-reencrypt/main.go` | 数据面操作 + 本地环境，无 HTTP 中继；架构最干净；单一职责 | 多一个二进制需 Makefile 维护 |
| B. omcctl 加 `backup re-encrypt` 子命令 | 复用 omcctl entry 框架 | omcctl 是 HTTP client（API key + server URL）；混入直访 MinIO 模式让 omcctl 变成"两种工作模式"，违反单一职责 |
| C. ACS HTTP 端点 + omcctl 调用 | RPC 化操作 | 本任务范围远超：要新 ACS endpoint + auth + RBAC + progress streaming → est XL |

采纳 A。理由：
- re-encrypt 是 long-running ops 任务（数千文件 × 数十 GB），HTTP 中继不合适
- 运维已熟悉直接 `OMC_BACKUP_ENCRYPTION_KEY` 配 env 的部署模式（cmd/acs/main.go
  也是这样工作的）— 同一 host 跑同一 env 即可
- 一次性命令（不是常驻服务），独立二进制更好打包

### 2.2 复用既有 KeyProvider + Encryptor

不引入新 crypto 路径。re-encrypt 工具：

```
input  blob ──> NewEncryptor(algo, kp).Decrypt(blob, AAD) ──> plaintext
plaintext ──> NewEncryptor(activeAlgo, kp).Encrypt(plaintext, AAD) ──> output blob
```

**同一 KeyProvider 实例**：源 KEK（envelope 标的 kek_id）和目标 KEK（kp.ActiveKEKID()）
都通过 KEKByID 查询。运维只需保 KEY_HISTORY 含源 KEK + KEY_ID 设为目标。

### 2.3 AAD 重建

T-0085 review HIGH-1 fix：AAD = 文件名（含 cmp.ext，不含 .enc）。

re-encrypt 路径 AAD 重建：
- input key: `backup/2026/03/29/backup-aabbccdd-SN999.xml.gz.enc`
- 取 basename: `backup-aabbccdd-SN999.xml.gz.enc`
- 剥 `.enc` suffix: `backup-aabbccdd-SN999.xml.gz`
- 此即 AAD（与 ACS upload 写入时计算逻辑对称）

工具不需要解析其他元数据 — 文件名规则是契约。

### 2.4 算法选择 — 跨算法 re-encrypt 不在此任务

如果源是 GCM 而目标 algo 改为 CBC，re-encrypt 是否切算法？

**不切**。本任务专注 KEK 旋转（kek_id 变化），算法保持 envelope 标记的原值。
跨算法迁移是独立场景：运维若需切 GCM→CBC，应：
1. 改 BackupPolicy.EncryptionAlgorithm + 重启 ACS
2. 跑此工具迁移到新 active KEK（envelope 仍是旧算法）
3. **未来**单独工具 cross-algo re-encrypt（非本任务）

实施层面：parseEnvelopeHeader 已返 algo byte → 用同算法 NewEncryptor 解密 +
**用同算法**重加密。读 hdr.algo 决定。

### 2.5 跳过条件

工具在 walk bucket 过程中跳过下列文件：

| 跳过原因 | 计数 metric label | 行为 |
|---------|-----------------|------|
| 已是 target kek_id | `skip_already_target` | 不下载，不重写 |
| 不是 .enc 文件 | `skip_not_encrypted` | 不下载（直接看后缀） |
| envelope 不可解析（magic mismatch / version 不支持 / 篡改）| `skip_envelope_invalid` | log warn，不动文件 |
| 源 KEK 在 KeyProvider 不存在 | `skip_source_kek_unavailable` | log warn，建议运维补 KEY_HISTORY；不动文件 |
| MinIO API 错误（GET / PUT） | `skip_minio_error` | log warn，不动文件 |

**安全默认**：失败的文件保持原样，下次运行可重试。

### 2.6 并发控制

每 worker 处理一个文件：GET(64MB cap) → decrypt → encrypt → PUT。peak 内存 ≈
2×plaintext + envelope overhead ≈ 130MB。

默认 concurrency=4 → peak ≈ 520MB。CLI flag `--concurrency=N` 可调（1..32）。
mirror T-0089 semaphore pattern。

### 2.7 进度报告

不引入第三方 TUI 依赖。每 N 文件（N=100 默认）log 一行 progress：

```
{"level":"info","msg":"reencrypt progress","processed":1234,"reencrypted":890,"skipped":344,"failed":0,"elapsed":"2m15s"}
```

最终 summary log：

```
{"level":"info","msg":"reencrypt completed","total":5000,"reencrypted":3850,"skip_already_target":1100,"skip_envelope_invalid":3,"skip_minio_error":2,"failed":0,"elapsed":"15m23s"}
```

### 2.8 观察性 metrics

2 新 collector 加入既有 backup.PolicyMetrics（当 ProcessRegistry 配 prom 时启用）：

- `omc_backup_reencrypt_total{result}` — counter，result ∈ {reencrypted, skip_already_target, skip_not_encrypted, skip_envelope_invalid, skip_source_kek_unavailable, skip_minio_error, failed}
- `omc_backup_reencrypt_duration_seconds` — histogram，单文件 re-encrypt 延迟

工具运行时如果配了 `--metrics-listen=:9094`，启动 HTTP /metrics endpoint
让 Prometheus 在工具运行期间抓取（可选；默认 disabled）。

### 2.9 测试矩阵

mirror policy_orphan_reaper_test.go fakeBucketLister + fakeMinIORemover 模式
+ 新 fakeMinIOObjectIO（接 GetObject + PutObject）：

- T1: 单文件 v1 envelope (kek_id="v1") → re-encrypt → output kek_id="v2"
- T2: 单文件已 target → skip_already_target，不调 PutObject
- T3: 多文件混合（v1 + 已 v2 + 非 .enc）
- T4: source KEK missing → skip_source_kek_unavailable
- T5: envelope corrupt → skip_envelope_invalid
- T6: PutObject 失败 → failed，继续下一文件
- T7: ctx cancel 中途 → 优雅退出，已处理文件保留
- T8: 三算法（GCM/CBC/ChaCha20）re-encrypt 各跑一遍

---

## 3. 用户故事

| 角色 | 故事 |
|------|------|
| SecOps（KEK 紧急轮换）| KEK 泄漏后跑 `OMC_BACKUP_ENCRYPTION_KEY=<new> OMC_BACKUP_ENCRYPTION_KEY_ID=v2 OMC_BACKUP_ENCRYPTION_KEY_HISTORY="v1=<old>" backup-reencrypt --target-kek-id=v2` —— 30 分钟内完成 N 千文件迁移 |
| 合规审计 | 季度密钥轮换流程化 — 工具 stdout JSON log 可作 audit trail |
| 运维（首次部署 KEY_ID）| 老部署 KEY_ID="" → 改 KEY_ID="v1" → 跑工具 target=v1 让所有文件显式标记 v1 envelope |
| QA | 工具支持 dry-run 验证（`--dry-run` 跳过 PutObject + 报告会做什么） |
| 后端开发 | 工具复用现有 KeyProvider + Encryptor — 无新 crypto 路径 |

---

## 4. 验收标准（GWT）

### V1 — single-file v1 → v2

- **Given** bucket 含 1 文件 envelope kek_id="v1"，KeyProvider active=v2 + history v1
- **When** Reencryptor.Run(ctx, target="v2")
- **Then** PutObject 调一次 同 key；新 envelope kek_id="v2"；reencrypted=1

### V2 — already on target

- **Given** 文件已 envelope kek_id="v2"
- **When** Reencryptor.Run(ctx, target="v2")
- **Then** PutObject 不调；skip_already_target=1

### V3 — mixed bucket

- **Given** 5 文件：3 v1，1 v2，1 非-.enc
- **When** Reencryptor.Run target="v2"
- **Then** 3 reencrypted + 1 skip_already_target + 1 skip_not_encrypted

### V4 — source KEK missing

- **Given** 文件 kek_id="v0"，KeyProvider 仅有 v1+v2
- **When** Reencryptor.Run target="v2"
- **Then** skip_source_kek_unavailable=1；不动文件

### V5 — envelope corrupt

- **Given** 文件 magic 错乱
- **When** Reencryptor.Run
- **Then** skip_envelope_invalid=1；不动文件；不 panic

### V6 — PutObject 失败继续

- **Given** 3 文件，第 2 个 PutObject 返 error
- **When** Reencryptor.Run
- **Then** 3 文件都尝试；2 reencrypted + 1 failed

### V7 — ctx cancellation 优雅退出

- **Given** 长 list of 文件，runner 启动后 cancel ctx
- **When** Reencryptor.Run
- **Then** 已处理的保留；未处理的不动；返 ctx.Err

### V8 — 三算法

- **Given** bucket 含 1 GCM + 1 CBC + 1 ChaCha20-Poly1305 文件，全 v1
- **When** Reencryptor.Run target="v2"
- **Then** 各 re-encrypt；输出 envelope 算法不变（algo byte 保留），仅 kek_id 改 v2

### V9 — dry-run 不写

- **Given** 5 v1 文件，target=v2，--dry-run
- **When** Run
- **Then** 0 PutObject 调用；progress log 报告 "would re-encrypt N"

### V10 — concurrency 1..32 边界

- **Given** --concurrency=0 或 -1 或 33
- **When** main 启动
- **Then** clamp 到 1..32（≤0 → 1，>32 → 32）+ warn log

---

## 5. 运营商差异矩阵

无差异。KEK 旋转策略是 OMC 内部 ops，与运营商无关。

---

## 6. 非目标

| # | 非目标 | 原因 |
|---|--------|------|
| N1 | 跨算法 re-encrypt（GCM↔CBC↔ChaCha20）| §2.4：独立场景；本任务专注 kek_id 旋转 |
| N2 | omcctl 子命令集成 | §2.1 架构选择；HTTP 中继不适合 long-running ops |
| N3 | 进度持久化（resume 状态文件）| 工具自然幂等（已 target 跳过）；操作可重启 |
| N4 | 加 ACS endpoint 触发 | T-0091 + 真 KMS adapter 时再考虑（届时密钥已远程托管）|
| N5 | TUI / 进度条 UI | JSON log 已足够；不引入第三方依赖 |
| N6 | 加密算法升级建议（如 v0 deprecated）| 不在本任务；文档化由运维决策 |

---

## 7. 依赖

| 依赖 | 状态 |
|------|------|
| T-0075 加密框架 | ✅ done |
| T-0085 三算法矩阵 | ✅ done |
| T-0086 KMSKeyProvider skeleton | ✅ done |
| T-0087 envelope OENC v2 + multi-version KeyProvider | ✅ done |
| `golang.org/x/sync/semaphore` 或本地 channel | 用本地 buffered channel（mirror T-0089）|
| `github.com/spf13/cobra` | 已 direct dep（omcctl 用） |

不引入新第三方 dep。

---

## 8. 度量

§2.8 已列。

---

## 9. 设计备忘（S2）

### 9.1 文件清单

新增：

- `omcgo/internal/backup/reencryptor.go` — Reencryptor service + narrow ObjectGetter/ObjectPutter consumer interfaces + sem 控制 + 单文件 re-encrypt loop
- `omcgo/internal/backup/reencryptor_test.go` — V1-V10 测试 + fake io
- `omcgo/cmd/backup-reencrypt/main.go` — 独立二进制 entry：load env + MinIO + KeyProvider + Encryptor factory + Reencryptor + signal handling

修改：

- `omcgo/internal/backup/policy_metrics.go` — +2 collector + 2 Record helper

无新依赖、无 schema、无 cmd 既有改动、无 encryption.go 改动。

### 9.2 Reencryptor 接口

```go
type Reencryptor struct {
    bucket       string
    targetKekID  string
    concurrency  int
    dryRun       bool
    bucketLister BucketLister      // T-0082 既有 narrow iface
    objectIO     ObjectIO          // 新 narrow iface — GetObject + PutObject
    kp           KeyProvider       // T-0087 接口
    metrics      *PolicyMetrics
    logger       *zap.Logger
}

type ObjectIO interface {
    GetObject(ctx context.Context, bucket, key string, opts minio.GetObjectOptions) (*minio.Object, error)
    PutObject(ctx context.Context, bucket, key string, reader io.Reader, size int64, opts minio.PutObjectOptions) (minio.UploadInfo, error)
}

type ReencryptStats struct {
    Reencrypted               int64
    SkipAlreadyTarget         int64
    SkipNotEncrypted          int64
    SkipEnvelopeInvalid       int64
    SkipSourceKEKUnavailable  int64
    SkipMinIOError            int64
    Failed                    int64
}

func (r *Reencryptor) Run(ctx context.Context) (ReencryptStats, error)
```

### 9.3 单文件 re-encrypt 流程

```go
func (r *Reencryptor) reencryptOne(ctx context.Context, key string) reencryptOutcome {
    // 1. 不是 .enc 后缀 → skip_not_encrypted
    if !strings.HasSuffix(key, ".enc") { return outcomeSkipNotEncrypted }
    
    // 2. GetObject 拉 ciphertext blob
    obj, err := r.objectIO.GetObject(ctx, r.bucket, key, ...)
    if err != nil { return outcomeSkipMinIOError }
    blob, err := io.ReadAll(io.LimitReader(obj, encMaxPlaintext+encEnvelopeOverhead))
    if err != nil { return outcomeSkipMinIOError }
    
    // 3. parseEnvelopeHeader → kek_id + algo
    hdr, err := parseEnvelopeHeader(blob)
    if err != nil { return outcomeSkipEnvelopeInvalid }
    
    // 4. 已 target → skip_already_target
    if hdr.kekID == r.targetKekID { return outcomeSkipAlreadyTarget }
    
    // 5. AAD = basename(key) - .enc
    aad := []byte(strings.TrimSuffix(filepath.Base(key), ".enc"))
    
    // 6. 用源 algo + kek 解密
    enc, err := NewEncryptor(algoToString(hdr.algo), r.kp)
    if err != nil { return outcomeSkipSourceKEKUnavailable }  // KEKByID 在 Decrypt 内查
    plaintext, err := enc.Decrypt(blob, aad)
    if err != nil { return outcomeSkipSourceKEKUnavailable } // wrap 已统一
    
    // 7. 用 same algo + active kek 重加密
    newBlob, err := enc.Encrypt(plaintext, aad)
    if err != nil { return outcomeFailed }
    
    // 8. dry-run 提前出
    if r.dryRun { return outcomeReencryptedDryRun }
    
    // 9. PutObject 覆盖
    _, err = r.objectIO.PutObject(ctx, r.bucket, key, bytes.NewReader(newBlob), int64(len(newBlob)), ...)
    if err != nil { return outcomeFailed }
    return outcomeReencrypted
}
```

### 9.4 cmd/backup-reencrypt/main.go entry

```go
func main() {
    var (
        targetKekID  string
        concurrency  int
        dryRun       bool
        configPath   string
        // metricsListen string  // 可选；MVP 不接
    )
    rootCmd := &cobra.Command{
        Use:   "backup-reencrypt",
        Short: "Re-encrypt all backup files in MinIO bucket to a target KEK",
        RunE: func(cmd *cobra.Command, args []string) error {
            if targetKekID == "" {
                return fmt.Errorf("--target-kek-id required")
            }
            // clamp concurrency
            if concurrency <= 0 { concurrency = 1 }
            if concurrency > 32 { concurrency = 32 }
            
            cfg, err := loadConfig(configPath)
            if err != nil { return err }
            
            kp, err := backup.NewEnvKeyProvider()
            if err != nil { return err }
            if !kp.Available() { return fmt.Errorf("KEK not configured") }
            
            mc, err := minioClientFromConfig(cfg)
            if err != nil { return err }
            
            // sanity: target KEK must be available
            if _, err := kp.KEKByID(cmd.Context(), targetKekID); err != nil {
                return fmt.Errorf("target KEK id=%q not available; check OMC_BACKUP_ENCRYPTION_KEY_ID + KEY_HISTORY env: %w", targetKekID, err)
            }
            
            r := backup.NewReencryptor(backup.ReencryptorConfig{
                Bucket:      cfg.MinIO.Buckets.ConfigBackup,
                TargetKekID: targetKekID,
                Concurrency: concurrency,
                DryRun:      dryRun,
                Lister:      mc,
                IO:          mc,
                KeyProvider: kp,
                Logger:      logger,
            })
            
            ctx, cancel := context.WithCancel(cmd.Context())
            defer cancel()
            // signal handling
            sigCh := make(chan os.Signal, 1)
            signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
            go func() {
                <-sigCh
                logger.Info("interrupt received; cancelling")
                cancel()
            }()
            
            stats, err := r.Run(ctx)
            logger.Info("reencrypt summary", zap.Any("stats", stats))
            if err != nil {
                return err
            }
            return nil
        },
    }
    rootCmd.Flags().StringVar(&targetKekID, "target-kek-id", "", "Target KEK id (required)")
    rootCmd.Flags().IntVar(&concurrency, "concurrency", 4, "Worker count (clamped 1..32)")
    rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Walk bucket but skip PutObject")
    rootCmd.Flags().StringVar(&configPath, "config", "", "Path to config YAML (defaults to env-only)")
    
    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}
```

实际实施时 loadConfig + minioClientFromConfig 可简化（仅 env-based MinIO config，
不依赖 yaml）— mirror cmd/migrate/main.go 简单模式。

### 9.5 安全自查清单

- [ ] AAD 重建逻辑与 ACS upload/download 完全一致（filename - .enc，basename）
- [ ] dry-run 默认关闭 — 显式 flag 才生效
- [ ] target KEK 启动期校验 → fail-fast 防误删 / 误覆写
- [ ] PutObject 失败保持原文件 — 操作可重试
- [ ] KEK bytes 不进入任何 log（既有规范延续）
- [ ] envelope 解析失败的文件不动 — 可能是手工放置的 / 损坏的 / future 格式
- [ ] concurrency clamp（防 1000-worker 误配压垮 MinIO）
- [ ] ctx cancellation 优雅 — 已处理保留 + 已传输的不能撤销但 PutObject 单调原子（同 key overwrite）

---

## 10. 实施要点

预计工作量：M（约 1.5 人日）— Reencryptor 服务 + cmd entry + 8 case 测试

预计涉及模块：`omcgo/internal/backup/` 2 新文件 + 1 修改 + `omcgo/cmd/backup-reencrypt/` 1 新文件

预计新增端点：无；预计新迁移：无；FE 影响：无；DI 改动：无（独立二进制）

---

## 11. 审批

| 角色 | 占位 | 日期 | 备注 |
|------|------|------|------|
| 产品 / 架构 / 安全 / 运维 / QA | Claude | 2026-04-30 | 独立二进制路线；不引入新 dep；同 KeyProvider + Encryptor 复用 — 0 新 crypto 路径 |

---

## 12. 变更记录

| 日期 | 版本 | 摘要 | 作者 |
|------|------|------|------|
| 2026-04-30 | v1.0 | 初稿；ULTRATHINK 9 决策；独立二进制 + 服务层；测试矩阵 V1-V10 | Claude |
