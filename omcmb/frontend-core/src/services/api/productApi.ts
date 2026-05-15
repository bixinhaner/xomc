import http from '../http';
import type {
  Product,
  ProductPattern,
  MatchOrderRow,
  OrphanDevice,
  ProductMatchResult,
  CreateProductInput,
  UpdateProductInput,
  ProductListFilter,
  DeviceAttrsOverride,
} from '../../types/product';

// 后端 productView 返回字段（snake_case）
interface BackendProduct {
  id: string;
  name: string;
  vendor: string;
  tech: string;
  radio_modes: string;
  description: string;
  param_model_id?: string;
  indicator_device_type: string;
  indicator_platform: string;
  alarm_ne_type: string;
  enable_filetype11: boolean;
  device_attrs_override: DeviceAttrsOverride;
  enable_unknown_alarm: boolean;
  device_count?: number;
  patterns?: string[];
}

// 后端 PatternView 字段缺 json tag，序列化为 PascalCase
interface BackendPattern {
  ID: string;
  ProductID: string;
  ProductClass: string;
  SortOrder: number;
  IsActive: boolean;
}

interface BackendMatchOrderRow {
  pattern_id: string;
  product_id: string;
  product_name: string;
  product_class: string;
  sort_order: number;
  is_active: boolean;
}

interface BackendOrphanDevice {
  id: string;
  serial_number: string;
  oui: string;
  product_class: string;
  carrier: string;
  manufacturer: string;
  last_inform_at?: string;
}

function mapBackendProduct(bp: BackendProduct): Product {
  return {
    id: bp.id,
    name: bp.name,
    vendor: bp.vendor,
    tech: bp.tech,
    radioModes: bp.radio_modes,
    description: bp.description,
    paramModelId: bp.param_model_id,
    indicatorDeviceType: bp.indicator_device_type,
    indicatorPlatform: bp.indicator_platform,
    alarmNeType: bp.alarm_ne_type,
    enableFiletype11: bp.enable_filetype11,
    deviceAttrsOverride: bp.device_attrs_override || {},
    enableUnknownAlarm: bp.enable_unknown_alarm,
    deviceCount: bp.device_count ?? 0,
    patterns: bp.patterns ?? [],
  };
}

function mapBackendPattern(bp: BackendPattern): ProductPattern {
  return {
    id: bp.ID,
    productId: bp.ProductID,
    productClass: bp.ProductClass,
    sortOrder: bp.SortOrder,
    isActive: bp.IsActive,
  };
}

function mapBackendMatchOrder(b: BackendMatchOrderRow): MatchOrderRow {
  return {
    patternId: b.pattern_id,
    productId: b.product_id,
    productName: b.product_name,
    productClass: b.product_class,
    sortOrder: b.sort_order,
    isActive: b.is_active,
  };
}

function mapBackendOrphan(b: BackendOrphanDevice): OrphanDevice {
  return {
    id: b.id,
    serialNumber: b.serial_number,
    oui: b.oui,
    productClass: b.product_class,
    carrier: b.carrier,
    manufacturer: b.manufacturer,
    lastInformAt: b.last_inform_at,
  };
}

function buildCreatePayload(input: CreateProductInput): Record<string, unknown> {
  const payload: Record<string, unknown> = {
    name: input.name,
    indicator_device_type: input.indicatorDeviceType,
    indicator_platform: input.indicatorPlatform,
    alarm_ne_type: input.alarmNeType,
  };
  if (input.vendor !== undefined) payload.vendor = input.vendor;
  if (input.tech !== undefined) payload.tech = input.tech;
  if (input.radioModes !== undefined) payload.radio_modes = input.radioModes;
  if (input.description !== undefined) payload.description = input.description;
  if (input.paramModelName !== undefined) payload.param_model_name = input.paramModelName;
  if (input.paramModelId !== undefined) payload.param_model_id = input.paramModelId;
  if (input.enableFiletype11 !== undefined) payload.enable_filetype11 = input.enableFiletype11;
  if (input.deviceAttrsOverride !== undefined) payload.device_attrs_override = input.deviceAttrsOverride;
  if (input.enableUnknownAlarm !== undefined) payload.enable_unknown_alarm = input.enableUnknownAlarm;
  if (input.patterns && input.patterns.length > 0) payload.patterns = input.patterns;
  return payload;
}

function buildUpdatePayload(input: UpdateProductInput): Record<string, unknown> {
  const payload: Record<string, unknown> = {};
  if (input.name !== undefined) payload.name = input.name;
  if (input.vendor !== undefined) payload.vendor = input.vendor;
  if (input.tech !== undefined) payload.tech = input.tech;
  if (input.radioModes !== undefined) payload.radio_modes = input.radioModes;
  if (input.description !== undefined) payload.description = input.description;
  if (input.paramModelName !== undefined) payload.param_model_name = input.paramModelName;
  if (input.paramModelId !== undefined) payload.param_model_id = input.paramModelId;
  if (input.clearParamModel) payload.clear_param_model = true;
  if (input.indicatorDeviceType !== undefined) payload.indicator_device_type = input.indicatorDeviceType;
  if (input.indicatorPlatform !== undefined) payload.indicator_platform = input.indicatorPlatform;
  if (input.alarmNeType !== undefined) payload.alarm_ne_type = input.alarmNeType;
  if (input.enableFiletype11 !== undefined) payload.enable_filetype11 = input.enableFiletype11;
  if (input.deviceAttrsOverride !== undefined) payload.device_attrs_override = input.deviceAttrsOverride;
  if (input.enableUnknownAlarm !== undefined) payload.enable_unknown_alarm = input.enableUnknownAlarm;
  return payload;
}

