package paramsync

import (
	"context"
	"testing"

	"github.com/omcgo/omcgo/internal/task"
)

type recordingPlannedTaskLifecycle struct {
	released         int
	releasedNoWake   int
	wokenDeviceCount map[string]int
}

func (r *recordingPlannedTaskLifecycle) ReleasePlannedTask(_ context.Context, _ *task.Task) (bool, error) {
	r.released++
	return true, nil
}

func (r *recordingPlannedTaskLifecycle) ReleasePlannedTaskWithoutWake(_ context.Context, _ *task.Task) (bool, error) {
	r.releasedNoWake++
	return true, nil
}

func (r *recordingPlannedTaskLifecycle) WakePlannedDevice(deviceSN string) {
	if r.wokenDeviceCount == nil {
		r.wokenDeviceCount = make(map[string]int)
	}
	r.wokenDeviceCount[deviceSN]++
}

func (r *recordingPlannedTaskLifecycle) EvictPlannedTask(_ context.Context, _ *task.Task) error {
	return nil
}

func TestPlannedTaskReleaseBatchCoalescesWakeByDevice(t *testing.T) {
	lifecycle := &recordingPlannedTaskLifecycle{}
	batch := newPlannedTaskReleaseBatch(lifecycle)
	ctx := context.Background()

	for i := 0; i < 13; i++ {
		if _, err := batch.Release(ctx, &task.Task{DeviceSN: "device-a"}); err != nil {
			t.Fatalf("release device-a task %d: %v", i, err)
		}
	}
	for i := 0; i < 2; i++ {
		if _, err := batch.Release(ctx, &task.Task{DeviceSN: "device-b"}); err != nil {
			t.Fatalf("release device-b task %d: %v", i, err)
		}
	}
	batch.WakeDevices()

	if lifecycle.released != 0 {
		t.Fatalf("ordinary release calls = %d, want 0", lifecycle.released)
	}
	if lifecycle.releasedNoWake != 15 {
		t.Fatalf("release-without-wake calls = %d, want 15", lifecycle.releasedNoWake)
	}
	if lifecycle.wokenDeviceCount["device-a"] != 1 || lifecycle.wokenDeviceCount["device-b"] != 1 {
		t.Fatalf("wake counts = %#v, want one per device", lifecycle.wokenDeviceCount)
	}
}

type legacyPlannedTaskLifecycle struct {
	released int
}

func (r *legacyPlannedTaskLifecycle) ReleasePlannedTask(_ context.Context, _ *task.Task) (bool, error) {
	r.released++
	return true, nil
}

func (r *legacyPlannedTaskLifecycle) EvictPlannedTask(_ context.Context, _ *task.Task) error {
	return nil
}

func TestPlannedTaskReleaseBatchFallsBackToOrdinaryLifecycle(t *testing.T) {
	lifecycle := &legacyPlannedTaskLifecycle{}
	batch := newPlannedTaskReleaseBatch(lifecycle)

	for i := 0; i < 3; i++ {
		if _, err := batch.Release(context.Background(), &task.Task{DeviceSN: "device-a"}); err != nil {
			t.Fatalf("release task %d: %v", i, err)
		}
	}
	batch.WakeDevices()

	if lifecycle.released != 3 {
		t.Fatalf("ordinary release calls = %d, want 3", lifecycle.released)
	}
}
