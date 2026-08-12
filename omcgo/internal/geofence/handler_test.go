package geofence

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/admin"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	coremodel "github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func newHandlerTestRouter(service *Service, actorID *uuid.UUID) *gin.Engine {
	return newManualBatchHandlerTestRouter(
		service,
		actorID,
		false,
		nil,
	)
}

func newManualBatchHandlerTestRouter(
	service *Service,
	actorID *uuid.UUID,
	superAdministrator bool,
	visibleGroups []uuid.UUID,
) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	if actorID != nil {
		router.Use(func(c *gin.Context) {
			c.Set(admin.CtxKeyUserID, *actorID)
			c.Set(admin.CtxKeyIsSuperAdmin, superAdministrator)
			c.Next()
		})
	}
	handler := NewHandler(service, zap.NewNop())
	if visibleGroups != nil || actorID != nil {
		handler.SetPermissionService(handlerVisibleGroupsResolver{
			groups: visibleGroups,
		})
	}
	handler.RegisterRoutes(router.Group("/api/v1"))
	return router
}

type handlerVisibleGroupsResolver struct {
	groups []uuid.UUID
	err    error
}

type handlerAuditRepository struct {
	created chan *admin.AuditLog
}

func newHandlerAuditRepository() *handlerAuditRepository {
	return &handlerAuditRepository{created: make(chan *admin.AuditLog, 2)}
}

func (r *handlerAuditRepository) Create(
	_ context.Context,
	log *admin.AuditLog,
) error {
	r.created <- log
	return nil
}

func (r *handlerAuditRepository) List(
	context.Context,
	admin.AuditLogFilter,
) (*coremodel.ListResponse[admin.AuditLog], error) {
	return &coremodel.ListResponse[admin.AuditLog]{}, nil
}

func newAuditedHandlerTestRouter(
	service *Service,
	actorID uuid.UUID,
	auditRepository admin.AuditRepository,
) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(admin.CtxKeyUserID, actorID)
		c.Set(admin.CtxKeyUsername, "geofence-operator")
		c.Set(admin.CtxKeyIsSuperAdmin, true)
		c.Next()
	})
	router.Use(admin.AuditLogger(auditRepository))
	handler := NewHandler(service, zap.NewNop())
	handler.SetPermissionService(handlerVisibleGroupsResolver{})
	handler.RegisterRoutes(router.Group("/api/v1"))
	return router
}

func awaitHandlerAudit(
	t *testing.T,
	repository *handlerAuditRepository,
) *admin.AuditLog {
	t.Helper()
	select {
	case log := <-repository.created:
		return log
	case <-time.After(time.Second):
		t.Fatal("audit entry was not created")
		return nil
	}
}

func (r handlerVisibleGroupsResolver) GetUserVisibleGroupIDs(
	context.Context,
	uuid.UUID,
	bool,
) ([]uuid.UUID, error) {
	return r.groups, r.err
}

type handlerBatchRepository struct {
	Repository
	snapshot        ManualBindSnapshot
	snapshotErr     error
	visibleGroups   []uuid.UUID
	createParams    CreateManualBindJobParams
	createResult    BatchJobAccepted
	createErr       error
	job             *BatchJob
	getErr          error
	itemFilter      BatchItemFilter
	itemPage        BatchItemPage
	listErr         error
	createCallCount int
}

func (r *handlerBatchRepository) GetDefinition(
	context.Context,
	uuid.UUID,
) (*Definition, error) {
	return &r.snapshot.Definition, nil
}

func (r *handlerBatchRepository) IsCarrierVisible(
	context.Context,
	string,
	[]uuid.UUID,
) (bool, error) {
	return true, nil
}

func (r *handlerBatchRepository) LoadManualBindSnapshot(
	_ context.Context,
	_ uuid.UUID,
	_ []BindingInput,
	visibleGroups []uuid.UUID,
) (ManualBindSnapshot, error) {
	r.visibleGroups = visibleGroups
	return r.snapshot, r.snapshotErr
}

func (r *handlerBatchRepository) CreateManualBindJob(
	_ context.Context,
	params CreateManualBindJobParams,
) (BatchJobAccepted, error) {
	r.createCallCount++
	r.createParams = params
	return r.createResult, r.createErr
}

func (r *handlerBatchRepository) GetManualBindJob(
	context.Context,
	uuid.UUID,
) (*BatchJob, error) {
	return r.job, r.getErr
}

func (r *handlerBatchRepository) ListManualBindItems(
	_ context.Context,
	filter BatchItemFilter,
) (BatchItemPage, error) {
	r.itemFilter = filter
	return r.itemPage, r.listErr
}

func manualBatchSnapshot(
	geofenceID uuid.UUID,
	deviceID uuid.UUID,
) ManualBindSnapshot {
	versionID := uuid.New()
	return ManualBindSnapshot{
		Definition: Definition{
			ID:               geofenceID,
			Carrier:          "cmcc",
			RuleType:         RuleTypePolygonAllowZone,
			Status:           DefinitionStatusEnabled,
			CurrentVersionID: &versionID,
		},
		Inputs: []BindingCandidateFact{{
			Input: BindingInput{
				Key:      "id:" + deviceID.String(),
				Kind:     BindingInputDeviceID,
				Value:    deviceID.String(),
				DeviceID: &deviceID,
			},
			Device: &DeviceIdentity{
				ID: deviceID, SerialNumber: "SN001", Carrier: "cmcc",
			},
		}},
	}
}

func TestHandlerRegisterRoutesIncludesManualBatchBinding(t *testing.T) {
	router := newHandlerTestRouter(
		NewService(&fakeRepository{}, nil),
		nil,
	)
	got := make(map[string]struct{})
	for _, route := range router.Routes() {
		got[route.Method+" "+route.Path] = struct{}{}
	}
	for _, route := range []string{
		"POST /api/v1/geofences/:id/binding-preview",
		"POST /api/v1/geofences/:id/bindings",
		"GET /api/v1/geofences/:id/bindings/export",
		"GET /api/v1/geofence-jobs/:id",
		"GET /api/v1/geofence-jobs/:id/items",
	} {
		require.Contains(t, got, route)
	}
}

