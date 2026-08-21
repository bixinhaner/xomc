import type { PageRequest, PageResponse } from '../../types/pagination';
import type {
  CreateUnifiedFileTransferTaskInput,
  CreateUnifiedFileTransferTypeInput,
  TransferExecutionMode,
  UnifiedFileTransferDeviceItem,
  UnifiedFileTransferCategory,
  UnifiedFileTransferOverview,
  UnifiedFileTransferTask,
  UnifiedFileTransferTaskType,
  UpdateUnifiedFileTransferTaskTypeInput,
} from '../../types/unifiedFileTransfer';
import { softwareService } from './softwareService';
import { delay, generateId, paginate } from '../utils';

const now = Date.now();

const DEFAULT_CATEGORY_LABELS: Record<string, string> = {
  gnb_upgrade: '5G升级',
  enb_upgrade: '4G升级',
  gsm_upgrade: '2G升级', // qa-614 c6 #365 #373
  ups_upgrade: 'UPS升级',
  version_rollback: '基站版本回退',
  station_log: '基站日志',
  config_backup: '配置备份',
  config_restore: '配置恢复',
};

function createCategoryKey(): string {
  return `custom_${Date.now().toString(36)}_${Math.random().toString(36).slice(2, 6)}`;
}

function getCategoryLabel(category: UnifiedFileTransferCategory, explicitLabel?: string): string {
  if (explicitLabel) {
    return explicitLabel;
  }
  const matchedType = taskTypes.find((item) => item.category === category);
  if (matchedType) {
    return matchedType.categoryLabel;
  }
  return DEFAULT_CATEGORY_LABELS[category] ?? category;
}

function createDevicePreviewItems(task: UnifiedFileTransferTask, typeDef: UnifiedFileTransferTaskType | undefined): UnifiedFileTransferDeviceItem[] {
  const previewCount = Math.max(2, Math.min(task.totalCount, 4));
  return Array.from({ length: previewCount }, (_, index) => ({
    id: generateId('ufte-device'),
    taskId: task.id,
    taskName: task.taskName,
    category: task.category,
    categoryLabel: task.categoryLabel,
    typeCode: task.typeCode,
    typeDisplayName: task.typeDisplayName,
    deviceName: `${task.categoryLabel}-演示设备-${String(index + 1).padStart(2, '0')}`,
    deviceSn: `${task.category.slice(0, 3).toUpperCase()}${String(index + 1).padStart(5, '0')}`,
    productType: task.productType ?? typeDef?.platformScope[0] ?? '待绑定平台',
    currentVersion: task.category.includes('upgrade') || task.category.includes('rollback') ? `V1.${index}.0` : 'baseline',
    targetVersion: task.targetVersion ?? typeDef?.fileNameTemplate ?? typeDef?.targetFileNameTemplate ?? typeDef?.fileTypeLabel ?? '按模板生成',
    status: index === 0 ? 'downloading' : index === 1 ? 'verifying' : task.status === 'ended' ? 'ended' : 'pending',
    result: index > 1 && task.status === 'ended' ? 'success' : undefined,
    progress: index === 0 ? 48 : index === 1 ? 76 : task.status === 'ended' ? 100 : 16,
    lastReportAt: new Date(Date.now() - (index + 1) * 18 * 60 * 1000).toISOString(),
    operatorScope: task.operatorScope,
  }));
}

