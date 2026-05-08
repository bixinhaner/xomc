import type {
  AlarmDefinition,
  AlarmDefinitionFilter,
  CreateAlarmDefinitionInput,
  UpdateAlarmDefinitionInput,
  UnknownStatsFilter,
} from '../../types/alarmDefinition';
import {
  mockAlarmDefinitions,
  mockAlarmSeverityLevels,
  mockUnknownStats,
} from '../data/alarmDefinition';

let definitions = [...mockAlarmDefinitions];

function clone<T>(v: T): T {
  return JSON.parse(JSON.stringify(v)) as T;
}

export const alarmDefinitionService = {
  async list(filter?: AlarmDefinitionFilter) {
    let items = [...definitions];
    if (filter?.neType) items = items.filter((d) => d.neType === filter.neType);
    if (filter?.severityCode !== undefined)
      items = items.filter((d) => d.severityCode === filter.severityCode);
    if (filter?.keyword) {
      const k = filter.keyword.toLowerCase();
      items = items.filter((d) => (d.identifier + d.cnName + d.enName).toLowerCase().includes(k));
    }
    if (filter?.isUnknown !== undefined) items = items.filter((d) => Boolean(d.isUnknown) === filter.isUnknown);
    const page = filter?.page || 1;
    const pageSize = filter?.pageSize || 20;
    const start = (page - 1) * pageSize;
    return {
      items: clone(items.slice(start, start + pageSize)),
      total: items.length,
      page,
      pageSize,
    };
  },

  async get(identifier: string): Promise<AlarmDefinition> {
    const d = definitions.find((x) => x.identifier === identifier);
    if (!d) throw new Error(`alarm definition ${identifier} not found`);
    return clone(d);
  },

  async create(input: CreateAlarmDefinitionInput): Promise<AlarmDefinition> {
    const sev = mockAlarmSeverityLevels.find((s) => s.code === input.severityCode);
    const created: AlarmDefinition = {
      id: `ad-${Date.now()}`,
      identifier: input.identifier,
      neType: input.neType,
      cnName: input.cnName,
      enName: input.enName,
      severityCode: input.severityCode,
      severityName: sev?.enName || 'Warning',
      eventType: input.eventType,
      cnProbableCause: input.cnProbableCause,
      enProbableCause: input.enProbableCause,
      cnSuggestion: input.cnSuggestion,
      enSuggestion: input.enSuggestion,
      isShow: input.isShow ?? true,
      isUnknown: false,
    };
    definitions.push(created);
    return clone(created);
  },

  async update(identifier: string, input: UpdateAlarmDefinitionInput): Promise<AlarmDefinition> {
    const idx = definitions.findIndex((d) => d.identifier === identifier);
    if (idx < 0) throw new Error(`alarm definition ${identifier} not found`);
    definitions[idx] = {
      ...definitions[idx],
      ...(input.neType !== undefined && { neType: input.neType }),
      ...(input.cnName !== undefined && { cnName: input.cnName }),
      ...(input.enName !== undefined && { enName: input.enName }),
      ...(input.severityCode !== undefined && {
        severityCode: input.severityCode,
        severityName: mockAlarmSeverityLevels.find((s) => s.code === input.severityCode)?.enName ||
          definitions[idx].severityName,
      }),
      ...(input.eventType !== undefined && { eventType: input.eventType }),
      ...(input.cnProbableCause !== undefined && { cnProbableCause: input.cnProbableCause }),
      ...(input.enProbableCause !== undefined && { enProbableCause: input.enProbableCause }),
      ...(input.cnSuggestion !== undefined && { cnSuggestion: input.cnSuggestion }),
      ...(input.enSuggestion !== undefined && { enSuggestion: input.enSuggestion }),
      ...(input.isShow !== undefined && { isShow: input.isShow }),
    };
    return clone(definitions[idx]);
  },

  async delete(identifier: string): Promise<void> {
    definitions = definitions.filter((d) => d.identifier !== identifier);
  },

  async unknownStats(filter?: UnknownStatsFilter) {
    let items = [...mockUnknownStats];
    if (filter?.productId) items = items.filter((u) => u.productId === filter.productId);
    return { items: clone(items), days: filter?.days || 7 };
  },

  async severityLevels() {
    return { items: clone(mockAlarmSeverityLevels) };
  },

  async cacheRefresh() {
    return { refreshed: true };
  },

  async importDirectory() {
    return { reloaded: 'alarm-definition' };
  },
};
