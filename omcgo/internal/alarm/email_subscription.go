package alarm

import (
	"errors"
	"fmt"
	"net/mail"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	AlarmEmailJobType = "alarm_email_delivery"

	AlarmEmailIntervalRealtime = 0
	AlarmEmailInterval10Min    = 10
	AlarmEmailInterval30Min    = 30
	AlarmEmailInterval60Min    = 60
)

var (
	ErrAlarmEmailInvalidInterval  = errors.New("invalid alarm email interval")
	ErrAlarmEmailInvalidTolerance = errors.New("invalid alarm email tolerance")
	ErrAlarmEmailNoRecipients     = errors.New("alarm email subscription has no recipients")
	ErrAlarmEmailInvalidRecipient = errors.New("invalid alarm email recipient")
	ErrAlarmEmailInvalidWindow    = errors.New("invalid alarm email window")
	ErrAlarmEmailSubscriptionName = errors.New("alarm email subscription name is required")
)

// AlarmEmailGlobalSetting contains the OMC-wide recipients that a subscription
// may opt into. It is deliberately separate from the SMTP sender account.
type AlarmEmailGlobalSetting struct {
	Enabled           bool       `json:"enabled" db:"enabled"`
	DefaultRecipients []string   `json:"default_recipients" db:"default_recipients"`
	UpdatedBy         *uuid.UUID `json:"updated_by,omitempty" db:"updated_by"`
	UpdatedAt         time.Time  `json:"updated_at" db:"updated_at"`
}

// AlarmEmailSubscription is an email-only alarm view. Its scope mirrors the
// existing alarm dimensions without reusing alarm_filters actions.
type AlarmEmailSubscription struct {
	ID                       uuid.UUID   `json:"id" db:"id"`
	Name                     string      `json:"name" db:"name"`
	Description              string      `json:"description" db:"description"`
	Enabled                  bool        `json:"enabled" db:"enabled"`
	IntervalMinutes          int         `json:"interval_minutes" db:"interval_minutes"`
	ToleranceMinutes         int         `json:"tolerance_minutes" db:"tolerance_minutes"`
	Recipients               []string    `json:"recipients" db:"recipients"`
	IncludeDefaultRecipients bool        `json:"include_default_recipients" db:"include_default_recipients"`
	AlarmIdentifiers         []string    `json:"alarm_identifiers" db:"alarm_identifiers"`
	Severities               []int16     `json:"severities" db:"severities"`
	AlarmSources             []string    `json:"alarm_sources" db:"alarm_sources"`
	EventTypes               []string    `json:"event_types" db:"event_types"`
	DeviceIDs                []uuid.UUID `json:"device_ids" db:"device_ids"`
	DeviceGroupIDs           []uuid.UUID `json:"device_group_ids" db:"device_group_ids"`
	CreatedBy                *uuid.UUID  `json:"created_by,omitempty" db:"created_by"`
	UpdatedBy                *uuid.UUID  `json:"updated_by,omitempty" db:"updated_by"`
	CreatedAt                time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt                time.Time   `json:"updated_at" db:"updated_at"`
}

type AlarmEmailWindow struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

func (s *AlarmEmailSubscription) NormalizeAndValidate(defaultRecipients []string) error {
	s.Name = strings.TrimSpace(s.Name)
	if s.Name == "" || len([]rune(s.Name)) > 128 {
		return ErrAlarmEmailSubscriptionName
	}
	if !validAlarmEmailMinuteOption(s.IntervalMinutes) {
		return fmt.Errorf("%w: %d", ErrAlarmEmailInvalidInterval, s.IntervalMinutes)
	}
	if !validAlarmEmailMinuteOption(s.ToleranceMinutes) {
		return fmt.Errorf("%w: %d", ErrAlarmEmailInvalidTolerance, s.ToleranceMinutes)
	}

	var err error
	s.Recipients, err = NormalizeEmailRecipients(s.Recipients)
	if err != nil {
		return err
	}
	if s.Enabled {
		resolved, resolveErr := ResolveAlarmEmailRecipients(
			s.Recipients,
			defaultRecipients,
			s.IncludeDefaultRecipients,
		)
		if resolveErr != nil {
			return resolveErr
		}
		if len(resolved) == 0 {
			return ErrAlarmEmailNoRecipients
		}
	}
	return nil
}

func validAlarmEmailMinuteOption(value int) bool {
	switch value {
	case AlarmEmailIntervalRealtime, AlarmEmailInterval10Min, AlarmEmailInterval30Min, AlarmEmailInterval60Min:
		return true
	default:
		return false
	}
}

// NormalizeEmailRecipients validates, lower-cases and de-duplicates addresses.
// Display names are intentionally discarded so delivery identity is stable.
func NormalizeEmailRecipients(input []string) ([]string, error) {
	seen := make(map[string]struct{}, len(input))
	out := make([]string, 0, len(input))
	for _, raw := range input {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		parsed, err := mail.ParseAddress(raw)
		if err != nil || parsed.Address == "" {
			return nil, fmt.Errorf("%w: %q", ErrAlarmEmailInvalidRecipient, raw)
		}
		address := strings.ToLower(strings.TrimSpace(parsed.Address))
		if _, exists := seen[address]; exists {
			continue
		}
		seen[address] = struct{}{}
		out = append(out, address)
	}
	sort.Strings(out)
	return out, nil
}

func ResolveAlarmEmailRecipients(templateRecipients, defaultRecipients []string, includeDefault bool) ([]string, error) {
	combined := append([]string(nil), templateRecipients...)
	if includeDefault {
		combined = append(combined, defaultRecipients...)
	}
	resolved, err := NormalizeEmailRecipients(combined)
	if err != nil {
		return nil, err
	}
	if len(resolved) == 0 {
		return nil, ErrAlarmEmailNoRecipients
	}
	return resolved, nil
}

// BuildAlarmEmailWindow returns the deterministic periodic source window.
// Tolerance shifts the whole interval backwards: alarms are not eligible until
// they have waited for the configured duration. If they clear while waiting,
// they remain eligible as cleared alarms once this window opens.
func BuildAlarmEmailWindow(now time.Time, intervalMinutes, toleranceMinutes int) (AlarmEmailWindow, error) {
	if intervalMinutes == AlarmEmailIntervalRealtime || !validAlarmEmailMinuteOption(intervalMinutes) {
		return AlarmEmailWindow{}, fmt.Errorf("%w: %d", ErrAlarmEmailInvalidInterval, intervalMinutes)
	}
	if !validAlarmEmailMinuteOption(toleranceMinutes) {
		return AlarmEmailWindow{}, fmt.Errorf("%w: %d", ErrAlarmEmailInvalidTolerance, toleranceMinutes)
	}
	interval := time.Duration(intervalMinutes) * time.Minute
	eligibleEnd := now.Add(-time.Duration(toleranceMinutes) * time.Minute).Truncate(interval)
	window := AlarmEmailWindow{Start: eligibleEnd.Add(-interval), End: eligibleEnd}
	if !window.Start.Before(window.End) {
		return AlarmEmailWindow{}, ErrAlarmEmailInvalidWindow
	}
	return window, nil
}

func AlarmEmailRealtimeScheduledAt(raisedAt time.Time, toleranceMinutes int) (time.Time, error) {
	if !validAlarmEmailMinuteOption(toleranceMinutes) {
		return time.Time{}, fmt.Errorf("%w: %d", ErrAlarmEmailInvalidTolerance, toleranceMinutes)
	}
	return raisedAt.Add(time.Duration(toleranceMinutes) * time.Minute), nil
}
