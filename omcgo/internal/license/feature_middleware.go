// Package license — P7-B:
//
// RequireFeature 中间件在 HTTP 层按 license 功能授权对 API 端点做准入控制。
// 前端菜单隐藏（P7-A）只影响 UX；直接发 HTTP 请求仍可能绕过授权。P7-B 在后端
// 补齐拦截：未经 license feature 授权的 API 请求返回 403，biz_code 12114。
//
// 用法（router.go）：
//
//	featGroup := func(permRes string, path string) *gin.RouterGroup {
//	    g := v1.Group("")
//	    g.Use(admin.RequireAPIPermission(...))
//	    g.Use(license.RequireFeature(systemLicenseSvc, path))
//	    return g
//	}
//	deviceHandler.RegisterRoutes(featGroup("devices", "eNB.Monitor"))
package license

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

// RequireFeature returns a Gin middleware that checks whether the current system
// license authorizes at least one of the given dot-separated feature paths
// (OR semantics). If none is authorized, the request is aborted with HTTP 403
// and a human-readable message (biz_code 12114, "license feature not authorized").
//
// Design:
//   - Fail-closed: no license / transient check error → 403. This matches the
//     device-level enforcement (fail-closed) and differs from the menu-level
//     filter (fail-open). P7-B is a security boundary.
//   - OR semantics: passing multiple paths authorizes if any matches, consistent
//     with the menu's OR semantics (menus.feature_code is an array).
//   - Zero paths: noop (always passes). Intended to simplify wiring when a route
//     group has mixed protected/unprotected endpoints; attach only where needed.
func RequireFeature(svc *SystemLicenseService, paths ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if len(paths) == 0 {
			c.Next()
			return
		}

		ctx := c.Request.Context()
		var lastErr error
		for _, path := range paths {
			authorized, err := svc.CheckFeature(ctx, path)
			if err != nil {
				lastErr = err
				continue // transient? try next path, then fail-closed below
			}
			if authorized {
				c.Next()
				return
			}
		}

		if lastErr != nil {
			commonerrors.AbortWithError(c, http.StatusForbidden,
				fmt.Errorf("license feature check failed: %w", lastErr))
			return
		}

		commonerrors.AbortWithError(c, http.StatusForbidden,
			fmt.Errorf("license feature not authorized: none of %v granted: %w", paths, commonerrors.ErrLicenseFeatureNotAuthorized))
	}
}
