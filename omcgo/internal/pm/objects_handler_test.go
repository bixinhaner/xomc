package pm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// device 过滤 + distinct + 排序：不带制式时只有 $1，无 devices 子查询。
func Test_buildObjectsQuery_DeviceOnly(t *testing.T) {
	q, args := buildObjectsQuery([]string{"SN-1", "SN-2"}, nil)

	assert.Contains(t, q, "SELECT DISTINCT object_ldn")
	assert.Contains(t, q, "FROM pm_metrics")
	assert.Contains(t, q, "device_sn = ANY($1)")
	assert.Contains(t, q, "object_ldn <> ''")
	assert.NotContains(t, q, "FROM devices WHERE technology")
	assert.Contains(t, q, "ORDER BY object_ldn")
	assert.Equal(t, []any{[]string{"SN-1", "SN-2"}}, args)
}

// 带制式：叠加 devices 子查询制式过滤（$2），照 applyCommonFilters 范式。
func Test_buildObjectsQuery_WithTechnology(t *testing.T) {
	q, args := buildObjectsQuery([]string{"SN-1"}, []string{"lte"})

	assert.Contains(t, q, "device_sn = ANY($1)")
	assert.Contains(t, q, "(device_oui, device_sn) IN (SELECT oui, serial_number FROM devices WHERE technology = ANY($2))")
	assert.Len(t, args, 2)
	assert.Equal(t, []string{"SN-1"}, args[0])
	assert.Equal(t, []string{"lte"}, args[1])
}

// 空入参：SQL 仍合法（device_sn = ANY($1) 对空切片 → 无命中），返回空清单由 handler 兜底。
func Test_buildObjectsQuery_EmptyDevices(t *testing.T) {
	q, args := buildObjectsQuery([]string{}, nil)

	assert.Contains(t, q, "device_sn = ANY($1)")
	assert.NotContains(t, q, "technology")
	assert.Len(t, args, 1)
}

// parseObjectLDN 拆 Cellid / PLMN。
func Test_parseObjectLDN(t *testing.T) {
	tests := []struct {
		name       string
		ldn        string
		wantCellID string
		wantPLMN   string
	}{
		{"full", "Cellid=111172245,PLMN=46068", "111172245", "46068"},
		{"cell only", "Cellid=12345", "12345", ""},
		{"plmn only", "PLMN=46000", "", "46000"},
		{"none", "Foo=bar", "", ""},
		{"empty", "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cellID, plmn := parseObjectLDN(tt.ldn)
			assert.Equal(t, tt.wantCellID, cellID)
			assert.Equal(t, tt.wantPLMN, plmn)
		})
	}
}
