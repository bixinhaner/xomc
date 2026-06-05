import type {
  ParamModel,
  ParamMapping,
  StandardParam,
  DiscoveredVersion,
  CreateMappingInput,
  UpdateMappingInput,
  UpsertStandardInput,
  UpdateParamModelInput,
  StandardParamFilter,
  TranslateRequest,
  TranslateResponse,
} from '../../types/paramModel';
import { mockParamModels, mockMappings, mockStandardParams } from '../data/paramModel';

let paramModels = [...mockParamModels];
const mappings: Record<string, ParamMapping[]> = JSON.parse(JSON.stringify(mockMappings));
let standardParams = [...mockStandardParams];

function clone<T>(v: T): T {
  return JSON.parse(JSON.stringify(v)) as T;
}

export const paramModelService = {
  async list() {
    return { items: clone(paramModels), total: paramModels.length };
  },

  async get(name: string): Promise<ParamModel> {
    const m = paramModels.find((p) => p.name === name);
    if (!m) throw new Error(`param model ${name} not found`);
    return clone(m);
  },

  async update(name: string, input: UpdateParamModelInput): Promise<ParamModel> {
    const idx = paramModels.findIndex((p) => p.name === name);
    if (idx < 0) throw new Error(`param model ${name} not found`);
    paramModels[idx] = {
      ...paramModels[idx],
      ...(input.description !== undefined && { description: input.description }),
      ...(input.isActive !== undefined && { isActive: input.isActive }),
    };
    return clone(paramModels[idx]);
  },

  async delete(name: string): Promise<void> {
    paramModels = paramModels.filter((p) => p.name !== name);
    delete mappings[name];
  },

  async listMappings(name: string) {
    return { items: clone(mappings[name] || []), total: (mappings[name] || []).length, paramModel: name };
  },

  async createMapping(name: string, input: CreateMappingInput): Promise<ParamMapping> {
    const arr = mappings[name] || [];
    const created: ParamMapping = {
      id: `map-${Date.now()}`,
      paramModelId: paramModels.find((p) => p.name === name)?.id || 'unknown',
      standardPath: input.standardPath,
      privatePath: input.privatePath,
      entryType: input.entryType,
      access: input.access,
      dataType: input.dataType,
      changeApplies: input.changeApplies || 'reload',
      minValue: input.minValue,
      maxValue: input.maxValue,
      isStorable: input.isStorable ?? true,
      isActive: input.isActive ?? true,
      softwareVersion: input.softwareVersion,
    };
    arr.push(created);
    mappings[name] = arr;
    return clone(created);
  },

  async updateMapping(name: string, id: string, input: UpdateMappingInput): Promise<ParamMapping> {
    const arr = mappings[name] || [];
    const idx = arr.findIndex((m) => m.id === id);
    if (idx < 0) throw new Error(`mapping ${id} not found`);
    arr[idx] = {
      ...arr[idx],
      ...(input.standardPath !== undefined && { standardPath: input.standardPath }),
      ...(input.privatePath !== undefined && { privatePath: input.privatePath }),
      ...(input.entryType !== undefined && { entryType: input.entryType }),
      ...(input.access !== undefined && { access: input.access }),
      ...(input.dataType !== undefined && { dataType: input.dataType }),
      ...(input.changeApplies !== undefined && { changeApplies: input.changeApplies }),
      ...(input.minValue !== undefined && { minValue: input.minValue }),
      ...(input.maxValue !== undefined && { maxValue: input.maxValue }),
      ...(input.isStorable !== undefined && { isStorable: input.isStorable }),
      ...(input.isActive !== undefined && { isActive: input.isActive }),
      ...(input.softwareVersion !== undefined && { softwareVersion: input.softwareVersion }),
    };
    return clone(arr[idx]);
  },

  async deleteMapping(name: string, id: string): Promise<void> {
    mappings[name] = (mappings[name] || []).filter((m) => m.id !== id);
  },

  async listStandard(filter?: StandardParamFilter) {
    let items = [...standardParams];
    if (filter?.keyword) {
      const k = filter.keyword.toLowerCase();
      items = items.filter((s) => s.standardPath.toLowerCase().includes(k));
    }
    if (filter?.entryType) items = items.filter((s) => s.entryType === filter.entryType);
    return { items: clone(items), total: items.length };
  },

  async getStandard(path: string): Promise<StandardParam> {
    const s = standardParams.find((x) => x.standardPath === path);
    if (!s) throw new Error(`standard ${path} not found`);
    return clone(s);
  },

  async createStandard(input: UpsertStandardInput): Promise<StandardParam> {
    const created: StandardParam = { ...input, changeApplies: input.changeApplies || 'reload' };
    standardParams.push(created);
    return clone(created);
  },

  async updateStandard(path: string, input: UpsertStandardInput): Promise<StandardParam> {
    const idx = standardParams.findIndex((s) => s.standardPath === path);
    if (idx < 0) throw new Error(`standard ${path} not found`);
    standardParams[idx] = { ...input, changeApplies: input.changeApplies || 'reload' };
    return clone(standardParams[idx]);
  },

  async deleteStandard(path: string): Promise<void> {
    standardParams = standardParams.filter((s) => s.standardPath !== path);
  },

  async translate(request: TranslateRequest): Promise<TranslateResponse> {
    return {
      productId: request.productId,
      softwareVersion: request.softwareVersion,
      direction: request.direction,
      results: request.paths.map((p) => ({ source: p, target: p, matched: true })),
      source: 'mock',
    };
  },

  async listDiscovered(_productId: string, _swVersion?: string) {
    return { items: [] as ParamMapping[], total: 0 };
  },

  async listDiscoveredVersions(_productId: string) {
    return { items: [] as DiscoveredVersion[], total: 0 };
  },

  async deleteDiscovered(_productId: string, _swVersion?: string) {
    return { deleted: 0 };
  },

  // mock 上传(三库 XML 导入重构):传 name + file,不真存盘。
  // 双唯一性硬拒,无 force —— 文件名或模型名已存在即抛错(模拟 409)。
  async uploadXML(
    file: File,
    name: string,
  ): Promise<{ filename: string; modelName: string; size: number }> {
    const filename = `${name}.xml`;
    const lower = name.toLowerCase();
    const existing = paramModels.find((p) => p.name.toLowerCase() === lower);
    if (existing) {
      throw new Error(`name ${filename} already exists; please rename`);
    }
    paramModels = [
      ...paramModels,
      {
        id: `pm-mock-${Date.now()}`,
        name,
        totalEntries: 50,
        totalObjects: 10,
        totalParams: 40,
        description: 'Mock 自定义 paramModel(上传成功)',
        isActive: true,
        loadedFrom: `param-mappings/${filename}`,
        source: 'custom',
        deletable: true,
      },
    ];
    return { filename, modelName: name, size: file.size };
  },
};
