package backup

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/model"
)

// ---------------------------------------------------------------------------
// Mock: TaskRepository
// ---------------------------------------------------------------------------

type fakeTaskRepo struct {
	tasks map[uuid.UUID]*BackupTask
}

func newFakeTaskRepo() *fakeTaskRepo {
	return &fakeTaskRepo{tasks: make(map[uuid.UUID]*BackupTask)}
}

func (m *fakeTaskRepo) Create(_ context.Context, task *BackupTask) error {
	task.ID = uuid.New()
	now := time.Now()
	task.CreatedAt = now
	task.UpdatedAt = now
	m.tasks[task.ID] = task
	return nil
}

func (m *fakeTaskRepo) GetByID(_ context.Context, id uuid.UUID) (*BackupTask, error) {
	t, ok := m.tasks[id]
	if !ok {
		return nil, nil
	}
	return t, nil
}

func (m *fakeTaskRepo) Update(_ context.Context, task *BackupTask) error {
	task.UpdatedAt = time.Now()
	m.tasks[task.ID] = task
	return nil
}

func (m *fakeTaskRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(m.tasks, id)
	return nil
}

func (m *fakeTaskRepo) List(_ context.Context, filter TaskFilter) (*model.ListResponse[BackupTask], error) {
	items := make([]BackupTask, 0, len(m.tasks))
	for _, t := range m.tasks {
		if filter.Status != nil && t.Status != *filter.Status {
			continue
		}
		if filter.TaskType != nil && t.TaskType != *filter.TaskType {
			continue
		}
		items = append(items, *t)
	}
	return model.NewListResponse(items, int64(len(items)), filter.Page, filter.PageSize), nil
}

func (m *fakeTaskRepo) CleanupOldRows(_ context.Context, _ time.Time, _ int) (int64, error) {
	return 0, nil
}

// ---------------------------------------------------------------------------
// Mock: ScheduleRepository
// ---------------------------------------------------------------------------

type fakeScheduleRepo struct {
	schedules map[uuid.UUID]*BackupSchedule
}

func newFakeScheduleRepo() *fakeScheduleRepo {
	return &fakeScheduleRepo{schedules: make(map[uuid.UUID]*BackupSchedule)}
}

func (m *fakeScheduleRepo) Create(_ context.Context, s *BackupSchedule) error {
	s.ID = uuid.New()
	now := time.Now()
	s.CreatedAt = now
	s.UpdatedAt = now
	m.schedules[s.ID] = s
	return nil
}

func (m *fakeScheduleRepo) GetByID(_ context.Context, id uuid.UUID) (*BackupSchedule, error) {
	s, ok := m.schedules[id]
	if !ok {
		return nil, nil
	}
	return s, nil
}

func (m *fakeScheduleRepo) Update(_ context.Context, s *BackupSchedule) error {
	s.UpdatedAt = time.Now()
	m.schedules[s.ID] = s
	return nil
}

func (m *fakeScheduleRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(m.schedules, id)
	return nil
}

func (m *fakeScheduleRepo) List(_ context.Context, filter ScheduleFilter) (*model.ListResponse[BackupSchedule], error) {
	items := make([]BackupSchedule, 0, len(m.schedules))
	for _, s := range m.schedules {
		if filter.Enabled != nil && s.Enabled != *filter.Enabled {
			continue
		}
		items = append(items, *s)
	}
	return model.NewListResponse(items, int64(len(items)), filter.Page, filter.PageSize), nil
}

// ---------------------------------------------------------------------------
// Mock: FTPConfigRepository
// ---------------------------------------------------------------------------

type fakeFTPConfigRepo struct {
	configs map[uuid.UUID]*FTPConfig
}

func newFakeFTPConfigRepo() *fakeFTPConfigRepo {
	return &fakeFTPConfigRepo{configs: make(map[uuid.UUID]*FTPConfig)}
}

func (m *fakeFTPConfigRepo) Create(_ context.Context, config *FTPConfig) error {
	config.ID = uuid.New()
	now := time.Now()
	config.CreatedAt = now
	config.UpdatedAt = now
	m.configs[config.ID] = config
	return nil
}

