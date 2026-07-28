import type { AdhocTask } from '@core/types/pmAdhoc';

export function displayAdhocTaskName(
  task: Pick<AdhocTask, 'name' | 'isBuiltin' | 'technology'>,
  labelForTechnology: (technology?: string | null) => string,
): string {
  if (!task.isBuiltin || !task.technology) return task.name;
  const label = labelForTechnology(task.technology);
  const parts = task.name.split('-');
  if (parts.length < 3) return `${task.name}-${label}`;
  return [...parts.slice(0, -1), label].join('-');
}
