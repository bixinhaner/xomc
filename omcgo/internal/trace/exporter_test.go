package trace

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrettyXML_EmptyInput(t *testing.T) {
	assert.Equal(t, "", prettyXML(""))
	assert.Equal(t, "   ", prettyXML("   "))
}

func TestPrettyXML_SingleLineBecomesMultiLine(t *testing.T) {
	input := `<soap-env:Envelope><soap-env:Header><cwmp:ID>123</cwmp:ID></soap-env:Header><soap-env:Body><cwmp:Inform/></soap-env:Body></soap-env:Envelope>`
	out := prettyXML(input)
	lines := strings.Split(out, "\n")
	assert.True(t, len(lines) >= 5, "expect multi-line output, got %d lines", len(lines))
	// 验证缩进：<cwmp:ID>123</cwmp:ID> 在 <soap-env:Header> 内应有 4 空格缩进
	idLine := ""
	for _, l := range lines {
		if strings.Contains(l, "<cwmp:ID>123</cwmp:ID>") {
			idLine = l
			break
		}
	}
	assert.NotEmpty(t, idLine, "should find <cwmp:ID> line")
	assert.True(t, strings.HasPrefix(idLine, "    "), "expected 4-space indent on cwmp:ID, got %q", idLine)
}

func TestPrettyXML_CDATAPreserved(t *testing.T) {
	// CDATA 内部含 < > 字符，绝不能被 tag boundary 切割
	input := `<root><script><![CDATA[if (a < b) { return a > b; }]]></script></root>`
	out := prettyXML(input)
	assert.Contains(t, out, `<![CDATA[if (a < b) { return a > b; }]]>`,
		"CDATA content must be preserved verbatim")
	// 同时 root/script 应被换行 + 缩进
	assert.Contains(t, out, "  <script>")
}

func TestPrettyXML_InlineTagNoExtraIndent(t *testing.T) {
	// <tag>text</tag> 同行的不应导致后续元素增加缩进
	input := `<root><a>x</a><b>y</b></root>`
	out := prettyXML(input)
	lines := strings.Split(out, "\n")
	// 应该是 3 行：<root> / 两个 inline / </root>
	assert.Len(t, lines, 4)
	assert.Equal(t, "<root>", lines[0])
	assert.Equal(t, "  <a>x</a>", lines[1])
	assert.Equal(t, "  <b>y</b>", lines[2])
	assert.Equal(t, "</root>", lines[3])
}

func TestPrettyXML_XMLDeclarationNotIndented(t *testing.T) {
	input := `<?xml version="1.0"?><root><a/></root>`
	out := prettyXML(input)
	lines := strings.Split(out, "\n")
	assert.Equal(t, `<?xml version="1.0"?>`, lines[0],
		"<?xml ...?> declaration should not increase depth")
	assert.Equal(t, "<root>", lines[1])
}

func TestPrettyXML_SelfClosingTagNoExtraIndent(t *testing.T) {
	input := `<root><a/><b/></root>`
	out := prettyXML(input)
	lines := strings.Split(out, "\n")
	assert.Equal(t, "<root>", lines[0])
	assert.Equal(t, "  <a/>", lines[1])
	assert.Equal(t, "  <b/>", lines[2])
	assert.Equal(t, "</root>", lines[3])
}
