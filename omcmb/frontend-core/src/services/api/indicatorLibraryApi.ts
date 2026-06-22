import http from '../http';
import { saveBlob } from '../../utils/saveBlob';
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
  // #193：后端 PerfIndicator 实际下发 data_type（受控码 int/real/float）+
  // data_type_label（按 locale 本地化的"整数/实数/浮点数"，#161/#67 §4 由 handler 填充）。
  // 旧前端读 counter_type/unit 两个后端从不存在的字段，故详情恒显"—"。
  data_type?: string;
  data_type_label?: string;
  indicator_level?: string;
  unit_id?: string;
  description?: string;
  // 后端实际下发字符串 '0' / '1'（非布尔）；需归一化，否则 JS 里非空字符串 '0' 也是真值。
  is_counter?: boolean | number | string;
  // PM-P3:编号版公式(perf_indicators_*.arithmetic),后端 PerfIndicator JSON 已带 arithmetic。
  arithmetic?: string;
  product_type?: string;
  // 后端 PerfIndicator 实际字段是 product_types（复数）；product_type 为旧别名兼容。
  product_types?: string;
  operator_code?: string;
  is_enabled?: boolean;
  device_type?: DeviceType;
  // 后端下发字符串 '0'/'1'：'1' 表示内置指标（编辑时归属分组只读，XML 真相源覆盖）。
  is_build_in?: string;
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
  // 后端下发字符串 '0'/'1'：'1' 表示内置组（禁删、禁编辑）。
  is_build_in?: string;
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
    // #193：优先本地化标签，回退原始受控码；counter 型才有值，派生 KPI 型为 undefined
    // （XML 无 dataType → data_type=NULL），抽屉据 isCounter 决定是否展示该字段。
    counterType: b.data_type_label ?? b.data_type,
    indicatorLevel: b.indicator_level,
    // #193：详情接口当前只回原始 unit_id 码（%/ppm/number），先显示原始码消除"—"；
    // 本地化（百分比/百万分比/个）为后续 UX 增强（需后端 +unit_label 字段）。
    unit: b.unit_id,
    description: b.description,
    // 归一化：后端发 '0'/'1' 字符串（与 indicatorApi.ts 一致），不能直接当布尔用
    isCounter: b.is_counter === true || b.is_counter === 1 || b.is_counter === '1' || b.is_counter === 'true',
    arithmetic: b.arithmetic,
    productClass: b.product_types ?? b.product_type,
    operatorCode: b.operator_code,
    isEnabled: b.is_enabled,
    deviceType: b.device_type ?? deviceType,
    // 归一化：后端发 '0'/'1' 字符串，'1' = 内置指标（编辑时归属分组只读）。
    isBuildIn: b.is_build_in === '1' || b.is_build_in === 'true',
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
    isBuildIn: b.is_build_in === '1',
    children: b.children?.map((c) => mapGroup(c, deviceType)),
  };
}

/**
 * 把分组树拍平为「带缩进层级」的下拉选项 —— 所有选择分组的位置(分组筛选 / 归属分组 /
 * 父分组)统一用它，按树结构(缩进)展示而非扁平列表。label 以全角空格按 depth 缩进。
 * excludeId:排除该节点**及其整棵子树**(父分组下拉用，避免把自己/后代选作父，形成环)。
 *
 * 2026-06-22 起,UI 三皮肤改用真·树形选择器(可展开收起,见 groupTreeData),本函数
 * 仅作为兼容备份保留;新代码请用 groupTreeData。
 */
export function groupTreeOptions(
  nodes: IndicatorGroup[] | undefined,
  opts?: { excludeId?: string },
): { label: string; value: string; depth: number }[] {
  const out: { label: string; value: string; depth: number }[] = [];
  const walk = (list: IndicatorGroup[] | undefined, depth: number) => {
    (list || []).forEach((n) => {
      if (opts?.excludeId && n.id === opts.excludeId) return; // 跳过自身及其子树(不递归)
      out.push({ value: n.id, depth, label: '　'.repeat(depth) + (n.name || n.id) });
      if (n.children) walk(n.children, depth + 1);
    });
  };
  walk(nodes, 0);
  return out;
}

/**
 * 通用「分组树节点」结构 — 三皮肤的 GroupTreeSelect / TreeSelect 共用:
 *   · v1 (Antd) 直接喂给 `<TreeSelect treeData={...} />`(字段名天然对齐)。
 *   · v2 (shadcn) / v3 (HUD) 自制 Popover + 递归 Tree,按 children 渲展开/收起。
 *
 * excludeId:排除该节点**及其整棵子树** — 编辑「父分组」下拉用,避免把自己/后代选作父
 * (形成环);整棵子树跳过(剪枝,不递归子代),与 groupTreeOptions 行为一致。
 */
export interface GroupTreeNode {
  /** 显示文本(优先中文名,fallback 英文名/id) */
  title: string;
  /** 选中值(group.id) */
  value: string;
  /** React key(同 value) */
  key: string;
  /** 子节点;无则不下设(让 UI 不出展开箭头) */
  children?: GroupTreeNode[];
}