func TestHandlerAvailabilityRouteReturnsOnlySystemFeatureState(t *testing.T) {
	settingsRepository := &fakeSettingsRepository{settings: Settings{
		SystemMode: RuntimeModeObserve,
		Carriers: []CarrierSetting{{
			Carrier: "cmcc",
			Mode:    RuntimeModeOff,
		}},
	}}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewHandler(
		NewService(&fakeRepository{}, settingsRepository),
		zap.NewNop(),
	)
	handler.RegisterAvailabilityRoute(router.Group("/api/v1"))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/geofences/availability",
		nil,
	)

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	var envelope struct {
		Ret  int          `json:"ret"`
		Data Availability `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	assert.Equal(t, 1, envelope.Ret)
	assert.True(t, envelope.Data.Enabled)
}

func TestHandlerBindingPreviewEnforcesActorAndVisibleGroups(t *testing.T) {
	geofenceID := uuid.New()
	deviceID := uuid.New()
	body := `{"device_ids":["` + deviceID.String() + `"]}`

	t.Run("requires authenticated actor", func(t *testing.T) {
		router := newHandlerTestRouter(
			NewService(&handlerBatchRepository{}, nil),
			nil,
		)
		recorder := serveHandlerJSON(
			router,
			http.MethodPost,
			"/api/v1/geofences/"+geofenceID.String()+"/binding-preview",
			body,
		)
		require.Equal(t, http.StatusUnauthorized, recorder.Code)
		assertErrorEnvelope(t, recorder)
	})

	t.Run("empty visible groups fail closed", func(t *testing.T) {
		actorID := uuid.New()
		router := newManualBatchHandlerTestRouter(
			NewService(&handlerBatchRepository{}, nil),
			&actorID,
			false,
			[]uuid.UUID{},
		)
		recorder := serveHandlerJSON(
			router,
			http.MethodPost,
			"/api/v1/geofences/"+geofenceID.String()+"/binding-preview",
			body,
		)
		require.Equal(t, http.StatusForbidden, recorder.Code)
		assertErrorEnvelope(t, recorder)
	})

	t.Run("non empty visible groups are server derived", func(t *testing.T) {
		actorID := uuid.New()
		groupID := uuid.New()
		repository := &handlerBatchRepository{
			snapshot: manualBatchSnapshot(geofenceID, deviceID),
		}
		router := newManualBatchHandlerTestRouter(
			NewService(repository, nil),
			&actorID,
			false,
			[]uuid.UUID{groupID},
		)
		recorder := serveHandlerJSON(
			router,
			http.MethodPost,
			"/api/v1/geofences/"+geofenceID.String()+"/binding-preview",
			`{"device_ids":["`+deviceID.String()+`"],`+
				`"visible_groups":["`+uuid.NewString()+`"]}`,
		)
		require.Equal(t, http.StatusOK, recorder.Code)
		require.Equal(t, []uuid.UUID{groupID}, repository.visibleGroups)
		assertSuccessEnvelope(t, recorder)
	})

	t.Run("nil scope lets super administrator see all", func(t *testing.T) {
		actorID := uuid.New()
		repository := &handlerBatchRepository{
			snapshot: manualBatchSnapshot(geofenceID, deviceID),
		}
		router := newManualBatchHandlerTestRouter(
			NewService(repository, nil),
			&actorID,
			true,
			nil,
		)
		recorder := serveHandlerJSON(
			router,
			http.MethodPost,
			"/api/v1/geofences/"+geofenceID.String()+"/binding-preview",
			body,
		)
		require.Equal(t, http.StatusOK, recorder.Code)
		require.Nil(t, repository.visibleGroups)
	})
}

func TestHandlerVisibleGroupResolverErrorUsesInternalEnvelopeAndLogsCause(
	t *testing.T,
) {
	actorID := uuid.New()
	geofenceID := uuid.New()
	jobID := uuid.New()
	deviceID := uuid.New()
	paths := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{
			name:   "preview",
			method: http.MethodPost,
			path: "/api/v1/geofences/" + geofenceID.String() +
				"/binding-preview",
			body: `{"device_ids":["` + deviceID.String() + `"]}`,
		},
		{
			name:   "create",
			method: http.MethodPost,
			path:   "/api/v1/geofences/" + geofenceID.String() + "/bindings",
			body: `{
				"device_ids":["` + deviceID.String() + `"],
				"preview_fingerprint":"sha256:preview",
				"reason":"maintenance"
			}`,
		},
		{
			name:   "items",
			method: http.MethodGet,
			path:   "/api/v1/geofence-jobs/" + jobID.String() + "/items",
		},
	}
	backendErrors := []struct {
		name    string
		err     error
		rawText string
	}{
		{
			name:    "plain",
			err:     errors.New("database password=do-not-leak"),
			rawText: "database password=do-not-leak",
		},
		{
			name:    "forbidden",
			err:     commonerrors.ErrForbidden,
			rawText: commonerrors.ErrForbidden.Error(),
		},
		{
			name: "wrapped invalid input",
			err: fmt.Errorf(
				"permission query rejected: %w",
				commonerrors.ErrInvalidInput,
			),
			rawText: "permission query rejected",
		},
		{
			name: "wrapped unavailable",
			err: fmt.Errorf(
				"permission cache unavailable: %w",
				commonerrors.ErrUnavailable,
			),
			rawText: "permission cache unavailable",
		},
	}
	for _, path := range paths {
		for _, backend := range backendErrors {
			t.Run(path.name+"/"+backend.name, func(t *testing.T) {
				gin.SetMode(gin.TestMode)
				router := gin.New()
				router.Use(func(c *gin.Context) {
					c.Set(admin.CtxKeyUserID, actorID)
					c.Set(admin.CtxKeyIsSuperAdmin, false)
					c.Next()
				})
				logCore, observedLogs := observer.New(zap.WarnLevel)
				handler := NewHandler(
					NewService(&handlerBatchRepository{}, nil),
					zap.New(logCore),
				)
				handler.SetPermissionService(handlerVisibleGroupsResolver{
					err: backend.err,
				})
				handler.RegisterRoutes(router.Group("/api/v1"))

				recorder := serveHandlerJSON(
					router,
					path.method,
					path.path,
					path.body,
				)

				require.Equal(t, http.StatusInternalServerError, recorder.Code)
				var envelope struct {
					Ret  int    `json:"ret"`
					Msg  string `json:"msg"`
					Data any    `json:"data"`
				}
				require.NoError(
					t,
					json.Unmarshal(recorder.Body.Bytes(), &envelope),
				)
				require.Zero(t, envelope.Ret)
				require.Nil(t, envelope.Data)
				require.Equal(
					t,
					commonerrors.ErrInternal.Error(),
					envelope.Msg,
				)
				require.NotContains(
					t,
					recorder.Body.String(),
					backend.rawText,
				)

				entries := observedLogs.All()
				require.Len(t, entries, 1)
				require.Equal(t, "geofence request failed", entries[0].Message)
				loggedError, ok := entries[0].ContextMap()["error"].(string)
				require.True(t, ok)
				require.Contains(t, loggedError, backend.rawText)
			})
		}
	}
}

func TestHandlerBindingPreviewRejectsInvalidInput(t *testing.T) {
	actorID := uuid.New()
	groupID := uuid.New()
	geofenceID := uuid.New()
	tooManyIDs := make([]uuid.UUID, MaxBatchBindingInputs+1)
	for i := range tooManyIDs {
		tooManyIDs[i] = uuid.New()
	}
	tooManyBody, err := json.Marshal(BindingInputRequest{DeviceIDs: tooManyIDs})
	require.NoError(t, err)

	tests := []struct {
		name string
		path string
		body string
	}{
		{
			name: "malformed json",
			path: "/api/v1/geofences/" + geofenceID.String() + "/binding-preview",
			body: `{"device_ids":`,
		},
		{
			name: "empty input",
			path: "/api/v1/geofences/" + geofenceID.String() + "/binding-preview",
			body: `{}`,
		},
		{
			name: "over one thousand inputs",
			path: "/api/v1/geofences/" + geofenceID.String() + "/binding-preview",
			body: string(tooManyBody),
		},
		{
			name: "invalid geofence uuid",
			path: "/api/v1/geofences/not-a-uuid/binding-preview",
			body: `{"device_sns":["SN001"]}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := newManualBatchHandlerTestRouter(
				NewService(&handlerBatchRepository{}, nil),
				&actorID,
				false,
				[]uuid.UUID{groupID},
			)
			recorder := serveHandlerJSON(
				router,
				http.MethodPost,
				tt.path,
				tt.body,
			)
			require.Equal(t, http.StatusBadRequest, recorder.Code)
			assertErrorEnvelope(t, recorder)
		})
	}
}

