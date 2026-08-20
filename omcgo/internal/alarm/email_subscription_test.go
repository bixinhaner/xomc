package alarm

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestResolveAlarmEmailRecipientsMergesNormalizesAndDeduplicates(t *testing.T) {
	got, err := ResolveAlarmEmailRecipients(
		[]string{" Ops@Example.com ", "Alice <alice@example.com>"},
		[]string{"ops@example.com", "bob@example.com"},
		true,
	)
	require.NoError(t, err)
	require.Equal(t, []string{"alice@example.com", "bob@example.com", "ops@example.com"}, got)
}

func TestResolveAlarmEmailRecipientsDoesNotUseDefaultsUnlessRequested(t *testing.T) {
	got, err := ResolveAlarmEmailRecipients([]string{"ops@example.com"}, []string{"default@example.com"}, false)
	require.NoError(t, err)
	require.Equal(t, []string{"ops@example.com"}, got)
}

func TestResolveAlarmEmailRecipientsRejectsInvalidAndEmpty(t *testing.T) {
	_, err := ResolveAlarmEmailRecipients([]string{"not-an-email"}, nil, false)
	require.ErrorIs(t, err, ErrAlarmEmailInvalidRecipient)

	_, err = ResolveAlarmEmailRecipients(nil, []string{"default@example.com"}, false)
	require.ErrorIs(t, err, ErrAlarmEmailNoRecipients)
}

func TestBuildAlarmEmailWindowAppliesToleranceWithoutDroppingClearedAlarm(t *testing.T) {
	now := time.Date(2026, 8, 17, 14, 37, 45, 0, time.FixedZone("CST", 8*60*60))
	window, err := BuildAlarmEmailWindow(now, 10, 10)
	require.NoError(t, err)
	require.Equal(t, time.Date(2026, 8, 17, 14, 10, 0, 0, now.Location()), window.Start)
	require.Equal(t, time.Date(2026, 8, 17, 14, 20, 0, 0, now.Location()), window.End)

	// 14:12 raised and 14:17 cleared is still selected by raised_at in the
	// delayed source window once 14:30 is reached; clearing does not suppress it.
	raisedAt := time.Date(2026, 8, 17, 14, 12, 0, 0, now.Location())
	clearedAt := time.Date(2026, 8, 17, 14, 17, 0, 0, now.Location())
	require.True(t, !raisedAt.Before(window.Start) && raisedAt.Before(window.End))
	require.True(t, clearedAt.Before(window.End))
}

func TestAlarmEmailRealtimeScheduledAtWaitsTolerance(t *testing.T) {
	raised := time.Date(2026, 8, 17, 15, 0, 0, 0, time.UTC)
	got, err := AlarmEmailRealtimeScheduledAt(raised, 10)
	require.NoError(t, err)
	require.Equal(t, raised.Add(10*time.Minute), got)
}

func TestAlarmEmailSubscriptionNormalizeAndValidate(t *testing.T) {
	subscription := AlarmEmailSubscription{
		Name:                     "  NOC critical alarms  ",
		Enabled:                  true,
		IntervalMinutes:          10,
		ToleranceMinutes:         0,
		Recipients:               []string{"NOC@example.com", "noc@example.com"},
		IncludeDefaultRecipients: true,
	}
	require.NoError(t, subscription.NormalizeAndValidate([]string{"default@example.com"}))
	require.Equal(t, "NOC critical alarms", subscription.Name)
	require.Equal(t, []string{"noc@example.com"}, subscription.Recipients)
}

func TestAlarmEmailSubscriptionRejectsUnsupportedMinuteOptions(t *testing.T) {
	subscription := AlarmEmailSubscription{
		Name:             "bad interval",
		Enabled:          true,
		IntervalMinutes:  5,
		ToleranceMinutes: 0,
		Recipients:       []string{"noc@example.com"},
	}
	require.ErrorIs(t, subscription.NormalizeAndValidate(nil), ErrAlarmEmailInvalidInterval)

	subscription.IntervalMinutes = 10
	subscription.ToleranceMinutes = 5
	require.ErrorIs(t, subscription.NormalizeAndValidate(nil), ErrAlarmEmailInvalidTolerance)
}
