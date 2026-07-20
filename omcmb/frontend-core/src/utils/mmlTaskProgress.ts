import type { MMLTask } from '../types/mml';

export type MMLTaskProgress =
  | { kind: 'execution'; done: number; total: number }
  | { kind: 'awaiting_first_run'; done: 0; total: 0 };

type MMLTaskProgressSource = Pick<
  MMLTask,
  'successCount' | 'failedCount' | 'totalDevices' | 'commandCount' | 'executeMode' | 'planItemCount'
> & {
  commands?: unknown[];
  planItems?: unknown[];
};

function getExecutionProgress(task: MMLTaskProgressSource): MMLTaskProgress {
  const done = (task.successCount ?? 0) + (task.failedCount ?? 0);
  const planItemCount = task.planItemCount ?? task.planItems?.length ?? 0;
  const commandCount = task.commandCount ?? task.commands?.length ?? 0;
  if (task.executeMode === 'device_bound' && planItemCount > 0) {
    const executableCount = commandCount > 0 ? commandCount : planItemCount;
    return { kind: 'execution', done, total: Math.max(done, executableCount) };
  }
  const total = (task.totalDevices ?? 0) * Math.max(1, commandCount || 1);
  return { kind: 'execution', done, total };
}

export function getMmlTaskProgress(task: MMLTask): MMLTaskProgress {
  if (task.executeType === 'periodic' && !task.parentTaskId) {
    if (task.latestRun) {
      return getExecutionProgress(task.latestRun);
    }
    return { kind: 'awaiting_first_run', done: 0, total: 0 };
  }
  return getExecutionProgress(task);
}
