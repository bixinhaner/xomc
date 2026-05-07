package dictloader

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReport_Finish(t *testing.T) {
	r := NewReport("test")
	time.Sleep(time.Millisecond)
	r.Finish()
	assert.Greater(t, r.Duration, time.Duration(0))
	assert.False(t, r.FinishedAt.Before(r.StartedAt))
}

func TestReport_AddError_HasErrors(t *testing.T) {
	r := NewReport("test")
	assert.False(t, r.HasErrors())

	r.AddError("file.xml", "parse", errors.New("bad token"))
	assert.True(t, r.HasErrors())
	require.Len(t, r.Errors, 1)
	assert.Equal(t, "file.xml", r.Errors[0].File)
	assert.Equal(t, "parse", r.Errors[0].Stage)

	// Nil error is no-op.
	r.AddError("file2.xml", "parse", nil)
	assert.Len(t, r.Errors, 1)
}

func TestReport_FirstError(t *testing.T) {
	r := NewReport("test")
	assert.NoError(t, r.FirstError())

	want := errors.New("a")
	r.AddError("1.xml", "parse", want)
	assert.ErrorIs(t, r.FirstError(), want)
}

func TestReport_Combined(t *testing.T) {
	r := NewReport("test")
	assert.NoError(t, r.Combined())

	e1 := errors.New("a")
	e2 := errors.New("b")
	r.AddError("1.xml", "parse", e1)
	r.AddError("2.xml", "validate", e2)

	combined := r.Combined()
	require.Error(t, combined)
	assert.ErrorIs(t, combined, e1)
	assert.ErrorIs(t, combined, e2)
}

func TestReport_String(t *testing.T) {
	r := NewReport("loader-x")
	r.FilesScanned = 10
	r.FilesLoaded = 8
	r.FilesSkipped = 2
	r.RowsAffected = 123
	r.Duration = 250 * time.Millisecond

	s := r.String()
	assert.Contains(t, s, "loader=loader-x")
	assert.Contains(t, s, "scanned=10")
	assert.Contains(t, s, "loaded=8")
	assert.Contains(t, s, "skipped=2")
	assert.Contains(t, s, "rows=123")
	assert.NotContains(t, s, "errors=")

	r.AddError("bad.xml", "parse", errors.New("oops"))
	assert.Contains(t, r.String(), "errors=1")
}

func TestReportError_Error_Unwrap(t *testing.T) {
	base := errors.New("io")
	re := ReportError{File: "f.xml", Stage: "parse", Err: base}
	assert.Contains(t, re.Error(), "f.xml")
	assert.Contains(t, re.Error(), "parse")
	assert.ErrorIs(t, re, base)
}
