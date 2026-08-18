package license

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCapacityGroupKey(t *testing.T) {
	assert.Equal(t, "ENB", capacityGroupKey("GSM"), "GSM shares eNB capacity (issue #318)")
	assert.Equal(t, "ENB", capacityGroupKey("gsm"), "grouping is case-insensitive")
	assert.Equal(t, "ENB", capacityGroupKey(" eNB "))
	assert.Equal(t, "GNB", capacityGroupKey("gNB"), "ungrouped types map to themselves (upper)")
	assert.Equal(t, "", capacityGroupKey(""))
}

func TestCapacityGroupUsage(t *testing.T) {
	usedByType := map[string]int{"ENB": 3, "GSM": 4, "GNB": 5}
	assert.Equal(t, 7, capacityGroupUsage(usedByType, "eNB"), "eNB pool usage = ENB + GSM")
	assert.Equal(t, 4, capacityGroupUsage(usedByType, "GSM"), "quota keys are group keys; raw per-ne_type lookup")
	assert.Equal(t, 5, capacityGroupUsage(usedByType, "gNB"), "ungrouped usage passes through")
	assert.Equal(t, 0, capacityGroupUsage(nil, "eNB"))
}
