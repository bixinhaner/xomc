import http from '../http';
import { saveBlob } from '../../utils/saveBlob';
import type {
  ParamModel,
  ParamModelSource,
  ParamMapping,
  StandardParam,
  DiscoveredVersion,
  TranslateRequest,
  TranslateResponse,
  CreateMappingInput,
  UpdateMappingInput,
  UpsertStandardInput,
  UpdateParamModelInput,
  StandardParamFilter,
} from '../../types/paramModel';

interface BackendParamModel {
  id: string;
  name: string;
  total_entries: number;
  total_objects: number;
  total_params: number;
  description: string;
  is_active: boolean;
  loaded_from: string;
  // T-0178:后端 handler.go::toModelView 派生的 source / deletable 真值源,
  // 前端"来源"列 Tag + 删除按钮可见性都依赖这两个字段。
  // mapBackendParamModel 必须把它们映射出去,否则 ModelsTab 的
  // .filter((m) => (m.source ?? 'unknown') !== 'unknown') 会把全部数据过滤掉。
  source?: 'builtin' | 'custom' | 'unknown';
  deletable?: boolean;
  created_at?: string;
  updated_at?: string;
}

interface BackendMapping {
  id: string;
  param_model_id: string;
  standard_path: string;
  private_path: string;
  entry_type: string;
  access: string;
  data_type: string;
  change_applies: string;
  min_value?: string;
  max_value?: string;
  enum_values?: string;
  enum_labels?: string;
  validation_pattern?: string;
  is_storable: boolean;
  is_active: boolean;
  software_version?: string;
  source?: string; // T-PMSRC: 'builtin' | 'custom'
  deletable?: boolean; // T-PMSRC: 仅 custom 可删
}

interface BackendStandardParam {
  standard_path: string;
  entry_type: string;
  access: string;
  data_type: string;
  change_applies: string;
  min_value?: string;
  max_value?: string;
  updated_at: string;
  updated_fields?: string[];
}

interface BackendDiscoveredVersion {
  software_version: string;
  total_rows: number;
  last_seen_at?: string;
}

interface BackendTranslateItem {
  source: string;
  target?: string;
  matched: boolean;
  from_discovered?: boolean;
  reason?: string;
}

function mapModel(b: BackendParamModel): ParamModel {
  return {
    id: b.id,
    name: b.name,
    totalEntries: b.total_entries,
    totalObjects: b.total_objects,
    totalParams: b.total_params,
    description: b.description,
    isActive: b.is_active,
    loadedFrom: b.loaded_from,
    // 2026-05-29 修复:漏映射 source / deletable 导致 ModelsTab.filter 把所有
    // 行当 unknown 隐藏 — 用户实测页面变空(DB 9 行 + API 200 但 UI 0 条)。
    source: b.source,
    deletable: b.deletable,
    createdAt: b.created_at,
    updatedAt: b.updated_at,
  };
}

function mapMapping(b: BackendMapping): ParamMapping {
  return {
    id: b.id,
    paramModelId: b.param_model_id,
    standardPath: b.standard_path,
    privatePath: b.private_path,
    entryType: b.entry_type,
    access: b.access,
    dataType: b.data_type,
    changeApplies: b.change_applies,
    minValue: b.min_value,
    maxValue: b.max_value,
    enumValues: b.enum_values,
    enumLabels: b.enum_labels,
    validationPattern: b.validation_pattern,
    isStorable: b.is_storable,
    isActive: b.is_active,
    softwareVersion: b.software_version,
    source: (b.source as ParamModelSource) ?? 'builtin',
    deletable: b.deletable ?? false,
  };
}

function mapStandard(b: BackendStandardParam): StandardParam {
  return {
    standardPath: b.standard_path,
    entryType: b.entry_type,
    access: b.access,
    dataType: b.data_type,
    changeApplies: b.change_applies,
    minValue: b.min_value,
    maxValue: b.max_value,
    updatedAt: b.updated_at,
    updatedFields: b.updated_fields ?? [],
  };
}

function mapDiscovered(b: BackendDiscoveredVersion): DiscoveredVersion {
  return {
    softwareVersion: b.software_version,
    totalRows: b.total_rows,
    lastSeenAt: b.last_seen_at,
  };
}

