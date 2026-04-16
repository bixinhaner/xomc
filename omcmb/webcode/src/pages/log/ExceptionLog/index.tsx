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
import { useT } from '@/hooks/useT';

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
  const t = useT();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [exceptionLogs, setExceptionLogs] = useState<ExceptionLog[]>(initialMockExceptionLogs);
  const [deleteModalVisible, setDeleteModalVisible] = useState(false);
  const [logsToDelete, setLogsToDelete] = useState<ExceptionLog[]>([]);
  const [logDetailVisible, setLogDetailVisible] = useState(false);
  const [selectedLog, setSelectedLog] = useState<ExceptionLog | null>(null);

  // 收集状态配置（依赖 t）
  const collectStatusConfig = useMemo<Record<CollectStatus, { text: string; color: string }>>(() => ({
    '0': { text: t('log.exception.notCollected'), color: 'default' },
    '1': { text: t('log.exception.collecting'), color: 'processing' },
    '2': { text: t('log.exception.collectFailed'), color: 'error' },
    '3': { text: t('log.exception.collectCompleted'), color: 'success' },
  }), [t]);

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
    { name: 'keyword', label: t('log.exception.column.deviceCode'), type: 'input', placeholder: t('log.exception.devCodeNameIp') },
    {
      name: 'onlineStatus',
      label: t('log.exception.onlineStatus'),
      type: 'select',
      options: [
        { label: t('log.exception.online'), value: '1' },
        { label: t('log.exception.offline'), value: '0' },
      ],
    },
    {
      name: 'collectStatus',
      label: t('log.exception.collectStatus'),
      type: 'select',
      options: [
        { label: t('log.exception.notCollected'), value: '0' },
        { label: t('log.exception.collecting'), value: '1' },
        { label: t('log.exception.collectFailed'), value: '2' },
        { label: t('log.exception.collectCompleted'), value: '3' },
      ],
    },
    { name: 'timeRange', label: t('log.timeRange'), type: 'date-range', span: 2 },
  ], [t]);

  // 开始收集
  const handleCollect = (record: ExceptionLog) => {
    if (record.onlineStatus === '0') {
      void message.warning(t('log.exception.devNotOnline', { code: record.deviceCode }));
      return;
    }
    // 更新状态为收集中
    setExceptionLogs((prev) =>
      prev.map((log) =>
        log.id === record.id ? { ...log, manualCollectionStatus: '1' as CollectStatus } : log
      )
    );
    void message.success(t('log.exception.startCollecting', { code: record.deviceCode }));
  };

  // 重新收集
  const handleRecollect = (record: ExceptionLog) => {
    if (record.onlineStatus === '0') {
      void message.warning(t('log.exception.devNotOnline', { code: record.deviceCode }));
      return;
    }
    // 更新状态为收集中
    setExceptionLogs((prev) =>
      prev.map((log) =>
        log.id === record.id ? { ...log, manualCollectionStatus: '1' as CollectStatus } : log
      )
    );
    void message.success(t('log.exception.recollecting', { code: record.deviceCode }));
  };

  // 下载日志
  const handleDownload = (record: ExceptionLog) => {
    void message.success(t('log.exception.downloading', { code: record.deviceCode, fileName: record.fileName ?? '' }));
  };

  // 查看文件
  const handleViewFile = (record: ExceptionLog) => {
    setSelectedLog(record);
    setLogDetailVisible(true);
  };

  // 导出
  const handleExport = () => {
    void message.success(t('log.exception.exporting'));
  };

  // 批量删除
  const handleBatchDelete = () => {
    if (selectedRowKeys.length === 0) {
      void message.warning(t('log.exception.selectDelete'));
      return;
    }

    // 过滤出选中的记录
    const selectedLogs = filteredData.filter((log) => selectedRowKeys.includes(log.id));

    // 检查是否有收集中的记录（状态为 '1'）
    const collectingLogs = selectedLogs.filter((log) => log.manualCollectionStatus === '1');

    if (collectingLogs.length > 0) {
      Modal.warning({
        title: t('log.exception.cannotDelete'),
        content: t('log.exception.collectingCannotDelete', { count: collectingLogs.length }),
      });
      return;
    }

    // 只删除非收集中的记录
    const deletableLogs = selectedLogs.filter((log) => log.manualCollectionStatus !== '1');

    if (deletableLogs.length === 0) {
      void message.warning(t('log.exception.allCollecting'));
      return;
    }

    setLogsToDelete(deletableLogs);
    setDeleteModalVisible(true);
  };

  const handleDeleteConfirm = () => {
    void message.success(t('log.exception.deletedCount', { count: logsToDelete.length }));
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
        label: t('log.exception.startCollect'),
        icon: <PlayCircleOutlined />,
        onClick: () => handleCollect(record),
      });
    }

    // 重新收集 - 收集失败且设备在线时可用
    if (manualCollectionStatus === '2' && isOnline) {
      items.push({
        key: 'recollect',
        label: t('log.exception.recollect'),
        icon: <ReloadOutlined />,
        onClick: () => handleRecollect(record),
      });
    }

    // 下载日志 - 收集完成且有文件时可用
    if (manualCollectionStatus === '3' && hasFile) {
      items.push({
        key: 'download',
        label: t('log.exception.downloadLog'),
        icon: <DownloadOutlined />,
        onClick: () => handleDownload(record),
      });
    }

    // 查看详情 - 有文件时可用
    if (hasFile) {
      items.push({
        key: 'view',
        label: t('log.exception.viewDetail'),
        icon: <EyeOutlined />,
        onClick: () => handleViewFile(record),
      });
    }

    // 设备离线提示
    if (!isOnline && (manualCollectionStatus === '0' || manualCollectionStatus === '2')) {
      items.push({
        key: 'offline',
        label: t('log.exception.devOffline'),
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
      label: t('log.exception.batchDelete'),
      icon: <DeleteOutlined />,
      danger: true,
      onClick: handleBatchDelete,
    },
  ], [selectedRowKeys, t]);

  // 表格列 - 在线状态放在设备名称前面
  const columns: DataTableColumn<ExceptionLog>[] = useMemo(() => [
    {
      key: 'operation',
      title: t('common.operation'),
      width: 100,
      fixed: 'right',
      render: (_: unknown, record: ExceptionLog) => {
        // 收集中状态不允许任何操作，显示"-"
        if (record.manualCollectionStatus === '1') {
          return '-';
        }
        const items = getActionMenu(record) ?? [];
        if (items.length === 0) return '-';
        if (items.length === 1) {
          const item = items[0] as NonNullable<MenuProps['items']>[number];
          if ('label' in item && 'onClick' in item) {
            return <Button type="link" size="small" onClick={() => item.onClick?.()}>{item.label as string}</Button>;
          }
        }
        return (
          <Dropdown menu={{ items }} trigger={['click']}>
            <Button type="link" size="small" icon={<MoreOutlined />} />
          </Dropdown>
        );
      },
    },
    {
      key: 'deviceCode',
      title: t('log.exception.column.deviceCode'),
      dataIndex: 'deviceCode',
      width: 140,
      render: (val: string) => <Typography.Text style={{ fontFamily: 'monospace' }}>{val}</Typography.Text>,
    },
    {
      key: 'onlineStatus',
      title: t('log.exception.onlineStatus'),
      dataIndex: 'onlineStatus',
      width: 100,
      render: (val: OnlineStatus) => (
        <Tag color={val === '1' ? 'success' : 'default'}>
          {val === '1' ? t('log.exception.online') : t('log.exception.offline')}
        </Tag>
      ),
    },
    {
      key: 'deviceName',
      title: t('log.exception.column.deviceName'),
      dataIndex: 'deviceName',
      width: 180,
      ellipsis: true,
    },
    {
      key: 'deviceType',
      title: t('log.exception.column.deviceType'),
      dataIndex: 'deviceType',
      width: 100,
    },
    {
      key: 'operateIp',
      title: t('log.exception.column.baseIp'),
      dataIndex: 'operateIp',
      width: 140,
      render: (val: string) => <Typography.Text style={{ fontFamily: 'monospace' }}>{val}</Typography.Text>,
    },
    {
      key: 'product',
      title: t('log.exception.column.productType'),
      dataIndex: 'product',
      width: 120,
    },
    {
      key: 'softwareVersion',
      title: t('log.exception.column.softwareVersion'),
      dataIndex: 'softwareVersion',
      width: 120,
    },
    {
      key: 'operationName',
      title: t('log.exception.column.exceptionType'),
      dataIndex: 'operationName',
      width: 100,
      render: (val: string) => <Tag color="orange">{val}</Tag>,
    },
    {
      key: 'manualCollectionStatus',
      title: t('log.exception.collectStatus'),
      dataIndex: 'manualCollectionStatus',
      width: 100,
      render: (val: CollectStatus) => {
        const cfg = collectStatusConfig[val];
        return <Tag color={cfg.color}>{cfg.text}</Tag>;
      },
    },
    {
      key: 'fileName',
      title: t('log.exception.column.fileName'),
      dataIndex: 'fileName',
      width: 200,
      ellipsis: true,
      render: (val?: string) => val ?? '-',
    },
    {
      key: 'opStartTime',
      title: t('log.exception.column.time'),
      dataIndex: 'opStartTime',
      width: 160,
    },
    {
      key: 'runtimeBeforeReboot',
      title: t('log.exception.column.runtime'),
      dataIndex: 'runtimeBeforeReboot',
      width: 120,
      render: (val?: string) => val ?? '-',
    },
    {
      key: 'haltDetailReason',
      title: t('log.exception.column.haltReason'),
      dataIndex: 'haltDetailReason',
      width: 140,
      ellipsis: true,
      render: (val?: string) => val ?? '-',
    },
  ], [t, collectStatusConfig, getActionMenu]);

  return (
    <ListPageLayout
      title={t('log.exceptionLog')}
      extra={
        <Button type="primary" icon={<ExportOutlined />} onClick={handleExport}>
          {t('log.export')}
        </Button>
      }
    >
      <FilterBar
        filterId="exception-log-filter"
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
          rowNumberTitle={t('log.exception.column.seq')}
        />
      </Card>

      {/* 日志详情抽屉 */}
      <Drawer
        title={t('log.exception.detail.title')}
        placement="right"
        width={640}
        open={logDetailVisible}
        onClose={() => setLogDetailVisible(false)}
      >
        {selectedLog && (
          <div>
            {/* 设备信息 */}
            <Card size="small" title={t('log.exception.detail.devInfo')} style={{ marginBottom: 16 }}>
              <Descriptions column={2} size="small">
                <Descriptions.Item label={t('log.exception.detail.devCode')}>
                  <Typography.Text style={{ fontFamily: 'monospace' }}>{selectedLog.deviceCode}</Typography.Text>
                </Descriptions.Item>
                <Descriptions.Item label={t('log.exception.detail.devName')}>{selectedLog.deviceName}</Descriptions.Item>
                <Descriptions.Item label={t('log.exception.detail.devType')}>{selectedLog.deviceType}</Descriptions.Item>
                <Descriptions.Item label={t('log.exception.detail.baseIp')}>
                  <Typography.Text style={{ fontFamily: 'monospace' }}>{selectedLog.operateIp}</Typography.Text>
                </Descriptions.Item>
                <Descriptions.Item label={t('log.exception.detail.productType')}>{selectedLog.product}</Descriptions.Item>
                <Descriptions.Item label={t('log.exception.detail.softwareVersion')}>{selectedLog.softwareVersion}</Descriptions.Item>
                <Descriptions.Item label={t('log.exception.detail.onlineStatus')}>
                  <Tag color={selectedLog.onlineStatus === '1' ? 'success' : 'default'}>
                    {selectedLog.onlineStatus === '1' ? t('log.exception.online') : t('log.exception.offline')}
                  </Tag>
                </Descriptions.Item>
              </Descriptions>
            </Card>

            {/* 异常信息 */}
            <Card size="small" title={t('log.exception.detail.exceptionInfo')} style={{ marginBottom: 16 }}>
              <Descriptions column={2} size="small">
                <Descriptions.Item label={t('log.exception.detail.exceptionType')}>
                  <Tag color="orange">{selectedLog.operationName}</Tag>
                </Descriptions.Item>
                <Descriptions.Item label={t('log.exception.detail.collectStatus')}>
                  <Tag color={collectStatusConfig[selectedLog.manualCollectionStatus].color}>
                    {collectStatusConfig[selectedLog.manualCollectionStatus].text}
                  </Tag>
                </Descriptions.Item>
                <Descriptions.Item label={t('log.exception.detail.occurTime')}>{selectedLog.opStartTime}</Descriptions.Item>
                <Descriptions.Item label={t('log.exception.detail.runtime')}>{selectedLog.runtimeBeforeReboot ?? '-'}</Descriptions.Item>
                <Descriptions.Item label={t('log.exception.detail.haltReason')} span={2}>{selectedLog.haltDetailReason ?? '-'}</Descriptions.Item>
              </Descriptions>
            </Card>

            {/* 文件信息 */}
            {selectedLog.fileName && (
              <Card size="small" title={t('log.exception.detail.fileInfo')}>
                <Descriptions column={1} size="small">
                  <Descriptions.Item label={t('log.exception.detail.fileName')}>
                    <Typography.Text style={{ fontFamily: 'monospace' }} copyable>
                      {selectedLog.fileName}
                    </Typography.Text>
                  </Descriptions.Item>
                </Descriptions>

                <Divider style={{ margin: '12px 0' }} />

                <div>
                  <Typography.Text strong style={{ marginBottom: 8, display: 'block' }}>{t('log.exception.detail.logPreview')}</Typography.Text>
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
        title={t('log.exception.deleteConfirm.title')}
        open={deleteModalVisible}
        onCancel={() => setDeleteModalVisible(false)}
        onOk={handleDeleteConfirm}
        okText={t('log.exception.deleteConfirm.ok')}
        cancelText={t('common.cancel')}
        okButtonProps={{ danger: true }}
      >
        <p>{t('log.exception.deleteConfirm.content', { count: logsToDelete.length })}</p>
      </Modal>
    </ListPageLayout>
  );
}
