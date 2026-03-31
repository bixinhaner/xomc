package topology

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Mock: DeviceGroupRepository
// ---------------------------------------------------------------------------

type mockDeviceGroupRepo struct {
	groups  map[uuid.UUID]*DeviceGroup
	devices map[uuid.UUID][]uuid.UUID // groupID -> deviceIDs
}

func newMockDeviceGroupRepo() *mockDeviceGroupRepo {
	return &mockDeviceGroupRepo{
		groups:  make(map[uuid.UUID]*DeviceGroup),
		devices: make(map[uuid.UUID][]uuid.UUID),
	}
}

func (m *mockDeviceGroupRepo) Create(_ context.Context, group *DeviceGroup) error {
	if group.ID == uuid.Nil {
		group.ID = uuid.New()
	}
	now := time.Now()
	group.CreatedAt = now
	group.UpdatedAt = now
	m.groups[group.ID] = group
	return nil
}

func (m *mockDeviceGroupRepo) GetByID(_ context.Context, id uuid.UUID) (*DeviceGroup, error) {
	g, ok := m.groups[id]
	if !ok {
		return nil, errGroupNotFound
	}
	return g, nil
}

func (m *mockDeviceGroupRepo) Update(_ context.Context, group *DeviceGroup) error {
	if _, ok := m.groups[group.ID]; !ok {
		return errGroupNotFound
	}
	group.UpdatedAt = time.Now()
	m.groups[group.ID] = group
	return nil
}

func (m *mockDeviceGroupRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := m.groups[id]; !ok {
		return errGroupNotFound
	}
	delete(m.groups, id)
	return nil
}

func (m *mockDeviceGroupRepo) ListRoots(_ context.Context) ([]DeviceGroup, error) {
	var roots []DeviceGroup
	for _, g := range m.groups {
		if g.ParentID == nil {
			roots = append(roots, *g)
		}
	}
	return roots, nil
}

func (m *mockDeviceGroupRepo) ListChildren(_ context.Context, parentID uuid.UUID) ([]DeviceGroup, error) {
	var children []DeviceGroup
	for _, g := range m.groups {
		if g.ParentID != nil && *g.ParentID == parentID {
			children = append(children, *g)
		}
	}
	return children, nil
}

func (m *mockDeviceGroupRepo) GetTree(_ context.Context) ([]DeviceGroup, error) {
	flat := make([]DeviceGroup, 0, len(m.groups))
	for _, g := range m.groups {
		flat = append(flat, *g)
	}
	return flat, nil
}

func (m *mockDeviceGroupRepo) AddDevice(_ context.Context, groupID, deviceID uuid.UUID) error {
	m.devices[groupID] = append(m.devices[groupID], deviceID)
	return nil
}

func (m *mockDeviceGroupRepo) RemoveDevice(_ context.Context, groupID, deviceID uuid.UUID) error {
	ids := m.devices[groupID]
	for i, id := range ids {
		if id == deviceID {
			m.devices[groupID] = append(ids[:i], ids[i+1:]...)
			return nil
		}
	}
	return errDeviceNotInGroup
}

func (m *mockDeviceGroupRepo) ListDeviceIDs(_ context.Context, groupID uuid.UUID) ([]uuid.UUID, error) {
	return m.devices[groupID], nil
}

func (m *mockDeviceGroupRepo) GetTreeWithCounts(_ context.Context) ([]DeviceGroup, error) {
	return m.GetTree(context.Background())
}

func (m *mockDeviceGroupRepo) ExistsByParentAndName(_ context.Context, _ *uuid.UUID, _ string, _ *uuid.UUID) (bool, error) {
	return false, nil
}

func (m *mockDeviceGroupRepo) GetStats(_ context.Context) (*GroupStats, error) {
	return &GroupStats{}, nil
}

func (m *mockDeviceGroupRepo) CountDevicesByGroup(_ context.Context, groupID uuid.UUID) (int, error) {
	return len(m.devices[groupID]), nil
}

func (m *mockDeviceGroupRepo) ListChildIDs(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}

func (m *mockDeviceGroupRepo) BatchAddDevices(_ context.Context, groupID uuid.UUID, deviceIDs []uuid.UUID) (int64, error) {
	m.devices[groupID] = append(m.devices[groupID], deviceIDs...)
	return int64(len(deviceIDs)), nil
}

func (m *mockDeviceGroupRepo) BatchRemoveDevices(_ context.Context, groupID uuid.UUID, deviceIDs []uuid.UUID) (int64, error) {
	return 0, nil
}

func (m *mockDeviceGroupRepo) MoveDevices(_ context.Context, _ []uuid.UUID, _ uuid.UUID) (int64, error) {
	return 0, nil
}

