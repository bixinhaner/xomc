package mml

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================
// mml_renderer_test.go — T-0123-P1
//
// 覆盖：
//   - LST/MOD/ADD/RMV 4 op × (空字段 / 单字段 / 多字段 / 含特殊字符)
//   - 老 OMC playwright 实测样例 round-trip 验证
//   - sort_order 决定顺序（同 sort_order 内字典序稳定）
//   - 多 statement 渲染（; 分隔 + 末尾 ;）
//   - 错误路径（空 logical_code / 非法 op）
// ============================================================

// fixtureSubFields 构造一组测试用 sub_fields。
// 模拟老 OMC 实测的 Device info 命令 14 sub-field 中的几条。
func fixtureSubFields() []MMLCommandSubField {
	return []MMLCommandSubField{
		{ID: uuid.MustParse("11111111-0000-0000-0000-000000000001"), MMLCode: "LTE_GSM_MODEL_NAME", SortOrder: 1},
		{ID: uuid.MustParse("22222222-0000-0000-0000-000000000002"), MMLCode: "LTE_GSM_SYS_TIME", SortOrder: 2},
		{ID: uuid.MustParse("33333333-0000-0000-0000-000000000003"), MMLCode: "LTE_GSM_IP", SortOrder: 3},
		{ID: uuid.MustParse("44444444-0000-0000-0000-000000000004"), MMLCode: "DEVICEGSM_MCC", SortOrder: 4},
		{ID: uuid.MustParse("55555555-0000-0000-0000-000000000005"), MMLCode: "DEVICEGSM_MNC", SortOrder: 5},
	}
}

// ============================================================
// RenderStatement — LST
// ============================================================

func TestRenderStatement_LST_AllSelected_MatchesOldOMC(t *testing.T) {
	subFields := fixtureSubFields()
	stmt := Statement{
		LogicalCode:   "DEVICE_INFO",
		OperationType: "LST",
		SelectedSubFieldIDs: []uuid.UUID{
			subFields[0].ID, subFields[1].ID, subFields[2].ID,
		},
	}
	got, err := RenderStatement(stmt, subFields)
	require.NoError(t, err)
	// 老 OMC 实测样例：LST DEVICE_INFO:lstId={LTE_GSM_MODEL_NAME,LTE_GSM_SYS_TIME,LTE_GSM_IP}
	assert.Equal(t, "LST DEVICE_INFO:lstId={LTE_GSM_MODEL_NAME,LTE_GSM_SYS_TIME,LTE_GSM_IP}", got)
}

func TestRenderStatement_LST_EmptySelection_BareOp(t *testing.T) {
	stmt := Statement{LogicalCode: "DEVICE_INFO", OperationType: "LST"}
	got, err := RenderStatement(stmt, fixtureSubFields())
	require.NoError(t, err)
	assert.Equal(t, "LST DEVICE_INFO", got)
}

func TestRenderStatement_LST_OutOfOrderInput_SortedBySortOrder(t *testing.T) {
	subFields := fixtureSubFields()
	stmt := Statement{
		LogicalCode:   "DEVICE_INFO",
		OperationType: "LST",
		// 故意 reverse 顺序输入
		SelectedSubFieldIDs: []uuid.UUID{
			subFields[4].ID, subFields[2].ID, subFields[0].ID,
		},
	}
	got, err := RenderStatement(stmt, subFields)
	require.NoError(t, err)
	// 输出严格按 sort_order
	assert.Equal(t, "LST DEVICE_INFO:lstId={LTE_GSM_MODEL_NAME,LTE_GSM_IP,DEVICEGSM_MNC}", got)
}

func TestRenderStatement_LST_UnknownIDsIgnored(t *testing.T) {
	subFields := fixtureSubFields()
	unknownID := uuid.New()
	stmt := Statement{
		LogicalCode:   "DEVICE_INFO",
		OperationType: "LST",
		SelectedSubFieldIDs: []uuid.UUID{
			unknownID, subFields[0].ID,
		},
	}
	got, err := RenderStatement(stmt, subFields)
	require.NoError(t, err)
	// 未知 ID 静默忽略
	assert.Equal(t, "LST DEVICE_INFO:lstId={LTE_GSM_MODEL_NAME}", got)
}

func TestRenderStatement_LST_LowercaseOpNormalized(t *testing.T) {
	subFields := fixtureSubFields()
	stmt := Statement{
		LogicalCode:         "DEVICE_INFO",
		OperationType:       "lst", // 小写
		SelectedSubFieldIDs: []uuid.UUID{subFields[0].ID},
	}
	got, err := RenderStatement(stmt, subFields)
	require.NoError(t, err)
	assert.Equal(t, "LST DEVICE_INFO:lstId={LTE_GSM_MODEL_NAME}", got)
}

