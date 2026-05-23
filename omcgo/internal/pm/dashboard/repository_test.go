//go:build integration

package dashboard

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 集成测试（依赖 docker postgres + migration 000163）：
//   1. Create + Get + List by owner
//   2. Update（partial fields）
//   3. Share / Unshare（shared_with 列表）
//   4. List by shared user 能看到
//   5. Fork（panels 同步复制）
//   6. Panel CRUD + dashboard 删除级联
//   7. UserPreferences upsert + get
//
// 运行：OMCGO_DB_DSN=postgres://omcgo:omcgo123@localhost:5432/omcgo?sslmode=disable \
//        go test -tags integration -count=1 ./internal/pm/dashboard/...

func openPoolOrSkip(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("OMCGO_DB_DSN")
	if dsn == "" {
		t.Skip("OMCGO_DB_DSN not set; skipping integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	return pool
}

// 创建一个临时用户 + 返 cleanup
func ensureUser(t *testing.T, pool *pgxpool.Pool, username string) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
INSERT INTO users (username, password_hash, status) VALUES ($1, 'x', 'active')
ON CONFLICT (username) DO UPDATE SET status='active'
RETURNING id`, username).Scan(&id)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE username=$1`, username)
	})
	return id
}

func Test_Repository_DashboardCreateGetList(t *testing.T) {
	pool := openPoolOrSkip(t)
	defer pool.Close()
	r := NewPgRepository(pool)
	ctx := context.Background()

	owner := ensureUser(t, pool, "test-dash-owner-"+uuid.New().String()[:8])

	d, err := r.CreateDashboard(ctx, owner, CreateDashboardRequest{
		Name: "test-dash", Description: "test", Technology: TechLTE,
	})
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, d.ID)
	defer pool.Exec(ctx, `DELETE FROM pm_dashboards WHERE id=$1`, d.ID)

	got, err := r.GetDashboard(ctx, d.ID)
	require.NoError(t, err)
	assert.Equal(t, d.ID, got.ID)
	assert.Equal(t, TechLTE, got.Technology)

	list, err := r.ListByOwnerOrShared(ctx, owner)
	require.NoError(t, err)
	found := false
	for _, x := range list {
		if x.ID == d.ID {
			found = true
		}
	}
	assert.True(t, found)
}

func Test_Repository_UpdateDashboardPartial(t *testing.T) {
	pool := openPoolOrSkip(t)
	defer pool.Close()
	r := NewPgRepository(pool)
	ctx := context.Background()

	owner := ensureUser(t, pool, "test-dash-update-"+uuid.New().String()[:8])
	d, err := r.CreateDashboard(ctx, owner, CreateDashboardRequest{Name: "old", Technology: TechLTE})
	require.NoError(t, err)
	defer pool.Exec(ctx, `DELETE FROM pm_dashboards WHERE id=$1`, d.ID)

	newName := "new-name"
	newTech := TechNR
	require.NoError(t, r.UpdateDashboard(ctx, d.ID, UpdateDashboardRequest{
		Name: &newName, Technology: &newTech,
	}))
	got, _ := r.GetDashboard(ctx, d.ID)
	assert.Equal(t, "new-name", got.Name)
	assert.Equal(t, TechNR, got.Technology)
}

func Test_Repository_ShareAndUnshare(t *testing.T) {
	pool := openPoolOrSkip(t)
	defer pool.Close()
	r := NewPgRepository(pool)
	ctx := context.Background()

	owner := ensureUser(t, pool, "test-dash-share-owner-"+uuid.New().String()[:8])
	other := ensureUser(t, pool, "test-dash-share-other-"+uuid.New().String()[:8])

	d, _ := r.CreateDashboard(ctx, owner, CreateDashboardRequest{Name: "shared", Technology: TechLTE})
	defer pool.Exec(ctx, `DELETE FROM pm_dashboards WHERE id=$1`, d.ID)

	require.NoError(t, r.AddShare(ctx, d.ID, []uuid.UUID{other}))
	got, _ := r.GetDashboard(ctx, d.ID)
	assert.Contains(t, got.SharedWith, other)

	// other 能在 List 看到
	otherList, _ := r.ListByOwnerOrShared(ctx, other)
	found := false
	for _, x := range otherList {
		if x.ID == d.ID {
			found = true
		}
	}
	assert.True(t, found, "shared user should see the dashboard")

	require.NoError(t, r.RemoveShare(ctx, d.ID, other))
	got, _ = r.GetDashboard(ctx, d.ID)
	assert.NotContains(t, got.SharedWith, other)
}

