package notification

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Compile-time interface check. Failing this would surface as a build error,
// but having a test makes the contract explicit.
var _ TemplateRepository = (*PgTemplateRepository)(nil)

// TestPgTemplateRepository_ColumnsContainVariables guards against the W1.5
// regression where the variables column was missing from one of the SQL
// statements. All five SQL operations (List/Get/GetByName/Create/Update) reuse
// templateColumns (or the matching Set list), so a single check on the column
// list is enough.
func TestPgTemplateRepository_ColumnsContainVariables(t *testing.T) {
	t.Parallel()
	want := map[string]bool{
		"id":         false,
		"name":       false,
		"channel":    false,
		"language":   false,
		"subject":    false,
		"body":       false,
		"variables":  false,
		"enabled":    false,
		"created_at": false,
		"updated_at": false,
	}
	for _, c := range templateColumns {
		if _, ok := want[c]; ok {
			want[c] = true
		}
	}
	for col, found := range want {
		assert.Truef(t, found, "templateColumns missing column %q", col)
	}
	// Reject silent column drift.
	assert.Equal(t, len(want), len(templateColumns), "templateColumns size drift")
}

// TestPgTemplateRepository_AllowedSortColumns ensures the sort whitelist only
// includes columns that exist in templateColumns.
func TestPgTemplateRepository_AllowedSortColumns(t *testing.T) {
	t.Parallel()
	cols := make(map[string]struct{}, len(templateColumns))
	for _, c := range templateColumns {
		cols[c] = struct{}{}
	}
	for sortCol := range templateAllowedSortColumns {
		_, ok := cols[sortCol]
		assert.Truef(t, ok, "sort column %q not in templateColumns", sortCol)
	}
}
