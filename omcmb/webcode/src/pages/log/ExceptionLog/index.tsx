import { useState, useMemo } from 'react';
import {
  Button,
  Space,
  Tag,
  Typography,
  Tooltip,
  Modal,
  message,
  Drawer,
} from 'antd';
import {
  ExportOutlined,
  DeleteOutlined,
  PlayCircleOutlined,
  LoadingOutlined,
  WarningOutlined,
  EyeOutlined,
} from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn, BatchAction } from '@/components/DataTable';
import { useT } from '@/hooks/useT';

// 收集状态类型
type CollectStatus = '0' | '1' | '2';

interface ExceptionLog {
  id: string;
  serialNumber: string;
  deviceName: string;
  deviceType: string;
  operateIp: string;
  product: string;
  softwareVersion: string;
  operationName: string;
  fileName?: string;
  opStartTime: string;
  runtimeBeforeReboot?: string;
  haltDetailReason?: string;
  manualCollectionStatus: CollectStatus;
  isFileDeleted: string;
}

// 收集状态配置
const collectStatusConfig: Record<CollectStatus, { text: string; icon: React.ReactNode }> = {
  '0': { text: '未收集', icon: <PlayCircleOutlined /> },
  '1': { text: '收集中', icon: <LoadingOutlined /> },
  '2': { text: '收集失败', icon: <WarningOutlined /> },
};

// Mock 数据
const mockExceptionLogs: ExceptionLog[] = [
  {
    id: '1',
    serialNumber: 'ENB00001',
    deviceName: '北京-eNB-0001',
    deviceType: 'eNB',
    operateIp: '10.1.0.101',
    product: 'PM-B4860',
    softwareVersion: 'V1.3.0',
    operationName: '异常重启',
    fileName: 'exception_ENB00001_20260326.log',
    opStartTime: '2026-03-26 10:00:00',
    runtimeBeforeReboot: '5天3小时',
    haltDetailReason: '看门狗超时',
    manualCollectionStatus: '0',
    isFileDeleted: '0',
  },
  {
    id: '2',
    serialNumber: 'GNB00002',
    deviceName: '北京-gNB-0002',
    deviceType: 'gNB',
    operateIp: '10.1.1.102',
    product: 'BaiBNX',
    softwareVersion: 'V2.1.0',
    operationName: '异常重启',
    opStartTime: '2026-03-26 08:30:00',
    runtimeBeforeReboot: '12天',
    haltDetailReason: '内存溢出',
    manualCollectionStatus: '1',
    isFileDeleted: '0',
  },
  {
    id: '3',
    serialNumber: 'ENB00003',
    deviceName: '上海-eNB-0003',
    deviceType: 'eNB',
    operateIp: '10.2.0.103',
    product: 'QAFA',
    softwareVersion: 'V1.2.5',
    operationName: '异常重启',
    fileName: 'exception_ENB00003_20260325.log',
    opStartTime: '2026-03-25 16:00:00',
    runtimeBeforeReboot: '8小时',
    haltDetailReason: '软件异常',
    manualCollectionStatus: '0',
    isFileDeleted: '0',
  },
  {
    id: '4',
    serialNumber: 'ENB00004',
    deviceName: '深圳-eNB-0004',
    deviceType: 'eNB',
    operateIp: '10.3.0.104',
    product: 'QATA',
    softwareVersion: 'V1.3.0',
    operationName: '异常重启',
    opStartTime: '2026-03-25 14:00:00',
    manualCollectionStatus: '2',
    isFileDeleted: '0',
  },
  {
    id: '5',
    serialNumber: 'GNB00005',
    deviceName: '广州-gNB-0005',
    deviceType: 'gNB',
    operateIp: '10.4.0.105',
    product: 'BaiBNQ',
    softwareVersion: 'V2.0.0',
    operationName: '异常重启',
    fileName: 'exception_GNB00005_20260324.log',
    opStartTime: '2026-03-24 10:00:00',
    runtimeBeforeReboot: '3天',
    haltDetailReason: '电源异常',
    manualCollectionStatus: '0',
    isFileDeleted: '0',
  },
];

