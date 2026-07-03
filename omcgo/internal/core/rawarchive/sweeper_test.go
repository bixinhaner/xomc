package rawarchive

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/compress"
)

// fakeRegistry 是 RawFileRegistry 的内存假实现：uncompressed 为待压对象键集合，
// marked 记已标记压缩的键。ListUncompressed 返回未标记的键（忽略 olderThan，由测试控
// grace=0 等价全取）。listErr / markErr 可注入故障路径。
type fakeRegistry struct {
	mu           sync.Mutex
	uncompressed []string
	marked       map[string]bool
	listErr      error
	markErr      error
	listCalls    int
}

func newFakeRegistry(objs ...string) *fakeRegistry {
	return &fakeRegistry{uncompressed: append([]string(nil), objs...), marked: map[string]bool{}}
}

func (r *fakeRegistry) ListUncompressed(_ context.Context, _ time.Time, limit int) ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.listCalls++
	if r.listErr != nil {
		return nil, r.listErr
	}
	out := make([]string, 0, limit)
	for _, o := range r.uncompressed {
		if r.marked[o] {
			continue
		}
		out = append(out, o)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (r *fakeRegistry) MarkCompressed(_ context.Context, renames map[string]string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.markErr != nil {
		return r.markErr
	}
	// 按旧键标记（旧键即 uncompressed 列表里的待压键），模拟 UPDATE ... WHERE minio_path=old。
	for oldKey := range renames {
		r.marked[oldKey] = true
	}
	return nil
}

func (r *fakeRegistry) isMarked(o string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.marked[o]
}

func newTestSweeper(a *Archiver, sources []SweepSource) *Sweeper {
	// grace=0 → cutoff=now，假注册表忽略 olderThan，全部待压对象立即可扫。
	return NewSweeper(a, sources, time.Minute, 0, 10, 2, NewSweepMetrics(nil), nil)
}

func TestSweeper_CompressesAndMarksResidue(t *testing.T) {
	store := newFakeStore()
	store.put("pm-files", "p1.xml", []byte(largeXML()))
	store.put("pm-files", "p2.xml", []byte(largeXML()))
	store.put("mr-files", "m1.xml", []byte(largeXML()))
	reg := newFakeRegistry("p1.xml", "p2.xml")
	mreg := newFakeRegistry("m1.xml")

	a := newTestArchiver(store, nil) // 默认启用
	s := newTestSweeper(a, []SweepSource{
		{Name: "pm", Bucket: "pm-files", Registry: reg},
		{Name: "mr", Bucket: "mr-files", Registry: mreg},
	})
	s.SweepOnce(context.Background())

	for _, o := range []string{"p1.xml", "p2.xml"} {
		// 压缩回写到 .gz 新键，原明文键由 RemoveOld 删除。
		rec, ok := store.puts[key("pm-files", o+".gz")]
		require.True(t, ok, "%s 应被补压回写到 .gz 新键", o)
		assert.True(t, compress.IsGzip(rec.data), "%s 回写应是 gzip", o)
		assert.True(t, reg.isMarked(o), "%s 应被标记 raw_compressed", o)
		assert.True(t, store.removed[key("pm-files", o)], "%s 改键后旧明文键应被删除", o)
	}
	_, ok := store.puts[key("mr-files", "m1.xml.gz")]
	assert.True(t, ok, "MR 源也应被补压到 .gz 新键")
	assert.True(t, mreg.isMarked("m1.xml"), "MR 对象应被标记")
}

func TestSweeper_AlreadyGzipMarkedNotRewritten(t *testing.T) {
	store := newFakeStore()
	store.put("pm-files", "g.xml.gz", gzipOf(t, largeXML())) // 真机已压上传
	reg := newFakeRegistry("g.xml.gz")

	a := newTestArchiver(store, nil)
	s := newTestSweeper(a, []SweepSource{{Name: "pm", Bucket: "pm-files", Registry: reg}})
	s.SweepOnce(context.Background())

	_, ok := store.puts[key("pm-files", "g.xml.gz")]
	assert.False(t, ok, "已是 gzip 的对象不应被重写")
	assert.True(t, reg.isMarked("g.xml.gz"), "已 gzip 也属终态，应标记以移出待扫集")
}

func TestSweeper_DisabledSkipsEntirely(t *testing.T) {
	store := newFakeStore()
	store.put("pm-files", "d.xml", []byte(largeXML()))
	reg := newFakeRegistry("d.xml")

	lookup := func(_ context.Context, category, k string) (string, bool) {
		if category == Category && k == KeyEnabled {
			return "false", true
		}
		return "", false
	}
	a := newTestArchiver(store, lookup)
	s := newTestSweeper(a, []SweepSource{{Name: "pm", Bucket: "pm-files", Registry: reg}})
	s.SweepOnce(context.Background())

	assert.Equal(t, 0, reg.listCalls, "禁用时整轮跳过，不应触达注册表")
	assert.False(t, reg.isMarked("d.xml"))
}

func TestSweeper_ListErrorDoesNotMark(t *testing.T) {
	store := newFakeStore()
	reg := newFakeRegistry("x.xml")
	reg.listErr = assertErr{}

	a := newTestArchiver(store, nil)
	s := newTestSweeper(a, []SweepSource{{Name: "pm", Bucket: "pm-files", Registry: reg}})
	s.SweepOnce(context.Background()) // 不应 panic

	assert.False(t, reg.isMarked("x.xml"), "列表失败时不应标记任何对象")
}

func TestSweeper_NoGainDoesNotMarkCompressed(t *testing.T) {
	store := newFakeStore()
	store.put("pm-files", "tiny.xml", []byte("<a/>")) // 压不动
	reg := newFakeRegistry("tiny.xml")

	a := newTestArchiver(store, nil)
	s := newTestSweeper(a, []SweepSource{{Name: "pm", Bucket: "pm-files", Registry: reg}})
	s.SweepOnce(context.Background())

	_, ok := store.puts[key("pm-files", "tiny.xml")]
	assert.False(t, ok, "压不动的小文件不回写")
	assert.False(t, reg.isMarked("tiny.xml"), "no_gain 没有实际 gzip 存储，不能标记 raw_compressed")
}

type assertErr struct{}

func (assertErr) Error() string { return "injected list error" }
