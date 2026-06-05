package definition

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/global"
)

// mockFileRepository 是 FileRepository 的内存 stub。
type mockFileRepository struct {
	count        int
	countErr     error
	deleteRows   int
	deleteErr    error
	calledDelete bool
	calledPath   string

	neTypeExists    bool
	neTypeExistsErr error
}

func (m *mockFileRepository) CountByLoadedFrom(ctx context.Context, loadedFrom string) (int, error) {
	if m.countErr != nil {
		return 0, m.countErr
	}
	return m.count, nil
}

func (m *mockFileRepository) DeleteByLoadedFrom(ctx context.Context, loadedFrom string) (int, error) {
	m.calledDelete = true
	m.calledPath = loadedFrom
	if m.deleteErr != nil {
		return 0, m.deleteErr
	}
	return m.deleteRows, nil
}

func (m *mockFileRepository) NeTypeExists(ctx context.Context, neType string) (bool, error) {
	if m.neTypeExistsErr != nil {
		return false, m.neTypeExistsErr
	}
	return m.neTypeExists, nil
}

// stubReloader 记录 ReloadOne 调用,可注入失败。
type stubReloader struct {
	calls int
	err   error
}

func (s *stubReloader) ReloadOne(ctx context.Context, name string) error {
	s.calls++
	return s.err
}

// newTestRouter 装配 FileHandler(service=nil → 跳过 orphan cleanup / RefreshCache)。
func newTestRouter(t *testing.T, repo FileRepository, reloader Reloader, baseDir string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	rg := r.Group("/api/v1")
	h := NewFileHandler(repo, nil, reloader, baseDir, zap.NewNop())
	h.RegisterRoutes(rg)
	return r
}

// writeCustomXML 在 baseDir/alarm-definitions/<file>.xml 写空 alarmModel + sidecar 标记。
func writeCustomXML(t *testing.T, baseDir, name string) string {
	t.Helper()
	rel := filepath.Join(BuiltinDirSubdir, name)
	abs := filepath.Join(baseDir, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	body := []byte(`<alarmModel neType="ENB"><alarms></alarms></alarmModel>`)
	if err := os.WriteFile(abs, body, 0o644); err != nil {
		t.Fatalf("write %s: %v", abs, err)
	}
	if err := os.WriteFile(abs+CustomMarkerSuffix, []byte{}, 0o644); err != nil {
		t.Fatalf("write sidecar: %v", err)
	}
	return filepath.ToSlash(rel)
}

// writeBuiltinXML 在 baseDir/alarm-definitions/<file>.xml 写空 alarmModel(无 sidecar = builtin)。
func writeBuiltinXML(t *testing.T, baseDir, name string) string {
	t.Helper()
	rel := filepath.Join(BuiltinDirSubdir, name)
	abs := filepath.Join(baseDir, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(abs, []byte(`<alarmModel neType="ENB"></alarmModel>`), 0o644); err != nil {
		t.Fatalf("write %s: %v", abs, err)
	}
	return filepath.ToSlash(rel)
}

// buildMultipart 构造 multipart body:name 字段 + file part(file 自身 filename 被忽略)。
func buildMultipart(t *testing.T, name string, content []byte) (*bytes.Buffer, string) {
	t.Helper()
	buf := &bytes.Buffer{}
	w := multipart.NewWriter(buf)
	if name != "" {
		_ = w.WriteField("name", name)
	}
	part, err := w.CreateFormFile("file", "ignored.xml")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	_, _ = part.Write(content)
	_ = w.Close()
	return buf, w.FormDataContentType()
}

// ── validAlarmFilePath ──────────────────────────────────────────────

func TestValidAlarmFilePath(t *testing.T) {
	cases := []struct {
		name string
		path string
		want bool
	}{
		{"builtin two-segment", "alarm-definitions/ENB.xml", true},
		{"custom-prefix rejected", "alarm-definitions-custom/MY.xml", false},
		{"bare filename rejected", "ENB.xml", false},
		{"wrong prefix rejected", "other/ENB.xml", false},
		{"too many segments", "alarm-definitions/sub/ENB.xml", false},
		{"traversal rejected", "alarm-definitions/../etc.xml", false},
		{"missing filename", "alarm-definitions/", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, validAlarmFilePath(tc.path))
		})
	}
}

// ── parseUploadedNeType ─────────────────────────────────────────────

