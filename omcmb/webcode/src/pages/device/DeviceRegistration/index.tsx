import { useCallback, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Alert,
  Button,
  Card,
  Col,
  Form,
  Input,
  InputNumber,
  Row,
  Select,
  Space,
  Steps,
  Typography,
  message,
} from 'antd';
import {
  ArrowLeftOutlined,
  ArrowRightOutlined,
  CheckCircleOutlined,
  SaveOutlined,
} from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useCreateDevice } from '@core/hooks/api/useDevices';
import { useT } from '@/hooks/useT';
import type { CarrierCode, DeviceTechnology, CreateDeviceInput } from '@core/types/device';

const { Title, Text } = Typography;

interface BasicFormValues {
  serialNumber: string;
  oui: string;
  carrier: CarrierCode;
  technology: DeviceTechnology;
  manufacturer?: string;
  productClass?: string;
  modelName?: string;
}

interface NetworkFormValues {
  ipAddress?: string;
  deviceName?: string;
  siteId?: string;
  latitude?: number;
  longitude?: number;
}

type Step = 'basic' | 'network' | 'confirm';

const CARRIER_OPTIONS: { label: string; value: CarrierCode }[] = [
  { label: '中国移动 (cmcc)', value: 'cmcc' },
  { label: '中国电信 (ctcc)', value: 'ctcc' },
  { label: '中国联通 (cucc)', value: 'cucc' },
];

const TECHNOLOGY_OPTIONS: { label: string; value: DeviceTechnology }[] = [
  { label: 'LTE (4G)', value: 'lte' },
  { label: 'NR (5G)', value: 'nr' },
];

// productClass 是 ProductRegistry 路由 key（CPE 通过 TR-069 Inform 上报真实值后会覆盖此处占位）。
// 这里给的几个粗分类供运维预登记时选一个，CPE 上线后自动更新。
const PRODUCT_CLASS_OPTIONS = [
  { label: 'eNB (LTE 基站)', value: 'eNB' },
  { label: 'gNB (5G 基站)', value: 'gNB' },
  { label: 'CPE', value: 'CPE' },
  { label: 'eGW', value: 'eGW' },
];