export function groupTreeData(
  nodes: IndicatorGroup[] | undefined,
  opts?: { excludeId?: string },
): GroupTreeNode[] {
  const walk = (list: IndicatorGroup[] | undefined): GroupTreeNode[] => {
    const out: GroupTreeNode[] = [];
    (list || []).forEach((n) => {
      if (opts?.excludeId && n.id === opts.excludeId) return; // 剪枝:自身及子树都不出现
      const node: GroupTreeNode = {
        title: n.name || n.id,
        value: n.id,
        key: n.id,
      };
      if (n.children && n.children.length > 0) {
        const kids = walk(n.children);
        if (kids.length > 0) node.children = kids;
      }
      out.push(node);
    });
    return out;
  };
  return walk(nodes);
}

function mapFormula(b: BackendFormula): PlatformFormula {
  return {
    platformName: b.platform_name,
    indicatorId: b.indicator_id,
    formula: b.formula,
    description: b.description,
  };
}

// 对齐后端 CreateIndicatorRequest / UpdateIndicatorRequest（model.go）字段名：
//   en_name←enName, cn_name←cnName, group_id←groupId(必填三件套，device_type 由 query 注入),
//   data_type←dataType, unit_id←unit, is_counter←isCounter, arithmetic←arithmetic,
//   statis_type←statisType, product_types←productClass, indicator_level←indicatorLevel,
//   en_description/cn_description←enDescription/cnDescription, operator_code←operatorCode, id←id。
// 旧 bug：发了 counter_type/unit/product_type/name（后端不识别）→ 建/改必失败。
// undefined 字段不发，保持向后兼容（PUT 部分更新）。
function indicatorPayload(
  input: CreateIndicatorInput | UpdateIndicatorInput
): Record<string, unknown> {
  const p: Record<string, unknown> = {};
  if ('id' in input && input.id !== undefined) p.id = input.id;
  if (input.cnName !== undefined) p.cn_name = input.cnName;
  if (input.enName !== undefined) p.en_name = input.enName;
  if (input.groupId !== undefined) p.group_id = input.groupId;
  if (input.dataType !== undefined) p.data_type = input.dataType;
  if (input.unit !== undefined) p.unit_id = input.unit;
  if (input.isCounter !== undefined) p.is_counter = input.isCounter;
  if (input.arithmetic !== undefined) p.arithmetic = input.arithmetic;
  if (input.statisType !== undefined) p.statis_type = input.statisType;
  if (input.indicatorLevel !== undefined) p.indicator_level = input.indicatorLevel;
  if (input.productClass !== undefined) p.product_types = input.productClass;
  if (input.enDescription !== undefined) p.en_description = input.enDescription;
  if (input.cnDescription !== undefined) p.cn_description = input.cnDescription;
  if (input.operatorCode !== undefined) p.operator_code = input.operatorCode;
  // platform 仅 CreateIndicatorInput 有（UpdateIndicatorInput 是 Partial<Omit<..., 'id'>>
  // 但实际 update 不写公式，TS 类型检查不报错就放过；undefined 不下发）。
  if ('platform' in input && input.platform !== undefined) p.platform = input.platform;
  return p;
}

function groupPayload(input: CreateGroupInput | UpdateGroupInput): Record<string, unknown> {
  const p: Record<string, unknown> = {};
  if ('id' in input && input.id !== undefined) p.id = input.id;
  // 后端 Create/UpdateGroupRequest 用 en_name/cn_name（无 name 字段）；name 同时写入两者。
  if (input.name !== undefined) {
    p.en_name = input.name;
    p.cn_name = input.name;
  }
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
        deletable?: boolean;
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
        deletable: b.deletable ?? false,
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

  // multipart 上传自定义 XML(2026-06-05 取消手填名称:名称取自 XML platform 属性,
  // 文件名 = <platform>.xml)。重复允许覆盖:不带 force 时重复返 409
  // (data.overwritable=true),前端弹二次确认后带 force=true 重试 → 覆盖归属文件
  // (旧文件自动备份 .bak.<ts>)。
  // 落地目录:ENB → indicator-library/enb/,GSM/GNB → indicator-library/ 根级。
  async uploadXml(tech: TechLower, file: File, force = false): Promise<IndicatorUploadResult> {
    const form = new FormData();
    form.append('file', file);
    const { data } = await http.post<{
      uploaded: boolean;
      filename: string;
      loaded_from: string;
      tech: string;
      platform: string;
      overwritten: boolean;
      reloaded: boolean;
    }>('/indicators/upload-xml', form, {
      params: force ? { tech, force: 'true' } : { tech },
      // 必须显式声明 multipart/form-data — http.ts axios.create 设了
      // 默认 'Content-Type': 'application/json',不显式覆盖会沿用 JSON
      // 导致 body 被序列化为 "{}" + Gin c.FormFile("file") 返
      // "Content-Type isn't multipart/form-data"(与 paramModelApi.uploadXML / fileApi /
      // adminApi / softwareApi 等 8 处上传同范式)。
      headers: { 'Content-Type': 'multipart/form-data' },
    });
    return {
      uploaded: data.uploaded,
      filename: data.filename,
      loadedFrom: data.loaded_from,
      tech: data.tech as TechLower,
      platform: data.platform,
      overwritten: data.overwritten,
      reloaded: data.reloaded,
    };
  },

  // 下载指标 XML 原文件(builtin / custom 均可,2026-06-05 操作列下载功能)
  async downloadXml(loadedFrom: string): Promise<void> {
    const resp = await http.get('/indicators/file-content', {
      params: { loaded_from: loadedFrom },
      responseType: 'blob',
    });
    saveBlob(resp.data as BlobPart, loadedFrom.split('/').pop() || 'indicator.xml');
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
