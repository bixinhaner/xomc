package ufte

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	coremodel "github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/software"
)

// fakeGroupReader 用固定的 设备ID → 分组ID 映射模拟 authz.GroupReader。
type fakeGroupReader struct {
	byDevice map[uuid.UUID][]uuid.UUID
}

func (f *fakeGroupReader) GetDeviceGroupIDs(_ context.Context, deviceID uuid.UUID) ([]uuid.UUID, error) {
	return f.byDevice[deviceID], nil
}

// stubSubTaskRepo 只实现 ListByTaskID，其余方法由内嵌接口兜底（测试不触达即可）。
type stubSubTaskRepo struct {
	software.SubTaskRepository
	byTask map[uuid.UUID][]software.UpgradeSubTaskWithTaskName
	err    error
}

func (s *stubSubTaskRepo) ListByTaskID(_ context.Context, taskID uuid.UUID, _ software.SubTaskFilter) (*coremodel.ListResponse[software.UpgradeSubTaskWithTaskName], error) {
	if s.err != nil {
		return nil, s.err
	}
	items := s.byTask[taskID]
	return &coremodel.ListResponse[software.UpgradeSubTaskWithTaskName]{Items: items, Total: int64(len(items))}, nil
}

func newAuthzService(reader *fakeGroupReader, subRepo software.SubTaskRepository) *Service {
	svc := NewService(nil, nil, nil, subRepo, nil, zap.NewNop())
	if reader != nil {
		svc.SetGroupReader(reader)
	}
	return svc
}

func TestAuthorizeDevices_CreateTaskScope(t *testing.T) {
	gVisible, gOther := uuid.New(), uuid.New()
	devIn, devOut := uuid.New(), uuid.New()
	reader := &fakeGroupReader{byDevice: map[uuid.UUID][]uuid.UUID{
		devIn:  {gVisible},
		devOut: {gOther},
	}}
	svc := newAuthzService(reader, nil)

	tests := []struct {
		name          string
		visibleGroups []uuid.UUID
		deviceIDs     []uuid.UUID
		wantErr       error
	}{
		{"超管 nil 放行任意设备", nil, []uuid.UUID{devIn, devOut}, nil},
		{"空可见组 fail-closed", []uuid.UUID{}, []uuid.UUID{devIn}, commonerrors.ErrForbidden},
		{"全部可见放行", []uuid.UUID{gVisible}, []uuid.UUID{devIn}, nil},
		{"混入域外设备整批拒绝", []uuid.UUID{gVisible}, []uuid.UUID{devIn, devOut}, commonerrors.ErrForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.authorizeDevices(context.Background(), tt.visibleGroups, tt.deviceIDs...)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("authorizeDevices() = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestAuthorizeDevices_NoReaderDegrades(t *testing.T) {
	// 未注入 groupReader（dev/test）→ 放行，即便 visibleGroups 非空。
	svc := newAuthzService(nil, nil)
	if err := svc.authorizeDevices(context.Background(), []uuid.UUID{uuid.New()}, uuid.New()); err != nil {
		t.Fatalf("nil reader should degrade to allow, got %v", err)
	}
}

func TestAuthorizeTaskDevices_ReverseLookup(t *testing.T) {
	gVisible, gOther := uuid.New(), uuid.New()
	devIn, devOut := uuid.New(), uuid.New()
	reader := &fakeGroupReader{byDevice: map[uuid.UUID][]uuid.UUID{
		devIn:  {gVisible},
		devOut: {gOther},
	}}

	taskVisible := uuid.New()
	taskMixed := uuid.New()
	taskEmpty := uuid.New()
	subRepo := &stubSubTaskRepo{byTask: map[uuid.UUID][]software.UpgradeSubTaskWithTaskName{
		taskVisible: {{UpgradeSubTask: software.UpgradeSubTask{DeviceID: devIn}}},
		taskMixed: {
			{UpgradeSubTask: software.UpgradeSubTask{DeviceID: devIn}},
			{UpgradeSubTask: software.UpgradeSubTask{DeviceID: devOut}},
		},
		taskEmpty: {},
	}}
	svc := newAuthzService(reader, subRepo)

	tests := []struct {
		name          string
		taskID        uuid.UUID
		visibleGroups []uuid.UUID
		wantErr       error
	}{
		{"超管 nil 放行不查 sub_tasks", taskMixed, nil, nil},
		{"任务全部设备可见放行", taskVisible, []uuid.UUID{gVisible}, nil},
		{"任务含域外设备拒绝", taskMixed, []uuid.UUID{gVisible}, commonerrors.ErrForbidden},
		{"空可见组对有设备任务 fail-closed", taskVisible, []uuid.UUID{}, commonerrors.ErrForbidden},
		{"空设备任务放行（无可越权对象）", taskEmpty, []uuid.UUID{gVisible}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.authorizeTaskDevices(context.Background(), tt.taskID, tt.visibleGroups)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("authorizeTaskDevices() = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestAuthorizeTaskDevices_RepoError(t *testing.T) {
	reader := &fakeGroupReader{byDevice: map[uuid.UUID][]uuid.UUID{}}
	subRepo := &stubSubTaskRepo{err: errors.New("db down")}
	svc := newAuthzService(reader, subRepo)
	err := svc.authorizeTaskDevices(context.Background(), uuid.New(), []uuid.UUID{uuid.New()})
	if err == nil {
		t.Fatal("expected error when sub_task repo fails")
	}
}
