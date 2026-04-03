import http from '../http';
import type { User, Role, Permission, OperationLog, OperationType, Group, MenuItem, CreateMenuRequest, UpdateMenuRequest, MenuFilter, MenuListResponse, RoleWithMenus, SetRoleMenusRequest } from '@/types/system';
import type { PageRequest, PageResponse } from '@/types/pagination';

// Backend user model - matches Go User struct JSON tags
interface BackendUser {
  id: string;
  username: string;
  displayName?: string;
  email?: string;
  carrier?: string;
  status: string; // active, disabled
  roles?: BackendRole[];
  failedLoginAttempts?: number;
  lockedUntil?: string;
  lastFailedLoginAt?: string;
  lastLoginAt?: string;
  createdAt: string;
  updatedAt: string;
}

// Backend role model - matches Go Role struct JSON tags
interface BackendRole {
  id: string;
  name: string;
  description: string;
  isSystem: boolean;
  status?: string; // active, disabled - may be empty
  createdAt: string;
  updatedAt: string;
  // Fields that may or may not be included in list response
  permissions?: BackendPermission[];
  deviceGroupIds?: string[];
  menuIds?: string[];
  menus?: BackendMenu[];
  userCount?: number;
  createdBy?: string;
  updatedBy?: string;
}

// Backend menu model
interface BackendMenu {
  id: string;
  name: string;
  title: string;
  icon?: string;
  path?: string;
  component?: string;
  type: 'menu' | 'button' | 'link';
  parentId?: string;
  sortOrder: number;
  status: 'active' | 'disabled';
  visible: boolean;
  children?: BackendMenu[];
  createdAt: string;
  updatedAt: string;
}

interface BackendPermission {
  id: string;
  roleId: string;
  resource: string;
  action: string;
}

// Backend group model
interface BackendGroup {
  id: string;
  groupName: string;
  description: string;
  builtIn: number;
  userCount: number;
  roleCount: number;
  updUser?: string;
  updTime?: string;
  createdAt?: string;
  updatedAt?: string;
}

// Backend audit log model
interface BackendAuditLog {
  id: string;
  userId?: string;
  username: string;
  action: string;
  resource: string;
  resourceId?: string;
  details?: Record<string, unknown>;
  ipAddress?: string;
  userAgent?: string;
  createdAt: string;
}

