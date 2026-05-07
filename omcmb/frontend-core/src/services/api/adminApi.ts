import http from '../http';
import type {
  User,
  UserRole,
  UserStatus,
  UserSource,
  Role,
  Permission,
  OperationLog,
  OperationType,
  Group,
  ApiEndpoint,
  ApiPermission,
  ApiEndpointListParams,
  ApiEndpointPayload,
  SyncApiResult,
  MenuItem,
  MenuFilter,
  MenuListResponse,
  CreateMenuRequest,
  UpdateMenuRequest,
  SetRoleMenusRequest,
  RoleWithMenus,
  SysConfigItem,
  SysConfigValueType,
  BatchUpdateSysConfigPayload,
} from '../../types/system';
import type { PageRequest, PageResponse } from '../../types/pagination';

// ---- Dictionary types ----
export interface Dictionary {
  id: number;
  name: string;
  type: string;
  status: boolean;
  desc: string;
  sysDictionaryDetails?: DictionaryDetail[];
  createdAt?: string;
  updatedAt?: string;
}

export interface DictionaryDetail {
  id: number;
  label: string;
  value: string;
  extend: string;
  status: boolean;
  sort: number;
  sysDictionaryId: number;
  // PRD docs/prd/system/data-dictionary.md §10 v0.2 新增字段
  parentId?: number | null; // null/undefined = 顶层项
  level: number;            // 0 = 顶层 / 1 = 一级子 / 2 = 二级子（最大深度 3）
  createdAt?: string;
  updatedAt?: string;
}

export interface CreateDictionaryPayload {
  name: string;
  type: string;
  status?: boolean;
  desc?: string;
}

export interface UpdateDictionaryPayload {
  id: number;
  name?: string;
  type?: string;
  status?: boolean;
  desc?: string;
}

export interface CreateDictionaryDetailPayload {
  label: string;
  value: string;
  extend?: string;
  status?: boolean;
  sort?: number;
  sysDictionaryId: number;
  // PRD §10：可选父明细 ID。nil/undefined = 顶层项（level=0）。
  parentId?: number | null;
}

export interface UpdateDictionaryDetailPayload {
  id: number;
  label?: string;
  value?: string;
  extend?: string;
  status?: boolean;
  sort?: number;
  sysDictionaryId?: number;
  // PRD §10：携带 parent_id 时即触发换父；undefined = 不变更，null = 设为顶层。
  parentId?: number | null;
}

export interface DictDetailListParams {
  sysDictionaryId?: number;
  label?: string;
  page?: number;
  pageSize?: number;
}

export interface DictListResponse {
  list: Dictionary[];
  total: number;
}

export interface DictDetailListResponse {
  list: DictionaryDetail[];
  total: number;
}

interface BackendDictionary {
  id: number;
  name: string;
  type: string;
  status: boolean;
  desc: string;
  created_at?: string;
  updated_at?: string;
}

interface BackendDictionaryDetail {
  id: number;
  label: string;
  value: string;
  extend: string;
  status: boolean;
  sort: number;
  sysDictionaryId: number;
  // PRD §10 v0.2：parent_id (snake) + level
  parent_id?: number | null;
  level?: number;
  created_at?: string;
  updated_at?: string;
}

function mapBackendDictionary(b: BackendDictionary): Dictionary {
  return {
    id: b.id,
    name: b.name,
    type: b.type,
    status: b.status,
    desc: b.desc || '',
    createdAt: b.created_at,
    updatedAt: b.updated_at,
  };
}

function mapBackendDictionaryDetail(b: BackendDictionaryDetail): DictionaryDetail {
  return {
    id: b.id,
    label: b.label,
    value: b.value,
    extend: b.extend || '',
    status: b.status,
    sort: b.sort || 0,
    sysDictionaryId: b.sysDictionaryId,
    parentId: b.parent_id ?? null,
    level: b.level ?? 0,
    createdAt: b.created_at,
    updatedAt: b.updated_at,
  };
}

