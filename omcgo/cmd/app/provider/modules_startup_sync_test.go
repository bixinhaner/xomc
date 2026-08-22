package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShouldScheduleStartupFullSyncSkipsReleaseCampaignPackages(t *testing.T) {
	assert.False(t, shouldScheduleStartupFullSync(true, true, true))
	assert.True(t, shouldScheduleStartupFullSync(true, true, false))
	assert.False(t, shouldScheduleStartupFullSync(false, true, false))
	assert.False(t, shouldScheduleStartupFullSync(true, false, false))
}
