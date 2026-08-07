package pageconfig

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strings"
	"time"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

const (
	defaultRunLimit = 200
	maxRunLimit     = 5000
)

func (s *Service) RunFileProfile(ctx context.Context, idOrCode string, req RunProfileRequest) (RunProfileResponse, error) {
	if s.repo == nil {
		return RunProfileResponse{}, fmt.Errorf("northbound page-config repository is not configured")
	}
	if err := s.repo.EnsureDefaults(ctx, s.catalog); err != nil {
		return RunProfileResponse{}, err
	}
	profile, err := s.getFileProfile(ctx, idOrCode)
	if err != nil {
		return RunProfileResponse{}, err
	}
	groups := selectFileGroups(profile.Groups, req.GroupID)
	if len(groups) == 0 {
		return RunProfileResponse{}, commonerrors.ErrNotFound
	}

	items := make([]FileRun, 0, len(groups))
	for _, group := range groups {
		for _, object := range group.Objects {
			run, runErr := s.buildFileRun(ctx, *profile, group, object, req)
			if runErr != nil {
				run = failedFileRun(ProfileKindFile, profile.Code, group, object, req, runErr)
			}
			created, err := s.repo.CreateFileRun(ctx, run)
			if err != nil {
				return RunProfileResponse{}, err
			}
			if _, err := s.repo.CreateEvent(ctx, eventFromFileRun(*created)); err != nil {
				return RunProfileResponse{}, err
			}
			archiveResult, _ := s.archiveRunLocally(ctx, *created)
			if err := s.deliverRunToTargets(ctx, *created); err != nil {
				return RunProfileResponse{}, err
			}
			if err := s.markRunArchived(ctx, created, archiveResult); err != nil {
				return RunProfileResponse{}, err
			}
			items = append(items, *created)
		}
	}
	return RunProfileResponse{
		ProfileKind: ProfileKindFile,
		ProfileCode: profile.Code,
		Items:       items,
		Total:       len(items),
	}, nil
}

func (s *Service) RunInventoryProfile(ctx context.Context, idOrCode string, req RunProfileRequest) (*FileRun, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("northbound page-config repository is not configured")
	}
	if err := s.repo.EnsureDefaults(ctx, s.catalog); err != nil {
		return nil, err
	}
	profile, err := s.getInventoryProfile(ctx, idOrCode)
	if err != nil {
		return nil, err
	}

	run, runErr := s.buildInventoryRun(ctx, *profile, req)
	if runErr != nil {
		run = failedInventoryRun(*profile, req, runErr)
	}
	created, err := s.repo.CreateFileRun(ctx, run)
	if err != nil {
		return nil, err
	}
	if _, err := s.repo.CreateEvent(ctx, eventFromFileRun(*created)); err != nil {
		return nil, err
	}
	archiveResult, _ := s.archiveRunLocally(ctx, *created)
	if err := s.deliverRunToTargets(ctx, *created); err != nil {
		return nil, err
	}
	if err := s.markRunArchived(ctx, created, archiveResult); err != nil {
		return nil, err
	}
	return created, nil
}

func (s *Service) ListRuns(ctx context.Context, filter RunFilter) (RunListResult, error) {
	if s.repo == nil {
		return RunListResult{}, fmt.Errorf("northbound page-config repository is not configured")
	}
	return s.repo.ListFileRuns(ctx, filter)
}

func (s *Service) GetRun(ctx context.Context, id string) (*FileRun, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("northbound page-config repository is not configured")
	}
	run, err := s.repo.GetFileRun(ctx, id)
	if err != nil {
		return nil, err
	}
	// Local archive hydration is best-effort: metadata remains readable even if
	// MinIO is temporarily unavailable, while successful archives show content
	// in the page-config result preview.
	_ = s.hydrateRunArtifactContent(ctx, run)
	return run, nil
}

