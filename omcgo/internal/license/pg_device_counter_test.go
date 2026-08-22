package license

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCountDevicesSQLUsesRealtimeOnlineActiveDevices(t *testing.T) {
	assert.Contains(t, countDevicesSQL, "FROM devices")
	assert.Contains(t, countDevicesSQL, "is_online = true")
	assert.Contains(t, countDevicesSQL, "deleted_at IS NULL")
	assert.NotContains(t, countDevicesSQL, "system_license")
	assert.NotContains(t, countDevicesSQL, "cache")
}

func TestCountDevicesByTypeSQLPreAggregatesByProductID(t *testing.T) {
	assert.Contains(t, countDevicesByTypeSQL, "WITH online_products AS")
	assert.Contains(t, countDevicesByTypeSQL, "GROUP BY product_id")
	assert.Contains(t, countDevicesByTypeSQL, "LEFT JOIN products p ON op.product_id = p.id")
	assert.Contains(t, countDevicesByTypeSQL, "GROUP BY UPPER(COALESCE(p.alarm_ne_type, ''))")
	assert.NotContains(t, countDevicesByTypeSQL, "FROM devices d\n\t\tLEFT JOIN products")
}
