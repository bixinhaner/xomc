import http from '../http';
import { preparePasswordPayload } from '../crypto/passwordCipher';
import type {
  User,
  UserListParams,
  UserRole,
  UserStatus,
  UserSource,
  Role,
  Permission,
  OperationLog,
  NorthboundAPIInvocationLog,
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
  BatchUpdateSysConfigResult,
  ConfigApplyBatch,
  ConfigApplyStatus,
} from '../../types/system';
import type { PageRequest, PageResponse } from '../../types/pagination';

// ---- Dictionary types ----
export interface Dictionary {
  id: number;
  name: string;
  /** i18n JSONB (migration 000003)。Axios camel 转换后字段名,前端 useI18nText 兼容两种形态。 */
  nameI18n?: Record<string, string>;
  descriptionI18n?: Record<string, string>;
  type: string;
  status: boolean;
  desc: string;
  sysDictionaryDetails?: DictionaryDetail[];
  createdAt?: string;
  updatedAt?: string;
  // T-0182 数据源绑定。三字段全 null/undefined = 手工字典；
  // 三字段全 string 非空 = 托管字典(后端同步 auto 项)。
  sourceTable?: string | null;
  sourceLabelField?: string | null;
  sourceValueField?: string | null;
  lastRefreshAt?: string | null;
  lastRefreshStatus?: 'ok' | 'failed' | 'running' | 'timeout' | null;
  lastRefreshError?: string | null;
  lastRefreshCount?: number | null;
}

// 字典项来源 — T-0182
export type DictionaryDetailOrigin = 'manual' | 'auto';

export interface DictionaryDetail {
  id: number;
  label: string;
  /** i18n JSONB (migration 000003)。 */
  labelI18n?: Record<string, string>;
  value: string;
  extend: string;
  status: boolean;
  sort: number;
  sysDictionaryId: number;
  // PRD docs/prd/system/data-dictionary.md §10 v0.2 新增字段
  parentId?: number | null; // null/undefined = 顶层项
  level: number;            // 0 = 顶层 / 1 = 一级子 / 2 = 二级子（最大深度 3）
  // T-0182:manual=手工录入 / auto=数据源同步;UI 据此渲染来源 Tag + 禁用编辑/删除。
  origin: DictionaryDetailOrigin;
  createdAt?: string;
  updatedAt?: string;
}

export interface CreateDictionaryPayload {
  name: string;
  type: string;
  status?: boolean;
  desc?: string;
  // T-0182:三字段同时空 = 手工字典;同时非空 = 托管字典。部分填写后端 400。
  sourceTable?: string;
  sourceLabelField?: string;
  sourceValueField?: string;
}

export interface UpdateDictionaryPayload {
  id: number;
  name?: string;
  type?: string;
  status?: boolean;
  desc?: string;
  // T-0182:
  //   - 三字段都不传 → 不动数据源绑定
  //   - 三字段都传非空 → 绑定/切换
  //   - 三字段都传空字符串 → 解绑(删 auto 项,保留 manual)
  sourceTable?: string | null;
  sourceLabelField?: string | null;
  sourceValueField?: string | null;
}

// T-0182 数据源白名单 ListSources 返回结构。
export interface DictionarySourceField {
  column: string;
  display: string;
  type: string;
}
export interface DictionarySource {
  table: string;
  display_name: string;
  fields: DictionarySourceField[];
}

// T-0182 预览(dry-run)返回结构。
export interface DictionaryPreviewRow {
  label: string;
  value: string;
}
export interface DictionaryPreviewResponse {
  rows: DictionaryPreviewRow[];
  total: number;
}

// T-0182 手动刷新返回结构。
export interface DictionaryRefreshResponse {
  inserted: number;
  updated: number;
  deleted: number;
  total: number;
  duration_ms: number;
}

