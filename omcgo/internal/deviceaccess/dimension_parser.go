package deviceaccess

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
)

var ecgiPattern = regexp.MustCompile(`^[1-9][0-9]{4,5}:([0-9]+)$`)

func RuleDimensionHeaders(dimension ImportDimension) ([]string, error) {
	switch dimension {
	case ImportDimensionSN:
		return []string{"Serial Number"}, nil
	case ImportDimensionTAC:
		return []string{"TAC"}, nil
	case ImportDimensionECGI:
		return []string{"ECGI"}, nil
	case ImportDimensionIP:
		return []string{"Start IP", "End IP"}, nil
	case ImportDimensionGPS:
		return []string{"Longitude Range", "Latitude Range"}, nil
	default:
		return nil, fmt.Errorf("%w: unsupported rule dimension %q", ErrImportTemplateInvalid, dimension)
	}
}

func GenerateRuleDimensionTemplate(dimension ImportDimension) ([]byte, error) {
	return GenerateRuleDimensionCSV(dimension, nil)
}

func GenerateRuleDimensionXLSXTemplate(dimension ImportDimension) ([]byte, error) {
	headers, err := RuleDimensionHeaders(dimension)
	if err != nil {
		return nil, err
	}
	return generateImportXLSX(headers, nil)
}