function mappingPayload(input: CreateMappingInput | UpdateMappingInput): Record<string, unknown> {
  const p: Record<string, unknown> = {};
  if (input.standardPath !== undefined) p.standard_path = input.standardPath;
  if (input.privatePath !== undefined) p.private_path = input.privatePath;
  if (input.entryType !== undefined) p.entry_type = input.entryType;
  if (input.access !== undefined) p.access = input.access;
  if (input.dataType !== undefined) p.data_type = input.dataType;
  if (input.changeApplies !== undefined) p.change_applies = input.changeApplies;
  if (input.minValue !== undefined) p.min_value = input.minValue;
  if (input.maxValue !== undefined) p.max_value = input.maxValue;
  if (input.isStorable !== undefined) p.is_storable = input.isStorable;
  if (input.isActive !== undefined) p.is_active = input.isActive;
  if (input.softwareVersion !== undefined) p.software_version = input.softwareVersion;
  return p;
}

function standardPayload(input: UpsertStandardInput): Record<string, unknown> {
  return {
    standard_path: input.standardPath,
    entry_type: input.entryType,
    access: input.access,
    data_type: input.dataType,
    change_applies: input.changeApplies,
    min_value: input.minValue,
    max_value: input.maxValue,
  };
}

export const paramModelApi = {
  async list(): Promise<{ items: ParamModel[]; total: number }> {
    const { data } = await http.get<{ items: BackendParamModel[]; total: number }>('/param-models');
    return {
      items: (data.items || []).map(mapModel),
      total: data.total || 0,
    };
  },

  async get(name: string): Promise<ParamModel> {
    const { data } = await http.get<BackendParamModel>(`/param-models/${encodeURIComponent(name)}`);
    return mapModel(data);
  },

  async update(name: string, input: UpdateParamModelInput): Promise<ParamModel> {
    const payload: Record<string, unknown> = {};
    if (input.description !== undefined) payload.description = input.description;
    if (input.isActive !== undefined) payload.is_active = input.isActive;
    const { data } = await http.put<BackendParamModel>(
      `/param-models/${encodeURIComponent(name)}`,
      payload
    );
    return mapModel(data);
  },

  async delete(name: string): Promise<void> {
    await http.delete(`/param-models/${encodeURIComponent(name)}`);
  },

  async listMappings(name: string): Promise<{ items: ParamMapping[]; total: number; paramModel?: string }> {
    const { data } = await http.get<{ items: BackendMapping[]; total: number; param_model?: string }>(
      `/param-models/${encodeURIComponent(name)}/mappings`
    );
    return {
      items: (data.items || []).map(mapMapping),
      total: data.total || 0,
      paramModel: data.param_model,
    };
  },

  async createMapping(name: string, input: CreateMappingInput): Promise<ParamMapping> {
    const { data } = await http.post<BackendMapping>(
      `/param-models/${encodeURIComponent(name)}/mappings`,
      mappingPayload(input)
    );
    return mapMapping(data);
  },

  async updateMapping(name: string, id: string, input: UpdateMappingInput): Promise<ParamMapping> {
    const { data } = await http.put<BackendMapping>(
      `/param-models/${encodeURIComponent(name)}/mappings/${id}`,
      mappingPayload(input)
    );
    return mapMapping(data);
  },

  async deleteMapping(name: string, id: string): Promise<void> {
    await http.delete(`/param-models/${encodeURIComponent(name)}/mappings/${id}`);
  },

  async listStandard(filter?: StandardParamFilter): Promise<{ items: StandardParam[]; total: number }> {
    const params: Record<string, unknown> = {};
    if (filter?.keyword) params.keyword = filter.keyword;
    if (filter?.entryType) params.entry_type = filter.entryType;
    const { data } = await http.get<{ items: BackendStandardParam[]; total: number }>(
      '/param-models/standard',
      { params }
    );
    return {
      items: (data.items || []).map(mapStandard),
      total: data.total || 0,
    };
  },

  async getStandard(path: string): Promise<StandardParam> {
    const { data } = await http.get<BackendStandardParam>(
      `/param-models/standard/${encodeURIComponent(path)}`
    );
    return mapStandard(data);
  },

  async createStandard(input: UpsertStandardInput): Promise<StandardParam> {
    const { data } = await http.post<BackendStandardParam>('/param-models/standard', standardPayload(input));
    return mapStandard(data);
  },

  async updateStandard(path: string, input: UpsertStandardInput): Promise<StandardParam> {
    const { data } = await http.put<BackendStandardParam>(
      `/param-models/standard/${encodeURIComponent(path)}`,
      standardPayload(input)
    );
    return mapStandard(data);
  },

  async deleteStandard(path: string): Promise<void> {
    await http.delete(`/param-models/standard/${encodeURIComponent(path)}`);
  },

  async translate(request: TranslateRequest): Promise<TranslateResponse> {
    const { data } = await http.post<{
      results: BackendTranslateItem[];
      product_id: string;
      software_version?: string;
      direction: TranslateRequest['direction'];
      source?: string;
    }>('/param-models/translate', {
      product_id: request.productId,
      software_version: request.softwareVersion,
      direction: request.direction,
      paths: request.paths,
    });
    return {
      productId: data.product_id,
      softwareVersion: data.software_version,
      direction: data.direction,
      source: data.source,
      results: (data.results || []).map((r) => ({
        source: r.source,
        target: r.target,
        matched: r.matched,
        fromDiscovered: r.from_discovered,
        reason: r.reason,
      })),
    };
  },

  async listDiscovered(productId: string, swVersion?: string): Promise<{ items: ParamMapping[]; total: number }> {
    const params: Record<string, unknown> = {};
    if (swVersion) params.swVersion = swVersion;
    const { data } = await http.get<{ items: BackendMapping[]; total: number }>(
      `/products/${productId}/discovered`,
      { params }
    );
    return {
      items: (data.items || []).map(mapMapping),
      total: data.total || 0,
    };
  },

  async listDiscoveredVersions(productId: string): Promise<{ items: DiscoveredVersion[]; total: number }> {
    const { data } = await http.get<{ items: BackendDiscoveredVersion[]; total: number }>(
      `/products/${productId}/discovered/versions`
    );
    return {
      items: (data.items || []).map(mapDiscovered),
      total: data.total || 0,
    };
  },

  async deleteDiscovered(productId: string, swVersion?: string): Promise<{ deleted: number }> {
    const url = swVersion
      ? `/products/${productId}/discovered/versions/${encodeURIComponent(swVersion)}`
      : `/products/${productId}/discovered`;
    const { data } = await http.delete<{ deleted: number }>(url);
    return data;
  },

  // 下载参数模型 XML 原文件(builtin / custom 均可,2026-06-05 操作列下载功能)。
  async downloadXml(loadedFrom: string): Promise<void> {
    const resp = await http.get('/param-models/file-content', {
      params: { loaded_from: loadedFrom },
      responseType: 'blob',
    });
    saveBlob(resp.data as BlobPart, loadedFrom.split('/').pop() || 'param-model.xml');
  },

  // 上传自定义 paramModel XML(multipart/form-data)。
  // 导入 XML 调整(2026-06-05 取消手填名称):
  //   - 仅传 file;名称取自 XML <parameterModel paramModel="..."> 属性,
  //     目标文件名 = <paramModel>.xml,上传文件自身的 filename 被忽略。
  //   - 重复允许覆盖:不带 force 时重复返 409(data.overwritable=true),前端弹
  //     二次确认后带 force=true 重试 → 覆盖归属文件(旧文件自动备份 .bak.<ts>)。
  //   - 后端上传端点内部自动 destructive 重载(删孤儿)+ 刷新缓存。
  //   - 后端校验:大小 ≤ 1MiB + XML 根元素 = parameterModel + paramModel 推导文件名白名单 + 路径包含。
  async uploadXML(
    file: File,
    force = false,
  ): Promise<{ filename: string; modelName: string; size: number; overwritten: boolean }> {
    const fd = new FormData();
    fd.append('file', file);
    const { data } = await http.post<{
      filename: string;
      model_name: string;
      size: number;
      overwritten: boolean;
    }>('/param-models/upload-xml', fd, {
      params: force ? { force: 'true' } : undefined,
      headers: { 'Content-Type': 'multipart/form-data' },
    });
    return {
      filename: data.filename,
      modelName: data.model_name,
      size: data.size,
      overwritten: data.overwritten,
    };
  },
};
