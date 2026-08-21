export type VersionStatus = 'current' | 'deprecated' | 'beta' | 'archived';
export type UpgradePlanStatus = 'pending' | 'running' | 'success' | 'failed' | 'cancelled' | 'scheduled' | 'partial';

// Main task status (maps to backend TaskStatus)
export type TaskStatusType = 'pending' | 'in_progress' | 'suspended' | 'ended';
// Main task result
export type TaskResultType = 'success' | 'partial' | 'failed' | 'terminated';
// Sub-task status (maps to backend UpgradeState)
export type SubTaskStatusType = 'pending' | 'downloading' | 'rebooting' | 'verifying' | 'completed' | 'failed' | 'suspended' | 'terminated';
// Task type (maps to backend TaskType)
export type TaskTypeValue = 1 | 2 | 4 | 6 | 8;
// File type (maps to backend FileType)
export type FileTypeValue = 0 | 1 | 5 | 6;

export interface SoftwareVersion {
  id: string;
  versionName: string;
  versionCode: string;
  fileName: string;
  deviceType: string;
  /** #492 固件所属产品（products.id）。上传选产品名 → 存此；前端按 useProductList 映射展示产品名。历史固件为空。 */
  productId?: string;
  /** #638 固件适用的全部产品列表（products.id[]）。product_id = productIds[0]（"主产品"，唯一索引仍走它）。
   *  上传/编辑表单按多选写入；列表与编辑回填以此为准。历史固件为空数组或未带。 */
  productIds?: string[];
  vendor: string;
  releaseDate: string;
  status: VersionStatus;
  fileSize: number;
  checksum: string;
  downloadUrl: string;
  releaseNotes: string;
  minHardwareVersion: string;
  features: string[];
  bugFixes: string[];
  known_issues?: string[];
  // New fields for backend alignment
  fileType?: FileTypeValue;
  recommend?: boolean;
  uploader?: string;
  manufacturer?: string;
  description?: string;
}

export interface UpgradePlan {
  id: string;
  planName: string;
  targetVersionId: string;
  targetVersionCode: string;
  deviceSns: string[];
  status: UpgradePlanStatus;
  progress: number;
  successCount: number;
  failCount: number;
  totalCount: number;
  scheduledTime?: string;
  startTime?: string;
  endTime?: string;
  createdAt: string;
  updatedAt: string;
  creator: string;
  preCheckRequired: boolean;
  rollbackEnabled: boolean;
  message?: string;
}

// Canary upgrade strategy types (T-0019, mirrors backend T-0018).
// strategy='full' is the legacy path; 'canary' adds staged rollout with
// per-stage failure-rate gating.
export type CanaryStageStatus =
  | 'pending'
  | 'running'
  | 'paused'
  | 'aborted'
  | 'completed';

export interface CanaryStage {
  percent: number;          // 1..100 cumulative
  failureThreshold: number; // 1..100 percentage
}

export interface StageHistoryEntry {
  stage: number;
  percent: number;
  devicesInStage: number;
  successCount: number;
  failCount: number;
  failureRate: number;      // 0.0 - 1.0
  action: string;           // 'advanced' / 'paused' / 'aborted' / 'completed'
  at: string;
  reason?: string;
}

// UpgradeTaskInfo — frontend model for main upgrade task (upgrade_tasks table)
export interface UpgradeTaskInfo {
  id: string;
  taskName: string;
  taskType: TaskTypeValue;
  firmwareId?: string;
  fileName?: string;
  fileMd5?: string;
  status: TaskStatusType;
  result?: TaskResultType;
  productClass: string;
  isKeepConfig: boolean;
  createStatus: string;
  createUser: string;
  totalCount: number;
  successCount: number;
  failCount: number;
  maxConcurrent: number;
  startedAt?: string;
  endedAt?: string;
  createdAt: string;
  updatedAt: string;

  // Canary fields (T-0019, optional for backwards compat with full-strategy tasks)
  strategy?: 'full' | 'canary';
  canaryStages?: CanaryStage[];
  currentStage?: number;          // 1-indexed
  stageStatus?: CanaryStageStatus;
  stageHistory?: StageHistoryEntry[];
  autoAdvance?: boolean;
  autoAdvanceMinutes?: number;
}

// UpgradeSubTaskInfo — frontend model for sub-task (upgrade_sub_tasks table)
export interface UpgradeSubTaskInfo {
  id: string;
  taskId: string;
  taskName?: string;
  deviceId: string;
  firmwareId?: string;
  status: SubTaskStatusType;
  errorMessage?: string;
  retryCount: number;
  maxRetries: number;
  deviceSn?: string;
  oriVersion?: string;
  destVersion?: string;
  commandKey?: string;
  failureReason?: string;
  preSuspendStatus?: string;
  startedAt?: string;
  completedAt?: string;
  createdAt: string;
  updatedAt: string;
}

