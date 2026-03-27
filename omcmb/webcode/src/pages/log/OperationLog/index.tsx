import { useState, useMemo, useCallback } from 'react';
import {
  App,
  Button,
  Card,
  Radio,
  Tag,
  Space,
  Tooltip,
} from 'antd';
import {
  DownloadOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import { useOperationLogs, useExportLogs } from '@/hooks/api/useLogs';
import type { OperationLog } from '@/types/system';
import { useT } from '@/hooks/useT';

// 日志类型
type LogType = 'operation' | 'security' | 'system' | 'northbound';

function truncate(str: string, maxLen = 50): string {
  if (!str) return '-';
  if (str.length <= maxLen) return str;
  return str.substring(0, maxLen) + '...';
}

export default function OperationLogPage() {
  const t = useT();
  const { message } = App.useApp();

  // Tab 状态
  const [activeTab, setActiveTab] = useState<LogType>('operation');

  // 搜索条件
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  // 查询日志
  const { data, isLoading, refetch } = useOperationLogs({
    operator: filters.operator as string | undefined,
    module: filters.logName as string | undefined,
    keyword: filters.searchText as string | undefined,
    result: filters.result as 'success' | 'failure' | undefined,
    timeRange: filters.startTime && filters.endTime
      ? [filters.startTime as string, filters.endTime as string]
      : undefined,
    page,
    pageSize,
  });

  // 导出
  const exportLogs = useExportLogs();

  // 处理导出
  const handleExport = useCallback(() => {
    exportLogs.mutate(
      { type: 'operation', params: filters },
      {
        onSuccess: () => {
          void message.success(t('log.exportSuccess'));
        },
      }
    );
  }, [filters, exportLogs, message, t]);

  // 重置搜索
  const handleReset = useCallback(() => {
    setFilters({});
    setPage(1);
  }, []);

  // 搜索
  const handleSearch = useCallback((values: Record<string, unknown>) => {
    setFilters(values);
    setPage(1);
    setTimeout(() => {
      void refetch();
    }, 0);
  }, [refetch]);

  // Tab 切换时重置
  const handleTabChange = useCallback((key: string) => {
    setActiveTab(key as LogType);
    handleReset();
  }, [handleReset]);

  // 获取操作日志名称选项
  const getOperationLogNameOptions = useMemo(() => [
    { label: t('log.userLogin'), value: 'user_login' },
    { label: t('log.userLogout'), value: 'user_logout' },
    { label: t('log.addDevice'), value: 'device_add' },
    { label: t('log.deleteDevice'), value: 'device_delete' },
    { label: t('log.modifyConfig'), value: 'config_modify' },
    { label: t('log.softwareUpgrade'), value: 'software_upgrade' },
    { label: t('log.exportData'), value: 'data_export' },
    { label: t('log.importData'), value: 'data_import' },
  ], [t]);

  // 获取安全日志名称选项
  const getSecurityLogNameOptions = useMemo(() => [
    { label: t('log.loginSuccess'), value: 'login_success' },
    { label: t('log.loginFailure'), value: 'login_failure' },
    { label: t('log.logout'), value: 'logout' },
    { label: t('log.passwordChange'), value: 'password_change' },
    { label: t('log.permissionChange'), value: 'permission_change' },
  ], [t]);

  // 获取系统日志名称选项
  const getSystemLogNameOptions = useMemo(() => [
    { label: t('log.systemStart'), value: 'system_start' },
    { label: t('log.systemStop'), value: 'system_stop' },
    { label: t('log.configBackup'), value: 'config_backup' },
    { label: t('log.configRestore'), value: 'config_restore' },
    { label: t('log.dbBackup'), value: 'db_backup' },
  ], [t]);

  // 获取北向接口类型选项
  const getNorthboundTypeOptions = useMemo(() => [
    { label: t('log.all'), value: '' },
    { label: t('log.syncRequest'), value: '1' },
    { label: t('log.asyncRequest'), value: '2' },
    { label: t('log.alarm'), value: 'Real Alarm' },
    { label: t('log.syncMsg'), value: 'Sync Msg' },
    { label: t('log.login'), value: 'Login' },
    { label: t('log.syncFile'), value: 'Sync File' },
    { label: t('log.disconnection'), value: 'Disconnection' },
    { label: t('log.connection'), value: 'Connection' },
    { label: t('log.connectionTimeout'), value: 'Connection Timeout' },
    { label: t('log.idleTimeout'), value: 'Idle Timeout' },
    { label: t('log.heartbeat'), value: 'HEARTBEAT' },
  ], [t]);

  // 北向接口类型显示映射
  const northboundTypeTextMap = useMemo(() => ({
    '1': t('log.syncRequest'),
    '2': t('log.asyncRequest'),
    'Real Alarm': t('log.alarm'),
    'Sync Msg': t('log.syncMsg'),
    'Login': t('log.login'),
    'Sync File': t('log.syncFile'),
    'Disconnection': t('log.disconnection'),
    'Connection': t('log.connection'),
    'Connection Timeout': t('log.connectionTimeout'),
    'Idle Timeout': t('log.idleTimeout'),
    'HEARTBEAT': t('log.heartbeat'),
  }), [t]);

  // 结果颜色映射
  const resultColorMap: Record<string, string> = {
    '1': 'success',
    '0': 'error',
  };

  // 结果文本映射
  const resultTextMap = useMemo(() => ({
    '1': t('log.success'),
    '0': t('log.failure'),
  }), [t]);

  // Mock 用户选项 - 实际应从 API 获取
  const mockUserOptions = useMemo(() => [
    { label: 'admin', value: 'admin' },
    { label: 'operator', value: 'operator' },
    { label: 'viewer', value: 'viewer' },
  ], []);

  // 操作日志筛选字段
  const operationFilterFields: FilterField[] = useMemo(() => [
    { name: 'operator', label: t('log.operator'), type: 'select', options: mockUserOptions },
    { name: 'operateIp', label: t('log.clientIp'), type: 'input' },
    { name: 'logName', label: t('log.logName'), type: 'select', options: getOperationLogNameOptions },
    {
      name: 'result',
      label: t('log.result'),
      type: 'select',
      options: [
        { label: t('log.success'), value: '1' },
        { label: t('log.failure'), value: '0' },
      ],
    },
    { name: 'reason', label: t('log.reason'), type: 'input' },
    { name: 'timeRange', label: t('log.timeRange'), type: 'date-range', span: 2 },
  ], [t, mockUserOptions, getOperationLogNameOptions]);

  // 安全日志筛选字段
  const securityFilterFields: FilterField[] = useMemo(() => [
    { name: 'id', label: 'ID', type: 'input' },
    { name: 'operator', label: t('log.operator'), type: 'select', options: mockUserOptions },
    { name: 'operateIp', label: t('log.clientIp'), type: 'input' },
    { name: 'logName', label: t('log.logName'), type: 'select', options: getSecurityLogNameOptions },
    {
      name: 'result',
      label: t('log.result'),
      type: 'select',
      options: [
        { label: t('log.success'), value: '1' },
        { label: t('log.failure'), value: '0' },
      ],
    },
    { name: 'timeRange', label: t('log.timeRange'), type: 'date-range', span: 2 },
  ], [t, mockUserOptions, getSecurityLogNameOptions]);

  // 系统日志筛选字段
  const systemFilterFields: FilterField[] = useMemo(() => [
    { name: 'id', label: 'ID', type: 'input' },
    { name: 'logName', label: t('log.logName'), type: 'select', options: getSystemLogNameOptions },
    {
      name: 'result',
      label: t('log.result'),
      type: 'select',
      options: [
        { label: t('log.success'), value: '1' },
        { label: t('log.failure'), value: '0' },
      ],
    },
    { name: 'timeRange', label: t('log.timeRange'), type: 'date-range', span: 2 },
  ], [t, getSystemLogNameOptions]);

  // 北向接口日志筛选字段
  const northboundFilterFields: FilterField[] = useMemo(() => [
    { name: 'ipAddress', label: t('log.ipAddress'), type: 'input' },
    { name: 'name', label: t('log.name'), type: 'input' },
    { name: 'type', label: t('log.type'), type: 'select', options: getNorthboundTypeOptions },
    { name: 'timeRange', label: t('log.timeRange'), type: 'date-range', span: 2 },
  ], [t, getNorthboundTypeOptions]);

  // 根据当前 tab 获取筛选字段
  const getFilterFields = useCallback(() => {
    switch (activeTab) {
      case 'security':
        return securityFilterFields;
      case 'system':
        return systemFilterFields;
      case 'northbound':
        return northboundFilterFields;
      default:
        return operationFilterFields;
    }
  }, [activeTab, operationFilterFields, securityFilterFields, systemFilterFields, northboundFilterFields]);

  // 操作日志表格列
  const operationColumns: DataTableColumn<OperationLog & Record<string, unknown>>[] = useMemo(() => [
    {
      key: 'id',
      title: 'ID',
      dataIndex: 'id',
      width: 80,
    },
    {
      key: 'operator',
      title: t('log.operator'),
      dataIndex: 'operator',
      width: 150,
    },
    {
      key: 'clientIp',
      title: t('log.clientIp'),
      dataIndex: 'clientIp',
      width: 200,
      render: (val) => <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{String(val)}</span>,
    },
    {
      key: 'logName',
      title: t('log.logName'),
      dataIndex: 'logName',
      width: 200,
      render: (val) => val || '-',
    },
    {
      key: 'detail',
      title: t('log.detailContent'),
      dataIndex: 'detail',
      ellipsis: true,
      render: (val) => (
        <Tooltip title={String(val)}>
          <span>{truncate(String(val), 50)}</span>
        </Tooltip>
      ),
    },
    {
      key: 'result',
      title: t('log.result'),
      dataIndex: 'result',
      width: 100,
      render: (val) => {
        const result = String(val);
        return (
          <Tag color={resultColorMap[result] || 'default'}>
            {resultTextMap[result as keyof typeof resultTextMap] || result}
          </Tag>
        );
      },
    },
    {
      key: 'reason',
      title: t('log.reason'),
      dataIndex: 'reason',
      width: 200,
      ellipsis: true,
      render: (val) => (
        <Tooltip title={String(val)}>
          <span>{truncate(String(val), 30)}</span>
        </Tooltip>
      ),
    },
    {
      key: 'startTime',
      title: t('log.startTime'),
      dataIndex: 'startTime',
      width: 150,
      render: (val) => (val ? dayjs(String(val)).format('YYYY-MM-DD HH:mm:ss') : '-'),
    },
    {
      key: 'endTime',
      title: t('log.endTime'),
      dataIndex: 'endTime',
      width: 150,
      render: (val) => (val ? dayjs(String(val)).format('YYYY-MM-DD HH:mm:ss') : '-'),
    },
  ], [t, resultTextMap]);

  // 获取表格列
  const getColumns = useCallback(() => {
    switch (activeTab) {
      case 'security':
        return operationColumns.filter((col) => col.key !== 'endTime');
      case 'system':
        return operationColumns.filter((col) => !['operator', 'clientIp', 'endTime'].includes(col.key as string));
      case 'northbound':
        return [
          { key: 'id', title: 'ID', dataIndex: 'id', width: 80 },
          { key: 'logName', title: t('log.logName'), dataIndex: 'logName', width: 300 },
          { key: 'clientIp', title: t('log.clientIp'), dataIndex: 'clientIp', width: 200 },
          {
            key: 'type',
            title: t('log.type'),
            dataIndex: 'type',
            width: 150,
            render: (val: string) => northboundTypeTextMap[val as keyof typeof northboundTypeTextMap] || val,
          },
          { key: 'reqParams', title: t('log.reqParams'), dataIndex: 'reqParams', width: 250, ellipsis: true },
          { key: 'resParams', title: t('log.resParams'), dataIndex: 'resParams', ellipsis: true },
          {
            key: 'createTime',
            title: t('log.createTime'),
            dataIndex: 'createTime',
            width: 200,
            render: (val: string) => (val ? dayjs(val).format('YYYY-MM-DD HH:mm:ss') : '-'),
          },
        ];
      default:
        return operationColumns;
    }
  }, [activeTab, operationColumns, t, northboundTypeTextMap]);

  return (
    <ListPageLayout title={t('log.operationLog')}>
      {/* 覆盖 FilterBar 样式 */}
      <style>{`
        .operation-log-filter-wrapper [class*="_filterBarWrapper_"] {
          padding: 0 !important;
          margin-bottom: 0 !important;
        }
      `}</style>
      {/* Tab 页签 + 工具栏 */}
      <Card bordered={false} style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Radio.Group
            value={activeTab}
            onChange={(e) => handleTabChange(e.target.value)}
            optionType="button"
            buttonStyle="solid"
          >
            <Radio.Button value="operation">{t('log.operationLog')}</Radio.Button>
            <Radio.Button value="security">{t('log.securityLog')}</Radio.Button>
            <Radio.Button value="system">{t('log.systemLog')}</Radio.Button>
            <Radio.Button value="northbound">{t('log.northboundLog')}</Radio.Button>
          </Radio.Group>
          <Space>
            <Button
              type="primary"
              icon={<DownloadOutlined />}
              onClick={handleExport}
              loading={exportLogs.isPending}
            >
              {t('common.export')}
            </Button>
          </Space>
        </div>
      </Card>

      {/* 搜索表单 */}
      <Card bordered={false} style={{ marginBottom: 16 }} className="operation-log-filter-wrapper">
        <FilterBar
          filterId={`${activeTab}-log-filter`}
          fields={getFilterFields()}
          onSearch={handleSearch}
          onReset={handleReset}
          collapsedRows={1}
        />
      </Card>

      {/* 日志列表 */}
      <Card bordered={false}>
        <DataTable<OperationLog & Record<string, unknown>>
          tableId={`${activeTab}-log-list`}
          columns={getColumns() as DataTableColumn<OperationLog & Record<string, unknown>>[]}
          dataSource={(data?.items ?? []) as (OperationLog & Record<string, unknown>)[]}
          loading={isLoading}
          rowKey="id"
          total={data?.total ?? 0}
          pageSize={pageSize}
          currentPage={page}
          onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
          onRefresh={() => void refetch()}
          scroll={{ x: 1400, y: 'calc(100vh - 420px)' }}
        />
      </Card>
    </ListPageLayout>
  );
}
