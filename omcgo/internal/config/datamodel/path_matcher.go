package datamodel

import (
	"regexp"
	"strconv"
	"strings"
)

var placeholderRegex = regexp.MustCompile(`\{(\d+)\}`)

// ContainsPlaceholder checks whether the path contains a {N} placeholder.
func ContainsPlaceholder(path string) bool {
	return placeholderRegex.MatchString(path)
}

// TemplateToBasePath strips the first {N} placeholder and everything after it,
// returning the base path for GPN discovery.
// "Device.Services.FAPService.{12}." → "Device.Services.FAPService."
func TemplateToBasePath(templatePath string) string {
	loc := placeholderRegex.FindStringIndex(templatePath)
	if loc == nil {
		return templatePath
	}
	return templatePath[:loc[0]]
}

// TemplateToRegex converts a template path to a compiled regex that matches
// actual device paths. Each {N} placeholder is replaced with a pattern that
// matches either a plain instance number (\d+) or another placeholder ({N}).
func TemplateToRegex(templatePath string) *regexp.Regexp {
	parts := placeholderRegex.Split(templatePath, -1)
	var sb strings.Builder
	sb.WriteString("^")
	for i, part := range parts {
		sb.WriteString(regexp.QuoteMeta(part))
		if i < len(parts)-1 {
			// Match either a plain instance number or a {N} placeholder.
			sb.WriteString(`(?:\d+|\{\d+\})`)
		}
	}
	sb.WriteString("$")
	return regexp.MustCompile(sb.String())
}

// InstanceRef describes an instance number found in a path.
type InstanceRef struct {
	Segment  string // The object name before the instance number, e.g. "FAPService"
	Instance int    // Instance number, e.g. 2
}

// ExtractInstanceNumbers extracts all instance numbers from an actual device path.
// "Device.Services.FAPService.2.CellConfig.LTE.EPC.PLMNList.3.PLMNID"
// → [{Segment:"FAPService", Instance:2}, {Segment:"PLMNList", Instance:3}]
func ExtractInstanceNumbers(actualPath string) []InstanceRef {
	parts := strings.Split(actualPath, ".")
	var refs []InstanceRef
	for i, part := range parts {
		if num, err := strconv.Atoi(part); err == nil && i > 0 {
			refs = append(refs, InstanceRef{
				Segment:  parts[i-1],
				Instance: num,
			})
		}
	}
	return refs
}

// ReplaceInstance replaces the (placeholderIndex)-th {N} placeholder in a template
// path with the given instance number.
func ReplaceInstance(templatePath string, placeholderIndex int, instanceNum int) string {
	count := 0
	return placeholderRegex.ReplaceAllStringFunc(templatePath, func(match string) string {
		idx := count
		count++
		if idx == placeholderIndex {
			return strconv.Itoa(instanceNum)
		}
		return match
	})
}

// CountPlaceholders returns the number of {N} placeholders in a path.
func CountPlaceholders(path string) int {
	return len(placeholderRegex.FindAllString(path, -1))
}
