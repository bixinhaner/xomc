package cases

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/omcgo/omcgo/internal/interop"
)

// TestCases_CategoryCounts pins the per-category case counts so accidental
// deletions / merges are caught immediately. Counts after T-0114 fault-inject
// category (PRD §8 Phase 2 option): DM=7 / Protocol=8 / RPC=17 / Inform=5 /
// FaultInject=4 (total 41). Phase 1 baseline was 30; Phase 2 second-negative
// + carrier private RPC raised to 37; T-0114 fault-inject adds 4.
func TestCases_CategoryCounts(t *testing.T) {
	tests := []struct {
		name     string
		got      []interop.TestCase
		expected int
		category interop.TestCategory
	}{
		{"datamodel", DataModelCases(), 7, interop.CategoryDataModel},
		{"protocol", ProtocolCases(), 8, interop.CategoryProtocol},
		{"rpc", RPCCases(), 17, interop.CategoryRPC},
		{"inform", InformCases(), 5, interop.CategoryInform},
		{"fault_inject", FaultInjectCases(), 4, interop.CategoryFaultInject},
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

// TestCases_NegativePathPerCategory enforces PRD §7 Phase 2 修订门槛
// "每 category 至少 2 个 negative path" — guards against regressions where
// negative tests get deleted while well-meaning contributors trim suites.
// Phase 1 baseline was ≥ 1; Phase 2 raised to ≥ 2; fault_inject treated
// looser (≥ 1) since the entire category serves as fault tooling.
func TestCases_NegativePathPerCategory(t *testing.T) {
	categories := []struct {
		name    string
		cases   []interop.TestCase
		minNeg  int
		comment string
	}{
		{"datamodel", DataModelCases(), 2, "PRD §7 Phase 2"},
		{"protocol", ProtocolCases(), 2, "PRD §7 Phase 2"},
		{"rpc", RPCCases(), 2, "PRD §7 Phase 2"},
		{"inform", InformCases(), 2, "PRD §7 Phase 2"},
		{"fault_inject", FaultInjectCases(), 1, "T-0114: whole category is fault tooling; ≥1 negative suffices"},
	}
	for _, cat := range categories {
		t.Run(cat.name, func(t *testing.T) {
			negCount := 0
			for _, c := range cat.cases {
				if c.ExpectedOutcome == "fail" {
					negCount++
				}
			}
			assert.GreaterOrEqual(t, negCount, cat.minNeg, "%s must contain at least %d negative path case(s) (%s)", cat.name, cat.minNeg, cat.comment)
		})
	}
}

// TestCases_CarrierCoverage enforces PRD §7 Phase 2 修订门槛 "三家运营商覆盖"
// — at least one case per carrier (cmcc / ctcc / cucc) tagged in Description.
// The runner has no carrier-specific code branches (§16.4); coverage is
// declared via "Carrier: <code>" prefix in Description per PRD §9.6.
func TestCases_CarrierCoverage(t *testing.T) {
	all := append(append(append(append(
		DataModelCases(),
		ProtocolCases()...),
		RPCCases()...),
		InformCases()...),
		FaultInjectCases()...)

	carriers := map[string]bool{"cmcc": false, "ctcc": false, "cucc": false}
	for _, c := range all {
		for code := range carriers {
			if containsTag(c.Description, "Carrier: "+code) {
				carriers[code] = true
			}
		}
	}
	for code, seen := range carriers {
		assert.True(t, seen, "no test case tagged 'Carrier: %s' — PRD §7 Phase 2 三家运营商覆盖", code)
	}
}

// containsTag is a substring helper to avoid importing strings just for this.
func containsTag(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

// TestCases_IDsUniqueAndExpectedOutcomeValid ensures (a) every case ID is
// globally unique across categories, and (b) ExpectedOutcome only takes the
// documented values "", "pass", or "fail".
func TestCases_IDsUniqueAndExpectedOutcomeValid(t *testing.T) {
	all := append(append(append(append(
		DataModelCases(),
		ProtocolCases()...),
		RPCCases()...),
		InformCases()...),
		FaultInjectCases()...)

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

	// Sanity: total count after T-0114 fault-inject category (≥ 39).
	assert.GreaterOrEqual(t, len(all), 39, "total case count fell below T-0114 threshold (37 Phase 2 + 4 fault-inject - 2 buffer)")
}
