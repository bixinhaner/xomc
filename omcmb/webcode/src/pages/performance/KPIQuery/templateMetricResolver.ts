import { indicatorLibraryApi } from '@core/services/api/indicatorLibraryApi';
import type { DeviceType } from '@core/types/indicatorLibrary';

const KPI_CODE_RE = /^(C\d+|KGNB\d+|KGSM\d+|K\d+)$/;

type IndicatorList = typeof indicatorLibraryApi.list;

export async function resolveTemplateMetricPaths(
  deviceType: DeviceType,
  paths: string[],
  list: IndicatorList = indicatorLibraryApi.list,
): Promise<{ paths: string[]; labels: Record<string, string>; ambiguous: string[] }> {
  const outPaths: string[] = [];
  const labels: Record<string, string> = {};
  const ambiguous: string[] = [];
  for (const path of paths) {
    let items: Awaited<ReturnType<IndicatorList>>['items'] = [];
    try {
      ({ items } = await list(deviceType, { keyword: path, pageSize: 50 }));
    } catch {
      outPaths.push(path);
      continue;
    }

    if (KPI_CODE_RE.test(path)) {
      outPaths.push(path);
      const match = items.find((item) => item.id === path);
      if (match) labels[path] = match.cnName || match.enName || match.id;
      continue;
    }

    const exact = items.filter((item) => item.enName === path || item.cnName === path);
    const kpiMatches = exact.filter((item) => !item.isCounter);
    if (kpiMatches.length === 1) {
      const match = kpiMatches[0];
      outPaths.push(match.id);
      labels[match.id] = match.cnName || match.enName || match.id;
    } else if (kpiMatches.length > 1) {
      ambiguous.push(path);
      outPaths.push(path);
    } else {
      outPaths.push(path);
      const counter = exact.find((item) => item.isCounter);
      if (counter) labels[path] = counter.cnName || path;
    }
  }
  return { paths: outPaths, labels, ambiguous };
}
