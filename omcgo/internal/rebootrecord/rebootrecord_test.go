package rebootrecord

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// --- ParseRebootType 归一化（纯函数，含未知值回退）---

func TestParseRebootType(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want RebootType
	}{
		{name: "normal", in: "normal", want: RebootTypeNormal},
		{name: "abnormal", in: "abnormal", want: RebootTypeAbnormal},
		{name: "空串→全部", in: "", want: RebootTypeAll},
		{name: "未知值回退全部", in: "garbage", want: RebootTypeAll},
		{name: "大小写不匹配回退全部", in: "Normal", want: RebootTypeAll},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ParseRebootType(tc.in))
		})
	}
}

func TestFilterOffsetLimit(t *testing.T) {
	tests := []struct {
		name       string
		page       int
		pageSize   int
		wantOffset int
		wantLimit  int
	}{
		{name: "首页默认大小", page: 1, pageSize: 0, wantOffset: 0, wantLimit: 20},
		{name: "page<=0 视首页", page: 0, pageSize: 10, wantOffset: 0, wantLimit: 10},
		{name: "第二页", page: 2, pageSize: 10, wantOffset: 10, wantLimit: 10},
		{name: "负 pageSize 回退默认", page: 1, pageSize: -3, wantOffset: 0, wantLimit: 20},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := Filter{Page: tc.page, PageSize: tc.pageSize}
			assert.Equal(t, tc.wantOffset, f.Offset())
			assert.Equal(t, tc.wantLimit, f.Limit())
		})
	}
}

func TestBuildUnionBackfillsSnapshotFieldsFromDeviceTables(t *testing.T) {
	sql, args := buildUnion(Filter{})

	require.Empty(t, args)
	assert.Contains(t, sql, "FROM event_logs el")
	assert.Contains(t, sql, "LEFT JOIN devices d ON d.id = el.device_id")
	assert.Contains(t, sql, "LEFT JOIN device_info di ON di.device_id = d.id")
	assert.Contains(t, sql, "COALESCE(NULLIF(el.device_sn, ''), d.serial_number, '') AS device_sn")
	assert.Contains(t, sql, "COALESCE(NULLIF(el.device_name, ''), NULLIF(di.device_name, ''), d.site_name, '') AS device_name")
	assert.Contains(t, sql, "COALESCE(NULLIF(el.operate_ip, ''), host(d.ip_address), '') AS operate_ip")
	assert.Contains(t, sql, "COALESCE(NULLIF(el.software_version, ''), d.firmware_version, '') AS software_version")

	assert.Contains(t, sql, "FROM station_fault_logs fl")
	assert.Contains(t, sql, "LEFT JOIN devices fd ON fd.id = fl.device_id")
	assert.Contains(t, sql, "LEFT JOIN device_info fdi ON fdi.device_id = fd.id")
	assert.Contains(t, sql, "COALESCE(NULLIF(fl.device_sn, ''), fd.serial_number, '') AS device_sn")
	assert.Contains(t, sql, "COALESCE(NULLIF(fl.device_sn, ''), fd.serial_number, '') <> ''")
	assert.Contains(t, sql, "COALESCE(NULLIF(fl.device_name, ''), NULLIF(fdi.device_name, ''), fd.site_name, '') AS device_name")
}

func TestBuildUnionFiltersUseBackfilledSnapshotFields(t *testing.T) {
	sql, args := buildUnion(Filter{DeviceSN: "SN-1", DeviceType: "gNB"})

	require.Equal(t, []interface{}{"%SN-1%", "gNB"}, args)
	assert.Contains(t, sql, "COALESCE(NULLIF(el.device_sn, ''), d.serial_number, '') ILIKE $1")
	assert.Contains(t, sql, "COALESCE(NULLIF(fl.device_sn, ''), fd.serial_number, '') ILIKE $1")
	assert.Contains(t, sql, "COALESCE(NULLIF(el.device_type, ''), CASE d.technology WHEN 'nr' THEN 'gNB' WHEN 'lte' THEN 'eNB' WHEN 'gsm' THEN 'GSM' ELSE '' END, '') = $2")
	assert.Contains(t, sql, "COALESCE(NULLIF(fl.device_type, ''), CASE fd.technology WHEN 'nr' THEN 'gNB' WHEN 'lte' THEN 'eNB' WHEN 'gsm' THEN 'GSM' ELSE '' END, '') = $2")
	assert.False(t, strings.Contains(sql, " device_sn ILIKE $1"), "filters must not use raw snapshot device_sn only")
	assert.False(t, strings.Contains(sql, " device_type = $2"), "filters must not use raw snapshot device_type only")
}

// --- 内存 fake Repository（只读合并查询，无 DB）---

type fakeRepo struct {
	listResult []*RebootRecord
	listTotal  int64
	listErr    error
	statResult []*DeviceRebootStat
	statErr    error
}

func (f *fakeRepo) List(_ context.Context, _ Filter) ([]*RebootRecord, int64, error) {
	if f.listErr != nil {
		return nil, 0, f.listErr
	}
	return f.listResult, f.listTotal, nil
}

func (f *fakeRepo) StatByDevice(_ context.Context, _ Filter) ([]*DeviceRebootStat, error) {
	if f.statErr != nil {
		return nil, f.statErr
	}
	return f.statResult, nil
}

func newService(repo Repository) *Service {
	return NewService(repo, zap.NewNop())
}

// --- List / StatByDevice 成功转发 ---

func TestList_Success(t *testing.T) {
	want := []*RebootRecord{
		{ID: "r-1", Source: "event", IsAbnormal: false, DeviceSN: "SN-001"},
		{ID: "r-2", Source: "fault", IsAbnormal: true, DeviceSN: "SN-001"},
	}
	svc := newService(&fakeRepo{listResult: want, listTotal: 2})

	got, total, err := svc.List(context.Background(), Filter{RebootType: RebootTypeAll})
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)
	assert.Equal(t, want, got)
}

func TestStatByDevice_Success(t *testing.T) {
	want := []*DeviceRebootStat{{DeviceSN: "SN-001", TotalCount: 5, AbnormalCount: 2}}
	svc := newService(&fakeRepo{statResult: want})

	got, err := svc.StatByDevice(context.Background(), Filter{})
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

// --- 失败路径：repo 错误原样上抛 ---

func TestList_Error(t *testing.T) {
	sentinel := errors.New("union query failed")
	svc := newService(&fakeRepo{listErr: sentinel})

	_, _, err := svc.List(context.Background(), Filter{})
	assert.ErrorIs(t, err, sentinel)
}

func TestStatByDevice_Error(t *testing.T) {
	sentinel := errors.New("aggregate failed")
	svc := newService(&fakeRepo{statErr: sentinel})

	_, err := svc.StatByDevice(context.Background(), Filter{})
	assert.ErrorIs(t, err, sentinel)
}