let taskTypes: UnifiedFileTransferTaskType[] = [
  {
    typeCode: 'ENB_IMG_UPGRADE',
    category: 'enb_upgrade',
    categoryLabel: DEFAULT_CATEGORY_LABELS.enb_upgrade,
    displayName: '4G 基站软件升级',
    description: '面向 4G 基站 IMG 主包升级，保留现有 Download + TransferComplete 链路，并开放 file type 调整。',
    rpcType: 'DOWNLOAD',
    builtIn: true,
    enabled: true,
    stepChain: [
      'CHECK_PERMISSION',
      'CHECK_ONLINE',
      'CHECK_CONFLICT',
      'SEND_RPC',
      'WAIT_RPC_RESPONSE',
      'WAIT_FILE_TRANSFER',
      'WAIT_TRANSFER_COMPLETE',
    ],
    permissionCode: 'CODE_ENB_UPGRADE_IMAGE',
    platformScope: ['4G eNB', 'QAFA', 'QAFB'],
    fileType: '1',
    fileTypeLabel: 'Firmware Upgrade Image',
    fileTypeEditable: true,
    urlTemplate: 'firmware/{minio_path}',
    targetFileNameTemplate: '{firmware_name}',
    fileNameTemplate: '{firmware_name}',
    fileSizeField: 'firmware.fileSize',
    checksumField: 'firmware.md5',
    rawMode: 'false',
    delaySeconds: 0,
    transportPath: '/smallcell/FileDownloadService/firmware/img/{path}',
    lastEditor: 'system-seed',
    taskCount30d: 128,
    successRate30d: 96,
    updatedAt: new Date(now - 2 * 60 * 60 * 1000).toISOString(),
  },
  {
    typeCode: 'ENB_PATCH_UPGRADE',
    category: 'enb_upgrade',
    categoryLabel: DEFAULT_CATEGORY_LABELS.enb_upgrade,
    displayName: '4G Patch 增量升级',
    description: '用于展示同一业务 tab 内多模板并存的场景，file type 可按厂商协议单独调整。',
    rpcType: 'DOWNLOAD',
    builtIn: true,
    enabled: true,
    stepChain: [
      'CHECK_PERMISSION',
      'CHECK_ONLINE',
      'CHECK_CONFLICT',
      'SEND_RPC',
      'WAIT_RPC_RESPONSE',
      'WAIT_FILE_TRANSFER',
      'WAIT_TRANSFER_COMPLETE',
    ],
    permissionCode: 'CODE_ENB_UPGRADE_PATCH',
    platformScope: ['4G eNB', 'Patch'],
    fileType: '1',
    fileTypeLabel: 'Patch Package',
    fileTypeEditable: true,
    urlTemplate: 'firmware/{patch_path}',
    targetFileNameTemplate: '{patch_name}',
    fileNameTemplate: '{patch_name}',
    fileSizeField: 'firmware.fileSize',
    checksumField: 'firmware.md5',
    rawMode: 'true',
    delaySeconds: 0,
    transportPath: '/smallcell/FileDownloadService/firmware/patch/{path}',
    lastEditor: 'system-seed',
    taskCount30d: 42,
    successRate30d: 94,
    updatedAt: new Date(now - 3 * 60 * 60 * 1000).toISOString(),
  },
  {
    typeCode: 'GNB_IMG_UPGRADE',
    category: 'gnb_upgrade',
    categoryLabel: DEFAULT_CATEGORY_LABELS.gnb_upgrade,
    displayName: '5G 基站软件升级',
    description: '在 TransferComplete 之后继续等待 102 UPGRADE FINISH，更适合验证配置化事件等待能力。',
    rpcType: 'DOWNLOAD',
    builtIn: true,
    enabled: true,
    stepChain: [
      'CHECK_PERMISSION',
      'CHECK_ONLINE',
      'CHECK_CONFLICT',
      'SEND_RPC',
      'WAIT_RPC_RESPONSE',
      'WAIT_FILE_TRANSFER',
      'WAIT_TRANSFER_COMPLETE',
      'WAIT_INFORM_EVENT',
    ],
    postTcEventCode: '102 UPGRADE FINISH',
    permissionCode: 'CODE_GNB_UPGRADE_IMAGE',
    platformScope: ['5G gNB', 'BBU-XSS', 'BBU-QSS'],
    fileType: '1',
    fileTypeLabel: 'Firmware Upgrade Image',
    fileTypeEditable: true,
    urlTemplate: 'firmware/{minio_path}',
    targetFileNameTemplate: '{firmware_name}',
    fileNameTemplate: '{firmware_name}',
    fileSizeField: 'firmware.fileSize',
    checksumField: 'firmware.md5',
    rawMode: 'false',
    delaySeconds: 0,
    transportPath: '/smallcell/FileDownloadService/firmware/img/{path}',
    lastEditor: 'system-seed',
    taskCount30d: 64,
    successRate30d: 92,
    updatedAt: new Date(now - 4 * 60 * 60 * 1000).toISOString(),
  },
  {
    // qa-614 c6 #365 #373：2G(GSM) 基站软件升级 mock 模板，使 mock 模式
    // 分类列表也出现『2G升级』，与真实后端 builtInTaskTypes() 的 GSM_IMG_UPGRADE 对齐。
    typeCode: 'GSM_IMG_UPGRADE',
    category: 'gsm_upgrade',
    categoryLabel: DEFAULT_CATEGORY_LABELS.gsm_upgrade,
    displayName: '2G 基站软件升级',
    description: '复用现网软件升级链路，统一承载 2G(GSM) 基站镜像升级任务。',
    rpcType: 'DOWNLOAD',
    builtIn: true,
    enabled: true,
    stepChain: [
      'CHECK_PERMISSION',
      'CHECK_ONLINE',
      'CHECK_CONFLICT',
      'SEND_RPC',
      'WAIT_RPC_RESPONSE',
      'WAIT_FILE_TRANSFER',
      'WAIT_TRANSFER_COMPLETE',
    ],
    permissionCode: 'CODE_GSM_UPGRADE_IMAGE',
    platformScope: ['2G BSC', '2G BTS', 'BSC', 'BTS', 'PGSM'],
    fileType: '1',
    fileTypeLabel: 'Firmware Upgrade Image',
    fileTypeEditable: true,
    urlTemplate: 'firmware/{minio_path}',
    targetFileNameTemplate: '{firmware_name}',
    fileNameTemplate: '{firmware_name}',
    fileSizeField: 'firmware.fileSize',
    checksumField: 'firmware.md5',
    rawMode: 'false',
    delaySeconds: 0,
    transportPath: '/smallcell/FileDownloadService/firmware/img/{path}',
    lastEditor: 'system-seed',
    taskCount30d: 12,
    successRate30d: 90,
    updatedAt: new Date(now - 5 * 60 * 60 * 1000).toISOString(),
  },
  {
    typeCode: 'UPS_AP_UPGRADE',
    category: 'ups_upgrade',
    categoryLabel: DEFAULT_CATEGORY_LABELS.ups_upgrade,
    displayName: 'UPS 软件升级',
    description: '复用 TR-069 Download + TransferComplete + 1 BOOT 版本确认链路，统一承载 UPS AP 固件升级任务。',
    rpcType: 'DOWNLOAD',
    builtIn: true,
    enabled: true,
    stepChain: [
      'CHECK_PERMISSION',
      'CHECK_ONLINE',
      'CHECK_CONFLICT',
      'SEND_RPC',
      'WAIT_RPC_RESPONSE',
      'WAIT_FILE_TRANSFER',
      'WAIT_TRANSFER_COMPLETE',
      'WAIT_REBOOT_COMPLETE',
    ],
    permissionCode: 'CODE_UPS_UPGRADE_IMAGE',
    platformScope: ['UPS'],
    products: ['UPS'],
    fileType: '1 Firmware Upgrade Image',
    fileTypeLabel: '1 Firmware Upgrade Image',
    fileTypeEditable: true,
    firmwareFileType: 5,
    urlTemplate: 'firmware/{ap_path}',
    targetFileNameTemplate: '{firmware_name}',
    fileNameTemplate: '{firmware_name}',
    fileSizeField: 'firmware.fileSize',
    checksumField: 'firmware.md5',
    rawMode: 'false',
    delaySeconds: 0,
    transportPath: '/smallcell/FileDownloadService/firmware/ap/{path}',
    lastEditor: 'system-seed',
    taskCount30d: 0,
    successRate30d: 0,
    updatedAt: new Date(now - 3 * 60 * 60 * 1000).toISOString(),
  },
  {
    typeCode: 'GNB_FPGA_UPGRADE',
    category: 'gnb_upgrade',
    categoryLabel: DEFAULT_CATEGORY_LABELS.gnb_upgrade,
    displayName: '5G FPGA 升级',
    description: '用于展示 5G 升级里不同包类型模板的管理方式，支持独立 file type 与事件等待配置。',
    rpcType: 'DOWNLOAD',
    builtIn: true,
    enabled: true,
    stepChain: [
      'CHECK_PERMISSION',
      'CHECK_ONLINE',
      'CHECK_CONFLICT',
      'SEND_RPC',
      'WAIT_RPC_RESPONSE',
      'WAIT_FILE_TRANSFER',
      'WAIT_TRANSFER_COMPLETE',
      'WAIT_INFORM_EVENT',
    ],
    postTcEventCode: '102 UPGRADE FINISH',
    permissionCode: 'CODE_GNB_UPGRADE_FPGA',
    platformScope: ['5G gNB', 'FPGA'],
    fileType: '1',
    fileTypeLabel: 'FPGA Package',
    fileTypeEditable: true,
    urlTemplate: 'firmware/{fpga_path}',
    targetFileNameTemplate: '{fpga_name}',
    fileNameTemplate: '{fpga_name}',
    fileSizeField: 'firmware.fileSize',
    checksumField: 'firmware.md5',
    rawMode: 'false',
    delaySeconds: 0,
    transportPath: '/smallcell/FileDownloadService/firmware/fpga/{path}',
    lastEditor: 'system-seed',
    taskCount30d: 19,
    successRate30d: 91,
    updatedAt: new Date(now - 4 * 60 * 60 * 1000).toISOString(),
  },
  {
    typeCode: 'VERSION_ROLLBACK',
    category: 'version_rollback',
    categoryLabel: DEFAULT_CATEGORY_LABELS.version_rollback,
    displayName: '基站版本回退',
    description: '与升级共用 Download 主链路，通过 taskType 区分回退版本来源和审批范围。',
    rpcType: 'DOWNLOAD',
    builtIn: true,
    enabled: true,
    stepChain: [
      'CHECK_PERMISSION',
      'CHECK_ONLINE',
      'CHECK_CONFLICT',
      'PRE_VALIDATE',
      'SEND_RPC',
      'WAIT_RPC_RESPONSE',
      'WAIT_TRANSFER_COMPLETE',
    ],
    permissionCode: 'CODE_VERSION_ROLLBACK',
    platformScope: ['4G eNB', '5G gNB'],
    fileType: '1',
    fileTypeLabel: 'Rollback Image',
    fileTypeEditable: true,
    urlTemplate: 'firmware/rollback/{rollback_path}',
    targetFileNameTemplate: '{rollback_name}',
    fileNameTemplate: '{rollback_name}',
    fileSizeField: 'rollback.fileSize',
    checksumField: 'rollback.md5',
    rawMode: 'false',
    delaySeconds: 0,
    transportPath: '/smallcell/FileDownloadService/firmware/rollback/{path}',
    lastEditor: 'system-seed',
    taskCount30d: 11,
    successRate30d: 95,
    updatedAt: new Date(now - 5 * 60 * 60 * 1000).toISOString(),
  },
  {
    typeCode: 'RUNTIME_LOG_COLLECT',
    category: 'station_log',
    categoryLabel: DEFAULT_CATEGORY_LABELS.station_log,
    displayName: '运行日志采集',
    description: '复用 Upload + TransferComplete 回写，适合作为设备主动上传类任务的统一入口。',
    rpcType: 'UPLOAD',
    builtIn: true,
    enabled: true,
    stepChain: [
      'CHECK_PERMISSION',
      'CHECK_ONLINE',
      'PRE_VALIDATE',
      'SEND_RPC',
      'WAIT_RPC_RESPONSE',
      'WAIT_TRANSFER_COMPLETE',
    ],
    permissionCode: 'CODE_RUNTIME_LOG_COLLECT',
    platformScope: ['4G eNB', '5G gNB', 'DXDF'],
    fileType: '6',
    fileTypeLabel: '运行日志',
    fileTypeEditable: true,
    targetFileNameTemplate: 'runtime-{task_id8}-{sn}.tar.gz',
    delaySeconds: 0,
    transportPath: '/smallcell/FileUploadService?fileType={fileType}&filename={targetFileName}',
    lastEditor: 'system-seed',
    taskCount30d: 211,
    successRate30d: 89,
    updatedAt: new Date(now - 90 * 60 * 1000).toISOString(),
  },
  {
    typeCode: 'FAULT_LOG_COLLECT',
    category: 'station_log',
    categoryLabel: DEFAULT_CATEGORY_LABELS.station_log,
    displayName: '故障日志采集',
    description: '用于展示同一日志 tab 下不同 file type 模板的区分方式。',
    rpcType: 'UPLOAD',
    builtIn: true,
    enabled: true,
    stepChain: [
      'CHECK_PERMISSION',
      'CHECK_ONLINE',
      'PRE_VALIDATE',
      'SEND_RPC',
      'WAIT_RPC_RESPONSE',
      'WAIT_TRANSFER_COMPLETE',
    ],
    permissionCode: 'CODE_FAULT_LOG_COLLECT',
    platformScope: ['4G eNB', '5G gNB'],
    fileType: '8',
    fileTypeLabel: '故障日志',
    fileTypeEditable: true,
    targetFileNameTemplate: 'fault-{task_id8}-{sn}.zip',
    delaySeconds: 0,
    transportPath: '/smallcell/FileUploadService?fileType={fileType}&filename={targetFileName}',
    lastEditor: 'system-seed',
    taskCount30d: 73,
    successRate30d: 87,
    updatedAt: new Date(now - 75 * 60 * 1000).toISOString(),
  },
  {
    typeCode: 'CONFIG_BACKUP',
    category: 'config_backup',
    categoryLabel: DEFAULT_CATEGORY_LABELS.config_backup,
    displayName: '配置备份',
    description: '统一承载人工备份和定时备份场景，突出 Upload 类模板的 target_file_name 管理。',
    rpcType: 'UPLOAD',
    builtIn: true,
    enabled: true,
    stepChain: [
      'CHECK_PERMISSION',
      'CHECK_ONLINE',
      'CHECK_CONFLICT',
      'SEND_RPC',
      'WAIT_RPC_RESPONSE',
      'WAIT_TRANSFER_COMPLETE',
    ],
    permissionCode: 'CODE_CONFIG_BACKUP',
    platformScope: ['4G eNB', '5G gNB'],
    fileType: '3',
    fileTypeLabel: '配置备份文件',
    fileTypeEditable: true,
    targetFileNameTemplate: 'backup-{task_id8}-{sn}.xml',
    delaySeconds: 0,
    transportPath: '/smallcell/FileUploadService?fileType={fileType}&filename={targetFileName}',
    lastEditor: 'system-seed',
    taskCount30d: 37,
    successRate30d: 93,
    updatedAt: new Date(now - 60 * 60 * 1000).toISOString(),
  },
  {
    typeCode: 'CONFIG_RESTORE',
    category: 'config_restore',
    categoryLabel: DEFAULT_CATEGORY_LABELS.config_restore,
    displayName: '配置恢复',
    description: '恢复场景与升级明显不同：只保留 file type、URL 和目标文件名三类核心参数。',
    rpcType: 'DOWNLOAD',
    builtIn: true,
    enabled: true,
    stepChain: [
      'CHECK_PERMISSION',
      'CHECK_ONLINE',
      'PRE_VALIDATE',
      'SEND_RPC',
      'WAIT_RPC_RESPONSE',
      'WAIT_TRANSFER_COMPLETE',
    ],
    permissionCode: 'CODE_CONFIG_RESTORE',
    platformScope: ['4G eNB', '5G gNB'],
    fileType: '3',
    fileTypeLabel: '配置恢复文件',
    fileTypeEditable: true,
    urlTemplate: 'config_backup/{backup_path}',
    targetFileNameTemplate: '{backup_name}',
    delaySeconds: 0,
    transportPath: '/smallcell/FileDownloadService/config_backup/{path}',
    lastEditor: 'system-seed',
    taskCount30d: 18,
    successRate30d: 90,
    updatedAt: new Date(now - 40 * 60 * 1000).toISOString(),
  },
  {
    typeCode: 'CFG_ARCHIVE_PULL',
    category: 'config_backup',
    categoryLabel: DEFAULT_CATEGORY_LABELS.config_backup,
    displayName: '配置归档采集',
    description: '演示用自定义类型，展示后续如何把配置归档或诊断文件纳入同一任务模型，并允许直接编辑。',
    rpcType: 'UPLOAD',
    builtIn: false,
    enabled: true,
    stepChain: [
      'CHECK_PERMISSION',
      'CHECK_ONLINE',
      'CHECK_CONFLICT',
      'SEND_RPC',
      'WAIT_RPC_RESPONSE',
      'WAIT_TRANSFER_COMPLETE',
    ],
    permissionCode: 'CODE_CFG_ARCHIVE_PULL',
    platformScope: ['4G eNB', '5G gNB'],
    fileType: '3',
    fileTypeLabel: '归档配置文件',
    fileTypeEditable: true,
    targetFileNameTemplate: 'archive-{task_id8}-{sn}.cfg',
    delaySeconds: 0,
    transportPath: '/smallcell/FileUploadService?fileType={fileType}&filename={targetFileName}',
    lastEditor: 'product-demo',
    taskCount30d: 14,
    successRate30d: 100,
    updatedAt: new Date(now - 30 * 60 * 1000).toISOString(),
  },
];

