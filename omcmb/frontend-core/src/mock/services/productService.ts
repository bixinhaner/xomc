import type {
  Product,
  ProductPattern,
  MatchOrderRow,
  OrphanDevice,
  ProductMatchResult,
  CreateProductInput,
  UpdateProductInput,
  ProductListFilter,
} from '../../types/product';
import { mockProducts, mockPatterns, mockMatchOrder, mockOrphans } from '../data/product';

let products = [...mockProducts];
const patterns: Record<string, ProductPattern[]> = JSON.parse(JSON.stringify(mockPatterns));
let orphans = [...mockOrphans];

function clone<T>(v: T): T {
  return JSON.parse(JSON.stringify(v)) as T;
}

export const productService = {
  async list(filter?: ProductListFilter): Promise<{ items: Product[]; total: number }> {
    let items = [...products];
    if (filter?.vendor) items = items.filter((p) => p.vendor.toLowerCase() === filter.vendor!.toLowerCase());
    if (filter?.tech) items = items.filter((p) => p.tech.toLowerCase() === filter.tech!.toLowerCase());
    if (filter?.keyword) {
      const k = filter.keyword.toLowerCase();
      items = items.filter((p) => (p.name + p.vendor + p.description).toLowerCase().includes(k));
    }
    return { items: clone(items), total: items.length };
  },

  async get(id: string): Promise<{ product: Product; patterns: ProductPattern[] }> {
    const p = products.find((x) => x.id === id);
    if (!p) throw new Error(`product ${id} not found`);
    return { product: clone(p), patterns: clone(patterns[id] || []) };
  },

  async create(input: CreateProductInput): Promise<Product> {
    const newP: Product = {
      id: `p-${Date.now()}`,
      name: input.name,
      vendor: input.vendor || '',
      tech: input.tech || '',
      radioModes: input.radioModes || '',
      description: input.description || '',
      paramModelId: input.paramModelId,
      indicatorDeviceType: input.indicatorDeviceType,
      indicatorPlatform: input.indicatorPlatform,
      alarmNeType: input.alarmNeType,
      enableFiletype11: input.enableFiletype11 ?? true,
      deviceAttrsOverride: input.deviceAttrsOverride || {},
      enableUnknownAlarm: input.enableUnknownAlarm ?? false,
      deviceCount: 0,
    };
    products.push(newP);
    patterns[newP.id] = [];
    return clone(newP);
  },

  async update(id: string, input: UpdateProductInput): Promise<Product> {
    const idx = products.findIndex((p) => p.id === id);
    if (idx < 0) throw new Error(`product ${id} not found`);
    const cur = products[idx];
    products[idx] = {
      ...cur,
      ...(input.name !== undefined && { name: input.name }),
      ...(input.vendor !== undefined && { vendor: input.vendor }),
      ...(input.tech !== undefined && { tech: input.tech }),
      ...(input.radioModes !== undefined && { radioModes: input.radioModes }),
      ...(input.description !== undefined && { description: input.description }),
      ...(input.paramModelId !== undefined && { paramModelId: input.paramModelId }),
      ...(input.clearParamModel && { paramModelId: undefined }),
      ...(input.indicatorDeviceType !== undefined && { indicatorDeviceType: input.indicatorDeviceType }),
      ...(input.indicatorPlatform !== undefined && { indicatorPlatform: input.indicatorPlatform }),
      ...(input.alarmNeType !== undefined && { alarmNeType: input.alarmNeType }),
      ...(input.enableFiletype11 !== undefined && { enableFiletype11: input.enableFiletype11 }),
      ...(input.deviceAttrsOverride !== undefined && { deviceAttrsOverride: input.deviceAttrsOverride }),
      ...(input.enableUnknownAlarm !== undefined && { enableUnknownAlarm: input.enableUnknownAlarm }),
    };
    return clone(products[idx]);
  },

  async delete(id: string): Promise<void> {
    products = products.filter((p) => p.id !== id);
    delete patterns[id];
  },

  async resetDiscovered(id: string) {
    return { deletedRows: 0, productId: id, boundDevices: 0 };
  },

  async createPattern(productId: string, productClass: string): Promise<ProductPattern> {
    const arr = patterns[productId] || [];
    const newPat: ProductPattern = {
      id: `pat-${Date.now()}`,
      productId,
      productClass,
      sortOrder: arr.length + 1,
      isActive: true,
    };
    arr.push(newPat);
    patterns[productId] = arr;
    return clone(newPat);
  },

  async updatePattern(productId: string, patternId: string, productClass: string): Promise<ProductPattern> {
    const arr = patterns[productId] || [];
    const idx = arr.findIndex((p) => p.id === patternId);
    if (idx < 0) throw new Error(`pattern ${patternId} not found`);
    arr[idx] = { ...arr[idx], productClass };
    return clone(arr[idx]);
  },

  async deletePattern(productId: string, patternId: string): Promise<void> {
    patterns[productId] = (patterns[productId] || []).filter((p) => p.id !== patternId);
  },

  async movePattern(
    productId: string,
    patternId: string,
    direction: 'up' | 'down'
  ): Promise<ProductPattern> {
    const arr = patterns[productId] || [];
    const idx = arr.findIndex((p) => p.id === patternId);
    if (idx < 0) throw new Error(`pattern ${patternId} not found`);
    const swap = direction === 'up' ? idx - 1 : idx + 1;
    if (swap < 0 || swap >= arr.length) return clone(arr[idx]);
    const tmp = arr[swap].sortOrder;
    arr[swap].sortOrder = arr[idx].sortOrder;
    arr[idx].sortOrder = tmp;
    [arr[swap], arr[idx]] = [arr[idx], arr[swap]];
    return clone(arr[direction === 'up' ? swap : swap]);
  },

  async match(productClass: string): Promise<ProductMatchResult> {
    for (const row of mockMatchOrder) {
      try {
        const re = new RegExp(row.productClass);
        if (re.test(productClass)) {
          const p = products.find((x) => x.id === row.productId);
          return {
            matched: true,
            product: p ? clone(p) : undefined,
            matchedPattern: row.productClass,
            globalOrder: row.sortOrder,
            productClass,
          };
        }
      } catch {
        // ignore invalid regex
      }
    }
    return { matched: false, productClass };
  },

  async matchOrder(): Promise<{ items: MatchOrderRow[]; total: number }> {
    return { items: clone(mockMatchOrder), total: mockMatchOrder.length };
  },

  async listOrphan(_limit?: number): Promise<{ items: OrphanDevice[]; total: number }> {
    return { items: clone(orphans), total: orphans.length };
  },

  async rematchOrphan() {
    const before = orphans.length;
    orphans = [];
    return { rebound: before, scanned: before };
  },

  async bindOrphan(deviceId: string, productId: string) {
    orphans = orphans.filter((d) => d.id !== deviceId);
    return { deviceId, productId };
  },

  async cacheRefresh() {
    return { refreshed: true };
  },

  async importDirectory() {
    return { reloaded: 'product' };
  },

  async listIndicatorPlatforms(deviceType: string): Promise<string[]> {
    const dt = (deviceType || '').toUpperCase();
    if (dt === 'ENB') return ['enb-default', 'enb-comba', 'enb-baicells'];
    return [];
  },

  async listAlarmNeTypes(): Promise<string[]> {
    return ['ENB', 'GNB', 'OMC', 'EPC', 'EGW', 'CPE', 'UPS'];
  },
};
