import { useState, useMemo, useCallback } from 'react';
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
  ReloadOutlined,
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
import {
  useProvisioningTasks,
  useRetryProvisioningTask,
} from '@core/hooks/api/useProvisioning';
import {
  mapTaskToExecuteView,
  type ProvisioningExecuteView,
  type ProvisioningTaskStatusCode,
} from '@core/services/api/provisionApi';
import styles from './index.module.css';
import DetectDialog from './components/DetectDialog';
import ExecuteDetailPanel from './components/ExecuteDetailPanel';
import BatchRetryDialog from './components/BatchRetryDialog';

const { Text } = Typography;

// Types
type ExecuteType = '0' | '1'; // 0-auto, 1-manual

interface Policy {
  policyId: string;
  policyName: string;
  productClass: string;
  executeType: ExecuteType;
  selfStartEnable: '0' | '1';
  upgradeEnable: '0' | '1';
  targetVersion: string[];
  licenseEnable: '0' | '1';
  selfConfigEnable: '0' | '1';
  createTime: string;
  updateTime: string;
}

// Product types
const PRODUCT_TYPES = [
  { label: 'QAFA', value: 'QAFA' },
  { label: 'QAFB', value: 'QAFB' },
  { label: 'QAFC', value: 'QAFC' },
  { label: 'CPE-A100', value: 'CPE-A100' },
  { label: 'CPE-B200', value: 'CPE-B200' },
];

// 策略管理（自动开通策略）目前**没有后端端点**——provision 域后端（F09）只暴露
// /api/v1/provisioning/tasks（任务列表/详情/创建/重试），无 policies 资源。
// 因此策略列表仍为前端配置态（增删改/开关只改本地 state），等后端补 policies API 后再接线。
// 见 issue #140：本次只接线「执行状态」任务列表为真实 API。
const MOCK_POLICIES: Policy[] = [
  {
    policyId: 'pnp-1',
    policyName: 'QAFA自动开通策略',
    productClass: 'QAFA',
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
    productClass: 'QAFB',
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
    productClass: 'QAFA',
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
    productClass: 'CPE-A100',
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
    productClass: 'QAFC',
    executeType: '0',
    selfStartEnable: '1',
    upgradeEnable: '1',
    targetVersion: ['V3.0.0'],
    licenseEnable: '1',
    selfConfigEnable: '1',
    createTime: '2026-03-18 09:30:00',
    updateTime: '2026-04-02 10:00:00',
  },
];

const PAGE_SIZE = 10;

