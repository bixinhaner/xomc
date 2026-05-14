package mml

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================
// mml_parser_test.go — T-0123-P1
//
// 覆盖：
//   - LST / MOD / ADD / RMV 4 op 基本解析
//   - 大小写容忍（lst → LST）+ 空格容忍
//   - 双引号值 + 内部转义 \" 还原
//   - 多 statement ; 分隔 + 连续 ;; 跳过 + 缺末尾 ;
//   - 双引号内 ; 不切割
//   - lookup 注入：SelectedMMLCodes → SelectedSubFieldIDs / unknown_codes 累加
//   - lookup 失败：累入 ParseError + 保留 stmt 语法解析结果
//   - 老 OMC 实测样例 round-trip：parse(render(x)) == x
//   - 错误路径：缺空格 / 非法 op / 缺 = / 空 code 等
// ============================================================

// ============================================================
// mock CommandLookup
// ============================================================

type fakeCommandLookup struct {
	commands       map[string]*MMLCommand        // key = "OP:CODE"
	subFieldsByCmd map[uuid.UUID][]MMLCommandSubField
	lookupErr      error
}

func newFakeLookup() *fakeCommandLookup {
	return &fakeCommandLookup{
		commands:       map[string]*MMLCommand{},
		subFieldsByCmd: map[uuid.UUID][]MMLCommandSubField{},
	}
}

func (f *fakeCommandLookup) addCommand(op, logical string, cmdID uuid.UUID, subFields []MMLCommandSubField) {
	c := &MMLCommand{ID: cmdID, OperationType: op, LogicalCode: logical}
	f.commands[op+":"+logical] = c
	f.subFieldsByCmd[cmdID] = subFields
}

func (f *fakeCommandLookup) LookupByLogicalCode(_ context.Context, op, logical string) (*MMLCommand, []MMLCommandSubField, error) {
	if f.lookupErr != nil {
		return nil, nil, f.lookupErr
	}
	c, ok := f.commands[op+":"+logical]
	if !ok {
		return nil, nil, ErrCommandNotFound
	}
	return c, f.subFieldsByCmd[c.ID], nil
}

// ============================================================
// 基本解析
// ============================================================

func TestParseMMLString_LST_BasicSyntax(t *testing.T) {
	stmts, errs := ParseMMLString(context.Background(),
		"LST DEVICE_INFO:lstId={LTE_GSM_MODEL_NAME,LTE_GSM_IP};", nil)
	require.Empty(t, errs)
	require.Len(t, stmts, 1)
	assert.Equal(t, "LST", stmts[0].OperationType)
	assert.Equal(t, "DEVICE_INFO", stmts[0].LogicalCode)
	assert.Equal(t, []string{"LTE_GSM_MODEL_NAME", "LTE_GSM_IP"}, stmts[0].SelectedMMLCodes)
}

func TestParseMMLString_MOD_BasicSyntax(t *testing.T) {
	stmts, errs := ParseMMLString(context.Background(),
		"MOD DEVICE_INFO:Mcc=460,Mnc=00;", nil)
	require.Empty(t, errs)
	require.Len(t, stmts, 1)
	assert.Equal(t, "MOD", stmts[0].OperationType)
	assert.Equal(t, "DEVICE_INFO", stmts[0].LogicalCode)
	assert.Equal(t, map[string]string{"Mcc": "460", "Mnc": "00"}, stmts[0].Values)
}

func TestParseMMLString_ADD_BasicSyntax(t *testing.T) {
	stmts, errs := ParseMMLString(context.Background(),
		"ADD BTS_INFO:BtsNum=3;", nil)
	require.Empty(t, errs)
	require.Len(t, stmts, 1)
	assert.Equal(t, "ADD", stmts[0].OperationType)
	assert.Equal(t, "BTS_INFO", stmts[0].LogicalCode)
}

