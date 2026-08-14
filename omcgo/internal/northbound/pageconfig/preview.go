package pageconfig

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

var (
	optionalSegmentPattern = regexp.MustCompile(`\[-#[^\]]+#\]`)
	choicePattern          = regexp.MustCompile(`\{([^{}|]+)(?:\|[^{}]*)*\}`)
	tokenPattern           = regexp.MustCompile(`#[A-Za-z0-9-]+#`)
	knownExtensionPattern  = regexp.MustCompile(`(?i)\.(csv|xml|txt)$`)
)

var supportedTemplateTokens = map[string]struct{}{
	"#FTPRoot#":         {},
	"#Province#":        {},
	"#OMC-R#":           {},
	"#DateTime#":        {},
	"#Date#":            {},
	"#PeriodStartTime#": {},
	"#PeriodEndTime#":   {},
	"#LocalHost#":       {},
	"#DataVersion#":     {},
	"#DataPeriod#":      {},
	"#Object#":          {},
	"#ModuleType#":      {},
	"#eNBID#":           {},
	"#Ri#":              {},
	"#FileID#":          {},
}

func buildFileProfilePreview(profile FileProfile) FileProfilePreview {
	return buildFileProfilePreviewWithLocalHost(profile, configuredLocalHostToken(""))
}

func buildFileProfilePreviewWithLocalHost(profile FileProfile, localHost string) FileProfilePreview {
	items := make([]FileGroupPreview, 0, len(profile.Groups))
	for _, group := range profile.Groups {
		items = append(items, buildFileGroupPreviewWithLocalHost(group, localHost))
	}
	return FileProfilePreview{
		ProfileCode: profile.Code,
		Items:       items,
	}
}

func buildFileGroupPreview(group FileGroup) FileGroupPreview {
	return buildFileGroupPreviewWithLocalHost(group, configuredLocalHostToken(""))
}

func buildFileGroupPreviewWithLocalHost(group FileGroup, localHost string) FileGroupPreview {
	if group.Domain == DomainLOG {
		return buildLogFileGroupPreviewWithLocalHost(group, localHost)
	}
	tokens := previewTokenValues(group, localHost)
	path, pathUnknownTokens := renderTemplate(group.PathTemplate, tokens)
	fileName, fileUnknownTokens := renderTemplate(group.FileNameTemplate, tokens)
	fileName = ensurePreviewFileExtension(fileName, group.Format)
	artifactName := fileName
	if group.CompressionEnabled {
		artifactName = fmt.Sprintf("%s.%s", artifactName, compressionFormatOrDefault(group.CompressionFormat))
	}

	unknownTokens := uniqueStrings(append(pathUnknownTokens, fileUnknownTokens...))
	warnings := append(validateTemplatePath(path), validateTemplateFileName(fileName)...)
	if len(unknownTokens) > 0 {
		warnings = append(warnings, "template contains unsupported tokens")
	}

	return FileGroupPreview{
		GroupID:              group.ID,
		Domain:               group.Domain,
		Format:               group.Format,
		Period:               group.Period,
		PathTemplate:         group.PathTemplate,
		FileNameTemplate:     group.FileNameTemplate,
		CSVSeparator:         group.CSVSeparator,
		PreviewPath:          path,
		PreviewFileName:      fileName,
		CompressionEnabled:   group.CompressionEnabled,
		CompressionFormat:    compressionFormatOrDefault(group.CompressionFormat),
		PreviewArtifactName:  artifactName,
		UnsupportedTokenKeys: unknownTokens,
		Warnings:             uniqueStrings(warnings),
	}
}

func buildLogFileGroupPreview(group FileGroup) FileGroupPreview {
	return buildLogFileGroupPreviewWithLocalHost(group, configuredLocalHostToken(""))
}

func buildLogFileGroupPreviewWithLocalHost(group FileGroup, localHost string) FileGroupPreview {
	windowEnd := time.Date(2026, 8, 4, 0, 0, 0, 0, time.Local)
	windowStart := windowEnd.Add(-periodDuration(group.Period))
	previewObject := ScenarioObject{Code: previewObjectCode(group)}
	path, artifactName := renderLogArtifactNameWithLocalHost(group, previewObject, windowStart, windowEnd, 1, localHost)
	fileName := artifactName
	if group.CompressionEnabled {
		suffix := "." + string(compressionFormatOrDefault(group.CompressionFormat))
		fileName = strings.TrimSuffix(fileName, suffix)
	}
	warnings := append(validateTemplatePath(path), validateTemplateFileName(fileName)...)
	return FileGroupPreview{
		GroupID:             group.ID,
		Domain:              group.Domain,
		Format:              group.Format,
		Period:              group.Period,
		PathTemplate:        group.PathTemplate,
		FileNameTemplate:    group.FileNameTemplate,
		CSVSeparator:        group.CSVSeparator,
		PreviewPath:         path,
		PreviewFileName:     fileName,
		CompressionEnabled:  group.CompressionEnabled,
		CompressionFormat:   compressionFormatOrDefault(group.CompressionFormat),
		PreviewArtifactName: artifactName,
		Warnings:            uniqueStrings(warnings),
	}
}

