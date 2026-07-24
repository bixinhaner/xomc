package compare

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCompareLogicalRowsIndependentOfOrdering(t *testing.T) {
	t.Parallel()

	first := Row{Device: "device-a", Time: mustTime(t, "2026-07-24T00:00:00Z"), Counter: "Signal.RSRP", Value: value(-95)}
	second := Row{Device: "device-a", Time: mustTime(t, "2026-07-24T00:15:00Z"), Counter: "Traffic.Bytes", Value: value(42)}

	got := Compare([]Row{first, second}, []Row{second, first}, 1000, 200)

	if !got.LogicalEqual {
		t.Fatalf("LogicalEqual = false, mismatches = %#v", got.Mismatches)
	}
	if len(got.Mismatches) != 0 {
		t.Fatalf("Mismatches = %#v, want none", got.Mismatches)
	}
	if !got.Accepted {
		t.Fatalf("Accepted = false, want true")
	}
}

func TestCompareReportsExactMismatchKeys(t *testing.T) {
	t.Parallel()

	timeA := mustTime(t, "2026-07-24T00:00:00Z")
	timeB := mustTime(t, "2026-07-24T00:15:00Z")
	oldRows := []Row{
		{Device: "device-b", Time: timeB, Counter: "Traffic.Bytes", Value: value(7)},
		{Device: "device-a", Time: timeA, Counter: "Signal.RSRP", Value: value(-95)},
	}
	sparseRows := []Row{
		{Device: "device-a", Time: timeA, Counter: "Signal.RSRP", Value: value(-90)},
		{Device: "device-c", Time: timeB, Counter: "Users.Active", Value: value(3)},
	}

	got := Compare(oldRows, sparseRows, 1000, 100)

	if got.LogicalEqual {
		t.Fatal("LogicalEqual = true, want false")
	}
	want := []Key{
		{Device: "device-a", Time: timeA, Counter: "Signal.RSRP"},
		{Device: "device-b", Time: timeB, Counter: "Traffic.Bytes"},
		{Device: "device-c", Time: timeB, Counter: "Users.Active"},
	}
	if len(got.Mismatches) != len(want) {
		t.Fatalf("mismatch count = %d, want %d: %#v", len(got.Mismatches), len(want), got.Mismatches)
	}
	for i, key := range want {
		if got.Mismatches[i].Key != key {
			t.Errorf("mismatch[%d].Key = %#v, want %#v", i, got.Mismatches[i].Key, key)
		}
	}
	if len(got.Mismatches[0].OldRows) != 1 || len(got.Mismatches[0].SparseRows) != 1 {
		t.Fatalf("value mismatch rows = %#v, want one row on each side", got.Mismatches[0])
	}
	if len(got.Mismatches[1].OldRows) != 1 || len(got.Mismatches[1].SparseRows) != 0 {
		t.Fatalf("old-only mismatch rows = %#v", got.Mismatches[1])
	}
	if len(got.Mismatches[2].OldRows) != 0 || len(got.Mismatches[2].SparseRows) != 1 {
		t.Fatalf("sparse-only mismatch rows = %#v", got.Mismatches[2])
	}
}

func TestCompareDetectsDuplicateKeys(t *testing.T) {
	t.Parallel()

	row := Row{
		Device:  "device-a",
		Time:    mustTime(t, "2026-07-24T00:00:00Z"),
		Counter: "Signal.RSRP",
		Value:   value(-95),
	}

	got := Compare([]Row{row, row}, []Row{row}, 1000, 100)

	if got.LogicalEqual {
		t.Fatal("LogicalEqual = true with duplicate old key")
	}
	if len(got.Mismatches) != 1 {
		t.Fatalf("Mismatches = %#v, want one duplicate mismatch", got.Mismatches)
	}
	if got.Mismatches[0].Kind != MismatchDuplicate {
		t.Fatalf("Kind = %q, want %q", got.Mismatches[0].Kind, MismatchDuplicate)
	}
	if len(got.Mismatches[0].OldRows) != 2 || len(got.Mismatches[0].SparseRows) != 1 {
		t.Fatalf("duplicate mismatch rows = %#v", got.Mismatches[0])
	}
}

