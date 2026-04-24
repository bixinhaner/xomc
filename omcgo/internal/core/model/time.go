package model

import (
	"database/sql/driver"
	"fmt"
	"time"
)

const TimeFormat = "2006/1/2 15:04:05"

// Time wraps time.Time for custom JSON serialization in API responses.
// It formats timestamps as "2026/4/24 15:29:03" instead of RFC 3339.
type Time time.Time

func (t Time) MarshalJSON() ([]byte, error) {
	if time.Time(t).IsZero() {
		return []byte(`""`), nil
	}
	s := time.Time(t).Format(TimeFormat)
	return []byte(`"` + s + `"`), nil
}

func (t *Time) UnmarshalJSON(data []byte) error {
	s := string(data)
	if s == "null" || s == `""` {
		return nil
	}
	// Strip quotes
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	if s == "" {
		return nil
	}
	// Try our format first
	parsed, err := time.ParseInLocation(TimeFormat, s, time.Local)
	if err == nil {
		*t = Time(parsed)
		return nil
	}
	// Fallback to RFC 3339
	parsed, err = time.Parse(time.RFC3339, s)
	if err != nil {
		return fmt.Errorf("parse time %q: %w", s, err)
	}
	*t = Time(parsed)
	return nil
}

func (t *Time) Scan(value interface{}) error {
	if value == nil {
		*t = Time{}
		return nil
	}
	switch v := value.(type) {
	case time.Time:
		*t = Time(v)
	default:
		return fmt.Errorf("cannot scan %T into model.Time", value)
	}
	return nil
}

func (t Time) Value() (driver.Value, error) {
	return time.Time(t), nil
}

func (t Time) Std() time.Time {
	return time.Time(t)
}

func (t Time) IsZero() bool {
	return time.Time(t).IsZero()
}

func NowTime() Time {
	return Time(time.Now())
}
