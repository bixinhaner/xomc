package notification

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/model"
)

// newDeadPool returns a pgxpool.Pool that points at an unreachable address with
// short timeouts. Any query against it will fail quickly, exercising the
// pre-query SQL building branches in PgRepository methods. This gives partial
// coverage without needing a real PostgreSQL server in CI.
func newDeadPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	cfg, err := pgxpool.ParseConfig("postgres://x:y@127.0.0.1:1/none?sslmode=disable&connect_timeout=1")
	require.NoError(t, err)
	cfg.MaxConns = 1
	cfg.MinConns = 0
	cfg.HealthCheckPeriod = time.Hour // disable background pings
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	return pool
}

// fastCtx returns a context that cancels quickly so dial attempts on the dead
// pool don't stall the test for the full TCP timeout.
func fastCtx(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	t.Cleanup(cancel)
	return ctx
}

// ---------- NewPgRepository ----------

func TestNewPgRepository(t *testing.T) {
	pool := newDeadPool(t)
	repo := NewPgRepository(pool)
	require.NotNil(t, repo)
	assert.Equal(t, pool, repo.pool)
}

// ---------- joinColumns ----------

func TestJoinColumns(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want string
	}{
		{"empty", []string{}, ""},
		{"single", []string{"a"}, "a"},
		{"multiple", []string{"a", "b", "c"}, "a, b, c"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, joinColumns(tt.in))
		})
	}
}

// ---------- scanNotification ----------

// fakeRow implements the scan target interface used by scanNotification.
type fakeRow struct {
	scanFn func(dest ...any) error
}

func (f *fakeRow) Scan(dest ...any) error { return f.scanFn(dest...) }

func TestScanNotification_OK(t *testing.T) {
	id := uuid.New()
	now := time.Now()
	row := &fakeRow{
		scanFn: func(dest ...any) error {
			// pg_repository scans 11 columns:
			// id, user_id, type, priority, title, content, link, sender,
			// is_read, read_at, created_at
			require.Len(t, dest, 11)
			*dest[0].(*uuid.UUID) = id
			*dest[1].(*string) = "alice"
			*dest[2].(*NotificationType) = NotifTypeAlarm
			*dest[3].(*NotificationPriority) = PriorityHigh
			*dest[4].(*string) = "title"
			*dest[5].(*string) = "content"
			*dest[6].(*string) = "/x"
			*dest[7].(*string) = "system"
			*dest[8].(*bool) = false
			*dest[9].(**time.Time) = nil
			*dest[10].(*time.Time) = now
			return nil
		},
	}
	got, err := scanNotification(row)
	require.NoError(t, err)
	assert.Equal(t, id, got.ID)
	assert.Equal(t, "alice", got.UserID)
	assert.Equal(t, NotifTypeAlarm, got.Type)
	assert.Equal(t, PriorityHigh, got.Priority)
	assert.False(t, got.IsRead)
}

func TestScanNotification_Error(t *testing.T) {
	row := &fakeRow{scanFn: func(_ ...any) error { return assertErr }}
	_, err := scanNotification(row)
	require.Error(t, err)
	assert.ErrorIs(t, err, assertErr)
}

var assertErr = errBoom

// ---------- Method failure paths against a dead pool ----------
//
// These tests exercise the SQL building / error-wrapping branches of every
// PgRepository method. Since the pool can't connect, every database call
// errors out — but the SQL has been built first, so we cover those statements.

func TestPgRepository_List_DeadPool(t *testing.T) {
	repo := NewPgRepository(newDeadPool(t))
	_, err := repo.List(fastCtx(t), NotificationFilter{
		UserID:      "alice",
		ListRequest: model.DefaultListRequest(),
	})
	require.Error(t, err)
}

func TestPgRepository_List_FullFilters_DeadPool(t *testing.T) {
	repo := NewPgRepository(newDeadPool(t))
	typ := NotifTypeSystem
	read := true
	filter := NotificationFilter{
		UserID:      "alice",
		Type:        &typ,
		IsRead:      &read,
		ListRequest: model.ListRequest{Page: 2, PageSize: 10, SortBy: "priority", SortDir: "asc"},
	}
	_, err := repo.List(fastCtx(t), filter)
	require.Error(t, err)
}

func TestPgRepository_List_DisallowedSortBy_DeadPool(t *testing.T) {
	// SortBy not in whitelist falls back to "created_at" (default branch).
	repo := NewPgRepository(newDeadPool(t))
	filter := NotificationFilter{
		UserID:      "alice",
		ListRequest: model.ListRequest{Page: 1, PageSize: 5, SortBy: "drop_table", SortDir: "asc"},
	}
	_, err := repo.List(fastCtx(t), filter)
	require.Error(t, err)
}

func TestPgRepository_GetByID_DeadPool(t *testing.T) {
	repo := NewPgRepository(newDeadPool(t))
	_, err := repo.GetByID(fastCtx(t), uuid.New())
	require.Error(t, err)
}

func TestPgRepository_Create_DeadPool(t *testing.T) {
	repo := NewPgRepository(newDeadPool(t))
	n := &Notification{
		UserID:   "alice",
		Type:     NotifTypeSystem,
		Priority: PriorityNormal,
		Title:    "t",
	}
	err := repo.Create(fastCtx(t), n)
	require.Error(t, err)
}

func TestPgRepository_MarkRead_DeadPool(t *testing.T) {
	repo := NewPgRepository(newDeadPool(t))
	err := repo.MarkRead(fastCtx(t), uuid.New(), "alice")
	require.Error(t, err)
}

func TestPgRepository_MarkAllRead_DeadPool(t *testing.T) {
	repo := NewPgRepository(newDeadPool(t))
	err := repo.MarkAllRead(fastCtx(t), "alice")
	require.Error(t, err)
}

func TestPgRepository_GetUnreadCount_DeadPool(t *testing.T) {
	repo := NewPgRepository(newDeadPool(t))
	_, err := repo.GetUnreadCount(fastCtx(t), "alice")
	require.Error(t, err)
}

func TestPgRepository_Delete_DeadPool(t *testing.T) {
	repo := NewPgRepository(newDeadPool(t))
	err := repo.Delete(fastCtx(t), uuid.New(), "alice")
	require.Error(t, err)
}
