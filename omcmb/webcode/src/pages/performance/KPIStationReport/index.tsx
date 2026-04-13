import React, { useCallback, useMemo, useState } from 'react';
import { App, Button, Card, Switch, Tag } from 'antd';
import {
  EyeOutlined,
  PlayCircleOutlined,
  StopOutlined,
} from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn, BatchAction } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import { useT } from '@/hooks/useT';
import MeasurementFileDrawer from './components/MeasurementFileDrawer';

/** 测量维护行数据 */
interface MeasurementRow extends Record<string, unknown> {
  id: string;
  serialNumber: string;
  hostName: string;
  cellId: string;
  smallCellCode: string;
  reportEnable: '0' | '1';
  reportPeriod: number;
  status: '0' | '1' | '2';
  needReboot: '0' | '1';
  startTime: string;
  updateTime: string;
}

// 状态配置
const STATUS_CONFIG: Record<string, { label: string; color: string }> = {
  '0': { label: 'status.off', color: 'default' },
  '1': { label: 'status.normal', color: 'success' },
  '2': { label: 'status.damaged', color: 'error' },
};

// Mock 数据
const mockData: MeasurementRow[] = [
  { id: '1', serialNumber: 'ENB00001', hostName: '北京朝阳基站01', cellId: '10001', smallCellCode: 'SC001', reportEnable: '1', reportPeriod: 15, status: '1', needReboot: '0', startTime: '2026-04-01 08:00:00', updateTime: '2026-04-02 10:30:00' },
  { id: '2', serialNumber: 'ENB00002', hostName: '北京海淀基站01', cellId: '10002', smallCellCode: 'SC002', reportEnable: '1', reportPeriod: 15, status: '1', needReboot: '0', startTime: '2026-04-01 08:00:00', updateTime: '2026-04-02 10:30:00' },
  { id: '3', serialNumber: 'ENB00003', hostName: '上海浦东基站01', cellId: '20001', smallCellCode: 'SC003', reportEnable: '0', reportPeriod: 30, status: '0', needReboot: '0', startTime: '2026-04-01 08:00:00', updateTime: '2026-04-02 10:30:00' },
  { id: '4', serialNumber: 'GNB00001', hostName: '北京5G基站01', cellId: '30001', smallCellCode: 'SC004', reportEnable: '1', reportPeriod: 15, status: '2', needReboot: '1', startTime: '2026-04-01 08:00:00', updateTime: '2026-04-02 10:30:00' },
  { id: '5', serialNumber: 'GNB00002', hostName: '北京5G基站02', cellId: '30002', smallCellCode: 'SC005', reportEnable: '1', reportPeriod: 15, status: '1', needReboot: '0', startTime: '2026-04-01 08:00:00', updateTime: '2026-04-02 10:30:00' },
  { id: '6', serialNumber: 'ENB00004', hostName: '深圳南山基站01', cellId: '40001', smallCellCode: 'SC006', reportEnable: '0', reportPeriod: 30, status: '0', needReboot: '1', startTime: '2026-04-01 08:00:00', updateTime: '2026-04-02 10:30:00' },
];