// 把前端 camelCase payload 转成后端 JSON：parentId → parent_id（仅当字段存在时携带）。
// undefined 不发，null 发（语义=切顶层 / 清父）。
function toBackendDictionaryDetailBody(
  p: Partial<CreateDictionaryDetailPayload & UpdateDictionaryDetailPayload>,
): Record<string, unknown> {
  const body: Record<string, unknown> = {};
  if (p.id !== undefined) body.id = p.id;
  if (p.label !== undefined) body.label = p.label;
  if (p.value !== undefined) body.value = p.value;
  if (p.extend !== undefined) body.extend = p.extend;
  if (p.status !== undefined) body.status = p.status;
  if (p.sort !== undefined) body.sort = p.sort;
  if (p.sysDictionaryId !== undefined) body.sysDictionaryId = p.sysDictionaryId;
  if (Object.prototype.hasOwnProperty.call(p, 'parentId')) {
    body.parent_id = p.parentId; // 含 null（切顶层）
  }
  return body;
}
// ---- End Dictionary types ----

// 用户批量导入响应（与后端 admin.ImportUserResult 对齐）。
export interface ImportUserError {
  row: number;
  message: string;
}
export interface ImportUserResult {
  created: number;
  failed: number;
  errors?: ImportUserError[];
}

// Backend user model - matches Go User struct JSON tags (snake_case).
// Axios 不做 body/response 转换，必须按后端原样字段名访问。
interface BackendUser {
  id: string;
  username: string;
  display_name?: string;
  email?: string;
  phone?: string;
  description?: string;
  // carrier?: string; — v1.0 删除（详见 omcgo/docs/prd/system/users.md §11.11）
  status: string; // active, disabled
  source?: string; // builtIn / admin / LDAP
  roles?: BackendRole[];
  failed_login_attempts?: number;
  locked_until?: string;
  last_failed_login_at?: string;
  last_login_at?: string;
  expire_at?: string;
  created_by?: string;
  updated_by?: string;
  // 派生字段：后端 ListUsers / GetUser 反查 users 表注入；前端直接读，
  // 取代历史的「拉全量 /admin/users 建 ID→username 映射」做法。
  creator_username?: string;
  updater_username?: string;
  created_at: string;
  updated_at: string;
}

// Backend role model - matches Go Role struct JSON tags (snake_case).
// Axios 不做 body/response 转换，必须按后端原样字段名访问。
interface BackendRole {
  id: string;
  name: string;
  code?: string; // v0.6: 程序化引用编码（可选，全局唯一）
  description: string;
  is_system: boolean;
  status?: string; // active, disabled - may be empty
  created_at: string;
  updated_at: string;
  // Fields that may or may not be included in list response
  permissions?: BackendPermission[];
  device_group_ids?: string[];
  menu_ids?: string[];
  menus?: BackendMenu[];
  user_count?: number;
  created_by?: string;
  updated_by?: string;
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

// Backend api_endpoint shape — matches Go ApiEndpointDB JSON tags (snake_case).
// http.ts 不转 body 字段名，必须经 mapBackendApiEndpoint 转成前端 ApiEndpoint。
interface BackendApiEndpoint {
  id: string;
  path: string;
  method: string;
  name: string;
  description: string;
  api_group: string;
  is_auto: boolean;
  created_at: string;
  updated_at: string;
}

// Backend sys_configs row — DDL omcgo/migrations/000009_sys_admin.sql:113。
interface BackendSysConfig {
  id: string;
  category: string;
  key: string;
  value: string;
  value_type: string;
  desc?: string;
  is_public?: boolean;
  created_at?: string;
  updated_at?: string;
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
  const validStatuses: readonly UserStatus[] = ['active', 'disabled', 'inactive', 'locked'];
  const status: UserStatus = (validStatuses as readonly string[]).includes(bu.status)
    ? (bu.status as UserStatus)
    : 'disabled';
  const roleList = bu.roles ?? [];
  const roleNames = roleList.map((r) => r.name).filter(Boolean);
  const roleIds = roleList.map((r) => r.id).filter(Boolean);
  const role: UserRole = (roleNames[0] ?? 'viewer') as UserRole;

  const validSources: readonly UserSource[] = ['builtIn', 'admin', 'LDAP'];
  const source: UserSource | undefined = bu.source && (validSources as readonly string[]).includes(bu.source)
    ? (bu.source as UserSource)
    : undefined;

