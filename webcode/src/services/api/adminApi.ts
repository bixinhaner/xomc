import http from '../http';
import type { User, Role, UserRole, UserStatus, OperationLog, OperationType } from '@/types/system';
import type { PageRequest, PageResponse } from '@/types/pagination';
import { systemService } from '@/mock/services/systemService';

// Backend user model
interface BackendUser {
  id: string;
  username: string;
  display_name: string;
  email: string;
  carrier?: string;
  status: string; // active, disabled
  roles: BackendRole[];
  last_login_at?: string;
  created_at: string;
  updated_at: string;
}

// Backend role model
interface BackendRole {
  id: string;
  name: string;
  description: string;
  is_system: boolean;
  permissions: BackendPermission[];
  created_at: string;
  updated_at: string;
}

interface BackendPermission {
  id: string;
  role_id: string;
  resource: string;
  action: string;
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

function mapBackendStatus(status: string): UserStatus {
  if (status === 'disabled') return 'inactive';
  return status as UserStatus; // 'active' passes through
}

function mapFrontendStatus(status: UserStatus): string {
  if (status === 'inactive') return 'disabled';
  return status; // 'active' passes through; 'locked' has no backend equivalent
}

function mapBackendUser(bu: BackendUser): User {
  return {
    id: bu.id,
    username: bu.username,
    displayName: bu.display_name || '',
    email: bu.email || '',
    phone: '', // backend has no phone field
    role: (bu.roles?.[0]?.name || 'viewer') as UserRole,
    status: mapBackendStatus(bu.status),
    lastLoginTime: bu.last_login_at || '',
    createTime: bu.created_at,
  };
}

function mapBackendRole(br: BackendRole): Role {
  return {
    id: br.id,
    roleName: br.name,
    description: br.description || '',
    permissions: (br.permissions || []).map(
      (p) => `${p.resource}:${p.action}`
    ),
    userCount: 0, // backend does not provide user count
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

// Cache roles for role name → ID lookup
let cachedRoles: BackendRole[] | null = null;

async function fetchBackendRoles(): Promise<BackendRole[]> {
  if (cachedRoles) return cachedRoles;
  const { data } = await http.get<BackendRole[]>('/admin/roles');
  cachedRoles = Array.isArray(data) ? data : [];
  return cachedRoles;
}

export const adminApi = {
  // Users
  async getUsers(
    params: { role?: UserRole; status?: UserStatus; keyword?: string } & PageRequest
  ): Promise<PageResponse<User>> {
    const query: Record<string, unknown> = {
      page: params.page,
      pageSize: params.pageSize,
    };
    if (params.status) query.status = mapFrontendStatus(params.status);
    if (params.keyword) query.search = params.keyword;

    const { data } = await http.get<BackendListResponse<BackendUser>>(
      '/admin/users',
      { params: query }
    );
    let result = mapUserListResponse(data);

    // Client-side filter by role (backend doesn't support role filter)
    if (params.role) {
      result = {
        ...result,
        items: result.items.filter((u) => u.role === params.role),
        total: result.items.filter((u) => u.role === params.role).length,
      };
    }

    return result;
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
    data: Omit<User, 'id' | 'createTime' | 'lastLoginTime'>
  ): Promise<User> {
    // Resolve role name to role ID
    const roles = await fetchBackendRoles();
    const matchedRole = roles.find(
      (r) => r.name === data.role || r.name.toLowerCase() === data.role
    );

    const { data: bu } = await http.post<BackendUser>('/admin/users', {
      username: data.username,
      password: 'Default@123', // Default password for new users
      display_name: data.displayName,
      email: data.email || undefined,
      role_ids: matchedRole ? [matchedRole.id] : [],
    });
    return mapBackendUser(bu);
  },

  async updateUser(id: string, data: Partial<User>): Promise<User> {
    const body: Record<string, unknown> = {};
    if (data.displayName !== undefined) body.display_name = data.displayName;
    if (data.email !== undefined) body.email = data.email;
    if (data.status !== undefined) body.status = mapFrontendStatus(data.status);

    const { data: bu } = await http.put<BackendUser>(
      `/admin/users/${id}`,
      body
    );
    return mapBackendUser(bu);
  },

  async deleteUsers(ids: string[]): Promise<void> {
    for (const id of ids) {
      await http.delete(`/admin/users/${id}`);
    }
  },

  // Roles — backend only has list (returns full array, not paginated)
  async getRoles(params: PageRequest): Promise<PageResponse<Role>> {
    const roles = await fetchBackendRoles();
    const mapped = roles.map(mapBackendRole);
    // Client-side pagination
    const start = (params.page - 1) * params.pageSize;
    const paged = mapped.slice(start, start + params.pageSize);
    return {
      items: paged,
      total: mapped.length,
      page: params.page,
      pageSize: params.pageSize,
    };
  },

  async getAllRoles(): Promise<Role[]> {
    const roles = await fetchBackendRoles();
    return roles.map(mapBackendRole);
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

  // Operations not in backend — delegate to mock
  resetPassword: systemService.resetPassword.bind(systemService),
  lockUser: systemService.lockUser.bind(systemService),
  unlockUser: systemService.unlockUser.bind(systemService),
};