func (s *Service) getInventoryProfile(ctx context.Context, idOrCode string) (*InventoryProfile, error) {
	items, err := s.ListInventoryProfiles(ctx)
	if err != nil {
		return nil, err
	}
	for i := range items {
		if strings.EqualFold(items[i].Code, idOrCode) || strings.EqualFold(items[i].ID, idOrCode) {
			return &items[i], nil
		}
	}
	return nil, commonerrors.ErrNotFound
}

func (s *Service) buildFileRun(ctx context.Context, profile FileProfile, group FileGroup, object ScenarioObject, req RunProfileRequest) (FileRun, error) {
	windowStart, windowEnd := normalizeRunWindow(group.Period, req)
	content, rowCount, err := s.generateFileContent(ctx, group, object, windowStart, windowEnd, req.Limit)
	if err != nil {
		return FileRun{}, err
	}
	artifactPath, artifactName := renderArtifactName(group, object, windowStart, windowEnd, 1)
	return FileRun{
		ProfileKind:        ProfileKindFile,
		ProfileCode:        profile.Code,
		GroupID:            group.ID,
		Domain:             group.Domain,
		ObjectCode:         object.Code,
		Status:             RunStatusSuccess,
		WindowStart:        &windowStart,
		WindowEnd:          &windowEnd,
		ArtifactPath:       artifactPath,
		ArtifactName:       artifactName,
		ArtifactContent:    content,
		ArtifactSize:       int64(len(content)),
		RowCount:           rowCount,
		CompressionEnabled: group.CompressionEnabled,
		CompressionFormat:  compressionFormatOrDefault(group.CompressionFormat),
		Summary: map[string]any{
			"profile_name": profile.Name,
			"format":       group.Format,
			"period":       group.Period,
		},
	}, nil
}

func (s *Service) buildInventoryRun(ctx context.Context, profile InventoryProfile, req RunProfileRequest) (FileRun, error) {
	windowStart, windowEnd := normalizeRunWindow(profile.Period, req)
	group := FileGroup{
		ID:                 profile.Code,
		Domain:             DomainInventory,
		Format:             FormatCSV,
		Period:             profile.Period,
		StartMinute:        profile.StartMinute,
		PathTemplate:       profile.PathTemplate,
		FileNameTemplate:   profile.FileNameTemplate,
		CompressionEnabled: profile.CompressionEnabled,
		CompressionFormat:  profile.CompressionFormat,
		Objects: []ScenarioObject{{
			Code: profile.ObjectCode,
			Tech: profile.Tech,
		}},
	}
	object := ScenarioObject{Code: profile.ObjectCode, Tech: profile.Tech}
	content, rowCount, err := s.generateInventoryContent(ctx, profile, req.Limit)
	if err != nil {
		return FileRun{}, err
	}
	artifactPath, artifactName := renderArtifactName(group, object, windowStart, windowEnd, 1)
	return FileRun{
		ProfileKind:        ProfileKindInventory,
		ProfileCode:        profile.Code,
		GroupID:            profile.Code,
		Domain:             DomainInventory,
		ObjectCode:         profile.ObjectCode,
		Status:             RunStatusSuccess,
		WindowStart:        &windowStart,
		WindowEnd:          &windowEnd,
		ArtifactPath:       artifactPath,
		ArtifactName:       artifactName,
		ArtifactContent:    content,
		ArtifactSize:       int64(len(content)),
		RowCount:           rowCount,
		CompressionEnabled: profile.CompressionEnabled,
		CompressionFormat:  compressionFormatOrDefault(profile.CompressionFormat),
		Summary: map[string]any{
			"profile_name": profile.Name,
			"format":       FormatCSV,
			"period":       profile.Period,
		},
	}, nil
}

// applySelectedFields narrows the catalog field set to the explicitly selected keys.
// A nil/empty wanted list means "all fields" (default), returning fields unchanged.
// Matching is by FieldDefinition.Key; catalog order is preserved. Unknown keys are
// silently dropped (the generator naturally ignores them).
func applySelectedFields(fields []FieldDefinition, wanted []string) []FieldDefinition {
	if len(wanted) == 0 {
		return fields
	}
	keep := make(map[string]struct{}, len(wanted))
	for _, k := range wanted {
		keep[k] = struct{}{}
	}
	out := make([]FieldDefinition, 0, len(fields))
	for _, f := range fields {
		if _, ok := keep[f.Key]; ok {
			out = append(out, f)
		}
	}
	if len(out) == 0 {
		// None of the selected keys matched this object's catalog fields — e.g. PM
		// per-metric keys against the fixed PM column set, or a stale/foreign
		// selection. Keep all fields rather than producing an empty/failed export.
		return fields
	}
	return out
}

