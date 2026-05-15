import { useEffect, useMemo, useState } from 'react';
import {
  Button,
  Card,
  Checkbox,
  Drawer,
  Form,
  Input,
  Progress,
  Radio,
  Select,
  Space,
  Table,
  Tag,
  Tabs,
  Typography,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { PlusOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';

import ListPageLayout from '@/components/Layout/ListPageLayout';
import {
  useCreateUnifiedFileTransferTask,
  useUnifiedFileTransferDeviceCandidates,
  useUnifiedFileTransferDevices,
  useUnifiedFileTransferTasks,
  useUnifiedFileTransferTaskTypes,
} from '@core/hooks/api/useUnifiedFileTransfer';
import { useSoftwareVersions } from '@core/hooks/api/useSoftware';
import type { SoftwareVersion } from '@core/mock/data/software';
import type {
  CreateUnifiedFileTransferTaskInput,
  UnifiedFileTransferDeviceItem,
  UnifiedFileTransferTask,
  UnifiedFileTransferTaskType,
} from '@core/types/unifiedFileTransfer';
import {
  buildCategoryTabs,
  EXECUTION_MODE_OPTIONS,
  renderDeviceStatus,
  renderTaskStatus,
  STEP_LABELS,
  UPGRADE_LIKE_CATEGORIES,
} from '../shared';

const { Text, Title } = Typography;

function isUpgradeTaskCategory(category?: string) {
  return category === 'gnb_upgrade' || category === 'enb_upgrade';
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

function buildFirmwareCandidateList(
  versions: SoftwareVersion[],
  taskType?: UnifiedFileTransferTaskType,
) {
  if (!taskType) {
    return versions;
  }

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

  const haystack = (version: SoftwareVersion) =>
    `${version.deviceType} ${version.versionName} ${version.versionCode} ${version.fileName}`.toLowerCase();

  const primaryKeyword = taskType.category === 'gnb_upgrade'
    ? 'gnb'
    : taskType.category === 'enb_upgrade'
      ? 'enb'
      : '';

  let filtered = sorted;
  if (primaryKeyword) {
    const matched = sorted.filter((item) => haystack(item).includes(primaryKeyword));
    if (matched.length > 0) {
      filtered = matched;
    }
  }

  const displayName = `${taskType.displayName} ${taskType.fileTypeLabel}`.toLowerCase();
  const subtypeKeyword = displayName.includes('fpga')
    ? 'fpga'
    : displayName.includes('patch')
      ? 'patch'
      : '';
  if (subtypeKeyword) {
    const matched = filtered.filter((item) => haystack(item).includes(subtypeKeyword));
    if (matched.length > 0) {
      filtered = matched;
    }
  }

  return filtered;
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
  const { data: taskTypes = [], isLoading: taskTypesLoading } = useUnifiedFileTransferTaskTypes();
  const { data: firmwareData } = useSoftwareVersions({ page: 1, pageSize: 200 });
  const categories = useMemo(() => buildCategoryTabs(taskTypes), [taskTypes]);
  const [selectedCategory, setSelectedCategory] = useState('gnb_upgrade');
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
  const drawerDeviceIds = Form.useWatch('deviceIds', taskForm) ?? [];

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
  });

  const drawerDeviceCandidates = drawerDevicesData?.items ?? [];

  const firmwareCandidates = useMemo(
    () => buildFirmwareCandidateList(firmwareData?.items ?? [], drawerTaskType),
    [drawerTaskType, firmwareData?.items],
  );

  const drawerProductTypeOptions = useMemo(() => {
    const values = new Set<string>();
    firmwareCandidates.forEach((item) => {
      splitDeviceTypes(item.deviceType).forEach((entry) => values.add(entry));
    });
    if (values.size === 0) {
      (drawerTaskType?.platformScope ?? []).forEach((entry) => values.add(entry));
    }
    return Array.from(values).map((item) => ({ label: item, value: item }));
  }, [drawerTaskType?.platformScope, firmwareCandidates]);

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
      label: `${item.versionCode} / ${item.deviceType || '未标识型号'} / ${item.fileName}`,
      value: item.id,
    })),
    [filteredFirmwareCandidates],
  );

  const deviceProductTypeOptions = useMemo(() => {
    const values = new Set<string>();
    recentDevices.forEach((item) => {
      if (item.productType) {
        values.add(item.productType);
      }
    });
    (activeTaskType?.platformScope ?? []).forEach((entry) => values.add(entry));
    return Array.from(values).map((item) => ({ label: item, value: item }));
  }, [activeTaskType?.platformScope, recentDevices]);

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
    () => drawerDeviceCandidates.filter((item) => drawerDeviceIds.includes(item.id)),
    [drawerDeviceCandidates, drawerDeviceIds],
  );

  const drawerDeviceColumns: ColumnsType<UnifiedFileTransferDeviceItem> = useMemo(
    () => [
      { title: '设备名称', dataIndex: 'deviceName', key: 'deviceName', ellipsis: true },
      { title: '设备 SN', dataIndex: 'deviceSn', key: 'deviceSn', width: 120 },
      { title: '产品类型', dataIndex: 'productType', key: 'productType', width: 120 },
      { title: '当前版本', dataIndex: 'currentVersion', key: 'currentVersion', width: 120 },
    ],
    [],
  );

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
    if (record.targetVersion) {
      return record.targetVersion;
    }
    const typeDef = getTypeDef(record.typeCode);
    return typeDef?.fileNameTemplate || typeDef?.targetFileNameTemplate || typeDef?.fileTypeLabel || '-';
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
        {
          title: '任务名称',
          dataIndex: 'taskName',
          key: 'taskName',
          width: 180,
          render: (_, record) => record.taskName,
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
      {
        title: '任务名称',
        dataIndex: 'taskName',
        key: 'taskName',
        render: (_, record) => (
          <Space direction="vertical" size={2}>
            <Text strong>{record.taskName}</Text>
            <Text type="secondary">{record.typeDisplayName}</Text>
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
      {
        title: '当前步骤',
        dataIndex: 'currentStep',
        key: 'currentStep',
        width: 170,
        render: (value) => STEP_LABELS[value],
      },
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
        title: '设备名称',
        dataIndex: 'deviceName',
        key: 'deviceName',
        render: (_, record) => (
          <Space direction="vertical" size={2}>
            <Text strong>{record.deviceName}</Text>
            <Text type="secondary">{record.taskName}</Text>
          </Space>
        ),
      },
      { title: '设备 SN', dataIndex: 'deviceSn', key: 'deviceSn', width: 120 },
      { title: '产品类型', dataIndex: 'productType', key: 'productType', width: 130 },
      { title: '当前版本', dataIndex: 'currentVersion', key: 'currentVersion', width: 130 },
      { title: '目标版本/目标文件', dataIndex: 'targetVersion', key: 'targetVersion', width: 180 },
      {
        title: '状态',
        dataIndex: 'status',
        key: 'status',
        width: 110,
        render: (_, record) => renderDeviceStatus(record.status),
      },
      { title: '进度', dataIndex: 'progress', key: 'progress', width: 180, render: (value: number) => <Progress percent={value} size="small" status={value === 100 ? 'success' : 'active'} /> },
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
    taskForm.setFieldsValue({
      typeCode: nextTypeCode,
      productType: undefined,
      firmwareId: undefined,
      isKeepConfig: true,
      deviceIds: [],
      executionMode: 'immediate',
      deviceCount: 0,
    });
    setSelectedTypeCode(nextTypeCode);
    setTaskDrawerOpen(true);
  };

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

    const validDeviceIds = drawerDeviceIds.filter((deviceId) => drawerDeviceCandidates.some((item) => item.id === deviceId));
    if (validDeviceIds.length !== drawerDeviceIds.length) {
      taskForm.setFieldValue('deviceIds', validDeviceIds);
      taskForm.setFieldValue('deviceCount', validDeviceIds.length);
      return;
    }
    if (taskForm.getFieldValue('deviceCount') !== drawerDeviceIds.length) {
      taskForm.setFieldValue('deviceCount', drawerDeviceIds.length);
    }
  }, [drawerDeviceCandidates, drawerDeviceIds, drawerProductType, drawerProductTypeOptions, drawerTaskType, firmwareOptions, taskDrawerOpen, taskForm]);

  const handleCreateTask = async () => {
    const values = await taskForm.validateFields();
    const selectedDeviceIds = values.deviceIds ?? [];
    if (selectedDeviceIds.length === 0) {
      void message.warning('请选择设备。');
      return;
    }
    await createTaskMutation.mutateAsync({
      ...values,
      deviceCount: selectedDeviceIds.length,
    });
    void message.success('演示任务已创建。');
    setTaskDrawerOpen(false);
    taskForm.resetFields();
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
                      scroll={{ x: 1600 }}
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
                        placeholder="按产品类型过滤"
                        value={deviceProductTypeFilter}
                        onChange={(value) => setDeviceProductTypeFilter(value)}
                        options={deviceProductTypeOptions}
                        style={{ width: 220 }}
                      />
                      <Select
                        allowClear
                        placeholder="按状态过滤"
                        value={deviceStatusFilter}
                        onChange={(value) => setDeviceStatusFilter(value)}
                        options={[
                          { label: '待执行', value: 'pending' },
                          { label: '下载中', value: 'downloading' },
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
                      scroll={{ x: 1600 }}
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
            <Input placeholder="例如：华东试点-统一入口演示" />
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
                  placeholder="请选择产品类型"
                  options={drawerProductTypeOptions}
                />
              </Form.Item>
              <Form.Item
                label="升级文件"
                name="firmwareId"
                rules={[{ required: true, message: '请选择升级文件' }]}
                extra={(
                  <Button type="link" style={{ paddingInline: 0 }} onClick={() => navigate('/software/firmware')}>
                    维护升级文件
                  </Button>
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
                        : '暂无可用升级文件，请先到软件管理上传'
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
              <Text type="secondary">已选 {drawerSelectedDevices.length} 台</Text>
              <Table<UnifiedFileTransferDeviceItem>
                size="small"
                rowKey="id"
                loading={drawerDevicesLoading}
                columns={drawerDeviceColumns}
                dataSource={drawerDeviceCandidates}
                pagination={false}
                rowSelection={{
                  selectedRowKeys: drawerDeviceIds,
                  onChange: (selectedRowKeys) => {
                    const nextIds = selectedRowKeys.map((item) => String(item));
                    taskForm.setFieldValue('deviceIds', nextIds);
                    taskForm.setFieldValue('deviceCount', nextIds.length);
                  },
                }}
                scroll={{ y: 220 }}
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
    </ListPageLayout>
  );
}