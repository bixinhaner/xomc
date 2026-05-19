package notification

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
)

// setupHandlerTest builds a gin router with notification handler routes
// registered. The optional userID is injected as the Gin context user_id
// (UUID derived from userID string for deterministic test data) so handler
// can read it via admin.UserIDStringFromCtx (T-0157 C5/C10)。
//
// 实现细节：userID 字符串（如 "alice"）被映射到稳定的 uuid v5，再 c.Set 进
// CtxKeyUserID；mockRepository.seed 时也需要传同一 uuid string 形式作为 UserID。
func setupHandlerTest(t *testing.T, userID string) (*gin.Engine, *mockRepository) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	repo := newMockRepository()
	svc := newTestService(repo, nil)
	h := NewHandler(svc, zap.NewNop())

	r := gin.New()
	r.Use(func(c *gin.Context) {
		if userID != "" {
			c.Set("user_id", testUUIDForUser(userID))
			c.Set("username", userID) // 保留旧 key 兼容审计日志等
		}
		c.Next()
	})
	h.RegisterRoutes(r.Group(""))
	return r, repo
}

// testUUIDForUser 把测试用户名映射到稳定 uuid（同名一致），供 setupHandlerTest 和
// 测试断言数据共用。基于 namespace + name 的 uuid v5 保证跨测试可复现。
func testUUIDForUser(name string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceDNS, []byte("test-user:"+name))
}

// testUserID 给测试代码取 string 形式（mock seed / 断言用）。
func testUserID(name string) string { return testUUIDForUser(name).String() }

// ---------- NewHandler ----------

func TestNewHandler_Basic(t *testing.T) {
	repo := newMockRepository()
	svc := newTestService(repo, nil)
	h := NewHandler(svc, zap.NewNop())
	require.NotNil(t, h)
	require.NotNil(t, h.service)
	require.NotNil(t, h.logger)
}

// ---------- List ----------

func TestHandler_List_OK(t *testing.T) {
	r, repo := setupHandlerTest(t, "alice")
	repo.seed(&Notification{UserID: testUserID("alice"), Type: NotifTypeAlarm, Title: "n1"})
	repo.seed(&Notification{UserID: testUserID("alice"), Type: NotifTypeSystem, Title: "n2"})
	repo.seed(&Notification{UserID: testUserID("bob"), Type: NotifTypeAlarm, Title: "n3"})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/notifications", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp model.ListResponse[Notification]
	response.DecodeData(t, w.Body, &resp)
	assert.Equal(t, int64(2), resp.Total)
}

func TestHandler_List_Unauthorized(t *testing.T) {
	r, _ := setupHandlerTest(t, "")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/notifications", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_List_FilterByType(t *testing.T) {
	r, repo := setupHandlerTest(t, "alice")
	repo.seed(&Notification{UserID: testUserID("alice"), Type: NotifTypeAlarm, Title: "a1"})
	repo.seed(&Notification{UserID: testUserID("alice"), Type: NotifTypeSystem, Title: "s1"})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/notifications?type=alarm", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp model.ListResponse[Notification]
	response.DecodeData(t, w.Body, &resp)
	assert.Equal(t, int64(1), resp.Total)
	assert.Equal(t, NotifTypeAlarm, resp.Items[0].Type)
}

func TestHandler_List_FilterByIsRead_True(t *testing.T) {
	r, repo := setupHandlerTest(t, "alice")
	repo.seed(&Notification{UserID: testUserID("alice"), Type: NotifTypeAlarm, IsRead: true, Title: "read"})
	repo.seed(&Notification{UserID: testUserID("alice"), Type: NotifTypeAlarm, IsRead: false, Title: "unread"})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/notifications?is_read=true", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp model.ListResponse[Notification]
	response.DecodeData(t, w.Body, &resp)
	assert.Equal(t, int64(1), resp.Total)
	assert.True(t, resp.Items[0].IsRead)
}

func TestHandler_List_FilterByIsRead_False(t *testing.T) {
	r, repo := setupHandlerTest(t, "alice")
	repo.seed(&Notification{UserID: testUserID("alice"), IsRead: true})
	repo.seed(&Notification{UserID: testUserID("alice"), IsRead: false})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/notifications?is_read=false", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp model.ListResponse[Notification]
	response.DecodeData(t, w.Body, &resp)
	assert.Equal(t, int64(1), resp.Total)
	assert.False(t, resp.Items[0].IsRead)
}

