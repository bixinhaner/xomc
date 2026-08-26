package device

import (
	"strings"
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestApplyDeviceFilters_ProductIDAndSNList 是「product_id 在 /devices 列表静默失效」的回归测试。
// 根因：product_id 过滤只加在了未被列表端点使用的 PgDeviceRepository.List，而 live 路径
// （ListDevicesWithInfo → applyDeviceFilters）漏了。此处直接断言 applyDeviceFilters 生成的
// SQL 同时包含 product_id 与 serial_number IN（sn_list 批量输入），防止再次回退。
func TestApplyDeviceFilters_ProductIDAndSNList(t *testing.T) {
	pid := uuid.New()
	filter := DeviceFilter{
		ProductID: &pid,
		SNList:    []string{"SN-A", "SN-B", "SN-C"},
	}

	base := sq.Select("*").From("devices d").PlaceholderFormat(sq.Dollar)
	q, args, err := applyDeviceFilters(base, filter).ToSql()
	require.NoError(t, err)

	assert.Contains(t, q, "product_id", "product_id 过滤必须出现在 SQL 中")
	assert.Contains(t, q, "serial_number IN", "sn_list 必须渲染为 serial_number IN (...)")
	// 1 个 product_id + 3 个 SN = 4 个占位参数。
	assert.Len(t, args, 4)

	// product_id 单独传入也必须命中（最小复现原 bug 的场景）。
	q2, args2, err := applyDeviceFilters(
		sq.Select("*").From("devices d").PlaceholderFormat(sq.Dollar),
		DeviceFilter{ProductID: &pid},
	).ToSql()
	require.NoError(t, err)
	assert.True(t, strings.Contains(q2, "product_id"))
	require.Len(t, args2, 1)
	// squirrel 把 uuid.UUID 以字符串形式入参，pgx 端再转回 uuid——值正确即可。
	assert.Equal(t, pid.String(), toStr(args2[0]))
}

func toStr(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case uuid.UUID:
		return x.String()
	default:
		return ""
	}
}

// #11: software_version 子查询必须用 SELECT DISTINCT device_id，避免 device_parameters
// 同 device_id 多行导致 IN 半连接膨胀。直接断言 helper 生成的 SQL。
func TestSoftwareVersionDeviceIDSubquery_UsesDistinct(t *testing.T) {
	sub := softwareVersionDeviceIDSubquery([]string{"v1.2.3"})
	q, args, err := sub.PlaceholderFormat(sq.Dollar).ToSql()
	require.NoError(t, err)

	assert.Contains(t, q, "SELECT DISTINCT device_id", "子查询必须 DISTINCT device_id（#11）")
	assert.Contains(t, q, "device_parameters")
	assert.Contains(t, q, "parameter_path")
	assert.Contains(t, q, "parameter_value")
	// 1 个 path + 1 个 value = 2 个占位参数。
	assert.Len(t, args, 2)
}

// 多值 software_version 仍 DISTINCT，且渲染为 parameter_value IN (...)。
func TestSoftwareVersionDeviceIDSubquery_MultiValue(t *testing.T) {
	sub := softwareVersionDeviceIDSubquery([]string{"v1", "v2", "v3"})
	q, args, err := sub.PlaceholderFormat(sq.Dollar).ToSql()
	require.NoError(t, err)

	assert.Contains(t, q, "SELECT DISTINCT device_id")
	assert.Contains(t, q, "parameter_value IN")
	// 1 个 path + 3 个 value = 4 个占位参数。
	assert.Len(t, args, 4)
}

// applyDeviceFilters 接入 software_version 过滤时，整条 SQL 必须把子查询的 DISTINCT 带上。
func TestApplyDeviceFilters_SoftwareVersion_Distinct(t *testing.T) {
	sv := "v9.9"
	base := sq.Select("*").From("devices d").PlaceholderFormat(sq.Dollar)
	q, args, err := applyDeviceFilters(base, DeviceFilter{SoftwareVersion: &sv}).ToSql()
	require.NoError(t, err)

	assert.Contains(t, q, "d.id IN", "software_version 走 device_id 半连接")
	assert.Contains(t, q, "SELECT DISTINCT device_id", "子查询 DISTINCT 必须出现在最终 SQL（#11）")
	assert.Len(t, args, 2)
}

// 失败/边界路径：未设 software_version 时不应出现该子查询。
func TestApplyDeviceFilters_NoSoftwareVersion_NoSubquery(t *testing.T) {
	base := sq.Select("*").From("devices d").PlaceholderFormat(sq.Dollar)
	q, _, err := applyDeviceFilters(base, DeviceFilter{}).ToSql()
	require.NoError(t, err)
	assert.NotContains(t, q, "device_parameters")
}

func TestApplyDeviceFilters_DeviceTypeTabs(t *testing.T) {
	base := sq.Select("*").From("devices d").PlaceholderFormat(sq.Dollar)

	upsSQL, upsArgs, err := applyDeviceFilters(base, DeviceFilter{DeviceType: DeviceListDeviceTypeUPS}).ToSql()
	require.NoError(t, err)
	assert.Contains(t, upsSQL, "COALESCE(d.product_class, '') LIKE 'UPS%'")
	assert.Empty(t, upsArgs)

	baseStationSQL, baseStationArgs, err := applyDeviceFilters(
		base,
		DeviceFilter{DeviceType: DeviceListDeviceTypeBaseStation},
	).ToSql()
	require.NoError(t, err)
	assert.Contains(t, baseStationSQL, "NOT (COALESCE(d.product_class, '') LIKE 'UPS%')")
	assert.Contains(t, baseStationSQL, "NOT (UPPER(COALESCE(d.product_class, '')) LIKE '%IMSCORE%')")
	assert.Empty(t, baseStationArgs)

	coreSQL, coreArgs, err := applyDeviceFilters(base, DeviceFilter{DeviceType: DeviceListDeviceTypeCoreNetwork}).ToSql()
	require.NoError(t, err)
	assert.Contains(t, coreSQL, "UPPER(COALESCE(d.product_class, '')) LIKE '%IMSCORE%'")
	assert.Empty(t, coreArgs)
}
