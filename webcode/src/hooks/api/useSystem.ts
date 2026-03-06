import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { User, Role, UserRole, UserStatus } from '@/types/system';
import type { PageRequest } from '@/types/pagination';
import { systemService } from '@/mock/services/systemService';

export function useUsers(
  params: { role?: UserRole; status?: UserStatus; keyword?: string } & PageRequest
) {
  return useQuery({
    queryKey: ['system', 'users', params],
    queryFn: () => systemService.getUsers(params),
  });
}

export function useUserById(id: string) {
  return useQuery({
    queryKey: ['system', 'users', 'detail', id],
    queryFn: () => systemService.getUserById(id),
    enabled: Boolean(id),
  });
}

export function useCreateUser() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Omit<User, 'id' | 'createTime' | 'lastLoginTime'>) =>
      systemService.createUser(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['system', 'users'] });
    },
  });
}

export function useUpdateUser() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<User> }) =>
      systemService.updateUser(id, data),
    onSuccess: (_result, { id }) => {
      void queryClient.invalidateQueries({ queryKey: ['system', 'users', 'detail', id] });
      void queryClient.invalidateQueries({ queryKey: ['system', 'users'] });
    },
  });
}

export function useDeleteUsers() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) => systemService.deleteUsers(ids),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['system', 'users'] });
    },
  });
}

export function useResetPassword() {
  return useMutation({
    mutationFn: ({ id, newPassword }: { id: string; newPassword: string }) =>
      systemService.resetPassword(id, newPassword),
  });
}

export function useLockUser() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => systemService.lockUser(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['system', 'users'] });
    },
  });
}

export function useUnlockUser() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => systemService.unlockUser(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['system', 'users'] });
    },
  });
}

export function useRoles(params: PageRequest) {
  return useQuery({
    queryKey: ['system', 'roles', params],
    queryFn: () => systemService.getRoles(params),
    staleTime: 5 * 60 * 1000,
  });
}

export function useAllRoles() {
  return useQuery({
    queryKey: ['system', 'roles', 'all'],
    queryFn: () => systemService.getAllRoles(),
    staleTime: 5 * 60 * 1000,
  });
}

export function useRoleById(id: string) {
  return useQuery({
    queryKey: ['system', 'roles', 'detail', id],
    queryFn: () => systemService.getRoleById(id),
    enabled: Boolean(id),
  });
}

export function useCreateRole() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Omit<Role, 'id' | 'userCount'>) =>
      systemService.createRole(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['system', 'roles'] });
    },
  });
}

export function useUpdateRole() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Role> }) =>
      systemService.updateRole(id, data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['system', 'roles'] });
    },
  });
}

export function useDeleteRoles() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) => systemService.deleteRoles(ids),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['system', 'roles'] });
    },
  });
}

export function usePermissions(params: PageRequest) {
  return useQuery({
    queryKey: ['system', 'permissions', params],
    queryFn: () => systemService.getPermissions(params),
    staleTime: 10 * 60 * 1000,
  });
}

export function useAllPermissions() {
  return useQuery({
    queryKey: ['system', 'permissions', 'all'],
    queryFn: () => systemService.getAllPermissions(),
    staleTime: 10 * 60 * 1000,
  });
}

export function useSystemInfo() {
  return useQuery({
    queryKey: ['system', 'info'],
    queryFn: () => systemService.getSystemInfo(),
    refetchInterval: 60000,
  });
}