// ============================================================
// RenderStatement — MOD / ADD
// ============================================================

func TestRenderStatement_MOD_Multi_OldOMCSample(t *testing.T) {
	subFields := fixtureSubFields()
	stmt := Statement{
		LogicalCode:   "DEVICE_INFO",
		OperationType: "MOD",
		Values: map[string]string{
			"DEVICEGSM_MCC": "460",
			"DEVICEGSM_MNC": "00",
		},
	}
	got, err := RenderStatement(stmt, subFields)
	require.NoError(t, err)
	// MCC sort_order=4 在 MNC sort_order=5 前
	assert.Equal(t, "MOD DEVICE_INFO:DEVICEGSM_MCC=460,DEVICEGSM_MNC=00", got)
}

func TestRenderStatement_MOD_EmptyValues_BareOp(t *testing.T) {
	stmt := Statement{LogicalCode: "DEVICE_INFO", OperationType: "MOD"}
	got, err := RenderStatement(stmt, fixtureSubFields())
	require.NoError(t, err)
	assert.Equal(t, "MOD DEVICE_INFO", got)
}

func TestRenderStatement_MOD_UnknownKeyAppendedAlphabetically(t *testing.T) {
	subFields := fixtureSubFields()
	stmt := Statement{
		LogicalCode:   "DEVICE_INFO",
		OperationType: "MOD",
		Values: map[string]string{
			"DEVICEGSM_MCC": "460", // 已知 sort_order=4
			"ZZ_UNKNOWN":    "x",   // 未知 → 末尾
			"AA_UNKNOWN":    "y",   // 未知 → 末尾（字典序在 ZZ 前）
		},
	}
	got, err := RenderStatement(stmt, subFields)
	require.NoError(t, err)
	assert.Equal(t, "MOD DEVICE_INFO:DEVICEGSM_MCC=460,AA_UNKNOWN=y,ZZ_UNKNOWN=x", got)
}

func TestRenderStatement_ADD_SameSyntaxAsMOD(t *testing.T) {
	subFields := fixtureSubFields()
	stmt := Statement{
		LogicalCode:   "BTS_INFO",
		OperationType: "ADD",
		Values: map[string]string{
			"DEVICEGSM_MCC": "460",
		},
	}
	got, err := RenderStatement(stmt, subFields)
	require.NoError(t, err)
	assert.Equal(t, "ADD BTS_INFO:DEVICEGSM_MCC=460", got)
}

// ============================================================
// RenderStatement — RMV
// ============================================================

func TestRenderStatement_RMV_WithIndex(t *testing.T) {
	idx := 3
	stmt := Statement{
		LogicalCode:      "BTS_INFO",
		OperationType:    "RMV",
		RmvInstanceIndex: &idx,
	}
	got, err := RenderStatement(stmt, nil)
	require.NoError(t, err)
	assert.Equal(t, "RMV BTS_INFO:Index=3", got)
}

func TestRenderStatement_RMV_NoIndex_BareOp(t *testing.T) {
	stmt := Statement{LogicalCode: "BTS_INFO", OperationType: "RMV"}
	got, err := RenderStatement(stmt, nil)
	require.NoError(t, err)
	assert.Equal(t, "RMV BTS_INFO", got)
}

// ============================================================
// quoteValueIfNeeded — 特殊字符处理
// ============================================================

func TestRenderStatement_MOD_ValueWithSpace_Quoted(t *testing.T) {
	subFields := []MMLCommandSubField{
		{ID: uuid.New(), MMLCode: "DESC", SortOrder: 1},
	}
	stmt := Statement{
		LogicalCode:   "DEVICE_INFO",
		OperationType: "MOD",
		Values:        map[string]string{"DESC": "hello world"},
	}
	got, err := RenderStatement(stmt, subFields)
	require.NoError(t, err)
	assert.Equal(t, `MOD DEVICE_INFO:DESC="hello world"`, got)
}

func TestRenderStatement_MOD_ValueWithComma_Quoted(t *testing.T) {
	subFields := []MMLCommandSubField{
		{ID: uuid.New(), MMLCode: "LIST", SortOrder: 1},
	}
	stmt := Statement{
		LogicalCode:   "DEVICE_INFO",
		OperationType: "MOD",
		Values:        map[string]string{"LIST": "a,b,c"},
	}
	got, err := RenderStatement(stmt, subFields)
	require.NoError(t, err)
	assert.Equal(t, `MOD DEVICE_INFO:LIST="a,b,c"`, got)
}

