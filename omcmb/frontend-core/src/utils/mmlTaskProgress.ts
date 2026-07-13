import type { MMLTask } from '../types/mml';

export function getMmlTaskProgress(task: MMLTask): { done: number; total: number } {
  const done = (task.successCount ?? 0) + (task.failedCount ?? 0);
  const planItemCount = task.planItemCount ?? task.planItems?.length ?? 0;
  if (task.executeMode === 'device_bound' && planItemCount > 0) {
    return { done, total: planItemCount };
  }
  const commandCount = task.commandCount ?? task.commands?.length ?? 1;
  const total = (task.totalDevices ?? 0) * Math.max(1, commandCount);
  return { done, total };
}
