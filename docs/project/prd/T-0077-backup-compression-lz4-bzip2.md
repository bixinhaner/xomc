# PRD: backup 压缩 lz4 + bzip2 算法实施（T-0077 / R-102 followup）

> **关联**: Backlog T-0077 / Sprint-07..08 / Domain=F06/backup / Type=feat
> **作者**: Claude（代 Owner=Go）
> **创建**: 2026-04-29
> **状态**: 草案 → 实施
> **关键决策**: scope 重定为 lz4 + bzip2 实施（FE dropdown disable cancels by 实施；zstd encoder pool defer）

---

## 1. 业务背景

T-0074 交付了 backup 压缩 MVP，gzip/zstd 完整实施，lz4/bzip2 stub 返回 `ErrCompressionFormatNotImplemented`。policy_service.validatePolicy 在 EnableCompression=true + format ∈ {lz4, bzip2} 时拒绝 PUT，避免持久化到无法生效的配置。

T-0074 review-agent 提出 H2：UI 仍展示 lz4/bzip2 但后端拒绝，体验割裂。当时方案：
- 选项 A：后端实施 lz4/bzip2 → 0 frontend change
- 选项 B：前端 dropdown disable lz4/bzip2 → 0 backend change

T-0077 originally listed **both** A + B + zstd encoder pool。**ULTRATHINK 重审**：A 完成后 B 自动 N/A（dropdown 全部可选），二者互斥；C（zstd encoder pool）属性能优化，在 backup 频率（每设备每日 1 次）下分配压力可忽略，profile-driven 再做。

---

## 2. ULTRATHINK 决策

### 2.1 子项重审

| # | 原描述 | 决策 | 理由 |
|---|--------|------|------|
| A | lz4 + bzip2 算法实施 | **本任务执行** | T-0074 stub 真正闭环；lift validation 拒绝；用户 PUT lz4/bzip2 + EnableCompression=true 真生效 |
| B | FE BackupPolicy dropdown disable lz4/bzip2 when EnableCompression=true | **N/A**（被 A cancel） | A 完成后 4 算法都支持，无须 disable 任何选项；dropdown 维持全 4 可选 |
| C | zstd encoder pool 优化 | **defer**（不立 followup） | backup 频率 = 每设备每日 ~1 次；10k devices = 10k encoders/day，分配压力 < 1 MB/s。profile-driven，热点出现再做 |

### 2.2 lib 选择