func TestParseMMLString_RMV_BasicSyntax(t *testing.T) {
	stmts, errs := ParseMMLString(context.Background(), "RMV BTS_INFO:Index=5;", nil)
	require.Empty(t, errs)
	require.Len(t, stmts, 1)
	assert.Equal(t, "RMV", stmts[0].OperationType)
	require.NotNil(t, stmts[0].RmvInstanceIndex)
	assert.Equal(t, 5, *stmts[0].RmvInstanceIndex)
}

// ============================================================
// 大小写 + 空格容忍
// ============================================================

func TestParseMMLString_LowercaseOp_Normalized(t *testing.T) {
	stmts, errs := ParseMMLString(context.Background(), "lst DEVICE_INFO:lstid={X};", nil)
	require.Empty(t, errs)
	require.Len(t, stmts, 1)
	assert.Equal(t, "LST", stmts[0].OperationType)
	assert.Equal(t, []string{"X"}, stmts[0].SelectedMMLCodes)
}

func TestParseMMLString_ExtraWhitespace_Tolerated(t *testing.T) {
	stmts, errs := ParseMMLString(context.Background(),
		"  LST   DEVICE_INFO  :  lstId  =  {  X , Y }  ; ", nil)
	require.Empty(t, errs)
	require.Len(t, stmts, 1)
	// 注意：LST params 内部解析 lstId={...} 自身用 inner trim，所以多 token 间空格被吃
	// 但 {  X , Y } → 内 inner="X , Y" → split "," → trim → ["X", "Y"]
	assert.Equal(t, []string{"X", "Y"}, stmts[0].SelectedMMLCodes)
}

// ============================================================
// 多 statement + 分号容错
// ============================================================

func TestParseMMLString_MultipleStatements(t *testing.T) {
	stmts, errs := ParseMMLString(context.Background(),
		"LST DEVICE_INFO:lstId={A};MOD DEVICE_INFO:Mcc=460;RMV BTS_INFO:Index=1;",
		nil)
	require.Empty(t, errs)
	require.Len(t, stmts, 3)
	assert.Equal(t, "LST", stmts[0].OperationType)
	assert.Equal(t, "MOD", stmts[1].OperationType)
	assert.Equal(t, "RMV", stmts[2].OperationType)
}

func TestParseMMLString_ConsecutiveSemicolons_Skipped(t *testing.T) {
	stmts, errs := ParseMMLString(context.Background(),
		"LST A:lstId={X};;;MOD A:K=V;",
		nil)
	require.Empty(t, errs)
	require.Len(t, stmts, 2)
}

func TestParseMMLString_MissingTrailingSemicolon_Tolerated(t *testing.T) {
	stmts, errs := ParseMMLString(context.Background(),
		"LST DEVICE_INFO:lstId={X}", nil)
	require.Empty(t, errs)
	require.Len(t, stmts, 1)
}

func TestParseMMLString_EmptyInput_NoStatementsNoError(t *testing.T) {
	stmts, errs := ParseMMLString(context.Background(), "", nil)
	assert.Empty(t, stmts)
	assert.Empty(t, errs)
}

func TestParseMMLString_OnlyWhitespace_NoStatementsNoError(t *testing.T) {
	stmts, errs := ParseMMLString(context.Background(), "   ;  ;  ", nil)
	assert.Empty(t, stmts)
	assert.Empty(t, errs)
}

// ============================================================
// 双引号
// ============================================================

func TestParseMMLString_MOD_QuotedValueWithComma(t *testing.T) {
	stmts, errs := ParseMMLString(context.Background(),
		`MOD DEVICE_INFO:DESC="hello, world",K=V;`, nil)
	require.Empty(t, errs)
	require.Len(t, stmts, 1)
	assert.Equal(t, "hello, world", stmts[0].Values["DESC"])
	assert.Equal(t, "V", stmts[0].Values["K"])
}

func TestParseMMLString_MOD_QuotedValueWithSemicolon(t *testing.T) {
	// 双引号内的 ; 不切割 statement
	stmts, errs := ParseMMLString(context.Background(),
		`MOD DEVICE_INFO:DESC="a;b";LST X:lstId={Y};`, nil)
	require.Empty(t, errs)
	require.Len(t, stmts, 2)
	assert.Equal(t, "a;b", stmts[0].Values["DESC"])
}

