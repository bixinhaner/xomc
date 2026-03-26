import type { User, Role, Permission, Group } from '@/types/system';
import type { PageRequest, PageResponse } from '@/types/pagination';
import { mockUsers, mockRoles, mockPermissions, mockGroups } from '../data/system';
import { delay, paginate, generateId } from '../utils';

let users = [...mockUsers];
let roles = [...mockRoles];
let groups = [...mockGroups];

export const systemService = {
  // Users
  async getUsers(
    params: { userName?: string } & PageRequest
  ): Promise<PageResponse<User>> {
    await delay(100, 200);
    let filtered = [...users];
    if (params.userName) {
      const kw = params.userName.toLowerCase();
      filtered = filtered.filter(
        (u) =>
          u.userName.toLowerCase().includes(kw) ||
          u.email.toLowerCase().includes(kw)
      );
    }
    return paginate(filtered, params.page, params.pageSize);
  },

  async getAllUsers(): Promise<User[]> {
    await delay(80, 150);
    return users;
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
    // Cannot delete built-in users
    users = users.filter((u) => !ids.includes(u.id) || u.builtIn === 1);
  },

  async resetPassword(id: string, _newPassword: string): Promise<void> {
    await delay(300, 600);
    void id;
    void _newPassword;
  },

  async lockUser(id: string): Promise<void> {
    await delay(150, 300);
    const idx = users.findIndex((u) => u.id === id);
    if (idx !== -1) users[idx] = { ...users[idx], lockStatus: 1 };
  },

  async unlockUser(id: string): Promise<void> {
    await delay(150, 300);
    const idx = users.findIndex((u) => u.id === id);
    if (idx !== -1) users[idx] = { ...users[idx], lockStatus: 0 };
  },

  async forceLogout(ids: string[]): Promise<void> {
    await delay(150, 300);
    ids.forEach((id) => {
      const idx = users.findIndex((u) => u.id === id);
      if (idx !== -1) users[idx] = { ...users[idx], onlineStatus: 'offline' };
    });
  },

  async moveUsersToGroup(userIds: string[], groupId: string): Promise<void> {
    await delay(150, 300);
    const group = groups.find((g) => g.id === groupId);
    if (group) {
      userIds.forEach((id) => {
        const idx = users.findIndex((u) => u.id === id);
        if (idx !== -1) {
          if (!users[idx].groupNames.includes(group.groupName)) {
            users[idx].groupNames.push(group.groupName);
          }
        }
      });
    }
  },

  async copyUser(id: string): Promise<User> {
    await delay(200, 400);
    const user = users.find((u) => u.id === id);
    if (!user) throw new Error(`User ${id} not found`);
    const newUser: User = {
      ...user,
      id: generateId('user'),
      userName: `${user.userName}_copy`,
      createTime: new Date().toISOString(),
      lastLoginTime: new Date().toISOString(),
    };
    users.push(newUser);
    return newUser;
  },

  // Roles
  async getRoles(params: PageRequest & { roleName?: string }): Promise<PageResponse<Role>> {
    await delay(80, 150);
    let filtered = [...roles];
    if (params.roleName) {
      const kw = params.roleName.toLowerCase();
      filtered = filtered.filter((r) => r.roleName.toLowerCase().includes(kw));
    }
    return paginate(filtered, params.page, params.pageSize);
  },

  async getAllRoles(): Promise<Role[]> {
    await delay(80, 150);
    return roles;
  },

  async getRoleById(id: string): Promise<Role | null> {
    await delay(80, 150);
    return roles.find((r) => r.id === id) ?? null;
  },

  async createRole(data: Omit<Role, 'id' | 'userCount' | 'updUser' | 'updTime'>): Promise<Role> {
    await delay(200, 400);
    const newRole: Role = {
      ...data,
      id: generateId('role'),
      userCount: 0,
      updUser: 'admin',
      updTime: new Date().toISOString(),
    };
    roles.push(newRole);
    return newRole;
  },

  async updateRole(id: string, data: Partial<Role>): Promise<Role> {
    await delay(150, 300);
    const idx = roles.findIndex((r) => r.id === id);
    if (idx === -1) throw new Error(`Role ${id} not found`);
    roles[idx] = { ...roles[idx], ...data, updTime: new Date().toISOString() };
    return roles[idx];
  },

  async deleteRoles(ids: string[]): Promise<void> {
    await delay(150, 300);
    // Cannot delete built-in roles
    roles = roles.filter((r) => !ids.includes(r.id) || r.builtIn === 1 || r.builtIn === 2);
  },

  // Groups
  async getGroups(params: PageRequest & { groupName?: string }): Promise<PageResponse<Group>> {
    await delay(80, 150);
    let filtered = [...groups];
    if (params.groupName) {
      const kw = params.groupName.toLowerCase();
      filtered = filtered.filter((g) => g.groupName.toLowerCase().includes(kw));
    }
    return paginate(filtered, params.page, params.pageSize);
  },

  async getAllGroups(): Promise<Group[]> {
    await delay(80, 150);
    return groups;
  },

  async getGroupById(id: string): Promise<Group | null> {
    await delay(80, 150);
    return groups.find((g) => g.id === id) ?? null;
  },

  async createGroup(data: Omit<Group, 'id' | 'userCount' | 'roleCount' | 'updUser' | 'updTime'>): Promise<Group> {
    await delay(200, 400);
    const newGroup: Group = {
      ...data,
      id: generateId('group'),
      userCount: 0,
      roleCount: 0,
      updUser: 'admin',
      updTime: new Date().toISOString(),
    };
    groups.push(newGroup);
    return newGroup;
  },

  async updateGroup(id: string, data: Partial<Group>): Promise<Group> {
    await delay(150, 300);
    const idx = groups.findIndex((g) => g.id === id);
    if (idx === -1) throw new Error(`Group ${id} not found`);
    groups[idx] = { ...groups[idx], ...data, updTime: new Date().toISOString() };
    return groups[idx];
  },

  async deleteGroups(ids: string[]): Promise<void> {
    await delay(150, 300);
    // Cannot delete built-in groups (builtIn = 1 or 2)
    const toDelete = ids.filter((id) => {
      const group = groups.find((g) => g.id === id);
      return group && group.builtIn !== 1 && group.builtIn !== 2;
    });
    groups = groups.filter((g) => !toDelete.includes(g.id));
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
