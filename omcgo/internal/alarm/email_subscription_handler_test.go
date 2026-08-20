package alarm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/admin"
)

type fakeAlarmEmailVisibleGroups struct{ groups []uuid.UUID }

func (f fakeAlarmEmailVisibleGroups) GetUserVisibleGroupIDs(context.Context, uuid.UUID, bool) ([]uuid.UUID, error) {
	return f.groups, nil
}

type fakeAlarmEmailManagementRepository struct {
	setting AlarmEmailGlobalSetting
	created *AlarmEmailSubscription
	items   []AlarmEmailSubscription
}

func (f *fakeAlarmEmailManagementRepository) GetGlobalSetting(context.Context) (*AlarmEmailGlobalSetting, error) {
	setting := f.setting
	return &setting, nil
}

func (f *fakeAlarmEmailManagementRepository) UpdateGlobalSetting(_ context.Context, setting *AlarmEmailGlobalSetting) error {
	f.setting = *setting
	return nil
}

func (f *fakeAlarmEmailManagementRepository) ListSubscriptions(context.Context) ([]AlarmEmailSubscription, error) {
	return append([]AlarmEmailSubscription(nil), f.items...), nil
}

func (f *fakeAlarmEmailManagementRepository) GetSubscription(_ context.Context, id uuid.UUID) (*AlarmEmailSubscription, error) {
	for i := range f.items {
		if f.items[i].ID == id {
			item := f.items[i]
			return &item, nil
		}
	}
	return nil, ErrAlarmEmailSubscriptionName
}

func TestAlarmEmailSubscriptionHandlerRejectsEmptyScopeForRestrictedUser(t *testing.T) {
	repository := &fakeAlarmEmailManagementRepository{
		setting: AlarmEmailGlobalSetting{DefaultRecipients: []string{"default@example.com"}},
	}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(admin.CtxKeyUserID, uuid.New())
		c.Set(admin.CtxKeyIsSuperAdmin, false)
		c.Next()
	})
	handler := NewAlarmEmailSubscriptionHandler(repository, fakeAlarmEmailVisibleGroups{groups: []uuid.UUID{uuid.New()}}, nil)
	handler.RegisterRoutes(router.Group("/api/v1"))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/alarms/email-subscriptions", strings.NewReader(`{
		"name":"empty scope",
		"enabled":true,
		"interval_minutes":10,
		"tolerance_minutes":0,
		"recipients":["ops@example.com"]
	}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusForbidden, response.Code)
	require.Nil(t, repository.created)
}

func TestAlarmEmailSubscriptionHandlerListsOnlyOwnedSubscriptions(t *testing.T) {
	callerID := uuid.New()
	otherID := uuid.New()
	repository := &fakeAlarmEmailManagementRepository{items: []AlarmEmailSubscription{
		{ID: uuid.New(), Name: "mine", CreatedBy: &callerID},
		{ID: uuid.New(), Name: "other", CreatedBy: &otherID},
	}}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(admin.CtxKeyUserID, callerID)
		c.Set(admin.CtxKeyIsSuperAdmin, false)
		c.Next()
	})
	handler := NewAlarmEmailSubscriptionHandler(repository, fakeAlarmEmailVisibleGroups{groups: []uuid.UUID{uuid.New()}}, nil)
	handler.RegisterRoutes(router.Group("/api/v1"))
	response := httptest.NewRecorder()

	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/alarms/email-subscriptions", nil))

	require.Equal(t, http.StatusOK, response.Code)
	require.Contains(t, response.Body.String(), "mine")
	require.NotContains(t, response.Body.String(), "other")
}

func TestAlarmEmailSubscriptionHandlerRejectsGlobalUpdateForNonSuperAdmin(t *testing.T) {
	repository := &fakeAlarmEmailManagementRepository{}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(admin.CtxKeyUserID, uuid.New())
		c.Set(admin.CtxKeyIsSuperAdmin, false)
		c.Next()
	})
	handler := NewAlarmEmailSubscriptionHandler(repository, fakeAlarmEmailVisibleGroups{groups: []uuid.UUID{uuid.New()}}, nil)
	handler.RegisterRoutes(router.Group("/api/v1"))
	request := httptest.NewRequest(http.MethodPut, "/api/v1/alarms/email-settings", strings.NewReader(`{"enabled":true}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusForbidden, response.Code)
	require.False(t, repository.setting.Enabled)
}

func (f *fakeAlarmEmailManagementRepository) CreateSubscription(_ context.Context, subscription *AlarmEmailSubscription, defaultRecipients []string) error {
	if err := subscription.NormalizeAndValidate(defaultRecipients); err != nil {
		return err
	}
	copy := *subscription
	f.created = &copy
	return nil
}

func (f *fakeAlarmEmailManagementRepository) UpdateSubscription(context.Context, *AlarmEmailSubscription, []string) error {
	return nil
}

func (f *fakeAlarmEmailManagementRepository) DeleteSubscription(context.Context, uuid.UUID) error {
	return nil
}

func newAlarmEmailManagementRouter(repository AlarmEmailManagementRepository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewAlarmEmailSubscriptionHandler(repository, nil, nil)
	handler.RegisterRoutes(router.Group("/api/v1"))
	return router
}

func TestAlarmEmailSubscriptionHandlerRejectsEnabledSubscriptionWithoutRecipients(t *testing.T) {
	repository := &fakeAlarmEmailManagementRepository{}
	router := newAlarmEmailManagementRouter(repository)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/alarms/email-subscriptions", strings.NewReader(`{
		"name":"critical alarms",
		"enabled":true,
		"interval_minutes":10,
		"tolerance_minutes":0,
		"recipients":[],
		"include_default_recipients":true
	}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusBadRequest, response.Code)
	require.Nil(t, repository.created)
}

func TestAlarmEmailSubscriptionHandlerNormalizesDefaultRecipients(t *testing.T) {
	repository := &fakeAlarmEmailManagementRepository{}
	router := newAlarmEmailManagementRouter(repository)
	request := httptest.NewRequest(http.MethodPut, "/api/v1/alarms/email-settings", strings.NewReader(`{
		"enabled":true,
		"default_recipients":[" TEST@example.com ","test@example.com"]
	}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	require.True(t, repository.setting.Enabled)
	require.Equal(t, []string{"test@example.com"}, repository.setting.DefaultRecipients)
}

func TestAlarmEmailSubscriptionHandlerRejectsGroupOutsideCallerScope(t *testing.T) {
	repository := &fakeAlarmEmailManagementRepository{
		setting: AlarmEmailGlobalSetting{DefaultRecipients: []string{"default@example.com"}},
	}
	visibleGroupID := uuid.New()
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(admin.CtxKeyUserID, uuid.New())
		c.Next()
	})
	handler := NewAlarmEmailSubscriptionHandler(
		repository,
		fakeAlarmEmailVisibleGroups{groups: []uuid.UUID{visibleGroupID}},
		nil,
	)
	handler.RegisterRoutes(router.Group("/api/v1"))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/alarms/email-subscriptions", strings.NewReader(`{
		"name":"outside scope",
		"enabled":true,
		"interval_minutes":10,
		"tolerance_minutes":0,
		"recipients":[],
		"include_default_recipients":true,
		"device_group_ids":["`+uuid.NewString()+`"]
	}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusForbidden, response.Code)
	require.Nil(t, repository.created)
}
