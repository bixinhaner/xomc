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
	"github.com/omcgo/omcgo/internal/admin"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func commonErrInvalidInput() error  { return commonerrors.ErrInvalidInput }
func commonErrAlreadyExists() error { return commonerrors.ErrAlreadyExists }

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

func (m *mockDeviceGroupRepo) AddDeviceWithSource(_ context.Context, groupID, deviceID uuid.UUID, _ string, _ *uuid.UUID) (int64, error) {
	m.devices[groupID] = append(m.devices[groupID], deviceID)
	return 1, nil
}

func (m *mockDeviceGroupRepo) MoveDeviceAutoMatched(_ context.Context, _, groupID, deviceID uuid.UUID) (int64, error) {
	m.devices[groupID] = append(m.devices[groupID], deviceID)
	return 1, nil
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

func (m *mockDeviceGroupRepo) RemoveDevicesFromAllGroups(_ context.Context, deviceIDs []uuid.UUID) (int64, error) {
	rm := make(map[uuid.UUID]struct{}, len(deviceIDs))
	for _, id := range deviceIDs {
		rm[id] = struct{}{}
	}
	var removed int64
	for gid, devs := range m.devices {
		kept := devs[:0:0]
		for _, d := range devs {
			if _, drop := rm[d]; drop {
				removed++
				continue
			}
			kept = append(kept, d)
		}
		m.devices[gid] = kept
	}
	return removed, nil
}

func (m *mockDeviceGroupRepo) MoveGroupDevicesToDefault(_ context.Context, _ []uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *mockDeviceGroupRepo) ClearBoundRule(_ context.Context, _ uuid.UUID) error {
	return nil
}
func (m *mockDeviceGroupRepo) UpdateBoundRule(_ context.Context, _, _ uuid.UUID) error {
	return nil
}

// ---------------------------------------------------------------------------
// Mock: SiteRepository
// ---------------------------------------------------------------------------

type mockSiteRepo struct {
	sites     map[uuid.UUID]*Site
	createErr error // when set, Create returns this error without persisting
}

func newMockSiteRepo() *mockSiteRepo {
	return &mockSiteRepo{sites: make(map[uuid.UUID]*Site)}
}

func (m *mockSiteRepo) Create(_ context.Context, site *Site) error {
	if m.createErr != nil {
		return m.createErr
	}
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
	nodes map[uuid.UUID]TopoNode
}

func newMockTopoNodeRepo() *mockTopoNodeRepo {
	return &mockTopoNodeRepo{nodes: make(map[uuid.UUID]TopoNode)}
}

func (m *mockTopoNodeRepo) Create(_ context.Context, node *TopoNode) error {
	if node.ID == uuid.Nil {
		node.ID = uuid.New()
	}
	now := time.Now()
	node.CreatedAt = now
	node.UpdatedAt = now
	m.nodes[node.ID] = *node
	return nil
}

func (m *mockTopoNodeRepo) GetByID(_ context.Context, id uuid.UUID) (*TopoNode, error) {
	n, ok := m.nodes[id]
	if !ok {
		return nil, commonerrors.ErrNotFound
	}
	return &n, nil
}

func (m *mockTopoNodeRepo) Update(_ context.Context, node *TopoNode) error {
	if _, ok := m.nodes[node.ID]; !ok {
		return commonerrors.ErrNotFound
	}
	node.UpdatedAt = time.Now()
	m.nodes[node.ID] = *node
	return nil
}

func (m *mockTopoNodeRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := m.nodes[id]; !ok {
		return commonerrors.ErrNotFound
	}
	delete(m.nodes, id)
	return nil
}

func (m *mockTopoNodeRepo) List(_ context.Context, filter TopoNodeFilter) (*model.ListResponse[TopoNode], error) {
	items := make([]TopoNode, 0, len(m.nodes))
	for _, n := range m.nodes {
		items = append(items, n)
	}
	return model.NewListResponse(items, int64(len(items)), filter.Page, filter.PageSize), nil
}

func (m *mockTopoNodeRepo) ListAll(_ context.Context, _ *uuid.UUID, _ *string, _ *string, _ int) ([]TopoNode, error) {
	items := make([]TopoNode, 0, len(m.nodes))
	for _, n := range m.nodes {
		items = append(items, n)
	}
	return items, nil
}

// ---------------------------------------------------------------------------
// Mock: TopoEdgeRepository
// ---------------------------------------------------------------------------

type mockTopoEdgeRepo struct {
	edges map[uuid.UUID]TopoEdge
}

func newMockTopoEdgeRepo() *mockTopoEdgeRepo {
	return &mockTopoEdgeRepo{edges: make(map[uuid.UUID]TopoEdge)}
}

func (m *mockTopoEdgeRepo) Create(_ context.Context, edge *TopoEdge) error {
	if edge.ID == uuid.Nil {
		edge.ID = uuid.New()
	}
	edge.CreatedAt = time.Now()
	m.edges[edge.ID] = *edge
	return nil
}

func (m *mockTopoEdgeRepo) GetByID(_ context.Context, id uuid.UUID) (*TopoEdge, error) {
	e, ok := m.edges[id]
	if !ok {
		return nil, commonerrors.ErrNotFound
	}
	return &e, nil
}

func (m *mockTopoEdgeRepo) Update(_ context.Context, edge *TopoEdge) error {
	if _, ok := m.edges[edge.ID]; !ok {
		return commonerrors.ErrNotFound
	}
	m.edges[edge.ID] = *edge
	return nil
}

func (m *mockTopoEdgeRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := m.edges[id]; !ok {
		return commonerrors.ErrNotFound
	}
	delete(m.edges, id)
	return nil
}

func (m *mockTopoEdgeRepo) List(_ context.Context, filter TopoEdgeFilter) (*model.ListResponse[TopoEdge], error) {
	items := make([]TopoEdge, 0, len(m.edges))
	for _, e := range m.edges {
		items = append(items, e)
	}
	return model.NewListResponse(items, int64(len(items)), filter.Page, filter.PageSize), nil
}

func (m *mockTopoEdgeRepo) ListAll(_ context.Context) ([]TopoEdge, error) {
	items := make([]TopoEdge, 0, len(m.edges))
	for _, e := range m.edges {
		items = append(items, e)
	}
	return items, nil
}

func (m *mockTopoEdgeRepo) ListByNodeIDs(_ context.Context, nodeIDs []uuid.UUID) ([]TopoEdge, error) {
	if len(nodeIDs) == 0 {
		return []TopoEdge{}, nil
	}
	nodeIDSet := make(map[uuid.UUID]bool)
	for _, id := range nodeIDs {
		nodeIDSet[id] = true
	}
	var items []TopoEdge
	for _, e := range m.edges {
		if nodeIDSet[e.SourceID] || nodeIDSet[e.TargetID] {
			items = append(items, e)
		}
	}
	if items == nil {
		items = []TopoEdge{}
	}
	return items, nil
}

// ---------------------------------------------------------------------------
// Sentinel errors used by mocks
// ---------------------------------------------------------------------------

var (
	errGroupNotFound    = fmt.Errorf("group not found")
	errDeviceNotInGroup = fmt.Errorf("device not in group")
	errSiteNotFound     = fmt.Errorf("site not found")
)

// fkLikeErr returns an error that wraps the ErrInvalidInput sentinel; this is
// what the site repository should yield after classifying a PG 23503 (foreign
// key violation), so the handler maps it to 400.
func fkLikeErr() error {
	return fmt.Errorf("%w: foreign key violation simulated",
		commonErrInvalidInput())
}

// uniqueLikeErr returns an error wrapping ErrAlreadyExists, simulating PG 23505.
func uniqueLikeErr() error {
	return fmt.Errorf("%w: unique violation simulated",
		commonErrAlreadyExists())
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func setupRouter(h *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h.RegisterRoutes(r.Group("/api/v1"))
	return r
}

func setupRouterWithAuth(h *Handler, userID uuid.UUID, isSuperAdmin bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(admin.CtxKeyUserID, userID)
		c.Set(admin.CtxKeyIsSuperAdmin, isSuperAdmin)
		c.Next()
	})
	h.RegisterRoutes(r.Group("/api/v1"))
	return r
}

