package parammodel

import (
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateStandardParamSQLRejectsDuplicateInsteadOfUpserting(t *testing.T) {
	upperSQL := strings.ToUpper(createStandardParamSQL)
	require.Contains(t, upperSQL, "INSERT INTO STANDARD_PARAMS")
	assert.NotContains(t, upperSQL, "ON CONFLICT")
	assert.NotContains(t, upperSQL, "DO UPDATE")
}

func TestIsStandardPathUniqueViolation(t *testing.T) {
	duplicatePath := &pgconn.PgError{
		Code:           "23505",
		ConstraintName: "standard_params_standard_path_key",
	}
	assert.True(t, isStandardPathUniqueViolation(duplicatePath))
	assert.True(t, isStandardPathUniqueViolation(errors.Join(errors.New("insert failed"), duplicatePath)))
	assert.False(t, isStandardPathUniqueViolation(&pgconn.PgError{Code: "23505", ConstraintName: "other_key"}))
}

func TestUpdateStandardParamSQLTracksChangedFields(t *testing.T) {
	upperSQL := strings.ToUpper(updateStandardParamSQL)
	require.Contains(t, upperSQL, "UPDATE STANDARD_PARAMS")
	assert.Contains(t, upperSQL, "UPDATED_FIELDS")
	assert.Contains(t, upperSQL, "IS DISTINCT FROM")
	assert.NotContains(t, upperSQL, "INSERT INTO")
}
