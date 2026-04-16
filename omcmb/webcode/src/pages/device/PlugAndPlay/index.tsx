import React, { useState, useMemo, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { Button, Space, Tag, Switch, Dropdown, Input, App, Typography, Card, Select } from 'antd';
import type { MenuProps } from 'antd';
import {
  PlusOutlined,
  SearchOutlined,
  EditOutlined,
  DeleteOutlined,
  ScanOutlined,
  RedoOutlined,
  PlayCircleOutlined,
  MoreOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  LoadingOutlined,
  ClockCircleOutlined,
  ForwardOutlined,
} from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { DataTableColumn, BatchAction } from '@/components/DataTable';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useT } from '@/hooks/useT';
import DetectDialog from './components/DetectDialog';
import ExecuteDetailPanel from './components/ExecuteDetailPanel';
import BatchRetryDialog from './components/BatchRetryDialog';

const { Text } = Typography;

// Types
type ExecuteType = '0' | '1'; // 0-auto, 1-manual
type TaskStatus = '0' | '1' | '2' | '3' | '4'; // 0-success, 1-fail, 2-running, 3-pending, 4-skip

interface Policy {
  policyId: string;
  policyName: string;
  productType: string;
  executeType: ExecuteType;
  selfStartEnable: '0' | '1';
  upgradeEnable: '0' | '1';
  targetVersion: string[];
  licenseEnable: '0' | '1';
  selfConfigEnable: '0' | '1';
  createTime: string;
  updateTime: string;
}

interface ExecuteTask {
  taskId: string;
  serialNumber: string;
  productType: string;
  policyName: string;
  executeType: ExecuteType;
  startTime: string;
  endTime: string;
  executeProcedure: string;
  status: TaskStatus;
  failureReason: string;
  policyId: string;
  originalVersion: string;
  targetVersion: string;
  licenseFile: string;
}

// Product types
const PRODUCT_TYPES = [
  { label: 'QAFA', value: 'QAFA' },
  { label: 'QAFB', value: 'QAFB' },
  { label: 'QAFC', value: 'QAFC' },
  { label: 'CPE-A100', value: 'CPE-A100' },
  { label: 'CPE-B200', value: 'CPE-B200' },
];

// Mock data - combined policies
const MOCK_POLICIES: Policy[] = [
  {
    policyId: 'pnp-1',
    policyName: 'QAFA自动开通策略',
    productType: 'QAFA',
    executeType: '0',
    selfStartEnable: '1',
    upgradeEnable: '1',
    targetVersion: ['V2.1.0'],
    licenseEnable: '1',
    selfConfigEnable: '1',
    createTime: '2026-03-15 10:00:00',
    updateTime: '2026-04-01 14:30:00',
  },
  {
    policyId: 'pnp-2',
    policyName: 'QAFB手动升级策略',
    productType: 'QAFB',
    executeType: '1',
    selfStartEnable: '0',
    upgradeEnable: '1',
    targetVersion: ['V2.2.0'],
    licenseEnable: '0',
    selfConfigEnable: '1',
    createTime: '2026-03-20 09:00:00',
    updateTime: '2026-03-25 11:00:00',
  },
  {
    policyId: 'pnp-3',
    policyName: 'License更新策略',
    productType: 'QAFA',
    executeType: '0',
    selfStartEnable: '1',
    upgradeEnable: '0',
    targetVersion: [],
    licenseEnable: '1',
    selfConfigEnable: '0',
    createTime: '2026-04-01 08:00:00',
    updateTime: '2026-04-01 08:00:00',
  },
  {
    policyId: 'pnp-4',
    policyName: 'CPE自动开通策略',
    productType: 'CPE-A100',
    executeType: '0',
    selfStartEnable: '1',
    upgradeEnable: '1',
    targetVersion: ['V1.5.0'],
    licenseEnable: '0',
    selfConfigEnable: '1',
    createTime: '2026-03-10 10:00:00',
    updateTime: '2026-03-15 14:00:00',
  },
  {
    policyId: 'pnp-5',
    policyName: 'QAFC全量开通策略',
    productType: 'QAFC',
    executeType: '0',
    selfStartEnable: '1',
    upgradeEnable: '1',
    targetVersion: ['V3.0.0'],
    licenseEnable: '1',
    selfConfigEnable: '1',
    createTime: '2026-03-18 09:30:00',
    updateTime: '2026-04-02 10:00:00',
  },
  {
    policyId: 'pnp-6',
    policyName: 'CPE-B200手动升级策略',
    productType: 'CPE-B200',
    executeType: '1',
    selfStartEnable: '0',
    upgradeEnable: '1',
    targetVersion: ['V2.0.0'],
    licenseEnable: '0',
    selfConfigEnable: '0',
    createTime: '2026-03-22 14:00:00',
    updateTime: '2026-03-28 16:00:00',
  },
  {
    policyId: 'pnp-7',
    policyName: 'QAFA参数自配置策略',
    productType: 'QAFA',
    executeType: '0',
    selfStartEnable: '1',
    upgradeEnable: '0',
    targetVersion: [],
    licenseEnable: '0',
    selfConfigEnable: '1',
    createTime: '2026-04-03 11:00:00',
    updateTime: '2026-04-03 11:00:00',
  },
  {
    policyId: 'pnp-8',
    policyName: 'QAFB自动开通策略',
    productType: 'QAFB',
    executeType: '0',
    selfStartEnable: '1',
    upgradeEnable: '1',
    targetVersion: ['V2.3.0'],
    licenseEnable: '1',
    selfConfigEnable: '1',
    createTime: '2026-04-05 08:30:00',
    updateTime: '2026-04-06 09:00:00',
  },
  {
    policyId: 'pnp-9',
    policyName: 'CPE-A100 License更新策略',
    productType: 'CPE-A100',
    executeType: '0',
    selfStartEnable: '1',
    upgradeEnable: '0',
    targetVersion: [],
    licenseEnable: '1',
    selfConfigEnable: '0',
    createTime: '2026-04-08 10:00:00',
    updateTime: '2026-04-08 10:00:00',
  },
  {
    policyId: 'pnp-10',
    policyName: 'QAFC手动升级策略',
    productType: 'QAFC',
    executeType: '1',
    selfStartEnable: '0',
    upgradeEnable: '1',
    targetVersion: ['V3.1.0'],
    licenseEnable: '0',
    selfConfigEnable: '1',
    createTime: '2026-04-10 13:00:00',
    updateTime: '2026-04-10 13:00:00',
  },
];

