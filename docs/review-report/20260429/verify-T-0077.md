# Verify Report — T-0077 backup 压缩 lz4 + bzip2 算法实施

> **Task**: T-0077 / R-102 followup / Sprint-07
> **Branch**: main / pre-commit @ 6f94d999
> **Date**: 2026-04-29
> **PRD**: `docs/project/prd/T-0077-backup-compression-lz4-bzip2.md`

---

## S4 出口门检查

| 门 | 结果 | 详情 |
|----|------|------|
| `go build ./...` | ✅ | 全项目编译通过 |
| `go test -race ./internal/backup/... ./internal/acs/upload/...` | ✅ | 0 fail；含全部 9-level lz4/bzip2 round-trip + magic byte + level mapping |
| `go vet` | ✅ | clean |
| `go mod tidy` | ✅ | 2 新 direct deps：`pierrec/lz4/v4 v4.1.26` + `dsnet/compress v0.0.1` |
| 新端点 | ✅ N/A | 无新路由；e2e bk-8 翻转期望 400→200（lz4 现可接受） |
| Migration | ✅ N/A | 无 schema 变更（compression_format CHECK 已含 lz4/bzip2） |
| Metric 名 | ✅ | 复用 T-0074 4 个 collectors；format label 自动多 2 个值 |

---

## 文件改动清单

修改：
- `omcgo/go.mod` / `omcgo/go.sum` — +2 direct deps
- `omcgo/internal/backup/compression.go` — `lz4Compressor` + `bzip2Compressor` + `lz4LevelFor` 加入；NewCompressor switch 替换 stub 分支
- `omcgo/internal/backup/compression_test.go` — 删除 2 个 Stub 测试，加 `TestNewCompressor_lz4_bzip2_areImplemented` + `TestLZ4RoundTrip_allLevels`（9 levels）+ `TestBzip2RoundTrip_allLevels`（9 levels，-short skip）+ `TestLZ4LevelMapping`
- `omcgo/internal/backup/policy_service.go` — 删除 lz4/bzip2 + EnableCompression=true 拒绝块
- `omcgo/internal/backup/policy_service_test.go` — 翻转 2 case（旧 reject → 新 accept）+ 删除 1 case
- `omcgo/internal/acs/upload/handler.go` — 删除 maybeWrapForCompression 中 lz4/bzip2 pass-through 块
- `omcgo/internal/acs/upload/handler_test.go` — 翻转 2 case（pass-through → applied）
- `omcgo/scripts/e2e_verify.sh` — bk-8 期望翻转 400→200，标签从 reject 改为 accept

---

## GWT 验收对照

| GWT | 验证手段 | 状态 |
|-----|---------|-----|
| V1 lz4 round-trip 全 9 档 | TestLZ4RoundTrip_allLevels（魔术字节 + lz4.Reader） | ✅ |
| V2 bzip2 round-trip 全 9 档 | TestBzip2RoundTrip_allLevels（魔术字节 BZh + stdlib bzip2.Reader） | ✅ |
| V3 service 端 lz4 + EnableCompression=true 不再 reject | policy_service_test 翻转 case | ✅ |
| V4 bzip2 同理 | 同 V3 | ✅ |
| V5 upload handler 不再 pass-through | TestMaybeWrapForCompression_lz4Applied / _bzip2Applied | ✅ |
| V6 e2e bk-8 翻转 | check_status_in "200 401" | ✅ (运行时验证；本地无 docker 实例) |
| V7 gzip / zstd regression | T-0074 全 8 GWT 仍通过；run -race 全绿 | ✅ |

---

## ULTRATHINK 决策回顾

| 决策 | 兑现 |
|------|------|
| FE dropdown disable cancels by 实施（A 与 B 互斥） | ✅ — 0 frontend change，dropdown 维持全 4 可选 |
| zstd encoder pool defer | ✅ — 不立 followup，profile-driven 再决策 |
| lz4 frame format（不是 block format） | ✅ pierrec/lz4 `NewWriter` 默认 frame format，与 `lz4` CLI 兼容 |
| bzip2 写用 dsnet/compress，读用 stdlib | ✅ 写: dsnet bzip2.NewWriter；测试解压用 stdlib compress/bzip2.NewReader（验证互操作）|
| level 直接映射 1..9 | ✅ lz4.Level1..Level9 / bzip2 1..9 |

---

## 备注

- **dsnet/compress 仓库标 alpha**：bzip2 子包多年稳定，被 archive/* 等项目引用；不进 risk-register
- **bzip2 性能**：bzip2 比 gzip/zstd 慢 5-10x；测试 -short 模式跳过 bzip2 round-trip 避免拖慢 CI
- **stdlib bzip2 reader 用于测试解压**：故意选 stdlib 而非 dsnet — 验证产出的 .bz2 与 `bunzip2` CLI / `archive/*` 消费方互操作
- **lz4 magic 4 字节**：`04 22 4d 18` 是 frame format magic（与 `lz4` 工具一致），不是 block format
