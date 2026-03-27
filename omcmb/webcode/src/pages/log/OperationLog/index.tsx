import { useState, useMemo, useCallback } from 'react';
import {
  App,
  Button,
  Card,
  DatePicker,
  Form,
  Input,
  Select,
  Tabs,
  Tag,
  Space,
  Tooltip,
  Dropdown,
} from 'antd';
import type { TabsProps, MenuProps } from 'antd';
import {
  DownloadOutlined,
  MoreOutlined,
  EyeOutlined,
  HistoryOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import LogDetailDrawer from './components/LogDetailDrawer';
import { useOperationLogs, useExportLogs } from '@/hooks/api/useLogs';
import type { OperationLog } from '@/types/system';
import { useT } from '@/hooks/useT';

const { RangePicker } = DatePicker;

// 结果颜色映射
const resultColorMap: Record<string, string> = {
  '1': 'success',
  '0': 'error',
};

// 结果文本映射
const resultTextMap: Record<string, string> = {
  '1': '成功',
  '0': '失败',
};

// 北向接口类型选项
const northboundTypeOptions = [
  { label: '所有', value: '' },
  { label: '同步请求', value: '1' },
  { label: '异步请求', value: '2' },
  { label: '告警', value: 'Real Alarm' },
  { label: '同步消息', value: 'Sync Msg' },
  { label: '登录', value: 'Login' },
  { label: '同步文件', value: 'Sync File' },
  { label: '非连接', value: 'Disconnection' },
  { label: '连接', value: 'Connection' },
  { label: '连接超时', value: 'Connection Timeout' },
  { label: '空闲超时', value: 'Idle Timeout' },
  { label: '心跳', value: 'HEARTBEAT' },
];

// 北向接口类型显示映射
const northboundTypeTextMap: Record<string, string> = {
  '1': '同步请求',
  '2': '异步请求',
  'Real Alarm': '告警',
  'Sync Msg': '同步消息',
  'Login': '登录',
  'Sync File': '同步文件',
  'Disconnection': '非连接',
  'Connection': '连接',
  'Connection Timeout': '连接超时',
  'Idle Timeout': '空闲超时',
  'HEARTBEAT': '心跳',
};

// Mock 数据 - 实际应从 API 获取
const mockUserOptions = [
  { label: 'admin', value: 'admin' },
  { label: 'operator', value: 'operator' },
  { label: 'viewer', value: 'viewer' },
];

const mockLogNameOptions = [
  { label: '用户登录', value: 'user_login' },
  { label: '用户登出', value: 'user_logout' },
  { label: '添加设备', value: 'device_add' },
  { label: '删除设备', value: 'device_delete' },
  { label: '修改配置', value: 'config_modify' },
  { label: '软件升级', value: 'software_upgrade' },
  { label: '导出数据', value: 'data_export' },
  { label: '导入数据', value: 'data_import' },
];

const mockSecurityLogNameOptions = [
  { label: '登录成功', value: 'login_success' },
  { label: '登录失败', value: 'login_failure' },
  { label: '退出登录', value: 'logout' },
  { label: '密码修改', value: 'password_change' },
  { label: '权限变更', value: 'permission_change' },
];

const mockSystemLogNameOptions = [
  { label: '系统启动', value: 'system_start' },
  { label: '系统关闭', value: 'system_stop' },
  { label: '配置备份', value: 'config_backup' },
  { label: '配置恢复', value: 'config_restore' },
  { label: '数据库备份', value: 'db_backup' },
];

function truncate(str: string, maxLen = 50): string {
  if (!str) return '-';
  if (str.length <= maxLen) return str;
  return str.substring(0, maxLen) + '...';
}

// 日志类型
type LogType = 'operation' | 'security' | 'system' | 'northbound';

export default function OperationLogPage() {
  const t = useT();
  const { message } = App.useApp();

  // Tab 状态
  const [activeTab, setActiveTab] = useState<LogType>('operation');

  // 搜索条件
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  // 时间范围
  const [timeRange, setTimeRange] = useState<[dayjs.Dayjs | null, dayjs.Dayjs | null] | null>(null);

  // 详情抽屉
  const [detailVisible, setDetailVisible] = useState(false);
  const [selectedLog, setSelectedLog] = useState<OperationLog | null>(null);

  // 查询日志
  const { data, isLoading, refetch } = useOperationLogs({
    operator: filters.operator as string | undefined,
    module: filters.logName as string | undefined,
    keyword: filters.searchText as string | undefined,
    result: filters.result as 'success' | 'failure' | undefined,
    timeRange: timeRange?.[0] && timeRange?.[1]
      ? [timeRange[0]!.format('YYYY-MM-DD HH:mm:ss'), timeRange[1]!.format('YYYY-MM-DD HH:mm:ss')]
      : undefined,
    page,
    pageSize,
  });

  // 导出
  const exportLogs = useExportLogs();

  // 处理导出
  const handleExport = useCallback(() => {
    const params = {
      ...filters,
      startTime: timeRange?.[0]?.format('YYYY-MM-DD HH:mm:ss'),
      endTime: timeRange?.[1]?.format('YYYY-MM-DD HH:mm:ss'),
    };
    exportLogs.mutate(
      { type: 'operation', params: params as Record<string, unknown> },
      {
        onSuccess: () => {
          void message.success(t('log.exportSuccess'));
        },
      }
    );
  }, [filters, timeRange, exportLogs, message, t]);

  // 查看详情
  const handleViewDetail = useCallback((log: OperationLog) => {
    setSelectedLog(log);
    setDetailVisible(true);
  }, []);

  // 重置搜索
  const handleReset = useCallback(() => {
    setFilters({});
    setTimeRange(null);
    setPage(1);
  }, []);

  // 搜索
  const handleSearch = useCallback(() => {
    setPage(1);
    void refetch();
  }, [refetch]);

  // Tab 切换时重置
  const handleTabChange = useCallback((key: string) => {
    setActiveTab(key as LogType);
    handleReset();
  }, [handleReset]);

  // 渲染操作日志搜索表单
  const renderOperationSearchForm = () => (
    <Card bordered={false} style={{ marginBottom: 16 }}>
      <Form layout="inline" style={{ gap: 16 }}>
        <Form.Item label={t('log.operator')}>
          <Select
            allowClear
            style={{ width: 150 }}
            placeholder={t('common.pleaseSelect')}
            options={mockUserOptions}
            value={filters.operator as string | undefined}
            onChange={(val) => setFilters((prev) => ({ ...prev, operator: val }))}
          />
        </Form.Item>
        <Form.Item label={t('log.clientIp')}>
          <Input
            style={{ width: 150 }}
            placeholder={t('common.pleaseInput')}
            value={filters.operateIp as string | undefined}
            onChange={(e) => setFilters((prev) => ({ ...prev, operateIp: e.target.value }))}
          />
        </Form.Item>
        <Form.Item label={t('log.logName')}>
          <Select
            allowClear
            style={{ width: 150 }}
            placeholder={t('common.pleaseSelect')}
            options={mockLogNameOptions}
            value={filters.logName as string | undefined}
            onChange={(val) => setFilters((prev) => ({ ...prev, logName: val }))}
          />
        </Form.Item>
        <Form.Item label={t('log.result')}>
          <Select
            allowClear
            style={{ width: 120 }}
            placeholder={t('common.all')}
            options={[
              { label: t('status.success'), value: '1' },
              { label: t('status.failed'), value: '0' },
            ]}
            value={filters.result as string | undefined}
            onChange={(val) => setFilters((prev) => ({ ...prev, result: val }))}
          />
        </Form.Item>
        <Form.Item label={t('log.reason')}>
          <Input
            style={{ width: 150 }}
            placeholder={t('common.pleaseInput')}
            value={filters.reason as string | undefined}
            onChange={(e) => setFilters((prev) => ({ ...prev, reason: e.target.value }))}
          />
        </Form.Item>
        <Form.Item label={t('log.timeRange')}>
          <RangePicker
            showTime
            value={timeRange}
            onChange={(dates) => setTimeRange(dates)}
            format="YYYY-MM-DD HH:mm:ss"
            placeholder={[t('log.startTime'), t('log.endTime')]}
            style={{ width: 360 }}
          />
        </Form.Item>
        <Form.Item>
          <Space>
            <Button type="primary" onClick={handleSearch}>
              {t('common.search')}
            </Button>
            <Button onClick={handleReset}>
              {t('common.reset')}
            </Button>
          </Space>
        </Form.Item>
      </Form>
    </Card>
  );

  // 渲染安全日志搜索表单
  const renderSecuritySearchForm = () => (
    <Card bordered={false} style={{ marginBottom: 16 }}>
      <Form layout="inline" style={{ gap: 16 }}>
        <Form.Item label="ID">
          <Input
            style={{ width: 100 }}
            placeholder="ID"
            value={filters.id as string | undefined}
            onChange={(e) => setFilters((prev) => ({ ...prev, id: e.target.value }))}
          />
        </Form.Item>
        <Form.Item label={t('log.operator')}>
          <Select
            allowClear
            style={{ width: 150 }}
            placeholder={t('common.pleaseSelect')}
            options={mockUserOptions}
            value={filters.operator as string | undefined}
            onChange={(val) => setFilters((prev) => ({ ...prev, operator: val }))}
          />
        </Form.Item>
        <Form.Item label={t('log.clientIp')}>
          <Input
            style={{ width: 150 }}
            placeholder={t('common.pleaseInput')}
            value={filters.operateIp as string | undefined}
            onChange={(e) => setFilters((prev) => ({ ...prev, operateIp: e.target.value }))}
          />
        </Form.Item>
        <Form.Item label={t('log.logName')}>
          <Select
            allowClear
            style={{ width: 150 }}
            placeholder={t('common.pleaseSelect')}
            options={mockSecurityLogNameOptions}
            value={filters.logName as string | undefined}
            onChange={(val) => setFilters((prev) => ({ ...prev, logName: val }))}
          />
        </Form.Item>
        <Form.Item label={t('log.result')}>
          <Select
            allowClear
            style={{ width: 120 }}
            placeholder={t('common.all')}
            options={[
              { label: t('status.success'), value: '1' },
              { label: t('status.failed'), value: '0' },
            ]}
            value={filters.result as string | undefined}
            onChange={(val) => setFilters((prev) => ({ ...prev, result: val }))}
          />
        </Form.Item>
        <Form.Item label={t('log.timeRange')}>
          <RangePicker
            showTime
            value={timeRange}
            onChange={(dates) => setTimeRange(dates)}
            format="YYYY-MM-DD HH:mm:ss"
            placeholder={[t('log.startTime'), t('log.endTime')]}
            style={{ width: 360 }}
          />
        </Form.Item>
        <Form.Item>
          <Space>
            <Button type="primary" onClick={handleSearch}>
              {t('common.search')}
            </Button>
            <Button onClick={handleReset}>
              {t('common.reset')}
            </Button>
          </Space>
        </Form.Item>
      </Form>
    </Card>
  );

  // 渲染系统日志搜索表单
  const renderSystemSearchForm = () => (
    <Card bordered={false} style={{ marginBottom: 16 }}>
      <Form layout="inline" style={{ gap: 16 }}>
        <Form.Item label="ID">
          <Input
            style={{ width: 100 }}
            placeholder="ID"
            value={filters.id as string | undefined}
            onChange={(e) => setFilters((prev) => ({ ...prev, id: e.target.value }))}
          />
        </Form.Item>
        <Form.Item label={t('log.logName')}>
          <Select
            allowClear
            style={{ width: 180 }}
            placeholder={t('common.pleaseSelect')}
            options={mockSystemLogNameOptions}
            value={filters.logName as string | undefined}
            onChange={(val) => setFilters((prev) => ({ ...prev, logName: val }))}
          />
        </Form.Item>
        <Form.Item label={t('log.result')}>
          <Select
            allowClear
            style={{ width: 120 }}
            placeholder={t('common.all')}
            options={[
              { label: t('status.success'), value: '1' },
              { label: t('status.failed'), value: '0' },
            ]}
            value={filters.result as string | undefined}
            onChange={(val) => setFilters((prev) => ({ ...prev, result: val }))}
          />
        </Form.Item>
        <Form.Item label={t('log.timeRange')}>
          <RangePicker
            showTime
            value={timeRange}
            onChange={(dates) => setTimeRange(dates)}
            format="YYYY-MM-DD HH:mm:ss"
            placeholder={[t('log.startTime'), t('log.endTime')]}
            style={{ width: 360 }}
          />
        </Form.Item>
        <Form.Item>
          <Space>
            <Button type="primary" onClick={handleSearch}>
              {t('common.search')}
            </Button>
            <Button onClick={handleReset}>
              {t('common.reset')}
            </Button>
          </Space>
        </Form.Item>
      </Form>
    </Card>
  );

  // 渲染北向接口日志搜索表单
  const renderNorthboundSearchForm = () => (
    <Card bordered={false} style={{ marginBottom: 16 }}>
      <Form layout="inline" style={{ gap: 16 }}>
        <Form.Item label={t('log.clientIp')}>
          <Input
            style={{ width: 150 }}
            placeholder={t('common.pleaseInput')}
            value={filters.ipAddress as string | undefined}
            onChange={(e) => setFilters((prev) => ({ ...prev, ipAddress: e.target.value }))}
          />
        </Form.Item>
        <Form.Item label={t('log.logName')}>
          <Input
            style={{ width: 150 }}
            placeholder={t('common.pleaseInput')}
            value={filters.name as string | undefined}
            onChange={(e) => setFilters((prev) => ({ ...prev, name: e.target.value }))}
          />
        </Form.Item>
        <Form.Item label={t('log.type')}>
          <Select
            allowClear
            style={{ width: 150 }}
            placeholder={t('common.all')}
            options={northboundTypeOptions}
            value={filters.type as string | undefined}
            onChange={(val) => setFilters((prev) => ({ ...prev, type: val }))}
          />
        </Form.Item>
        <Form.Item label={t('log.timeRange')}>
          <RangePicker
            showTime
            value={timeRange}
            onChange={(dates) => setTimeRange(dates)}
            format="YYYY-MM-DD HH:mm:ss"
            placeholder={[t('log.startTime'), t('log.endTime')]}
            style={{ width: 360 }}
          />
        </Form.Item>
        <Form.Item>
          <Space>
            <Button type="primary" onClick={handleSearch}>
              {t('common.search')}
            </Button>
            <Button onClick={handleReset}>
              {t('common.reset')}
            </Button>
          </Space>
        </Form.Item>
      </Form>
    </Card>
  );

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
            {resultTextMap[result] || result}
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
    {
      key: 'operation',
      title: t('table.operation'),
      width: 80,
      fixed: 'right',
      render: (_, record) => {
        const log = record as OperationLog;
        const items: MenuProps['items'] = [
          {
            key: 'view',
            label: t('common.view'),
            icon: <EyeOutlined />,
            onClick: () => handleViewDetail(log),
          },
        ];
        return (
          <Dropdown menu={{ items }} trigger={['click']}>
            <Button type="link" size="small" icon={<MoreOutlined />} />
          </Dropdown>
        );
      },
    },
  ], [t, handleViewDetail]);

  // Tab 项
  const tabItems: TabsProps['items'] = useMemo(() => [
    {
      key: 'operation',
      label: t('log.operationLog'),
    },
    {
      key: 'security',
      label: t('log.securityLog'),
    },
    {
      key: 'system',
      label: t('log.systemLog'),
    },
    {
      key: 'northbound',
      label: t('log.northboundLog'),
    },
  ], [t]);

  // 渲染搜索表单
  const renderSearchForm = () => {
    switch (activeTab) {
      case 'security':
        return renderSecuritySearchForm();
      case 'system':
        return renderSystemSearchForm();
      case 'northbound':
        return renderNorthboundSearchForm();
      default:
        return renderOperationSearchForm();
    }
  };

  // 获取表格列
  const getColumns = () => {
    switch (activeTab) {
      case 'security':
        return [
          { key: 'id', title: 'ID', dataIndex: 'id', width: 80 },
          ...operationColumns.filter((col) => col.key !== 'endTime'),
        ];
      case 'system':
        return [
          { key: 'id', title: 'ID', dataIndex: 'id', width: 80 },
          ...operationColumns.filter((col) => !['operator', 'clientIp', 'endTime'].includes(col.key as string)),
        ];
      case 'northbound':
        return [
          { key: 'logName', title: t('log.logName'), dataIndex: 'logName', width: 300 },
          { key: 'clientIp', title: t('log.clientIp'), dataIndex: 'clientIp', width: 200 },
          {
            key: 'type',
            title: t('log.type'),
            dataIndex: 'type',
            width: 150,
            render: (val: string) => northboundTypeTextMap[val] || val,
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
          operationColumns[operationColumns.length - 1],
        ];
      default:
        return operationColumns;
    }
  };

  return (
    <ListPageLayout title={t('log.operationLog')} subtitle={t('log.operationLog')}>
      {/* Tab 页签 + 工具栏 */}
      <Card bordered={false} style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Tabs
            activeKey={activeTab}
            onChange={handleTabChange}
            items={tabItems}
            style={{ marginBottom: -16 }}
          />
          <Space>
            <Button
              icon={<HistoryOutlined />}
              onClick={() => message.info(t('log.viewOldVersion'))}
            >
              {t('log.viewOldVersion')}
            </Button>
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
      {renderSearchForm()}

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
          scroll={{ x: 1400 }}
        />
      </Card>

      {/* 详情抽屉 */}
      <LogDetailDrawer
        open={detailVisible}
        log={selectedLog}
        onClose={() => {
          setDetailVisible(false);
          setSelectedLog(null);
        }}
      />
    </ListPageLayout>
  );
}
