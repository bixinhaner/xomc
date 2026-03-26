export type UserStatus = 'active' | 'inactive' | 'locked';
export type UserRole = 'admin' | 'operator' | 'viewer' | 'auditor';

export interface User {
  id: string;
  username: string;
  displayName: string;
  email: string;
  phone: string;
  role: UserRole;
  status: UserStatus;
  lastLoginTime: string;
  createTime: string;
}

export interface Role {
  id: string;
  roleName: string;
  description: string;
  permissions: string[];
  userCount: number;
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
  operator: string;
  clientIp: string;
  module: string;
  operationType: OperationType;
  target: string;
  content: string;
  result: OperationResult;
  message: string;
  operationTime: string;
}
