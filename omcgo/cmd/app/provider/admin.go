package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/omcgo/omcgo/internal/admin"
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

	adminService := admin.NewAdminService(userRepo, roleRepo, menuRepo, auditRepo, jwtService, logger)
	adminHandler := admin.NewHandler(adminService, logger)

	captchaService := admin.NewCaptchaService(c.Redis)
	loginGuard := admin.NewLoginGuard(c.Redis)
	adminHandler.SetCaptchaService(captchaService)
	adminHandler.SetLoginGuard(loginGuard)
	adminHandler.SetRoleDeviceGroupRepo(roleRepo)
	adminHandler.SetApiPermRepo(roleRepo)

	permService := admin.NewPermissionService(roleRepo, c.GroupRepo, c.Redis, logger)
	adminHandler.SetPermissionService(permService)

	apiKeyRepo := admin.NewPgAPIKeyRepository(c.PgPool)
	apiKeySvc := admin.NewAPIKeyService(apiKeyRepo, userRepo, logger)
	apiKeyHandler := admin.NewAPIKeyHandler(apiKeySvc)

	// Casbin RBAC engine — in-memory permission evaluation
	authorizer, err := admin.NewCasbinAuthorizer(c.PgPool, c.Redis, "configs/casbin_model.conf", logger)
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

	// Dictionary module
	dictRepo := admin.NewPgDictionaryRepository(c.PgPool)
	dictDetailRepo := admin.NewPgDictionaryDetailRepository(c.PgPool)
	dictService := admin.NewDictionaryService(dictRepo, dictDetailRepo)
	dictHandler := admin.NewDictionaryHandler(dictService)

	// System config module
	sysConfigRepo := admin.NewPgSysConfigRepository(c.PgPool)
	sysConfigService := admin.NewSysConfigService(sysConfigRepo)
	sysConfigHandler := admin.NewSysConfigHandler(sysConfigService)

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
		logHandler:       logHandler,
		jwtService:       jwtService,
		apiKeySvc:        apiKeySvc,
		userRepo:         userRepo,
		roleRepo:         roleRepo,
		auditRepo:        auditRepo,
	}

	logger.Info("admin/RBAC module initialized")
	return nil
}

type adminHandlerDeps struct {
	adminHandler     *admin.Handler
	apiKeyHandler    *admin.APIKeyHandler
	dictHandler      *admin.DictionaryHandler
	sysConfigHandler *admin.SysConfigHandler
	logHandler       *admin.LogHandler
	jwtService       *admin.JWTService
	apiKeySvc        *admin.APIKeyService
	userRepo         *admin.PgUserRepository
	roleRepo         *admin.PgRoleRepository
	auditRepo        *admin.PgAuditRepository
}