interface BackendListResponse<T> {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

function mapBackendUser(bu: BackendUser): User {
  // Map status: active -> enabled, disabled -> disabled
  const status: UserStatus = bu.status === 'active' ? 'enabled' : 'disabled';

  // Get role names
  const groupNames = (bu.roles || []).map(r => r.name);

  return {
    id: bu.id,
    userName: bu.username,
    email: bu.email || '',
    phone: '', // Backend doesn't have phone field yet
    groupNames,
    department: '', // Backend doesn't have department field yet
    lastLoginTime: bu.lastLoginAt || undefined,
    source: bu.carrier || '本地',
    onlineStatus: 'offline', // Backend doesn't have online status yet, default to offline
    status,
    expireTime: bu.lockedUntil || undefined,
    builtIn: 0, // Backend doesn't have builtIn field for users
    description: bu.displayName || '',
    createTime: bu.createdAt,
    updateTime: bu.updatedAt,
    createUser: undefined,
    updateUser: undefined,
  };
}

function mapFrontendUser(user: Partial<User>): Record<string, unknown> {
  const payload: Record<string, unknown> = {};
  if (user.userName !== undefined) payload.username = user.userName;
  if (user.email !== undefined) payload.email = user.email;
  if (user.description !== undefined) payload.displayName = user.description;
  if (user.status !== undefined) {
    // Map 'enabled' -> 'active', 'disabled' -> 'disabled'
    payload.status = user.status === 'enabled' ? 'active' : 'disabled';
  }
  return payload;
}

function mapBackendRole(br: BackendRole): Role {
  // Map permissions to string array
  const permissions = (br.permissions || []).map(
    (p) => `${p.resource}:${p.action}`
  );

  // Ensure required fields have values
  const roleName = br.name || '';

  return {
    id: br.id,
    roleName,
    roleCode: roleName,
    batchOperation: 0, // Backend doesn't have this field
    description: br.description || '',
    permissions,
    deviceGroupIds: br.deviceGroupIds,
    userCount: br.userCount ?? 0,
    builtIn: br.isSystem ? 1 : 0,
    createUser: br.createdBy,
    updateUser: br.updatedBy,
    createTime: br.createdAt,
    updateTime: br.updatedAt,
  };
}

function mapBackendAuditLog(ba: BackendAuditLog): OperationLog {
  return {
    id: ba.id,
    operator: ba.username,
    clientIp: ba.ipAddress || '',
    module: ba.resource || '',
    operationType: (ba.action || 'query') as OperationType,
    target: ba.resourceId || '',
    content: ba.details ? JSON.stringify(ba.details) : '',
    result: 'success',
    message: '',
    operationTime: ba.createdAt,
  };
}

function mapBackendGroup(bg: BackendGroup): Group {
  return {
    id: bg.id,
    groupName: bg.groupName,
    description: bg.description || '',
    userCount: bg.userCount || 0,
    roleCount: bg.roleCount || 0,
    builtIn: bg.builtIn || 0,
    updUser: bg.updUser || '',
    updTime: bg.updTime || bg.updatedAt || '',
  };
}

function mapUserListResponse(
  resp: BackendListResponse<BackendUser>
): PageResponse<User> {
  return {
    items: (resp.items || []).map(mapBackendUser),
    total: resp.total,
    page: resp.page,
    pageSize: resp.pageSize,
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
      username: data.userName,
      password: data.password,
      email: data.email || undefined,
      displayName: data.description || '',
      status: data.status === 'enabled' ? 'active' : 'disabled',
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
    await http.post(`/admin/users/${id}/reset-password`, { newPassword });
  },

  async lockUser(id: string): Promise<void> {
    await http.post(`/admin/users/${id}/lock`);
  },

  async unlockUser(id: string): Promise<void> {
    await http.post(`/admin/users/${id}/unlock`);
  },

  async forceLogout(ids: string[]): Promise<void> {
    await http.post('/admin/users/force-logout', { userIds: ids });
  },

  async moveUsersToGroup(userIds: string[], groupId: string): Promise<void> {
    await http.post('/admin/users/move-group', { userIds, groupId });
  },

  async copyUser(id: string): Promise<User> {
    const { data } = await http.post<BackendUser>(`/admin/users/${id}/copy`);
    return mapBackendUser(data);
  },

  // Roles - backend now supports pagination
  async getRoles(params: PageRequest & { roleName?: string }): Promise<PageResponse<Role>> {
    const query: Record<string, unknown> = {
      page: params.page,
      pageSize: params.pageSize,
    };
    if (params.roleName) query.name = params.roleName;

    const { data } = await http.get<BackendListResponse<BackendRole>>('/admin/roles', { params: query });
    return {
      items: (data.items || []).map(mapBackendRole),
      total: data.total,
      page: data.page,
      pageSize: data.pageSize,
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

  async createRole(data: Omit<Role, 'id' | 'userCount' | 'createUser' | 'updateUser' | 'createTime' | 'updateTime'>): Promise<Role> {
    const { data: result } = await http.post<BackendRole>('/admin/roles', {
      name: data.roleName,
      description: data.description,
      permissions: (data.permissions || []).map((p) => {
        const parts = p.split(':');
        return { resource: parts[0], action: parts[1] || 'read' };
      }),
      deviceGroupIds: data.deviceGroupIds,
      menuIds: data.menuIds || [],
    });
    return mapBackendRole(result);
  },

  async updateRole(id: string, data: Partial<Role>): Promise<Role> {
    const payload: Record<string, unknown> = {};
    if (data.roleName !== undefined) payload.name = data.roleName;
    if (data.description !== undefined) payload.description = data.description;
    if (data.permissions !== undefined) {
      payload.permissions = data.permissions.map((p) => {
        const parts = p.split(':');
        return { resource: parts[0], action: parts[1] || 'read' };
      });
    }
    if (data.deviceGroupIds !== undefined) payload.deviceGroupIds = data.deviceGroupIds;
    if (data.menuIds !== undefined) payload.menuIds = data.menuIds;
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
      pageSize: data.pageSize,
    };
  },

  async getAllGroups(): Promise<Group[]> {
    const { data } = await http.get<BackendRole[]>('/admin/roles/all');
    // 将 BackendRole[] 映射为 Group[]，因为用户管理页面需要的是角色
    const roles: BackendRole[] = Array.isArray(data) ? data : [];
    return roles.map((role) => ({
      id: role.id,
      groupName: role.name,
      description: role.description,
      builtIn: role.isSystem ? 1 : 0,
      userCount: role.userCount || 0,
      roleCount: 0, // 角色没有"角色数"概念
      updUser: role.updatedBy || '',
      updTime: role.updatedAt || '',
    }));
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
      groupName: data.groupName,
      description: data.description,
      builtIn: data.builtIn,
      roleIds: data.roleIds,
      userIds: data.userIds,
    });
    return mapBackendGroup(result);
  },

