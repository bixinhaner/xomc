export interface NameFilterItem {
  id: string;
  condition: 'contain' | 'notContain' | 'startWith' | 'endWith';
  value: string;
  andOr?: 'and' | 'or';
}

/** Group data from the API */
export interface GroupItem {
  id: string;
  name: string;
  parentId: string | null;
  deviceCount: number;
  description: string;
  /** 是否为内置设备组：1=内置, 0=自定义 */
  builtIn: number;
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