func (s *Service) generateFileContent(ctx context.Context, group FileGroup, object ScenarioObject, windowStart, windowEnd time.Time, limit int) (string, int, error) {
	fields := s.catalog.Fields(FieldFilter{
		Domain:     group.Domain,
		ObjectCode: object.Code,
		Tech:       object.Tech,
		Profile:    object.Profile,
	})
	// Honor per-profile field selection: empty SelectedFields = all catalog fields
	// (default); non-empty = only the listed field keys, preserving catalog order.
	fields = applySelectedFields(fields, group.SelectedFields)
	if len(fields) == 0 {
		return "", 0, fmt.Errorf("%w: no supported fields for %s %s", commonerrors.ErrInvalidInput, group.Domain, object.Code)
	}

	var rows []ExportDataRow
	var err error
	switch group.Domain {
	case DomainCM:
		rows, err = s.repo.LoadDeviceSnapshotRows(ctx, object.Tech, normalizeLimit(limit))
	case DomainPM:
		// PM exports in fixed-column long format; per-profile selection filters WHICH
		// metric_paths are exported (SelectedFields carries metric paths for PM).
		metricPaths := metricPathsFromFields(fields)
		if len(group.SelectedFields) > 0 {
			metricPaths = group.SelectedFields
		}
		rows, err = s.repo.LoadPMMetricRows(ctx, PMMetricQuery{
			MetricPaths: metricPaths,
			Tech:        object.Tech,
			WindowStart: &windowStart,
			WindowEnd:   &windowEnd,
			Limit:       normalizeLimit(limit),
		})
	case DomainMR:
		rows, err = s.repo.LoadMRRows(ctx, object.Code, normalizeLimit(limit))
	case DomainLOG:
		rows, err = s.repo.LoadLogRows(ctx, object.Code, normalizeLimit(limit))
	default:
		err = fmt.Errorf("%w: unsupported file domain %s", commonerrors.ErrInvalidInput, group.Domain)
	}
	if err != nil {
		return "", 0, err
	}
	rows = enrichRows(rows, object, windowStart, windowEnd)
	// No real data in the window → produce no file (do not fabricate or deliver empties).
	if len(rows) == 0 {
		return "", 0, nil
	}
	// CM uses the Baicells DataFile XML format when format is XML (MR keeps its own XML format).
	if group.Domain == DomainCM && group.Format == FormatXML {
		return renderCMXML(fields, rows, windowEnd), len(rows), nil
	}
	return renderRows(group.Format, fields, rows)
}

func (s *Service) generateInventoryContent(ctx context.Context, profile InventoryProfile, limit int) (string, int, error) {
	fields := s.inventoryFields(profile)
	if len(fields) == 0 {
		return "", 0, fmt.Errorf("%w: no supported inventory fields for %s", commonerrors.ErrInvalidInput, profile.ObjectCode)
	}
	var rows []ExportDataRow
	var err error
	if strings.EqualFold(profile.ObjectCode, "OMC") {
		rows, err = s.repo.LoadOMCInventoryRows(ctx)
	} else {
		rows, err = s.repo.LoadDeviceSnapshotRows(ctx, profile.Tech, normalizeLimit(limit))
	}
	if err != nil {
		return "", 0, err
	}
	now := time.Now()
	rows = enrichRows(rows, ScenarioObject{Code: profile.ObjectCode, Tech: profile.Tech}, now, now)
	return renderRows(FormatCSV, fields, rows)
}

