import http from '../http';
import type {
  IndicatorInfo,
  IndicatorListFilter,
  IndicatorGroup,
  PlatformFormula,
  IndicatorUnit,
  CreateIndicatorInput,
  UpdateIndicatorInput,
  CreateGroupInput,
  UpdateGroupInput,
  EnabledIndicatorsRequest,
  UnitInput,
  DeviceType,
  IndicatorPlatformSummary,
  IndicatorFile,
  IndicatorReloadMode,
  IndicatorReloadResult,
  IndicatorUploadResult,
  IndicatorDeleteFileResult,
  TechLower,
} from '../../types/indicatorLibrary';

interface BackendIndicator {
  id: string;
  name: string;
  cn_name?: string;
  en_name?: string;
  group_id?: string;
  group_name?: string;
  counter_type?: string;
  indicator_level?: string;
  unit?: string;
  description?: string;
  // 后端实际下发字符串 '0' / '1'（非布尔）；需归一化，否则 JS 里非空字符串 '0' 也是真值。
  is_counter?: boolean | number | string;
  product_type?: string;
  operator_code?: string;
  is_enabled?: boolean;
  device_type?: DeviceType;
}

interface BackendGroup {
  id: string;
  name: string;
  parent_id?: string;
  description?: string;
  operator_code?: string;
  device_type?: DeviceType;
  children?: BackendGroup[];
}

interface BackendFormula {
  platform_name: string;
  indicator_id: string;
  formula: string;
  description?: string;
}

interface BackendUnit {
  id: string;
  en_name: string;
  cn_name: string;
}

function mapIndicator(b: BackendIndicator, deviceType: DeviceType): IndicatorInfo {
  return {
    id: b.id,
    name: b.name,
    cnName: b.cn_name,
    enName: b.en_name,
    groupId: b.group_id,
    groupName: b.group_name,
    counterType: b.counter_type,
    indicatorLevel: b.indicator_level,
    unit: b.unit,
    description: b.description,
    // 归一化：后端发 '0'/'1' 字符串（与 indicatorApi.ts 一致），不能直接当布尔用
    isCounter: b.is_counter === true || b.is_counter === 1 || b.is_counter === '1' || b.is_counter === 'true',
    productClass: b.product_type,
    operatorCode: b.operator_code,
    isEnabled: b.is_enabled,
    deviceType: b.device_type ?? deviceType,
  };
}

function mapGroup(b: BackendGroup, deviceType: DeviceType): IndicatorGroup {
  return {
    id: b.id,
    name: b.name,
    parentId: b.parent_id,
    description: b.description,
    operatorCode: b.operator_code,
    deviceType: b.device_type ?? deviceType,
    children: b.children?.map((c) => mapGroup(c, deviceType)),
  };
}

function mapFormula(b: BackendFormula): PlatformFormula {
  return {
    platformName: b.platform_name,
    indicatorId: b.indicator_id,
    formula: b.formula,
    description: b.description,
  };
}

function mapUnit(b: BackendUnit): IndicatorUnit {
  return { id: b.id, enName: b.en_name, cnName: b.cn_name };
}

function indicatorPayload(
  input: CreateIndicatorInput | UpdateIndicatorInput
): Record<string, unknown> {
  const p: Record<string, unknown> = {};
  if ('id' in input && input.id !== undefined) p.id = input.id;
  if (input.name !== undefined) p.name = input.name;
  if (input.cnName !== undefined) p.cn_name = input.cnName;
  if (input.enName !== undefined) p.en_name = input.enName;
  if (input.groupId !== undefined) p.group_id = input.groupId;
  if (input.counterType !== undefined) p.counter_type = input.counterType;
  if (input.indicatorLevel !== undefined) p.indicator_level = input.indicatorLevel;
  if (input.unit !== undefined) p.unit = input.unit;
  if (input.description !== undefined) p.description = input.description;
  if (input.productClass !== undefined) p.product_type = input.productClass;
  if (input.operatorCode !== undefined) p.operator_code = input.operatorCode;
  return p;
}

