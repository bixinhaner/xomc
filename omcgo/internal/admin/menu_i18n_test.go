package admin

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMarshalNameI18n 锁回归：nil/空 map 归一化为 NULL，避免空 JSONB '{}'
// 与 NULL 在 fallback 链路里行为分裂（前端 nameI18n['zh-CN'] 取不到时应直接走 name）。
func TestMarshalNameI18n(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		in      map[string]string
		wantNil bool
	}{
		{"nil map → NULL", nil, true},
		{"empty map → NULL", map[string]string{}, true},
		{"single locale → JSON", map[string]string{"zh-CN": "运维"}, false},
		{"multi locale → JSON", map[string]string{"zh-CN": "运维", "en-US": "Ops"}, false},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := marshalNameI18n(tc.in)
			require.NoError(t, err)
			if tc.wantNil {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			raw, ok := got.([]byte)
			require.True(t, ok, "expected []byte")
			assert.Contains(t, string(raw), "zh-CN")
		})
	}
}

// TestUnmarshalNameI18n 验证 NULL / 空 bytes / 合法 JSON 三条路径。
func TestUnmarshalNameI18n(t *testing.T) {
	t.Parallel()

	got, err := unmarshalNameI18n(nil)
	require.NoError(t, err)
	assert.Nil(t, got)

	got, err = unmarshalNameI18n([]byte{})
	require.NoError(t, err)
	assert.Nil(t, got)

	got, err = unmarshalNameI18n([]byte(`{"zh-CN":"运维","en-US":"Ops"}`))
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"zh-CN": "运维", "en-US": "Ops"}, got)

	_, err = unmarshalNameI18n([]byte(`{not json`))
	assert.Error(t, err, "非法 JSON 必须返回 error，不能静默吞")
}

// TestApplyMenuOptionalCols_I18n 验证 scan helper 把可空列回填到 Menu 后，
// i18n_key 与 name_i18n 都进入正确的结构体字段。
func TestApplyMenuOptionalCols_I18n(t *testing.T) {
	t.Parallel()

	m := &Menu{}
	i18nKey := "nav.ops.command"
	raw := []byte(`{"zh-CN":"运维命令","en-US":"Commands"}`)

	err := applyMenuOptionalCols(m, nil, nil, nil, nil, nil, nil, &i18nKey, raw)
	require.NoError(t, err)
	assert.Equal(t, "nav.ops.command", m.I18nKey)
	assert.Equal(t, map[string]string{"zh-CN": "运维命令", "en-US": "Commands"}, m.NameI18n)
}

// TestApplyMenuOptionalCols_NullI18n 验证 NULL 时 Menu.I18nKey/NameI18n 保持零值。
func TestApplyMenuOptionalCols_NullI18n(t *testing.T) {
	t.Parallel()

	m := &Menu{}
	err := applyMenuOptionalCols(m, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err)
	assert.Empty(t, m.I18nKey)
	assert.Nil(t, m.NameI18n)
}

// TestCreateMenuRequest_AcceptsI18n 简单冒烟：CreateMenuRequest JSON Bind 能接收
// name_i18n / i18n_key 字段；不依赖 DB，仅验证类型签名与字段绑定。
func TestCreateMenuRequest_AcceptsI18n(t *testing.T) {
	t.Parallel()
	req := CreateMenuRequest{
		Name:          "运维管理",
		NameI18n:      map[string]string{"zh-CN": "运维管理", "en-US": "Operations"},
		I18nKey:       "nav.ops",
		Type:          "directory",
		PermissionKey: "ops",
	}
	assert.Equal(t, "Operations", req.NameI18n["en-US"])
	assert.Equal(t, "nav.ops", req.I18nKey)
}

// TestUpdateMenuRequest_PatchSemantics 指针字段的 patch 语义：
//   - nil → 不变更
//   - 非 nil 指向 empty map → 清空所有译文（repo 归一为 NULL）
//   - 非 nil 指向非空 map → 写入
func TestUpdateMenuRequest_PatchSemantics(t *testing.T) {
	t.Parallel()
	id := uuid.New()
	_ = id // 仅占位避免 lint，本测试不涉及 repo 真写入

	var nilReq UpdateMenuRequest
	assert.Nil(t, nilReq.NameI18n, "未初始化时 NameI18n 必须是 nil（语义：不变更）")

	empty := map[string]string{}
	reqEmpty := UpdateMenuRequest{NameI18n: &empty}
	require.NotNil(t, reqEmpty.NameI18n)
	assert.Empty(t, *reqEmpty.NameI18n)

	full := map[string]string{"zh-CN": "新名", "en-US": "New"}
	reqFull := UpdateMenuRequest{NameI18n: &full}
	require.NotNil(t, reqFull.NameI18n)
	assert.Equal(t, "New", (*reqFull.NameI18n)["en-US"])
}