let tasks: UnifiedFileTransferTask[] = [
  {
    id: 'ufte-task-001',
    taskName: '华北演示批次-4G升级',
    category: 'enb_upgrade',
    categoryLabel: DEFAULT_CATEGORY_LABELS.enb_upgrade,
    typeCode: 'ENB_IMG_UPGRADE',
    typeDisplayName: '4G 基站软件升级',
    status: 'in_progress',
    progress: 68,
    totalCount: 25,
    successCount: 16,
    failCount: 1,
    currentStep: 'WAIT_TRANSFER_COMPLETE',
    executionMode: 'immediate',
    createUser: 'admin',
    createdAt: new Date(now - 35 * 60 * 1000).toISOString(),
    operatorScope: 'CMCC-华北',
  },
  {
    id: 'ufte-task-002',
    taskName: '研发联调-5G升级验证',
    category: 'gnb_upgrade',
    categoryLabel: DEFAULT_CATEGORY_LABELS.gnb_upgrade,
    typeCode: 'GNB_IMG_UPGRADE',
    typeDisplayName: '5G 基站软件升级',
    status: 'in_progress',
    progress: 82,
    totalCount: 12,
    successCount: 9,
    failCount: 0,
    currentStep: 'WAIT_INFORM_EVENT',
    executionMode: 'scheduled',
    createUser: 'operator',
    createdAt: new Date(now - 90 * 60 * 1000).toISOString(),
    scheduledAt: new Date(now - 60 * 60 * 1000).toISOString(),
    operatorScope: 'CTCC-测试域',
  },
  {
    id: 'ufte-task-003',
    taskName: '疑难站点运行日志拉取',
    category: 'station_log',
    categoryLabel: DEFAULT_CATEGORY_LABELS.station_log,
    typeCode: 'RUNTIME_LOG_COLLECT',
    typeDisplayName: '运行日志采集',
    status: 'ended',
    result: 'success',
    progress: 100,
    totalCount: 8,
    successCount: 8,
    failCount: 0,
    currentStep: 'WAIT_TRANSFER_COMPLETE',
    executionMode: 'immediate',
    createUser: 'ops-admin',
    createdAt: new Date(now - 5 * 60 * 60 * 1000).toISOString(),
    operatorScope: 'CUCC-现网',
  },
  {
    id: 'ufte-task-004',
    taskName: '重点站版本回退演示',
    category: 'version_rollback',
    categoryLabel: DEFAULT_CATEGORY_LABELS.version_rollback,
    typeCode: 'VERSION_ROLLBACK',
    typeDisplayName: '基站版本回退',
    status: 'pending',
    progress: 12,
    totalCount: 16,
    successCount: 0,
    failCount: 0,
    currentStep: 'PRE_VALIDATE',
    executionMode: 'scheduled',
    createUser: 'release-admin',
    createdAt: new Date(now - 7 * 60 * 60 * 1000).toISOString(),
    scheduledAt: new Date(now + 30 * 60 * 1000).toISOString(),
    operatorScope: '回退保护窗',
  },
  {
    id: 'ufte-task-005',
    taskName: '周度配置归档演示',
    category: 'config_backup',
    categoryLabel: DEFAULT_CATEGORY_LABELS.config_backup,
    typeCode: 'CFG_ARCHIVE_PULL',
    typeDisplayName: '配置归档采集',
    status: 'suspended',
    progress: 24,
    totalCount: 40,
    successCount: 7,
    failCount: 0,
    currentStep: 'CHECK_CONFLICT',
    executionMode: 'suspended',
    createUser: 'product-demo',
    createdAt: new Date(now - 9 * 60 * 60 * 1000).toISOString(),
    operatorScope: '演示租户',
  },
  {
    id: 'ufte-task-006',
    taskName: '批量配置恢复验证',
    category: 'config_restore',
    categoryLabel: DEFAULT_CATEGORY_LABELS.config_restore,
    typeCode: 'CONFIG_RESTORE',
    typeDisplayName: '配置恢复',
    status: 'in_progress',
    progress: 54,
    totalCount: 20,
    successCount: 10,
    failCount: 1,
    currentStep: 'WAIT_TRANSFER_COMPLETE',
    executionMode: 'immediate',
    createUser: 'backup-admin',
    createdAt: new Date(now - 2 * 60 * 60 * 1000).toISOString(),
    operatorScope: '配置恢复试点',
  },
];

