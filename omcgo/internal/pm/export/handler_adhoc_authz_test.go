package export

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/pm/adhoc"
)

type stubAdhocTaskReader struct {
	task *adhoc.Task
	err  error
}

func (s *stubAdhocTaskReader) Get(_ context.Context, _ uuid.UUID) (*adhoc.Task, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.task, nil
}

func TestHandler_Create_AdhocExportRejectsPrivateTaskFromOtherUser(t *testing.T) {
	adhocTaskID := uuid.New()
	repo, enq, router := newAdhocExportTestRouter(&stubAdhocTaskReader{task: &adhoc.Task{
		ID:         adhocTaskID,
		Creator:    "alice",
		Visibility: adhoc.VisibilityPrivate,
	}}, func(c *gin.Context) {
		c.Set("username", "bob")
	})

	w := performAdhocExportCreate(router, adhocTaskID)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Nil(t, repo.created)
	assert.Empty(t, enq.inserted)
}

func TestHandler_Create_AdhocExportAllowsPublicTaskFromOtherUser(t *testing.T) {
	for _, source := range []SourceType{SourcePMDashboard, SourceAdhocResult} {
		t.Run(string(source), func(t *testing.T) {
			adhocTaskID := uuid.New()
			repo, enq, router := newAdhocExportTestRouter(&stubAdhocTaskReader{task: &adhoc.Task{
				ID:         adhocTaskID,
				Creator:    "alice",
				Visibility: adhoc.VisibilityPublic,
			}}, func(c *gin.Context) {
				c.Set("username", "bob")
			})

			w := performAdhocExportCreateForSource(router, source, adhocTaskID)

			assert.Equal(t, http.StatusCreated, w.Code)
			require.NotNil(t, repo.created)
			assert.Equal(t, source, repo.created.SourceType)
			assert.Len(t, enq.inserted, 1)
		})
	}
}

func TestHandler_Create_RejectsDeprecatedAdhocSourceType(t *testing.T) {
	adhocTaskID := uuid.New()
	repo, enq, router := newAdhocExportTestRouter(&stubAdhocTaskReader{task: &adhoc.Task{
		ID:         adhocTaskID,
		Creator:    "alice",
		Visibility: adhoc.VisibilityPublic,
	}}, func(c *gin.Context) {
		c.Set("username", "alice")
	})

	w := performAdhocExportCreateForSource(router, SourceType("adhoc"), adhocTaskID)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Nil(t, repo.created)
	assert.Empty(t, enq.inserted)
}

func TestHandler_Create_AdhocExportAllowsPrivateTaskOwner(t *testing.T) {
	adhocTaskID := uuid.New()
	repo, enq, router := newAdhocExportTestRouter(&stubAdhocTaskReader{task: &adhoc.Task{
		ID:         adhocTaskID,
		Creator:    "alice",
		Visibility: adhoc.VisibilityPrivate,
	}}, func(c *gin.Context) {
		c.Set("username", "alice")
	})

	w := performAdhocExportCreate(router, adhocTaskID)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.NotNil(t, repo.created)
	assert.Len(t, enq.inserted, 1)
}

func TestHandler_Create_AdhocExportAllowsPrivateTaskForAdminRole(t *testing.T) {
	adhocTaskID := uuid.New()
	repo, enq, router := newAdhocExportTestRouter(&stubAdhocTaskReader{task: &adhoc.Task{
		ID:         adhocTaskID,
		Creator:    "alice",
		Visibility: adhoc.VisibilityPrivate,
	}}, func(c *gin.Context) {
		c.Set("username", "bob")
		c.Set("roles", []string{"admin"})
	})

	w := performAdhocExportCreate(router, adhocTaskID)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.NotNil(t, repo.created)
	assert.Len(t, enq.inserted, 1)
}

func TestHandler_Create_AdhocExportFailsClosedWithoutTaskReader(t *testing.T) {
	adhocTaskID := uuid.New()
	repo, enq, router := newAdhocExportTestRouter(nil, func(c *gin.Context) {
		c.Set("username", "bob")
	})

	w := performAdhocExportCreate(router, adhocTaskID)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Nil(t, repo.created)
	assert.Empty(t, enq.inserted)
}

