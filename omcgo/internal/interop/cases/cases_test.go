package cases

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/omcgo/omcgo/internal/interop"
)

// TestCases_CategoryCounts pins the per-category case counts so accidental
// deletions / merges are caught immediately. Counts come from T-0030 Phase 1
// targets (PRD §8): DM=6 / Protocol=7 / RPC=13 / Inform=4.
func TestCases_CategoryCounts(t *testing.T) {
	tests := []struct {
		name     string
		got      []interop.TestCase
		expected int
		category interop.TestCategory
	}{
		{"datamodel", DataModelCases(), 6, interop.CategoryDataModel},
		{"protocol", ProtocolCases(), 7, interop.CategoryProtocol},
		{"rpc", RPCCases(), 13, interop.CategoryRPC},
		{"inform", InformCases(), 4, interop.CategoryInform},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Len(t, tt.got, tt.expected, "%s case count drifted from PRD §8 target", tt.name)
			for _, c := range tt.got {
				assert.Equal(t, tt.category, c.Category, "case %s has wrong category", c.ID)
			}
		})
	}
}

// TestCases_NegativePathPerCategory enforces PRD §3 V3 "每 category 至少 1 个
// negative path" — guards against regressions where negative tests get deleted
// while well-meaning contributors trim suites.
func TestCases_NegativePathPerCategory(t *testing.T) {
	categories := map[string][]interop.TestCase{
		"datamodel": DataModelCases(),
		"protocol":  ProtocolCases(),
		"rpc":       RPCCases(),
		"inform":    InformCases(),
	}
	for name, cases := range categories {
		t.Run(name, func(t *testing.T) {
			negCount := 0
			for _, c := range cases {
				if c.ExpectedOutcome == "fail" {
					negCount++
				}
			}
			assert.GreaterOrEqual(t, negCount, 1, "%s must contain at least one negative path case", name)
		})
	}
}

// TestCases_IDsUniqueAndExpectedOutcomeValid ensures (a) every case ID is
// globally unique across categories, and (b) ExpectedOutcome only takes the
// documented values "", "pass", or "fail".
func TestCases_IDsUniqueAndExpectedOutcomeValid(t *testing.T) {
	all := append(append(append(
		DataModelCases(),
		ProtocolCases()...),
		RPCCases()...),
		InformCases()...)

	seen := make(map[string]string, len(all))
	for _, c := range all {
		if prevCat, dup := seen[c.ID]; dup {
			t.Errorf("duplicate case ID %s (already in %s, now in %s)", c.ID, prevCat, c.Category)
		}
		seen[c.ID] = string(c.Category)

		switch c.ExpectedOutcome {
		case "", "pass", "fail":
			// ok
		default:
			t.Errorf("case %s has invalid ExpectedOutcome %q (allowed: \"\", \"pass\", \"fail\")", c.ID, c.ExpectedOutcome)
		}

		assert.NotEmpty(t, c.Steps, "case %s has no steps", c.ID)
		assert.NotEmpty(t, c.Name, "case %s has no Name", c.ID)
	}

	// Sanity: total count matches PRD §3 V1 "≥ 30".
	assert.GreaterOrEqual(t, len(all), 30, "total case count fell below PRD §3 V1 threshold")
}