export interface CreateDictionaryDetailPayload {
  label: string;
  // 展示值多语言({"zh-CN":...,"en-US":...}) → 后端 label_i18n。
  labelI18n?: Record<string, string>;
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
  labelI18n?: Record<string, string>;
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

// 批量字典查询响应类型
export interface DictBatchResponse {
  [code: string]: {
    sysDictionaryDetails?: DictionaryDetail[];
  } | undefined;
}

interface BackendDictionary {
  id: number;
  name: string;
  name_i18n?: Record<string, string> | null;
  description_i18n?: Record<string, string> | null;
  type: string;
  status: boolean;
  desc: string;
  created_at?: string;
  updated_at?: string;
  // T-0182 数据源字段(后端 JSON tag 用 snake_case)。
  source_table?: string | null;
  source_label_field?: string | null;
  source_value_field?: string | null;
  last_refresh_at?: string | null;
  last_refresh_status?: string | null;
  last_refresh_error?: string | null;
  last_refresh_count?: number | null;
}

interface BackendDictionaryDetail {
  id: number;
  label: string;
  label_i18n?: Record<string, string> | null;
  value: string;
  extend: string;
  status: boolean;
  sort: number;
  sysDictionaryId: number;
  // PRD §10 v0.2：parent_id (snake) + level
  parent_id?: number | null;
  level?: number;
  // T-0182 数据源同步:origin = manual / auto
  origin?: DictionaryDetailOrigin;
  created_at?: string;
  updated_at?: string;
}

function mapBackendDictionary(b: BackendDictionary): Dictionary {
  return {
    id: b.id,
    name: b.name,
    nameI18n: b.name_i18n ?? undefined,
    descriptionI18n: b.description_i18n ?? undefined,
    type: b.type,
    status: b.status,
    desc: b.desc || '',
    createdAt: b.created_at,
    updatedAt: b.updated_at,
    sourceTable: b.source_table ?? null,
    sourceLabelField: b.source_label_field ?? null,
    sourceValueField: b.source_value_field ?? null,
    lastRefreshAt: b.last_refresh_at ?? null,
    lastRefreshStatus: (b.last_refresh_status as Dictionary['lastRefreshStatus']) ?? null,
    lastRefreshError: b.last_refresh_error ?? null,
    lastRefreshCount: b.last_refresh_count ?? null,
  };
}

function mapBackendDictionaryDetail(b: BackendDictionaryDetail): DictionaryDetail {
  return {
    id: b.id,
    label: b.label,
    labelI18n: b.label_i18n ?? undefined,
    value: b.value,
    extend: b.extend || '',
    status: b.status,
    sort: b.sort || 0,
    sysDictionaryId: b.sysDictionaryId,
    parentId: b.parent_id ?? null,
    level: b.level ?? 0,
    origin: (b.origin as DictionaryDetailOrigin) ?? 'manual',
    createdAt: b.created_at,
    updatedAt: b.updated_at,
  };
}

// 把前端 Dictionary 创建/更新 payload 转为后端 JSON(snake_case)。
// source_* 字段始终携带(undefined 跳过、null 携带 null = 解绑),让后端区分"未传"vs"显式置空"。
//
// 用 union 而非 Partial<Create & Update> — Create.sourceTable 是 string,
// Update.sourceTable 是 string|null,Partial<Create & Update> 交集后 sourceTable
// 落到 string,导致 Update payload 传 null 时不通过(v2 严格 typecheck 暴露此问题)。
function toBackendDictionaryBody(
  p: CreateDictionaryPayload | UpdateDictionaryPayload,
): Record<string, unknown> {
  const body: Record<string, unknown> = {};
  const anyP = p as CreateDictionaryPayload & UpdateDictionaryPayload;
  if (anyP.id !== undefined) body.id = anyP.id;
  if (anyP.name !== undefined) body.name = anyP.name;
  if (anyP.type !== undefined) body.type = anyP.type;
  if (anyP.status !== undefined) body.status = anyP.status;
  if (anyP.desc !== undefined) body.desc = anyP.desc;
  if (Object.prototype.hasOwnProperty.call(p, 'sourceTable')) body.source_table = anyP.sourceTable;
  if (Object.prototype.hasOwnProperty.call(p, 'sourceLabelField')) body.source_label_field = anyP.sourceLabelField;
  if (Object.prototype.hasOwnProperty.call(p, 'sourceValueField')) body.source_value_field = anyP.sourceValueField;
  return body;
}

// 把前端 camelCase payload 转成后端 JSON：parentId → parent_id（仅当字段存在时携带）。
// undefined 不发，null 发（语义=切顶层 / 清父）。
function toBackendDictionaryDetailBody(
  p: Partial<CreateDictionaryDetailPayload & UpdateDictionaryDetailPayload>,
): Record<string, unknown> {
  const body: Record<string, unknown> = {};
  if (p.id !== undefined) body.id = p.id;
  if (p.label !== undefined) body.label = p.label;
  if (p.labelI18n !== undefined) body.label_i18n = p.labelI18n;
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
  creator_username?: string;
  updater_username?: string;
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
  is_secret?: boolean;
  is_configured?: boolean;
  created_at?: string;
  updated_at?: string;
}

// Backend group model.
// 注意：后端 /admin/groups 是 roles 的只读别名（GET → ListRoles），返回的是 Role 形态：
// { id, name, description, user_count, is_system, updated_by, updated_at }（snake→camel 后）。
// 历史的 groupName/builtIn/roleCount/updUser 字段后端并不返回，故都为可选 + 读取时回退到 role 字段。
interface BackendGroup {
  id: string;
  groupName?: string;
  name?: string; // 实际后端字段（role.name）
  description: string;
  builtIn?: number;
  isSystem?: boolean; // role.is_system
  userCount?: number;
  roleCount?: number;
  updUser?: string;
  updatedBy?: string;
  updTime?: string;
  createdAt?: string;
  updatedAt?: string;
}

// Backend audit log model
interface BackendAuditLog {
  id: string;
  user_id?: string;
  userId?: string;
  username: string;
  action: string;
  resource: string;
  resource_id?: string;
  resourceId?: string;
  details?: Record<string, unknown>;
  ip_address?: string;
  ipAddress?: string;
  user_agent?: string;
  userAgent?: string;
  created_at?: string;
  createdAt?: string;
}

interface BackendNorthboundAPIInvocationLog {
  id: string;
  api_key?: string;
  apiKey?: string;
  name?: string;
  method?: string;
  path?: string;
  request_params?: string;
  requestParams?: string;
  response_body?: string;
  responseBody?: string;
  status_code?: number;
  statusCode?: number;
  status?: string;
  create_user?: string;
  createUser?: string;
  ip_address?: string;
  ipAddress?: string;
  duration_ms?: number;
  durationMs?: number;
  created_at?: string;
  createdAt?: string;
  updated_at?: string;
  updatedAt?: string;
}

interface BackendListResponse<T> {
  items: T[];
  total: number;
  page: number;
  pageSize?: number;
  page_size?: number;
  totalPages?: number;
  total_pages?: number;
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
    createUser: br.creator_username || br.created_by,
    updateUser: br.updater_username || br.updated_by,
    createTime: br.created_at,
    updateTime: br.updated_at,
  };
}

