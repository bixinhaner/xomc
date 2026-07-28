package calendarfilter

import (
	"context"
	"strings"
	"time"
)

const DefaultTimezone = "UTC"

// TimezoneProvider is the minimal system-timezone contract shared by app and worker code.
type TimezoneProvider interface {
	Location(ctx context.Context) *time.Location
}

func ProviderName(ctx context.Context, provider TimezoneProvider) string {
	if provider == nil {
		return DefaultTimezone
	}
	loc := provider.Location(ctx)
	if loc == nil {
		return DefaultTimezone
	}
	return NormalizeName(loc.String())
}

func NormalizeName(name string) string {
	if strings.TrimSpace(name) == "" {
		return DefaultTimezone
	}
	return strings.TrimSpace(name)
}

func Location(name string) (*time.Location, string) {
	normalized := NormalizeName(name)
	loc, err := time.LoadLocation(normalized)
	if err != nil {
		return time.UTC, DefaultTimezone
	}
	return loc, normalized
}

func LocalTimestampExpr(column string) string {
	return "(" + column + " AT TIME ZONE ?)"
}

func ExtractDOWPredicate(column string) string {
	return "EXTRACT(dow FROM " + LocalTimestampExpr(column) + ")::int = ANY(?)"
}

func ExtractHourPredicate(column string) string {
	return "EXTRACT(hour FROM " + LocalTimestampExpr(column) + ")::int = ANY(?)"
}
