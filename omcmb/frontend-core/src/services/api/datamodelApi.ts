import http from '../http';
import type { PageRequest, PageResponse } from '../../types/pagination';

// --- Backend response types ---

interface BackendDataModel {
  id: string;
  carrier: string;
  technology: string;
  version: string;
  oui: string;
  product_class: string;
  scope: string;
  status: string;
  is_active: boolean;
  root_object: string;
  parameter_tree: unknown;
  source: string;
  imported_by: string;
  spec_document_ref: string;
  description: string;
  created_at: string;
  updated_at: string;
}

interface BackendDataModelStats {
  total: number;
  active: number;
  draft: number;
  deprecated: number;
  by_carrier: Record<string, number>;
  by_scope: Record<string, number>;
}

interface BackendOUIEntry {
  oui: string;
  manufacturer: string;
  short_name: string;
  country: string;
  created_at: string;
}

interface BackendListResponse<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
}

// --- Frontend types ---

export interface DataModel {
  id: string;
  carrier: string;
  technology: string;
  version: string;
  oui: string;
  productClass: string;
  scope: string;
  status: string;
  isActive: boolean;
  rootObject: string;
  parameterTree: unknown;
  source: string;
  importedBy: string;
  specDocumentRef: string;
  description: string;
  createdAt: string;
  updatedAt: string;
}

export interface DataModelStats {
  total: number;
  active: number;
  draft: number;
  deprecated: number;
  byCarrier: Record<string, number>;
  byScope: Record<string, number>;
}

export interface OUIEntry {
  oui: string;
  manufacturer: string;
  shortName: string;
  country: string;
  createdAt: string;
}

export interface CreateDataModelRequest {
  carrier: string;
  technology: string;
  version: string;
  oui?: string;
  productClass?: string;
  rootObject?: string;
  parameterTree: unknown;
  source?: string;
  specDocumentRef?: string;
  description?: string;
}

export interface UpdateDataModelRequest {
  version: string;
  rootObject?: string;
  parameterTree: unknown;
  source?: string;
  specDocumentRef?: string;
  description?: string;
}

export interface CreateOUIRequest {
  oui: string;
  manufacturer: string;
  shortName: string;
  country?: string;
}

// --- Mapping functions ---

function mapBackendDataModel(d: BackendDataModel): DataModel {
  return {
    id: d.id,
    carrier: d.carrier,
    technology: d.technology,
    version: d.version,
    oui: d.oui || '',
    productClass: d.product_class || '',
    scope: d.scope,
    status: d.status,
    isActive: d.is_active,
    rootObject: d.root_object,
    parameterTree: d.parameter_tree,
    source: d.source || '',
    importedBy: d.imported_by || '',
    specDocumentRef: d.spec_document_ref || '',
    description: d.description || '',
    createdAt: d.created_at,
    updatedAt: d.updated_at,
  };
}

function mapBackendOUIEntry(e: BackendOUIEntry): OUIEntry {
  return {
    oui: e.oui,
    manufacturer: e.manufacturer,
    shortName: e.short_name,
    country: e.country || '',
    createdAt: e.created_at,
  };
}

function mapBackendStats(s: BackendDataModelStats): DataModelStats {
  return {
    total: s.total,
    active: s.active,
    draft: s.draft,
    deprecated: s.deprecated,
    byCarrier: s.by_carrier || {},
    byScope: s.by_scope || {},
  };
}

// --- Exported service ---

export const datamodelApi = {
  async getDataModels(
    params: {
      carrier?: string;
      technology?: string;
      oui?: string;
      productClass?: string;
      scope?: string;
      status?: string;
    } & PageRequest
  ): Promise<PageResponse<DataModel>> {
    const query: Record<string, unknown> = {
      page: params.page,
      pageSize: params.pageSize,
    };
    if (params.carrier) query.carrier = params.carrier;
    if (params.technology) query.technology = params.technology;
    if (params.oui) query.oui = params.oui;
    if (params.productClass) query.product_class = params.productClass;
    if (params.scope) query.scope = params.scope;
    if (params.status) query.status = params.status;

    const { data } = await http.get<BackendListResponse<BackendDataModel>>(
      '/datamodels',
      { params: query }
    );

    return {
      items: (data.items || []).map(mapBackendDataModel),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async getDataModel(id: string): Promise<DataModel> {
    const { data } = await http.get<BackendDataModel>(`/datamodels/${id}`);
    return mapBackendDataModel(data);
  },

  async createDataModel(req: CreateDataModelRequest): Promise<DataModel> {
    const body = {
      carrier: req.carrier,
      technology: req.technology,
      version: req.version,
      oui: req.oui,
      product_class: req.productClass,
      root_object: req.rootObject,
      parameter_tree: req.parameterTree,
      source: req.source,
      spec_document_ref: req.specDocumentRef,
      description: req.description,
    };
    const { data } = await http.post<BackendDataModel>('/datamodels', body);
    return mapBackendDataModel(data);
  },

  async updateDataModel(id: string, req: UpdateDataModelRequest): Promise<DataModel> {
    const body = {
      version: req.version,
      root_object: req.rootObject,
      parameter_tree: req.parameterTree,
      source: req.source,
      spec_document_ref: req.specDocumentRef,
      description: req.description,
    };
    const { data } = await http.put<BackendDataModel>(`/datamodels/${id}`, body);
    return mapBackendDataModel(data);
  },

  async deleteDataModel(id: string): Promise<void> {
    await http.delete(`/datamodels/${id}`);
  },

  async activateDataModel(id: string): Promise<void> {
    await http.post(`/datamodels/${id}/activate`);
  },

  async deprecateDataModel(id: string): Promise<void> {
    await http.post(`/datamodels/${id}/deprecate`);
  },

  async getStatistics(): Promise<DataModelStats> {
    const { data } = await http.get<BackendDataModelStats>('/datamodels/statistics');
    return mapBackendStats(data);
  },

  async refreshCache(): Promise<void> {
    await http.post('/datamodels/cache/refresh');
  },

  async exportDataModel(id: string): Promise<Blob> {
    const { data } = await http.get(`/datamodels/${id}/export`, {
      responseType: 'blob',
    });
    return data as Blob;
  },

  async importDataModel(file: File): Promise<DataModel> {
    const formData = new FormData();
    formData.append('file', file);
    const { data } = await http.post<BackendDataModel>('/datamodels/import', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
    return mapBackendDataModel(data);
  },

  // --- OUI Management ---

  async getOUIList(): Promise<OUIEntry[]> {
    const { data } = await http.get<{ items: BackendOUIEntry[]; total: number }>('/oui');
    return (data.items || []).map(mapBackendOUIEntry);
  },

  async createOUI(req: CreateOUIRequest): Promise<OUIEntry> {
    const body = {
      oui: req.oui,
      manufacturer: req.manufacturer,
      short_name: req.shortName,
      country: req.country,
    };
    const { data } = await http.post<BackendOUIEntry>('/oui', body);
    return mapBackendOUIEntry(data);
  },
};
