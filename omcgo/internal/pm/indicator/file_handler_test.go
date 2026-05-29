package indicator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

// mockFileRepository 是 FileRepository 的内存 stub,可注入 countByLoadedFrom / err 行为。
type mockFileRepository struct {
	count        int
	countErr     error
	deleteRows   int
	deleteErr    error
	calledDelete bool
	calledTech   string
	calledPath   string
}

func (m *mockFileRepository) CountByLoadedFrom(ctx context.Context, tech, loadedFrom string) (int, error) {
	if m.countErr != nil {
		return 0, m.countErr
	}
	return m.count, nil
}

func (m *mockFileRepository) DeleteByLoadedFrom(ctx context.Context, tech, loadedFrom string) (int, error) {
	m.calledDelete = true
	m.calledTech = tech
	m.calledPath = loadedFrom
	if m.deleteErr != nil {
		return 0, m.deleteErr
	}
	return m.deleteRows, nil
}

// newTestRouter 装配 FileHandler + 测试 gin engine(silent mode)。
func newTestRouter(t *testing.T, repo FileRepository, baseDir string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	rg := r.Group("/api/v1")
	h := NewFileHandler(repo, baseDir, zap.NewNop())
	h.RegisterRoutes(rg)
	return r
}

// writeCustomXML 在 baseDir/indicator-library-custom/<tech>/<file>.xml 写入空 indicator XML。
func writeCustomXML(t *testing.T, baseDir, tech, name string) string {
	t.Helper()
	rel := filepath.Join(CustomDirSubdir, tech, name)
	abs := filepath.Join(baseDir, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	body := []byte(`<indicatorModel platform="X" indicatorCount="0"></indicatorModel>`)
	if err := os.WriteFile(abs, body, 0o644); err != nil {
		t.Fatalf("write %s: %v", abs, err)
	}
	return filepath.ToSlash(rel)
}

// ── parseCustomTech 表驱动 ─────────────────────────────────────────

func TestParseCustomTech(t *testing.T) {
	cases := []struct {
		name     string
		path     string
		wantTech string
		wantErr  bool
	}{
		{"valid enb", "indicator-library-custom/enb/MY.xml", "enb", false},
		{"valid gsm", "indicator-library-custom/gsm/X.xml", "gsm", false},
		{"valid gnb", "indicator-library-custom/gnb/Y.xml", "gnb", false},

		{"builtin path rejected", "indicator-library/enb/ALL.xml", "", true},
		{"empty path rejected", "", "", true},
		{"wrong prefix rejected", "other/enb/X.xml", "", true},
		{"too few segments", "indicator-library-custom/MY.xml", "", true},
		{"too many segments", "indicator-library-custom/enb/sub/MY.xml", "", true},
		{"invalid tech", "indicator-library-custom/lte/MY.xml", "", true},
		{"traversal in filename", "indicator-library-custom/enb/../etc/passwd", "", true},
		{"missing filename", "indicator-library-custom/enb/", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseCustomTech(tc.path)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantTech, got)
			}
		})
	}
}

// ── DeleteFile HTTP 行为 ────────────────────────────────────────────

func TestDeleteFile_BuiltinRejected_403(t *testing.T) {
	baseDir := t.TempDir()
	repo := &mockFileRepository{}
	r := newTestRouter(t, repo, baseDir)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/indicators/files/indicator-library/enb/ALL.xml", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, fmt.Sprintf("code=%d", global.ErrCodeIndicatorBuiltinNotDeletable))
	assert.False(t, repo.calledDelete, "守门应在 repo 之前命中")
}