func TestParseMMLString_MOD_EscapedQuoteInValue(t *testing.T) {
	stmts, errs := ParseMMLString(context.Background(),
		`MOD DEVICE_INFO:DESC="say \"hi\"";`, nil)
	require.Empty(t, errs)
	require.Len(t, stmts, 1)
	assert.Equal(t, `say "hi"`, stmts[0].Values["DESC"])
}

func TestParseMMLString_MOD_UnterminatedQuote_Error(t *testing.T) {
	_, errs := ParseMMLString(context.Background(),
		`MOD DEVICE_INFO:DESC="hello;`, nil)
	require.NotEmpty(t, errs)
	assert.Contains(t, errs[0].Reason, "未闭合")
}

// ============================================================
// 错误路径
// ============================================================

func TestParseMMLString_MissingSpaceBetweenOpAndCode(t *testing.T) {
	_, errs := ParseMMLString(context.Background(), "LSTDEVICE_INFO:lstId={X};", nil)
	require.NotEmpty(t, errs)
	assert.Contains(t, errs[0].Reason, "操作类型")
}

func TestParseMMLString_InvalidOp(t *testing.T) {
	_, errs := ParseMMLString(context.Background(), "GET DEVICE_INFO;", nil)
	require.NotEmpty(t, errs)
	assert.Contains(t, errs[0].Reason, "非法操作类型")
}

func TestParseMMLString_LST_MissingLstIdPrefix(t *testing.T) {
	_, errs := ParseMMLString(context.Background(),
		"LST DEVICE_INFO:wrongPrefix={X};", nil)
	require.NotEmpty(t, errs)
	assert.Contains(t, errs[0].Reason, "lstId")
}

func TestParseMMLString_LST_MissingBraces(t *testing.T) {
	_, errs := ParseMMLString(context.Background(),
		"LST DEVICE_INFO:lstId=X;", nil)
	require.NotEmpty(t, errs)
}

func TestParseMMLString_MOD_MissingEqualSign(t *testing.T) {
	_, errs := ParseMMLString(context.Background(),
		"MOD DEVICE_INFO:Mcc460;", nil)
	require.NotEmpty(t, errs)
	assert.Contains(t, errs[0].Reason, "=")
}

func TestParseMMLString_RMV_NonIntegerIndex(t *testing.T) {
	_, errs := ParseMMLString(context.Background(),
		"RMV BTS_INFO:Index=abc;", nil)
	require.NotEmpty(t, errs)
	assert.Contains(t, errs[0].Reason, "整数")
}

func TestParseMMLString_PartialFailure_ContinuesParsing(t *testing.T) {
	stmts, errs := ParseMMLString(context.Background(),
		"BAD CODE:bad;LST GOOD:lstId={X};INVALID;", nil)
	// Statement 2 (LST GOOD) 应该成功 — 即使 1/3 错
	require.Len(t, stmts, 1)
	assert.Equal(t, "LST", stmts[0].OperationType)
	assert.Equal(t, "GOOD", stmts[0].LogicalCode)
	require.Len(t, errs, 2)
}

// ============================================================
// LST 空字段 / MOD 空字段 / bare op
// ============================================================

func TestParseMMLString_LST_EmptyLstId(t *testing.T) {
	stmts, errs := ParseMMLString(context.Background(),
		"LST DEVICE_INFO:lstId={};", nil)
	require.Empty(t, errs)
	require.Len(t, stmts, 1)
	assert.Empty(t, stmts[0].SelectedMMLCodes)
}

func TestParseMMLString_BareLST_NoParams(t *testing.T) {
	stmts, errs := ParseMMLString(context.Background(),
		"LST DEVICE_INFO;", nil)
	require.Empty(t, errs)
	require.Len(t, stmts, 1)
	assert.Equal(t, "DEVICE_INFO", stmts[0].LogicalCode)
	assert.Empty(t, stmts[0].SelectedMMLCodes)
}