export default function KPIMeasurement() {
  const t = useT();
  const { modal, message } = App.useApp();
  const [
    filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);

  // 测量文件抽屉状态
  const [fileDrawerOpen, setFileDrawerOpen] = useState(false);
  const [currentDevice, setCurrentDevice] = useState<MeasurementRow | null>(null);

  // 筛选字段配置
  const filterFields: FilterField[] = useMemo(() => [
    {
      name: 'searchText',
      label: t('common.search'),
      type: 'input',
      placeholder: t('perf.measurement.searchPlaceholder'),
      span: 2,
    },
    {
      name: 'status',
      label: t('common.status'),
      type: 'select',
      options: [
        { label: t('common.all'), value: '' },
        { label: t('status.off'), value: '0' },
        { label: t('status.normal'), value: '1' },
        { label: t('status.damaged'), value: '2' },
      ],
    },
    {
      name: 'measEnable',
      label: t('perf.measurement.enableStatus'),
      type: 'select',
      options: [
        { label: t('common.all'), value: '' },
        { label: t('common.enabled'), value: '1' },
        { label: t('common.disabled'), value: '0' },
      ],
    },
  ], [t]);

  // 表格列配置
  const columns: DataTableColumn<MeasurementRow>[] = useMemo(() => [
    {
      key: 'operation',
      title: '',
      dataIndex: 'id',
      width: 80,
      fixed: 'left',
      render: (_, record) => (
        <Button
          type="link"
          size="small"
          icon={<EyeOutlined />}
          onClick={() => handleOpenFileDrawer(record)}
        >
          {t('perf.measurement.viewFiles')}
        </Button>
      ),
    },
    {
      key: 'reportEnable',
      title: t('perf.measurement.enable'),
      dataIndex: 'reportEnable',
      width: 100,
      render: (val, record) => (
        <Switch
          checked={val === '1'}
          size="small"
          onChange={() => handleToggleEnable(record)}
        />
      ),
    },
    {
      key: 'status',
      title: t('common.status'),
      dataIndex: 'status',
      width: 100,
      render: (val) => {
        const config = STATUS_CONFIG[val as string] || STATUS_CONFIG['0'];
        return (
          <Tag color={config.color}>
            {t(config.label)}
          </Tag>
        );
      },
    },
    {
      key: 'serialNumber',
      title: t('device.code'),
      dataIndex: 'serialNumber',
      width: 120,
      mono: true,
      copyable: true,
    },
    {
      key: 'hostName',
      title: t('device.hostName'),
      dataIndex: 'hostName',
      width: 160,
      ellipsis: true,
    },
    {
      key: 'reportPeriod',
      title: t('perf.measurement.period'),
      dataIndex: 'reportPeriod',
      width: 100,
      render: (val) => `${val}${t('common.minutes')}`,
    },
    {
      key: 'startTime',
      title: t('perf.measurement.startTime'),
      dataIndex: 'startTime',
      width: 160,
    },
    {
      key: 'updateTime',
      title: t('common.updateTime'),
      dataIndex: 'updateTime',
      width: 160,
    },
  ], [t]);

  // 打开测量文件抽屉
  const handleOpenFileDrawer = useCallback((record: MeasurementRow) => {
    setCurrentDevice(record);
    setFileDrawerOpen(true);
  }, []);

  // 关闭测量文件抽屉
  const handleCloseFileDrawer = useCallback(() => {
    setFileDrawerOpen(false);
    setCurrentDevice(null);
  }, []);

  // 切换启用状态
  const handleToggleEnable = useCallback((record: MeasurementRow) => {
    const newEnable = record.reportEnable === '1' ? '0' : '1';
    const actionText = newEnable === '1' ? t('common.enable') : t('common.disable');
    const confirmMsg = record.needReboot === '1'
      ? t('perf.measurement.rebootConfirmMsg', { action: actionText })
      : t('perf.measurement.confirmMsg', { action: actionText });

    modal.confirm({
      title: t('common.confirm'),
      content: confirmMsg,
      okText: t('common.confirm'),
      onOk: async () => {
        console.log('切换测量开关:', { smallCellCode: record.smallCellCode, activeReport: newEnable });
        void message.success(t('common.success'));
      },
    });
  }, [t, modal, message]);

  // 批量启用
  const handleBatchEnable = useCallback((enable: '0' | '1') => {
    if (selectedRowKeys.length === 0) {
      void message.warning(t('common.selectAtLeastOne'));
      return;
    }

    const actionText = enable === '1' ? t('common.enable') : t('common.disable');
    const hasReboot = mockData.some(
      (item) => selectedRowKeys.includes(item.id) && item.needReboot === '1'
    );
    const confirmMsg = hasReboot
      ? t('perf.measurement.batchRebootConfirmMsg', { action: actionText, count: selectedRowKeys.length })
      : t('perf.measurement.batchConfirmMsg', { action: actionText, count: selectedRowKeys.length });

    modal.confirm({
      title: t('common.confirm'),
      content: confirmMsg,
      okText: t('common.confirm'),
      onOk: async () => {
        const codes = mockData
          .filter((item) => selectedRowKeys.includes(item.id))
          .map((item) => item.smallCellCode);
        console.log('批量切换测量开关:', { smallCellCodes: codes, activeReport: enable });
        void message.success(t('common.success'));
        setSelectedRowKeys([]);
      },
    });
  }, [selectedRowKeys, t, modal, message]);

  // 批量操作
  const batchActions = useMemo((): BatchAction[] => [
    {
      key: 'enable',
      label: t('common.enable'),
      icon: <PlayCircleOutlined />,
      onClick: () => handleBatchEnable('1'),
    },
    {
      key: 'disable',
      label: t('common.disable'),
      icon: <StopOutlined />,
      onClick: () => handleBatchEnable('0'),
    },
  ], [t, handleBatchEnable]);

  // 过滤数据
  const filteredData = useMemo(() => {
    let data = mockData;

    const searchText = (filters.searchText as string)?.toLowerCase() || '';
    if (searchText) {
      data = data.filter(item =>
        item.serialNumber.toLowerCase().includes(searchText) ||
        item.hostName.toLowerCase().includes(searchText)
      );
    }

    const status = filters.status as string;
    if (status) {
      data = data.filter(item => item.status === status);
    }

    const measEnable = filters.measEnable as string;
    if (measEnable) {
      data = data.filter(item => item.reportEnable === measEnable);
    }

    return data;
  }, [filters]);

  // 搜索处理
  const handleSearch = useCallback((vals: Record<string, unknown>) => {
    setFilters(vals);
    setPage(1);
  }, []);

  // 重置处理
  const handleReset = useCallback(() => {
    setFilters({});
    setPage(1);
  }, []);

  return (
    <ListPageLayout title={t('nav.performance.kpiStation')}>
      <Card
        size="small"
        bordered
        style={{ marginBottom: 12 }}
        styles={{ body: { padding: '12px 16px 0' } }}
      >
        <FilterBar
          filterId="kpi-measurement"
          fields={filterFields}
          onSearch={handleSearch}
          onReset={handleReset}
          collapsedRows={1}
          noDefaultStyle
        />
      </Card>

      <Card
        size="small"
        bordered
        styles={{ body: { padding: 0, display: 'flex', flexDirection: 'column', flex: 1, overflow: 'hidden' } }}
      >
        <DataTable<MeasurementRow>
          tableId="kpi-measurement-table"
          columns={columns}
          dataSource={filteredData}
          loading={false}
          rowKey="id"
          selectable
          selectedRowKeys={selectedRowKeys}
          onSelectionChange={setSelectedRowKeys}
          total={filteredData.length}
          pageSize={pageSize}
          currentPage={page}
          onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
          batchActions={batchActions}
          scroll={{ x: 1200, y: 'calc(100vh - 380px)' }}
          showRowNumber
          rowNumberTitle={t('table.rowNumber')}
          defaultDensity="compact"
        />
      </Card>

      <MeasurementFileDrawer
        open={fileDrawerOpen}
        device={currentDevice}
        onClose={handleCloseFileDrawer}
      />
    </ListPageLayout>
  );
}
