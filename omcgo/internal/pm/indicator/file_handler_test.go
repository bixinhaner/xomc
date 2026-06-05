package indicator

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
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/global"
)

// mockFileRepository 是 FileRepository 的内存 stub,可注入各方法返回值/错误。
type mockFileRepository struct {
	count        int
	countErr     error
	deleteRows   int
	deleteErr    error
	calledDelete bool
	calledTech   string
	calledPath   string

	summary       []PlatformSummary
	summaryErr    error
	listByTech    map[string][]FileGroup // tech → groups
	listByTechErr error

	// UpsertFileDescription stub(2026-06-02:按 (tech, platform) 维度)
	upsertDescTech     string
	upsertDescPlatform string
	upsertDescValue    string
	upsertDescErr      error

	// T-0180 P1.5: DeleteOrphansBefore stub
	orphanRows    map[string]int // tech → rows to "delete" (deterministic per tech)
	orphanErr     error
	orphanCalls   []string    // 记录每次调用的 tech 序列
	orphanCutoffs []time.Time // 记录每次 before 时刻

	// 内容主键(platform)唯一性 stub
	platformExists    bool
	platformExistsErr error
}

func (m *mockFileRepository) PlatformExists(ctx context.Context, tech, platform string) (bool, error) {
	if m.platformExistsErr != nil {
		return false, m.platformExistsErr
	}
	return m.platformExists, nil
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

func (m *mockFileRepository) SummaryByTech(ctx context.Context) ([]PlatformSummary, error) {
	if m.summaryErr != nil {
		return nil, m.summaryErr
	}
	return m.summary, nil
}

func (m *mockFileRepository) UpsertFileDescription(ctx context.Context, tech, platform, description string) error {
	m.upsertDescTech = tech
	m.upsertDescPlatform = platform
	m.upsertDescValue = description
	return m.upsertDescErr
}

func (m *mockFileRepository) ListFilesByTech(ctx context.Context, tech string) ([]FileGroup, error) {
	if m.listByTechErr != nil {
		return nil, m.listByTechErr
	}
	if m.listByTech == nil {
		return nil, nil
	}
	return m.listByTech[tech], nil
}

func (m *mockFileRepository) DeleteOrphansBefore(ctx context.Context, tech string, before time.Time) (int, error) {
	m.orphanCalls = append(m.orphanCalls, tech)
	m.orphanCutoffs = append(m.orphanCutoffs, before)
	if m.orphanErr != nil {
		return 0, m.orphanErr
	}
	if m.orphanRows == nil {
		return 0, nil
	}
	return m.orphanRows[tech], nil
}

// stubReloader 是测试用 Reloader,可注入失败行为或记录调用。
type stubReloader struct {
	calls   int
	lastCtx context.Context
	err     error
	sleep   time.Duration // 模拟 Reload 耗时,验证 start/before 时序(P1.5 StartMonotonicity)
}

func (s *stubReloader) ReloadOne(ctx context.Context, name string) error {
	s.calls++
	s.lastCtx = ctx
	if s.sleep > 0 {
		time.Sleep(s.sleep)
	}
	return s.err
}

// newTestRouter 装配 FileHandler + 测试 gin engine(silent mode)。
// reloader 可传 nil,DELETE/Summary/ListFiles 测试不需要 Reload。
func newTestRouter(t *testing.T, repo FileRepository, baseDir string) *gin.Engine {
	t.Helper()
	return newTestRouterWithReloader(t, repo, nil, baseDir)
}

func newTestRouterWithReloader(t *testing.T, repo FileRepository, reloader Reloader, baseDir string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	rg := r.Group("/api/v1")
	// cache 传 nil:测试不验证缓存刷新(BumpCacheVersion 走 service+redis,单测无 redis)
	h := NewFileHandler(repo, reloader, nil, baseDir, zap.NewNop())
	h.RegisterRoutes(rg)
	return r
}

// writeCustomXML 在 baseDir/indicator-library/<tech>/<file>.xml 写入空 indicator XML
// + sidecar 标记(三库 XML 导入重构:custom 文件落地同一目录树,sidecar 判来源)。
func writeCustomXML(t *testing.T, baseDir, tech, name string) string {
	t.Helper()
	rel := filepath.Join(BuiltinDirSubdir, tech, name)
	abs := filepath.Join(baseDir, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	body := []byte(`<indicatorModel platform="X" indicatorCount="0"></indicatorModel>`)
	if err := os.WriteFile(abs, body, 0o644); err != nil {
		t.Fatalf("write %s: %v", abs, err)
	}
	// sidecar 标记:UI 上传的 custom 文件(2026-06-04 sidecar 模型)。
	if err := os.WriteFile(abs+CustomMarkerSuffix, []byte{}, 0o644); err != nil {
		t.Fatalf("write sidecar %s: %v", abs+CustomMarkerSuffix, err)
	}
	return filepath.ToSlash(rel)
}

// writeBuiltinENBXML 在 baseDir/indicator-library/enb/<file>.xml 写入空 indicator XML
// (无 sidecar = builtin;同名冲突测试需预置)。
func writeBuiltinENBXML(t *testing.T, baseDir, name string) string {
	t.Helper()
	rel := filepath.Join(BuiltinDirSubdir, "enb", name)
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

// ── parseFileTech 表驱动 ─────────────────────────────────────────
//
// 三库 XML 导入重构(2026-06-04 单目录):所有 XML 同住 indicator-library/ 目录树;
// custom 前缀路径不再存在,带该前缀的路径一律被拒。

func TestParseFileTech(t *testing.T) {
	cases := []struct {
		name     string
		path     string
		wantTech string
		wantErr  bool
	}{
		// 单目录树(enb/gsm/gnb 子目录 + 根级单文件)
		{"enb subdir", "indicator-library/enb/ALL.xml", "enb", false},
		{"gsm subdir", "indicator-library/gsm/X.xml", "gsm", false},
		{"gnb subdir", "indicator-library/gnb/Y.xml", "gnb", false},
		{"gsm root single file", "indicator-library/GSM.xml", "gsm", false},
		{"gnb root single file", "indicator-library/GNB.xml", "gnb", false},

		{"empty path rejected", "", "", true},
		{"wrong prefix rejected", "other/enb/X.xml", "", true},
		{"custom prefix rejected", "indicator-library-custom/enb/MY.xml", "", true},
		{"bare filename rejected", "BARE.xml", "", true},
		{"too many segments", "indicator-library/enb/sub/MY.xml", "", true},
		{"invalid tech subdir", "indicator-library/lte/X.xml", "", true},
		{"unknown root file", "indicator-library/OTHER.xml", "", true},
		{"traversal in filename", "indicator-library/enb/../etc/passwd", "", true},
		{"missing filename", "indicator-library/enb/", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseFileTech(tc.path)
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

// 2026-06-04 用户决策:builtin(当前目录 XML 加载的内置数据)不可删 → DELETE 返 403。
func TestDeleteFile_BuiltinForbidden_403(t *testing.T) {
	baseDir := t.TempDir()
	// 在 builtin enb 子目录写一个文件
	abs := filepath.Join(baseDir, BuiltinDirSubdir, "enb", "ALL.xml")
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(abs, []byte(`<indicatorModel platform="X"></indicatorModel>`), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	repo := &mockFileRepository{count: 5, deleteRows: 5}
	r := newTestRouter(t, repo, baseDir)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/indicators/files/indicator-library/enb/ALL.xml", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code, "body=%s", w.Body.String())
	assert.False(t, repo.calledDelete)
}

// builtin 根级 GSM.xml 也是内置 → 403,不可删。
func TestDeleteFile_BuiltinGsmRootForbidden_403(t *testing.T) {
	baseDir := t.TempDir()
	abs := filepath.Join(baseDir, BuiltinDirSubdir, "GSM.xml")
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(abs, []byte(`<indicatorModel platform="BSC"></indicatorModel>`), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	repo := &mockFileRepository{count: 3, deleteRows: 3}
	r := newTestRouter(t, repo, baseDir)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/indicators/files/indicator-library/GSM.xml", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code, "body=%s", w.Body.String())
	assert.False(t, repo.calledDelete)
}

func TestDeleteFile_BarePathRejected_400(t *testing.T) {
	baseDir := t.TempDir()
	repo := &mockFileRepository{}
	r := newTestRouter(t, repo, baseDir)

	// 裸文件名(无前缀)无法推断 tech → 400
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/indicators/files/BARE.xml", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, repo.calledDelete)
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
	// custom 文件:有 sidecar(过删除守门)但 .xml 已不在 + DB 0 行 → 404
	scDir := filepath.Join(baseDir, BuiltinDirSubdir, "enb")
	if err := os.MkdirAll(scDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scDir, "NO_FILE.xml"+CustomMarkerSuffix), []byte{}, 0o644); err != nil {
		t.Fatalf("write sidecar: %v", err)
	}
	repo := &mockFileRepository{count: 0}
	r := newTestRouter(t, repo, baseDir)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/indicators/files/indicator-library/enb/NO_FILE.xml", nil)
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
	loadedFrom := writeCustomXML(t, baseDir, "gsm", "ORPHAN.xml")
	_ = os.Remove(filepath.Join(baseDir, loadedFrom))
	repo := &mockFileRepository{count: 1, deleteRows: 1}
	r := newTestRouter(t, repo, baseDir)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/indicators/files/"+loadedFrom, nil)
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
	parent := filepath.Join(baseDir, BuiltinDirSubdir, "enb")
	if err := os.Chmod(parent, 0o555); err != nil {
		t.Fatalf("chmod ro: %v", err)
	}
	defer func() { _ = os.Chmod(parent, 0o755) }()

	repo := &mockFileRepository{count: 1, deleteRows: 1}
	r := newTestRouter(t, repo, baseDir)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/indicators/files/indicator-library/enb/RO.xml", nil)
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
	// 2026-06-03 用户决策:上传写 builtin 目录,EnsureBaseDir 确保 builtin enb 子目录存在
	_, err = os.Stat(filepath.Join(baseDir, BuiltinDirSubdir, "enb"))
	assert.NoError(t, err, "builtin enb 子目录应存在")

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

// ── P1.4 Summary HTTP 行为 ───────────────────────────────────────────────

func TestSummary_HappyPath(t *testing.T) {
	baseDir := t.TempDir()
	repo := &mockFileRepository{
		// 2026-06-02:Summary 调整为"一个平台一条"粒度
		summary: []PlatformSummary{
			{Tech: "enb", Platform: "ALL", Indicators: 700},
			{Tech: "enb", Platform: "BLQ", Indicators: 463},
			{Tech: "gsm", Platform: "BSC", Indicators: 73},
			{Tech: "gnb", Platform: "BaiBNQ", Indicators: 211},
		},
	}
	r := newTestRouter(t, repo, baseDir)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/indicators/summary", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

	var resp struct {
		Data struct {
			Items []PlatformSummary `json:"items"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Len(t, resp.Data.Items, 4)
	assert.Equal(t, "enb", resp.Data.Items[0].Tech)
	assert.Equal(t, "ALL", resp.Data.Items[0].Platform)
	assert.Equal(t, 700, resp.Data.Items[0].Indicators)
}

func TestSummary_RepoError_500(t *testing.T) {
	baseDir := t.TempDir()
	repo := &mockFileRepository{summaryErr: errors.New("db down")}
	r := newTestRouter(t, repo, baseDir)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/indicators/summary", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ── P1.4 ListFiles HTTP 行为 ─────────────────────────────────────────────

func TestListFiles_DBAndDiskMerge(t *testing.T) {
	baseDir := t.TempDir()
	// 物理:indicator-library/enb/UPLOADED_NOT_LOADED.xml(custom + sidecar,DB 0 行)
	writeCustomXML(t, baseDir, "enb", "UPLOADED_NOT_LOADED.xml")
	// LOADED.xml 是 DB-only 行(磁盘无 .xml,OnDisk=false),但来源 custom → 单独写 sidecar 标记。
	loadedSidecar := filepath.Join(baseDir, BuiltinDirSubdir, "enb", "LOADED.xml"+CustomMarkerSuffix)
	if err := os.WriteFile(loadedSidecar, []byte{}, 0o644); err != nil {
		t.Fatalf("write LOADED sidecar: %v", err)
	}
	// DB:builtin/enb/ALL.xml(123 条,无 sidecar)+ custom/enb/LOADED.xml(7 条,有 sidecar)
	repo := &mockFileRepository{
		listByTech: map[string][]FileGroup{
			"enb": {
				{LoadedFrom: "indicator-library/enb/ALL.xml", Count: 123},
				{LoadedFrom: "indicator-library/enb/LOADED.xml", Count: 7},
			},
		},
	}
	r := newTestRouter(t, repo, baseDir)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/indicators/files?tech=enb", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

	var resp struct {
		Data struct {
			Items []fileEntry `json:"items"`
			Tech  string      `json:"tech"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "enb", resp.Data.Tech)
	// 3 行:DB 2 行 + 物理 uploaded-not-loaded 1 行
	assert.Len(t, resp.Data.Items, 3)

	// 索引便于断言
	byLF := map[string]fileEntry{}
	for _, it := range resp.Data.Items {
		byLF[it.LoadedFrom] = it
	}

	// 1. builtin/ALL — 无 sidecar → builtin 不可删;DB 有 + 物理无 → OnDisk=false
	if it, ok := byLF["indicator-library/enb/ALL.xml"]; ok {
		assert.Equal(t, "builtin", it.Source)
		assert.False(t, it.Deletable)
		assert.Equal(t, 123, it.Count)
	}
	// 2. custom/LOADED — 有 sidecar → custom 可删;DB 有 + 物理无(.xml 没写) → OnDisk=false
	if it, ok := byLF["indicator-library/enb/LOADED.xml"]; ok {
		assert.Equal(t, "custom", it.Source)
		assert.True(t, it.Deletable)
		assert.Equal(t, 7, it.Count)
	}
	// 3. uploaded-not-loaded — DB 无 + 物理有(+sidecar) → Count=0 OnDisk=true custom 可删
	if it, ok := byLF["indicator-library/enb/UPLOADED_NOT_LOADED.xml"]; ok {
		assert.Equal(t, "custom", it.Source)
		assert.True(t, it.Deletable)
		assert.Equal(t, 0, it.Count)
		assert.True(t, it.OnDisk)
	}
}

func TestListFiles_InvalidTech_400(t *testing.T) {
	baseDir := t.TempDir()
	repo := &mockFileRepository{}
	r := newTestRouter(t, repo, baseDir)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/indicators/files?tech=lte", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ── P1.4 UploadXML HTTP 行为 ─────────────────────────────────────────────

// buildMultipart 构造 multipart/form-data,body 含 "name" 字段 + "file" part。
// 三库 XML 导入重构:上传以 name 表单字段定文件名,上传文件自身 filename 被忽略。
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

func TestUpload_HappyPath_201(t *testing.T) {
	baseDir := t.TempDir()
	reloader := &stubReloader{}
	repo := &mockFileRepository{}
	r := newTestRouterWithReloader(t, repo, reloader, baseDir)

	xml := []byte(`<indicatorModel platform="MY_PLATFORM" indicatorCount="0"></indicatorModel>`)
	body, ct := buildMultipart(t, "MY", xml)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/indicators/upload-xml?tech=enb", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code, "body=%s", w.Body.String())

	// 物理文件写进 baseDir/indicator-library/enb/MY.xml
	target := filepath.Join(baseDir, BuiltinDirSubdir, "enb", "MY.xml")
	got, err := os.ReadFile(target)
	assert.NoError(t, err)
	assert.Equal(t, xml, got)

	// sidecar 标记写入 → custom
	_, scErr := os.Stat(target + CustomMarkerSuffix)
	assert.NoError(t, scErr, "sidecar 标记应写入")
	assert.True(t, IsDeletable(baseDir, "indicator-library/enb/MY.xml"))

	// Reloader 被调一次(destructive 重载内部调 ReloadOne)
	assert.Equal(t, 1, reloader.calls)

	// 响应体含 loaded_from / platform
	var resp struct {
		Data map[string]any `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "indicator-library/enb/MY.xml", resp.Data["loaded_from"])
	assert.Equal(t, "MY_PLATFORM", resp.Data["platform"])
	assert.Equal(t, true, resp.Data["reloaded"])
}

// GSM/GNB 上传落地三制式子目录(gsm/gnb),不再覆盖根级 GSM.xml/GNB.xml。
func TestUpload_GsmLandsInSubdir_201(t *testing.T) {
	baseDir := t.TempDir()
	reloader := &stubReloader{}
	r := newTestRouterWithReloader(t, &mockFileRepository{}, reloader, baseDir)

	xml := []byte(`<indicatorModel platform="MY_BSC" deviceType="GSM" indicatorCount="0"></indicatorModel>`)
	body, ct := buildMultipart(t, "MY_GSM", xml)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/indicators/upload-xml?tech=gsm", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code, "body=%s", w.Body.String())

	// 落地 indicator-library/gsm/MY_GSM.xml,根级 GSM.xml 未被触碰
	_, err := os.Stat(filepath.Join(baseDir, BuiltinDirSubdir, "gsm", "MY_GSM.xml"))
	assert.NoError(t, err)
	_, rootErr := os.Stat(filepath.Join(baseDir, BuiltinDirSubdir, "GSM.xml"))
	assert.True(t, os.IsNotExist(rootErr), "根级 GSM.xml 不应被上传创建")
}

// 文件名唯一性:同名 <name>.xml 已存在 → 409,不覆盖。
func TestUpload_NameConflict_409(t *testing.T) {
	baseDir := t.TempDir()
	writeBuiltinENBXML(t, baseDir, "MY.xml")
	reloader := &stubReloader{}
	r := newTestRouterWithReloader(t, &mockFileRepository{}, reloader, baseDir)

	xml := []byte(`<indicatorModel platform="X" indicatorCount="0"></indicatorModel>`)
	body, ct := buildMultipart(t, "MY", xml)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/indicators/upload-xml?tech=enb", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusConflict, w.Code, "body=%s", w.Body.String())
	assert.Equal(t, 0, reloader.calls, "冲突时不触发 Reload")
}

// 内容主键唯一性:platform 已在该 tech 表 → 409。
func TestUpload_PlatformConflict_409(t *testing.T) {
	baseDir := t.TempDir()
	reloader := &stubReloader{}
	repo := &mockFileRepository{platformExists: true}
	r := newTestRouterWithReloader(t, repo, reloader, baseDir)

	xml := []byte(`<indicatorModel platform="DUP" indicatorCount="0"></indicatorModel>`)
	body, ct := buildMultipart(t, "NEW_NAME", xml)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/indicators/upload-xml?tech=enb", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusConflict, w.Code, "body=%s", w.Body.String())
	assert.Contains(t, w.Body.String(), "platform")
	assert.Equal(t, 0, reloader.calls)
	// 文件不应落地(平台冲突在写盘前拒绝)
	_, err := os.Stat(filepath.Join(baseDir, BuiltinDirSubdir, "enb", "NEW_NAME.xml"))
	assert.True(t, os.IsNotExist(err))
}

// 缺 name 字段 → 400(name 为空 → ".xml" 不过白名单)。
func TestUpload_MissingName_400(t *testing.T) {
	baseDir := t.TempDir()
	r := newTestRouter(t, &mockFileRepository{}, baseDir)
	body, ct := buildMultipart(t, "", []byte(`<indicatorModel platform="X"></indicatorModel>`))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/indicators/upload-xml?tech=enb", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), fmt.Sprintf("code=%d", global.ErrCodeIndicatorUploadInvalidName))
}

func TestUpload_InvalidTech_400(t *testing.T) {
	baseDir := t.TempDir()
	r := newTestRouter(t, &mockFileRepository{}, baseDir)
	body, ct := buildMultipart(t, "MY", []byte(`<indicatorModel platform="X"></indicatorModel>`))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/indicators/upload-xml?tech=lte", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), fmt.Sprintf("code=%d", global.ErrCodeIndicatorUploadInvalidTech))
}

func TestUpload_InvalidName_400(t *testing.T) {
	baseDir := t.TempDir()
	r := newTestRouter(t, &mockFileRepository{}, baseDir)
	// name 含点 → <name>.xml = "MY.foo.xml" 多扩展名,被白名单拒
	body, ct := buildMultipart(t, "MY.foo", []byte(`<indicatorModel platform="X"></indicatorModel>`))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/indicators/upload-xml?tech=enb", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), fmt.Sprintf("code=%d", global.ErrCodeIndicatorUploadInvalidName))
}

func TestUpload_DeviceTypeMismatch_400(t *testing.T) {
	baseDir := t.TempDir()
	r := newTestRouter(t, &mockFileRepository{}, baseDir)
	// 上传 tech=enb 但 XML 内 deviceType="GSM"
	body, ct := buildMultipart(t, "MY", []byte(`<indicatorModel platform="X" deviceType="GSM"></indicatorModel>`))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/indicators/upload-xml?tech=enb", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), fmt.Sprintf("code=%d", global.ErrCodeIndicatorUploadInvalidRoot))
}

func TestUpload_NoFile_400(t *testing.T) {
	baseDir := t.TempDir()
	r := newTestRouter(t, &mockFileRepository{}, baseDir)
	// 有 name 字段但无 file part
	buf := &bytes.Buffer{}
	mw := multipart.NewWriter(buf)
	_ = mw.WriteField("name", "MY")
	_ = mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/indicators/upload-xml?tech=enb", buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpload_ReloadFailedStillSucceeds(t *testing.T) {
	baseDir := t.TempDir()
	reloader := &stubReloader{err: errors.New("simulated reload fail")}
	r := newTestRouterWithReloader(t, &mockFileRepository{}, reloader, baseDir)
	xml := []byte(`<indicatorModel platform="X" indicatorCount="0"></indicatorModel>`)
	body, ct := buildMultipart(t, "MY", xml)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/indicators/upload-xml?tech=enb", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// Upload 成功(文件已写),Reload 失败仅响应 reloaded=false
	assert.Equal(t, http.StatusCreated, w.Code)
	var resp struct {
		Data map[string]any `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, false, resp.Data["reloaded"])
	assert.Equal(t, 1, reloader.calls)
}
