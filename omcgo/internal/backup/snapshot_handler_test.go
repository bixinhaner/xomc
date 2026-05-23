package backup

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// snapshotHandlerSetup returns a *Handler with a wired SnapshotService backed
// by in-memory fakes. fakeMover lets tests assert MinIO interactions.
func snapshotHandlerSetup(t *testing.T) (*Handler, *fakeSnapshotRepo, *fakeMover) {
	t.Helper()
	repo := newFakeSnapshotRepo()
	mover := &fakeMover{}
	svc := NewSnapshotService(repo, mover, nil, nil, "config-snapshots", zap.NewNop())

	h := NewHandler(nil, nil, zap.NewNop())
	h.SetSnapshotService(svc)
	return h, repo, mover
}

func snapshotRouter(h *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	rg := r.Group("/api/v1")
	h.RegisterRoutes(rg)
	return r
}

// ─────────────────────────────────────────────────────────────────────────
// List / Get / BatchGet
// ─────────────────────────────────────────────────────────────────────────

func TestSnapshotHandler_List_Empty(t *testing.T) {
	h, _, _ := snapshotHandlerSetup(t)
	r := snapshotRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/backup/config-snapshots", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var env struct {
		Ret  int `json:"ret"`
		Data struct {
			Items []ConfigSnapshot `json:"items"`
			Total int64            `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, 1, env.Ret)
	assert.Empty(t, env.Data.Items)
	assert.EqualValues(t, 0, env.Data.Total)
}

func TestSnapshotHandler_Get_NotFound(t *testing.T) {
	h, _, _ := snapshotHandlerSetup(t)
	r := snapshotRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/backup/config-snapshots/SN_GHOST", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSnapshotHandler_Get_Found(t *testing.T) {
	h, repo, _ := snapshotHandlerSetup(t)
	require.NoError(t, repo.Upsert(t.Context(), &ConfigSnapshot{
		SerialNumber: "SN001", FileName: "SN001_CFG.xml", FileExt: "xml",
		ObjectBucket: "config-snapshots", ObjectPath: "SN001_CFG.xml",
		Source: SnapshotSourceManualUpload,
	}))
	r := snapshotRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/backup/config-snapshots/SN001", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var env struct {
		Ret  int             `json:"ret"`
		Data *ConfigSnapshot `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, 1, env.Ret)
	require.NotNil(t, env.Data)
	assert.Equal(t, "SN001", env.Data.SerialNumber)
	assert.Equal(t, "SN001_CFG.xml", env.Data.FileName)
}

func TestSnapshotHandler_BatchGet_MixedFoundAndMissing(t *testing.T) {
	h, repo, _ := snapshotHandlerSetup(t)
	require.NoError(t, repo.Upsert(t.Context(), &ConfigSnapshot{
		SerialNumber: "SN001", FileName: "SN001_CFG.xml", FileExt: "xml",
		ObjectBucket: "config-snapshots", ObjectPath: "SN001_CFG.xml",
		Source: SnapshotSourceManualUpload,
	}))
	r := snapshotRouter(h)

	body, _ := json.Marshal(BatchGetSnapshotsRequest{
		SerialNumbers: []string{"SN001", "SN_MISSING_A", "SN_MISSING_B"},
	})
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/backup/config-snapshots/batch-get", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var env struct {
		Ret  int                       `json:"ret"`
		Data BatchGetSnapshotsResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Contains(t, env.Data.Found, "SN001")
	assert.ElementsMatch(t, []string{"SN_MISSING_A", "SN_MISSING_B"}, env.Data.Missing)
}

// ─────────────────────────────────────────────────────────────────────────
// Import (multipart)
// ─────────────────────────────────────────────────────────────────────────

func TestSnapshotHandler_Import_Success(t *testing.T) {
	h, repo, mover := snapshotHandlerSetup(t)
	r := snapshotRouter(h)

	body, contentType := buildMultipart(t, map[string][]byte{
		"SN001_CFG.xml": []byte("<config/>"),
		"SN002_CFG.nv":  {0x01, 0x02},
	})
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/backup/config-snapshots/import", body)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var env struct {
		Ret  int                   `json:"ret"`
		Data SnapshotImportResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.ElementsMatch(t, []string{"SN001", "SN002"}, env.Data.Succeeded)
	assert.Empty(t, env.Data.Failed)
	assert.Len(t, mover.putCalls, 2)
	assert.Len(t, repo.upserts, 2)
}

