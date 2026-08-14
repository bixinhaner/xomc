import type {
  AlarmDefinition,
  AlarmDefinitionFilter,
  AlarmNeTypeStat,
  CreateAlarmDefinitionInput,
  UpdateAlarmDefinitionInput,
  UnknownStatsFilter,
  AlarmUploadResult,
  AlarmDeleteFileResult,
} from '../../types/alarmDefinition';
import {
  mockAlarmDefinitions,
  mockAlarmSeverityLevels,
  mockUnknownStats,
} from '../data/alarmDefinition';
import { extractXmlRootAttr } from '../../utils/xmlRootAttr';
import { saveBlob } from '../../utils/saveBlob';

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

  // 2026-06-03:与 real api 对齐 — 全量返回(mock 数据量小,一次性返回,不分页)。
  async listAll(filter?: Omit<AlarmDefinitionFilter, 'page'>) {
    let items = [...definitions];
    if (filter?.neType) items = items.filter((d) => d.neType === filter.neType);
    if (filter?.severityCode !== undefined)
      items = items.filter((d) => d.severityCode === filter.severityCode);
    if (filter?.keyword) {
      const k = filter.keyword.toLowerCase();
      items = items.filter((d) => (d.identifier + d.cnName + d.enName).toLowerCase().includes(k));
    }
    if (filter?.isUnknown !== undefined) items = items.filter((d) => Boolean(d.isUnknown) === filter.isUnknown);
    return { items: clone(items), total: items.length, page: 1, pageSize: items.length };
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

  async listNeTypes(): Promise<{ items: AlarmNeTypeStat[] }> {
    const grouped = new Map<string, AlarmNeTypeStat>();
    for (const d of definitions) {
      const key = `${d.neType}__`;
      let stat = grouped.get(key);
      if (!stat) {
        stat = {
          neType: d.neType,
          loadedFrom: `${d.neType}.xml`,
          total: 0,
          criticalCnt: 0,
          majorCnt: 0,
          minorCnt: 0,
          warningCnt: 0,
        };
        grouped.set(key, stat);
      }
      stat.total += 1;
      switch (d.severityCode) {
        case 31001: stat.criticalCnt += 1; break;
        case 31002: stat.majorCnt += 1; break;
        case 31003: stat.minorCnt += 1; break;
        case 31004: stat.warningCnt += 1; break;
      }
    }
    return { items: Array.from(grouped.values()).sort((a, b) => a.neType.localeCompare(b.neType)) };
  },

  // mock 上传:名称取自 XML neType 属性(与后端同口径);读不到回退 'CUSTOM'。
  // force = 二次确认后的覆盖(mock 不真存盘,直接回成功)。
  async uploadXml(file: File, force = false): Promise<AlarmUploadResult> {
    const neType = (await extractXmlRootAttr(file, 'neType')) ?? 'CUSTOM';
    return {
      uploaded: true,
      filename: `${neType}.xml`,
      loadedFrom: `alarm-definitions/${neType}.xml`,
      neType,
      overwritten: force,
      reloaded: true,
    };
  },

  // mock 下载:生成占位 XML 触发浏览器另存(无真实文件)。
  async downloadXml(loadedFrom: string): Promise<void> {
    const name = loadedFrom.split('/').pop() || 'alarm.xml';
    saveBlob(`<?xml version="1.0" encoding="UTF-8"?>\n<!-- mock ${name} -->\n<alarmModel neType="${name.replace(/\.xml$/i, '')}"></alarmModel>\n`, name);
  },

  async deleteFile(loadedFrom: string): Promise<AlarmDeleteFileResult> {
    return { deleted: true, loadedFrom, rowsAffected: 0, backup: '' };
  },
};
