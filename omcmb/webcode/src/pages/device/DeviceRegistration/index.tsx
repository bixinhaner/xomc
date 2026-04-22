import React, { useCallback, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Alert,
  Button,
  Card,
  Col,
  Form,
  Input,
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

const { Title, Text } = Typography;
const { TextArea } = Input;

interface BasicInfoFormValues {
  sn: string;
  name: string;
  vendor: string;
  productType: string;
  networkType: string;
  deviceModel: string;
}

interface NetworkFormValues {
  ipAddress: string;
  subnet: string;
  region: string;
  site: string;
  latitude?: number;
  longitude?: number;
}

type Step = 'basic' | 'network' | 'confirm';

export default function DeviceRegistration() {
  const t = useT();
  const navigate = useNavigate();
  const createDevice = useCreateDevice();
  const [currentStep, setCurrentStep] = useState(0);
  const [basicForm] = Form.useForm<BasicInfoFormValues>();
  const [networkForm] = Form.useForm<NetworkFormValues>();
  const [basicData, setBasicData] = useState<BasicInfoFormValues | null>(null);
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
      // validation error handled by form
    }
  }, [basicForm]);

  const handleNextFromNetwork = useCallback(async () => {
    try {
      const values = await networkForm.validateFields();
      setNetworkData(values);
      setCurrentStep(2);
    } catch {
      // validation error handled by form
    }
  }, [networkForm]);

  const handleSubmit = useCallback(async () => {
    if (!basicData || !networkData) return;
    try {
      await createDevice.mutateAsync({
        ...basicData,
        ...networkData,
        longitude: Number(networkData.longitude ?? 0),
        latitude: Number(networkData.latitude ?? 0),
      } as Parameters<typeof createDevice.mutateAsync>[0]);
      setSubmitted(true);
      void message.success(t('status.success'));
    } catch {
      void message.error(t('status.failed'));
    }
  }, [basicData, networkData, createDevice, t]);

  const renderBasicStep = () => (
    <Form form={basicForm} layout="vertical" size="middle">
      <Row gutter={24}>
        <Col span={12}>
          <Form.Item
            name="sn"
            label={t('device.sn')}
            rules={[
              { required: true, message: t('common.placeholder') },
              { pattern: /^[A-Za-z0-9-_]+$/, message: 'SN: A-Z, 0-9, -, _' },
            ]}
          >
            <Input placeholder={t('common.placeholder')} style={{ fontFamily: 'monospace' }} />
          </Form.Item>
        </Col>
        <Col span={12}>
          <Form.Item
            name="name"
            label={t('device.name')}
            rules={[{ required: true, message: t('common.placeholder') }]}
          >
            <Input placeholder={t('common.placeholder')} />
          </Form.Item>
        </Col>
        <Col span={12}>
          <Form.Item
            name="vendor"
            label={t('device.vendor')}
            rules={[{ required: true, message: t('common.pleaseSelect') }]}
          >
            <Select
              placeholder={t('common.pleaseSelect')}
              options={[
                { label: '华为', value: '华为' },
                { label: '中兴', value: '中兴' },
                { label: '爱立信', value: '爱立信' },
                { label: '大唐', value: '大唐' },
                { label: '京信', value: '京信' },
              ]}
            />
          </Form.Item>
        </Col>
        <Col span={12}>
          <Form.Item
            name="productType"
            label={t('device.productType')}
            rules={[{ required: true, message: t('common.pleaseSelect') }]}
          >
            <Select
              placeholder={t('common.pleaseSelect')}
              options={[
                { label: 'eNB (LTE)', value: 'eNB' },
                { label: 'gNB (5G)', value: 'gNB' },
                { label: 'CPE', value: 'CPE' },
                { label: 'eGW', value: 'eGW' },
              ]}
            />
          </Form.Item>
        </Col>
        <Col span={12}>
          <Form.Item
            name="networkType"
            label={t('device.networkType')}
            rules={[{ required: true, message: t('common.pleaseSelect') }]}
          >
            <Select
              placeholder={t('common.pleaseSelect')}
              options={[
                { label: 'LTE-FDD', value: 'LTE-FDD' },
                { label: 'LTE-TDD', value: 'LTE-TDD' },
                { label: 'NR (5G)', value: 'NR' },
                { label: 'NB-IoT', value: 'NB-IoT' },
              ]}
            />
          </Form.Item>
        </Col>
        <Col span={12}>
          <Form.Item
            name="deviceModel"
            label={t('device.model')}
            rules={[{ required: true, message: t('common.placeholder') }]}
          >
            <Input placeholder="BBU3910, AAU5613" />
          </Form.Item>
        </Col>
      </Row>
    </Form>
  );

  const renderNetworkStep = () => (
    <Form form={networkForm} layout="vertical" size="middle">
      <Row gutter={24}>
        <Col span={12}>
          <Form.Item
            name="ipAddress"
            label={t('device.ipAddress')}
            rules={[
              { required: true, message: t('common.placeholder') },
              {
                pattern: /^(\d{1,3}\.){3}\d{1,3}$/,
                message: 'IP',
              },
            ]}
          >
            <Input placeholder="192.168.1.100" style={{ fontFamily: 'monospace' }} />
          </Form.Item>
        </Col>
        <Col span={12}>
          <Form.Item
            name="subnet"
            label={t('table.description')}
            rules={[{ required: true, message: t('common.placeholder') }]}
          >
            <Input placeholder="192.168.1.0/24" style={{ fontFamily: 'monospace' }} />
          </Form.Item>
        </Col>
        <Col span={12}>
          <Form.Item
            name="region"
            label={t('device.region')}
            rules={[{ required: true, message: t('common.pleaseSelect') }]}
          >
            <Select
              placeholder={t('common.pleaseSelect')}
              options={[
                { label: '华北区', value: '华北区' },
                { label: '华东区', value: '华东区' },
                { label: '华南区', value: '华南区' },
                { label: '西南区', value: '西南区' },
                { label: '西北区', value: '西北区' },
                { label: '东北区', value: '东北区' },
              ]}
            />
          </Form.Item>
        </Col>
        <Col span={12}>
          <Form.Item
            name="site"
            label={t('table.site')}
            rules={[{ required: true, message: t('common.placeholder') }]}
          >
            <Input placeholder={t('common.placeholder')} />
          </Form.Item>
        </Col>
        <Col span={12}>
          <Form.Item name="latitude" label={t('alarm.location')}>
            <Input placeholder="39.9042" type="number" />
          </Form.Item>
        </Col>
        <Col span={12}>
          <Form.Item name="longitude" label={t('alarm.location')}>
            <Input placeholder="116.4074" type="number" />
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
                  { label: t('device.sn'), value: basicData?.sn },
                  { label: t('device.name'), value: basicData?.name },
                  { label: t('device.vendor'), value: basicData?.vendor },
                  { label: t('device.productType'), value: basicData?.productType },
                  { label: t('device.networkType'), value: basicData?.networkType },
                  { label: t('device.model'), value: basicData?.deviceModel },
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
                  { label: t('device.ipAddress'), value: networkData?.ipAddress },
                  { label: t('table.description'), value: networkData?.subnet },
                  { label: t('device.region'), value: networkData?.region },
                  { label: t('table.site'), value: networkData?.site },
                  { label: 'Lat', value: networkData?.latitude ?? '-' },
                  { label: 'Lng', value: networkData?.longitude ?? '-' },
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
        {basicData?.sn}
      </Text>
      <Space size={12}>
        <Button
          type="primary"
          onClick={() => void navigate(`/device/detail/${basicData?.sn ?? ''}`)}
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