func (s *Service) inventoryFields(profile InventoryProfile) []FieldDefinition {
	if profile.Fields == nil {
		return s.catalog.Fields(FieldFilter{
			Domain:     DomainInventory,
			ObjectCode: profile.ObjectCode,
			Tech:       profile.Tech,
		})
	}

	fields := make([]FieldDefinition, 0, len(profile.Fields))
	for _, configured := range profile.Fields {
		if !configured.Enabled || strings.TrimSpace(configured.SystemField) == "" {
			continue
		}
		fields = append(fields, FieldDefinition{
			Key:           configured.Key,
			Domain:        DomainInventory,
			ObjectCode:    profile.ObjectCode,
			OutputAlias:   configured.OutputAlias,
			SystemField:   configured.SystemField,
			Source:        configured.Source,
			DataType:      configured.DataType,
			Renderer:      configured.Renderer,
			SupportStatus: SupportSupported,
		})
	}
	return fields
}

func renderRows(format OutputFormat, fields []FieldDefinition, rows []ExportDataRow) (string, int, error) {
	switch format {
	case FormatCSV:
		content, err := renderCSV(fields, rows)
		return content, len(rows), err
	case FormatTXT:
		content, err := renderDelimited(fields, rows, "|")
		return content, len(rows), err
	case FormatXML:
		return renderSimpleXML(fields, rows), len(rows), nil
	default:
		return "", 0, fmt.Errorf("%w: unsupported output format %s", commonerrors.ErrInvalidInput, format)
	}
}

func renderCSV(fields []FieldDefinition, rows []ExportDataRow) (string, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	if err := writer.Write(outputAliases(fields)); err != nil {
		return "", err
	}
	for _, row := range rows {
		if err := writer.Write(valuesForFields(fields, row)); err != nil {
			return "", err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func renderDelimited(fields []FieldDefinition, rows []ExportDataRow, sep string) (string, error) {
	var b strings.Builder
	b.WriteString(strings.Join(outputAliases(fields), sep))
	b.WriteByte('\n')
	for _, row := range rows {
		values := valuesForFields(fields, row)
		for i := range values {
			values[i] = strings.ReplaceAll(values[i], sep, " ")
		}
		b.WriteString(strings.Join(values, sep))
		b.WriteByte('\n')
	}
	return b.String(), nil
}

func renderSimpleXML(fields []FieldDefinition, rows []ExportDataRow) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	b.WriteByte('\n')
	b.WriteString("<NorthboundExport>\n")
	for _, row := range rows {
		b.WriteString("  <Row>\n")
		for _, field := range fields {
			b.WriteString("    <")
			b.WriteString(xmlTag(field.OutputAlias))
			b.WriteString(">")
			b.WriteString(escapeXML(row[field.SystemField]))
			b.WriteString("</")
			b.WriteString(xmlTag(field.OutputAlias))
			b.WriteString(">\n")
		}
		b.WriteString("  </Row>\n")
	}
	b.WriteString("</NorthboundExport>\n")
	return b.String()
}

func outputAliases(fields []FieldDefinition) []string {
	headers := make([]string, 0, len(fields))
	for _, field := range fields {
		headers = append(headers, field.OutputAlias)
	}
	return headers
}

func valuesForFields(fields []FieldDefinition, row ExportDataRow) []string {
	values := make([]string, 0, len(fields))
	for _, field := range fields {
		values = append(values, row[field.SystemField])
	}
	return values
}

func selectFileGroups(groups []FileGroup, groupID string) []FileGroup {
	if strings.TrimSpace(groupID) == "" {
		return groups
	}
	out := make([]FileGroup, 0, 1)
	for _, group := range groups {
		if strings.EqualFold(group.ID, groupID) {
			out = append(out, group)
		}
	}
	return out
}

func normalizeRunWindow(period Period, req RunProfileRequest) (time.Time, time.Time) {
	end := time.Now().Truncate(time.Minute)
	if req.WindowEnd != nil && !req.WindowEnd.IsZero() {
		end = req.WindowEnd.Truncate(time.Minute)
	}
	start := end.Add(-periodDuration(period))
	if req.WindowStart != nil && !req.WindowStart.IsZero() {
		start = req.WindowStart.Truncate(time.Minute)
	}
	return start, end
}

func periodDuration(period Period) time.Duration {
	switch period {
	case Period15M:
		return 15 * time.Minute
	case Period60M:
		return time.Hour
	case Period24H:
		return 24 * time.Hour
	case Period7D:
		return 7 * 24 * time.Hour
	case Period1MO:
		return 30 * 24 * time.Hour
	default:
		return 15 * time.Minute
	}
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return defaultRunLimit
	}
	if limit > maxRunLimit {
		return maxRunLimit
	}
	return limit
}

func normalizeOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	return offset
}