type mockVisibleGroupsResolver struct {
	visibleGroups []uuid.UUID
	err           error
}

func (m *mockVisibleGroupsResolver) GetUserVisibleGroupIDs(_ context.Context, _ uuid.UUID, _ bool) ([]uuid.UUID, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.visibleGroups, nil
}

func newTestHandler() (*Handler, *mockDeviceGroupRepo, *mockSiteRepo, *mockTopoNodeRepo, *mockTopoEdgeRepo) {
	groupRepo := newMockDeviceGroupRepo()
	siteRepo := newMockSiteRepo()
	nodeRepo := newMockTopoNodeRepo()
	edgeRepo := newMockTopoEdgeRepo()
	logger := zap.NewNop()
	service := NewDeviceGroupService(groupRepo, nodeRepo, nil, logger)
	h := NewHandler(groupRepo, service, siteRepo, nodeRepo, edgeRepo, nil, logger)
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
	response.DecodeData(t, w.Body, &resp)
	assert.Contains(t, resp, "items")
}

func TestHandler_GetTreeWithCounts_FiltersVisibleGroups(t *testing.T) {
	h, groupRepo, _, _, _ := newTestHandler()
	visibleChildID := uuid.New()
	h.SetPermissionService(&mockVisibleGroupsResolver{visibleGroups: []uuid.UUID{visibleChildID}})
	router := setupRouterWithAuth(h, uuid.New(), false)

	rootID := uuid.New()
	hiddenChildID := uuid.New()
	otherRootID := uuid.New()
	otherChildID := uuid.New()

	seedGroup(groupRepo, rootID, "Root", nil)
	seedGroup(groupRepo, visibleChildID, "Visible Child", &rootID).DeviceCount = 3
	seedGroup(groupRepo, hiddenChildID, "Hidden Child", &rootID).DeviceCount = 5
	seedGroup(groupRepo, otherRootID, "Other Root", nil)
	seedGroup(groupRepo, otherChildID, "Other Child", &otherRootID).DeviceCount = 7

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/device-groups/tree", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp TreeResponse
	response.DecodeData(t, w.Body, &resp)
	require.Len(t, resp.Items, 1)
	assert.Equal(t, rootID, resp.Items[0].ID)
	assert.Equal(t, 3, resp.Items[0].DeviceCount)
	require.Len(t, resp.Items[0].Children, 1)
	assert.Equal(t, visibleChildID, resp.Items[0].Children[0].ID)
	assert.Equal(t, 3, resp.Items[0].Children[0].DeviceCount)
	require.NotNil(t, resp.Stats)
	assert.Equal(t, 2, resp.Stats.TotalGroups)
	assert.Equal(t, 3, resp.Stats.GroupedDevices)
	assert.Equal(t, 0, resp.Stats.UngroupedDevices)
}

