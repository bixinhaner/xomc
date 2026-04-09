package provider

import (
	"fmt"

	"github.com/omcgo/omcgo/internal/admin"
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

	permService := admin.NewPermissionService(roleRepo, c.GroupRepo, c.Redis, logger)
	adminHandler.SetPermissionService(permService)

	apiKeyRepo := admin.NewPgAPIKeyRepository(c.PgPool)
	apiKeySvc := admin.NewAPIKeyService(apiKeyRepo, userRepo, logger)
	apiKeyHandler := admin.NewAPIKeyHandler(apiKeySvc)

	// Set shared services
	c.JWTService = jwtService
	c.APIKeySvc = apiKeySvc
	c.UserRepo = userRepo
	c.RoleRepo = roleRepo
	c.AuditRepo = auditRepo
	c.PermService = permService

	// Store handlers for route registration
	c.adminHandlerDeps = &adminHandlerDeps{
		adminHandler:  adminHandler,
		apiKeyHandler: apiKeyHandler,
		jwtService:    jwtService,
		apiKeySvc:     apiKeySvc,
		userRepo:      userRepo,
		roleRepo:      roleRepo,
		auditRepo:     auditRepo,
	}

	logger.Info("admin/RBAC module initialized")
	return nil
}

type adminHandlerDeps struct {
	adminHandler  *admin.Handler
	apiKeyHandler *admin.APIKeyHandler
	jwtService    *admin.JWTService
	apiKeySvc     *admin.APIKeyService
	userRepo      *admin.PgUserRepository
	roleRepo      *admin.PgRoleRepository
	auditRepo     *admin.PgAuditRepository
}
