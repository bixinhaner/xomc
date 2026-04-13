import { useState, useMemo } from 'react';
import {
  Button,
  Tag,
  Typography,
  Modal,
  message,
  Drawer,
  Dropdown,
  Descriptions,
  Divider,
  Card,
} from 'antd';
import {
  ExportOutlined,
  DeleteOutlined,
  PlayCircleOutlined,
  ReloadOutlined,
  DownloadOutlined,
  EyeOutlined,
  MoreOutlined,
} from '@ant-design/icons';
import type { MenuProps } from 'antd';
import type { Dayjs } from 'dayjs';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn, BatchAction } from '@/components/DataTable';

// 收集状态类型: 0-未收集, 1-收集中, 2-收集失败, 3-收集完成
type CollectStatus = '0' | '1' | '2' | '3';

// 设备在线状态: 0-离线, 1-在线
type OnlineStatus = '0' | '1';

interface ExceptionLog {
  id: string;
  deviceCode: string;
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
  onlineStatus: OnlineStatus;
  isFileDeleted: string;
}

// 收集状态配置
const collectStatusConfig: Record<CollectStatus, { text: string; color: string }> = {
  '0': { text: '未收集', color: 'default' },
  '1': { text: '收集中', color: 'processing' },
  '2': { text: '收集失败', color: 'error' },
  '3': { text: '收集完成', color: 'success' },
};

// 初始 Mock 数据
const initialMockExceptionLogs: ExceptionLog[] = [
  {
    id: '1',
    deviceCode: 'ENB00001',
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
    manualCollectionStatus: '3',
    onlineStatus: '1',
    isFileDeleted: '0',
  },
  {
    id: '2',
    deviceCode: 'GNB00002',
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
    onlineStatus: '1',
    isFileDeleted: '0',
  },
  {
    id: '3',
    deviceCode: 'ENB00003',
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
    onlineStatus: '1',
    isFileDeleted: '0',
  },
  {
    id: '4',
    deviceCode: 'ENB00004',
    deviceName: '深圳-eNB-0004',
    deviceType: 'eNB',
    operateIp: '10.3.0.104',
    product: 'QATA',
    softwareVersion: 'V1.3.0',
    operationName: '异常重启',
    opStartTime: '2026-03-25 14:00:00',
    manualCollectionStatus: '2',
    onlineStatus: '1',
    isFileDeleted: '0',
  },
  {
    id: '5',
    deviceCode: 'GNB00005',
    deviceName: '广州-gNB-0005',
    deviceType: 'gNB',
    operateIp: '10.4.0.105',
    product: 'BaiBNQ',
    softwareVersion: 'V2.0.0',
    operationName: '异常重启',
    opStartTime: '2026-03-24 10:00:00',
    runtimeBeforeReboot: '3天',
    haltDetailReason: '电源异常',
    manualCollectionStatus: '0',
    onlineStatus: '0',
    isFileDeleted: '0',
  },
];