function mapBackendAuditLog(ba: BackendAuditLog): OperationLog {
  const createdAt = ba.created_at ?? ba.createdAt ?? '';
  const resourceId = ba.resource_id ?? ba.resourceId ?? '';
  const action = (ba.action || '').toLowerCase();
  const details = ba.details && Object.keys(ba.details).length > 0 ? ba.details : undefined;
  const detailsText = JSON.stringify(details || {}).toLowerCase();
  const isFailure =
    /fail|error|denied|forbid|unauthor|invalid/.test(action) ||
    /"status"\s*:\s*false|"success"\s*:\s*false|fail|error|denied|forbid/.test(detailsText);
  const reason = isFailure && ba.details && typeof ba.details === 'object' && typeof (ba.details as Record<string, unknown>).reason === 'string'
    ? (ba.details as Record<string, unknown>).reason as string
    : '';
  const summaryParts = [ba.username, ba.action, ba.resource, resourceId].map((part) => String(part || '').trim()).filter(Boolean);
  const fallbackSummary = summaryParts.length > 0 ? summaryParts.join(' · ') : '审计日志';
  const detailText = details ? JSON.stringify(details, null, 2) : fallbackSummary;

  return {
    id: ba.id,
    operator: ba.username,
    clientIp: ba.ip_address ?? ba.ipAddress ?? '',
    module: ba.resource || '',
    operationType: (ba.action || 'query') as OperationType,
    target: resourceId,
    content: detailText,
    result: isFailure ? 'failure' : 'success',
    message: reason || fallbackSummary,
    operationTime: createdAt,
    logName: `${ba.action} ${ba.resource}`.trim(),
    detail: detailText,
    reason,
    startTime: createdAt,
    endTime: createdAt,
  };
}

