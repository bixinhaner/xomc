import http from '../http';
import { saveBlob } from '../../utils/saveBlob';
import type {
  AlarmDefinition,
  AlarmNeTypeStat,
  AlarmSeverityLevel,
  UnknownAlarmStat,
  AlarmDefinitionFilter,
  CreateAlarmDefinitionInput,
  UpdateAlarmDefinitionInput,
  UnknownStatsFilter,
  AlarmUploadResult,
  AlarmDeleteFileResult,
} from '../../types/alarmDefinition';

interface BackendDefinition {
  id: string;
  identifier: string;
  ne_type: string;
  cn_name: string;
  en_name: string;
  severity_id?: string;
  severity_code: number;
  severity_name: string;
  event_type?: number | string;
  cn_probable_cause?: string;
  en_probable_cause?: string;
  cn_suggestion?: string;
  en_suggestion?: string;
  description?: string;
  is_show: boolean;
  is_unknown?: boolean;
  created_at?: string;
  updated_at?: string;
}

interface BackendSeverityLevel {
  id: string;
  code: number | string;
  // 后端 SeverityLevel struct 只有单 name 列(DB alarm_severity_levels.name);
  // cn_name/en_name/color_hex 仅 mock 数据用,可选保留兼容。
  name?: string;
  cn_name?: string;
  en_name?: string;
  color_hex?: string;
}

interface BackendUnknownStat {
  product_id?: string;
  product_name?: string;
  identifier: string;
  count: number;
  last_seen_at?: string;
}

interface BackendNeTypeStat {
  ne_type: string;
  loaded_from: string;
  source?: string;
  deletable?: boolean;
  total: number;
  critical_cnt: number;
  major_cnt: number;
  minor_cnt: number;
  warning_cnt: number;
}

function normalizeNumericValue(raw: number | string | undefined): number | undefined {
  if (raw === undefined || raw === null || raw === '') {
    return undefined;
  }

  const value = typeof raw === 'number' ? raw : Number(String(raw).trim());
  return Number.isFinite(value) ? value : undefined;
}

function normalizeDefinitionEventType(raw: number | string | undefined): number | string | undefined {
  const numeric = normalizeNumericValue(raw);
  if (numeric !== undefined) {
    return numeric;
  }

  if (raw === undefined || raw === null) {
    return undefined;
  }

  const value = String(raw).trim();
  return value === '' ? undefined : value;
}

function mapNeTypeStat(b: BackendNeTypeStat): AlarmNeTypeStat {
  return {
    neType: b.ne_type,
    loadedFrom: b.loaded_from,
    source: (b.source ?? 'unknown') as AlarmNeTypeStat['source'],
    deletable: Boolean(b.deletable),
    total: b.total,
    criticalCnt: b.critical_cnt,
    majorCnt: b.major_cnt,
    minorCnt: b.minor_cnt,
    warningCnt: b.warning_cnt,
  };
}

function mapDef(b: BackendDefinition): AlarmDefinition {
  return {
    id: b.id,
    identifier: b.identifier,
    neType: b.ne_type,
    cnName: b.cn_name,
    enName: b.en_name,
    severityId: b.severity_id,
    severityCode: b.severity_code,
    severityName: b.severity_name,
    eventType: normalizeDefinitionEventType(b.event_type),
    cnProbableCause: b.cn_probable_cause,
    enProbableCause: b.en_probable_cause,
    cnSuggestion: b.cn_suggestion,
    enSuggestion: b.en_suggestion,
    description: b.description,
    isShow: b.is_show,
    isUnknown: b.is_unknown,
    createdAt: b.created_at,
    updatedAt: b.updated_at,
  };
}

function mapSeverity(b: BackendSeverityLevel): AlarmSeverityLevel {
  return {
    id: b.id,
    code: normalizeNumericValue(b.code) ?? 0,
    name: b.name,
    cnName: b.cn_name,
    enName: b.en_name,
    colorHex: b.color_hex,
  };
}

function mapUnknown(b: BackendUnknownStat): UnknownAlarmStat {
  return {
    productId: b.product_id,
    productName: b.product_name,
    identifier: b.identifier,
    count: b.count,
    lastSeenAt: b.last_seen_at,
  };
}

function defPayload(
  input: CreateAlarmDefinitionInput | UpdateAlarmDefinitionInput
): Record<string, unknown> {
  const p: Record<string, unknown> = {};
  if ('identifier' in input && input.identifier !== undefined) p.identifier = input.identifier;
  if (input.neType !== undefined) p.ne_type = input.neType;
  if (input.cnName !== undefined) p.cn_name = input.cnName;
  if (input.enName !== undefined) p.en_name = input.enName;
  if (input.severityCode !== undefined) p.severity_code = input.severityCode;
  if (input.eventType !== undefined) p.event_type = input.eventType;
  if (input.cnProbableCause !== undefined) p.cn_probable_cause = input.cnProbableCause;
  if (input.enProbableCause !== undefined) p.en_probable_cause = input.enProbableCause;
  if (input.cnSuggestion !== undefined) p.cn_suggestion = input.cnSuggestion;
  if (input.enSuggestion !== undefined) p.en_suggestion = input.enSuggestion;
  if (input.description !== undefined) p.description = input.description;
  if (input.isShow !== undefined) p.is_show = input.isShow;
  return p;
}