export const mockSoftwareVersions: SoftwareVersion[] = [
  {
    id: 'ver-001',
    versionName: 'eNB V100R011C10SPC200',
    versionCode: 'V100R011C10SPC200',
    deviceType: 'eNB',
    vendor: '华为',
    releaseDate: '2024-03-15T00:00:00.000Z',
    status: 'current',
    fileSize: 1024 * 1024 * 512,
    checksum: 'a1b2c3d4e5f6789012345678901234567890abcd',
    fileName: 'mock-firmware.bin',
    downloadUrl: '/software/enb/V100R011C10SPC200.tar.gz',
    releaseNotes: '修复多个已知问题，提升系统稳定性，支持新频段配置',
    minHardwareVersion: 'BBU3900 V5.0',
    features: ['载波聚合增强', '波束成形优化', '节能算法V3', '5G-4G协同'],
    bugFixes: ['修复X2链路偶发中断', '修复计数器溢出问题', '修复GPS时钟漂移'],
  },
  {
    id: 'ver-002',
    versionName: 'eNB V100R011C10SPC100',
    versionCode: 'V100R011C10SPC100',
    deviceType: 'eNB',
    vendor: '华为',
    releaseDate: '2023-09-01T00:00:00.000Z',
    status: 'deprecated',
    fileSize: 1024 * 1024 * 480,
    checksum: 'b2c3d4e5f67890123456789012345678901abcde',
    fileName: 'mock-firmware.bin',
    downloadUrl: '/software/enb/V100R011C10SPC100.tar.gz',
    releaseNotes: '基础功能完善，修复关键安全漏洞',
    minHardwareVersion: 'BBU3900 V4.0',
    features: ['基础LTE功能', '载波聚合', 'VoLTE支持'],
    bugFixes: ['安全漏洞修复CVE-2023-12345', '切换成功率提升'],
    known_issues: ['已知：载波聚合场景下偶发计数器异常'],
  },
  {
    id: 'ver-003',
    versionName: 'gNB V200R001C10SPC100',
    versionCode: 'V200R001C10SPC100',
    deviceType: 'gNB',
    vendor: '华为',
    releaseDate: '2024-04-01T00:00:00.000Z',
    status: 'current',
    fileSize: 1024 * 1024 * 768,
    checksum: 'c3d4e5f678901234567890123456789012abcdef',
    fileName: 'mock-firmware.bin',
    downloadUrl: '/software/gnb/V200R001C10SPC100.tar.gz',
    releaseNotes: '5G NR全面优化版本，支持SA和NSA双模',
    minHardwareVersion: 'BBU5900 V2.0',
    features: ['5G SA模式', '5G NSA模式', '网络切片', '上行增强', '大规模MIMO优化'],
    bugFixes: ['修复NG接口连接稳定性', '修复网络切片资源泄漏', '修复XN接口切换失败'],
  },
  {
    id: 'ver-004',
    versionName: 'gNB V200R001C00SPC100',
    versionCode: 'V200R001C00SPC100',
    deviceType: 'gNB',
    vendor: '华为',
    releaseDate: '2023-12-01T00:00:00.000Z',
    status: 'deprecated',
    fileSize: 1024 * 1024 * 720,
    checksum: 'd4e5f6789012345678901234567890123abcdef0',
    fileName: 'mock-firmware.bin',
    downloadUrl: '/software/gnb/V200R001C00SPC100.tar.gz',
    releaseNotes: '5G NR基础商用版本',
    minHardwareVersion: 'BBU5900 V1.0',
    features: ['5G NSA模式', '基础网络切片', '毫米波支持'],
    bugFixes: ['基础稳定性修复'],
    known_issues: ['已知：SA模式下特定场景连接中断'],
  },
  {
    id: 'ver-005',
    versionName: 'CPE V300R001C00',
    versionCode: 'V300R001C00',
    deviceType: 'CPE',
    vendor: '华为',
    releaseDate: '2024-02-01T00:00:00.000Z',
    status: 'current',
    fileSize: 1024 * 1024 * 128,
    checksum: 'e5f67890123456789012345678901234abcdef01',
    fileName: 'mock-firmware.bin',
    downloadUrl: '/software/cpe/V300R001C00.tar.gz',
    releaseNotes: 'CPE设备最新固件，提升无线性能和稳定性',
    minHardwareVersion: 'CPE Pro 2',
    features: ['5G/4G双模', 'WiFi 6支持', '远程管理增强', '安全加固'],
    bugFixes: ['修复WiFi频繁断连', '修复远程管理超时', '修复固件升级失败'],
  },
  {
    id: 'ver-006',
    versionName: 'eGW V100R002C10',
    versionCode: 'V100R002C10',
    deviceType: 'eGW',
    vendor: '华为',
    releaseDate: '2024-05-01T00:00:00.000Z',
    status: 'current',
    fileSize: 1024 * 1024 * 256,
    checksum: 'f67890123456789012345678901234567abcdef012',
    fileName: 'mock-firmware.bin',
    downloadUrl: '/software/egw/V100R002C10.tar.gz',
    releaseNotes: 'eGW边缘网关增强版，支持边缘计算功能',
    minHardwareVersion: 'EGW200 V2.0',
    features: ['边缘计算支持', '低时延优化', 'MEC接口', '流量调度增强'],
    bugFixes: ['修复路由表溢出', '修复NAT转换错误'],
  },
  {
    id: 'ver-007',
    versionName: 'eNB V100R011C10SPC300-Beta',
    versionCode: 'V100R011C10SPC300',
    deviceType: 'eNB',
    vendor: '华为',
    releaseDate: '2024-06-01T00:00:00.000Z',
    status: 'beta',
    fileSize: 1024 * 1024 * 520,
    checksum: '0123456789012345678901234567890abcdef0123',
    fileName: 'mock-firmware.bin',
    downloadUrl: '/software/enb/V100R011C10SPC300-beta.tar.gz',
    releaseNotes: 'Beta版本，包含AI智能运维特性，仅用于测试环境',
    minHardwareVersion: 'BBU3900 V5.0',
    features: ['AI故障预测', '智能节能V4', 'O-RAN接口', '云原生支持'],
    bugFixes: [],
    known_issues: ['Beta版本，可能存在未知问题，请勿用于生产环境'],
  },
  {
    id: 'ver-008',
    versionName: 'gNB V200R002C00-Beta',
    versionCode: 'V200R002C00',
    deviceType: 'gNB',
    vendor: '华为',
    releaseDate: '2024-06-10T00:00:00.000Z',
    status: 'beta',
    fileSize: 1024 * 1024 * 800,
    checksum: '123456789012345678901234567890abcdef01234',
    fileName: 'mock-firmware.bin',
    downloadUrl: '/software/gnb/V200R002C00-beta.tar.gz',
    releaseNotes: '5G高级特性Beta版，包含RedCap和NTN初步支持',
    minHardwareVersion: 'BBU5900 V3.0',
    features: ['RedCap支持', 'NTN接口(实验)', '切片增强V2', 'AI无线资源管理'],
    bugFixes: [],
    known_issues: ['NTN功能仍在开发中，请勿在生产环境使用'],
  },
  {
    id: 'ver-009',
    versionName: 'eNB V100R010C10SPC600',
    versionCode: 'V100R010C10SPC600',
    deviceType: 'eNB',
    vendor: '华为',
    releaseDate: '2023-01-01T00:00:00.000Z',
    status: 'archived',
    fileSize: 1024 * 1024 * 420,
    checksum: '23456789012345678901234567890abcdef012345',
    fileName: 'mock-firmware.bin',
    downloadUrl: '/software/enb/V100R010C10SPC600.tar.gz',
    releaseNotes: '老版本，已归档，不建议使用',
    minHardwareVersion: 'BBU3910 V3.0',
    features: ['基础LTE', 'VoLTE', 'CoMP'],
    bugFixes: ['历史版本累计修复'],
    known_issues: ['不再维护，存在已知安全漏洞'],
  },
  {
    id: 'ver-010',
    versionName: 'CPE V200R002C00',
    versionCode: 'V200R002C00',
    deviceType: 'CPE',
    vendor: '华为',
    releaseDate: '2023-08-01T00:00:00.000Z',
    status: 'deprecated',
    fileSize: 1024 * 1024 * 96,
    checksum: '3456789012345678901234567890abcdef0123456',
    fileName: 'mock-firmware.bin',
    downloadUrl: '/software/cpe/V200R002C00.tar.gz',
    releaseNotes: '旧版CPE固件，功能有限',
    minHardwareVersion: 'CPE B2351',
    features: ['4G LTE', 'WiFi 5', '基础管理'],
    bugFixes: ['基础稳定性'],
    known_issues: ['不支持5G功能'],
  },
  {
    id: 'ver-ups-001',
    versionName: 'UPS AP V1.0.0',
    versionCode: 'UPS-AP-V1.0.0',
    deviceType: 'UPS',
    productId: 'p-003',
    productIds: ['p-003'],
    vendor: 'Baicells',
    releaseDate: '2026-08-19T00:00:00.000Z',
    status: 'current',
    fileSize: 1024 * 1024 * 32,
    checksum: '456789012345678901234567890abcdef01234567',
    fileName: 'ups-ap-v1.0.0.bin',
    downloadUrl: '/software/ups/ups-ap-v1.0.0.bin',
    releaseNotes: 'UPS AP 固件演示版本',
    minHardwareVersion: 'UPS-M3',
    features: ['UPS 运行状态上报', 'BMS 信息采集'],
    bugFixes: ['修复升级演示数据缺失'],
    fileType: 5,
    recommend: true,
  },
];

