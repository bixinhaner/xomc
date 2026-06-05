package definition

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestClassifySource_Sidecar 验证 sidecar 判定:有 X.xml.custom → custom,无 → builtin,空串 → unknown。
func TestClassifySource_Sidecar(t *testing.T) {
	baseDir := t.TempDir()
	dir := filepath.Join(baseDir, BuiltinDirSubdir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// builtin: 仅 XML,无 sidecar
	if err := os.WriteFile(filepath.Join(dir, "ENB.xml"), []byte(`<alarmModel/>`), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	// custom: XML + sidecar
	if err := os.WriteFile(filepath.Join(dir, "MY.xml"), []byte(`<alarmModel/>`), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "MY.xml"+CustomMarkerSuffix), []byte{}, 0o644); err != nil {
		t.Fatalf("write sidecar: %v", err)
	}

	assert.Equal(t, SourceBuiltin, ClassifySource(baseDir, "alarm-definitions/ENB.xml"))
	assert.Equal(t, SourceCustom, ClassifySource(baseDir, "alarm-definitions/MY.xml"))
	assert.Equal(t, SourceUnknown, ClassifySource(baseDir, ""))

	assert.False(t, IsDeletable(baseDir, "alarm-definitions/ENB.xml"))
	assert.True(t, IsDeletable(baseDir, "alarm-definitions/MY.xml"))
	assert.False(t, IsDeletable(baseDir, ""))

	assert.True(t, IsCustom(baseDir, "alarm-definitions/MY.xml"))
	assert.False(t, IsCustom(baseDir, "alarm-definitions/ENB.xml"))
	assert.False(t, IsCustom(baseDir, ""))
}
