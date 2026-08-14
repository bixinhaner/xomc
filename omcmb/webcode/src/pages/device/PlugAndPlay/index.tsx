import { useState, useMemo, useCallback, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { Button, Space, Tag, Switch, Dropdown, Input, App, Typography, Card, Select, DatePicker, Radio } from 'antd';
import type { MenuProps } from 'antd';
import type { Dayjs } from 'dayjs';
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
  useRetryPlugAndPlayTask,
  usePlugAndPlayPolicies,
  useSetPlugAndPlayPolicyEnabled,
  useDeletePlugAndPlayPolicy,
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
import { formatSystemTime } from '@core/utils/systemTime';
import { useAppStore } from '@core/store/appStore';
import { getI18nText } from '@core/utils/i18nText';
import { useProductList } from '@core/hooks/api/useProducts';
import { toSupportedProductNameOptions } from './productClassOptions';
import ProductClassSelect from './components/ProductClassSelect';
import AutoRefreshDropdown from '@/pages/alarm/components/AutoRefreshDropdown';
import { getPolicyActionAvailability } from './policyActionAvailability';
import { findEnabledPolicyProductConflict } from './policyEnableConflict';
import { serializeTaskTimeRange } from './taskTimeFilter';

const { Text } = Typography;

// Types
type ExecuteType = '0' | '1'; // 0-auto, 1-manual
type TaskModule = '' | 'software_upgrade' | 'license' | 'self_config';

interface Policy {
  policyId: string;
  policyName: string;
  policyNameI18n?: Record<string, string>;
  productName: string;
  productNames: string[];
  productClass: string;
  productClasses: string[];
  executeType: ExecuteType;
  selfStartEnable: '0' | '1';
  upgradeEnable: '0' | '1';
  targetVersion: string[];
  licenseEnable: '0' | '1';
  selfConfigEnable: '0' | '1';
  createTime: string;
  updateTime: string;
}

const PAGE_SIZE = 10;

export default function PlugAndPlay() {
  const t = useT();
  const navigate = useNavigate();
  const { message } = App.useApp();
  const appLocale = useAppStore((s) => s.locale);
  const systemTimezone = useAppStore((s) => s.systemTimezone);
  const { data: productCatalog, isLoading: productCatalogLoading } = useProductList();
  const productClassOptions = useMemo(
    () => toSupportedProductNameOptions(productCatalog?.items),
    [productCatalog?.items],
  );

  // Policy filter state
  const [policyProductName, setPolicyProductName] = useState<string>('');
  const [policySearchText, setPolicySearchText] = useState('');
  const { data: policyData, isLoading: policiesLoading } = usePlugAndPlayPolicies({
    page: 1, pageSize: 100,
    productName: policyProductName || undefined,
    search: policySearchText || undefined,
  });
  const setPolicyEnabledMutation = useSetPlugAndPlayPolicyEnabled();
  const deletePolicyMutation = useDeletePlugAndPlayPolicy();
  const policies = useMemo<Policy[]>(() => (policyData?.items ?? []).map((p) => ({
    policyId: p.id,
    policyName: p.name,
    productName: p.productNames.join(', ') || p.productClasses.join(', '),
    productNames: p.productNames.length ? p.productNames : p.productClasses,
    productClass: p.productClass,
    productClasses: p.productClasses,
    executeType: p.executeType === 'auto' ? '0' : '1',
    selfStartEnable: p.enabled ? '1' : '0',
    upgradeEnable: p.upgradeEnabled ? '1' : '0',
    targetVersion: p.targetVersion ? [p.targetVersion] : [],
    licenseEnable: p.licenseEnabled ? '1' : '0',
    selfConfigEnable: p.selfConfigEnabled ? '1' : '0',
    createTime: p.createdAt,
    updateTime: p.updatedAt,
  })), [policyData]);

  // Task filter state (执行状态 — real provisioning/tasks API)
  const [taskStatus, setTaskStatus] = useState<string>('');
  const [taskSearchText, setTaskSearchText] = useState('');
  const [taskProductName, setTaskProductName] = useState('');
  const [taskModule, setTaskModule] = useState<TaskModule>('');
  const [taskTimeRange, setTaskTimeRange] = useState<[Dayjs, Dayjs] | null>(null);
  const [taskPage, setTaskPage] = useState(1);
  const [taskPageSize, setTaskPageSize] = useState(PAGE_SIZE);
  const [taskAutoRefresh, setTaskAutoRefresh] = useState(false);
  const [taskRefreshInterval, setTaskRefreshInterval] = useState(30);

  // Map UI status code → backend lifecycle filter. Only the terminal states map
  // 1:1; the page's "running" bucket spans several backend states, so it is
  // filtered client-side rather than passed to the API.
  const backendStatusFilter = useMemo(() => {
    if (taskStatus === '0') return 'completed';
    if (taskStatus === '1') return 'failed';
    if (taskStatus === '2') return 'running';
    return undefined;
  }, [taskStatus]);

  const taskTimeFilter = useMemo(
    () => serializeTaskTimeRange(taskTimeRange, systemTimezone),
    [taskTimeRange, systemTimezone],
  );

  const {
    data: taskData,
    isLoading: tasksLoading,
    refetch: refetchTasks,
    isFetching: tasksFetching,
  } = useProvisioningTasks({
    page: taskPage,
    pageSize: taskPageSize,
    status: backendStatusFilter,
    policyOnly: true,
    search: taskSearchText || undefined,
    productName: taskProductName || undefined,
    module: taskModule || undefined,
    startedAfter: taskTimeFilter.startedAfter,
    startedBefore: taskTimeFilter.startedBefore,
  }, {
    refetchInterval: taskAutoRefresh ? taskRefreshInterval * 1000 : false,
  });

  useEffect(() => {
    if (!taskAutoRefresh) return;
    void refetchTasks();
  }, [taskAutoRefresh, taskRefreshInterval, refetchTasks]);

  const retryTaskMutation = useRetryPlugAndPlayTask();

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

  const handlePolicyMenuClick = useCallback(async (key: string, record: Policy) => {
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
        try {
          await deletePolicyMutation.mutateAsync(record.policyId);
          message.success(t('common.deleteSuccess'));
        } catch {
          message.error(t('common.operationFailed'));
        }
        break;
    }
  }, [navigate, message, t, deletePolicyMutation]);

  const handlePolicySwitch = useCallback(async (record: Policy, checked: boolean) => {
    if (checked) {
      const conflict = findEnabledPolicyProductConflict({
        policyId: record.policyId,
        productNames: record.productNames,
        enabled: false,
      }, policies.map((policy) => ({
        policyId: policy.policyId,
        productNames: policy.productNames,
        enabled: policy.selfStartEnable === '1',
      })));
      if (conflict) {
        message.error(t('provision.enabledPolicyProductConflict'));
        return;
      }
    }
    try {
      await setPolicyEnabledMutation.mutateAsync({ id: record.policyId, enabled: checked });
      message.success(t('common.success'));
    } catch (error) {
      const status = (error as { response?: { status?: number } })?.response?.status;
      message.error(status === 409
        ? t('provision.enabledPolicyProductConflict')
        : t('common.operationFailed'));
    }
  }, [message, policies, t, setPolicyEnabledMutation]);

  // Handlers - Task (real API)
  const handleRetryTask = useCallback(async (record: ProvisioningExecuteView) => {
    try {
      if (!record.policyId) throw new Error('missing policy ID');
      await retryTaskMutation.mutateAsync({
        policyId: record.policyId,
        deviceId: record.deviceId,
      });
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
    void refetchTasks();
  }, [message, refetchTasks, t]);

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
      render: (_, record) => {
        const actions = getPolicyActionAvailability({ enabled: record.selfStartEnable === '1' });
        return (
          <Space size={4}>
            <Button type="link" size="small" onClick={() => handlePolicyMenuClick('info', record)}>
              {t('common.detail')}
            </Button>
            <Dropdown
              menu={{
                items: [
                  { key: 'edit', label: t('common.edit'), icon: <EditOutlined />, disabled: !actions.edit },
                  { key: 'detect', label: t('provision.detect'), icon: <ScanOutlined />, disabled: !actions.detect },
                  { type: 'divider' as const },
                  { key: 'delete', label: t('common.delete'), icon: <DeleteOutlined />, danger: true, disabled: !actions.delete },
                ] as MenuProps['items'],
                onClick: ({ key }) => handlePolicyMenuClick(key, record),
              }}
              trigger={['click']}
            >
              <Button type="text" size="small" icon={<MoreOutlined />} onClick={(e) => e.stopPropagation()} />
            </Dropdown>
          </Space>
        );
      },
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
      key: 'productName',
      title: t('provision.productName'),
      dataIndex: 'productName',
      width: 120,
      ellipsis: true,
    },
    {
      key: 'policyName',
      title: t('provision.policyName'),
      dataIndex: 'policyName',
      ellipsis: true,
      render: (_val, record) => getI18nText(record.policyNameI18n, appLocale, record.policyName),
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
  ], [appLocale, t, handlePolicyMenuClick, handlePolicySwitch]);

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
      key: 'serialNumber',
      title: t('provision.serialNumber'),
      dataIndex: 'serialNumber',
      width: 280,
      mono: true,
      ellipsis: true,
    },
    {
      key: 'productName',
      title: t('provision.productName'),
      dataIndex: 'productName',
      width: 140,
      ellipsis: true,
      render: (val: unknown) => (val as string) || '-',
    },
    {
      key: 'policyName',
      title: t('provision.policyName'),
      dataIndex: 'policyName',
      width: 180,
      ellipsis: true,
      render: (val: unknown) => (val as string) || '-',
    },
    {
      key: 'executeType',
      title: t('provision.executeType'),
      dataIndex: 'executeType',
      width: 110,
      render: (val: unknown) => val === 'auto'
        ? t('provision.autoExecute')
        : val === 'manual' ? t('provision.manualExecute') : '-',
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
        return s ? formatSystemTime(s) : '-';
      },
    },
    {
      key: 'endTime',
      title: t('provision.endTime'),
      dataIndex: 'endTime',
      width: 170,
      render: (val: unknown) => {
        const s = val as string;
        return s ? formatSystemTime(s) : '-';
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

  // The server applies the same filters; this local pass keeps UI responsive
  // while a new query is in flight.
  const filteredPolicies = useMemo(() => {
    let result = policies;
    if (policyProductName) {
      result = result.filter(p => p.productNames.includes(policyProductName));
    }
    if (policySearchText) {
      const search = policySearchText.toLowerCase();
      result = result.filter(p =>
        getI18nText(p.policyNameI18n, appLocale, p.policyName).toLowerCase().includes(search) ||
        p.productNames.some(productName => productName.toLowerCase().includes(search)) ||
        p.targetVersion?.[0]?.toLowerCase().includes(search)
      );
    }
    return result;
  }, [appLocale, policies, policyProductName, policySearchText]);

  // Tasks from real API → view model. Filtering and pagination stay on the
  // server so module/time/product queries cover the full result set.
  const allTasks = useMemo<ProvisioningExecuteView[]>(
    () => (taskData?.items ?? []).map(mapTaskToExecuteView),
    [taskData]
  );

  const filteredTasks = allTasks;

  const successCount = taskData?.statusCounts.completed ?? 0;
  const failCount = taskData?.statusCounts.failed ?? 0;

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
        {/* Persisted plug-and-play policies */}
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
              loading={policiesLoading}
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
                  <ProductClassSelect
                    placeholder={t('provision.productName')}
                    value={policyProductName || undefined}
                    onChange={(value) => setPolicyProductName(value || '')}
                    allowClear
                    loading={productCatalogLoading}
                    style={{ width: 240 }}
                    options={productClassOptions}
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
            extraToolbarLeft={
              <Radio.Group
                optionType="button"
                buttonStyle="solid"
                value={taskModule}
                onChange={(event) => { setTaskModule(event.target.value as TaskModule); setTaskPage(1); }}
                options={[
                  { label: t('provision.allTasks'), value: '' },
                  { label: t('provision.softwareUpgrade'), value: 'software_upgrade' },
                  { label: t('provision.license'), value: 'license' },
                  { label: t('provision.selfConfig'), value: 'self_config' },
                ]}
              />
            }
            extraToolbarRight={
              <Space size={8}>
                <Input
                  placeholder={t('provision.searchPlaceholder')}
                  prefix={<SearchOutlined />}
                  value={taskSearchText}
                  onChange={(e) => { setTaskSearchText(e.target.value); setTaskPage(1); }}
                  style={{ width: 210 }}
                  allowClear
                />
                <DatePicker.RangePicker
                  showTime
                  value={taskTimeRange}
                  onChange={(value) => {
                    setTaskTimeRange(value ? [value[0]!, value[1]!] : null);
                    setTaskPage(1);
                  }}
                  style={{ width: 310 }}
                />
                <ProductClassSelect
                  placeholder={t('provision.productName')}
                  value={taskProductName || undefined}
                  onChange={(value) => { setTaskProductName(value || ''); setTaskPage(1); }}
                  allowClear
                  loading={productCatalogLoading}
                  style={{ width: 180 }}
                  options={productClassOptions}
                />
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
                <Button
                  size="small"
                  icon={<ReloadOutlined />}
                  loading={tasksFetching}
                  onClick={handleRefreshTasks}
                >
                  {t('common.refresh')}
                </Button>
                <AutoRefreshDropdown
                  enabled={taskAutoRefresh}
                  intervalSeconds={taskRefreshInterval}
                  onEnabledChange={setTaskAutoRefresh}
                  onIntervalChange={setTaskRefreshInterval}
                  spinning={taskAutoRefresh && tasksFetching}
                  size="small"
                />
              </Space>
            }
            hideRealtime
            hideRefresh
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
            currentStep: detailTask.currentStep,
            totalSteps: detailTask.totalSteps,
            currentStepName: detailTask.currentStepName,
            technology: detailTask.technology,
            xmlFileId: detailTask.xmlFileId,
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
          const failedTasks = filteredTasks.filter(item =>
            selectedTaskIds.includes(item.taskId) && item.status === '1' && item.policyId
          );
          const results = await Promise.allSettled(
            failedTasks.map(item => retryTaskMutation.mutateAsync({
              policyId: item.policyId,
              deviceId: item.deviceId,
            }))
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
