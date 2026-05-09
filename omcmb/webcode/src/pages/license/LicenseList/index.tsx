import { useState, useMemo } from 'react';
import { Button, Tag, Space, Progress, Descriptions, Card, Row, Col, Statistic, Tabs, List, Empty, Tooltip } from 'antd';
import { EyeOutlined, StopOutlined, AppstoreOutlined, ThunderboltOutlined, ClockCircleOutlined, AlertOutlined, ArrowRightOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import SplitPanelLayout from '@/components/Layout/SplitPanelLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import {
  useLicenses,
  useLicenseSummary,
  useRevokeLicense,
  useLicenseLogsByLicense,
} from '@core/hooks/api/useLicense';
import type { License, LicenseStatus, LicenseType } from '@core/mock/data/license';
import type { LicenseLog, LicenseLogType, LicenseLogResult } from '@core/services/api/licenseApi';
import { message } from 'antd';
import { useT } from '@/hooks/useT';

const statusColorMap: Record<LicenseStatus, string> = {
  active: 'green',
  expired: 'red',
  pending: 'orange',
  trial: 'blue',
  revoked: 'default',
};

// 与 LicenseLogs 页面保持一致的颜色映射
const logTypeColorMap: Record<LicenseLogType, string> = {
  import: 'blue',
  activate: 'green',
  revoke: 'orange',
  query_detail: 'default',
  enforcement_capacity: 'volcano',
  enforcement_expiry: 'volcano',
  capacity_alert: 'gold',
  expiry_alert: 'gold',
  auto_expire: 'purple',
};
const resultColorMap: Record<LicenseLogResult, string> = {
  success: 'green',
  failed: 'red',
  denied: 'orange',
  warning: 'gold',
};

