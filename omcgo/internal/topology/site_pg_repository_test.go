package topology

import (
	gerrors "errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

// TestClassifyPgError covers the SQLSTATE → sentinel-error mapping used by the
// site repository so that handler-layer HTTPStatusFromError returns the right
// HTTP status (400/409) instead of the default 500.
func TestClassifyPgError(t *testing.T) {
	t.Parallel()

	otherErr := gerrors.New("some non-pg error")

	cases := []struct {
		name        string
		input       error
		wantSent    error // sentinel that errors.Is should match (nil = exact-equal expected)
		wantSame    bool  // true means the function must return the input unchanged
		wantNil     bool
	}{
		{
			name:    "nil input returns nil",
			input:   nil,
			wantNil: true,
		},
		{
			name:     "non-pg error is returned untouched",
			input:    otherErr,
			wantSame: true,
		},
		{
			name:     "unique violation maps to ErrAlreadyExists",
			input:    &pgconn.PgError{Code: pgUniqueViolation, Message: "duplicate key"},
			wantSent: commonerrors.ErrAlreadyExists,
		},
		{
			name:     "foreign key violation maps to ErrInvalidInput",
			input:    &pgconn.PgError{Code: pgForeignKeyViolation, Message: "fk violation"},
			wantSent: commonerrors.ErrInvalidInput,
		},
		{
			name:     "not null violation maps to ErrInvalidInput",
			input:    &pgconn.PgError{Code: pgNotNullViolation, Message: "null in NOT NULL column"},
			wantSent: commonerrors.ErrInvalidInput,
		},
		{
			name:     "check violation maps to ErrInvalidInput",
			input:    &pgconn.PgError{Code: pgCheckViolation, Message: "check failed"},
			wantSent: commonerrors.ErrInvalidInput,
		},
		{
			name:     "unknown SQLSTATE returned unchanged",
			input:    &pgconn.PgError{Code: "42P01", Message: "undefined_table"},
			wantSame: true,
		},
		{
			name:     "wrapped fk violation still classified",
			input:    fmt.Errorf("insert site: %w", &pgconn.PgError{Code: pgForeignKeyViolation, Message: "wrapped"}),
			wantSent: commonerrors.ErrInvalidInput,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := classifyPgError(tc.input)

			switch {
			case tc.wantNil:
				assert.NoError(t, got)
			case tc.wantSame:
				assert.Equal(t, tc.input, got, "expected the original error returned unchanged")
			default:
				assert.True(t, gerrors.Is(got, tc.wantSent),
					"want sentinel %v wrapped, got %v", tc.wantSent, got)
				// And HTTPStatusFromError must map this to a non-500 code.
				status := commonerrors.HTTPStatusFromError(got)
				assert.NotEqual(t, 500, status, "constraint violation must not map to 500")
			}
		})
	}
}
