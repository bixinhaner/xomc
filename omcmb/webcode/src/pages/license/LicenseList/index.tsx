import { useState, useMemo } from 'react';
import { Button, Tag, Space, Progress, Descriptions, Card } from 'antd';
import { EyeOutlined, StopOutlined } from '@ant-design/icons';
import SplitPanelLayout from '@/components/Layout/SplitPanelLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useLicenses, useRevokeLicense } from '@/hooks/api/useLicense';
import type { License, LicenseStatus, LicenseType } from '@/mock/data/license';
import { message } from 'antd';
import { useT } from '@/hooks/useT';

const statusColorMap: Record<LicenseStatus, string> = {
  active: 'green',
  expired: 'red',
  pending: 'orange',
  trial: 'blue',
  revoked: 'default',
};

export default function LicenseList() {
  const t = useT();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(15);
  const [selectedLicense, setSelectedLicense] = useState<License | null>(null);

  const { data, isLoading, refetch } = useLicenses({
    status: filters.status as string | undefined,
    licenseType: filters.licenseType as string | undefined,
    page,
    pageSize,
  });

  const revoke = useRevokeLicense();

  const statusLabelMap: Record<LicenseStatus, string> = useMemo(() => ({
    active: t('status.active'),
    expired: t('license.expired'),
    pending: t('status.pending'),
    trial: t('license.trial'),
    revoked: t('license.revoked'),
  }), [t]);

  const licenseTypeLabelMap: Record<LicenseType, string> = useMemo(() => ({
    perpetual: t('license.perpetual'),
    subscription: t('license.subscription'),
    trial: t('license.trialType'),
    evaluation: t('license.evaluation'),
  }), [t]);

  const filterFields: FilterField[] = useMemo(() => [
    { name: 'keyword', label: t('license.idOrName'), type: 'input', placeholder: t('license.idOrNamePlaceholder') },
    {
      name: 'licenseType',
      label: t('table.type'),
      type: 'select',
      options: [
        { label: t('license.perpetual'), value: 'perpetual' },
        { label: t('license.subscription'), value: 'subscription' },
        { label: t('license.trialType'), value: 'trial' },
        { label: t('license.evaluation'), value: 'evaluation' },
      ],
    },
    {
      name: 'status',
      label: t('table.status'),
      type: 'select',
      options: [
        { label: t('status.active'), value: 'active' },
        { label: t('license.expired'), value: 'expired' },
        { label: t('status.pending'), value: 'pending' },
        { label: t('license.trial'), value: 'trial' },
        { label: t('license.revoked'), value: 'revoked' },
      ],
    },
  ], [t]);

  const columns: DataTableColumn<License & Record<string, unknown>>[] = useMemo(() => [
    {
      key: 'licenseCode',
      title: 'License ID',
      dataIndex: 'licenseCode',
      width: 200,
      render: (val) => <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{String(val)}</span>,
    },
    {
      key: 'licenseType',
      title: t('table.type'),
      dataIndex: 'licenseType',
      width: 100,
      render: (val) => <Tag>{licenseTypeLabelMap[val as LicenseType] ?? String(val)}</Tag>,
    },
    {
      key: 'deviceType',
      title: t('device.productType'),
      dataIndex: 'deviceType',
      width: 100,
    },
    {
      key: 'capacity',
      title: t('license.capacityUsage'),
      dataIndex: 'maxDevices',
      width: 150,
      render: (_, record) => {
        const lic = record as License;
        const pct = Math.round((lic.usedDevices / lic.maxDevices) * 100);
        return (
          <div>
            <Progress percent={pct} size="small" status={pct > 90 ? 'exception' : 'normal'} />
            <span style={{ fontSize: 11, color: '#999' }}>{lic.usedDevices}/{lic.maxDevices}</span>
          </div>
        );
      },
    },
    {
      key: 'expiryDate',
      title: t('license.validity'),
      dataIndex: 'expiryDate',
      width: 120,
      render: (val) => {
        if (!val) return <span style={{ color: '#52c41a', fontWeight: 500 }}>{t('license.permanent')}</span>;
        const expiry = new Date(String(val));
        const now = new Date();
        const daysLeft = Math.ceil((expiry.getTime() - now.getTime()) / (1000 * 60 * 60 * 24));
        const color = daysLeft < 0 ? '#ff4d4f' : daysLeft < 30 ? '#faad14' : '#52c41a';
        return <span style={{ color, fontWeight: daysLeft < 30 ? 600 : 400 }}>{expiry.toLocaleDateString('zh-CN')}</span>;
      },
    },
    {
      key: 'status',
      title: t('table.status'),
      dataIndex: 'status',
      width: 90,
      render: (val) => {
        const s = val as LicenseStatus;
        return <Tag color={statusColorMap[s]}>{statusLabelMap[s]}</Tag>;
      },
    },
    {
      key: 'actions',
      title: t('table.operation'),
      dataIndex: 'id',
      width: 120,
      fixed: 'right',
      render: (_, record) => {
        const lic = record as License;
        return (
          <Space size="small">
            <Button type="link" size="small" icon={<EyeOutlined />}
              onClick={() => setSelectedLicense(lic)}>
              {t('common.detail')}
            </Button>
            {lic.status === 'active' && (
              <Button type="link" size="small" danger icon={<StopOutlined />}
                onClick={() => revoke.mutate(lic.id, { onSuccess: () => void message.success(t('license.revokeSuccess')) })}>
                {t('license.revoke')}
              </Button>
            )}
          </Space>
        );
      },
    },
  ], [t, licenseTypeLabelMap, statusLabelMap, revoke]);

  const upperPanel = (
    <div style={{ padding: '8px 12px' }}>
      <FilterBar
        filterId="license-list-filter"
        fields={filterFields}
        onSearch={(vals) => { setFilters(vals); setPage(1); }}
        onReset={() => { setFilters({}); setPage(1); }}
      />
      <DataTable
        tableId="license-list"
        columns={columns}
        dataSource={(data?.items ?? []) as (License & Record<string, unknown>)[]}
        loading={isLoading}
        rowKey="id"
        total={data?.total ?? 0}
        pageSize={pageSize}
        currentPage={page}
        onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
        onRefresh={() => void refetch()}
        scroll={{ x: 1000 }}
      />
    </div>
  );

  const lowerPanel = selectedLicense ? (
    <div style={{ padding: 16, overflow: 'auto', height: '100%' }}>
      <Card title={`${t('license.detail')} — ${selectedLicense.licenseName}`} size="small">
        <Descriptions bordered column={2} size="small">
          <Descriptions.Item label={t('license.licenseName')} span={2}>{selectedLicense.licenseName}</Descriptions.Item>
          <Descriptions.Item label="License ID">
            <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{selectedLicense.licenseCode}</span>
          </Descriptions.Item>
          <Descriptions.Item label={t('license.productName')}>{selectedLicense.productName}</Descriptions.Item>
          <Descriptions.Item label={t('license.licenseType')}>
            <Tag>{licenseTypeLabelMap[selectedLicense.licenseType]}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label={t('table.status')}>
            <Tag color={statusColorMap[selectedLicense.status]}>{statusLabelMap[selectedLicense.status]}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label={t('device.productType')}>{selectedLicense.deviceType}</Descriptions.Item>
          <Descriptions.Item label={t('table.region')}>{selectedLicense.region}</Descriptions.Item>
          <Descriptions.Item label={t('license.capacity')}>{selectedLicense.maxDevices}</Descriptions.Item>
          <Descriptions.Item label={t('license.used')}>
            <Progress
              percent={Math.round((selectedLicense.usedDevices / selectedLicense.maxDevices) * 100)}
              size="small"
              format={() => `${selectedLicense.usedDevices}/${selectedLicense.maxDevices}`}
            />
          </Descriptions.Item>
          <Descriptions.Item label={t('license.licensor')}>{selectedLicense.licensor}</Descriptions.Item>
          <Descriptions.Item label={t('license.issueDate')}>{new Date(selectedLicense.issueDate).toLocaleDateString('zh-CN')}</Descriptions.Item>
          <Descriptions.Item label={t('license.expiryDate')}>
            {selectedLicense.expiryDate ? new Date(selectedLicense.expiryDate).toLocaleDateString('zh-CN') : t('license.permanentValid')}
          </Descriptions.Item>
          <Descriptions.Item label={t('license.features')} span={2}>
            {selectedLicense.features.map((f, i) => <Tag key={i} color="blue" style={{ marginBottom: 4 }}>{f}</Tag>)}
          </Descriptions.Item>
          {selectedLicense.notes && (
            <Descriptions.Item label={t('license.notes')} span={2}>{selectedLicense.notes}</Descriptions.Item>
          )}
        </Descriptions>
      </Card>
    </div>
  ) : (
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%', color: '#999' }}>
      {t('license.selectToViewDetail')}
    </div>
  );

  return (
    <SplitPanelLayout
      upper={upperPanel}
      lower={lowerPanel}
      upperTitle={t('nav.license.list')}
      lowerTitle={t('license.detail')}
      defaultSplitRatio={0.65}
    />
  );
}
