import { useState, useMemo, useCallback } from 'react';
import { Button, Modal, Space, Table, Tag, message } from 'antd';
import { EyeOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';

import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import { useT } from '@/hooks/useT';

import type { MMLScript, MMLScriptStatus, MMLScriptType } from '@core/types/mml';
import {
  useMMLScripts,
  useDeleteMMLScripts,
} from '@core/hooks/api/useMML';
import { useDictionary } from '@core/hooks/api/useSystem';

// -------------------------------------------------------------------------
// Display mappings — mml_scripts columns
// -------------------------------------------------------------------------

const SCRIPT_TYPE_TAGS: Record<MMLScriptType, { color: string; label: string }> = {
  manual: { color: 'geekblue', label: '手动' },
  batch: { color: 'purple', label: '批量' },
};

const SCRIPT_STATUS_TAGS: Record<MMLScriptStatus, { color: string; label: string }> = {
  active: { color: 'processing', label: '进行中' },
  archived: { color: 'default', label: '已归档' },
};

// Shape of a single device execution row displayed in the 查看 modal.
// Parsed tolerantly from the loosely-typed `MMLScript.result` JSONB blob
// that the backend persists (mml_scripts.result).
interface ScriptExecutionRow {
  deviceSn: string;
  status: 'success' | 'failed' | 'running' | 'pending' | string;
  executionTimeMs?: number;
  output?: string;
}

function formatTime(iso?: string | null): string {
  if (!iso) return '-';
  const d = dayjs(iso);
  return d.isValid() ? d.format('YYYY-MM-DD HH:mm:ss') : '-';
}

// Parse an arbitrary JSON-ish result payload into a list of per-device rows
// for the 查看 modal. Accepts:
//   - { items: [{device_sn, status, execution_time, output}, ...] }
//   - [{device_sn, ...}, ...]
// Missing fields degrade gracefully.
function parseExecutionRows(raw: Record<string, unknown> | undefined): ScriptExecutionRow[] {
  if (!raw) return [];
  const candidate =
    Array.isArray(raw) ? raw :
    Array.isArray((raw as Record<string, unknown>).items)
      ? ((raw as Record<string, unknown>).items as unknown[])
      : [];
  return candidate
    .filter((v): v is Record<string, unknown> => v !== null && typeof v === 'object')
    .map((entry) => {
      const deviceSn = (entry.device_sn as string) || (entry.deviceSn as string) || '';
      const status =
        (entry.status as string) ||
        (entry.success === true ? 'success' : entry.success === false ? 'failed' : '');
      const execMs =
        (entry.execution_time as number) ??
        (entry.executionTime as number) ??
        (entry.duration_ms as number);
      const output =
        (entry.output as string) ||
        (entry.raw_output as string) ||
        (entry.rawOutput as string) ||
        '';
      return {
        deviceSn,
        status,
        executionTimeMs: typeof execMs === 'number' ? execMs : undefined,
        output,
      };
    });
}

export default function ScriptTask() {
  const t = useT();

  // ---- paging / filter state ------------------------------------------------
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(50);
  const [filters, setFilters] = useState<{
    scriptName?: string;
    deviceType?: string;
    creator?: string;
  }>({});

  const { data: productTypeDict } = useDictionary('product_type');
  const productTypeOptions = useMemo(() => {
    const details = productTypeDict?.sysDictionaryDetails;
    return details?.length ? details.map((d) => ({ label: d.label, value: d.value })) : [];
  }, [productTypeDict]);

  const { data, isLoading, refetch } = useMMLScripts({
    page,
    pageSize,
    search: filters.scriptName,
    deviceType: filters.deviceType && filters.deviceType !== 'all' ? filters.deviceType : undefined,
    creator: filters.creator,
  });
  const deleteScriptsMutation = useDeleteMMLScripts();

  const scripts = useMemo(() => data?.items ?? [], [data]);

  // ---- filter fields --------------------------------------------------------
  const filterFields: FilterField[] = useMemo(() => [
    {
      name: 'scriptName',
      label: t('mml.scriptName'),
      type: 'input',
      placeholder: t('mml.inputScriptName'),
    },
    {
      name: 'deviceType',
      label: t('mml.deviceType'),
      type: 'select',
      placeholder: t('mml.deviceType'),
      options: [{ label: t('common.all'), value: 'all' }, ...productTypeOptions],
    },
    {
      name: 'creator',
      label: t('mml.creator'),
      type: 'input',
      placeholder: t('mml.creator'),
    },
  ], [t, productTypeOptions]);

  const handleSearch = useCallback((values: Record<string, unknown>) => {
    setPage(1);
    setFilters({
      scriptName: typeof values.scriptName === 'string' ? values.scriptName.trim() : undefined,
      deviceType: typeof values.deviceType === 'string' ? values.deviceType : undefined,
      creator: typeof values.creator === 'string' ? values.creator.trim() : undefined,
    });
  }, []);

  const handleReset = useCallback(() => {
    setFilters({});
    setPage(1);
  }, []);

  // ---- 查看 modal state -----------------------------------------------------
  const [viewing, setViewing] = useState<MMLScript | null>(null);
  const executionRows = useMemo(() => parseExecutionRows(viewing?.result), [viewing]);

  const handleDelete = useCallback((id: string) => {
    deleteScriptsMutation.mutate([id], {
      onSuccess: () => void message.success(t('common.deleteSuccess')),
      onError: (err) => void message.error(err instanceof Error ? err.message : 'Unknown'),
    });
  }, [deleteScriptsMutation, t]);

  const columns: DataTableColumn<MMLScript>[] = useMemo(() => [
    {
      key: 'operation',
      title: t('table.operation'),
      dataIndex: 'id',
      width: 140,
      fixed: 'right',
      render: (_, record) => (
        <Space size={4}>
          <Button
            type="link"
            size="small"
            icon={<EyeOutlined />}
            onClick={() => setViewing(record)}
          >
            {t('common.view')}
          </Button>
          <Button
            type="link"
            size="small"
            danger
            onClick={() => handleDelete(record.id)}
          >
            {t('common.delete')}
          </Button>
        </Space>
      ),
    },
    { key: 'scriptName', title: t('mml.scriptName'), dataIndex: 'scriptName', ellipsis: true },
    {
      key: 'description',
      title: t('mml.description'),
      dataIndex: 'description',
      ellipsis: true,
      render: (v: string) => v || '-',
    },
    {
      key: 'type',
      title: t('mml.type'),
      dataIndex: 'type',
      width: 90,
      render: (v: MMLScriptType) => {
        const tag = SCRIPT_TYPE_TAGS[v];
        return tag ? <Tag color={tag.color}>{tag.label}</Tag> : <Tag>{v || '-'}</Tag>;
      },
    },
    {
      key: 'status',
      title: t('mml.status'),
      dataIndex: 'status',
      width: 100,
      render: (v: MMLScriptStatus) => {
        const tag = SCRIPT_STATUS_TAGS[v];
        return tag ? <Tag color={tag.color}>{tag.label}</Tag> : <Tag>{v || '-'}</Tag>;
      },
    },
    {
      key: 'progress',
      title: t('mml.progress'),
      dataIndex: 'progress',
      width: 90,
      render: (v: number) => `${(v ?? 0).toFixed(0)}%`,
    },
    {
      key: 'deviceType',
      title: t('mml.deviceType'),
      dataIndex: 'deviceType',
      width: 110,
      render: (v: string) => v || '-',
    },
    { key: 'creator', title: t('mml.creator'), dataIndex: 'creator', width: 100 },
    {
      key: 'startTime',
      title: t('mml.startTime'),
      dataIndex: 'startTime',
      width: 160,
      render: (v?: string) => formatTime(v),
    },
    {
      key: 'endTime',
      title: t('mml.endTime'),
      dataIndex: 'endTime',
      width: 160,
      render: (v?: string) => formatTime(v),
    },
    {
      key: 'updateTime',
      title: t('mml.updateTime'),
      dataIndex: 'updateTime',
      width: 160,
      render: (v: string) => formatTime(v),
    },
  ], [t, handleDelete]);

  // ---- 查看 modal columns（对应 image-5.png：设备SN / 状态 / 执行时间 / 输出）
  const resultColumns = useMemo(() => [
    { key: 'deviceSn', title: t('mml.deviceSn'), dataIndex: 'deviceSn', width: 160 },
    {
      key: 'status',
      title: t('mml.status'),
      dataIndex: 'status',
      width: 100,
      render: (v: string) => {
        if (v === 'success') return <Tag color="success">{t('status.success')}</Tag>;
        if (v === 'failed') return <Tag color="error">{t('status.failed')}</Tag>;
        return <Tag>{v || '-'}</Tag>;
      },
    },
    {
      key: 'executionTimeMs',
      title: t('mml.executionTime'),
      dataIndex: 'executionTimeMs',
      width: 120,
      align: 'right' as const,
      render: (v?: number) => (typeof v === 'number' ? v : '-'),
    },
    {
      key: 'output',
      title: t('mml.output'),
      dataIndex: 'output',
      ellipsis: true,
      render: (v: string) => v || '-',
    },
  ], [t]);

  return (
    <ListPageLayout title={t('nav.mml.script')}>
      <FilterBar
        filterId="mml-script-task"
        fields={filterFields}
        onSearch={handleSearch}
        onReset={handleReset}
      />
      <DataTable<MMLScript>
        tableId="mml-script-task"
        columns={columns}
        dataSource={scripts}
        loading={isLoading}
        rowKey="id"
        total={data?.total ?? 0}
        currentPage={page}
        pageSize={pageSize}
        onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
        onRefresh={() => void refetch()}
        scroll={{ x: 1400 }}
      />

      <Modal
        title={viewing ? t('mml.executionResult', { name: viewing.scriptName }) : t('common.view')}
        open={Boolean(viewing)}
        onCancel={() => setViewing(null)}
        footer={null}
        width={820}
        destroyOnClose
      >
        {viewing && (
          <>
            {executionRows.length > 0 ? (
              <Table
                size="small"
                rowKey={(row) => row.deviceSn || Math.random().toString(36).slice(2)}
                dataSource={executionRows}
                columns={resultColumns}
                pagination={false}
                scroll={{ y: 360 }}
              />
            ) : (
              <div style={{ padding: 24, textAlign: 'center', color: '#999' }}>
                {t('mml.noExecutionResult')}
              </div>
            )}
          </>
        )}
      </Modal>
    </ListPageLayout>
  );
}
