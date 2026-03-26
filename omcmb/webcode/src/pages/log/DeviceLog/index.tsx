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
  serialNumber: string;
  taskStatus: TaskStatus;
  logType: string;
  executeType: ExecuteType;
  reportPeriod?: string;
  failureReason?: string;
  updateTime: string;
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
  { id: '1', serialNumber: 'ENB00001', taskStatus: 2, logType: '系统日志', executeType: 'Immediately', updateTime: '2026-03-26 10:00:00' },
  { id: '2', serialNumber: 'GNB00002', taskStatus: 1, logType: '调试日志', executeType: 'Immediately', updateTime: '2026-03-26 09:30:00' },
  { id: '3', serialNumber: 'ENB00003', taskStatus: 0, logType: '系统日志', executeType: 'Period', reportPeriod: '每天', updateTime: '2026-03-26 08:00:00' },
  { id: '4', serialNumber: 'ENB00004', taskStatus: 3, logType: '性能日志', executeType: 'Immediately', failureReason: '设备离线', updateTime: '2026-03-25 16:00:00' },
  { id: '5', serialNumber: 'GNB00005', taskStatus: 4, logType: '系统日志', executeType: 'Immediately', updateTime: '2026-03-25 14:00:00' },
];

export default function DeviceLog() {
  const t = useT();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [createDrawerVisible, setCreateDrawerVisible] = useState(false);
  const [detailDrawerVisible, setDetailDrawerVisible] = useState(false);
  const [selectedTask, setSelectedTask] = useState<DeviceLogTask | null>(null);
  const [deleteModalVisible, setDeleteModalVisible] = useState(false);
  const [taskToDelete, setTaskToDelete] = useState<DeviceLogTask | null>(null);
  const [form] = Form.useForm();

  // 过滤数据
  const filteredData = useMemo(() => {
    return mockDeviceLogs.filter((row) => {
      if (filters.serialNumber && typeof filters.serialNumber === 'string') {
        if (!row.serialNumber.toLowerCase().includes(filters.serialNumber.toLowerCase())) {
          return false;
        }
      }
      if (filters.status !== undefined && filters.status !== null) {
        if (row.taskStatus !== filters.status) {
          return false;
        }
      }
      return true;
    });
  }, [filters]);

  // 筛选字段
  const filterFields: FilterField[] = useMemo(() => [
    { name: 'serialNumber', label: '设备标识', type: 'input', placeholder: '请输入设备唯一标识' },
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
    setDetailDrawerVisible(true);
  };

  // 终止任务
  const handleTerminate = (record: DeviceLogTask) => {
    Modal.confirm({
      title: '确认终止',
      content: `确定要终止设备 ${record.serialNumber} 的日志收集任务吗？`,
      onOk: () => {
        void message.success('任务已终止');
      },
    });
  };

  // 下载
  const handleDownload = (record: DeviceLogTask) => {
    void message.success(`正在下载设备 ${record.serialNumber} 的日志文件`);
  };

  // 删除
  const handleDeleteClick = (record: DeviceLogTask) => {
    setTaskToDelete(record);
    setDeleteModalVisible(true);
  };

  const handleDeleteConfirm = () => {
    if (taskToDelete) {
      void message.success(`已删除任务: ${taskToDelete.serialNumber}`);
      setDeleteModalVisible(false);
      setTaskToDelete(null);
    }
  };

  // 批量删除
  const handleBatchDelete = () => {
    Modal.confirm({
      title: '批量删除',
      content: `确定要删除选中的 ${selectedRowKeys.length} 个任务吗？`,
      onOk: () => {
        void message.success(`已删除 ${selectedRowKeys.length} 个任务`);
        setSelectedRowKeys([]);
      },
    });
  };

  // 批量下载
  const handleBatchDownload = () => {
    void message.success(`正在下载 ${selectedRowKeys.length} 个任务的日志文件`);
  };

  // 新建任务
  const handleCreateTask = () => {
    form.validateFields().then((values) => {
      console.log('创建任务:', values);
      void message.success('任务创建成功');
      setCreateDrawerVisible(false);
      form.resetFields();
    });
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
      key: 'serialNumber',
      title: '设备唯一标识',
      dataIndex: 'serialNumber',
      width: 200,
      render: (val: string) => <Typography.Text style={{ fontFamily: 'monospace' }}>{val}</Typography.Text>,
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

  return (
    <ListPageLayout
      title="设备上报日志"
      extra={
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateDrawerVisible(true)}>
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
        scroll={{ x: 1200 }}
      />

      {/* 新建任务抽屉 */}
      <Drawer
        title="新建日志收集任务"
        placement="right"
        width={450}
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
          <Form.Item name="devices" label="选择设备" rules={[{ required: true }]}>
            <Select mode="multiple" placeholder="请选择设备">
              <Select.Option value="ENB00001">ENB00001</Select.Option>
              <Select.Option value="GNB00002">GNB00002</Select.Option>
              <Select.Option value="ENB00003">ENB00003</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item name="logType" label="日志类型" rules={[{ required: true }]}>
            <Select placeholder="请选择日志类型">
              <Select.Option value="system">系统日志</Select.Option>
              <Select.Option value="debug">调试日志</Select.Option>
              <Select.Option value="performance">性能日志</Select.Option>
            </Select>
          </Form.Item>
        </Form>
      </Drawer>

      {/* 结果详情抽屉 */}
      <Drawer
        title="任务结果"
        placement="bottom"
        height={300}
        open={detailDrawerVisible}
        onClose={() => setDetailDrawerVisible(false)}
      >
        {selectedTask && (
          <div>
            <p><strong>设备标识:</strong> {selectedTask.serialNumber}</p>
            <p><strong>日志类型:</strong> {selectedTask.logType}</p>
            <p><strong>收集状态:</strong> {statusConfig[selectedTask.taskStatus].text}</p>
            <p><strong>更新时间:</strong> {selectedTask.updateTime}</p>
          </div>
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
            <strong>设备标识:</strong> {taskToDelete.serialNumber}
          </p>
        )}
      </Modal>
    </ListPageLayout>
  );
}