func GenerateRuleDimensionCSV(dimension ImportDimension, values []RuleDimensionValue) ([]byte, error) {
	headers, err := RuleDimensionHeaders(dimension)
	if err != nil {
		return nil, err
	}
	var output bytes.Buffer
	output.Write([]byte("\xef\xbb\xbf"))
	writer := csv.NewWriter(&output)
	writer.UseCRLF = true
	if err := writer.Write(headers); err != nil {
		return nil, fmt.Errorf("write rule dimension CSV template: %w", err)
	}
	for _, value := range values {
		var record []string
		switch dimension {
		case ImportDimensionSN:
			record = []string{value.SerialNumber}
		case ImportDimensionTAC, ImportDimensionECGI:
			record = []string{value.Value}
		case ImportDimensionIP:
			if value.IPRange == nil {
				return nil, errors.New("rule dimension export contains an empty IP range")
			}
			record = []string{value.IPRange.Start, value.IPRange.End}
		case ImportDimensionGPS:
			if value.GeoBounds == nil {
				return nil, errors.New("rule dimension export contains empty GPS bounds")
			}
			record = []string{
				fmt.Sprintf("%g~%g", value.GeoBounds.MinLongitude, value.GeoBounds.MaxLongitude),
				fmt.Sprintf("%g~%g", value.GeoBounds.MinLatitude, value.GeoBounds.MaxLatitude),
			}
		}
		for index := range record {
			record[index] = sanitizeSpreadsheetCSVCell(record[index])
		}
		if err := writer.Write(record); err != nil {
			return nil, fmt.Errorf("write rule dimension CSV row: %w", err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("flush rule dimension CSV template: %w", err)
	}
	return output.Bytes(), nil
}

func ParseRuleDimensionFile(data []byte, filename string, dimension ImportDimension) (RuleDimensionPreview, error) {
	if len(data) > defaultImportMaxBytes {
		return RuleDimensionPreview{}, ErrImportFileTooLarge
	}
	headers, err := RuleDimensionHeaders(dimension)
	if err != nil {
		return RuleDimensionPreview{}, err
	}
	records, err := readRuleDimensionRecords(data, filename)
	if err != nil {
		return RuleDimensionPreview{}, err
	}
	if len(records) == 0 || !equalImportStrings(records[0], headers) {
		return RuleDimensionPreview{}, fmt.Errorf("%w: unexpected header", ErrImportTemplateInvalid)
	}
	preview := RuleDimensionPreview{}
	seen := make(map[string]struct{})
	for offset, record := range records[1:] {
		if emptyImportRecord(record) {
			continue
		}
		if len(record) > len(headers) {
			return RuleDimensionPreview{}, fmt.Errorf("%w: row %d has %d columns", ErrImportTemplateInvalid, offset+2, len(record))
		}
		preview.TotalCount++
		if preview.TotalCount > defaultImportMaxRows {
			return RuleDimensionPreview{}, ErrImportFileTooLarge
		}
		row := ImportRow{RowNumber: offset + 2, RawValue: rawDimensionRow(headers, record)}
		value, key, normalizeErr := normalizeRuleDimensionRow(record, dimension)
		if normalizeErr != nil {
			row.ValidationStatus = ImportRowInvalid
			row.ErrorCode = dimensionErrorCode(normalizeErr)
			row.ErrorMessage = normalizeErr.Error()
			preview.InvalidCount++
			preview.Rows = append(preview.Rows, row)
			continue
		}
		row.NormalizedValue = normalizedDimensionRow(value)
		if _, duplicate := seen[key]; duplicate {
			row.ValidationStatus = ImportRowDuplicate
			row.ErrorCode = "import_duplicate_dimension_value"
			row.ErrorMessage = "duplicate rule dimension value in file"
			preview.InvalidCount++
			preview.Rows = append(preview.Rows, row)
			continue
		}
		seen[key] = struct{}{}
		row.ValidationStatus = ImportRowValid
		preview.ValidCount++
		preview.Rows = append(preview.Rows, row)
		preview.Values = append(preview.Values, value)
	}
	return preview, nil
}

func readRuleDimensionRecords(data []byte, filename string) ([][]string, error) {
	return readImportRecords(data, filename)
}

func normalizeRuleDimensionRow(record []string, dimension ImportDimension) (RuleDimensionValue, string, error) {
	valueAt := func(index int) string {
		if index >= len(record) {
			return ""
		}
		return restoreSpreadsheetCSVCell(strings.TrimSpace(record[index]))
	}
	value := RuleDimensionValue{Dimension: dimension}
	switch dimension {
	case ImportDimensionSN:
		value.SerialNumber = valueAt(0)
		if value.SerialNumber == "" || len(value.SerialNumber) > 256 {
			return value, "", errors.New("serial number is required or too long")
		}
		return value, value.SerialNumber, nil
	case ImportDimensionTAC:
		number, err := strconv.ParseUint(valueAt(0), 10, 16)
		if err != nil {
			return value, "", errors.New("TAC must be an integer from 0 to 65535")
		}
		value.Value = strconv.FormatUint(number, 10)
		return value, value.Value, nil
	case ImportDimensionECGI:
		matches := ecgiPattern.FindStringSubmatch(valueAt(0))
		if len(matches) != 2 {
			return value, "", errors.New("ECGI must use PLMN:ECI format")
		}
		eci, err := strconv.ParseUint(matches[1], 10, 28)
		if err != nil || eci > 268435455 {
			return value, "", errors.New("ECGI ECI must be from 0 to 268435455")
		}
		value.Value = valueAt(0)
		return value, value.Value, nil
	case ImportDimensionIP:
		start, startBits := comparableIP(valueAt(0))
		end, endBits := comparableIP(valueAt(1))
		if start == nil || end == nil || startBits != endBits || bytes.Compare(start, end) > 0 {
			return value, "", errors.New("IP range must contain ordered addresses from the same address family")
		}
		value.IPRange = &IPRange{Start: canonicalIP(valueAt(0)), End: canonicalIP(valueAt(1))}
		return value, value.IPRange.Start + "\x00" + value.IPRange.End, nil
	case ImportDimensionGPS:
		minLongitude, maxLongitude, err := parseCoordinateRange(valueAt(0), -180, 180)
		if err != nil {
			return value, "", fmt.Errorf("invalid longitude range: %w", err)
		}
		minLatitude, maxLatitude, err := parseCoordinateRange(valueAt(1), -90, 90)
		if err != nil {
			return value, "", fmt.Errorf("invalid latitude range: %w", err)
		}
		value.GeoBounds = &GeoBounds{MinLongitude: minLongitude, MaxLongitude: maxLongitude, MinLatitude: minLatitude, MaxLatitude: maxLatitude}
		key := strconv.FormatFloat(minLongitude, 'g', -1, 64) + "~" + strconv.FormatFloat(maxLongitude, 'g', -1, 64) + ";" + strconv.FormatFloat(minLatitude, 'g', -1, 64) + "~" + strconv.FormatFloat(maxLatitude, 'g', -1, 64)
		return value, key, nil
	default:
		return value, "", fmt.Errorf("unsupported rule dimension %q", dimension)
	}
}

// Spreadsheet applications may interpret cells beginning with these
// characters as formulas even when they originate from a CSV file. Prefixing
// the value with an apostrophe makes the exported cell literal. The import
// parser removes only the exact escape that this exporter adds, preserving a
// safe export -> import round trip.
func sanitizeSpreadsheetCSVCell(value string) string {
	trimmed := strings.TrimLeft(value, " \t\r\n")
	if trimmed == "" {
		return value
	}
	switch trimmed[0] {
	case '=', '+', '-', '@':
		return "'" + value
	default:
		return value
	}
}

func restoreSpreadsheetCSVCell(value string) string {
	if len(value) < 2 || value[0] != '\'' {
		return value
	}
	trimmed := strings.TrimLeft(value[1:], " \t\r\n")
	if trimmed == "" {
		return value
	}
	switch trimmed[0] {
	case '=', '+', '-', '@':
		return value[1:]
	default:
		return value
	}
}

func parseCoordinateRange(value string, minimum, maximum float64) (float64, float64, error) {
	parts := strings.Split(value, "~")
	if len(parts) != 2 {
		return 0, 0, errors.New("range must use min~max format")
	}
	start, startErr := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	end, endErr := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if startErr != nil || endErr != nil || start < minimum || end > maximum || start > end {
		return 0, 0, errors.New("range is outside bounds or reversed")
	}
	return start, end, nil
}

func canonicalIP(value string) string { return net.ParseIP(value).String() }

func rawDimensionRow(headers, record []string) map[string]any {
	result := make(map[string]any, len(headers))
	for index, header := range headers {
		if index < len(record) {
			result[header] = record[index]
		} else {
			result[header] = ""
		}
	}
	return result
}

func normalizedDimensionRow(value RuleDimensionValue) map[string]any {
	result := map[string]any{"dimension": value.Dimension}
	if value.SerialNumber != "" {
		result["serial_number"] = value.SerialNumber
	}
	if value.Value != "" {
		result["value"] = value.Value
	}
	if value.IPRange != nil {
		result["start_ip"], result["end_ip"] = value.IPRange.Start, value.IPRange.End
	}
	if value.GeoBounds != nil {
		result["min_longitude"], result["max_longitude"] = value.GeoBounds.MinLongitude, value.GeoBounds.MaxLongitude
		result["min_latitude"], result["max_latitude"] = value.GeoBounds.MinLatitude, value.GeoBounds.MaxLatitude
	}
	return result
}

func dimensionErrorCode(err error) string {
	switch {
	case strings.Contains(err.Error(), "serial"):
		return "import_invalid_serial"
	case strings.Contains(err.Error(), "TAC"):
		return "import_invalid_tac"
	case strings.Contains(err.Error(), "ECGI"):
		return "import_invalid_ecgi"
	case strings.Contains(err.Error(), "IP"):
		return "import_invalid_ip_range"
	case strings.Contains(err.Error(), "longitude"), strings.Contains(err.Error(), "latitude"):
		return "import_invalid_gps_bounds"
	default:
		return "import_invalid_row"
	}
}

func emptyImportRecord(record []string) bool {
	for _, value := range record {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
}

func ruleDimensionValueFromRow(row ImportRow, dimension ImportDimension) (RuleDimensionValue, error) {
	value := RuleDimensionValue{Dimension: dimension}
	switch dimension {
	case ImportDimensionSN:
		value.SerialNumber, _ = row.NormalizedValue["serial_number"].(string)
		if value.SerialNumber == "" {
			return value, errors.New("normalized serial number is missing")
		}
	case ImportDimensionTAC, ImportDimensionECGI:
		value.Value, _ = row.NormalizedValue["value"].(string)
		if value.Value == "" {
			return value, errors.New("normalized dimension value is missing")
		}
	case ImportDimensionIP:
		start, _ := row.NormalizedValue["start_ip"].(string)
		end, _ := row.NormalizedValue["end_ip"].(string)
		if start == "" || end == "" {
			return value, errors.New("normalized IP range is missing")
		}
		value.IPRange = &IPRange{Start: start, End: end}
	case ImportDimensionGPS:
		minLongitude, ok1 := importNumber(row.NormalizedValue["min_longitude"])
		maxLongitude, ok2 := importNumber(row.NormalizedValue["max_longitude"])
		minLatitude, ok3 := importNumber(row.NormalizedValue["min_latitude"])
		maxLatitude, ok4 := importNumber(row.NormalizedValue["max_latitude"])
		if !ok1 || !ok2 || !ok3 || !ok4 {
			return value, errors.New("normalized GPS bounds are missing")
		}
		value.GeoBounds = &GeoBounds{MinLongitude: minLongitude, MaxLongitude: maxLongitude, MinLatitude: minLatitude, MaxLatitude: maxLatitude}
	default:
		return value, fmt.Errorf("unsupported rule dimension %q", dimension)
	}
	return value, nil
}

func importNumber(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case json.Number:
		parsed, err := typed.Float64()
		return parsed, err == nil
	default:
		return 0, false
	}
}
