export type UserOnlineStatus = 'online' | 'offline';
export type UserStatus = 'active' | 'disabled' | 'inactive' | 'locked';
export type UserRole = 'admin' | 'operator' | 'viewer' | 'auditor';
// 与后端 admin.UserSource 保持一致，字面值大小写敏感（DB CHECK 锁定）。
// 详见 omcgo/docs/prd/system/users.md §1.1 / §11.3。
export type UserSource = 'builtIn' | 'admin' | 'LDAP';

export interface User {
  id: string;
  username: string;
  displayName: string;
  email: string;
  role: UserRole;
  roles?: string[];
  roleIds?: string[];
  status: UserStatus;
  source?: UserSource;
  phone?: string;
  carrier?: string;
  /** ISO 8601 时间戳；登录时若 expireTime <= 现在则拒绝登录（见后端 service.Login）。 */
  expireTime?: string;
  lastLoginTime?: string;
  createTime: string;
  updateTime?: string;
  /** 创建人 / 更新人：后端返回 UUID，前端通过 useAllUsers 查 username 显示。 */
  createdBy?: string;
  updatedBy?: string;
  department?: string;
  description?: string;
}

/** 派生：是否为内置用户。用于 UI 操作权限判定（删除/禁用按钮 disable）。 */
export const isBuiltInUser = (u: Pick<User, 'source'>): boolean => u.source === 'builtIn';
/** 派生：是否为 LDAP 用户。重置密码按钮 disable。 */
export const isLdapUser = (u: Pick<User, 'source'>): boolean => u.source === 'LDAP';

export interface Role {
  id: string;
  roleName: string;
  roleCode: string;
  batchOperation: number;
  description: string;
  permissions: string[];
  apiPermissions?: ApiPermission[];
  deviceGroupIds?: string[];
  networkTypes?: string[];
  menuIds?: string[];
  userCount: number;
  builtIn: number;
  createUser?: string;
  updateUser?: string;
  createTime?: string;
  updateTime?: string;
}

export interface Group {
  id: string;
  groupName: string;
  description: string;
  userCount: number;
  roleCount: number;
  builtIn: number;
  updUser: string;
  updTime: string;
}

export interface Permission {
  id: string;
  permCode: string;
  permName: string;
  module: string;
  description: string;
}

export interface ApiEndpoint {
  id: string;
  path: string;
  method: string;
  name: string;
  apiGroup?: string;
  module: string;
  description: string;
  createdAt?: string;
  updatedAt?: string;
}

export interface ApiEndpointListParams {
  page?: number;
  pageSize?: number;
  path?: string;
  method?: string;
  apiGroup?: string;
  name?: string;
}

export interface ApiEndpointPayload {
  path: string;
  method: string;
  name?: string;
  description?: string;
  apiGroup?: string;
}

export interface SyncApiResult {
  created: number;
  updated: number;
  total: number;
}

export interface ApiPermission {
  path: string;
  method: string;
}

export type OperationResult = 'success' | 'failure' | 'partial';
export type OperationType =
  | 'create'
  | 'update'
  | 'delete'
  | 'query'
  | 'export'
  | 'import'
  | 'login'
  | 'logout'
  | 'execute'
  | 'deploy'
  | 'approve';

export interface OperationLog {
  id: string;
  operator: string;
  clientIp: string;
  module: string;
  operationType: OperationType;
  target: string;
  content: string;
  result: OperationResult;
  message: string;
  operationTime: string;
  // Optional fields for richer audit-log style backends
  logName?: string;
  detail?: string;
  reason?: string;
  startTime?: string;
  endTime?: string;
}

// ---- Menu types (admin RBAC) ----
export type MenuType = 'menu' | 'button' | 'link';
export type MenuStatus = 'active' | 'disabled';

export interface MenuItem {
  id: string;
  name: string;
  title: string;
  icon?: string;
  path?: string;
  component?: string;
  type: MenuType;
  parentId?: string;
  sortOrder: number;
  status: MenuStatus;
  visible: boolean;
  children?: MenuItem[];
  createdAt?: string;
  updatedAt?: string;
}

export interface MenuFilter {
  page?: number;
  pageSize?: number;
  type?: MenuType;
  status?: MenuStatus;
  parentId?: string;
}

export interface MenuListResponse {
  items: MenuItem[];
  total: number;
  page: number;
  pageSize: number;
}

export interface CreateMenuRequest {
  name: string;
  title: string;
  icon?: string;
  path?: string;
  component?: string;
  type: MenuType;
  parentId?: string;
  sortOrder?: number;
  status?: MenuStatus;
  visible?: boolean;
}

export interface UpdateMenuRequest {
  name?: string;
  title?: string;
  icon?: string;
  path?: string;
  component?: string;
  type?: MenuType;
  parentId?: string;
  sortOrder?: number;
  status?: MenuStatus;
  visible?: boolean;
}

export interface SetRoleMenusRequest {
  menuIds: string[];
}

export interface RoleWithMenus {
  id: string;
  name: string;
  description: string;
  menus: MenuItem[];
}
