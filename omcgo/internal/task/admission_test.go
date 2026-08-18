package task

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type admissionGuardStub struct {
	requests []TaskAdmissionRequest
	allowed  bool
}

type TaskAdmissionGuardFunc func(context.Context, TaskAdmissionRequest) (bool, string, error)

func (f TaskAdmissionGuardFunc) Allow(ctx context.Context, request TaskAdmissionRequest) (bool, string, error) {
	return f(ctx, request)
}

func (s *admissionGuardStub) Allow(_ context.Context, request TaskAdmissionRequest) (bool, string, error) {
	s.requests = append(s.requests, request)
	return s.allowed, "test_policy", nil
}

func TestCheckCreateAdmissionPassesCompleteTaskContract(t *testing.T) {
	guard := &admissionGuardStub{allowed: true}
	service := &TaskService{admissionGuard: guard}
	req := &CreateTaskRequest{
		DeviceSN: "SN-1", AdmissionClass: AdmissionClassAccessProbe,
		Source: TaskSourceDeviceAccess, Method: "GetParameterValues",
		SourceID: "c57b244b-f72b-4f57-b921-374344bbdb64",
		Params:   []byte(`{"names":["Device.FAP.GPS.LockedLatitude"]}`),
	}

	require.NoError(t, service.checkCreateAdmission(context.Background(), req))
	require.Len(t, guard.requests, 1)
	require.Equal(t, req.DeviceSN, guard.requests[0].DeviceSN)
	require.Equal(t, req.AdmissionClass, guard.requests[0].AdmissionClass)
	require.Equal(t, req.Source, guard.requests[0].Source)
	require.Equal(t, req.SourceID, guard.requests[0].SourceID)
	require.Equal(t, req.Method, guard.requests[0].Method)
	require.JSONEq(t, string(req.Params), string(guard.requests[0].Params))
}

func TestCheckCreateAdmissionRejectsDeniedTask(t *testing.T) {
	service := &TaskService{admissionGuard: &admissionGuardStub{allowed: false}}

	err := service.checkCreateAdmission(context.Background(), &CreateTaskRequest{
		DeviceSN: "SN-REVIEW", Method: "Reboot", AdmissionClass: AdmissionClassNormal,
	})

	require.ErrorIs(t, err, ErrTaskAdmissionDenied)
}
