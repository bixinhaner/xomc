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
import { extractXmlRootAttr } from '../../utils/xmlRootAttr';
import { saveBlob } from '../../utils/saveBlob';

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
    if (standardParams.some((item) => item.standardPath === input.standardPath)) {
      const error = new Error(
        `standard path "${input.standardPath}" already exists; update the existing record instead`,
      ) as Error & { bizCode: number };
      error.bizCode = 2034;
      throw error;
    }
    const created: StandardParam = {
      ...input,
      changeApplies: input.changeApplies || 'reload',
      updatedAt: new Date().toISOString(),
      updatedFields: [],
    };
    standardParams.push(created);
    return clone(created);
  },

  async updateStandard(path: string, input: UpsertStandardInput): Promise<StandardParam> {
    const idx = standardParams.findIndex((s) => s.standardPath === path);
    if (idx < 0) throw new Error(`standard ${path} not found`);
    const previous = standardParams[idx];
    const next = { ...input, changeApplies: input.changeApplies || 'reload' };
    const updatedFields = (Object.keys(next) as Array<keyof typeof next>).filter(
      (key) => previous[key] !== next[key]
    );
    standardParams[idx] = {
      ...next,
      updatedAt: new Date().toISOString(),
      updatedFields,
    };
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

  // mock 下载:生成占位 XML 触发浏览器另存(无真实文件)。
  async downloadXml(loadedFrom: string): Promise<void> {
    const name = loadedFrom.split('/').pop() || 'param-model.xml';
    saveBlob(`<?xml version="1.0" encoding="UTF-8"?>\n<!-- mock ${name} -->\n<parameterModel paramModel="${name.replace(/\.xml$/i, '')}"></parameterModel>\n`, name);
  },

  // mock 上传:名称取自 XML paramModel 属性(与后端同口径),不真存盘。
  // 重复:无 force 抛错(真实端为 409);force=true 覆盖既有条目。
  async uploadXML(
    file: File,
    force = false,
  ): Promise<{ filename: string; modelName: string; size: number; overwritten: boolean }> {
    const name = (await extractXmlRootAttr(file, 'paramModel')) ?? 'CUSTOM';
    const filename = `${name}.xml`;
    const lower = name.toLowerCase();
    const existing = paramModels.find((p) => p.name.toLowerCase() === lower);
    if (existing && !force) {
      throw new Error(`paramModel ${name} already exists; confirm overwrite`);
    }
    if (existing && force) {
      return { filename, modelName: name, size: file.size, overwritten: true };
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
    return { filename, modelName: name, size: file.size, overwritten: false };
  },
};
