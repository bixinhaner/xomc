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

import { useMutation, useQueryClient } from '@tanstack/react-query';
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
} from '../../types/mmlAdmin';

const QK_GROUP_TREE = ['mml', 'console', 'group-tree'];

function invalidateAfterWrite(qc: ReturnType<typeof useQueryClient>): void {
  void qc.invalidateQueries({ queryKey: QK_GROUP_TREE });
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
