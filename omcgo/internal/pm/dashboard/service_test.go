package dashboard

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubRepo 是 Service 单测用的内存 Repository。
type stubRepo struct {
	mu         sync.Mutex
	dashboards map[uuid.UUID]*Dashboard
	panels     map[uuid.UUID]*Panel
	prefsByKey map[string]*UserPreferences // key = userID + "/" + technology
}

func newStubRepo() *stubRepo {
	return &stubRepo{
		dashboards: map[uuid.UUID]*Dashboard{},
		panels:     map[uuid.UUID]*Panel{},
		prefsByKey: map[string]*UserPreferences{},
	}
}

func (s *stubRepo) CreateDashboard(_ context.Context, ownerID uuid.UUID, req CreateDashboardRequest) (*Dashboard, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d := &Dashboard{
		ID: uuid.New(), Name: req.Name, Description: req.Description,
		OwnerID: ownerID, Technology: req.Technology, SharedWith: []uuid.UUID{},
		Layout: []byte(`{"panels":[]}`),
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	s.dashboards[d.ID] = d
	return d, nil
}
func (s *stubRepo) GetDashboard(_ context.Context, id uuid.UUID) (*Dashboard, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if d, ok := s.dashboards[id]; ok {
		return d, nil
	}
	return nil, ErrNotFound
}
func (s *stubRepo) ListByOwnerOrShared(_ context.Context, userID uuid.UUID) ([]Dashboard, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []Dashboard{}
	for _, d := range s.dashboards {
		if d.OwnerID == userID || contains(d.SharedWith, userID) {
			out = append(out, *d)
		}
	}
	return out, nil
}
func (s *stubRepo) UpdateDashboard(_ context.Context, id uuid.UUID, req UpdateDashboardRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.dashboards[id]
	if !ok {
		return ErrNotFound
	}
	if req.Name != nil {
		d.Name = *req.Name
	}
	if req.Description != nil {
		d.Description = *req.Description
	}
	if req.Technology != nil {
		d.Technology = *req.Technology
	}
	return nil
}
func (s *stubRepo) DeleteDashboard(_ context.Context, id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.dashboards[id]; !ok {
		return ErrNotFound
	}
	delete(s.dashboards, id)
	return nil
}
func (s *stubRepo) Fork(_ context.Context, srcID uuid.UUID, ownerID uuid.UUID, newName string) (*Dashboard, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	src, ok := s.dashboards[srcID]
	if !ok {
		return nil, ErrNotFound
	}
	parent := srcID
	d := &Dashboard{
		ID: uuid.New(), Name: newName, OwnerID: ownerID, Technology: src.Technology,
		ParentDashboardID: &parent, SharedWith: []uuid.UUID{},
	}
	s.dashboards[d.ID] = d
	return d, nil
}
func (s *stubRepo) AddShare(_ context.Context, dashID uuid.UUID, userIDs []uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.dashboards[dashID]
	if !ok {
		return ErrNotFound
	}
	for _, u := range userIDs {
		if !contains(d.SharedWith, u) {
			d.SharedWith = append(d.SharedWith, u)
		}
	}
	return nil
}
func (s *stubRepo) RemoveShare(_ context.Context, dashID uuid.UUID, userID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.dashboards[dashID]
	if !ok {
		return ErrNotFound
	}
	for i, u := range d.SharedWith {
		if u == userID {
			d.SharedWith = append(d.SharedWith[:i], d.SharedWith[i+1:]...)
			break
		}
	}
	return nil
}
func (s *stubRepo) CreatePanel(_ context.Context, req CreatePanelRequest) (*Panel, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := &Panel{ID: uuid.New(), DashboardID: req.DashboardID, PanelType: req.PanelType, Title: req.Title}
	s.panels[p.ID] = p
	return p, nil
}
func (s *stubRepo) GetPanel(_ context.Context, id uuid.UUID) (*Panel, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p, ok := s.panels[id]; ok {
		return p, nil
	}
	return nil, ErrNotFound
}
func (s *stubRepo) ListPanels(_ context.Context, dashID uuid.UUID) ([]Panel, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []Panel{}
	for _, p := range s.panels {
		if p.DashboardID == dashID {
			out = append(out, *p)
		}
	}
	return out, nil
}
func (s *stubRepo) UpdatePanel(_ context.Context, id uuid.UUID, req CreatePanelRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p, ok := s.panels[id]; ok {
		p.PanelType = req.PanelType
		p.Title = req.Title
		return nil
	}
	return ErrNotFound
}
func (s *stubRepo) DeletePanel(_ context.Context, id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.panels[id]; !ok {
		return ErrNotFound
	}
	delete(s.panels, id)
	return nil
}
func (s *stubRepo) GetUserPreferences(_ context.Context, userID uuid.UUID, tech Technology) (*UserPreferences, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := userID.String() + "/" + string(tech)
	if p, ok := s.prefsByKey[key]; ok {
		return p, nil
	}
	return nil, ErrNotFound
}
func (s *stubRepo) UpsertUserPreferences(_ context.Context, p *UserPreferences) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.prefsByKey == nil {
		s.prefsByKey = map[string]*UserPreferences{}
	}
	s.prefsByKey[p.UserID.String()+"/"+string(p.Technology)] = p
	return nil
}

