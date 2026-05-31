package dictsource

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDefault_BuildsRegistryWithKnownTables(t *testing.T) {
	reg, err := LoadDefault()
	require.NoError(t, err)

	tables := reg.ListTables()
	assert.GreaterOrEqual(t, len(tables), 7, "至少应有 7 张推荐核心表")
	// 抽样核心表
	for _, want := range []string{"devices", "products", "param_models", "topo_nodes", "alarm_definitions"} {
		_, ok := reg.Resolve(want)
		assert.True(t, ok, "expected %s in default whitelist", want)
	}
}

func TestResolve_TableHitAndMiss(t *testing.T) {
	reg, err := LoadDefault()
	require.NoError(t, err)

	spec, ok := reg.Resolve("devices")
	require.True(t, ok)
	assert.Equal(t, "devices", spec.Table)
	assert.NotEmpty(t, spec.DisplayName)
	assert.NotEmpty(t, spec.Fields)

	_, ok = reg.Resolve("not_a_real_table")
	assert.False(t, ok, "未登记表必须 miss")
}

func TestResolveField_HitAndMiss(t *testing.T) {
	reg, err := LoadDefault()
	require.NoError(t, err)

	fs, ok := reg.ResolveField("devices", "product_class")
	require.True(t, ok)
	assert.Equal(t, "product_class", fs.Column)
	assert.NotEmpty(t, fs.Type)

	// table 在白名单但字段不在
	_, ok = reg.ResolveField("devices", "password_hash")
	assert.False(t, ok, "未登记字段必须 miss")

	// table 不在白名单
	_, ok = reg.ResolveField("evil_table", "any_field")
	assert.False(t, ok)
}

func TestSecurity_SensitiveTablesNotInWhitelist(t *testing.T) {
	reg, err := LoadDefault()
	require.NoError(t, err)

	// 这些表绝不能进白名单
	for _, blocked := range []string{
		"sys_login_logs",
		"api_keys",
		"audit_logs",
		"system_license",
		"system_license_history",
	} {
		_, ok := reg.Resolve(blocked)
		assert.False(t, ok, "sensitive table %s must never be whitelisted", blocked)
	}
}

func TestLoadFromBytes_RejectsInvalidYAML(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string // substring expected in error
	}{
		{
			name: "empty file",
			body: ``,
			want: "empty sources list",
		},
		{
			name: "empty sources",
			body: "sources: []\n",
			want: "empty sources list",
		},
		{
			name: "missing table name",
			body: "sources:\n  - display_name: x\n    fields:\n      - {column: a, display: A, type: string}\n",
			want: "empty table name",
		},
		{
			name: "no fields",
			body: "sources:\n  - table: t1\n    display_name: X\n    fields: []\n",
			want: "no fields",
		},
		{
			name: "duplicate table",
			body: strings.Join([]string{
				"sources:",
				"  - table: t1",
				"    display_name: A",
				"    fields: [{column: c, display: C, type: string}]",
				"  - table: t1",
				"    display_name: B",
				"    fields: [{column: c, display: C, type: string}]",
				"",
			}, "\n"),
			want: "duplicate table",
		},
		{
			name: "duplicate field",
			body: strings.Join([]string{
				"sources:",
				"  - table: t1",
				"    display_name: A",
				"    fields:",
				"      - {column: c, display: C1, type: string}",
				"      - {column: c, display: C2, type: string}",
				"",
			}, "\n"),
			want: "duplicate field",
		},
		{
			name: "malformed yaml",
			body: "sources:\n  - table: t1\n    fields: not_a_list\n",
			want: "unmarshal",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LoadFromBytes([]byte(tc.body))
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}

func TestListTables_ReturnsSnapshot(t *testing.T) {
	reg, err := LoadDefault()
	require.NoError(t, err)

	got1 := reg.ListTables()
	got2 := reg.ListTables()
	require.Equal(t, len(got1), len(got2))

	// 改 caller 拷贝不应影响 registry 内部状态
	if len(got1) > 0 {
		got1[0].Table = "mutated"
	}
	got3 := reg.ListTables()
	if len(got3) > 0 {
		assert.NotEqual(t, "mutated", got3[0].Table, "ListTables must return defensive copy")
	}
}
