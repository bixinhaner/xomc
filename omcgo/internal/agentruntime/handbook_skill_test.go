package agentruntime

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOMCOperationsSkillUsesProgressiveLocalHandbook(t *testing.T) {
	skillRoot := filepath.Join("..", "..", "data", "agent-skill", "omc-operations")
	raw, err := os.ReadFile(filepath.Join(skillRoot, "SKILL.md"))
	require.NoError(t, err)
	content := string(raw)

	require.Contains(t, content, "references/manifest.json")
	require.Contains(t, content, "references/api-index.jsonl")
	require.Contains(t, content, "references/api-categories")
	require.Contains(t, content, "references/api-docs")
	require.Contains(t, content, "catalogVersion")
	require.Contains(t, content, "rg")
	require.NotContains(t, content, "/api/v1/agent/catalog")
	require.NotContains(t, content, "node \"$CLI\" describe")
	commonRaw, err := os.ReadFile(filepath.Join(skillRoot, "references", "common-operations.md"))
	require.NoError(t, err)
	common := string(commonRaw)
	require.NotContains(t, common, "live catalog")
	require.NotContains(t, common, "Describe the operation")

	_, err = os.Stat(filepath.Join(skillRoot, "references", "manifest.json"))
	require.NoError(t, err)
	entries, err := os.ReadDir(filepath.Join(skillRoot, "references", "api-categories"))
	require.NoError(t, err)
	require.NotEmpty(t, entries)
	documents, err := os.ReadDir(filepath.Join(skillRoot, "references", "api-docs"))
	require.NoError(t, err)
	require.NotEmpty(t, documents)
}
