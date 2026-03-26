import http from '../http';
import type { User, Role, Permission, OperationLog, OperationType, Group } from '@/types/system';
import type { PageRequest, PageResponse } from '@/types/pagination';

// Backend user model
interface BackendUser {
  id: string;
  user_name: string;
  email: string;
  group_names: string[];
  last_login_time: string;
  source: string;
  online_status: string; // online, offline
  lock_status: number; // 0=解锁, 1=有效期锁定, 2=有效期锁定
  built_in: number;
  description: string;
  created_at: string;
  updated_at: string;
}

// Backend role model
interface BackendRole {
  id: string;
  role_name: string;
  batch_operation: number; // 1=是, 0=否
  description: string;
  permissions: BackendPermission[];
  user_count: number;
  upd_user: string;
  upd_time: string;
  is_system: boolean;
  built_in: number;
  created_at: string;
  updated_at: string;
}

interface BackendPermission {
  id: string;
  role_id: string;
  resource: string;
  action: string;
}

// Backend group model
interface BackendGroup {
  id: string;
  group_name: string;
  description: string;
  built_in: number;
  user_count: number;
  role_count: number;
  upd_user: string;
  upd_time: string;
  created_at: string;
  updated_at: string;
}

// Backend audit log model
interface BackendAuditLog {
  id: string;
  user_id?: string;
  username: string;
  action: string;
  resource: string;
  resource_id: string;
  details?: Record<string, unknown>;
  ip_address: string;
  user_agent: string;
  created_at: string;
}

interface BackendListResponse<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

function mapBackendUser(bu: BackendUser): User {
  return {
    id: bu.id,
    userName: bu.user_name,
    email: bu.email || '',
    groupNames: bu.group_names || [],
    lastLoginTime: bu.last_login_time || '',
    source: bu.source || '本地',
    onlineStatus: (bu.online_status === 'online' ? 'online' : 'offline') as 'online' | 'offline',
    lockStatus: (bu.lock_status ?? 0) as 0 | 1 | 2,
    builtIn: bu.built_in ?? 0,
    description: bu.description || '',
    createTime: bu.created_at,
  };
}

function mapFrontendUser(user: Partial<User>): Record<string, unknown> {
  const payload: Record<string, unknown> = {};
  if (user.userName !== undefined) payload.user_name = user.userName;
  if (user.email !== undefined) payload.email = user.email;
  if (user.groupNames !== undefined) payload.group_names = user.groupNames;
  if (user.source !== undefined) payload.source = user.source;
  if (user.lockStatus !== undefined) payload.lock_status = user.lockStatus;
  if (user.description !== undefined) payload.description = user.description;
  return payload;
}

function mapBackendRole(br: BackendRole): Role {
  return {
    id: br.id,
    roleName: br.role_name,
    batchOperation: br.batch_operation ?? 0,
    description: br.description || '',
    permissions: (br.permissions || []).map(
      (p) => `${p.resource}:${p.action}`
    ),
    userCount: br.user_count ?? 0,
    updUser: br.upd_user || '',
    updTime: br.upd_time || br.updated_at || '',
    builtIn: br.built_in ?? (br.is_system ? 1 : 0),
  };
}

function mapBackendAuditLog(ba: BackendAuditLog): OperationLog {
  return {
    id: ba.id,
    operator: ba.username,
    clientIp: ba.ip_address || '',
    module: ba.resource || '',
    operationType: (ba.action || 'query') as OperationType,
    target: ba.resource_id || '',
    content: ba.details ? JSON.stringify(ba.details) : '',
    result: 'success',
    message: '',
    operationTime: ba.created_at,
  };
}

function mapBackendGroup(bg: BackendGroup): Group {
  return {
    id: bg.id,
    groupName: bg.group_name,
    description: bg.description || '',
    userCount: bg.user_count || 0,
    roleCount: bg.role_count || 0,
    builtIn: bg.built_in || 0,
    updUser: bg.upd_user || '',
    updTime: bg.upd_time || bg.updated_at || '',
  };
}

