package device

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDedupeDeviceParameterRowsKeepsLastValuePerDeviceAndPath(t *testing.T) {
	firstDevice := uuid.New()
	secondDevice := uuid.New()
	rows := []deviceParameterUpsertRow{
		{deviceID: firstDevice, parameter: model.DeviceParameter{
			ParameterPath: "Device.DeviceInfo.UpTime", ParameterValue: "1",
		}},
		{deviceID: secondDevice, parameter: model.DeviceParameter{
			ParameterPath: "Device.DeviceInfo.UpTime", ParameterValue: "2",
		}},
		{deviceID: firstDevice, parameter: model.DeviceParameter{
			ParameterPath: "Device.DeviceInfo.UpTime", ParameterValue: "3",
		}},
	}

	got := dedupeDeviceParameterRows(rows)

	require.Len(t, got, 2)
	assert.Equal(t, firstDevice, got[0].deviceID)
	assert.Equal(t, "3", got[0].parameter.ParameterValue)
	assert.Equal(t, secondDevice, got[1].deviceID)
}

func TestBuildDeviceParameterUpsertUsesOneConditionalMultiRowStatement(t *testing.T) {
	rows := []deviceParameterUpsertRow{
		{deviceID: uuid.New(), parameter: model.DeviceParameter{
			ParameterPath:  "Device.FAPService.2.CellConfig.LTE.RAN.RF.EARFCNDL",
			ParameterValue: "100", ParameterType: model.ParameterType("unsignedInt"),
		}},
		{deviceID: uuid.New(), parameter: model.DeviceParameter{
			ParameterPath:  "Device.DeviceInfo.UpTime",
			ParameterValue: "10", ParameterType: model.ParameterType("unsignedInt"),
		}},
	}

	query, args, err := buildDeviceParameterUpsert(rows, time.Unix(100, 0))

	require.NoError(t, err)
	assert.Equal(t, 16, len(args))
	assert.Equal(t, 1, strings.Count(query, "),("))
	assert.Contains(t, query, "ON CONFLICT (device_id, parameter_path) DO UPDATE")
	assert.Contains(t, query, "device_parameters.parameter_value IS DISTINCT FROM EXCLUDED.parameter_value")
	assert.Contains(t, query, "RETURNING device_id")
	assert.Equal(t, 2, args[6])
	assert.Equal(t, "radio", args[7])
}

func TestOrderedDeviceParameterWriteIDsAreUniqueAndStable(t *testing.T) {
	first := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	second := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	rows := []deviceParameterUpsertRow{
		{deviceID: second},
		{deviceID: first},
		{deviceID: second},
	}

	assert.Equal(t, []uuid.UUID{first, second}, orderedDeviceParameterWriteIDs(rows))
}

func TestPartitionDeviceParameterRowsDefersOnlyContendedDevices(t *testing.T) {
	readyID := uuid.New()
	contendedID := uuid.New()
	rows := []deviceParameterUpsertRow{
		{deviceID: readyID, parameter: model.DeviceParameter{ParameterPath: "Device.Ready.1"}},
		{deviceID: contendedID, parameter: model.DeviceParameter{ParameterPath: "Device.Busy.1"}},
		{deviceID: readyID, parameter: model.DeviceParameter{ParameterPath: "Device.Ready.2"}},
		{deviceID: contendedID, parameter: model.DeviceParameter{ParameterPath: "Device.Busy.2"}},
	}

	ready, deferred := partitionDeviceParameterRows(rows, map[uuid.UUID]struct{}{readyID: {}})

	require.Len(t, ready, 2)
	assert.Equal(t, []string{"Device.Ready.1", "Device.Ready.2"}, []string{
		ready[0].parameter.ParameterPath,
		ready[1].parameter.ParameterPath,
	})
	require.Len(t, deferred, 1)
	require.Len(t, deferred[contendedID], 2)
	assert.Equal(t, "Device.Busy.1", deferred[contendedID][0].parameter.ParameterPath)
	assert.Equal(t, "Device.Busy.2", deferred[contendedID][1].parameter.ParameterPath)
}

func TestRunDeferredDeviceParameterWritesDoesNotHeadOfLineBlock(t *testing.T) {
	firstID := uuid.MustParse("10000000-0000-0000-0000-000000000001")
	secondID := uuid.MustParse("20000000-0000-0000-0000-000000000002")
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	secondDone := make(chan struct{})
	t.Cleanup(func() {
		select {
		case <-releaseFirst:
		default:
			close(releaseFirst)
		}
	})

	resultCh := make(chan struct {
		result deviceParameterUpsertResult
		err    error
	}, 1)
	go func() {
		result, err := runDeferredDeviceParameterWrites(
			[]uuid.UUID{firstID, secondID},
			2,
			func(deviceID uuid.UUID) (deviceParameterUpsertResult, error) {
				writeResult := deviceParameterUpsertResult{changedDevices: map[uuid.UUID]struct{}{deviceID: {}}}
				if deviceID == firstID {
					close(firstStarted)
					<-releaseFirst
					return writeResult, errors.New("first failed")
				}
				close(secondDone)
				writeResult.changed = 1
				return writeResult, nil
			},
		)
		resultCh <- struct {
			result deviceParameterUpsertResult
			err    error
		}{result: result, err: err}
	}()

	require.Eventually(t, func() bool {
		select {
		case <-firstStarted:
			return true
		default:
			return false
		}
	}, time.Second, time.Millisecond)
	require.Eventually(t, func() bool {
		select {
		case <-secondDone:
			return true
		default:
			return false
		}
	}, time.Second, time.Millisecond, "a later device must finish while the first device waits")
	close(releaseFirst)
	outcome := <-resultCh
	require.EqualError(t, outcome.err, "first failed")
	assert.Equal(t, 1, outcome.result.changed)
	assert.Contains(t, outcome.result.changedDevices, firstID)
	assert.Contains(t, outcome.result.changedDevices, secondID)
}
