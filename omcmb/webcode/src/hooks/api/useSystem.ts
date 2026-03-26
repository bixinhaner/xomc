import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { User, Role, UserRole, UserStatus, Group } from '@/types/system';
import type { PageRequest } from '@/types/pagination';
import { systemService } from '@/mock/services/systemService';
import { adminApi } from '@/services/api/adminApi';
import { systemApi } from '@/services/api/systemApi';
import { useMock } from '@/services/apiSwitch';

export function useUsers(
  params: { role?: UserRole; status?: UserStatus; keyword?: string } & PageRequest
) {
  return useQuery({
    queryKey: ['system', 'users', params],
    queryFn: () =>
      useMock ? systemService.getUsers(params) : adminApi.getUsers(params),
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
    mutationFn: (data: Omit<User, 'id' | 'createTime' | 'lastLoginTime'>) =>
      useMock ? systemService.createUser(data) : adminApi.createUser(data),
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

export function useResetPassword() {
  return useMutation({
    mutationFn: ({ id, newPassword }: { id: string; newPassword: string }) =>
      useMock ? systemService.resetPassword(id, newPassword) : adminApi.resetPassword(id, newPassword),
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

export function useRoles(params: PageRequest) {
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
    mutationFn: (data: Omit<Role, 'id' | 'userCount'>) =>
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
    mutationFn: async (ids: string[]) => {
      if (useMock) {
        return systemService.deleteRoles(ids);
      }
      for (const id of ids) {
        await adminApi.deleteRole(id);
      }
    },
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
    mutationFn: (data: Omit<Group, 'id' | 'userCount' | 'roleCount' | 'updUser' | 'updTime'>) =>
      useMock ? systemService.createGroup(data) : adminApi.createGroup(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['system', 'groups'] });
    },
  });
}

export function useUpdateGroup() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Group> }) =>
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

export function useSystemInfo() {
  return useQuery({
    queryKey: ['system', 'info'],
    queryFn: () => useMock ? systemService.getSystemInfo() : systemApi.getSystemInfo(),
    refetchInterval: 60000,
  });
}
