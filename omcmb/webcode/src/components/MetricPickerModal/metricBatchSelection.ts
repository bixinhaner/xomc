export interface MetricBatchIndicator {
  id: string;
}

interface BuildMetricBatchSelectionInput<T extends MetricBatchIndicator> {
  text: string;
  indicators: T[];
  selected: string[];
  maxSelected?: number;
}

export interface MetricBatchSelectionResult {
  requestedIds: string[];
  validIds: string[];
  invalidIds: string[];
  addedIds: string[];
  omittedValidIds: string[];
  nextSelected: string[];
  limitExceeded: boolean;
}

const METRIC_ID_SEPARATOR = /[\s,，、;；]+/u;

export function parseMetricBatchIds(text: string): string[] {
  const ids = text
    .split(METRIC_ID_SEPARATOR)
    .map((part) => part.trim())
    .filter(Boolean);
  return Array.from(new Set(ids));
}

export function buildMetricBatchSelection<T extends MetricBatchIndicator>({
  text,
  indicators,
  selected,
  maxSelected,
}: BuildMetricBatchSelectionInput<T>): MetricBatchSelectionResult {
  const requestedIds = parseMetricBatchIds(text);
  const indicatorIds = new Set(indicators.map((indicator) => indicator.id));
  const validIds = requestedIds.filter((id) => indicatorIds.has(id));
  const invalidIds = requestedIds.filter((id) => !indicatorIds.has(id));

  if (validIds.length === 0) {
    return {
      requestedIds,
      validIds,
      invalidIds,
      addedIds: [],
      omittedValidIds: [],
      nextSelected: selected,
      limitExceeded: false,
    };
  }

  const merged = Array.from(new Set([...selected, ...validIds]));
  const nextSelected = maxSelected === undefined ? merged : merged.slice(0, maxSelected);
  const selectedSet = new Set(selected);
  const nextSelectedSet = new Set(nextSelected);
  const addedIds = validIds.filter((id) => !selectedSet.has(id) && nextSelectedSet.has(id));
  const omittedValidIds = validIds.filter((id) => !selectedSet.has(id) && !nextSelectedSet.has(id));

  return {
    requestedIds,
    validIds,
    invalidIds,
    addedIds,
    omittedValidIds,
    nextSelected,
    limitExceeded: maxSelected !== undefined && merged.length > maxSelected,
  };
}

export function formatMetricIdSamples(ids: string[], limit = 5): string {
  const samples = ids.slice(0, limit).join(', ');
  return ids.length > limit ? `${samples} ...` : samples;
}
