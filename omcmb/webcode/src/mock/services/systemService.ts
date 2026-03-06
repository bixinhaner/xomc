import type { User, Role, Permission, UserRole, UserStatus } from '@/types/system';
import type { PageRequest, PageResponse } from '@/types/pagination';
import { mockUsers, mockRoles, mockPermissions } from '../data/system';
import { delay, paginate, generateId } from '../utils';

let users = [...mockUsers];
let roles = [...mockRoles];

export const systemService = {
  // Users
  async getUsers(
    params: { role?: UserRole; status?: UserStatus; keyword?: string } & PageRequest
  ): Promise<PageResponse<User>> {
    await delay(100, 200);
    let filtered = [...users];
    if (params.role) filtered = filtered.filter((u) => u.role === params.role);
    if (params.status) filtered = filtered.filter((u) => u.status === params.status);
    if (params.keyword) {
      const kw = params.keyword.toLowerCase();
      filtered = filtered.filter(
        (u) =>
          u.username.toLowerCase().includes(kw) ||
          u.displayName.includes(kw) ||
          u.email.toLowerCase().includes(kw)
      );
    }
    return paginate(filtered, params.page, params.pageSize);
  },

  async getUserById(id: string): Promise<User | null> {
    await delay(80, 150);
    return users.find((u) => u.id === id) ?? null;
  },

  async createUser(data: Omit<User, 'id' | 'createTime' | 'lastLoginTime'>): Promise<User> {
    await delay(200, 400);
    const newUser: User = {
      ...data,
      id: generateId('user'),
      lastLoginTime: new Date().toISOString(),
      createTime: new Date().toISOString(),
    };
    users.push(newUser);
    return newUser;
  },

  async updateUser(id: string, data: Partial<User>): Promise<User> {
    await delay(150, 300);
    const idx = users.findIndex((u) => u.id === id);
    if (idx === -1) throw new Error(`User ${id} not found`);
    users[idx] = { ...users[idx], ...data };
    return users[idx];
  },

  async deleteUsers(ids: string[]): Promise<void> {
    await delay(150, 300);
    users = users.filter((u) => !ids.includes(u.id));
  },

  async resetPassword(id: string, newPassword: string): Promise<void> {
    await delay(300, 600);
    void id;
    void newPassword;
  },

  async lockUser(id: string): Promise<void> {
    await delay(150, 300);
    const idx = users.findIndex((u) => u.id === id);
    if (idx !== -1) users[idx] = { ...users[idx], status: 'locked' };
  },

  async unlockUser(id: string): Promise<void> {
    await delay(150, 300);
    const idx = users.findIndex((u) => u.id === id);
    if (idx !== -1) users[idx] = { ...users[idx], status: 'active' };
  },

  // Roles
  async getRoles(params: PageRequest): Promise<PageResponse<Role>> {
    await delay(80, 150);
    return paginate(roles, params.page, params.pageSize);
  },

  async getAllRoles(): Promise<Role[]> {
    await delay(80, 150);
    return roles;
  },

  async getRoleById(id: string): Promise<Role | null> {
    await delay(80, 150);
    return roles.find((r) => r.id === id) ?? null;
  },

  async createRole(data: Omit<Role, 'id' | 'userCount'>): Promise<Role> {
    await delay(200, 400);
    const newRole: Role = {
      ...data,
      id: generateId('role'),
      userCount: 0,
    };
    roles.push(newRole);
    return newRole;
  },

  async updateRole(id: string, data: Partial<Role>): Promise<Role> {
    await delay(150, 300);
    const idx = roles.findIndex((r) => r.id === id);
    if (idx === -1) throw new Error(`Role ${id} not found`);
    roles[idx] = { ...roles[idx], ...data };
    return roles[idx];
  },

  async deleteRoles(ids: string[]): Promise<void> {
    await delay(150, 300);
    roles = roles.filter((r) => !ids.includes(r.id));
  },

  // Permissions
  async getPermissions(params: PageRequest): Promise<PageResponse<Permission>> {
    await delay(80, 150);
    return paginate(mockPermissions, params.page, params.pageSize);
  },

  async getAllPermissions(): Promise<Permission[]> {
    await delay(80, 150);
    return mockPermissions;
  },

  // System info
  async getSystemInfo(): Promise<{
    version: string;
    buildDate: string;
    serverTime: string;
    uptimeHours: number;
    dbStatus: string;
    cacheStatus: string;
  }> {
    await delay(100, 200);
    return {
      version: '3.2.1',
      buildDate: '2024-06-01',
      serverTime: new Date().toISOString(),
      uptimeHours: 720,
      dbStatus: 'normal',
      cacheStatus: 'normal',
    };
  },
};
