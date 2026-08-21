package deviceaccess

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

const (
	defaultImportMaxBytes = 10 << 20
	defaultImportMaxRows  = 10000
)

var (
	ErrImportFileTooLarge                = errors.New("import_file_too_large")
	ErrImportTemplateInvalid             = errors.New("import_template_invalid")
	ErrImportBatchConflict               = errors.New("import_batch_state_conflict")
	ErrImportReplaceConfirmationRequired = errors.New("import_replace_confirmation_required")
)

var accessListImportHeaders = []string{
	"Serial Number", "List Type", "Reason", "Valid From", "Valid Until",
}

type AccessListImportOptions struct {
	EntryType     ListEntryType
	FailurePolicy ImportFailurePolicy
	MaxBytes      int
	MaxRows       int
}

type AccessListImportPreview struct {
	Rows         []ImportRow
	Entries      []CompiledListEntry
	TotalCount   int
	ValidCount   int
	InvalidCount int
}

func ParseAccessListCSV(data []byte, options AccessListImportOptions) (AccessListImportPreview, error) {
	return parseAccessListRecords(data, "access-list.csv", options)
}

func ParseAccessListFile(data []byte, filename string, options AccessListImportOptions) (AccessListImportPreview, error) {
	return parseAccessListRecords(data, filename, options)
}

func parseAccessListRecords(data []byte, filename string, options AccessListImportOptions) (AccessListImportPreview, error) {
	maxBytes := options.MaxBytes
	if maxBytes <= 0 {
		maxBytes = defaultImportMaxBytes
	}
	maxRows := options.MaxRows
	if maxRows <= 0 {
		maxRows = defaultImportMaxRows
	}
	if len(data) > maxBytes {
		return AccessListImportPreview{}, ErrImportFileTooLarge
	}
	if err := validateImportEntryType(options.EntryType); err != nil {
		return AccessListImportPreview{}, err
	}
	records, err := readImportRecords(data, filename)
	if err != nil {
		return AccessListImportPreview{}, err
	}
	if len(records) == 0 {
		return AccessListImportPreview{}, fmt.Errorf("%w: missing header", ErrImportTemplateInvalid)
	}
	if !equalImportStrings(records[0], accessListImportHeaders) {
		return AccessListImportPreview{}, fmt.Errorf("%w: unexpected header", ErrImportTemplateInvalid)
	}

	preview := AccessListImportPreview{Rows: make([]ImportRow, 0), Entries: make([]CompiledListEntry, 0)}
	seen := make(map[string]struct{})
	for offset, record := range records[1:] {
		rowNumber := offset + 2
		if emptyImportRecord(record) {
			continue
		}
		if len(record) > len(accessListImportHeaders) {
			return AccessListImportPreview{}, fmt.Errorf("%w: row %d has %d columns", ErrImportTemplateInvalid, rowNumber, len(record))
		}
		if len(record) < len(accessListImportHeaders) {
			record = append(record, make([]string, len(accessListImportHeaders)-len(record))...)
		}
		preview.TotalCount++
		if preview.TotalCount > maxRows {
			return AccessListImportPreview{}, ErrImportFileTooLarge
		}
		row := ImportRow{RowNumber: rowNumber, RawValue: rawAccessListRow(record)}
		entry, normalizeErr := normalizeImportedAccessListRow(record, options.EntryType)
		if normalizeErr != nil {
			row.ValidationStatus = ImportRowInvalid
			row.ErrorCode = importRowErrorCode(normalizeErr)
			row.ErrorMessage = normalizeErr.Error()
			preview.InvalidCount++
			preview.Rows = append(preview.Rows, row)
			continue
		}
		row.NormalizedValue = normalizedAccessListRow(entry)
		if _, duplicate := seen[entry.IdentityValue]; duplicate {
			row.ValidationStatus = ImportRowDuplicate
			row.ErrorCode = "import_duplicate_serial"
			row.ErrorMessage = "duplicate serial number in file"
			preview.InvalidCount++
			preview.Rows = append(preview.Rows, row)
			continue
		}
		seen[entry.IdentityValue] = struct{}{}
		row.ValidationStatus = ImportRowValid
		preview.ValidCount++
		preview.Rows = append(preview.Rows, row)
		preview.Entries = append(preview.Entries, entry)
	}
	return preview, nil
}