func TestCompareTreatsObjectAsPartOfLogicalKey(t *testing.T) {
	t.Parallel()

	timestamp := mustTime(t, "2026-07-24T00:00:00Z")
	cellA := Row{Device: "device-a", Object: "Cell=1", Time: timestamp, Counter: "Signal.RSRP", Value: value(-95)}
	cellB := Row{Device: "device-a", Object: "Cell=2", Time: timestamp, Counter: "Signal.RSRP", Value: value(-90)}

	got := Compare([]Row{cellA, cellB}, []Row{cellB, cellA}, 1000, 100)

	if !got.LogicalEqual {
		t.Fatalf("LogicalEqual = false for distinct objects: %#v", got.Mismatches)
	}
}

func TestCompareCalculatesPhysicalBytesAndRatio(t *testing.T) {
	t.Parallel()

	got := Compare(nil, nil, 4096, 512)

	if got.OldBytes != 4096 || got.SparseBytes != 512 {
		t.Fatalf("bytes = (%d, %d), want (4096, 512)", got.OldBytes, got.SparseBytes)
	}
	if got.SparseRatio != 0.125 {
		t.Fatalf("SparseRatio = %v, want 0.125", got.SparseRatio)
	}
	if !got.WithinSizeTarget || !got.Accepted {
		t.Fatalf("WithinSizeTarget = %v, Accepted = %v, want true/true", got.WithinSizeTarget, got.Accepted)
	}
}

func TestCompareHandlesEmptyAndZeroByteRuns(t *testing.T) {
	t.Parallel()

	t.Run("both representations empty", func(t *testing.T) {
		got := Compare(nil, nil, 0, 0)

		if !got.LogicalEqual || got.SparseRatio != 0 || !got.PhysicalValid || !got.WithinSizeTarget || !got.Accepted {
			t.Fatalf("empty result = %#v", got)
		}
	})

	t.Run("nonempty rows without a physical measurement", func(t *testing.T) {
		row := Row{
			Device:  "device-a",
			Object:  "Cell=1",
			Time:    mustTime(t, "2026-07-24T00:00:00Z"),
			Counter: "Signal.RSRP",
			Value:   value(-95),
		}
		got := Compare([]Row{row}, []Row{row}, 0, 0)

		if !got.LogicalEqual {
			t.Fatalf("LogicalEqual = false, mismatches = %#v", got.Mismatches)
		}
		if got.PhysicalValid || got.WithinSizeTarget || got.Accepted {
			t.Fatalf("nonempty zero-byte result = %#v, want explicit unmeasured failure", got)
		}
		if got.PhysicalReason == "" {
			t.Fatal("PhysicalReason is empty")
		}
	})

	t.Run("sparse bytes without old baseline", func(t *testing.T) {
		got := Compare(nil, nil, 0, 1)

		if !math.IsInf(got.SparseRatio, 1) {
			t.Fatalf("SparseRatio = %v, want +Inf", got.SparseRatio)
		}
		if !got.PhysicalValid || got.WithinSizeTarget || got.Accepted {
			t.Fatalf("PhysicalValid = %v, WithinSizeTarget = %v, Accepted = %v, want true/false/false",
				got.PhysicalValid, got.WithinSizeTarget, got.Accepted)
		}
	})
}

