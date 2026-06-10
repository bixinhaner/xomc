package software

import (
	"context"
	"sync"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	devtask "github.com/omcgo/omcgo/internal/task"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// recordingSubTaskRepo extends svcMockSubTaskRepo with status-update recording so
// tests can assert the verification-failure path persists the right failure code.
type recordingSubTaskRepo struct {
	svcMockSubTaskRepo
	mu       sync.Mutex
	statuses []recordedStatus
}

type recordedStatus struct {
	status UpgradeState
	msg    string
	code   FailureCode
}

func (r *recordingSubTaskRepo) UpdateStatus(_ context.Context, _ uuid.UUID, status UpgradeState, msg string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.statuses = append(r.statuses, recordedStatus{status: status, msg: msg})
	return nil
}

func (r *recordingSubTaskRepo) UpdateStatusWithCode(_ context.Context, _ uuid.UUID, status UpgradeState, msg string, code FailureCode) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.statuses = append(r.statuses, recordedStatus{status: status, msg: msg, code: code})
	return nil
}

func (r *recordingSubTaskRepo) lastFailure() (recordedStatus, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := len(r.statuses) - 1; i >= 0; i-- {
		if r.statuses[i].status == UpgradeFailed {
			return r.statuses[i], true
		}
	}
	return recordedStatus{}, false
}

func newVerifyExecutor(t *testing.T, subRepo SubTaskRepository, cmdQueue devtask.Enqueuer, deviceSN string) *UpgradeExecutor {
	t.Helper()
	mr := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })

	deviceRepo := &svcMockDeviceRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Device, error) {
			return &model.Device{ID: id, SerialNumber: deviceSN, Status: model.DeviceActive, FirmwareVersion: "V1.0.0"}, nil
		},
	}
	return NewUpgradeExecutor(
		&svcMockTaskRepo{}, subRepo, deviceRepo, &svcMockFirmwareRepo{},
		cmdQueue, nil, redisClient, &svcMockEventBus{}, zap.NewNop(),
	)
}

// TestExecuteOne_AbortsWhenNoIntegrityDigest is the core failure-path test:
// a firmware with neither sha256 nor md5 must NOT dispatch a Download command and
// must mark the sub-task failed with INTEGRITY_CHECK_FAILED.
func TestExecuteOne_AbortsWhenNoIntegrityDigest(t *testing.T) {
	subRepo := &recordingSubTaskRepo{}
	var dispatched bool
	cmdQueue := &svcMockCmdQueue{
		createFn: func(_ context.Context, req *devtask.CreateTaskRequest) (*devtask.Task, error) {
			dispatched = true
			return devtask.NewTask(req), nil
		},
	}
	exec := newVerifyExecutor(t, subRepo, cmdQueue, "SN-NO-DIGEST")

	exec.ExecuteOne(context.Background(), &UpgradeSubTask{
		ID: uuid.New(), TaskID: uuid.New(), DeviceID: uuid.New(),
	}, &FirmwareVersion{
		ID: uuid.New(), Version: "V2.0.0", FileName: "fw.bin", MinIOPath: "p/fw.bin",
		// no MD5Val, no SHA256Val → must be rejected
	}, false, "")

	assert.False(t, dispatched, "Download command must NOT be dispatched when integrity verification fails")
	failure, ok := subRepo.lastFailure()
	require.True(t, ok, "sub-task must be marked failed")
	assert.Equal(t, FailureIntegrityCheck, failure.code)
	assert.Contains(t, failure.msg, "firmware verification failed")
}

// TestExecuteOne_AbortsWhenSignatureRequiredButUnsigned verifies the config-gated
// signature enforcement path aborts dispatch for unsigned firmware.
func TestExecuteOne_AbortsWhenSignatureRequiredButUnsigned(t *testing.T) {
	subRepo := &recordingSubTaskRepo{}
	var dispatched bool
	cmdQueue := &svcMockCmdQueue{
		createFn: func(_ context.Context, req *devtask.CreateTaskRequest) (*devtask.Task, error) {
			dispatched = true
			return devtask.NewTask(req), nil
		},
	}
	exec := newVerifyExecutor(t, subRepo, cmdQueue, "SN-UNSIGNED")
	// Enforce signatures: unsigned firmware (even with a valid sha256) must abort.
	exec.SetFirmwareVerifier(NewSignatureVerifier(NewHashOnlyVerifier(), true, nil))

	exec.ExecuteOne(context.Background(), &UpgradeSubTask{
		ID: uuid.New(), TaskID: uuid.New(), DeviceID: uuid.New(),
	}, &FirmwareVersion{
		ID: uuid.New(), Version: "V2.0.0", FileName: "fw.bin", MinIOPath: "p/fw.bin",
		SHA256Val: "deadbeefdeadbeef", // good integrity, but unsigned
	}, false, "")

	assert.False(t, dispatched, "Download must NOT dispatch unsigned firmware when signature enforcement is on")
	failure, ok := subRepo.lastFailure()
	require.True(t, ok)
	assert.Equal(t, FailureSignatureCheck, failure.code)
}

// TestExecuteOne_DispatchesWhenSha256Present is the success-path test: a firmware
// with a sha256 digest passes verification and dispatches the Download command.
func TestExecuteOne_DispatchesWhenSha256Present(t *testing.T) {
	subRepo := &recordingSubTaskRepo{}
	var dispatched bool
	cmdQueue := &svcMockCmdQueue{
		createFn: func(_ context.Context, req *devtask.CreateTaskRequest) (*devtask.Task, error) {
			dispatched = true
			assert.Equal(t, "Download", req.Method)
			return devtask.NewTask(req), nil
		},
	}
	exec := newVerifyExecutor(t, subRepo, cmdQueue, "SN-SHA256-OK")

	exec.ExecuteOne(context.Background(), &UpgradeSubTask{
		ID: uuid.New(), TaskID: uuid.New(), DeviceID: uuid.New(),
	}, &FirmwareVersion{
		ID: uuid.New(), Version: "V2.0.0", FileName: "fw.bin", MinIOPath: "p/fw.bin",
		SHA256Val: "deadbeefdeadbeef",
	}, false, "")

	assert.True(t, dispatched, "Download command must dispatch when sha256 integrity verification passes")
	_, failed := subRepo.lastFailure()
	assert.False(t, failed, "sub-task must not be failed on the success path")
}

// TestExecuteOne_DispatchesLegacyMD5Only confirms backward compatibility: a legacy
// firmware with only md5 still dispatches (does not break the happy-path).
func TestExecuteOne_DispatchesLegacyMD5Only(t *testing.T) {
	subRepo := &recordingSubTaskRepo{}
	var dispatched bool
	cmdQueue := &svcMockCmdQueue{
		createFn: func(_ context.Context, req *devtask.CreateTaskRequest) (*devtask.Task, error) {
			dispatched = true
			return devtask.NewTask(req), nil
		},
	}
	exec := newVerifyExecutor(t, subRepo, cmdQueue, "SN-LEGACY-MD5")

	exec.ExecuteOne(context.Background(), &UpgradeSubTask{
		ID: uuid.New(), TaskID: uuid.New(), DeviceID: uuid.New(),
	}, &FirmwareVersion{
		ID: uuid.New(), Version: "V2.0.0", FileName: "fw.bin", MinIOPath: "p/fw.bin",
		MD5Val: "abc123", // legacy: md5-only
	}, false, "")

	assert.True(t, dispatched, "legacy md5-only firmware must still dispatch (backward compat)")
	_, failed := subRepo.lastFailure()
	assert.False(t, failed)
}
