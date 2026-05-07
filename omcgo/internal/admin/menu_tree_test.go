package admin

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestAssembleMenuTree_ThreeLevels 锁回归：早期 buildTree 因值拷贝 bug 只能装 2 层。
// 现在必须保留 3 层（目录 → 菜单 → 按钮）。
func TestAssembleMenuTree_ThreeLevels(t *testing.T) {
	dir := uuid.New()
	menu := uuid.New()
	btn1 := uuid.New()
	btn2 := uuid.New()
	flat := []Menu{
		{ID: dir, Name: "目录", Type: "directory", ParentID: nil},
		{ID: menu, Name: "菜单", Type: "menu", ParentID: &dir},
		{ID: btn1, Name: "查询", Type: "button", ParentID: &menu},
		{ID: btn2, Name: "添加", Type: "button", ParentID: &menu},
	}

	roots := assembleMenuTree(flat)

	if assert.Len(t, roots, 1, "1 个根目录") {
		root := roots[0]
		assert.Equal(t, dir, root.ID)
		if assert.Len(t, root.Children, 1, "目录下 1 个菜单") {
			m := root.Children[0]
			assert.Equal(t, menu, m.ID)
			assert.Len(t, m.Children, 2, "菜单下 2 个按钮（旧实现这里为 0）")
		}
	}
}
