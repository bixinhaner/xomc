package deviceaccess

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type actorResolverStub struct {
	actor PolicyActor
}

type managementStoreStub struct{}

func (managementStoreStub) ListStates(context.Context, ManagementFilter) ([]AccessStateItem, int64, error) {
	return []AccessStateItem{}, 0, nil
}

func (managementStoreStub) GetDetail(context.Context, ManagementFilter) (AccessDetail, error) {
	return AccessDetail{}, nil
}

func (managementStoreStub) ListPolicyVersions(context.Context, ManagementFilter) ([]PolicyVersionSummary, int64, error) {
	return nil, 0, nil
}

func (managementStoreStub) ListEntries(context.Context, ManagementFilter) ([]AccessListItem, int64, error) {
	return nil, 0, nil
}

func (managementStoreStub) ListCandidates(context.Context, ManagementFilter) ([]CandidateItem, int64, error) {
	return nil, 0, nil
}

func (managementStoreStub) ReviewCandidate(context.Context, string, uuid.UUID, uuid.UUID, []uuid.UUID, ListEntryType, string) error {
	return nil
}

func (s actorResolverStub) ResolveActor(*gin.Context) (PolicyActor, error) {
	return s.actor, nil
}

func TestPolicyHTTPHandlerCreateDraftUsesActorCarrier(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &policyStoreStub{}
	service := NewPolicyService(store, nil)
	handler := NewPolicyHTTPHandler(service, nil, actorResolverStub{
		actor: PolicyActor{Carrier: "cmcc", SubjectID: "user-1"},
	})
	router := gin.New()
	router.POST("/policies", handler.CreateDraft)

	body := `{"name":"draft","carrier":"ctcc","policy":{"rules":[{"id":"rule-1","enabled":true,"serial_scope":{"type":"all"},"conditions":[{"id":"tac-1","type":"tac","operator":"equal","expected":"100","required":true}]}]}}`
	req := httptest.NewRequest(http.MethodPost, "/policies", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)

	require.Equal(t, http.StatusCreated, response.Code)
	require.Equal(t, "cmcc", store.created.Carrier)
}

func TestApplyManagementAuditQueryMapsExternalRuleDimensionNames(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, test := range []struct {
		raw  string
		want ConditionType
	}{
		{raw: "sn", want: ConditionTypeIdentity},
		{raw: "ip", want: ConditionTypeObservedIP},
		{raw: "gps", want: ConditionTypeGPS},
	} {
		t.Run(test.raw, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/?dimension="+test.raw, nil)
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = request
			var filter ManagementFilter

			require.NoError(t, applyManagementAuditQuery(ctx, &filter))
			require.Equal(t, test.want, filter.Dimension)
		})
	}
}

func TestApplyManagementAuditQueryRejectsInvalidUUIDAndTime(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, rawQuery := range []string{
		"policy_version_id=not-a-uuid",
		"matched_rule_id=not-a-uuid",
		"started_at=not-a-time",
		"ended_at=not-a-time",
	} {
		t.Run(rawQuery, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/?"+rawQuery, nil)
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = request

			require.Error(t, applyManagementAuditQuery(ctx, &ManagementFilter{}))
		})
	}
}

func TestPolicyHTTPHandlerRejectsMissingActorResolver(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewPolicyHTTPHandler(NewPolicyService(&policyStoreStub{}, nil), nil, nil)
	router := gin.New()
	router.POST("/policies", handler.CreateDraft)
	body := `{"name":"draft","policy":{"rules":[]}}`
	req := httptest.NewRequest(http.MethodPost, "/policies", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	require.Equal(t, http.StatusInternalServerError, response.Code)
}

func TestPolicyHTTPHandlerReturnsBadRequestForInvalidPolicy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewPolicyHTTPHandler(NewPolicyService(&policyStoreStub{}, nil), nil, actorResolverStub{
		actor: PolicyActor{Carrier: "cmcc", SubjectID: "user-1"},
	})
	router := gin.New()
	router.POST("/policies", handler.CreateDraft)
	req := httptest.NewRequest(http.MethodPost, "/policies", strings.NewReader(`{"name":"draft","policy":{"default_action":"accept","rules":[]}}`))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)

	require.Equal(t, http.StatusBadRequest, response.Code)
}

func TestPolicyHTTPHandlerAcceptsMultiSerialListMutation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &listStoreStub{}
	handler := NewPolicyHTTPHandler(nil, NewListService(store), actorResolverStub{
		actor: PolicyActor{Carrier: "cmcc", SubjectID: "user-1"},
	})
	router := gin.New()
	router.POST("/access-list", handler.UpsertList)
	body := `{"entries":[{"type":"deny","identity_type":"serial_number","identity_value":"SN-1"},{"type":"deny","identity_type":"serial_number","identity_value":"SN-2"}]}`
	req := httptest.NewRequest(http.MethodPost, "/access-list", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, req)

	require.Equal(t, http.StatusAccepted, response.Code)
	require.Len(t, store.entries, 2)
	require.Equal(t, "cmcc", store.carrier)
}

func TestManagementFilterPreservesSupportedServerSideFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest(http.MethodGet,
		"/device-access/access-list?page=2&page_size=50&serial_number=SN-1&product_name=QRTB&state=rejected&status=active&entry_type=deny", nil)

	filter := managementFilter(context, PolicyActor{Carrier: "cmcc", VisibleGroups: []uuid.UUID{uuid.MustParse("00000000-0000-0000-0000-000000000001")}})

	require.Equal(t, "cmcc", filter.Carrier)
	require.Equal(t, "SN-1", filter.SerialNumber)
	require.Equal(t, "QRTB", filter.ProductName)
	require.Equal(t, AccessStateRejected, filter.State)
	require.Equal(t, "active", filter.Status)
	require.Equal(t, ListEntryTypeDeny, filter.EntryType)
	require.Equal(t, 2, filter.Page)
	require.Equal(t, 50, filter.PageSize)
	require.Len(t, filter.VisibleGroups, 1)
}