  return {
    id: bu.id,
    username: bu.username,
    displayName: bu.display_name || bu.username,
    email: bu.email || '',
    phone: bu.phone,
    description: bu.description,
    role,
    roles: roleNames,
    roleIds,
    status,
    source,
    // carrier: bu.carrier — v1.0 删除
    expireTime: bu.expire_at || undefined,
    lastLoginTime: bu.last_login_at || undefined,
    createTime: bu.created_at,
    updateTime: bu.updated_at,
    createdBy: bu.created_by,
    updatedBy: bu.updated_by,
    creatorUsername: bu.creator_username,
    updaterUsername: bu.updater_username,
  };
}

function mapFrontendUser(user: Partial<User>): Record<string, unknown> {
  const payload: Record<string, unknown> = {};
  if (user.username !== undefined) payload.username = user.username;
  if (user.email !== undefined) payload.email = user.email;
  if (user.phone !== undefined) payload.phone = user.phone;
  if (user.description !== undefined) payload.description = user.description;
  if (user.expireTime !== undefined) payload.expire_at = user.expireTime;
  if (user.displayName !== undefined) payload.display_name = user.displayName;
  if (user.status !== undefined) payload.status = user.status;
  // role_ids === null 表示"清空角色"，undefined 表示"不变更"。
  if (user.roleIds !== undefined) payload.role_ids = user.roleIds;
  return payload;
}

function mapBackendRole(br: BackendRole): Role {
  // v0.7 修复：menu permissions 回显
  //
  // 前端 PERMISSION_MODULES 的叶子 key 格式是 `module.sub.op`（点分 3 段，无 action 后缀）；
  // 前端 permissionsToArray 提交时硬编码 action='read'，后端入库 (resource="module.sub.op", action="read")。
  // 此处回读只取 resource，去掉 action 维度，使 round-trip 与 ALL_PERMISSION_LEAF_KEYS 对齐：
  //   写入：["device.list.query"] → DB ("device.list.query", "read")
  //   读取：DB → ["device.list.query"]（不再带 ":read" 后缀）
  // 用 Set 去重避免历史数据中同 resource 多 action 行产生重复 key。
  const permissions = Array.from(
    new Set((br.permissions || []).map((p) => p.resource))
  );

  // Ensure required fields have values
  const roleName = br.name || '';

  return {
    id: br.id,
    roleName,
    // v0.6：roleCode 优先用后端 code（程序化引用编码），未配 code 时用 name 兜底
    roleCode: br.code || roleName,
    batchOperation: 0, // Backend doesn't have this field
    description: br.description || '',
    permissions,
    deviceGroupIds: br.device_group_ids ?? [],
    userCount: br.user_count ?? 0,
    builtIn: br.is_system ? 1 : 0,
    createUser: br.created_by,
    updateUser: br.updated_by,
    createTime: br.created_at,
    updateTime: br.updated_at,
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
    logName: `${ba.action} ${ba.resource}`.trim(),
    detail: ba.details ? JSON.stringify(ba.details) : '',
    reason: '',
    startTime: ba.createdAt,
    endTime: ba.createdAt,
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

// 后端 CreateApiEndpointRequest / UpdateApiEndpointRequest 的 ApiGroup json tag
// 是 "api_group"。http.ts 不做 body 字段转换，需在此显式映射前端 camelCase → 后端 snake_case。
function toBackendApiEndpointBody(payload: Partial<ApiEndpointPayload>): Record<string, unknown> {
  const body: Record<string, unknown> = {};
  if (payload.path !== undefined) body.path = payload.path;
  if (payload.method !== undefined) body.method = payload.method;
  if (payload.name !== undefined) body.name = payload.name;
  if (payload.description !== undefined) body.description = payload.description;
  if (payload.apiGroup !== undefined) body.api_group = payload.apiGroup;
  return body;
}

// 后端 ApiEndpointDB JSON 字段全 snake_case，前端 ApiEndpoint 接口走 camelCase。
// http.ts 仅转 query 参数，不动 body，所以列表/创建/更新的响应必须经此映射，否则
// record.apiGroup 永远 undefined → 表格列空、编辑下拉不回显。
function mapBackendApiEndpoint(b: BackendApiEndpoint): ApiEndpoint {
  return {
    id: b.id,
    path: b.path,
    method: b.method,
    name: b.name ?? '',
    description: b.description ?? '',
    apiGroup: b.api_group ?? '',
    // module 字段类型上是 string（早期遗留），为兼容现有列定义保留 path 兜底。
    module: b.api_group ?? '',
    createdAt: b.created_at,
    updatedAt: b.updated_at,
  };
}

function mapBackendSysConfig(b: BackendSysConfig): SysConfigItem {
  const allowed: SysConfigValueType[] = ['string', 'int', 'float', 'bool', 'json'];
  const vt = (allowed as readonly string[]).includes(b.value_type)
    ? (b.value_type as SysConfigValueType)
    : 'string';
  return {
    id: b.id,
    category: b.category,
    key: b.key,
    value: b.value,
    valueType: vt,
    description: b.desc,
    isPublic: b.is_public,
    createdAt: b.created_at,
    updatedAt: b.updated_at,
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

  // GET /admin/users 是分页接口（返回 {items, total, page, page_size}），不是裸数组。
  // 给一个足够大的 page_size 一次拉全；当用户数 > 1000 时再考虑分页拼装。
  async getAllUsers(): Promise<User[]> {
    const { data } = await http.get<BackendListResponse<BackendUser>>(
      '/admin/users',
      { params: { page: 1, page_size: 1000 } }
    );
    return (data?.items ?? []).map(mapBackendUser);
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
    data: Omit<User, 'id' | 'createTime' | 'lastLoginTime'> & {
      password: string;
      roleIds?: string[];
    }
  ): Promise<User> {
    const { data: bu } = await http.post<BackendUser>('/admin/users', {
      username: data.username,
      password: data.password,
      email: data.email || undefined,
      phone: data.phone || undefined,
      description: data.description || undefined,
      expire_at: data.expireTime || undefined,
      display_name: data.displayName || data.username,
      status: data.status,
      role_ids: data.roleIds && data.roleIds.length > 0 ? data.roleIds : undefined,
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
    // 后端字段名 new_password（snake_case），axios 不转 body 字段。
    await http.post(`/admin/users/${id}/reset-password`, { new_password: newPassword });
  },

  async lockUser(id: string): Promise<void> {
    await http.post(`/admin/users/${id}/lock`);
  },

  async unlockUser(id: string): Promise<void> {
    await http.post(`/admin/users/${id}/unlock`);
  },

  async forceLogout(ids: string[]): Promise<void> {
    // 后端字段名 user_ids（snake_case），axios 不转 body 字段。
    await http.post('/admin/users/force-logout', { user_ids: ids });
  },

  async moveUsersToGroup(userIds: string[], groupId: string): Promise<void> {
    await http.post('/admin/users/move-group', { userIds, groupId });
  },

  async copyUser(id: string): Promise<{ user: User; tempPassword: string }> {
    // 后端返回 { user: BackendUser, temp_password: string }
    const { data } = await http.post<{ user: BackendUser; temp_password: string }>(
      `/admin/users/${id}/copy`,
    );
    return {
      user: mapBackendUser(data.user),
      tempPassword: data.temp_password,
    };
  },

  // 批量分配角色：与后端 admin.BatchAssignRoles (POST /admin/users/assign-roles) 对齐。
  async batchAssignRoles(userIds: string[], roleIds: string[]): Promise<void> {
    await http.post('/admin/users/assign-roles', {
      user_ids: userIds,
      role_ids: roleIds,
    });
  },

  // 用户批量导入：与后端 admin.ImportUsers (POST /admin/users/import) 对齐。
  async importUsers(file: File): Promise<ImportUserResult> {
    const formData = new FormData();
    formData.append('file', file);
    const { data } = await http.post<ImportUserResult>('/admin/users/import', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
    return data;
  },

  // 用户导入模板下载：与后端 admin.DownloadImportTemplate (GET /admin/users/import/template) 对齐。
  async downloadImportTemplate(): Promise<Blob> {
    const { data } = await http.get<Blob>('/admin/users/import/template', {
      responseType: 'blob',
    });
    return data;
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
    // PRD §11.10 v0.9：下拉用全量端点，不复用分页端点（/admin/roles 返回 PageResponse 而非数组）。
    const { data } = await http.get<BackendRole[]>('/admin/roles/all');
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

  // v0.6（roles.md §7 P2 #9）：一键复制角色，副本名 base_copy / base_copy_2 ...
  async copyRole(id: string): Promise<Role> {
    const { data } = await http.post<BackendRole>(`/admin/roles/${id}/copy`);
    return mapBackendRole(data);
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
      builtIn: role.is_system ? 1 : 0,
      userCount: role.user_count || 0,
      roleCount: 0, // 角色没有"角色数"概念
      updUser: role.updated_by || '',
      updTime: role.updated_at || '',
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

  // ---- Dictionary management ----
  // 历史 API 形态为 {code, data, msg}（旧 gin-vue-admin 风格）；后端 v0.6 起统一切到
  // {ret, msg, data} 信封（参 omgo/docs/architecture/api-envelope.md），http.ts 拦截器
  // 自动剥掉外层信封 → 业务侧 `const { data } = await http.get(...)` 拿到的就是裸业务体。
  // 因此这里所有读取直接走 `data.X`，不再 `data.data.X`。
  async getDictionaryList(): Promise<DictListResponse> {
    const { data } = await http.get<{ list: BackendDictionary[]; total: number }>('/admin/sysDictionary/getSysDictionaryList');
    const list = (data.list || []).map(mapBackendDictionary);
    return { list, total: data.total ?? list.length };
  },

  async createDictionary(req: CreateDictionaryPayload): Promise<Dictionary> {
    const { data } = await http.post<BackendDictionary>('/admin/sysDictionary/createSysDictionary', req);
    return mapBackendDictionary(data);
  },

  async updateDictionary(req: UpdateDictionaryPayload): Promise<Dictionary> {
    const { data } = await http.put<BackendDictionary>('/admin/sysDictionary/updateSysDictionary', req);
    return mapBackendDictionary(data);
  },

  async deleteDictionary(id: number): Promise<void> {
    await http.delete('/admin/sysDictionary/deleteSysDictionary', { params: { id } });
  },

  async getDictionaryDetailList(params: DictDetailListParams): Promise<DictDetailListResponse> {
    const query: Record<string, unknown> = {
      page: params.page ?? 1,
      page_size: params.pageSize ?? 100,
    };
    if (params.sysDictionaryId !== undefined) query.sysDictionaryId = params.sysDictionaryId;
    if (params.label) query.label = params.label;
    const { data } = await http.get<{ list: BackendDictionaryDetail[]; total: number }>('/admin/sysDictionaryDetail/getSysDictionaryDetailList', { params: query });
    const list = (data.list || []).map(mapBackendDictionaryDetail);
    return { list, total: data.total ?? list.length };
  },

  async createDictionaryDetail(req: CreateDictionaryDetailPayload): Promise<DictionaryDetail> {
    // PRD §10：parentId(camel) → parent_id(snake)。toBackendDictionaryDetailBody 处理。
    const { data } = await http.post<BackendDictionaryDetail>(
      '/admin/sysDictionaryDetail/createSysDictionaryDetail',
      toBackendDictionaryDetailBody(req),
    );
    return mapBackendDictionaryDetail(data);
  },

  async updateDictionaryDetail(req: UpdateDictionaryDetailPayload): Promise<DictionaryDetail> {
    const { data } = await http.put<BackendDictionaryDetail>(
      '/admin/sysDictionaryDetail/updateSysDictionaryDetail',
      toBackendDictionaryDetailBody(req),
    );
    return mapBackendDictionaryDetail(data);
  },

  async deleteDictionaryDetail(id: number): Promise<void> {
    await http.delete('/admin/sysDictionaryDetail/deleteSysDictionaryDetail', { params: { id } });
  },

  async findDictionaryByType(type: string): Promise<Dictionary | null> {
    const { data } = await http.get<BackendDictionary & { sysDictionaryDetails?: BackendDictionaryDetail[] }>('/admin/sysDictionary/findSysDictionary', { params: { type } });
    const dict = mapBackendDictionary(data);
    dict.sysDictionaryDetails = (data.sysDictionaryDetails || []).map(mapBackendDictionaryDetail);
    return dict;
  },

  // API 权限
  // 后端 ListApiEndpoints 走分页（model.ListResponse），裸响应是 {items, total, page, page_size}；
  // 这里统一用全量分页拼装函数 getApiEndpoints 拿全集即可，不再读 data.data。
  async listApiEndpoints(): Promise<ApiEndpoint[]> {
    const { items } = await this.getApiEndpoints({ page: 1, pageSize: 1000 });
    return items;
  },

  async getRoleApiPermissions(roleId: string): Promise<ApiPermission[]> {
    // 注意：后端真实路径返回 {endpoint_ids: string[]}（参 internal/admin/api_endpoint_handler.go:160），
    // 该签名 ApiPermission[] 与后端不一致；本函数无活跃业务消费者。这里仅修信封读取，
    // 后续请用 apiPermissionApi.getRolePermissions（返回 string[] endpoint id 列表）。
    const { data } = await http.get<{ endpoint_ids?: string[] }>(`/admin/roles/${roleId}/api-permissions`);
    return ((data.endpoint_ids || []) as unknown) as ApiPermission[];
  },

  async setRoleApiPermissions(roleId: string, permissions: ApiPermission[]): Promise<void> {
    await http.put(`/admin/roles/${roleId}/api-permissions`, { permissions });
  },

  // API 管理（CRUD）
  async getApiEndpoints(params: ApiEndpointListParams): Promise<PageResponse<ApiEndpoint>> {
    const { data } = await http.get<BackendListResponse<BackendApiEndpoint>>('/admin/api-endpoints', { params });
    return {
      items: (data.items || []).map(mapBackendApiEndpoint),
      total: data.total,
      page: data.page,
      pageSize: data.pageSize,
    };
  },

  async createApiEndpoint(payload: ApiEndpointPayload): Promise<ApiEndpoint> {
    // 请求 body 字段：apiGroup → api_group；响应 body 字段：api_group → apiGroup。
    const { data } = await http.post<BackendApiEndpoint>('/admin/api-endpoints', toBackendApiEndpointBody(payload));
    return mapBackendApiEndpoint(data);
  },

  async updateApiEndpoint(id: string, payload: Partial<ApiEndpointPayload>): Promise<ApiEndpoint> {
    const { data } = await http.put<BackendApiEndpoint>(`/admin/api-endpoints/${id}`, toBackendApiEndpointBody(payload));
    return mapBackendApiEndpoint(data);
  },

  async deleteApiEndpoint(id: string): Promise<void> {
    await http.delete(`/admin/api-endpoints/${id}`);
  },

  async batchDeleteApiEndpoints(ids: string[]): Promise<void> {
    await http.delete('/admin/api-endpoints/batch', { data: { ids } });
  },

  async getApiGroups(): Promise<string[]> {
    const { data } = await http.get<{ data: string[] }>('/admin/api-endpoints/groups');
    return data.data || [];
  },

  async syncApiEndpoints(): Promise<SyncApiResult> {
    // 后端 SyncApiEndpoints handler 直接 c.JSON(StatusOK, result)，无 {data:...} 包裹。
    // 之前读 data.data → undefined → 弹窗永远显示 0 / 0 / 0。
    const { data } = await http.post<SyncApiResult>('/admin/api-endpoints/sync');
    return data;
  },

  // Role device groups with network types
  async getRoleDeviceGroups(roleId: string): Promise<{ deviceGroupIds: string[]; networkTypes: string[] }> {
    const { data } = await http.get<{ device_group_ids?: string[]; network_types?: string[] }>(`/admin/roles/${roleId}/device-groups`);
    return {
      deviceGroupIds: data.device_group_ids || [],
      networkTypes: data.network_types || [],
    };
  },

  async setRoleDeviceGroups(roleId: string, payload: { deviceGroupIds: string[]; networkTypes: string[] }): Promise<void> {
    await http.put(`/admin/roles/${roleId}/device-groups`, {
      device_group_ids: payload.deviceGroupIds,
      network_types: payload.networkTypes,
    });
  },

  // Change password (current user)
  async changePassword(data: { old_password: string; new_password: string }): Promise<void> {
    await http.post('/auth/change-password', data);
  },

  // ---- System Config (KV by category, batch upsert) ----
  // 参 omgo/docs/prd/system/config.md §5。前端 system/config 页面 7 Tab 用：
  // 进入页面/切 Tab 调 getSysConfigsByCategory；保存按钮调 batchUpdateSysConfigs。
  async getSysConfigsByCategory(category: string): Promise<SysConfigItem[]> {
    const { data } = await http.get<BackendSysConfig[]>('/admin/sysConfig', { params: { category } });
    return (Array.isArray(data) ? data : []).map(mapBackendSysConfig);
  },

  async batchUpdateSysConfigs(payload: BatchUpdateSysConfigPayload): Promise<number> {
    const { data } = await http.post<{ updated?: number }>('/admin/sysConfig/batch', payload);
    return data?.updated ?? 0;
  },
};
