import { useCallback, useState } from 'react';
import type { NameFilterItem } from './types';
import { generateId } from './types';

const INITIAL_FILTER = (): NameFilterItem => ({
  id: generateId(),
  condition: 'contain',
  value: '',
});

export interface UseNameFiltersOptions {
  /** Maximum number of filter rows. Default 10. */
  max?: number;
  /** Callback when filter list reaches max — usually a toast warning. */
  onMaxReached?: () => void;
}

export interface UseNameFiltersReturn {
  filters: NameFilterItem[];
  setFilters: React.Dispatch<React.SetStateAction<NameFilterItem[]>>;
  reset: () => void;
  add: () => void;
  remove: (id: string) => void;
  update: (id: string, field: keyof NameFilterItem, value: string) => void;
}

/**
 * 维护"设备名称匹配条件列表"的本地状态 + 增删改语义。
 * 拆分自 DeviceGrouping/index.tsx（W2.C.2 / T-0054）。
 */
export function useNameFilters(options: UseNameFiltersOptions = {}): UseNameFiltersReturn {
  const { max = 10, onMaxReached } = options;
  const [filters, setFilters] = useState<NameFilterItem[]>([INITIAL_FILTER()]);

  const reset = useCallback(() => {
    setFilters([INITIAL_FILTER()]);
  }, []);

  const add = useCallback(() => {
    setFilters((prev) => {
      if (prev.length >= max) {
        onMaxReached?.();
        return prev;
      }
      const hasOr = prev.some((f, index) => index > 0 && f.andOr === 'or');
      return [
        ...prev,
        { id: generateId(), condition: 'contain', value: '', andOr: hasOr ? 'or' : 'and' },
      ];
    });
  }, [max, onMaxReached]);

  const remove = useCallback((id: string) => {
    setFilters((prev) => {
      if (prev.length <= 1) return prev;
      const newFilters = prev.filter((f) => f.id !== id);
      if (newFilters.length > 0 && newFilters[0].andOr !== undefined) {
        const { andOr: _andOr, ...rest } = newFilters[0];
        void _andOr;
        newFilters[0] = rest as NameFilterItem;
      }
      return newFilters;
    });
  }, []);

  const update = useCallback((id: string, field: keyof NameFilterItem, value: string) => {
    setFilters((prev) => prev.map((f) => (f.id === id ? { ...f, [field]: value } : f)));
  }, []);

  return { filters, setFilters, reset, add, remove, update };
}
