package provider

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/admin"
	minioinfra "github.com/omcgo/omcgo/internal/core/components/minio"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

// issue #548 切片 2 · D 后端 ── MinIO public_endpoint 运行时订阅桥 wiring。
//
// 问题（issue #548 背景）：MinIO 预签名 URL 之前由 *minio.Client 在启动期一次性构造，
// endpoint 固化在 SDK 内部；改 sys_configs / .env 后必须重启 app 才生效。运维改 IP
// 体验差，且 .env 漏配 MINIO_PUBLIC_ENDPOINT 时 URL 会落到内部 docker host
// （`minio:9000`），浏览器 ERR_NAME_NOT_RESOLVED。
//
// 本模块装配（仿 minio_ilm.go 模板）：
//  1. 启动期建桥 NewPresignBridge：初值 = cfg.PublicEndpoint（YAML 已 expand env），
//     空则回退内部 cfg.Endpoint + 一次 Warn（兜底语义保留，qa-614 #377）
//  2. 启动期一次 reload：若 sys_configs.storage.minio_public_endpoint 已配，覆盖 envFallback
//  3. 注册持久化 ConfigApplyHandler：保存后重读 → bridge.SetPublicEndpoint，并记录应用结果
//  4. 注册 SysConfigValidator on (storage, minio_public_endpoint) = ValidatePublicEndpoint，
//     非法值（带 scheme / path / 越界 port）BatchUpsert 整批拒绝并 HTTP 400
//
// 配置优先级（运行时合成）：
//   sys_configs.storage.minio_public_endpoint  (UI 改 → 应用任务热生效)
//     ↓ 空 则回退
//   cfg.PublicEndpoint                         (启动期 envFallback，YAML 里 ${MINIO_PUBLIC_ENDPOINT} 已展开)
//     ↓ 空 则回退
//   cfg.Endpoint 内部 host (minio:9000)         (硬兜底 + 首次 Warn)
//
// 当前消费方（issue #548 切片 2 范围）：
//   - trace.Handler.SetPresignProvider — TR069 报文导出下载链接（PoC，验证端到端通路）
//
// 留下切片清单（issue #548 后续切片）：
//   - mml 任务 CSV 导出预签名 URL
//   - license 文件下载预签名 URL
//   - 配置备份下载预签名 URL
//   - 其它 NewPresignClient 调用点（dashboard 报表、ufte、pm 等）
//
// 这些消费方目前仍走旧 NewPresignClient（启动期固化 endpoint），sys_configs UI 热改
// 暂时只对 trace 生效——切片 3 / 后续主线统一切换。

const (
	storageCategory          = "storage"
	storagePublicEndpointKey = "minio_public_endpoint"
)

// initMinIOPresignBridgeModule 在 ModuleGraph 中作为 "minio-presign-bridge" 模块初始化，
// 依赖 admin（拿 SysConfigSvc 挂应用器 / validator）。
func initMinIOPresignBridgeModule(c *Container) error {
	logger := c.Logger.Named("minio-presign-bridge")

	bridge, err := minioinfra.NewPresignBridge(c.Cfg.MinIO, logger)
	if err != nil {
		// 构造失败（access_key 非法 / endpoint 完全无法解析）：fatal——预签名 URL
		// 是 license / backup / 报文导出的必要基础设施，宁可启动失败也不要静默降级。
		return err
	}
	c.PresignBridge = bridge

	// 启动期一次 reload：若 sys_configs 已配 storage.minio_public_endpoint，
	// 用它覆盖 envFallback（典型场景：运维先经 UI 改过、后又重启进程）。
	if c.SysConfigSvc != nil {
		if _, err := applyFromSysConfigs(context.Background(), c, bridge); err != nil {
			logger.Warn("initial MinIO presign bridge apply failed", zap.Error(err))
		}

		// 保存后的 endpoint 更新必须进入可重试的应用状态；不能使用只记录日志的
		// SavedHook，否则数据库写入成功会被错误地报告为已生效。
		c.SysConfigSvc.RegisterApplyHandler(storageCategory, "minio_presign_endpoint", func(ctx context.Context, _ admin.ConfigApplyWork) (map[string]any, error) {
			endpoint, err := applyFromSysConfigs(ctx, c, bridge)
			if err != nil {
				return nil, err
			}
			return map[string]any{"public_endpoint": endpoint}, nil
		})

		// 注册 Validator：BatchUpsert 写入非法值时整批 HTTP 400 拒绝。
		c.SysConfigSvc.RegisterValidator(storageCategory, storagePublicEndpointKey,
			minioinfra.ValidatePublicEndpoint)

		logger.Info("MinIO presign bridge wired",
			zap.String("initial_endpoint", bridge.CurrentEndpoint()))
	} else {
		logger.Warn("SysConfigSvc not wired; MinIO presign bridge won't auto-reload on sys_configs save",
			zap.String("initial_endpoint", bridge.CurrentEndpoint()))
	}
	return nil
}

// applyFromSysConfigs 从 sys_configs 读 storage.minio_public_endpoint 并应用到桥。
//
// Missing configuration is an intentional env/YAML fallback. Database and
// endpoint validation failures are returned so the apply batch can be marked
// failed and retried instead of silently claiming success.
func applyFromSysConfigs(ctx context.Context, c *Container, bridge *minioinfra.PresignBridge) (string, error) {
	repo := admin.NewPgSysConfigRepository(c.PgPool)
	row, err := repo.GetByKey(ctx, storageCategory, storagePublicEndpointKey)
	if err != nil {
		if errors.Is(err, commonerrors.ErrNotFound) {
			if err := bridge.SetPublicEndpoint(""); err != nil {
				return "", fmt.Errorf("apply MinIO endpoint fallback: %w", err)
			}
			return bridge.CurrentEndpoint(), nil
		}
		return "", fmt.Errorf("read storage.minio_public_endpoint: %w", err)
	}
	if applyErr := bridge.SetPublicEndpoint(row.Value); applyErr != nil {
		return "", fmt.Errorf("apply storage.minio_public_endpoint: %w", applyErr)
	}
	return bridge.CurrentEndpoint(), nil
}