function groupPayload(input: CreateGroupInput | UpdateGroupInput): Record<string, unknown> {
  const p: Record<string, unknown> = {};
  if ('id' in input && input.id !== undefined) p.id = input.id;
  if (input.name !== undefined) p.name = input.name;
  if (input.parentId !== undefined) p.parent_id = input.parentId;
  if (input.description !== undefined) p.description = input.description;
  if (input.operatorCode !== undefined) p.operator_code = input.operatorCode;
  return p;
}

export const indicatorLibraryApi = {
  async list(deviceType: DeviceType, filter?: IndicatorListFilter): Promise<{
    items: IndicatorInfo[];
    total: number;
  }> {
    const params: Record<string, unknown> = { deviceType };
    if (filter?.groupId) params.groupId = filter.groupId;
    if (filter?.keyword) params.keyword = filter.keyword;
    if (filter?.operatorCode) params.operatorCode = filter.operatorCode;
    if (filter?.productClass) params.productClass = filter.productClass;
    if (filter?.indicatorLevel) params.indicatorLevel = filter.indicatorLevel;
    if (filter?.isEnabled !== undefined) params.isEnabled = filter.isEnabled;
    if (filter?.isCounter !== undefined) params.isCounter = filter.isCounter;
    if (filter?.platformName) params.platformName = filter.platformName;
    if (filter?.page) params.page = filter.page;
    if (filter?.pageSize) params.pageSize = filter.pageSize;

    const { data } = await http.get<{ items: BackendIndicator[]; total: number }>(
      '/indicators',
      { params }
    );
    return {
      items: (data.items || []).map((b) => mapIndicator(b, deviceType)),
      total: data.total || 0,
    };
  },

  async get(deviceType: DeviceType, id: string): Promise<IndicatorInfo> {
    const { data } = await http.get<BackendIndicator>(`/indicators/${id}`, { params: { deviceType } });
    return mapIndicator(data, deviceType);
  },

  async listPlatforms(deviceType: DeviceType): Promise<{ items: string[]; total: number }> {
    const { data } = await http.get<{ items: string[]; total: number }>('/indicators/platforms', {
      params: { deviceType },
    });
    return { items: data.items || [], total: data.total || 0 };
  },

  async create(deviceType: DeviceType, input: CreateIndicatorInput): Promise<IndicatorInfo> {
    const { data } = await http.post<BackendIndicator>('/indicators', indicatorPayload(input), {
      params: { deviceType },
    });
    return mapIndicator(data, deviceType);
  },

  async update(deviceType: DeviceType, id: string, input: UpdateIndicatorInput): Promise<IndicatorInfo> {
    const { data } = await http.put<BackendIndicator>(`/indicators/${id}`, indicatorPayload(input), {
      params: { deviceType },
    });
    return mapIndicator(data, deviceType);
  },

  async delete(deviceType: DeviceType, id: string): Promise<void> {
    await http.delete(`/indicators/${id}`, { params: { deviceType } });
  },

  async listFormulas(deviceType: DeviceType, indicatorId: string): Promise<{ items: PlatformFormula[]; total: number }> {
    const { data } = await http.get<{ items: BackendFormula[]; total: number }>(
      `/indicators/${indicatorId}/formulas`,
      { params: { deviceType } }
    );
    return {
      items: (data.items || []).map(mapFormula),
      total: data.total || 0,
    };
  },

  async getFormula(deviceType: DeviceType, indicatorId: string, platform: string): Promise<PlatformFormula> {
    const { data } = await http.get<BackendFormula>(
      `/indicators/${indicatorId}/formulas/${encodeURIComponent(platform)}`,
      { params: { deviceType } }
    );
    return mapFormula(data);
  },

  async upsertFormula(
    deviceType: DeviceType,
    indicatorId: string,
    platform: string,
    formula: string
  ): Promise<{ indicatorId: string; platform: string; formula: string }> {
    const { data } = await http.post<{ indicator_id: string; platform: string; formula: string }>(
      `/indicators/${indicatorId}/formulas`,
      { platform, formula },
      { params: { deviceType } }
    );
    return { indicatorId: data.indicator_id, platform: data.platform, formula: data.formula };
  },

  async deleteFormula(deviceType: DeviceType, indicatorId: string, platform: string): Promise<void> {
    await http.delete(`/indicators/${indicatorId}/formulas/${encodeURIComponent(platform)}`, {
      params: { deviceType },
    });
  },

  async listGroups(deviceType: DeviceType, operatorCode?: string): Promise<{ items: IndicatorGroup[]; total: number }> {
    const params: Record<string, unknown> = { deviceType };
    if (operatorCode) params.operatorCode = operatorCode;
    const { data } = await http.get<{ items: BackendGroup[]; total: number }>('/indicator-groups', { params });
    return {
      items: (data.items || []).map((b) => mapGroup(b, deviceType)),
      total: data.total || 0,
    };
  },

  async createGroup(deviceType: DeviceType, input: CreateGroupInput): Promise<IndicatorGroup> {
    const { data } = await http.post<BackendGroup>('/indicator-groups', groupPayload(input), {
      params: { deviceType },
    });
    return mapGroup(data, deviceType);
  },

  async updateGroup(deviceType: DeviceType, id: string, input: UpdateGroupInput): Promise<IndicatorGroup> {
    const { data } = await http.put<BackendGroup>(`/indicator-groups/${id}`, groupPayload(input), {
      params: { deviceType },
    });
    return mapGroup(data, deviceType);
  },

  async deleteGroup(deviceType: DeviceType, id: string): Promise<void> {
    await http.delete(`/indicator-groups/${id}`, { params: { deviceType } });
  },

  async listEnabled(deviceType: DeviceType, operatorCode: string): Promise<{
    items: string[];
    total: number;
    operatorCode: string;
  }> {
    const { data } = await http.get<{ items: string[]; total: number; operator_code: string }>(
      '/enabled-indicators',
      { params: { deviceType, operatorCode } }
    );
    return {
      items: data.items || [],
      total: data.total || 0,
      operatorCode: data.operator_code,
    };
  },

  async setEnabled(request: EnabledIndicatorsRequest): Promise<{ updated: number; operatorCode: string; enable: boolean }> {
    const { data } = await http.put<{ updated: number; operator_code: string; enable: boolean }>(
      '/enabled-indicators',
      { indicator_ids: request.indicatorIds, enable: request.enable },
      { params: { deviceType: request.deviceType, operatorCode: request.operatorCode } }
    );
    return {
      updated: data.updated,
      operatorCode: data.operator_code,
      enable: data.enable,
    };
  },

  async listUnits(): Promise<{ items: IndicatorUnit[]; total: number }> {
    const { data } = await http.get<{ items: BackendUnit[]; total: number }>('/indicator-units');
    return {
      items: (data.items || []).map(mapUnit),
      total: data.total || 0,
    };
  },

  async upsertUnit(input: UnitInput): Promise<IndicatorUnit> {
    const { data } = await http.post<BackendUnit>('/indicator-units', {
      id: input.id,
      en_name: input.enName,
      cn_name: input.cnName,
    });
    return mapUnit(data);
  },

  async updateUnit(id: string, input: UnitInput): Promise<IndicatorUnit> {
    const { data } = await http.put<BackendUnit>(`/indicator-units/${id}`, {
      id: input.id,
      en_name: input.enName,
      cn_name: input.cnName,
    });
    return mapUnit(data);
  },

  async deleteUnit(id: string): Promise<void> {
    await http.delete(`/indicator-units/${id}`);
  },

  async cacheRefresh(): Promise<{ refreshed: boolean; note?: string }> {
    const { data } = await http.post<{ refreshed: boolean; note?: string }>('/indicators/cache/refresh');
    return data;
  },

  // T-0180 P1.5: import-directory 加 ?mode= 路由
  //   "import"(默认) — 加法 UPSERT,不删孤儿(向后兼容)
  //   "reload"        — destructive 全量重载 + 三制式孤儿删除
  async importDirectory(mode: IndicatorReloadMode = 'import'): Promise<IndicatorReloadResult> {
    const { data } = await http.post<{
      reloaded: string;
      mode: string;
      orphans?: Record<string, number>;
    }>(`/indicators/import-directory?mode=${mode}`);
    return {
      reloaded: data.reloaded,
      mode: (data.mode as IndicatorReloadMode) ?? mode,
      orphans: data.orphans as Record<TechLower, number> | undefined,
    };
  },

  // T-0180 P1.4(2026-05-29 用户调整粒度):一级 SummaryTab 数据源 — (制式, 平台) 一行
  // 后端返 { tech, platform, loaded_from, indicators },前端原样映射 camelCase
  async summary(): Promise<{ items: IndicatorPlatformSummary[] }> {
    const { data } = await http.get<{
      items: Array<{
        tech: string;
        platform: string;
        loaded_from: string;
        indicators: number;
      }>;
    }>('/indicators/summary');
    return {
      items: (data.items || []).map((b) => ({
        tech: b.tech as TechLower,
        platform: b.platform,
        loadedFrom: b.loaded_from,
        indicators: b.indicators,
      })),
    };
  },

  // T-0180 P1.4: 列出指定 tech 下所有 XML 文件(DB 计数 + 物理盘扫描合并)
  async listFiles(tech: TechLower): Promise<{ items: IndicatorFile[]; tech: TechLower }> {
    const { data } = await http.get<{
      items: Array<{
        loaded_from: string;
        source: string;
        deletable: boolean;
        count: number;
        on_disk: boolean;
      }>;
      tech: string;
    }>('/indicators/files', { params: { tech } });
    return {
      tech: (data.tech as TechLower) ?? tech,
      items: (data.items || []).map((b) => ({
        loadedFrom: b.loaded_from,
        source: b.source as IndicatorFile['source'],
        deletable: b.deletable,
        count: b.count,
        onDisk: b.on_disk,
      })),
    };
  },

  // T-0180 P1.4: multipart 上传自定义 XML
  // file = File 对象(浏览器 FormData);返回 backend 同结构 UploadXmlResult
  async uploadXml(
    tech: TechLower,
    file: File,
    options: { force?: boolean } = {}
  ): Promise<IndicatorUploadResult> {
    const form = new FormData();
    form.append('file', file);
    const params: Record<string, string> = { tech };
    if (options.force) params.force = 'true';
    const { data } = await http.post<{
      uploaded: boolean;
      filename: string;
      loaded_from: string;
      tech: string;
      overwrite: boolean;
      backup: string;
      reloaded: boolean;
    }>('/indicators/upload-xml', form, {
      params,
      // FormData 自带 multipart/form-data;axios 自动处理 boundary
    });
    return {
      uploaded: data.uploaded,
      filename: data.filename,
      loadedFrom: data.loaded_from,
      tech: data.tech as TechLower,
      overwrite: data.overwrite,
      backup: data.backup,
      reloaded: data.reloaded,
    };
  },

  // T-0180 P1.3: 按 loadedFrom 删自定义 XML(内置返 403);URL path 携带完整 loadedFrom
  async deleteFile(loadedFrom: string): Promise<IndicatorDeleteFileResult> {
    const { data } = await http.delete<{
      deleted: boolean;
      loaded_from: string;
      tech: string;
      rows_affected: number;
      backup: string;
    }>(`/indicators/files/${loadedFrom}`);
    return {
      deleted: data.deleted,
      loadedFrom: data.loaded_from,
      tech: data.tech as TechLower,
      rowsAffected: data.rows_affected,
      backup: data.backup,
    };
  },
};
