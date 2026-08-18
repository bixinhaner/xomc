package definition

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
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
	"github.com/google/uuid"
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

	// 导入 XML 覆盖调整:neType → 归属文件 stub
	neTypeLoadedFroms    []string
	neTypeLoadedFromsErr error

	// #268 生成模式:ne_type → 手工新增定义 stub
	manualDefs    []ResolvedDefinition
	manualDefsErr error
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

func (m *mockFileRepository) LoadedFromsByNeType(ctx context.Context, neType string) ([]string, error) {
	if m.neTypeLoadedFromsErr != nil {
		return nil, m.neTypeLoadedFromsErr
	}
	if m.neTypeLoadedFroms != nil {
		return m.neTypeLoadedFroms, nil
	}
	// 兼容旧 stub:neTypeExists=true 但未配置归属 → 视为有冲突但路径未知
	if m.neTypeExists {
		return []string{BuiltinDirPrefix + neType + ".xml"}, nil
	}
	return nil, nil
}

func (m *mockFileRepository) ListManualByNeType(ctx context.Context, neType string) ([]ResolvedDefinition, error) {
	if m.manualDefsErr != nil {
		return nil, m.manualDefsErr
	}
	return m.manualDefs, nil
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
	// dictRefresher 传 nil:测试不验证导入后字典刷新(走 DB,单测无)
	h := NewFileHandler(repo, nil, reloader, nil, baseDir, zap.NewNop())
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

// buildMultipart 构造 multipart body:仅 file part(名称取自 XML neType 属性,
// file 自身 filename 被忽略)。
func buildMultipart(t *testing.T, content []byte) (*bytes.Buffer, string) {
	t.Helper()
	buf := &bytes.Buffer{}
	w := multipart.NewWriter(buf)
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
	// XML neType 属性是唯一来源
	nt := parseUploadedNeType([]byte(`<alarmModel neType="GNB"></alarmModel>`))
	assert.Equal(t, "GNB", nt)

	// 缺省 → 空(由 handler 拒绝,不再回退文件名)
	nt = parseUploadedNeType([]byte(`<alarmModel></alarmModel>`))
	assert.Equal(t, "", nt)
}

// ── UploadXML ───────────────────────────────────────────────────────

func TestUpload_HappyPath_201(t *testing.T) {
	baseDir := t.TempDir()
	reloader := &stubReloader{}
	repo := &mockFileRepository{}
	r := newTestRouter(t, repo, reloader, baseDir)

	xml := []byte(`<alarmModel neType="MY_NE"><alarms></alarms></alarmModel>`)
	body, ct := buildMultipart(t, xml)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alarm-definitions/upload-xml", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code, "body=%s", w.Body.String())

	// 名称取自 neType:物理文件写进 baseDir/alarm-definitions/MY_NE.xml
	target := filepath.Join(baseDir, BuiltinDirSubdir, "MY_NE.xml")
	got, err := os.ReadFile(target)
	assert.NoError(t, err)
	assert.Equal(t, xml, got)

	// sidecar 标记写入 → custom 可删
	_, scErr := os.Stat(target + CustomMarkerSuffix)
	assert.NoError(t, scErr, "sidecar 标记应写入")
	assert.True(t, IsDeletable(baseDir, "alarm-definitions/MY_NE.xml"))

	assert.Equal(t, 1, reloader.calls)

	var resp struct {
		Data map[string]any `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "alarm-definitions/MY_NE.xml", resp.Data["loaded_from"])
	assert.Equal(t, "MY_NE", resp.Data["ne_type"])
	assert.Equal(t, true, resp.Data["reloaded"])
}

// 文件名重复且无 force → 409 + overwritable 标记,不覆盖、不重载。
func TestUpload_NameConflict_409(t *testing.T) {
	baseDir := t.TempDir()
	writeBuiltinXML(t, baseDir, "MY.xml")
	reloader := &stubReloader{}
	r := newTestRouter(t, &mockFileRepository{}, reloader, baseDir)

	body, ct := buildMultipart(t, []byte(`<alarmModel neType="MY"></alarmModel>`))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alarm-definitions/upload-xml", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusConflict, w.Code, "body=%s", w.Body.String())
	assert.Contains(t, w.Body.String(), `"overwritable":true`)
	assert.Equal(t, 0, reloader.calls)
}

// force=true 覆盖自定义文件:旧文件备份 .bak.<ts>,sidecar 保持(仍 custom 可删)。
func TestUpload_ForceOverwriteCustom_201(t *testing.T) {
	baseDir := t.TempDir()
	loadedFrom := writeCustomXML(t, baseDir, "MY_NE.xml") // 含 sidecar
	reloader := &stubReloader{}
	r := newTestRouter(t, &mockFileRepository{neTypeLoadedFroms: []string{loadedFrom}}, reloader, baseDir)

	newXML := []byte(`<alarmModel neType="MY_NE" totalCount="1"><alarms></alarms></alarmModel>`)
	body, ct := buildMultipart(t, newXML)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alarm-definitions/upload-xml?force=true", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code, "body=%s", w.Body.String())
	assert.Contains(t, w.Body.String(), `"overwritten":true`)

	target := filepath.Join(baseDir, loadedFrom)
	got, err := os.ReadFile(target)
	assert.NoError(t, err)
	assert.Equal(t, newXML, got, "覆盖后内容应为新 XML")
	// 备份存在
	baks, _ := filepath.Glob(target + ".bak.*")
	assert.Len(t, baks, 1, "旧文件应备份为 .bak.<ts>")
	// sidecar 保持 → 仍 custom 可删
	assert.True(t, IsDeletable(baseDir, loadedFrom))
	assert.Equal(t, 1, reloader.calls)
}

// force=true 覆盖内置文件:备份 + 内容替换,但不写 sidecar(保持 builtin,仍不可删)。
func TestUpload_ForceOverwriteBuiltin_201(t *testing.T) {
	baseDir := t.TempDir()
	loadedFrom := writeBuiltinXML(t, baseDir, "ENB.xml") // 无 sidecar
	reloader := &stubReloader{}
	r := newTestRouter(t, &mockFileRepository{neTypeLoadedFroms: []string{loadedFrom}}, reloader, baseDir)

	newXML := []byte(`<alarmModel neType="ENB" totalCount="2"><alarms></alarms></alarmModel>`)
	body, ct := buildMultipart(t, newXML)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alarm-definitions/upload-xml?force=true", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code, "body=%s", w.Body.String())

	target := filepath.Join(baseDir, loadedFrom)
	got, err := os.ReadFile(target)
	assert.NoError(t, err)
	assert.Equal(t, newXML, got)
	baks, _ := filepath.Glob(target + ".bak.*")
	assert.Len(t, baks, 1)
	// 不写 sidecar → 保持 builtin 身份,仍不可删
	_, scErr := os.Stat(target + CustomMarkerSuffix)
	assert.True(t, os.IsNotExist(scErr), "覆盖 builtin 不应写 sidecar")
	assert.False(t, IsDeletable(baseDir, loadedFrom))
}

// 内容主键唯一性:neType 已在 DB → 409,文件不落地。
func TestUpload_NeTypeConflict_409(t *testing.T) {
	baseDir := t.TempDir()
	reloader := &stubReloader{}
	repo := &mockFileRepository{neTypeExists: true}
	r := newTestRouter(t, repo, reloader, baseDir)

	body, ct := buildMultipart(t, []byte(`<alarmModel neType="DUP"></alarmModel>`))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alarm-definitions/upload-xml", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusConflict, w.Code, "body=%s", w.Body.String())
	assert.Contains(t, w.Body.String(), "neType")
	assert.Equal(t, 0, reloader.calls)
	_, err := os.Stat(filepath.Join(baseDir, BuiltinDirSubdir, "DUP.xml"))
	assert.True(t, os.IsNotExist(err), "neType 冲突时文件不应落地")
}

// neType 属性缺失 → 400(名称唯一来源是 XML neType,不再回退文件名)。
func TestUpload_MissingNeType_400(t *testing.T) {
	baseDir := t.TempDir()
	r := newTestRouter(t, &mockFileRepository{}, &stubReloader{}, baseDir)
	body, ct := buildMultipart(t, []byte(`<alarmModel><alarms></alarms></alarmModel>`))
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
	// neType 含点 → 推导文件名 "MY.foo.xml" 多扩展名,被白名单拒
	body, ct := buildMultipart(t, []byte(`<alarmModel neType="MY.foo"></alarmModel>`))
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
	body, ct := buildMultipart(t, []byte(`<indicatorModel platform="X"></indicatorModel>`))
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
	_ = mw.WriteField("other", "MY")
	_ = mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alarm-definitions/upload-xml", buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ── neType 长度守门(#123:ne_type 列 varchar(16),超长曾导致 201 假成功 0 行入库) ──

func TestValidateUploadNeType(t *testing.T) {
	cases := []struct {
		name    string
		neType  string
		wantErr bool
	}{
		{"短名通过", "ENB", false},
		{"恰好 16 字符通过", strings.Repeat("A", 16), false},
		{"17 字符拒绝", strings.Repeat("A", 17), true},
		{"issue #123 实例 21 字符拒绝", "SMKsmk178109434922378", true},
		{"多字节按 rune 计:16 个汉字通过", strings.Repeat("基", 16), false},
		{"多字节按 rune 计:17 个汉字拒绝", strings.Repeat("基", 17), true},
		{"空值放行(由调用方先拒)", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateUploadNeType(tc.neType)
			if tc.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "16")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// 超长 neType(>16) → 400 + 明确错误消息,文件不落地,不触发 reload。
func TestUpload_NeTypeTooLong_400(t *testing.T) {
	baseDir := t.TempDir()
	reloader := &stubReloader{}
	r := newTestRouter(t, &mockFileRepository{}, reloader, baseDir)

	const longNeType = "SMKsmk178109434922378" // 21 字符,issue #123 复现实例
	body, ct := buildMultipart(t, []byte(`<alarmModel neType="`+longNeType+`"><alarms></alarms></alarmModel>`))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alarm-definitions/upload-xml", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())
	assert.Contains(t, w.Body.String(), fmt.Sprintf("code=%d", global.ErrCodeAlarmUploadInvalidName))
	assert.Contains(t, w.Body.String(), "16", "错误消息应说明长度上限")
	assert.Equal(t, 0, reloader.calls)
	_, err := os.Stat(filepath.Join(baseDir, BuiltinDirSubdir, longNeType+".xml"))
	assert.True(t, os.IsNotExist(err), "超长 neType 时文件不应落地")
}

// ── 重载失败回滚(#123:此前只 Warn 仍报 201+reloaded:true,形成静默失败) ──

// 新建上传 + 重载失败 → 500,落盘 XML 与 sidecar 一并回滚(不留半成品)。
func TestUpload_ReloadFailed_NewFile_500_RollsBack(t *testing.T) {
	baseDir := t.TempDir()
	reloader := &stubReloader{err: errors.New("simulated reload fail")}
	r := newTestRouter(t, &mockFileRepository{}, reloader, baseDir)
	body, ct := buildMultipart(t, []byte(`<alarmModel neType="X"></alarmModel>`))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alarm-definitions/upload-xml", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code, "body=%s", w.Body.String())
	assert.Equal(t, 1, reloader.calls)

	target := filepath.Join(baseDir, BuiltinDirSubdir, "X.xml")
	_, statErr := os.Stat(target)
	assert.True(t, os.IsNotExist(statErr), "重载失败后落盘 XML 应回滚删除")
	_, scErr := os.Stat(target + CustomMarkerSuffix)
	assert.True(t, os.IsNotExist(scErr), "重载失败后 sidecar 应回滚删除")
}

// force 覆盖 + 重载失败 → 500,.bak 备份还原回原位(旧内容保留,无 .bak 残留)。
func TestUpload_ReloadFailed_Overwrite_500_RestoresBackup(t *testing.T) {
	baseDir := t.TempDir()
	loadedFrom := writeCustomXML(t, baseDir, "MY_NE.xml") // 旧文件 + sidecar
	target := filepath.Join(baseDir, loadedFrom)
	oldXML, err := os.ReadFile(target)
	assert.NoError(t, err)

	reloader := &stubReloader{err: errors.New("simulated reload fail")}
	r := newTestRouter(t, &mockFileRepository{neTypeLoadedFroms: []string{loadedFrom}}, reloader, baseDir)

	newXML := []byte(`<alarmModel neType="MY_NE" totalCount="1"><alarms></alarms></alarmModel>`)
	body, ct := buildMultipart(t, newXML)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alarm-definitions/upload-xml?force=true", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code, "body=%s", w.Body.String())
	assert.Equal(t, 1, reloader.calls)

	got, err := os.ReadFile(target)
	assert.NoError(t, err, "重载失败后原文件应还原")
	assert.Equal(t, oldXML, got, "重载失败后应还原旧内容而非新 XML")
	baks, _ := filepath.Glob(target + ".bak.*")
	assert.Empty(t, baks, "备份应 rename 回原位,不残留 .bak")
	// sidecar 本就在位(覆盖 custom 不动 sidecar),仍 custom 可删
	assert.True(t, IsDeletable(baseDir, loadedFrom))
}

// ── DownloadFile ────────────────────────────────────────────────────

func TestDownloadFile_OK(t *testing.T) {
	baseDir := t.TempDir()
	loadedFrom := writeBuiltinXML(t, baseDir, "ENB.xml")
	r := newTestRouter(t, &mockFileRepository{}, &stubReloader{}, baseDir)

	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/alarm-definitions/file-content?loaded_from="+loadedFrom, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	assert.Contains(t, w.Header().Get("Content-Disposition"), "ENB.xml")
	assert.Contains(t, w.Body.String(), "<alarmModel")
}

func TestDownloadFile_NotFound_404(t *testing.T) {
	baseDir := t.TempDir()
	r := newTestRouter(t, &mockFileRepository{}, &stubReloader{}, baseDir)
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/alarm-definitions/file-content?loaded_from=alarm-definitions/GONE.xml", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDownloadFile_BadPath_400(t *testing.T) {
	baseDir := t.TempDir()
	r := newTestRouter(t, &mockFileRepository{}, &stubReloader{}, baseDir)
	for _, lf := range []string{"", "other/X.xml", "alarm-definitions/../etc.xml", "alarm-definitions/a/b.xml"} {
		req := httptest.NewRequest(http.MethodGet,
			"/api/v1/alarm-definitions/file-content?loaded_from="+lf, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code, "loaded_from=%q", lf)
	}
}

// ── DownloadFile 生成模式(#268 手工新增) ─────────────────────────────

func manualDef(identifier, severityName string, eventType *int, isShow bool) ResolvedDefinition {
	return ResolvedDefinition{
		AlarmDefinition: AlarmDefinition{
			ID:              uuid.New(),
			Identifier:      identifier,
			NeType:          "ENB",
			CnName:          "手工告警" + identifier,
			EnName:          "Manual alarm " + identifier,
			SeverityID:      uuid.New(),
			EventType:       eventType,
			CnProbableCause: "原因" + identifier,
			EnProbableCause: "Cause " + identifier,
			IsShow:          isShow,
		},
		SeverityCode: 31001,
		SeverityName: severityName,
	}
}

func TestDownloadFile_GeneratedManual_OK(t *testing.T) {
	baseDir := t.TempDir()
	et := 30003
	repo := &mockFileRepository{manualDefs: []ResolvedDefinition{
		manualDef("11109", "Critical", &et, true),
		manualDef("11110", "Warning", nil, false),
	}}
	r := newTestRouter(t, repo, &stubReloader{}, baseDir)

	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/alarm-definitions/file-content?ne_type=ENB", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	assert.Equal(t, "application/xml; charset=utf-8", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Content-Disposition"), `ENB-manual.xml`)

	body := w.Body.Bytes()
	assert.Contains(t, string(body), `<?xml version="1.0" encoding="UTF-8"?>`)
	// 导出的 XML 必须能被导入侧同一套结构重新解析(round-trip)。
	var parsed xmlAlarmModel
	assert.NoError(t, xml.Unmarshal(body, &parsed))
	assert.Equal(t, "ENB", parsed.NeType)
	assert.Equal(t, 2, parsed.TotalCount)
	assert.Len(t, parsed.Alarms, 2)
	assert.Equal(t, "11109", parsed.Alarms[0].Identifier)
	assert.Equal(t, "Critical", parsed.Alarms[0].Severity)
	assert.Equal(t, "30003", parsed.Alarms[0].EventType)
	assert.Equal(t, "Y", parsed.Alarms[0].IsShow)
	assert.Equal(t, "Warning", parsed.Alarms[1].Severity)
	assert.Equal(t, "", parsed.Alarms[1].EventType)
	assert.Equal(t, "N", parsed.Alarms[1].IsShow)
}

func TestDownloadFile_GeneratedManual_Empty_404(t *testing.T) {
	baseDir := t.TempDir()
	r := newTestRouter(t, &mockFileRepository{}, &stubReloader{}, baseDir)
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/alarm-definitions/file-content?ne_type=GSM", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDownloadFile_GeneratedManual_MissingParams_400(t *testing.T) {
	baseDir := t.TempDir()
	r := newTestRouter(t, &mockFileRepository{}, &stubReloader{}, baseDir)
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/alarm-definitions/file-content", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSanitizeDownloadName(t *testing.T) {
	assert.Equal(t, "ENB-manual.xml", sanitizeDownloadName("ENB"))
	assert.Equal(t, "my_ne_1-manual.xml", sanitizeDownloadName("my.ne 1"))
	// 非 ASCII / 特殊字符 neType 收敛为 '_',仍匹配回传文件名模式
	assert.Equal(t, "__-manual.xml", sanitizeDownloadName("基站"))
	assert.Equal(t, "___-manual.xml", sanitizeDownloadName("!!!"))
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