func TestParseMMLString_BareMOD_NoParams(t *testing.T) {
	stmts, errs := ParseMMLString(context.Background(),
		"MOD DEVICE_INFO;", nil)
	require.Empty(t, errs)
	require.Len(t, stmts, 1)
	assert.Empty(t, stmts[0].Values)
}

// ============================================================
// lookup 注入：SelectedMMLCodes → SelectedSubFieldIDs
// ============================================================

func TestParseMMLString_WithLookup_LST_ResolvesSubFieldIDs(t *testing.T) {
	cmdID := uuid.New()
	sfA := uuid.New()
	sfB := uuid.New()
	lookup := newFakeLookup()
	lookup.addCommand("LST", "DEVICE_INFO", cmdID, []MMLCommandSubField{
		{ID: sfA, MMLCode: "LTE_GSM_MODEL_NAME", SortOrder: 1},
		{ID: sfB, MMLCode: "LTE_GSM_IP", SortOrder: 3},
	})

	stmts, errs := ParseMMLString(context.Background(),
		"LST DEVICE_INFO:lstId={LTE_GSM_MODEL_NAME,LTE_GSM_IP};", lookup)
	require.Empty(t, errs)
	require.Len(t, stmts, 1)
	require.NotNil(t, stmts[0].CommandID)
	assert.Equal(t, cmdID, *stmts[0].CommandID)
	assert.Equal(t, []uuid.UUID{sfA, sfB}, stmts[0].SelectedSubFieldIDs)
	assert.Empty(t, stmts[0].UnknownCodes)
}

func TestParseMMLString_WithLookup_LST_UnknownCodesAccumulated(t *testing.T) {
	cmdID := uuid.New()
	sfA := uuid.New()
	lookup := newFakeLookup()
	lookup.addCommand("LST", "DEVICE_INFO", cmdID, []MMLCommandSubField{
		{ID: sfA, MMLCode: "LTE_GSM_MODEL_NAME", SortOrder: 1},
	})

	stmts, errs := ParseMMLString(context.Background(),
		"LST DEVICE_INFO:lstId={LTE_GSM_MODEL_NAME,UNKNOWN_FOO,ALSO_BAD};", lookup)
	require.Empty(t, errs)
	require.Len(t, stmts, 1)
	assert.Equal(t, []uuid.UUID{sfA}, stmts[0].SelectedSubFieldIDs)
	assert.ElementsMatch(t, []string{"UNKNOWN_FOO", "ALSO_BAD"}, stmts[0].UnknownCodes)
}

func TestParseMMLString_WithLookup_MOD_UnknownKeysFlagged(t *testing.T) {
	cmdID := uuid.New()
	lookup := newFakeLookup()
	lookup.addCommand("MOD", "DEVICE_INFO", cmdID, []MMLCommandSubField{
		{ID: uuid.New(), MMLCode: "Mcc"},
	})

	stmts, errs := ParseMMLString(context.Background(),
		"MOD DEVICE_INFO:Mcc=460,Unknown1=x,Unknown2=y;", lookup)
	require.Empty(t, errs)
	require.Len(t, stmts, 1)
	assert.ElementsMatch(t, []string{"Unknown1", "Unknown2"}, stmts[0].UnknownCodes)
}

func TestParseMMLString_WithLookup_CommandNotFound_ContinuesWithError(t *testing.T) {
	lookup := newFakeLookup()
	// no commands added → all lookups return ErrCommandNotFound

	stmts, errs := ParseMMLString(context.Background(),
		"LST DEVICE_INFO:lstId={X};MOD UNKNOWN_CMD:Y=1;", lookup)
	require.Len(t, errs, 2)
	for _, e := range errs {
		assert.Contains(t, e.Reason, "lookup")
	}
	// 语法解析成功的 stmt 仍保留
	require.Len(t, stmts, 2)
	assert.Nil(t, stmts[0].CommandID)
}

