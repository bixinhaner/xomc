package mmlstandardloader

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// 业务套餐 v1 自洽性测试：每条规格的字段约束。
// 这些断言反退化：未来人工调整套餐表时，违反约定立刻挂掉。
func TestPackageSpecs_Invariants(t *testing.T) {
	pkgs := PackageSpecs()

	assert.GreaterOrEqual(t, len(pkgs), 10, "首版至少 10 个套餐")
	assert.LessOrEqual(t, len(pkgs), 30, "套餐数 30 是 UI 列表的友好上限，再多就该考虑二级分类")

	seenCode := make(map[string]bool, len(pkgs))
	for _, p := range pkgs {
		t.Run(p.Code, func(t *testing.T) {
			// Code 命名约定
			assert.True(t, strings.HasPrefix(p.Code, "LST_PKG_"),
				"Code 必须以 LST_PKG_ 开头，区分于 grouper 派生的 LST_<group_code>")
			assert.False(t, seenCode[p.Code], "Code 全局唯一（DB UNIQUE 约束）")
			seenCode[p.Code] = true

			// 中英文名都非空
			assert.NotEmpty(t, p.NameZh, "NameZh 不可空")
			assert.NotEmpty(t, p.NameEn, "NameEn 不可空")

			// Category 必须是 7 类之一
			validCats := map[string]bool{
				CategoryCellMgmt: true, CategoryNeighborMgmt: true,
				CategoryBaseStation: true, CategoryAlarmQuery: true,
				CategoryPerfMgmt: true, CategoryTransportMgmt: true,
				CategoryVersionMgmt: true,
			}
			assert.True(t, validCats[p.Category],
				"Category=%q 不在 7 类合法范围", p.Category)

			// target_paths 至少 1 条且每条形态合理
			assert.NotEmpty(t, p.TargetPaths, "TargetPaths 至少 1 条")
			for _, path := range p.TargetPaths {
				assert.NotEmpty(t, path, "path 不可空字符串")
				// 套餐用 partial path（以 "." 结尾）为主；精确 path 也允许
				// 但不应含 "*" / "?" 通配符（TR-069 不支持）
				assert.NotContains(t, path, "*", "TR-069 不支持 * 通配符")
				assert.NotContains(t, path, "?", "TR-069 不支持 ? 通配符")
				// path 必须以 "Device" 或 "DeviceGSM" 开头（standard-model 唯一两个顶层命名空间）
				validRoot := strings.HasPrefix(path, "Device.") ||
					strings.HasPrefix(path, "DeviceGSM.")
				assert.True(t, validRoot, "path %q 必须以 Device. 或 DeviceGSM. 开头", path)
			}
		})
	}
}

// 三个用户重点要求的套餐必须存在：DeviceInfo / CellConfig 整树 / LTE / NR
func TestPackageSpecs_UserHighPrioritySetMustExist(t *testing.T) {
	pkgs := PackageSpecs()
	codes := make(map[string]PackageSpec, len(pkgs))
	for _, p := range pkgs {
		codes[p.Code] = p
	}

	required := []string{
		"LST_PKG_DEVICE_BASIC",   // 基站基本信息 (DeviceInfo)
		"LST_PKG_CELL_ALL",       // 小区配置全集 (CellConfig.* 整树)
		"LST_PKG_CELL_LTE",       // 小区 LTE 分制式
		"LST_PKG_CELL_NR",        // 小区 NR 分制式
		"LST_PKG_FAULT_CURRENT",  // 当前告警
		"LST_PKG_TR069",          // TR-069 接口
	}
	for _, code := range required {
		assert.Contains(t, codes, code, "用户首版必要套餐 %s 缺失", code)
	}

	// 确认关键套餐的 path 指向预期子树
	assert.Equal(t,
		[]string{"Device.DeviceInfo."},
		codes["LST_PKG_DEVICE_BASIC"].TargetPaths)
	assert.Equal(t,
		[]string{"Device.Services.FAPService.{i}.CellConfig."},
		codes["LST_PKG_CELL_ALL"].TargetPaths)
	assert.Equal(t,
		[]string{"Device.Services.FAPService.{i}.CellConfig.LTE."},
		codes["LST_PKG_CELL_LTE"].TargetPaths)
	assert.Equal(t,
		[]string{"Device.Services.FAPService.{i}.CellConfig.{i}.NR."},
		codes["LST_PKG_CELL_NR"].TargetPaths)
}