let deviceItems: UnifiedFileTransferDeviceItem[] = [
  {
    id: 'ufte-device-001',
    taskId: 'ufte-task-001',
    taskName: '华北演示批次-4G升级',
    category: 'enb_upgrade',
    categoryLabel: DEFAULT_CATEGORY_LABELS.enb_upgrade,
    typeCode: 'ENB_IMG_UPGRADE',
    typeDisplayName: '4G 基站软件升级',
    deviceName: '华北-eNB-0001',
    deviceSn: 'ENB00001',
    productType: 'QAFA',
    currentVersion: 'V1.0.2',
    targetVersion: 'IMG_2026_05_A',
    status: 'downloading',
    progress: 55,
    lastReportAt: new Date(now - 20 * 60 * 1000).toISOString(),
    operatorScope: 'CMCC-华北',
  },
  {
    id: 'ufte-device-002',
    taskId: 'ufte-task-001',
    taskName: '华北演示批次-4G升级',
    category: 'enb_upgrade',
    categoryLabel: DEFAULT_CATEGORY_LABELS.enb_upgrade,
    typeCode: 'ENB_IMG_UPGRADE',
    typeDisplayName: '4G 基站软件升级',
    deviceName: '华北-eNB-0002',
    deviceSn: 'ENB00002',
    productType: 'QAFB',
    currentVersion: 'V1.0.2',
    targetVersion: 'IMG_2026_05_A',
    status: 'verifying',
    progress: 86,
    lastReportAt: new Date(now - 12 * 60 * 1000).toISOString(),
    operatorScope: 'CMCC-华北',
  },
  {
    id: 'ufte-device-003',
    taskId: 'ufte-task-002',
    taskName: '研发联调-5G升级验证',
    category: 'gnb_upgrade',
    categoryLabel: DEFAULT_CATEGORY_LABELS.gnb_upgrade,
    typeCode: 'GNB_IMG_UPGRADE',
    typeDisplayName: '5G 基站软件升级',
    deviceName: '研发-gNB-0101',
    deviceSn: 'GNB00101',
    productType: 'BBU-XSS',
    currentVersion: 'V2.1.7',
    targetVersion: 'NR_IMG_2026_05',
    status: 'verifying',
    progress: 79,
    lastReportAt: new Date(now - 16 * 60 * 1000).toISOString(),
    operatorScope: 'CTCC-测试域',
  },
  {
    id: 'ufte-device-004',
    taskId: 'ufte-task-003',
    taskName: '疑难站点运行日志拉取',
    category: 'station_log',
    categoryLabel: DEFAULT_CATEGORY_LABELS.station_log,
    typeCode: 'RUNTIME_LOG_COLLECT',
    typeDisplayName: '运行日志采集',
    deviceName: '现网-eNB-2203',
    deviceSn: 'ENB02203',
    productType: 'DXDF',
    currentVersion: 'runtime',
    targetVersion: 'runtime-{task_id8}-{sn}.tar.gz',
    status: 'ended',
    result: 'success',
    progress: 100,
    lastReportAt: new Date(now - 4 * 60 * 60 * 1000).toISOString(),
    operatorScope: 'CUCC-现网',
  },
  {
    id: 'ufte-device-005',
    taskId: 'ufte-task-004',
    taskName: '重点站版本回退演示',
    category: 'version_rollback',
    categoryLabel: DEFAULT_CATEGORY_LABELS.version_rollback,
    typeCode: 'VERSION_ROLLBACK',
    typeDisplayName: '基站版本回退',
    deviceName: '重点站-gNB-0312',
    deviceSn: 'GNB00312',
    productType: 'BBU-QSS',
    currentVersion: 'V2.3.1',
    targetVersion: 'Rollback_V2.2.8',
    status: 'pending',
    progress: 18,
    lastReportAt: new Date(now - 28 * 60 * 1000).toISOString(),
    operatorScope: '回退保护窗',
  },
  {
    id: 'ufte-device-006',
    taskId: 'ufte-task-005',
    taskName: '周度配置归档演示',
    category: 'config_backup',
    categoryLabel: DEFAULT_CATEGORY_LABELS.config_backup,
    typeCode: 'CFG_ARCHIVE_PULL',
    typeDisplayName: '配置归档采集',
    deviceName: '演示-eNB-0418',
    deviceSn: 'ENB00418',
    productType: '4G eNB',
    currentVersion: 'baseline',
    targetVersion: 'archive-{task_id8}-{sn}.cfg',
    status: 'suspended',
    progress: 24,
    lastReportAt: new Date(now - 3 * 60 * 60 * 1000).toISOString(),
    operatorScope: '演示租户',
  },
  {
    id: 'ufte-device-007',
    taskId: 'ufte-task-006',
    taskName: '批量配置恢复验证',
    category: 'config_restore',
    categoryLabel: DEFAULT_CATEGORY_LABELS.config_restore,
    typeCode: 'CONFIG_RESTORE',
    typeDisplayName: '配置恢复',
    deviceName: '恢复-eNB-0501',
    deviceSn: 'ENB00501',
    productType: '5G gNB',
    currentVersion: 'restore-ready',
    targetVersion: 'backup-20260513.xml',
    status: 'downloading',
    progress: 61,
    lastReportAt: new Date(now - 25 * 60 * 1000).toISOString(),
    operatorScope: '配置恢复试点',
  },
];