func TestHandlerManualBindingCreateContract(t *testing.T) {
	actorID := uuid.New()
	groupID := uuid.New()
	geofenceID := uuid.New()
	deviceID := uuid.New()
	jobID := uuid.New()
	body := `{
		"device_ids":["` + deviceID.String() + `"],
		"preview_fingerprint":"sha256:preview",
		"reason":"maintenance",
		"requested_by":"` + uuid.NewString() + `",
		"visible_groups":["` + uuid.NewString() + `"]
	}`

	t.Run("accepted and duplicate return same server owned job", func(t *testing.T) {
		repository := &handlerBatchRepository{
			createResult: BatchJobAccepted{JobID: jobID},
		}
		router := newManualBatchHandlerTestRouter(
			NewService(repository, nil),
			&actorID,
			false,
			[]uuid.UUID{groupID},
		)
		for range 2 {
			recorder := serveHandlerJSON(
				router,
				http.MethodPost,
				"/api/v1/geofences/"+geofenceID.String()+"/bindings",
				body,
			)
			require.Equal(t, http.StatusAccepted, recorder.Code)
			var envelope struct {
				Ret  int              `json:"ret"`
				Data BatchJobAccepted `json:"data"`
			}
			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
			require.Equal(t, 1, envelope.Ret)
			require.Equal(t, jobID, envelope.Data.JobID)
		}
		require.Equal(t, 2, repository.createCallCount)
		require.Equal(t, actorID, repository.createParams.ActorID)
		require.Equal(t, []uuid.UUID{groupID}, repository.createParams.VisibleGroups)
	})

	t.Run("stale preview maps to conflict", func(t *testing.T) {
		repository := &handlerBatchRepository{createErr: ErrStaleBindingPreview}
		router := newManualBatchHandlerTestRouter(
			NewService(repository, nil),
			&actorID,
			false,
			[]uuid.UUID{groupID},
		)
		recorder := serveHandlerJSON(
			router,
			http.MethodPost,
			"/api/v1/geofences/"+geofenceID.String()+"/bindings",
			body,
		)
		require.Equal(t, http.StatusConflict, recorder.Code)
		assertErrorEnvelope(t, recorder)
	})

	t.Run("zero eligible inputs maps to unprocessable entity", func(t *testing.T) {
		repository := &handlerBatchRepository{
			createErr: ErrNoEligibleBindingInputs,
		}
		router := newManualBatchHandlerTestRouter(
			NewService(repository, nil),
			&actorID,
			false,
			[]uuid.UUID{groupID},
		)
		recorder := serveHandlerJSON(
			router,
			http.MethodPost,
			"/api/v1/geofences/"+geofenceID.String()+"/bindings",
			body,
		)
		require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
		assertErrorEnvelope(t, recorder)
	})
}

func TestHandlerGeofenceJobOwnershipAndSuperAdministrator(t *testing.T) {
	requesterID := uuid.New()
	otherActorID := uuid.New()
	jobID := uuid.New()
	repository := &handlerBatchRepository{
		job: &BatchJob{ID: jobID, RequestedBy: requesterID},
	}

	t.Run("requester may inspect own job", func(t *testing.T) {
		router := newManualBatchHandlerTestRouter(
			NewService(repository, nil),
			&requesterID,
			false,
			[]uuid.UUID{},
		)
		recorder := serveHandlerJSON(
			router,
			http.MethodGet,
			"/api/v1/geofence-jobs/"+jobID.String(),
			"",
		)
		require.Equal(t, http.StatusOK, recorder.Code)
		assertSuccessEnvelope(t, recorder)
	})

	t.Run("non owner is forbidden", func(t *testing.T) {
		router := newManualBatchHandlerTestRouter(
			NewService(repository, nil),
			&otherActorID,
			false,
			[]uuid.UUID{uuid.New()},
		)
		recorder := serveHandlerJSON(
			router,
			http.MethodGet,
			"/api/v1/geofence-jobs/"+jobID.String(),
			"",
		)
		require.Equal(t, http.StatusForbidden, recorder.Code)
		assertErrorEnvelope(t, recorder)
	})

	t.Run("super administrator may inspect another requester job", func(t *testing.T) {
		router := newManualBatchHandlerTestRouter(
			NewService(repository, nil),
			&otherActorID,
			true,
			nil,
		)
		recorder := serveHandlerJSON(
			router,
			http.MethodGet,
			"/api/v1/geofence-jobs/"+jobID.String(),
			"",
		)
		require.Equal(t, http.StatusOK, recorder.Code)
		assertSuccessEnvelope(t, recorder)
	})

	t.Run("non owner may not inspect items", func(t *testing.T) {
		router := newManualBatchHandlerTestRouter(
			NewService(repository, nil),
			&otherActorID,
			false,
			[]uuid.UUID{uuid.New()},
		)
		recorder := serveHandlerJSON(
			router,
			http.MethodGet,
			"/api/v1/geofence-jobs/"+jobID.String()+"/items",
			"",
		)
		require.Equal(t, http.StatusForbidden, recorder.Code)
		assertErrorEnvelope(t, recorder)
	})

	t.Run("super administrator may inspect another requester items", func(t *testing.T) {
		router := newManualBatchHandlerTestRouter(
			NewService(repository, nil),
			&otherActorID,
			true,
			nil,
		)
		recorder := serveHandlerJSON(
			router,
			http.MethodGet,
			"/api/v1/geofence-jobs/"+jobID.String()+"/items",
			"",
		)
		require.Equal(t, http.StatusOK, recorder.Code)
		assertSuccessEnvelope(t, recorder)
	})
}

func TestHandlerGeofenceJobItemsValidatesQueryAndScopesResults(t *testing.T) {
	actorID := uuid.New()
	groupID := uuid.New()
	jobID := uuid.New()

	tests := []struct {
		name  string
		query string
	}{
		{name: "invalid status", query: "?status=unknown"},
		{name: "invalid page", query: "?page=nope"},
		{name: "zero page", query: "?page=0"},
		{name: "invalid page size", query: "?page_size=nope"},
		{
			name:  "oversized page size",
			query: "?page_size=" + strconv.Itoa(MaxBatchItemPageSize+1),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &handlerBatchRepository{
				job: &BatchJob{ID: jobID, RequestedBy: actorID},
			}
			router := newManualBatchHandlerTestRouter(
				NewService(repository, nil),
				&actorID,
				false,
				[]uuid.UUID{groupID},
			)
			recorder := serveHandlerJSON(
				router,
				http.MethodGet,
				"/api/v1/geofence-jobs/"+jobID.String()+"/items"+tt.query,
				"",
			)
			require.Equal(t, http.StatusBadRequest, recorder.Code)
			assertErrorEnvelope(t, recorder)
		})
	}

	repository := &handlerBatchRepository{
		job: &BatchJob{ID: jobID, RequestedBy: actorID},
		itemPage: BatchItemPage{
			Items: []BatchItem{}, Total: 0, Page: 2, PageSize: 25,
		},
	}
	router := newManualBatchHandlerTestRouter(
		NewService(repository, nil),
		&actorID,
		false,
		[]uuid.UUID{groupID},
	)
	recorder := serveHandlerJSON(
		router,
		http.MethodGet,
		"/api/v1/geofence-jobs/"+jobID.String()+
			"/items?status=failed&page=2&page_size=25",
		"",
	)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, BatchItemFailed, repository.itemFilter.Status)
	require.Equal(t, 2, repository.itemFilter.Page)
	require.Equal(t, 25, repository.itemFilter.PageSize)
	require.Equal(t, []uuid.UUID{groupID}, repository.itemFilter.VisibleGroups)
	assertSuccessEnvelope(t, recorder)
}

func TestHandlerGeofenceJobInternalErrorIsNotLeaked(t *testing.T) {
	actorID := uuid.New()
	jobID := uuid.New()
	repository := &handlerBatchRepository{
		getErr: errors.New("database password=do-not-leak"),
	}
	router := newManualBatchHandlerTestRouter(
		NewService(repository, nil),
		&actorID,
		false,
		[]uuid.UUID{uuid.New()},
	)
	recorder := serveHandlerJSON(
		router,
		http.MethodGet,
		"/api/v1/geofence-jobs/"+jobID.String(),
		"",
	)

	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.NotContains(t, recorder.Body.String(), "do-not-leak")
	assertErrorEnvelope(t, recorder)
}

