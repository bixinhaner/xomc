package admin

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/casbin/casbin/v2"
	casbinModel "github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// --- Helpers ---

const testModelPath = "../../configs/casbin_model.conf"

func newTestEnforcer(t *testing.T) *casbin.SyncedEnforcer {
	t.Helper()
	m, err := casbinModel.NewModelFromFile(testModelPath)
	require.NoError(t, err, "failed to load casbin model")
	e, err := casbin.NewSyncedEnforcer(m)
	require.NoError(t, err, "failed to create enforcer")
	return e
}

type concurrentPolicyAdapter struct {
	mu    sync.RWMutex
	allow bool
}

func (a *concurrentPolicyAdapter) LoadPolicy(m casbinModel.Model) error {
	a.mu.RLock()
	allow := a.allow
	a.mu.RUnlock()

	if allow {
		persist.LoadPolicyLine("p, role:viewer, system, /api/v1/admin/sysConfig, GET", m)
	}
	persist.LoadPolicyLine("g, 20000000-0000-0000-0000-000000000001, role:viewer, system", m)
	return nil
}

func (a *concurrentPolicyAdapter) SavePolicy(casbinModel.Model) error { return nil }

func (a *concurrentPolicyAdapter) AddPolicy(string, string, []string) error { return nil }

func (a *concurrentPolicyAdapter) RemovePolicy(string, string, []string) error { return nil }

func (a *concurrentPolicyAdapter) RemoveFilteredPolicy(string, string, int, ...string) error {
	return nil
}

func (a *concurrentPolicyAdapter) toggle() {
	a.mu.Lock()
	a.allow = !a.allow
	a.mu.Unlock()
}

// newTestAuthorizer creates a CasbinAuthorizer with pre-loaded policies for testing.
// No database or Redis required — policies are added directly to the enforcer.
func newTestAuthorizer(t *testing.T) *CasbinAuthorizer {
	t.Helper()
	e := newTestEnforcer(t)

	// --- Permission policies: p = sub, dom, obj, act ---

	// admin: full access via wildcard action
	e.AddPolicy("role:admin", "system", "devices", "*")
	e.AddPolicy("role:admin", "system", "users", "*")
	e.AddPolicy("role:admin", "system", "roles", "*")
	e.AddPolicy("role:admin", "system", "dashboard", "read")
	e.AddPolicy("role:admin", "system", "api/v1/*", "*")

	// operator: read/write devices, read dashboard
	e.AddPolicy("role:operator", "system", "devices", "read")
	e.AddPolicy("role:operator", "system", "devices", "write")
	e.AddPolicy("role:operator", "system", "dashboard", "read")

	// viewer: read-only
	e.AddPolicy("role:viewer", "system", "devices", "read")
	e.AddPolicy("role:viewer", "system", "dashboard", "read")

	// carrier_admin: full device access, limited user access
	e.AddPolicy("role:carrier_admin", "system", "devices", "*")
	e.AddPolicy("role:carrier_admin", "system", "users", "read")
	e.AddPolicy("role:carrier_admin", "system", "users", "list")

	return &CasbinAuthorizer{
		enforcer: e,
		logger:   zap.NewNop(),
	}
}

// --- getDomainFromContext: removed in v1.0 (Casbin domain 已统一为 "system") ---
// 旧测试 TestGetDomainFromContext_* 与 carrier domain 一同移除（详见 PRD §11.11 决议 Q3）。

// --- CheckPermission: wildcard action ---

func TestCheckPermission_AdminWildcardAction(t *testing.T) {
	auth := newTestAuthorizer(t)
	adminID := uuid.New()
	auth.enforcer.AddNamedGroupingPolicy("g", adminID.String(), "role:admin", "system")

	ctx := context.Background()

	for _, action := range []string{"read", "write", "delete", "update"} {
		ok, err := auth.CheckPermission(ctx, adminID, "devices", action)
		require.NoError(t, err)
		assert.True(t, ok, "admin should be allowed to %s devices", action)
	}
}

// --- CheckPermission: specific action ---