func TestHandler_GetTreeWithCounts_VisibleParentDoesNotExposeHiddenChildCounts(t *testing.T) {
	h, groupRepo, _, _, _ := newTestHandler()
	rootID := uuid.New()
	hiddenChildID := uuid.New()
	h.SetPermissionService(&mockVisibleGroupsResolver{visibleGroups: []uuid.UUID{rootID}})
	router := setupRouterWithAuth(h, uuid.New(), false)

	root := seedGroup(groupRepo, rootID, "Root", nil)
	root.Level = 1
	root.DeviceCount = 5
	hiddenChild := seedGroup(groupRepo, hiddenChildID, "Hidden Child", &rootID)
	hiddenChild.Level = 2
	hiddenChild.DeviceCount = 5

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/device-groups/tree", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp TreeResponse
	response.DecodeData(t, w.Body, &resp)
	require.Len(t, resp.Items, 1)
	assert.Equal(t, rootID, resp.Items[0].ID)
	assert.Empty(t, resp.Items[0].Children)
	assert.Zero(t, resp.Items[0].DeviceCount)
	require.NotNil(t, resp.Stats)
	assert.Equal(t, 1, resp.Stats.TotalGroups)
	assert.Zero(t, resp.Stats.GroupedDevices)
	assert.Zero(t, resp.Stats.UngroupedDevices)
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
	response.DecodeData(t, w.Body, &resp)
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
	response.DecodeData(t, w.Body, &resp)
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
	response.DecodeData(t, w.Body, &resp)
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

	assert.Equal(t, http.StatusOK, w.Code)
	response.DecodeData(t, w.Body, nil)

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

	assert.Equal(t, http.StatusOK, w.Code)
	response.DecodeData(t, w.Body, nil)

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
	response.DecodeData(t, w.Body, &resp)
	assert.Equal(t, int64(2), resp.Total)
	assert.Len(t, resp.Items, 2)
}

