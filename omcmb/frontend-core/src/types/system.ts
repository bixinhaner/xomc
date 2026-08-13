export type UserOnlineStatus = 'online' | 'offline';
export type UserStatus = 'active' | 'disabled' | 'inactive' | 'locked';
export type UserRole = 'admin' | 'operator' | 'viewer' | 'auditor';
// 与后端 admin.UserSource 保持一致，字面值大小写敏感（DB CHECK 锁定）。
// 详见 omcgo/docs/prd/system/users.md §1.1 / §11.3。
export type UserSource = 'builtIn' | 'admin' | 'LDAP';

export interface UserListParams {
  userName?: string;
  roleId?: string;
  status?: UserStatus;
}

export interface User {
  id: string;
  username: string;
  displayName: string;
  email: string;
  role: UserRole;
  /** 派生：是否为超级管理员（source === 'builtIn'，对齐后端 user.IsSuperAdmin()）。
   *  T-0098-P4 用于产品中心治理菜单 / 路由守卫；admin/operator/viewer 一律不可见。 */
  isSuperAdmin?: boolean;
  roles?: string[];
  roleIds?: string[];
  status: UserStatus;
  source?: UserSource;
  phone?: string;
  // carrier?: string; — v1.0 删除（后端 users.carrier 已移除，详见 omcgo/docs/prd/system/users.md §11.11）
  /** ISO 8601 时间戳；登录时若 expireTime <= 现在则拒绝登录（见后端 service.Login）。 */
  expireTime?: string;
  lastLoginTime?: string;
  createTime: string;
  updateTime?: string;
  /** 创建人 / 更新人 ID：后端 UUID，可能为空（seed 写入 / 内置）。 */
  createdBy?: string;
  updatedBy?: string;
  /** 创建人 / 更新人 username：由后端 ListUsers / GetUser 反查 users 表注入；
   *  当 ID 为空或对应记录已删除时为空字符串，前端按空值渲染"内置"。 */
  creatorUsername?: string;
  updaterUsername?: string;
  department?: string;
  description?: string;
  /** 头像 URL（前端 UserDropdown 显示用，optional；后端未存储 → undefined 走 fallback initial） */
  avatar?: string;
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

// ---- System Config (sys_configs KV table)
// 参 omgo/docs/prd/system/config.md。后端 7 个 category：
// basic / security / device / notify / storage / omc / northbound。
export type SysConfigValueType = 'string' | 'int' | 'float' | 'bool' | 'json';

export interface SysConfigItem {
  id: string;
  category: string;
  key: string;
  value: string;
  valueType?: SysConfigValueType;
  description?: string;
  isPublic?: boolean;
  /** 后端标记该配置为 write-only 敏感值；value 始终为空。 */
  isSecret?: boolean;
  /** 仅对敏感值有意义：服务端是否已保存非空值。 */
  isConfigured?: boolean;
  createdAt?: string;
  updatedAt?: string;
}

export interface BatchUpdateSysConfigItem {
  key: string;
  value: string;
  // 后端 sys_configs.value_type 列允许 'string'|'int'|'float'|'bool'|'json'；
  // 缺省按 'string' 入库（仅新增时用，已存在的行不会改 value_type）。
  value_type?: SysConfigValueType;
}

export interface BatchUpdateSysConfigPayload {
  category: string;
  items: BatchUpdateSysConfigItem[];
}

export type ConfigApplyStatus = 'pending' | 'applying' | 'applied' | 'failed';
export type ConfigApplySuccessScope = 'runtime_applied' | 'event_delivered';

export interface ConfigApplyTarget {
  target: string;
  status: ConfigApplyStatus;
  /** A successful worker result may prove runtime application or only reliable event delivery. */
  successScope: ConfigApplySuccessScope;
  attempts: number;
  appliedAt?: string;
  lastError?: string;
}

export interface ConfigApplyBatch {
  id: string;
  category: string;
  configVersion: number;
  status: ConfigApplyStatus;
  createdAt: string;
  updatedAt: string;
  targets: ConfigApplyTarget[];
}

export interface BatchUpdateSysConfigResult {
  updated: number;
  batch: ConfigApplyBatch;
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

export interface NorthboundAPIInvocationLog {
  id: string;
  apiKey: string;
  name: string;
  method: string;
  path: string;
  requestParams: string;
  responseBody: string;
  statusCode: number;
  status: string;
  createUser: string;
  ipAddress: string;
  durationMs: number;
  createdAt: string;
  updatedAt?: string;
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
