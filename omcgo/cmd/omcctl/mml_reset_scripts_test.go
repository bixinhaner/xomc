package main

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/task"
)

func TestMMLResetScripts_ApplyRequiresConfirmation(t *testing.T) {
	err := runMMLResetScripts("127.0.0.1:1", false, true, "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "DELETE-MML-RUNTIME")
}

func TestMMLResetScripts_DryRunLeavesNonMMLTasks(t *testing.T) {
	m := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: m.Addr()})
	q := task.NewRedisTaskQueue(client)
	ctx := context.Background()

	mml := &task.Task{ID: "m1", DeviceSN: "SN1", Source: task.TaskSourceMML}
	api := &task.Task{ID: "a1", DeviceSN: "SN1", Source: task.TaskSourceAPI}
	require.NoError(t, q.Push(ctx, mml))
	require.NoError(t, q.Push(ctx, api))

	result, err := executeMMLResetScripts(m.Addr(), true, false)
	require.NoError(t, err)
	require.Equal(t, int64(1), result.Matched)
	require.Equal(t, int64(0), result.Deleted)
	remaining, err := q.GetByID(ctx, "m1")
	require.NoError(t, err)
	require.NotNil(t, remaining)
	remaining, err = q.GetByID(ctx, "a1")
	require.NoError(t, err)
	require.NotNil(t, remaining)
}

func TestMMLResetScripts_ApplyDeletesOnlyMMLTasks(t *testing.T) {
	m := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: m.Addr()})
	q := task.NewRedisTaskQueue(client)
	ctx := context.Background()

	mml := &task.Task{ID: "m-apply", DeviceSN: "SN1", Source: task.TaskSourceMML}
	api := &task.Task{ID: "a-apply", DeviceSN: "SN1", Source: task.TaskSourceAPI}
	require.NoError(t, q.Push(ctx, mml))
	require.NoError(t, q.Push(ctx, api))

	result, err := executeMMLResetScripts(m.Addr(), false, true)
	require.NoError(t, err)
	require.Equal(t, int64(1), result.Matched)
	require.Equal(t, int64(1), result.Deleted)
	deleted, err := q.GetByID(ctx, mml.ID)
	require.NoError(t, err)
	require.Nil(t, deleted)
	remaining, err := q.GetByID(ctx, api.ID)
	require.NoError(t, err)
	require.NotNil(t, remaining)
}