func serveHandlerJSON(
	router http.Handler,
	method string,
	path string,
	body string,
) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func assertSuccessEnvelope(
	t *testing.T,
	recorder *httptest.ResponseRecorder,
) {
	t.Helper()
	var envelope struct {
		Ret  int             `json:"ret"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	require.Equal(t, 1, envelope.Ret)
	require.Equal(t, "ok", envelope.Msg)
	require.NotEqual(t, "null", string(envelope.Data))
}

func TestHandler_CreateDefinitionRequiresAuthenticatedActor(t *testing.T) {
	router := newHandlerTestRouter(NewService(&fakeRepository{}, nil), nil)
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/geofences",
		bytes.NewBufferString(`{}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	assert.Equal(t, http.StatusUnauthorized, response.Code)
}

func TestHandlerListMapDefinitionsReturnsMapReadyEnvelope(t *testing.T) {
	geofenceID := uuid.New()
	versionID := uuid.New()
	repository := &fakeRepository{
		mapDefinitions: []MapDefinition{{
			Definition: Definition{
				ID:               geofenceID,
				Name:             "west",
				Carrier:          "cmcc",
				RuleType:         RuleTypePolygonAllowZone,
				Status:           DefinitionStatusEnabled,
				CurrentVersionID: &versionID,
			},
			CurrentVersion: &Version{
				ID:         versionID,
				GeofenceID: geofenceID,
				Version:    2,
				Status:     VersionStatusPublished,
				GeometryJSON: json.RawMessage(
					`{"type":"Polygon","coordinates":[[[120,30],[121,30],[121,31],[120,30]]]}`,
				),
				PolicyJSON: json.RawMessage(`{"exit_action":"notify_only"}`),
			},
		}},
	}
	router := newHandlerTestRouter(
		NewService(repository, nil),
		nil,
	)
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/geofences/map?bounds=120,121,30,31"+
			"&carrier=CMCC&status=enabled&name=%20west%20",
		nil,
	)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.NotNil(t, repository.mapDefinitionFilter.Bounds)
	require.Equal(t, float64(120), repository.mapDefinitionFilter.Bounds.MinLongitude)
	require.Equal(t, "cmcc", repository.mapDefinitionFilter.Carrier)
	require.Equal(t, DefinitionStatusEnabled, repository.mapDefinitionFilter.Status)
	require.Equal(t, "west", repository.mapDefinitionFilter.Name)
	var envelope struct {
		Ret  int `json:"ret"`
		Data struct {
			Items []MapDefinition `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	require.Equal(t, 1, envelope.Ret)
	require.Len(t, envelope.Data.Items, 1)
	require.Equal(t, geofenceID, envelope.Data.Items[0].Definition.ID)
	require.Equal(
		t,
		versionID,
		envelope.Data.Items[0].CurrentVersion.ID,
	)
}

func TestHandlerListMapDefinitionsPassesServerDerivedVisibleGroups(
	t *testing.T,
) {
	groupID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	actorID := uuid.New()
	repository := &fakeRepository{}
	router := newManualBatchHandlerTestRouter(
		NewService(repository, nil),
		&actorID,
		false,
		[]uuid.UUID{groupID},
	)
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/geofences/map",
		nil,
	)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(
		t,
		[]uuid.UUID{groupID},
		repository.mapDefinitionFilter.VisibleGroups,
	)
}

func TestHandlerPublishRejectsGeofenceOutsideVisibleCarrierScope(
	t *testing.T,
) {
	geofenceID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	versionID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	actorID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	repository := &fakeRepository{
		definition: &Definition{
			ID:      geofenceID,
			Carrier: "ctcc",
		},
		version: &Version{
			ID:         versionID,
			GeofenceID: geofenceID,
			Status:     VersionStatusDraft,
			PolicyJSON: json.RawMessage(`{"exit_action":"notify_only"}`),
		},
		denyCarrierVisibility: true,
	}
	router := newManualBatchHandlerTestRouter(
		NewService(repository, nil),
		&actorID,
		false,
		[]uuid.UUID{uuid.New()},
	)
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/geofences/"+geofenceID.String()+"/publish",
		bytes.NewBufferString(`{"version_id":"`+versionID.String()+`"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusForbidden, recorder.Code)
	require.Equal(t, 0, repository.publishCalls)
	assertErrorEnvelope(t, recorder)
}

func TestHandlerListMapDefinitionsRejectsMalformedBounds(t *testing.T) {
	repository := &fakeRepository{}
	router := newHandlerTestRouter(
		NewService(repository, nil),
		nil,
	)
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/geofences/map?bounds=121,120,30,31",
		nil,
	)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Nil(t, repository.mapDefinitionFilter.Bounds)
	assertErrorEnvelope(t, recorder)
}

func TestHandlerRegisterRoutesIncludesGeofenceMapBeforeDynamicID(
	t *testing.T,
) {
	router := newHandlerTestRouter(
		NewService(&fakeRepository{}, nil),
		nil,
	)
	routes := make(map[string]struct{})
	for _, route := range router.Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
	}

	_, ok := routes["GET /api/v1/geofences/map"]
	require.True(t, ok)
}

