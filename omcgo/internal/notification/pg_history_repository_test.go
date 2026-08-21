package notification

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Compile-time interface check.
var _ HistoryRepository = (*PgHistoryRepository)(nil)

// TestPgHistoryRepository_ColumnsCoverAllFields guards against column drift.
func TestPgHistoryRepository_ColumnsCoverAllFields(t *testing.T) {
	t.Parallel()
	want := map[string]bool{
		"id":             false,
		"template_id":    false,
		"channel":        false,
		"recipients":     false,
		"subject":        false,
		"body":           false,
		"status":         false,
		"error_message":  false,
		"alarm_id":       false,
		"source_type":    false,
		"source_id":      false,
		"event_id":       false,
		"correlation_id": false,
		"retry_count":    false,
		"sent_at":        false,
		"created_at":     false,
	}
	for _, c := range historyColumns {
		if _, ok := want[c]; ok {
			want[c] = true
		}
	}
	for col, found := range want {
		assert.Truef(t, found, "historyColumns missing column %q", col)
	}
	assert.Equal(t, len(want), len(historyColumns), "historyColumns size drift")
}

// TestPgHistoryRepository_AllowedSortColumns ensures the sort whitelist only
// includes columns present in historyColumns.
func TestPgHistoryRepository_AllowedSortColumns(t *testing.T) {
	t.Parallel()
	cols := make(map[string]struct{}, len(historyColumns))
	for _, c := range historyColumns {
		cols[c] = struct{}{}
	}
	for sortCol := range historyAllowedSortColumns {
		_, ok := cols[sortCol]
		assert.Truef(t, ok, "sort column %q not in historyColumns", sortCol)
	}
}
