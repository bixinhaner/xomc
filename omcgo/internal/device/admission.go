package device

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/pkg/tr069"
)

const (
	AccessDecisionAccepted         = "accepted"
	AccessDecisionBypassed         = "bypassed"
	AccessDecisionReviewRequired   = "review_required"
	AccessDecisionCollecting       = "collecting"
	AccessDecisionRejected         = "rejected"
	AccessDecisionRevoked          = "revoked"
	AccessTriggerInformFirstSeen   = "inform_first_seen"
	AccessTriggerInformBoot        = "inform_boot"
	AccessTriggerInformReconnected = "inform_reconnected"
	AccessTriggerInformPeriodic    = "inform_periodic"
)

var ErrAccessAdmissionUnavailable = errors.New("device access admission unavailable")

type AccessObservation struct {
	Carrier            model.CarrierCode
	SerialNumber       string
	OUI                string
	ProductClass       string
	SoftwareVersion    string
	RemoteIP           string
	Authenticated      bool
	AuthMethod         string
	CredentialID       string
	DeviceCode         string
	CloudKey           string
	DeviceCodeRequired bool
	CloudKeyRequired   bool
	TriggerType        string
	InformEvent        string
	// CarrierIdentityResolved records whether the OUI/ProductClass pair was
	// explicitly mapped to Carrier. A deployment default is not identity proof.
	CarrierIdentityResolved bool
	Inform                  *tr069.InformMessage
	EventID                 string
}

// CandidateRFTarget is the deliberately restricted southbound identity used
// before a candidate is admitted into the formal device inventory. It is only
// valid for access-control containment (RF off plus correlated readback); it
// must never be used to enable RF or to run ordinary OMC tasks.
type CandidateRFTarget struct {
	CandidateID    uuid.UUID
	Carrier        model.CarrierCode
	SerialNumber   string
	ProductClass   string
	Technology     model.Technology
	RFControlPaths []string
}

type AccessDecision struct {
	State      string
	ReasonCode string
}

type AccessGate interface {
	Admit(ctx context.Context, observation AccessObservation) (AccessDecision, error)
}

func accessDecisionAllowsRegistration(decision AccessDecision) bool {
	return decision.State == AccessDecisionAccepted || decision.State == AccessDecisionBypassed
}
