/**
 * T-0123-P3 admin Catalog 管理 React Query hooks — frontend-core
 *
 * 设计要点（PRD §Q.5 cache_version 失效机制）：
 *   - 写 mutation onSuccess → queryClient.invalidateQueries
 *     - 全部失效 ['mml', 'console', 'group-tree'] （Console 读路径）
 *   - 后端 admin_service.go 已在写表后自动 INCR parammodel:cache_version
 *     (T-0123-P0 触发器维护)，其他 ACS 实例 Redis 监听失效
 *
 * 备注：mml_params 表 + XML 导入端点已下线，相关 hooks (useParamsList /
 * useParamReferences / useCreate|Update|DeleteParam / useImportPreview /
 * useImportApply) 同步移除。
 */

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { mmlAdminApi } from '../../services/api/mmlAdminApi';
import type {
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
import type { PageResponse } from '../../types/pagination';

const QK_GROUP_TREE = ['mml', 'console', 'group-tree'];
const QK_ADMIN_GROUPS = ['mml', 'admin', 'groups'];
const QK_ADMIN_COMMANDS = ['mml', 'admin', 'commands'];
const QK_ADMIN_SUB_FIELDS = ['mml', 'admin', 'sub-fields'];
const QK_STANDARD_PARAMS = ['mml', 'admin', 'standard-params'];

function invalidateAfterWrite(qc: ReturnType<typeof useQueryClient>): void {
  // 写入后失效 console 端的读路径（避免老数据残留）+ admin 端的列表缓存
  void qc.invalidateQueries({ queryKey: QK_GROUP_TREE });
  void qc.invalidateQueries({ queryKey: QK_ADMIN_GROUPS });
  void qc.invalidateQueries({ queryKey: QK_ADMIN_COMMANDS });
  void qc.invalidateQueries({ queryKey: QK_ADMIN_SUB_FIELDS });
}

// ============================================================
// Groups
// ============================================================

export function useCreateGroup() {
  const qc = useQueryClient();
  return useMutation<GroupAdmin, Error, CreateGroupRequest>({
    mutationFn: (req) => mmlAdminApi.createGroup(req),
    onSuccess: () => invalidateAfterWrite(qc),
  });
}

export function useUpdateGroup() {
  const qc = useQueryClient();
  return useMutation<GroupAdmin, Error, { id: string; req: UpdateGroupRequest }>({
    mutationFn: ({ id, req }) => mmlAdminApi.updateGroup(id, req),
    onSuccess: () => invalidateAfterWrite(qc),
  });
}

export function useDeleteGroup() {
  const qc = useQueryClient();
  return useMutation<void, Error, string>({
    mutationFn: (id) => mmlAdminApi.deleteGroup(id),
    onSuccess: () => invalidateAfterWrite(qc),
  });
}

// ============================================================
// Commands
// ============================================================

export function useCreateCommand() {
  const qc = useQueryClient();
  return useMutation<CommandAdmin, Error, CreateCommandRequest>({
    mutationFn: (req) => mmlAdminApi.createCommand(req),
    onSuccess: () => invalidateAfterWrite(qc),
  });
}

export function useUpdateCommand() {
  const qc = useQueryClient();
  return useMutation<CommandAdmin, Error, { id: string; req: UpdateCommandRequest }>({
    mutationFn: ({ id, req }) => mmlAdminApi.updateCommand(id, req),
    onSuccess: () => invalidateAfterWrite(qc),
  });
}

export function useDeleteCommand() {
  const qc = useQueryClient();
  return useMutation<void, Error, string>({
    mutationFn: (id) => mmlAdminApi.deleteCommand(id),
    onSuccess: () => invalidateAfterWrite(qc),
  });
}

// ============================================================
// SubFields
// ============================================================

export function useCreateSubField() {
  const qc = useQueryClient();
  return useMutation<
    SubFieldAdmin,
    Error,
    { commandId: string; req: CreateSubFieldRequest }
  >({
    mutationFn: ({ commandId, req }) => mmlAdminApi.createSubField(commandId, req),
    onSuccess: () => invalidateAfterWrite(qc),
  });
}

export function useUpdateSubField() {
  const qc = useQueryClient();
  return useMutation<
    SubFieldAdmin,
    Error,
    { commandId: string; subFieldId: string; req: UpdateSubFieldRequest }
  >({
    mutationFn: ({ commandId, subFieldId, req }) =>
      mmlAdminApi.updateSubField(commandId, subFieldId, req),
    onSuccess: () => invalidateAfterWrite(qc),
  });
}

export function useDeleteSubField() {
  const qc = useQueryClient();
  return useMutation<void, Error, { commandId: string; subFieldId: string }>({
    mutationFn: ({ commandId, subFieldId }) =>
      mmlAdminApi.deleteSubField(commandId, subFieldId),
    onSuccess: () => invalidateAfterWrite(qc),
  });
}

// ============================================================
// T-Mml-Admin: 读 hooks（list + standard params + batch）
// ============================================================

/** standard_params 下拉 —— 用户规则 #3：path 必须从 standard_params 选择。
 * 默认 30s stale，因为字典变更频率极低；user 翻页 / 筛选触发新 fetch。
 */
export function useStandardParamsList(params: {
  q?: string;
  entryType?: string;
  page?: number;
  pageSize?: number;
} = {}) {
  return useQuery<PageResponse<StandardParamView>>({
    queryKey: [...QK_STANDARD_PARAMS, params],
    queryFn: () => mmlAdminApi.listStandardParams(params),
    staleTime: 30_000,
  });
}

export function useStandardParamDetail(id: string | undefined) {
  return useQuery<StandardParamView>({
    queryKey: [...QK_STANDARD_PARAMS, 'detail', id],
    queryFn: () => mmlAdminApi.getStandardParam(id as string),
    enabled: !!id,
    staleTime: 5 * 60_000,
  });
}

/** Admin 视角全集 group。 */
export function useAdminGroupList(params: {
  source?: string;
  paramVersion?: string;
  q?: string;
} = {}) {
  return useQuery<AdminGroup[]>({
    queryKey: [...QK_ADMIN_GROUPS, params],
    queryFn: () => mmlAdminApi.listGroups(params),
    staleTime: 30_000,
  });
}

/** Admin 命令列表（默认 source='standard'）。 */
export function useAdminCommandList(params: {
  groupId?: string;
  source?: string;
  category?: string;
  q?: string;
  page?: number;
  pageSize?: number;
} = {}) {
  return useQuery<PageResponse<AdminCommand>>({
    queryKey: [...QK_ADMIN_COMMANDS, params],
    queryFn: () => mmlAdminApi.listCommands(params),
    staleTime: 30_000,
  });
}

/** Admin 视角 sub_field 列表（含 is_supported=false 行）。 */
export function useAdminSubFieldList(commandId: string | undefined) {
  return useQuery<AdminSubFieldEnriched[]>({
    queryKey: [...QK_ADMIN_SUB_FIELDS, commandId],
    queryFn: () => mmlAdminApi.listSubFields(commandId as string),
    enabled: !!commandId,
    staleTime: 30_000,
  });
}

/** 用户规则 #4：按 path 批量创建 sub_field，后端 autofill 派生默认字段。 */
export function useBatchCreateSubFields() {
  const qc = useQueryClient();
  return useMutation<
    SubFieldAdmin[],
    Error,
    { commandId: string; req: BatchCreateSubFieldsRequest }
  >({
    mutationFn: ({ commandId, req }) => mmlAdminApi.batchCreateSubFields(commandId, req),
    onSuccess: () => invalidateAfterWrite(qc),
  });
}
