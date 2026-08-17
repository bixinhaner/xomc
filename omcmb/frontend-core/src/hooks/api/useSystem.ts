import { useMemo } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type {
  User,
  UserListParams,
  Role,
  Group,
  ApiEndpointListParams,
  ApiEndpointPayload,
  SysConfigItem,
  BatchUpdateSysConfigPayload,
  BatchUpdateSysConfigResult,
  ConfigApplyStatus,
} from '../../types/system';
import type { Dictionary } from '../../services/api/adminApi';
import type { PageRequest } from '../../types/pagination';
import { systemService } from '../../mock/services/systemService';
import { adminApi } from '../../services/api/adminApi';
import { systemApi } from '../../services/api/systemApi';
import { deviceApi } from '../../services/api/deviceApi';
import { deviceService } from '../../mock/services/deviceService';
import { useMock } from '../../services/apiSwitch';
import { useAppStore } from '../../store/appStore';
import {
  SYSTEM_TIMEZONE_QUERY_KEY,
  TIMEZONE_CONFIG_CATEGORY,
  TIMEZONE_CONFIG_KEY,
} from './useSystemTimezone';

// Users
export function useUsers(params: UserListParams & PageRequest) {
  return useQuery({
    queryKey: ['system', 'users', params],
    queryFn: () =>
      useMock ? systemService.getUsers(params) : adminApi.getUsers(params),
  });
}

export function useAllUsers() {
  return useQuery({
    queryKey: ['system', 'users', 'all'],
    queryFn: () =>
      useMock ? systemService.getAllUsers() : adminApi.getAllUsers(),
    staleTime: 5 * 60 * 1000,
  });
}

export function useUserById(id: string) {
  return useQuery({
    queryKey: ['system', 'users', 'detail', id],
    queryFn: () =>
      useMock ? systemService.getUserById(id) : adminApi.getUserById(id),
    enabled: Boolean(id),
  });
}

export function useCreateUser() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (
      data: Omit<User, 'id' | 'createTime' | 'lastLoginTime'> & {
        password?: string;
        useDefaultPassword?: boolean;
        roleIds?: string[];
      },
    ) => (useMock ? systemService.createUser(data) : adminApi.createUser(data)),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['system', 'users'] });
    },
  });
}

export function useUpdateUser() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<User> }) =>
      useMock ? systemService.updateUser(id, data) : adminApi.updateUser(id, data),
    onSuccess: (_result, { id }) => {
      void queryClient.invalidateQueries({ queryKey: ['system', 'users', 'detail', id] });
      void queryClient.invalidateQueries({ queryKey: ['system', 'users'] });
    },
  });
}

export function useDeleteUsers() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) =>
      useMock ? systemService.deleteUsers(ids) : adminApi.deleteUsers(ids),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['system', 'users'] });
    },
  });
}

// Issue #649：参数对象化，支持 useDefaultPassword 开关；newPassword 二选一。
export function useResetPassword() {
  return useMutation({
    mutationFn: ({
      id,
      newPassword,
      useDefaultPassword,
    }: {
      id: string;
      newPassword?: string;
      useDefaultPassword?: boolean;
    }) =>
      useMock
        ? systemService.resetPassword(id, { newPassword, useDefaultPassword })
        : adminApi.resetPassword(id, { newPassword, useDefaultPassword }),
  });
}

export function useLockUser() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      useMock ? systemService.lockUser(id) : adminApi.lockUser(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['system', 'users'] });
    },
  });
}

export function useUnlockUser() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      useMock ? systemService.unlockUser(id) : adminApi.unlockUser(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['system', 'users'] });
    },
  });
}

export function useForceLogout() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) =>
      useMock ? systemService.forceLogout(ids) : adminApi.forceLogout(ids),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['system', 'users'] });
    },
  });
}

export function useMoveUsersToGroup() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ userIds, groupId }: { userIds: string[]; groupId: string }) =>
      useMock ? systemService.moveUsersToGroup(userIds, groupId) : adminApi.moveUsersToGroup(userIds, groupId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['system', 'users'] });
    },
  });
}

// 批量分配角色（PRD §5.5 / §11.5）：整体替换语义，与 UpdateUser.RoleIDs 一致。
export function useBatchAssignRoles() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ userIds, roleIds }: { userIds: string[]; roleIds: string[] }) =>
      adminApi.batchAssignRoles(userIds, roleIds),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['system', 'users'] });
    },
  });
}

export function useCopyUser() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      useMock ? systemService.copyUser(id) : adminApi.copyUser(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['system', 'users'] });
    },
  });
}

// Roles
export function useRoles(params: PageRequest & { roleName?: string }) {
  return useQuery({
    queryKey: ['system', 'roles', params],
    queryFn: () =>
      useMock ? systemService.getRoles(params) : adminApi.getRoles(params),
    staleTime: 5 * 60 * 1000,
  });
}

