package devsweep

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sampleXML = `<?xml version="1.0" encoding="UTF-8"?>
<parameterModel paramModel="TEST" totalEntries="4">
  <parameters>
    <param name="Device.A.B" standardPath="Device.A.B" access="readWrite" type="string"/>
    <param name="Device.A.C" standardPath="Device.A.C" type="int" supported="true"/>
    <param name="Device.A.D" standardPath="Device.A.D" type="int" supported="false"/>
    <param name="Device.E.F" type="int"/>
  </parameters>
</parameterModel>
`

func writeSampleXML(t *testing.T) (dir, file string) {
	t.Helper()
	dir = t.TempDir()
	file = filepath.Join(dir, "TEST.xml")
	require.NoError(t, os.WriteFile(file, []byte(sampleXML), 0o644))
	return dir, file
}

func Test_XMLWriter_AddSupportedFalse(t *testing.T) {
	dir, file := writeSampleXML(t)
	w := NewXMLWriter(dir)

	outcome, err := w.Apply("TEST", []string{"Device.A.B"})
	require.NoError(t, err)
	assert.Equal(t, 1, outcome.LinesModified)
	assert.Equal(t, []string{"Device.A.B"}, outcome.PathsApplied)
	assert.Empty(t, outcome.PathsAlreadyMarked)
	assert.Empty(t, outcome.PathsNotFound)

	got, _ := os.ReadFile(file)
	assert.Contains(t, string(got), `<param name="Device.A.B" standardPath="Device.A.B" access="readWrite" type="string" supported="false"/>`)
}

func Test_XMLWriter_ReplaceSupportedTrue(t *testing.T) {
	dir, file := writeSampleXML(t)
	w := NewXMLWriter(dir)

	outcome, err := w.Apply("TEST", []string{"Device.A.C"})
	require.NoError(t, err)
	assert.Equal(t, 1, outcome.LinesModified)
	assert.Equal(t, []string{"Device.A.C"}, outcome.PathsApplied)

	got, _ := os.ReadFile(file)
	assert.Contains(t, string(got), `supported="false"`)
	assert.NotContains(t, string(got), `Device.A.C" type="int" supported="true"`)
}

func Test_XMLWriter_SkipAlreadyFalse(t *testing.T) {
	dir, _ := writeSampleXML(t)
	w := NewXMLWriter(dir)

	outcome, err := w.Apply("TEST", []string{"Device.A.D"})
	require.NoError(t, err)
	assert.Equal(t, 0, outcome.LinesModified)
	assert.Empty(t, outcome.PathsApplied)
	assert.Equal(t, []string{"Device.A.D"}, outcome.PathsAlreadyMarked)
}

func Test_XMLWriter_FallbackNameAttr(t *testing.T) {
	dir, file := writeSampleXML(t)
	w := NewXMLWriter(dir)

	// Device.E.F 没 standardPath,只有 name → 走 fallback
	outcome, err := w.Apply("TEST", []string{"Device.E.F"})
	require.NoError(t, err)
	assert.Equal(t, 1, outcome.LinesModified)
	assert.Equal(t, []string{"Device.E.F"}, outcome.PathsApplied)

	got, _ := os.ReadFile(file)
	assert.Contains(t, string(got), `<param name="Device.E.F" type="int" supported="false"/>`)
}

func Test_XMLWriter_PathsNotFound(t *testing.T) {
	dir, _ := writeSampleXML(t)
	w := NewXMLWriter(dir)

	outcome, err := w.Apply("TEST", []string{"Device.Z.NOTEXIST"})
	require.NoError(t, err)
	assert.Equal(t, 0, outcome.LinesModified)
	assert.Equal(t, []string{"Device.Z.NOTEXIST"}, outcome.PathsNotFound)
}

func Test_XMLWriter_PreservesFormatting(t *testing.T) {
	dir, file := writeSampleXML(t)
	w := NewXMLWriter(dir)

	_, err := w.Apply("TEST", []string{"Device.A.B"})
	require.NoError(t, err)

	got, _ := os.ReadFile(file)
	// 注释/XML 声明/缩进保留
	assert.True(t, strings.HasPrefix(string(got), `<?xml version="1.0"`),
		"XML declaration preserved")
	assert.Contains(t, string(got), `  <parameters>`,
		"original indentation preserved")
	// 没改的行原样
	assert.Contains(t, string(got),
		`<param name="Device.A.D" standardPath="Device.A.D" type="int" supported="false"/>`)
}

func Test_XMLWriter_AtomicNoLeftoverTmp(t *testing.T) {
	dir, _ := writeSampleXML(t)
	w := NewXMLWriter(dir)

	_, err := w.Apply("TEST", []string{"Device.A.B"})
	require.NoError(t, err)

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	for _, e := range entries {
		assert.False(t, strings.Contains(e.Name(), ".tmp"),
			"no leftover tmp file: %s", e.Name())
	}
}

func Test_XMLWriter_MultiplePaths(t *testing.T) {
	dir, file := writeSampleXML(t)
	w := NewXMLWriter(dir)

	outcome, err := w.Apply("TEST", []string{
		"Device.A.B", // add
		"Device.A.C", // replace
		"Device.A.D", // already
		"Device.NOPE", // not found
	})
	require.NoError(t, err)
	assert.Equal(t, 2, outcome.LinesModified)
	assert.ElementsMatch(t, []string{"Device.A.B", "Device.A.C"}, outcome.PathsApplied)
	assert.Equal(t, []string{"Device.A.D"}, outcome.PathsAlreadyMarked)
	assert.Equal(t, []string{"Device.NOPE"}, outcome.PathsNotFound)

	got, _ := os.ReadFile(file)
	assert.Equal(t, 3, strings.Count(string(got), `supported="false"`))
}

func Test_XMLWriter_FileNotFound(t *testing.T) {
	dir := t.TempDir()
	w := NewXMLWriter(dir)

	_, err := w.Apply("MISSING", []string{"foo"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read XML")
}

func Test_setSupportedFalse(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		want     string
		changed  bool
		already  bool
	}{
		{
			name:    "add new",
			in:      `<param name="X" type="int"/>`,
			want:    `<param name="X" type="int" supported="false"/>`,
			changed: true,
		},
		{
			name:    "replace true",
			in:      `<param name="X" supported="true"/>`,
			want:    `<param name="X" supported="false"/>`,
			changed: true,
		},
		{
			name:    "already false",
			in:      `<param name="X" supported="false"/>`,
			want:    `<param name="X" supported="false"/>`,
			changed: false,
			already: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, changed, already := setSupportedFalse(tt.in)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.changed, changed)
			assert.Equal(t, tt.already, already)
		})
	}
}