func TestParseMMLString_WithLookup_DBError_PassedThrough(t *testing.T) {
	lookup := newFakeLookup()
	lookup.lookupErr = errors.New("connection refused")

	_, errs := ParseMMLString(context.Background(),
		"LST DEVICE_INFO:lstId={X};", lookup)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Reason, "connection refused")
}

// ============================================================
// Round-trip：parse(render(x)) == x
// ============================================================

func TestRoundTrip_LST_OldOMCSample(t *testing.T) {
	subFields := fixtureSubFields()
	original := Statement{
		LogicalCode:         "DEVICE_INFO",
		OperationType:       "LST",
		SelectedSubFieldIDs: []uuid.UUID{subFields[0].ID, subFields[1].ID, subFields[2].ID},
	}
	rendered, err := RenderStatement(original, subFields)
	require.NoError(t, err)

	// 用 lookup 解析回来
	lookup := newFakeLookup()
	cmdID := uuid.New()
	lookup.addCommand("LST", "DEVICE_INFO", cmdID, subFields)

	parsed, errs := ParseMMLString(context.Background(), rendered+";", lookup)
	require.Empty(t, errs)
	require.Len(t, parsed, 1)

	// 顺序一致
	assert.Equal(t, original.SelectedSubFieldIDs, parsed[0].SelectedSubFieldIDs)
	assert.Equal(t, "LST", parsed[0].OperationType)
	assert.Equal(t, "DEVICE_INFO", parsed[0].LogicalCode)
}

func TestRoundTrip_MOD_QuotedValueSurvives(t *testing.T) {
	subFields := []MMLCommandSubField{
		{ID: uuid.New(), MMLCode: "DESC", SortOrder: 1},
	}
	original := Statement{
		LogicalCode:   "DEVICE_INFO",
		OperationType: "MOD",
		Values:        map[string]string{"DESC": `hello, "world"`},
	}
	rendered, err := RenderStatement(original, subFields)
	require.NoError(t, err)

	lookup := newFakeLookup()
	lookup.addCommand("MOD", "DEVICE_INFO", uuid.New(), subFields)

	parsed, errs := ParseMMLString(context.Background(), rendered+";", lookup)
	require.Empty(t, errs)
	require.Len(t, parsed, 1)
	assert.Equal(t, `hello, "world"`, parsed[0].Values["DESC"])
}

func TestRoundTrip_RMV(t *testing.T) {
	idx := 7
	original := Statement{
		LogicalCode:      "BTS_INFO",
		OperationType:    "RMV",
		RmvInstanceIndex: &idx,
	}
	rendered, err := RenderStatement(original, nil)
	require.NoError(t, err)
	assert.Equal(t, "RMV BTS_INFO:Index=7", rendered)

	parsed, errs := ParseMMLString(context.Background(), rendered+";", nil)
	require.Empty(t, errs)
	require.Len(t, parsed, 1)
	require.NotNil(t, parsed[0].RmvInstanceIndex)
	assert.Equal(t, 7, *parsed[0].RmvInstanceIndex)
}

// ============================================================
// splitStatements / 内部 helper
// ============================================================

func TestSplitStatements_NoQuotes(t *testing.T) {
	got := splitStatements("a;b;c;")
	assert.Equal(t, []string{"a", "b", "c"}, got)
}

func TestSplitStatements_QuotedSemicolonNotSplit(t *testing.T) {
	got := splitStatements(`a;"b;c";d`)
	assert.Equal(t, []string{`a`, `"b;c"`, `d`}, got)
}

func TestSplitStatements_EscapedQuote(t *testing.T) {
	got := splitStatements(`a;"b\";c";d`)
	// inside quotes \" is escape sequence; outer " closes at ; check
	require.Len(t, got, 3)
	assert.Equal(t, "a", got[0])
	assert.Contains(t, got[1], `b\"`)
	assert.Equal(t, "d", got[2])
}

