package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/admin/loginpwd"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/topology"
	"go.uber.org/zap"
)

// initAdminModule 初始化 F06 用户管理与 RBAC 模块。
// 依赖: GroupRepo (from TopologyModule)
// 设置: JWTService, APIKeySvc, UserRepo, RoleRepo, AuditRepo, PermService
func initAdminModule(c *Container) error {
	logger := c.Logger.Named("admin")

	userRepo := admin.NewPgUserRepository(c.PgPool)
	roleRepo := admin.NewPgRoleRepository(c.PgPool)
	auditRepo := admin.NewPgAuditRepository(c.PgPool)
	menuRepo := admin.NewPgMenuRepository(c.PgPool)

	jwtService, err := admin.NewJWTServiceWithTTL(c.Cfg.JWT.Secret, c.Cfg.JWT.AccessTokenTTL, c.Cfg.JWT.RefreshTokenTTL)
	if err != nil {
		return fmt.Errorf("init JWT service: %w", err)
	}

	// TokenRevoker：基于 Redis 的用户级强制下线，TTL 与 refresh token 寿命对齐。
	tokenRevoker := admin.NewTokenRevoker(c.Redis, c.Cfg.JWT.RefreshTokenTTL)

	adminService := admin.NewAdminService(userRepo, roleRepo, menuRepo, auditRepo, jwtService, logger)
	adminService.SetTokenRevoker(tokenRevoker)
	// PRD §10 DoD：失效失败计数器 omc_perm_cache_invalidate_failed_total。
	adminService.SetMetrics(admin.NewAdminMetrics(c.MetricsReg))
	adminHandler := admin.NewHandler(adminService, logger)

	// 共享 SecurityPolicy 实例 — sys_configs (category='security') 全部字段
	// 30s 缓存；LoginGuard / IPGuard / AdminService(单点登录/密码策略) 共用一份。
	// 任一字段（FE SecuritySettings 保存）变化后最多 30s 生效。
	sysConfigRepoForPolicy := admin.NewPgSysConfigRepository(c.PgPool)
	securityPolicy := admin.NewSecurityPolicy(sysConfigRepoForPolicy)

	captchaService := admin.NewCaptchaService(c.Redis)
	loginGuard := admin.NewLoginGuard(c.Redis)
	loginGuard.SetPolicy(securityPolicy)

	// P0-③ IP 限流（Redis 滑动窗口 + 黑名单 TTL）— 替代旧 in-memory token-bucket。
	// 字段映射：limitMinus(窗口) / limitCount(阈值) / limitTimes(锁时长)。
	ipGuard := admin.NewIPGuard(c.Redis)
	ipGuard.SetPolicy(securityPolicy)

	adminHandler.SetCaptchaService(captchaService)
	adminHandler.SetLoginGuard(loginGuard)
	adminHandler.SetIPGuard(ipGuard)
	adminHandler.SetRoleDeviceGroupRepo(roleRepo)
	adminHandler.SetApiPermRepo(roleRepo)

	// P0-① 单点登录：AdminService.Login 读 policy.AllowConcurrent 决定是否撤旧 token。
	adminService.SetSecurityPolicy(securityPolicy)

	// 登录类接口的密码 RSA-OAEP 加密：keystore 启动时加载或生成私钥；
	// 失败为 fatal —— 没有密钥所有登录请求都会拒绝，不能静默降级到明文。
	keystore, err := loginpwd.NewKeystore(c.Cfg.LoginCrypto.PrivateKeyPath)
	if err != nil {
		return fmt.Errorf("init login password keystore: %w", err)
	}
	loginCipher := loginpwd.NewCipher(keystore, loginpwd.NewRedisReplayGuard(c.Redis))
	pubKeyHandler := loginpwd.NewPublicKeyHandler(loginCipher)
	adminHandler.SetLoginCipher(loginCipher)
	// T-0120：是否接受明文密码登录（非 secure context 部署 escape hatch）
	adminHandler.SetAllowPlaintextPassword(c.Cfg.LoginCrypto.AllowPlaintext)
	logger.Info("login password cipher initialized",
		zap.String("key_id", keystore.ActiveKeyID()),
		zap.String("private_key_path", c.Cfg.LoginCrypto.PrivateKeyPath),
		zap.Bool("allow_plaintext", c.Cfg.LoginCrypto.AllowPlaintext),
	)
	if c.Cfg.LoginCrypto.AllowPlaintext {
		logger.Warn("login_crypto.allow_plaintext=true — 明文密码登录已启用；仅推荐内网部署 + 完整 audit 闭环；公网部署应改 false 并部署 TLS（参 deployments/docker/TLS-SETUP.md）")
	}

	permService := admin.NewPermissionService(roleRepo, c.GroupRepo, c.Redis, logger)
	adminHandler.SetPermissionService(permService)
	// PRD §10 DoD：用户写操作（assign/remove role、UpdateUser、DeleteUser、批量分配）后必须同步失效用户权限缓存。
	adminService.SetPermissionInvalidator(permService)

	apiKeyRepo := admin.NewPgAPIKeyRepository(c.PgPool)
	apiKeySvc := admin.NewAPIKeyService(apiKeyRepo, userRepo, logger)
	apiKeyHandler := admin.NewAPIKeyHandler(apiKeySvc)

	// 启动期幂等签发 omc-internal API key,写 /var/lib/omcgo/secrets/.api-key,
	// 供容器内 omcctl / scripts 零配置使用 (详见 omcgo/CLAUDE.md §5.6)。
	// 失败不阻塞 app 启动 — 业务接口仍可通过 --api-key / env 覆盖。
	if err := admin.EnsureInternalAPIKey(context.Background(), apiKeySvc, apiKeyRepo,
		admin.DefaultInternalAPIKeyPath, logger); err != nil {
		logger.Warn("EnsureInternalAPIKey failed — omcctl 零配置入口不可用,但 app 启动继续",
			zap.Error(err))
	}

	// Casbin RBAC engine — in-memory permission evaluation.
	// 策略变更广播走 NATS JetStream（sys.casbin.policy.reload），替代原
	// Redis Pub/Sub，获得持久化 + 订阅者断线重连回放。
	//
	// 失败处理：B3-Phase2-B 起 permissions 表已 DROP，无 SQL fallback —— authorizer
	// 未注入则 PgRoleRepository.CheckPermission 直接返 error，所有走
	// RequireAPIPermission/RequirePermission 的端点对非超管用户全部 500。
	// 因此 Casbin 初始化失败必须直接终止进程，不能让进程"看似正常启动"后所有
	// 业务接口故障；常见根因是 configs/casbin_model.conf 没挂进容器（参
	// deployments/docker/Dockerfile.app）。
	authorizer, err := admin.NewCasbinAuthorizer(c.PgPool, c.EventBus, "configs/casbin_model.conf", logger)
	if err != nil {
		return fmt.Errorf("init casbin authorizer (model=configs/casbin_model.conf): %w", err)
	}
	roleRepo.SetAuthorizer(authorizer)
	c.GS.Register("casbin-watcher", 1, func(_ context.Context) error { authorizer.Stop(); return nil })
	authorizer.StartPeriodicRefresh(5 * time.Minute)
	logger.Info("casbin RBAC engine initialized")

	// P2-⑨ 长期未登录自动锁定（每天 03:30 跑）。policy.AutoLockUnusedEnabled=false
	// 时 cron 仍跑但 RunOnce 立即返回；FE 改配置最多等下一天 03:30 生效。
	inactiveLocker := admin.NewInactiveUserLocker(c.PgPool, securityPolicy, logger)
	inactiveLocker.Start()
	c.GS.Register("inactive-user-locker", 1, func(_ context.Context) error { inactiveLocker.Stop(); return nil })

	// Set shared services
	c.JWTService = jwtService
	c.APIKeySvc = apiKeySvc
	c.UserRepo = userRepo
	c.RoleRepo = roleRepo
	c.AuditRepo = auditRepo
	c.PermService = permService

	// PRD users.md §11.7 决议③ / roles.md §11.4：注入"删除设备分组联动"钩子。
	// admin 模块在 topology 之后初始化，此处把 roleRepo + adminService 适配到
	// topology 包内定义的接口（避免 topology → admin 反向依赖）。
	if c.GroupService != nil {
		c.GroupService.SetGroupDeleteHooks(
			&roleAffectedQueryAdapter{repo: roleRepo},
			adminService,
		)
	}

	// Dictionary module
	dictRepo := admin.NewPgDictionaryRepository(c.PgPool)
	dictDetailRepo := admin.NewPgDictionaryDetailRepository(c.PgPool)
	dictService := admin.NewDictionaryService(dictRepo, dictDetailRepo)
	// T-0182 数据字典数据源:启动期加载白名单 + 注入 SyncEngine。
	// 失败仅警告,字典模块继续按"无数据源"模式运行,避免阻塞 app 启动。
	// worker daily cron(P2 阶段加入)会通过 grpc/直接调本 service 的 SyncSourceBoundAll。
	if reg, err := admin.LoadDictSourceRegistry(); err != nil {
		logger.Warn("dict_source_registry_disabled", zap.Error(err))
	} else {
		engine := admin.NewDictSyncEngine(reg, c.PgPool, dictRepo, dictDetailRepo, logger.Named("dict_source"))
		dictService.SetSourceWiring(reg, engine, logger.Named("dict_source"))
	}
	dictHandler := admin.NewDictionaryHandler(dictService)

	// System config module
	sysConfigRepo := admin.NewPgSysConfigRepository(c.PgPool)
	sysConfigService := admin.NewSysConfigService(sysConfigRepo)
	sysConfigHandler := admin.NewSysConfigHandler(sysConfigService)

	// FE 保存"安全设置"页（category='security'）后立刻让 SecurityPolicy 30s 缓存失效，
	// 避免管理员看到设置变更但实际生效要等 ≤30s 的体验断层。
	sysConfigService.RegisterSavedHook(func(_ context.Context, category string) {
		if category == "security" {
			securityPolicy.InvalidateCache()
		}
	})
	if c.EventBus != nil {
		sysConfigService.RegisterSavedHook(func(ctx context.Context, category string) {
			evt, err := event.NewEvent(event.SubjectSysConfigSaved, event.SysConfigSavedPayload{Category: category})
			if err != nil {
				logger.Warn("build sys config saved event", zap.String("category", category), zap.Error(err))
				return
			}
			if err := c.EventBus.Publish(ctx, event.SubjectSysConfigSaved, evt); err != nil {
				logger.Warn("publish sys config saved event", zap.String("category", category), zap.Error(err))
			}
		})
	}

	// 暴露到 Container 让其他模块（如 provision.PeriodicSyncPolicy）也能挂 hook。
	c.SysConfigSvc = sysConfigService
	c.SecurityPolicy = securityPolicy

	// UI 定制化资产上传 / 公开下载（参 docs/prd/system/ui-customization.md）
	uiAssetHandler := admin.NewUIAssetHandler(c.MinIO, c.Cfg.MinIO.Buckets.UIAssets)

	// Log module
	logRepo := admin.NewPgLogRepository(c.PgPool)
	logHandler := admin.NewLogHandler(logRepo)

	// API Endpoint module
	apiEndpointRepo := admin.NewPgApiEndpointRepository(c.PgPool)
	apiEndpointService := admin.NewApiEndpointService(apiEndpointRepo, logger)
	adminHandler.SetApiEndpointService(apiEndpointService)

	// Store handlers for route registration
	c.adminHandlerDeps = &adminHandlerDeps{
		adminHandler:     adminHandler,
		apiKeyHandler:    apiKeyHandler,
		dictHandler:      dictHandler,
		sysConfigHandler: sysConfigHandler,
		uiAssetHandler:   uiAssetHandler,
		logHandler:       logHandler,
		pubKeyHandler:    pubKeyHandler,
		jwtService:       jwtService,
		tokenRevoker:     tokenRevoker,
		apiKeySvc:        apiKeySvc,
		userRepo:         userRepo,
		roleRepo:         roleRepo,
		auditRepo:        auditRepo,
	}

	logger.Info("admin/RBAC module initialized")
	return nil
}

