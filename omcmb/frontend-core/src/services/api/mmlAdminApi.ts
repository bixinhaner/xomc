/**
 * T-0123-P3 admin Catalog 管理 API 客户端封装 — frontend-core
 *
 * 13 endpoints (T-0123-P0 已 ship)：
 *   POST   /admin/groups
 *   PATCH  /admin/groups/:id
 *   DELETE /admin/groups/:id
 *   POST   /admin/commands
 *   PATCH  /admin/commands/:id
 *   DELETE /admin/commands/:id
 *   POST   /admin/commands/:cid/sub-fields
 *   PATCH  /admin/commands/:cid/sub-fields/:sid
 *   DELETE /admin/commands/:cid/sub-fields/:sid
 *   GET    /admin/params
 *   POST   /admin/params
 *   PATCH  /admin/params/:id
 *   DELETE /admin/params/:id
 *
 * 读端点复用 P1 /mml/group-tree（GroupsTab + CommandsTab 数据源）。
 *
 * http.ts 拦截器自动:
 *   - Request body: camelCase → snake_case
 *   - Response data: snake_case → camelCase (但本文件用 Backend*Admin → mapBackend* 显式处理)
 *   - Bearer Token 注入 + 自动续期
 */

import http from '../http';
import type {
  BackendGroupAdmin,
  BackendCommandAdmin,
  BackendSubFieldAdmin,
  BackendParamAdmin,
  GroupAdmin,
  CommandAdmin,
  SubFieldAdmin,
  ParamAdmin,
  CreateGroupRequest,
  UpdateGroupRequest,
  CreateCommandRequest,
  UpdateCommandRequest,
  CreateSubFieldRequest,
  UpdateSubFieldRequest,
  CreateParamRequest,
  UpdateParamRequest,
  ListParamsRequest,
  ListParamsResponse,
} from '../../types/mmlAdmin';
import {
  mapBackendGroup,
  mapBackendCommand,
  mapBackendSubField,
  mapBackendParam,
} from '../../types/mmlAdmin';

const BASE = '/admin';

interface BackendListParamsResponse {
  items: BackendParamAdmin[];
  total: number;
  page: number;
  page_size: number;
}

export const mmlAdminApi = {
  // ---------------- Groups ----------------
  async createGroup(req: CreateGroupRequest): Promise<GroupAdmin> {
    const { data } = await http.post<BackendGroupAdmin>(`${BASE}/groups`, req);
    return mapBackendGroup(data);
  },

  async updateGroup(id: string, req: UpdateGroupRequest): Promise<GroupAdmin> {
    const { data } = await http.patch<BackendGroupAdmin>(`${BASE}/groups/${id}`, req);
    return mapBackendGroup(data);
  },

  async deleteGroup(id: string): Promise<void> {
    await http.delete<void>(`${BASE}/groups/${id}`);
  },

  // ---------------- Commands ----------------
  async createCommand(req: CreateCommandRequest): Promise<CommandAdmin> {
    const { data } = await http.post<BackendCommandAdmin>(`${BASE}/commands`, req);
    return mapBackendCommand(data);
  },

  async updateCommand(id: string, req: UpdateCommandRequest): Promise<CommandAdmin> {
    const { data } = await http.patch<BackendCommandAdmin>(`${BASE}/commands/${id}`, req);
    return mapBackendCommand(data);
  },

  async deleteCommand(id: string): Promise<void> {
    await http.delete<void>(`${BASE}/commands/${id}`);
  },

  // ---------------- SubFields ----------------
  async createSubField(
    commandId: string,
    req: CreateSubFieldRequest,
  ): Promise<SubFieldAdmin> {
    const { data } = await http.post<BackendSubFieldAdmin>(
      `${BASE}/commands/${commandId}/sub-fields`,
      req,
    );
    return mapBackendSubField(data);
  },

  async updateSubField(
    commandId: string,
    subFieldId: string,
    req: UpdateSubFieldRequest,
  ): Promise<SubFieldAdmin> {
    const { data } = await http.patch<BackendSubFieldAdmin>(
      `${BASE}/commands/${commandId}/sub-fields/${subFieldId}`,
      req,
    );
    return mapBackendSubField(data);
  },

  async deleteSubField(commandId: string, subFieldId: string): Promise<void> {
    await http.delete<void>(`${BASE}/commands/${commandId}/sub-fields/${subFieldId}`);
  },

  // ---------------- Params ----------------
  async listParams(req: ListParamsRequest = {}): Promise<ListParamsResponse> {
    const { data } = await http.get<BackendListParamsResponse>(`${BASE}/params`, { params: req });
    return {
      items: (data.items ?? []).map(mapBackendParam),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async createParam(req: CreateParamRequest): Promise<ParamAdmin> {
    const { data } = await http.post<BackendParamAdmin>(`${BASE}/params`, req);
    return mapBackendParam(data);
  },

  async updateParam(id: string, req: UpdateParamRequest): Promise<ParamAdmin> {
    const { data } = await http.patch<BackendParamAdmin>(`${BASE}/params/${id}`, req);
    return mapBackendParam(data);
  },

  async deleteParam(id: string): Promise<void> {
    await http.delete<void>(`${BASE}/params/${id}`);
  },
};
