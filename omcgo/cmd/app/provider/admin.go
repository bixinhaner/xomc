package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/admin"
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

	captchaService := admin.NewCaptchaService(c.Redis)
	loginGuard := admin.NewLoginGuard(c.Redis)
	adminHandler.SetCaptchaService(captchaService)
	adminHandler.SetLoginGuard(loginGuard)
	adminHandler.SetRoleDeviceGroupRepo(roleRepo)
	adminHandler.SetApiPermRepo(roleRepo)

	permService := admin.NewPermissionService(roleRepo, c.GroupRepo, c.Redis, logger)
	adminHandler.SetPermissionService(permService)
	// PRD §10 DoD：用户写操作（assign/remove role、UpdateUser、DeleteUser、批量分配）后必须同步失效用户权限缓存。
	adminService.SetPermissionInvalidator(permService)

	apiKeyRepo := admin.NewPgAPIKeyRepository(c.PgPool)
	apiKeySvc := admin.NewAPIKeyService(apiKeyRepo, userRepo, logger)
	apiKeyHandler := admin.NewAPIKeyHandler(apiKeySvc)

	// Casbin RBAC engine — in-memory permission evaluation.
	// 策略变更广播走 NATS JetStream（sys.casbin.policy.reload），替代原
	// Redis Pub/Sub，获得持久化 + 订阅者断线重连回放。
	authorizer, err := admin.NewCasbinAuthorizer(c.PgPool, c.EventBus, "configs/casbin_model.conf", logger)
	if err != nil {
		logger.Warn("casbin init failed, falling back to SQL permission checks", zap.Error(err))
	} else {
		roleRepo.SetAuthorizer(authorizer)
		c.GS.Register("casbin-watcher", 1, func(_ context.Context) error { authorizer.Stop(); return nil })
		authorizer.StartPeriodicRefresh(5 * time.Minute)
		logger.Info("casbin RBAC engine initialized")
	}

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
	dictHandler := admin.NewDictionaryHandler(dictService)

	// System config module
	sysConfigRepo := admin.NewPgSysConfigRepository(c.PgPool)
	sysConfigService := admin.NewSysConfigService(sysConfigRepo)
	sysConfigHandler := admin.NewSysConfigHandler(sysConfigService)

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
	jwtService       *admin.JWTService
	tokenRevoker     *admin.TokenRevoker
	apiKeySvc        *admin.APIKeyService
	userRepo         *admin.PgUserRepository
	roleRepo         *admin.PgRoleRepository
	auditRepo        *admin.PgAuditRepository
}
