package admin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Internal API key constants for the auto-provisioned system account.
//
// 设计:
//   - system 用户由 seed/000204_system_internal_user.sql 创建,UUID 固定
//   - app 启动期幂等签发 API key,plaintext 写入容器内固定路径
//   - 容器内任何工具/脚本默认从此文件读 key (omcctl / scripts/*.sh)
//
// 详见 omcgo/CLAUDE.md §5.6 全局 API 密钥约定。
const (
	// SystemInternalUserID 是 system 内部账户的固定 UUID,与 seed/000204 一致。
	SystemInternalUserID = "00000000-0000-0000-0000-000000000001"

	// InternalAPIKeyName 标识该 API key 在 api_keys 表中的 name 字段,
	// EnsureInternalAPIKey 用此名称定位 / 吊销历史 key。
	InternalAPIKeyName = "omc-internal"

	// DefaultInternalAPIKeyPath 是容器内 plaintext key 文件的默认路径。
	// docker-compose 用 named volume 挂载到 app(rw) 与 worker(ro)。
	// K8s 部署改用 Secret + projected volume,路径保持不变。
	DefaultInternalAPIKeyPath = "/var/lib/omcgo/secrets/.api-key"

	internalAPIKeyFileMode = 0o640
	internalAPIKeyDirMode  = 0o750
)

// EnsureInternalAPIKey 在 app 启动期幂等签发 system 用户的 API key。
//
// 调用语义:
//  1. 文件存在且 Validate 通过 → 直接返回 nil
//  2. 文件缺失 / Validate 失败 → 吊销该用户名下所有 InternalAPIKeyName 旧 key
//     → Create 新 key → 原子写文件 (tmp + rename)
//
// 该函数永不返回 fatal:即使文件写失败,业务接口仍可通过 --api-key / env override
// 使用 key,app 必须能起来。failure 信息走 logger.Warn,运维通过 docker logs 排查。
func EnsureInternalAPIKey(
	ctx context.Context,
	svc *APIKeyService,
	repo *PgAPIKeyRepository,
	path string,
	logger *zap.Logger,
) error {
	if path == "" {
		path = DefaultInternalAPIKeyPath
	}
	log := logger.Named("internal-apikey")

	userID, err := uuid.Parse(SystemInternalUserID)
	if err != nil {
		return fmt.Errorf("parse system user uuid: %w", err)
	}

	if raw, err := os.ReadFile(path); err == nil {
		key := strings.TrimSpace(string(raw))
		if key != "" {
			if _, vErr := svc.Validate(ctx, key); vErr == nil {
				log.Debug("internal api key file already valid", zap.String("path", path))
				return nil
			}
			log.Info("internal api key file invalid, will reissue",
				zap.String("path", path))
		}
	}

	existing, err := repo.ListByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("list existing internal api keys: %w", err)
	}
	for _, k := range existing {
		if k.Name == InternalAPIKeyName && k.RevokedAt == nil {
			if rErr := svc.Revoke(ctx, k.ID); rErr != nil {
				log.Warn("revoke stale internal api key failed",
					zap.String("key_id", k.ID.String()), zap.Error(rErr))
			}
		}
	}

	resp, err := svc.Create(ctx, userID, CreateAPIKeyRequest{
		Name:   InternalAPIKeyName,
		Scopes: []string{"*"},
	})
	if err != nil {
		return fmt.Errorf("create internal api key: %w", err)
	}

	if err := writeAPIKeyAtomic(path, resp.Key); err != nil {
		return fmt.Errorf("write internal api key file: %w", err)
	}

	log.Info("internal api key provisioned",
		zap.String("key_prefix", resp.KeyPrefix),
		zap.String("path", path),
	)
	return nil
}

// writeAPIKeyAtomic 写文件用 tmp + rename 保证原子性。
// 同时确保父目录存在 (0750) 与文件权限 (0640)。
func writeAPIKeyAtomic(path, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, internalAPIKeyDirMode); err != nil {
		return fmt.Errorf("ensure dir %s: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, ".api-key.*.tmp")
	if err != nil {
		return fmt.Errorf("create tmp file in %s: %w", dir, err)
	}
	tmpPath := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpPath) }

	if _, err := tmp.WriteString(content); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("write tmp: %w", err)
	}
	if err := tmp.Chmod(internalAPIKeyFileMode); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("chmod tmp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("sync tmp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close tmp: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		cleanup()
		return fmt.Errorf("rename %s → %s: %w", tmpPath, path, err)
	}
	return nil
}

// LoadInternalAPIKey 从 DefaultInternalAPIKeyPath (或自定义路径) 读取并 trim
// 返回 plaintext key。供 omcctl / scripts 在容器内默认入口使用。
//
// 返回空串 + nil error 表示文件不存在 — 调用方应继续 fallback (env / --flag)。
// 返回非空 string + nil error 表示成功。
func LoadInternalAPIKey(path string) (string, error) {
	if path == "" {
		path = DefaultInternalAPIKeyPath
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("read api key file %s: %w", path, err)
	}
	return strings.TrimSpace(string(raw)), nil
}