export default function LicenseList() {
  const t = useT();
  const navigate = useNavigate();
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
  const { data: summary } = useLicenseSummary();

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

  const logTypeLabelMap: Record<LicenseLogType, string> = useMemo(() => ({
    import: t('license.logs.type.import'),
    activate: t('license.logs.type.activate'),
    revoke: t('license.logs.type.revoke'),
    query_detail: t('license.logs.type.queryDetail'),
    enforcement_capacity: t('license.logs.type.enforcementCapacity'),
    enforcement_expiry: t('license.logs.type.enforcementExpiry'),
    capacity_alert: t('license.logs.type.capacityAlert'),
    expiry_alert: t('license.logs.type.expiryAlert'),
    auto_expire: t('license.logs.type.autoExpire'),
  }), [t]);

  const resultLabelMap: Record<LicenseLogResult, string> = useMemo(() => ({
    success: t('license.logs.result.success'),
    failed: t('license.logs.result.failed'),
    denied: t('license.logs.result.denied'),
    warning: t('license.logs.result.warning'),
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
    { key: 'deviceType', title: t('device.productType'), dataIndex: 'deviceType', width: 100 },
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
          <Space size={4}>
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

  // T-0100-P2: 顶部 4 张统计卡片（active 数 / 容量使用率 / 30 天内过期 / 近 7 天 enforcement 命中）
  const summaryCards = (
    <Row gutter={12} style={{ marginBottom: 12 }}>
      <Col span={6}>
        <Card size="small" hoverable onClick={() => { setFilters({ status: 'active' }); setPage(1); }}>
          <Statistic
            title={t('license.summary.activeCount')}
            value={summary?.active ?? 0}
            prefix={<AppstoreOutlined style={{ color: '#52c41a' }} />}
            valueStyle={{ color: '#52c41a' }}
          />
        </Card>
      </Col>
      <Col span={6}>
        <Card size="small">
          <Statistic
            title={t('license.summary.totalCapacityUsage')}
            value={summary?.total ? `${summary.active}/${summary.total}` : '0'}
            prefix={<ThunderboltOutlined style={{ color: '#1677ff' }} />}
            suffix={summary?.total ? `(${Math.round((summary.active / Math.max(summary.total, 1)) * 100)}%)` : ''}
          />
        </Card>
      </Col>
      <Col span={6}>
        <Card size="small" hoverable onClick={() => { setFilters({ status: 'active' }); setPage(1); }}>
          <Statistic
            title={t('license.summary.expiringSoon')}
            value={summary?.expiringSoon ?? 0}
            prefix={<ClockCircleOutlined style={{ color: '#faad14' }} />}
            valueStyle={{ color: (summary?.expiringSoon ?? 0) > 0 ? '#faad14' : undefined }}
          />
        </Card>
      </Col>
      <Col span={6}>
        <Card size="small" hoverable onClick={() => navigate('/license/logs?result=denied')}>
          <Statistic
            title={t('license.summary.enforcementHits7d')}
            value={summary?.enforcementHits7d ?? 0}
            prefix={<AlertOutlined style={{ color: (summary?.enforcementHits7d ?? 0) > 0 ? '#ff4d4f' : '#999' }} />}
            valueStyle={{ color: (summary?.enforcementHits7d ?? 0) > 0 ? '#ff4d4f' : undefined }}
          />
        </Card>
      </Col>
    </Row>
  );

  const upperPanel = (
    <div style={{ padding: '8px 12px' }}>
      {summaryCards}
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
      <Card
        title={`${t('license.detail')} — ${selectedLicense.licenseName}`}
        size="small"
        extra={
          <Button
            type="link"
            size="small"
            icon={<ArrowRightOutlined />}
            onClick={() => navigate(`/license/logs?license_id=${selectedLicense.id}`)}
          >
            {t('license.detail.viewFullAudit')}
          </Button>
        }
      >
        <LicenseDetailTabs
          license={selectedLicense}
          licenseTypeLabelMap={licenseTypeLabelMap}
          statusLabelMap={statusLabelMap}
          logTypeLabelMap={logTypeLabelMap}
          resultLabelMap={resultLabelMap}
          onJumpToLogs={() =>
            navigate(`/license/logs?license_id=${selectedLicense.id}`)
          }
          t={t}
        />
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

// ---------------------------------------------------------------------------
// 详情 4 Tabs：基本信息 / 容量信息 / 功能特性 / 审计记录
// ---------------------------------------------------------------------------

interface LicenseDetailTabsProps {
  license: License;
  licenseTypeLabelMap: Record<LicenseType, string>;
  statusLabelMap: Record<LicenseStatus, string>;
  logTypeLabelMap: Record<LicenseLogType, string>;
  resultLabelMap: Record<LicenseLogResult, string>;
  onJumpToLogs: () => void;
  t: (key: string, vars?: Record<string, unknown>) => string;
}

function LicenseDetailTabs({
  license,
  licenseTypeLabelMap,
  statusLabelMap,
  logTypeLabelMap,
  resultLabelMap,
  onJumpToLogs,
  t,
}: LicenseDetailTabsProps) {
  const usedPct = Math.round((license.usedDevices / Math.max(license.maxDevices, 1)) * 100);
  const expiry = license.expiryDate ? new Date(license.expiryDate) : null;
  const daysLeft = expiry
    ? Math.ceil((expiry.getTime() - Date.now()) / (1000 * 60 * 60 * 24))
    : null;

  return (
    <Tabs
      defaultActiveKey="basic"
      items={[
        // ---- Tab 1: 基本信息 ----
        {
          key: 'basic',
          label: t('license.detail.tab.basic'),
          children: (
            <Descriptions bordered column={2} size="small">
              <Descriptions.Item label={t('license.licenseName')} span={2}>{license.licenseName}</Descriptions.Item>
              <Descriptions.Item label="License ID">
                <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{license.licenseCode}</span>
              </Descriptions.Item>
              <Descriptions.Item label={t('license.productName')}>{license.productName}</Descriptions.Item>
              <Descriptions.Item label={t('license.licenseType')}>
                <Tag>{licenseTypeLabelMap[license.licenseType]}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label={t('table.status')}>
                <Tag color={statusColorMap[license.status]}>{statusLabelMap[license.status]}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label={t('device.productType')}>{license.deviceType || '—'}</Descriptions.Item>
              <Descriptions.Item label={t('table.region')}>{license.region || '—'}</Descriptions.Item>
              <Descriptions.Item label={t('license.licensor')}>{license.licensor || '—'}</Descriptions.Item>
              <Descriptions.Item label={t('license.issueDate')}>
                {new Date(license.issueDate).toLocaleDateString('zh-CN')}
              </Descriptions.Item>
              <Descriptions.Item label={t('license.expiryDate')}>
                {expiry ? expiry.toLocaleDateString('zh-CN') : t('license.permanentValid')}
              </Descriptions.Item>
              {license.notes && (
                <Descriptions.Item label={t('license.notes')} span={2}>{license.notes}</Descriptions.Item>
              )}
            </Descriptions>
          ),
        },

        // ---- Tab 2: 容量信息 ----
        {
          key: 'capacity',
          label: t('license.detail.tab.capacity'),
          children: (
            <Descriptions bordered column={2} size="small">
              <Descriptions.Item label={t('license.summary.maxDevices')}>{license.maxDevices}</Descriptions.Item>
              <Descriptions.Item label={t('license.summary.usedDevices')}>{license.usedDevices}</Descriptions.Item>
              <Descriptions.Item label={t('license.detail.usageRatio')} span={2}>
                <Progress
                  percent={usedPct}
                  size="default"
                  status={usedPct >= 90 ? 'exception' : usedPct >= 80 ? 'active' : 'normal'}
                  format={() => `${license.usedDevices} / ${license.maxDevices} (${usedPct}%)`}
                />
              </Descriptions.Item>
              <Descriptions.Item label={t('license.detail.daysRemaining')} span={2}>
                {daysLeft === null ? (
                  <Tag color="green">{t('license.permanent')}</Tag>
                ) : daysLeft < 0 ? (
                  <Tag color="red">{t('license.expired')} ({-daysLeft} {t('license.detail.daysAgo')})</Tag>
                ) : daysLeft < 7 ? (
                  <Tag color="red">{daysLeft} {t('license.detail.daysLeft')}</Tag>
                ) : daysLeft < 30 ? (
                  <Tag color="orange">{daysLeft} {t('license.detail.daysLeft')}</Tag>
                ) : (
                  <Tag color="green">{daysLeft} {t('license.detail.daysLeft')}</Tag>
                )}
              </Descriptions.Item>
            </Descriptions>
          ),
        },

        // ---- Tab 3: 功能特性 ----
        {
          key: 'features',
          label: t('license.detail.tab.features'),
          children: license.features && license.features.length > 0 ? (
            <Space wrap>
              {license.features.map((f, i) => (
                <Tag key={i} color="blue">{f}</Tag>
              ))}
            </Space>
          ) : (
            <Empty description={t('license.detail.noFeatures')} />
          ),
        },

        // ---- Tab 4: 审计记录（最近 10 条 + 跳转完整 Logs）----
        {
          key: 'audit',
          label: t('license.detail.tab.audit'),
          children: (
            <LicenseAuditList
              licenseId={license.id}
              logTypeLabelMap={logTypeLabelMap}
              resultLabelMap={resultLabelMap}
              onJumpToFull={onJumpToLogs}
              t={t}
            />
          ),
        },
      ]}
    />
  );
}

// ---------------------------------------------------------------------------
// 审计 Tab：最近 10 条 + 跳转完整 Logs
// ---------------------------------------------------------------------------

interface LicenseAuditListProps {
  licenseId: string;
  logTypeLabelMap: Record<LicenseLogType, string>;
  resultLabelMap: Record<LicenseLogResult, string>;
  onJumpToFull: () => void;
  t: (key: string, vars?: Record<string, unknown>) => string;
}

function LicenseAuditList({
  licenseId,
  logTypeLabelMap,
  resultLabelMap,
  onJumpToFull,
  t,
}: LicenseAuditListProps) {
  const { data: logs, isLoading } = useLicenseLogsByLicense(licenseId, 10);

  if (isLoading) {
    return <div style={{ color: '#999' }}>{t('common.loading')}</div>;
  }
  if (!logs || logs.length === 0) {
    return <Empty description={t('license.detail.noAudit')} />;
  }

  return (
    <>
      <List<LicenseLog>
        dataSource={logs}
        size="small"
        renderItem={(log) => (
          <List.Item
            key={log.id}
            extra={
              <span style={{ fontSize: 11, color: '#999' }}>
                {new Date(log.createdAt).toLocaleString('zh-CN')}
              </span>
            }
          >
            <Space size={6}>
              <Tag color={logTypeColorMap[log.logType]}>{logTypeLabelMap[log.logType]}</Tag>
              <Tag color={resultColorMap[log.result]}>{resultLabelMap[log.result]}</Tag>
              <Tooltip
                title={
                  <pre style={{ margin: 0, fontSize: 11, maxWidth: 480 }}>
                    {JSON.stringify(log.details, null, 2)}
                  </pre>
                }
              >
                <span style={{ cursor: 'help' }}>
                  {(log.details?.summary as string) || JSON.stringify(log.details).slice(0, 80)}
                </span>
              </Tooltip>
            </Space>
          </List.Item>
        )}
      />
      <div style={{ marginTop: 12, textAlign: 'right' }}>
        <Button type="link" size="small" icon={<ArrowRightOutlined />} onClick={onJumpToFull}>
          {t('license.detail.viewFullAudit')}
        </Button>
      </div>
    </>
  );
}
