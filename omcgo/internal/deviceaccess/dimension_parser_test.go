package deviceaccess

import (
	"bytes"
	"testing"
)

func TestParseRuleDimensionFileAcceptsLegacyCSVTemplates(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		dimension ImportDimension
		content   string
		assert    func(*testing.T, RuleDimensionValue)
	}{
		{name: "sn", dimension: ImportDimensionSN, content: "Serial Number\r\n SN-1 \r\n", assert: func(t *testing.T, value RuleDimensionValue) {
			t.Helper()
			if value.SerialNumber != "SN-1" {
				t.Fatalf("serial = %q", value.SerialNumber)
			}
		}},
		{name: "tac", dimension: ImportDimensionTAC, content: "TAC\n00042\n", assert: func(t *testing.T, value RuleDimensionValue) {
			t.Helper()
			if value.Value != "42" {
				t.Fatalf("TAC = %q", value.Value)
			}
		}},
		{name: "ecgi", dimension: ImportDimensionECGI, content: "ECGI\n46001:268435455\n", assert: func(t *testing.T, value RuleDimensionValue) {
			t.Helper()
			if value.Value != "46001:268435455" {
				t.Fatalf("ECGI = %q", value.Value)
			}
		}},
		{name: "ip", dimension: ImportDimensionIP, content: "Start IP,End IP\n192.0.2.3,192.0.2.17\n", assert: func(t *testing.T, value RuleDimensionValue) {
			t.Helper()
			if value.IPRange == nil || value.IPRange.Start != "192.0.2.3" || value.IPRange.End != "192.0.2.17" {
				t.Fatalf("range = %#v", value.IPRange)
			}
		}},
		{name: "gps", dimension: ImportDimensionGPS, content: "Longitude Range,Latitude Range\n104.01~104.09,30.1~30.9\n", assert: func(t *testing.T, value RuleDimensionValue) {
			t.Helper()
			if value.GeoBounds == nil || value.GeoBounds.MinLongitude != 104.01 || value.GeoBounds.MaxLatitude != 30.9 {
				t.Fatalf("bounds = %#v", value.GeoBounds)
			}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			preview, err := ParseRuleDimensionFile([]byte(test.content), "legacy.csv", test.dimension)
			if err != nil {
				t.Fatalf("ParseRuleDimensionFile: %v", err)
			}
			if preview.ValidCount != 1 || preview.InvalidCount != 0 || len(preview.Values) != 1 {
				t.Fatalf("preview = %#v", preview)
			}
			test.assert(t, preview.Values[0])
		})
	}
}

func TestRuleDimensionCSVTemplateIsDirectlyReimportable(t *testing.T) {
	t.Parallel()
	for _, dimension := range []ImportDimension{ImportDimensionSN, ImportDimensionTAC, ImportDimensionECGI, ImportDimensionIP, ImportDimensionGPS} {
		content, err := GenerateRuleDimensionTemplate(dimension)
		if err != nil {
			t.Fatalf("GenerateRuleDimensionTemplate(%s): %v", dimension, err)
		}
		headers, _ := RuleDimensionHeaders(dimension)
		rows, err := readRuleDimensionRecords(content, "template.csv")
		if err != nil || len(rows) != 1 || !equalImportStrings(rows[0], headers) {
			t.Fatalf("%s template rows = %#v, err = %v", dimension, rows, err)
		}
	}
}

func TestParseRuleDimensionFileReportsStableRowErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		dimension ImportDimension
		content   string
		code      string
	}{
		{ImportDimensionTAC, "TAC\n65536\n", "import_invalid_tac"},
		{ImportDimensionECGI, "ECGI\n46001:268435456\n", "import_invalid_ecgi"},
		{ImportDimensionIP, "Start IP,End IP\n192.0.2.20,192.0.2.1\n", "import_invalid_ip_range"},
		{ImportDimensionIP, "Start IP,End IP\n192.0.2.1,2001:db8::1\n", "import_invalid_ip_range"},
		{ImportDimensionGPS, "Longitude Range,Latitude Range\n170~-170,30~31\n", "import_invalid_gps_bounds"},
	}
	for _, test := range tests {
		preview, err := ParseRuleDimensionFile([]byte(test.content), "legacy.csv", test.dimension)
		if err != nil {
			t.Fatalf("ParseRuleDimensionFile(%s): %v", test.dimension, err)
		}
		if preview.InvalidCount != 1 || len(preview.Rows) != 1 || preview.Rows[0].ErrorCode != test.code {
			t.Fatalf("%s preview = %#v", test.dimension, preview)
		}
	}
}

