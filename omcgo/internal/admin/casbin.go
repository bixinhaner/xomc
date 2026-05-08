package admin

import (
	"context"
	"fmt"
	"time"

	"github.com/casbin/casbin/v2"
	casbinModel "github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/components/redisx"
	"github.com/omcgo/omcgo/internal/core/event"
)

// casbinPolicyChannel 保留一个文件内私有的别名，方便测试代码继续引用；
// 真正的字符串字面量统一归集到 redisx/keys.go。
var casbinPolicyChannel = redisx.Keys.CasbinPolicyChannel()

// --- Adapter ---

// pgAdapter implements persist.Adapter, reading from existing tables.
type pgAdapter struct {
	pool *pgxpool.Pool
}

var _ persist.Adapter = (*pgAdapter)(nil)

func newPgAdapter(pool *pgxpool.Pool) *pgAdapter {
	return &pgAdapter{pool: pool}
}

func (a *pgAdapter) LoadPolicy(m casbinModel.Model) error {
	ctx := context.Background()

	// 1. Load permission policies: p = (sub, dom, obj, act)
	// Domain is "system" because permissions are role-level (carrier-agnostic).
	// v1.0：users.carrier 已删除，g 策略也统一在 'system' 单域，多 carrier 隔离失效（参 §11.11 Q3）。
	rows, err := a.pool.Query(ctx, `
		SELECT r.name AS role_name, p.resource, p.action
		FROM permissions p
		JOIN roles r ON r.id = p.role_id
	`)
	if err != nil {
		return fmt.Errorf("load permission policies: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var roleName, resource, action string
		if err := rows.Scan(&roleName, &resource, &action); err != nil {
			return fmt.Errorf("scan permission policy: %w", err)
		}
		m.AddPolicy("p", "p", []string{"role:" + roleName, "system", resource, action})
	}
	if err := rows.Err(); err != nil {
		return err
	}

	// 1.5. B3-Phase1：双源 LoadPolicy — 同时加载端点级策略
	//
	// 当前 (resource, action) 模型仍由 Step 1 的 permissions 表喂；本步骤额外加载
	// role_api_permissions JOIN api_endpoints 输出的端点级 (path, method) 策略，
	// 喂入同一 model.conf（matcher 含 keyMatch 兼容）。Phase1 不改 router/中间件，
	// 端点级策略暂不被 Enforce 命中——仅为 Phase2 切换 RequireAPIPermission 中间件
	// 准备数据基础。详见 docs/prd/system/menu-dynamic-loading.md §4.2.4 (B3-Phase1)。
	//
	// admin/operator 角色 role_api_permissions 各有 445 行 = api_endpoints 全集；
	// viewer 由 seed/000064_seed_role_api_permissions_viewer.sql 兜底为 GET 类全集。
	rowsAPI, err := a.pool.Query(ctx, `
		SELECT r.name AS role_name, ae.path, ae.method
		FROM role_api_permissions rap
		JOIN api_endpoints ae ON ae.id = rap.endpoint_id
		JOIN roles r ON r.id = rap.role_id
	`)
	if err != nil {
		// role_api_permissions 表可能在旧环境未建（v1.0 前）→ 不阻断
		// （已 v1.0 必有此表，参 migrations/000056_roles_v1_extras.sql）
		return nil
	}
	defer rowsAPI.Close()

	for rowsAPI.Next() {
		var roleName, path, method string
		if err := rowsAPI.Scan(&roleName, &path, &method); err != nil {
			return fmt.Errorf("scan api permission policy: %w", err)
		}
		// 端点级策略：obj=path, act=method
		m.AddPolicy("p", "p", []string{"role:" + roleName, "system", path, method})
	}
	if err := rowsAPI.Err(); err != nil {
		return err
	}

	// 2. Load role assignments: g = (user_id, role:{name}, domain)
	// v1.0：users.carrier 已删除，所有 g 策略统一在 'system' 单域。多 carrier RBAC 隔离能力不再保留。
	// 详见 docs/prd/system/users.md §11.11 决议 Q3。
	rows2, err := a.pool.Query(ctx, `
		SELECT u.id::text AS user_id, r.name AS role_name
		FROM user_roles ur
		JOIN roles r ON r.id = ur.role_id
		JOIN users u ON u.id = ur.user_id
	`)
	if err != nil {
		return fmt.Errorf("load role assignments: %w", err)
	}
	defer rows2.Close()

	for rows2.Next() {
		var userID, roleName string
		if err := rows2.Scan(&userID, &roleName); err != nil {
			return fmt.Errorf("scan role assignment: %w", err)
		}
		m.AddPolicy("g", "g", []string{userID, "role:" + roleName, "system"})
	}
	if err := rows2.Err(); err != nil {
		return err
	}

	// 3. Load role inheritance: g = (child, parent, domain)
	rows3, err := a.pool.Query(ctx, `
		SELECT rc.name AS child, rp.name AS parent, ri.domain
		FROM role_inheritance ri
		JOIN roles rp ON rp.id = ri.parent_role_id
		JOIN roles rc ON rc.id = ri.child_role_id
	`)
	if err != nil {
		// Table may not exist yet (migration not applied)
		return nil
	}
	defer rows3.Close()

	for rows3.Next() {
		var child, parent, domain string
		if err := rows3.Scan(&child, &parent, &domain); err != nil {
			return fmt.Errorf("scan role inheritance: %w", err)
		}
		m.AddPolicy("g", "g", []string{"role:" + child, "role:" + parent, domain})
	}
	return rows3.Err()
}

func (a *pgAdapter) SavePolicy(m casbinModel.Model) error {
	// Not needed — we load from existing tables managed by AdminService.
	return nil
}

func (a *pgAdapter) AddPolicy(sec string, ptype string, rule []string) error {
	return nil
}

func (a *pgAdapter) RemovePolicy(sec string, ptype string, rule []string) error {
	return nil
}

func (a *pgAdapter) RemoveFilteredPolicy(sec string, ptype string, fieldIndex int, fieldValues ...string) error {
	return nil
}

// --- Watcher ---

// redisWatcher implements persist.Watcher using Redis Pub/Sub.
type redisWatcher struct {
	client   redis.UniversalClient
	callback func(string)
}

var _ persist.Watcher = (*redisWatcher)(nil)

func newRedisWatcher(client redis.UniversalClient) *redisWatcher {
	return &redisWatcher{client: client}
}

func (w *redisWatcher) SetUpdateCallback(callback func(string)) error {
	w.callback = callback
	return nil
}

func (w *redisWatcher) Update() error {
	return w.client.Publish(context.Background(), casbinPolicyChannel, "reload").Err()
}

func (w *redisWatcher) StartListener() {
	go func() {
		sub := w.client.Subscribe(context.Background(), casbinPolicyChannel)
		defer sub.Close()

		for msg := range sub.Channel() {
			if w.callback != nil {
				w.callback(msg.Payload)
			}
		}
	}()
}

func (w *redisWatcher) Close() {
	// Subscription is closed via deferred sub.Close() in StartListener
}

func (w *redisWatcher) Notify() error {
	return w.Update()
}

// --- Enforcer ---

// CasbinAuthorizer provides Casbin-based permission checking.
// Watcher 接口化允许在 NATS（生产）和 Redis（历史兼容 / 测试）之间切换。
type CasbinAuthorizer struct {
	enforcer *casbin.Enforcer
	adapter  *pgAdapter
	watcher  persist.Watcher
	logger   *zap.Logger
}

// NewCasbinAuthorizer 通过 NATS JetStream（项目统一事件总线）广播策略变更。
// bus 为 nil 时降级为 no-op watcher，策略同步退化为 StartPeriodicRefresh 的
// 周期性全量刷新（单实例部署 / 单测场景可接受）。
func NewCasbinAuthorizer(pool *pgxpool.Pool, bus event.EventBus, modelPath string, logger *zap.Logger) (*CasbinAuthorizer, error) {
	m, err := casbinModel.NewModelFromFile(modelPath)
	if err != nil {
		return nil, fmt.Errorf("load casbin model: %w", err)
	}

	adapter := newPgAdapter(pool)

	enforcer, err := casbin.NewEnforcer(m, adapter)
	if err != nil {
		return nil, fmt.Errorf("create casbin enforcer: %w", err)
	}

	watcher := newNATSCasbinWatcher(bus, logger)
	enforcer.SetWatcher(watcher)

	watcher.SetUpdateCallback(func(_ string) {
		if err := enforcer.LoadPolicy(); err != nil {
			logger.Error("casbin policy reload failed", zap.Error(err))
		} else {
			logger.Info("casbin policy reloaded")
		}
	})

	watcher.StartListener()

	return &CasbinAuthorizer{
		enforcer: enforcer,
		adapter:  adapter,
		watcher:  watcher,
		logger:   logger,
	}, nil
}

// CheckPermission checks if a subject has permission (implements PermissionChecker interface).
// v1.0：domain 固定为 "system"（去 carrier 多租户隔离，详见 §11.11 Q3）。
func (a *CasbinAuthorizer) CheckPermission(_ context.Context, userID uuid.UUID, resource, action string) (bool, error) {
	ok, err := a.enforcer.Enforce(userID.String(), "system", resource, action)
	if err != nil {
		return false, fmt.Errorf("casbin enforce: %w", err)
	}
	return ok, nil
}

// ReloadPolicy manually triggers policy reload.
func (a *CasbinAuthorizer) ReloadPolicy() error {
	return a.enforcer.LoadPolicy()
}

// NotifyPolicyChange notifies all instances to reload policies.
// Uses the standard persist.Watcher.Update() which maps to a broadcast publish.
func (a *CasbinAuthorizer) NotifyPolicyChange() error {
	return a.watcher.Update()
}

// Stop closes the watcher.
func (a *CasbinAuthorizer) Stop() {
	a.watcher.Close()
}

// StartPeriodicRefresh starts a background goroutine that reloads policies every interval.
func (a *CasbinAuthorizer) StartPeriodicRefresh(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			if err := a.enforcer.LoadPolicy(); err != nil {
				a.logger.Error("casbin periodic reload failed", zap.Error(err))
			}
		}
	}()
}
