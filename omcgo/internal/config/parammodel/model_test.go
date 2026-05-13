package parammodel

import (
	"encoding/xml"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestXMLParamEntry_SupportedAttribute 验证 T-0103 新加的 supported XML 属性正确解析。
//
// 字典侧约定：缺省 / 任意非 "false" 字面值 → 视为 supported=true（默认含义），
// 仅严格 "false"（含大小写不敏感比较）→ 视为 false。
// 这里只断言原始字符串透传到 xmlParamEntry.Supported；
// 字符串 → bool 的转换在 loader.batchInsertMappings 由 EqualFold 完成。
func TestXMLParamEntry_SupportedAttribute(t *testing.T) {
	const snippet = `<?xml version="1.0"?>
<parameterModel paramModel="TestModel" totalEntries="3">
  <parameters>
    <param name="A.Default" standardPath="A.Default" type="STRING"/>
    <param name="A.SupportedFalse" standardPath="A.SupportedFalse" type="STRING" supported="false"/>
    <param name="A.SupportedTrue"  standardPath="A.SupportedTrue"  type="STRING" supported="true"/>
  </parameters>
</parameterModel>`

	var doc xmlParameterModel
	require.NoError(t, xml.Unmarshal([]byte(snippet), &doc))
	require.Len(t, doc.Params, 3)

	byName := map[string]xmlParamEntry{}
	for _, p := range doc.Params {
		byName[p.Name] = p
	}

	// 缺省 → 空串（loader 后续 EqualFold("","false")=false → IsSupported=true）
	assert.Equal(t, "", byName["A.Default"].Supported)
	// 显式 false
	assert.Equal(t, "false", byName["A.SupportedFalse"].Supported)
	// 显式 true
	assert.Equal(t, "true", byName["A.SupportedTrue"].Supported)
}
