//go:build ignore

// Command pm_sparse_compare compares logical PM JSON or CSV exports produced by
// the legacy and sparse implementations. It intentionally does not compare
// physical row counts.
package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
)

type summary struct {
	Rows     int
	Nulls    int
	Filled   int
	NonNulls int
}

func main() {
	oldPath := flag.String("old", "", "legacy logical JSON or CSV export")
	newPath := flag.String("new", "", "sparse logical JSON or CSV export")
	flag.Parse()
	if *oldPath == "" || *newPath == "" {
		fatal(errors.New("both --old and --new are required"))
	}
	oldValue, oldSummary, err := readLogical(*oldPath)
	if err != nil {
		fatal(fmt.Errorf("read old export: %w", err))
	}
	newValue, newSummary, err := readLogical(*newPath)
	if err != nil {
		fatal(fmt.Errorf("read new export: %w", err))
	}
	if !reflect.DeepEqual(oldValue, newValue) {
		oldJSON, _ := json.MarshalIndent(oldValue, "", "  ")
		newJSON, _ := json.MarshalIndent(newValue, "", "  ")
		fmt.Fprintf(os.Stderr, "logical exports differ\nold summary: %+v\nnew summary: %+v\n", oldSummary, newSummary)
		printFirstDifference(oldJSON, newJSON)
		os.Exit(1)
	}
	fmt.Printf("logical exports match: rows=%d nulls=%d non_nulls=%d filled=%d\n",
		newSummary.Rows, newSummary.Nulls, newSummary.NonNulls, newSummary.Filled)
}

func readLogical(path string) (any, summary, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, summary{}, err
	}
	defer f.Close()
	if filepath.Ext(path) == ".csv" {
		return readCSV(f)
	}
	return readJSON(f)
}

func readJSON(r io.Reader) (any, summary, error) {
	dec := json.NewDecoder(r)
	dec.UseNumber()
	var value any
	if err := dec.Decode(&value); err != nil {
		return nil, summary{}, err
	}
	if err := ensureEOF(dec); err != nil {
		return nil, summary{}, err
	}
	value = normalizeJSONExport(value)
	var out summary
	countJSON(value, "", &out)
	if rows, ok := value.([]any); ok {
		out.Rows = len(rows)
	} else if object, ok := value.(map[string]any); ok {
		for _, key := range []string{"items", "data", "rows"} {
			if rows, ok := object[key].([]any); ok {
				out.Rows = len(rows)
				break
			}
		}
	}
	return value, out, nil
}

var ignoredJSONFields = map[string]struct{}{
	"id": {}, "ingest_time": {}, "created_at": {}, "updated_at": {},
	"request_id": {}, "trace_id": {}, "next_cursor": {},
}

func normalizeJSONExport(value any) any {
	// API wrappers vary between endpoints and releases; compare their logical
	// row payload, not transport-only envelope fields.
	if object, ok := value.(map[string]any); ok {
		for _, key := range []string{"items", "data", "rows"} {
			if rows, ok := object[key].([]any); ok {
				value = rows
				break
			}
		}
	}
	rows, ok := value.([]any)
	if !ok {
		return normalizeJSONValue(value)
	}
	normalized := make([]any, len(rows))
	for i, row := range rows {
		normalized[i] = normalizeJSONValue(row)
	}
	sort.SliceStable(normalized, func(i, j int) bool {
		left, _ := json.Marshal(normalized[i])
		right, _ := json.Marshal(normalized[j])
		return bytes.Compare(left, right) < 0
	})
	return normalized
}

func normalizeJSONValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			if _, ignored := ignoredJSONFields[key]; ignored {
				continue
			}
			out[key] = normalizeJSONValue(child)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, child := range typed {
			out[i] = normalizeJSONValue(child)
		}
		return out
	default:
		return value
	}
}

func ensureEOF(dec *json.Decoder) error {
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func countJSON(value any, key string, out *summary) {
	switch typed := value.(type) {
	case nil:
		out.Nulls++
	case map[string]any:
		for childKey, child := range typed {
			if childKey == "filled" {
				if filled, ok := child.(bool); ok && filled {
					out.Filled++
				}
			}
			countJSON(child, childKey, out)
		}
	case []any:
		for _, child := range typed {
			countJSON(child, key, out)
		}
	default:
		if key == "metric_value" || key == "value" {
			out.NonNulls++
		}
	}
}

func readCSV(r io.Reader) (any, summary, error) {
	records, err := csv.NewReader(r).ReadAll()
	if err != nil {
		return nil, summary{}, err
	}
	records = normalizeCSV(records)
	out := summary{}
	if len(records) > 0 {
		out.Rows = len(records) - 1
	}
	for rowNo, row := range records {
		if rowNo == 0 {
			continue
		}
		for _, value := range row {
			if value == "" {
				out.Nulls++
			}
		}
	}
	return records, out, nil
}

func normalizeCSV(records [][]string) [][]string {
	if len(records) == 0 {
		return records
	}
	ignored := map[string]struct{}{
		"id": {}, "ingest_time": {}, "created_at": {}, "updated_at": {},
	}
	keep := make([]int, 0, len(records[0]))
	for i, header := range records[0] {
		if _, skip := ignored[header]; !skip {
			keep = append(keep, i)
		}
	}
	out := make([][]string, 0, len(records))
	for _, record := range records {
		row := make([]string, 0, len(keep))
		for _, index := range keep {
			if index < len(record) {
				row = append(row, record[index])
			} else {
				row = append(row, "")
			}
		}
		out = append(out, row)
	}
	return out
}

func printFirstDifference(oldJSON, newJSON []byte) {
	oldLines := bytes.Split(oldJSON, []byte{'\n'})
	newLines := bytes.Split(newJSON, []byte{'\n'})
	limit := len(oldLines)
	if len(newLines) < limit {
		limit = len(newLines)
	}
	for i := 0; i < limit; i++ {
		if !bytes.Equal(oldLines[i], newLines[i]) {
			fmt.Fprintf(os.Stderr, "first difference at line %d\nold: %s\nnew: %s\n", i+1, oldLines[i], newLines[i])
			return
		}
	}
	fmt.Fprintf(os.Stderr, "document lengths differ: old=%d lines new=%d lines\n", len(oldLines), len(newLines))
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(2)
}