// TestHandler_CreateSite_Success exercises the happy path: 201 + persisted record.
func TestHandler_CreateSite_Success(t *testing.T) {
	h, _, siteRepo, _, _ := newTestHandler()
	router := setupRouter(h)

	body := map[string]interface{}{
		"name":    "Site-OK",
		"address": "addr",
		"status":  "active",
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sites",
		bytes.NewReader(mustMarshal(t, body)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	assert.Len(t, siteRepo.sites, 1)
}

// TestHandler_CreateSite_BadJSON guards the input-validation branch (binding error → 400).
func TestHandler_CreateSite_BadJSON(t *testing.T) {
	h, _, _, _, _ := newTestHandler()
	router := setupRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sites",
		bytes.NewReader([]byte(`{not-json`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestHandler_CreateSite_InvalidDomainID guards the uuid.Parse branch.
func TestHandler_CreateSite_InvalidDomainID(t *testing.T) {
	h, _, _, _, _ := newTestHandler()
	router := setupRouter(h)

	body := map[string]interface{}{
		"name":      "Site-Bad-Domain",
		"domain_id": "not-a-uuid",
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sites",
		bytes.NewReader(mustMarshal(t, body)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestHandler_CreateSite_RepoErrors verifies that sentinel errors from the
// repository (which represent constraint violations from PostgreSQL) are
// translated into 4xx HTTP statuses by the handler instead of 500.
//
// This is the regression test for T-0057 / W2.D.1.b real-bug #7 (POST /sites
// returned 500 when domain_id violated the FK to device_groups).
func TestHandler_CreateSite_RepoErrors(t *testing.T) {
	commonerrPkg := "github.com/omcgo/omcgo/internal/core/errors"
	_ = commonerrPkg // documentation only

	cases := []struct {
		name       string
		repoErr    error
		wantStatus int
	}{
		{
			name:       "fk violation maps to 400",
			repoErr:    fmt.Errorf("simulated fk violation: %w", fkLikeErr()),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "unique violation maps to 409",
			repoErr:    fmt.Errorf("simulated unique violation: %w", uniqueLikeErr()),
			wantStatus: http.StatusConflict,
		},
		{
			name:       "unmapped error stays 500",
			repoErr:    fmt.Errorf("some unrelated DB failure"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			h, _, siteRepo, _, _ := newTestHandler()
			siteRepo.createErr = tc.repoErr
			router := setupRouter(h)

			body := map[string]interface{}{
				"name":   "Site-X",
				"status": "active",
			}

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/sites",
				bytes.NewReader(mustMarshal(t, body)))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.wantStatus, w.Code,
				"repo err %v should produce HTTP %d but got %d", tc.repoErr, tc.wantStatus, w.Code)
		})
	}
}

func TestHandler_ListTopoNodes(t *testing.T) {
	h, _, _, nodeRepo, _ := newTestHandler()
	router := setupRouter(h)

	node1 := TopoNode{ID: uuid.New(), Label: "eNB-001", NodeType: "enb", Status: NodeOnline}
	node2 := TopoNode{ID: uuid.New(), Label: "GW-001", NodeType: "gateway", Status: NodeOnline}
	nodeRepo.nodes[node1.ID] = node1
	nodeRepo.nodes[node2.ID] = node2

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/topology/nodes?page=1&page_size=20", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.ListResponse[TopoNode]
	response.DecodeData(t, w.Body, &resp)
	assert.Equal(t, int64(2), resp.Total)
	assert.Len(t, resp.Items, 2)
}

// ---------------------------------------------------------------------------
// Tests: TopoNode CRUD
// ---------------------------------------------------------------------------

func TestHandler_CreateTopoNode_Success(t *testing.T) {
	h, _, _, nodeRepo, _ := newTestHandler()
	router := setupRouter(h)

	body := map[string]interface{}{
		"label":     "eNB-Test-001",
		"node_type": "eNB",
		"x":         100.5,
		"y":         200.3,
		"status":    "online",
		"device_sn": "SN12345",
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/topology/nodes",
		bytes.NewReader(mustMarshal(t, body)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)

	var resp TopoNode
	response.DecodeData(t, w.Body, &resp)
	assert.Equal(t, "eNB-Test-001", resp.Label)
	assert.Equal(t, "eNB", resp.NodeType)
	assert.Equal(t, 100.5, resp.X)
	assert.Equal(t, 200.3, resp.Y)
	assert.Equal(t, NodeOnline, resp.Status)
	assert.Equal(t, "SN12345", resp.DeviceSN)
	assert.NotEqual(t, uuid.Nil, resp.ID)
	assert.Len(t, nodeRepo.nodes, 1)
}

func TestHandler_CreateTopoNode_ValidationError(t *testing.T) {
	h, _, _, _, _ := newTestHandler()
	router := setupRouter(h)

	cases := []struct {
		name       string
		body       map[string]interface{}
		wantStatus int
	}{
		{
			name: "missing label",
			body: map[string]interface{}{
				"node_type": "eNB",
				"x":         100,
				"y":         200,
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "missing node_type",
			body: map[string]interface{}{
				"label": "eNB-001",
				"x":     100,
				"y":     200,
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "invalid node_type",
			body: map[string]interface{}{
				"label":     "eNB-001",
				"node_type": "INVALID",
				"x":         100,
				"y":         200,
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "invalid status",
			body: map[string]interface{}{
				"label":     "eNB-001",
				"node_type": "eNB",
				"status":    "invalid",
				"x":         100,
				"y":         200,
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/topology/nodes",
				bytes.NewReader(mustMarshal(t, tc.body)))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.wantStatus, w.Code)
		})
	}
}

func TestHandler_CreateTopoNode_InvalidSiteID(t *testing.T) {
	h, _, _, _, _ := newTestHandler()
	router := setupRouter(h)

	body := map[string]interface{}{
		"label":     "eNB-001",
		"node_type": "eNB",
		"x":         100,
		"y":         200,
		"site_id":   "not-a-uuid",
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/topology/nodes",
		bytes.NewReader(mustMarshal(t, body)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetTopoNode_Success(t *testing.T) {
	h, _, _, nodeRepo, _ := newTestHandler()
	router := setupRouter(h)

	nodeID := uuid.New()
	node := TopoNode{
		ID:       nodeID,
		Label:    "eNB-001",
		NodeType: "eNB",
		X:        100,
		Y:        200,
		Status:   NodeOnline,
	}
	nodeRepo.nodes[nodeID] = node

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/topology/nodes/"+nodeID.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp TopoNode
	response.DecodeData(t, w.Body, &resp)
	assert.Equal(t, nodeID, resp.ID)
	assert.Equal(t, "eNB-001", resp.Label)
}

func TestHandler_GetTopoNode_NotFound(t *testing.T) {
	h, _, _, _, _ := newTestHandler()
	router := setupRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/topology/nodes/"+uuid.New().String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_UpdateTopoNode_Success(t *testing.T) {
	h, _, _, nodeRepo, _ := newTestHandler()
	router := setupRouter(h)

	nodeID := uuid.New()
	node := TopoNode{
		ID:       nodeID,
		Label:    "eNB-001",
		NodeType: "eNB",
		X:        100,
		Y:        200,
		Status:   NodeOnline,
	}
	nodeRepo.nodes[nodeID] = node

	body := map[string]interface{}{
		"label":  "eNB-001-Updated",
		"x":      150.5,
		"y":      250.3,
		"status": "alarm",
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/topology/nodes/"+nodeID.String(),
		bytes.NewReader(mustMarshal(t, body)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp TopoNode
	response.DecodeData(t, w.Body, &resp)
	assert.Equal(t, "eNB-001-Updated", resp.Label)
	assert.Equal(t, 150.5, resp.X)
	assert.Equal(t, 250.3, resp.Y)
	assert.Equal(t, NodeAlarm, resp.Status)
}

func TestHandler_DeleteTopoNode_Success(t *testing.T) {
	h, _, _, nodeRepo, _ := newTestHandler()
	router := setupRouter(h)

	nodeID := uuid.New()
	node := TopoNode{
		ID:       nodeID,
		Label:    "eNB-001",
		NodeType: "eNB",
		Status:   NodeOnline,
	}
	nodeRepo.nodes[nodeID] = node

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/topology/nodes/"+nodeID.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, nodeRepo.nodes)
}

// ---------------------------------------------------------------------------
// Tests: TopoEdge CRUD
// ---------------------------------------------------------------------------

func TestHandler_CreateTopoEdge_Success(t *testing.T) {
	h, _, _, _, edgeRepo := newTestHandler()
	router := setupRouter(h)

	sourceID := uuid.New()
	targetID := uuid.New()

	body := map[string]interface{}{
		"source_id": sourceID.String(),
		"target_id": targetID.String(),
		"label":     "S1-C",
		"status":    "active",
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/topology/edges",
		bytes.NewReader(mustMarshal(t, body)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)

	var resp TopoEdge
	response.DecodeData(t, w.Body, &resp)
	assert.Equal(t, sourceID, resp.SourceID)
	assert.Equal(t, targetID, resp.TargetID)
	assert.Equal(t, "S1-C", resp.Label)
	assert.Equal(t, EdgeActive, resp.Status)
	assert.NotEqual(t, uuid.Nil, resp.ID)
	assert.Len(t, edgeRepo.edges, 1)
}

func TestHandler_CreateTopoEdge_ValidationError(t *testing.T) {
	h, _, _, _, _ := newTestHandler()
	router := setupRouter(h)

	cases := []struct {
		name       string
		body       map[string]interface{}
		wantStatus int
	}{
		{
			name: "missing source_id",
			body: map[string]interface{}{
				"target_id": uuid.New().String(),
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "missing target_id",
			body: map[string]interface{}{
				"source_id": uuid.New().String(),
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "invalid source_id",
			body: map[string]interface{}{
				"source_id": "not-a-uuid",
				"target_id": uuid.New().String(),
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "invalid target_id",
			body: map[string]interface{}{
				"source_id": uuid.New().String(),
				"target_id": "not-a-uuid",
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "invalid status",
			body: map[string]interface{}{
				"source_id": uuid.New().String(),
				"target_id": uuid.New().String(),
				"status":    "invalid",
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/topology/edges",
				bytes.NewReader(mustMarshal(t, tc.body)))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.wantStatus, w.Code)
		})
	}
}

func TestHandler_GetTopoEdge_Success(t *testing.T) {
	h, _, _, _, edgeRepo := newTestHandler()
	router := setupRouter(h)

	edgeID := uuid.New()
	sourceID := uuid.New()
	targetID := uuid.New()
	edge := TopoEdge{
		ID:       edgeID,
		SourceID: sourceID,
		TargetID: targetID,
		Label:    "S1-C",
		Status:   EdgeActive,
	}
	edgeRepo.edges[edgeID] = edge

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/topology/edges/"+edgeID.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp TopoEdge
	response.DecodeData(t, w.Body, &resp)
	assert.Equal(t, edgeID, resp.ID)
	assert.Equal(t, "S1-C", resp.Label)
}

func TestHandler_UpdateTopoEdge_Success(t *testing.T) {
	h, _, _, _, edgeRepo := newTestHandler()
	router := setupRouter(h)

	edgeID := uuid.New()
	sourceID := uuid.New()
	targetID := uuid.New()
	edge := TopoEdge{
		ID:       edgeID,
		SourceID: sourceID,
		TargetID: targetID,
		Label:    "S1-C",
		Status:   EdgeActive,
	}
	edgeRepo.edges[edgeID] = edge

	body := map[string]interface{}{
		"label":  "S1-U",
		"status": "degraded",
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/topology/edges/"+edgeID.String(),
		bytes.NewReader(mustMarshal(t, body)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp TopoEdge
	response.DecodeData(t, w.Body, &resp)
	assert.Equal(t, "S1-U", resp.Label)
	assert.Equal(t, EdgeDegraded, resp.Status)
}

func TestHandler_DeleteTopoEdge_Success(t *testing.T) {
	h, _, _, _, edgeRepo := newTestHandler()
	router := setupRouter(h)

	edgeID := uuid.New()
	edge := TopoEdge{
		ID:       edgeID,
		SourceID: uuid.New(),
		TargetID: uuid.New(),
		Status:   EdgeActive,
	}
	edgeRepo.edges[edgeID] = edge

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/topology/edges/"+edgeID.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, edgeRepo.edges)
}

func TestHandler_DeleteTopoEdge_NotFound(t *testing.T) {
	h, _, _, _, _ := newTestHandler()
	router := setupRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/topology/edges/"+uuid.New().String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