// roleAffectedQueryAdapter 把 *admin.PgRoleRepository.ListRolesByGroupIDs 的
// []admin.Role 结果转成 topology.AffectedRole（仅 ID + Name 摘要）。
type roleAffectedQueryAdapter struct {
	repo *admin.PgRoleRepository
}

func (a *roleAffectedQueryAdapter) ListRolesByGroupIDs(ctx context.Context, groupIDs []uuid.UUID) ([]topology.AffectedRole, error) {
	roles, err := a.repo.ListRolesByGroupIDs(ctx, groupIDs)
	if err != nil {
		return nil, err
	}
	out := make([]topology.AffectedRole, 0, len(roles))
	for _, r := range roles {
		out = append(out, topology.AffectedRole{ID: r.ID, Name: r.Name})
	}
	return out, nil
}

type adminHandlerDeps struct {
	adminHandler     *admin.Handler
	apiKeyHandler    *admin.APIKeyHandler
	dictHandler      *admin.DictionaryHandler
	sysConfigHandler *admin.SysConfigHandler
	uiAssetHandler   *admin.UIAssetHandler
	logHandler       *admin.LogHandler
	pubKeyHandler    *loginpwd.PublicKeyHandler
	jwtService       *admin.JWTService
	tokenRevoker     *admin.TokenRevoker
	apiKeySvc        *admin.APIKeyService
	userRepo         *admin.PgUserRepository
	roleRepo         *admin.PgRoleRepository
	auditRepo        *admin.PgAuditRepository
}
