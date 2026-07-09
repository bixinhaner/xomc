import type { MMLTask } from '../types/mml';

export function getMmlTaskProgress(task: MMLTask): { done: number; total: number } {
  const done = (task.successCount ?? 0) + (task.failedCount ?? 0);
  if (task.executeMode === 'device_bound' && task.planItems && task.planItems.length > 0) {
    return { done, total: task.planItems.length };
  }
  const total = (task.totalDevices ?? 0) * Math.max(1, task.commands?.length ?? 1);
  return { done, total };
}