export const productApi = {
  async list(filter?: ProductListFilter): Promise<{ items: Product[]; total: number }> {
    const params: Record<string, unknown> = {};
    if (filter?.vendor) params.vendor = filter.vendor;
    if (filter?.tech) params.tech = filter.tech;
    if (filter?.keyword) params.keyword = filter.keyword;
    const { data } = await http.get<{ items: BackendProduct[]; total: number }>('/products', { params });
    return {
      items: (data.items || []).map(mapBackendProduct),
      total: data.total || 0,
    };
  },

  async get(id: string): Promise<{ product: Product; patterns: ProductPattern[] }> {
    const { data } = await http.get<{ product: BackendProduct; patterns: BackendPattern[] }>(
      `/products/${id}`
    );
    return {
      product: mapBackendProduct(data.product),
      patterns: (data.patterns || []).map(mapBackendPattern),
    };
  },

  async create(input: CreateProductInput): Promise<Product> {
    const { data } = await http.post<BackendProduct>('/products', buildCreatePayload(input));
    return mapBackendProduct(data);
  },

  async update(id: string, input: UpdateProductInput): Promise<Product> {
    const { data } = await http.put<BackendProduct>(`/products/${id}`, buildUpdatePayload(input));
    return mapBackendProduct(data);
  },

  async delete(id: string): Promise<void> {
    await http.delete(`/products/${id}`);
  },

  async resetDiscovered(id: string): Promise<{ deletedRows: number; productId: string; boundDevices: number }> {
    const { data } = await http.delete<{ deleted_rows: number; product_id: string; bound_devices: number }>(
      `/products/${id}/discovered`
    );
    return {
      deletedRows: data.deleted_rows,
      productId: data.product_id,
      boundDevices: data.bound_devices,
    };
  },

  async createPattern(productId: string, productClass: string): Promise<ProductPattern> {
    const { data } = await http.post<BackendPattern>(`/products/${productId}/patterns`, {
      product_class: productClass,
    });
    return mapBackendPattern(data);
  },

  async updatePattern(productId: string, patternId: string, productClass: string): Promise<ProductPattern> {
    const { data } = await http.put<BackendPattern>(
      `/products/${productId}/patterns/${patternId}`,
      { product_class: productClass }
    );
    return mapBackendPattern(data);
  },

  async deletePattern(productId: string, patternId: string): Promise<void> {
    await http.delete(`/products/${productId}/patterns/${patternId}`);
  },

  async movePattern(productId: string, patternId: string, direction: 'up' | 'down'): Promise<ProductPattern> {
    const { data } = await http.put<BackendPattern>(
      `/products/${productId}/patterns/${patternId}/move`,
      { direction }
    );
    return mapBackendPattern(data);
  },

  async match(productClass: string): Promise<ProductMatchResult> {
    const { data } = await http.get<{
      matched: boolean;
      product?: BackendProduct;
      matched_pattern?: string;
      global_order?: number;
      product_class?: string;
    }>('/products/match', { params: { productClass } });
    return {
      matched: data.matched,
      product: data.product ? mapBackendProduct(data.product) : undefined,
      matchedPattern: data.matched_pattern,
      globalOrder: data.global_order,
      productClass: data.product_class,
    };
  },

  async matchOrder(): Promise<{ items: MatchOrderRow[]; total: number }> {
    const { data } = await http.get<{ items: BackendMatchOrderRow[]; total: number }>(
      '/products/match-order'
    );
    return {
      items: (data.items || []).map(mapBackendMatchOrder),
      total: data.total || 0,
    };
  },

  async listOrphan(limit = 200): Promise<{ items: OrphanDevice[]; total: number }> {
    const { data } = await http.get<{ items: BackendOrphanDevice[]; total: number }>(
      '/products/orphan-devices',
      { params: { limit } }
    );
    return {
      items: (data.items || []).map(mapBackendOrphan),
      total: data.total || 0,
    };
  },

  async rematchOrphan(): Promise<{ rebound: number; scanned?: number }> {
    const { data } = await http.post<{ rebound: number; scanned?: number }>(
      '/products/orphan-devices/rematch'
    );
    return data;
  },

  async bindOrphan(deviceId: string, productId: string): Promise<{ deviceId: string; productId: string }> {
    const { data } = await http.put<{ device_id: string; product_id: string }>(
      `/products/orphan-devices/${deviceId}/bind`,
      { product_id: productId }
    );
    return { deviceId: data.device_id, productId: data.product_id };
  },

  async cacheRefresh(): Promise<{ refreshed: boolean }> {
    const { data } = await http.post<{ refreshed: boolean }>('/products/cache/refresh');
    return data;
  },

  async importDirectory(): Promise<{ reloaded: string }> {
    const { data } = await http.post<{ reloaded: string }>('/products/import-directory');
    return data;
  },

  async listIndicatorPlatforms(deviceType: string): Promise<string[]> {
    const { data } = await http.get<{ items: string[]; total: number; device_type: string }>(
      '/products/indicator-platforms',
      { params: { deviceType } }
    );
    return data.items || [];
  },

  async listAlarmNeTypes(): Promise<string[]> {
    const { data } = await http.get<{ items: string[]; total: number }>(
      '/products/alarm-ne-types'
    );
    return data.items || [];
  },
};
