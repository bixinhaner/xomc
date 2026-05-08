package dictloader

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// ReportError describes a non-fatal error encountered during a load operation.
// Loaders should accumulate ReportErrors and continue when a single file fails;
// fatal errors should bubble up via the LoadOnce/Reload return value instead.
type ReportError struct {
	File  string
	Stage string // "scan" | "parse" | "validate" | "persist" | ...
	Err   error
}

func (e ReportError) Error() string {
	return fmt.Sprintf("%s: stage=%s: %v", e.File, e.Stage, e.Err)
}

func (e ReportError) Unwrap() error { return e.Err }

// Report aggregates the result of one LoadOnce or Reload run.
type Report struct {
	LoaderName   string
	StartedAt    time.Time
	FinishedAt   time.Time
	Duration     time.Duration
	FilesScanned int
	FilesLoaded  int
	FilesSkipped int
	RowsAffected int
	Errors       []ReportError
}

// NewReport returns a Report with StartedAt set to now and LoaderName populated.
func NewReport(loaderName string) Report {
	return Report{LoaderName: loaderName, StartedAt: time.Now()}
}

// Finish stamps FinishedAt/Duration based on time.Now.
func (r *Report) Finish() {
	r.FinishedAt = time.Now()
	r.Duration = r.FinishedAt.Sub(r.StartedAt)
}

// AddError appends a non-fatal ReportError. A nil err is a no-op.
func (r *Report) AddError(file, stage string, err error) {
	if err == nil {
		return
	}
	r.Errors = append(r.Errors, ReportError{File: file, Stage: stage, Err: err})
}

// HasErrors reports whether any non-fatal errors were collected.
func (r Report) HasErrors() bool { return len(r.Errors) > 0 }

// FirstError returns the first ReportError as an error value, or nil if no errors.
func (r Report) FirstError() error {
	if len(r.Errors) == 0 {
		return nil
	}
	return r.Errors[0]
}

// Combined wraps all accumulated ReportErrors via errors.Join, or returns nil.
func (r Report) Combined() error {
	if len(r.Errors) == 0 {
		return nil
	}
	errs := make([]error, len(r.Errors))
	for i, e := range r.Errors {
		errs[i] = e
	}
	return errors.Join(errs...)
}

// String returns a single-line summary suitable for INFO logging.
func (r Report) String() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "loader=%s scanned=%d loaded=%d skipped=%d rows=%d duration=%s",
		r.LoaderName, r.FilesScanned, r.FilesLoaded, r.FilesSkipped, r.RowsAffected, r.Duration)
	if r.HasErrors() {
		fmt.Fprintf(&sb, " errors=%d first=%q", len(r.Errors), r.Errors[0].Error())
	}
	return sb.String()
}