func TestCheckPermission_OperatorReadWrite(t *testing.T) {
	auth := newTestAuthorizer(t)
	opID := uuid.New()
	auth.enforcer.AddNamedGroupingPolicy("g", opID.String(), "role:operator", "system")

	ctx := context.Background()

	ok, err := auth.CheckPermission(ctx, opID, "devices", "read")
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = auth.CheckPermission(ctx, opID, "devices", "write")
	require.NoError(t, err)
	assert.True(t, ok)

	// operator has no delete permission
	ok, err = auth.CheckPermission(ctx, opID, "devices", "delete")
	require.NoError(t, err)
	assert.False(t, ok)
}

// --- CheckPermission: deny for unassigned user ---

func TestCheckPermission_DenyNoRole(t *testing.T) {
	auth := newTestAuthorizer(t)
	unknownID := uuid.New()

	ctx := context.Background()
	ok, err := auth.CheckPermission(ctx, unknownID, "devices", "read")
	require.NoError(t, err)
	assert.False(t, ok)
}

// --- CheckPermission: deny for viewer writing ---

func TestCheckPermission_ViewerDenyWrite(t *testing.T) {
	auth := newTestAuthorizer(t)
	viewerID := uuid.New()
	auth.enforcer.AddNamedGroupingPolicy("g", viewerID.String(), "role:viewer", "system")

	ctx := context.Background()

	ok, err := auth.CheckPermission(ctx, viewerID, "devices", "read")
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = auth.CheckPermission(ctx, viewerID, "devices", "write")
	require.NoError(t, err)
	assert.False(t, ok)
}

// --- CheckPermission: role inheritance ---