export const alarmDefinitionApi = {
  async list(filter?: AlarmDefinitionFilter): Promise<{
    items: AlarmDefinition[];
    total: number;
    page: number;
    pageSize: number;
  }> {
    const params: Record<string, unknown> = {};
    if (filter?.neType) params.ne_type = filter.neType;
    if (filter?.loadedFrom !== undefined) {
      params.loaded_from = filter.loadedFrom === '' ? '__empty__' : filter.loadedFrom;
    }
    if (filter?.severityCode !== undefined) params.severity_code = filter.severityCode;
    if (filter?.keyword) params.keyword = filter.keyword;
    if (filter?.isUnknown !== undefined) params.is_unknown = filter.isUnknown;
    if (filter?.page) params.page = filter.page;
    if (filter?.pageSize) params.page_size = filter.pageSize;

    const { data } = await http.get<{
      items: BackendDefinition[];
      total: number;
      page: number;
      page_size: number;
    }>('/alarm-definitions', { params });
    return {
      items: (data.items || []).map(mapDef),
      total: data.total || 0,
      page: data.page || 1,
      pageSize: data.page_size || 20,
    };
  },

  async listAll(filter?: Omit<AlarmDefinitionFilter, 'page'>): Promise<{
    items: AlarmDefinition[];
    total: number;
    page: number;
    pageSize: number;
  }> {
    const pageSize = filter?.pageSize && filter.pageSize > 0
      ? Math.min(filter.pageSize, 500)
      : 500;
    const items: AlarmDefinition[] = [];
    let page = 1;
    let total = 0;

    while (true) {
      const result = await alarmDefinitionApi.list({
        ...filter,
        page,
        pageSize,
      });
      total = result.total;
      items.push(...result.items);

      if (result.items.length === 0 || items.length >= total) {
        break;
      }
      page += 1;
    }

    return {
      items,
      total,
      page: 1,
      pageSize: items.length,
    };
  },

  async get(identifier: string): Promise<AlarmDefinition> {
    const { data } = await http.get<BackendDefinition>(
      `/alarm-definitions/${encodeURIComponent(identifier)}`
    );
    return mapDef(data);
  },

  async create(input: CreateAlarmDefinitionInput): Promise<AlarmDefinition> {
    const { data } = await http.post<BackendDefinition>('/alarm-definitions', defPayload(input));
    return mapDef(data);
  },

  async update(identifier: string, input: UpdateAlarmDefinitionInput): Promise<AlarmDefinition> {
    const { data } = await http.put<BackendDefinition>(
      `/alarm-definitions/${encodeURIComponent(identifier)}`,
      defPayload(input)
    );
    return mapDef(data);
  },

  async delete(identifier: string): Promise<void> {
    await http.delete(`/alarm-definitions/${encodeURIComponent(identifier)}`);
  },

  async unknownStats(filter?: UnknownStatsFilter): Promise<{ items: UnknownAlarmStat[]; days: number }> {
    const params: Record<string, unknown> = {};
    if (filter?.productId) params.productId = filter.productId;
    if (filter?.days) params.days = filter.days;
    const { data } = await http.get<{ items: BackendUnknownStat[]; days: number }>(
      '/alarm-definitions/unknown-stats',
      { params }
    );
    return {
      items: (data.items || []).map(mapUnknown),
      days: data.days || 7,
    };
  },

  async listNeTypes(): Promise<{ items: AlarmNeTypeStat[] }> {
    const { data } = await http.get<{ items: BackendNeTypeStat[] }>('/alarm-definitions/ne-types');
    return { items: (data.items || []).map(mapNeTypeStat) };
  },

  async severityLevels(): Promise<{ items: AlarmSeverityLevel[] }> {
    const { data } = await http.get<{ items: BackendSeverityLevel[] }>('/alarm-severity-levels');
    return {
      items: (data.items || []).map(mapSeverity),
    };
  },

  /** 上传自定义告警 XML(multipart)。名称取自 XML neType 属性(2026-06-05 取消手填 name)。
   *  重复允许覆盖(2026-06-05 调整):不带 force 时重复返 409(data.overwritable=true),
   *  前端弹二次确认后带 force=true 重试 → 覆盖归属文件(旧文件自动备份 .bak.<ts>)。
   *  后端上传端点内部已自动 destructive 重载(删孤儿)+ 刷新缓存。 */
  async uploadXml(file: File, force = false): Promise<AlarmUploadResult> {
    const form = new FormData();
    form.append('file', file);
    const { data } = await http.post<{
      uploaded: boolean;
      filename: string;
      loaded_from: string;
      ne_type: string;
      overwritten: boolean;
      reloaded: boolean;
    }>('/alarm-definitions/upload-xml', form, {
      params: force ? { force: 'true' } : undefined,
      headers: { 'Content-Type': 'multipart/form-data' },
    });
    return {
      uploaded: data.uploaded,
      filename: data.filename,
      loadedFrom: data.loaded_from,
      neType: data.ne_type,
      overwritten: data.overwritten,
      reloaded: data.reloaded,
    };
  },

  /** 下载告警 XML 原文件(builtin / custom 均可,2026-06-05 操作列下载功能)。 */
  async downloadXml(loadedFrom: string): Promise<void> {
    const resp = await http.get('/alarm-definitions/file-content', {
      params: { loaded_from: loadedFrom },
      responseType: 'blob',
    });
    saveBlob(resp.data as BlobPart, loadedFrom.split('/').pop() || 'alarm.xml');
  },

  /** 删除自定义告警 XML(loadedFrom 含 / 须 encodeURIComponent)。仅 custom 可删,内置后端返 403。 */
  async deleteFile(loadedFrom: string): Promise<AlarmDeleteFileResult> {
    const { data } = await http.delete<{
      deleted: boolean;
      loaded_from: string;
      rows_affected: number;
      backup: string;
    }>(`/alarm-definitions/files/${encodeURIComponent(loadedFrom)}`);
    return {
      deleted: data.deleted,
      loadedFrom: data.loaded_from,
      rowsAffected: data.rows_affected,
      backup: data.backup,
    };
  },
};
