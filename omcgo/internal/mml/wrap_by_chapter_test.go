package mml

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestWrapByChapter_MixedChapterAndLoose 验证 chapter 折叠 + 老 catalog 保持顶层。
//
// 输入：7 个扁平 group，分布 SA(2) + SB(1) + ""(空, 4)。
// 期望：[SA 父(2 children), SB 父(1 child), 4 个空 chapter 的原顶层 group]。
func TestWrapByChapter_MixedChapterAndLoose(t *testing.T) {
	in := []GroupTreeNode{
		{GroupCode: "G_SA_A", ChapterCode: "SA", DisplayOrder: 1},
		{GroupCode: "G_SA_B", ChapterCode: "SA", DisplayOrder: 2},
		{GroupCode: "G_SB_A", ChapterCode: "SB", DisplayOrder: 1},
		{GroupCode: "G_OLD_1", ChapterCode: "", DisplayOrder: 90},
		{GroupCode: "G_OLD_2", ChapterCode: "", DisplayOrder: 91},
		{GroupCode: "G_OLD_3", ChapterCode: "", DisplayOrder: 92},
		{GroupCode: "G_OLD_4", ChapterCode: "", DisplayOrder: 93},
	}

	out := wrapByChapter(in)

	// 顶层 = 2 chapter 父 + 4 loose
	assert.Equal(t, 6, len(out), "2 chapters + 4 loose at top level")

	// 第 0 个：SA 章节合成父节点
	sa := out[0]
	assert.Equal(t, "chapter:SA", sa.GroupCode, "first wrapper is SA")
	assert.Equal(t, "SA", sa.ChapterCode)
	assert.Equal(t, "synthetic", sa.Source)
	assert.True(t, sa.CatalogProtected)
	assert.Nil(t, sa.Commands, "chapter node has no commands of its own")
	assert.Equal(t, 2, len(sa.Children), "SA wraps 2 original groups")
	assert.Equal(t, "G_SA_A", sa.Children[0].GroupCode)
	assert.Equal(t, "G_SA_B", sa.Children[1].GroupCode)
	// Name 含 SA · DeviceInfo — 设备信息参数管理
	assert.Contains(t, sa.Name, "SA")
	assert.Contains(t, sa.Name, "DeviceInfo")
	assert.Contains(t, sa.Name, "设备信息参数管理")
	assert.Equal(t, "SA · DeviceInfo", sa.NameI18n["en-US"])
	// 确定性 UUID（同 input 应同 ID）
	out2 := wrapByChapter(in)
	assert.Equal(t, sa.ID, out2[0].ID, "chapter UUID is deterministic")

	// 第 1 个：SB 章节
	sb := out[1]
	assert.Equal(t, "chapter:SB", sb.GroupCode)
	assert.Equal(t, 1, len(sb.Children))
	assert.Equal(t, "G_SB_A", sb.Children[0].GroupCode)

	// 第 2-5 个：4 个 loose group（保持原顺序）
	assert.Equal(t, "G_OLD_1", out[2].GroupCode)
	assert.Equal(t, "G_OLD_2", out[3].GroupCode)
	assert.Equal(t, "G_OLD_3", out[4].GroupCode)
	assert.Equal(t, "G_OLD_4", out[5].GroupCode)
}

// TestWrapByChapter_SingleGroupSingleChapter 单 chapter 单 group 仍包一层，保持 UI 一致性。
func TestWrapByChapter_SingleGroupSingleChapter(t *testing.T) {
	in := []GroupTreeNode{
		{GroupCode: "G_LONELY", ChapterCode: "SC", DisplayOrder: 5},
	}
	out := wrapByChapter(in)
	assert.Equal(t, 1, len(out))
	assert.Equal(t, "chapter:SC", out[0].GroupCode)
	assert.Equal(t, 1, len(out[0].Children))
	assert.Equal(t, "G_LONELY", out[0].Children[0].GroupCode)
	assert.Equal(t, "synthetic", out[0].Source)
}

// TestWrapByChapter_AllEmptyChapter 全部老 catalog（chapter_code 空）→ 完全不包装。
func TestWrapByChapter_AllEmptyChapter(t *testing.T) {
	in := []GroupTreeNode{
		{GroupCode: "G_OLD_A", ChapterCode: "", DisplayOrder: 1},
		{GroupCode: "G_OLD_B", ChapterCode: "", DisplayOrder: 2},
		{GroupCode: "G_OLD_C", ChapterCode: "", DisplayOrder: 3},
	}
	out := wrapByChapter(in)
	assert.Equal(t, 3, len(out), "no wrapping when all chapter_code empty")
	assert.Equal(t, "G_OLD_A", out[0].GroupCode)
	assert.Equal(t, "G_OLD_B", out[1].GroupCode)
	assert.Equal(t, "G_OLD_C", out[2].GroupCode)
}

// TestWrapByChapter_UnknownChapterFallback chapter_code 不在 chapterMetadata 中时
// 用 code 自身作为 name（容错未来扩展，如 SS、ST 等）。
func TestWrapByChapter_UnknownChapterFallback(t *testing.T) {
	in := []GroupTreeNode{
		{GroupCode: "G_UNKNOWN", ChapterCode: "SS", DisplayOrder: 1},
	}
	out := wrapByChapter(in)
	assert.Equal(t, 1, len(out))
	assert.Equal(t, "chapter:SS", out[0].GroupCode)
	assert.Equal(t, "SS", out[0].Name, "fallback name = code when metadata missing")
	assert.Equal(t, "SS", out[0].NameI18n["en-US"])
}

// TestWrapByChapter_ChapterOrder 18 个章节按 SA→SR 排序，DisplayOrder 1..18。
func TestWrapByChapter_ChapterOrder(t *testing.T) {
	// 故意以乱序输入
	codes := []string{"SR", "SA", "SK", "SF", "SC", "SB"}
	in := make([]GroupTreeNode, 0, len(codes))
	for _, c := range codes {
		in = append(in, GroupTreeNode{GroupCode: "G_" + c, ChapterCode: c, DisplayOrder: 1})
	}
	out := wrapByChapter(in)
	gotCodes := make([]string, len(out))
	for i, n := range out {
		gotCodes[i] = n.ChapterCode
	}
	assert.Equal(t, []string{"SA", "SB", "SC", "SF", "SK", "SR"}, gotCodes,
		"chapter wrappers sorted by SA→SR")

	// 抽样 DisplayOrder
	assert.Equal(t, 1, out[0].DisplayOrder)  // SA=1
	assert.Equal(t, 18, out[len(out)-1].DisplayOrder, "SR=18")
}

// TestChapterDisplayOrder 验证 chapter code → 1..18 映射。
func TestChapterDisplayOrder(t *testing.T) {
	assert.Equal(t, 1, chapterDisplayOrder("SA"))
	assert.Equal(t, 2, chapterDisplayOrder("SB"))
	assert.Equal(t, 18, chapterDisplayOrder("SR"))
	// 异常输入
	assert.Equal(t, 1000, chapterDisplayOrder(""))
	assert.Equal(t, 1000, chapterDisplayOrder("X"))
	assert.Equal(t, 1000, chapterDisplayOrder("ABC"))
}
