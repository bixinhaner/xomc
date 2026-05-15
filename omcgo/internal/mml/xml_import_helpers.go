// T-0132 XML import helpers (admin Tab 4)
//
// Helpers in this file are duplicated from omcctl/mml.go (`runMMLImport`) by
// design — admin UI import path is server-side runtime execution rather than
// SQL file rendering, but the *parsing logic / value type mapping / param_code
// derivation* must remain byte-for-byte identical to keep both pipelines
// (omcctl seed migration + admin UI re-import) writing the same rows.
//
// Future refactor: move shared helpers to `internal/config/parammodel/mmlstandardloader/import_builder.go`
// once admin UI import is proven stable. Until then this file stays the
// runtime mirror of omcctl/mml.go.

package mml

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/omcgo/omcgo/internal/config/parammodel/mmlstandardloader"
)

// importRow 单条 INSERT 行的中间数据（与 omcctl/mml.go 同步保持）。
type importRow struct {
	ParamCode          string
	Tr069Path          string
	ValueType          string
	AccessType         string
	IsObject           bool
	SupportsAdd        bool
	SupportsDelete     bool
	ChangeApplies      string
	NameI18n           map[string]string
	ConstraintTextI18n map[string]string
}

// buildImportRows 合并 params + objects → []importRow（按 tr069_path 升序）。
// 同步保持 omcctl/mml.go::buildImportRows。
func buildImportRows(params []mmlstandardloader.ParamSpec, objects []mmlstandardloader.ObjectSpec) []importRow {
	seenPath := make(map[string]struct{}, len(params)+len(objects))
	codeUsed := make(map[string]string, len(params)+len(objects))
	rows := make([]importRow, 0, len(params)+len(objects))

	addRow := func(p, valueType, access, change string, isObject bool, min, max *int64, maxLen *int) {
		if _, dup := seenPath[p]; dup {
			return
		}
		seenPath[p] = struct{}{}

		code := buildParamCodeXMLImport(p, codeUsed)
		codeUsed[code] = p

		access = normalizeAccessXMLImport(access)
		change = normalizeChangeAppliesXMLImport(change)
		supportsAdd := isObject && access == mmlstandardloader.AccessReadWrite
		supportsDelete := supportsAdd

		rows = append(rows, importRow{
			ParamCode:          code,
			Tr069Path:          p,
			ValueType:          mapValueTypeToDBXMLImport(valueType),
			AccessType:         access,
			IsObject:           isObject,
			SupportsAdd:        supportsAdd,
			SupportsDelete:     supportsDelete,
			ChangeApplies:      change,
			NameI18n:           renderNameI18nXMLImport(p),
			ConstraintTextI18n: renderConstraintTextI18nXMLImport(valueType, min, max, maxLen),
		})
	}

	for _, p := range params {
		addRow(p.StandardPath, p.Type, p.Access, p.ChangeApplies, false, p.Min, p.Max, p.MaxLen)
	}
	for _, o := range objects {
		addRow(o.StandardPath, "OBJECT", o.Access, o.ChangeApplies, true, nil, nil, nil)
	}

	// 不排序（preview 显示按 XML 中顺序更直观；apply UPSERT 顺序不影响结果）
	return rows
}

// buildParamCodeXMLImport 与 omcctl buildParamCode 一致：UPPER(leaf) + "_" + hash8(fullPath)。
func buildParamCodeXMLImport(p string, codeUsed map[string]string) string {
	leaf := lastSegmentXMLImport(p)
	leaf = strings.ToUpper(sanitizeCodePartXMLImport(leaf))
	if leaf == "" {
		leaf = "PATH"
	}

	full := sha256.Sum256([]byte(p))
	short := hex.EncodeToString(full[:])[:8]
	code := leaf + "_" + short

	if existPath, conflict := codeUsed[code]; conflict && existPath != p {
		code = leaf + "_" + hex.EncodeToString(full[:])[:12]
	}
	return code
}

func lastSegmentXMLImport(p string) string {
	t := strings.TrimSuffix(p, ".")
	if i := strings.LastIndex(t, "."); i >= 0 {
		return t[i+1:]
	}
	return t
}

func sanitizeCodePartXMLImport(s string) string {
	s = strings.ReplaceAll(s, "{i}", "")
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'A' && r <= 'Z',
			r >= 'a' && r <= 'z',
			r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	return b.String()
}

