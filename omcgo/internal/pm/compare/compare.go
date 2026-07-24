// Package compare validates logical equivalence and physical size reduction
// between legacy and sparse PM runs.
package compare

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	MaxSparseRatio         = 1.0 / 5.0
	sparseRatioDenominator = int64(5)
)

type Row struct {
	Device  string    `json:"device"`
	Object  string    `json:"object"`
	Time    time.Time `json:"time"`
	Counter string    `json:"counter"`
	Value   *float64  `json:"value"`
}

type Key struct {
	Device  string    `json:"device"`
	Object  string    `json:"object"`
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
	PhysicalValid    bool       `json:"physical_valid"`
	PhysicalReason   string     `json:"physical_reason"`
	WithinSizeTarget bool       `json:"within_size_target"`
	Accepted         bool       `json:"accepted"`
}

func (r Result) SparseRatioText() string {
	return strconv.FormatFloat(r.SparseRatio, 'g', 17, 64)
}

type rowKey struct {
	device  string
	object  string
	time    time.Time
	counter string
}

var (
	minLogicalTimestamp = time.Unix(0, 0).UTC()
	maxLogicalTimestamp = time.Date(9999, 12, 31, 23, 59, 59, 999999999, time.UTC)
	canonicalRowFields  = [...]string{"device", "object", "time", "counter", "value"}
)

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
				Time:    key.time,
				Counter: key.counter,
			},
			Kind:       kind,
			OldRows:    oldGroup,
			SparseRows: sparseGroup,
		})
	}

	ratio, physicalValid, withinTarget, physicalReason := physicalAssessment(
		oldBytes,
		sparseBytes,
		len(oldRows) == 0 && len(sparseRows) == 0,
	)
	logicalEqual := len(mismatches) == 0
	return Result{
		LogicalEqual:     logicalEqual,
		Mismatches:       mismatches,
		OldBytes:         oldBytes,
		SparseBytes:      sparseBytes,
		SparseRatio:      ratio,
		PhysicalValid:    physicalValid,
		PhysicalReason:   physicalReason,
		WithinSizeTarget: withinTarget,
		Accepted:         logicalEqual && physicalValid && withinTarget,
	}
}

func readJSONRows(reader io.Reader) ([]Row, error) {
	decoder := json.NewDecoder(reader)
	var document json.RawMessage
	if err := decoder.Decode(&document); err != nil {
		return nil, fmt.Errorf("decode JSON rows: %w", err)
	}
	trimmed := bytes.TrimSpace(document)
	if len(trimmed) == 0 || trimmed[0] != '[' {
		return nil, fmt.Errorf("decode JSON rows: top-level value must be an array")
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("decode JSON rows: multiple JSON values")
		}
		return nil, fmt.Errorf("decode JSON rows: %w", err)
	}

	var documents []json.RawMessage
	if err := json.Unmarshal(trimmed, &documents); err != nil {
		return nil, fmt.Errorf("decode JSON rows: %w", err)
	}
	rows := make([]Row, 0, len(documents))
	for index, document := range documents {
		row, err := decodeJSONRow(document)
		if err != nil {
			return nil, fmt.Errorf("decode JSON row %d: %w", index+1, err)
		}
		if err := validateRow(&row); err != nil {
			return nil, fmt.Errorf("decode JSON row %d: %w", index+1, err)
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func decodeJSONRow(document json.RawMessage) (Row, error) {
	decoder := json.NewDecoder(bytes.NewReader(document))
	token, err := decoder.Token()
	if err != nil {
		return Row{}, fmt.Errorf("decode object: %w", err)
	}
	if delimiter, ok := token.(json.Delim); !ok || delimiter != '{' {
		return Row{}, fmt.Errorf("row must be an object")
	}

	seen := make(map[string]struct{}, len(canonicalRowFields))
	row := Row{}
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return Row{}, fmt.Errorf("decode member name: %w", err)
		}
		name, ok := token.(string)
		if !ok {
			return Row{}, fmt.Errorf("row member name must be a string")
		}
		if !canonicalRowField(name) {
			return Row{}, fmt.Errorf("unknown member %q", name)
		}
		if _, duplicate := seen[name]; duplicate {
			return Row{}, fmt.Errorf("duplicate member %q", name)
		}
		seen[name] = struct{}{}

		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return Row{}, fmt.Errorf("decode member %q: %w", name, err)
		}
		switch name {
		case "device":
			err = json.Unmarshal(value, &row.Device)
		case "object":
			err = json.Unmarshal(value, &row.Object)
		case "time":
			err = json.Unmarshal(value, &row.Time)
		case "counter":
			err = json.Unmarshal(value, &row.Counter)
		case "value":
			err = json.Unmarshal(value, &row.Value)
		}
		if err != nil {
			return Row{}, fmt.Errorf("decode member %q: %w", name, err)
		}
	}
	if _, err := decoder.Token(); err != nil {
		return Row{}, fmt.Errorf("decode object end: %w", err)
	}
	if err := decoder.Decode(&token); err != io.EOF {
		if err == nil {
			return Row{}, fmt.Errorf("multiple values in row object")
		}
		return Row{}, fmt.Errorf("decode object: %w", err)
	}
	for _, required := range canonicalRowFields {
		if _, ok := seen[required]; !ok {
			return Row{}, fmt.Errorf("missing member %q", required)
		}
	}
	return row, nil
}

