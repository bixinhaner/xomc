package stationlog

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

func TestPgRepository_CreateRejectsFaultFileWithoutDeviceSN(t *testing.T) {
	repo := NewPgFaultRepository(nil)

	err := repo.Create(context.Background(), &LogFile{
		FileName:   "ErrorLog_20260715.1539 0800_dieLog.tar.gz",
		ObjectPath: "fault/2026/07/15/dbc91d19/ErrorLog_20260715.1539 0800_dieLog.tar.gz",
		Bucket:     "logs",
		FileSize:   1317251,
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, commonerrors.ErrInvalidInput)
}