export default function ExceptionLog() {
  const t = useT();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [deleteModalVisible, setDeleteModalVisible] = useState(false);
  const [logsToDelete, setLogsToDelete] = useState<ExceptionLog[]>([]);
  const [logDetailVisible, setLogDetailVisible] = useState(false);
  const [selectedLog, setSelectedLog] = useState<ExceptionLog | null>(null);

  // 过滤数据
  const filteredData = useMemo(() => {
    return mockExceptionLogs.filter((row) => {
      if (filters.keyword && typeof filters.keyword === 'string') {
        const keyword = filters.keyword.toLowerCase();
        if (!row.serialNumber.toLowerCase().includes(keyword) &&
            !row.deviceName.toLowerCase().includes(keyword) &&
            !row.operateIp.includes(keyword)) {
          return false;
        }
      }
      return true;
    });
  }, [filters]);

  // 筛选字段
  const filterFields: FilterField[] = useMemo(() => [
    { name: 'keyword', label: '搜索', type: 'input', placeholder: '设备标识/名称/IP' },
  ], []);

  // 开始收集
  const handleCollect = (record: ExceptionLog) => {
    void message.success(`开始收集设备 ${record.serialNumber} 的日志`);
  };

  // 查看文件
  const handleViewFile = (record: ExceptionLog) => {
    setSelectedLog(record);
    setLogDetailVisible(true);
  };

  // 导出
  const handleExport = () => {
    void message.success('正在导出异常日志数据...');
  };

  // 批量删除
  const handleBatchDelete = () => {
    if (selectedRowKeys.length === 0) {
      void message.warning('请选择要删除的记录');
      return;
    }
    const logs = filteredData.filter((log) => selectedRowKeys.includes(log.id));
    setLogsToDelete(logs);
    setDeleteModalVisible(true);
  };

  const handleDeleteConfirm = () => {
    void message.success(`已删除 ${logsToDelete.length} 条异常日志记录`);
    setDeleteModalVisible(false);
    setLogsToDelete([]);
    setSelectedRowKeys([]);
  };

  // 获取收集按钮
  const renderCollectButton = (record: ExceptionLog) => {
    const status = record.manualCollectionStatus;
    const hasFile = record.fileName && record.isFileDeleted === '0';

    if (status === '1') {
      return (
        <Button size="small" loading>
          Collecting
        </Button>
      );
    }

    if (status === '2') {
      return (
        <Tooltip title="收集失败，点击重试">
          <Button
            size="small"
            icon={<WarningOutlined style={{ color: '#faad14' }} />}
            onClick={() => handleCollect(record)}
          />
        </Tooltip>
      );
    }

    if (hasFile) {
      return (
        <Tooltip title="文件已存在">
          <PlayCircleOutlined style={{ color: '#999', fontSize: 16 }} />
        </Tooltip>
      );
    }

    return (
      <Tooltip title="开始收集">
        <PlayCircleOutlined
          style={{ color: '#1890ff', fontSize: 16, cursor: 'pointer' }}
          onClick={() => handleCollect(record)}
        />
      </Tooltip>
    );
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
  ], [selectedRowKeys]);

  // 表格列
  const columns: DataTableColumn<ExceptionLog>[] = useMemo(() => [
    {
      key: 'operation',
      title: '操作',
      width: 100,
      align: 'center',
      render: (_: unknown, record: ExceptionLog) => renderCollectButton(record),
    },
    {
      key: 'serialNumber',
      title: '设备唯一标识',
      dataIndex: 'serialNumber',
      width: 200,
      render: (val: string) => <Typography.Text style={{ fontFamily: 'monospace' }}>{val}</Typography.Text>,
    },
    {
      key: 'deviceName',
      title: '设备名称',
      dataIndex: 'deviceName',
      width: 200,
      ellipsis: true,
    },
    {
      key: 'deviceType',
      title: '设备类型',
      dataIndex: 'deviceType',
      width: 150,
    },
    {
      key: 'operateIp',
      title: '基站IP',
      dataIndex: 'operateIp',
      width: 200,
      render: (val: string) => <Typography.Text style={{ fontFamily: 'monospace' }}>{val}</Typography.Text>,
    },
    {
      key: 'product',
      title: '产品类型',
      dataIndex: 'product',
      width: 150,
    },
    {
      key: 'softwareVersion',
      title: '软件版本',
      dataIndex: 'softwareVersion',
      width: 200,
    },
    {
      key: 'operationName',
      title: '异常类型',
      dataIndex: 'operationName',
      width: 150,
      render: (val: string) => <Tag color="orange">{val}</Tag>,
    },
    {
      key: 'fileName',
      title: '文件名',
      dataIndex: 'fileName',
      width: 200,
      ellipsis: true,
      render: (val?: string, record?: ExceptionLog) => {
        if (!val) return '-';
        return (
          <Button
            type="link"
            size="small"
            style={{ padding: 0 }}
            onClick={() => handleViewFile(record!)}
          >
            {val}
          </Button>
        );
      },
    },
    {
      key: 'opStartTime',
      title: '时间',
      dataIndex: 'opStartTime',
      width: 160,
    },
    {
      key: 'runtimeBeforeReboot',
      title: '运行时间',
      dataIndex: 'runtimeBeforeReboot',
      width: 160,
      render: (val?: string) => val ?? '-',
    },
    {
      key: 'haltDetailReason',
      title: '死机原因',
      dataIndex: 'haltDetailReason',
      width: 160,
      ellipsis: true,
      render: (val?: string) => val ?? '-',
    },
  ], []);

  return (
    <ListPageLayout
      title="设备异常日志"
      extra={
        <Button icon={<ExportOutlined />} onClick={handleExport}>
          导出
        </Button>
      }
    >
      <FilterBar
        filterId="exception-log-filter"
        fields={filterFields}
        onSearch={setFilters}
        onReset={() => setFilters({})}
      />

      <DataTable<ExceptionLog>
        tableId="exception-log-list"
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
        scroll={{ x: 2000 }}
      />

      {/* 日志详情抽屉 */}
      <Drawer
        title="日志文件详情"
        placement="right"
        width={600}
        open={logDetailVisible}
        onClose={() => setLogDetailVisible(false)}
      >
        {selectedLog && (
          <div>
            <p><strong>设备标识:</strong> {selectedLog.serialNumber}</p>
            <p><strong>设备名称:</strong> {selectedLog.deviceName}</p>
            <p><strong>异常类型:</strong> {selectedLog.operationName}</p>
            <p><strong>发生时间:</strong> {selectedLog.opStartTime}</p>
            <p><strong>运行时间:</strong> {selectedLog.runtimeBeforeReboot ?? '-'}</p>
            <p><strong>死机原因:</strong> {selectedLog.haltDetailReason ?? '-'}</p>
            <p><strong>文件名:</strong> {selectedLog.fileName}</p>
            <div style={{ marginTop: 16 }}>
              <strong>日志内容:</strong>
              <pre style={{
                background: '#f5f5f5',
                padding: 12,
                marginTop: 8,
                maxHeight: 400,
                overflow: 'auto',
              }}>
                {`[2026-03-26 10:00:00] WARN: Watchdog timeout detected
[2026-03-26 10:00:01] ERROR: System restarting...
[2026-03-26 10:00:02] INFO: Boot sequence initiated
[2026-03-26 10:00:05] INFO: Hardware check passed
[2026-03-26 10:00:10] INFO: Network interface up
...`}
              </pre>
            </div>
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
        <p>确定要删除选中的 {logsToDelete.length} 条异常日志记录吗？此操作不可恢复。</p>
      </Modal>
    </ListPageLayout>
  );
}
