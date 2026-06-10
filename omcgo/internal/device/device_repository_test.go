package device

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/model"
)

// fakeGeoRow 模拟 pgx 行扫描语义供 scanGeoDeviceRow 单测使用：
// vals 中 nil 表示 SQL NULL；与 pgx 一致，NULL 只能扫进"指针的指针"目标
// （**T → 置 nil），扫进非空目标（*float64 / *string 等）直接报错。
// 这保证若有人把坐标列扫描目标改回 &d.Latitude（*float64），测试会像
// 线上 pgx 一样失败（#117 回归保护）。
type fakeGeoRow struct {
	vals []any
}

func (f fakeGeoRow) Scan(dest ...interface{}) error {
	if len(dest) != len(f.vals) {
		return fmt.Errorf("number of field descriptions must equal number of destinations, got %d and %d", len(f.vals), len(dest))
	}
	for i, v := range f.vals {
		if v == nil {
			switch d := dest[i].(type) {
			case **float64:
				*d = nil
			case **string:
				*d = nil
			case **uuid.UUID:
				*d = nil
			default:
				return fmt.Errorf("can't scan into dest[%d]: cannot scan NULL into %T", i, dest[i])
			}
			continue
		}
		switch d := dest[i].(type) {
		case *uuid.UUID:
			*d = v.(uuid.UUID)
		case **uuid.UUID:
			u := v.(uuid.UUID)
			*d = &u
		case *string:
			*d = v.(string)
		case **string:
			s := v.(string)
			*d = &s
		case *float64:
			*d = v.(float64)
		case **float64:
			fv := v.(float64)
			*d = &fv
		case *bool:
			*d = v.(bool)
		case *int:
			*d = v.(int)
		case *model.DeviceLifecycle:
			*d = v.(model.DeviceLifecycle)
		default:
			return fmt.Errorf("fakeGeoRow: unsupported dest[%d] type %T", i, dest[i])
		}
	}
	return nil
}

// geoRowVals 按 ListGeo / SearchDevices 的 SELECT 列顺序构造一行（16 列）。
func geoRowVals(id uuid.UUID, lat, lng any) []any {
	return []any{
		id, "SN-001", "SN-001", // id, serial_number, name
		model.LifecycleCommissioned, true, // lifecycle_state, is_online
		lat, lng, // latitude, longitude（可为 nil = SQL NULL）
		nil, nil, // group_id, group_name
		"site-a", 0, "FAP-100", // address, alarm_count, type
		"10.0.0.1", "AA:BB:CC", "120", "dev-a", // ip, mac, pci, device_name
	}
}

// TestScanGeoDeviceRow_NullableCoordinates 是 #117 的回归测试：
// GET /devices/search 命中 NULL 坐标设备时，repository 扫描曾把可空的
// latitude/longitude 列扫进非空 float64 目标，报
// "scan search result: can't scan into dest[5] (col: latitude)" 导致整个搜索 500。
func TestScanGeoDeviceRow_NullableCoordinates(t *testing.T) {
	id := uuid.New()

	tests := []struct {
		name    string
		lat     any // float64 或 nil（SQL NULL）
		lng     any
		wantLat float64
		wantLng float64
	}{
		{
			name:    "NULL 坐标设备不再扫描失败（#117 主回归）",
			lat:     nil,
			lng:     nil,
			wantLat: 0,
			wantLng: 0,
		},
		{
			name:    "有坐标设备坐标正常透传",
			lat:     30.5928,
			lng:     114.3055,
			wantLat: 30.5928,
			wantLng: 114.3055,
		},
		{
			name:    "单侧 NULL（仅 longitude 缺失）也容忍",
			lat:     30.5928,
			lng:     nil,
			wantLat: 30.5928,
			wantLng: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := scanGeoDeviceRow(fakeGeoRow{vals: geoRowVals(id, tt.lat, tt.lng)})
			require.NoError(t, err)

			// 响应字段兼容：Latitude/Longitude 仍是 float64，NULL → 零值
			assert.Equal(t, tt.wantLat, d.Latitude)
			assert.Equal(t, tt.wantLng, d.Longitude)

			// 其余字段不受坐标可空性影响
			assert.Equal(t, id, d.ID)
			assert.Equal(t, "SN-001", d.SerialNumber)
			assert.Equal(t, model.DeviceActive, d.Status, "commissioned+online 派生 active")
			require.NotNil(t, d.IPAddress)
			assert.Equal(t, "10.0.0.1", *d.IPAddress)
		})
	}
}

// TestScanGeoDeviceRow_NullableJoinColumns 验证 LEFT JOIN 可空列
// （group_id/group_name/address/type）与 COALESCE 空串字段的处理不回退。
func TestScanGeoDeviceRow_NullableJoinColumns(t *testing.T) {
	id := uuid.New()
	vals := []any{
		id, "SN-002", "SN-002",
		model.LifecycleCommissioned, false,
		nil, nil, // 坐标 NULL
		nil, nil, // group_id, group_name NULL
		nil, 0, nil, // address, alarm_count, type（site_name/model_name 可空）
		"", "", "", "", // COALESCE 空串 → 响应里省略（nil 指针）
	}

	d, err := scanGeoDeviceRow(fakeGeoRow{vals: vals})
	require.NoError(t, err)

	assert.Nil(t, d.GroupID)
	assert.Empty(t, d.GroupName)
	assert.Empty(t, d.Address)
	assert.Empty(t, d.Type)
	assert.Equal(t, model.DeviceOffline, d.Status, "commissioned+offline 派生 offline")
	assert.Nil(t, d.IPAddress)
	assert.Nil(t, d.MAC)
	assert.Nil(t, d.PCI)
	assert.Nil(t, d.DeviceName)
}
