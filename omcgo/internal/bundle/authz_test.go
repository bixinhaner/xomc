package bundle

import (
	"archive/zip"
	"bytes"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeSNReader 用固定的 可见 SN 集合模拟 SNVisibilityReader：
//   - visibleGroups == nil → 全部可见（超管）
//   - 否则只放行 allowed 集合里的 SN
type fakeSNReader struct {
	allowed map[string]struct{}
	calls   int
}

func (f *fakeSNReader) VisibleSerialNumbers(_ context.Context, visibleGroups []uuid.UUID, sns []string) (map[string]struct{}, error) {
	f.calls++
	out := make(map[string]struct{}, len(sns))
	if visibleGroups == nil {
		for _, sn := range sns {
			out[sn] = struct{}{}
		}
		return out, nil
	}
	for _, sn := range sns {
		if _, ok := f.allowed[sn]; ok {
			out[sn] = struct{}{}
		}
	}
	return out, nil
}

func entryNames(t *testing.T, buf *bytes.Buffer) map[string]struct{} {
	t.Helper()
	if buf.Len() == 0 {
		return map[string]struct{}{}
	}
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)
	out := make(map[string]struct{}, len(zr.File))
	for _, f := range zr.File {
		out[f.Name] = struct{}{}
	}
	return out
}

// SN-keyed 模块（ModuleConfigSnapshot）：入参 SN 在调 Source 前就被收窄到可见集合，
// 域外 SN 根本不会触达 Source。
func TestWriteZipTo_SNKeyed_PreFiltersInvisible(t *testing.T) {
	svc := newService()
	reader := &fakeSNReader{allowed: map[string]struct{}{"SN-VISIBLE": {}}}
	svc.SetSNVisibilityReader(reader)

	var gotTargets []string
	svc.Register(ModuleConfigSnapshot, func(_ context.Context, sns []string) ([]BundleFile, error) {
		gotTargets = sns
		return []BundleFile{}, nil // 内容不重要，只验证传入 Source 的 SN
	})

	visible := []uuid.UUID{uuid.New()}
	var buf bytes.Buffer
	_, err := svc.WriteZipTo(context.Background(), ModuleConfigSnapshot,
		[]string{"SN-VISIBLE", "SN-FOREIGN"}, visible, &buf)
	require.NoError(t, err)
	assert.Equal(t, []string{"SN-VISIBLE"}, gotTargets, "域外 SN 应在调 Source 前被剔除")
}

// SN-keyed 模块：全部目标都不可见 → 0 文件 + 不调 Source。
func TestWriteZipTo_SNKeyed_AllInvisibleReturnsZero(t *testing.T) {
	svc := newService()
	svc.SetSNVisibilityReader(&fakeSNReader{allowed: map[string]struct{}{}})
	called := false
	svc.Register(ModuleDeviceLicense, func(_ context.Context, _ []string) ([]BundleFile, error) {
		called = true
		return []BundleFile{{Bucket: "b", ObjectPath: "p", EntryName: "x"}}, nil
	})
	var buf bytes.Buffer
	n, err := svc.WriteZipTo(context.Background(), ModuleDeviceLicense,
		[]string{"SN-FOREIGN"}, []uuid.UUID{uuid.New()}, &buf)
	require.NoError(t, err)
	assert.Equal(t, 0, n)
	assert.False(t, called, "全部不可见时不应调 Source")
}

// 文件 ID 模块（ModuleMRFiles）：Source 已填 BundleFile.DeviceSN，WriteZipTo 后置剔除
// 域外设备文件，保留可见设备文件。
func TestWriteZipTo_FileID_PostFiltersByDeviceSN(t *testing.T) {
	svc := newService()
	svc.SetSNVisibilityReader(&fakeSNReader{allowed: map[string]struct{}{"SN-OK": {}}})
	svc.Register(ModuleMRFiles, func(_ context.Context, _ []string) ([]BundleFile, error) {
		// Bucket="" 让 GetObject 名称校验失败 → 写 .error.txt 占位（不需 MinIO），
		// 但被保留的文件才会生成占位；被剔除的域外文件不应出现任何条目。
		return []BundleFile{
			{Bucket: "", ObjectPath: "a", EntryName: "ok.bin", DeviceSN: "SN-OK"},
			{Bucket: "", ObjectPath: "b", EntryName: "foreign.bin", DeviceSN: "SN-FOREIGN"},
		}, nil
	})
	var buf bytes.Buffer
	_, err := svc.WriteZipTo(context.Background(), ModuleMRFiles,
		[]string{"id-1", "id-2"}, []uuid.UUID{uuid.New()}, &buf)
	require.NoError(t, err)

	names := entryNames(t, &buf)
	assert.Contains(t, names, "ok.bin.error.txt", "可见设备文件应进入打包（GetObject 失败转占位）")
	assert.NotContains(t, names, "foreign.bin.error.txt", "域外设备文件应被剔除")
}

// 超管 nil visibleGroups：不过滤，SN-keyed 与 file-ID 文件全保留，且不调 reader。
func TestWriteZipTo_SuperadminBypassesFilter(t *testing.T) {
	svc := newService()
	reader := &fakeSNReader{allowed: map[string]struct{}{}}
	svc.SetSNVisibilityReader(reader)
	svc.Register(ModuleConfigSnapshot, func(_ context.Context, sns []string) ([]BundleFile, error) {
		assert.ElementsMatch(t, []string{"SN-A", "SN-B"}, sns, "超管不应预过滤 SN")
		return []BundleFile{}, nil
	})
	var buf bytes.Buffer
	_, err := svc.WriteZipTo(context.Background(), ModuleConfigSnapshot,
		[]string{"SN-A", "SN-B"}, nil, &buf)
	require.NoError(t, err)
	assert.Equal(t, 0, reader.calls, "超管 nil 不应调用 reader")
}

// firmware 是全局制品（无 DeviceSN）：即便非超管也不被设备组过滤剔除。
func TestWriteZipTo_FirmwareGlobalArtifactsKept(t *testing.T) {
	svc := newService()
	svc.SetSNVisibilityReader(&fakeSNReader{allowed: map[string]struct{}{}})
	svc.Register(ModuleFirmware, func(_ context.Context, _ []string) ([]BundleFile, error) {
		return []BundleFile{
			{Bucket: "", ObjectPath: "img", EntryName: "fw.img"}, // DeviceSN 留空
		}, nil
	})
	var buf bytes.Buffer
	_, err := svc.WriteZipTo(context.Background(), ModuleFirmware,
		[]string{uuid.NewString()}, []uuid.UUID{uuid.New()}, &buf)
	require.NoError(t, err)
	names := entryNames(t, &buf)
	assert.Contains(t, names, "fw.img.error.txt", "全局固件制品不应被设备组过滤剔除")
}