function mapUserListResponse(
  resp: BackendListResponse<BackendUser>
): PageResponse<User> {
  return {
    items: (resp.items || []).map(mapBackendUser),
    total: resp.total,
    page: resp.page,
    pageSize: resp.page_size,
  };
}

export const adminApi = {
  // Users
  async getUsers(
    params: { userName?: string } & PageRequest
  ): Promise<PageResponse<User>> {
    const query: Record<string, unknown> = {
      page: params.page,
      pageSize: params.pageSize,
    };
    if (params.userName) query.search = params.userName;

    const { data } = await http.get<BackendListResponse<BackendUser>>(
      '/admin/users',
      { params: query }
    );
    return mapUserListResponse(data);
  },

  async getAllUsers(): Promise<User[]> {
    const { data } = await http.get<BackendUser[]>('/admin/users');
    return (Array.isArray(data) ? data : []).map(mapBackendUser);
  },

  async getUserById(id: string): Promise<User | null> {
    try {
      const { data } = await http.get<BackendUser>(`/admin/users/${id}`);
      return mapBackendUser(data);
    } catch {
      return null;
    }
  },

  async createUser(
    data: Omit<User, 'id' | 'createTime' | 'lastLoginTime'> & { password: string }
  ): Promise<User> {
    const { data: bu } = await http.post<BackendUser>('/admin/users', {
      user_name: data.userName,
      password: data.password,
      email: data.email || undefined,
      group_names: data.groupNames || [],
      source: data.source || '本地',
      description: data.description,
    });
    return mapBackendUser(bu);
  },

  async updateUser(id: string, data: Partial<User>): Promise<User> {
    const payload = mapFrontendUser(data);
    const { data: bu } = await http.put<BackendUser>(
      `/admin/users/${id}`,
      payload
    );
    return mapBackendUser(bu);
  },

  async deleteUsers(ids: string[]): Promise<void> {
    for (const id of ids) {
      await http.delete(`/admin/users/${id}`);
    }
  },

  async resetPassword(id: string, newPassword: string): Promise<void> {
    await http.post(`/admin/users/${id}/reset-password`, { new_password: newPassword });
  },

  async lockUser(id: string): Promise<void> {
    await http.post(`/admin/users/${id}/lock`);
  },

  async unlockUser(id: string): Promise<void> {
    await http.post(`/admin/users/${id}/unlock`);
  },

  async forceLogout(ids: string[]): Promise<void> {
    await http.post('/admin/users/force-logout', { user_ids: ids });
  },

  async moveUsersToGroup(userIds: string[], groupId: string): Promise<void> {
    await http.post('/admin/users/move-group', { user_ids: userIds, group_id: groupId });
  },

  async copyUser(id: string): Promise<User> {
    const { data } = await http.post<BackendUser>(`/admin/users/${id}/copy`);
    return mapBackendUser(data);
  },

  // Roles
  async getRoles(params: PageRequest & { roleName?: string }): Promise<PageResponse<Role>> {
    const query: Record<string, unknown> = {
      page: params.page,
      pageSize: params.pageSize,
    };
    if (params.roleName) query.search = params.roleName;
    const { data } = await http.get<BackendListResponse<BackendRole>>('/admin/roles', { params: query });
    return {
      items: (data.items || []).map(mapBackendRole),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async getAllRoles(): Promise<Role[]> {
    const { data } = await http.get<BackendRole[]>('/admin/roles');
    return (Array.isArray(data) ? data : []).map(mapBackendRole);
  },

  async getRoleById(id: string): Promise<Role | null> {
    try {
      const { data } = await http.get<BackendRole>(`/admin/roles/${id}`);
      return mapBackendRole(data);
    } catch {
      return null;
    }
  },

  async createRole(data: Omit<Role, 'id' | 'userCount' | 'updUser' | 'updTime'>): Promise<Role> {
    const { data: result } = await http.post<BackendRole>('/admin/roles', {
      role_name: data.roleName,
      batch_operation: data.batchOperation ?? 0,
      description: data.description,
      permissions: (data.permissions || []).map((p) => {
        const parts = p.split(':');
        return { resource: parts[0], action: parts[1] || 'read' };
      }),
    });
    return mapBackendRole(result);
  },

  async updateRole(id: string, data: Partial<Role>): Promise<Role> {
    const payload: Record<string, unknown> = {};
    if (data.roleName !== undefined) payload.role_name = data.roleName;
    if (data.batchOperation !== undefined) payload.batch_operation = data.batchOperation;
    if (data.description !== undefined) payload.description = data.description;
    if (data.permissions !== undefined) {
      payload.permissions = data.permissions.map((p) => {
        const parts = p.split(':');
        return { resource: parts[0], action: parts[1] || 'read' };
      });
    }
    const { data: result } = await http.put<BackendRole>(`/admin/roles/${id}`, payload);
    return mapBackendRole(result);
  },

  async deleteRoles(ids: string[]): Promise<void> {
    for (const id of ids) {
      await http.delete(`/admin/roles/${id}`);
    }
  },

  async getPermissions(): Promise<Permission[]> {
    const { data } = await http.get('/admin/permissions');
    const items = Array.isArray(data) ? data : (data.items || []);
    return items.map((p: { id: string; resource: string; action: string; description?: string }) => ({
      id: p.id,
      permCode: p.resource + ':' + p.action,
      permName: p.resource + ':' + p.action,
      module: p.resource,
      description: p.description || '',
    }));
  },

  // Groups
  async getGroups(params: PageRequest & { groupName?: string }): Promise<PageResponse<Group>> {
    const query: Record<string, unknown> = {
      page: params.page,
      pageSize: params.pageSize,
    };
    if (params.groupName) query.search = params.groupName;
    const { data } = await http.get<BackendListResponse<BackendGroup>>('/admin/groups', { params: query });
    return {
      items: (data.items || []).map(mapBackendGroup),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async getAllGroups(): Promise<Group[]> {
    const { data } = await http.get<BackendGroup[]>('/admin/groups');
    return (Array.isArray(data) ? data : []).map(mapBackendGroup);
  },

  async getGroupById(id: string): Promise<Group | null> {
    try {
      const { data } = await http.get<BackendGroup>(`/admin/groups/${id}`);
      return mapBackendGroup(data);
    } catch {
      return null;
    }
  },

  async createGroup(data: Omit<Group, 'id' | 'userCount' | 'roleCount' | 'updUser' | 'updTime'> & { roleIds?: string[]; userIds?: string[] }): Promise<Group> {
    const { data: result } = await http.post<BackendGroup>('/admin/groups', {
      group_name: data.groupName,
      description: data.description,
      built_in: data.builtIn,
      role_ids: data.roleIds,
      user_ids: data.userIds,
    });
    return mapBackendGroup(result);
  },

  async updateGroup(id: string, data: Partial<Group> & { roleIds?: string[]; userIds?: string[] }): Promise<Group> {
    const payload: Record<string, unknown> = {};
    if (data.groupName !== undefined) payload.group_name = data.groupName;
    if (data.description !== undefined) payload.description = data.description;
    if (data.roleIds !== undefined) payload.role_ids = data.roleIds;
    if (data.userIds !== undefined) payload.user_ids = data.userIds;
    const { data: result } = await http.put<BackendGroup>(`/admin/groups/${id}`, payload);
    return mapBackendGroup(result);
  },

  async deleteGroups(ids: string[]): Promise<void> {
    for (const id of ids) {
      await http.delete(`/admin/groups/${id}`);
    }
  },

  // Audit logs
  async getOperationLogs(
    params: {
      operator?: string;
      module?: string;
      operationType?: OperationType;
      result?: string;
      timeRange?: [string, string];
      keyword?: string;
    } & PageRequest
  ): Promise<PageResponse<OperationLog>> {
    const query: Record<string, unknown> = {
      page: params.page,
      pageSize: params.pageSize,
    };
    if (params.module) query.resource = params.module;
    if (params.operationType) query.action = params.operationType;
    if (params.timeRange) {
      query.start_time = params.timeRange[0];
      query.end_time = params.timeRange[1];
    }

    const { data } = await http.get<BackendListResponse<BackendAuditLog>>(
      '/admin/audit-logs',
      { params: query }
    );

    return {
      items: (data.items || []).map(mapBackendAuditLog),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },
};
