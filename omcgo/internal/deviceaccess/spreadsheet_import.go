package deviceaccess

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/xuri/excelize/v2"
)

const (
	importXLSXUnzipLimit    = 64 << 20
	importXLSXXMLUnzipLimit = 16 << 20
)

func readImportRecords(data []byte, filename string) ([][]string, error) {
	switch strings.ToLower(filepath.Ext(strings.TrimSpace(filename))) {
	case ".csv":
		return readCSVImportRecords(data)
	case ".xlsx":
		return readXLSXImportRecords(data)
	default:
		return nil, fmt.Errorf("%w: only .csv and .xlsx files are supported", ErrImportTemplateInvalid)
	}
}

func readCSVImportRecords(data []byte) ([][]string, error) {
	reader := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))))
	reader.FieldsPerRecord = -1
	records := make([][]string, 0)
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			return records, nil
		}
		if err != nil {
			return nil, fmt.Errorf("%w: read CSV: %v", ErrImportTemplateInvalid, err)
		}
		records = append(records, record)
		if len(records) > defaultImportMaxRows+1 {
			return nil, ErrImportFileTooLarge
		}
	}
}

func readXLSXImportRecords(data []byte) ([][]string, error) {
	book, err := excelize.OpenReader(bytes.NewReader(data), excelize.Options{
		RawCellValue: true, UnzipSizeLimit: importXLSXUnzipLimit, UnzipXMLSizeLimit: importXLSXXMLUnzipLimit,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: open XLSX: %v", ErrImportTemplateInvalid, err)
	}
	defer func() { _ = book.Close() }()
	sheets := book.GetSheetList()
	if len(sheets) != 1 {
		return nil, fmt.Errorf("%w: XLSX must contain exactly one worksheet", ErrImportTemplateInvalid)
	}
	rows, err := book.Rows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("%w: read XLSX worksheet: %v", ErrImportTemplateInvalid, err)
	}
	defer func() { _ = rows.Close() }()
	records := make([][]string, 0)
	for rows.Next() {
		record, err := rows.Columns()
		if err != nil {
			return nil, fmt.Errorf("%w: read XLSX row: %v", ErrImportTemplateInvalid, err)
		}
		records = append(records, record)
		if len(records) > defaultImportMaxRows+1 {
			return nil, ErrImportFileTooLarge
		}
	}
	if err := rows.Error(); err != nil {
		return nil, fmt.Errorf("%w: iterate XLSX rows: %v", ErrImportTemplateInvalid, err)
	}
	return records, nil
}

func generateImportXLSX(headers []string, rows [][]string) ([]byte, error) {
	book := excelize.NewFile()
	defer func() { _ = book.Close() }()
	const sheet = "Import"
	book.SetSheetName("Sheet1", sheet)
	headerStyle, err := book.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"D9EAF7"}, Pattern: 1},
	})
	if err != nil {
		return nil, fmt.Errorf("create import XLSX header style: %w", err)
	}
	for column, value := range headers {
		cell, err := excelize.CoordinatesToCellName(column+1, 1)
		if err != nil {
			return nil, fmt.Errorf("locate import XLSX header: %w", err)
		}
		if err := book.SetCellStr(sheet, cell, value); err != nil {
			return nil, fmt.Errorf("write import XLSX header: %w", err)
		}
	}
	end, _ := excelize.CoordinatesToCellName(len(headers), 1)
	if err := book.SetCellStyle(sheet, "A1", end, headerStyle); err != nil {
		return nil, fmt.Errorf("style import XLSX header: %w", err)
	}
	for rowIndex, row := range rows {
		for column, value := range row {
			cell, err := excelize.CoordinatesToCellName(column+1, rowIndex+2)
			if err != nil {
				return nil, fmt.Errorf("locate import XLSX value: %w", err)
			}
			if err := book.SetCellStr(sheet, cell, value); err != nil {
				return nil, fmt.Errorf("write import XLSX value: %w", err)
			}
		}
	}
	if err := book.SetColWidth(sheet, "A", end[:1], 24); err != nil {
		return nil, fmt.Errorf("size import XLSX columns: %w", err)
	}
	buffer, err := book.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("serialize import XLSX: %w", err)
	}
	return buffer.Bytes(), nil
}
