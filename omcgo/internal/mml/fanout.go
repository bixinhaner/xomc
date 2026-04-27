package mml

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/task"
)

// DeviceTaskCreator creates device_tasks from an MML task.
// Implemented by task.TaskService; defined here to avoid circular dependency.
type DeviceTaskCreator interface {
	BatchCreateTasks(ctx context.Context, reqs []*task.CreateTaskRequest) ([]*task.Task, error)
}

// Fanouter fans out an MML task into individual device_tasks.
type Fanouter struct {
	taskCreator DeviceTaskCreator
	logger      *zap.Logger
}

// NewFanouter creates a new Fanouter.
func NewFanouter(taskCreator DeviceTaskCreator, logger *zap.Logger) *Fanouter {
	return &Fanouter{
		taskCreator: taskCreator,
		logger:      logger.Named("mml-fanout"),
	}
}

// Fanout creates device_tasks for each (command, device) pair in the MML task.
// Only called for immediate execution; scheduled/periodic tasks are fan-outed when started.
func (f *Fanouter) Fanout(ctx context.Context, mmlTask *MMLTask) (int, error) {
	if len(mmlTask.DeviceSNs) == 0 || len(mmlTask.Commands) == 0 {
		return 0, nil
	}

	reqs := f.buildDeviceTaskRequests(mmlTask)
	if len(reqs) == 0 {
		return 0, nil
	}

	created, err := f.taskCreator.BatchCreateTasks(ctx, reqs)
	if err != nil {
		return 0, fmt.Errorf("fanout mml task %s: %w", mmlTask.ID, err)
	}

	f.logger.Info("mml task fan-out completed",
		zap.String("mml_task_id", mmlTask.ID.String()),
		zap.Int("device_count", len(mmlTask.DeviceSNs)),
		zap.Int("command_count", len(mmlTask.Commands)),
		zap.Int("device_tasks_created", len(created)),
	)

	return len(created), nil
}

// buildDeviceTaskRequests converts an MML task into individual device task creation requests.
//
// 每条 command 经 BuildTR069Params 翻译为 TR-069 wire 格式（{"names":[...]} /
// {"values":[...]} 等），写入 device_tasks.params。翻译失败的 command 会被跳过
// 并打 Warn 日志（payload 不合规会让 ACS 在 BuildRequest 阶段就报错，不如这里
// 直接拒绝下发）。每条成功翻译的 command 还会打 Info 级 schema 摘要，方便排查
// "device_tasks 下发了什么形态的报文"。
func (f *Fanouter) buildDeviceTaskRequests(mmlTask *MMLTask) []*task.CreateTaskRequest {
	var reqs []*task.CreateTaskRequest
	parentID := mmlTask.ID.String()

	for cmdIdx, cmd := range mmlTask.Commands {
		rpcMethod, _ := cmd["rpc_method"].(string)
		if rpcMethod == "" {
			f.logger.Warn("skip command without rpc_method",
				zap.String("mml_task_id", parentID),
				zap.Int("cmd_idx", cmdIdx),
				zap.Any("command_code", cmd["command_code"]),
			)
			continue
		}

		paramRefs := paramRefsFromEntry(cmd)
		formValues, _ := cmd["parameters"].(map[string]interface{})
		operationType, _ := cmd["operation_type"].(string)
		commandCode, _ := cmd["command_code"].(string)

		params, err := BuildTR069Params(rpcMethod, paramRefs, formValues, operationType)
		if err != nil {
			f.logger.Warn("build tr069 params failed, skip command",
				zap.String("mml_task_id", parentID),
				zap.Int("cmd_idx", cmdIdx),
				zap.String("command_code", commandCode),
				zap.String("rpc_method", rpcMethod),
				zap.String("operation_type", operationType),
				zap.Int("param_refs_count", len(paramRefs)),
				zap.Int("form_values_count", len(formValues)),
				zap.Error(err),
			)
			continue
		}

		summary := SummarizeSchema(params)
		f.logger.Info("device_task params built",
			zap.String("mml_task_id", parentID),
			zap.Int("cmd_idx", cmdIdx),
			zap.String("command_code", commandCode),
			zap.String("rpc_method", rpcMethod),
			zap.String("operation_type", operationType),
			zap.Int("payload_size", summary.PayloadSize),
			zap.Bool("has_names", summary.HasNames),
			zap.Int("names_count", summary.NamesCount),
			zap.Bool("has_values", summary.HasValues),
			zap.Int("values_count", summary.ValuesCount),
			zap.Bool("has_object_name", summary.HasObjectName),
			zap.Bool("has_attributes", summary.HasAttributes),
			zap.Bool("has_path", summary.HasPath),
		)

		description := fmt.Sprintf("MML %s", commandCode)
		if mmlTask.TaskName != "" {
			description = fmt.Sprintf("MML %s: %s", commandCode, mmlTask.TaskName)
		}

		for devIdx, sn := range mmlTask.DeviceSNs {
			reqs = append(reqs, &task.CreateTaskRequest{
				DeviceSN:    sn,
				Method:      rpcMethod,
				Params:      params,
				Priority:    10,
				Source:      task.TaskSourceMML,
				CreatorID:   mmlTask.Creator,
				Description: description,

				SourceID:     parentID,
				CommandIndex: cmdIdx,
				DeviceIndex:  devIdx,
			})
		}
	}

	return reqs
}

// paramRefsFromEntry pulls param_refs out of a commands[] entry, tolerating
// both the typed []MMLParamRef form (when service stashed it in-process) and
// the JSON-roundtrip form ([]interface{} of map[string]interface{}, after the
// task has been written to mml_tasks.commands JSONB and read back).
func paramRefsFromEntry(cmd map[string]interface{}) []MMLParamRef {
	raw, ok := cmd["param_refs"]
	if !ok || raw == nil {
		return nil
	}
	if typed, ok := raw.([]MMLParamRef); ok {
		return typed
	}
	bs, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var refs []MMLParamRef
	if err := json.Unmarshal(bs, &refs); err != nil {
		return nil
	}
	return refs
}
