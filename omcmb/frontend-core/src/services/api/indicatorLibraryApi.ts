import http from '../http';
import type {
  IndicatorInfo,
  IndicatorListFilter,
  IndicatorGroup,
  PlatformFormula,
  CreateIndicatorInput,
  UpdateIndicatorInput,
  CreateGroupInput,
  UpdateGroupInput,
  EnabledIndicatorsRequest,
  DeviceType,
  IndicatorPlatformSummary,
  IndicatorFile,
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
  // PM-P3:编号版公式(perf_indicators_*.arithmetic),后端 PerfIndicator JSON 已带 arithmetic。
  arithmetic?: string;
  product_type?: string;
  operator_code?: string;
  is_enabled?: boolean;
  device_type?: DeviceType;
}

interface BackendGroup {
  id: string;
  // 2026-05-29:后端 IndicatorGroup struct 字段是 EnName / CnName(JSON:
  // en_name / cn_name),没有 `name` 字段。前端 mapGroup 合成
  // name = cn_name || en_name || id 给下拉 label 用。`name` 保留可选,
  // 兼容未来后端补字段;DB 表 indicator_group_enb/gsm/gnb 也只有 en_name/cn_name。
  name?: string;
  en_name?: string;
  cn_name?: string;
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
    arithmetic: b.arithmetic,
    productClass: b.product_type,
    operatorCode: b.operator_code,
    isEnabled: b.is_enabled,
    deviceType: b.device_type ?? deviceType,
  };
}

function mapGroup(b: BackendGroup, deviceType: DeviceType): IndicatorGroup {
  return {
    id: b.id,
    // 2026-05-29 修复:后端只给 en_name/cn_name,前端原本读 b.name 永远 undefined
    // 导致分组下拉显示为 id 哈希(用户实测列表里看到 HO/EQPT,下拉里全是 hex)。
    // 优先中文名,再 fallback 英文,最后 id。
    name: b.cn_name || b.en_name || b.name || b.id,
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

  /** 2026-05-29:加 platform 过滤 — 详情态(?tech=&platform=)下拉只显示当前
   *  platform 实际涉及的分组,避免下拉里有 23 个 group 但选 16 个都返空。
   *  后端 EXISTS 嵌套 EXISTS 与 ListIndicators 的 PlatformName 同语义。 */
  async listGroups(
    deviceType: DeviceType,
    operatorCode?: string,
    platform?: string,
  ): Promise<{ items: IndicatorGroup[]; total: number }> {
    const params: Record<string, unknown> = { deviceType };
    if (operatorCode) params.operatorCode = operatorCode;
    if (platform) params.platform = platform;
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

  // 一级 SummaryTab 数据源 — "一个平台一条"(2026-06-02 用户决策)
  // 后端返 { tech, platform, indicators, description },映射 camelCase
  async summary(): Promise<{ items: IndicatorPlatformSummary[] }> {
    const { data } = await http.get<{
      items: Array<{
        tech: string;
        platform: string;
        indicators: number;
        loaded_from?: string;
        source?: string;
        description?: string;
      }>;
    }>('/indicators/summary');
    return {
      items: (data.items || []).map((b) => ({
        tech: b.tech as TechLower,
        platform: b.platform,
        indicators: b.indicators,
        loadedFrom: b.loaded_from ?? '',
        source: (b.source ?? 'unknown') as IndicatorPlatformSummary['source'],
        description: b.description ?? '',
      })),
    };
  },

  // 2026-06-02:按 (tech, platform) upsert 描述。
  async updateFileDescription(tech: TechLower, platform: string, description: string): Promise<void> {
    await http.put('/indicators/file-description', { tech, platform, description });
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
      // 必须显式声明 multipart/form-data — http.ts axios.create 设了
      // 默认 'Content-Type': 'application/json',不显式覆盖会沿用 JSON
      // 导致 body 被序列化为 "{}" + Gin c.FormFile("file") 返
      // "Content-Type isn't multipart/form-data"(与 paramModelApi.uploadXML / fileApi /
      // adminApi / softwareApi 等 8 处上传同范式;历史漏掉本处)。
      headers: { 'Content-Type': 'multipart/form-data' },
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
