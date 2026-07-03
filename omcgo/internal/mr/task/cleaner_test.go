package task

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	minioinfra "github.com/omcgo/omcgo/internal/core/components/minio"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/mr"
	"github.com/omcgo/omcgo/internal/mr/parser"
)

func TestCleanerConfigDefaults(t *testing.T) {
	got := CleanerConfig{}.Defaults()
	assert.Equal(t, "@daily", got.Schedule)

	got = CleanerConfig{Schedule: "@hourly"}.Defaults()
	assert.Equal(t, "@hourly", got.Schedule)
}

// fakeMRStore 是 cleaner 单测用的 mr.MRStore 替身，只实现 RunOnce 会用到的
// DeleteFilesBefore，其余方法未被本文件测试用例调用到，panic 即可及早暴露误用。
type fakeMRStore struct {
	deleteCutoff time.Time
	deleteCalled bool
	deleteReturn int64
	deleteErr    error
}

func (f *fakeMRStore) SaveFile(context.Context, *mr.MRFileInfo) error { panic("not implemented") }
func (f *fakeMRStore) UpdateFileParsed(context.Context, uuid.UUID, int) error {
	panic("not implemented")
}
func (f *fakeMRStore) BatchInsertRecords(context.Context, uuid.UUID, uuid.UUID, string, []parser.MRRecord) error {
	panic("not implemented")
}
func (f *fakeMRStore) ListFiles(context.Context, mr.MRFileFilter) (*model.ListResponse[mr.MRFileInfo], error) {
	panic("not implemented")
}
func (f *fakeMRStore) GetFileByID(context.Context, uuid.UUID) (*mr.MRFileInfo, error) {
	panic("not implemented")
}
func (f *fakeMRStore) QueryRecords(context.Context, mr.MRRecordFilter) (*model.ListResponse[mr.MRRecordEntry], error) {
	panic("not implemented")
}
func (f *fakeMRStore) DeleteFilesBefore(_ context.Context, cutoff time.Time) (int64, error) {
	f.deleteCalled = true
	f.deleteCutoff = cutoff
	return f.deleteReturn, f.deleteErr
}
func (f *fakeMRStore) ListFileDeviceAggregates(context.Context, mr.MRFileDeviceFilter) (*model.ListResponse[mr.MRFileDeviceAggregate], error) {
	panic("not implemented")
}
func (f *fakeMRStore) ListFilesBySN(context.Context, string) ([]mr.MRFileInfo, error) {
	panic("not implemented")
}
func (f *fakeMRStore) DeleteFilesBySN(context.Context, string) (int64, error) {
	panic("not implemented")
}
func (f *fakeMRStore) ListUncompressed(context.Context, time.Time, int) ([]string, error) {
	panic("not implemented")
}
func (f *fakeMRStore) MarkCompressed(context.Context, map[string]string) error {
	panic("not implemented")
}

// TestRunOnce_CutoffUsesRetentionDaysLookup 锁定 #798：MR 文件保留天数并入原始件 ILM 后，
// RunOnce 清理 mr_files PG 行的 cutoff 不再来自部署配置静态值，而是每次都实时读取注入的
// retentionDays lookup（生产上对应 sys_configs minio.retention.raw_object_days）。
func TestRunOnce_CutoffUsesRetentionDaysLookup(t *testing.T) {
	store := &fakeMRStore{deleteReturn: 3}
	c := NewCleaner(store, CleanerConfig{Bucket: "mr-files"}, func(context.Context) int { return 10 }, nil)

	before := time.Now().UTC().AddDate(0, 0, -10)
	removed, err := c.RunOnce(context.Background())
	after := time.Now().UTC().AddDate(0, 0, -10)

	require.NoError(t, err)
	assert.Equal(t, 3, removed)
	assert.True(t, store.deleteCalled)
	assert.True(t, !store.deleteCutoff.Before(before) && !store.deleteCutoff.After(after),
		"cutoff 应等于 now - 10 天（lookup 返回值），实际 %v，期望落在 [%v, %v]", store.deleteCutoff, before, after)
}

// TestRunOnce_FallsBackToDefaultRetentionDays 锁定失败/兜底路径：lookup 为 nil 或返回
// 非正数时，cutoff 回落到 minioinfra.DefaultRawFileRetentionDays（60 天），而不是 0 天
// 清空整表。
func TestRunOnce_FallsBackToDefaultRetentionDays(t *testing.T) {
	cases := []struct {
		name   string
		lookup RetentionDaysLookup
	}{
		{"nil lookup", nil},
		{"zero lookup", func(context.Context) int { return 0 }},
		{"negative lookup", func(context.Context) int { return -5 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := &fakeMRStore{}
			c := NewCleaner(store, CleanerConfig{Bucket: "mr-files"}, tc.lookup, nil)

			before := time.Now().UTC().AddDate(0, 0, -minioinfra.DefaultRawFileRetentionDays)
			_, err := c.RunOnce(context.Background())
			after := time.Now().UTC().AddDate(0, 0, -minioinfra.DefaultRawFileRetentionDays)

			require.NoError(t, err)
			assert.True(t, !store.deleteCutoff.Before(before) && !store.deleteCutoff.After(after),
				"cutoff 应回落默认 %d 天", minioinfra.DefaultRawFileRetentionDays)
		})
	}
}

// TestRunOnce_PropagatesStoreError 失败路径：mr_files 行删除失败应把 error 原样向上传播，
// 让 cron 闭包记 Error 日志（此前 MinIO 对象已交给 ILM 生命周期规则独立治理，不再有
// "MinIO 已删除、PG 失败也不算整体失败"的折中，PG 删除就是本 cleaner 唯一职责）。
func TestRunOnce_PropagatesStoreError(t *testing.T) {
	store := &fakeMRStore{deleteErr: assert.AnError}
	c := NewCleaner(store, CleanerConfig{Bucket: "mr-files"}, func(context.Context) int { return 60 }, nil)

	_, err := c.RunOnce(context.Background())
	require.Error(t, err)
}