export function useAllRoles() {
  return useQuery({
    queryKey: ['system', 'roles', 'all'],
    queryFn: () =>
      useMock ? systemService.getAllRoles() : adminApi.getAllRoles(),
    staleTime: 5 * 60 * 1000,
  });
}

export function useRoleById(id: string) {
  return useQuery({
    queryKey: ['system', 'roles', 'detail', id],
    queryFn: () =>
      useMock ? systemService.getRoleById(id) : adminApi.getRoleById(id),
    enabled: Boolean(id),
  });
}

export function useCreateRole() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Omit<Role, 'id' | 'userCount' | 'updUser' | 'updTime'>) =>
      useMock ? systemService.createRole(data) : adminApi.createRole(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['system', 'roles'] });
    },
  });
}

export function useUpdateRole() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Role> }) =>
      useMock ? systemService.updateRole(id, data) : adminApi.updateRole(id, data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['system', 'roles'] });
    },
  });
}

export function useDeleteRoles() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) =>
      useMock ? systemService.deleteRoles(ids) : adminApi.deleteRoles(ids),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['system', 'roles'] });
    },
  });
}

export function usePermissions(params: PageRequest) {
  return useQuery({
    queryKey: ['system', 'permissions', params],
    queryFn: async () => {
      if (useMock) return systemService.getPermissions(params);
      const all = await adminApi.getPermissions();
      const start = (params.page - 1) * params.pageSize;
      const paged = all.slice(start, start + params.pageSize);
      return { items: paged, total: all.length, page: params.page, pageSize: params.pageSize };
    },
    staleTime: 10 * 60 * 1000,
  });
}

export function useAllPermissions() {
  return useQuery({
    queryKey: ['system', 'permissions', 'all'],
    queryFn: () =>
      useMock ? systemService.getAllPermissions() : adminApi.getPermissions(),
    staleTime: 10 * 60 * 1000,
  });
}

// Groups
export function useGroups(params: PageRequest & { groupName?: string }) {
  return useQuery({
    queryKey: ['system', 'groups', params],
    queryFn: () =>
      useMock ? systemService.getGroups(params) : adminApi.getGroups(params),
  });
}

export function useAllGroups() {
  return useQuery({
    queryKey: ['system', 'groups', 'all'],
    queryFn: () =>
      useMock ? systemService.getAllGroups() : adminApi.getAllGroups(),
    staleTime: 5 * 60 * 1000,
  });
}

export function useGroupById(id: string) {
  return useQuery({
    queryKey: ['system', 'groups', 'detail', id],
    queryFn: () =>
      useMock ? systemService.getGroupById(id) : adminApi.getGroupById(id),
    enabled: Boolean(id),
  });
}

export function useCreateGroup() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Omit<Group, 'id' | 'userCount' | 'roleCount' | 'updUser' | 'updTime'> & { roleIds?: string[]; userIds?: string[] }) =>
      useMock ? systemService.createGroup(data) : adminApi.createGroup(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['system', 'groups'] });
    },
  });
}

export function useUpdateGroup() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Group> & { roleIds?: string[]; userIds?: string[] } }) =>
      useMock ? systemService.updateGroup(id, data) : adminApi.updateGroup(id, data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['system', 'groups'] });
    },
  });
}

export function useDeleteGroups() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) =>
      useMock ? systemService.deleteGroups(ids) : adminApi.deleteGroups(ids),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['system', 'groups'] });
    },
  });
}

// System info
export function useSystemInfo() {
  return useQuery({
    queryKey: ['system', 'info'],
    queryFn: () => useMock ? systemService.getSystemInfo() : systemApi.getSystemInfo(),
    refetchInterval: 60000,
  });
}

// Device Groups
//
// 历史 bug：此 hook 之前直写 `deviceService.getGroups()`（mock 服务），导致
// 角色/数据权限编辑面板始终显示 mock 树（grp-default、grp-bj…），切真后端
// 提交时后端 uuid.Parse 报 "invalid UUID length: 11"。
// 修复：按 useMock 双轨切换；真实后端走 deviceApi.getGroups()。
export function useAllDeviceGroups() {
  return useQuery({
    queryKey: ['system', 'deviceGroups', 'all'],
    queryFn: async () => {
      const result = useMock
        ? await deviceService.getGroups()
        : await deviceApi.getGroups();
      return result.groups;
    },
    staleTime: 5 * 60 * 1000,
  });
}

// ---- API Management ----
export function useApiEndpoints(params: ApiEndpointListParams) {
  return useQuery({
    queryKey: ['system', 'apiEndpoints', params],
    queryFn: () => adminApi.getApiEndpoints(params),
  });
}

export function useApiGroups() {
  return useQuery({
    queryKey: ['system', 'apiGroups'],
    queryFn: () => adminApi.getApiGroups(),
    staleTime: 5 * 60 * 1000,
  });
}