func TestParseUploadedNeType(t *testing.T) {
	// XML neType 属性优先
	nt := parseUploadedNeType([]byte(`<alarmModel neType="GNB"></alarmModel>`), "myfile.xml")
	assert.Equal(t, "GNB", nt)

	// 缺省回退文件名(去 .xml 大写)
	nt = parseUploadedNeType([]byte(`<alarmModel></alarmModel>`), "enb.xml")
	assert.Equal(t, "ENB", nt)
}

// ── UploadXML ───────────────────────────────────────────────────────

func TestUpload_HappyPath_201(t *testing.T) {
	baseDir := t.TempDir()
	reloader := &stubReloader{}
	repo := &mockFileRepository{}
	r := newTestRouter(t, repo, reloader, baseDir)

	xml := []byte(`<alarmModel neType="MY_NE"><alarms></alarms></alarmModel>`)
	body, ct := buildMultipart(t, "MY", xml)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alarm-definitions/upload-xml", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code, "body=%s", w.Body.String())

	// 物理文件写进 baseDir/alarm-definitions/MY.xml
	target := filepath.Join(baseDir, BuiltinDirSubdir, "MY.xml")
	got, err := os.ReadFile(target)
	assert.NoError(t, err)
	assert.Equal(t, xml, got)

	// sidecar 标记写入 → custom 可删
	_, scErr := os.Stat(target + CustomMarkerSuffix)
	assert.NoError(t, scErr, "sidecar 标记应写入")
	assert.True(t, IsDeletable(baseDir, "alarm-definitions/MY.xml"))

	assert.Equal(t, 1, reloader.calls)

	var resp struct {
		Data map[string]any `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "alarm-definitions/MY.xml", resp.Data["loaded_from"])
	assert.Equal(t, "MY_NE", resp.Data["ne_type"])
	assert.Equal(t, true, resp.Data["reloaded"])
}

// 文件名唯一性:同名 <name>.xml 已存在 → 409,不覆盖、不重载。
func TestUpload_NameConflict_409(t *testing.T) {
	baseDir := t.TempDir()
	writeBuiltinXML(t, baseDir, "MY.xml")
	reloader := &stubReloader{}
	r := newTestRouter(t, &mockFileRepository{}, reloader, baseDir)

	body, ct := buildMultipart(t, "MY", []byte(`<alarmModel neType="X"></alarmModel>`))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alarm-definitions/upload-xml", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusConflict, w.Code, "body=%s", w.Body.String())
	assert.Equal(t, 0, reloader.calls)
}

// 内容主键唯一性:neType 已在 DB → 409,文件不落地。
func TestUpload_NeTypeConflict_409(t *testing.T) {
	baseDir := t.TempDir()
	reloader := &stubReloader{}
	repo := &mockFileRepository{neTypeExists: true}
	r := newTestRouter(t, repo, reloader, baseDir)

	body, ct := buildMultipart(t, "NEW_NAME", []byte(`<alarmModel neType="DUP"></alarmModel>`))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alarm-definitions/upload-xml", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusConflict, w.Code, "body=%s", w.Body.String())
	assert.Contains(t, w.Body.String(), "ne_type")
	assert.Equal(t, 0, reloader.calls)
	_, err := os.Stat(filepath.Join(baseDir, BuiltinDirSubdir, "NEW_NAME.xml"))
	assert.True(t, os.IsNotExist(err), "neType 冲突时文件不应落地")
}

func TestUpload_MissingName_400(t *testing.T) {
	baseDir := t.TempDir()
	r := newTestRouter(t, &mockFileRepository{}, &stubReloader{}, baseDir)
	body, ct := buildMultipart(t, "", []byte(`<alarmModel neType="X"></alarmModel>`))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alarm-definitions/upload-xml", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), fmt.Sprintf("code=%d", global.ErrCodeAlarmUploadInvalidName))
}

func TestUpload_InvalidName_400(t *testing.T) {
	baseDir := t.TempDir()
	r := newTestRouter(t, &mockFileRepository{}, &stubReloader{}, baseDir)
	// name 含点 → <name>.xml = "MY.foo.xml" 多扩展名,被白名单拒
	body, ct := buildMultipart(t, "MY.foo", []byte(`<alarmModel neType="X"></alarmModel>`))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alarm-definitions/upload-xml", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), fmt.Sprintf("code=%d", global.ErrCodeAlarmUploadInvalidName))
}

func TestUpload_InvalidRoot_400(t *testing.T) {
	baseDir := t.TempDir()
	r := newTestRouter(t, &mockFileRepository{}, &stubReloader{}, baseDir)
	body, ct := buildMultipart(t, "MY", []byte(`<indicatorModel platform="X"></indicatorModel>`))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alarm-definitions/upload-xml", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), fmt.Sprintf("code=%d", global.ErrCodeAlarmUploadInvalidRoot))
}

func TestUpload_NoFile_400(t *testing.T) {
	baseDir := t.TempDir()
	r := newTestRouter(t, &mockFileRepository{}, &stubReloader{}, baseDir)
	buf := &bytes.Buffer{}
	mw := multipart.NewWriter(buf)
	_ = mw.WriteField("name", "MY")
	_ = mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alarm-definitions/upload-xml", buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpload_ReloadFailedStillSucceeds(t *testing.T) {
	baseDir := t.TempDir()
	reloader := &stubReloader{err: errors.New("simulated reload fail")}
	r := newTestRouter(t, &mockFileRepository{}, reloader, baseDir)
	body, ct := buildMultipart(t, "MY", []byte(`<alarmModel neType="X"></alarmModel>`))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alarm-definitions/upload-xml", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
	var resp struct {
		Data map[string]any `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, false, resp.Data["reloaded"])
	assert.Equal(t, 1, reloader.calls)
}