  async updateGroup(id: string, data: Partial<Group> & { roleIds?: string[]; userIds?: string[] }): Promise<Group> {
    const payload: Record<string, unknown> = {};
    if (data.groupName !== undefined) payload.groupName = data.groupName;
    if (data.description !== undefined) payload.description = data.description;
    if (data.roleIds !== undefined) payload.roleIds = data.roleIds;
    if (data.userIds !== undefined) payload.userIds = data.userIds;
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
      query.startTime = params.timeRange[0];
      query.endTime = params.timeRange[1];
    }

    const { data } = await http.get<BackendListResponse<BackendAuditLog>>(
      '/admin/audit-logs',
      { params: query }
    );

    return {
      items: (data.items || []).map(mapBackendAuditLog),
      total: data.total,
      page: data.page,
      pageSize: data.pageSize,
    };
  },

  // Menus (backend returns camelCase directly, no mapping needed)
  async getMenus(params: MenuFilter = {}): Promise<MenuListResponse> {
    const query: Record<string, unknown> = {};
    if (params.page !== undefined) query.page = params.page;
    if (params.pageSize !== undefined) query.pageSize = params.pageSize;
    if (params.type) query.type = params.type;
    if (params.status) query.status = params.status;
    if (params.parentId) query.parentId = params.parentId;

    const { data } = await http.get<MenuListResponse>('/admin/menus', { params: query });
    return data;
  },

  async getMenuTree(): Promise<MenuItem[]> {
    const { data } = await http.get<{data: MenuItem[]}>('/admin/menus/tree');
    return data.data || [];
  },

  async getUserMenuTree(): Promise<MenuItem[]> {
    const { data } = await http.get<{data: MenuItem[]}>('/admin/menus/user');
    return data.data || [];
  },

  async getMenuById(id: string): Promise<MenuItem | null> {
    try {
      const { data } = await http.get<MenuItem>(`/admin/menus/${id}`);
      return data;
    } catch {
      return null;
    }
  },

  async createMenu(data: CreateMenuRequest): Promise<MenuItem> {
    const { data: result } = await http.post<MenuItem>('/admin/menus', data);
    return result;
  },

  async updateMenu(id: string, data: UpdateMenuRequest): Promise<MenuItem> {
    const { data: result } = await http.put<MenuItem>(`/admin/menus/${id}`, data);
    return result;
  },

  async deleteMenus(ids: string[]): Promise<void> {
    await http.delete('/admin/menus', { data: { ids } });
  },

  // Role Menus
  async setRoleMenus(roleId: string, data: SetRoleMenusRequest): Promise<void> {
    await http.put(`/admin/roles/${roleId}/menus`, data);
  },

  async getRoleMenus(roleId: string): Promise<RoleWithMenus | null> {
    try {
      const { data } = await http.get<{data: RoleWithMenus}>(`/admin/roles/${roleId}/menus`);
      return data.data;
    } catch {
      return null;
    }
  },
};
