import React, { useState, useMemo, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { Button, Space, Tag, Switch, Dropdown, Input, App, Typography, Card, Select } from 'antd';
import type { MenuProps } from 'antd';
import {
  PlusOutlined,
  SearchOutlined,
  InfoCircleOutlined,
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
  SyncOutlined,
  ReloadOutlined,
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
];

// Mock data - combined tasks
const MOCK_TASKS: ExecuteTask[] = [
  {
    taskId: 'task-1',
    serialNumber: 'ENB00001',
    productType: 'QAFA',
    policyName: 'QAFA Auto Provision Policy',
    executeType: '0',
    startTime: '2026-04-07 10:00:00',
    endTime: '2026-04-07 10:15:00',
    executeProcedure: 'software_upgrade > license > self_config',
    status: '0',
    failureReason: '',
    policyId: 'pnp-1',
  },
  {
    taskId: 'task-2',
    serialNumber: 'ENB00002',
    productType: 'QAFA',
    policyName: 'QAFA Auto Provision Policy',
    executeType: '0',
    startTime: '2026-04-07 10:00:00',
    endTime: '2026-04-07 10:20:00',
    executeProcedure: 'software_upgrade > license',
    status: '1',
    failureReason: 'license_download_failed',
    policyId: 'pnp-1',
  },
  {
    taskId: 'task-3',
    serialNumber: 'ENB00003',
    productType: 'QAFB',
    policyName: 'QAFB Manual Upgrade Policy',
    executeType: '1',
    startTime: '',
    endTime: '',
    executeProcedure: 'waiting',
    status: '3',
    failureReason: '',
    policyId: 'pnp-2',
  },
  {
    taskId: 'task-4',
    serialNumber: 'ENB00004',
    productType: 'QAFA',
    policyName: 'QAFA Auto Provision Policy',
    executeType: '0',
    startTime: '2026-04-07 10:30:00',
    endTime: '',
    executeProcedure: 'software_upgrade_running',
    status: '2',
    failureReason: '',
    policyId: 'pnp-1',
  },
  {
    taskId: 'task-5',
    serialNumber: 'ENB00005',
    productType: 'QAFA',
    policyName: 'QAFA Auto Provision Policy',
    executeType: '0',
    startTime: '2026-04-07 09:00:00',
    endTime: '2026-04-07 09:10:00',
    executeProcedure: 'skipped_latest',
    status: '4',
    failureReason: '',
    policyId: 'pnp-1',
  },
  {
    taskId: 'task-6',
    serialNumber: 'CPE00001',
    productType: 'CPE-A100',
    policyName: 'CPE Auto Provision Policy',
    executeType: '0',
    startTime: '2026-04-07 10:00:00',
    endTime: '2026-04-07 10:10:00',
    executeProcedure: 'software_upgrade > self_config',
    status: '0',
    failureReason: '',
    policyId: 'pnp-4',
  },
  {
    taskId: 'task-7',
    serialNumber: 'CPE00002',
    productType: 'CPE-A100',
    policyName: 'CPE Auto Provision Policy',
    executeType: '0',
    startTime: '2026-04-07 10:05:00',
    endTime: '',
    executeProcedure: 'self_config_running',
    status: '2',
    failureReason: '',
    policyId: 'pnp-4',
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
  const [taskTimeRange, setTaskTimeRange] = useState<[string, string] | null>(null);
  const [taskSearchText, setTaskSearchText] = useState('');
  const [taskTab, setTaskTab] = useState<string>('0');

  // Dialog state
  const [detectDialogOpen, setDetectDialogOpen] = useState(false);
  const [selectedPolicy, setSelectedPolicy] = useState<Policy | null>(null);
  const [detailTaskId, setDetailTaskId] = useState<string | null>(null);
  const [selectedTaskIds, setSelectedTaskIds] = useState<string[]>([]);
  const [batchRetryOpen, setBatchRetryOpen] = useState(false);
  const [autoRefresh, setAutoRefresh] = useState(false);
  const [policyAutoRefresh, setPolicyAutoRefresh] = useState(false);

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

  // Policy columns
  const policyColumns: DataTableColumn<Policy>[] = useMemo(() => [
    {
      key: 'actions',
      title: '',
      width: 80,
      fixed: 'left',
      render: (_, record) => (
        <Dropdown
          menu={{
            items: [
              { key: 'info', label: t('common.detail'), icon: <InfoCircleOutlined /> },
              { key: 'edit', label: t('common.edit'), icon: <EditOutlined />, disabled: record.selfStartEnable === '1' },
              { key: 'detect', label: t('provision.detect'), icon: <ScanOutlined /> },
              { key: 'delete', label: t('common.delete'), icon: <DeleteOutlined />, danger: true, disabled: record.selfStartEnable === '1' },
            ] as MenuProps['items'],
            onClick: ({ key }) => handlePolicyMenuClick(key, record),
          }}
          trigger={['click']}
        >
          <Button size="small" icon={<MoreOutlined />}>{t('common.more')}</Button>
        </Dropdown>
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
      width: 160,
      render: (val, record) => (
        <Space size={4}>
          {val === '1' ? (
            <Tag icon={<CheckCircleOutlined />} color="success">{t('common.enabled')}</Tag>
          ) : (
            <Tag color="default">{t('common.disabled')}</Tag>
          )}
          {val === '1' && record.targetVersion?.[0] && (
            <Text type="secondary" style={{ fontSize: 12 }}>V{record.targetVersion[0]}</Text>
          )}
        </Space>
      ),
    },
    {
      key: 'licenseEnable',
      title: 'License',
      dataIndex: 'licenseEnable',
      width: 80,
      render: (val) => (
        val === '1' ? (
          <Tag icon={<CheckCircleOutlined />} color="success" />
        ) : (
          <Tag color="default" />
        )
      ),
    },
    {
      key: 'selfConfigEnable',
      title: t('provision.selfConfig'),
      dataIndex: 'selfConfigEnable',
      width: 120,
      render: (val) => (
        val === '1' ? (
          <Tag icon={<CheckCircleOutlined />} color="success">{t('common.enabled')}</Tag>
        ) : (
          <Tag color="default">{t('common.disabled')}</Tag>
        )
      ),
    },
  ], [t]);

  // Task columns
  const taskColumns: DataTableColumn<ExecuteTask>[] = useMemo(() => [
    {
      key: 'actions',
      title: '',
      width: 80,
      fixed: 'left',
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
          {
            key: 'detail',
            label: t('common.detail'),
            icon: <InfoCircleOutlined />,
            onClick: () => setDetailTaskId(record.taskId),
          },
        ].filter(Boolean) as MenuProps['items'];

        if (!items || items.length === 0) return null;

        return (
          <Dropdown menu={{ items }} trigger={['click']}>
            <Button size="small" icon={<MoreOutlined />}>{t('common.more')}</Button>
          </Dropdown>
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
    ...(taskTab === '0' ? [{
      key: 'executeProcedure',
      title: t('provision.progress'),
      dataIndex: 'executeProcedure',
      width: 200,
      ellipsis: true,
      render: (val: string) => translateProcedure(val),
    }] : []),
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
  ], [t, taskTab, STATUS_CONFIG, translateProcedure, translateFailureReason]);

  // Handlers
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

  const handleRetryTask = useCallback((record: ExecuteTask) => {
    setTasks(prev => prev.map(t =>
      t.taskId === record.taskId ? { ...t, status: '2', startTime: new Date().toISOString(), endTime: '' } : t
    ));
    message.success(t('common.success'));
  }, [message, t]);

  const handleStartTask = useCallback((record: ExecuteTask) => {
    setTasks(prev => prev.map(t =>
      t.taskId === record.taskId ? { ...t, status: '2', startTime: new Date().toISOString() } : t
    ));
    message.success(t('common.commandSent'));
  }, [message, t]);

  const handleDeleteTask = useCallback((record: ExecuteTask) => {
    setTasks(prev => prev.filter(t => t.taskId !== record.taskId));
    message.success(t('common.deleteSuccess'));
  }, [message, t]);

  const handleAddPolicy = useCallback(() => {
    navigate('/device/plug-and-play/add');
  }, [navigate]);

  const handleDetectSuccess = useCallback(() => {
    setDetectDialogOpen(false);
    message.success(t('provision.detectSuccess'));
  }, [message, t]);

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
          title={t('provision.policyList')}
          subtitle={`${t('table.total')} ${filteredPolicies.length}`}
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
            pageSize={10}
            defaultDensity="compact"
            showRowNumber
            rowNumberTitle={t('table.rowNumber')}
            pagination={false}
            scroll={{ x: 'max-content', y: 200 }}
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
                <Button
                  type={policyAutoRefresh ? 'primary' : 'default'}
                  icon={<SyncOutlined spin={policyAutoRefresh} />}
                  onClick={() => setPolicyAutoRefresh(!policyAutoRefresh)}
                  title={policyAutoRefresh ? t('provision.stopAutoRefresh') : t('provision.startAutoRefresh')}
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
        {/* Header row: Task Type tabs | Execute Status title + counts */}
        <div style={{ padding: '12px 16px', borderBottom: '1px solid #f0f0f0', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Space>
            <Text type="secondary">{t('provision.taskType')}:</Text>
            {[
              { key: '0', label: t('provision.allTasks') },
              { key: '1', label: t('provision.softwareUpgrade') },
              { key: '2', label: 'License' },
              { key: '3', label: t('provision.selfConfig') },
            ].map(item => (
              <Tag
                key={item.key}
                color={taskTab === item.key ? 'blue' : 'default'}
                style={{ cursor: 'pointer' }}
                onClick={() => setTaskTab(item.key)}
              >
                {item.label}
              </Tag>
            ))}
          </Space>
          <Space>
            <Text strong>{t('provision.executeStatus')}</Text>
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
          </Space>
        </div>

        {/* Task table with filters in toolbar */}
        <div style={{ flex: 1, minHeight: 0 }}>
          <DataTable<ExecuteTask>
            tableId="task-table"
            columns={taskColumns}
            dataSource={filteredTasks}
            rowKey="taskId"
            pageSize={10}
            defaultDensity="compact"
            showRowNumber
            rowNumberTitle={t('table.rowNumber')}
            selectable={taskTab === '0'}
            selectedRowKeys={selectedTaskIds}
            onSelectionChange={(keys) => setSelectedTaskIds(keys as string[])}
            batchActions={taskTab === '0' ? taskBatchActions : undefined}
            scroll={{ x: 'max-content', y: 180 }}
            extraToolbarRight={
              <Space>
                <Select
                  placeholder={t('table.status')}
                  value={taskStatus || undefined}
                  onChange={(value) => setTaskStatus(value || '')}
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
                  onChange={(e) => setTaskSearchText(e.target.value)}
                  style={{ width: 180 }}
                  allowClear
                />
                <Button
                  type={autoRefresh ? 'primary' : 'default'}
                  icon={<SyncOutlined spin={autoRefresh} />}
                  onClick={() => setAutoRefresh(!autoRefresh)}
                  title={autoRefresh ? t('provision.stopAutoRefresh') : t('provision.startAutoRefresh')}
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
