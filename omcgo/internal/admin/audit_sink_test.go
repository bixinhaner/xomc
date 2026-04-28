package admin

import (
	"context"
	"errors"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/admin/audit"
	"github.com/omcgo/omcgo/internal/core/model"
)

// ----------------------------------------------------------------------------
// Tests for AuditSink (cross-module audit -> admin.AuditRepository adapter).
// ----------------------------------------------------------------------------

// auditSinkMockRepo captures Create calls; List is not exercised here.
type auditSinkMockRepo struct {
	mu      sync.Mutex
	written []*AuditLog
	createE error
}

func (m *auditSinkMockRepo) Create(_ context.Context, log *AuditLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.written = append(m.written, log)
	return m.createE
}

func (m *auditSinkMockRepo) List(_ context.Context, _ AuditLogFilter) (*model.ListResponse[AuditLog], error) {
	return nil, nil
}

func (m *auditSinkMockRepo) snapshot() []*AuditLog {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*AuditLog, len(m.written))
	copy(out, m.written)
	return out
}

func TestAuditSink_Write_MapsEntryToAuditLog(t *testing.T) {
	repo := &auditSinkMockRepo{}
	sink := NewAuditSink(repo)

	uid := uuid.New()
	err := sink.Write(context.Background(), audit.Entry{
		UserID:       &uid,
		Username:     "alice",
		Action:       audit.ActionDelete,
		ResourceType: audit.ResourceDevice,
		ResourceID:   "dev-1",
		Details:      map[string]interface{}{"k": "v"},
		IPAddress:    "10.0.0.1",
		UserAgent:    "TestAgent",
		Success:      true,
	})
	require.NoError(t, err)

	got := repo.snapshot()
	require.Len(t, got, 1)
	assert.Equal(t, "alice", got[0].Username)
	assert.Equal(t, audit.ActionDelete, got[0].Action) // success: action unchanged
	assert.Equal(t, audit.ResourceDevice, got[0].Resource)
	assert.Equal(t, "dev-1", got[0].ResourceID)
	assert.Equal(t, "10.0.0.1", got[0].IPAddress)
	assert.Equal(t, "TestAgent", got[0].UserAgent)
	assert.Equal(t, "v", got[0].Details["k"])
}

func TestAuditSink_Write_FailureSuffixesActionAndIncludesError(t *testing.T) {
	repo := &auditSinkMockRepo{}
	sink := NewAuditSink(repo)

	err := sink.Write(context.Background(), audit.Entry{
		Action:       audit.ActionReboot,
		ResourceType: audit.ResourceDevice,
		ResourceID:   "dev-2",
		Success:      false,
		ErrorMessage: "device offline",
	})
	require.NoError(t, err)

	got := repo.snapshot()
	require.Len(t, got, 1)
	assert.Equal(t, "reboot_failed", got[0].Action,
		"failed entries should suffix action with _failed")
	assert.Equal(t, "device offline", got[0].Details["error"])
}

func TestAuditSink_Write_NilRepoIsNoop(t *testing.T) {
	var sink *AuditSink // intentionally nil
	require.NotPanics(t, func() {
		_ = sink.Write(context.Background(), audit.Entry{Action: audit.ActionLogin})
	})

	sink = NewAuditSink(nil)
	require.NoError(t, sink.Write(context.Background(), audit.Entry{Action: audit.ActionLogin}))
}

func TestAuditSink_Write_PropagatesRepoError(t *testing.T) {
	repo := &auditSinkMockRepo{createE: errors.New("db down")}
	sink := NewAuditSink(repo)

	err := sink.Write(context.Background(), audit.Entry{
		Action: audit.ActionConfig, Success: true,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db down")
}

// ----------------------------------------------------------------------------
// Tests for AuditContextFromGin helper.
// ----------------------------------------------------------------------------

func TestAuditContextFromGin_ExtractsUserAndRequestMeta(t *testing.T) {
	gin.SetMode(gin.TestMode)

	uid := uuid.New()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/x", nil)
	c.Request.Header.Set("User-Agent", "Mozilla/Test")
	c.Set(CtxKeyUserID, uid)
	c.Set(CtxKeyUsername, "bob")

	got := AuditContextFromGin(c)
	require.NotNil(t, got.UserID)
	assert.Equal(t, uid, *got.UserID)
	assert.Equal(t, "bob", got.Username)
	assert.Equal(t, "Mozilla/Test", got.UserAgent)
	assert.NotEmpty(t, got.IPAddress)
}

func TestAuditContextFromGin_NilContextReturnsEmpty(t *testing.T) {
	got := AuditContextFromGin(nil)
	assert.Nil(t, got.UserID)
	assert.Empty(t, got.Username)
}

func TestAuditContextFromGin_UnauthenticatedReturnsZeroUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/x", nil)

	got := AuditContextFromGin(c)
	assert.Nil(t, got.UserID, "unauthenticated request should not produce UserID")
}

// ----------------------------------------------------------------------------
// Tests for NewAdminService side-effect: package-singleton wiring.
// ----------------------------------------------------------------------------

func TestNewAdminService_SetsDefaultAuditSink(t *testing.T) {
	// After construction, audit.Default() should be a sink that ultimately
	// hits our mock repo. Use audit.Log as the smoke test.
	repo := &auditSinkMockRepo{}
	jwt, err := NewJWTService("test-secret-key-minimum-32-chars!!")
	require.NoError(t, err)
	svc := NewAdminService(
		&handlerMockUserRepo{},
		&handlerMockRoleRepo{},
		&handlerMockMenuRepo{},
		repo,
		jwt,
		zap.NewNop(),
	)
	require.NotNil(t, svc)

	audit.Log(context.Background(), audit.Entry{
		Action:       audit.ActionLogin,
		ResourceType: audit.ResourceAuth,
		Success:      true,
		Username:     "alice",
	})

	got := repo.snapshot()
	require.NotEmpty(t, got, "NewAdminService should register a default audit sink that routes audit.Log to the AuditRepository")
	assert.Equal(t, audit.ActionLogin, got[0].Action)

	// Reset for other tests.
	audit.SetDefault(nil)
}
