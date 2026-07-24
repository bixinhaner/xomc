// Package compare validates logical equivalence and physical size reduction
// between legacy and sparse PM runs.
package compare

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"
)

const MaxSparseRatio = 0.20

type Row struct {
	Device  string    `json:"device"`
	Object  string    `json:"object,omitempty"`
	Time    time.Time `json:"time"`
	Counter string    `json:"counter"`
	Value   *float64  `json:"value"`
}

type Key struct {
	Device  string    `json:"device"`
	Object  string    `json:"object,omitempty"`
	Time    time.Time `json:"time"`
	Counter string    `json:"counter"`
}

type MismatchKind string

const (
	MismatchValue         MismatchKind = "value"
	MismatchMissingOld    MismatchKind = "missing_old"
	MismatchMissingSparse MismatchKind = "missing_sparse"
	MismatchDuplicate     MismatchKind = "duplicate_key"
)

type Mismatch struct {
	Key        Key          `json:"key"`
	Kind       MismatchKind `json:"kind"`
	OldRows    []Row        `json:"old_rows,omitempty"`
	SparseRows []Row        `json:"sparse_rows,omitempty"`
}

type Result struct {
	LogicalEqual     bool       `json:"logical_equal"`
	Mismatches       []Mismatch `json:"mismatches"`
	OldBytes         int64      `json:"old_bytes"`
	SparseBytes      int64      `json:"sparse_bytes"`
	SparseRatio      float64    `json:"sparse_ratio"`
	WithinSizeTarget bool       `json:"within_size_target"`
	Accepted         bool       `json:"accepted"`
}

type rowKey struct {
	device  string
	object  string
	timeNS  int64
	counter string
}

func ReadRows(path string) ([]Row, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open logical rows: %w", err)
	}
	defer file.Close()

	switch filepath.Ext(path) {
	case ".json":
		return readJSONRows(file)
	case ".csv":
		return readCSVRows(file)
	default:
		return nil, fmt.Errorf("unsupported logical row format %q", filepath.Ext(path))
	}
}

func Compare(oldRows, sparseRows []Row, oldBytes, sparseBytes int64) Result {
	oldByKey := groupRows(oldRows)
	sparseByKey := groupRows(sparseRows)
	keys := unionKeys(oldByKey, sparseByKey)

	mismatches := make([]Mismatch, 0)
	for _, key := range keys {
		oldGroup := sortedRows(oldByKey[key])
		sparseGroup := sortedRows(sparseByKey[key])
		kind, different := mismatchKind(oldGroup, sparseGroup)
		if !different {
			continue
		}
		mismatches = append(mismatches, Mismatch{
			Key: Key{
				Device:  key.device,
				Object:  key.object,
				Time:    time.Unix(0, key.timeNS).UTC(),
				Counter: key.counter,
			},
			Kind:       kind,
			OldRows:    oldGroup,
			SparseRows: sparseGroup,
		})
	}

	ratio, withinTarget := physicalRatio(oldBytes, sparseBytes)
	logicalEqual := len(mismatches) == 0
	return Result{
		LogicalEqual:     logicalEqual,
		Mismatches:       mismatches,
		OldBytes:         oldBytes,
		SparseBytes:      sparseBytes,
		SparseRatio:      ratio,
		WithinSizeTarget: withinTarget,
		Accepted:         logicalEqual && withinTarget,
	}
}

func readJSONRows(reader io.Reader) ([]Row, error) {
	decoder := json.NewDecoder(reader)
	var rows []Row
	if err := decoder.Decode(&rows); err != nil {
		return nil, fmt.Errorf("decode JSON rows: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("decode JSON rows: multiple JSON values")
		}
		return nil, fmt.Errorf("decode JSON rows: %w", err)
	}
	return rows, nil
}

func readCSVRows(reader io.Reader) ([]Row, error) {
	records, err := csv.NewReader(reader).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("decode CSV rows: %w", err)
	}
	if len(records) == 0 {
		return []Row{}, nil
	}
	columns := make(map[string]int, len(records[0]))
	for index, name := range records[0] {
		columns[name] = index
	}
	for _, required := range []string{"device", "time", "counter", "value"} {
		if _, ok := columns[required]; !ok {
			return nil, fmt.Errorf("decode CSV rows: missing %q column", required)
		}
	}

	rows := make([]Row, 0, len(records)-1)
	for index, record := range records[1:] {
		timestamp, err := time.Parse(time.RFC3339Nano, record[columns["time"]])
		if err != nil {
			return nil, fmt.Errorf("decode CSV row %d time: %w", index+2, err)
		}
		var value *float64
		if raw := record[columns["value"]]; raw != "" {
			parsed, err := strconv.ParseFloat(raw, 64)
			if err != nil {
				return nil, fmt.Errorf("decode CSV row %d value: %w", index+2, err)
			}
			value = &parsed
		}
		object := ""
		if objectColumn, ok := columns["object"]; ok {
			object = record[objectColumn]
		}
		rows = append(rows, Row{
			Device:  record[columns["device"]],
			Object:  object,
			Time:    timestamp,
			Counter: record[columns["counter"]],
			Value:   value,
		})
	}
	return rows, nil
}

func groupRows(rows []Row) map[rowKey][]Row {
	grouped := make(map[rowKey][]Row, len(rows))
	for _, row := range rows {
		key := rowKey{device: row.Device, object: row.Object, timeNS: row.Time.UnixNano(), counter: row.Counter}
		grouped[key] = append(grouped[key], row)
	}
	return grouped
}

func unionKeys(left, right map[rowKey][]Row) []rowKey {
	seen := make(map[rowKey]struct{}, len(left)+len(right))
	for key := range left {
		seen[key] = struct{}{}
	}
	for key := range right {
		seen[key] = struct{}{}
	}
	keys := make([]rowKey, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].device != keys[j].device {
			return keys[i].device < keys[j].device
		}
		if keys[i].object != keys[j].object {
			return keys[i].object < keys[j].object
		}
		if keys[i].timeNS != keys[j].timeNS {
			return keys[i].timeNS < keys[j].timeNS
		}
		return keys[i].counter < keys[j].counter
	})
	return keys
}

func sortedRows(rows []Row) []Row {
	if len(rows) == 0 {
		return nil
	}
	out := append([]Row(nil), rows...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Value == nil {
			return out[j].Value != nil
		}
		if out[j].Value == nil {
			return false
		}
		return *out[i].Value < *out[j].Value
	})
	return out
}

func mismatchKind(oldRows, sparseRows []Row) (MismatchKind, bool) {
	if len(oldRows) > 1 || len(sparseRows) > 1 {
		return MismatchDuplicate, true
	}
	if len(oldRows) == 0 {
		return MismatchMissingOld, true
	}
	if len(sparseRows) == 0 {
		return MismatchMissingSparse, true
	}
	if equalValues(oldRows[0].Value, sparseRows[0].Value) {
		return "", false
	}
	return MismatchValue, true
}

func equalValues(left, right *float64) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func physicalRatio(oldBytes, sparseBytes int64) (float64, bool) {
	if oldBytes < 0 || sparseBytes < 0 {
		return math.Inf(1), false
	}
	if oldBytes == 0 {
		if sparseBytes == 0 {
			return 0, true
		}
		return math.Inf(1), false
	}
	ratio := float64(sparseBytes) / float64(oldBytes)
	return ratio, ratio <= MaxSparseRatio
}
