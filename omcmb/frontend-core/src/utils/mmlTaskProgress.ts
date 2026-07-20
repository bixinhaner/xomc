import type { MMLTask } from '../types/mml';

export type MMLTaskProgress =
  | { kind: 'execution'; done: number; total: number }
  | { kind: 'schedule_template'; done: 0; total: 0 };

export function getMmlTaskProgress(task: MMLTask): MMLTaskProgress {
  if (task.executeType === 'periodic' && !task.parentTaskId) {
    return { kind: 'schedule_template', done: 0, total: 0 };
  }
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