function calcOverview(): UnifiedFileTransferOverview {
  const enabledTypeCount = taskTypes.filter((item) => item.enabled).length;
  const runningTaskCount = tasks.filter((item) => item.status === 'in_progress').length;
  const customTypeCount = taskTypes.filter((item) => !item.builtIn).length;
  const successRate30d = taskTypes.length === 0
    ? 0
    : Math.round(taskTypes.reduce((sum, item) => sum + item.successRate30d, 0) / taskTypes.length);
  return {
    enabledTypeCount,
    runningTaskCount,
    customTypeCount,
    successRate30d,
  };
}

function mapExecutionModeLabel(mode: TransferExecutionMode): string {
  switch (mode) {
    case 'scheduled':
      return '计划执行';
    case 'suspended':
      return '挂起创建';
    default:
      return '立即执行';
  }
}

export const unifiedFileTransferService = {
  async getOverview(): Promise<UnifiedFileTransferOverview> {
    await delay(80, 180);
    return calcOverview();
  },

  async getTaskTypes(): Promise<UnifiedFileTransferTaskType[]> {
    await delay(100, 220);
    return [...taskTypes].sort((left, right) => {
      if (left.category !== right.category) {
        return left.categoryLabel.localeCompare(right.categoryLabel, 'zh-CN');
      }
      if (left.builtIn !== right.builtIn) {
        return left.builtIn ? -1 : 1;
      }
      return left.displayName.localeCompare(right.displayName, 'zh-CN');
    });
  },

  async getTasks(
    params: { status?: string; typeCode?: string; keyword?: string; category?: string } & PageRequest,
  ): Promise<PageResponse<UnifiedFileTransferTask>> {
    await delay(120, 260);
    let filtered = [...tasks].sort((left, right) => right.createdAt.localeCompare(left.createdAt));
    if (params.category) {
      filtered = filtered.filter((item) => item.category === params.category);
    }
    if (params.status) {
      filtered = filtered.filter((item) => item.status === params.status);
    }
    if (params.typeCode) {
      filtered = filtered.filter((item) => item.typeCode === params.typeCode);
    }
    if (params.keyword) {
      const keyword = params.keyword.toLowerCase();
      filtered = filtered.filter((item) =>
        item.taskName.toLowerCase().includes(keyword) ||
        item.typeDisplayName.toLowerCase().includes(keyword) ||
        item.operatorScope.toLowerCase().includes(keyword),
      );
    }
    return paginate(filtered, params.page, params.pageSize);
  },

  async getDevices(
    params: { status?: string; typeCode?: string; keyword?: string; category?: string; productType?: string; taskId?: string } & PageRequest,
  ): Promise<PageResponse<UnifiedFileTransferDeviceItem>> {
    await delay(120, 260);
    let filtered = [...deviceItems].sort((left, right) => right.lastReportAt.localeCompare(left.lastReportAt));
    // #615：mock 同步真实后端按 taskId 收窄设备列表（任务详情抽屉「已选设备」用）。
    if (params.taskId) {
      filtered = filtered.filter((item) => item.taskId === params.taskId);
    }
    if (params.category) {
      filtered = filtered.filter((item) => item.category === params.category);
    }
    if (params.status) {
      filtered = filtered.filter((item) => item.status === params.status);
    }
    if (params.typeCode) {
      filtered = filtered.filter((item) => item.typeCode === params.typeCode);
    }
    if (params.productType) {
      filtered = filtered.filter((item) => item.productType === params.productType);
    }
    if (params.keyword) {
      const keyword = params.keyword.toLowerCase();
      filtered = filtered.filter((item) =>
        item.taskName.toLowerCase().includes(keyword) ||
        item.deviceName.toLowerCase().includes(keyword) ||
        item.deviceSn.toLowerCase().includes(keyword) ||
        item.operatorScope.toLowerCase().includes(keyword),
      );
    }
    return paginate(filtered, params.page, params.pageSize);
  },

  async getDeviceCandidates(
    params: { keyword?: string; category?: string; typeCode?: string; productType?: string } & PageRequest,
  ): Promise<PageResponse<UnifiedFileTransferDeviceItem>> {
    await delay(120, 260);
    let filtered = [...deviceItems].sort((left, right) => right.lastReportAt.localeCompare(left.lastReportAt));
    if (params.category) {
      filtered = filtered.filter((item) => item.category === params.category);
    }
    if (params.typeCode) {
      filtered = filtered.filter((item) => item.typeCode === params.typeCode);
    }
    if (params.productType) {
      filtered = filtered.filter((item) => item.productType === params.productType);
    }
    if (params.keyword) {
      const keyword = params.keyword.toLowerCase();
      filtered = filtered.filter((item) =>
        item.taskName.toLowerCase().includes(keyword) ||
        item.deviceName.toLowerCase().includes(keyword) ||
        item.deviceSn.toLowerCase().includes(keyword) ||
        item.operatorScope.toLowerCase().includes(keyword),
      );
    }
    return paginate(filtered, params.page, params.pageSize);
  },

  async createTask(input: CreateUnifiedFileTransferTaskInput): Promise<UnifiedFileTransferTask> {
    await delay(120, 240);
    const typeDef = taskTypes.find((item) => item.typeCode === input.typeCode);
    const firmware = input.firmwareId ? await softwareService.getVersionById(input.firmwareId) : null;
    const created: UnifiedFileTransferTask = {
      id: generateId('ufte-task'),
      taskName: input.taskName,
      category: typeDef?.category ?? 'enb_upgrade',
      categoryLabel: typeDef?.categoryLabel ?? getCategoryLabel('enb_upgrade'),
      typeCode: input.typeCode,
      typeDisplayName: typeDef?.displayName ?? input.typeCode,
      firmwareId: firmware?.id,
      targetVersion: firmware?.versionCode ?? typeDef?.fileNameTemplate ?? typeDef?.targetFileNameTemplate ?? typeDef?.fileTypeLabel,
      productType: firmware?.deviceType ?? typeDef?.platformScope?.[0],
      status: input.executionMode === 'suspended' ? 'suspended' : 'pending',
      progress: input.executionMode === 'suspended' ? 0 : 6,
      totalCount: input.deviceCount,
      successCount: 0,
      failCount: 0,
      currentStep: input.executionMode === 'suspended' ? 'CHECK_PERMISSION' : 'CHECK_ONLINE',
      executionMode: input.executionMode,
      createUser: 'ui-preview',
      createdAt: new Date().toISOString(),
      scheduledAt: input.executionMode === 'scheduled'
        ? new Date(Date.now() + 2 * 60 * 60 * 1000).toISOString()
        : undefined,
      operatorScope: `演示任务 / ${mapExecutionModeLabel(input.executionMode)}`,
    };
    tasks = [created, ...tasks];
    deviceItems = [...createDevicePreviewItems(created, typeDef), ...deviceItems];
    return created;
  },

  async createTaskType(input: CreateUnifiedFileTransferTypeInput): Promise<UnifiedFileTransferTaskType> {
    await delay(120, 240);
    const prefix = input.category.replace(/[^a-zA-Z0-9]/g, '').toUpperCase() || createCategoryKey().toUpperCase();
    const typeCode = `CUSTOM_${prefix}_${Date.now().toString().slice(-6)}`;
    const created: UnifiedFileTransferTaskType = {
      typeCode,
      category: input.category,
      categoryLabel: input.categoryLabel,
      displayName: input.displayName,
      description: input.description,
      rpcType: input.rpcType,
      builtIn: false,
      enabled: input.enabled,
      stepChain: input.stepChain,
      postTcEventCode: input.postTcEventCode || undefined,
      permissionCode: `CODE_${typeCode}`,
      platformScope: input.platformScope,
      fileType: input.fileType,
      fileTypeLabel: input.fileTypeLabel,
      fileTypeEditable: input.fileTypeEditable,
      urlTemplate: input.urlTemplate || undefined,
      targetFileNameTemplate: input.targetFileNameTemplate || undefined,
      fileNameTemplate: input.fileNameTemplate || undefined,
      fileSizeField: input.fileSizeField || undefined,
      checksumField: input.checksumField || undefined,
      rawMode: input.rawMode || undefined,
      delaySeconds: input.delaySeconds,
      transportPath: input.transportPath || undefined,
      lastEditor: 'ui-preview',
      taskCount30d: 0,
      successRate30d: 0,
      updatedAt: new Date().toISOString(),
    };
    taskTypes = [created, ...taskTypes];
    return created;
  },

  async updateTaskType(input: UpdateUnifiedFileTransferTaskTypeInput): Promise<UnifiedFileTransferTaskType> {
    await delay(120, 240);
    const index = taskTypes.findIndex((item) => item.typeCode === input.typeCode);
    if (index === -1) {
      throw new Error('Task type not found');
    }
    const previous = taskTypes[index];
    const updated: UnifiedFileTransferTaskType = {
      ...previous,
      category: input.category,
      categoryLabel: input.categoryLabel,
      displayName: input.displayName,
      description: input.description,
      rpcType: input.rpcType,
      enabled: input.enabled,
      stepChain: input.stepChain,
      postTcEventCode: input.postTcEventCode || undefined,
      platformScope: input.platformScope,
      fileType: input.fileType,
      fileTypeLabel: input.fileTypeLabel,
      fileTypeEditable: input.fileTypeEditable,
      urlTemplate: input.urlTemplate || undefined,
      targetFileNameTemplate: input.targetFileNameTemplate || undefined,
      fileNameTemplate: input.fileNameTemplate || undefined,
      fileSizeField: input.fileSizeField || undefined,
      checksumField: input.checksumField || undefined,
      rawMode: input.rawMode || undefined,
      delaySeconds: input.delaySeconds,
      transportPath: input.transportPath || undefined,
      lastEditor: 'ui-preview',
      updatedAt: new Date().toISOString(),
    };
    taskTypes = taskTypes.map((item) => item.typeCode === input.typeCode ? updated : item);
    tasks = tasks.map((item) => item.typeCode === input.typeCode
      ? {
          ...item,
          category: updated.category,
          categoryLabel: updated.categoryLabel,
          typeDisplayName: updated.displayName,
        }
      : item);
    deviceItems = deviceItems.map((item) => item.typeCode === input.typeCode
      ? {
          ...item,
          category: updated.category,
          categoryLabel: updated.categoryLabel,
          typeDisplayName: updated.displayName,
          targetVersion: updated.fileNameTemplate ?? updated.targetFileNameTemplate ?? updated.fileTypeLabel,
        }
      : item);
    return updated;
  },

  async deleteTaskType(typeCode: string): Promise<void> {
    await delay(80, 160);
    taskTypes = taskTypes.filter((item) => item.typeCode !== typeCode);
  },

  async startTask(id: string): Promise<void> {
    await delay(80, 160);
    tasks = tasks.map((item) => item.id === id
      ? { ...item, status: 'in_progress', currentStep: 'CHECK_ONLINE' }
      : item);
  },

  async suspendTask(id: string): Promise<void> {
    await delay(80, 160);
    tasks = tasks.map((item) => item.id === id
      ? { ...item, status: 'suspended' }
      : item);
  },

  async terminateTask(id: string): Promise<void> {
    await delay(80, 160);
    tasks = tasks.map((item) => item.id === id
      ? { ...item, status: 'ended', result: 'terminated' }
      : item);
  },

  async deleteTask(id: string): Promise<void> {
    await delay(80, 160);
    tasks = tasks.filter((item) => item.id !== id);
    deviceItems = deviceItems.filter((item) => item.taskId !== id);
  },

  async batchDeleteTasks(taskIds: string[]): Promise<{
    succeeded: string[];
    failed: Array<{ taskId: string; error: string }>;
  }> {
    await delay(120, 240);
    const idSet = new Set(taskIds);
    const succeeded: string[] = [];
    const failed: Array<{ taskId: string; error: string }> = [];
    for (const id of taskIds) {
      if (tasks.some((item) => item.id === id)) {
        succeeded.push(id);
      } else {
        failed.push({ taskId: id, error: '任务不存在' });
      }
    }
    tasks = tasks.filter((item) => !idSet.has(item.id));
    deviceItems = deviceItems.filter((item) => !idSet.has(item.taskId));
    return { succeeded, failed };
  },

  async retryTask(id: string): Promise<void> {
    await delay(80, 160);
    tasks = tasks.map((item) => item.id === id
      ? { ...item, status: 'in_progress', result: undefined, failCount: 0, currentStep: 'CHECK_ONLINE' }
      : item);
  },
};
