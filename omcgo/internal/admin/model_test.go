package admin

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func Test_UserStatus_Constants(t *testing.T) {
	assert.Equal(t, UserStatus("active"), UserStatusActive)
	assert.Equal(t, UserStatus("disabled"), UserStatusDisabled)
	assert.NotEqual(t, UserStatusActive, UserStatusDisabled)
}

func Test_AuditLog_Construction(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name string
		log  AuditLog
		// field checks
		wantUserID   *uuid.UUID
		wantAction   string
		wantResource string
	}{
		{
			name: "full audit log",
			log: AuditLog{
				ID:         uuid.New(),
				UserID:     &userID,
				Username:   "admin",
				Action:     "login",
				Resource:   "session",
				ResourceID: "sess-123",
				Details:    map[string]interface{}{"ip": "10.0.0.1"},
				IPAddress:  "10.0.0.1",
				UserAgent:  "Mozilla/5.0",
				CreatedAt:  time.Now(),
			},
			wantUserID:   &userID,
			wantAction:   "login",
			wantResource: "session",
		},
		{
			name: "audit log without user (system action)",
			log: AuditLog{
				ID:       uuid.New(),
				UserID:   nil,
				Username: "system",
				Action:   "auto_backup",
			},
			wantUserID:   nil,
			wantAction:   "auto_backup",
			wantResource: "",
		},
		{
			name: "audit log with empty details",
			log: AuditLog{
				ID:       uuid.New(),
				UserID:   &userID,
				Username: "operator",
				Action:   "view",
				Resource: "device",
				Details:  nil,
			},
			wantUserID:   &userID,
			wantAction:   "view",
			wantResource: "device",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantUserID, tt.log.UserID)
			assert.Equal(t, tt.wantAction, tt.log.Action)
			assert.Equal(t, tt.wantResource, tt.log.Resource)
		})
	}
}

func Test_Claims_Construction(t *testing.T) {
	userID := uuid.New()

	claims := Claims{
		UserID:       userID,
		Username:     "testuser",
		IsSuperAdmin: false,
		Roles:        []string{"admin", "operator"},
	}

	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, "testuser", claims.Username)
	assert.False(t, claims.IsSuperAdmin)
	assert.Len(t, claims.Roles, 2)
	assert.Contains(t, claims.Roles, "admin")
}

func Test_TokenPair_Fields(t *testing.T) {
	expiry := time.Now().Add(30 * time.Minute)
	tp := TokenPair{
		AccessToken:  "access-abc",
		RefreshToken: "refresh-xyz",
		ExpiresAt:    expiry,
		TokenType:    "Bearer",
	}

	assert.Equal(t, "access-abc", tp.AccessToken)
	assert.Equal(t, "refresh-xyz", tp.RefreshToken)
	assert.Equal(t, "Bearer", tp.TokenType)
	assert.False(t, tp.ExpiresAt.IsZero())
}

func Test_Permission_Fields(t *testing.T) {
	roleID := uuid.New()
	perm := Permission{
		ID:       uuid.New(),
		RoleID:   roleID,
		Resource: "device",
		Action:   "write",
	}

	assert.Equal(t, roleID, perm.RoleID)
	assert.Equal(t, "device", perm.Resource)
	assert.Equal(t, "write", perm.Action)
}

func Test_Role_SystemFlag(t *testing.T) {
	systemRole := Role{
		ID:       uuid.New(),
		Name:     "super_admin",
		IsSystem: true,
	}
	customRole := Role{
		ID:       uuid.New(),
		Name:     "custom_role",
		IsSystem: false,
	}

	assert.True(t, systemRole.IsSystem)
	assert.False(t, customRole.IsSystem)
}

func Test_NullableString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  interface{}
	}{
		{"empty returns nil", "", nil},
		{"non-empty returns string", "hello", "hello"},
		{"whitespace returns string", " ", " "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := nullableString(tt.input)
			assert.Equal(t, tt.want, result)
		})
	}
}

// Test_NullableCarrier removed in v1.0: nullableCarrier helper was deleted
// alongside users.carrier column. See PRD §11.11.

func Test_NullableTime(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		assert.Nil(t, nullableTime(nil))
	})

	t.Run("non-nil returns time value", func(t *testing.T) {
		now := time.Now()
		result := nullableTime(&now)
		assert.Equal(t, now, result)
	})
}
