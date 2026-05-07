package filemanager

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Mock: FileRepository (fmH prefix)
// ---------------------------------------------------------------------------

type fmHFileRepo struct {
	createFn  func(ctx context.Context, file *ManagedFile) error
	getByIDFn func(ctx context.Context, id uuid.UUID) (*ManagedFile, error)
	updateFn  func(ctx context.Context, file *ManagedFile) error
	deleteFn  func(ctx context.Context, id uuid.UUID) error
	listFn    func(ctx context.Context, filter FileFilter) (*model.ListResponse[ManagedFile], error)
}

func (m *fmHFileRepo) Create(ctx context.Context, file *ManagedFile) error {
	if m.createFn != nil {
		return m.createFn(ctx, file)
	}
	return nil
}
func (m *fmHFileRepo) GetByID(ctx context.Context, id uuid.UUID) (*ManagedFile, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, commonerrors.ErrNotFound
}
func (m *fmHFileRepo) Update(ctx context.Context, file *ManagedFile) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, file)
	}
	return nil
}
func (m *fmHFileRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}
func (m *fmHFileRepo) List(ctx context.Context, filter FileFilter) (*model.ListResponse[ManagedFile], error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return model.NewListResponse([]ManagedFile{}, 0, 1, 20), nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func fmHSetupRouter(repo FileRepository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// Pass nil for minioClient — only test endpoints that don't use MinIO
	svc := NewFileService(repo, nil, "test-bucket", nil, zap.NewNop())
	h := NewHandler(svc, zap.NewNop())
	h.RegisterRoutes(r.Group("/api/v1"))
	return r
}

func fmHSampleFile(id uuid.UUID) *ManagedFile {
	return &ManagedFile{
		ID:          id,
		FileName:    "config_v1.xml",
		FileType:    FileConfig,
		FileSize:    1024,
		MinIOPath:   "managed-files/config/2026-03-10/config_v1.xml",
		ContentType: "application/xml",
		Status:      FileReady,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestHandler_ListFiles_Default(t *testing.T) {
	repo := &fmHFileRepo{
		listFn: func(_ context.Context, _ FileFilter) (*model.ListResponse[ManagedFile], error) {
			return model.NewListResponse([]ManagedFile{
				{FileName: "a.xml", FileType: FileConfig, Status: FileReady},
				{FileName: "b.bin", FileType: FileFirmware, Status: FileReady},
			}, 2, 1, 20), nil
		},
	}
	router := fmHSetupRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/files", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp model.ListResponse[ManagedFile]
	response.DecodeData(t, w.Body, &resp)
	assert.Equal(t, int64(2), resp.Total)
	assert.Len(t, resp.Items, 2)
}

func TestHandler_ListFiles_WithFilter(t *testing.T) {
	repo := &fmHFileRepo{
		listFn: func(_ context.Context, f FileFilter) (*model.ListResponse[ManagedFile], error) {
			require.NotNil(t, f.FileType)
			assert.Equal(t, FileConfig, *f.FileType)
			return model.NewListResponse([]ManagedFile{
				{FileName: "a.xml", FileType: FileConfig},
			}, 1, 1, 20), nil
		},
	}
	router := fmHSetupRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/files?file_type=config", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetFileByID_Success(t *testing.T) {
	fileID := uuid.New()
	repo := &fmHFileRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*ManagedFile, error) {
			assert.Equal(t, fileID, id)
			return fmHSampleFile(fileID), nil
		},
	}
	router := fmHSetupRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/files/"+fileID.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp ManagedFile
	response.DecodeData(t, w.Body, &resp)
	assert.Equal(t, fileID, resp.ID)
	assert.Equal(t, "config_v1.xml", resp.FileName)
}

func TestHandler_GetFileByID_NotFound(t *testing.T) {
	repo := &fmHFileRepo{} // default returns ErrNotFound
	router := fmHSetupRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/files/"+uuid.New().String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_GetFileByID_InvalidUUID(t *testing.T) {
	router := fmHSetupRouter(&fmHFileRepo{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/files/not-a-uuid", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
