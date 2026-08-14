package alarm

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNormalizeAlarmRuleEmailRecipients(t *testing.T) {
	t.Run("normalizes and deduplicates email recipients", func(t *testing.T) {
		recipients, err := normalizeAlarmRuleEmailRecipients(FilterActionNotifyEmail, []string{
			" ops@example.com ",
			"OPS@example.com",
			"noc@example.com",
		})

		require.NoError(t, err)
		require.Equal(t, []string{"ops@example.com", "noc@example.com"}, recipients)
	})

	t.Run("requires recipients for notify email action", func(t *testing.T) {
		_, err := normalizeAlarmRuleEmailRecipients(FilterActionNotifyEmail, nil)

		require.ErrorIs(t, err, commonerrors.ErrInvalidInput)
	})

	t.Run("rejects invalid recipient", func(t *testing.T) {
		_, err := normalizeAlarmRuleEmailRecipients(FilterActionNotifyEmail, []string{"not-an-email"})

		require.ErrorIs(t, err, commonerrors.ErrInvalidInput)
	})

	t.Run("enforces recipient limit", func(t *testing.T) {
		recipients := make([]string, 0, maxAlarmEmailRecipients+1)
		for i := 0; i <= maxAlarmEmailRecipients; i++ {
			recipients = append(recipients, fmt.Sprintf("user%d@example.com", i))
		}

		_, err := normalizeAlarmRuleEmailRecipients(FilterActionNotifyEmail, recipients)

		require.ErrorIs(t, err, commonerrors.ErrInvalidInput)
	})

	t.Run("clears recipients for other actions", func(t *testing.T) {
		recipients, err := normalizeAlarmRuleEmailRecipients(FilterActionIgnore, []string{"ops@example.com"})

		require.NoError(t, err)
		require.Empty(t, recipients)
	})
}

func TestFilterHandlerCreate_PersistsNormalizedEmailRecipients(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var captured *AlarmFilterRule
	handler := NewFilterHandler(&stubAlarmFilterRuleRepository{
		createFn: func(_ context.Context, rule *AlarmFilterRule) error {
			captured = rule
			return nil
		},
	}, zap.NewNop())
	router := gin.New()
	router.POST("/rules", handler.Create)

	req := httptest.NewRequest(http.MethodPost, "/rules", strings.NewReader(
		`{"name":"email-rule","filter_type":"device","action":"notify_email","email_recipients":[" ops@example.com ","OPS@example.com"]}`,
	))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusCreated, resp.Code)
	require.NotNil(t, captured)
	require.Equal(t, []string{"ops@example.com"}, captured.EmailRecipients)
}

func TestFilterHandlerCreate_RejectsNotifyEmailWithoutRecipients(t *testing.T) {
	gin.SetMode(gin.TestMode)

	createCalled := false
	handler := NewFilterHandler(&stubAlarmFilterRuleRepository{
		createFn: func(_ context.Context, _ *AlarmFilterRule) error {
			createCalled = true
			return nil
		},
	}, zap.NewNop())
	router := gin.New()
	router.POST("/rules", handler.Create)

	req := httptest.NewRequest(http.MethodPost, "/rules", strings.NewReader(
		`{"name":"email-rule","filter_type":"device","action":"notify_email"}`,
	))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusBadRequest, resp.Code)
	require.False(t, createCalled)
}

func TestFilterHandlerUpdate_ClearsRecipientsWhenActionChanges(t *testing.T) {
	gin.SetMode(gin.TestMode)

	id := uuid.New()
	var captured *AlarmFilterRule
	handler := NewFilterHandler(&stubAlarmFilterRuleRepository{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*AlarmFilterRule, error) {
			return &AlarmFilterRule{
				ID:              id,
				Action:          FilterActionNotifyEmail,
				EmailRecipients: []string{"ops@example.com"},
			}, nil
		},
		updateFn: func(_ context.Context, rule *AlarmFilterRule) error {
			captured = rule
			return nil
		},
	}, zap.NewNop())
	router := gin.New()
	router.PUT("/rules/:id", handler.Update)

	req := httptest.NewRequest(http.MethodPut, "/rules/"+id.String(), strings.NewReader(`{"action":"ignore"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	require.NotNil(t, captured)
	require.Empty(t, captured.EmailRecipients)
}
