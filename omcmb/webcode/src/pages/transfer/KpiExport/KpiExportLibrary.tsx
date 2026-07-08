/**
 * KpiExportLibrary — 文件管理 ▸「KPI 导出」Tab（第 6 个）。
 *
 * 设计：~/Documents/notes/PM功能设计/kpi-export-design-20260604.md §3.3 / §6.2
 * 列：文件名 | 来源 | 行数 | 大小 | 生成时间 | 下载。只列已成功且文件就绪的导出（后端 /pm/exports/files）。
 * 复用现成的列表/下载交互（同配置快照库）。
 */
import { useMemo, useState } from 'react';
import { Button, Card, Input, Select, Space, Tag, message } from 'antd';
import { DownloadOutlined, ReloadOutlined } from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useT } from '@/hooks/useT';
import { useKpiExportFiles } from '@core/hooks/api/useKpiExport';
import { kpiExportApi } from '@core/services/api/kpiExportApi';
import type { KpiExportSource, KpiExportTask } from '@core/types/kpiExport';
import { SOURCE_LABEL_KEY, formatBytes, deriveFileName } from './shared';
import { formatSystemTime } from '@core/utils/systemTime';

const LIST_LIMIT = 500;

export default function KpiExportLibrary() {
  const t = useT();
  const [nameFilter, setNameFilter] = useState('');
  const [sourceFilter, setSourceFilter] = useState<KpiExportSource | ''>('');

  const { data, isLoading, refetch } = useKpiExportFiles({
    sourceType: sourceFilter || undefined,
    limit: LIST_LIMIT,
  });

  const rows = useMemo(() => {
    const list = data ?? [];
    const kw = nameFilter.trim().toLowerCase();
    return kw ? list.filter((x) => x.taskName.toLowerCase().includes(kw)) : list;
  }, [data, nameFilter]);

  async function handleDownload(record: KpiExportTask) {
    try {
      await kpiExportApi.download(record.id, deriveFileName(record));
    } catch (e) {
      message.error(
        t('kpiExport.msg.downloadFailed', { reason: (e as Error).message ?? '' }),
      );
    }
  }

  const columns: DataTableColumn<KpiExportTask>[] = [
    {
      key: 'fileName',
      title: t('kpiExport.col.fileName'),
      width: 320,
      mono: true,
      render: (_, r) => deriveFileName(r),
    },
    {
      key: 'source',
      title: t('kpiExport.col.source'),
      dataIndex: 'sourceType',
      width: 120,
      render: (v) => <Tag>{t(SOURCE_LABEL_KEY[v as KpiExportSource])}</Tag>,
    },
    {
      key: 'rowCount',
      title: t('kpiExport.col.rowCount'),
      dataIndex: 'rowCount',
      width: 110,
      render: (v) => (typeof v === 'number' ? v.toLocaleString() : '—'),
    },
    {
      key: 'fileSize',
      title: t('kpiExport.col.fileSize'),
      dataIndex: 'fileSize',
      width: 110,
      render: (v) => formatBytes(v as number),
    },
    {
      key: 'generatedAt',
      title: t('kpiExport.col.generatedAt'),
      dataIndex: 'finishedAt',
      width: 180,
      render: (v) => (v ? formatSystemTime(v as string, { format: 'YYYY-MM-DD HH:mm:ss', placeholder: '-' }) : '—'),
    },
    {
      key: 'actions',
      title: t('kpiExport.col.actions'),
      width: 120,
      fixed: 'right',
      render: (_, record) => (
        <Button
          type="link"
          size="small"
          icon={<DownloadOutlined />}
          onClick={() => {
            void handleDownload(record);
          }}
        >
          {t('kpiExport.action.download')}
        </Button>
      ),
    },
  ];

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16, flex: 1, minHeight: 0 }}>
      <Card size="small">
        <Space wrap>
          <Input
            placeholder={t('kpiExport.filter.taskName')}
            allowClear
            value={nameFilter}
            onChange={(e) => setNameFilter(e.target.value)}
            style={{ width: 220 }}
          />
          <Select<KpiExportSource | ''>
            placeholder={t('kpiExport.filter.source')}
            allowClear
            value={sourceFilter || undefined}
            onChange={(v) => setSourceFilter(v ?? '')}
            style={{ width: 160 }}
            options={[
              { value: 'dashboard', label: t('kpiExport.source.dashboard') },
              { value: 'kpi_query', label: t('kpiExport.source.kpiQuery') },
              { value: 'adhoc', label: t('kpiExport.source.adhoc') },
            ]}
          />
          <Button icon={<ReloadOutlined />} onClick={() => refetch()}>
            {t('kpiExport.action.refresh')}
          </Button>
        </Space>
      </Card>
      <DataTable<KpiExportTask>
        tableId="kpi-export-library"
        columns={columns}
        dataSource={rows}
        loading={isLoading}
        rowKey={(r) => r.id}
      />
    </div>
  );
}