func TestRenderStatement_MOD_ValueWithEmbeddedQuote_Escaped(t *testing.T) {
	subFields := []MMLCommandSubField{
		{ID: uuid.New(), MMLCode: "DESC", SortOrder: 1},
	}
	stmt := Statement{
		LogicalCode:   "DEVICE_INFO",
		OperationType: "MOD",
		Values:        map[string]string{"DESC": `say "hi"`},
	}
	got, err := RenderStatement(stmt, subFields)
	require.NoError(t, err)
	assert.Equal(t, `MOD DEVICE_INFO:DESC="say \"hi\""`, got)
}

// ============================================================
// 错误路径
// ============================================================

func TestRenderStatement_EmptyLogicalCode_Error(t *testing.T) {
	stmt := Statement{LogicalCode: "", OperationType: "LST"}
	_, err := RenderStatement(stmt, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty logical_code")
}

func TestRenderStatement_InvalidOp_Error(t *testing.T) {
	stmt := Statement{LogicalCode: "X", OperationType: "GET"} // GET 非 4 op 之一
	_, err := RenderStatement(stmt, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported operation")
}

// ============================================================
// RenderStatements — 多语句
// ============================================================

func TestRenderStatements_MultipleOps_TerminatedWithSemicolon(t *testing.T) {
	subFields := fixtureSubFields()
	stmts := []StatementWithSubFields{
		{
			Statement: Statement{
				LogicalCode:         "DEVICE_INFO",
				OperationType:       "LST",
				SelectedSubFieldIDs: []uuid.UUID{subFields[0].ID},
			},
			SubFields: subFields,
		},
		{
			Statement: Statement{
				LogicalCode:   "DEVICE_INFO",
				OperationType: "MOD",
				Values:        map[string]string{"DEVICEGSM_MCC": "460"},
			},
			SubFields: subFields,
		},
	}
	got, err := RenderStatements(stmts)
	require.NoError(t, err)
	assert.Equal(t,
		"LST DEVICE_INFO:lstId={LTE_GSM_MODEL_NAME};MOD DEVICE_INFO:DEVICEGSM_MCC=460;",
		got)
}

func TestRenderStatements_Empty_EmptyString(t *testing.T) {
	got, err := RenderStatements(nil)
	require.NoError(t, err)
	assert.Equal(t, "", got)
}

func TestRenderStatements_FirstFails_NoPartialOutput(t *testing.T) {
	stmts := []StatementWithSubFields{
		{Statement: Statement{LogicalCode: "", OperationType: "LST"}}, // 错误
		{Statement: Statement{LogicalCode: "DEVICE_INFO", OperationType: "LST"}},
	}
	_, err := RenderStatements(stmts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "statement[0]")
}

// ============================================================
// 老 OMC 实测样例完整 round-trip
// ============================================================

// 老 OMC playwright 实测：Device info(LST DEVICE_INFO) 14 sub-field 全勾选时的
// MML 字符串（截取前 5 个验证排序正确）。
func TestRenderStatement_LST_OldOMCFullSample(t *testing.T) {
	subFields := fixtureSubFields()
	all := []uuid.UUID{
		subFields[0].ID, subFields[1].ID, subFields[2].ID, subFields[3].ID, subFields[4].ID,
	}
	stmt := Statement{
		LogicalCode:         "DEVICE_INFO",
		OperationType:       "LST",
		SelectedSubFieldIDs: all,
	}
	got, err := RenderStatement(stmt, subFields)
	require.NoError(t, err)
	assert.Equal(t,
		"LST DEVICE_INFO:lstId={LTE_GSM_MODEL_NAME,LTE_GSM_SYS_TIME,LTE_GSM_IP,DEVICEGSM_MCC,DEVICEGSM_MNC}",
		got)
}

// ============================================================
// helper: isValidOp + collectSelectedMMLCodes 边界
// ============================================================

func TestIsValidOp_All4Pass(t *testing.T) {
	for _, op := range []string{"LST", "MOD", "ADD", "RMV"} {
		assert.True(t, isValidOp(op), "op %q should be valid", op)
	}
	for _, op := range []string{"GET", "SET", "lst", "", "DELETE"} {
		assert.False(t, isValidOp(op), "op %q should NOT be valid", op)
	}
}

func TestCollectSelectedMMLCodes_NilInputs(t *testing.T) {
	assert.Nil(t, collectSelectedMMLCodes(nil, nil))
	assert.Nil(t, collectSelectedMMLCodes([]uuid.UUID{uuid.New()}, nil))
	assert.Nil(t, collectSelectedMMLCodes(nil, fixtureSubFields()))
}
