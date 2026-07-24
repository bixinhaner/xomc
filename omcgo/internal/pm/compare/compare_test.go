package compare

import (
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

		if !got.LogicalEqual || got.SparseRatio != 0 || !got.WithinSizeTarget || !got.Accepted {
			t.Fatalf("empty result = %#v", got)
		}
	})

	t.Run("sparse bytes without old baseline", func(t *testing.T) {
		got := Compare(nil, nil, 0, 1)

		if !math.IsInf(got.SparseRatio, 1) {
			t.Fatalf("SparseRatio = %v, want +Inf", got.SparseRatio)
		}
		if got.WithinSizeTarget || got.Accepted {
			t.Fatalf("WithinSizeTarget = %v, Accepted = %v, want false/false", got.WithinSizeTarget, got.Accepted)
		}
	})
}

func TestCompareFailsAboveTwentyPercentPhysicalThreshold(t *testing.T) {
	t.Parallel()

	atLimit := Compare(nil, nil, 1000, 200)
	if !atLimit.WithinSizeTarget || !atLimit.Accepted {
		t.Fatalf("20%% result = %#v, want accepted", atLimit)
	}

	overLimit := Compare(nil, nil, 1000, 201)
	if overLimit.WithinSizeTarget || overLimit.Accepted {
		t.Fatalf("20.1%% result = %#v, want rejected", overLimit)
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
		"unsupported.txt": "not a supported export",
		"missing.csv":     "device,time,value\n001/serial-a,2026-07-24T00:00:00Z,1\n",
		"bad-time.csv":    "device,time,counter,value\n001/serial-a,yesterday,Signal.RSRP,1\n",
		"bad-value.csv":   "device,time,counter,value\n001/serial-a,2026-07-24T00:00:00Z,Signal.RSRP,nope\n",
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
