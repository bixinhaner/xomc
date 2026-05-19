package device

import (
	"strings"
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildSearchOR(t *testing.T) {
	tests := []struct {
		name        string
		search      string
		fields      []string
		wantNil     bool
		wantClauses int // 期望 OR 子句数量 = keywords × fields
	}{
		{
			name:    "empty search returns nil",
			search:  "",
			fields:  []string{"a", "b"},
			wantNil: true,
		},
		{
			name:    "only whitespace returns nil",
			search:  "   ",
			fields:  []string{"a"},
			wantNil: true,
		},
		{
			name:    "only commas returns nil",
			search:  ", , ,",
			fields:  []string{"a"},
			wantNil: true,
		},
		{
			name:    "empty fields returns nil",
			search:  "x",
			fields:  []string{},
			wantNil: true,
		},
		{
			name:        "single keyword × 3 fields → 3 ILIKE",
			search:      "ABC0027",
			fields:      []string{"sn", "name", "ip"},
			wantClauses: 3,
		},
		{
			name:        "two keywords × 3 fields → 6 ILIKE",
			search:      "ABC0027,ABC0039",
			fields:      []string{"sn", "name", "ip"},
			wantClauses: 6,
		},
		{
			name:        "mixed spaces and empty entries trimmed",
			search:      "  SN1 , ,SN2,,  SN3  ",
			fields:      []string{"sn"},
			wantClauses: 3,
		},
		{
			name:        "over limit truncates to MaxSearchKeywords",
			search:      strings.Repeat("k,", MaxSearchKeywords+10), // MaxSearchKeywords+10 个 "k"
			fields:      []string{"f"},
			wantClauses: MaxSearchKeywords,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cond := BuildSearchOR(tc.search, tc.fields)
			if tc.wantNil {
				assert.Nil(t, cond)
				return
			}
			require.NotNil(t, cond)
			assert.Equal(t, tc.wantClauses, len(cond))
		})
	}
}

// TestBuildSearchOR_GeneratesValidSQL — 确认 sq.Or 能被 ToSql 渲染（防 panic 兜底）。
func TestBuildSearchOR_GeneratesValidSQL(t *testing.T) {
	cond := BuildSearchOR("SN1,SN2", []string{"d.serial_number", "d.site_name"})
	require.NotNil(t, cond)

	// 用 SELECT * FROM t WHERE <cond> 作为 wrapper，验 ToSql 不报错。
	q, args, err := sq.Select("*").From("t").Where(cond).
		PlaceholderFormat(sq.Dollar).ToSql()
	require.NoError(t, err)
	assert.Contains(t, q, "ILIKE")
	assert.Contains(t, q, "OR")
	assert.Len(t, args, 4, "2 keywords × 2 fields = 4 placeholders")
}
