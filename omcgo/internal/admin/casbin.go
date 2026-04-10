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
)

const casbinPolicyChannel = "casbin:policy:reload"

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
	// Carrier-level isolation is enforced via g (role assignment) domain.
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

	// 2. Load role assignments: g = (user_id, role:{name}, domain)
	rows2, err := a.pool.Query(ctx, `
		SELECT u.id::text AS user_id, r.name AS role_name,
		       COALESCE(u.carrier, 'system') AS domain
		FROM user_roles ur
		JOIN roles r ON r.id = ur.role_id
		JOIN users u ON u.id = ur.user_id
	`)
	if err != nil {
		return fmt.Errorf("load role assignments: %w", err)
	}
	defer rows2.Close()

	for rows2.Next() {
		var userID, roleName, domain string
		if err := rows2.Scan(&userID, &roleName, &domain); err != nil {
			return fmt.Errorf("scan role assignment: %w", err)
		}
		m.AddPolicy("g", "g", []string{userID, "role:" + roleName, domain})
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
type CasbinAuthorizer struct {
	enforcer *casbin.Enforcer
	adapter  *pgAdapter
	watcher  *redisWatcher
	logger   *zap.Logger
}

// NewCasbinAuthorizer creates and initializes the Casbin permission engine.
func NewCasbinAuthorizer(pool *pgxpool.Pool, redisClient redis.UniversalClient, modelPath string, logger *zap.Logger) (*CasbinAuthorizer, error) {
	// Load model from file
	m, err := casbinModel.NewModelFromFile(modelPath)
	if err != nil {
		return nil, fmt.Errorf("load casbin model: %w", err)
	}

	// Create adapter
	adapter := newPgAdapter(pool)

	// Create enforcer
	enforcer, err := casbin.NewEnforcer(m, adapter)
	if err != nil {
		return nil, fmt.Errorf("create casbin enforcer: %w", err)
	}

	// Create watcher
	watcher := newRedisWatcher(redisClient)
	enforcer.SetWatcher(watcher)

	// Set up reload callback
	watcher.SetUpdateCallback(func(_ string) {
		if err := enforcer.LoadPolicy(); err != nil {
			logger.Error("casbin policy reload failed", zap.Error(err))
		} else {
			logger.Info("casbin policy reloaded")
		}
	})

	// Start listening for updates
	watcher.StartListener()

	return &CasbinAuthorizer{
		enforcer: enforcer,
		adapter:  adapter,
		watcher:  watcher,
		logger:   logger,
	}, nil
}

// CheckPermission checks if a subject has permission (implements PermissionChecker interface).
func (a *CasbinAuthorizer) CheckPermission(ctx context.Context, userID uuid.UUID, resource, action string) (bool, error) {
	domain := getDomainFromContext(ctx)
	ok, err := a.enforcer.Enforce(userID.String(), domain, resource, action)
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
func (a *CasbinAuthorizer) NotifyPolicyChange() error {
	return a.watcher.Notify()
}

// Stop closes the watcher.
func (a *CasbinAuthorizer) Stop() {
	a.watcher.Close()
}

// getDomainFromContext extracts the domain (carrier code or "system") from context.
func getDomainFromContext(ctx context.Context) string {
	if carrier := ctx.Value(CtxKeyCarrier); carrier != nil {
		if code, ok := carrier.(string); ok && code != "" {
			return code
		}
	}
	return "system"
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
