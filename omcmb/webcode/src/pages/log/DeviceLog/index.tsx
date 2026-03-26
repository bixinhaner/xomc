import { useState, useMemo } from 'react';
import {
  Button,
  Space,
  Tag,
  Typography,
  Dropdown,
  Modal,
  message,
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
  productType: string;
}

// 状态配置
const statusConfig: Record<TaskStatus, { text: string; color: string }> = {
  0: { text: '等待', color: 'blue' },
  1: { text: '进行中', color: 'processing' },
  2: { text: '成功', color: 'success' },
  3: { text: '失败', color: 'error' },
  4: { text: '终止', color: 'default' },
  5: { text: '正在停止上报', color: 'processing' },
  6: { text: '停止上报成功', color: 'default' },
  7: { text: '停止上报失败', color: 'processing' },
  8: { text: '周期上报设置成功，重启生效', color: 'blue' },
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
  { id: '1', deviceCode: 'ENB00001', deviceName: '北京朝阳1号站', productType: '4G LTE' },
  { id: '2', deviceCode: 'GNB00002', deviceName: '上海浦东2号站', productType: '5G NR' },
  { id: '3', deviceCode: 'ENB00003', deviceName: '广州天河3号站', productType: '4G LTE' },
  { id: '4', deviceCode: 'ENB00004', deviceName: '深圳南山4号站', productType: '4G LTE' },
  { id: '5', deviceCode: 'GNB00005', deviceName: '杭州西湖5号站', productType: '5G NR' },
  { id: '6', deviceCode: 'ENB00006', deviceName: '成都高新6号站', productType: '4G LTE' },
  { id: '7', deviceCode: 'GNB00007', deviceName: '武汉光谷7号站', productType: '5G NR' },
];

// 产品类型选项
const productTypeOptions = [
  { label: '4G LTE', value: '4G LTE' },
  { label: '5G NR', value: '5G NR' },
];

