package parammodel

import (
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuiltinParamMappingTypesMatchDeviceReferences(t *testing.T) {
	baseDir := filepath.Join("..", "..", "..", "data", "param-mappings")
	deliveryDir := filepath.Join("..", "..", "..", "docs", "param-model-delivery", "xml", "tr069-param-mapping")

	modelReferences := map[string]string{
		"BM":     "BM全量参数集.csv",
		"BSC":    "GSM全量参数集.csv",
		"BTS":    "GSM全量参数集.csv",
		"BLN":    "LTE全量参数集.csv",
		"BLQ":    "LTE全量参数集.csv",
		"MLN":    "LTE全量参数集.csv",
		"MLQ":    "LTE全量参数集.csv",
		"BaiBNQ": "NR全量参数集.csv",
	}

	for model, referenceFile := range modelReferences {
		model := model
		referenceFile := referenceFile
		t.Run(model, func(t *testing.T) {
			referenceTypes := loadReferenceTypes(t, filepath.Join(baseDir, "references", referenceFile))
			for _, dir := range []string{baseDir, deliveryDir} {
				xmlPath := filepath.Join(dir, model+".xml")
				body, err := os.ReadFile(xmlPath)
				require.NoError(t, err)

				var doc xmlParameterModel
				require.NoError(t, xml.Unmarshal(body, &doc))

				matched := 0
				mismatches := make([]string, 0)
				for _, param := range doc.Params {
					want, ok := referenceTypes[normalizeReferencePath(param.Name)]
					if !ok {
						continue
					}
					matched++
					if got := strings.ToUpper(strings.TrimSpace(param.DataType)); got != want {
						mismatches = append(mismatches, fmt.Sprintf("%s: got %s, want %s", param.Name, got, want))
					}
				}

				require.NotZero(t, matched, "%s has no paths matched by %s", xmlPath, referenceFile)
				if len(mismatches) > 10 {
					mismatches = append(mismatches[:10], fmt.Sprintf("... and %d more", len(mismatches)-10))
				}
				require.Empty(t, mismatches, "%s types differ from %s", xmlPath, referenceFile)
			}
		})
	}
}

func loadReferenceTypes(t *testing.T, path string) map[string]string {
	t.Helper()
	file, err := os.Open(path)
	require.NoError(t, err)
	defer file.Close()

	reader := csv.NewReader(file)
	reader.ReuseRecord = true
	header, err := reader.Read()
	require.NoError(t, err)
	pathColumn := csvColumnIndex(t, header, "trpath.name")
	typeColumn := csvColumnIndex(t, header, "trpath.datatype")

	result := make(map[string]string)
	for {
		record, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		require.NoError(t, readErr)
		if pathColumn >= len(record) || typeColumn >= len(record) {
			continue
		}

		privatePath := strings.TrimSpace(record[pathColumn])
		dataType := strings.ToUpper(strings.TrimSpace(record[typeColumn]))
		if privatePath == "" || privatePath == "--" || dataType == "" || dataType == "--" {
			continue
		}

		key := normalizeReferencePath(privatePath)
		if existing, ok := result[key]; ok {
			require.Equal(t, existing, dataType, "reference has conflicting types for %s", key)
			continue
		}
		result[key] = dataType
	}
	return result
}

func csvColumnIndex(t *testing.T, header []string, name string) int {
	t.Helper()
	for i, column := range header {
		if strings.TrimPrefix(strings.TrimSpace(column), "\ufeff") == name {
			return i
		}
	}
	t.Fatalf("CSV column %q not found", name)
	return -1
}

func normalizeReferencePath(path string) string {
	parts := strings.Split(strings.TrimSpace(path), ".")
	for i, part := range parts {
		if isDecimalPathSegment(part) {
			parts[i] = "{i}"
		}
	}
	return strings.Join(parts, ".")
}

func isDecimalPathSegment(segment string) bool {
	if segment == "" {
		return false
	}
	for _, r := range segment {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