func contains(list []uuid.UUID, x uuid.UUID) bool {
	for _, u := range list {
		if u == x {
			return true
		}
	}
	return false
}

// ── Tests ─────────────────────────────────────────────────────────────────

func Test_Service_Get_OwnerReads(t *testing.T) {
	repo := newStubRepo()
	svc := NewService(repo, nil)
	owner := uuid.New()
	d, _ := svc.Create(context.Background(), owner, CreateDashboardRequest{Name: "test", Technology: TechLTE})

	got, err := svc.Get(context.Background(), owner, d.ID)
	require.NoError(t, err)
	assert.Equal(t, d.ID, got.ID)
}

func Test_Service_Get_OtherUserDenied(t *testing.T) {
	repo := newStubRepo()
	svc := NewService(repo, nil)
	owner := uuid.New()
	other := uuid.New()
	d, _ := svc.Create(context.Background(), owner, CreateDashboardRequest{Name: "test", Technology: TechLTE})

	_, err := svc.Get(context.Background(), other, d.ID)
	assert.ErrorIs(t, err, ErrPermissionDenied)
}

func Test_Service_Get_SharedUserAllowed(t *testing.T) {
	repo := newStubRepo()
	svc := NewService(repo, nil)
	owner := uuid.New()
	other := uuid.New()
	d, _ := svc.Create(context.Background(), owner, CreateDashboardRequest{Name: "test", Technology: TechLTE})

	require.NoError(t, svc.Share(context.Background(), owner, ShareRequest{DashboardID: d.ID, UserIDs: []uuid.UUID{other}}))

	got, err := svc.Get(context.Background(), other, d.ID)
	require.NoError(t, err)
	assert.Equal(t, d.ID, got.ID)
}

func Test_Service_Update_OnlyOwner(t *testing.T) {
	repo := newStubRepo()
	svc := NewService(repo, nil)
	owner := uuid.New()
	other := uuid.New()
	d, _ := svc.Create(context.Background(), owner, CreateDashboardRequest{Name: "old", Technology: TechLTE})

	// 给 other share 读权限
	svc.Share(context.Background(), owner, ShareRequest{DashboardID: d.ID, UserIDs: []uuid.UUID{other}})

	newName := "new"
	// owner 可改
	require.NoError(t, svc.Update(context.Background(), owner, d.ID, UpdateDashboardRequest{Name: &newName}))
	// shared 用户不可改
	err := svc.Update(context.Background(), other, d.ID, UpdateDashboardRequest{Name: &newName})
	assert.ErrorIs(t, err, ErrPermissionDenied)
}

func Test_Service_Fork_RequiresReadAccess(t *testing.T) {
	repo := newStubRepo()
	svc := NewService(repo, nil)
	owner := uuid.New()
	other := uuid.New()
	d, _ := svc.Create(context.Background(), owner, CreateDashboardRequest{Name: "src", Technology: TechLTE})

	// other 没读权限，不可 fork
	_, err := svc.Fork(context.Background(), other, d.ID, "forked")
	assert.ErrorIs(t, err, ErrPermissionDenied)

	// share 后可 fork
	svc.Share(context.Background(), owner, ShareRequest{DashboardID: d.ID, UserIDs: []uuid.UUID{other}})
	forked, err := svc.Fork(context.Background(), other, d.ID, "forked")
	require.NoError(t, err)
	assert.Equal(t, other, forked.OwnerID)
	assert.NotNil(t, forked.ParentDashboardID)
}

