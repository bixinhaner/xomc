import type { QueryTemplate, QueryTemplatePayload } from '@core/types/pmQuery';

interface QueryTemplatePageState {
  formPayload: QueryTemplatePayload;
  submittedPayload: QueryTemplatePayload | null;
}

export function synchronizeUpdatedTemplateState(
  activeTemplateId: string | undefined,
  current: QueryTemplatePageState,
  updated: QueryTemplate,
  resolvedMetricPaths: string[],
): QueryTemplatePageState {
  if (activeTemplateId !== updated.id) return current;
  return {
    formPayload: { ...updated.payload, metricPaths: resolvedMetricPaths },
    submittedPayload: current.submittedPayload,
  };
}