func GenerateAccessListXLSXTemplate() ([]byte, error) {
	return generateImportXLSX(accessListImportHeaders, nil)
}

func validateImportEntryType(entryType ListEntryType) error {
	switch entryType {
	case ListEntryTypeDeny, ListEntryTypeAllow, ListEntryTypeRevoked:
		return nil
	default:
		return fmt.Errorf("%w: unsupported list type %q", ErrImportTemplateInvalid, entryType)
	}
}

func normalizeImportedAccessListRow(record []string, entryType ListEntryType) (CompiledListEntry, error) {
	serialNumber := strings.TrimSpace(record[0])
	if serialNumber == "" || len(serialNumber) > 256 {
		return CompiledListEntry{}, errors.New("serial number is required or too long")
	}
	if ListEntryType(strings.ToLower(strings.TrimSpace(record[1]))) != entryType {
		return CompiledListEntry{}, errors.New("list type does not match selected type")
	}
	validFrom, err := parseImportTime(record[3])
	if err != nil {
		return CompiledListEntry{}, fmt.Errorf("invalid valid-from: %w", err)
	}
	validUntil, err := parseImportTime(record[4])
	if err != nil {
		return CompiledListEntry{}, fmt.Errorf("invalid valid-until: %w", err)
	}
	if validFrom != nil && validUntil != nil && !validUntil.After(*validFrom) {
		return CompiledListEntry{}, errors.New("valid-until must be after valid-from")
	}
	return CompiledListEntry{
		Type: entryType, IdentityType: IdentityTypeSerialNumber, IdentityValue: serialNumber,
		Reason: strings.TrimSpace(record[2]), ValidFrom: validFrom, ValidUntil: validUntil,
		Status: ListEntryStatusActive,
	}, nil
}

func parseImportTime(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		utc := parsed.UTC()
		return &utc, nil
	}
	if serial, err := strconv.ParseFloat(value, 64); err == nil && serial > 0 {
		parsed, err := excelize.ExcelDateToTime(serial, false)
		if err == nil {
			utc := parsed.UTC()
			return &utc, nil
		}
	}
	parsed, err := time.ParseInLocation("2006-01-02", value, time.Local)
	if err != nil {
		return nil, err
	}
	utc := parsed.UTC()
	return &utc, nil
}

func rawAccessListRow(record []string) map[string]any {
	values := make(map[string]any, len(accessListImportHeaders))
	for index, header := range accessListImportHeaders {
		values[header] = record[index]
	}
	return values
}

func normalizedAccessListRow(entry CompiledListEntry) map[string]any {
	values := map[string]any{
		"serial_number": entry.IdentityValue,
		"entry_type":    entry.Type,
		"reason":        entry.Reason,
	}
	if entry.ValidFrom != nil {
		values["valid_from"] = entry.ValidFrom.UTC().Format(time.RFC3339)
	}
	if entry.ValidUntil != nil {
		values["valid_until"] = entry.ValidUntil.UTC().Format(time.RFC3339)
	}
	return values
}

func importRowErrorCode(err error) string {
	message := err.Error()
	switch {
	case strings.Contains(message, "serial number"):
		return "import_invalid_serial"
	case strings.Contains(message, "list type"):
		return "import_list_type_mismatch"
	case strings.Contains(message, "valid-from"), strings.Contains(message, "valid-until"):
		return "import_invalid_time_range"
	default:
		return "import_invalid_row"
	}
}

func equalImportStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if strings.TrimSpace(left[index]) != right[index] {
			return false
		}
	}
	return true
}