function mapBackendNorthboundAPIInvocationLog(b: BackendNorthboundAPIInvocationLog): NorthboundAPIInvocationLog {
  return {
    id: b.id,
    apiKey: b.api_key ?? b.apiKey ?? '',
    name: b.name ?? '',
    method: b.method ?? '',
    path: b.path ?? '',
    requestParams: b.request_params ?? b.requestParams ?? '',
    responseBody: b.response_body ?? b.responseBody ?? '',
    statusCode: b.status_code ?? b.statusCode ?? 0,
    status: b.status ?? '',
    createUser: b.create_user ?? b.createUser ?? '',
    ipAddress: b.ip_address ?? b.ipAddress ?? '',
    durationMs: b.duration_ms ?? b.durationMs ?? 0,
    createdAt: b.created_at ?? b.createdAt ?? '',
    updatedAt: b.updated_at ?? b.updatedAt ?? '',
  };
}

function applyOperationLogNameFilter(query: Record<string, unknown>, logName: string) {
  const mappings: Record<string, { action?: string; resource?: string; keyword?: string }> = {
    config_modify: { action: 'PUT', resource: 'config' },
    device_delete: { action: 'DELETE', resource: 'device' },
    user_create: { action: 'POST', resource: 'user' },
    password_reset: { action: 'password_reset', resource: 'user' },
    login_success: { action: 'login_success', resource: 'auth' },
    login_failure: { action: 'login_failed', resource: 'auth' },
    logout: { action: 'logout', resource: 'auth' },
    software_upgrade: { action: 'upgrade', resource: 'software' },
    reboot: { action: 'reboot', resource: 'device' },
    password_change: { keyword: 'password' },
    permission_change: { keyword: 'permission' },
  };
  const mapped = mappings[logName];
  if (!mapped) {
    query.resource = logName;
    return;
  }
  if (mapped.action) query.action = mapped.action;
  if (mapped.resource) query.resource = mapped.resource;
  if (mapped.keyword) query.keyword = mapped.keyword;
}

function mapBackendGroup(bg: BackendGroup): Group {
  return {
    id: bg.id,
    // /admin/groups 实为 roles 别名，名称落在 role.name；回退兼容历史 groupName 字段。
    groupName: bg.name ?? bg.groupName ?? '',
    description: bg.description || '',
    userCount: bg.userCount || 0,
    roleCount: bg.roleCount || 0,
    builtIn: bg.builtIn ?? (bg.isSystem ? 1 : 0),
    updUser: bg.updUser || bg.updatedBy || '',
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
    pageSize: resp.page_size ?? resp.pageSize ?? 20,
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
    isSecret: b.is_secret,
    isConfigured: b.is_configured,
    createdAt: b.created_at,
    updatedAt: b.updated_at,
  };
}