func (m *fakeFTPConfigRepo) GetByID(_ context.Context, id uuid.UUID) (*FTPConfig, error) {
	c, ok := m.configs[id]
	if !ok {
		return nil, nil
	}
	return c, nil
}

func (m *fakeFTPConfigRepo) Update(_ context.Context, config *FTPConfig) error {
	config.UpdatedAt = time.Now()
	m.configs[config.ID] = config
	return nil
}

func (m *fakeFTPConfigRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(m.configs, id)
	return nil
}

func (m *fakeFTPConfigRepo) List(_ context.Context, filter FTPConfigFilter) (*model.ListResponse[FTPConfig], error) {
	items := make([]FTPConfig, 0, len(m.configs))
	for _, c := range m.configs {
		if filter.Enabled != nil && c.Enabled != *filter.Enabled {
			continue
		}
		items = append(items, *c)
	}
	return model.NewListResponse(items, int64(len(items)), filter.Page, filter.PageSize), nil
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func setupBackupRouter(h *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h.RegisterRoutes(r.Group("/api/v1"))
	return r
}

func newTestBackupHandler() (*Handler, *fakeTaskRepo, *fakeScheduleRepo, *fakeFTPConfigRepo) {
	taskRepo := newFakeTaskRepo()
	scheduleRepo := newFakeScheduleRepo()
	ftpRepo := newFakeFTPConfigRepo()
	logger := zap.NewNop()
	svc := NewService(taskRepo, scheduleRepo, nil, logger)
	h := NewHandler(svc, ftpRepo, logger)
	return h, taskRepo, scheduleRepo, ftpRepo
}

func seedTask(repo *fakeTaskRepo, taskType TaskType, targetType string, status TaskStatus) *BackupTask {
	now := time.Now()
	t := &BackupTask{
		ID:         uuid.New(),
		TaskType:   taskType,
		TargetType: targetType,
		TargetIDs:  []string{"dev-001", "dev-002"},
		Status:     status,
		Progress:   0,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	repo.tasks[t.ID] = t
	return t
}

func seedSchedule(repo *fakeScheduleRepo, name string, cronExpr string, enabled bool) *BackupSchedule {
	now := time.Now()
	s := &BackupSchedule{
		ID:        uuid.New(),
		Name:      name,
		CronExpr:  cronExpr,
		Enabled:   enabled,
		TaskType:  TaskFull,
		TargetIDs: []string{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	repo.schedules[s.ID] = s
	return s
}

func seedFTPConfig(repo *fakeFTPConfigRepo, name string, host string, enabled bool) *FTPConfig {
	now := time.Now()
	c := &FTPConfig{
		ID:         uuid.New(),
		ConfigName: name,
		Host:       host,
		Port:       21,
		Username:   "ftpuser",
		Protocol:   "FTP",
		RemotePath: "/backup",
		Passive:    true,
		Enabled:    enabled,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	repo.configs[c.ID] = c
	return c
}

func mustMarshalBackup(t *testing.T, v interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestHandler_ListTasks(t *testing.T) {
	h, taskRepo, _, _ := newTestBackupHandler()
	router := setupBackupRouter(h)

	seedTask(taskRepo, TaskFull, "device", TaskPending)
	seedTask(taskRepo, TaskIncremental, "group", TaskCompleted)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/backup/tasks?page=1&page_size=20", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.ListResponse[BackupTask]
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, int64(2), resp.Total)
	assert.Len(t, resp.Items, 2)
	assert.Equal(t, 1, resp.Page)
	assert.Equal(t, 20, resp.PageSize)
}

func TestHandler_CreateTask(t *testing.T) {
	h, _, _, _ := newTestBackupHandler()
	router := setupBackupRouter(h)

	body := CreateTaskRequest{
		TaskType:   TaskFull,
		TargetType: "device",
		TargetIDs:  []string{"dev-100", "dev-200"},
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backup/tasks", bytes.NewReader(mustMarshalBackup(t, body)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp BackupTask
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, resp.ID)
	assert.Equal(t, TaskFull, resp.TaskType)
	assert.Equal(t, "device", resp.TargetType)
	assert.Equal(t, TaskPending, resp.Status)
	assert.Equal(t, 0, resp.Progress)
	assert.Equal(t, []string{"dev-100", "dev-200"}, resp.TargetIDs)
}

func TestHandler_GetTask(t *testing.T) {
	h, taskRepo, _, _ := newTestBackupHandler()
	router := setupBackupRouter(h)

	task := seedTask(taskRepo, TaskConfigOnly, "device", TaskRunning)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/backup/tasks/"+task.ID.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp BackupTask
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, task.ID, resp.ID)
	assert.Equal(t, TaskConfigOnly, resp.TaskType)
	assert.Equal(t, TaskRunning, resp.Status)
}

func TestHandler_DeleteTask(t *testing.T) {
	h, taskRepo, _, _ := newTestBackupHandler()
	router := setupBackupRouter(h)

	task := seedTask(taskRepo, TaskFull, "device", TaskCompleted)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/backup/tasks/"+task.ID.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Body.String())

	// Verify task was removed from the repo.
	_, exists := taskRepo.tasks[task.ID]
	assert.False(t, exists)
}

func TestHandler_CancelTask(t *testing.T) {
	h, taskRepo, _, _ := newTestBackupHandler()
	router := setupBackupRouter(h)

	task := seedTask(taskRepo, TaskFull, "device", TaskPending)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backup/tasks/"+task.ID.String()+"/cancel", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "cancelled", resp["status"])

	// Verify the task status was updated in the repo.
	assert.Equal(t, TaskCancelled, taskRepo.tasks[task.ID].Status)
}

func TestHandler_CreateSchedule(t *testing.T) {
	h, _, _, _ := newTestBackupHandler()
	router := setupBackupRouter(h)

	body := CreateScheduleRequest{
		Name:       "Daily Full Backup",
		CronExpr:   "0 2 * * *",
		Enabled:    true,
		TaskType:   TaskFull,
		TargetType: "device",
		TargetIDs:  []string{"dev-001"},
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backup/schedules", bytes.NewReader(mustMarshalBackup(t, body)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp BackupSchedule
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, resp.ID)
	assert.Equal(t, "Daily Full Backup", resp.Name)
	assert.Equal(t, "0 2 * * *", resp.CronExpr)
	assert.True(t, resp.Enabled)
	assert.Equal(t, TaskFull, resp.TaskType)
	require.NotNil(t, resp.TargetType)
	assert.Equal(t, "device", *resp.TargetType)
}

func TestHandler_ListFTPConfigs(t *testing.T) {
	h, _, _, ftpRepo := newTestBackupHandler()
	router := setupBackupRouter(h)

	seedFTPConfig(ftpRepo, "Primary FTP", "ftp.example.com", true)
	seedFTPConfig(ftpRepo, "Backup FTP", "ftp2.example.com", false)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/backup/ftp-configs?page=1&page_size=20", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.ListResponse[FTPConfig]
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, int64(2), resp.Total)
	assert.Len(t, resp.Items, 2)
}

func TestHandler_CreateFTPConfig(t *testing.T) {
	h, _, _, _ := newTestBackupHandler()
	router := setupBackupRouter(h)

	pw := "s3cret"
	body := CreateFTPConfigRequest{
		ConfigName:        "New FTP Server",
		Host:              "ftp.newhost.com",
		Port:              2121,
		Username:          "admin",
		PasswordEncrypted: &pw,
		Protocol:          "SFTP",
		RemotePath:        "/data/backup",
		Passive:           true,
		Enabled:           true,
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backup/ftp-configs", bytes.NewReader(mustMarshalBackup(t, body)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp FTPConfig
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, resp.ID)
	assert.Equal(t, "New FTP Server", resp.ConfigName)
	assert.Equal(t, "ftp.newhost.com", resp.Host)
	assert.Equal(t, 2121, resp.Port)
	assert.Equal(t, "admin", resp.Username)
	assert.Equal(t, "SFTP", resp.Protocol)
	assert.Equal(t, "/data/backup", resp.RemotePath)
	assert.True(t, resp.Passive)
	assert.True(t, resp.Enabled)
}
