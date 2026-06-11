package bundle

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

// newService 构造一个 minio client 为 nil 的 Service。
//
// 设计：本测试只覆盖「Source 路由 / 入参校验 / zip 容错占位」这些不触达对象存储的分支。
// WriteZipTo 在 source 返回非空 BundleFile 时才会调 minio.GetObject；我们用 Bucket="" 的
// 非法对象让 GetObject 在 s3utils 名称校验阶段就返回 error（早于解引用 client），从而稳定
// 命中「写 .error.txt 占位、其余继续」的容错分支，无需起 MinIO。
func newService() *Service {
	return NewService(nil, nil) // logger=nil → NewService 内部回退 zap.NewNop()
}

// readZipEntries 把 buf 当作 zip 解开，返回 entryName → content。
func readZipEntries(t *testing.T, buf *bytes.Buffer) map[string]string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)
	out := make(map[string]string, len(zr.File))
	for _, f := range zr.File {
		rc, err := f.Open()
		require.NoError(t, err)
		data, err := io.ReadAll(rc)
		require.NoError(t, err)
		_ = rc.Close()
		out[f.Name] = string(data)
	}
	return out
}

// --- 失败路径：未注册 Source 的模块 ---

func TestWriteZipTo_NoSourceRegistered(t *testing.T) {
	svc := newService()
	var buf bytes.Buffer
	n, err := svc.WriteZipTo(context.Background(), ModuleFirmware, []string{"id-1"}, nil, &buf)
	assert.Equal(t, 0, n)
	require.Error(t, err)
	assert.ErrorIs(t, err, commonerrors.ErrInvalidInput)
	assert.Contains(t, err.Error(), "no source registered")
}

// --- 失败路径：空 targetIDs ---

func TestWriteZipTo_EmptyTargetIDs(t *testing.T) {
	svc := newService()
	svc.Register(ModuleMR, func(context.Context, []string) ([]BundleFile, error) {
		return nil, nil
	})
	var buf bytes.Buffer
	n, err := svc.WriteZipTo(context.Background(), ModuleMR, nil, nil, &buf)
	assert.Equal(t, 0, n)
	require.Error(t, err)
	assert.ErrorIs(t, err, commonerrors.ErrInvalidInput)
	assert.Contains(t, err.Error(), "target_ids empty")
}

// --- 失败路径：Source 解析报错 → 整体 error，不写 zip ---

func TestWriteZipTo_SourceError(t *testing.T) {
	svc := newService()
	sentinel := errors.New("query firmware failed")
	svc.Register(ModuleFirmware, func(context.Context, []string) ([]BundleFile, error) {
		return nil, sentinel
	})
	var buf bytes.Buffer
	n, err := svc.WriteZipTo(context.Background(), ModuleFirmware, []string{"id-1"}, nil, &buf)
	assert.Equal(t, 0, n)
	require.Error(t, err)
	assert.ErrorIs(t, err, sentinel)
	assert.Contains(t, err.Error(), "resolve files")
}

// --- 边界：Source 返回 0 文件 → (0, nil)，caller 据此回 4xx ---

func TestWriteZipTo_ZeroFiles(t *testing.T) {
	svc := newService()
	svc.Register(ModuleMR, func(context.Context, []string) ([]BundleFile, error) {
		return []BundleFile{}, nil
	})
	var buf bytes.Buffer
	n, err := svc.WriteZipTo(context.Background(), ModuleMR, []string{"SN-001"}, nil, &buf)
	assert.Equal(t, 0, n)
	assert.NoError(t, err)
	assert.Zero(t, buf.Len(), "0 文件不应写出任何 zip 字节")
}

// --- 容错路径：单文件 GetObject 失败 → 写 .error.txt 占位，整体不报错 ---
//
// Bucket="" 触发 minio 名称校验失败（早于 client 解引用），稳定命中占位分支。
func TestWriteZipTo_ObjectErrorWritesPlaceholder(t *testing.T) {
	svc := newService()
	svc.Register(ModuleMR, func(context.Context, []string) ([]BundleFile, error) {
		return []BundleFile{
			{Bucket: "", ObjectPath: "missing/a.bin", EntryName: "SN-001/a.bin"},
		}, nil
	})

	var buf bytes.Buffer
	n, err := svc.WriteZipTo(context.Background(), ModuleMR, []string{"SN-001"}, nil, &buf)
	require.NoError(t, err, "单对象失败不应让整批 5xx")
	assert.Equal(t, 0, n, "失败对象不计入写入数")

	entries := readZipEntries(t, &buf)
	placeholder, ok := entries["SN-001/a.bin.error.txt"]
	require.True(t, ok, "应为失败对象写出 .error.txt 占位，实际条目: %v", keysOf(entries))
	assert.NotEmpty(t, placeholder, "占位文件应记录失败原因")
}

// --- 容错路径：混合（部分失败部分缺失）仍稳健 ---

func TestWriteZipTo_AllObjectsFail(t *testing.T) {
	svc := newService()
	svc.Register(ModuleFirmware, func(context.Context, []string) ([]BundleFile, error) {
		return []BundleFile{
			{Bucket: "", ObjectPath: "x", EntryName: "fw1.img"},
			{Bucket: "", ObjectPath: "y", EntryName: "fw2.img"},
		}, nil
	})
	var buf bytes.Buffer
	n, err := svc.WriteZipTo(context.Background(), ModuleFirmware, []string{"a", "b"}, nil, &buf)
	require.NoError(t, err)
	assert.Equal(t, 0, n)

	entries := readZipEntries(t, &buf)
	assert.Contains(t, entries, "fw1.img.error.txt")
	assert.Contains(t, entries, "fw2.img.error.txt")
}

// --- Register 覆盖：后注册的 Source 覆盖先前的（map 语义）---

func TestRegister_Overwrite(t *testing.T) {
	svc := newService()
	called := ""
	svc.Register(ModuleMR, func(context.Context, []string) ([]BundleFile, error) {
		called = "first"
		return nil, errors.New("first")
	})
	svc.Register(ModuleMR, func(context.Context, []string) ([]BundleFile, error) {
		called = "second"
		return nil, errors.New("second")
	})
	var buf bytes.Buffer
	_, err := svc.WriteZipTo(context.Background(), ModuleMR, []string{"x"}, nil, &buf)
	require.Error(t, err)
	assert.Equal(t, "second", called, "后注册的 Source 应生效")
}

func keysOf(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
