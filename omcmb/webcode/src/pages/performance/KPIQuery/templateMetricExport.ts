import type { QueryTemplate } from '@core/types/pmQuery';

export function buildTemplateMetricExportText(template: QueryTemplate): string {
  return template.payload.metricPaths.join('\n');
}

export function buildTemplateMetricExportFilename(template: QueryTemplate): string {
  const safeName = template.name
    .trim()
    .replace(/[\\/:*?"<>|]+/g, '_')
    .replace(/\s+/g, '_')
    .slice(0, 80) || 'query-template';
  return `${safeName}_metrics.txt`;
}
