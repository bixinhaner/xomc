interface CurrentOnlineDurationInput {
  isOnline: boolean | undefined;
  onlineTime: string | undefined | null;
  offlineTime: string | undefined | null;
  fallbackOnlineDuration: number | null | undefined;
  nowMs?: number;
}

interface CumulativeOnlineDurationInput {
  isOnline: boolean | undefined;
  onlineTime: string | undefined | null;
  offlineTime: string | undefined | null;
  fallbackOnlineDuration: number | null | undefined;
  cumulativeOnlineDuration: number | null | undefined;
  nowMs?: number;
}

const normalizePositive = (value: number | null | undefined): number | null => {
  if (typeof value !== 'number' || Number.isNaN(value) || value <= 0) {
    return null;
  }
  return Math.floor(value);
};

const parseTime = (value: string | undefined | null): number | null => {
  if (!value) return null;
  const ts = Date.parse(value);
  return Number.isNaN(ts) ? null : ts;
};

export function computeCurrentOnlineDurationSeconds(input: CurrentOnlineDurationInput): number | null {
  const fallback = normalizePositive(input.fallbackOnlineDuration);
  const onlineStart = parseTime(input.onlineTime);
  if (onlineStart == null) return fallback;

  const now = input.nowMs ?? Date.now();
  if (input.isOnline) {
    return Math.max(0, Math.floor((now - onlineStart) / 1000));
  }

  const offlineAt = parseTime(input.offlineTime);
  if (offlineAt != null && offlineAt > onlineStart) {
    return Math.floor((offlineAt - onlineStart) / 1000);
  }

  return fallback;
}

export function computeCumulativeOnlineDurationSeconds(input: CumulativeOnlineDurationInput): number | null {
  const cumulative = normalizePositive(input.cumulativeOnlineDuration) ?? 0;
  if (!input.isOnline) {
    return cumulative > 0 ? cumulative : null;
  }

  const currentSegment = computeCurrentOnlineDurationSeconds({
    isOnline: input.isOnline,
    onlineTime: input.onlineTime,
    offlineTime: input.offlineTime,
    fallbackOnlineDuration: input.fallbackOnlineDuration,
    nowMs: input.nowMs,
  });
  const total = cumulative + (currentSegment ?? 0);
  return total > 0 ? total : null;
}