func TestCompareUsesExactTwentyPercentPhysicalThreshold(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		oldBytes    int64
		sparseBytes int64
		want        bool
	}{
		{name: "exact boundary", oldBytes: 1000, sparseBytes: 200, want: true},
		{name: "one byte above", oldBytes: 1000, sparseBytes: 201, want: false},
		{name: "fractional boundary below", oldBytes: 9, sparseBytes: 1, want: true},
		{name: "fractional boundary above", oldBytes: 9, sparseBytes: 2, want: false},
		{name: "max int64 below", oldBytes: math.MaxInt64, sparseBytes: math.MaxInt64 / 5, want: true},
		{name: "max int64 above", oldBytes: math.MaxInt64, sparseBytes: math.MaxInt64/5 + 1, want: false},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := Compare(nil, nil, tt.oldBytes, tt.sparseBytes)
			if !got.PhysicalValid {
				t.Fatalf("PhysicalValid = false, reason = %q", got.PhysicalReason)
			}
			if got.WithinSizeTarget != tt.want || got.Accepted != tt.want {
				t.Fatalf("result = %#v, want WithinSizeTarget/Accepted = %v", got, tt.want)
			}
			if got.PhysicalReason == "" {
				t.Fatal("PhysicalReason is empty")
			}
		})
	}
}

func TestCompareRejectsNegativePhysicalBytes(t *testing.T) {
	t.Parallel()

	for _, bytes := range [][2]int64{{-1, 0}, {0, -1}, {-1, -1}} {
		got := Compare(nil, nil, bytes[0], bytes[1])
		if got.PhysicalValid || got.WithinSizeTarget || got.Accepted {
			t.Fatalf("Compare(nil, nil, %d, %d) = %#v, want invalid", bytes[0], bytes[1], got)
		}
		if got.PhysicalReason == "" {
			t.Fatal("PhysicalReason is empty")
		}
	}
}

func TestResultSparseRatioTextPreservesDecisionPrecision(t *testing.T) {
	t.Parallel()

	got := Compare(nil, nil, 9, 1)
	if got.SparseRatioText() != "0.1111111111111111" {
		t.Fatalf("SparseRatioText() = %q", got.SparseRatioText())
	}

	infinite := Compare(nil, nil, 0, 1)
	if infinite.SparseRatioText() != "+Inf" {
		t.Fatalf("infinite SparseRatioText() = %q", infinite.SparseRatioText())
	}
}

func TestReadRowsSupportsNormalizedJSONAndCSV(t *testing.T) {
	t.Parallel()

	want := []Row{
		{
			Device:  "001/serial-a",
			Object:  "Cell=1",
			Time:    mustTime(t, "2026-07-24T00:00:00Z"),
			Counter: "Signal.RSRP",
			Value:   value(-95.5),
		},
		{
			Device:  "001/serial-a",
			Object:  "Cell=2",
			Time:    mustTime(t, "2026-07-24T00:15:00Z"),
			Counter: "Traffic.Bytes",
			Value:   nil,
		},
	}
	tests := map[string]string{
		"rows.json": `[
			{"device":"001/serial-a","object":"Cell=1","time":"2026-07-24T00:00:00Z","counter":"Signal.RSRP","value":-95.5},
			{"device":"001/serial-a","object":"Cell=2","time":"2026-07-24T00:15:00Z","counter":"Traffic.Bytes","value":null}
		]`,
		"rows.csv": "device,object,time,counter,value\n" +
			"001/serial-a,Cell=1,2026-07-24T00:00:00Z,Signal.RSRP,-95.5\n" +
			"001/serial-a,Cell=2,2026-07-24T00:15:00Z,Traffic.Bytes,\n",
		"rows-reordered.csv": "value,counter,time,object,device\n" +
			"-95.5,Signal.RSRP,2026-07-24T00:00:00Z,Cell=1,001/serial-a\n" +
			",Traffic.Bytes,2026-07-24T00:15:00Z,Cell=2,001/serial-a\n",
	}

	for name, contents := range tests {
		name, contents := name, contents
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), name)
			if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
				t.Fatalf("write fixture: %v", err)
			}

			got, err := ReadRows(path)
			if err != nil {
				t.Fatalf("ReadRows() error = %v", err)
			}
			if len(got) != len(want) {
				t.Fatalf("len(rows) = %d, want %d", len(got), len(want))
			}
			for i := range want {
				if got[i].Device != want[i].Device || got[i].Object != want[i].Object ||
					!got[i].Time.Equal(want[i].Time) || got[i].Counter != want[i].Counter ||
					!equalValues(got[i].Value, want[i].Value) {
					t.Errorf("row[%d] = %#v, want %#v", i, got[i], want[i])
				}
			}
		})
	}
}