func TestParseRuleDimensionFileRejectsDuplicateNormalizedValues(t *testing.T) {
	t.Parallel()
	preview, err := ParseRuleDimensionFile([]byte("TAC\n42\n00042\n"), "legacy.csv", ImportDimensionTAC)
	if err != nil {
		t.Fatalf("ParseRuleDimensionFile: %v", err)
	}
	if preview.ValidCount != 1 || preview.InvalidCount != 1 || preview.Rows[1].ValidationStatus != ImportRowDuplicate {
		t.Fatalf("preview = %#v", preview)
	}
}

func TestParseRuleDimensionFileSupportsXLSX(t *testing.T) {
	content, err := generateImportXLSX([]string{"Start IP", "End IP"}, [][]string{{"10.0.0.1", "10.0.0.10"}})
	if err != nil {
		t.Fatal(err)
	}

	preview, err := ParseRuleDimensionFile(content, "ip-ranges.xlsx", ImportDimensionIP)
	if err != nil {
		t.Fatal(err)
	}
	if preview.ValidCount != 1 || preview.Values[0].IPRange == nil {
		t.Fatalf("unexpected XLSX preview: %#v", preview)
	}
}

func TestRuleDimensionCSVNeutralizesSpreadsheetFormulaAndRoundTrips(t *testing.T) {
	t.Parallel()
	for _, serialNumber := range []string{"=1+1", "+cmd", "-2+3", "@SUM(A1:A2)"} {
		content, err := GenerateRuleDimensionCSV(ImportDimensionSN, []RuleDimensionValue{{
			Dimension: ImportDimensionSN, SerialNumber: serialNumber,
		}})
		if err != nil {
			t.Fatalf("GenerateRuleDimensionCSV(%q): %v", serialNumber, err)
		}
		if bytes.Contains(content, []byte("\r\n"+serialNumber+"\r\n")) {
			t.Fatalf("unsafe formula prefix was exported verbatim: %q", content)
		}
		preview, err := ParseRuleDimensionFile(content, "serials.csv", ImportDimensionSN)
		if err != nil {
			t.Fatalf("ParseRuleDimensionFile(%q): %v", serialNumber, err)
		}
		if len(preview.Values) != 1 || preview.Values[0].SerialNumber != serialNumber {
			t.Fatalf("serial number did not round trip: %#v", preview.Values)
		}
	}
}

func TestRuleDimensionCSVNegativeGPSBoundsRoundTrip(t *testing.T) {
	t.Parallel()
	want := &GeoBounds{
		MinLongitude: -180,
		MaxLongitude: 120.5,
		MinLatitude:  -45.25,
		MaxLatitude:  30,
	}
	content, err := GenerateRuleDimensionCSV(ImportDimensionGPS, []RuleDimensionValue{{
		Dimension: ImportDimensionGPS,
		GeoBounds: want,
	}})
	if err != nil {
		t.Fatalf("GenerateRuleDimensionCSV: %v", err)
	}
	preview, err := ParseRuleDimensionFile(content, "gps.csv", ImportDimensionGPS)
	if err != nil {
		t.Fatalf("ParseRuleDimensionFile: %v", err)
	}
	if len(preview.Values) != 1 || preview.Values[0].GeoBounds == nil || *preview.Values[0].GeoBounds != *want {
		t.Fatalf("GPS bounds did not round trip: %#v", preview.Values)
	}
}

func TestParseRuleDimensionFileRejectsUnexpectedColumns(t *testing.T) {
	t.Parallel()
	_, err := ParseRuleDimensionFile([]byte("TAC\n42,ignored\n"), "tac.csv", ImportDimensionTAC)
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte(ErrImportTemplateInvalid.Error())) {
		t.Fatalf("expected template error, got %v", err)
	}
}