func TestPolicyHTTPHandlerRegisterRoutesUsesNonConflictingResourcePaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewPolicyHTTPHandler(nil, nil, nil)
	router := gin.New()

	require.NotPanics(t, func() {
		handler.RegisterRoutes(router.Group("/api/v1"))
	})

	paths := make([]string, 0, len(router.Routes()))
	for _, route := range router.Routes() {
		paths = append(paths, route.Method+" "+route.Path)
	}
	sort.Strings(paths)
	require.Equal(t, []string{
		"DELETE /api/v1/device-access/policies/:versionID",
		"GET /api/v1/device-access/access-list/template",
		"GET /api/v1/device-access/import-templates/:importType",
		"GET /api/v1/device-access/policies/:versionID",
		"GET /api/v1/device-access/policies/:versionID/difference",
		"POST /api/v1/device-access/access-list",
		"POST /api/v1/device-access/access-list/batch-disable",
		"POST /api/v1/device-access/devices/:serialNumber/reevaluate",
		"POST /api/v1/device-access/policies/:versionID/publish",
		"POST /api/v1/device-access/policies/:versionID/rollback",
		"POST /api/v1/device-access/policies/drafts",
		"PUT /api/v1/device-access/policies/:versionID",
	}, paths)
}

func TestPolicyHTTPHandlerDownloadsReusableListTemplate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewPolicyHTTPHandler(nil, nil, actorResolverStub{
		actor: PolicyActor{Carrier: "cmcc", SubjectID: "user-1"},
	})
	router := gin.New()
	router.GET("/access-list/template", handler.DownloadListTemplate)
	req := httptest.NewRequest(http.MethodGet, "/access-list/template", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, req)

	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", response.Header().Get("Content-Type"))
	require.Contains(t, response.Header().Get("Content-Disposition"), "device-access-list-template.xlsx")
	records, err := readImportRecords(response.Body.Bytes(), "template.xlsx")
	require.NoError(t, err)
	require.Equal(t, [][]string{accessListImportHeaders}, records)
}

func TestPolicyHTTPHandlerBatchDisableUsesActorCarrier(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &listStoreStub{}
	handler := NewPolicyHTTPHandler(nil, NewListService(store), actorResolverStub{
		actor: PolicyActor{Carrier: "cmcc", SubjectID: "user-1"},
	})
	router := gin.New()
	router.POST("/access-list/batch-disable", handler.DisableListBatch)
	body := `{"carrier":"ctcc","entry_type":"deny","serial_numbers":["SN-1","SN-2"],"reason":"expired"}`
	req := httptest.NewRequest(http.MethodPost, "/access-list/batch-disable", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, req)

	require.Equal(t, http.StatusAccepted, response.Code)
	require.Equal(t, "cmcc", store.carrier)
	require.Equal(t, []string{"SN-1", "SN-2"}, store.disabledSerials)
}

func TestPolicyHTTPHandlerReadOnlyActionsDoNotExposeMutationRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewPolicyHTTPHandler(nil, nil, nil)
	handler.SetActionReader(NewActionService(nil, nil, nil, nil, nil))
	router := gin.New()
	handler.RegisterRoutes(router.Group("/api/v1"))

	paths := make([]string, 0, len(router.Routes()))
	for _, route := range router.Routes() {
		if strings.Contains(route.Path, "/actions") {
			paths = append(paths, route.Method+" "+route.Path)
		}
	}
	require.Equal(t, []string{
		"GET /api/v1/device-access/actions",
		"GET /api/v1/device-access/actions/:actionID/attempts",
	}, paths)
}

func TestPolicyHTTPHandlerRuntimeSettingsUseActorOperatorScope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	settings := &runtimeSettingsStub{}
	handler := NewPolicyHTTPHandler(nil, nil, actorResolverStub{
		actor: PolicyActor{Carrier: "cmcc", SubjectID: "user-1"},
	})
	handler.SetRuntimeSettingsStore(settings)
	router := gin.New()
	router.PUT("/settings", handler.UpdateRuntimeSettings)

	request := httptest.NewRequest(http.MethodPut, "/settings", strings.NewReader(`{"enabled":true}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "cmcc", settings.last.Carrier)
	require.True(t, settings.last.Enabled)
	require.Equal(t, "user-1", settings.last.UpdatedBy)
	var body struct {
		Data RuntimeSettings `json:"data"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	require.True(t, body.Data.Enabled)
}

func TestPolicyHTTPHandlerDoesNotQueueManualReevaluationWhileBusinessSwitchIsOff(t *testing.T) {
	gin.SetMode(gin.TestMode)
	queue := &reevaluationQueueStub{}
	handler := NewPolicyHTTPHandler(NewPolicyService(&policyStoreStub{}, queue), nil, actorResolverStub{
		actor: PolicyActor{Carrier: "cmcc", SubjectID: "user-1"},
	})
	handler.SetRuntimeSettingsStore(&runtimeSettingsStub{enabled: false})
	router := gin.New()
	router.POST("/devices/:serialNumber/reevaluate", handler.ReevaluateOne)

	request := httptest.NewRequest(http.MethodPost, "/devices/SN-1/reevaluate", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusConflict, response.Code)
	require.Empty(t, queue.requests)
}

func TestWritePolicyErrorMapsOwnershipConflictToConflict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/ownership-conflict", func(c *gin.Context) {
		writePolicyError(c, ErrAssetOwnershipConflict)
	})

	request := httptest.NewRequest(http.MethodGet, "/ownership-conflict", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusConflict, response.Code)
}
