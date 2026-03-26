import { useState, useMemo } from 'react';
import { Tag, Button, Tooltip, Modal } from 'antd';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import JSONViewer from '@/components/JSONViewer';
import { useOperationLogs } from '@/hooks/api/useLogs';
import type { OperationLog, OperationType, OperationResult } from '@/types/system';
import { useT } from '@/hooks/useT';

const opTypeColorMap: Record<OperationType, string> = {
  create: 'blue',
  update: 'cyan',
  delete: 'red',
  query: 'default',
  export: 'green',
  import: 'orange',
  login: 'geekblue',
  logout: 'default',
  execute: 'purple',
  deploy: 'magenta',
  approve: 'gold',
};

const opTypeLabelMap: Record<OperationType, string> = {
  create: '创建',
  update: '更新',
  delete: '删除',
  query: '查询',
  export: '导出',
  import: '导入',
  login: '登录',
  logout: '退出',
  execute: '执行',
  deploy: '部署',
  approve: '审批',
};

const resultColorMap: Record<OperationResult, string> = {
  success: 'green',
  failure: 'red',
  partial: 'orange',
};

const resultLabelMap: Record<OperationResult, string> = {
  success: '成功',
  failure: '失败',
  partial: '部分成功',
};

function truncate(str: string, maxLen = 50): string {
  if (str.length <= maxLen) return str;
  return str.substring(0, maxLen) + '...';
}

export default function OperationLogPage() {
  const t = useT();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [jsonVisible, setJsonVisible] = useState(false);
  const [jsonData, setJsonData] = useState<{ title: string; data: Record<string, unknown> } | null>(null);

  const { data, isLoading, refetch } = useOperationLogs({
    operator: filters.operator as string | undefined,
    module: filters.module as string | undefined,
    operationType: filters.operationType as OperationType | undefined,
    result: filters.result as OperationResult | undefined,
    page,
    pageSize,
  });

  const filterFields: FilterField[] = useMemo(() => [
    { name: 'operator', label: t('table.operator'), type: 'input', placeholder: t('table.operator') },
    {
      name: 'module',
      label: t('table.type'),
      type: 'select',
      options: [
        { label: '设备管理', value: '设备管理' },
        { label: '软件版本', value: '软件版本' },
        { label: '告警管理', value: '告警管理' },
        { label: '性能管理', value: '性能管理' },
        { label: '文件管理', value: '文件管理' },
        { label: '用户管理', value: '用户管理' },
        { label: '系统配置', value: '系统配置' },
      ],
    },
    {
      name: 'operationType',
      label: t('table.type'),
      type: 'select',
      options: [
        { label: '创建', value: 'create' },
        { label: '更新', value: 'update' },
        { label: '删除', value: 'delete' },
        { label: '查询', value: 'query' },
        { label: '导出', value: 'export' },
        { label: '导入', value: 'import' },
        { label: '执行', value: 'execute' },
        { label: '登录', value: 'login' },
        { label: '退出', value: 'logout' },
      ],
    },
    { name: 'target', label: t('table.name'), type: 'input', placeholder: t('table.name') },
    {
      name: 'result',
      label: t('table.result'),
      type: 'select',
      options: [
        { label: t('status.success'), value: 'success' },
        { label: t('status.failed'), value: 'failure' },
        { label: '部分成功', value: 'partial' },
      ],
    },
  ], [t]);

  const columns: DataTableColumn<OperationLog & Record<string, unknown>>[] = useMemo(() => [
    {
      key: 'operationTime',
      title: t('table.time'),
      dataIndex: 'operationTime',
      width: 180,
      render: (val) => new Date(String(val)).toLocaleString('zh-CN'),
    },
    { key: 'operator', title: t('table.operator'), dataIndex: 'operator', width: 100 },
    {
      key: 'clientIp',
      title: t('table.ip'),
      dataIndex: 'clientIp',
      width: 140,
      render: (val) => <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{String(val)}</span>,
    },
    {
      key: 'target',
      title: t('table.name'),
      dataIndex: 'target',
      width: 150,
      ellipsis: true,
      render: (val) => (
        <Tooltip title={String(val)}>
          <span>{truncate(String(val), 20)}</span>
        </Tooltip>
      ),
    },
    { key: 'module', title: t('table.type'), dataIndex: 'module', width: 110 },
    {
      key: 'operationType',
      title: t('table.type'),
      dataIndex: 'operationType',
      width: 90,
      render: (val) => {
        const tp = val as OperationType;
        return <Tag color={opTypeColorMap[tp] ?? 'default'}>{opTypeLabelMap[tp] ?? String(val)}</Tag>;
      },
    },
    {
      key: 'content',
      title: t('table.description'),
      dataIndex: 'content',
      ellipsis: true,
      render: (val) => (
        <Tooltip title={String(val)}>
          <span style={{ fontSize: 12 }}>{truncate(String(val), 40)}</span>
        </Tooltip>
      ),
    },
    {
      key: 'result',
      title: t('table.result'),
      dataIndex: 'result',
      width: 90,
      render: (val) => {
        const r = val as OperationResult;
        return <Tag color={resultColorMap[r] ?? 'default'}>{resultLabelMap[r] ?? String(val)}</Tag>;
      },
    },
    {
      key: 'message',
      title: t('common.view'),
      dataIndex: 'message',
      width: 90,
      fixed: 'right',
      render: (val, record) => {
        const log = record as OperationLog;
        return (
          <Button
            type="link"
            size="small"
            onClick={() => {
              setJsonData({
                title: `${log.operator} - ${opTypeLabelMap[log.operationType] ?? log.operationType}`,
                data: {
                  operator: log.operator,
                  operationType: log.operationType,
                  module: log.module,
                  target: log.target,
                  content: log.content,
                  result: log.result,
                  message: String(val),
                  clientIp: log.clientIp,
                  operationTime: log.operationTime,
                },
              });
              setJsonVisible(true);
            }}
          >
            {t('common.view')}
          </Button>
        );
      },
    },
  ], [t]);

  return (
    <ListPageLayout title={t('nav.log.operation')} subtitle={t('nav.log.operation')}>
      <FilterBar
        filterId="operation-log-filter"
        fields={filterFields}
        onSearch={(vals) => { setFilters(vals); setPage(1); }}
        onReset={() => { setFilters({}); setPage(1); }}
      />
      <DataTable
        tableId="operation-log-list"
        columns={columns}
        dataSource={(data?.items ?? []) as (OperationLog & Record<string, unknown>)[]}
        loading={isLoading}
        rowKey="id"
        total={data?.total ?? 0}
        pageSize={pageSize}
        currentPage={page}
        onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
        onRefresh={() => void refetch()}
        onExport={(format) => void console.log('导出', format)}
        scroll={{ x: 1300 }}
      />

      <Modal
        title={jsonData?.title}
        open={jsonVisible}
        onCancel={() => setJsonVisible(false)}
        footer={null}
        width={600}
      >
        {jsonData && <JSONViewer data={jsonData.data} />}
      </Modal>
    </ListPageLayout>
  );
}