export default function DeviceRegistration() {
  const t = useT();
  const navigate = useNavigate();
  const createDevice = useCreateDevice();
  const [currentStep, setCurrentStep] = useState(0);
  const [basicForm] = Form.useForm<BasicFormValues>();
  const [networkForm] = Form.useForm<NetworkFormValues>();
  const [basicData, setBasicData] = useState<BasicFormValues | null>(null);
  const [networkData, setNetworkData] = useState<NetworkFormValues | null>(null);
  const [submitted, setSubmitted] = useState(false);

  const STEPS: { title: string; key: Step }[] = [
    { title: t('common.detail'), key: 'basic' },
    { title: t('device.ipAddress'), key: 'network' },
    { title: t('common.submit'), key: 'confirm' },
  ];

  const handleNextFromBasic = useCallback(async () => {
    try {
      const values = await basicForm.validateFields();
      setBasicData(values);
      setCurrentStep(1);
    } catch {
      // validation error shown by form
    }
  }, [basicForm]);

  const handleNextFromNetwork = useCallback(async () => {
    try {
      const values = await networkForm.validateFields();
      setNetworkData(values);
      setCurrentStep(2);
    } catch {
      // validation error shown by form
    }
  }, [networkForm]);

  const handleSubmit = useCallback(async () => {
    if (!basicData || !networkData) return;
    const input: CreateDeviceInput = {
      ...basicData,
      ...networkData,
    };
    try {
      await createDevice.mutateAsync(input as unknown as Parameters<typeof createDevice.mutateAsync>[0]);
      setSubmitted(true);
      void message.success(t('status.success'));
    } catch (err: unknown) {
      // 把后端 BizCode/Message 显式露给用户，否则只能看到通用 failed
      const msg = err instanceof Error ? err.message : t('status.failed');
      void message.error(msg);
    }
  }, [basicData, networkData, createDevice, t]);

  const renderBasicStep = () => (
    <Form form={basicForm} layout="vertical" size="middle">
      <Row gutter={24}>
        <Col span={12}>
          <Form.Item
            name="serialNumber"
            label={t('device.sn')}
            rules={[
              { required: true, message: t('common.placeholder') },
              { pattern: /^[A-Za-z0-9-_]+$/, message: 'SN: A-Z, 0-9, -, _' },
            ]}
          >
            <Input placeholder="例如 BCL2024001234" style={{ fontFamily: 'monospace' }} />
          </Form.Item>
        </Col>
        <Col span={12}>
          <Form.Item
            name="oui"
            label={t('device.oui')}
            tooltip={t('device.ouiHint')}
            rules={[
              { required: true, message: t('common.placeholder') },
              { pattern: /^[0-9A-Fa-f]{6}$/, message: 'OUI: 6-hex (e.g. 48575A)' },
            ]}
          >
            <Input placeholder="48575A" maxLength={6} style={{ fontFamily: 'monospace', textTransform: 'uppercase' }} />
          </Form.Item>
        </Col>
        <Col span={12}>
          <Form.Item
            name="carrier"
            label={t('device.carrier')}
            rules={[{ required: true, message: t('common.pleaseSelect') }]}
          >
            <Select placeholder={t('common.pleaseSelect')} options={CARRIER_OPTIONS} />
          </Form.Item>
        </Col>
        <Col span={12}>
          <Form.Item
            name="technology"
            label={t('device.networkType')}
            rules={[{ required: true, message: t('common.pleaseSelect') }]}
          >
            <Select placeholder={t('common.pleaseSelect')} options={TECHNOLOGY_OPTIONS} />
          </Form.Item>
        </Col>
        <Col span={12}>
          <Form.Item name="manufacturer" label={t('device.vendor')}>
            <Input placeholder="Baicells" />
          </Form.Item>
        </Col>
        <Col span={12}>
          <Form.Item
            name="productClass"
            label={t('device.productClass')}
            tooltip={t('device.productClassHint')}
          >
            <Select
              placeholder={t('common.pleaseSelect')}
              options={PRODUCT_CLASS_OPTIONS}
              allowClear
            />
          </Form.Item>
        </Col>
        <Col span={12}>
          <Form.Item name="modelName" label={t('device.model')}>
            <Input placeholder="BBU3910 / AAU5613" />
          </Form.Item>
        </Col>
      </Row>
    </Form>
  );

  const renderNetworkStep = () => (
    <Form form={networkForm} layout="vertical" size="middle">
      <Alert
        type="info"
        showIcon
        message={t('device.networkStepHint')}
        style={{ marginBottom: 16 }}
      />
      <Row gutter={24}>
        <Col span={12}>
          <Form.Item
            name="ipAddress"
            label={t('device.ipAddress')}
            rules={[
              { pattern: /^(\d{1,3}\.){3}\d{1,3}$/, message: 'IPv4 dotted quad' },
            ]}
          >
            <Input placeholder="192.168.1.100" style={{ fontFamily: 'monospace' }} />
          </Form.Item>
        </Col>
        <Col span={12}>
          <Form.Item name="deviceName" label={t('table.site')}>
            <Input placeholder={t('common.placeholder')} />
          </Form.Item>
        </Col>
        <Col span={12}>
          <Form.Item name="siteId" label={t('device.siteId')}>
            <Input placeholder="SITE-001" />
          </Form.Item>
        </Col>
        <Col span={12}>
          {/* placeholder spacer to align grid */}
        </Col>
        <Col span={12}>
          <Form.Item name="latitude" label={t('device.latitude')}>
            <InputNumber
              placeholder="39.9042"
              style={{ width: '100%' }}
              min={-90}
              max={90}
              step={0.0001}
            />
          </Form.Item>
        </Col>
        <Col span={12}>
          <Form.Item name="longitude" label={t('device.longitude')}>
            <InputNumber
              placeholder="116.4074"
              style={{ width: '100%' }}
              min={-180}
              max={180}
              step={0.0001}
            />
          </Form.Item>
        </Col>
      </Row>
    </Form>
  );

  const renderConfirmStep = () => (
    <div>
      <Alert
        message={t('common.confirm')}
        type="info"
        showIcon
        style={{ marginBottom: 20 }}
      />
      <Row gutter={24}>
        <Col span={12}>
          <Card title={t('common.detail')} size="small">
            <table style={{ width: '100%', fontSize: 14, borderCollapse: 'collapse' }}>
              <tbody>
                {[
                  { label: t('device.sn'), value: basicData?.serialNumber },
                  { label: t('device.oui'), value: basicData?.oui?.toUpperCase() },
                  { label: t('device.carrier'), value: basicData?.carrier },
                  { label: t('device.networkType'), value: basicData?.technology },
                  { label: t('device.vendor'), value: basicData?.manufacturer || '-' },
                  { label: t('device.productClass'), value: basicData?.productClass || '-' },
                  { label: t('device.model'), value: basicData?.modelName || '-' },
                ].map(({ label, value }) => (
                  <tr key={label} style={{ borderBottom: '1px solid #f0f0f0' }}>
                    <td style={{ padding: '8px 0', color: '#8c8c8c', width: '40%' }}>{label}</td>
                    <td style={{ padding: '8px 0', fontWeight: 500 }}>{value ?? '-'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </Card>
        </Col>
        <Col span={12}>
          <Card title={t('device.ipAddress')} size="small">
            <table style={{ width: '100%', fontSize: 14, borderCollapse: 'collapse' }}>
              <tbody>
                {[
                  { label: t('device.ipAddress'), value: networkData?.ipAddress || '-' },
                  { label: t('table.site'), value: networkData?.deviceName || '-' },
                  { label: t('device.siteId'), value: networkData?.siteId || '-' },
                  { label: t('device.latitude'), value: networkData?.latitude ?? '-' },
                  { label: t('device.longitude'), value: networkData?.longitude ?? '-' },
                ].map(({ label, value }) => (
                  <tr key={label} style={{ borderBottom: '1px solid #f0f0f0' }}>
                    <td style={{ padding: '8px 0', color: '#8c8c8c', width: '40%' }}>{label}</td>
                    <td style={{ padding: '8px 0', fontWeight: 500 }}>{String(value ?? '-')}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </Card>
        </Col>
      </Row>
    </div>
  );

  const renderSuccessStep = () => (
    <div style={{ textAlign: 'center', padding: '48px 0' }}>
      <CheckCircleOutlined style={{ fontSize: 64, color: '#52C41A', marginBottom: 24 }} />
      <Title level={3} style={{ color: '#52C41A' }}>
        {t('status.success')}
      </Title>
      <Text type="secondary" style={{ display: 'block', marginBottom: 24 }}>
        {basicData?.serialNumber}
      </Text>
      <Space size={12}>
        <Button
          type="primary"
          onClick={() => void navigate(`/device/detail/${basicData?.serialNumber ?? ''}`)}
        >
          {t('common.view')}
        </Button>
        <Button
          onClick={() => {
            basicForm.resetFields();
            networkForm.resetFields();
            setBasicData(null);
            setNetworkData(null);
            setCurrentStep(0);
            setSubmitted(false);
          }}
        >
          {t('common.add')}
        </Button>
        <Button onClick={() => void navigate('/device/list')}>{t('common.back')}</Button>
      </Space>
    </div>
  );

  if (submitted) {
    return (
      <ListPageLayout title={t('nav.device.register')}>
        <Card>{renderSuccessStep()}</Card>
      </ListPageLayout>
    );
  }

  return (
    <ListPageLayout
      title={t('nav.device.register')}
      extra={
        <Button icon={<ArrowLeftOutlined />} onClick={() => void navigate('/device/list')}>
          {t('common.back')}
        </Button>
      }
    >
      <Card>
        <Steps
          current={currentStep}
          items={STEPS.map((s) => ({ title: s.title }))}
          style={{ marginBottom: 32, maxWidth: 600, margin: '0 auto 32px' }}
        />

        <div style={{ maxWidth: 900, margin: '0 auto' }}>
          {currentStep === 0 && renderBasicStep()}
          {currentStep === 1 && renderNetworkStep()}
          {currentStep === 2 && renderConfirmStep()}
        </div>

        <div style={{ display: 'flex', justifyContent: 'center', marginTop: 32, gap: 12 }}>
          {currentStep > 0 && (
            <Button icon={<ArrowLeftOutlined />} onClick={() => setCurrentStep((s) => s - 1)}>
              {t('common.prev')}
            </Button>
          )}
          {currentStep === 0 && (
            <Button type="primary" icon={<ArrowRightOutlined />} onClick={() => void handleNextFromBasic()}>
              {t('common.next')}
            </Button>
          )}
          {currentStep === 1 && (
            <Button type="primary" icon={<ArrowRightOutlined />} onClick={() => void handleNextFromNetwork()}>
              {t('common.next')}
            </Button>
          )}
          {currentStep === 2 && (
            <Button
              type="primary"
              icon={<SaveOutlined />}
              loading={createDevice.isPending}
              onClick={() => void handleSubmit()}
            >
              {t('common.submit')}
            </Button>
          )}
        </div>
      </Card>
    </ListPageLayout>
  );
}
