package mml

import (
	"strings"
	"testing"
)

// Issue #115 调整2：按 PATH 搜索只匹配 standard_path，纯描述命中不再返回。
// 搜索条件抽成纯函数后可不依赖 PG 直接断言生成的 SQL 形态。
func TestStandardParamSearchCondition(t *testing.T) {
	t.Run("空 / 纯空白返回 nil（不加 WHERE）", func(t *testing.T) {
		for _, in := range []string{"", "   ", "\t\n"} {
			if cond := standardParamSearchCondition(in); cond != nil {
				t.Fatalf("search=%q 期望 nil 条件，实际非 nil", in)
			}
		}
	})

	t.Run("非空只搜 standard_path，不再联合 description", func(t *testing.T) {
		cond := standardParamSearchCondition("Device.DeviceInfo")
		if cond == nil {
			t.Fatal("非空搜索期望返回条件，实际 nil")
		}
		sql, args, err := cond.ToSql()
		if err != nil {
			t.Fatalf("ToSql 失败: %v", err)
		}
		if !strings.Contains(sql, "standard_path ILIKE") {
			t.Errorf("SQL 应含 standard_path ILIKE，实际: %s", sql)
		}
		// 核心回归点：description 列绝不能再参与搜索（否则纯描述命中会返回不匹配 PATH 的条目）。
		if strings.Contains(strings.ToLower(sql), "description") {
			t.Errorf("SQL 不应再引用 description 列，实际: %s", sql)
		}
		if len(args) != 1 || args[0] != "%Device.DeviceInfo%" {
			t.Errorf("期望单参数 %%Device.DeviceInfo%%，实际: %v", args)
		}
	})
}