func TestCheckPermission_RoleInheritance(t *testing.T) {
	auth := newTestAuthorizer(t)
	superID := uuid.New()

	// super_admin inherits from admin
	auth.enforcer.AddNamedGroupingPolicy("g", "role:super_admin", "role:admin", "system")
	auth.enforcer.AddNamedGroupingPolicy("g", superID.String(), "role:super_admin", "system")

	ctx := context.Background()

	// super_admin should inherit all admin permissions
	ok, err := auth.CheckPermission(ctx, superID, "devices", "delete")
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = auth.CheckPermission(ctx, superID, "users", "create")
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = auth.CheckPermission(ctx, superID, "roles", "update")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestCheckPermission_BuiltInHierarchyDoesNotElevateViewer(t *testing.T) {
	auth := newTestAuthorizer(t)
	auth.enforcer.AddPolicy("role:admin", "system", "/api/v1/device-access/policies/drafts", "POST")
	auth.enforcer.AddPolicy("role:operator", "system", "/api/v1/device-access/devices/:serialNumber/reevaluate", "POST")
	auth.enforcer.AddPolicy("role:viewer", "system", "/api/v1/device-access/states", "GET")
	auth.enforcer.AddNamedGroupingPolicy("g", "role:admin", "role:operator", "system")
	auth.enforcer.AddNamedGroupingPolicy("g", "role:operator", "role:viewer", "system")

	viewerID := uuid.New()
	auth.enforcer.AddNamedGroupingPolicy("g", viewerID.String(), "role:viewer", "system")

	allowed, err := auth.CheckPermission(context.Background(), viewerID, "devices", "read")
	require.NoError(t, err)
	require.True(t, allowed)

	allowed, err = auth.CheckPermission(context.Background(), viewerID, "devices", "write")
	require.NoError(t, err)
	require.False(t, allowed)

	allowed, err = auth.CheckPermission(context.Background(), viewerID, "users", "create")
	require.NoError(t, err)
	require.False(t, allowed)

	allowed, err = auth.CheckPermission(context.Background(), viewerID, "/api/v1/device-access/states", "GET")
	require.NoError(t, err)
	require.True(t, allowed)

	allowed, err = auth.CheckPermission(context.Background(), viewerID, "/api/v1/device-access/policies/drafts", "POST")
	require.NoError(t, err)
	require.False(t, allowed)

	allowed, err = auth.CheckPermission(context.Background(), viewerID, "/api/v1/device-access/devices/:serialNumber/reevaluate", "POST")
	require.NoError(t, err)
	require.False(t, allowed)

	operatorID := uuid.New()
	auth.enforcer.AddNamedGroupingPolicy("g", operatorID.String(), "role:operator", "system")
	allowed, err = auth.CheckPermission(context.Background(), operatorID, "/api/v1/device-access/devices/:serialNumber/reevaluate", "POST")
	require.NoError(t, err)
	require.True(t, allowed)

	adminID := uuid.New()
	auth.enforcer.AddNamedGroupingPolicy("g", adminID.String(), "role:admin", "system")
	allowed, err = auth.CheckPermission(context.Background(), adminID, "/api/v1/device-access/devices/:serialNumber/reevaluate", "POST")
	require.NoError(t, err)
	require.True(t, allowed)
}

// --- CheckPermission: domain isolation ---
// v1.0：domain 已统一为 "system"（详见 PRD §11.11 决议 Q3），多 carrier domain 隔离测试已废弃。
// 旧 TestCheckPermission_DomainIsolation 与 carrier-specific group policy 一同移除。

// --- CheckPermission: keyMatch resource ---

func TestCheckPermission_KeyMatchResource(t *testing.T) {
	auth := newTestAuthorizer(t)
	adminID := uuid.New()
	auth.enforcer.AddNamedGroupingPolicy("g", adminID.String(), "role:admin", "system")

	ctx := context.Background()

	// Admin has policy for "api/v1/*" — keyMatch should match sub-paths
	ok, err := auth.CheckPermission(ctx, adminID, "api/v1/devices", "read")
	require.NoError(t, err)
	assert.True(t, ok, "keyMatch should match api/v1/devices against api/v1/*")

	ok, err = auth.CheckPermission(ctx, adminID, "api/v1/users/123", "update")
	require.NoError(t, err)
	assert.True(t, ok, "keyMatch should match api/v1/users/123 against api/v1/*")

	// Non-matching resource
	ok, err = auth.CheckPermission(ctx, adminID, "api/v2/devices", "read")
	require.NoError(t, err)
	assert.False(t, ok, "keyMatch should NOT match api/v2/devices against api/v1/*")
}

// --- CheckPermission: system domain only (v1.0：carrier domain 已废弃) ---

func TestCheckPermission_SystemUserAllDomains(t *testing.T) {
	auth := newTestAuthorizer(t)
	sysAdminID := uuid.New()

	auth.enforcer.AddNamedGroupingPolicy("g", sysAdminID.String(), "role:admin", "system")

	ctx := context.Background()
	ok, err := auth.CheckPermission(ctx, sysAdminID, "users", "delete")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestCasbinAuthorizerConcurrentEnforceAndReload(t *testing.T) {
	adapter := &concurrentPolicyAdapter{allow: true}
	m, err := casbinModel.NewModelFromFile(testModelPath)
	require.NoError(t, err)
	enforcer, err := casbin.NewSyncedEnforcer(m, adapter)
	require.NoError(t, err)

	auth := &CasbinAuthorizer{enforcer: enforcer, logger: zap.NewNop()}
	userID := uuid.MustParse("20000000-0000-0000-0000-000000000001")

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 500 {
				_, enforceErr := auth.CheckPermission(
					context.Background(),
					userID,
					"/api/v1/admin/sysConfig",
					"GET",
				)
				assert.NoError(t, enforceErr)
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 200 {
			adapter.toggle()
			assert.NoError(t, auth.ReloadPolicy())
		}
	}()
	wg.Wait()
}

// --- pgAdapter no-op methods ---

func TestPgAdapter_SavePolicy_NoOp(t *testing.T) {
	var a pgAdapter
	assert.NoError(t, a.SavePolicy(nil))
}

func TestPgAdapter_AddPolicy_NoOp(t *testing.T) {
	var a pgAdapter
	assert.NoError(t, a.AddPolicy("p", "p", []string{"sub", "dom", "obj", "act"}))
}

func TestPgAdapter_RemovePolicy_NoOp(t *testing.T) {
	var a pgAdapter
	assert.NoError(t, a.RemovePolicy("p", "p", []string{"sub", "dom", "obj", "act"}))
}

func TestPgAdapter_RemoveFilteredPolicy_NoOp(t *testing.T) {
	var a pgAdapter
	assert.NoError(t, a.RemoveFilteredPolicy("p", "p", 0, "sub"))
}

// --- redisWatcher ---

func TestRedisWatcher_SetUpdateCallback(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer client.Close()

	w := newRedisWatcher(client)

	called := false
	require.NoError(t, w.SetUpdateCallback(func(msg string) {
		called = true
		assert.Equal(t, "reload", msg)
	}))

	w.callback("reload")
	assert.True(t, called)
}

func TestRedisWatcher_NotifyPublishes(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer client.Close()

	// Subscribe before publishing
	subs := client.Subscribe(context.Background(), casbinPolicyChannel)
	defer subs.Close()

	w := newRedisWatcher(client)
	require.NoError(t, w.Notify())

	select {
	case msg := <-subs.Channel():
		assert.Equal(t, "reload", msg.Payload)
	case <-time.After(time.Second):
		t.Fatal("expected to receive reload notification")
	}
}

func TestRedisWatcher_ListenerCallback(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer client.Close()

	w := newRedisWatcher(client)

	received := make(chan string, 1)
	w.SetUpdateCallback(func(msg string) {
		received <- msg
	})

	w.StartListener()
	defer w.Close()

	// Allow subscription to establish
	time.Sleep(50 * time.Millisecond)

	require.NoError(t, client.Publish(context.Background(), casbinPolicyChannel, "reload").Err())

	select {
	case msg := <-received:
		assert.Equal(t, "reload", msg)
	case <-time.After(time.Second):
		t.Fatal("callback was not invoked")
	}
}

func TestRedisWatcher_UpdateIsNotify(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer client.Close()

	w := newRedisWatcher(client)

	subs := client.Subscribe(context.Background(), casbinPolicyChannel)
	defer subs.Close()

	require.NoError(t, w.Update())

	select {
	case msg := <-subs.Channel():
		assert.Equal(t, "reload", msg.Payload)
	case <-time.After(time.Second):
		t.Fatal("Update() should publish to Redis channel")
	}
}

// --- Model loading ---

func TestCasbinModel_Load(t *testing.T) {
	m, err := casbinModel.NewModelFromFile(testModelPath)
	require.NoError(t, err)
	require.NotNil(t, m)

	// Verify key sections exist
	assert.Contains(t, m, "r")
	assert.Contains(t, m, "p")
	assert.Contains(t, m, "g")
	assert.Contains(t, m, "e")
	assert.Contains(t, m, "m")
}

// --- Benchmarks ---

func BenchmarkCheckPermission_Allow(b *testing.B) {
	m, err := casbinModel.NewModelFromFile(testModelPath)
	if err != nil {
		b.Fatal(err)
	}
	e, err := casbin.NewSyncedEnforcer(m)
	if err != nil {
		b.Fatal(err)
	}

	e.AddPolicy("role:admin", "system", "devices", "*")
	e.AddPolicy("role:admin", "system", "users", "*")
	e.AddPolicy("role:admin", "system", "roles", "*")

	adminID := uuid.New().String()
	e.AddNamedGroupingPolicy("g", adminID, "role:admin", "system")

	auth := &CasbinAuthorizer{enforcer: e, logger: zap.NewNop()}
	ctx := context.Background()
	uid := uuid.MustParse(adminID)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		auth.CheckPermission(ctx, uid, "devices", "read")
	}
}

func BenchmarkCheckPermission_Deny(b *testing.B) {
	m, err := casbinModel.NewModelFromFile(testModelPath)
	if err != nil {
		b.Fatal(err)
	}
	e, err := casbin.NewSyncedEnforcer(m)
	if err != nil {
		b.Fatal(err)
	}

	e.AddPolicy("role:viewer", "system", "devices", "read")

	viewerID := uuid.New().String()
	e.AddNamedGroupingPolicy("g", viewerID, "role:viewer", "system")

	auth := &CasbinAuthorizer{enforcer: e, logger: zap.NewNop()}
	ctx := context.Background()
	uid := uuid.MustParse(viewerID)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		auth.CheckPermission(ctx, uid, "devices", "write")
	}
}

func BenchmarkCheckPermission_WithRoleInheritance(b *testing.B) {
	m, err := casbinModel.NewModelFromFile(testModelPath)
	if err != nil {
		b.Fatal(err)
	}
	e, err := casbin.NewSyncedEnforcer(m)
	if err != nil {
		b.Fatal(err)
	}

	e.AddPolicy("role:admin", "system", "devices", "*")
	e.AddPolicy("role:admin", "system", "users", "*")
	e.AddNamedGroupingPolicy("g", "role:super_admin", "role:admin", "system")

	superID := uuid.New().String()
	e.AddNamedGroupingPolicy("g", superID, "role:super_admin", "system")

	auth := &CasbinAuthorizer{enforcer: e, logger: zap.NewNop()}
	ctx := context.Background()
	uid := uuid.MustParse(superID)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		auth.CheckPermission(ctx, uid, "devices", "read")
	}
}