export const mockUpgradePlans: UpgradePlan[] = [
  {
    id: 'upg-001',
    planName: '华北eNB批量升级至SPC200',
    targetVersionId: 'ver-001',
    targetVersionCode: 'V100R011C10SPC200',
    deviceSns: ['ENB00001', 'ENB00002', 'ENB00003', 'ENB00004', 'ENB00005'],
    status: 'success',
    progress: 100,
    successCount: 5,
    failCount: 0,
    totalCount: 5,
    startTime: '2024-06-10T02:00:00.000Z',
    endTime: '2024-06-10T04:30:00.000Z',
    createdAt: '2024-06-09T18:00:00.000Z',
    updatedAt: '2024-06-10T04:30:00.000Z',
    creator: 'admin',
    preCheckRequired: true,
    rollbackEnabled: true,
    message: '升级成功，所有设备运行正常',
  },
  {
    id: 'upg-002',
    planName: '华东gNB升级至V200R001C10SPC100',
    targetVersionId: 'ver-003',
    targetVersionCode: 'V200R001C10SPC100',
    deviceSns: ['GNB00010', 'GNB00011', 'GNB00012'],
    status: 'partial',
    progress: 100,
    successCount: 2,
    failCount: 1,
    totalCount: 3,
    startTime: '2024-06-12T22:00:00.000Z',
    endTime: '2024-06-13T01:00:00.000Z',
    createdAt: '2024-06-12T10:00:00.000Z',
    updatedAt: '2024-06-13T01:00:00.000Z',
    creator: 'operator01',
    preCheckRequired: true,
    rollbackEnabled: true,
    message: 'GNB00012升级失败已自动回退，其余2台成功',
  },
  {
    id: 'upg-003',
    planName: 'CPE全国固件升级',
    targetVersionId: 'ver-005',
    targetVersionCode: 'V300R001C00',
    deviceSns: ['CPE00001', 'CPE00002', 'CPE00003', 'CPE00010'],
    status: 'running',
    progress: 50,
    successCount: 2,
    failCount: 0,
    totalCount: 4,
    scheduledTime: new Date(Date.now() - 3600000).toISOString(),
    startTime: new Date(Date.now() - 3600000).toISOString(),
    createdAt: new Date(Date.now() - 7200000).toISOString(),
    updatedAt: new Date(Date.now() - 300000).toISOString(),
    creator: 'admin',
    preCheckRequired: false,
    rollbackEnabled: false,
  },
  {
    id: 'upg-004',
    planName: '华南eNB计划升级',
    targetVersionId: 'ver-001',
    targetVersionCode: 'V100R011C10SPC200',
    deviceSns: ['ENB00020', 'ENB00021', 'ENB00022'],
    status: 'scheduled',
    progress: 0,
    successCount: 0,
    failCount: 0,
    totalCount: 3,
    scheduledTime: new Date(Date.now() + 86400000).toISOString(),
    createdAt: new Date(Date.now() - 3600000).toISOString(),
    updatedAt: new Date(Date.now() - 3600000).toISOString(),
    creator: 'operator02',
    preCheckRequired: true,
    rollbackEnabled: true,
  },
  {
    id: 'upg-005',
    planName: 'eGW升级测试',
    targetVersionId: 'ver-006',
    targetVersionCode: 'V100R002C10',
    deviceSns: ['EGW00001'],
    status: 'failed',
    progress: 30,
    successCount: 0,
    failCount: 1,
    totalCount: 1,
    startTime: '2024-06-11T14:00:00.000Z',
    endTime: '2024-06-11T14:20:00.000Z',
    createdAt: '2024-06-11T13:00:00.000Z',
    updatedAt: '2024-06-11T14:20:00.000Z',
    creator: 'admin',
    preCheckRequired: true,
    rollbackEnabled: true,
    message: '预检查失败：磁盘空间不足，需要至少2GB可用空间',
  },
];
