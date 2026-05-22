import { useState, useMemo, useCallback } from 'react';
import {
  App,
  Button,
  Card,
  Space,
  Tag,
  Typography,
  Dropdown,
  Modal,
  Drawer,
  Form,
  Select,
  Table,
  Input,
  Alert,
  Checkbox,
  DatePicker,
} from 'antd';
import {
  PlusOutlined,
  MoreOutlined,
  EyeOutlined,
  StopOutlined,
  DownloadOutlined,
  DeleteOutlined,
} from '@ant-design/icons';
import type { MenuProps } from 'antd';
import type { Dayjs } from 'dayjs';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn, BatchAction } from '@/components/DataTable';
import type { ColumnsType } from 'antd/es/table';
import { useT } from '@/hooks/useT';

// 任务状态类型
type TaskStatus = 0 | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8;

// 执行方式类型
type ExecuteType = 'Immediately' | 'Period';

interface DeviceLogTask {
  id: string;
  deviceCode: string;
  deviceName: string;
  taskStatus: TaskStatus;
  logType: string;
  executeType: ExecuteType;
  reportPeriod?: string;
  failureReason?: string;
  updateTime: string;
}

// 日志结果项
interface LogResultItem {
  id: string;
  fileName: string;
  fileSize: string;
  uploadTime: string;
  status: 'success' | 'failed';
}

// 设备项（用于设备选择）
interface DeviceItem {
  id: string;
  deviceCode: string;
  deviceName: string;
  productClass: string;
}

// 状态配置 - use i18n keys
const statusKeys: Record<TaskStatus, { key: string; color: string }> = {
  0: { key: 'status.waiting', color: 'blue' },
  1: { key: 'status.inProgress', color: 'processing' },
  2: { key: 'status.success', color: 'success' },
  3: { key: 'status.failed', color: 'error' },
  4: { key: 'status.terminated', color: 'default' },
  5: { key: 'log.stoppingUpload', color: 'processing' },
  6: { key: 'log.stopUploadSuccess', color: 'default' },
  7: { key: 'log.stopUploadFailed', color: 'processing' },
  8: { key: 'log.periodicUploadSuccess', color: 'blue' },
};

// Mock 数据
const mockDeviceLogs: DeviceLogTask[] = [
  { id: '1', deviceCode: 'ENB00001', deviceName: '北京朝阳1号站', taskStatus: 2, logType: '系统日志', executeType: 'Immediately', updateTime: '2026-03-26 10:00:00' },
  { id: '2', deviceCode: 'GNB00002', deviceName: '上海浦东2号站', taskStatus: 1, logType: '调试日志', executeType: 'Immediately', updateTime: '2026-03-26 09:30:00' },
  { id: '3', deviceCode: 'ENB00003', deviceName: '广州天河3号站', taskStatus: 0, logType: '系统日志', executeType: 'Period', reportPeriod: '每天', updateTime: '2026-03-26 08:00:00' },
  { id: '4', deviceCode: 'ENB00004', deviceName: '深圳南山4号站', taskStatus: 3, logType: '性能日志', executeType: 'Immediately', failureReason: '设备离线', updateTime: '2026-03-25 16:00:00' },
  { id: '5', deviceCode: 'GNB00005', deviceName: '杭州西湖5号站', taskStatus: 4, logType: '系统日志', executeType: 'Immediately', updateTime: '2026-03-25 14:00:00' },
];

// Mock 日志结果数据
const mockLogResults: LogResultItem[] = [
  { id: '1', fileName: 'ENB00001_system_20260326_100000.tar.gz', fileSize: '15.2 MB', uploadTime: '2026-03-26 10:05:32', status: 'success' },
  { id: '2', fileName: 'ENB00001_debug_20260326_100000.tar.gz', fileSize: '8.7 MB', uploadTime: '2026-03-26 10:05:45', status: 'success' },
];

// Mock 可选设备列表
const mockAvailableDevices: DeviceItem[] = [
  { id: '1', deviceCode: 'ENB00001', deviceName: '北京朝阳1号站', productClass: '4G LTE' },
  { id: '2', deviceCode: 'GNB00002', deviceName: '上海浦东2号站', productClass: '5G NR' },
  { id: '3', deviceCode: 'ENB00003', deviceName: '广州天河3号站', productClass: '4G LTE' },
  { id: '4', deviceCode: 'ENB00004', deviceName: '深圳南山4号站', productClass: '4G LTE' },
  { id: '5', deviceCode: 'GNB00005', deviceName: '杭州西湖5号站', productClass: '5G NR' },
  { id: '6', deviceCode: 'ENB00006', deviceName: '成都高新6号站', productClass: '4G LTE' },
  { id: '7', deviceCode: 'GNB00007', deviceName: '武汉光谷7号站', productClass: '5G NR' },
];

