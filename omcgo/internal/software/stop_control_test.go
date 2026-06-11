package software

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestTaskCancelRegistry_DeriveAndCancel(t *testing.T) {
	reg := newTaskCancelRegistry()
	taskID := uuid.New()

	ctx, _ := reg.derive(context.Background(), taskID)
	if ctx.Err() != nil {
		t.Fatalf("derived ctx should be live, got err=%v", ctx.Err())
	}

	if !reg.cancel(taskID) {
		t.Fatal("cancel should report a hit for a registered task")
	}
	if ctx.Err() == nil {
		t.Fatal("ctx should be canceled after cancel()")
	}
	// 幂等：第二次 cancel 不再命中。
	if reg.cancel(taskID) {
		t.Fatal("second cancel should not hit (already removed)")
	}
}

func TestTaskCancelRegistry_CancelUnknownIsNoop(t *testing.T) {
	reg := newTaskCancelRegistry()
	if reg.cancel(uuid.New()) {
		t.Fatal("cancel of unregistered task should be a no-op (no hit)")
	}
}

func TestTaskCancelRegistry_RemoveDoesNotCancel(t *testing.T) {
	reg := newTaskCancelRegistry()
	taskID := uuid.New()
	ctx, cancel := reg.derive(context.Background(), taskID)
	defer cancel()

	reg.remove(taskID)
	if ctx.Err() != nil {
		t.Fatal("remove must not cancel the ctx (cleanup only)")
	}
	// remove 后 cancel 不再命中注册表。
	if reg.cancel(taskID) {
		t.Fatal("cancel after remove should not hit")
	}
}

func TestTaskCancelRegistry_DeriveReplacesOld(t *testing.T) {
	reg := newTaskCancelRegistry()
	taskID := uuid.New()

	oldCtx, _ := reg.derive(context.Background(), taskID)
	// 同一 taskID 再次 derive：旧 ctx 应被取消，新 ctx 在册。
	newCtx, _ := reg.derive(context.Background(), taskID)

	if oldCtx.Err() == nil {
		t.Fatal("re-derive should cancel the old ctx for the same task")
	}
	if newCtx.Err() != nil {
		t.Fatal("the newly derived ctx should be live")
	}
	if !reg.cancel(taskID) {
		t.Fatal("the new ctx should be the one registered")
	}
}