func TestSnapshotHandler_Import_InvalidNameReported(t *testing.T) {
	h, _, _ := snapshotHandlerSetup(t)
	r := snapshotRouter(h)

	body, contentType := buildMultipart(t, map[string][]byte{
		"SN001_CFG.xml": []byte("ok"),
		"weird.txt":     []byte("nope"),
	})
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/backup/config-snapshots/import", body)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var env struct {
		Ret  int                   `json:"ret"`
		Data SnapshotImportResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Contains(t, env.Data.Succeeded, "SN001")
	require.Len(t, env.Data.Failed, 1)
	assert.Equal(t, "weird.txt", env.Data.Failed[0].FileName)
	assert.Equal(t, ImportErrInvalidName, env.Data.Failed[0].ErrorCode)
	assert.Contains(t, env.Data.Failed[0].Message, "weird.txt")
}

func TestSnapshotHandler_Import_NoFiles(t *testing.T) {
	h, _, _ := snapshotHandlerSetup(t)
	r := snapshotRouter(h)

	body, contentType := buildMultipart(t, map[string][]byte{})
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/backup/config-snapshots/import", body)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─────────────────────────────────────────────────────────────────────────
// Delete / BatchDelete
// ─────────────────────────────────────────────────────────────────────────

func TestSnapshotHandler_Delete(t *testing.T) {
	h, repo, mover := snapshotHandlerSetup(t)
	require.NoError(t, repo.Upsert(t.Context(), &ConfigSnapshot{
		SerialNumber: "SN001", FileName: "SN001_CFG.xml", FileExt: "xml",
		ObjectBucket: "config-snapshots", ObjectPath: "SN001_CFG.xml",
		Source: SnapshotSourceManualUpload,
	}))
	r := snapshotRouter(h)

	req := httptest.NewRequest(http.MethodDelete,
		"/api/v1/backup/config-snapshots/SN001", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, repo.rows, "SN001")
	assert.Len(t, mover.rmCalls, 1)
}

func TestSnapshotHandler_BatchDelete(t *testing.T) {
	h, repo, _ := snapshotHandlerSetup(t)
	for _, sn := range []string{"SN001", "SN002"} {
		require.NoError(t, repo.Upsert(t.Context(), &ConfigSnapshot{
			SerialNumber: sn, FileName: sn + "_CFG.xml", FileExt: "xml",
			ObjectBucket: "config-snapshots", ObjectPath: sn + "_CFG.xml",
			Source: SnapshotSourceManualUpload,
		}))
	}
	r := snapshotRouter(h)

	body, _ := json.Marshal(BatchDeleteSnapshotsRequest{
		SerialNumbers: []string{"SN001", "SN002"},
	})
	// 路由已迁移到 POST /batch-delete（见 handler.go），避免 DELETE+body 被代理吞。
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/backup/config-snapshots/batch-delete", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var env struct {
		Ret  int                          `json:"ret"`
		Data BatchDeleteSnapshotsResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.ElementsMatch(t, []string{"SN001", "SN002"}, env.Data.Succeeded)
}

// ─────────────────────────────────────────────────────────────────────────
// 503 when service not wired
// ─────────────────────────────────────────────────────────────────────────

func TestSnapshotHandler_503WhenNotWired(t *testing.T) {
	h := NewHandler(nil, nil, zap.NewNop())
	// NOT calling SetSnapshotService — must 503 on all endpoints.
	r := snapshotRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/backup/config-snapshots", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// ─────────────────────────────────────────────────────────────────────────
// helpers
// ─────────────────────────────────────────────────────────────────────────

// buildMultipart constructs a multipart form whose "files" field carries each
// (filename, content) pair from m.
func buildMultipart(t *testing.T, m map[string][]byte) (io.Reader, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for name, body := range m {
		fw, err := mw.CreateFormFile("files", name)
		require.NoError(t, err)
		_, err = fw.Write(body)
		require.NoError(t, err)
	}
	require.NoError(t, mw.Close())
	return &buf, mw.FormDataContentType()
}
