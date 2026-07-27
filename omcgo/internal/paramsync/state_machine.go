package paramsync

import (
	"errors"
	"fmt"
)

var ErrInvalidStateTransition = errors.New("invalid state transition")

var requestTransitions = map[RequestStatus]map[RequestStatus]struct{}{
	RequestStatusAccepted:     setRequest(RequestStatusQueued, RequestStatusRunning, RequestStatusSucceeded, RequestStatusFailed, RequestStatusTimedOut, RequestStatusCancelled, RequestStatusDeduplicated, RequestStatusRejected),
	RequestStatusQueued:       setRequest(RequestStatusRunning, RequestStatusFailed, RequestStatusTimedOut, RequestStatusCancelled, RequestStatusDeduplicated, RequestStatusRejected),
	RequestStatusRunning:      setRequest(RequestStatusSucceeded, RequestStatusFailed, RequestStatusTimedOut, RequestStatusCancelled),
	RequestStatusSucceeded:    {},
	RequestStatusFailed:       {},
	RequestStatusTimedOut:     {},
	RequestStatusCancelled:    {},
	RequestStatusDeduplicated: {},
	RequestStatusRejected:     {},
}

var runTransitions = map[RunStatus]map[RunStatus]struct{}{
	RunStatusPlanning:      setRun(RunStatusEnqueuing, RunStatusCancelling, RunStatusFailed, RunStatusCancelled),
	RunStatusEnqueuing:     setRun(RunStatusWaitingDevice, RunStatusExecuting, RunStatusCancelling, RunStatusFailed, RunStatusCancelled),
	RunStatusWaitingDevice: setRun(RunStatusExecuting, RunStatusProcessing, RunStatusCancelling, RunStatusCancelled),
	RunStatusExecuting:     setRun(RunStatusProcessing, RunStatusCancelling, RunStatusCancelled),
	RunStatusProcessing:    setRun(RunStatusSucceeded, RunStatusCancelling, RunStatusCancelled),
	RunStatusCancelling:    setRun(RunStatusFailed, RunStatusCancelled),
	RunStatusSucceeded:     {},
	RunStatusFailed:        {},
	RunStatusCancelled:     {},
}

func setRequest(statuses ...RequestStatus) map[RequestStatus]struct{} {
	m := make(map[RequestStatus]struct{}, len(statuses))
	for _, status := range statuses {
		m[status] = struct{}{}
	}
	return m
}

func setRun(statuses ...RunStatus) map[RunStatus]struct{} {
	m := make(map[RunStatus]struct{}, len(statuses))
	for _, status := range statuses {
		m[status] = struct{}{}
	}
	return m
}

func ValidateRequestTransition(from, to RequestStatus) error {
	if from == to {
		return nil
	}
	allowed, known := requestTransitions[from]
	if !known {
		return fmt.Errorf("%w: unknown request status %q", ErrInvalidStateTransition, from)
	}
	if _, ok := allowed[to]; !ok {
		return fmt.Errorf("%w: request %s -> %s", ErrInvalidStateTransition, from, to)
	}
	return nil
}

func ValidateRunTransition(from, to RunStatus) error {
	if from == to {
		return nil
	}
	allowed, known := runTransitions[from]
	if !known {
		return fmt.Errorf("%w: unknown run status %q", ErrInvalidStateTransition, from)
	}
	if _, ok := allowed[to]; !ok {
		return fmt.Errorf("%w: run %s -> %s", ErrInvalidStateTransition, from, to)
	}
	return nil
}
