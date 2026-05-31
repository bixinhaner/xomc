import type { NameFilterItem } from '@core/types/device';
// NameFilterItem 已提升到 @core/types/device 作为全局类型，此处重新导出保持向后兼容
export type { NameFilterItem };

/** Group data from the API */
export interface GroupItem {
  id: string;
  name: string;
  /** i18n JSONB (后端 migration 000003)。Axios camel 转换 + useI18nText 兼容两种 key 形态。 */
  nameI18n?: Record<string, string>;
  descriptionI18n?: Record<string, string>;
  remarkI18n?: Record<string, string>;
  parentId: string | null;
  deviceCount: number;
  description: string;
  /** 是否为内置设备组：1=内置, 0=自定义 */
  builtIn: number;
  /**
   * 匹配规则字段 — 由 deviceApi.getGroups() walk() 透传后端 device_groups 表。
   * L2 子分组编辑入口（useGroupActions.openEditLevel2）回填表单依赖这些字段。
   */
  matchingMode?: 'deviceName' | 'lac' | 'tac' | 'serialNumber';
  nameRuleList?: NameFilterItem[];
  lacList?: number[];
  tacList?: number[];
}

// Filter condition options helper
export const getFilterConditionOptions = (t: (key: string) => string) => [
  { label: t('filter.contain'), value: 'contain' },
  { label: t('filter.notContain'), value: 'notContain' },
  { label: t('filter.startWith'), value: 'startWith' },
  { label: t('filter.endWith'), value: 'endWith' },
];

// And/Or options helper
export const getAndOrOptions = (t: (key: string) => string) => [
  { label: t('filter.and'), value: 'and' },
  { label: t('filter.or'), value: 'or' },
];

// Generate unique ID
export const generateId = () => Math.random().toString(36).substring(2, 9);

/**
 * Parse range string to number array
 * Examples: "1,2,3" => [1,2,3], "1-3" => [1,2,3], "1,2,10-12" => [1,2,10,11,12]
 */
export function parseRangeString(input: string): number[] {
  if (!input || !input.trim()) return [];

  const result: number[] = [];
  const parts = input.split(',');

  for (const part of parts) {
    const trimmed = part.trim();
    if (!trimmed) continue;

    // Check if it's a range (e.g., "10-20")
    if (trimmed.includes('-')) {
      const [startStr, endStr] = trimmed.split('-');
      const start = parseInt(startStr, 10);
      const end = parseInt(endStr, 10);

      if (!isNaN(start) && !isNaN(end) && start <= end) {
        for (let i = start; i <= end; i++) {
          result.push(i);
        }
      }
    } else {
      const num = parseInt(trimmed, 10);
      if (!isNaN(num)) {
        result.push(num);
      }
    }
  }

  // Remove duplicates and sort
  return [...new Set(result)].sort((a, b) => a - b);
}

// Generate operators description
export function generateOperators(
  rule: { matchingMode?: string; nameRuleList?: NameFilterItem[]; tacRag?: string },
  t: (key: string) => string
): string {
  if (rule.matchingMode === 'deviceName' && rule.nameRuleList?.length) {
    const orGroups: NameFilterItem[][] = [[]];

    rule.nameRuleList.forEach((filter, index) => {
      if (filter.value && filter.value.trim() !== '') {
        if (index > 0 && filter.andOr === 'or') {
          orGroups.push([]);
        }
        orGroups[orGroups.length - 1].push(filter);
      }
    });

    const filteredGroups = orGroups.filter((g) => g.length > 0);
    if (filteredGroups.length === 0) return '';

    const verbMap: Record<string, string> = {
      contain: t('filter.contain'),
      notContain: t('filter.notContain'),
      startWith: t('filter.startWith'),
      endWith: t('filter.endWith'),
    };

    const groupParts = filteredGroups.map((group) => {
      const conditionParts = group.map((item) => `${verbMap[item.condition]} "${item.value}"`);
      const groupText = conditionParts.join(` ${t('filter.and')} `);
      return filteredGroups.length > 1 || group.length > 1 ? `(${groupText})` : groupText;
    });

    return groupParts.join(` ${t('filter.or')} `);
  } else if (rule.matchingMode === 'tac') {
    return `TAC: ${rule.tacRag || ''}`;
  } else if (rule.matchingMode === 'lac') {
    return `LAC: ${rule.tacRag || ''}`;
  }
  return '';
}