func TestReadRowsRejectsMalformedInput(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"unsupported.txt":       "not a supported export",
		"null.json":             "null",
		"object.json":           "{}",
		"missing-device.json":   `[{"object":"Cell=1","time":"2026-07-24T00:00:00Z","counter":"Signal.RSRP","value":1}]`,
		"missing-object.json":   `[{"device":"001/serial-a","time":"2026-07-24T00:00:00Z","counter":"Signal.RSRP","value":1}]`,
		"missing-counter.json":  `[{"device":"001/serial-a","object":"Cell=1","time":"2026-07-24T00:00:00Z","value":1}]`,
		"missing-time.json":     `[{"device":"001/serial-a","object":"Cell=1","counter":"Signal.RSRP","value":1}]`,
		"missing-value.json":    `[{"device":"001/serial-a","object":"Cell=1","time":"2026-07-24T00:00:00Z","counter":"Signal.RSRP"}]`,
		"case-variant.json":     `[{"Device":"001/serial-a","object":"Cell=1","time":"2026-07-24T00:00:00Z","counter":"Signal.RSRP","value":1}]`,
		"duplicate-device.json": `[{"device":"001/serial-a","device":"other","object":"Cell=1","time":"2026-07-24T00:00:00Z","counter":"Signal.RSRP","value":1}]`,
		"duplicate-value.json":  `[{"device":"001/serial-a","object":"Cell=1","time":"2026-07-24T00:00:00Z","counter":"Signal.RSRP","value":null,"value":1}]`,
		"empty-device.json":     `[{"device":" ","object":"Cell=1","time":"2026-07-24T00:00:00Z","counter":"Signal.RSRP","value":1}]`,
		"empty-object.json":     `[{"device":"001/serial-a","object":"","time":"2026-07-24T00:00:00Z","counter":"Signal.RSRP","value":1}]`,
		"empty-counter.json":    `[{"device":"001/serial-a","object":"Cell=1","time":"2026-07-24T00:00:00Z","counter":" ","value":1}]`,
		"zero-time.json":        `[{"device":"001/serial-a","object":"Cell=1","time":"0001-01-01T00:00:00Z","counter":"Signal.RSRP","value":1}]`,
		"out-of-range.json":     `[{"device":"001/serial-a","object":"Cell=1","time":"1969-12-31T23:59:59Z","counter":"Signal.RSRP","value":1}]`,
		"unknown-field.json":    `[{"device":"001/serial-a","object":"Cell=1","time":"2026-07-24T00:00:00Z","counter":"Signal.RSRP","value":1,"extra":true}]`,
		"empty.csv":             "",
		"missing.csv":           "device,time,counter,value\n001/serial-a,2026-07-24T00:00:00Z,Signal.RSRP,1\n",
		"duplicate-header.csv":  "device,object,time,counter,value,device\n001/serial-a,Cell=1,2026-07-24T00:00:00Z,Signal.RSRP,1,other\n",
		"extra-header.csv":      "device,object,time,counter,value,extra\n001/serial-a,Cell=1,2026-07-24T00:00:00Z,Signal.RSRP,1,ignored\n",
		"case-header.csv":       "Device,object,time,counter,value\n001/serial-a,Cell=1,2026-07-24T00:00:00Z,Signal.RSRP,1\n",
		"empty-device.csv":      "device,object,time,counter,value\n,Cell=1,2026-07-24T00:00:00Z,Signal.RSRP,1\n",
		"empty-object.csv":      "device,object,time,counter,value\n001/serial-a,,2026-07-24T00:00:00Z,Signal.RSRP,1\n",
		"empty-counter.csv":     "device,object,time,counter,value\n001/serial-a,Cell=1,2026-07-24T00:00:00Z,,1\n",
		"empty-time.csv":        "device,object,time,counter,value\n001/serial-a,Cell=1,,Signal.RSRP,1\n",
		"bad-time.csv":          "device,object,time,counter,value\n001/serial-a,Cell=1,yesterday,Signal.RSRP,1\n",
		"out-of-range-time.csv": "device,object,time,counter,value\n001/serial-a,Cell=1,1969-12-31T23:59:59Z,Signal.RSRP,1\n",
		"bad-value.csv":         "device,object,time,counter,value\n001/serial-a,Cell=1,2026-07-24T00:00:00Z,Signal.RSRP,nope\n",
		"nan-value.csv":         "device,object,time,counter,value\n001/serial-a,Cell=1,2026-07-24T00:00:00Z,Signal.RSRP,NaN\n",
		"inf-value.csv":         "device,object,time,counter,value\n001/serial-a,Cell=1,2026-07-24T00:00:00Z,Signal.RSRP,+Inf\n",
	}

	for name, contents := range tests {
		name, contents := name, contents
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), name)
			if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
				t.Fatalf("write fixture: %v", err)
			}
			if _, err := ReadRows(path); err == nil {
				t.Fatal("ReadRows() error = nil, want malformed-input error")
			}
		})
	}
}

