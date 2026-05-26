/**
 * F05 MR 测量任务类型定义。
 *
 * 与后端 `omcgo/internal/mr/task/` 的 REST API 对齐：
 *   POST   /api/v1/mr/tasks
 *   GET    /api/v1/mr/tasks
 *   GET    /api/v1/mr/tasks/:id
 *   POST   /api/v1/mr/tasks/:id/stop
 *   DELETE /api/v1/mr/tasks/:id
 *   GET    /api/v1/mr/tasks/:id/progress
 *
 * 命名约定：业务层（services/api/mrTaskApi.ts）从后端 snake_case 拿到
 * BackendMRTask 形状，经 mapBackendMRTask 映射为前端 camelCase MRTask。
 * 这里只暴露前端（消费侧）形状；Backend* 形状声明在 mrTaskApi.ts 内私有。
 */

/** 任务整体状态。后端枚举值（注意 waitting 拼写与后端一致，不要纠正）。 */
export type MRTaskStatus =
  | 'waitting'
  | 'on'
  | 'off'
  | 'suspend'
  | 'termination';

/** Cell 维度的下发结果状态。 */
export type MRProgressStatus =
  | 'pending'
  | 'openSuccess'
  | 'openFailure'
  | 'closeSuccess'
  | 'closeFailure'
  | 'unsupport'
  | 'timeOut'
  | 'noPermission';

/** Cell 维度的上报心跳健康度（独立于 progressStatus）。 */
export type MRHealthStatus = 'normal' | 'abnormal' | 'unknown';

/** 合法的上报周期（UI 分钟值）。下发时由后端 × 60 转为 UploadPeriod 秒。 */
export type MRReportPeriod = '15' | '30' | '60';

/**
 * 合法的统计周期（PeriodicReportInterval 下发原值）。
 * ms 前缀只是展示标签，后端存的就是裸数字字符串。
 */
export type MRStatisPeriod =
  | '2048'
  | '5120'
  | '10240'
  | '1'
  | '6'
  | '12'
  | '30'
  | '60';

/** MR 任务主记录。
 *
 * 简化版（2026-05-25）：去除 operator_code / note；creator 由后端自动填，
 * 前端只读展示。
 */
export interface MRTask {
  taskId: string;
  taskName: string;
  /** 逗号分隔，强制含 MRS,MRE,MRO（后端校验）。 */
  mrType: string;
  statisPeriod: MRStatisPeriod;
  reportPeriod: MRReportPeriod;
  /** UTC ISO 字符串。前端按用户时区渲染。 */
  startTime: string;
  /** UTC ISO 字符串；undefined 表示无限制。 */
  endTime?: string;
  taskStatus: MRTaskStatus;
  taskResult?: string;
  /** 后端从 auth ctx 自动填，前端只读。 */
  creator: string;
  /** 用户选中的目标设备 SN 列表（2026-05-26 加回）。 */
  targetDeviceSns: string[];
  createdAt: string;
  updatedAt: string;
}

/** Cell 维度进度。 */
export interface MRTaskProgress {
  id: string;
  taskId: string;
  smallCellCode: string;
  serialNumber: string;
  hostName?: string;
  progressStatus: MRProgressStatus;
  healthStatus: MRHealthStatus;
  faultCode?: string;
  /** UTC ISO 字符串；undefined 表示从未上报。 */
  lastHeartbeat?: string;
  missedHeartbeat: number;
  createdAt: string;
  updatedAt: string;
}

/** 创建任务请求体。
 *
 * 简化版（2026-05-25）：去除 operator_code / creator / note / targets。
 * creator 后端自动从 auth ctx 取；targets 后端在 scheduler 开启任务时
 * 从 mr_device_mappings 动态枚举（所有 enabled=true 的 cell）。
 */
export interface CreateMRTaskRequest {
  taskName: string;
  /** 缺省 'MRS,MRE,MRO'。 */
  mrType?: string;
  /** 缺省 '5120'。 */
  statisPeriod?: MRStatisPeriod;
  /** 缺省 '15'。 */
  reportPeriod?: MRReportPeriod;
  /** UTC ISO 字符串。 */
  startTime: string;
  /** UTC ISO 字符串；空 / undefined 表示无限制。 */
  endTime?: string;
  /** 目标设备 SN 列表，至少 1 个。 */
  targetDeviceSns: string[];
}

/** 列表过滤参数。 */
export interface MRTaskListFilter {
  status?: MRTaskStatus;
  keyword?: string;
  page: number;
  pageSize: number;
  sortBy?: string;
  sortDir?: 'asc' | 'desc';
}

/** 进度列表过滤参数。 */
export interface MRTaskProgressFilter {
  status?: MRProgressStatus;
  health?: MRHealthStatus;
  page: number;
  pageSize: number;
}
