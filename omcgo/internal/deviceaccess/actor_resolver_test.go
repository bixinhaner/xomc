package deviceaccess

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/stretchr/testify/require"
)

type visibleGroupsResolverStub struct {
	groups []uuid.UUID
}

func (s visibleGroupsResolverStub) ResolveFromContext(*gin.Context) ([]uuid.UUID, error) {
	return s.groups, nil
}

type carrierVisibilityCheckerStub struct {
	allowed bool
	carrier string
	groups  []uuid.UUID
}

func (s *carrierVisibilityCheckerStub) CanAccessCarrier(_ context.Context, carrier string, groups []uuid.UUID) (bool, error) {
	s.carrier = carrier
	s.groups = groups
	return s.allowed, nil
}

func TestContextActorResolverUsesServerContextAndIgnoresBodyScope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/", nil)
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	c.Set("user_id", userID)
	c.Set("username", "system-admin")
	c.Set("operator_code", "cmcc")
	c.Request.Header.Set("X-Operator-Code", "ctcc")

	groupID := uuid.New()
	actor, err := NewContextActorResolver(
		visibleGroupsResolverStub{groups: []uuid.UUID{groupID}}, nil,
	).ResolveActor(c)

	require.NoError(t, err)
	require.Equal(t, PolicyActor{
		Carrier: "cmcc", SubjectID: userID.String(), Username: "system-admin", VisibleGroups: []uuid.UUID{groupID},
	}, actor)
}

func TestContextActorResolverRejectsMissingOperatorScope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Set("user_id", "user-1")

	_, err := (ContextActorResolver{}).ResolveActor(c)
	require.Error(t, err)
}

func TestContextActorResolverUsesExplicitCarrierForAuthorizedUser(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	c.Set("user_id", "user-1")
	c.Request.Header.Set("X-Operator-Code", " CMCC ")

	groupID := uuid.New()
	checker := &carrierVisibilityCheckerStub{allowed: true}
	resolver := NewContextActorResolver(visibleGroupsResolverStub{groups: []uuid.UUID{groupID}}, checker)
	actor, err := resolver.ResolveActor(c)

	require.NoError(t, err)
	require.Equal(t, "cmcc", actor.Carrier)
	require.Equal(t, []uuid.UUID{groupID}, actor.VisibleGroups)
	require.Equal(t, "cmcc", checker.carrier)
	require.Equal(t, []uuid.UUID{groupID}, checker.groups)
}

func TestContextActorResolverRejectsCarrierOutsideVisibleDeviceScope(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Set("user_id", "user-1")
	c.Request.Header.Set("X-Operator-Code", "ctcc")
	resolver := NewContextActorResolver(visibleGroupsResolverStub{groups: []uuid.UUID{uuid.New()}}, &carrierVisibilityCheckerStub{})

	_, err := resolver.ResolveActor(c)

	require.ErrorIs(t, err, commonerrors.ErrForbidden)
}

func TestContextActorResolverDoesNotTrustHeaderWithoutScopeDependencies(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Set("user_id", "user-1")
	c.Request.Header.Set("X-Operator-Code", "cmcc")

	_, err := (ContextActorResolver{}).ResolveActor(c)

	require.ErrorIs(t, err, ErrAccessGateDependencyMissing)
}

func TestContextActorResolverRejectsInvalidCarrierScope(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	c.Set("user_id", "user-1")
	c.Request.Header.Set("X-Operator-Code", "cmcc;drop")

	_, err := (ContextActorResolver{}).ResolveActor(c)

	require.Error(t, err)
}
