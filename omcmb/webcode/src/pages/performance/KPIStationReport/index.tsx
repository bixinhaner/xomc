import React, { useCallback, useMemo, useState } from 'react';
import { App, Button, Card, Switch, Tag } from 'antd';
import {
  PlayCircleOutlined,
  StopOutlined,
} from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn, BatchAction } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import { useT } from '@/hooks/useT';
import { useDeviceList } from '@core/hooks/api/useDevices';
import { useEnableIndicators, useDisableIndicators } from '@core/hooks/api/useIndicator';
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

export default function KPIMeasurement() {
  const t = useT();
  const { modal, message } = App.useApp();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);

  // 测量文件抽屉状态
  const [fileDrawerOpen, setFileDrawerOpen] = useState(false);
  const [currentDevice, setCurrentDevice] = useState<MeasurementRow | null>(null);

  // ── Mutations ──────────────────────────────────────────────────────────────
  const enableMutation = useEnableIndicators();
  const disableMutation = useDisableIndicators();

  // ── 设备列表查询 ────────────────────────────────────────────────────────────
  const deviceParams = useMemo(() => ({
    searchText: (filters.searchText as string) || undefined,
    page,
    pageSize,
  }), [filters.searchText, page, pageSize]);

  const { data: devicePage, isLoading: deviceLoading } = useDeviceList(deviceParams);

  // 将设备数据映射为 MeasurementRow
  const tableData: MeasurementRow[] = useMemo(() => {
    if (!devicePage?.items) return [];
    return devicePage.items.map((device) => ({
      id: device.id,
      serialNumber: device.sn,
      hostName: device.hostName || device.name,
      cellId: device.cellId || '',
      smallCellCode: device.sn,
      reportEnable: (device.pmReportStatus === 'enabled' || device.connStatus === 'online') ? '1' as const : '0' as const,
      reportPeriod: 15,
      status: device.connStatus === 'online' ? '1' as const : '0' as const,
      needReboot: '0' as const,
      startTime: device.createTime || '',
      updateTime: device.lastOnlineTime || '',
    }));
  }, [devicePage]);

  const totalCount = devicePage?.total ?? 0;

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

  // 过滤数据（前端二次过滤 status / measEnable）
  const filteredData = useMemo(() => {
    let data = tableData;

    const status = filters.status as string;
    if (status) {
      data = data.filter(item => item.status === status);
    }

    const measEnable = filters.measEnable as string;
    if (measEnable) {
      data = data.filter(item => item.reportEnable === measEnable);
    }

    return data;
  }, [tableData, filters.status, filters.measEnable]);

  // 表格列配置
  const columns: DataTableColumn<MeasurementRow>[] = useMemo(() => [
    {
      key: 'operation',
      title: t('table.operation'),
      dataIndex: 'id',
      width: 80,
      fixed: 'right',
      render: (_, record) => (
        <Button
          type="link"
          size="small"
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
        try {
          const mutation = newEnable === '1' ? enableMutation : disableMutation;
          await mutation.mutateAsync({
            deviceType: 'ENB',
            operatorCode: '',
            indicatorIds: [record.id],
            enable: newEnable === '1',
          });
          void message.success(t('common.success'));
        } catch (err) {
          void message.error(t('common.operationFailed'));
          console.error('Toggle measurement failed:', err);
        }
      },
    });
  }, [t, modal, message, enableMutation, disableMutation]);

  // 批量启用
  const handleBatchEnable = useCallback((enable: '0' | '1') => {
    if (selectedRowKeys.length === 0) {
      void message.warning(t('common.selectAtLeastOne'));
      return;
    }

    const actionText = enable === '1' ? t('common.enable') : t('common.disable');
    const hasReboot = tableData.some(
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
        try {
          const mutation = enable === '1' ? enableMutation : disableMutation;
          await mutation.mutateAsync({
            deviceType: 'ENB',
            operatorCode: '',
            indicatorIds: selectedRowKeys as string[],
            enable: enable === '1',
          });
          void message.success(t('common.success'));
          setSelectedRowKeys([]);
        } catch (err) {
          void message.error(t('common.operationFailed'));
          console.error('Batch toggle measurement failed:', err);
        }
      },
    });
  }, [selectedRowKeys, tableData, t, modal, message, enableMutation, disableMutation]);

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
      <FilterBar
        filterId="kpi-measurement"
        fields={filterFields}
        onSearch={handleSearch}
        onReset={handleReset}
        collapsedRows={1}
      />

      <Card
        size="small"
        variant="outlined"
        styles={{ body: { padding: 0, display: 'flex', flexDirection: 'column', flex: 1, overflow: 'hidden' } }}
      >
        <DataTable<MeasurementRow>
          tableId="kpi-measurement-table"
          columns={columns}
          dataSource={filteredData}
          loading={deviceLoading}
          rowKey="id"
          selectable
          selectedRowKeys={selectedRowKeys}
          onSelectionChange={setSelectedRowKeys}
          total={totalCount}
          pageSize={pageSize}
          currentPage={page}
          onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
          batchActions={batchActions}
          scroll={{ x: 1200, y: 'calc(100vh - 380px)' }}
          showRowNumber
          rowNumberTitle={t('table.rowNumber')}
          defaultDensity="default"
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
