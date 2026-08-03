package buildinfo

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReleaseCampaignIDIsOpaqueAndStable(t *testing.T) {
	oldVersion, oldCommit := ReleaseVersion, GitCommit
	t.Cleanup(func() {
		ReleaseVersion, GitCommit = oldVersion, oldCommit
	})

	ReleaseVersion = "V100R003C20-customer-A"
	GitCommit = "abcdef123456"
	first, ok := ReleaseCampaignID()
	require.True(t, ok)
	second, ok := ReleaseCampaignID()
	require.True(t, ok)
	assert.Equal(t, first, second)
	assert.NotEqual(t, uuid.Nil, first)

	ReleaseVersion = "2027Q1-GA"
	repackaged, ok := ReleaseCampaignID()
	require.True(t, ok)
	assert.Equal(t, first, repackaged, "repackaging the same commit must not restart a full release campaign")

	GitCommit = "fedcba654321"
	changed, ok := ReleaseCampaignID()
	require.True(t, ok)
	assert.NotEqual(t, first, changed)
}

func TestReleaseCampaignIDRejectsDevelopmentMetadata(t *testing.T) {
	oldVersion, oldCommit := ReleaseVersion, GitCommit
	t.Cleanup(func() {
		ReleaseVersion, GitCommit = oldVersion, oldCommit
	})

	for _, tc := range []struct{ version, commit string }{
		{"", "abcdef"},
		{"dev", "abcdef"},
		{"1.0.0", ""},
		{"1.0.0", "unknown"},
		{"1.0.0", "n/a"},
	} {
		ReleaseVersion, GitCommit = tc.version, tc.commit
		id, ok := ReleaseCampaignID()
		assert.False(t, ok)
		assert.Equal(t, uuid.Nil, id)
	}
}
