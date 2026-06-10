package topology

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuildBatchSortSQL covers the CASE WHEN SQL built by BatchSort. Regression
// for issue #120: bare $N placeholders in the CASE branches were inferred as
// text by PG, so assigning to the integer sort_order column always failed with
// SQLSTATE 42804 — placeholders must carry explicit ::int / ::uuid casts.
func TestBuildBatchSortSQL(t *testing.T) {
	t.Parallel()

	id1 := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	id2 := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	cases := []struct {
		name     string
		items    []BatchSortItem
		wantSQL  string
		wantArgs []interface{}
	}{
		{
			name:  "single item casts value to int and id to uuid",
			items: []BatchSortItem{{ID: id1, SortOrder: 9}},
			wantSQL: "UPDATE device_groups SET sort_order = CASE " +
				"WHEN id = $1::uuid THEN $2::int" +
				" END WHERE id IN ($1::uuid)",
			wantArgs: []interface{}{id1, 9},
		},
		{
			name: "multiple items number placeholders pairwise",
			items: []BatchSortItem{
				{ID: id1, SortOrder: 1},
				{ID: id2, SortOrder: 2},
			},
			wantSQL: "UPDATE device_groups SET sort_order = CASE " +
				"WHEN id = $1::uuid THEN $2::int WHEN id = $3::uuid THEN $4::int" +
				" END WHERE id IN ($1::uuid, $3::uuid)",
			wantArgs: []interface{}{id1, 1, id2, 2},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotSQL, gotArgs := buildBatchSortSQL(tc.items)

			assert.Equal(t, tc.wantSQL, gotSQL)
			require.Len(t, gotArgs, len(tc.wantArgs))
			assert.Equal(t, tc.wantArgs, gotArgs)
		})
	}
}
