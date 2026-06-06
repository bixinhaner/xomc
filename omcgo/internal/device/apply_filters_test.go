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