func renderArtifactName(group FileGroup, object ScenarioObject, windowStart, windowEnd time.Time, fileID int) (string, string) {
	tokens := runtimeTokenValues(group, object, windowStart, windowEnd, fileID)
	path, _ := renderTemplate(group.PathTemplate, tokens)
	fileName, _ := renderTemplate(group.FileNameTemplate, tokens)
	fileName = ensurePreviewFileExtension(fileName, group.Format)
	artifactName := fileName
	if group.CompressionEnabled {
		artifactName = fmt.Sprintf("%s.%s", artifactName, compressionFormatOrDefault(group.CompressionFormat))
	}
	return path, artifactName
}

func runtimeTokenValues(group FileGroup, object ScenarioObject, windowStart, windowEnd time.Time, fileID int) map[string]string {
	objectCode := strings.TrimSpace(object.Code)
	if objectCode == "" {
		objectCode = previewObjectCode(group)
	}
	return map[string]string{
		"#FTPRoot#":         "northupload",
		"#Province#":        "GD",
		"#OMC-R#":           "BaiOMC",
		"#DateTime#":        windowEnd.Format("20060102150405"),
		"#PeriodStartTime#": windowStart.Format("20060102150405"),
		"#PeriodEndTime#":   windowEnd.Format("20060102150405"),
		"#LocalHost#":       "127.0.0.1",
		"#DataVersion#":     "1.0",
		"#DataPeriod#":      previewDataPeriod(group.Period),
		"#Object#":          objectCode,
		"#ModuleType#":      objectCode,
		"#eNBID#":           "100001",
		"#Ri#":              "1",
		"#FileID#":          fmt.Sprintf("%04d", fileID),
	}
}

func enrichRows(rows []ExportDataRow, object ScenarioObject, windowStart, windowEnd time.Time) []ExportDataRow {
	out := make([]ExportDataRow, 0, len(rows))
	for _, row := range rows {
		next := make(ExportDataRow, len(row)+10)
		for key, value := range row {
			next[key] = value
		}
		next["system.omc_r"] = valueOrDefault(next["system.omc_r"], "BaiOMC")
		next["system.province"] = valueOrDefault(next["system.province"], "GD")
		next["runtime.data_version"] = valueOrDefault(next["runtime.data_version"], "1.0")
		next["runtime.local_host"] = valueOrDefault(next["runtime.local_host"], "127.0.0.1")
		next["runtime.object"] = valueOrDefault(next["runtime.object"], object.Code)
		next["runtime.tech"] = valueOrDefault(next["runtime.tech"], object.Tech)
		next["task.window_start"] = valueOrDefault(next["task.window_start"], windowStart.Format(time.RFC3339))
		next["task.window_end"] = valueOrDefault(next["task.window_end"], windowEnd.Format(time.RFC3339))
		out = append(out, next)
	}
	// No placeholder row when there is no source data: a northbound file must reflect
	// real device/PM/MR data only. Empty input yields zero rows, and generateFileContent
	// then produces no artifact (no fabrication, no delivery of an empty file).
	return out
}

func valueOrDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func metricPathsFromFields(fields []FieldDefinition) []string {
	out := make([]string, 0, len(fields))
	seen := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		if field.Source != "pm_metric_dictionary.metric_path" || field.SystemField == "" {
			continue
		}
		if _, ok := seen[field.SystemField]; ok {
			continue
		}
		seen[field.SystemField] = struct{}{}
		out = append(out, field.SystemField)
	}
	return out
}