func TestHandlerListVersionsReturnsNewestFirstHistory(t *testing.T) {
	geofenceID := uuid.New()
	repository := &fakeRepository{
		definition: &Definition{ID: geofenceID},
		versions: []Version{
			{ID: uuid.New(), GeofenceID: geofenceID, Version: 2},
			{ID: uuid.New(), GeofenceID: geofenceID, Version: 1},
		},
	}
	router := newHandlerTestRouter(
		NewService(repository, nil),
		nil,
	)
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/geofences/"+geofenceID.String()+"/versions",
		nil,
	)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	var envelope struct {
		Ret  int `json:"ret"`
		Data struct {
			Items []Version `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	require.Equal(t, 1, envelope.Ret)
	require.Len(t, envelope.Data.Items, 2)
	require.Equal(t, int64(2), envelope.Data.Items[0].Version)
	require.Equal(t, repository.versions[0].ID, envelope.Data.Items[0].ID)
	require.Equal(t, int64(1), envelope.Data.Items[1].Version)
	require.Equal(t, repository.versions[1].ID, envelope.Data.Items[1].ID)
}

func TestHandlerListVersionsRejectsInvalidGeofenceID(t *testing.T) {
	router := newHandlerTestRouter(
		NewService(&fakeRepository{}, nil),
		nil,
	)
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/geofences/not-a-uuid/versions",
		nil,
	)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	assertErrorEnvelope(t, recorder)
}

func TestHandlerRegisterRoutesIncludesVersionHistory(t *testing.T) {
	router := newHandlerTestRouter(
		NewService(&fakeRepository{}, nil),
		nil,
	)
	routes := make(map[string]struct{})
	for _, route := range router.Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
	}

	_, ok := routes["GET /api/v1/geofences/:id/versions"]
	require.True(t, ok)
}

func TestHandlerListBindingsReturnsPermissionSafePage(t *testing.T) {
	actorID := uuid.New()
	groupID := uuid.New()
	geofenceID := uuid.New()
	bindingID := uuid.New()
	deviceID := uuid.New()
	lastDistance := 35.5
	repository := &fakeRepository{
		definition: &Definition{ID: geofenceID},
		bindingDetailPage: BindingDetailPage{
			Items: []BindingDetail{{
				Binding: Binding{
					ID:         bindingID,
					DeviceID:   deviceID,
					GeofenceID: geofenceID,
					Status:     BindingStatusActive,
				},
				DeviceSN:        "SN001",
				DeviceName:      "station-west",
				DeviceCarrier:   "cmcc",
				DeviceGroupID:   &groupID,
				DeviceGroupName: "group-west",
				HasLocation:     true,
				Evaluation: BindingEvaluationDetail{
					ConfirmedState:         ConfirmedStateInside,
					CandidateState:         CandidateStateExit,
					CandidateCount:         2,
					LastObservationVersion: 12,
					EvaluationHealth:       "healthy",
					LastDistanceToBoundary: &lastDistance,
				},
			}},
			Total: 1, Page: 2, PageSize: 25,
		},
	}
	router := newManualBatchHandlerTestRouter(
		NewService(repository, nil),
		&actorID,
		false,
		[]uuid.UUID{groupID},
	)
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/geofences/"+geofenceID.String()+
			"/bindings?status=active&keyword=station&page=2&page_size=25",
		nil,
	)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(
		t,
		[]uuid.UUID{groupID},
		repository.bindingDetailFilter.VisibleGroups,
	)
	require.Equal(t, BindingStatusActive, repository.bindingDetailFilter.Status)
	require.Equal(t, "station", repository.bindingDetailFilter.Keyword)
	require.Equal(t, 2, repository.bindingDetailFilter.Page)
	require.Equal(t, 25, repository.bindingDetailFilter.PageSize)
	var envelope struct {
		Ret  int               `json:"ret"`
		Data BindingDetailPage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	require.Equal(t, 1, envelope.Ret)
	require.Len(t, envelope.Data.Items, 1)
	require.Equal(t, "SN001", envelope.Data.Items[0].DeviceSN)
	require.Equal(t, "station-west", envelope.Data.Items[0].DeviceName)
	require.Equal(
		t,
		ConfirmedStateInside,
		envelope.Data.Items[0].Evaluation.ConfirmedState,
	)
	require.Equal(
		t,
		CandidateStateExit,
		envelope.Data.Items[0].Evaluation.CandidateState,
	)
	require.NotContains(t, recorder.Body.String(), "latitude")
	require.NotContains(t, recorder.Body.String(), "longitude")
}

func TestHandlerListBindingsMapsCurrentStatusScope(t *testing.T) {
	actorID := uuid.New()
	geofenceID := uuid.New()
	repository := &fakeRepository{
		definition: &Definition{ID: geofenceID},
	}
	router := newManualBatchHandlerTestRouter(
		NewService(repository, nil),
		&actorID,
		true,
		nil,
	)
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/geofences/"+geofenceID.String()+
			"/bindings?status=current",
		nil,
	)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.True(t, repository.bindingDetailFilter.CurrentOnly)
	require.Empty(t, repository.bindingDetailFilter.Status)
}

func TestHandlerExportBindingsReturnsPermissionSafeCSV(t *testing.T) {
	actorID := uuid.New()
	groupID := uuid.New()
	geofenceID := uuid.New()
	boundAt := time.Date(2026, 7, 31, 8, 30, 0, 0, time.UTC)
	repository := &fakeRepository{
		definition: &Definition{ID: geofenceID},
		bindingDetailPage: BindingDetailPage{
			Items: []BindingDetail{{
				Binding: Binding{
					ID:         uuid.New(),
					DeviceID:   uuid.New(),
					GeofenceID: geofenceID,
					Status:     BindingStatusActive,
					BoundAt:    boundAt,
				},
				DeviceSN:        "SN001",
				DeviceName:      "杭州,西站",
				DeviceCarrier:   "cmcc",
				DeviceGroupID:   &groupID,
				DeviceGroupName: "杭州组",
				Evaluation: BindingEvaluationDetail{
					ConfirmedState:   ConfirmedStateInside,
					EvaluationHealth: "healthy",
				},
			}},
			Total: 1,
		},
	}
	router := newManualBatchHandlerTestRouter(
		NewService(repository, nil),
		&actorID,
		false,
		[]uuid.UUID{groupID},
	)
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/geofences/"+geofenceID.String()+
			"/bindings/export?status=active&keyword=SN001",
		nil,
	)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "text/csv; charset=utf-8", recorder.Header().Get("Content-Type"))
	require.Contains(
		t,
		recorder.Header().Get("Content-Disposition"),
		"geofence-bindings.csv",
	)
	require.Equal(
		t,
		[]uuid.UUID{groupID},
		repository.bindingDetailFilter.VisibleGroups,
	)
	require.Equal(t, BindingStatusActive, repository.bindingDetailFilter.Status)
	require.Equal(t, "SN001", repository.bindingDetailFilter.Keyword)

	body := recorder.Body.Bytes()
	require.True(t, bytes.HasPrefix(body, []byte{0xEF, 0xBB, 0xBF}))
	rows, err := csv.NewReader(bytes.NewReader(body[3:])).ReadAll()
	require.NoError(t, err)
	require.Equal(t, [][]string{
		{
			"device_sn",
			"device_name",
			"carrier",
			"device_group",
			"binding_status",
			"bound_at",
			"confirmed_state",
			"evaluation_health",
		},
		{
			"SN001",
			"杭州,西站",
			"cmcc",
			"杭州组",
			"active",
			"2026-07-31T08:30:00Z",
			"inside",
			"healthy",
		},
	}, rows)
	require.NotContains(t, recorder.Body.String(), "latitude")
	require.NotContains(t, recorder.Body.String(), "longitude")
}

func TestHandlerListBindingsRejectsEmptyVisibleGroups(t *testing.T) {
	actorID := uuid.New()
	router := newManualBatchHandlerTestRouter(
		NewService(&fakeRepository{}, nil),
		&actorID,
		false,
		[]uuid.UUID{},
	)
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/geofences/"+uuid.NewString()+"/bindings",
		nil,
	)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusForbidden, recorder.Code)
	assertErrorEnvelope(t, recorder)
}

func TestHandlerListBindingsRejectsInvalidPagination(t *testing.T) {
	actorID := uuid.New()
	router := newManualBatchHandlerTestRouter(
		NewService(&fakeRepository{}, nil),
		&actorID,
		true,
		nil,
	)
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/geofences/"+uuid.NewString()+
			"/bindings?page=0",
		nil,
	)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	assertErrorEnvelope(t, recorder)
}

func TestHandler_CreateDefinitionRejectsMalformedGeometry(t *testing.T) {
	actorID := uuid.New()
	router := newHandlerTestRouter(NewService(&fakeRepository{}, nil), &actorID)
	body := `{
		"name":"bad polygon",
		"carrier":"cmcc",
		"rule_type":"polygon_allow_zone",
		"geometry":{"type":"Polygon","coordinates":[[[121.1,31.1],[121.2,31.1]]]},
		"policy":{"exit_action":"notify_only"}
	}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/geofences", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	assert.Equal(t, http.StatusBadRequest, response.Code)
}

func TestHandler_CreateDefinitionReturnsDefinitionAndDraftVersion(t *testing.T) {
	actorID := uuid.New()
	repository := &fakeRepository{}
	router := newHandlerTestRouter(NewService(repository, nil), &actorID)
	body := `{
		"name":"杭州测试围栏",
		"carrier":"cmcc",
		"rule_type":"polygon_allow_zone",
		"geometry":{"type":"Polygon","coordinates":[[[121.1,31.1],[121.2,31.1],[121.2,31.2]]]},
		"policy":{"exit_action":"notify_only"}
	}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/geofences", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusCreated, response.Code)
	var envelope struct {
		Ret  int `json:"ret"`
		Data struct {
			Definition struct {
				ID uuid.UUID `json:"id"`
			} `json:"definition"`
			DraftVersion struct {
				ID uuid.UUID `json:"id"`
			} `json:"draft_version"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &envelope))
	assert.Equal(t, 1, envelope.Ret)
	assert.NotEqual(t, uuid.Nil, envelope.Data.Definition.ID)
	assert.NotEqual(t, uuid.Nil, envelope.Data.DraftVersion.ID)
	assert.Equal(t, actorID, repository.createdDefinition.CreatedBy)
}