// 产品类型选项
const productClassOptions = [
  { label: '4G LTE', value: '4G LTE' },
  { label: '5G NR', value: '5G NR' },
];

export default function DeviceLog() {
  const t = useT();
  const { modal, message } = App.useApp();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [createDrawerVisible, setCreateDrawerVisible] = useState(false);
  const [resultDrawerVisible, setResultDrawerVisible] = useState(false);
  const [selectedTask, setSelectedTask] = useState<DeviceLogTask | null>(null);
  const [logResults, setLogResults] = useState<LogResultItem[]>([]);
  const [deleteModalVisible, setDeleteModalVisible] = useState(false);
  const [taskToDelete, setTaskToDelete] = useState<DeviceLogTask | null>(null);
  const [form] = Form.useForm();

  // 新建任务相关状态
  const [selectedDevices, setSelectedDevices] = useState<DeviceItem[]>([]);
  const [productClassFilter, setProductClassFilter] = useState<string>('');
  const [addDeviceModalVisible, setAddDeviceModalVisible] = useState(false);
  const [selectedNewDevices, setSelectedNewDevices] = useState<React.Key[]>([]);
  const [addDeviceKeyword, setAddDeviceKeyword] = useState('');
  const [batchInputModalVisible, setBatchInputModalVisible] = useState(false);
  const [batchInputValue, setBatchInputValue] = useState('');

  // 过滤数据
  const filteredData = useMemo(() => {
    return mockDeviceLogs.filter((row) => {
      if (filters.deviceCode && typeof filters.deviceCode === 'string') {
        if (!row.deviceCode.toLowerCase().includes(filters.deviceCode.toLowerCase())) {
          return false;
        }
      }
      if (filters.status !== undefined && filters.status !== null) {
        if (row.taskStatus !== filters.status) {
          return false;
        }
      }
      // 时间范围过滤
      if (filters.timeRange && Array.isArray(filters.timeRange)) {
        const [startTime, endTime] = filters.timeRange as [Dayjs, Dayjs];
        const rowTime = row.updateTime;
        if (startTime && endTime) {
          const rowDate = new Date(rowTime);
          if (rowDate < startTime.toDate() || rowDate > endTime.toDate()) {
            return false;
          }
        }
      }
      return true;
    });
  }, [filters]);

  // 可添加的设备列表（同产品类型且不在已选列表中）
  const availableDevices = useMemo(() => {
    const existingIds = new Set(selectedDevices.map((d) => d.id));
    return mockAvailableDevices.filter((d) => !existingIds.has(d.id));
  }, [selectedDevices]);

  // 搜索过滤后的可选设备
  const filteredAvailableDevices = useMemo(() => {
    let devices = availableDevices;
    if (productClassFilter) {
      devices = devices.filter((d) => d.productClass === productClassFilter);
    }
    if (addDeviceKeyword) {
      const keyword = addDeviceKeyword.toLowerCase();
      devices = devices.filter(
        (d) =>
          d.deviceCode.toLowerCase().includes(keyword) ||
          d.deviceName.toLowerCase().includes(keyword)
      );
    }
    return devices;
  }, [availableDevices, productClassFilter, addDeviceKeyword]);

  // 筛选字段
  const filterFields: FilterField[] = useMemo(() => [
    { name: 'deviceCode', label: t('log.deviceCode'), type: 'input', placeholder: t('log.inputDeviceCode') },
    {
      name: 'status',
      label: t('log.collectionStatus'),
      type: 'select',
      options: [
        { label: t('status.waiting'), value: 0 },
        { label: t('status.inProgress'), value: 1 },
        { label: t('status.success'), value: 2 },
        { label: t('status.failed'), value: 3 },
        { label: t('status.terminated'), value: 4 },
      ],
    },
    { name: 'timeRange', label: t('common.timeRange'), type: 'date-range', span: 2 },
  ], [t]);

  // 获取操作菜单
  const getActionMenu = (record: DeviceLogTask): MenuProps['items'] => {
    const status = record.taskStatus;
    const items: MenuProps['items'] = [];

    // 结果 - 仅成功时可用
    if (status === 2) {
      items.push({
        key: 'view',
        label: t('log.result'),
        icon: <EyeOutlined />,
        onClick: () => handleViewResult(record),
      });
    }

    // 终止任务 - 状态为 0/1/7/8 时可用
    if ([0, 1, 7, 8].includes(status)) {
      items.push({
        key: 'terminate',
        label: t('common.terminate'),
        icon: <StopOutlined />,
        onClick: () => handleTerminate(record),
      });
    }

    // 下载 - 仅成功时可用
    if (status === 2) {
      items.push({
        key: 'download',
        label: t('common.download'),
        icon: <DownloadOutlined />,
        onClick: () => handleDownload(record),
      });
    }

    // 删除 - 状态为 2/3/4 时可用
    if ([2, 3, 4].includes(status)) {
      items.push({
        key: 'delete',
        label: t('common.delete'),
        icon: <DeleteOutlined />,
        danger: true,
        onClick: () => handleDeleteClick(record),
      });
    }

    return items;
  };

  // 查看结果
  const handleViewResult = (record: DeviceLogTask) => {
    setSelectedTask(record);
    // 模拟加载结果数据
    setLogResults(mockLogResults);
    setResultDrawerVisible(true);
  };

  // 终止任务
  const handleTerminate = (record: DeviceLogTask) => {
    modal.confirm({
      title: t('log.confirmTerminate'),
      content: t('log.confirmTerminateContent', { code: record.deviceCode }),
      onOk: () => {
        message.success(t('log.taskTerminated'));
      },
    });
  };

  // 下载
  const handleDownload = (record: DeviceLogTask) => {
    message.success(t('log.downloadingLog', { code: record.deviceCode }));
  };

  // 删除
  const handleDeleteClick = (record: DeviceLogTask) => {
    setTaskToDelete(record);
    setDeleteModalVisible(true);
  };

  const handleDeleteConfirm = () => {
    if (taskToDelete) {
      message.success(t('log.taskDeleted', { code: taskToDelete.deviceCode }));
      setDeleteModalVisible(false);
      setTaskToDelete(null);
    }
  };

  // 批量删除
  const handleBatchDelete = useCallback(
    (ids: React.Key[]) => {
      // 过滤出可删除的任务（状态为 2/3/4：成功、失败、终止）
      const deletableTasks = filteredData.filter(
        (task) => ids.includes(task.id) && [2, 3, 4].includes(task.taskStatus)
      );
      // 收集中状态包括：0-等待、1-进行中、5-正在停止上报、7-停止上报失败、8-周期上报设置成功
      const collectingTasks = filteredData.filter(
        (task) => ids.includes(task.id) && [0, 1, 5, 7, 8].includes(task.taskStatus)
      );

      if (collectingTasks.length > 0) {
        modal.warning({
          title: t('log.cannotDelete'),
          content: t('log.cannotDeleteContent', { count: collectingTasks.length }),
        });
        return;
      }

      if (deletableTasks.length === 0) {
        message.warning(t('log.selectDeletableTask'));
        return;
      }

      modal.confirm({
        title: t('log.batchDelete'),
        content: t('log.batchDeleteConfirm', { count: deletableTasks.length }),
        okText: t('common.confirm'),
        cancelText: t('common.cancel'),
        onOk: () => {
          message.success(t('log.batchDeleteSuccess', { count: deletableTasks.length }));
          setSelectedRowKeys([]);
        },
      });
    },
    [filteredData, modal, message, t]
  );

  // 批量下载
  const handleBatchDownload = useCallback(
    (ids: React.Key[]) => {
      // 过滤出可下载的任务（仅成功状态）
      const downloadableTasks = filteredData.filter(
        (task) => ids.includes(task.id) && task.taskStatus === 2
      );

      if (downloadableTasks.length === 0) {
        message.warning(t('log.selectDownloadableTask'));
        return;
      }

      modal.confirm({
        title: t('log.batchDownload'),
        content: t('log.batchDownloadConfirm', { count: downloadableTasks.length }),
        okText: t('common.confirm'),
        cancelText: t('common.cancel'),
        onOk: () => {
          message.success(t('log.batchDownloadSuccess', { count: downloadableTasks.length }));
        },
      });
    },
    [filteredData, modal, message, t]
  );

  // 从已选列表中移除设备
  const handleRemoveDevice = (deviceId: string) => {
    setSelectedDevices((prev) => prev.filter((d) => d.id !== deviceId));
  };

  // 打开添加设备弹窗
  const handleOpenAddDeviceModal = () => {
    setSelectedNewDevices([]);
    setAddDeviceKeyword('');
    setAddDeviceModalVisible(true);
  };

  // 全选/取消全选
  const handleSelectAllDevices = (checked: boolean) => {
    if (checked) {
      setSelectedNewDevices(filteredAvailableDevices.map((d) => d.id));
    } else {
      setSelectedNewDevices([]);
    }
  };

  // 确认添加设备
  const handleConfirmAddDevices = () => {
    if (selectedNewDevices.length === 0) {
      message.warning(t('log.selectAtLeastOne'));
      return;
    }
    const newDevices = availableDevices.filter((d) => selectedNewDevices.includes(d.id));
    setSelectedDevices((prev) => [...prev, ...newDevices]);
    setAddDeviceModalVisible(false);
    setSelectedNewDevices([]);
    setAddDeviceKeyword('');
    message.success(t('log.addedDeviceCount', { count: newDevices.length }));
  };

  // 打开批量输入弹窗
  const handleOpenBatchInput = () => {
    setBatchInputValue('');
    setBatchInputModalVisible(true);
  };

  // 确认批量输入
  const handleConfirmBatchInput = () => {
    if (!batchInputValue.trim()) {
      message.warning(t('log.selectAtLeastOneCode'));
      return;
    }
    const codes = batchInputValue
      .split(/[\n,;，；]/)
      .map((s) => s.trim())
      .filter((s) => s);
    const existingIds = new Set(selectedDevices.map((d) => d.id));
    const newDevices: DeviceItem[] = [];
    for (const code of codes) {
      const found = mockAvailableDevices.find((d) => d.deviceCode === code);
      if (found && !existingIds.has(found.id)) {
        newDevices.push(found);
        existingIds.add(found.id);
      } else if (!found) {
        // 手动添加未知设备
        newDevices.push({
          id: `manual-${Date.now()}-${code}`,
          deviceCode: code,
          deviceName: t('log.manualDevice'),
          productClass: t('log.unknownProductClass'),
        });
      }
    }
    if (newDevices.length > 0) {
      setSelectedDevices((prev) => [...prev, ...newDevices]);
      message.success(t('log.addedDeviceCount', { count: newDevices.length }));
    }
    setBatchInputModalVisible(false);
    setBatchInputValue('');
  };

  // 新建任务
  const handleCreateTask = () => {
    form.validateFields().then((values) => {
      if (selectedDevices.length === 0) {
        message.warning(t('log.selectAtLeastOne'));
        return;
      }
      console.log('创建任务:', { ...values, devices: selectedDevices });
      message.success(t('log.taskCreated', { count: selectedDevices.length }));
      setCreateDrawerVisible(false);
      form.resetFields();
      setSelectedDevices([]);
      setProductClassFilter('');
    });
  };

  // 打开新建抽屉时重置状态
  const handleOpenCreateDrawer = () => {
    setSelectedDevices([]);
    setProductClassFilter('');
    setCreateDrawerVisible(true);
  };

  // 批量操作
  const batchActions: BatchAction[] = useMemo(
    () => [
      {
        key: 'delete',
        label: t('log.batchDelete'),
        icon: <DeleteOutlined />,
        danger: true,
        onClick: handleBatchDelete,
      },
      {
        key: 'download',
        label: t('log.batchDownload'),
        icon: <DownloadOutlined />,
        onClick: handleBatchDownload,
      },
    ],
    [handleBatchDelete, handleBatchDownload, t]
  );

  // 表格列
  const columns: DataTableColumn<DeviceLogTask>[] = useMemo(() => [
    {
      key: 'operation',
      title: t('table.operation'),
      width: 100,
      fixed: 'right',
      render: (_: unknown, record: DeviceLogTask) => {
        const items = getActionMenu(record) ?? [];
        if (items.length === 0) return '-';
        return (
          <Dropdown menu={{ items }} trigger={['click']}>
            <Button type="link" size="small" icon={<MoreOutlined />} />
          </Dropdown>
        );
      },
    },
    {
      key: 'deviceCode',
      title: t('log.deviceCode'),
      dataIndex: 'deviceCode',
      width: 140,
      render: (val: unknown) => <Typography.Text style={{ fontFamily: 'monospace' }}>{val as string}</Typography.Text>,
    },
    {
      key: 'deviceName',
      title: t('log.logDeviceName'),
      dataIndex: 'deviceName',
      width: 180,
      ellipsis: true,
    },
    {
      key: 'taskStatus',
      title: t('log.collectionStatus'),
      dataIndex: 'taskStatus',
      width: 180,
      render: (raw: unknown) => {
        const val = raw as TaskStatus;
        const cfg = statusKeys[val];
        return <Tag color={cfg.color}>{t(cfg.key)}</Tag>;
      },
    },
    {
      key: 'logType',
      title: t('log.type'),
      dataIndex: 'logType',
      width: 120,
    },
    {
      key: 'executeType',
      title: t('log.executeMethod'),
      dataIndex: 'executeType',
      width: 120,
      render: (val: unknown) => (val as ExecuteType) === 'Immediately' ? t('log.immediateExecute') : t('log.periodicExecute'),
    },
    {
      key: 'reportPeriod',
      title: t('log.period'),
      dataIndex: 'reportPeriod',
      width: 120,
      render: (val: unknown) => (val as string | undefined) ?? '-',
    },
    {
      key: 'failureReason',
      title: t('table.failureReason'),
      dataIndex: 'failureReason',
      width: 200,
      ellipsis: true,
      render: (val: unknown) => (val as string | undefined) ?? '-',
    },
    {
      key: 'updateTime',
      title: t('common.updateTime'),
      dataIndex: 'updateTime',
      width: 180,
    },
  ], [t]);

  // 结果表格列
  const resultColumns: ColumnsType<LogResultItem> = [
    {
      title: t('log.logFileName'),
      dataIndex: 'fileName',
      key: 'fileName',
      ellipsis: true,
    },
    {
      title: t('log.logFileSize'),
      dataIndex: 'fileSize',
      key: 'fileSize',
      width: 120,
    },
    {
      title: t('log.logUploadTime'),
      dataIndex: 'uploadTime',
      key: 'uploadTime',
      width: 180,
    },
    {
      title: t('common.status'),
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: 'success' | 'failed') => (
        <Tag color={status === 'success' ? 'success' : 'error'}>
          {status === 'success' ? t('status.success') : t('status.failed')}
        </Tag>
      ),
    },
    {
      title: t('table.operation'),
      key: 'action',
      width: 80,
      fixed: 'right',
      render: (_: unknown, _record: LogResultItem) => (
        <Button type="link" size="small">
          {t('common.download')}
        </Button>
      ),
    },
  ];

  return (
    <ListPageLayout
      title={t('log.deviceLog')}
      extra={
        <Button type="primary" icon={<PlusOutlined />} onClick={handleOpenCreateDrawer}>
          {t('log.newTask')}
        </Button>
      }
    >
      <FilterBar
        filterId="device-log-filter"
        fields={filterFields}
        onSearch={setFilters}
        onReset={() => setFilters({})}
      />

      <Card
        size="small"
        bordered
        style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}
        styles={{ body: { padding: 0, display: 'flex', flexDirection: 'column', flex: 1, overflow: 'hidden' } }}
      >
        <DataTable<DeviceLogTask>
          tableId="device-log-list"
          columns={columns}
          dataSource={filteredData}
          rowKey="id"
          total={filteredData.length}
          currentPage={1}
          pageSize={20}
          onPageChange={() => {}}
          selectable
          selectedRowKeys={selectedRowKeys}
          onSelectionChange={setSelectedRowKeys}
          batchActions={batchActions}
          scroll={{ x: 'max-content', y: 'calc(100vh - 400px)' }}
          showRowNumber
          rowNumberTitle={t('common.rowNumber')}
        />
      </Card>

      {/* 新建任务抽屉 */}
      <Drawer
        title={t('log.newLogTask')}
        placement="right"
        width={500}
        open={createDrawerVisible}
        onClose={() => setCreateDrawerVisible(false)}
        footer={
          <Space style={{ width: '100%', justifyContent: 'flex-end' }}>
            <Button onClick={() => setCreateDrawerVisible(false)}>{t('common.cancel')}</Button>
            <Button type="primary" onClick={handleCreateTask}>{t('common.confirm')}</Button>
          </Space>
        }
      >
        <Form form={form} layout="vertical" size="small">
          <Form.Item name="executeType" label={t('log.executeMethod')} rules={[{ required: true }]}>
            <Select placeholder={t('log.selectExecuteMethod')}>
              <Select.Option value="Immediately">{t('log.immediateExecute')}</Select.Option>
              <Select.Option value="Period">{t('log.periodicExecute')}</Select.Option>
            </Select>
          </Form.Item>

          {/* 周期执行配置 - 仅周期执行时显示 */}
          <Form.Item noStyle shouldUpdate={(prev, cur) => prev.executeType !== cur.executeType}>
            {({ getFieldValue }) => {
              const executeType = getFieldValue('executeType');
              if (executeType !== 'Period') return null;
              return (
                <>
                  <Form.Item name="periodDateRange" label={t('log.periodDate')} rules={[{ required: true, message: t('log.selectPeriodDate') }]}>
                    <DatePicker.RangePicker
                      style={{ width: '100%' }}
                      placeholder={[t('log.startEndDate'), t('log.endDateString')]}
                    />
                  </Form.Item>
                  <Form.Item name="collectInterval" label={t('log.collectInterval')} rules={[{ required: true, message: t('log.selectCollectInterval') }]}>
                    <Select placeholder={t('log.selectCollectInterval')}>
                      <Select.Option value="15">{t('log.log15Minutes')}</Select.Option>
                      <Select.Option value="30">{t('log.log30Minutes')}</Select.Option>
                      <Select.Option value="60">{t('log.log60Minutes')}</Select.Option>
                    </Select>
                  </Form.Item>
                </>
              );
            }}
          </Form.Item>

          <Form.Item name="logType" label={t('log.logType')} rules={[{ required: true }]}>
            <Select placeholder={t('log.selectLogType')}>
              <Select.Option value="system">{t('log.systemLogType')}</Select.Option>
              <Select.Option value="debug">{t('log.debugLogType')}</Select.Option>
              <Select.Option value="performance">{t('log.performanceLogType')}</Select.Option>
            </Select>
          </Form.Item>

          {/* 产品类型筛选 */}
          <Form.Item label={t('log.productClass')}>
            <Select
              placeholder={t('log.filterByProductClass')}
              allowClear
              style={{ width: '100%' }}
              options={productClassOptions}
              value={productClassFilter || undefined}
              onChange={(val) => setProductClassFilter(val ?? '')}
            />
          </Form.Item>

          {/* 已选设备 */}
          <Form.Item
            label={
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', width: '100%' }}>
                <span>
                  {t('log.selectedDevices')}
                  {' '}
                  <Tag color="blue">{selectedDevices.length} {t('backup.deviceCountUnit', { count: selectedDevices.length }).replace(/\d+/, '').trim()}</Tag>
                </span>
                <Button
                  size="small"
                  icon={<PlusOutlined />}
                  onClick={handleOpenBatchInput}
                >
                  {t('log.batchInput')}
                </Button>
              </div>
            }
          >
            <>
              <div style={{ maxHeight: 200, overflow: 'auto', border: '1px solid #d9d9d9', borderRadius: 6 }}>
                <Table
                  size="small"
                  dataSource={selectedDevices}
                  rowKey="id"
                  pagination={false}
                  columns={[
                    { title: t('log.deviceCode'), dataIndex: 'deviceCode', width: 100 },
                    { title: t('log.logDeviceName'), dataIndex: 'deviceName', ellipsis: true },
                    { title: t('log.productClass'), dataIndex: 'productClass', width: 80 },
                    {
                      title: '',
                      width: 40,
                      render: (_: unknown, record: DeviceItem) => (
                        <Button
                          type="text"
                          size="small"
                          danger
                          icon={<DeleteOutlined />}
                          onClick={() => handleRemoveDevice(record.id)}
                        />
                      ),
                    },
                  ]}
                />
              </div>
              {/* 添加设备按钮 */}
              <div style={{ marginTop: 8 }}>
                <Button
                  type="dashed"
                  icon={<PlusOutlined />}
                  onClick={handleOpenAddDeviceModal}
                  style={{ width: '100%' }}
                >
                  {t('log.addDevice')}
                </Button>
              </div>
            </>
          </Form.Item>
        </Form>
      </Drawer>

      {/* 添加设备弹窗 */}
      <Modal
        title={t('log.addDevice')}
        open={addDeviceModalVisible}
        onCancel={() => setAddDeviceModalVisible(false)}
        onOk={handleConfirmAddDevices}
        okText={t('log.confirmAdd')}
        cancelText={t('common.cancel')}
        width={700}
        okButtonProps={{ disabled: selectedNewDevices.length === 0 }}
      >
        {availableDevices.length === 0 ? (
          <Alert
            type="info"
            showIcon
            message={t('log.noDeviceToAdd')}
            description={t('log.allDevicesAdded')}
          />
        ) : (
          <>
            <Alert
              type="info"
              showIcon
              message={t('log.availableDeviceCount', { total: availableDevices.length, shown: filteredAvailableDevices.length, selected: selectedNewDevices.length })}
              style={{ marginBottom: 16 }}
            />

            {/* 搜索和全选 */}
            <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <Input.Search
                placeholder={t('log.searchDeviceCodeOrName')}
                value={addDeviceKeyword}
                onChange={(e) => setAddDeviceKeyword(e.target.value)}
                style={{ width: 250 }}
                allowClear
              />
              <Checkbox
                checked={selectedNewDevices.length === filteredAvailableDevices.length && filteredAvailableDevices.length > 0}
                indeterminate={selectedNewDevices.length > 0 && selectedNewDevices.length < filteredAvailableDevices.length}
                onChange={(e) => handleSelectAllDevices(e.target.checked)}
              >
                {t('log.selectAllWithCount', { count: filteredAvailableDevices.length })}
              </Checkbox>
            </div>

            <Table
              size="small"
              dataSource={filteredAvailableDevices}
              rowKey="id"
              pagination={false}
              scroll={{ y: 250 }}
              rowSelection={{
                selectedRowKeys: selectedNewDevices,
                onChange: (keys) => setSelectedNewDevices(keys),
              }}
              columns={[
                { title: t('log.deviceCode'), dataIndex: 'deviceCode', width: 120 },
                { title: t('log.logDeviceName'), dataIndex: 'deviceName', ellipsis: true },
                { title: t('log.productClass'), dataIndex: 'productClass', width: 100 },
              ]}
            />
          </>
        )}
      </Modal>

      {/* 批量输入弹窗 */}
      <Modal
        title={t('log.batchInputCode')}
        open={batchInputModalVisible}
        onCancel={() => setBatchInputModalVisible(false)}
        onOk={handleConfirmBatchInput}
        okText={t('log.confirmAdd')}
        cancelText={t('common.cancel')}
      >
        <div style={{ marginBottom: 8 }}>
          {t('log.batchInputPlaceholder')}
        </div>
        <Input.TextArea
          placeholder="ENB00001&#10;ENB00002&#10;ENB00003"
          value={batchInputValue}
          onChange={(e) => setBatchInputValue(e.target.value)}
          rows={8}
        />
      </Modal>

      {/* 结果抽屉 */}
      <Drawer
        title={t('log.taskResult', { code: selectedTask?.deviceCode ?? '' })}
        placement="right"
        width={700}
        open={resultDrawerVisible}
        onClose={() => setResultDrawerVisible(false)}
      >
        {selectedTask && (
          <>
            <div style={{ marginBottom: 16 }}>
              <Space size="large">
                <span><strong>{t('log.logDeviceCodeLabel')}</strong> {selectedTask.deviceCode}</span>
                <span><strong>{t('log.logDeviceNameLabel')}</strong> {selectedTask.deviceName}</span>
              </Space>
            </div>
            <Table
              size="small"
              columns={resultColumns}
              dataSource={logResults}
              rowKey="id"
              pagination={false}
            />
          </>
        )}
      </Drawer>

      {/* 删除确认弹窗 */}
      <Modal
        title={t('log.confirmDeleteTitle')}
        open={deleteModalVisible}
        onCancel={() => setDeleteModalVisible(false)}
        onOk={handleDeleteConfirm}
        okText={t('common.confirmDelete')}
        cancelText={t('common.cancel')}
        okButtonProps={{ danger: true }}
      >
        <p>{t('log.confirmDeleteContent')}</p>
        {taskToDelete && (
          <p>
            <strong>{t('log.logDeviceCodeLabel')}</strong> {taskToDelete.deviceCode}
          </p>
        )}
      </Modal>
    </ListPageLayout>
  );
}