func canonicalRowField(name string) bool {
	for _, canonical := range canonicalRowFields {
		if name == canonical {
			return true
		}
	}
	return false
}

func readCSVRows(reader io.Reader) ([]Row, error) {
	records, err := csv.NewReader(reader).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("decode CSV rows: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("decode CSV rows: header row is required")
	}
	columns := make(map[string]int, len(records[0]))
	for index, name := range records[0] {
		if _, duplicate := columns[name]; duplicate {
			return nil, fmt.Errorf("decode CSV rows: duplicate %q column", name)
		}
		columns[name] = index
	}
	if len(columns) != len(canonicalRowFields) {
		return nil, fmt.Errorf("decode CSV rows: header must contain exactly device, object, time, counter, value")
	}
	for _, required := range canonicalRowFields {
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
		row := Row{
			Device:  record[columns["device"]],
			Object:  record[columns["object"]],
			Time:    timestamp,
			Counter: record[columns["counter"]],
			Value:   value,
		}
		if err := validateRow(&row); err != nil {
			return nil, fmt.Errorf("decode CSV row %d: %w", index+2, err)
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func validateRow(row *Row) error {
	switch {
	case strings.TrimSpace(row.Device) == "":
		return fmt.Errorf("device is required")
	case strings.TrimSpace(row.Object) == "":
		return fmt.Errorf("object is required")
	case strings.TrimSpace(row.Counter) == "":
		return fmt.Errorf("counter is required")
	case row.Time.IsZero():
		return fmt.Errorf("time must not be zero")
	}
	row.Time = row.Time.Round(0).UTC()
	if row.Time.Before(minLogicalTimestamp) || row.Time.After(maxLogicalTimestamp) {
		return fmt.Errorf("time %s is outside the supported range [%s, %s]",
			row.Time.Format(time.RFC3339Nano),
			minLogicalTimestamp.Format(time.RFC3339Nano),
			maxLogicalTimestamp.Format(time.RFC3339Nano),
		)
	}
	if row.Value != nil {
		if math.IsNaN(*row.Value) || math.IsInf(*row.Value, 0) {
			return fmt.Errorf("value must be finite")
		}
		if *row.Value == 0 {
			zero := float64(0)
			row.Value = &zero
		}
	}
	return nil
}

func groupRows(rows []Row) map[rowKey][]Row {
	grouped := make(map[rowKey][]Row, len(rows))
	for _, row := range rows {
		row = canonicalRow(row)
		key := rowKey{device: row.Device, object: row.Object, time: row.Time, counter: row.Counter}
		grouped[key] = append(grouped[key], row)
	}
	return grouped
}

func canonicalRow(row Row) Row {
	row.Time = row.Time.Round(0).UTC()
	if row.Value != nil && *row.Value == 0 {
		zero := float64(0)
		row.Value = &zero
	}
	return row
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
		if !keys[i].time.Equal(keys[j].time) {
			return keys[i].time.Before(keys[j].time)
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

func physicalAssessment(oldBytes, sparseBytes int64, logicalInputsEmpty bool) (float64, bool, bool, string) {
	if oldBytes < 0 || sparseBytes < 0 {
		return math.Inf(1), false, false, fmt.Sprintf(
			"invalid physical measurement: old_bytes=%d and sparse_bytes=%d must both be nonnegative",
			oldBytes,
			sparseBytes,
		)
	}
	if oldBytes == 0 {
		if sparseBytes == 0 {
			if logicalInputsEmpty {
				return 0, true, true, "empty run: both logical inputs and physical byte counts are empty"
			}
			return 0, false, false,
				"unmeasured physical size: nonempty logical inputs cannot use old_bytes=0 and sparse_bytes=0"
		}
		return math.Inf(1), true, false, fmt.Sprintf(
			"exact size gate failed: old_bytes=0 cannot cover sparse_bytes=%d",
			sparseBytes,
		)
	}
	ratio := float64(sparseBytes) / float64(oldBytes)
	maxSparseBytes := oldBytes / sparseRatioDenominator
	if sparseBytes <= maxSparseBytes {
		return ratio, true, true, fmt.Sprintf(
			"exact size gate passed: sparse_bytes=%d <= floor(old_bytes/5)=%d (equivalent to 5*sparse_bytes <= old_bytes)",
			sparseBytes,
			maxSparseBytes,
		)
	}
	return ratio, true, false, fmt.Sprintf(
		"exact size gate failed: sparse_bytes=%d > floor(old_bytes/5)=%d (equivalent to 5*sparse_bytes > old_bytes)",
		sparseBytes,
		maxSparseBytes,
	)
}