func TestReadRowsAllowsStrictEmptyExports(t *testing.T) {
	t.Parallel()

	for name, contents := range map[string]string{
		"empty.json": "[]",
		"empty.csv":  "device,object,time,counter,value\n",
	} {
		path := filepath.Join(t.TempDir(), name)
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
		rows, err := ReadRows(path)
		if err != nil {
			t.Fatalf("ReadRows(%s) error = %v", name, err)
		}
		if len(rows) != 0 {
			t.Fatalf("ReadRows(%s) = %#v, want empty", name, rows)
		}
	}
}

func TestCompareDuplicateOutputIsDeterministicAndCanonicalizesSignedZero(t *testing.T) {
	t.Parallel()

	timestamp := mustTime(t, "2026-07-24T00:00:00Z")
	base := Row{Device: "device-a", Object: "Cell=1", Time: timestamp, Counter: "Signal.RSRP"}
	negativeZero := math.Copysign(0, -1)
	rows := []Row{
		withValue(base, 2),
		withValue(base, negativeZero),
		withValue(base, 0),
		withValue(base, 1),
	}
	reversed := []Row{rows[3], rows[2], rows[1], rows[0]}
	sparse := []Row{withValue(base, 0)}

	first, err := json.Marshal(Compare(rows, sparse, 1000, 100).Mismatches)
	if err != nil {
		t.Fatalf("marshal first mismatch: %v", err)
	}
	second, err := json.Marshal(Compare(reversed, sparse, 1000, 100).Mismatches)
	if err != nil {
		t.Fatalf("marshal second mismatch: %v", err)
	}
	if string(first) != string(second) {
		t.Fatalf("duplicate output depends on input ordering:\nfirst:  %s\nsecond: %s", first, second)
	}
	if string(first) == "" || containsJSONNegativeZero(first) {
		t.Fatalf("duplicate output did not canonicalize signed zero: %s", first)
	}
}

func mustTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		t.Fatalf("parse time %q: %v", value, err)
	}
	return parsed
}

func value(v float64) *float64 {
	return &v
}

func withValue(row Row, v float64) Row {
	row.Value = value(v)
	return row
}

func containsJSONNegativeZero(value []byte) bool {
	for i := 0; i+2 < len(value); i++ {
		if value[i] == ':' && value[i+1] == '-' && value[i+2] == '0' {
			return true
		}
	}
	return false
}