func failedFileRun(profileKind ProfileKind, profileCode string, group FileGroup, object ScenarioObject, req RunProfileRequest, err error) FileRun {
	windowStart, windowEnd := normalizeRunWindow(group.Period, req)
	artifactPath, artifactName := renderArtifactName(group, object, windowStart, windowEnd, 1)
	return FileRun{
		ProfileKind:        profileKind,
		ProfileCode:        profileCode,
		GroupID:            group.ID,
		Domain:             group.Domain,
		ObjectCode:         object.Code,
		Status:             RunStatusFailed,
		WindowStart:        &windowStart,
		WindowEnd:          &windowEnd,
		ArtifactPath:       artifactPath,
		ArtifactName:       artifactName,
		CompressionEnabled: group.CompressionEnabled,
		CompressionFormat:  compressionFormatOrDefault(group.CompressionFormat),
		ErrorMessage:       err.Error(),
		Summary: map[string]any{
			"format": group.Format,
			"period": group.Period,
		},
	}
}

func failedInventoryRun(profile InventoryProfile, req RunProfileRequest, err error) FileRun {
	group := FileGroup{
		ID:                 profile.Code,
		Domain:             DomainInventory,
		Format:             FormatCSV,
		Period:             profile.Period,
		PathTemplate:       profile.PathTemplate,
		FileNameTemplate:   profile.FileNameTemplate,
		CompressionEnabled: profile.CompressionEnabled,
		CompressionFormat:  profile.CompressionFormat,
	}
	return failedFileRun(ProfileKindInventory, profile.Code, group, ScenarioObject{Code: profile.ObjectCode, Tech: profile.Tech}, req, err)
}

func xmlTag(value string) string {
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	tag := strings.Trim(b.String(), "_")
	if tag == "" {
		return "Field"
	}
	if tag[0] >= '0' && tag[0] <= '9' {
		return "F_" + tag
	}
	return tag
}

func escapeXML(value string) string {
	value = strings.ReplaceAll(value, "&", "&amp;")
	value = strings.ReplaceAll(value, "<", "&lt;")
	value = strings.ReplaceAll(value, ">", "&gt;")
	value = strings.ReplaceAll(value, `"`, "&quot;")
	value = strings.ReplaceAll(value, "'", "&apos;")
	return value
}

// renderCMXML renders CM data in the Baicells northbound DataFile XML format
// (column-oriented): <DataFile><FileHeader/><Objects><FieldName><N i="n">alias</N>...
// </FieldName><FieldValue><Object Dn="serial"><V i="n">value</V>...</Object>...
// </FieldValue></Objects></DataFile>. Matches the sample files under doc/20260805/cm/.
func renderCMXML(fields []FieldDefinition, rows []ExportDataRow, ts time.Time) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString("<DataFile>\n")
	b.WriteString("<FileHeader>\n")
	b.WriteString("    <TimeStamp>" + ts.Format("2006-01-02T15:04:05") + "</TimeStamp>\n")
	b.WriteString("    <TimeZone>UTC+8</TimeZone>\n")
	b.WriteString("</FileHeader>")
	b.WriteString("<Objects>\n")
	b.WriteString("<FieldName>\n")
	for i, f := range fields {
		fmt.Fprintf(&b, "    <N i=\"%d\">%s</N>\n", i+1, escapeXML(f.OutputAlias))
	}
	b.WriteString("</FieldName>")
	b.WriteString("<FieldValue>\n")
	for _, row := range rows {
		fmt.Fprintf(&b, "<Object Dn=\"%s\">\n", escapeXML(row["device.serial_number"]))
		for i, f := range fields {
			fmt.Fprintf(&b, "    <V i=\"%d\">%s</V>\n", i+1, escapeXML(row[f.SystemField]))
		}
		b.WriteString("</Object>\n")
	}
	b.WriteString("</FieldValue>\n")
	b.WriteString("</Objects>\n")
	b.WriteString("</DataFile>\n")
	return b.String()
}
