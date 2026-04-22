export type UserOnlineStatus = 'online' | 'offline';
export type UserStatus = 'active' | 'inactive' | 'locked';
export type UserRole = 'admin' | 'operator' | 'viewer' | 'auditor';

export interface User {
  id: string;
  username: string;
  displayName: string;
  email: string;
  role: UserRole;
  status: UserStatus;
  phone?: string;
  carrier?: string;
  lastLoginTime?: string;
  createTime: string;
  updateTime?: string;
  department?: string;
  description?: string;
}

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
