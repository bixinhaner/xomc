package mml

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSortNodesByDisplayOrder_ChapterAware 验证根节点排序按 (chapter_code,
// display_order, group_code) 三键。
//
// Plan §6.6 "object 一级 + 按 SA-SR 顺序排列"：跨章节的对象级 group 必须按
// SA→SB→SC 顺序，章节内按 display_order。空 chapter（老 catalog）排末位。
func TestSortNodesByDisplayOrder_ChapterAware(t *testing.T) {
	nodes := []GroupTreeNode{
		{GroupCode: "G_SC_1", ChapterCode: "SC", DisplayOrder: 1},
		{GroupCode: "G_SA_2", ChapterCode: "SA", DisplayOrder: 2},
		{GroupCode: "G_SA_1", ChapterCode: "SA", DisplayOrder: 1},
		{GroupCode: "G_SB_1", ChapterCode: "SB", DisplayOrder: 1},
		{GroupCode: "G_OLD", ChapterCode: "", DisplayOrder: 99}, // 老 catalog 无章节
	}

	sortNodesByDisplayOrder(nodes)

	gotCodes := make([]string, len(nodes))
	for i, n := range nodes {
		gotCodes[i] = n.GroupCode
	}
	// 期望：SA 章节先（按 display_order），然后 SB，然后 SC，最后老 catalog
	assert.Equal(t, []string{"G_SA_1", "G_SA_2", "G_SB_1", "G_SC_1", "G_OLD"}, gotCodes,
		"chapter as primary sort key, display_order as secondary, empty chapter at tail")
}

// TestSortNodesByDisplayOrder_SameChapterDifferentOrder 同章节按 display_order。
func TestSortNodesByDisplayOrder_SameChapterDifferentOrder(t *testing.T) {
	nodes := []GroupTreeNode{
		{GroupCode: "G_3", ChapterCode: "SA", DisplayOrder: 3},
		{GroupCode: "G_1", ChapterCode: "SA", DisplayOrder: 1},
		{GroupCode: "G_2", ChapterCode: "SA", DisplayOrder: 2},
	}
	sortNodesByDisplayOrder(nodes)
	assert.Equal(t, "G_1", nodes[0].GroupCode)
	assert.Equal(t, "G_2", nodes[1].GroupCode)
	assert.Equal(t, "G_3", nodes[2].GroupCode)
}

// TestSortNodesByDisplayOrder_TieBreakByGroupCode 同章节同 display_order 时按 group_code。
func TestSortNodesByDisplayOrder_TieBreakByGroupCode(t *testing.T) {
	nodes := []GroupTreeNode{
		{GroupCode: "BBB", ChapterCode: "SA", DisplayOrder: 1},
		{GroupCode: "AAA", ChapterCode: "SA", DisplayOrder: 1},
	}
	sortNodesByDisplayOrder(nodes)
	assert.Equal(t, "AAA", nodes[0].GroupCode)
	assert.Equal(t, "BBB", nodes[1].GroupCode)
}

// TestChapterSortKey 验证 chapter 排序键归一化。
func TestChapterSortKey(t *testing.T) {
	// 非空 chapter：原样返回，确保 "SA" < "SB" < ... 字典序自然正确
	assert.Less(t, chapterSortKey("SA"), chapterSortKey("SB"))
	assert.Less(t, chapterSortKey("SB"), chapterSortKey("SC"))
	assert.Less(t, chapterSortKey("SR"), chapterSortKey(""))

	// 空 chapter：高位 sentinel，排末位
	assert.Equal(t, "~~~~~", chapterSortKey(""))
}

func TestSortCommandsByLogicalCode_SectionThenOperation(t *testing.T) {
	cmds := []GroupTreeCommand{
		{CommandCode: "RMV PLMN_LIST", LogicalCode: "PLMN_LIST", OperationType: "RMV"},
		{CommandCode: "LST SECURITY_GATEWAY", LogicalCode: "SECURITY_GATEWAY", OperationType: "LST"},
		{CommandCode: "MOD PLMN_LIST", LogicalCode: "PLMN_LIST", OperationType: "MOD"},
		{CommandCode: "ADD PLMN_LIST", LogicalCode: "PLMN_LIST", OperationType: "ADD"},
		{CommandCode: "LST PLMN_LIST", LogicalCode: "PLMN_LIST", OperationType: "LST"},
		{CommandCode: "MOD SECURITY_GATEWAY", LogicalCode: "SECURITY_GATEWAY", OperationType: "MOD"},
	}

	sortCommandsByLogicalCode(cmds)

	got := make([]string, len(cmds))
	for i, cmd := range cmds {
		got[i] = cmd.CommandCode
	}
	assert.Equal(t, []string{
		"LST PLMN_LIST",
		"MOD PLMN_LIST",
		"ADD PLMN_LIST",
		"RMV PLMN_LIST",
		"LST SECURITY_GATEWAY",
		"MOD SECURITY_GATEWAY",
	}, got)
}