export default function DeviceLog() {
  const t = useT();
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
  const [productTypeFilter, setProductTypeFilter] = useState<string>('');
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
    if (productTypeFilter) {
      devices = devices.filter((d) => d.productType === productTypeFilter);
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
  }, [availableDevices, productTypeFilter, addDeviceKeyword]);

  // 筛选字段
  const filterFields: FilterField[] = useMemo(() => [
    { name: 'deviceCode', label: '设备编码', type: 'input', placeholder: '请输入设备编码' },
    {
      name: 'status',
      label: '收集状态',
      type: 'select',
      options: [
        { label: '等待', value: 0 },
        { label: '进行中', value: 1 },
        { label: '成功', value: 2 },
        { label: '失败', value: 3 },
        { label: '终止', value: 4 },
      ],
    },
    { name: 'timeRange', label: '时间范围', type: 'date-range', span: 2 },
  ], []);

  // 获取操作菜单
  const getActionMenu = (record: DeviceLogTask): MenuProps['items'] => {
    const status = record.taskStatus;
    const items: MenuProps['items'] = [];

    // 结果 - 仅成功时可用
    if (status === 2) {
      items.push({
        key: 'view',
        label: '结果',
        icon: <EyeOutlined />,
        onClick: () => handleViewResult(record),
      });
    }

    // 终止任务 - 状态为 0/1/7/8 时可用
    if ([0, 1, 7, 8].includes(status)) {
      items.push({
        key: 'terminate',
        label: '终止任务',
        icon: <StopOutlined />,
        onClick: () => handleTerminate(record),
      });
    }

    // 下载 - 仅成功时可用
    if (status === 2) {
      items.push({
        key: 'download',
        label: '下载',
        icon: <DownloadOutlined />,
        onClick: () => handleDownload(record),
      });
    }

    // 删除 - 状态为 2/3/4 时可用
    if ([2, 3, 4].includes(status)) {
      items.push({
        key: 'delete',
        label: '删除',
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
    Modal.confirm({
      title: '确认终止',
      content: `确定要终止设备 ${record.deviceCode} 的日志收集任务吗？`,
      onOk: () => {
        void message.success('任务已终止');
      },
    });
  };

  // 下载
  const handleDownload = (record: DeviceLogTask) => {
    void message.success(`正在下载设备 ${record.deviceCode} 的日志文件`);
  };

  // 删除
  const handleDeleteClick = (record: DeviceLogTask) => {
    setTaskToDelete(record);
    setDeleteModalVisible(true);
  };

  const handleDeleteConfirm = () => {
    if (taskToDelete) {
      void message.success(`已删除任务: ${taskToDelete.deviceCode}`);
      setDeleteModalVisible(false);
      setTaskToDelete(null);
    }
  };

  // 批量删除
  const handleBatchDelete = () => {
    // 过滤出可删除的任务（状态为 2/3/4：成功、失败、终止）
    const deletableTasks = filteredData.filter(
      (task) => selectedRowKeys.includes(task.id) && [2, 3, 4].includes(task.taskStatus)
    );
    // 收集中状态包括：0-等待、1-进行中、5-正在停止上报、7-停止上报失败、8-周期上报设置成功
    const collectingTasks = filteredData.filter(
      (task) => selectedRowKeys.includes(task.id) && [0, 1, 5, 7, 8].includes(task.taskStatus)
    );

    if (collectingTasks.length > 0) {
      Modal.warning({
        title: '无法删除',
        content: `选中的任务中有 ${collectingTasks.length} 个正在收集或等待中，不允许删除。请先终止这些任务后再删除。`,
      });
      return;
    }

    if (deletableTasks.length === 0) {
      void message.warning('请选择可删除的任务（成功、失败或终止状态）');
      return;
    }

    Modal.confirm({
      title: '批量删除',
      content: `确定要删除选中的 ${deletableTasks.length} 个任务吗？`,
      onOk: () => {
        void message.success(`已删除 ${deletableTasks.length} 个任务`);
        setSelectedRowKeys([]);
      },
    });
  };

  // 批量下载
  const handleBatchDownload = () => {
    void message.success(`正在下载 ${selectedRowKeys.length} 个任务的日志文件`);
  };

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
      void message.warning('请至少选择一个设备');
      return;
    }
    const newDevices = availableDevices.filter((d) => selectedNewDevices.includes(d.id));
    setSelectedDevices((prev) => [...prev, ...newDevices]);
    setAddDeviceModalVisible(false);
    setSelectedNewDevices([]);
    setAddDeviceKeyword('');
    void message.success(`已添加 ${newDevices.length} 台设备`);
  };

  // 打开批量输入弹窗
  const handleOpenBatchInput = () => {
    setBatchInputValue('');
    setBatchInputModalVisible(true);
  };

  // 确认批量输入
  const handleConfirmBatchInput = () => {
    if (!batchInputValue.trim()) {
      void message.warning('请输入设备编码');
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
          deviceName: '手动添加设备',
          productType: '未知',
        });
      }
    }
    if (newDevices.length > 0) {
      setSelectedDevices((prev) => [...prev, ...newDevices]);
      void message.success(`已添加 ${newDevices.length} 台设备`);
    }
    setBatchInputModalVisible(false);
    setBatchInputValue('');
  };

  // 新建任务
  const handleCreateTask = () => {
    form.validateFields().then((values) => {
      if (selectedDevices.length === 0) {
        void message.warning('请至少选择一个设备');
        return;
      }
      console.log('创建任务:', { ...values, devices: selectedDevices });
      void message.success(`任务创建成功，已选择 ${selectedDevices.length} 个设备`);
      setCreateDrawerVisible(false);
      form.resetFields();
      setSelectedDevices([]);
      setProductTypeFilter('');
    });
  };

  // 打开新建抽屉时重置状态
  const handleOpenCreateDrawer = () => {
    setSelectedDevices([]);
    setProductTypeFilter('');
    setCreateDrawerVisible(true);
  };

  // 批量操作
  const batchActions: BatchAction[] = useMemo(() => [
    {
      key: 'delete',
      label: '批量删除',
      icon: <DeleteOutlined />,
      danger: true,
      onClick: handleBatchDelete,
    },
    {
      key: 'download',
      label: '批量下载',
      icon: <DownloadOutlined />,
      onClick: handleBatchDownload,
    },
  ], [selectedRowKeys]);

  // 表格列
  const columns: DataTableColumn<DeviceLogTask>[] = useMemo(() => [
    {
      key: 'operation',
      title: '操作',
      width: 60,
      align: 'center',
      render: (_: unknown, record: DeviceLogTask) => {
        const items = getActionMenu(record);
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
      title: '设备编码',
      dataIndex: 'deviceCode',
      width: 140,
      render: (val: string) => <Typography.Text style={{ fontFamily: 'monospace' }}>{val}</Typography.Text>,
    },
    {
      key: 'deviceName',
      title: '设备名称',
      dataIndex: 'deviceName',
      width: 180,
      ellipsis: true,
    },
    {
      key: 'taskStatus',
      title: '收集状态',
      dataIndex: 'taskStatus',
      width: 180,
      render: (val: TaskStatus) => {
        const cfg = statusConfig[val];
        return <Tag color={cfg.color}>{cfg.text}</Tag>;
      },
    },
    {
      key: 'logType',
      title: '类型',
      dataIndex: 'logType',
      width: 120,
    },
    {
      key: 'executeType',
      title: '执行方式',
      dataIndex: 'executeType',
      width: 120,
      render: (val: ExecuteType) => val === 'Immediately' ? '立即执行' : '周期执行',
    },
    {
      key: 'reportPeriod',
      title: '周期',
      dataIndex: 'reportPeriod',
      width: 120,
      render: (val?: string) => val ?? '-',
    },
    {
      key: 'failureReason',
      title: '失败原因',
      dataIndex: 'failureReason',
      width: 200,
      ellipsis: true,
      render: (val?: string) => val ?? '-',
    },
    {
      key: 'updateTime',
      title: '更新时间',
      dataIndex: 'updateTime',
      width: 180,
    },
  ], []);

  // 结果表格列
  const resultColumns = [
    {
      title: '文件名',
      dataIndex: 'fileName',
      key: 'fileName',
      ellipsis: true,
    },
    {
      title: '文件大小',
      dataIndex: 'fileSize',
      key: 'fileSize',
      width: 120,
    },
    {
      title: '上传时间',
      dataIndex: 'uploadTime',
      key: 'uploadTime',
      width: 180,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: 'success' | 'failed') => (
        <Tag color={status === 'success' ? 'success' : 'error'}>
          {status === 'success' ? '成功' : '失败'}
        </Tag>
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 80,
      render: (_: unknown, record: LogResultItem) => (
        <Button type="link" size="small" icon={<DownloadOutlined />}>
          下载
        </Button>
      ),
    },
  ];

  return (
    <ListPageLayout
      title="设备上报日志"
      extra={
        <Button type="primary" icon={<PlusOutlined />} onClick={handleOpenCreateDrawer}>
          新建任务
        </Button>
      }
    >
      <FilterBar
        filterId="device-log-filter"
        fields={filterFields}
        onSearch={setFilters}
        onReset={() => setFilters({})}
      />

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
        scroll={{ x: 1400 }}
      />

      {/* 新建任务抽屉 */}
      <Drawer
        title="新建日志收集任务"
        placement="right"
        width={500}
        open={createDrawerVisible}
        onClose={() => setCreateDrawerVisible(false)}
        footer={
          <Space style={{ width: '100%', justifyContent: 'flex-end' }}>
            <Button onClick={() => setCreateDrawerVisible(false)}>取消</Button>
            <Button type="primary" onClick={handleCreateTask}>确定</Button>
          </Space>
        }
      >
        <Form form={form} layout="vertical" size="small">
          <Form.Item name="executeType" label="执行方式" rules={[{ required: true }]}>
            <Select placeholder="请选择执行方式">
              <Select.Option value="Immediately">立即执行</Select.Option>
              <Select.Option value="Period">周期执行</Select.Option>
            </Select>
          </Form.Item>

          {/* 周期执行配置 - 仅周期执行时显示 */}
          <Form.Item noStyle shouldUpdate={(prev, cur) => prev.executeType !== cur.executeType}>
            {({ getFieldValue }) => {
              const executeType = getFieldValue('executeType');
              if (executeType !== 'Period') return null;
              return (
                <>
                  <Form.Item name="periodDateRange" label="周期日期" rules={[{ required: true, message: '请选择周期日期' }]}>
                    <DatePicker.RangePicker
                      style={{ width: '100%' }}
                      placeholder={['开始日期', '结束日期']}
                    />
                  </Form.Item>
                  <Form.Item name="collectInterval" label="收集粒度" rules={[{ required: true, message: '请选择收集粒度' }]}>
                    <Select placeholder="请选择收集粒度">
                      <Select.Option value="15">15分钟</Select.Option>
                      <Select.Option value="30">30分钟</Select.Option>
                      <Select.Option value="60">60分钟</Select.Option>
                    </Select>
                  </Form.Item>
                </>
              );
            }}
          </Form.Item>

          <Form.Item name="logType" label="日志类型" rules={[{ required: true }]}>
            <Select placeholder="请选择日志类型">
              <Select.Option value="system">系统日志</Select.Option>
              <Select.Option value="debug">调试日志</Select.Option>
              <Select.Option value="performance">性能日志</Select.Option>
            </Select>
          </Form.Item>

          {/* 产品类型筛选 */}
          <Form.Item label="产品类型">
            <Select
              placeholder="按产品类型筛选设备"
              allowClear
              style={{ width: '100%' }}
              options={productTypeOptions}
              value={productTypeFilter || undefined}
              onChange={(val) => setProductTypeFilter(val ?? '')}
            />
          </Form.Item>

          {/* 已选设备 */}
          <Form.Item
            label={
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', width: '100%' }}>
                <span>
                  已选设备
                  {' '}
                  <Tag color="blue">{selectedDevices.length} 台</Tag>
                </span>
                <Button
                  size="small"
                  icon={<PlusOutlined />}
                  onClick={handleOpenBatchInput}
                >
                  批量输入
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
                    { title: '设备编码', dataIndex: 'deviceCode', width: 100 },
                    { title: '设备名称', dataIndex: 'deviceName', ellipsis: true },
                    { title: '产品类型', dataIndex: 'productType', width: 80 },
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
                  添加设备
                </Button>
              </div>
            </>
          </Form.Item>
        </Form>
      </Drawer>

      {/* 添加设备弹窗 */}
      <Modal
        title="添加设备"
        open={addDeviceModalVisible}
        onCancel={() => setAddDeviceModalVisible(false)}
        onOk={handleConfirmAddDevices}
        okText="确认添加"
        cancelText="取消"
        width={700}
        okButtonProps={{ disabled: selectedNewDevices.length === 0 }}
      >
        {availableDevices.length === 0 ? (
          <Alert
            type="info"
            showIcon
            message="没有可添加的设备"
            description="所有设备已全部在列表中"
          />
        ) : (
          <>
            <Alert
              type="info"
              showIcon
              message={`共 ${availableDevices.length} 台设备可选，当前显示 ${filteredAvailableDevices.length} 台，已选择 ${selectedNewDevices.length} 台`}
              style={{ marginBottom: 16 }}
            />

            {/* 搜索和全选 */}
            <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <Input.Search
                placeholder="搜索设备编码或名称"
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
                全选 ({filteredAvailableDevices.length} 台)
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
                { title: '设备编码', dataIndex: 'deviceCode', width: 120 },
                { title: '设备名称', dataIndex: 'deviceName', ellipsis: true },
                { title: '产品类型', dataIndex: 'productType', width: 100 },
              ]}
            />
          </>
        )}
      </Modal>

      {/* 批量输入弹窗 */}
      <Modal
        title="批量输入设备编码"
        open={batchInputModalVisible}
        onCancel={() => setBatchInputModalVisible(false)}
        onOk={handleConfirmBatchInput}
        okText="确认添加"
        cancelText="取消"
      >
        <div style={{ marginBottom: 8 }}>
          请输入设备编码，多个编码用换行、逗号或分号分隔
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
        title={`任务结果 - ${selectedTask?.deviceCode ?? ''}`}
        placement="right"
        width={700}
        open={resultDrawerVisible}
        onClose={() => setResultDrawerVisible(false)}
      >
        {selectedTask && (
          <>
            <div style={{ marginBottom: 16 }}>
              <Space size="large">
                <span><strong>设备编码:</strong> {selectedTask.deviceCode}</span>
                <span><strong>设备名称:</strong> {selectedTask.deviceName}</span>
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
        title="确认删除"
        open={deleteModalVisible}
        onCancel={() => setDeleteModalVisible(false)}
        onOk={handleDeleteConfirm}
        okText="确认删除"
        cancelText="取消"
        okButtonProps={{ danger: true }}
      >
        <p>确定要删除该日志收集任务吗？此操作不可恢复。</p>
        {taskToDelete && (
          <p>
            <strong>设备编码:</strong> {taskToDelete.deviceCode}
          </p>
        )}
      </Modal>
    </ListPageLayout>
  );
}
