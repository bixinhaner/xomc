package notification

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
)

func TestMatchRules_UsesWithinFieldORAndAcrossFieldAND(t *testing.T) {
	technology := "nr"
	snapshot := event.AlarmLifecycleSnapshot{
		DeviceID: uuid.New(), Carrier: model.CarrierCMCC, Technology: &technology,
		Severity: model.AlarmCritical, AlarmIdentifier: "CELL_UNAVAILABLE",
	}
	matchingID, wrongCarrierID := uuid.New(), uuid.New()
	matches := MatchRules(snapshot, []RuleCandidate{
		{ID: matchingID, Priority: 20, Conditions: RuleMatchConditions{
			AlarmIdentifiers: []string{"POWER_FAILURE", "CELL_UNAVAILABLE"},
			Severities:       []model.AlarmSeverity{model.AlarmCritical, model.AlarmMajor},
			Carriers:         []model.CarrierCode{model.CarrierCMCC},
			Technologies:     []model.Technology{model.TechNR},
		}},
		{ID: wrongCarrierID, Priority: 10, Conditions: RuleMatchConditions{
			AlarmIdentifiers: []string{"CELL_UNAVAILABLE"}, Carriers: []model.CarrierCode{model.CarrierCTCC},
		}},
	})

	require.Equal(t, []RuleMatch{{RuleID: matchingID, Priority: 20, Specificity: 4}}, matches)
}

func TestMatchRules_OrdersByPrioritySpecificityThenStableID(t *testing.T) {
	snapshot := event.AlarmLifecycleSnapshot{Severity: model.AlarmCritical, AlarmIdentifier: "A"}
	stableFirst := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	stableSecond := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	highPriority := uuid.New()
	matches := MatchRules(snapshot, []RuleCandidate{
		{ID: stableSecond, Priority: 20, Conditions: RuleMatchConditions{Severities: []model.AlarmSeverity{model.AlarmCritical}}},
		{ID: highPriority, Priority: 10},
		{ID: stableFirst, Priority: 20, Conditions: RuleMatchConditions{Severities: []model.AlarmSeverity{model.AlarmCritical}}},
	})

	require.Equal(t, highPriority, matches[0].RuleID)
	require.Equal(t, stableFirst, matches[1].RuleID)
	require.Equal(t, stableSecond, matches[2].RuleID)
}

func TestMatchRulesForDeviceGroups_MatchesCurrentGroupMembership(t *testing.T) {
	matchingGroup, otherGroup := uuid.New(), uuid.New()
	snapshot := event.AlarmLifecycleSnapshot{
		DeviceID: uuid.New(), Severity: model.AlarmMajor, AlarmIdentifier: "GPS_UNAVAILABLE",
	}
	matchingRule, nonMatchingRule := uuid.New(), uuid.New()

	matches := MatchRulesForDeviceGroups(snapshot, []uuid.UUID{matchingGroup}, []RuleCandidate{
		{ID: matchingRule, Priority: 10, Conditions: RuleMatchConditions{DeviceGroupIDs: []uuid.UUID{matchingGroup}}},
		{ID: nonMatchingRule, Priority: 20, Conditions: RuleMatchConditions{DeviceGroupIDs: []uuid.UUID{otherGroup}}},
	})

	require.Equal(t, []RuleMatch{{RuleID: matchingRule, Priority: 10, Specificity: 1}}, matches)
}

func TestMergeRuleRecipientCandidates_DeduplicatesSameAddressPerChannel(t *testing.T) {
	fingerprint := []byte("same-address")
	winningRule, losingRule := uuid.New(), uuid.New()
	merged := MergeRuleRecipientCandidates([]RuleRecipientCandidate{
		{Match: RuleMatch{RuleID: losingRule, Priority: 20}, Recipient: ResolvedRecipient{Channel: "email", Fingerprint: fingerprint}},
		{Match: RuleMatch{RuleID: winningRule, Priority: 10}, Recipient: ResolvedRecipient{Channel: "email", Fingerprint: fingerprint}},
		{Match: RuleMatch{RuleID: losingRule, Priority: 20}, Recipient: ResolvedRecipient{Channel: "sms", Fingerprint: fingerprint}},
	})

	require.Len(t, merged, 2)
	require.Equal(t, winningRule, merged[0].Match.RuleID)
	require.Equal(t, "email", merged[0].Recipient.Channel)
	require.Equal(t, "sms", merged[1].Recipient.Channel)
}
