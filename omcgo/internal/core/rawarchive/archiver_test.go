package rawarchive

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/compress"
)

type putRec struct {
	data            []byte
	contentEncoding string
}

type fakeStore struct {
	mu      sync.Mutex
	objects map[string][]byte
	puts    map[string]putRec
}

func newFakeStore() *fakeStore {
	return &fakeStore{objects: map[string][]byte{}, puts: map[string]putRec{}}
}

func key(b, o string) string { return b + "/" + o }

func (f *fakeStore) put(b, o string, data []byte) { f.objects[key(b, o)] = data }

func (f *fakeStore) ReadHead(_ context.Context, b, o string, n int) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	data := f.objects[key(b, o)]
	if n > len(data) {
		n = len(data)
	}
	return append([]byte(nil), data[:n]...), nil
}

func (f *fakeStore) Get(_ context.Context, b, o string) (io.ReadCloser, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return io.NopCloser(bytes.NewReader(f.objects[key(b, o)])), nil
}

func (f *fakeStore) Put(_ context.Context, b, o string, r io.Reader, _ int64, _, ce string) error {
	data, _ := io.ReadAll(r)
	f.mu.Lock()
	defer f.mu.Unlock()
	f.puts[key(b, o)] = putRec{data: data, contentEncoding: ce}
	f.objects[key(b, o)] = data
	return nil
}

func gzipOf(t *testing.T, s string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	_, err := w.Write([]byte(s))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return buf.Bytes()
}

// largeXML 生成一段高度可压缩、远大于 gzip 头开销的明文，确保压缩有正收益。
func largeXML() string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0"?><measCollecFile>`)
	for i := 0; i < 500; i++ {
		b.WriteString(`<r p="1">1000</r>`)
	}
	b.WriteString(`</measCollecFile>`)
	return b.String()
}

func newTestArchiver(store RawStore, lookup ConfigLookup) *Archiver {
	return New(context.Background(), store, lookup, NewMetrics(nil), nil, 2)
}

func TestArchive_PlaintextGetsCompressedInPlace(t *testing.T) {
	store := newFakeStore()
	plain := largeXML()
	store.put("pm-files", "a.xml", []byte(plain))

	a := newTestArchiver(store, nil) // nil lookup → default enabled
	a.archive(context.Background(), "pm-files", "a.xml")

	rec, ok := store.puts[key("pm-files", "a.xml")]
	require.True(t, ok, "明文对象应被压缩回写")
	assert.Equal(t, "gzip", rec.contentEncoding)
	assert.True(t, compress.IsGzip(rec.data), "回写内容应是 gzip 字节")
	assert.Less(t, len(rec.data), len(plain), "压缩后应更小")

	// 回写后的对象经解压应还原原文（与入库读取侧 MaybeGunzip 一致）。
	out, _, err := compress.MaybeGunzip(bytes.NewReader(rec.data))
	require.NoError(t, err)
	got, _ := io.ReadAll(out)
	assert.Equal(t, plain, string(got))
}

func TestArchive_AlreadyGzipSkipped(t *testing.T) {
	store := newFakeStore()
	store.put("pm-files", "b.xml.gz", gzipOf(t, largeXML()))

	a := newTestArchiver(store, nil)
	a.archive(context.Background(), "pm-files", "b.xml.gz")

	_, ok := store.puts[key("pm-files", "b.xml.gz")]
	assert.False(t, ok, "已压缩对象不应再被回写")
}

func TestArchive_DisabledByConfig(t *testing.T) {
	store := newFakeStore()
	store.put("pm-files", "c.xml", []byte(largeXML()))

	lookup := func(_ context.Context, category, k string) (string, bool) {
		if category == Category && k == KeyEnabled {
			return "false", true
		}
		return "", false
	}
	a := newTestArchiver(store, lookup)
	a.archive(context.Background(), "pm-files", "c.xml")

	_, ok := store.puts[key("pm-files", "c.xml")]
	assert.False(t, ok, "开关关闭时不应压缩")
}

func TestArchive_TinyInputNoGain(t *testing.T) {
	store := newFakeStore()
	store.put("pm-files", "d.xml", []byte("<a/>")) // gzip 头开销 > 收益

	a := newTestArchiver(store, nil)
	a.archive(context.Background(), "pm-files", "d.xml")

	_, ok := store.puts[key("pm-files", "d.xml")]
	assert.False(t, ok, "压不动的小文件不应回写")
}

func TestArchive_EmptyObject(t *testing.T) {
	store := newFakeStore()
	store.put("pm-files", "e.xml", []byte{})

	a := newTestArchiver(store, nil)
	a.archive(context.Background(), "pm-files", "e.xml")

	_, ok := store.puts[key("pm-files", "e.xml")]
	assert.False(t, ok, "空对象不应回写")
}

func TestParseBool(t *testing.T) {
	assert.True(t, parseBool("true", false))
	assert.True(t, parseBool("1", false))
	assert.True(t, parseBool(" ON ", false))
	assert.False(t, parseBool("false", true))
	assert.False(t, parseBool("0", true))
	assert.True(t, parseBool("garbage", true))   // 非法回落默认
	assert.False(t, parseBool("garbage", false)) // 非法回落默认
}

func TestNilArchiverScheduleSafe(t *testing.T) {
	var a *Archiver
	a.Schedule("b", "o") // 不应 panic

	a2 := New(context.Background(), nil, nil, NewMetrics(nil), nil, 1) // nil store
	a2.Schedule("b", "o")                                              // 无操作
}