// Mock data - combined tasks
const MOCK_TASKS: ExecuteTask[] = [
  {
    taskId: 'task-1',
    serialNumber: 'ENB00001',
    productType: 'QAFA',
    policyName: 'QAFA自动开通策略',
    executeType: '0',
    startTime: '2026-04-07 10:00:00',
    endTime: '2026-04-07 10:15:00',
    executeProcedure: 'software_upgrade > license > self_config',
    status: '0',
    failureReason: '',
    policyId: 'pnp-1',
    originalVersion: 'V2.0.0',
    targetVersion: 'V2.1.0',
    licenseFile: 'license_ENB00001.dat',
  },
  {
    taskId: 'task-2',
    serialNumber: 'ENB00002',
    productType: 'QAFA',
    policyName: 'QAFA自动开通策略',
    executeType: '0',
    startTime: '2026-04-07 10:00:00',
    endTime: '2026-04-07 10:20:00',
    executeProcedure: 'software_upgrade > license',
    status: '1',
    failureReason: 'license_download_failed',
    policyId: 'pnp-1',
    originalVersion: 'V2.0.0',
    targetVersion: 'V2.1.0',
    licenseFile: 'license_ENB00002.dat',
  },
  {
    taskId: 'task-3',
    serialNumber: 'ENB00003',
    productType: 'QAFB',
    policyName: 'QAFB手动升级策略',
    executeType: '1',
    startTime: '',
    endTime: '',
    executeProcedure: 'waiting',
    status: '3',
    failureReason: '',
    policyId: 'pnp-2',
    originalVersion: 'V2.1.0',
    targetVersion: 'V2.2.0',
    licenseFile: '',
  },
  {
    taskId: 'task-4',
    serialNumber: 'ENB00004',
    productType: 'QAFA',
    policyName: 'QAFA自动开通策略',
    executeType: '0',
    startTime: '2026-04-07 10:30:00',
    endTime: '',
    executeProcedure: 'software_upgrade_running',
    status: '2',
    failureReason: '',
    policyId: 'pnp-1',
    originalVersion: 'V2.0.0',
    targetVersion: 'V2.1.0',
    licenseFile: '',
  },
  {
    taskId: 'task-5',
    serialNumber: 'ENB00005',
    productType: 'QAFA',
    policyName: 'QAFA自动开通策略',
    executeType: '0',
    startTime: '2026-04-07 09:00:00',
    endTime: '2026-04-07 09:10:00',
    executeProcedure: 'skipped_latest',
    status: '4',
    failureReason: '',
    policyId: 'pnp-1',
    originalVersion: 'V2.1.0',
    targetVersion: 'V2.1.0',
    licenseFile: '',
  },
  {
    taskId: 'task-6',
    serialNumber: 'CPE00001',
    productType: 'CPE-A100',
    policyName: 'CPE自动开通策略',
    executeType: '0',
    startTime: '2026-04-07 10:00:00',
    endTime: '2026-04-07 10:10:00',
    executeProcedure: 'software_upgrade > self_config',
    status: '0',
    failureReason: '',
    policyId: 'pnp-4',
    originalVersion: 'V1.4.0',
    targetVersion: 'V1.5.0',
    licenseFile: '',
  },
  {
    taskId: 'task-7',
    serialNumber: 'CPE00002',
    productType: 'CPE-A100',
    policyName: 'CPE自动开通策略',
    executeType: '0',
    startTime: '2026-04-07 10:05:00',
    endTime: '',
    executeProcedure: 'self_config_running',
    status: '2',
    failureReason: '',
    policyId: 'pnp-4',
    originalVersion: 'V1.5.0',
    targetVersion: 'V1.5.0',
    licenseFile: '',
  },
  {
    taskId: 'task-8',
    serialNumber: 'ENB00006',
    productType: 'QAFA',
    policyName: 'License更新策略',
    executeType: '0',
    startTime: '2026-04-07 11:00:00',
    endTime: '2026-04-07 11:08:00',
    executeProcedure: 'license',
    status: '0',
    failureReason: '',
    policyId: 'pnp-3',
    originalVersion: '',
    targetVersion: '',
    licenseFile: 'license_QAFA_2026Q2.dat',
  },
  {
    taskId: 'task-9',
    serialNumber: 'ENB00007',
    productType: 'QAFB',
    policyName: 'QAFB手动升级策略',
    executeType: '1',
    startTime: '',
    endTime: '',
    executeProcedure: 'waiting',
    status: '3',
    failureReason: '',
    policyId: 'pnp-2',
    originalVersion: 'V2.1.0',
    targetVersion: 'V2.2.0',
    licenseFile: '',
  },
  {
    taskId: 'task-10',
    serialNumber: 'ENB00008',
    productType: 'QAFA',
    policyName: 'QAFA自动开通策略',
    executeType: '0',
    startTime: '2026-04-07 09:30:00',
    endTime: '2026-04-07 09:45:00',
    executeProcedure: 'software_upgrade > license > self_config',
    status: '0',
    failureReason: '',
    policyId: 'pnp-1',
    originalVersion: 'V2.0.0',
    targetVersion: 'V2.1.0',
    licenseFile: 'license_ENB00008.dat',
  },
  {
    taskId: 'task-11',
    serialNumber: 'CPE00003',
    productType: 'CPE-B200',
    policyName: 'CPE自动开通策略',
    executeType: '0',
    startTime: '2026-04-07 08:30:00',
    endTime: '2026-04-07 08:35:00',
    executeProcedure: 'software_upgrade',
    status: '1',
    failureReason: 'license_download_failed',
    policyId: 'pnp-4',
    originalVersion: 'V1.3.0',
    targetVersion: 'V1.5.0',
    licenseFile: '',
  },
  {
    taskId: 'task-12',
    serialNumber: 'ENB00009',
    productType: 'QAFA',
    policyName: 'License更新策略',
    executeType: '0',
    startTime: '2026-04-07 14:00:00',
    endTime: '',
    executeProcedure: 'license',
    status: '2',
    failureReason: '',
    policyId: 'pnp-3',
    originalVersion: '',
    targetVersion: '',
    licenseFile: 'license_QAFA_2026Q2_v2.dat',
  },
  {
    taskId: 'task-13',
    serialNumber: 'ENB00010',
    productType: 'QAFB',
    policyName: 'QAFB手动升级策略',
    executeType: '1',
    startTime: '2026-04-07 13:00:00',
    endTime: '2026-04-07 13:20:00',
    executeProcedure: 'software_upgrade > self_config',
    status: '0',
    failureReason: '',
    policyId: 'pnp-2',
    originalVersion: 'V2.0.0',
    targetVersion: 'V2.2.0',
    licenseFile: '',
  },
  {
    taskId: 'task-14',
    serialNumber: 'CPE00004',
    productType: 'CPE-A100',
    policyName: 'CPE自动开通策略',
    executeType: '0',
    startTime: '2026-04-07 07:50:00',
    endTime: '2026-04-07 07:55:00',
    executeProcedure: 'software_upgrade > self_config',
    status: '4',
    failureReason: '',
    policyId: 'pnp-4',
    originalVersion: 'V1.5.0',
    targetVersion: 'V1.5.0',
    licenseFile: '',
  },
  {
    taskId: 'task-15',
    serialNumber: 'ENB00011',
    productType: 'QAFA',
    policyName: 'QAFA自动开通策略',
    executeType: '0',
    startTime: '2026-04-07 11:30:00',
    endTime: '2026-04-07 11:50:00',
    executeProcedure: 'software_upgrade > license > self_config',
    status: '1',
    failureReason: 'license_download_failed',
    policyId: 'pnp-1',
    originalVersion: 'V2.0.0',
    targetVersion: 'V2.1.0',
    licenseFile: 'license_ENB00011.dat',
  },
];

