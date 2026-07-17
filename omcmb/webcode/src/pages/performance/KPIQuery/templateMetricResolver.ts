import { indicatorLibraryApi } from '@core/services/api/indicatorLibraryApi';
import { useAppStore } from '@core/store/appStore';
import type { DeviceType } from '@core/types/indicatorLibrary';

type IndicatorList = typeof indicatorLibraryApi.list;
type IndicatorItem = Awaited<ReturnType<IndicatorList>>['items'][number];

function metricLabelOf(item: IndicatorItem): string {
  const isEn = useAppStore.getState().locale === 'en-US';
  return isEn ? item.enName || item.id : item.cnName || item.enName || item.id;
}

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

    const idMatch = items.find((item) => item.id === path);
    if (idMatch) {
      outPaths.push(path);
      labels[path] = metricLabelOf(idMatch);
      continue;
    }

    const exact = items.filter((item) => item.enName === path || item.cnName === path);
    const kpiMatches = exact.filter((item) => !item.isCounter);
    if (kpiMatches.length === 1) {
      const match = kpiMatches[0];
      const label = metricLabelOf(match);
      outPaths.push(match.id);
      labels[match.id] = label;
      labels[path] = label;
    } else if (kpiMatches.length > 1) {
      ambiguous.push(path);
      outPaths.push(path);
    } else {
      outPaths.push(path);
      const counter = exact.find((item) => item.isCounter);
      if (counter) labels[path] = metricLabelOf(counter);
    }
  }
  return { paths: outPaths, labels, ambiguous };
}
