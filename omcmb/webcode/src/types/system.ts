export type UserOnlineStatus = 'online' | 'offline';
export type UserLockStatus = 0 | 1 | 2; // 0=解锁, 1=有效期锁定, 2=有效期锁定

export interface User {
  id: string;
  userName: string;
  email: string;
  groupNames: string[]; // 所属用户组列表
  lastLoginTime: string;
  source: string; // 来源
  onlineStatus: UserOnlineStatus; // 在线状态
  lockStatus: UserLockStatus; // 锁定状态
  builtIn: number; // 内置用户标识
  description: string;
  createTime: string;
}

export interface Role {
  id: string;
  roleName: string;
  batchOperation: number; // 1=是, 0=否
  description: string;
  permissions: string[];
  userCount: number;
  updUser: string; // 操作人
  updTime: string; // 更新时间
  builtIn: number; // 内置角色标识
}

export interface Group {
  id: string;
  groupName: string;
  description: string;
  userCount: number;
  roleCount: number;
  builtIn: number; // 1,2 = built-in, others = custom
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
  operator: string;         // 用户名称
  clientIp: string;         // IP地址
  logName: string;          // 日志名称
  detail: string;           // 详细记录
  result: '1' | '0';        // 结果：1-成功，0-失败
  reason: string;           // 失败原因
  startTime: string;        // 操作开始时间
  endTime: string;          // 操作结束时间
  // 兼容旧字段
  module?: string;
  operationType?: OperationType;
  target?: string;
  content?: string;
  message?: string;
  operationTime?: string;
}