func TestHandler_List_BadQueryParams(t *testing.T) {
	r, _ := setupHandlerTest(t, "alice")

	// page=0 violates the binding `min=1`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/notifications?page=0", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_List_RepoError(t *testing.T) {
	r, repo := setupHandlerTest(t, "alice")
	repo.listErr = errBoom

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/notifications", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---------- GetUnreadCount ----------

func TestHandler_GetUnreadCount_OK(t *testing.T) {
	r, repo := setupHandlerTest(t, "alice")
	repo.seed(&Notification{UserID: testUserID("alice"), IsRead: false})
	repo.seed(&Notification{UserID: testUserID("alice"), IsRead: false})
	repo.seed(&Notification{UserID: testUserID("alice"), IsRead: true})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/notifications/unread-count", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]int64
	response.DecodeData(t, w.Body, &resp)
	assert.Equal(t, int64(2), resp["count"])
}

func TestHandler_GetUnreadCount_Unauthorized(t *testing.T) {
	r, _ := setupHandlerTest(t, "")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/notifications/unread-count", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetUnreadCount_RepoError(t *testing.T) {
	r, repo := setupHandlerTest(t, "alice")
	repo.getUnreadCountErr = errBoom

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/notifications/unread-count", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---------- MarkRead ----------

func TestHandler_MarkRead_OK(t *testing.T) {
	r, repo := setupHandlerTest(t, "alice")
	n := repo.seed(&Notification{UserID: testUserID("alice")})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/notifications/"+n.ID.String()+"/read", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	response.DecodeData(t, w.Body, nil)
}

func TestHandler_MarkRead_BadID(t *testing.T) {
	r, _ := setupHandlerTest(t, "alice")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/notifications/not-a-uuid/read", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_MarkRead_Unauthorized(t *testing.T) {
	r, _ := setupHandlerTest(t, "")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/notifications/"+uuid.New().String()+"/read", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_MarkRead_NotFound(t *testing.T) {
	r, _ := setupHandlerTest(t, "alice")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/notifications/"+uuid.New().String()+"/read", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------- MarkAllRead ----------

func TestHandler_MarkAllRead_OK(t *testing.T) {
	r, repo := setupHandlerTest(t, "alice")
	repo.seed(&Notification{UserID: testUserID("alice"), IsRead: false})
	repo.seed(&Notification{UserID: testUserID("alice"), IsRead: false})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/notifications/read-all", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	response.DecodeData(t, w.Body, nil)
}

func TestHandler_MarkAllRead_Unauthorized(t *testing.T) {
	r, _ := setupHandlerTest(t, "")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/notifications/read-all", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_MarkAllRead_RepoError(t *testing.T) {
	r, repo := setupHandlerTest(t, "alice")
	repo.markAllReadErr = errBoom

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/notifications/read-all", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---------- Delete ----------

func TestHandler_Delete_OK(t *testing.T) {
	r, repo := setupHandlerTest(t, "alice")
	n := repo.seed(&Notification{UserID: testUserID("alice")})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/notifications/"+n.ID.String(), nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	response.DecodeData(t, w.Body, nil)
}

func TestHandler_Delete_BadID(t *testing.T) {
	r, _ := setupHandlerTest(t, "alice")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/notifications/not-a-uuid", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Delete_Unauthorized(t *testing.T) {
	r, _ := setupHandlerTest(t, "")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/notifications/"+uuid.New().String(), nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_Delete_NotFound(t *testing.T) {
	r, _ := setupHandlerTest(t, "alice")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/notifications/"+uuid.New().String(), nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------- DeleteAll (T-0157 C4) ----------

func TestHandler_DeleteAll_OK(t *testing.T) {
	r, repo := setupHandlerTest(t, "alice")
	repo.seed(&Notification{UserID: testUserID("alice"), Type: NotifTypeAlarm, Title: "a1"})
	repo.seed(&Notification{UserID: testUserID("alice"), Type: NotifTypeSystem, Title: "a2"})
	repo.seed(&Notification{UserID: testUserID("alice"), Type: NotifTypeTaskComplete, Title: "a3"})
	bobMsg := repo.seed(&Notification{UserID: testUserID("bob"), Title: "b1"})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/notifications", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Deleted int64 `json:"deleted"`
	}
	response.DecodeData(t, w.Body, &resp)
	assert.Equal(t, int64(3), resp.Deleted)

	// 验证 bob 的消息未受影响（user 隔离）
	_, err := repo.GetByID(t.Context(), bobMsg.ID)
	assert.NoError(t, err, "bob 的消息应该还在")
}

func TestHandler_DeleteAll_NoMessages_ReturnsZero(t *testing.T) {
	r, _ := setupHandlerTest(t, "alice")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/notifications", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Deleted int64 `json:"deleted"`
	}
	response.DecodeData(t, w.Body, &resp)
	assert.Equal(t, int64(0), resp.Deleted)
}

func TestHandler_DeleteAll_Unauthorized(t *testing.T) {
	r, _ := setupHandlerTest(t, "")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/notifications", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_DeleteAll_RepoError(t *testing.T) {
	r, repo := setupHandlerTest(t, "alice")
	repo.deleteAllErr = errBoom

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/notifications", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---------- RegisterRoutes ----------

func TestHandler_RegisterRoutes_AllPathsReachable(t *testing.T) {
	r, repo := setupHandlerTest(t, "alice")
	// seed a record so DELETE /:id returns 204 (not 404 due to no row).
	n := repo.seed(&Notification{UserID: testUserID("alice")})
	n2 := repo.seed(&Notification{UserID: testUserID("alice")})

	// Each registered route should resolve to its handler — i.e. status code is not
	// 404 because of "no matching route" (ie gin-level not-found, body empty),
	// and not the rare case of method mismatch returning 405.
	cases := []struct {
		method, path string
	}{
		{http.MethodGet, "/notifications"},
		{http.MethodGet, "/notifications/unread-count"},
		{http.MethodPut, "/notifications/" + n.ID.String() + "/read"},
		{http.MethodPut, "/notifications/read-all"},
		{http.MethodDelete, "/notifications/" + n2.ID.String()},
	}
	for _, c := range cases {
		t.Run(c.method+" "+c.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(c.method, c.path, nil)
			r.ServeHTTP(w, req)
			// Routes that match always return 2xx in this seeded scenario.
			assert.True(t, w.Code >= 200 && w.Code < 400,
				"unexpected status %d for %s %s", w.Code, c.method, c.path)
		})
	}
}