export default function PlugAndPlay() {
  const t = useT();
  const navigate = useNavigate();
  const { message } = App.useApp();

  // Policy state (front-end config-only, no backend endpoint yet — see comment above)
  const [policies, setPolicies] = useState<Policy[]>(MOCK_POLICIES);

  // Policy filter state
  const [policyProductClass, setPolicyProductClass] = useState<string>('');
  const [policySearchText, setPolicySearchText] = useState('');

  // Task filter state (执行状态 — real provisioning/tasks API)
  const [taskStatus, setTaskStatus] = useState<string>('');
  const [taskSearchText, setTaskSearchText] = useState('');
  const [taskPage, setTaskPage] = useState(1);
  const [taskPageSize, setTaskPageSize] = useState(PAGE_SIZE);

  // Map UI status code → backend lifecycle filter. Only the terminal states map
  // 1:1; the page's "running" bucket spans several backend states, so it is
  // filtered client-side rather than passed to the API.
  const backendStatusFilter = useMemo(() => {
    if (taskStatus === '0') return 'completed';
    if (taskStatus === '1') return 'failed';
    return undefined;
  }, [taskStatus]);

  const {
    data: taskData,
    isLoading: tasksLoading,
    refetch: refetchTasks,
  } = useProvisioningTasks({
    page: taskPage,
    pageSize: taskPageSize,
    status: backendStatusFilter,
  });

  const retryTaskMutation = useRetryProvisioningTask();

  // Dialog state
  const [detectDialogOpen, setDetectDialogOpen] = useState(false);
  const [selectedPolicy, setSelectedPolicy] = useState<Policy | null>(null);
  const [detailTaskId, setDetailTaskId] = useState<string | null>(null);
  const [selectedTaskIds, setSelectedTaskIds] = useState<string[]>([]);
  const [batchRetryOpen, setBatchRetryOpen] = useState(false);

  // Status config
  const STATUS_CONFIG = useMemo<
    Record<ProvisioningTaskStatusCode, { label: string; color: string; icon: React.ReactNode }>
  >(() => ({
    '0': { label: t('status.success'), color: 'success', icon: <CheckCircleOutlined /> },
    '1': { label: t('status.failed'), color: 'error', icon: <CloseCircleOutlined /> },
    '2': { label: t('status.running'), color: 'processing', icon: <LoadingOutlined /> },
    '3': { label: t('provision.pending'), color: 'default', icon: <ClockCircleOutlined /> },
    '4': { label: t('provision.skipped'), color: 'warning', icon: <ForwardOutlined /> },
  }), [t]);

  // Translate failure reason codes (backend may send raw error strings)
  const translateFailureReason = useCallback((reason: string | undefined | null) => {
    if (!reason || typeof reason !== 'string') return '-';
    const reasonMap: Record<string, string> = {
      'license_download_failed': t('provision.licenseDownloadFailed'),
    };
    return reasonMap[reason] || reason;
  }, [t]);

  // Handlers - Policy (local config state only — no backend policies API yet)
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

  // Handlers - Task (real API)
  const handleRetryTask = useCallback(async (record: ProvisioningExecuteView) => {
    try {
      await retryTaskMutation.mutateAsync(record.taskId);
      message.success(t('common.success'));
    } catch {
      message.error(t('common.operationFailed'));
    }
  }, [retryTaskMutation, message, t]);

  const handleAddPolicy = useCallback(() => {
    navigate('/device/plug-and-play/add');
  }, [navigate]);

  const handleDetectSuccess = useCallback(() => {
    setDetectDialogOpen(false);
    message.success(t('provision.detectSuccess'));
  }, [message, t]);

  const handleRefreshTasks = useCallback(() => {
    void refetchTasks();
  }, [refetchTasks]);

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
      key: 'productClass',
      title: t('provision.productClass'),
      dataIndex: 'productClass',
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

  // Task columns — driven by real ProvisioningTask shape
  const taskColumns: DataTableColumn<ProvisioningExecuteView>[] = useMemo(() => [
    {
      key: 'actions',
      title: '',
      width: 100,
      fixed: 'right',
      render: (_, record) => {
        const items: MenuProps['items'] = [
          // Backend only allows retrying failed tasks (handler.go Retry).
          record.status === '1' ? {
            key: 'retry',
            label: t('provision.retry'),
            icon: <RedoOutlined />,
            onClick: () => { void handleRetryTask(record); },
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
      key: 'deviceId',
      title: t('provision.deviceId'),
      dataIndex: 'deviceId',
      width: 280,
      mono: true,
      ellipsis: true,
    },
    {
      key: 'status',
      title: t('table.status'),
      dataIndex: 'status',
      width: 110,
      render: (raw: unknown) => {
        const val = raw as ProvisioningTaskStatusCode;
        const cfg = STATUS_CONFIG[val] ?? STATUS_CONFIG['2'];
        return <Tag color={cfg.color} icon={cfg.icon}>{cfg.label}</Tag>;
      },
    },
    {
      key: 'executeProcedure',
      title: t('provision.stepProgress'),
      dataIndex: 'executeProcedure',
      width: 120,
      render: (val: unknown) => (val as string) || '-',
    },
    {
      key: 'startTime',
      title: t('provision.startTime'),
      dataIndex: 'startTime',
      width: 170,
      render: (val: unknown) => {
        const s = val as string;
        return s ? new Date(s).toLocaleString('zh-CN') : '-';
      },
    },
    {
      key: 'endTime',
      title: t('provision.endTime'),
      dataIndex: 'endTime',
      width: 170,
      render: (val: unknown) => {
        const s = val as string;
        return s ? new Date(s).toLocaleString('zh-CN') : '-';
      },
    },
    {
      key: 'retryCount',
      title: t('provision.retry'),
      dataIndex: 'retryCount',
      width: 90,
      render: (_: unknown, record) => `${record.retryCount}/${record.maxRetries}`,
    },
    {
      key: 'failureReason',
      title: t('provision.failureReason'),
      dataIndex: 'failureReason',
      width: 200,
      ellipsis: true,
      render: (val: unknown) => translateFailureReason(val as string),
    },
  ], [t, STATUS_CONFIG, translateFailureReason, handleRetryTask]);

  // Filtered policies (local config state)
  const filteredPolicies = useMemo(() => {
    let result = policies;
    if (policyProductClass) {
      result = result.filter(p => p.productClass === policyProductClass);
    }
    if (policySearchText) {
      const search = policySearchText.toLowerCase();
      result = result.filter(p =>
        p.policyName.toLowerCase().includes(search) ||
        p.productClass.toLowerCase().includes(search) ||
        p.targetVersion?.[0]?.toLowerCase().includes(search)
      );
    }
    return result;
  }, [policies, policyProductClass, policySearchText]);

  // Tasks from real API → view model. Status '2'/'3'/'4' are filtered
  // client-side (the API only maps the terminal completed/failed states).
  const allTasks = useMemo<ProvisioningExecuteView[]>(
    () => (taskData?.items ?? []).map(mapTaskToExecuteView),
    [taskData]
  );

  const filteredTasks = useMemo(() => {
    let result = allTasks;
    if (taskStatus && taskStatus !== '0' && taskStatus !== '1') {
      // running / pending / skipped — backend has no exact filter, narrow locally
      result = result.filter(item => item.status === taskStatus);
    }
    if (taskSearchText) {
      const search = taskSearchText.toLowerCase();
      result = result.filter(item => item.deviceId.toLowerCase().includes(search));
    }
    return result;
  }, [allTasks, taskStatus, taskSearchText]);

  const successCount = useMemo(() => allTasks.filter(item => item.status === '0').length, [allTasks]);
  const failCount = useMemo(() => allTasks.filter(item => item.status === '1').length, [allTasks]);

  const totalTasks = taskData?.total ?? filteredTasks.length;

  // Batch actions for task table
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

  const detailTask = useMemo(
    () => filteredTasks.find(item => item.taskId === detailTaskId),
    [filteredTasks, detailTaskId]
  );

  return (
    <ListPageLayout
      title={t('provision.plugAndPlay')}
      extra={
        <Button type="primary" icon={<PlusOutlined />} onClick={handleAddPolicy}>
          {t('common.add')}
        </Button>
      }
    >
      <div className="plug-and-play-container" style={{ height: '100%', display: 'flex', flexDirection: 'column', gap: 12 }}>
        {/* Policy List Section (front-end config state — no backend policies API yet, see issue #140) */}
        <Card
          size="small"
          styles={{ body: { padding: 0, display: 'flex', flexDirection: 'column' } }}
          style={{ flexShrink: 0 }}
        >
          {/* Policy table with title in toolbar */}
          <div style={{ flex: 1, minHeight: 0 }}>
            <DataTable<Policy>
              tableId="policy-table"
              columns={policyColumns}
              dataSource={filteredPolicies}
              rowKey="policyId"
              total={filteredPolicies.length}
              showPagination
              currentPage={1}
              defaultDensity="default"
              showRowNumber
              rowNumberTitle={t('table.rowNumber')}
              scroll={{ x: 'max-content', y: 200 }}
              extraToolbarLeft={<Text className={styles.cardHeaderTitle}>{t('provision.policyList')}</Text>}
              extraToolbarRight={
                <Space>
                  <Select
                    placeholder={t('provision.productClass')}
                    value={policyProductClass || undefined}
                    onChange={(value) => setPolicyProductClass(value || '')}
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
          </div>
        </Card>

        {/* Execute Status Section (real provisioning/tasks API) */}
        <Card
          size="small"
          styles={{ body: { padding: 0, display: 'flex', flexDirection: 'column' } }}
          style={{ flex: 1, minHeight: 0 }}
        >
        {/* 执行状态标题 + 成功/失败计数 */}
        <div className={styles.cardHeader}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', width: '100%' }}>
            <Space size={16}>
              <Text className={styles.cardHeaderTitle}>{t('provision.executeStatus')}</Text>
            </Space>
            <Space size={12}>
              <div className={`${styles.statsBadge} ${styles.statsSuccess}`}>
                <CheckCircleOutlined className={styles.statsIcon} />
                <span>{successCount}</span>
              </div>
              <div className={`${styles.statsBadge} ${styles.statsFailed}`}>
                <CloseCircleOutlined className={styles.statsIcon} />
                <span>{failCount}</span>
              </div>
            </Space>
          </div>
        </div>

        {/* Task table with filters in toolbar */}
        <div style={{ flex: 1, minHeight: 0 }}>
          <DataTable<ProvisioningExecuteView>
            tableId="task-table"
            columns={taskColumns}
            dataSource={filteredTasks}
            rowKey="taskId"
            loading={tasksLoading}
            total={totalTasks}
            currentPage={taskPage}
            pageSize={taskPageSize}
            onPageChange={(p, s) => { setTaskPage(p); setTaskPageSize(s); }}
            defaultDensity="default"
            showRowNumber
            rowNumberTitle={t('table.rowNumber')}
            selectable
            selectedRowKeys={selectedTaskIds}
            onSelectionChange={(keys) => setSelectedTaskIds(keys as string[])}
            batchActions={taskBatchActions}
            scroll={{ x: 'max-content', y: 220 }}
            extraToolbarRight={
              <Space>
                <Select
                  placeholder={t('table.status')}
                  value={taskStatus || undefined}
                  onChange={(value) => { setTaskStatus(value || ''); setTaskPage(1); }}
                  allowClear
                  style={{ width: 110 }}
                  options={[
                    { label: t('status.success'), value: '0' },
                    { label: t('status.failed'), value: '1' },
                    { label: t('status.running'), value: '2' },
                  ]}
                />
                <Input
                  placeholder={t('provision.searchDeviceCode')}
                  prefix={<SearchOutlined />}
                  value={taskSearchText}
                  onChange={(e) => { setTaskSearchText(e.target.value); setTaskPage(1); }}
                  style={{ width: 200 }}
                  allowClear
                />
                <Button icon={<ReloadOutlined />} onClick={handleRefreshTasks}>
                  {t('common.refresh')}
                </Button>
              </Space>
            }
          />
        </div>
      </Card>
      </div>

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
          taskData={detailTask ? {
            status: detailTask.status,
            executeProcedure: detailTask.executeProcedure,
            failureReason: detailTask.failureReason,
            startTime: detailTask.startTime,
            endTime: detailTask.endTime,
          } : undefined}
          onClose={() => setDetailTaskId(null)}
        />
      )}

      {/* Batch Retry Dialog */}
      <BatchRetryDialog
        open={batchRetryOpen}
        taskCount={selectedTaskIds.length}
        onClose={() => setBatchRetryOpen(false)}
        onConfirm={async (_includeSuccess) => {
          // Backend only allows retrying failed tasks; retry each selected failed task.
          const failedIds = filteredTasks
            .filter(item => selectedTaskIds.includes(item.taskId) && item.status === '1')
            .map(item => item.taskId);
          const results = await Promise.allSettled(
            failedIds.map(id => retryTaskMutation.mutateAsync(id))
          );
          const ok = results.filter(r => r.status === 'fulfilled').length;
          setBatchRetryOpen(false);
          setSelectedTaskIds([]);
          if (ok > 0) {
            message.success(t('provision.addedDevices', { count: ok }));
          } else {
            message.warning(t('provision.cannotRetryHint'));
          }
        }}
      />
    </ListPageLayout>
  );
}
