package deviceaccess

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseAccessListCSVSupportsBOMAndStableColumns(t *testing.T) {
	preview, err := ParseAccessListCSV([]byte("\ufeffSerial Number,List Type,Reason,Valid From,Valid Until\nSN-1,deny,expired,2026-08-19,2026-08-20T12:00:00+08:00\n"), AccessListImportOptions{EntryType: ListEntryTypeDeny})

	require.NoError(t, err)
	require.Equal(t, 1, preview.TotalCount)
	require.Equal(t, 1, preview.ValidCount)
	require.Zero(t, preview.InvalidCount)
	require.Equal(t, ImportRowValid, preview.Rows[0].ValidationStatus)
	require.Equal(t, "SN-1", preview.Entries[0].IdentityValue)
}

func TestParseAccessListCSVReportsEveryInvalidRow(t *testing.T) {
	data := "Serial Number,List Type,Reason,Valid From,Valid Until\nSN-1,deny,,,\nSN-1,deny,,,\nSN-2,allow,,,\nSN-3,deny,,2026-08-20,2026-08-19\n"
	preview, err := ParseAccessListCSV([]byte(data), AccessListImportOptions{EntryType: ListEntryTypeDeny})

	require.NoError(t, err)
	require.Equal(t, 4, preview.TotalCount)
	require.Equal(t, 1, preview.ValidCount)
	require.Equal(t, 3, preview.InvalidCount)
	require.Equal(t, ImportRowDuplicate, preview.Rows[1].ValidationStatus)
	require.Equal(t, "import_list_type_mismatch", preview.Rows[2].ErrorCode)
	require.Equal(t, "import_invalid_time_range", preview.Rows[3].ErrorCode)
}

func TestParseAccessListCSVRejectsTemplateAndResourceLimits(t *testing.T) {
	_, err := ParseAccessListCSV([]byte("Serial Number,List Type\nSN-1,deny\n"), AccessListImportOptions{EntryType: ListEntryTypeDeny})
	require.ErrorIs(t, err, ErrImportTemplateInvalid)

	_, err = ParseAccessListCSV([]byte("Serial Number,List Type,Reason,Valid From,Valid Until\nSN-1,deny,,,\n"), AccessListImportOptions{EntryType: ListEntryTypeDeny, MaxBytes: 10})
	require.True(t, errors.Is(err, ErrImportFileTooLarge))

	data := "Serial Number,List Type,Reason,Valid From,Valid Until\n" + strings.Repeat("SN-1,deny,,,\n", 2)
	_, err = ParseAccessListCSV([]byte(data), AccessListImportOptions{EntryType: ListEntryTypeDeny, MaxRows: 1})
	require.ErrorIs(t, err, ErrImportFileTooLarge)
}

func TestParseAccessListFileSupportsXLSXAndExcelDates(t *testing.T) {
	content, err := generateImportXLSX(accessListImportHeaders, [][]string{
		{"SN-XLSX-1", "deny", "legacy workbook", "46252", "46253"},
	})
	require.NoError(t, err)

	preview, err := ParseAccessListFile(content, "deny.xlsx", AccessListImportOptions{EntryType: ListEntryTypeDeny})

	require.NoError(t, err)
	require.Equal(t, 1, preview.ValidCount)
	require.Equal(t, "SN-XLSX-1", preview.Entries[0].IdentityValue)
	require.NotNil(t, preview.Entries[0].ValidFrom)
	require.NotNil(t, preview.Entries[0].ValidUntil)
}

func TestParseAccessListFileAllowsBlankTrailingOptionalXLSXCells(t *testing.T) {
	content, err := generateImportXLSX(accessListImportHeaders, [][]string{
		{"SN-XLSX-2", "allow", "approved"},
	})
	require.NoError(t, err)

	preview, err := ParseAccessListFile(content, "allow.xlsx", AccessListImportOptions{EntryType: ListEntryTypeAllow})

	require.NoError(t, err)
	require.Equal(t, 1, preview.ValidCount)
	require.Nil(t, preview.Entries[0].ValidFrom)
	require.Nil(t, preview.Entries[0].ValidUntil)
}

func TestGenerateAccessListXLSXTemplateHasStableHeader(t *testing.T) {
	content, err := GenerateAccessListXLSXTemplate()
	require.NoError(t, err)

	records, err := readImportRecords(content, "template.xlsx")
	require.NoError(t, err)
	require.Equal(t, [][]string{accessListImportHeaders}, records)
}