// mapValueTypeToDBXMLImport 把 XML type 翻译为 mml_params.value_type CHECK 白名单。
// 同步 omcctl mapValueTypeToDB（CHECK 约束 migration 000022）。
func mapValueTypeToDBXMLImport(xmlType string) string {
	switch strings.ToUpper(xmlType) {
	case "INT":
		return "int"
	case "U_INT", "UINT", "UNSIGNEDINT":
		return "unsignedInt"
	case "BOOLEAN", "BOOL":
		return "boolean"
	default:
		return "string"
	}
}

func normalizeAccessXMLImport(a string) string {
	switch strings.ToUpper(a) {
	case "READ_WRITE", "WRITE_ONLY":
		return strings.ToUpper(a)
	default:
		return "READ_ONLY"
	}
}

func normalizeChangeAppliesXMLImport(c string) string {
	if strings.EqualFold(c, "OnReboot") {
		return "OnReboot"
	}
	return "Immediate"
}

// renderNameI18nXMLImport 根据 path 末段生成 i18n。本期 admin import 简化：
// 仅 en-US 用 PascalCase 拆词；zh-CN 留空（admin UI 加 "🟡 需翻译" badge，或人工补）。
// omcctl 路径用全词典翻译（splitPascalCase + translateZh），但词典在 omcctl 包内 unexported；
// 跨包共享 future 重构 — 本期 admin import 用户可后续在 UI 内补 zh-CN。
func renderNameI18nXMLImport(p string) map[string]string {
	leaf := lastSegmentXMLImport(p)
	leaf = strings.ReplaceAll(leaf, "{i}", "")
	leaf = strings.TrimSpace(leaf)
	if leaf == "" {
		return map[string]string{}
	}
	en := splitPascalCaseXMLImport(leaf)
	return map[string]string{"en-US": en}
}

func splitPascalCaseXMLImport(s string) string {
	s = strings.ReplaceAll(s, "_", " ")
	var b strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			prev := rune(s[i-1])
			if (prev >= 'a' && prev <= 'z') || (prev >= '0' && prev <= '9') {
				b.WriteRune(' ')
			}
		}
		b.WriteRune(r)
	}
	out := b.String()
	for strings.Contains(out, "  ") {
		out = strings.ReplaceAll(out, "  ", " ")
	}
	return strings.TrimSpace(out)
}

// renderConstraintTextI18nXMLImport 综合 type/min/max/maxLen 渲染双语提示。
// 同步 omcctl renderConstraintTextI18n。
func renderConstraintTextI18nXMLImport(valueType string, min, max *int64, maxLen *int) map[string]string {
	t := strings.ToUpper(valueType)
	switch t {
	case "INT", "U_INT", "UNSIGNEDINT":
		if min != nil && max != nil {
			return map[string]string{
				"en-US": fmt.Sprintf("Integer, range: %d-%d", *min, *max),
				"zh-CN": fmt.Sprintf("整数，范围：%d-%d", *min, *max),
			}
		}
		if max != nil {
			return map[string]string{
				"en-US": fmt.Sprintf("Integer, max: %d", *max),
				"zh-CN": fmt.Sprintf("整数，上限：%d", *max),
			}
		}
		return map[string]string{"en-US": "Integer", "zh-CN": "整数"}
	case "BOOLEAN":
		return map[string]string{
			"en-US": "Boolean (true/false)",
			"zh-CN": "布尔值（true/false）",
		}
	case "STRING":
		if maxLen != nil {
			return map[string]string{
				"en-US": fmt.Sprintf("String, max length: %d", *maxLen),
				"zh-CN": fmt.Sprintf("字符串，最大长度：%d", *maxLen),
			}
		}
		return map[string]string{"en-US": "String", "zh-CN": "字符串"}
	case "DATE_TIME", "DATETIME":
		return map[string]string{
			"en-US": "DateTime (ISO 8601)",
			"zh-CN": "日期时间（ISO 8601）",
		}
	case "OBJECT":
		return map[string]string{
			"en-US": "TR-069 object container",
			"zh-CN": "TR-069 对象容器",
		}
	default:
		return map[string]string{"en-US": t}
	}
}
