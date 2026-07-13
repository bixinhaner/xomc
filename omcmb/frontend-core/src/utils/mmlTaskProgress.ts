import type { MMLTask } from '../types/mml';

export function getMmlTaskProgress(task: MMLTask): { done: number; total: number } {
  const done = (task.successCount ?? 0) + (task.failedCount ?? 0);
  const planItemCount = task.planItemCount ?? task.planItems?.length ?? 0;
  const commandCount = task.commandCount ?? task.commands?.length ?? 0;
  if (task.executeMode === 'device_bound' && planItemCount > 0) {
    const executableCount = commandCount > 0 ? commandCount : planItemCount;
    return { done, total: Math.max(done, executableCount) };
  }
  const total = (task.totalDevices ?? 0) * Math.max(1, commandCount || 1);
  return { done, total };
}
