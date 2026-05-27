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
  BackendAdminSubFieldEnriched,
  AdminGroup,
  AdminCommand,
  BatchCreateSubFieldsRequest,
} from '../../types/mmlAdmin';
import {
  mapBackendGroup,
  mapBackendCommand,
  mapBackendSubField,
  mapBackendAdminSubFieldEnriched,
  mapBackendStandardParam,
  toBackendCreateGroup,
  toBackendUpdateGroup,
  toBackendCreateCommand,
  toBackendUpdateCommand,
  toBackendCreateSubField,
  toBackendUpdateSubField,
} from '../../types/mmlAdmin';
import type { BackendStandardParam } from '../../types/mmlAdmin';
import type { PageResponse } from '../../types/pagination';

const BASE = '/mml/admin';

export const mmlAdminApi = {
  // ---------------- Groups ----------------
  // 2026-05-27 修复:所有写端点通过 toBackend* 显式 camel→snake + 平铺 zh/en 字段,
  // 因为后端 admin_service.go 用 snake_case JSON tag 且 ParamVersion required,
  // HTTP 拦截器只转 query params 不转 body。
  async createGroup(req: CreateGroupRequest): Promise<GroupAdmin> {
    const { data } = await http.post<BackendGroupAdmin>(
      `${BASE}/groups`,
      toBackendCreateGroup(req),
    );
    return mapBackendGroup(data);
  },

  async updateGroup(id: string, req: UpdateGroupRequest): Promise<GroupAdmin> {
    const { data } = await http.patch<BackendGroupAdmin>(
      `${BASE}/groups/${id}`,
      toBackendUpdateGroup(req),
    );
    return mapBackendGroup(data);
  },

  async deleteGroup(id: string): Promise<void> {
    await http.delete<void>(`${BASE}/groups/${id}`);
  },

  // ---------------- Commands ----------------
  async createCommand(req: CreateCommandRequest): Promise<CommandAdmin> {
    const { data } = await http.post<BackendCommandAdmin>(
      `${BASE}/commands`,
      toBackendCreateCommand(req),
    );
    return mapBackendCommand(data);
  },

  async updateCommand(id: string, req: UpdateCommandRequest): Promise<CommandAdmin> {
    const { data } = await http.patch<BackendCommandAdmin>(
      `${BASE}/commands/${id}`,
      toBackendUpdateCommand(req),
    );
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
      toBackendCreateSubField(req),
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
      toBackendUpdateSubField(req),
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
    const { data } = await http.get<PageResponse<BackendStandardParam>>(
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
    // 2026-05-27 修复:后端 snake_case → 前端 camelCase 显式映射,
    // 否则 standardPath/entryType 等字段全为 undefined。
    return {
      ...data,
      items: (data.items ?? []).map(mapBackendStandardParam),
    };
  },

  /** 单条 standard_param 查询 —— autofill 兜底 / 编辑表单 prefill。 */
  async getStandardParam(id: string): Promise<StandardParamView> {
    const { data } = await http.get<BackendStandardParam>(`${BASE}/standard-params/${id}`);
    return mapBackendStandardParam(data);
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

  /** Admin 视角 sub_fields 列表（含 is_supported=false 行）。
   * 后端返回 snake_case (BackendAdminSubFieldEnriched),前端用 camelCase
   * (AdminSubFieldEnriched),用 mapBackendAdminSubFieldEnriched 显式转换。
   * 修复 2026-05-27:此前直返 data.items 导致 PathListSection 所有字段 undefined。
   */
  async listSubFields(commandId: string): Promise<AdminSubFieldEnriched[]> {
    const { data } = await http.get<{ items: BackendAdminSubFieldEnriched[] }>(
      `${BASE}/commands/${commandId}/sub-fields`,
    );
    return (data.items ?? []).map(mapBackendAdminSubFieldEnriched);
  },

  /** 用户规则 #4：按 standard_path_id 列表一次性创建 N 条 sub_field，后端
   * 按 standard_params 元数据自动派生 mml_code / label / sort_order 等。
   */
  async batchCreateSubFields(
    commandId: string,
    req: BatchCreateSubFieldsRequest,
  ): Promise<SubFieldAdmin[]> {
    // 2026-05-27 修复:后端 binding tag 是 snake_case (standard_path_ids),
    // 前端类型用 camelCase (standardPathIds),HTTP 拦截器不转 body,显式映射。
    const { data } = await http.post<{ items: BackendSubFieldAdmin[] }>(
      `${BASE}/commands/${commandId}/sub-fields/batch`,
      { standard_path_ids: req.standardPathIds },
    );
    return (data.items ?? []).map(mapBackendSubField);
  },
};
