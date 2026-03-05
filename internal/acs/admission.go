package acs

import "sync/atomic"

// AdmissionController limits the total number of concurrent ACS sessions.
type AdmissionController struct {
	maxSessions     int64
	currentSessions atomic.Int64
}

// NewAdmissionController creates an admission controller with the given session limit.
func NewAdmissionController(maxSessions int64) *AdmissionController {
	if maxSessions <= 0 {
		maxSessions = 10000
	}
	return &AdmissionController{maxSessions: maxSessions}
}

// Acquire attempts to claim a session slot. Returns true if admitted.
func (ac *AdmissionController) Acquire() bool {
	for {
		current := ac.currentSessions.Load()
		if current >= ac.maxSessions {
			return false
		}
		if ac.currentSessions.CompareAndSwap(current, current+1) {
			return true
		}
	}
}

// Release returns a session slot.
func (ac *AdmissionController) Release() {
	ac.currentSessions.Add(-1)
}

// Current returns the number of active sessions.
func (ac *AdmissionController) Current() int64 {
	return ac.currentSessions.Load()
}
