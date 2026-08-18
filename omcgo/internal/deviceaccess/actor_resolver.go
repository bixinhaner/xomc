package deviceaccess

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

// ContextActorResolver reads the authenticated identity placed by the existing
// admin middleware. Carrier is intentionally not taken from JSON input.
type VisibleGroupsResolver interface {
	ResolveFromContext(*gin.Context) ([]uuid.UUID, error)
}

type CarrierVisibilityChecker interface {
	CanAccessCarrier(context.Context, string, []uuid.UUID) (bool, error)
}

type ContextActorResolver struct {
	groups   VisibleGroupsResolver
	carriers CarrierVisibilityChecker
}

func NewContextActorResolver(groups VisibleGroupsResolver, carriers CarrierVisibilityChecker) ContextActorResolver {
	return ContextActorResolver{groups: groups, carriers: carriers}
}

var carrierScopePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,15}$`)

func (r ContextActorResolver) ResolveActor(c *gin.Context) (PolicyActor, error) {
	if c == nil {
		return PolicyActor{}, errors.New("request context is required")
	}
	subjectID := contextString(c, "user_id")
	if subjectID == "" {
		subjectID = contextString(c, "username")
	}
	if subjectID == "" {
		return PolicyActor{}, errors.New("authenticated user is required")
	}
	carrier := contextString(c, "operator_code")
	trustedCarrier := carrier != ""
	if !trustedCarrier {
		carrier = strings.TrimSpace(c.GetHeader("X-Operator-Code"))
	}
	if carrier == "" {
		return PolicyActor{}, errors.New("operator scope is required")
	}
	carrier = strings.ToLower(carrier)
	if !carrierScopePattern.MatchString(carrier) {
		return PolicyActor{}, fmt.Errorf("operator scope %q is invalid", carrier)
	}
	var visibleGroups []uuid.UUID
	if r.groups != nil {
		var err error
		visibleGroups, err = r.groups.ResolveFromContext(c)
		if err != nil {
			return PolicyActor{}, fmt.Errorf("resolve operator data scope: %w", err)
		}
	}
	if !trustedCarrier {
		if r.groups == nil || r.carriers == nil {
			return PolicyActor{}, fmt.Errorf("authorize operator scope: %w", ErrAccessGateDependencyMissing)
		}
		allowed, err := r.carriers.CanAccessCarrier(c.Request.Context(), carrier, visibleGroups)
		if err != nil {
			return PolicyActor{}, fmt.Errorf("authorize operator scope: %w", err)
		}
		if !allowed {
			return PolicyActor{}, commonerrors.ErrForbidden
		}
	}
	return PolicyActor{Carrier: carrier, SubjectID: subjectID, VisibleGroups: visibleGroups}, nil
}

func contextString(c *gin.Context, key string) string {
	value, ok := c.Get(key)
	if !ok {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case uuid.UUID:
		if typed != uuid.Nil {
			return typed.String()
		}
	case *uuid.UUID:
		if typed != nil && *typed != uuid.Nil {
			return typed.String()
		}
	}
	if text, ok := value.(interface{ String() string }); ok {
		return strings.TrimSpace(text.String())
	}
	return ""
}

var _ ActorResolver = ContextActorResolver{}
