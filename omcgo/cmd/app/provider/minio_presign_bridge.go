package provider

import (
	"context"
	"errors"

	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/admin"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	minioinfra "github.com/omcgo/omcgo/internal/core/components/minio"
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
//  3. 注册 SysConfigSavedHook on category="storage"：保存后重读 → bridge.SetPublicEndpoint
//  4. 注册 SysConfigValidator on (storage, minio_public_endpoint) = ValidatePublicEndpoint，
//     非法值（带 scheme / path / 越界 port）BatchUpsert 整批拒绝并 HTTP 400
//
// 配置优先级（运行时合成）：
//   sys_configs.storage.minio_public_endpoint  (UI 改 → SavedHook 热生效)
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
	storageCategory               = "storage"
	storagePublicEndpointKey      = "minio_public_endpoint"
)

// initMinIOPresignBridgeModule 在 ModuleGraph 中作为 "minio-presign-bridge" 模块初始化，
// 依赖 admin（拿 SysConfigSvc 挂 hook / validator）。
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
		applyFromSysConfigs(context.Background(), c, bridge, logger)

		// 注册 SavedHook：category=storage 保存后重读并应用。
		c.SysConfigSvc.RegisterSavedHook(func(ctx context.Context, category string) {
			if category != storageCategory {
				return
			}
			applyFromSysConfigs(ctx, c, bridge, logger)
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
// 三种结果（issue #548 切片 2 · 回合 2 修复 #Issue2，区分 ErrNotFound 与真 DB 错误）：
//  1. ErrNotFound（首次部署 / 用户没填）：Debug 日志 + 等价 Set("") 回退 envFallback。
//  2. 真实 DB 错误（pg 不可达、SQL 语法错等）：Warn 日志（运维需察觉）+ 等价 Set("") 回退；
//     不阻塞启动——env/YAML fallback 仍能撑住绝大多数生产场景。
//  3. 读到值但 SetPublicEndpoint 校验失败（历史脏数据：validator 是本切片新加，老库可能有非法值）：
//     Warn 日志 + 保留桥当前 endpoint，运维去 UI 修正即可。
func applyFromSysConfigs(ctx context.Context, c *Container, bridge *minioinfra.PresignBridge, logger *zap.Logger) {
	repo := admin.NewPgSysConfigRepository(c.PgPool)
	row, err := repo.GetByKey(ctx, storageCategory, storagePublicEndpointKey)
	if err != nil {
		switch {
		case errors.Is(err, commonerrors.ErrNotFound):
			// 正常路径：未配置 → 桥走 envFallback。Debug 仅供排查。
			logger.Debug("sys_configs storage.minio_public_endpoint 未配置；桥使用 env/YAML fallback")
		default:
			// 真错误（DB 不可达、scan 失败等）：运维必须察觉，否则会误判"配置已生效"。
			logger.Warn("sys_configs storage.minio_public_endpoint 读取失败；桥回退 env/YAML，请检查数据库",
				zap.Error(err))
		}
		_ = bridge.SetPublicEndpoint("") // Set("") 自身不会失败（"" 合法）
		return
	}
	if applyErr := bridge.SetPublicEndpoint(row.Value); applyErr != nil {
		// 历史已落库的非法值（validator 是本切片新加，老库可能有脏数据）：
		// Warn 不阻塞——保持桥当前 endpoint，运维去 UI 修正即可。
		logger.Warn("sys_configs storage.minio_public_endpoint 当前值不合法，桥保留原值",
			zap.String("value", row.Value), zap.Error(applyErr))
	}
}
