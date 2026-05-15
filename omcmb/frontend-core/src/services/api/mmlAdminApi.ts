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
  BackendParamReference,
  BackendImportPreviewResp,
  BackendImportApplyResp,
  GroupAdmin,
  CommandAdmin,
  SubFieldAdmin,
  ParamAdmin,
  ParamReference,
  ImportPreviewResp,
  ImportApplyResp,
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
  mapBackendParamReference,
  mapBackendImportPreview,
  mapBackendImportApply,
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

  /**
   * 反向查：返回引用该 param 的命令列表（T-0131 admin Tab 3 抽屉用）。
   * 不分页（单 param 的引用集通常 ≤ 50，全量返回足够）。
   */
  async listParamReferences(paramId: string): Promise<ParamReference[]> {
    const { data } = await http.get<{ items: BackendParamReference[] }>(
      `${BASE}/params/${paramId}/references`,
    );
    return (data.items ?? []).map(mapBackendParamReference);
  },

  // ---------------- T-0132 XML 导入 (Tab 4) ----------------
  // 注：后端路径是 /api/v1/mml/admin/import/... 字面量；既有 BASE='/admin' 与后端 /mml/admin 路径漂移
  // 是 T-0123-P3 verify report 留的"真后端实测时统一修"遗留 bug，本期 import 端点用正确字面量绕开。

  /**
   * dry-run 预览：上传 XML → 后端解析 + 三桶 diff → 返 summary + 前 200 行详情。不写表。
   * 客户端可重复调用（每次重新上传同一文件即可，无 server-side 缓存）。
   */
  async importPreview(
    file: File,
    versionCode: string,
  ): Promise<ImportPreviewResp> {
    const form = new FormData();
    form.append('file', file);
    form.append('version_code', versionCode);
    const { data } = await http.post<BackendImportPreviewResp>(
      '/mml/admin/import/preview',
      form,
      { headers: { 'Content-Type': 'multipart/form-data' } },
    );
    return mapBackendImportPreview(data);
  },

  /**
   * 真写入：上传 XML → 后端批量 UPSERT；catalog_protected=true 行被覆盖，false 的 admin 行被守护跳过。
   * Apply 与 Preview 是独立两步调用（防 server-side session），UI 应在 Apply 完成后 invalidate group-tree。
   */
  async importApply(
    file: File,
    versionCode: string,
  ): Promise<ImportApplyResp> {
    const form = new FormData();
    form.append('file', file);
    form.append('version_code', versionCode);
    const { data } = await http.post<BackendImportApplyResp>(
      '/mml/admin/import/apply',
      form,
      { headers: { 'Content-Type': 'multipart/form-data' } },
    );
    return mapBackendImportApply(data);
  },
};