// ── DeleteFile ──────────────────────────────────────────────────────

// builtin(无 sidecar)不可删 → 403。
func TestDeleteFile_BuiltinForbidden_403(t *testing.T) {
	baseDir := t.TempDir()
	writeBuiltinXML(t, baseDir, "ENB.xml")
	repo := &mockFileRepository{count: 5, deleteRows: 5}
	r := newTestRouter(t, repo, &stubReloader{}, baseDir)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/alarm-definitions/files/alarm-definitions/ENB.xml", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code, "body=%s", w.Body.String())
	assert.False(t, repo.calledDelete)
}

// custom(有 sidecar)happy path → 200 + 文件 rename 备份 + sidecar 移除。
func TestDeleteFile_HappyPath_200(t *testing.T) {
	baseDir := t.TempDir()
	loadedFrom := writeCustomXML(t, baseDir, "MY.xml")
	repo := &mockFileRepository{count: 3, deleteRows: 3}
	r := newTestRouter(t, repo, &stubReloader{}, baseDir)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/alarm-definitions/files/"+loadedFrom, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

	abs := filepath.Join(baseDir, loadedFrom)
	_, statErr := os.Stat(abs)
	assert.True(t, os.IsNotExist(statErr), "原文件应被 rename")

	// sidecar 应被移除
	_, scErr := os.Stat(abs + CustomMarkerSuffix)
	assert.True(t, os.IsNotExist(scErr), "sidecar 应被移除")

	// 同目录有 .deleted.* 备份
	entries, _ := os.ReadDir(filepath.Dir(abs))
	found := false
	for _, e := range entries {
		if strings.Contains(e.Name(), ".deleted.") {
			found = true
		}
	}
	assert.True(t, found, "备份文件应存在")

	assert.True(t, repo.calledDelete)
	assert.Equal(t, loadedFrom, repo.calledPath)
}

func TestDeleteFile_DBErrorRollsBackFile(t *testing.T) {
	baseDir := t.TempDir()
	loadedFrom := writeCustomXML(t, baseDir, "ROLLBACK.xml")
	repo := &mockFileRepository{count: 1, deleteErr: errors.New("simulated DB error")}
	r := newTestRouter(t, repo, &stubReloader{}, baseDir)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/alarm-definitions/files/"+loadedFrom, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	// 文件应被回滚
	_, statErr := os.Stat(filepath.Join(baseDir, loadedFrom))
	assert.NoError(t, statErr, "DB 失败后原文件应被回滚")
}

func TestDeleteFile_CustomPrefixRejected_400(t *testing.T) {
	baseDir := t.TempDir()
	repo := &mockFileRepository{}
	r := newTestRouter(t, repo, &stubReloader{}, baseDir)
	// 旧 custom 前缀不再合法(单目录后唯一前缀 = alarm-definitions/)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/alarm-definitions/files/alarm-definitions-custom/MY.xml", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, repo.calledDelete)
}

func TestEnsureBaseDir(t *testing.T) {
	baseDir := t.TempDir()
	assert.NoError(t, EnsureBaseDir(baseDir))
	_, err := os.Stat(filepath.Join(baseDir, BuiltinDirSubdir))
	assert.NoError(t, err, "builtin 目录应存在")
	// 幂等
	assert.NoError(t, EnsureBaseDir(baseDir))
}