func Test_Repository_ForkCopiesPanels(t *testing.T) {
	pool := openPoolOrSkip(t)
	defer pool.Close()
	r := NewPgRepository(pool)
	ctx := context.Background()

	owner := ensureUser(t, pool, "test-dash-fork-owner-"+uuid.New().String()[:8])
	src, _ := r.CreateDashboard(ctx, owner, CreateDashboardRequest{Name: "src", Technology: TechLTE})
	defer pool.Exec(ctx, `DELETE FROM pm_dashboards WHERE id=$1 OR parent_dashboard_id=$1`, src.ID)

	// 加 2 个 panel
	for i := 0; i < 2; i++ {
		_, err := r.CreatePanel(ctx, CreatePanelRequest{
			DashboardID: src.ID, PanelType: PanelLineChart, Title: "p" + string(rune('A'+i)),
			MetricPaths: []string{"L.Cell.Avail"}, Granularity: "hourly",
			Dimension: DimensionDevice, DeviceSNs: []string{"S1"},
			TimeRange: json.RawMessage(`{"start_offset":"-1h"}`),
		})
		require.NoError(t, err)
	}

	forkOwner := ensureUser(t, pool, "test-dash-fork-other-"+uuid.New().String()[:8])
	newDash, err := r.Fork(ctx, src.ID, forkOwner, "forked")
	require.NoError(t, err)
	require.NotEqual(t, src.ID, newDash.ID)
	assert.Equal(t, forkOwner, newDash.OwnerID)
	require.NotNil(t, newDash.ParentDashboardID)
	assert.Equal(t, src.ID, *newDash.ParentDashboardID)

	// 复制的 panels 应有 2 个，且 dashboard_id 指向新 dashboard
	newPanels, err := r.ListPanels(ctx, newDash.ID)
	require.NoError(t, err)
	assert.Len(t, newPanels, 2)
}

func Test_Repository_PanelCRUDAndCascade(t *testing.T) {
	pool := openPoolOrSkip(t)
	defer pool.Close()
	r := NewPgRepository(pool)
	ctx := context.Background()

	owner := ensureUser(t, pool, "test-dash-panel-"+uuid.New().String()[:8])
	d, _ := r.CreateDashboard(ctx, owner, CreateDashboardRequest{Name: "panel-test", Technology: TechLTE})

	p, err := r.CreatePanel(ctx, CreatePanelRequest{
		DashboardID: d.ID, PanelType: PanelKPICard, Title: "card",
		MetricPaths: []string{"M1"}, Granularity: "hourly", Dimension: DimensionDevice,
	})
	require.NoError(t, err)

	// Update
	require.NoError(t, r.UpdatePanel(ctx, p.ID, CreatePanelRequest{
		DashboardID: d.ID, PanelType: PanelLineChart, Title: "updated",
		MetricPaths: []string{"M1", "M2"}, Granularity: "daily", Dimension: DimensionDevice,
	}))
	got, _ := r.GetPanel(ctx, p.ID)
	assert.Equal(t, "updated", got.Title)
	assert.Equal(t, PanelLineChart, got.PanelType)

	// 级联：删 dashboard → panels 自动消失
	require.NoError(t, r.DeleteDashboard(ctx, d.ID))
	_, err = r.GetPanel(ctx, p.ID)
	assert.ErrorIs(t, err, ErrNotFound)
}

func Test_Repository_UserPreferencesUpsert(t *testing.T) {
	pool := openPoolOrSkip(t)
	defer pool.Close()
	r := NewPgRepository(pool)
	ctx := context.Background()

	user := ensureUser(t, pool, "test-dash-prefs-"+uuid.New().String()[:8])
	layout1 := []byte(`{"order":["k1","k2"]}`)
	require.NoError(t, r.UpsertUserPreferences(ctx, user, layout1))

	got, err := r.GetUserPreferences(ctx, user)
	require.NoError(t, err)
	assert.JSONEq(t, string(layout1), string(got.KPICardLayout))

	// upsert 覆盖
	layout2 := []byte(`{"order":["k3"]}`)
	require.NoError(t, r.UpsertUserPreferences(ctx, user, layout2))
	got2, _ := r.GetUserPreferences(ctx, user)
	assert.JSONEq(t, string(layout2), string(got2.KPICardLayout))
}