| 算法 | 候选 | 决策 | 理由 |
|------|------|------|------|
| lz4  | `github.com/pierrec/lz4/v4` | **采纳** | pure Go，v4 API 稳定，maintained，level 1..9 命名直接 |
| bzip2 写 | `github.com/dsnet/compress/bzip2` | **采纳** | 唯一活跃 pure Go bzip2 writer；包头 "alpha" 但 bzip2 子包稳定多年，被 archive/* 等大量项目引用 |
| bzip2 读 | stdlib `compress/bzip2` | （不需，restore 由 T-0072 用） | — |

新增 direct deps：2 个。

### 2.3 Level 映射

| OMC level | lz4 (`lz4.Level`) | bzip2 (`bzip2.WriterConfig.Level`) |
|-----------|-------------------|-----------------------------------|
| 1..9 直传 | `Level1`..`Level9` | 1..9 |

均为算法原生 9 档；零损映射。

### 2.4 后向兼容

数据库已存的 `compression_format ∈ {lz4, bzip2}` policy 行：
- T-0074 之前：写入但 executor 不读 → 用户感知 = "存了不生效"
- T-0074 之后：拒绝 PUT 含 lz4/bzip2+enable=true 的组合
- T-0077 之后：解禁 PUT；旧的"存了不生效"配置自动激活生效

如有运维担心"旧 policy 突然激活"，运维可在升级前 GET → 检查 → 必要时 disable。本任务不做迁移脚本，理由：T-0074 上线很短，生产几乎无 lz4/bzip2 用户。

---

## 3. 用户故事

| 角色 | 故事 |
|------|------|
| 网管运维 | 我希望 lz4 选项真的能用 — 选 lz4 + level 6 后 backup 文件以 `.lz4` 结尾，下载后 `lz4 -d` 能还原 |
| Go 开发者 | T-0074 的 Compressor 接口契约不变；本任务只是把 stub 替换为真实实现 |
| QA | e2e_verify.sh 的 W2D bk-8 需翻转：lz4 + EnableCompression=true 应 200 而非 400 |

---

## 4. 验收标准（GWT）

### V1 — lz4 round-trip 全 9 档
- **Given** OMC level ∈ {1..9}, format=lz4
- **When** 调 `NewCompressor("lz4", level)` 然后 Wrap 已知 plaintext
- **Then** 输出可被 `lz4.NewReader` 解压回原 plaintext；Format()="lz4"；Extension()=".lz4"

### V2 — bzip2 round-trip 全 9 档
- **Given** OMC level ∈ {1..9}, format=bzip2
- **When** 调 `NewCompressor("bzip2", level)` 然后 Wrap 已知 plaintext
- **Then** 输出可被 stdlib `compress/bzip2.NewReader` 解压；Format()="bzip2"；Extension()=".bz2"；魔术字节 `42 5a 68`（"BZh"）

### V3 — Service 端 lz4 + EnableCompression=true 不再 reject
- **Given** policy {EnableCompression=true, format=lz4, level=6, ...}
- **When** PUT /api/v1/backup/policy
- **Then** 200 OK；DB upsert 成功；GET 回响一致

### V4 — bzip2 同理
- 同 V3 但 format=bzip2；200 OK

### V5 — Upload handler 不再 pass-through lz4/bzip2
- **Given** policy.EnableCompression=true + format=lz4
- **When** CPE 上传 fileType=3 文件
- **Then** `maybeWrapForCompression` 返回 applied=true（不再 pass-through）；MinIO 对象路径以 `.lz4` 结尾

### V6 — e2e bk-8 翻转
- **Given** PUT body 含 lz4 + EnableCompression=true
- **Then** HTTP_CODE ∈ {200, 401}（之前是 400/401）

### V7 — gzip / zstd 行为不变（regression guard）
- T-0074 全 8 GWT 仍通过

---

## 5. 运营商差异矩阵

无差异（与 T-0074 一致；压缩算法选择是 OMC 内部存储优化，三家运营商无规范要求）。

---

## 6. 非目标

| # | 非目标 | 后续承接 |
|---|--------|---------|
| N1 | zstd encoder pool 优化 | 不立 followup；profile-driven |
| N2 | bzip2 性能基准（bzip2 慢，可能引发用户疑虑） | 文档 README 说明；不做 perf 调优 |
| N3 | lz4 frame format 选择（默认 frame format） | 用 pierrec/lz4 默认（frame format，与 lz4 命令行兼容） |
| N4 | restore 侧解压 .lz4 / .bz2 | T-0072 restore 实施时按本任务的扩展名契约消费 |
| N5 | DB schema 变更 | 无 — `compression_format` CHECK 已含 lz4/bzip2 |

---

## 7. 依赖

| 依赖 | 状态 |
|------|------|
| T-0074 ✅ | Compressor 接口 + factory 已就绪 |
| `pierrec/lz4/v4` | 新 direct dep |
| `dsnet/compress/bzip2` | 新 direct dep |

---

## 8. 度量

复用 T-0074 已建的 4 个 collectors：`omc_backup_compression_*{format=lz4|bzip2}` 自动多 2 个 label 取值，无需新增 metric。

---

## 9. 设计备忘（S2）

### 9.1 接口契约（不变）

`Compressor` 接口与工厂签名维持 T-0074 形态。本任务在 `NewCompressor` switch 中替换 lz4/bzip2 分支：

```go
case "lz4":
    return &lz4Compressor{level: lz4LevelFor(level)}, nil
case "bzip2":
    return &bzip2Compressor{level: level}, nil  // bzip2 直接接受 1..9
```

删除 `ErrCompressionFormatNotImplemented` 的发出路径（保留 sentinel，仍可表达"未来某算法不支持"）。

### 9.2 lz4 实现要点

```go
type lz4Compressor struct {
    level lz4.CompressionLevel
}

func (c *lz4Compressor) Format() string    { return "lz4" }
func (c *lz4Compressor) Extension() string { return ".lz4" }

func (c *lz4Compressor) Wrap(ctx context.Context, src io.Reader) (io.ReadCloser, error) {
    pr, pw := io.Pipe()
    lzw := lz4.NewWriter(pw)
    if err := lzw.Apply(lz4.CompressionLevelOption(c.level)); err != nil {
        _ = pw.Close()
        return nil, fmt.Errorf("lz4 writer level=%v: %w", c.level, err)
    }
    go pumpAndClose(ctx, src, lzw, pw)
    return pr, nil
}
```

### 9.3 bzip2 实现要点

```go
type bzip2Compressor struct {
    level int  // 1..9
}

func (c *bzip2Compressor) Format() string    { return "bzip2" }
func (c *bzip2Compressor) Extension() string { return ".bz2" }

func (c *bzip2Compressor) Wrap(ctx context.Context, src io.Reader) (io.ReadCloser, error) {
    pr, pw := io.Pipe()
    bzw, err := bzip2.NewWriter(pw, &bzip2.WriterConfig{Level: c.level})
    if err != nil {
        _ = pw.Close()
        return nil, fmt.Errorf("bzip2 writer level=%d: %w", c.level, err)
    }
    go pumpAndClose(ctx, src, bzw, pw)
    return pr, nil
}
```

### 9.4 移除 stub 测试

- 删除 `TestNewCompressor_lz4Stub`、`TestNewCompressor_bzip2Stub`
- 新增 `TestLZ4RoundTrip_allLevels`、`TestBzip2RoundTrip_allLevels`（按 gzip/zstd 同模式）
- bzip2 round-trip 用 stdlib `compress/bzip2.NewReader` 解压验证

### 9.5 service / handler 清理

- `policy_service.go` 删除 lz4/bzip2 reject 块
- `policy_service_test.go` 删除/翻转 3 个 case
- `handler.go` `maybeWrapForCompression` 删除 lz4/bzip2 pass-through 块
- `handler_test.go` 删除/翻转 `TestMaybeWrapForCompression_lz4PassThrough` / `_bzip2PassThrough`
- `e2e_verify.sh` 翻转 bk-8 期望 `200 401`

### 9.6 文件清单

修改：
- `omcgo/go.mod` / `omcgo/go.sum` — 2 新 direct deps
- `omcgo/internal/backup/compression.go` — 替换 stub + 2 实现
- `omcgo/internal/backup/compression_test.go` — 删 stub 测试 + 加 round-trip
- `omcgo/internal/backup/policy_service.go` — 删 reject 块
- `omcgo/internal/backup/policy_service_test.go` — 删/翻转 case
- `omcgo/internal/acs/upload/handler.go` — 删 pass-through 块
- `omcgo/internal/acs/upload/handler_test.go` — 删/翻转 case
- `omcgo/scripts/e2e_verify.sh` — 翻转 bk-8

### 9.7 待定点

| 待定 | 决策 |
|------|------|
| dsnet/compress 包头 alpha 标识是否阻塞 | 不阻塞。bzip2 子包是该仓库最稳定的部分，多年无 breaking change。risk-register 不新增条目 |
| lz4 frame format vs block format | 用 pierrec/lz4 默认 frame format（兼容 `lz4` CLI），block format 不暴露给用户 |
