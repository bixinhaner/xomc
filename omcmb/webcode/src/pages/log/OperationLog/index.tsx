import { useState, useMemo, useCallback } from 'react';
import {
  Button,
  Card,
  Modal,
  Tabs,
  Tag,
  theme,
  Tooltip,
} from 'antd';
import { EyeOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import { useNorthboundAPIInvocationLogs, useOperationLogs } from '@core/hooks/api/useLogs';
import type { NorthboundAPIInvocationLog, OperationLog } from '@core/types/system';
import { useT } from '@/hooks/useT';
import {
  localizeAuditAction,
  localizeAuditReason,
} from './geofenceAuditText';

// 日志类型
type LogType = 'operation' | 'security' | 'system' | 'northbound';
type LogTableRow = (OperationLog | NorthboundAPIInvocationLog) & Record<string, unknown>;
type PayloadDetail = { title: string; content: string };

const TAB_ACTION_FILTERS: Record<Exclude<LogType, 'northbound'>, string> = {
  operation: 'config,delete,user_create,password_reset',
  security: 'login_success,login_failure,logout,password_change,permission_change',
  system: 'software_upgrade,reboot',
};

function truncate(str: string, maxLen = 50): string {
  if (!str) return '-';
  if (str.length <= maxLen) return str;
  return str.substring(0, maxLen) + '...';
}

function normalizePayloadText(value: unknown): string {
  if (value === null || value === undefined) return '';
  if (typeof value === 'string') return value.trim();
  return JSON.stringify(value);
}

function formatPayloadForDisplay(value: unknown): string {
  const text = normalizePayloadText(value);
  if (!text) return '-';
  try {
    return JSON.stringify(JSON.parse(text), null, 2);
  } catch {
    return text;
  }
}

function payloadSummary(value: unknown): string {
  const text = normalizePayloadText(value);
  if (!text) return '-';
  try {
    return truncate(JSON.stringify(JSON.parse(text)), 86);
  } catch {
    return truncate(text.replace(/\s+/g, ' '), 86);
  }
}

export default function OperationLogPage() {
  const t = useT();
  const { token } = theme.useToken();

  // Tab 状态
  const [activeTab, setActiveTab] = useState<LogType>('operation');

  // 搜索条件
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [payloadDetail, setPayloadDetail] = useState<PayloadDetail | null>(null);

  const isNorthboundTab = activeTab === 'northbound';
  const auditTab: Exclude<LogType, 'northbound'> = isNorthboundTab
    ? 'operation'
    : (activeTab as Exclude<LogType, 'northbound'>);
  const timeRange = Array.isArray(filters.timeRange) && filters.timeRange.length === 2
    ? [filters.timeRange[0] as string, filters.timeRange[1] as string] as [string, string]
    : undefined;

  // 查询日志
  const operationLogsQuery = useOperationLogs({
    operator: filters.operator as string | undefined,
    clientIp: filters.operateIp as string | undefined,
    module: filters.logName as string | undefined,
    action: TAB_ACTION_FILTERS[auditTab],
    reason: filters.reason as string | undefined,
    keyword: filters.searchText as string | undefined,
    result: (filters.result as 'success' | 'failure' | undefined),
    timeRange,
    page,
    pageSize,
  }, { enabled: !isNorthboundTab });

  const northboundLogsQuery = useNorthboundAPIInvocationLogs({
    apiKey: filters.apiKey as string | undefined,
    name: filters.apiName as string | undefined,
    method: filters.method as string | undefined,
    path: filters.path as string | undefined,
    status: filters.status as string | undefined,
    createUser: filters.createUser as string | undefined,
    ipAddress: filters.ipAddress as string | undefined,
    timeRange,
    page,
    pageSize,
  }, { enabled: isNorthboundTab });

  const data = isNorthboundTab ? northboundLogsQuery.data : operationLogsQuery.data;
  const isLoading = isNorthboundTab ? northboundLogsQuery.isLoading : operationLogsQuery.isLoading;

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
      if (isNorthboundTab) {
        void northboundLogsQuery.refetch();
        return;
      }
      void operationLogsQuery.refetch();
    }, 0);
  }, [isNorthboundTab, northboundLogsQuery, operationLogsQuery]);

  // Tab 切换时重置
  const handleTabChange = useCallback((key: string) => {
    setActiveTab(key as LogType);
    handleReset();
  }, [handleReset]);

  // 获取操作日志名称选项
  const getOperationLogNameOptions = useMemo(() => [
    { label: t('log.modifyConfig'), value: 'config_modify' },
    { label: t('log.deleteDevice'), value: 'device_delete' },
    { label: t('log.userCreate'), value: 'user_create' },
    { label: t('log.passwordReset'), value: 'password_reset' },
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
    { label: t('log.reboot'), value: 'reboot' },
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

  const openPayloadDetail = useCallback((title: string, value: unknown) => {
    setPayloadDetail({ title, content: formatPayloadForDisplay(value) });
  }, []);

  const renderPayloadCell = useCallback((value: unknown, title: string) => {
    const text = normalizePayloadText(value);
    if (!text) return <span>-</span>;
    const summary = payloadSummary(text);
    return (
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, minWidth: 0 }}>
        <span
          style={{
            flex: 1,
            minWidth: 0,
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
            fontFamily: 'monospace',
            fontSize: 12,
          }}
        >
          {summary}
        </span>
        <Tooltip title={t('log.viewPayload', { name: title })}>
          <Button
            aria-label={t('log.viewPayload', { name: title })}
            icon={<EyeOutlined />}
            size="small"
            type="text"
            onClick={() => openPayloadDetail(title, text)}
          />
        </Tooltip>
      </div>
    );
  }, [openPayloadDetail, t]);

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

  const northboundFilterFields: FilterField[] = useMemo(() => [
    { name: 'apiKey', label: 'API Key', type: 'input', width: 180 },
    { name: 'apiName', label: t('log.northbound.apiName'), type: 'input', width: 180 },
    {
      name: 'method',
      label: t('log.northbound.method'),
      type: 'select',
      options: [
        { label: 'GET', value: 'GET' },
        { label: 'POST', value: 'POST' },
        { label: 'PUT', value: 'PUT' },
        { label: 'DELETE', value: 'DELETE' },
      ],
      width: 120,
    },
    { name: 'path', label: 'PATH', type: 'input', width: 240 },
    { name: 'createUser', label: t('log.northbound.callAccount'), type: 'input', width: 160 },
    { name: 'ipAddress', label: t('log.clientIp'), type: 'input', width: 160 },
    {
      name: 'status',
      label: t('log.result'),
      type: 'select',
      options: [
        { label: t('log.success'), value: 'success' },
        { label: t('log.failure'), value: 'failed' },
      ],
      width: 140,
    },
    { name: 'timeRange', label: t('log.timeRange'), type: 'date-range', showTime: true, width: 320 },
  ], [t]);

  // 根据当前 tab 获取筛选字段
  const getFilterFields = useCallback(() => {
    switch (activeTab) {
      case 'northbound':
        return northboundFilterFields;
      case 'security':
        return securityFilterFields;
      case 'system':
        return systemFilterFields;
      default:
        return operationFilterFields;
    }
  }, [activeTab, northboundFilterFields, operationFilterFields, securityFilterFields, systemFilterFields]);

  // 操作日志表格列
  const operationColumns: DataTableColumn<LogTableRow>[] = useMemo(() => [
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

  const northboundColumns: DataTableColumn<LogTableRow>[] = useMemo(() => [
    {
      key: 'name',
      title: t('log.northbound.apiName'),
      dataIndex: 'name',
      width: 180,
      ellipsis: true,
      render: (val) => <span>{String(val || '-')}</span>,
    },
    {
      key: 'apiKey',
      title: 'API Key',
      dataIndex: 'apiKey',
      width: 170,
      render: (val) => <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{String(val || '-')}</span>,
    },
    {
      key: 'method',
      title: t('log.northbound.method'),
      dataIndex: 'method',
      width: 90,
      render: (val) => <Tag color="processing">{String(val || '-')}</Tag>,
    },
    {
      key: 'path',
      title: 'PATH',
      dataIndex: 'path',
      width: 300,
      ellipsis: true,
      render: (val) => (
        <Tooltip title={String(val || '-')}>
          <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{truncate(String(val || '-'), 60)}</span>
        </Tooltip>
      ),
    },
    {
      key: 'createUser',
      title: t('log.northbound.callAccount'),
      dataIndex: 'createUser',
      width: 130,
    },
    {
      key: 'ipAddress',
      title: t('log.clientIp'),
      dataIndex: 'ipAddress',
      width: 150,
      render: (val) => <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{String(val || '-')}</span>,
    },
    {
      key: 'status',
      title: t('log.result'),
      dataIndex: 'status',
      width: 100,
      render: (val, record) => {
        const status = String(val || '');
        const statusCode = Number(record.statusCode || 0);
        const failed = status === 'failed' || statusCode >= 400;
        return <Tag color={failed ? 'error' : 'success'}>{failed ? t('log.failure') : t('log.success')}</Tag>;
      },
    },
    {
      key: 'statusCode',
      title: t('log.northbound.statusCode'),
      dataIndex: 'statusCode',
      width: 90,
    },
    {
      key: 'durationMs',
      title: t('log.northbound.durationMs'),
      dataIndex: 'durationMs',
      width: 100,
    },
    {
      key: 'requestParams',
      title: t('log.northbound.requestMessage'),
      dataIndex: 'requestParams',
      width: 300,
      ellipsis: true,
      render: (val) => renderPayloadCell(val, t('log.northbound.requestMessage')),
    },
    {
      key: 'responseBody',
      title: t('log.northbound.responseMessage'),
      dataIndex: 'responseBody',
      width: 300,
      ellipsis: true,
      render: (val) => renderPayloadCell(val, t('log.northbound.responseMessage')),
    },
    {
      key: 'createdAt',
      title: t('log.northbound.callTime'),
      dataIndex: 'createdAt',
      width: 160,
      render: (val) => (val ? dayjs(String(val)).format('YYYY-MM-DD HH:mm:ss') : '-'),
    },
  ], [renderPayloadCell, t]);

  // 获取表格列
  const getColumns = useCallback(() => {
    switch (activeTab) {
      case 'northbound':
        return northboundColumns;
      case 'security':
        return operationColumns.filter((col) => col.key !== 'endTime');
      case 'system':
        return operationColumns.filter((col) => col.key !== 'endTime');
      default:
        return operationColumns;
    }
  }, [activeTab, northboundColumns, operationColumns]);

  return (
    <ListPageLayout>
      {/* 四类日志改为标签页(Tabs) */}
      <Tabs
        activeKey={activeTab}
        onChange={handleTabChange}
        items={[
          { key: 'operation', label: t('log.operationLogTab') },
          { key: 'security', label: t('log.securityLogTab') },
          { key: 'system', label: t('log.systemAuditLogTab') },
          { key: 'northbound', label: t('log.northboundLog') },
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
        <DataTable<LogTableRow>
          tableId={`${activeTab}-log-list`}
          columns={getColumns()}
          dataSource={(data?.items ?? []) as LogTableRow[]}
          loading={isLoading}
          rowKey="id"
          total={data?.total ?? 0}
          pageSize={pageSize}
          currentPage={page}
          onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
          hideToolbar
          scroll={{ x: isNorthboundTab ? 1900 : 1400, y: 'calc(100vh - 420px)' }}
        />
      </Card>

      <Modal
        title={payloadDetail?.title}
        open={Boolean(payloadDetail)}
        onCancel={() => setPayloadDetail(null)}
        footer={null}
        width="min(960px, 92vw)"
      >
        <pre
          style={{
            maxHeight: '65vh',
            overflow: 'auto',
            margin: 0,
            padding: 12,
            border: `1px solid ${token.colorBorderSecondary}`,
            borderRadius: token.borderRadiusSM,
            background: token.colorFillQuaternary,
            color: token.colorText,
            fontFamily: 'Menlo, Monaco, Consolas, monospace',
            fontSize: 12,
            lineHeight: 1.6,
            whiteSpace: 'pre-wrap',
            wordBreak: 'break-word',
          }}
        >
          {payloadDetail?.content || '-'}
        </pre>
      </Modal>
    </ListPageLayout>
  );
}
