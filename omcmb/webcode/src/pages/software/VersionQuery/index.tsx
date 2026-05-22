import { useState, useMemo } from 'react';
import { Button, Tag, Space, Tooltip, Modal, Descriptions } from 'antd';
import { DownloadOutlined, EyeOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useSoftwareVersions } from '@core/hooks/api/useSoftware';
import type { SoftwareVersion, VersionStatus } from '@core/mock/data/software';
import { useT } from '@/hooks/useT';

const statusColorMap: Record<VersionStatus, string> = {
  current: 'green',
  deprecated: 'default',
  beta: 'blue',
  archived: 'gray',
};

const statusLabelKeyMap: Record<VersionStatus, string> = {
  current: 'status.enabled',
  deprecated: 'status.disabled',
  beta: 'status.pending',
  archived: 'status.offline',
};

function formatFileSize(bytes: number): string {
  if (bytes >= 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB`;
  if (bytes >= 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(2)} MB`;
  if (bytes >= 1024) return `${(bytes / 1024).toFixed(2)} KB`;
  return `${bytes} B`;
}

export default function VersionQuery() {
  const t = useT();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [detailVisible, setDetailVisible] = useState(false);
  const [selectedVersion, setSelectedVersion] = useState<SoftwareVersion | null>(null);

  const filterFields: FilterField[] = useMemo(() => [
    { name: 'versionCode', label: t('table.version'), type: 'input', placeholder: t('common.placeholder') },
    {
      name: 'deviceType',
      label: t('device.productClass'),
      type: 'select',
      options: [
        { label: 'eNB', value: 'eNB' },
        { label: 'gNB', value: 'gNB' },
        { label: 'RRU', value: 'RRU' },
        { label: 'AAU', value: 'AAU' },
      ],
    },
    {
      name: 'vendor',
      label: t('device.vendor'),
      type: 'select',
      options: [
        { label: '华为', value: '华为' },
        { label: '中兴', value: '中兴' },
        { label: '爱立信', value: '爱立信' },
        { label: '诺基亚', value: '诺基亚' },
      ],
    },
    {
      name: 'status',
      label: t('table.status'),
      type: 'select',
      options: [
        { label: t('status.enabled'), value: 'current' },
        { label: t('status.pending'), value: 'beta' },
        { label: t('status.disabled'), value: 'deprecated' },
        { label: t('status.offline'), value: 'archived' },
      ],
    },
  ], [t]);

  const { data, isLoading, refetch } = useSoftwareVersions({
    ...filters,
    page,
    pageSize,
  });

  const columns: DataTableColumn<SoftwareVersion & Record<string, unknown>>[] = useMemo(() => [
    {
      key: 'versionCode',
      title: t('table.version'),
      dataIndex: 'versionCode',
      width: 200,
      mono: true,
      render: (val) => <span style={{ fontFamily: 'monospace', fontSize: 13 }}>{String(val)}</span>,
    },
    { key: 'deviceType', title: t('device.productClass'), dataIndex: 'deviceType', width: 100 },
    { key: 'vendor', title: t('device.vendor'), dataIndex: 'vendor', width: 100 },
    {
      key: 'releaseNotes',
      title: t('table.description'),
      dataIndex: 'releaseNotes',
      ellipsis: true,
      render: (val) => (
        <Tooltip title={String(val)}>
          <span style={{ maxWidth: 200, display: 'inline-block', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {String(val)}
          </span>
        </Tooltip>
      ),
    },
    {
      key: 'releaseDate',
      title: t('table.createTime'),
      dataIndex: 'releaseDate',
      width: 120,
      render: (val) => new Date(String(val)).toLocaleDateString('zh-CN'),
    },
    {
      key: 'status',
      title: t('table.status'),
      dataIndex: 'status',
      width: 100,
      render: (val) => {
        const status = val as VersionStatus;
        return <Tag color={statusColorMap[status]}>{t(statusLabelKeyMap[status])}</Tag>;
      },
    },
    {
      key: 'fileSize',
      title: t('table.description'),
      dataIndex: 'fileSize',
      width: 110,
      render: (val) => formatFileSize(Number(val)),
    },
    {
      key: 'actions',
      title: t('table.operation'),
      dataIndex: 'id',
      width: 120,
      fixed: 'right',
      render: (_, record) => (
        <Space size={4}>
          <Button
            type="link"
            size="small"
            icon={<DownloadOutlined />}
            onClick={() => window.open((record as SoftwareVersion).downloadUrl)}
          >
            {t('common.download')}
          </Button>
          <Button
            type="link"
            size="small"
            icon={<EyeOutlined />}
            onClick={() => {
              setSelectedVersion(record as SoftwareVersion);
              setDetailVisible(true);
            }}
          >
            {t('common.detail')}
          </Button>
        </Space>
      ),
    },
  ], [t]);

  return (
    <ListPageLayout title={t('nav.software.version')}>
      <FilterBar
        filterId="software-version-query"
        fields={filterFields}
        onSearch={(vals) => { setFilters(vals); setPage(1); }}
        onReset={() => { setFilters({}); setPage(1); }}
      />
      <DataTable
        tableId="software-version-list"
        columns={columns}
        dataSource={(data?.items ?? []) as (SoftwareVersion & Record<string, unknown>)[]}
        loading={isLoading}
        rowKey="id"
        total={data?.total ?? 0}
        pageSize={pageSize}
        currentPage={page}
        onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
        onRefresh={() => void refetch()}
        scroll={{ x: 1000 }}
      />

      <Modal
        title={t('common.detail')}
        open={detailVisible}
        onCancel={() => setDetailVisible(false)}
        footer={null}
        width={700}
      >
        {selectedVersion && (
          <Descriptions bordered column={2} size="small">
            <Descriptions.Item label={t('table.version')} span={2}>
              <span style={{ fontFamily: 'monospace' }}>{selectedVersion.versionCode}</span>
            </Descriptions.Item>
            <Descriptions.Item label={t('table.name')}>{selectedVersion.versionName}</Descriptions.Item>
            <Descriptions.Item label={t('device.productClass')}>{selectedVersion.deviceType}</Descriptions.Item>
            <Descriptions.Item label={t('device.vendor')}>{selectedVersion.vendor}</Descriptions.Item>
            <Descriptions.Item label={t('table.status')}>
              <Tag color={statusColorMap[selectedVersion.status]}>{t(statusLabelKeyMap[selectedVersion.status])}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label={t('table.createTime')}>
              {new Date(selectedVersion.releaseDate).toLocaleDateString('zh-CN')}
            </Descriptions.Item>
            <Descriptions.Item label={t('table.description')}>{formatFileSize(selectedVersion.fileSize)}</Descriptions.Item>
            <Descriptions.Item label={t('table.version')}>{selectedVersion.minHardwareVersion}</Descriptions.Item>
            <Descriptions.Item label={t('table.description')} span={2}>
              <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{selectedVersion.checksum}</span>
            </Descriptions.Item>
            <Descriptions.Item label={t('table.description')} span={2}>{selectedVersion.releaseNotes}</Descriptions.Item>
            <Descriptions.Item label={t('table.description')} span={2}>
              {selectedVersion.features.map((f, i) => <Tag key={i} color="blue" style={{ marginBottom: 4 }}>{f}</Tag>)}
            </Descriptions.Item>
            <Descriptions.Item label={t('table.description')} span={2}>
              {selectedVersion.bugFixes.map((f, i) => <Tag key={i} color="green" style={{ marginBottom: 4 }}>{f}</Tag>)}
            </Descriptions.Item>
            {selectedVersion.known_issues && selectedVersion.known_issues.length > 0 && (
              <Descriptions.Item label={t('table.description')} span={2}>
                {selectedVersion.known_issues.map((f, i) => <Tag key={i} color="orange" style={{ marginBottom: 4 }}>{f}</Tag>)}
              </Descriptions.Item>
            )}
          </Descriptions>
        )}
      </Modal>
    </ListPageLayout>
  );
}
