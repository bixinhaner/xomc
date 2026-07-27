export interface AlarmTrendBucket {
  date: string;
  label: string;
  critical: number;
  major: number;
  minor: number;
  warning: number;
}

type AlarmTrendPoint = Omit<AlarmTrendBucket, 'label'>;

export function selectAlarmTrendBuckets(
  trendData: AlarmTrendPoint[] | undefined,
  days: number,
): AlarmTrendBucket[] {
  return [...(trendData ?? [])]
    .sort((left, right) => left.date.localeCompare(right.date))
    .slice(-days)
    .map((item) => ({
      ...item,
      label: item.date.length >= 10
        ? `${item.date.slice(5, 7)}/${item.date.slice(8, 10)}`
        : item.date,
    }));
}
