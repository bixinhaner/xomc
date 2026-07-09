package agentruntime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/admin"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

const (
	conversationCategory     = "agent_runtime"
	instanceConfigKey        = "instance_id"
	conversationConfigPrefix = "active_conversation."
	instanceIDPrefix         = "omcinst_"
	conversationIDPrefix     = "omcconv_"
)

type conversationConfigStore interface {
	GetByKey(ctx context.Context, category, key string) (*admin.SysConfig, error)
	BatchUpsert(ctx context.Context, category string, items []admin.BatchItem) (int, error)
}

type ConversationService struct {
	store conversationConfigStore
}

type ConversationResponse struct {
	ConversationID string `json:"conversationId"`
}

func NewConversationService(store conversationConfigStore) *ConversationService {
	return &ConversationService{store: store}
}

func (s *ConversationService) Active(ctx context.Context, connectorID string, claims *admin.Claims) (string, error) {
	return s.ensure(ctx, connectorID, claims, false)
}

func (s *ConversationService) Rotate(ctx context.Context, connectorID string, claims *admin.Claims) (string, error) {
	return s.ensure(ctx, connectorID, claims, true)
}

func (s *ConversationService) ensure(ctx context.Context, connectorID string, claims *admin.Claims, rotate bool) (string, error) {
	if s == nil || s.store == nil {
		return "", commonerrors.ErrUnavailable
	}
	userID, err := conversationUserID(claims)
	if err != nil {
		return "", err
	}
	connectorID = strings.TrimSpace(connectorID)
	if connectorID == "" {
		return "", fmt.Errorf("%w: missing agent connector id", commonerrors.ErrInvalidInput)
	}

	instanceID, found, err := s.getValue(ctx, instanceConfigKey)
	if err != nil {
		return "", err
	}

	upserts := make([]admin.BatchItem, 0, 2)
	if !found || !isAgentInstanceID(instanceID) {
		instanceID = newAgentInstanceID()
		upserts = append(upserts, admin.BatchItem{Key: instanceConfigKey, Value: instanceID, ValueType: "string"})
	}

	conversationKey := conversationConfigKey(connectorID, userID)
	if !rotate {
		conversationID, found, err := s.getValue(ctx, conversationKey)
		if err != nil {
			return "", err
		}
		if found && isAgentConversationID(conversationID) {
			if len(upserts) > 0 {
				if _, err := s.store.BatchUpsert(ctx, conversationCategory, upserts); err != nil {
					return "", fmt.Errorf("save agent instance config: %w", err)
				}
			}
			return conversationID, nil
		}
	}

	conversationID := newAgentConversationID(instanceID)
	upserts = append(upserts, admin.BatchItem{Key: conversationKey, Value: conversationID, ValueType: "string"})
	if _, err := s.store.BatchUpsert(ctx, conversationCategory, upserts); err != nil {
		return "", fmt.Errorf("save agent conversation config: %w", err)
	}
	return conversationID, nil
}

func (s *ConversationService) getValue(ctx context.Context, key string) (string, bool, error) {
	item, err := s.store.GetByKey(ctx, conversationCategory, key)
	if err != nil {
		if errors.Is(err, commonerrors.ErrNotFound) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("read agent conversation config %s: %w", key, err)
	}
	if item == nil {
		return "", false, nil
	}
	return strings.TrimSpace(item.Value), true, nil
}

func conversationUserID(claims *admin.Claims) (string, error) {
	if claims == nil || claims.UserID == uuid.Nil {
		return "", commonerrors.ErrUnauthorized
	}
	return claims.UserID.String(), nil
}

func conversationConfigKey(connectorID, userID string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(connectorID) + "\x00" + strings.TrimSpace(userID)))
	return conversationConfigPrefix + hex.EncodeToString(sum[:16])
}

func newAgentInstanceID() string {
	return instanceIDPrefix + strings.ReplaceAll(uuid.NewString(), "-", "")
}

func newAgentConversationID(instanceID string) string {
	instanceID = strings.TrimPrefix(strings.TrimSpace(instanceID), instanceIDPrefix)
	return conversationIDPrefix + instanceID + "_" + strings.ReplaceAll(uuid.NewString(), "-", "")
}

func isAgentInstanceID(value string) bool {
	return strings.HasPrefix(value, instanceIDPrefix) && len(value) > len(instanceIDPrefix)
}

func isAgentConversationID(value string) bool {
	return strings.HasPrefix(value, conversationIDPrefix) && len(value) > len(conversationIDPrefix)
}