func TestHandler_CreateDefinitionWritesSemanticAudit(t *testing.T) {
	actorID := uuid.New()
	repository := &fakeRepository{}
	auditRepository := newHandlerAuditRepository()
	router := newAuditedHandlerTestRouter(
		NewService(repository, nil),
		actorID,
		auditRepository,
	)
	body := `{
		"name":"杭州测试围栏",
		"carrier":"cmcc",
		"rule_type":"polygon_allow_zone",
		"geometry":{"type":"Polygon","coordinates":[[[121.1,31.1],[121.2,31.1],[121.2,31.2]]]},
		"policy":{"exit_action":"notify_only"}
	}`

	recorder := serveHandlerJSON(
		router,
		http.MethodPost,
		"/api/v1/geofences",
		body,
	)

	require.Equal(t, http.StatusCreated, recorder.Code)
	log := awaitHandlerAudit(t, auditRepository)
	require.Equal(t, "config_geofence_create", log.Action)
	require.Equal(t, "geofence", log.Resource)
	require.Equal(t, repository.createdDefinition.ID.String(), log.ResourceID)
	require.Equal(t, "杭州测试围栏", log.Details["name"])
	require.Equal(t, "cmcc", log.Details["carrier"])
	require.Equal(t, actorID, *log.UserID)
}

func TestHandler_CreateDefinitionFailureWritesAuditReason(t *testing.T) {
	actorID := uuid.New()
	auditRepository := newHandlerAuditRepository()
	router := newAuditedHandlerTestRouter(
		NewService(&fakeRepository{}, nil),
		actorID,
		auditRepository,
	)
	body := `{
		"name":"bad polygon",
		"carrier":"cmcc",
		"rule_type":"polygon_allow_zone",
		"geometry":{"type":"Polygon","coordinates":[[[121.1,31.1],[121.2,31.1]]]},
		"policy":{"exit_action":"notify_only"}
	}`

	recorder := serveHandlerJSON(
		router,
		http.MethodPost,
		"/api/v1/geofences",
		body,
	)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	log := awaitHandlerAudit(t, auditRepository)
	require.Equal(t, "config_geofence_create_failed", log.Action)
	require.Equal(t, "geofence", log.Resource)
	require.Contains(t, log.Details["error"], "polygon")
}

