/**
 * MML Admin Catalog API 客户端 — frontend-core
 *
 * T-0123-P3 原始 9 writes（Group/Command/SubField CRUD）+ T-Mml-Admin 扩展 6
 * 个 read + 1 个 batch：
 *   GET    /mml/admin/standard-params         （path 下拉 + autofill 数据源）
 *   GET    /mml/admin/standard-params/:id
 *   GET    /mml/admin/groups                  （admin 全集 group，不限 chapter）
 *   POST   /mml/admin/groups
 *   PATCH  /mml/admin/groups/:id
 *   DELETE /mml/admin/groups/:id
 *   GET    /mml/admin/commands                （含 GroupID/Source/Category/Search 过滤）
 *   GET    /mml/admin/commands/:id
 *   POST   /mml/admin/commands
 *   PATCH  /mml/admin/commands/:id
 *   DELETE /mml/admin/commands/:id
 *   GET    /mml/admin/commands/:id/sub-fields            （admin 全集，含 is_supported=false）
 *   POST   /mml/admin/commands/:id/sub-fields
 *   POST   /mml/admin/commands/:id/sub-fields/batch      （按 path 列表批量建 + autofill）
 *   PATCH  /mml/admin/commands/:id/sub-fields/:sid
 *   DELETE /mml/admin/commands/:id/sub-fields/:sid
 *
 * http.ts 拦截器自动：
 *   - Request body: camelCase → snake_case
 *   - Response data: snake_case → camelCase（部分用 mapBackend* 显式映射）
 *   - Bearer Token 注入 + 自动续期
 */

import http from '../http';
import type {
  BackendGroupAdmin,
  BackendCommandAdmin,
  BackendSubFieldAdmin,
  GroupAdmin,
  CommandAdmin,
  SubFieldAdmin,
  CreateGroupRequest,
  UpdateGroupRequest,
  CreateCommandRequest,
  UpdateCommandRequest,
  CreateSubFieldRequest,
  UpdateSubFieldRequest,
  StandardParamView,
  AdminSubFieldEnriched,
  AdminGroup,
  AdminCommand,
  BatchCreateSubFieldsRequest,
} from '../../types/mmlAdmin';
import {
  mapBackendGroup,
  mapBackendCommand,
  mapBackendSubField,
} from '../../types/mmlAdmin';
import type { PageResponse } from '../../types/pagination';

const BASE = '/mml/admin';

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

  // ================================================================
  // T-Mml-Admin: 列表 / 标准参数 / 批量
  // ================================================================

  /** path 下拉数据源 —— 用户规则 #3：path 必须从 standard_params 选择。
   * `q` 在 standard_path / description 上做 ILIKE %q%。entryType 可选过滤 'parameter' / 'object'。
   */
  async listStandardParams(params: {
    q?: string;
    entryType?: string;
    page?: number;
    pageSize?: number;
  } = {}): Promise<PageResponse<StandardParamView>> {
    const { data } = await http.get<PageResponse<StandardParamView>>(
      `${BASE}/standard-params`,
      {
        params: {
          q: params.q,
          entry_type: params.entryType,
          page: params.page ?? 1,
          page_size: params.pageSize ?? 50,
        },
      },
    );
    return data;
  },

  /** 单条 standard_param 查询 —— autofill 兜底 / 编辑表单 prefill。 */
  async getStandardParam(id: string): Promise<StandardParamView> {
    const { data } = await http.get<StandardParamView>(`${BASE}/standard-params/${id}`);
    return data;
  },

  /** Admin 全集 group 列表（不过滤 chapter 前缀）。 */
  async listGroups(params: {
    source?: string;
    paramVersion?: string;
    q?: string;
  } = {}): Promise<AdminGroup[]> {
    const { data } = await http.get<{ items: AdminGroup[] }>(`${BASE}/groups`, {
      params: { source: params.source, param_version: params.paramVersion, q: params.q },
    });
    return data.items ?? [];
  },

  /** Admin 命令列表：默认 source='standard'，可按 group_id / category / search 过滤。
   * 传 `source='all'` 看全集（不含 customized 仍排除 — customized 在另一张表）。
   */
  async listCommands(params: {
    groupId?: string;
    source?: string;
    category?: string;
    q?: string;
    page?: number;
    pageSize?: number;
  } = {}): Promise<PageResponse<AdminCommand>> {
    const { data } = await http.get<PageResponse<AdminCommand>>(`${BASE}/commands`, {
      params: {
        group_id: params.groupId,
        source: params.source,
        category: params.category,
        q: params.q,
        page: params.page ?? 1,
        page_size: params.pageSize ?? 50,
      },
    });
    return data;
  },

  /** 单条命令详情（编辑表单 prefill）。 */
  async getCommand(id: string): Promise<AdminCommand> {
    const { data } = await http.get<AdminCommand>(`${BASE}/commands/${id}`);
    return data;
  },

  /** Admin 视角 sub_fields 列表（含 is_supported=false 行）。 */
  async listSubFields(commandId: string): Promise<AdminSubFieldEnriched[]> {
    const { data } = await http.get<{ items: AdminSubFieldEnriched[] }>(
      `${BASE}/commands/${commandId}/sub-fields`,
    );
    return data.items ?? [];
  },

  /** 用户规则 #4：按 standard_path_id 列表一次性创建 N 条 sub_field，后端
   * 按 standard_params 元数据自动派生 mml_code / label / sort_order 等。
   */
  async batchCreateSubFields(
    commandId: string,
    req: BatchCreateSubFieldsRequest,
  ): Promise<SubFieldAdmin[]> {
    const { data } = await http.post<{ items: BackendSubFieldAdmin[] }>(
      `${BASE}/commands/${commandId}/sub-fields/batch`,
      req,
    );
    return (data.items ?? []).map(mapBackendSubField);
  },
};
