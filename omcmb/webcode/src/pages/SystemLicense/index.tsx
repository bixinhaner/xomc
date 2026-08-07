/**
 * SystemLicensePage — F06 System License 重构 Step 4 主皮肤单页。
 *
 * 与 PRD F06-system-license-redesign §6 wireframe 对齐：
 *   - 顶部 toolbar：刷新 + Update 按钮（左下角"Update"按钮的 wireframe 改到右上
 *     toolbar，与其他模块风格一致；功能等价）
 *   - Basic Info：License ID / Type / Expiry / Signature / Issued / Uploaded
 *   - Devices Support：所有 device_type 的方块卡片
 *   - Feature List：三级嵌套展示
 *   - 空态：未配置 license 时引导用户上传
 *
 * 路由：/license（替代老 /license/list）。History 列表见 /license/history。
 */
import { useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Button,
  Card,
  Col,
  Descriptions,
  Empty,
  Modal,
  Result,
  Row,
  Space,
  Spin,
  Statistic,
  Tag,
  Typography,
} from 'antd';
import {
  CloudUploadOutlined,
  DownloadOutlined,
  EyeOutlined,
  HistoryOutlined,
  ReloadOutlined,
} from '@ant-design/icons';

import { useT } from '@/hooks/useT';
import {
  useSystemLicense,
} from '@core/hooks/api/useSystemLicense';
import type {
  SystemLicense,
  SystemLicenseSignatureStatus,
} from '@core/services/api/systemLicenseApi';
import {
  decodeSystemLicenseRawContent,
  SystemLicenseErrorCodes,
  extractLicenseErrorCode,
} from '@core/services/api/systemLicenseApi';
import UpdateModal from './UpdateModal';
import { FeatureListView } from './FeatureListView';
import { formatSystemTime } from '@core/utils/systemTime';

const { Title, Text, Paragraph } = Typography;

function signatureTagColor(status: SystemLicenseSignatureStatus): string {
  switch (status) {
    case 'verified':
      return 'success';
    case 'invalid':
      return 'error';
    default:
      return 'warning';
  }
}

function signatureTagLabelKey(status: SystemLicenseSignatureStatus): string {
  switch (status) {
    case 'verified':
      return 'systemLicense.signature.verified';
    case 'invalid':
      return 'systemLicense.signature.invalid';
    default:
      return 'systemLicense.signature.unverified';
  }
}

function formatDateTime(iso: string | null | undefined): string {
  if (!iso) return '-';
  return formatSystemTime(iso);
}

function daysRemaining(expiryISO: string | null): number | null {
  if (!expiryISO) return null;
  const ms = new Date(expiryISO).getTime() - Date.now();
  return Math.floor(ms / (1000 * 60 * 60 * 24));
}

interface DevicesSupportCardsProps {
  devicesSupport: SystemLicense['devicesSupport'];
}

function DevicesSupportCards({ devicesSupport }: DevicesSupportCardsProps) {
  const t = useT();
  const entries = useMemo(
    () => Object.entries(devicesSupport ?? {}).sort(([a], [b]) => a.localeCompare(b)),
    [devicesSupport],
  );
  if (entries.length === 0) {
    return <Empty description={t('systemLicense.devicesSupport.empty')} />;
  }
  return (
    <Row gutter={[16, 16]}>
      {entries.map(([deviceType, capacity]) => (
        <Col key={deviceType} xs={12} sm={8} md={6} lg={4}>
          <Card size="small" variant="outlined">
            <Statistic
              title={<span style={{ fontWeight: 600 }}>{deviceType}</span>}
              value={capacity}
              valueStyle={{ fontSize: 24, color: '#1677ff' }}
            />
          </Card>
        </Col>
      ))}
    </Row>
  );
}

