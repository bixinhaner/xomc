package slothealth

import (
	"strings"
	"time"
)

const SlotDuration = 15 * time.Minute

type Status string

const (
	StatusComplete         Status = "complete"
	StatusPartial          Status = "partial"
	StatusMissing          Status = "missing"
	StatusBootstrapIgnored Status = "bootstrap_ignored"
)

type ExpectedGroup struct {
	Technology      string
	Carrier         string
	Devices         int64
	SnapshotVersion string
}

type ReceivedGroup struct {
	Technology string
	Carrier    string
	Devices    int64
}

type Snapshot struct {
	SlotStart               time.Time
	SlotEnd                 time.Time
	Technology              string
	Carrier                 string
	ExpectedDevices         int64
	ReceivedDevices         int64
	CoverageRatio           float64
	ExpectedSnapshotVersion string
	EvaluatedAt             time.Time
	Status                  Status
}

func LatestEligibleSlotEnd(now time.Time, grace time.Duration) time.Time {
	return now.UTC().Add(-grace).Truncate(SlotDuration)
}

func BuildSnapshots(
	slotEnd time.Time,
	evaluatedAt time.Time,
	startupAt time.Time,
	expected []ExpectedGroup,
	received []ReceivedGroup,
) []Snapshot {
	receivedByKey := make(map[string]int64, len(received))
	for _, group := range received {
		receivedByKey[groupKey(group.Technology, group.Carrier)] = group.Devices
	}
	slotEnd = slotEnd.UTC()
	slotStart := slotEnd.Add(-SlotDuration)
	// Existing pre-release databases gain measurement_end through an idempotent
	// schema reconciliation, but historical pm_files rows cannot be backfilled
	// safely from upload time. Suppress every slot that began before this observer
	// instance, including the crossing slot; the first fully observed slot is
	// eligible normally.
	bootstrapIgnored := !startupAt.IsZero() && slotStart.Before(startupAt)
	out := make([]Snapshot, 0, len(expected))
	for _, group := range expected {
		technology := strings.ToLower(strings.TrimSpace(group.Technology))
		carrier := strings.ToLower(strings.TrimSpace(group.Carrier))
		receivedDevices := receivedByKey[groupKey(technology, carrier)]
		coverage := 0.0
		if group.Devices > 0 {
			coverage = float64(receivedDevices) / float64(group.Devices)
			if coverage > 1 {
				coverage = 1
			}
		}
		status := StatusMissing
		switch {
		case coverage >= 0.98:
			status = StatusComplete
		case bootstrapIgnored:
			status = StatusBootstrapIgnored
		case receivedDevices > 0:
			status = StatusPartial
		}
		out = append(out, Snapshot{
			SlotStart: slotStart, SlotEnd: slotEnd,
			Technology: technology, Carrier: carrier,
			ExpectedDevices: group.Devices, ReceivedDevices: receivedDevices,
			CoverageRatio: coverage, ExpectedSnapshotVersion: group.SnapshotVersion,
			EvaluatedAt: evaluatedAt.UTC(), Status: status,
		})
	}
	return out
}

func groupKey(technology, carrier string) string {
	return strings.ToLower(strings.TrimSpace(technology)) + "\x00" +
		strings.ToLower(strings.TrimSpace(carrier))
}