func TestUnquoteIfNeeded_StripsAndUnescapes(t *testing.T) {
	assert.Equal(t, `hi`, unquoteIfNeeded(`hi`))
	assert.Equal(t, `hi`, unquoteIfNeeded(`"hi"`))
	assert.Equal(t, `say "hi"`, unquoteIfNeeded(`"say \"hi\""`))
}

// ============================================================
// 老 OMC 实测样例完整 parse
// ============================================================

func TestParseMMLString_OldOMCFullSample(t *testing.T) {
	// 老 OMC playwright 实测 LST DEVICE_INFO 全 14 字段勾选时的 MML 字符串
	raw := "LST DEVICE_INFO:lstId={LTE_GSM_MODEL_NAME,LTE_GSM_SYS_TIME,LTE_GSM_IP,LTE_GSM_MAC," +
		"LTE_GSM_SOFTWARE,LTE_GSM_HARDWARE,LTE_GSM_MME_STATUS,DEVICEGSM_MCC,DEVICEGSM_MNC," +
		"LTE_BTSNUM,BSC_ENCRYPTION,DEVICEGSM_TIMERNETT3212,DEVICEGSM_NRIBITLEN," +
		"DEVICEGSM_NRINULLADD};"

	stmts, errs := ParseMMLString(context.Background(), raw, nil)
	require.Empty(t, errs)
	require.Len(t, stmts, 1)
	assert.Len(t, stmts[0].SelectedMMLCodes, 14)
	// 第一个和最后一个验证（说明 split + trim 正确）
	assert.Equal(t, "LTE_GSM_MODEL_NAME", stmts[0].SelectedMMLCodes[0])
	assert.Equal(t, "DEVICEGSM_NRINULLADD", stmts[0].SelectedMMLCodes[13])
}

// ============================================================
// 反例：textbox 双向绑定模拟
// ============================================================

func TestParseMMLString_UserGradualEdit_OrderPreserved(t *testing.T) {
	// 模拟用户在 textbox 内手动调整 sub_field 顺序：
	// 渲染时按 sort_order，但用户手动改顺序后 textbox 里顺序不同。
	// 解析应保留用户输入顺序（不重新按 sort_order 排）。
	subFields := []MMLCommandSubField{
		{ID: uuid.MustParse("11111111-0000-0000-0000-000000000001"), MMLCode: "A", SortOrder: 1},
		{ID: uuid.MustParse("22222222-0000-0000-0000-000000000002"), MMLCode: "B", SortOrder: 2},
		{ID: uuid.MustParse("33333333-0000-0000-0000-000000000003"), MMLCode: "C", SortOrder: 3},
	}
	lookup := newFakeLookup()
	lookup.addCommand("LST", "X", uuid.New(), subFields)

	// 用户输入反序：C,B,A
	stmts, errs := ParseMMLString(context.Background(),
		"LST X:lstId={C,B,A};", lookup)
	require.Empty(t, errs)
	require.Len(t, stmts, 1)

	// SelectedSubFieldIDs 应保用户输入顺序 C,B,A
	assert.Equal(t,
		[]uuid.UUID{subFields[2].ID, subFields[1].ID, subFields[0].ID},
		stmts[0].SelectedSubFieldIDs)
	// 但 SelectedMMLCodes 也按用户顺序
	assert.Equal(t, []string{"C", "B", "A"}, stmts[0].SelectedMMLCodes)
}

// ============================================================
// 边界：长 statement / 长 mml_code 列表
// ============================================================

func TestParseMMLString_LargeSelection_NoErrors(t *testing.T) {
	codes := make([]string, 100)
	for i := range codes {
		codes[i] = "CODE_" + strings.Repeat("X", 30)
	}
	input := "LST X:lstId={" + strings.Join(codes, ",") + "};"

	stmts, errs := ParseMMLString(context.Background(), input, nil)
	require.Empty(t, errs)
	require.Len(t, stmts, 1)
	assert.Len(t, stmts[0].SelectedMMLCodes, 100)
}