export const adminApi = {
  // Users
  async getUsers(
    params: UserListParams & PageRequest
  ): Promise<PageResponse<User>> {
    const query: Record<string, unknown> = {
      page: params.page,
      pageSize: params.pageSize,
    };
    if (params.userName) query.search = params.userName;
    if (params.roleId) query.roleId = params.roleId;
    if (params.status) query.status = params.status;

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

  // Issue #649：password 可选 + useDefaultPassword 开关。开关为 true 时跳过加密、
  // 不传 encrypted_password / key_id，由后端从 sys_configs.security.defaultPasswd 取。
  async createUser(
    data: Omit<User, 'id' | 'createTime' | 'lastLoginTime'> & {
      password?: string;
      useDefaultPassword?: boolean;
      roleIds?: string[];
    }
  ): Promise<User> {
    const body: Record<string, unknown> = {
      username: data.username,
      email: data.email || undefined,
      phone: data.phone || undefined,
      description: data.description || undefined,
      expire_at: data.expireTime || undefined,
      display_name: data.displayName || data.username,
      status: data.status,
      role_ids: data.roleIds && data.roleIds.length > 0 ? data.roleIds : undefined,
    };
    if (data.useDefaultPassword) {
      body.use_default_password = true;
    } else {
      if (!data.password) {
        throw new Error('createUser: password is required when useDefaultPassword is false');
      }
      const { encryptedPassword, keyId } = await preparePasswordPayload(data.password);
      body.encrypted_password = encryptedPassword;
      body.key_id = keyId;
    }
    const { data: bu } = await http.post<BackendUser>('/admin/users', body);
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

  // Issue #649：签名改为对象参数（向后兼容：webcode 调用方原本只传 newPassword）。
  // useDefaultPassword=true 时不传加密载荷，由后端取 defaultPasswd；为 false 时走原加密路径。
  async resetPassword(
    id: string,
    opts: { newPassword?: string; useDefaultPassword?: boolean },
  ): Promise<void> {
    const body: Record<string, unknown> = {};
    if (opts.useDefaultPassword) {
      body.use_default_password = true;
    } else {
      if (!opts.newPassword) {
        throw new Error('resetPassword: newPassword is required when useDefaultPassword is false');
      }
      const { encryptedPassword, keyId } = await preparePasswordPayload(opts.newPassword);
      body.encrypted_new_password = encryptedPassword;
      body.key_id = keyId;
    }
    await http.post(`/admin/users/${id}/reset-password`, body);
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
      pageSize: data.page_size ?? data.pageSize ?? params.pageSize,
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
      pageSize: data.page_size ?? data.pageSize ?? params.pageSize,
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

  // 写操作走 /admin/roles：后端 /admin/groups 只有 GET（roles 只读别名），无 POST/PUT/DELETE，
  // 旧实现 POST /admin/groups 会 404。groups==roles，故名称→role.name 落库。
  async createGroup(data: Omit<Group, 'id' | 'userCount' | 'roleCount' | 'updUser' | 'updTime'> & { roleIds?: string[]; userIds?: string[] }): Promise<Group> {
    const { data: result } = await http.post<BackendGroup>('/admin/roles', {
      name: data.groupName,
      description: data.description,
    });
    return mapBackendGroup(result);
  },

  async updateGroup(id: string, data: Partial<Group> & { roleIds?: string[]; userIds?: string[] }): Promise<Group> {
    const payload: Record<string, unknown> = {};
    if (data.groupName !== undefined) payload.name = data.groupName;
    if (data.description !== undefined) payload.description = data.description;
    const { data: result } = await http.put<BackendGroup>(`/admin/roles/${id}`, payload);
    return mapBackendGroup(result);
  },

  async deleteGroups(ids: string[]): Promise<void> {
    for (const id of ids) {
      await http.delete(`/admin/roles/${id}`);
    }
  },

  // Audit logs
  async getOperationLogs(
    params: {
      operator?: string;
      clientIp?: string;
      module?: string;
      action?: string;
      operationType?: OperationType;
      result?: string;
      reason?: string;
      timeRange?: [string, string];
      keyword?: string;
    } & PageRequest
  ): Promise<PageResponse<OperationLog>> {
    const query: Record<string, unknown> = {
      page: params.page,
      pageSize: params.pageSize,
    };
    if (params.operator) query.username = params.operator;
    if (params.clientIp) query.ip_address = params.clientIp;
    if (params.module) applyOperationLogNameFilter(query, params.module);
    if (params.action) query.action = params.action;
    if (params.operationType) query.action = params.operationType;
    if (params.result) query.result = params.result;
    if (params.reason) query.reason = params.reason;
    if (params.keyword) query.keyword = params.keyword;
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
      pageSize: data.page_size ?? data.pageSize ?? params.pageSize,
    };
  },

  async getNorthboundAPIInvocationLogs(
    params: {
      apiKey?: string;
      name?: string;
      method?: string;
      path?: string;
      status?: string;
      createUser?: string;
      ipAddress?: string;
      timeRange?: [string, string];
    } & PageRequest
  ): Promise<PageResponse<NorthboundAPIInvocationLog>> {
    const query: Record<string, unknown> = {
      page: params.page,
      pageSize: params.pageSize,
    };
    if (params.apiKey) query.api_key = params.apiKey;
    if (params.name) query.name = params.name;
    if (params.method) query.method = params.method;
    if (params.path) query.path = params.path;
    if (params.status) query.status = params.status;
    if (params.createUser) query.create_user = params.createUser;
    if (params.ipAddress) query.ip_address = params.ipAddress;
    if (params.timeRange) {
      query.start_time = params.timeRange[0];
      query.end_time = params.timeRange[1];
    }

    const { data } = await http.get<BackendListResponse<BackendNorthboundAPIInvocationLog>>(
      '/northbound/page-config/api/invocation-logs',
      { params: query }
    );

    return {
      items: (data.items || []).map(mapBackendNorthboundAPIInvocationLog),
      total: data.total,
      page: data.page,
      pageSize: data.page_size ?? data.pageSize ?? params.pageSize,
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

  // getUserMenuTree (调 /admin/menus/user) 已删除：路径 PRD §3.2 / §4.2.1 标定为
  // 错路径，前端统一改走 menuApi.fetchUserMenus → GET /auth/menus（含
  // currentRoleID 感知）。

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
    const { data } = await http.post<BackendDictionary>(
      '/admin/sysDictionary/createSysDictionary',
      toBackendDictionaryBody(req),
    );
    return mapBackendDictionary(data);
  },

  async updateDictionary(req: UpdateDictionaryPayload): Promise<Dictionary> {
    const { data } = await http.put<BackendDictionary>(
      '/admin/sysDictionary/updateSysDictionary',
      toBackendDictionaryBody(req),
    );
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

  // ---- T-0182 数据字典数据源 ----
  // 白名单 + dry-run 预览 + 手动刷新 3 个端点。所有调用方都在 /system/data-dictionary
  // 页面;sourceTable 为 null 的字典不会触发这些路径(UI 隐藏对应入口)。

  async listDictionarySources(): Promise<DictionarySource[]> {
    const { data } = await http.get<{ sources: DictionarySource[] }>(
      '/admin/sysDictionary/sources',
    );
    return data.sources || [];
  },

  async previewDictionarySource(params: {
    table: string;
    label: string;
    value: string;
    limit?: number;
  }): Promise<DictionaryPreviewResponse> {
    const { data } = await http.get<DictionaryPreviewResponse>(
      '/admin/sysDictionary/sources/preview',
      {
        params: {
          table: params.table,
          label: params.label,
          value: params.value,
          limit: params.limit ?? 10,
        },
      },
    );
    return { rows: data.rows || [], total: data.total ?? 0 };
  },

  async refreshDictionarySource(id: number): Promise<DictionaryRefreshResponse> {
    const { data } = await http.post<DictionaryRefreshResponse>(
      '/admin/sysDictionary/refreshSource',
      undefined,
      { params: { id } },
    );
    return data;
  },

  // ---- Batch dictionary queries ----
  // 批量获取字典数据，减少 HTTP 请求次数（性能优化）
  async batchGetDicts(codes: string[]): Promise<DictBatchResponse> {
    if (codes.length === 0) {
      return {};
    }
    const { data } = await http.get<{ dicts: DictBatchResponse }>('/admin/sysDictionary/batch', {
      params: { codes: codes.join(',') }
    });
    return data.dicts;
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
      pageSize: data.page_size ?? data.pageSize ?? params.pageSize ?? 20,
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
    const oldPayload = await preparePasswordPayload(data.old_password);
    const newPayload = await preparePasswordPayload(data.new_password);
    // 两次 preparePasswordPayload 共用同一公钥缓存 → keyId 必然相同；取其一即可。
    await http.post('/auth/change-password', {
      encrypted_old_password: oldPayload.encryptedPassword,
      encrypted_new_password: newPayload.encryptedPassword,
      key_id: oldPayload.keyId,
    });
  },

  // ---- System Config (KV by category, batch upsert) ----
  // 参 omgo/docs/prd/system/config.md §5。前端 system/config 页面 7 Tab 用：
  // 进入页面/切 Tab 调 getSysConfigsByCategory；保存按钮调 batchUpdateSysConfigs。
  async getSysConfigsByCategory(category: string): Promise<SysConfigItem[]> {
    const { data } = await http.get<BackendSysConfig[]>('/admin/sysConfig', { params: { category } });
    return (Array.isArray(data) ? data : []).map(mapBackendSysConfig);
  },

  async batchUpdateSysConfigs(payload: BatchUpdateSysConfigPayload): Promise<BatchUpdateSysConfigResult> {
    const { data } = await http.post<{
      updated?: number;
      batch?: {
        id?: string;
        category?: string;
        config_version?: number;
        status?: ConfigApplyStatus;
        created_at?: string;
        updated_at?: string;
        targets?: Array<{
          target?: string;
          status?: ConfigApplyStatus;
          success_scope?: 'runtime_applied' | 'event_delivered';
          attempts?: number;
          applied_at?: string;
          last_error?: string;
        }>;
      };
    }>('/admin/sysConfig/batch', payload);
    const batch = data?.batch;
    if (!batch?.id || !batch.category || !batch.status) {
      throw new Error('配置保存响应缺少应用批次状态');
    }
    return {
      updated: data?.updated ?? 0,
      batch: {
        id: batch.id,
        category: batch.category,
        configVersion: batch.config_version ?? 0,
        status: batch.status,
        createdAt: batch.created_at ?? '',
        updatedAt: batch.updated_at ?? '',
        targets: (batch.targets ?? []).map((target) => ({
          target: target.target ?? '',
          status: target.status ?? 'pending',
          successScope: target.success_scope
            ?? (target.target === 'acs_transfer_event_delivery' ? 'event_delivered' : 'runtime_applied'),
          attempts: target.attempts ?? 0,
          appliedAt: target.applied_at,
          lastError: target.last_error,
        })),
      },
    };
  },

	async sendTestEmail(recipient: string): Promise<{ recipient: string }> {
		const { data } = await http.post<{ recipient: string }>(
			'/admin/notification/email/test',
			{ recipient },
		);
		return data;
	},

  async getSysConfigApplyBatch(id: string): Promise<ConfigApplyBatch> {
    const { data } = await http.get<{
      id: string;
      category: string;
      config_version: number;
      status: ConfigApplyStatus;
      created_at: string;
      updated_at: string;
      targets: Array<{
        target: string;
        status: ConfigApplyStatus;
        success_scope?: 'runtime_applied' | 'event_delivered';
        attempts: number;
        applied_at?: string;
        last_error?: string;
      }>;
    }>(`/admin/sysConfig/apply-batches/${id}`);
    return {
      id: data.id,
      category: data.category,
      configVersion: data.config_version,
      status: data.status,
      createdAt: data.created_at,
      updatedAt: data.updated_at,
      targets: data.targets.map((target) => ({
        target: target.target,
        status: target.status,
        successScope: target.success_scope
          ?? (target.target === 'acs_transfer_event_delivery' ? 'event_delivered' : 'runtime_applied'),
        attempts: target.attempts,
        appliedAt: target.applied_at,
        lastError: target.last_error,
      })),
    };
  },

  // ---- Public configs (no auth) ----
  // 登录页用：拉取 is_public=true 的配置项（产品名、Logo、登录背景等）。
  // 参 docs/prd/system/ui-customization.md §6。
  async getPublicSysConfigsByCategory(category: string): Promise<SysConfigItem[]> {
    const { data } = await http.get<BackendSysConfig[]>('/admin/public/configs', { params: { category } });
    return (Array.isArray(data) ? data : []).map(mapBackendSysConfig);
  },

  // ---- UI Customization asset upload ----
  // 上传 Logo / 登录背景图到 MinIO，返回相对 URL（写回 sys_configs.value）。
  // kind 决定后端体积上限（login_bg < 1 MiB；logo_small/logo_large < 400 KiB）。
  async uploadUIAsset(
    file: File | Blob,
    kind: 'login_bg' | 'logo_small' | 'logo_large',
  ): Promise<{ url: string; name: string; size: number }> {
    const form = new FormData();
    form.append('file', file);
    form.append('kind', kind);
    const { data } = await http.post<{ url: string; name: string; size: number }>(
      '/admin/uploads/ui-asset',
      form,
      { headers: { 'Content-Type': 'multipart/form-data' } },
    );
    return data;
  },
};
