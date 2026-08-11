import { useState, useMemo, useCallback } from 'react';
import {
  Card,
  Tabs,
  Tag,
  Tooltip,
} from 'antd';
import dayjs from 'dayjs';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import { useOperationLogs } from '@core/hooks/api/useLogs';
import type { OperationLog } from '@core/types/system';
import { useT } from '@/hooks/useT';
import {
  localizeAuditAction,
  localizeAuditReason,
} from './geofenceAuditText';

// 日志类型
// northbound（北向接口日志）暂不在此页展示：后端 northbound 模块只有 push/sync/deadletter，
// 没有「报文日志」列表端点，过去该 tab 复用 useOperationLogs（audit_logs）显示的是错配数据。
// 待后端补北向报文日志端点后再恢复（另开 issue）。
type LogType = 'operation' | 'security' | 'system';

const TAB_ACTION_FILTERS: Record<LogType, string> = {
  operation: 'config,delete,user_create,password_reset',
  security: 'login_success,login_failure,logout,password_change,permission_change',
  system: 'software_upgrade,reboot',
};

function truncate(str: string, maxLen = 50): string {
  if (!str) return '-';
  if (str.length <= maxLen) return str;
  return str.substring(0, maxLen) + '...';
}

export default function OperationLogPage() {
  const t = useT();

  // Tab 状态
  const [activeTab, setActiveTab] = useState<LogType>('operation');

  // 搜索条件
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  // 查询日志
  const { data, isLoading, refetch } = useOperationLogs({
    operator: filters.operator as string | undefined,
    clientIp: filters.operateIp as string | undefined,
    module: filters.logName as string | undefined,
    action: TAB_ACTION_FILTERS[activeTab],
    reason: filters.reason as string | undefined,
    keyword: filters.searchText as string | undefined,
    result: (filters.result as 'success' | 'failure' | undefined),
    timeRange: Array.isArray(filters.timeRange) && filters.timeRange.length === 2
      ? [filters.timeRange[0] as string, filters.timeRange[1] as string]
      : undefined,
    page,
    pageSize,
  });

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
    { label: t('log.modifyConfig'), value: 'config_modify' },
    { label: t('log.deleteDevice'), value: 'device_delete' },
    { label: '用户创建', value: 'user_create' },
    { label: '密码重置', value: 'password_reset' },
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
    { label: t('log.softwareUpgrade'), value: 'software_upgrade' },
    { label: '重启', value: 'reboot' },
  ], [t]);

  // 结果颜色映射
  const resultColorMap: Record<string, string> = {
    '1': 'success',
    '0': 'error',
    success: 'success',
    failure: 'error',
  };

  // 结果文本映射
  const resultTextMap = useMemo(() => ({
    '1': t('log.success'),
    '0': t('log.failure'),
    success: t('log.success'),
    failure: t('log.failure'),
  }), [t]);

  // Mock 用户选项 - 实际应从 API 获取
  const mockUserOptions = useMemo(() => [
    { label: 'admin', value: 'admin' },
    { label: 'operator', value: 'operator' },
    { label: 'viewer', value: 'viewer' },
  ], []);

  // 操作日志筛选字段
  const operationFilterFields: FilterField[] = useMemo(() => [
    { name: 'operator', label: t('log.operator'), type: 'select', options: mockUserOptions, width: 160 },
    { name: 'operateIp', label: t('log.clientIp'), type: 'input', width: 160 },
    { name: 'logName', label: t('log.logName'), type: 'select', options: getOperationLogNameOptions, width: 160 },
    {
      name: 'result',
      label: t('log.result'),
      type: 'select',
      options: [
        { label: t('log.success'), value: 'success' },
        { label: t('log.failure'), value: 'failure' },
      ],
      width: 160,
    },
    { name: 'reason', label: t('log.reason'), type: 'input', width: 160 },
    { name: 'timeRange', label: t('log.timeRange'), type: 'date-range', showTime: true, width: 320 },
  ], [t, mockUserOptions, getOperationLogNameOptions]);

  // 安全日志筛选字段
  const securityFilterFields: FilterField[] = useMemo(() => [
    { name: 'operator', label: t('log.operator'), type: 'select', options: mockUserOptions, width: 160 },
    { name: 'operateIp', label: t('log.clientIp'), type: 'input', width: 160 },
    { name: 'logName', label: t('log.logName'), type: 'select', options: getSecurityLogNameOptions, width: 160 },
    {
      name: 'result',
      label: t('log.result'),
      type: 'select',
      options: [
        { label: t('log.success'), value: 'success' },
        { label: t('log.failure'), value: 'failure' },
      ],
      width: 160,
    },
    { name: 'timeRange', label: t('log.timeRange'), type: 'date-range', showTime: true, width: 320 },
  ], [t, mockUserOptions, getSecurityLogNameOptions]);

  // 系统日志筛选字段
  const systemFilterFields: FilterField[] = useMemo(() => [
    { name: 'operator', label: t('log.operator'), type: 'select', options: mockUserOptions, width: 160 },
    { name: 'operateIp', label: t('log.clientIp'), type: 'input', width: 160 },
    { name: 'logName', label: t('log.logName'), type: 'select', options: getSystemLogNameOptions, width: 160 },
    {
      name: 'result',
      label: t('log.result'),
      type: 'select',
      options: [
        { label: t('log.success'), value: 'success' },
        { label: t('log.failure'), value: 'failure' },
      ],
      width: 160,
    },
    { name: 'timeRange', label: t('log.timeRange'), type: 'date-range', showTime: true, width: 320 },
  ], [t, mockUserOptions, getSystemLogNameOptions]);

  // 根据当前 tab 获取筛选字段
  const getFilterFields = useCallback(() => {
    switch (activeTab) {
      case 'security':
        return securityFilterFields;
      case 'system':
        return systemFilterFields;
      default:
        return operationFilterFields;
    }
  }, [activeTab, operationFilterFields, securityFilterFields, systemFilterFields]);

  // 操作日志表格列
  const operationColumns: DataTableColumn<OperationLog & Record<string, unknown>>[] = useMemo(() => [
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
      render: (val) => localizeAuditAction(String(val ?? ''), t),
    },
    {
      key: 'detail',
      title: t('log.detailContent'),
      dataIndex: 'detail',
      ellipsis: true,
      render: (val) => (
        <Tooltip title={localizeAuditReason(String(val ?? ''), t)}>
          <span>{truncate(localizeAuditReason(String(val ?? ''), t), 50)}</span>
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
        <Tooltip title={localizeAuditReason(String(val ?? ''), t)}>
          <span>{truncate(localizeAuditReason(String(val ?? ''), t), 30)}</span>
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
        return operationColumns.filter((col) => col.key !== 'endTime');
      default:
        return operationColumns;
    }
  }, [activeTab, operationColumns]);

  return (
    <ListPageLayout>
      {/* 四类日志改为标签页(Tabs) */}
      <Tabs
        activeKey={activeTab}
        onChange={handleTabChange}
        items={[
          { key: 'operation', label: `${t('log.operationLog')}（配置/删除/用户管理）` },
          { key: 'security', label: `${t('log.securityLog')}（登录/退出）` },
          { key: 'system', label: '系统管理审计（升级/重启）' },
        ]}
      />

      {/* 导出按钮暂时隐藏：原 useExportLogs 走 logService.exportLogs（mock，返回假 taskId
          不打真实后端），保留假成功提示反而误导用户。待补真实日志导出端点后再恢复。 */}
      <FilterBar
        filterId={`${activeTab}-log-filter`}
        fields={getFilterFields()}
        onSearch={handleSearch}
        onReset={handleReset}
        collapsedRows={1}
      />

      {/* 日志列表 */}
      <Card
        size="small"
        variant="outlined"
        style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}
        styles={{ body: { padding: 0, display: 'flex', flexDirection: 'column', flex: 1, overflow: 'hidden' } }}
      >
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
          hideToolbar
          scroll={{ x: 1400, y: 'calc(100vh - 420px)' }}
        />
      </Card>
    </ListPageLayout>
  );
}
