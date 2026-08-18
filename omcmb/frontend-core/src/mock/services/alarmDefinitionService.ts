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

// mock 行级加载源:显式 loadedFrom('' = 手工新增)优先,缺省视为 <neType>.xml
// 文件行 — 与后端 alarm_definitions.loaded_from 语义对齐(#268)。
function effectiveLoadedFrom(d: AlarmDefinition): string {
  return d.loadedFrom ?? `${d.neType}.xml`;
}

function applyCommonFilter(items: AlarmDefinition[], filter?: AlarmDefinitionFilter) {
  let out = items;
  if (filter?.neType) out = out.filter((d) => d.neType === filter.neType);
  if (filter?.loadedFrom !== undefined)
    out = out.filter((d) => effectiveLoadedFrom(d) === filter.loadedFrom);
  if (filter?.severityCode !== undefined)
    out = out.filter((d) => d.severityCode === filter.severityCode);
  if (filter?.keyword) {
    const k = filter.keyword.toLowerCase();
    out = out.filter((d) => (d.identifier + d.cnName + d.enName).toLowerCase().includes(k));
  }
  if (filter?.isUnknown !== undefined) out = out.filter((d) => Boolean(d.isUnknown) === filter.isUnknown);
  return out;
}

export const alarmDefinitionService = {
  async list(filter?: AlarmDefinitionFilter) {
    const items = applyCommonFilter([...definitions], filter);
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
    const items = applyCommonFilter([...definitions], filter);
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
      isShow: input.isShow ?? true,
      isUnknown: false,
      loadedFrom: '',
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
    // 按 (ne_type, loaded_from) 双键聚合,与后端 ListNeTypes 同口径;
    // 手工新增(loadedFrom '')单独成行(#268)。
    const grouped = new Map<string, AlarmNeTypeStat>();
    for (const d of definitions) {
      const lf = effectiveLoadedFrom(d);
      const key = `${d.neType}__${lf}`;
      let stat = grouped.get(key);
      if (!stat) {
        stat = {
          neType: d.neType,
          loadedFrom: lf,
          total: 0,
          criticalCnt: 0,
          majorCnt: 0,
          minorCnt: 0,
          warningCnt: 0,
        };
        grouped.set(key, stat);
      }
      stat.total += 1;
      // mock severity 用 1-4 编码(severityLevels 种子),兼容真后端 31001-31004。
      switch (d.severityCode) {
        case 1: case 31001: stat.criticalCnt += 1; break;
        case 2: case 31002: stat.majorCnt += 1; break;
        case 3: case 31003: stat.minorCnt += 1; break;
        case 4: case 31004: stat.warningCnt += 1; break;
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
  // #268: 手工新增行只带 neType,同样生成占位模型。
  async downloadXml(target: { loadedFrom?: string; neType?: string }): Promise<void> {
    if (target.loadedFrom) {
      const name = target.loadedFrom.split('/').pop() || 'alarm.xml';
      saveBlob(`<?xml version="1.0" encoding="UTF-8"?>\n<!-- mock ${name} -->\n<alarmModel neType="${name.replace(/\.xml$/i, '')}"></alarmModel>\n`, name);
      return;
    }
    const neType = target.neType ?? '';
    saveBlob(`<?xml version="1.0" encoding="UTF-8"?>\n<!-- mock manual ${neType} -->\n<alarmModel neType="${neType}" totalCount="0"><alarms></alarms></alarmModel>\n`, `${neType}-manual.xml`);
  },

  async deleteFile(loadedFrom: string): Promise<AlarmDeleteFileResult> {
    return { deleted: true, loadedFrom, rowsAffected: 0, backup: '' };
  },
};
