/**
 * T-0123-P3 admin Catalog 管理 React Query hooks — frontend-core
 *
 * 设计要点（PRD §Q.5 cache_version 失效机制）：
 *   - 写 mutation onSuccess → queryClient.invalidateQueries
 *     - 全部失效 ['mml', 'console', 'group-tree'] （Console 读路径）
 *     - 命中资源失效 ['mml', 'admin', '<resource>']
 *   - 后端 admin_service.go 已在写表后自动 INCR parammodel:cache_version
 *     (T-0123-P0 触发器维护)，其他 ACS 实例 Redis 监听失效
 */

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { mmlAdminApi } from '../../services/api/mmlAdminApi';
import type {
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

const QK_GROUP_TREE = ['mml', 'console', 'group-tree'];
const QK_ADMIN_PARAMS = ['mml', 'admin', 'params'];

function invalidateAfterWrite(qc: ReturnType<typeof useQueryClient>): void {
  void qc.invalidateQueries({ queryKey: QK_GROUP_TREE });
  void qc.invalidateQueries({ queryKey: QK_ADMIN_PARAMS });
}

// ============================================================
// Params list (only read endpoint on admin side)
// ============================================================

export function useParamsList(req: ListParamsRequest = {}) {
  return useQuery<ListParamsResponse>({
    queryKey: [...QK_ADMIN_PARAMS, req],
    queryFn: () => mmlAdminApi.listParams(req),
    staleTime: 5 * 60 * 1000,
  });
}

/**
 * T-0131: param 反向查 — 返回引用该 param 的命令列表（admin Tab 3 抽屉用）。
 * enabled = Boolean(paramId) 避免首次渲染抽屉关闭时空查。
 */
export function useParamReferences(paramId: string | undefined) {
  return useQuery<ParamReference[]>({
    queryKey: [...QK_ADMIN_PARAMS, paramId ?? '', 'references'],
    queryFn: () => mmlAdminApi.listParamReferences(paramId!),
    enabled: Boolean(paramId),
    staleTime: 5 * 60 * 1000,
  });
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
// Params (write)
// ============================================================

export function useCreateParam() {
  const qc = useQueryClient();
  return useMutation<ParamAdmin, Error, CreateParamRequest>({
    mutationFn: (req) => mmlAdminApi.createParam(req),
    onSuccess: () => invalidateAfterWrite(qc),
  });
}

export function useUpdateParam() {
  const qc = useQueryClient();
  return useMutation<ParamAdmin, Error, { id: string; req: UpdateParamRequest }>({
    mutationFn: ({ id, req }) => mmlAdminApi.updateParam(id, req),
    onSuccess: () => invalidateAfterWrite(qc),
  });
}

export function useDeleteParam() {
  const qc = useQueryClient();
  return useMutation<void, Error, string>({
    mutationFn: (id) => mmlAdminApi.deleteParam(id),
    onSuccess: () => invalidateAfterWrite(qc),
  });
}

// ============================================================
// T-0132 admin Tab 4 XML 导入：preview + apply
// ============================================================

/**
 * dry-run 预览：上传 XML → 后端解析 + 三桶 diff。不写表，可重复调用。
 * 不 invalidate（preview 是只读 + diff 视图，不影响 group-tree / params 数据）。
 */
export function useImportPreview() {
  return useMutation<
    ImportPreviewResp,
    Error,
    { file: File; versionCode: string }
  >({
    mutationFn: ({ file, versionCode }) => mmlAdminApi.importPreview(file, versionCode),
  });
}

/**
 * 真写入：批量 UPSERT 后强制 invalidate group-tree + params list 让 UI 拉新数据。
 * Apply 与 Preview 是独立两步调用（防 server-side session）；UI 应让用户在 Preview 完成后
 * 显式点 "确认导入" 按钮。
 */
export function useImportApply() {
  const qc = useQueryClient();
  return useMutation<
    ImportApplyResp,
    Error,
    { file: File; versionCode: string }
  >({
    mutationFn: ({ file, versionCode }) => mmlAdminApi.importApply(file, versionCode),
    onSuccess: () => invalidateAfterWrite(qc),
  });
}