func TestHandler_List_HidesPrivateAdhocExportsFromOtherUser(t *testing.T) {
	privateTaskID := uuid.New()
	publicTaskID := uuid.New()
	privateExportID := uuid.New()
	publicExportID := uuid.New()
	dashboardExportID := uuid.New()
	reader := &multiAdhocTaskReader{tasks: map[uuid.UUID]*adhoc.Task{
		privateTaskID: {ID: privateTaskID, Creator: "alice", Visibility: adhoc.VisibilityPrivate},
		publicTaskID:  {ID: publicTaskID, Creator: "alice", Visibility: adhoc.VisibilityPublic},
	}}
	repo, _, router := newAdhocExportTestRouter(reader, func(c *gin.Context) {
		c.Set("username", "bob")
	})
	repo.listResult = []Task{
		{ID: privateExportID, SourceType: SourcePMDashboard, Params: adhocExportParams(privateTaskID), Status: StatusSucceeded},
		{ID: publicExportID, SourceType: SourceAdhocResult, Params: adhocExportParams(publicTaskID), Status: StatusSucceeded},
		{ID: dashboardExportID, SourceType: SourceDashboard, Status: StatusSucceeded},
	}

	req := httptest.NewRequest(http.MethodGet, "/pm/exports", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), privateExportID.String())
	assert.Contains(t, w.Body.String(), publicExportID.String())
	assert.Contains(t, w.Body.String(), dashboardExportID.String())
}

func TestHandler_Download_RejectsPrivateAdhocExportFromOtherUser(t *testing.T) {
	adhocTaskID := uuid.New()
	exportID := uuid.New()
	reader := &stubAdhocTaskReader{task: &adhoc.Task{
		ID:         adhocTaskID,
		Creator:    "alice",
		Visibility: adhoc.VisibilityPrivate,
	}}
	repo, _, router := newAdhocExportTestRouter(reader, func(c *gin.Context) {
		c.Set("username", "bob")
	})
	repo.getTask = &Task{
		ID:         exportID,
		SourceType: SourcePMDashboard,
		Params:     adhocExportParams(adhocTaskID),
		Status:     StatusSucceeded,
		Bucket:     "pm",
		FilePath:   "private.csv",
	}

	req := httptest.NewRequest(http.MethodGet, "/pm/exports/"+exportID.String()+"/download", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "adhoc task is not visible to current user")
}

func TestHandler_Delete_RejectsPrivateAdhocExportFromOtherUser(t *testing.T) {
	adhocTaskID := uuid.New()
	exportID := uuid.New()
	reader := &stubAdhocTaskReader{task: &adhoc.Task{
		ID:         adhocTaskID,
		Creator:    "alice",
		Visibility: adhoc.VisibilityPrivate,
	}}
	repo, _, router := newAdhocExportTestRouter(reader, func(c *gin.Context) {
		c.Set("username", "bob")
	})
	repo.getTask = &Task{
		ID:         exportID,
		SourceType: SourceAdhocResult,
		Params:     adhocExportParams(adhocTaskID),
		Status:     StatusSucceeded,
	}

	req := httptest.NewRequest(http.MethodDelete, "/pm/exports/"+exportID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Empty(t, repo.deleted)
}

func newAdhocExportTestRouter(reader AdhocTaskReader, inject func(*gin.Context)) (*stubRepo, *stubEnqueuer, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	repo := &stubRepo{
		createID: uuid.New(),
		getTask:  &Task{ID: uuid.New(), SourceType: SourceAdhocResult, Status: StatusPending},
	}
	enq := &stubEnqueuer{insertID: uuid.New()}
	h := NewHandler(NewService(repo, enq), nil, zap.NewNop())
	h.SetAdhocTaskReader(reader)
	router := gin.New()
	if inject != nil {
		router.Use(func(c *gin.Context) {
			inject(c)
			c.Next()
		})
	}
	h.RegisterRoutes(router.Group(""))
	return repo, enq, router
}

func performAdhocExportCreate(router http.Handler, taskID uuid.UUID) *httptest.ResponseRecorder {
	return performAdhocExportCreateForSource(router, SourceAdhocResult, taskID)
}

func performAdhocExportCreateForSource(router http.Handler, source SourceType, taskID uuid.UUID) *httptest.ResponseRecorder {
	body := []byte(`{"source_type":"` + string(source) + `","params":{"task_id":"` + taskID.String() + `"}}`)
	req := httptest.NewRequest(http.MethodPost, "/pm/exports", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

type multiAdhocTaskReader struct {
	tasks map[uuid.UUID]*adhoc.Task
}

func (m *multiAdhocTaskReader) Get(_ context.Context, id uuid.UUID) (*adhoc.Task, error) {
	if task, ok := m.tasks[id]; ok {
		return task, nil
	}
	return nil, adhoc.ErrNotFound
}

func adhocExportParams(taskID uuid.UUID) []byte {
	return []byte(fmt.Sprintf(`{"task_id":"%s"}`, taskID.String()))
}
