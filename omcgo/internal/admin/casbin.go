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

	// 端点级策略（B3-Phase2-B 起为唯一来源）：(role, path, method)
	//
	// 历史 permissions 表（resource, action 抽象）已在 B3-Phase2-B 整体 DROP。
	// 当前所有受保护路由经 RequireAPIPermission 中间件触发 CheckPermission 鉴权，
	// Casbin 仅消费 role_api_permissions JOIN api_endpoints 端点级数据。
	//
	// admin/operator 角色 role_api_permissions 各 445 行 = api_endpoints 全集；
	// viewer 由 seed/000067_seed_role_api_permissions_viewer.sql 兜底 GET 全集。
	rowsAPI, err := a.pool.Query(ctx, `
		SELECT r.name AS role_name, ae.path, ae.method
		FROM role_api_permissions rap
		JOIN api_endpoints ae ON ae.id = rap.endpoint_id
		JOIN roles r ON r.id = rap.role_id
	`)
	if err != nil {
		return fmt.Errorf("load api permission policies: %w", err)
	}
	defer rowsAPI.Close()

	for rowsAPI.Next() {
		var roleName, path, method string
		if err := rowsAPI.Scan(&roleName, &path, &method); err != nil {
			return fmt.Errorf("scan api permission policy: %w", err)
		}
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

	// 3. Load role inheritance: parent_role_id inherits child_role_id.
	// The built-in hierarchy is admin -> operator -> viewer, so lower-privilege
	// roles must never inherit permissions from their parents.
	rows3, err := a.pool.Query(ctx, `
		SELECT rp.name AS parent, rc.name AS child, ri.domain
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
		var parent, child, domain string
		if err := rows3.Scan(&parent, &child, &domain); err != nil {
			return fmt.Errorf("scan role inheritance: %w", err)
		}
		addRoleInheritancePolicy(m, parent, child, domain)
	}
	return rows3.Err()
}

func addRoleInheritancePolicy(m casbinModel.Model, parent, child, domain string) {
	m.AddPolicy("g", "g", []string{"role:" + parent, "role:" + child, domain})
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
	enforcer *casbin.SyncedEnforcer
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

	enforcer, err := casbin.NewSyncedEnforcer(m, adapter)
	if err != nil {
		return nil, fmt.Errorf("create casbin synced enforcer: %w", err)
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
// B3-Phase2-B 起，obj/act 入参语义为端点级 (path, method)。domain 固定为 "system"
// （去 carrier 多租户隔离，详见 §11.11 Q3）。
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
