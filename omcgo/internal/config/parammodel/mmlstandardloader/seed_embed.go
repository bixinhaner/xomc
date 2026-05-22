package mmlstandardloader

// seed_embed.go — go:embed 嵌入 seed JSON 并提供启动时自动导入入口。
//
// ImportEmbeddedSeed 适合在应用 bootstrap 阶段调用：
//   - 自动 hash 比对跳过已导入的版本（减少重复启动开销）
//   - soft failure：导入失败只记录 warn 不阻止服务启动

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

//go:embed seeds/cmcc_tdlte_v23.json
var embeddedSeedJSON []byte

// ImportEmbeddedSeed 从 go:embed 嵌入的 seed 数据导入 MML 命令树。
//
// 行为：
//  1. 计算嵌入数据的 SHA-256 hash
//  2. 查询 mml_param_versions 中对应 version_code 是否已存在相同 hash
//  3. 若 hash 一致则跳过（日志 Info）
//  4. 否则写临时文件后调用 ImportSeedFile（UPSERT 幂等）
//
// 错误策略（soft failure）：
//   - standard_params 中缺失部分 path → 只产生 warning，不中断
//   - 数据库事务失败 → 记录错误，返回 error 但建议调用方不终止启动
func ImportEmbeddedSeed(ctx context.Context, pool *pgxpool.Pool, logger *zap.Logger) error {
	if logger == nil {
		logger = zap.NewNop()
	}
	logger = logger.Named("seed-bootstrap")

	// 1. 解析 seed 获取 version_code（不写库，仅读 metadata）
	root, err := ParseSeed(bytes.NewReader(embeddedSeedJSON))
	if err != nil {
		return fmt.Errorf("parse embedded seed: %w", err)
	}
	versionCode := root.VersionCode()

	// 2. 计算嵌入数据 hash
	sum := sha256.Sum256(embeddedSeedJSON)
	currentHash := hex.EncodeToString(sum[:])

	// 3. 查询已存储的 content_hash，如一致则跳过
	var existingHash *string
	err = pool.QueryRow(ctx,
		`SELECT content_hash FROM mml_param_versions WHERE version_code = $1`,
		versionCode,
	).Scan(&existingHash)
	if err == nil && existingHash != nil && *existingHash == currentHash {
		logger.Info("seed already imported, hash unchanged — skipping",
			zap.String("version_code", versionCode),
			zap.String("hash", currentHash[:12]+"..."))
		return nil
	}
	// err != nil 意味着 row 不存在或查询失败 → 继续导入

	// 4. 写入临时文件（ImportSeedFile 需要文件路径）
	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, "omcgo-seed-"+versionCode+".json")
	if err := os.WriteFile(tmpFile, embeddedSeedJSON, 0644); err != nil {
		return fmt.Errorf("write temp seed file: %w", err)
	}
	defer os.Remove(tmpFile) //nolint:errcheck

	// 5. 调用核心 importer
	logger.Info("importing MML seed data",
		zap.String("version_code", versionCode),
		zap.String("hash", currentHash[:12]+"..."),
		zap.Int("embedded_size_bytes", len(embeddedSeedJSON)))

	if err := ImportSeedFile(ctx, pool, tmpFile, logger); err != nil {
		return fmt.Errorf("import seed %s: %w", versionCode, err)
	}

	logger.Info("MML seed import completed successfully",
		zap.String("version_code", versionCode))
	return nil
}
