package device

import (
	"context"
	"errors"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/pkg/tr069"
)

const (
	AccessDecisionAccepted       = "accepted"
	AccessDecisionBypassed       = "bypassed"
	AccessDecisionReviewRequired = "review_required"
	AccessDecisionCollecting     = "collecting"
	AccessDecisionRejected       = "rejected"
	AccessDecisionRevoked        = "revoked"
)

var ErrAccessAdmissionUnavailable = errors.New("device access admission unavailable")

type AccessObservation struct {
	Carrier         model.CarrierCode
	SerialNumber    string
	OUI             string
	ProductClass    string
	SoftwareVersion string
	RemoteIP        string
	Authenticated   bool
	AuthMethod      string
	CredentialID    string
	// CarrierIdentityResolved records whether the OUI/ProductClass pair was
	// explicitly mapped to Carrier. A deployment default is not identity proof.
	CarrierIdentityResolved bool
	Inform                  *tr069.InformMessage
	EventID                 string
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