func TestHandler_PreviewEndpointsSkipComplianceAudit(t *testing.T) {
	actorID := uuid.New()
	auditRepository := newHandlerAuditRepository()
	router := newAuditedHandlerTestRouter(
		NewService(&fakeRepository{}, &fakeSettingsRepository{}),
		actorID,
		auditRepository,
	)
	body := `{
		"system_mode":"observe",
		"carriers":[{
			"carrier":"cmcc",
			"mode":"observe",
			"default_baseline_radius_meters":100
		}]
	}`

	recorder := serveHandlerJSON(
		router,
		http.MethodPost,
		"/api/v1/geofences/settings/preview",
		body,
	)

	require.Equal(t, http.StatusOK, recorder.Code)
	select {
	case log := <-auditRepository.created:
		t.Fatalf("preview unexpectedly created audit entry: %+v", log)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestHandler_OtherGeofenceMutationsWriteSemanticAudit(t *testing.T) {
	actorID := uuid.New()

	t.Run("rename definition", func(t *testing.T) {
		geofenceID := uuid.New()
		auditRepository := newHandlerAuditRepository()
		repository := &fakeRepository{}
		router := newAuditedHandlerTestRouter(
			NewService(repository, nil),
			actorID,
			auditRepository,
		)

		recorder := serveHandlerJSON(
			router,
			http.MethodPatch,
			"/api/v1/geofences/"+geofenceID.String(),
			`{"name":"  新围栏名称  "}`,
		)

		require.Equal(t, http.StatusOK, recorder.Code)
		log := awaitHandlerAudit(t, auditRepository)
		require.Equal(t, auditActionModify, log.Action)
		require.Equal(t, geofenceID.String(), log.ResourceID)
		require.Equal(t, "新围栏名称", log.Details["name"])
	})

	t.Run("definition lifecycle", func(t *testing.T) {
		geofenceID := uuid.New()
		auditRepository := newHandlerAuditRepository()
		router := newAuditedHandlerTestRouter(
			NewService(&fakeRepository{}, nil),
			actorID,
			auditRepository,
		)

		recorder := serveHandlerJSON(
			router,
			http.MethodPost,
			"/api/v1/geofences/"+geofenceID.String()+"/disable",
			`{"reason":"maintenance","preview_fingerprint":"fingerprint"}`,
		)

		require.Equal(t, http.StatusOK, recorder.Code)
		log := awaitHandlerAudit(t, auditRepository)
		require.Equal(t, auditActionDisable, log.Action)
		require.Equal(t, geofenceID.String(), log.ResourceID)
		require.Equal(t, "maintenance", log.Details["reason"])
		require.Equal(
			t,
			string(DefinitionStatusDisabled),
			fmt.Sprint(log.Details["target_status"]),
		)
	})

	t.Run("system and carrier settings", func(t *testing.T) {
		auditRepository := newHandlerAuditRepository()
		router := newAuditedHandlerTestRouter(
			NewService(&fakeRepository{}, &fakeSettingsRepository{}),
			actorID,
			auditRepository,
		)

		recorder := serveHandlerJSON(
			router,
			http.MethodPut,
			"/api/v1/geofences/settings",
			`{
				"system_mode":"observe",
				"carriers":[{
					"carrier":"cmcc",
					"mode":"observe",
					"default_baseline_radius_meters":100
				}]
			}`,
		)

		require.Equal(t, http.StatusOK, recorder.Code)
		log := awaitHandlerAudit(t, auditRepository)
		require.Equal(t, auditActionSettingsUpdate, log.Action)
		require.Equal(t, "system", log.ResourceID)
		require.Equal(
			t,
			string(RuntimeModeObserve),
			fmt.Sprint(log.Details["system_mode"]),
		)
	})

	t.Run("binding lifecycle", func(t *testing.T) {
		bindingID := uuid.New()
		auditRepository := newHandlerAuditRepository()
		router := newAuditedHandlerTestRouter(
			NewService(&fakeRepository{}, nil),
			actorID,
			auditRepository,
		)

		recorder := serveHandlerJSON(
			router,
			http.MethodPost,
			"/api/v1/geofence-bindings/"+bindingID.String()+"/suspend",
			`{"reason":"device moved"}`,
		)

		require.Equal(t, http.StatusOK, recorder.Code)
		log := awaitHandlerAudit(t, auditRepository)
		require.Equal(t, auditActionBindingSuspend, log.Action)
		require.Equal(t, bindingID.String(), log.ResourceID)
		require.Equal(t, "device moved", log.Details["reason"])
	})

	t.Run("binding resume", func(t *testing.T) {
		bindingID := uuid.New()
		auditRepository := newHandlerAuditRepository()
		router := newAuditedHandlerTestRouter(
			NewService(&fakeRepository{}, nil),
			actorID,
			auditRepository,
		)

		recorder := serveHandlerJSON(
			router,
			http.MethodPost,
			"/api/v1/geofence-bindings/"+bindingID.String()+"/resume",
			`{"reason":"maintenance completed"}`,
		)

		require.Equal(t, http.StatusOK, recorder.Code)
		log := awaitHandlerAudit(t, auditRepository)
		require.Equal(t, auditActionBindingResume, log.Action)
		require.Equal(t, bindingID.String(), log.ResourceID)
		require.Equal(t, "maintenance completed", log.Details["reason"])
	})

	t.Run("manual batch binding", func(t *testing.T) {
		geofenceID := uuid.New()
		deviceID := uuid.New()
		jobID := uuid.New()
		auditRepository := newHandlerAuditRepository()
		router := newAuditedHandlerTestRouter(
			NewService(&handlerBatchRepository{
				createResult: BatchJobAccepted{JobID: jobID},
			}, nil),

			actorID,
			auditRepository,
		)

		recorder := serveHandlerJSON(
			router,
			http.MethodPost,
			"/api/v1/geofences/"+geofenceID.String()+"/bindings",
			`{
				"device_ids":["`+deviceID.String()+`"],
				"device_sns":["SN001"],
				"preview_fingerprint":"sha256:preview",
				"reason":"initial rollout"
			}`,
		)

		require.Equal(t, http.StatusAccepted, recorder.Code)
		log := awaitHandlerAudit(t, auditRepository)
		require.Equal(t, auditActionBatchBind, log.Action)
		require.Equal(t, geofenceID.String(), log.ResourceID)
		require.Equal(t, jobID.String(), log.Details["job_id"])
		require.Equal(t, 1, log.Details["device_id_count"])
		require.Equal(t, 1, log.Details["device_sn_count"])
	})
}

func TestHandler_RegisterRoutesIncludesSettingsEndpoints(t *testing.T) {
	router := newHandlerTestRouter(
		NewService(&fakeRepository{}, &fakeSettingsRepository{}),
		nil,
	)
	routes := make(map[string]struct{})
	for _, route := range router.Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
	}

	for _, route := range []string{
		"GET /api/v1/geofences/settings",
		"POST /api/v1/geofences/settings/preview",
		"PUT /api/v1/geofences/settings",
	} {
		_, ok := routes[route]
		assert.True(t, ok, route)
	}
}

func TestHandler_GetSettingsReturnsResponseEnvelope(t *testing.T) {
	settingsRepository := &fakeSettingsRepository{settings: Settings{
		SystemMode: RuntimeModeOff,
		Carriers: []CarrierSetting{{
			Carrier: "cmcc", Mode: RuntimeModeOff,
			EffectiveMode:               RuntimeModeOff,
			DefaultBaselineRadiusMeters: 100,
		}},
	}}
	router := newHandlerTestRouter(
		NewService(&fakeRepository{}, settingsRepository),
		nil,
	)
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/geofences/settings",
		nil,
	)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	var envelope struct {
		Ret  int      `json:"ret"`
		Data Settings `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	assert.Equal(t, 1, envelope.Ret)
	assert.Equal(t, RuntimeModeOff, envelope.Data.SystemMode)
}

func TestHandler_PreviewSettingsAcceptsEnforce(t *testing.T) {
	actorID := uuid.New()
	router := newManualBatchHandlerTestRouter(
		NewService(&fakeRepository{}, &fakeSettingsRepository{}),
		&actorID,
		true,
		nil,
	)
	body := `{
		"system_mode":"enforce",
		"carriers":[{
			"carrier":"cmcc",
			"mode":"observe",
			"default_baseline_radius_meters":100
		}]
	}`
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/geofences/settings/preview",
		bytes.NewBufferString(body),
	)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestHandler_PreviewSettingsRejectsUnauthorizedCarrier(t *testing.T) {
	actorID := uuid.New()
	settingsRepository := &fakeSettingsRepository{settings: Settings{
		SystemMode: RuntimeModeObserve,
		Carriers: []CarrierSetting{{
			Carrier:                     "ctcc",
			Mode:                        RuntimeModeOff,
			DefaultBaselineRadiusMeters: 100,
		}},
	}}
	repository := &fakeRepository{denyCarrierVisibility: true}
	router := newManualBatchHandlerTestRouter(
		NewService(repository, settingsRepository),
		&actorID,
		false,
		[]uuid.UUID{uuid.New()},
	)
	recorder := serveHandlerJSON(
		router,
		http.MethodPost,
		"/api/v1/geofences/settings/preview",
		`{"system_mode":"observe","carriers":[{"carrier":"ctcc","mode":"observe","default_baseline_radius_meters":100}]}`,
	)

	assert.Equal(t, http.StatusForbidden, recorder.Code)
}

func TestHandler_UpdateSettingsRequiresAuthenticatedActor(t *testing.T) {
	router := newHandlerTestRouter(
		NewService(&fakeRepository{}, &fakeSettingsRepository{}),
		nil,
	)
	body := `{
		"system_mode":"observe",
		"carriers":[{
			"carrier":"cmcc",
			"mode":"observe",
			"default_baseline_radius_meters":100
		}]
	}`
	request := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/geofences/settings",
		bytes.NewBufferString(body),
	)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestHandler_UpdateSettingsReturnsCommittedSnapshot(t *testing.T) {
	actorID := uuid.New()
	settingsRepository := &fakeSettingsRepository{}
	router := newManualBatchHandlerTestRouter(
		NewService(&fakeRepository{}, settingsRepository),
		&actorID,
		true,
		nil,
	)
	body := `{
		"system_mode":"observe",
		"carriers":[{
			"carrier":"cmcc",
			"mode":"observe",
			"default_baseline_radius_meters":100
		}]
	}`
	request := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/geofences/settings",
		bytes.NewBufferString(body),
	)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, actorID, settingsRepository.updatedBy)
}

func TestHandler_UpdateSettingsRestrictsSystemModeToSuperAdministrator(t *testing.T) {
	actorID := uuid.New()
	settingsRepository := &fakeSettingsRepository{settings: Settings{
		SystemMode: RuntimeModeOff,
		Carriers: []CarrierSetting{{
			Carrier:                     "cmcc",
			Mode:                        RuntimeModeOff,
			DefaultBaselineRadiusMeters: 100,
		}},
	}}
	router := newManualBatchHandlerTestRouter(
		NewService(&fakeRepository{}, settingsRepository),
		&actorID,
		false,
		[]uuid.UUID{uuid.New()},
	)
	recorder := serveHandlerJSON(
		router,
		http.MethodPut,
		"/api/v1/geofences/settings",
		`{"system_mode":"observe","carriers":[{"carrier":"cmcc","mode":"off","default_baseline_radius_meters":100}]}`,
	)

	assert.Equal(t, http.StatusForbidden, recorder.Code)
	assert.Zero(t, settingsRepository.updateCalls)
}

func TestHandler_UpdateSettingsAllowsVisibleCarrierOnlyChange(t *testing.T) {
	actorID := uuid.New()
	visibleGroup := uuid.New()
	settingsRepository := &fakeSettingsRepository{settings: Settings{
		SystemMode: RuntimeModeObserve,
		Carriers: []CarrierSetting{{
			Carrier:                     "cmcc",
			Mode:                        RuntimeModeOff,
			DefaultBaselineRadiusMeters: 100,
		}},
	}}
	repository := &fakeRepository{}
	router := newManualBatchHandlerTestRouter(
		NewService(repository, settingsRepository),
		&actorID,
		false,
		[]uuid.UUID{visibleGroup},
	)
	recorder := serveHandlerJSON(
		router,
		http.MethodPut,
		"/api/v1/geofences/settings",
		`{"system_mode":"observe","carriers":[{"carrier":"cmcc","mode":"observe","default_baseline_radius_meters":100}]}`,
	)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, 1, settingsRepository.updateCalls)
	assert.Equal(t, []uuid.UUID{visibleGroup}, repository.carrierVisibilityGroups)
}

func TestHandler_CreateDraftVersionReturnsImmutableDraft(t *testing.T) {
	actorID := uuid.New()
	geofenceID := uuid.New()
	currentVersionID := uuid.New()
	repository := &fakeRepository{definition: &Definition{
		ID: geofenceID, RuleType: RuleTypePolygonAllowZone,
		Status: DefinitionStatusEnabled, CurrentVersionID: &currentVersionID,
	}}
	router := newHandlerTestRouter(
		NewService(repository, nil),
		&actorID,
	)
	body := `{
		"geometry":{"type":"Polygon","coordinates":[[[120,30],[121,30],[121,31],[120,30]]]},
		"policy":{"exit_action":"notify_only"}
	}`
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/geofences/"+geofenceID.String()+"/versions",
		bytes.NewBufferString(body),
	)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusCreated, recorder.Code)
	var envelope struct {
		Ret  int     `json:"ret"`
		Data Version `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	assert.Equal(t, int64(2), envelope.Data.Version)
	assert.Equal(t, VersionStatusDraft, envelope.Data.Status)
	assert.Equal(t, currentVersionID, *repository.definition.CurrentVersionID)
}

func TestHandler_CreateDraftVersionRequiresAuthenticatedActor(t *testing.T) {
	router := newHandlerTestRouter(
		NewService(&fakeRepository{}, nil),
		nil,
	)
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/geofences/"+uuid.NewString()+"/versions",
		bytes.NewBufferString(`{}`),
	)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestHandler_RegisterRoutesIncludesDefinitionLifecycle(t *testing.T) {
	router := newHandlerTestRouter(
		NewService(&fakeRepository{}, nil),
		nil,
	)
	routes := make(map[string]struct{})
	for _, route := range router.Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
	}
	for _, route := range []string{
		"POST /api/v1/geofences/:id/versions",
		"POST /api/v1/geofences/:id/enable-preview",
		"POST /api/v1/geofences/:id/enable",
		"POST /api/v1/geofences/:id/disable-preview",
		"POST /api/v1/geofences/:id/disable",
		"POST /api/v1/geofences/:id/archive-preview",
		"POST /api/v1/geofences/:id/archive",
	} {
		_, ok := routes[route]
		assert.True(t, ok, route)
	}
}