func Test_Service_Share_FiltersOwnerSelf(t *testing.T) {
	repo := newStubRepo()
	svc := NewService(repo, nil)
	owner := uuid.New()
	d, _ := svc.Create(context.Background(), owner, CreateDashboardRequest{Name: "test", Technology: TechLTE})

	// share to owner + other; owner 应被过滤
	other := uuid.New()
	require.NoError(t, svc.Share(context.Background(), owner, ShareRequest{
		DashboardID: d.ID, UserIDs: []uuid.UUID{owner, other},
	}))
	got, _ := repo.GetDashboard(context.Background(), d.ID)
	assert.NotContains(t, got.SharedWith, owner)
	assert.Contains(t, got.SharedWith, other)
}

func Test_Service_GetUserPreferences_ReturnsEmptyOnMiss(t *testing.T) {
	repo := newStubRepo()
	svc := NewService(repo, nil)
	user := uuid.New()

	p, err := svc.GetUserPreferences(context.Background(), user, TechLTE)
	require.NoError(t, err)
	require.NotNil(t, p)
	assert.Equal(t, user, p.UserID)
	assert.Equal(t, TechLTE, p.Technology)
}

// T-0164 收尾 G6-Gap-3：制式分键独立持久化
func Test_Service_UserPreferences_TechnologyIsolated(t *testing.T) {
	repo := newStubRepo()
	svc := NewService(repo, nil)
	user := uuid.New()

	lteLayout := []byte(`{"order":["rrc","erab"]}`)
	nrLayout := []byte(`{"order":["nr_sa","accessibility"]}`)
	require.NoError(t, svc.UpsertUserPreferences(context.Background(), &UserPreferences{
		UserID: user, Technology: TechLTE, KPICardLayout: lteLayout,
	}))
	require.NoError(t, svc.UpsertUserPreferences(context.Background(), &UserPreferences{
		UserID: user, Technology: TechNR, KPICardLayout: nrLayout,
	}))

	lte, _ := svc.GetUserPreferences(context.Background(), user, TechLTE)
	nr, _ := svc.GetUserPreferences(context.Background(), user, TechNR)
	assert.JSONEq(t, string(lteLayout), string(lte.KPICardLayout))
	assert.JSONEq(t, string(nrLayout), string(nr.KPICardLayout))

	// GSM 未设过 — 返默认空
	gsm, _ := svc.GetUserPreferences(context.Background(), user, TechGSM)
	assert.JSONEq(t, "{}", string(gsm.KPICardLayout))
}

func Test_Service_CreatePanel_NonOwnerDenied(t *testing.T) {
	repo := newStubRepo()
	svc := NewService(repo, nil)
	owner := uuid.New()
	other := uuid.New()
	d, _ := svc.Create(context.Background(), owner, CreateDashboardRequest{Name: "test", Technology: TechLTE})

	// 给 other share 读权限
	svc.Share(context.Background(), owner, ShareRequest{DashboardID: d.ID, UserIDs: []uuid.UUID{other}})

	// owner 可创建
	_, err := svc.CreatePanel(context.Background(), owner, CreatePanelRequest{
		DashboardID: d.ID, PanelType: PanelLineChart, Title: "p1",
		MetricPaths: []string{"M1"}, Granularities: []string{"hourly"}, Dimension: DimensionDevice,
	})
	require.NoError(t, err)

	// shared 用户不可创建（read-only）
	_, err = svc.CreatePanel(context.Background(), other, CreatePanelRequest{
		DashboardID: d.ID, PanelType: PanelLineChart, Title: "p2",
		MetricPaths: []string{"M1"}, Granularities: []string{"hourly"}, Dimension: DimensionDevice,
	})
	assert.True(t, errors.Is(err, ErrPermissionDenied))
}