export default function PlugAndPlay() {
  const t = useT();
  const navigate = useNavigate();
  const { message } = App.useApp();

  // State
  const [policies, setPolicies] = useState<Policy[]>(MOCK_POLICIES);
  const [tasks, setTasks] = useState<ExecuteTask[]>(MOCK_TASKS);

  // Policy filter state
  const [policyProductType, setPolicyProductType] = useState<string>('');
  const [policySearchText, setPolicySearchText] = useState('');

  // Task filter state
  const [taskStatus, setTaskStatus] = useState<string>('');
  const [taskSearchText, setTaskSearchText] = useState('');
  const [taskTab, setTaskTab] = useState<string>('0');
  const [taskPage, setTaskPage] = useState(1);
  const [taskPageSize, setTaskPageSize] = useState(10);

  // Dialog state
  const [detectDialogOpen, setDetectDialogOpen] = useState(false);
  const [selectedPolicy, setSelectedPolicy] = useState<Policy | null>(null);
  const [detailTaskId, setDetailTaskId] = useState<string | null>(null);
  const [selectedTaskIds, setSelectedTaskIds] = useState<string[]>([]);
  const [batchRetryOpen, setBatchRetryOpen] = useState(false);

  // Status config
  const STATUS_CONFIG = useMemo(() => ({
    '0': { label: t('status.success'), color: 'success', icon: <CheckCircleOutlined /> },
    '1': { label: t('status.failed'), color: 'error', icon: <CloseCircleOutlined /> },
    '2': { label: t('status.running'), color: 'processing', icon: <LoadingOutlined /> },
    '3': { label: t('provision.pending'), color: 'default', icon: <ClockCircleOutlined /> },
    '4': { label: t('provision.skipped'), color: 'warning', icon: <ForwardOutlined /> },
  }), [t]);

  // Translate execute procedure codes
  const translateProcedure = useCallback((procedure: string | undefined | null) => {
    if (!procedure || typeof procedure !== 'string') return '-';
    const procedureMap: Record<string, string> = {
      'software_upgrade': t('provision.softwareUpgrade'),
      'license': t('provision.license'),
      'self_config': t('provision.selfConfig'),
      'software_upgrade_running': t('provision.softwareUpgradeRunning'),
      'self_config_running': t('provision.selfConfigRunning'),
      'waiting': t('provision.waitingExecute'),
      'skipped_latest': t('provision.skippedLatestVersion'),
    };
    return procedure
      .split(/ > |,/)
      .map(step => procedureMap[step.trim()] || step)
      .join(' > ');
  }, [t]);

  // Translate failure reason codes
  const translateFailureReason = useCallback((reason: string | undefined | null) => {
    if (!reason || typeof reason !== 'string') return '-';
    const reasonMap: Record<string, string> = {
      'license_download_failed': t('provision.licenseDownloadFailed'),
    };
    return reasonMap[reason] || reason;
  }, [t]);

  // Handlers - Policy
  const handlePolicyMenuClick = useCallback((key: string, record: Policy) => {
    switch (key) {
      case 'info':
        navigate(`/device/plug-and-play/view/${record.policyId}`);
        break;
      case 'edit':
        navigate(`/device/plug-and-play/edit/${record.policyId}`);
        break;
      case 'detect':
        setSelectedPolicy(record);
        setDetectDialogOpen(true);
        break;
      case 'delete':
        setPolicies(prev => prev.filter(p => p.policyId !== record.policyId));
        message.success(t('common.deleteSuccess'));
        break;
    }
  }, [navigate, message, t]);

  const handlePolicySwitch = useCallback((record: Policy, checked: boolean) => {
    const newEnable = checked ? '1' : '0';
    setPolicies(prev => prev.map(p =>
      p.policyId === record.policyId ? { ...p, selfStartEnable: newEnable } : p
    ));
    message.success(t('common.success'));
  }, [message, t]);

  // Handlers - Task
  const handleRetryTask = useCallback((record: ExecuteTask) => {
    setTasks(prev => prev.map(item =>
      item.taskId === record.taskId ? { ...item, status: '2', startTime: new Date().toISOString(), endTime: '' } : item
    ));
    message.success(t('common.success'));
  }, [message, t]);

  const handleStartTask = useCallback((record: ExecuteTask) => {
    setTasks(prev => prev.map(item =>
      item.taskId === record.taskId ? { ...item, status: '2', startTime: new Date().toISOString() } : item
    ));
    message.success(t('common.commandSent'));
  }, [message, t]);

  const handleDeleteTask = useCallback((record: ExecuteTask) => {
    setTasks(prev => prev.filter(item => item.taskId !== record.taskId));
    message.success(t('common.deleteSuccess'));
  }, [message, t]);

  const handleAddPolicy = useCallback(() => {
    navigate('/device/plug-and-play/add');
  }, [navigate]);

  const handleDetectSuccess = useCallback(() => {
    setDetectDialogOpen(false);
    message.success(t('provision.detectSuccess'));
  }, [message, t]);

  // Policy columns
  const policyColumns: DataTableColumn<Policy>[] = useMemo(() => [
    {
      key: 'actions',
      title: '',
      width: 100,
      fixed: 'right',
      render: (_, record) => (
        <Space size={4}>
          <Button type="link" size="small" onClick={() => handlePolicyMenuClick('info', record)}>
            {t('common.detail')}
          </Button>
          <Dropdown
            menu={{
              items: [
                { key: 'edit', label: t('common.edit'), icon: <EditOutlined />, disabled: record.selfStartEnable === '1' },
                { key: 'detect', label: t('provision.detect'), icon: <ScanOutlined /> },
                { type: 'divider' as const },
                { key: 'delete', label: t('common.delete'), icon: <DeleteOutlined />, danger: true, disabled: record.selfStartEnable === '1' },
              ] as MenuProps['items'],
              onClick: ({ key }) => handlePolicyMenuClick(key, record),
            }}
            trigger={['click']}
          >
            <Button type="text" size="small" icon={<MoreOutlined />} onClick={(e) => e.stopPropagation()} />
          </Dropdown>
        </Space>
      ),
    },
    {
      key: 'selfStartEnable',
      title: t('provision.enabled'),
      dataIndex: 'selfStartEnable',
      width: 90,
      render: (val, record) => (
        <Switch
          size="small"
          checked={val === '1'}
          onChange={(checked) => handlePolicySwitch(record, checked)}
        />
      ),
    },
    {
      key: 'productType',
      title: t('provision.productType'),
      dataIndex: 'productType',
      width: 120,
      ellipsis: true,
    },
    {
      key: 'policyName',
      title: t('provision.policyName'),
      dataIndex: 'policyName',
      ellipsis: true,
    },
    {
      key: 'executeType',
      title: t('provision.executeType'),
      dataIndex: 'executeType',
      width: 120,
      ellipsis: true,
      render: (val) => (
        <Tag color={val === '0' ? 'green' : 'blue'}>
          {val === '0' ? t('provision.autoExecute') : t('provision.manualExecute')}
        </Tag>
      ),
    },
    {
      key: 'upgradeEnable',
      title: t('provision.softwareUpgrade'),
      dataIndex: 'upgradeEnable',
      width: 200,
      ellipsis: true,
      render: (val, record) => {
        const version = val === '1' && record.targetVersion?.[0]
          ? ` ${t('provision.targetVersion')}=${record.targetVersion[0]}`
          : '';
        return (
          <span style={{ whiteSpace: 'nowrap' }}>
            {val === '1' ? (
              <Tag color="success">{t('common.enabled')}</Tag>
            ) : (
              <Tag color="default">{t('common.disabled')}</Tag>
            )}
            {version && <Text type="secondary" style={{ fontSize: 12 }}>{version}</Text>}
          </span>
        );
      },
    },
    {
      key: 'licenseEnable',
      title: t('provision.license'),
      dataIndex: 'licenseEnable',
      width: 80,
      ellipsis: true,
      render: (val) => (
        val === '1' ? (
          <Tag color="success">{t('common.enabled')}</Tag>
        ) : (
          <Tag color="default">{t('common.disabled')}</Tag>
        )
      ),
    },
    {
      key: 'selfConfigEnable',
      title: t('provision.selfConfig'),
      dataIndex: 'selfConfigEnable',
      width: 120,
      ellipsis: true,
      render: (val) => (
        val === '1' ? (
          <Tag color="success">{t('common.enabled')}</Tag>
        ) : (
          <Tag color="default">{t('common.disabled')}</Tag>
        )
      ),
    },
  ], [t, handlePolicyMenuClick, handlePolicySwitch]);

  // Task columns - dynamic based on taskTab
  const taskColumns: DataTableColumn<ExecuteTask>[] = useMemo(() => {
    const commonColumns: DataTableColumn<ExecuteTask>[] = [
      {
        key: 'actions',
        title: '',
        width: 100,
        fixed: 'right',
        render: (_, record) => {
          const items: MenuProps['items'] = [
            ['0', '1', '4'].includes(record.status) ? {
              key: 'retry',
              label: t('provision.retry'),
              icon: <RedoOutlined />,
              onClick: () => handleRetryTask(record),
            } : null,
            record.executeType === '1' && record.status === '3' ? {
              key: 'execute',
              label: t('common.execute'),
              icon: <PlayCircleOutlined />,
              onClick: () => handleStartTask(record),
            } : null,
            ['0', '1', '3', '4'].includes(record.status) ? {
              key: 'delete',
              label: t('common.delete'),
              icon: <DeleteOutlined />,
              danger: true,
              onClick: () => handleDeleteTask(record),
            } : null,
          ].filter(Boolean) as NonNullable<MenuProps['items']>;

          return (
            <Space size={4}>
              <Button type="link" size="small" onClick={() => setDetailTaskId(record.taskId)}>
                {t('common.detail')}
              </Button>
              {items && items.length > 0 && (
                <Dropdown menu={{ items }} trigger={['click']}>
                  <Button type="text" size="small" icon={<MoreOutlined />} onClick={(e) => e.stopPropagation()} />
                </Dropdown>
              )}
            </Space>
          );
        },
      },
      {
        key: 'serialNumber',
        title: t('provision.deviceCode'),
        dataIndex: 'serialNumber',
        width: 140,
        mono: true,
      },
      {
        key: 'productType',
        title: t('provision.productType'),
        dataIndex: 'productType',
        width: 100,
      },
      {
        key: 'policyName',
        title: t('provision.policyName'),
        dataIndex: 'policyName',
        width: 160,
        ellipsis: true,
      },
      {
        key: 'executeType',
        title: t('provision.executeType'),
        dataIndex: 'executeType',
        width: 100,
        render: (val) => (
          <Tag color={val === '0' ? 'green' : 'blue'}>
            {val === '0' ? t('provision.autoExecute') : t('provision.manualExecute')}
          </Tag>
        ),
      },
      {
        key: 'startTime',
        title: t('provision.startTime'),
        dataIndex: 'startTime',
        width: 150,
      },
      {
        key: 'endTime',
        title: t('provision.endTime'),
        dataIndex: 'endTime',
        width: 150,
      },
    ];

    // Tab-specific columns
    const tabSpecificColumns: DataTableColumn<ExecuteTask>[] = (() => {
      switch (taskTab) {
        case '0': // All Tasks - show progress
          return [{
            key: 'executeProcedure',
            title: t('provision.progress'),
            dataIndex: 'executeProcedure',
            width: 200,
            ellipsis: true,
            render: (val: string) => translateProcedure(val),
          }];
        case '1': // Software Upgrade - show original/target version
          return [
            {
              key: 'originalVersion',
              title: t('provision.originalVersion'),
              dataIndex: 'originalVersion',
              width: 120,
              render: (val: string) => val || '-',
            },
            {
              key: 'targetVersion',
              title: t('provision.targetVersion'),
              dataIndex: 'targetVersion',
              width: 120,
              render: (val: string) => val || '-',
            },
          ];
        case '2': // License - show license file
          return [{
            key: 'licenseFile',
            title: t('provision.licenseFile'),
            dataIndex: 'licenseFile',
            width: 200,
            ellipsis: true,
            render: (val: string) => val || '-',
          }];
        default: // Self Config - no extra columns
          return [];
      }
    })();

    const tailColumns: DataTableColumn<ExecuteTask>[] = [
      {
        key: 'status',
        title: t('table.status'),
        dataIndex: 'status',
        width: 90,
        render: (val: TaskStatus) => {
          const cfg = STATUS_CONFIG[val];
          return <Tag color={cfg.color} icon={cfg.icon}>{cfg.label}</Tag>;
        },
      },
      {
        key: 'failureReason',
        title: t('provision.failureReason'),
        dataIndex: 'failureReason',
        width: 200,
        ellipsis: true,
        render: (val: string) => translateFailureReason(val),
      },
    ];

    return [...commonColumns, ...tabSpecificColumns, ...tailColumns];
  }, [t, taskTab, STATUS_CONFIG, translateProcedure, translateFailureReason, handleRetryTask, handleStartTask, handleDeleteTask]);

  // Filtered policies
  const filteredPolicies = useMemo(() => {
    let result = policies;
    if (policyProductType) {
      result = result.filter(p => p.productType === policyProductType);
    }
    if (policySearchText) {
      const search = policySearchText.toLowerCase();
      result = result.filter(p =>
        p.policyName.toLowerCase().includes(search) ||
        p.productType.toLowerCase().includes(search) ||
        p.targetVersion?.[0]?.toLowerCase().includes(search)
      );
    }
    return result;
  }, [policies, policyProductType, policySearchText]);

  // Filtered tasks
  const filteredTasks = useMemo(() => {
    let result = tasks;
    // Filter by tab (using symbolic codes)
    if (taskTab === '1') {
      result = result.filter(t => t.executeProcedure.includes('software_upgrade'));
    } else if (taskTab === '2') {
      result = result.filter(t => t.executeProcedure.includes('license'));
    } else if (taskTab === '3') {
      result = result.filter(t => t.executeProcedure.includes('self_config'));
    }
    // Filter by status
    if (taskStatus) {
      result = result.filter(t => t.status === taskStatus);
    }
    // Filter by search text
    if (taskSearchText) {
      const search = taskSearchText.toLowerCase();
      result = result.filter(t =>
        t.serialNumber.toLowerCase().includes(search) ||
        t.policyName.toLowerCase().includes(search)
      );
    }
    return result;
  }, [tasks, taskTab, taskStatus, taskSearchText]);

  const successCount = useMemo(() => filteredTasks.filter(t => t.status === '0').length, [filteredTasks]);
  const failCount = useMemo(() => filteredTasks.filter(t => t.status === '1').length, [filteredTasks]);

  // Batch actions for task table (like RecycleBin)
  const taskBatchActions: BatchAction[] = useMemo(() => [
    {
      key: 'batch-retry',
      label: t('provision.batchRetry'),
      icon: <RedoOutlined />,
      onClick: (keys) => {
        setSelectedTaskIds(keys as string[]);
        setBatchRetryOpen(true);
      },
    },
  ], [t]);

  return (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column', gap: 12 }}>
      {/* Policy List Section */}
      <div style={{ flexShrink: 0 }}>
        <ListPageLayout
          title={t('provision.plugAndPlay')}
          extra={
            <Button type="primary" icon={<PlusOutlined />} onClick={handleAddPolicy}>
              {t('common.add')}
            </Button>
          }
        >
          <DataTable<Policy>
            tableId="policy-table"
            columns={policyColumns}
            dataSource={filteredPolicies}
            rowKey="policyId"
            total={filteredPolicies.length}
            showPagination
            currentPage={1}
            pageSize={10}
            defaultDensity="default"
            showRowNumber
            rowNumberTitle={t('table.rowNumber')}
            scroll={{ x: 'max-content', y: 170 }}
            extraToolbarLeft={<Text strong>{t('provision.policyList')}</Text>}
            extraToolbarRight={
              <Space>
                <Select
                  placeholder={t('provision.productType')}
                  value={policyProductType || undefined}
                  onChange={(value) => setPolicyProductType(value || '')}
                  allowClear
                  style={{ width: 140 }}
                  options={PRODUCT_TYPES}
                />
                <Input
                  placeholder={t('provision.searchPolicyPlaceholder')}
                  prefix={<SearchOutlined />}
                  value={policySearchText}
                  onChange={(e) => setPolicySearchText(e.target.value)}
                  style={{ width: 200 }}
                  allowClear
                />
              </Space>
            }
          />
        </ListPageLayout>
      </div>

      {/* Execute Status Section */}
      <Card
        size="small"
        styles={{ body: { padding: 0, display: 'flex', flexDirection: 'column' } }}
        style={{ flex: 1, minHeight: 0 }}
      >
        {/* 执行状态标题 + 计数 + 任务类型筛选 */}
        <div style={{ padding: '12px 16px', borderBottom: '1px solid #f0f0f0', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Space>
            <Text strong style={{ fontSize: 14 }}>{t('provision.executeStatus')}</Text>
            {[
              { key: '0', label: t('provision.allTasks') },
              { key: '1', label: t('provision.softwareUpgrade') },
              { key: '2', label: t('provision.license') },
              { key: '3', label: t('provision.selfConfig') },
            ].map(item => (
              <Tag
                key={item.key}
                color={taskTab === item.key ? 'blue' : 'default'}
                style={{ cursor: 'pointer' }}
                onClick={() => { setTaskTab(item.key); setTaskPage(1); }}
              >
                {item.label}
              </Tag>
            ))}
          </Space>
          <Space split="|" size={8}>
            <Space size={4}>
              <CheckCircleOutlined style={{ color: '#52c41a' }} />
              <Text type="success">{successCount}</Text>
            </Space>
            <Space size={4}>
              <CloseCircleOutlined style={{ color: '#ff4d4f' }} />
              <Text type="danger">{failCount}</Text>
            </Space>
          </Space>
        </div>

        {/* Task table with filters in toolbar */}
        <div style={{ flex: 1, minHeight: 0 }}>
          <DataTable<ExecuteTask>
            tableId="task-table"
            columns={taskColumns}
            dataSource={filteredTasks}
            rowKey="taskId"
            total={filteredTasks.length}
            currentPage={taskPage}
            pageSize={taskPageSize}
            onPageChange={(p, s) => { setTaskPage(p); setTaskPageSize(s); }}
            defaultDensity="default"
            showRowNumber
            rowNumberTitle={t('table.rowNumber')}
            selectable={taskTab === '0'}
            selectedRowKeys={selectedTaskIds}
            onSelectionChange={(keys) => setSelectedTaskIds(keys as string[])}
            batchActions={taskTab === '0' ? taskBatchActions : undefined}
            scroll={{ x: 'max-content', y: 220 }}
            extraToolbarRight={
              <Space>
                <Select
                  placeholder={t('table.status')}
                  value={taskStatus || undefined}
                  onChange={(value) => { setTaskStatus(value || ''); setTaskPage(1); }}
                  allowClear
                  style={{ width: 100 }}
                  options={[
                    { label: t('status.success'), value: '0' },
                    { label: t('status.failed'), value: '1' },
                    { label: t('status.running'), value: '2' },
                    { label: t('provision.pending'), value: '3' },
                    { label: t('provision.skipped'), value: '4' },
                  ]}
                />
                <Input
                  placeholder={t('provision.searchPlaceholder')}
                  prefix={<SearchOutlined />}
                  value={taskSearchText}
                  onChange={(e) => { setTaskSearchText(e.target.value); setTaskPage(1); }}
                  style={{ width: 180 }}
                  allowClear
                />
              </Space>
            }
          />
        </div>
      </Card>

      {/* Detect Dialog */}
      <DetectDialog
        open={detectDialogOpen}
        policy={selectedPolicy}
        onClose={() => setDetectDialogOpen(false)}
        onSuccess={handleDetectSuccess}
      />

      {/* Execute Detail Panel */}
      {detailTaskId && (
        <ExecuteDetailPanel
          taskId={detailTaskId}
          taskData={tasks.find(t => t.taskId === detailTaskId)}
          onClose={() => setDetailTaskId(null)}
        />
      )}

      {/* Batch Retry Dialog */}
      <BatchRetryDialog
        open={batchRetryOpen}
        taskCount={selectedTaskIds.length}
        onClose={() => setBatchRetryOpen(false)}
        onConfirm={(includeSuccess) => {
          console.log('Batch retry:', selectedTaskIds, 'includeSuccess:', includeSuccess);
          setBatchRetryOpen(false);
          setSelectedTaskIds([]);
          message.success(t('common.success'));
        }}
      />
    </div>
  );
}