func (m *mockDeviceGroupRepo) MoveGroupDevicesToDefault(_ context.Context, _ []uuid.UUID) (int64, error) {
	return 0, nil
}

// ---------------------------------------------------------------------------
// Mock: SiteRepository
// ---------------------------------------------------------------------------

type mockSiteRepo struct {
	sites map[uuid.UUID]*Site
}

func newMockSiteRepo() *mockSiteRepo {
	return &mockSiteRepo{sites: make(map[uuid.UUID]*Site)}
}

func (m *mockSiteRepo) Create(_ context.Context, site *Site) error {
	if site.ID == uuid.Nil {
		site.ID = uuid.New()
	}
	now := time.Now()
	site.CreatedAt = now
	site.UpdatedAt = now
	m.sites[site.ID] = site
	return nil
}

func (m *mockSiteRepo) GetByID(_ context.Context, id uuid.UUID) (*Site, error) {
	s, ok := m.sites[id]
	if !ok {
		return nil, errSiteNotFound
	}
	return s, nil
}

func (m *mockSiteRepo) List(_ context.Context, filter SiteFilter) (*model.ListResponse[Site], error) {
	items := make([]Site, 0, len(m.sites))
	for _, s := range m.sites {
		items = append(items, *s)
	}
	return model.NewListResponse(items, int64(len(items)), filter.Page, filter.PageSize), nil
}

func (m *mockSiteRepo) ListWithCoordinates(_ context.Context) ([]Site, error) {
	var sites []Site
	for _, s := range m.sites {
		if s.Longitude != nil && s.Latitude != nil {
			sites = append(sites, *s)
		}
	}
	return sites, nil
}

// ---------------------------------------------------------------------------
// Mock: TopoNodeRepository
// ---------------------------------------------------------------------------

type mockTopoNodeRepo struct {
	nodes []TopoNode
}

func newMockTopoNodeRepo() *mockTopoNodeRepo {
	return &mockTopoNodeRepo{}
}

func (m *mockTopoNodeRepo) List(_ context.Context, filter TopoNodeFilter) (*model.ListResponse[TopoNode], error) {
	return model.NewListResponse(m.nodes, int64(len(m.nodes)), filter.Page, filter.PageSize), nil
}

func (m *mockTopoNodeRepo) ListAll(_ context.Context, _ *uuid.UUID) ([]TopoNode, error) {
	return m.nodes, nil
}

// ---------------------------------------------------------------------------
// Mock: TopoEdgeRepository
// ---------------------------------------------------------------------------

type mockTopoEdgeRepo struct {
	edges []TopoEdge
}

func newMockTopoEdgeRepo() *mockTopoEdgeRepo {
	return &mockTopoEdgeRepo{}
}

func (m *mockTopoEdgeRepo) List(_ context.Context, filter TopoEdgeFilter) (*model.ListResponse[TopoEdge], error) {
	return model.NewListResponse(m.edges, int64(len(m.edges)), filter.Page, filter.PageSize), nil
}

func (m *mockTopoEdgeRepo) ListAll(_ context.Context) ([]TopoEdge, error) {
	return m.edges, nil
}

// ---------------------------------------------------------------------------
// Sentinel errors used by mocks
// ---------------------------------------------------------------------------