func TestHandler_RegisterRoutesIncludesBindingLifecycle(t *testing.T) {
	router := newHandlerTestRouter(
		NewService(&fakeRepository{}, nil),
		nil,
	)
	routes := make(map[string]struct{})
	for _, route := range router.Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
	}
	for _, route := range []string{
		"POST /api/v1/geofence-bindings/:id/suspend",
		"POST /api/v1/geofence-bindings/:id/resume",
		"DELETE /api/v1/geofence-bindings/:id",
	} {
		_, ok := routes[route]
		assert.True(t, ok, route)
	}
}

func TestHandler_DefinitionLifecycleUsesAuthBodyAndErrorEnvelope(t *testing.T) {
	geofenceID := uuid.New()
	body := `{"reason":"maintenance","preview_fingerprint":"fingerprint"}`

	t.Run("mutation requires actor", func(t *testing.T) {
		router := newHandlerTestRouter(
			NewService(&fakeRepository{}, nil),
			nil,
		)
		request := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/geofences/"+geofenceID.String()+"/disable",
			bytes.NewBufferString(body),
		)
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()

		router.ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusUnauthorized, recorder.Code)
		assertErrorEnvelope(t, recorder)
	})

	t.Run("mutation requires reason and preview fingerprint", func(t *testing.T) {
		actorID := uuid.New()
		router := newHandlerTestRouter(
			NewService(&fakeRepository{}, nil),
			&actorID,
		)
		request := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/geofences/"+geofenceID.String()+"/disable",
			bytes.NewBufferString(`{}`),
		)
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()

		router.ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusBadRequest, recorder.Code)
		assertErrorEnvelope(t, recorder)
	})

	t.Run("stale preview maps to conflict envelope", func(t *testing.T) {
		actorID := uuid.New()
		repository := &fakeRepository{
			transitionDefinitionErr: ErrStaleLifecyclePreview,
		}
		router := newHandlerTestRouter(
			NewService(repository, nil),
			&actorID,
		)
		request := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/geofences/"+geofenceID.String()+"/disable",
			bytes.NewBufferString(body),
		)
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()

		router.ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusConflict, recorder.Code)
		assertErrorEnvelope(t, recorder)
	})
}

func TestHandler_BindingLifecycleRequiresAuthenticatedActor(t *testing.T) {
	router := newHandlerTestRouter(
		NewService(&fakeRepository{}, nil),
		nil,
	)
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/geofence-bindings/"+uuid.NewString()+"/suspend",
		bytes.NewBufferString(`{"reason":"maintenance"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	assertErrorEnvelope(t, recorder)
}

func TestHandler_SuspendBindingRequiresReason(t *testing.T) {
	actorID := uuid.New()
	router := newHandlerTestRouter(
		NewService(&fakeRepository{}, nil),
		&actorID,
	)
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/geofence-bindings/"+uuid.NewString()+"/suspend",
		bytes.NewBufferString(`{}`),
	)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assertErrorEnvelope(t, recorder)
}

func TestHandler_ResumeBindingRequiresReason(t *testing.T) {
	actorID := uuid.New()
	router := newHandlerTestRouter(
		NewService(&fakeRepository{}, nil),
		&actorID,
	)
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/geofence-bindings/"+uuid.NewString()+"/resume",
		bytes.NewBufferString(`{}`),
	)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assertErrorEnvelope(t, recorder)
}

func TestHandler_BindingLifecycleReturnsUpdatedBindings(t *testing.T) {
	actorID := uuid.New()
	bindingID := uuid.New()
	repository := &fakeRepository{}
	router := newHandlerTestRouter(
		NewService(repository, nil),
		&actorID,
	)

	tests := []struct {
		name       string
		method     string
		pathSuffix string
		body       string
		wantStatus BindingStatus
	}{
		{
			name:       "suspend",
			method:     http.MethodPost,
			pathSuffix: "/suspend",
			body:       `{"reason":"maintenance"}`,
			wantStatus: BindingStatusSuspended,
		},
		{
			name:       "resume",
			method:     http.MethodPost,
			pathSuffix: "/resume",
			body:       `{"reason":"maintenance completed"}`,
			wantStatus: BindingStatusActive,
		},
		{
			name:       "remove",
			method:     http.MethodDelete,
			body:       `{"reason":"device reassigned"}`,
			wantStatus: BindingStatusRemoved,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(
				tt.method,
				"/api/v1/geofence-bindings/"+
					bindingID.String()+tt.pathSuffix,
				bytes.NewBufferString(tt.body),
			)
			if tt.body != "" {
				request.Header.Set("Content-Type", "application/json")
			}
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusOK, recorder.Code)
			var envelope struct {
				Ret  int     `json:"ret"`
				Data Binding `json:"data"`
			}
			require.NoError(
				t,
				json.Unmarshal(recorder.Body.Bytes(), &envelope),
			)
			assert.Equal(t, 1, envelope.Ret)
			assert.Equal(t, bindingID, envelope.Data.ID)
			assert.Equal(t, tt.wantStatus, envelope.Data.Status)
		})
	}
	assert.Equal(t, 3, repository.transitionBindingCalls)
}

func TestHandler_BindingLifecycleMapsConflictToStandardEnvelope(t *testing.T) {
	actorID := uuid.New()
	repository := &fakeRepository{
		transitionBindingErr: commonerrors.ErrAlreadyExists,
	}
	router := newHandlerTestRouter(
		NewService(repository, nil),
		&actorID,
	)
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/geofence-bindings/"+uuid.NewString()+"/resume",
		bytes.NewBufferString(`{"reason":"resume conflict check"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusConflict, recorder.Code)
	assertErrorEnvelope(t, recorder)
}

func assertErrorEnvelope(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	var envelope struct {
		Ret  int    `json:"ret"`
		Data any    `json:"data"`
		Msg  string `json:"msg"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	assert.Zero(t, envelope.Ret)
	assert.Nil(t, envelope.Data)
	assert.NotEmpty(t, envelope.Msg)
}