export function useCreateApiEndpoint() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: ApiEndpointPayload) => adminApi.createApiEndpoint(payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['system', 'apiEndpoints'] });
      void queryClient.invalidateQueries({ queryKey: ['system', 'apiGroups'] });
    },
  });
}

export function useUpdateApiEndpoint() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: Partial<ApiEndpointPayload> }) =>
      adminApi.updateApiEndpoint(id, payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['system', 'apiEndpoints'] });
      void queryClient.invalidateQueries({ queryKey: ['system', 'apiGroups'] });
    },
  });
}

export function useDeleteApiEndpoint() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => adminApi.deleteApiEndpoint(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['system', 'apiEndpoints'] });
    },
  });
}

export function useBatchDeleteApiEndpoints() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) => adminApi.batchDeleteApiEndpoints(ids),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['system', 'apiEndpoints'] });
    },
  });
}

export function useSyncApiEndpoints() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => adminApi.syncApiEndpoints(),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['system', 'apiEndpoints'] });
      void queryClient.invalidateQueries({ queryKey: ['system', 'apiGroups'] });
    },
  });
}

// ---- Dictionary hooks ----

export function useDictionary(dictType: string) {
  return useQuery<Dictionary | null>({
    queryKey: ['dictionary', dictType],
    queryFn: () => adminApi.findDictionaryByType(dictType),
    enabled: !!dictType,
    staleTime: 5 * 60 * 1000,
  });
}

// 批量获取字典 Hook（性能优化：一次请求获取多个字典）
export function useDictionaryBatch(codes: string[]) {
  // 排序 codes 以确保 queryKey 稳定性
  const sortedCodes = useMemo(() => {
    const sorted = [...codes].sort();
    return sorted;
  }, [codes]);

  return useQuery({
    queryKey: ['dictionary', 'batch', sortedCodes],
    queryFn: () => adminApi.batchGetDicts(sortedCodes),
    staleTime: 10 * 60 * 1000, // 10 分钟缓存
    enabled: codes.length > 0,
  });
}

// ---- System Config (sys_configs) hooks — system/config 页面用 ----
// 参 omgo/docs/prd/system/config.md。enabled 控制只在 active tab 切到时拉取。
export function useSysConfigsByCategory(category: string, enabled = true) {
  return useQuery<SysConfigItem[]>({
    queryKey: ['system', 'sysConfig', category],
    queryFn: () => adminApi.getSysConfigsByCategory(category),
    enabled: enabled && !!category,
    staleTime: 30_000,
  });
}

export function useBatchUpdateSysConfigs() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: BatchUpdateSysConfigPayload) => adminApi.batchUpdateSysConfigs(payload),
    onSuccess: (_result: BatchUpdateSysConfigResult, variables) => {
      void queryClient.invalidateQueries({ queryKey: ['system', 'sysConfig', variables.category] });
      if (variables.category === TIMEZONE_CONFIG_CATEGORY) {
        const timezoneItem = variables.items.find((item) => item.key === TIMEZONE_CONFIG_KEY);
        if (!timezoneItem) return;
        const timezone = timezoneItem.value.trim();
        if (timezone) {
          useAppStore.getState().setSystemTimezone(timezone);
          queryClient.setQueryData<SysConfigItem[] | undefined>(
            SYSTEM_TIMEZONE_QUERY_KEY,
            (current) => current?.map((item) => (
              item.key === TIMEZONE_CONFIG_KEY ? { ...item, value: timezone } : item
            )),
          );
        }
        void queryClient.invalidateQueries({ queryKey: SYSTEM_TIMEZONE_QUERY_KEY });
      }
    },
  });
}

export function useSendTestEmail() {
  return useMutation({
    mutationFn: (recipient: string) => adminApi.sendTestEmail(recipient),
  });
}

export function sysConfigApplyRefetchInterval(status: ConfigApplyStatus | undefined): number | false {
  if (status === 'applied') return false;
  if (status === 'failed') return 15_000;
  return 2_000;
}

export function useSysConfigApplyBatch(id: string | undefined) {
  return useQuery({
    queryKey: ['system', 'sysConfig', 'applyBatch', id],
    queryFn: () => adminApi.getSysConfigApplyBatch(id as string),
    enabled: Boolean(id),
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      return sysConfigApplyRefetchInterval(status);
    },
  });
}

// UI 定制化资产上传 — POST /admin/uploads/ui-asset。
// 参 omgo/docs/prd/system/ui-customization.md §6.1。
export function useUploadUIAsset() {
  return useMutation({
    mutationFn: (vars: { file: File | Blob; kind: 'login_bg' | 'logo_small' | 'logo_large' }) =>
      adminApi.uploadUIAsset(vars.file, vars.kind),
  });
}