export default function ExceptionLog() {
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [exceptionLogs, setExceptionLogs] = useState<ExceptionLog[]>(initialMockExceptionLogs);
  const [deleteModalVisible, setDeleteModalVisible] = useState(false);
  const [logsToDelete, setLogsToDelete] = useState<ExceptionLog[]>([]);
  const [logDetailVisible, setLogDetailVisible] = useState(false);
  const [selectedLog, setSelectedLog] = useState<ExceptionLog | null>(null);

  // 过滤数据
  const filteredData = useMemo(() => {
    return exceptionLogs.filter((row) => {
      if (filters.keyword && typeof filters.keyword === 'string') {
        const keyword = filters.keyword.toLowerCase();
        if (!row.deviceCode.toLowerCase().includes(keyword) &&
            !row.deviceName.toLowerCase().includes(keyword) &&
            !row.operateIp.includes(keyword)) {
          return false;
        }
      }
      if (filters.onlineStatus !== undefined && filters.onlineStatus !== null && filters.onlineStatus !== '') {
        if (row.onlineStatus !== filters.onlineStatus) {
          return false;
        }
      }
      if (filters.collectStatus !== undefined && filters.collectStatus !== null && filters.collectStatus !== '') {
        if (row.manualCollectionStatus !== filters.collectStatus) {
          return false;
        }
      }
      // 时间范围过滤
      if (filters.timeRange && Array.isArray(filters.timeRange)) {
        const [startTime, endTime] = filters.timeRange as [Dayjs, Dayjs];
        const rowTime = row.opStartTime;
        if (startTime && endTime) {
          const rowDate = new Date(rowTime);
          if (rowDate < startTime.toDate() || rowDate > endTime.toDate()) {
            return false;
          }
        }
      }
      return true;
    });
  }, [filters, exceptionLogs]);

  // 筛选字段
  const filterFields: FilterField[] = useMemo(() => [
    { name: 'keyword', label: '设备编码', type: 'input', placeholder: '请输入设备编码/名称/IP' },
    {
      name: 'onlineStatus',
      label: '在线状态',
      type: 'select',
      options: [
        { label: '在线', value: '1' },
        { label: '离线', value: '0' },
      ],
    },
    {
      name: 'collectStatus',
      label: '收集状态',
      type: 'select',
      options: [
        { label: '未收集', value: '0' },
        { label: '收集中', value: '1' },
        { label: '收集失败', value: '2' },
        { label: '收集完成', value: '3' },
      ],
    },
    { name: 'timeRange', label: '时间范围', type: 'date-range', span: 2 },
  ], []);

  // 开始收集
  const handleCollect = (record: ExceptionLog) => {
    if (record.onlineStatus === '0') {
      void message.warning(`设备 ${record.deviceCode} 不在线，无法收集日志`);
      return;
    }
    // 更新状态为收集中
    setExceptionLogs((prev) =>
      prev.map((log) =>
        log.id === record.id ? { ...log, manualCollectionStatus: '1' as CollectStatus } : log
      )
    );
    void message.success(`开始收集设备 ${record.deviceCode} 的日志`);
  };

  // 重新收集
  const handleRecollect = (record: ExceptionLog) => {
    if (record.onlineStatus === '0') {
      void message.warning(`设备 ${record.deviceCode} 不在线，无法收集日志`);
      return;
    }
    // 更新状态为收集中
    setExceptionLogs((prev) =>
      prev.map((log) =>
        log.id === record.id ? { ...log, manualCollectionStatus: '1' as CollectStatus } : log
      )
    );
    void message.success(`重新收集设备 ${record.deviceCode} 的日志`);
  };

  // 下载日志
  const handleDownload = (record: ExceptionLog) => {
    void message.success(`正在下载设备 ${record.deviceCode} 的日志文件: ${record.fileName}`);
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

    // 过滤出选中的记录
    const selectedLogs = filteredData.filter((log) => selectedRowKeys.includes(log.id));

    // 检查是否有收集中的记录（状态为 '1'）
    const collectingLogs = selectedLogs.filter((log) => log.manualCollectionStatus === '1');

    if (collectingLogs.length > 0) {
      Modal.warning({
        title: '无法删除',
        content: `选中的记录中有 ${collectingLogs.length} 个正在收集中，不允许删除。请等待收集完成后再删除。`,
      });
      return;
    }

    // 只删除非收集中的记录
    const deletableLogs = selectedLogs.filter((log) => log.manualCollectionStatus !== '1');

    if (deletableLogs.length === 0) {
      void message.warning('选中的记录都在收集中，无法删除');
      return;
    }

    setLogsToDelete(deletableLogs);
    setDeleteModalVisible(true);
  };

  const handleDeleteConfirm = () => {
    void message.success(`已删除 ${logsToDelete.length} 条异常日志记录`);
    setDeleteModalVisible(false);
    setLogsToDelete([]);
    setSelectedRowKeys([]);
  };

  // 获取操作菜单
  const getActionMenu = (record: ExceptionLog): MenuProps['items'] => {
    const { manualCollectionStatus, onlineStatus, fileName, isFileDeleted } = record;
    const isOnline = onlineStatus === '1';
    const hasFile = fileName && isFileDeleted === '0';
    const items: MenuProps['items'] = [];

    // 开始收集 - 未收集且设备在线时可用
    if (manualCollectionStatus === '0' && isOnline) {
      items.push({
        key: 'collect',
        label: '开始收集',
        icon: <PlayCircleOutlined />,
        onClick: () => handleCollect(record),
      });
    }

    // 重新收集 - 收集失败且设备在线时可用
    if (manualCollectionStatus === '2' && isOnline) {
      items.push({
        key: 'recollect',
        label: '重新收集',
        icon: <ReloadOutlined />,
        onClick: () => handleRecollect(record),
      });
    }

    // 下载日志 - 收集完成且有文件时可用
    if (manualCollectionStatus === '3' && hasFile) {
      items.push({
        key: 'download',
        label: '下载日志',
        icon: <DownloadOutlined />,
        onClick: () => handleDownload(record),
      });
    }

    // 查看详情 - 有文件时可用
    if (hasFile) {
      items.push({
        key: 'view',
        label: '查看详情',
        icon: <EyeOutlined />,
        onClick: () => handleViewFile(record),
      });
    }

    // 设备离线提示
    if (!isOnline && (manualCollectionStatus === '0' || manualCollectionStatus === '2')) {
      items.push({
        key: 'offline',
        label: '设备离线，无法收集',
        icon: <PlayCircleOutlined style={{ color: '#999' }} />,
        disabled: true,
      });
    }

    return items;
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

  // 表格列 - 在线状态放在设备名称前面
  const columns: DataTableColumn<ExceptionLog>[] = useMemo(() => [
    {
      key: 'operation',
      title: '操作',
      width: 100,
      align: 'center',
      render: (_: unknown, record: ExceptionLog) => {
        // 收集中状态不允许任何操作，显示"-"
        if (record.manualCollectionStatus === '1') {
          return '-';
        }
        const items = getActionMenu(record) ?? [];
        if (items.length === 0) return '-';
        return (
          <Dropdown menu={{ items }} trigger={['click']}>
            <Button size="small" icon={<MoreOutlined />}>更多</Button>
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
      key: 'onlineStatus',
      title: '在线状态',
      dataIndex: 'onlineStatus',
      width: 100,
      render: (val: OnlineStatus) => (
        <Tag color={val === '1' ? 'success' : 'default'}>
          {val === '1' ? '在线' : '离线'}
        </Tag>
      ),
    },
    {
      key: 'deviceName',
      title: '设备名称',
      dataIndex: 'deviceName',
      width: 180,
      ellipsis: true,
    },
    {
      key: 'deviceType',
      title: '设备类型',
      dataIndex: 'deviceType',
      width: 100,
    },
    {
      key: 'operateIp',
      title: '基站IP',
      dataIndex: 'operateIp',
      width: 140,
      render: (val: string) => <Typography.Text style={{ fontFamily: 'monospace' }}>{val}</Typography.Text>,
    },
    {
      key: 'product',
      title: '产品类型',
      dataIndex: 'product',
      width: 120,
    },
    {
      key: 'softwareVersion',
      title: '软件版本',
      dataIndex: 'softwareVersion',
      width: 120,
    },
    {
      key: 'operationName',
      title: '异常类型',
      dataIndex: 'operationName',
      width: 100,
      render: (val: string) => <Tag color="orange">{val}</Tag>,
    },
    {
      key: 'manualCollectionStatus',
      title: '收集状态',
      dataIndex: 'manualCollectionStatus',
      width: 100,
      render: (val: CollectStatus) => {
        const cfg = collectStatusConfig[val];
        return <Tag color={cfg.color}>{cfg.text}</Tag>;
      },
    },
    {
      key: 'fileName',
      title: '文件名',
      dataIndex: 'fileName',
      width: 200,
      ellipsis: true,
      render: (val?: string) => val ?? '-',
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
      width: 120,
      render: (val?: string) => val ?? '-',
    },
    {
      key: 'haltDetailReason',
      title: '死机原因',
      dataIndex: 'haltDetailReason',
      width: 140,
      ellipsis: true,
      render: (val?: string) => val ?? '-',
    },
  ], [exceptionLogs]);

  return (
    <ListPageLayout
      title="设备异常日志"
      extra={
        <Button type="primary" icon={<ExportOutlined />} onClick={handleExport}>
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
        scroll={{ x: 'max-content', y: 'calc(100vh - 400px)' }}
        showRowNumber
        rowNumberTitle="序号"
      />

      {/* 日志详情抽屉 */}
      <Drawer
        title="日志文件详情"
        placement="right"
        width={640}
        open={logDetailVisible}
        onClose={() => setLogDetailVisible(false)}
      >
        {selectedLog && (
          <div>
            {/* 设备信息 */}
            <Card size="small" title="设备信息" style={{ marginBottom: 16 }}>
              <Descriptions column={2} size="small">
                <Descriptions.Item label="设备编码">
                  <Typography.Text style={{ fontFamily: 'monospace' }}>{selectedLog.deviceCode}</Typography.Text>
                </Descriptions.Item>
                <Descriptions.Item label="设备名称">{selectedLog.deviceName}</Descriptions.Item>
                <Descriptions.Item label="设备类型">{selectedLog.deviceType}</Descriptions.Item>
                <Descriptions.Item label="基站IP">
                  <Typography.Text style={{ fontFamily: 'monospace' }}>{selectedLog.operateIp}</Typography.Text>
                </Descriptions.Item>
                <Descriptions.Item label="产品类型">{selectedLog.product}</Descriptions.Item>
                <Descriptions.Item label="软件版本">{selectedLog.softwareVersion}</Descriptions.Item>
                <Descriptions.Item label="在线状态">
                  <Tag color={selectedLog.onlineStatus === '1' ? 'success' : 'default'}>
                    {selectedLog.onlineStatus === '1' ? '在线' : '离线'}
                  </Tag>
                </Descriptions.Item>
              </Descriptions>
            </Card>

            {/* 异常信息 */}
            <Card size="small" title="异常信息" style={{ marginBottom: 16 }}>
              <Descriptions column={2} size="small">
                <Descriptions.Item label="异常类型">
                  <Tag color="orange">{selectedLog.operationName}</Tag>
                </Descriptions.Item>
                <Descriptions.Item label="收集状态">
                  <Tag color={collectStatusConfig[selectedLog.manualCollectionStatus].color}>
                    {collectStatusConfig[selectedLog.manualCollectionStatus].text}
                  </Tag>
                </Descriptions.Item>
                <Descriptions.Item label="发生时间">{selectedLog.opStartTime}</Descriptions.Item>
                <Descriptions.Item label="运行时间">{selectedLog.runtimeBeforeReboot ?? '-'}</Descriptions.Item>
                <Descriptions.Item label="死机原因" span={2}>{selectedLog.haltDetailReason ?? '-'}</Descriptions.Item>
              </Descriptions>
            </Card>

            {/* 文件信息 */}
            {selectedLog.fileName && (
              <Card size="small" title="文件信息">
                <Descriptions column={1} size="small">
                  <Descriptions.Item label="文件名">
                    <Typography.Text style={{ fontFamily: 'monospace' }} copyable>
                      {selectedLog.fileName}
                    </Typography.Text>
                  </Descriptions.Item>
                </Descriptions>

                <Divider style={{ margin: '12px 0' }} />

                <div>
                  <Typography.Text strong style={{ marginBottom: 8, display: 'block' }}>日志内容预览</Typography.Text>
                  <pre style={{
                    background: '#1e1e1e',
                    color: '#d4d4d4',
                    padding: 12,
                    marginTop: 8,
                    maxHeight: 300,
                    overflow: 'auto',
                    borderRadius: 4,
                    fontSize: 12,
                    lineHeight: 1.6,
                  }}>
                    {`[2026-03-26 10:00:00] WARN: Watchdog timeout detected
[2026-03-26 10:00:01] ERROR: System restarting...
[2026-03-26 10:00:02] INFO: Boot sequence initiated
[2026-03-26 10:00:05] INFO: Hardware check passed
[2026-03-26 10:00:10] INFO: Network interface up
[2026-03-26 10:00:15] INFO: Services starting...
[2026-03-26 10:00:20] INFO: System ready
...`}
                  </pre>
                </div>
              </Card>
            )}
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