var (
	errGroupNotFound   = fmt.Errorf("group not found")
	errDeviceNotInGroup = fmt.Errorf("device not in group")
	errSiteNotFound    = fmt.Errorf("site not found")
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func setupRouter(h *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h.RegisterRoutes(r.Group("/api/v1"))
	return r
}

func newTestHandler() (*Handler, *mockDeviceGroupRepo, *mockSiteRepo, *mockTopoNodeRepo, *mockTopoEdgeRepo) {
	groupRepo := newMockDeviceGroupRepo()
	siteRepo := newMockSiteRepo()
	nodeRepo := newMockTopoNodeRepo()
	edgeRepo := newMockTopoEdgeRepo()
	logger := zap.NewNop()
	service := NewDeviceGroupService(groupRepo, logger)
	h := NewHandler(groupRepo, service, siteRepo, nodeRepo, edgeRepo)
	return h, groupRepo, siteRepo, nodeRepo, edgeRepo
}

func seedGroup(repo *mockDeviceGroupRepo, id uuid.UUID, name string, parentID *uuid.UUID) *DeviceGroup {
	now := time.Now()
	g := &DeviceGroup{
		ID:        id,
		Name:      name,
		ParentID:  parentID,
		SortOrder: 0,
		CreatedAt: now,
		UpdatedAt: now,
	}
	repo.groups[id] = g
	return g
}

func seedSite(repo *mockSiteRepo, id uuid.UUID, name string, lon, lat *float64) *Site {
	now := time.Now()
	s := &Site{
		ID:        id,
		Name:      name,
		Longitude: lon,
		Latitude:  lat,
		Status:    SiteActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	repo.sites[id] = s
	return s
}

func mustMarshal(t *testing.T, v interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestHandler_ListTree(t *testing.T) {
	h, groupRepo, _, _, _ := newTestHandler()
	router := setupRouter(h)

	rootID := uuid.New()
	childID := uuid.New()
	seedGroup(groupRepo, rootID, "Root", nil)
	seedGroup(groupRepo, childID, "Child", &rootID)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]json.RawMessage
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp, "items")
}

func TestHandler_Create(t *testing.T) {
	h, _, _, _, _ := newTestHandler()
	router := setupRouter(h)

	body := CreateGroupRequest{
		Name:    "New Group",
		Carrier: "cmcc",
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/groups", bytes.NewReader(mustMarshal(t, body)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp DeviceGroup
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "New Group", resp.Name)
	assert.Equal(t, model.CarrierCode("cmcc"), resp.Carrier)
	assert.NotEqual(t, uuid.Nil, resp.ID)
}

func TestHandler_Get(t *testing.T) {
	h, groupRepo, _, _, _ := newTestHandler()
	router := setupRouter(h)

	groupID := uuid.New()
	seedGroup(groupRepo, groupID, "Test Group", nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups/"+groupID.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp DeviceGroup
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, groupID, resp.ID)
	assert.Equal(t, "Test Group", resp.Name)
}

func TestHandler_Update(t *testing.T) {
	h, groupRepo, _, _, _ := newTestHandler()
	router := setupRouter(h)

	groupID := uuid.New()
	seedGroup(groupRepo, groupID, "Old Name", nil)

	updatedName := "Updated Name"
	body := UpdateGroupRequest{
		Name: &updatedName,
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/groups/"+groupID.String(), bytes.NewReader(mustMarshal(t, body)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp DeviceGroup
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", resp.Name)
}

func TestHandler_Delete(t *testing.T) {
	h, groupRepo, _, _, _ := newTestHandler()
	router := setupRouter(h)

	groupID := uuid.New()
	seedGroup(groupRepo, groupID, "ToDelete", nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/groups/"+groupID.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

	// Verify group removed.
	_, exists := groupRepo.groups[groupID]
	assert.False(t, exists)
}

func TestHandler_AddDevice(t *testing.T) {
	h, groupRepo, _, _, _ := newTestHandler()
	router := setupRouter(h)

	groupID := uuid.New()
	seedGroup(groupRepo, groupID, "Group1", nil)
	deviceID := uuid.New()

	body := map[string]string{"device_id": deviceID.String()}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/groups/"+groupID.String()+"/devices", bytes.NewReader(mustMarshal(t, body)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	// Verify device is in the group.
	ids := groupRepo.devices[groupID]
	assert.Contains(t, ids, deviceID)
}

func TestHandler_RemoveDevice(t *testing.T) {
	h, groupRepo, _, _, _ := newTestHandler()
	router := setupRouter(h)

	groupID := uuid.New()
	deviceID := uuid.New()
	seedGroup(groupRepo, groupID, "Group1", nil)
	groupRepo.devices[groupID] = []uuid.UUID{deviceID}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/groups/"+groupID.String()+"/devices/"+deviceID.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

	// Verify device removed from the group.
	ids := groupRepo.devices[groupID]
	assert.NotContains(t, ids, deviceID)
}

func TestHandler_ListSites(t *testing.T) {
	h, _, siteRepo, _, _ := newTestHandler()
	router := setupRouter(h)

	lon := 116.4074
	lat := 39.9042
	seedSite(siteRepo, uuid.New(), "Site-A", &lon, &lat)
	seedSite(siteRepo, uuid.New(), "Site-B", &lon, &lat)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sites?page=1&page_size=20", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.ListResponse[Site]
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, int64(2), resp.Total)
	assert.Len(t, resp.Items, 2)
}

func TestHandler_ListTopoNodes(t *testing.T) {
	h, _, _, nodeRepo, _ := newTestHandler()
	router := setupRouter(h)

	nodeRepo.nodes = []TopoNode{
		{ID: uuid.New(), Label: "eNB-001", NodeType: "enb", Status: NodeOnline},
		{ID: uuid.New(), Label: "GW-001", NodeType: "gateway", Status: NodeOnline},
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/topology/nodes?page=1&page_size=20", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.ListResponse[TopoNode]
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, int64(2), resp.Total)
	assert.Len(t, resp.Items, 2)
}
