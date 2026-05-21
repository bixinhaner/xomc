import { useEffect, useMemo, useRef, useState } from 'react';
import dayjs from 'dayjs';
import {
  Button,
  Card,
  Checkbox,
  Descriptions,
  Drawer,
  Form,
  Input,
  Popconfirm,
  Progress,
  Radio,
  Select,
  Space,
  Table,
  Tag,
  Tabs,
  Tooltip,
  Typography,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { EyeOutlined, PlusOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { useT } from '@/hooks/useT';

import ListPageLayout from '@/components/Layout/ListPageLayout';
import {
  useCreateUnifiedFileTransferTask,
  useDeleteUfteTask,
  useStartUfteTask,
  useSuspendUfteTask,
  useTerminateUfteTask,
  useUnifiedFileTransferDeviceCandidates,
  useUnifiedFileTransferDevices,
  useUnifiedFileTransferTasks,
  useUnifiedFileTransferTaskTypes,
} from '@core/hooks/api/useUnifiedFileTransfer';
import { useProductClasses } from '@core/hooks/api/useDevices';
import { useSoftwareVersions } from '@core/hooks/api/useSoftware';
import { useUserStore } from '@core/store/userStore';
import type { SoftwareVersion } from '@core/mock/data/software';
import type {
  CreateUnifiedFileTransferTaskInput,
  FirmwareLibraryFileType,
  UnifiedFileTransferDeviceItem,
  UnifiedFileTransferTask,
  UnifiedFileTransferTaskType,
} from '@core/types/unifiedFileTransfer';
import {
  buildCategoryTabs,
  EXECUTION_MODE_OPTIONS,
  getSoftwareLibraryFileTypeLabel,
  renderDeviceStatus,
  renderTaskStatus,
  STEP_LABELS,
  UPGRADE_LIKE_CATEGORIES,
} from '../shared';
import type { TransferStepId } from '@core/types/unifiedFileTransfer';

const { Text, Title } = Typography;

// TASK_NAME_PREFIX_BY_TYPE: UFTE 创建任务时默认 taskName 的业务前缀。
// 命名规则跟升级模块惯例对齐（Upgrade_admin_2026-05-21 05:33:55）：
//   ${prefix}_${username}_${YYYY-MM-DD HH:mm:ss}
// 用户可在表单里编辑覆盖；切换业务类型时若用户没改过，会自动按新业务重算。
const TASK_NAME_PREFIX_BY_TYPE: Record<string, string> = {
  ENB_IMG_UPGRADE: 'Upgrade',
  GNB_IMG_UPGRADE: 'Upgrade',
  ENB_PATCH_UPGRADE: 'Upgrade',
  ENB_FPGA_UPGRADE: 'Upgrade',
  VERSION_ROLLBACK: 'Rollback',
  CONFIG_BACKUP_NV: 'ConfigBackupNV',
  CONFIG_BACKUP_XML: 'ConfigBackupXML',
  CONFIG_RESTORE: 'ConfigRestore',
  RUNTIME_LOG_COLLECT: 'RuntimeLog',
  FAULT_LOG_COLLECT: 'FaultLog',
};

function buildDefaultTaskName(typeCode: string | undefined, username: string | undefined): string {
  const prefix = (typeCode && TASK_NAME_PREFIX_BY_TYPE[typeCode]) || 'Task';
  const user = (username && username.trim()) || 'user';
  const ts = dayjs().format('YYYY-MM-DD HH:mm:ss');
  return `${prefix}_${user}_${ts}`;
}

function isUpgradeTaskCategory(category?: string) {
  return category === 'gnb_upgrade' || category === 'enb_upgrade';
}

function matchesScope(scope: string[], _category: string, productClass: string): boolean {
  const upper = productClass.toUpperCase();
  for (const s of scope) {
    const su = s.toUpperCase();
    if (upper === su || upper.includes(su) || su.includes(upper)) return true;
    if (su.includes(' ')) {
      const tokens = su.split(/\s+/).filter((t) => t.length >= 2);
      for (const token of tokens) {
        if (upper.includes(token)) return true;
      }
    }
  }
  return false;
}

function splitDeviceTypes(deviceType?: string) {
  return (deviceType ?? '')
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean);
}

function needsFirmwareSelection(taskType?: UnifiedFileTransferTaskType) {
  return Boolean(taskType && taskType.rpcType === 'DOWNLOAD' && isUpgradeTaskCategory(taskType.category));
}

function buildFirmwareCandidateList(versions: SoftwareVersion[]) {
  const sorted = [...versions].sort((left, right) => {
    if (left.recommend !== right.recommend) {
      return left.recommend ? -1 : 1;
    }
    if (left.status !== right.status) {
      if (left.status === 'current') return -1;
      if (right.status === 'current') return 1;
    }
    return right.releaseDate.localeCompare(left.releaseDate);
  });

  return sorted;
}

function resolveFirmwareLibraryFileType(taskType?: UnifiedFileTransferTaskType): FirmwareLibraryFileType | undefined {
  if (taskType?.firmwareFileType !== undefined) {
    return taskType.firmwareFileType;
  }
  const haystack = `${taskType?.typeCode ?? ''} ${taskType?.displayName ?? ''} ${taskType?.fileType ?? ''} ${taskType?.fileTypeLabel ?? ''}`.toLowerCase();
  if (haystack.includes('fpga')) {
    return 6;
  }
  if (haystack.includes('patch')) {
    return 1;
  }
  if (taskType && isUpgradeTaskCategory(taskType.category)) {
    return 0;
  }
  return undefined;
}

function getUpgradeTypeLabel(category: string, fallback: string) {
  if (category === 'gnb_upgrade' || category === 'enb_upgrade') {
    return '软件升级';
  }
  if (category === 'version_rollback') {
    return '版本回退';
  }
  return fallback;
}

export default function FileTransferCenter() {
  const navigate = useNavigate();
  const t = useT();
  // 任务名称自动填充用：取登录用户名拼前缀，displayName / username 哪个有用哪个。
  const currentUser = useUserStore((s) => s.currentUser);
  const taskNameUser = currentUser?.username || currentUser?.displayName || 'user';
  // lastAutoFilledTaskNameRef 记录最近一次自动填的名字。用户在表单里手动改过 → ref
  // 跟 form 值不再一致 → typeCode 切换时不覆盖；用户没改 → 切换业务时跟着刷新。
  const lastAutoFilledTaskNameRef = useRef<string>('');
  const { data: taskTypes = [], isLoading: taskTypesLoading } = useUnifiedFileTransferTaskTypes();
  const { data: productClasses = [] } = useProductClasses();
  const categories = useMemo(() => buildCategoryTabs(taskTypes), [taskTypes]);
  const [selectedCategory, setSelectedCategory] = useState('');
  const [selectedTypeCode, setSelectedTypeCode] = useState('');
  const [taskPage, setTaskPage] = useState(1);
  const [taskPageSize, setTaskPageSize] = useState(10);
  const [taskKeyword, setTaskKeyword] = useState('');
  const [taskKeywordInput, setTaskKeywordInput] = useState('');
  const [taskStatusFilter, setTaskStatusFilter] = useState<string>();
  const [devicePage, setDevicePage] = useState(1);
  const [devicePageSize, setDevicePageSize] = useState(10);
  const [deviceKeyword, setDeviceKeyword] = useState('');
  const [deviceKeywordInput, setDeviceKeywordInput] = useState('');
  const [deviceStatusFilter, setDeviceStatusFilter] = useState<string>();
  const [deviceProductTypeFilter, setDeviceProductTypeFilter] = useState<string>();
  const [viewMode, setViewMode] = useState<'tasks' | 'devices'>('tasks');
  const [taskDrawerOpen, setTaskDrawerOpen] = useState(false);
  const [taskForm] = Form.useForm<CreateUnifiedFileTransferTaskInput>();
  const drawerTypeCode = Form.useWatch('typeCode', taskForm);
  const drawerProductType = Form.useWatch('productType', taskForm);
  const [selectedDrawerDeviceIds, setSelectedDrawerDeviceIds] = useState<string[]>([]);
  const [drawerDeviceKeyword, setDrawerDeviceKeyword] = useState('');
  const [drawerDeviceKeywordInput, setDrawerDeviceKeywordInput] = useState('');
  const [detailTask, setDetailTask] = useState<UnifiedFileTransferTask | null>(null);
  const [detailDrawerOpen, setDetailDrawerOpen] = useState(false);

  const { data: tasksData, isLoading: tasksLoading } = useUnifiedFileTransferTasks({
    page: taskPage,
    pageSize: taskPageSize,
    category: selectedCategory || undefined,
    keyword: taskKeyword || undefined,
    status: taskStatusFilter,
    typeCode: selectedTypeCode || undefined,
  });

  const { data: devicesData, isLoading: devicesLoading } = useUnifiedFileTransferDevices({
    page: devicePage,
    pageSize: devicePageSize,
    category: selectedCategory || undefined,
    keyword: deviceKeyword || undefined,
    status: deviceStatusFilter,
    typeCode: selectedTypeCode || undefined,
    productType: deviceProductTypeFilter,
  });

  const createTaskMutation = useCreateUnifiedFileTransferTask();
  const startTaskMutation = useStartUfteTask();
  const suspendTaskMutation = useSuspendUfteTask();
  const terminateTaskMutation = useTerminateUfteTask();
  const deleteTaskMutation = useDeleteUfteTask();
  const recentTasks = tasksData?.items ?? [];
  const recentDevices = devicesData?.items ?? [];

  const filteredTaskTypes = useMemo(
    () => taskTypes.filter((item) => item.category === selectedCategory),
    [selectedCategory, taskTypes],
  );

  const taskTypeOptions = useMemo(
    () => filteredTaskTypes.map((item) => ({ label: item.displayName, value: item.typeCode })),
    [filteredTaskTypes],
  );

  const activeTaskType = useMemo(
    () => filteredTaskTypes.find((item) => item.typeCode === selectedTypeCode) ?? filteredTaskTypes[0],
    [filteredTaskTypes, selectedTypeCode],
  );

  const drawerTaskType = useMemo(
    () => taskTypes.find((item) => item.typeCode === drawerTypeCode) ?? activeTaskType,
    [activeTaskType, drawerTypeCode, taskTypes],
  );
  const firmwareLibraryFileType = resolveFirmwareLibraryFileType(drawerTaskType);
  const { data: firmwareData } = useSoftwareVersions({
    page: 1,
    pageSize: 200,
    fileType: firmwareLibraryFileType,
  });

  const createExecutionModeOptions = useMemo(
    () => EXECUTION_MODE_OPTIONS.filter((item) => item.value !== 'scheduled'),
    [],
  );

  const { data: drawerDevicesData, isLoading: drawerDevicesLoading } = useUnifiedFileTransferDeviceCandidates({
    page: 1,
    pageSize: 200,
    category: selectedCategory || undefined,
    typeCode: drawerTaskType?.typeCode || selectedTypeCode || undefined,
    productType: needsFirmwareSelection(drawerTaskType) ? drawerProductType : undefined,
    keyword: drawerDeviceKeyword || undefined,
  });

  const drawerDeviceCandidates = drawerDevicesData?.items ?? [];

  const firmwareCandidates = useMemo(
    () => buildFirmwareCandidateList(firmwareData?.items ?? []),
    [firmwareData?.items],
  );

  const drawerProductTypeOptions = useMemo(() => {
    const scope = drawerTaskType?.platformScope ?? [];
    const category = drawerTaskType?.category ?? '';
    const realClasses = new Set<string>();
    productClasses.forEach((pc) => {
      if (scope.length === 0 || matchesScope(scope, category, pc)) {
        realClasses.add(pc);
      }
    });
    firmwareCandidates.forEach((item) => {
      splitDeviceTypes(item.deviceType).forEach((entry) => {
        if (scope.length === 0 || matchesScope(scope, category, entry)) {
          realClasses.add(entry);
        }
      });
    });
    return Array.from(realClasses).map((item) => ({ label: item, value: item }));
  }, [drawerTaskType, firmwareCandidates, productClasses]);

  const filteredFirmwareCandidates = useMemo(
    () => (drawerProductType
      ? firmwareCandidates.filter((item) => {
          const deviceTypes = splitDeviceTypes(item.deviceType);
          return deviceTypes.length === 0 || deviceTypes.includes(drawerProductType);
        })
      : []),
    [drawerProductType, firmwareCandidates],
  );

  const firmwareOptions = useMemo(
    () => filteredFirmwareCandidates.map((item) => ({
      label: item.fileName || item.versionCode,
      value: item.id,
    })),
    [filteredFirmwareCandidates],
  );

  const deviceProductTypeOptions = useMemo(() => {
    const values = new Set<string>(productClasses);
    recentDevices.forEach((item) => {
      if (item.productType) {
        values.add(item.productType);
      }
    });
    (activeTaskType?.platformScope ?? []).forEach((entry) => values.add(entry));
    return Array.from(values).map((item) => ({ label: item, value: item }));
  }, [activeTaskType?.platformScope, productClasses, recentDevices]);

  const templateTabItems = useMemo(
    () => filteredTaskTypes.map((item) => ({
      key: item.typeCode,
      label: (
        <Space size={6}>
          <span>{item.displayName}</span>
          <Tag color={item.builtIn ? 'blue' : 'gold'}>{item.builtIn ? '内置' : '自定义'}</Tag>
        </Space>
      ),
    })),
    [filteredTaskTypes],
  );

  const drawerSelectedDevices = useMemo(
    () => drawerDeviceCandidates.filter((item) => selectedDrawerDeviceIds.includes(item.id)),
    [drawerDeviceCandidates, selectedDrawerDeviceIds],
  );

  const drawerDeviceColumns: ColumnsType<UnifiedFileTransferDeviceItem> = useMemo(
    () => [
      { title: '设备 SN', dataIndex: 'deviceSn', key: 'deviceSn', width: 160 },
      { title: '站点名称', dataIndex: 'deviceName', key: 'deviceName', ellipsis: true },
      { title: '产品类型', dataIndex: 'productType', key: 'productType', width: 120 },
      { title: '当前版本', dataIndex: 'currentVersion', key: 'currentVersion', width: 120 },
    ],
    [],
  );

  // Failure reason i18n — mirrors UpgradePlan renderFailureReason
  const DEVICE_CODES = new Set(['DOWNLOAD_FAULT', 'TC_FAULT', 'UPGRADE_5G_FAILED']);
  const TIMEOUT_CODES = new Set(['DOWNLOAD_TIMEOUT', 'TASK_TIMEOUT']);

  const failureReasonColumn = {
    title: t('software.failureReason'),
    dataIndex: 'failureReason',
    key: 'failureReason',
    width: 260,
    render: (value: string, record: UnifiedFileTransferDeviceItem) => {
      if (!value) return '-';
      const i18nLabel = t(`software.failureCode.${value}` as Parameters<typeof t>[0]);
      const display = i18nLabel && i18nLabel !== `software.failureCode.${value}` ? i18nLabel : value;
      // 设备厂商原始 fault（FaultCode + FaultString）放 Tooltip 里——i18n label 只看到统一
      // 错误码描述，hover 后能拿到设备端原文（如 "Upgrade failed, there is FaultString in
      // TransferComplete msg. FaultCode: 0, FaultString: httpUpload OM Http Put Upload stat
      // file error"），方便厂商侧排查。
      const detail = record.failureDetail;
      const text = <span style={{ color: '#ff4d4f' }}>{display}</span>;
      if (!detail || detail === display) return text;
      return (
        <Tooltip title={detail} placement="topLeft" overlayStyle={{ maxWidth: 480 }}>
          {text}
        </Tooltip>
      );
    },
  };

  const getTaskActionErrorMessage = (error: unknown, fallback: string) => {
    if (error instanceof Error && error.message.trim().length > 0) {
      return error.message;
    }
    return fallback;
  };

  const handleStartTask = (record: UnifiedFileTransferTask) => {
    void startTaskMutation.mutateAsync(record.id)
      .then(() => void message.success('任务已启动'))
      .catch((error: unknown) => void message.error(getTaskActionErrorMessage(error, '任务启动失败')));
  };

  const handleSuspendTask = (record: UnifiedFileTransferTask) => {
    void suspendTaskMutation.mutateAsync(record.id)
      .then(() => void message.success('任务已暂停'))
      .catch((error: unknown) => void message.error(getTaskActionErrorMessage(error, '任务暂停失败')));
  };

  const handleTerminateTask = (record: UnifiedFileTransferTask) => {
    void terminateTaskMutation.mutateAsync(record.id)
      .then(() => void message.success('任务已终止'))
      .catch((error: unknown) => void message.error(getTaskActionErrorMessage(error, '任务终止失败')));
  };

  const handleDeleteTask = (record: UnifiedFileTransferTask) => {
    void deleteTaskMutation.mutateAsync(record.id)
      .then(() => void message.success('任务已删除'))
      .catch((error: unknown) => void message.error(getTaskActionErrorMessage(error, '任务删除失败')));
  };

  const openDetailDrawer = (record: UnifiedFileTransferTask) => {
    setDetailTask(record);
    setDetailDrawerOpen(true);
  };

  const renderTaskActions = (record: UnifiedFileTransferTask) => {
    const status = record.status;
    const showStart = status === 'pending' || status === 'suspended';
    const showSuspend = status === 'in_progress';
    const showTerminate = status !== 'ended';
    const showDelete = status === 'ended' || status === 'pending' || status === 'suspended';
    return (
      <Space size={4} wrap={false}>
        <Button
          type="link"
          size="small"
          icon={<EyeOutlined />}
          onClick={() => openDetailDrawer(record)}
          style={{ paddingInline: 0 }}
        >
          详情
        </Button>
        {showStart ? (
          <Button
            type="link"
            size="small"
            loading={startTaskMutation.isPending}
            onClick={() => handleStartTask(record)}
            style={{ paddingInline: 0 }}
          >
            开始
          </Button>
        ) : null}
        {showSuspend ? (
          <Button
            type="link"
            size="small"
            loading={suspendTaskMutation.isPending}
            onClick={() => handleSuspendTask(record)}
            style={{ paddingInline: 0 }}
          >
            暂停
          </Button>
        ) : null}
        {showTerminate ? (
          <Popconfirm title="确认终止该任务？" onConfirm={() => handleTerminateTask(record)}>
            <Button type="link" size="small" danger loading={terminateTaskMutation.isPending} style={{ paddingInline: 0 }}>
              终止
            </Button>
          </Popconfirm>
        ) : null}
        {showDelete ? (
          <Popconfirm title="确认删除该任务？" onConfirm={() => handleDeleteTask(record)}>
            <Button type="link" size="small" danger loading={deleteTaskMutation.isPending} style={{ paddingInline: 0 }}>
              删除
            </Button>
          </Popconfirm>
        ) : null}
      </Space>
    );
  };

  const taskActionColumn = {
    title: '操作',
    key: 'action',
    width: 190,
    align: 'center' as const,
    render: (_: unknown, record: UnifiedFileTransferTask) => renderTaskActions(record),
  };

  useEffect(() => {
    if (categories.length === 0) {
      setSelectedCategory('');
      return;
    }
    if (!categories.some((item) => item.category === selectedCategory)) {
      setSelectedCategory(categories[0].category);
    }
  }, [categories, selectedCategory]);

  useEffect(() => {
    const preferredType = filteredTaskTypes[0];
    if (!preferredType) {
      setSelectedTypeCode('');
      return;
    }
    if (!filteredTaskTypes.some((item) => item.typeCode === selectedTypeCode)) {
      setSelectedTypeCode(preferredType.typeCode);
    }
  }, [filteredTaskTypes, selectedTypeCode]);

  useEffect(() => {
    setTaskPage(1);
    setDevicePage(1);
  }, [selectedCategory, selectedTypeCode, taskKeyword, taskStatusFilter, deviceKeyword, deviceStatusFilter, deviceProductTypeFilter]);

  useEffect(() => {
    setDeviceProductTypeFilter(undefined);
  }, [selectedTypeCode]);

  const isUpgradeLikeCategory = UPGRADE_LIKE_CATEGORIES.has(selectedCategory);

  const getTypeDef = (typeCode: string) => taskTypes.find((item) => item.typeCode === typeCode);

  const getTaskTargetVersion = (record: UnifiedFileTransferTask) => {
    // 只有升级 / 回滚类才有"主任务目标版本"（固件版本号）；备份 / 日志采集 /
    // 配置恢复类后端返回空串，前端不再用 typeDef.fileNameTemplate 等模板字符串兜底
    // ——那只是占位符模板，对主任务来说没有意义。详见
    // docs/project/backup-display-fix-20260520.md F2。
    return record.targetVersion?.trim() ? record.targetVersion : '-';
  };

  const getTaskProductType = (record: UnifiedFileTransferTask) => {
    if (record.productType) {
      return record.productType;
    }
    const typeDef = getTypeDef(record.typeCode);
    return typeDef?.platformScope?.[0] || '-';
  };

  const taskColumns: ColumnsType<UnifiedFileTransferTask> = useMemo(() => {
    if (isUpgradeLikeCategory) {
      return [
        taskActionColumn,
        {
          // 任务名称：默认按 业务_用户_时间 自动生成（约 30-40 字符），固定 280 + Tooltip 兜底。
          title: '任务名称',
          dataIndex: 'taskName',
          key: 'taskName',
          width: 280,
          ellipsis: { showTitle: false },
          render: (_, record) => (
            <Tooltip title={record.taskName} placement="topLeft">
              <span>{record.taskName}</span>
            </Tooltip>
          ),
        },
        { title: '操作人', dataIndex: 'createUser', key: 'createUser', width: 110 },
        {
          title: '操作时间',
          dataIndex: 'createdAt',
          key: 'createdAt',
          width: 180,
          render: (value: string) => new Date(value).toLocaleString('zh-CN'),
        },
        {
          title: '状态',
          dataIndex: 'status',
          key: 'status',
          width: 110,
          render: (_, record) => renderTaskStatus(record.status),
        },
        {
          title: '目标版本',
          key: 'targetVersion',
          width: 150,
          render: (_, record) => getTaskTargetVersion(record),
        },
        {
          title: '升级类型',
          key: 'upgradeType',
          width: 120,
          render: (_, record) => <Tag color="blue">{getUpgradeTypeLabel(record.category, record.typeDisplayName)}</Tag>,
        },
        {
          title: '产品类型',
          key: 'productType',
          width: 120,
          render: (_, record) => getTaskProductType(record),
        },
        {
          title: '升级进度',
          dataIndex: 'progress',
          key: 'progress',
          width: 150,
          render: (value: number, record) => (
            <Progress percent={value} size="small" status={record.status === 'ended' ? 'success' : record.status === 'in_progress' ? 'active' : 'normal'} />
          ),
        },
        {
          title: '结果',
          dataIndex: 'result',
          key: 'result',
          width: 100,
          render: (value) => {
            if (!value) return '-';
            const color = value === 'success' ? 'success' : value === 'partial' ? 'warning' : 'error';
            const label = value === 'success' ? '成功' : value === 'partial' ? '部分成功' : value === 'terminated' ? '已终止' : '失败';
            return <Tag color={color}>{label}</Tag>;
          },
        },
        {
          title: '开始时间',
          key: 'startTime',
          width: 180,
          render: (_, record) => record.executionMode === 'scheduled' && record.scheduledAt ? new Date(record.scheduledAt).toLocaleString('zh-CN') : '-',
        },
        {
          title: '结束时间',
          key: 'endTime',
          width: 180,
          render: (_, record) => record.status === 'ended' ? new Date(record.createdAt).toLocaleString('zh-CN') : '-',
        },
      ];
    }

    return [
      taskActionColumn,
      {
        // 任务名称（非升级类）：默认按 业务_用户_时间 自动生成；副行展示业务类型。
        // 固定 280 + Tooltip 兜底，避免长名挤压后续列。
        title: '任务名称',
        dataIndex: 'taskName',
        key: 'taskName',
        width: 280,
        render: (_, record) => (
          <Space direction="vertical" size={2} style={{ maxWidth: '100%' }}>
            <Tooltip title={record.taskName} placement="topLeft">
              <Text strong ellipsis style={{ maxWidth: 260 }}>{record.taskName}</Text>
            </Tooltip>
            <Text type="secondary" ellipsis style={{ maxWidth: 260 }}>{record.typeDisplayName}</Text>
          </Space>
        ),
      },
      { title: '执行人', dataIndex: 'createUser', key: 'createUser', width: 110 },
      {
        title: '状态',
        dataIndex: 'status',
        key: 'status',
        width: 120,
        render: (_, record) => renderTaskStatus(record.status),
      },
      // 备份 / 日志采集 / 配置恢复 等 OUTPUT 文件类（非升级类）主任务跨多设备，没有
      // "当前步骤"概念——单设备的 RPC 步骤在设备列表展示。详见
      // docs/project/backup-display-fix-20260520.md F1。
      {
        title: '进度',
        dataIndex: 'progress',
        key: 'progress',
        width: 180,
        render: (value: number, record) => (
          <Space direction="vertical" size={4} style={{ width: '100%' }}>
            <Progress percent={value} size="small" status={record.status === 'ended' ? 'success' : 'active'} />
            <Text type="secondary">成功 {record.successCount} / 失败 {record.failCount} / 总数 {record.totalCount}</Text>
          </Space>
        ),
      },
      {
        title: '执行方式',
        dataIndex: 'executionMode',
        key: 'executionMode',
        width: 120,
        render: (value) => {
          const label = EXECUTION_MODE_OPTIONS.find((item) => item.value === value)?.label ?? value;
          return <Tag>{label}</Tag>;
        },
      },
      {
        title: '结果',
        dataIndex: 'result',
        key: 'result',
        width: 100,
        render: (value) => {
          if (!value) return '—';
          const color = value === 'success' ? 'success' : value === 'partial' ? 'warning' : 'error';
          const label = value === 'success' ? '成功' : value === 'partial' ? '部分成功' : value === 'terminated' ? '已终止' : '失败';
          return <Tag color={color}>{label}</Tag>;
        },
      },
      { title: '归属范围', dataIndex: 'operatorScope', key: 'operatorScope', width: 160 },
      {
        title: '创建时间',
        dataIndex: 'createdAt',
        key: 'createdAt',
        width: 180,
        render: (value: string) => new Date(value).toLocaleString('zh-CN'),
      },
    ];
  }, [isUpgradeLikeCategory, taskTypes]);

  const deviceColumns: ColumnsType<UnifiedFileTransferDeviceItem> = useMemo(() => {
    if (isUpgradeLikeCategory) {
      return [
        { title: '基站编码', dataIndex: 'deviceSn', key: 'deviceSn', width: 120 },
        { title: '任务名称', dataIndex: 'taskName', key: 'taskName', width: 180, ellipsis: true },
        { title: '源版本', dataIndex: 'currentVersion', key: 'currentVersion', width: 120 },
        { title: '目标版本', dataIndex: 'targetVersion', key: 'targetVersion', width: 140 },
        {
          title: '升级类型',
          key: 'upgradeType',
          width: 120,
          render: (_, record) => <Tag color="blue">{getUpgradeTypeLabel(record.category, record.typeDisplayName)}</Tag>,
        },
        { title: '产品类型', dataIndex: 'productType', key: 'productType', width: 120 },
        {
          title: '升级进度',
          dataIndex: 'progress',
          key: 'progress',
          width: 130,
          render: (value: number, record) => {
            let progressStatus: 'success' | 'exception' | 'active' | 'normal' = 'normal';
            if (record.status === 'ended') progressStatus = 'success';
            else if (record.status === 'failed') progressStatus = 'exception';
            else if (record.status === 'pending' || record.status === 'suspended') progressStatus = 'normal';
            else progressStatus = 'active';
            return <Progress percent={value} size="small" status={progressStatus} />;
          },
        },
        {
          title: '结果',
          dataIndex: 'status',
          key: 'status',
          width: 110,
          render: (_, record) => renderDeviceStatus(record.status),
        },
        { title: '操作人', dataIndex: 'operatorScope', key: 'operatorScope', width: 120, render: (value: string) => value || '-' },
        failureReasonColumn,
        {
          title: '操作时间',
          dataIndex: 'lastReportAt',
          key: 'lastReportAt',
          width: 180,
          render: (value: string) => new Date(value).toLocaleString('zh-CN'),
        },
      ];
    }

    return [
      {
        // 任务列表场景：每行是「某任务在某设备上的执行情况」。主显示任务名（用户操作
        // 上下文，他刚创建的 testNV 想看这个任务的进展），副显示设备名（区分多设备）。
        // 列宽 280：UFTE 自动生成的任务名形如 "ConfigBackupNV_admin_2026-05-21 13:39:55"
        // 约 36 字符，hover Tooltip 看全名。
        title: '任务/设备',
        dataIndex: 'taskName',
        key: 'taskName',
        width: 280,
        render: (_, record) => (
          <Space direction="vertical" size={2} style={{ maxWidth: '100%' }}>
            <Tooltip title={record.taskName} placement="topLeft">
              <Text strong ellipsis style={{ maxWidth: 260 }}>{record.taskName}</Text>
            </Tooltip>
            <Text type="secondary" ellipsis style={{ maxWidth: 260 }}>{record.deviceName}</Text>
          </Space>
        ),
      },
      { title: '设备 SN', dataIndex: 'deviceSn', key: 'deviceSn', width: 120 },
      { title: '产品类型', dataIndex: 'productType', key: 'productType', width: 130 },
      { title: '当前版本', dataIndex: 'currentVersion', key: 'currentVersion', width: 130 },
      {
        // 非升级类（备份 / 日志采集 / 配置恢复）：使用后端按 {task_id8}/{sn} 渲染过的
        // targetFile（出现"backup-a1b2c3d4-SN001.nv"形式）。CPE 上传完成且 metadata
        // 落地后，后端附带 downloadUrl，UI 渲染为可点击链接。详见
        // docs/project/backup-display-fix-20260520.md F3。
        title: '目标版本/目标文件',
        key: 'targetVersion',
        width: 220,
        render: (_, record) => {
          const file = record.targetFile?.trim();
          if (!file) {
            return '-';
          }
          if (record.downloadUrl) {
            return (
              <a href={record.downloadUrl} target="_blank" rel="noopener noreferrer">
                {file}
              </a>
            );
          }
          return <Text type="secondary">{file}</Text>;
        },
      },
      {
        title: '状态',
        dataIndex: 'status',
        key: 'status',
        width: 110,
        render: (_, record) => renderDeviceStatus(record.status),
      },
      { title: '进度', dataIndex: 'progress', key: 'progress', width: 180, render: (value: number) => <Progress percent={value} size="small" status={value === 100 ? 'success' : 'active'} /> },
      failureReasonColumn,
      {
        title: '上报时间',
        dataIndex: 'lastReportAt',
        key: 'lastReportAt',
        width: 180,
        render: (value: string) => new Date(value).toLocaleString('zh-CN'),
      },
    ];
  }, [isUpgradeLikeCategory]);

  const openTaskDrawer = (typeCode?: string) => {
    const nextTypeCode = typeCode || selectedTypeCode || activeTaskType?.typeCode;
    if (!nextTypeCode) {
      void message.warning('当前业务视图下还没有模板，请联系管理员先维护模板。');
      return;
    }
    const defaultTaskName = buildDefaultTaskName(nextTypeCode, taskNameUser);
    lastAutoFilledTaskNameRef.current = defaultTaskName;
    taskForm.setFieldsValue({
      taskName: defaultTaskName,
      typeCode: nextTypeCode,
      productType: undefined,
      firmwareId: undefined,
      isKeepConfig: true,
      executionMode: 'immediate',
    });
    setSelectedDrawerDeviceIds([]);
    setDrawerDeviceKeyword('');
    setDrawerDeviceKeywordInput('');
    setSelectedTypeCode(nextTypeCode);
    setTaskDrawerOpen(true);
  };

  // 切换"任务类型"时，如用户没动过 taskName（当前值 === 上次自动填的值）→ 重算填入；
  // 已被用户修改过 → 保留用户输入不打扰。
  useEffect(() => {
    if (!taskDrawerOpen || !drawerTypeCode) {
      return;
    }
    const currentTaskName = (taskForm.getFieldValue('taskName') as string | undefined) ?? '';
    if (currentTaskName && currentTaskName !== lastAutoFilledTaskNameRef.current) {
      return;
    }
    const nextName = buildDefaultTaskName(drawerTypeCode, taskNameUser);
    lastAutoFilledTaskNameRef.current = nextName;
    taskForm.setFieldValue('taskName', nextName);
  }, [drawerTypeCode, taskDrawerOpen, taskForm, taskNameUser]);

  useEffect(() => {
    if (!taskDrawerOpen) {
      return;
    }
    if (!needsFirmwareSelection(drawerTaskType)) {
      taskForm.setFieldValue('productType', undefined);
      taskForm.setFieldValue('firmwareId', undefined);
      taskForm.setFieldValue('isKeepConfig', undefined);
    } else {
      if (taskForm.getFieldValue('isKeepConfig') === undefined) {
        taskForm.setFieldValue('isKeepConfig', true);
      }
      if (!drawerProductType && drawerProductTypeOptions.length === 1) {
        taskForm.setFieldValue('productType', drawerProductTypeOptions[0].value);
        return;
      }
      const selectedFirmwareId = taskForm.getFieldValue('firmwareId') as string | undefined;
      if (selectedFirmwareId && !firmwareOptions.some((item) => item.value === selectedFirmwareId)) {
        taskForm.setFieldValue('firmwareId', undefined);
      }
    }

    const validIds = selectedDrawerDeviceIds.filter((id) => drawerDeviceCandidates.some((item) => item.id === id));
    if (validIds.length !== selectedDrawerDeviceIds.length) {
      setSelectedDrawerDeviceIds(validIds);
    }
  }, [drawerDeviceCandidates, drawerProductType, drawerProductTypeOptions, drawerTaskType, firmwareOptions, selectedDrawerDeviceIds, taskDrawerOpen, taskForm]);

  const handleCreateTask = async () => {
    // 防止用户连续点击「创建」按钮重复提交：mutation 进行中直接忽略后续点击。
    // 仅靠按钮 loading 不够 —— validateFields 是异步的，期间 isPending 仍为 false。
    if (createTaskMutation.isPending) {
      return;
    }
    const values = await taskForm.validateFields();
    if (selectedDrawerDeviceIds.length === 0) {
      void message.warning('请选择设备。');
      return;
    }
    if (createTaskMutation.isPending) {
      return;
    }
    void createTaskMutation
      .mutateAsync({
        ...values,
        deviceIds: selectedDrawerDeviceIds,
        deviceCount: selectedDrawerDeviceIds.length,
      })
      .then(() => {
        void message.success('任务已创建。');
        setTaskDrawerOpen(false);
        setSelectedDrawerDeviceIds([]);
        taskForm.resetFields();
      })
      .catch((error: unknown) => {
        void message.error(getTaskActionErrorMessage(error, '创建任务失败，请稍后重试'));
      });
  };

  return (
    <ListPageLayout
      title="任务创建"
      extra={(
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => openTaskDrawer()}
          disabled={taskTypesLoading || !activeTaskType}
        >
          新建任务
        </Button>
      )}
    >
      <Space direction="vertical" size={16} style={{ width: '100%' }}>
        <Card>
          <Space direction="vertical" size={12} style={{ width: '100%' }}>
            <Title level={4} style={{ margin: 0 }}>任务创建</Title>
            <Tabs
              activeKey={selectedCategory}
              items={categories.map((item) => ({ key: item.category, label: item.categoryLabel }))}
              onChange={(key) => setSelectedCategory(key)}
            />
            <Tabs
              activeKey={selectedTypeCode}
              items={templateTabItems}
              onChange={(key) => setSelectedTypeCode(key)}
            />
          </Space>
        </Card>

        <Card title="执行视图">
          <Tabs
            activeKey={viewMode}
            onChange={(key) => setViewMode(key as 'tasks' | 'devices')}
            items={[
              {
                key: 'tasks',
                label: '任务列表',
                children: (
                  <Space direction="vertical" size={12} style={{ width: '100%' }}>
                    <Space wrap>
                      <Input.Search
                        allowClear
                        placeholder="按任务名称、类型、归属范围搜索"
                        value={taskKeywordInput}
                        onChange={(event) => setTaskKeywordInput(event.target.value)}
                        onSearch={(value) => setTaskKeyword(value.trim())}
                        style={{ width: 280 }}
                      />
                      <Select
                        value={selectedTypeCode}
                        onChange={(value) => setSelectedTypeCode(value)}
                        options={taskTypeOptions}
                        style={{ width: 260 }}
                      />
                      <Select
                        allowClear
                        placeholder="按状态过滤"
                        value={taskStatusFilter}
                        onChange={(value) => setTaskStatusFilter(value)}
                        options={[
                          { label: '待执行', value: 'pending' },
                          { label: '执行中', value: 'in_progress' },
                          { label: '已挂起', value: 'suspended' },
                          { label: '已结束', value: 'ended' },
                        ]}
                        style={{ width: 160 }}
                      />
                    </Space>
                    <Table<UnifiedFileTransferTask>
                      rowKey="id"
                      columns={taskColumns}
                      dataSource={recentTasks}
                      loading={tasksLoading}
                      pagination={{
                        current: taskPage,
                        pageSize: taskPageSize,
                        total: tasksData?.total ?? 0,
                        showSizeChanger: true,
                        showTotal: (total) => `共 ${total} 条`,
                        onChange: (page, pageSize) => {
                          setTaskPage(page);
                          setTaskPageSize(pageSize);
                        },
                      }}
                      scroll={{ x: 1800 }}
                    />
                  </Space>
                ),
              },
              {
                key: 'devices',
                label: '设备列表',
                children: (
                  <Space direction="vertical" size={12} style={{ width: '100%' }}>
                    <Space wrap>
                      <Input.Search
                        allowClear
                        placeholder="按设备名称、SN、任务名称搜索"
                        value={deviceKeywordInput}
                        onChange={(event) => setDeviceKeywordInput(event.target.value)}
                        onSearch={(value) => setDeviceKeyword(value.trim())}
                        style={{ width: 280 }}
                      />
                      <Select
                        value={selectedTypeCode}
                        onChange={(value) => setSelectedTypeCode(value)}
                        options={taskTypeOptions}
                        style={{ width: 260 }}
                      />
                      <Select
                        allowClear
                        showSearch
                        placeholder="按产品类型过滤"
                        value={deviceProductTypeFilter}
                        onChange={(value) => setDeviceProductTypeFilter(value)}
                        options={deviceProductTypeOptions}
                        optionFilterProp="label"
                        style={{ width: 220 }}
                      />
                      <Select
                        allowClear
                        placeholder="按状态过滤"
                        value={deviceStatusFilter}
                        onChange={(value) => setDeviceStatusFilter(value)}
                        options={[
                          { label: '待执行', value: 'pending' },
                          // 升级 / 回滚类（Download RPC）
                          { label: '下载中', value: 'downloading' },
                          // 备份 / 日志采集类（Upload RPC）的两个子阶段
                          { label: '上传中', value: 'uploading' },
                          { label: '等待 TransferComplete', value: 'awaiting_tc' },
                          { label: '校验中', value: 'verifying' },
                          { label: '已挂起', value: 'suspended' },
                          { label: '已完成', value: 'ended' },
                          { label: '失败', value: 'failed' },
                        ]}
                        style={{ width: 160 }}
                      />
                    </Space>
                    <Table<UnifiedFileTransferDeviceItem>
                      rowKey="id"
                      columns={deviceColumns}
                      dataSource={recentDevices}
                      loading={devicesLoading}
                      pagination={{
                        current: devicePage,
                        pageSize: devicePageSize,
                        total: devicesData?.total ?? 0,
                        showSizeChanger: true,
                        showTotal: (total) => `共 ${total} 条`,
                        onChange: (page, pageSize) => {
                          setDevicePage(page);
                          setDevicePageSize(pageSize);
                        },
                      }}
                      scroll={{ x: 1800 }}
                    />
                  </Space>
                ),
              },
            ]}
          />
        </Card>
      </Space>

      <Drawer
        title="新建任务"
        width={520}
        open={taskDrawerOpen}
        onClose={() => setTaskDrawerOpen(false)}
        destroyOnClose
        extra={(
          <Space>
            <Button onClick={() => setTaskDrawerOpen(false)}>取消</Button>
            <Button type="primary" loading={createTaskMutation.isPending} onClick={() => void handleCreateTask()}>
              创建
            </Button>
          </Space>
        )}
      >
        <Form form={taskForm} layout="vertical">
          <Form.Item label="任务名称" name="taskName" rules={[{ required: true, message: '请输入任务名称' }]}>
            <Input placeholder="例如：Upgrade_admin_2026-05-21 05:33:55（默认按业务_用户_时间生成，可改）" />
          </Form.Item>
          <Form.Item label="任务类型" name="typeCode" rules={[{ required: true, message: '请选择任务类型' }]}> 
            <Select
              options={taskTypeOptions}
              placeholder="请选择任务类型"
              disabled={taskTypeOptions.length <= 1}
            />
          </Form.Item>
          {needsFirmwareSelection(drawerTaskType) ? (
            <>
              <Form.Item label="产品类型" name="productType" rules={[{ required: true, message: '请选择产品类型' }]}> 
                <Select
                  allowClear
                  showSearch
                  placeholder="请选择产品类型"
                  options={drawerProductTypeOptions}
                  optionFilterProp="label"
                />
              </Form.Item>
              <Form.Item
                label="升级文件"
                name="firmwareId"
                rules={[{ required: true, message: '请选择升级文件' }]}
                extra={(
                  <Space direction="vertical" size={0}>
                    <Text type="secondary">当前模板查询的软件库分类：{getSoftwareLibraryFileTypeLabel(firmwareLibraryFileType)}</Text>
                    <Button type="link" style={{ paddingInline: 0 }} onClick={() => navigate('/software/firmware')}>
                      维护升级文件
                    </Button>
                  </Space>
                )}
              >
                <Select
                  showSearch
                  allowClear
                  disabled={!drawerProductType}
                  placeholder={
                    !drawerProductType
                      ? '请先选择产品类型'
                      : firmwareOptions.length > 0
                        ? '请选择升级文件'
                        : `暂无可用${getSoftwareLibraryFileTypeLabel(firmwareLibraryFileType)}，请先到软件管理对应分类上传`
                  }
                  options={firmwareOptions}
                  optionFilterProp="label"
                />
              </Form.Item>
              <Form.Item name="isKeepConfig" valuePropName="checked">
                <Checkbox>保留配置</Checkbox>
              </Form.Item>
            </>
          ) : null}
          <Form.Item label="选择设备" required>
            <Space direction="vertical" size={12} style={{ width: '100%' }}>
              <Space>
                <Input.Search
                  placeholder="按 SN 搜索设备"
                  allowClear
                  style={{ width: 240 }}
                  value={drawerDeviceKeywordInput}
                  onChange={(e) => setDrawerDeviceKeywordInput(e.target.value)}
                  onSearch={(value) => setDrawerDeviceKeyword(value.trim())}
                />
                <Text type="secondary">已选 {drawerSelectedDevices.length} 台</Text>
              </Space>
              <Table<UnifiedFileTransferDeviceItem>
                size="small"
                rowKey="id"
                loading={drawerDevicesLoading}
                columns={drawerDeviceColumns}
                dataSource={drawerDeviceCandidates}
                pagination={false}
                rowSelection={{
                  selectedRowKeys: selectedDrawerDeviceIds,
                  onChange: (selectedRowKeys) => {
                    setSelectedDrawerDeviceIds(selectedRowKeys.map((item) => String(item)));
                  },
                }}
                scroll={{ x: 600, y: 240 }}
              />
            </Space>
          </Form.Item>
          <Form.Item label="执行方式" name="executionMode" rules={[{ required: true, message: '请选择执行方式' }]}> 
            <Radio.Group options={createExecutionModeOptions} optionType="button" buttonStyle="solid" />
          </Form.Item>
          <Form.Item label="备注" name="note">
            <Input.TextArea rows={4} placeholder="可填写灰度范围、验证目标或领导评审备注" />
          </Form.Item>
        </Form>
      </Drawer>

      <Drawer
        title={detailTask ? `任务详情 · ${detailTask.taskName}` : '任务详情'}
        width={640}
        open={detailDrawerOpen}
        onClose={() => { setDetailDrawerOpen(false); setDetailTask(null); }}
        destroyOnClose
      >
        {detailTask ? (
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            <Descriptions column={2} size="small" bordered>
              <Descriptions.Item label="任务名称">{detailTask.taskName}</Descriptions.Item>
              <Descriptions.Item label="任务类型">{detailTask.typeDisplayName}</Descriptions.Item>
              <Descriptions.Item label="状态">{renderTaskStatus(detailTask.status)}</Descriptions.Item>
              <Descriptions.Item label="结果">
                {detailTask.result
                  ? <Tag color={detailTask.result === 'success' ? 'success' : detailTask.result === 'partial' ? 'warning' : 'error'}>{detailTask.result === 'success' ? '成功' : detailTask.result === 'partial' ? '部分成功' : detailTask.result === 'terminated' ? '已终止' : '失败'}</Tag>
                  : '-'}
              </Descriptions.Item>
              <Descriptions.Item label="目标版本">{getTaskTargetVersion(detailTask)}</Descriptions.Item>
              <Descriptions.Item label="产品类型">{getTaskProductType(detailTask)}</Descriptions.Item>
              <Descriptions.Item label="执行方式">
                <Tag>{EXECUTION_MODE_OPTIONS.find((o) => o.value === detailTask.executionMode)?.label ?? detailTask.executionMode}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="当前步骤">{STEP_LABELS[detailTask.currentStep as TransferStepId] ?? '-'}</Descriptions.Item>
              <Descriptions.Item label="操作人">{detailTask.createUser}</Descriptions.Item>
              <Descriptions.Item label="创建时间">{new Date(detailTask.createdAt).toLocaleString('zh-CN')}</Descriptions.Item>
            </Descriptions>
            <Card title="执行进度" size="small">
              <Space direction="vertical" size={12} style={{ width: '100%' }}>
                <Progress
                  type="circle"
                  percent={detailTask.progress}
                  format={() => `${detailTask.progress}%`}
                />
                <Space wrap>
                  <Tag color="success">成功 {detailTask.successCount}</Tag>
                  <Tag color="error">失败 {detailTask.failCount}</Tag>
                  <Tag>总数 {detailTask.totalCount}</Tag>
                </Space>
              </Space>
            </Card>
          </Space>
        ) : null}
      </Drawer>
    </ListPageLayout>
  );
}