func previewTokenValues(group FileGroup, localHost string) map[string]string {
	objectCode := previewObjectCode(group)
	localHost = configuredLocalHostToken(localHost)
	return map[string]string{
		"#FTPRoot#":         "northupload",
		"#Province#":        "GD",
		"#OMC-R#":           "BaiOMC",
		"#DateTime#":        "20260804000000",
		"#Date#":            "20260804",
		"#PeriodStartTime#": "20260803234500",
		"#PeriodEndTime#":   "20260804000000",
		"#LocalHost#":       localHost,
		"#DataVersion#":     "1.0",
		"#DataPeriod#":      previewDataPeriod(group.Period),
		"#Object#":          objectCode,
		"#ModuleType#":      objectCode,
		"#eNBID#":           "100001",
		"#Ri#":              "1",
		"#FileID#":          "0001",
	}
}

func previewObjectCode(group FileGroup) string {
	if len(group.Objects) == 0 || strings.TrimSpace(group.Objects[0].Code) == "" {
		return string(group.Domain)
	}
	return strings.TrimSpace(group.Objects[0].Code)
}

func previewDataPeriod(period Period) string {
	switch period {
	case Period15M:
		return "15"
	case Period60M:
		return "60"
	case Period24H:
		return "1440"
	default:
		return strings.TrimFunc(string(period), func(r rune) bool {
			return r < '0' || r > '9'
		})
	}
}

func renderTemplate(template string, tokens map[string]string) (string, []string) {
	rendered := optionalSegmentPattern.ReplaceAllString(template, "")
	// Resolve choice tokens like {CP|EP|CC|CE}: pick the alternative matching the
	// current object (#Object#), falling back to the first option when there is no
	// object or no match (preserves previous behaviour for non-object choices).
	objectCode := tokens["#Object#"]
	rendered = choicePattern.ReplaceAllStringFunc(rendered, func(match string) string {
		options := strings.Split(strings.Trim(match, "{}"), "|")
		for _, opt := range options {
			if objectCode != "" && strings.EqualFold(strings.TrimSpace(opt), objectCode) {
				return strings.TrimSpace(opt)
			}
		}
		return options[0]
	})
	unsupported := make([]string, 0)
	rendered = tokenPattern.ReplaceAllStringFunc(rendered, func(token string) string {
		if value, ok := tokens[token]; ok {
			return value
		}
		unsupported = append(unsupported, token)
		return token
	})
	return rendered, uniqueStrings(unsupported)
}

func unsupportedTemplateTokens(template string) []string {
	matches := tokenPattern.FindAllString(template, -1)
	unsupported := make([]string, 0)
	for _, token := range matches {
		if _, ok := supportedTemplateTokens[token]; !ok {
			unsupported = append(unsupported, token)
		}
	}
	return uniqueStrings(unsupported)
}

func ensurePreviewFileExtension(fileName string, format OutputFormat) string {
	extension := strings.ToLower(string(format))
	if extension == "" {
		return fileName
	}
	normalizedName := knownExtensionPattern.ReplaceAllString(fileName, "")
	return fmt.Sprintf("%s.%s", normalizedName, extension)
}

func compressionFormatOrDefault(format CompressionFormat) CompressionFormat {
	if format == CompressionGz {
		return CompressionGz
	}
	return CompressionZip
}

func validateTemplatePath(path string) []string {
	warnings := make([]string, 0)
	if strings.TrimSpace(path) == "" {
		warnings = append(warnings, "path template must not be empty")
	}
	if !strings.HasPrefix(path, "/") {
		warnings = append(warnings, "path template must be an absolute path")
	}
	if strings.Contains(path, "..") {
		warnings = append(warnings, "path template must not contain '..'")
	}
	if strings.ContainsAny(path, "\x00\r\n") {
		warnings = append(warnings, "path template must not contain control characters")
	}
	return warnings
}

func validateTemplateFileName(fileName string) []string {
	warnings := make([]string, 0)
	if strings.TrimSpace(fileName) == "" {
		warnings = append(warnings, "file name template must not be empty")
	}
	if strings.ContainsAny(fileName, `/\`) {
		warnings = append(warnings, "file name template must not contain path separators")
	}
	if strings.Contains(fileName, "..") {
		warnings = append(warnings, "file name template must not contain '..'")
	}
	if strings.ContainsAny(fileName, "\x00\r\n") {
		warnings = append(warnings, "file name template must not contain control characters")
	}
	return warnings
}

func validateFileGroupTemplates(group FileGroup) []string {
	errors := append(validateTemplatePath(group.PathTemplate), validateTemplateFileName(group.FileNameTemplate)...)
	if tokens := unsupportedTemplateTokens(group.PathTemplate); len(tokens) > 0 {
		errors = append(errors, fmt.Sprintf("path template contains unsupported tokens: %s", strings.Join(tokens, ", ")))
	}
	if tokens := unsupportedTemplateTokens(group.FileNameTemplate); len(tokens) > 0 {
		errors = append(errors, fmt.Sprintf("file name template contains unsupported tokens: %s", strings.Join(tokens, ", ")))
	}
	return uniqueStrings(errors)
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