func TestDeleteFile_UnknownPathRejected_403(t *testing.T) {
	baseDir := t.TempDir()
	repo := &mockFileRepository{}
	r := newTestRouter(t, repo, baseDir)

	// 裸文件名(无前缀)被分类为 Unknown → 守门拒绝
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/indicators/files/BARE.xml", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestDeleteFile_InvalidTechSegment_400(t *testing.T) {
	baseDir := t.TempDir()
	repo := &mockFileRepository{}
	r := newTestRouter(t, repo, baseDir)

	// custom 前缀 + 非法 tech 段(lte 不在 enb/gsm/gnb)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/indicators/files/indicator-library-custom/lte/MY.xml", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteFile_NotFound_404(t *testing.T) {
	baseDir := t.TempDir()
	// 文件不在 + DB 0 行 → 404
	repo := &mockFileRepository{count: 0}
	r := newTestRouter(t, repo, baseDir)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/indicators/files/indicator-library-custom/enb/NO_FILE.xml", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteFile_HappyPath_200(t *testing.T) {
	baseDir := t.TempDir()
	loadedFrom := writeCustomXML(t, baseDir, "enb", "MY.xml")
	repo := &mockFileRepository{count: 3, deleteRows: 3}
	r := newTestRouter(t, repo, baseDir)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/indicators/files/"+loadedFrom, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	// 物理文件应被 rename 为 .deleted.<ts>
	_, statErr := os.Stat(filepath.Join(baseDir, loadedFrom))
	assert.True(t, os.IsNotExist(statErr), "原文件应被 rename")

	// 同目录下应有 .deleted.* 备份
	dir := filepath.Dir(filepath.Join(baseDir, loadedFrom))
	entries, _ := os.ReadDir(dir)
	found := false
	for _, e := range entries {
		if strings.Contains(e.Name(), ".deleted.") {
			found = true
			break
		}
	}
	assert.True(t, found, "备份文件应存在")

	// repo 被以 tech=enb 调用
	assert.True(t, repo.calledDelete)
	assert.Equal(t, "enb", repo.calledTech)
	assert.Equal(t, loadedFrom, repo.calledPath)

	// 响应体含 rows_affected
	var resp struct {
		Data map[string]any `json:"data"`
		Msg  string         `json:"msg"`
		Ret  int            `json:"ret"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(3), resp.Data["rows_affected"])
	assert.Equal(t, "enb", resp.Data["tech"])
	assert.Equal(t, true, resp.Data["deleted"])
}

func TestDeleteFile_FileGoneButDBExists_StillDeletes(t *testing.T) {
	baseDir := t.TempDir()
	// 写 → 立即外部删除模拟"文件已 gone",DB 仍有 1 行
	writeCustomXML(t, baseDir, "gsm", "ORPHAN.xml")
	_ = os.Remove(filepath.Join(baseDir, "indicator-library-custom/gsm/ORPHAN.xml"))
	repo := &mockFileRepository{count: 1, deleteRows: 1}
	r := newTestRouter(t, repo, baseDir)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/indicators/files/indicator-library-custom/gsm/ORPHAN.xml", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// 文件已不在但 DB 残留 → 仍允许清 DB 残留(rename 返 ENOENT 路径走 file_already_gone 分支)
	assert.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	assert.True(t, repo.calledDelete)
}

func TestDeleteFile_BackupFailedReadOnly_500(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root 用户绕过权限校验,跳过备份失败模拟")
	}
	baseDir := t.TempDir()
	writeCustomXML(t, baseDir, "enb", "RO.xml")
	// 把父目录置 read-only,让 rename 失败
	parent := filepath.Join(baseDir, "indicator-library-custom/enb")
	if err := os.Chmod(parent, 0o555); err != nil {
		t.Fatalf("chmod ro: %v", err)
	}
	defer func() { _ = os.Chmod(parent, 0o755) }()

	repo := &mockFileRepository{count: 1, deleteRows: 1}
	r := newTestRouter(t, repo, baseDir)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/indicators/files/indicator-library-custom/enb/RO.xml", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), fmt.Sprintf("code=%d", global.ErrCodeIndicatorBackupFailed))
	assert.False(t, repo.calledDelete, "备份失败时绝不调 repo")
}

func TestDeleteFile_DBErrorRollsBackFile(t *testing.T) {
	baseDir := t.TempDir()
	loadedFrom := writeCustomXML(t, baseDir, "gnb", "ROLLBACK.xml")
	repo := &mockFileRepository{count: 1, deleteErr: errors.New("simulated DB error")}
	r := newTestRouter(t, repo, baseDir)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/indicators/files/"+loadedFrom, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	// 文件应被回滚回原位置(rename .deleted.<ts> → 原 abs)
	_, statErr := os.Stat(filepath.Join(baseDir, loadedFrom))
	assert.NoError(t, statErr, "DB 失败后原文件应被回滚")
}

// ── 配套基础设施测试 ────────────────────────────────────────────

func TestEnsureBaseDir(t *testing.T) {
	baseDir := t.TempDir()
	err := EnsureBaseDir(context.Background(), baseDir)
	assert.NoError(t, err)
	for _, tech := range []string{"enb", "gsm", "gnb"} {
		_, err := os.Stat(filepath.Join(baseDir, CustomDirSubdir, tech))
		assert.NoError(t, err, "tech=%s 子目录应存在", tech)
	}

	// 幂等:重复跑不报错
	err = EnsureBaseDir(context.Background(), baseDir)
	assert.NoError(t, err)
}

func TestValidateTech(t *testing.T) {
	assert.NoError(t, validateTech("enb"))
	assert.NoError(t, validateTech("gsm"))
	assert.NoError(t, validateTech("gnb"))
	assert.Error(t, validateTech("lte"))
	assert.Error(t, validateTech(""))
	assert.Error(t, validateTech("ENB")) // 大小写敏感
}