export default function SystemLicensePage() {
  const t = useT();
  const navigate = useNavigate();
  const [updateOpen, setUpdateOpen] = useState(false);
  const [rawOpen, setRawOpen] = useState(false);

  const { data: lic, isLoading, error, refetch } = useSystemLicense();

  const errorCode = error ? extractLicenseErrorCode(error) : 0;
  const isNotConfigured = errorCode === SystemLicenseErrorCodes.NotConfigured;

  // 加载中
  if (isLoading) {
    return (
      <div style={{ padding: 48, textAlign: 'center' }}>
        <Spin size="large" />
      </div>
    );
  }

  // 真错误（非"表空"）
  if (error && !isNotConfigured) {
    return (
      <Result
        status="error"
        title={t('common.error')}
        subTitle={(error as Error)?.message ?? ''}
        extra={
          <Button type="primary" onClick={() => refetch()}>
            {t('common.retry') /* 已有 key */}
          </Button>
        }
      />
    );
  }

  // 空态（system_license 表无 current 行）
  if (isNotConfigured || !lic) {
    return (
      <Card>
        <Result
          icon={<CloudUploadOutlined style={{ color: '#1677ff' }} />}
          title={t('systemLicense.notConfigured')}
          subTitle={t('systemLicense.notConfiguredHint')}
          extra={
            <Button type="primary" icon={<CloudUploadOutlined />} onClick={() => setUpdateOpen(true)}>
              {t('systemLicense.update')}
            </Button>
          }
        />
        <UpdateModal open={updateOpen} onClose={() => setUpdateOpen(false)} />
      </Card>
    );
  }

  const remain = daysRemaining(lic.expiryDate);
  const expiryLabel = (() => {
    if (!lic.expiryDate) return t('systemLicense.basicInfo.perpetual');
    if (remain !== null && remain < 0) return t('systemLicense.basicInfo.expired');
    return `${formatDateTime(lic.expiryDate)} · ${t('systemLicense.basicInfo.remainDays', { days: remain ?? 0 })}`;
  })();

  const downloadRawLicense = () => {
    const raw = decodeSystemLicenseRawContent(lic.rawContent);
    const blob = new Blob([raw.slice().buffer], { type: 'application/octet-stream' });
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement('a');
    anchor.href = url;
    anchor.download = `${lic.licenseId}.lic`;
    anchor.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div style={{ padding: 24 }}>
      {/* 顶部 toolbar */}
      <div style={{ display: 'flex', alignItems: 'center', marginBottom: 16 }}>
        <Title level={4} style={{ margin: 0, flex: 1 }}>
          {t('systemLicense.title')}
        </Title>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => refetch()}>
            {t('common.refresh') /* 已有 key */}
          </Button>
          <Button icon={<HistoryOutlined />} onClick={() => navigate('/license/history')}>
            {t('systemLicense.history.title')}
          </Button>
          <Button icon={<EyeOutlined />} onClick={() => setRawOpen(true)}>
            {t('systemLicense.raw.view')}
          </Button>
          <Button icon={<DownloadOutlined />} onClick={downloadRawLicense}>
            {t('systemLicense.raw.download')}
          </Button>
          <Button type="primary" icon={<CloudUploadOutlined />} onClick={() => setUpdateOpen(true)}>
            {t('systemLicense.update')}
          </Button>
        </Space>
      </div>

      {/* Basic Info */}
      <Card title={t('systemLicense.basicInfo')} style={{ marginBottom: 16 }}>
        <Descriptions column={{ xs: 1, sm: 2, md: 2, lg: 3 }} bordered size="small">
          <Descriptions.Item label={t('systemLicense.basicInfo.licenseId')}>
            <Text copyable>{lic.licenseId}</Text>
          </Descriptions.Item>
          <Descriptions.Item label={t('systemLicense.basicInfo.licenseType')}>
            <Tag color="blue">{lic.licenseType}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label={t('systemLicense.basicInfo.expiryDate')}>
            {expiryLabel}
          </Descriptions.Item>
          <Descriptions.Item label={t('systemLicense.basicInfo.issuer')}>
            {lic.issuer ?? '-'}
          </Descriptions.Item>
          <Descriptions.Item label={t('systemLicense.basicInfo.licensee')}>
            {lic.licensee ?? '-'}
          </Descriptions.Item>
          <Descriptions.Item label={t('systemLicense.basicInfo.signature')}>
            <Tag color={signatureTagColor(lic.signatureStatus)}>
              {t(signatureTagLabelKey(lic.signatureStatus))}
            </Tag>
            {lic.signatureKeyId ? (
              <Text type="secondary" style={{ marginLeft: 8, fontSize: 12 }}>
                key: {lic.signatureKeyId.slice(0, 12)}…
              </Text>
            ) : null}
          </Descriptions.Item>
          <Descriptions.Item label={t('systemLicense.basicInfo.issuedAt')}>
            {formatDateTime(lic.issuedAt)}
          </Descriptions.Item>
          <Descriptions.Item label={t('systemLicense.basicInfo.uploadedAt')}>
            {formatDateTime(lic.uploadedAt)}
          </Descriptions.Item>
        </Descriptions>
      </Card>

      {/* Devices Support */}
      <Card title={t('systemLicense.devicesSupport')} style={{ marginBottom: 16 }}>
        <DevicesSupportCards devicesSupport={lic.devicesSupport} />
      </Card>

      {/* Feature List */}
      <Card title={t('systemLicense.featureList')}>
        <FeatureListView featureList={lic.featureList} />
      </Card>

      {/* "Update" 弹窗 */}
      <UpdateModal open={updateOpen} onClose={() => setUpdateOpen(false)} />

      <Modal
        title={t('systemLicense.raw.title')}
        open={rawOpen}
        onCancel={() => setRawOpen(false)}
        footer={null}
        width={860}
      >
        <pre style={{ maxHeight: 520, overflow: 'auto', margin: 0, whiteSpace: 'pre-wrap' }}>
          {lic.rawContent}
        </pre>
      </Modal>

      {/* 友好提示：右下角微小 raw_content 摘要（可选） */}
      <Paragraph type="secondary" style={{ marginTop: 16, fontSize: 12 }}>
        License PK: <Text code>{lic.id}</Text>
      </Paragraph>
    </div>
  );
}
