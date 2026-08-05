import { useMemo } from 'react';
import { Drawer, Form, Input, Select, Switch, Space, Button, Divider, App, Typography, Radio } from 'antd';
import { useT } from '@/hooks/useT';
import { useProductClasses } from '@core/hooks/api/useDevices';
import { useProductList } from '@core/hooks/api/useProducts';
import { toSupportedProductClassOptions } from '../productClassOptions';
import ProductClassSelect from './ProductClassSelect';

interface Policy {
  policyId: string;
  policyName: string;
  productClass: string;
  executeType: '0' | '1';
  selfStartEnable: '0' | '1';
  upgradeEnable: '0' | '1';
  targetVersion: string[];
  licenseEnable: '0' | '1';
  selfConfigEnable: '0' | '1';
  createTime?: string;
  updateTime?: string;
}

interface Props {
  open: boolean;
  mode: 'add' | 'edit' | 'view';
  policy: Policy | null;
  onClose: () => void;
  onSubmit: (values: Record<string, unknown>) => void;
}

const TARGET_VERSIONS = [
  { label: 'V2.2.0', value: 'V2.2.0' },
  { label: 'V2.1.0', value: 'V2.1.0' },
  { label: 'V2.0.5', value: 'V2.0.5' },
  { label: 'V2.0.0', value: 'V2.0.0' },
  { label: 'V1.5.0', value: 'V1.5.0' },
];

// Product types that support License (eNB types)
const LICENSE_SUPPORTED_TYPES = ['QAFA', 'QAFB', 'QAFC'];

export default function PolicyDrawer({ open, mode, policy, onClose, onSubmit }: Props) {
  const t = useT();
  const [form] = Form.useForm();
  void App.useApp();
  const { data: supportedProductClasses, isLoading: productClassesLoading } = useProductClasses();
  const { data: productCatalog, isLoading: productCatalogLoading } = useProductList();
  const productClassOptions = useMemo(
    () => toSupportedProductClassOptions(supportedProductClasses, productCatalog?.items),
    [productCatalog?.items, supportedProductClasses],
  );

  const isView = mode === 'view';
  const title = useMemo(() => {
    if (mode === 'add') return t('common.add') + ' Policy';
    if (mode === 'edit') return t('common.edit') + ' Policy';
    return t('common.detail');
  }, [mode, t]);

  // Initial values
  const initialValues = useMemo(() => {
    if (policy) {
      return {
        policyName: policy.policyName,
        productClass: policy.productClass,
        executeType: policy.executeType,
        upgradeEnable: policy.upgradeEnable === '1',
        targetVersion: policy.targetVersion?.[0],
        licenseEnable: policy.licenseEnable === '1',
        selfConfigEnable: policy.selfConfigEnable === '1',
      };
    }
    return {
      policyName: '',
      productClass: undefined,
      executeType: '0',
      upgradeEnable: false,
      targetVersion: undefined,
      licenseEnable: false,
      selfConfigEnable: false,
    };
  }, [policy]);

  // Handle submit
  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      onSubmit({
        ...values,
        upgradeEnable: values.upgradeEnable ? '1' : '0',
        licenseEnable: values.licenseEnable ? '1' : '0',
        selfConfigEnable: values.selfConfigEnable ? '1' : '0',
        targetVersion: values.targetVersion ? [values.targetVersion] : [],
      });
    } catch (_error) {
      // Form validation error
    }
  };

  // Check if License is supported based on product type
  const isLicenseSupported = (productClass: string) => {
    return LICENSE_SUPPORTED_TYPES.includes(productClass);
  };

  return (
    <Drawer
      title={title}
      open={open}
      onClose={onClose}
      size={600}
      footer={
        isView ? (
          <Button onClick={onClose}>{t('common.close')}</Button>
        ) : (
          <Space>
            <Button onClick={onClose}>{t('common.cancel')}</Button>
            <Button type="primary" onClick={handleSubmit}>
              {t('common.confirm')}
            </Button>
          </Space>
        )
      }
    >
      <Form
        form={form}
        layout="vertical"
        initialValues={initialValues}
        disabled={isView}
      >
        <Form.Item
          name="policyName"
          label={t('provision.policyName')}
          rules={[{ required: true, message: t('common.pleaseInput') }]}
        >
          <Input placeholder={t('provision.policyNamePlaceholder')} maxLength={50} />
        </Form.Item>

        <Form.Item
          name="productClass"
          label={t('provision.productClass')}
          rules={[{ required: true, message: t('common.pleaseSelect') }]}
        >
          <ProductClassSelect
            placeholder={t('common.pleaseSelect')}
            options={productClassOptions}
            loading={productClassesLoading || productCatalogLoading}
          />
        </Form.Item>

        <Form.Item
          name="executeType"
          label={t('provision.executeType')}
          rules={[{ required: true }]}
        >
          <Radio.Group>
            <Radio value="0">{t('provision.autoExecute')}</Radio>
            <Radio value="1">{t('provision.manualExecute')}</Radio>
          </Radio.Group>
        </Form.Item>

        <Divider>{t('provision.functionConfig')}</Divider>

        <Form.Item
          name="upgradeEnable"
          label={t('provision.softwareUpgrade')}
          valuePropName="checked"
        >
          <Switch checkedChildren={t('common.on')} unCheckedChildren={t('common.off')} />
        </Form.Item>

        <Form.Item
          noStyle
          shouldUpdate={(prev, curr) => prev.upgradeEnable !== curr.upgradeEnable}
        >
          {({ getFieldValue }) =>
            getFieldValue('upgradeEnable') && (
              <Form.Item
                name="targetVersion"
                label={t('provision.targetVersion')}
                rules={[{ required: true, message: t('common.pleaseSelect') }]}
              >
                <Select placeholder={t('common.pleaseSelect')} options={TARGET_VERSIONS} />
              </Form.Item>
            )
          }
        </Form.Item>

        <Form.Item
          noStyle
          shouldUpdate={(prev, curr) => prev.productClass !== curr.productClass}
        >
          {({ getFieldValue }) => {
            const productClass = getFieldValue('productClass');
            if (isLicenseSupported(productClass)) {
              return (
                <Form.Item
                  name="licenseEnable"
                  label="License"
                  valuePropName="checked"
                >
                  <Switch checkedChildren={t('common.on')} unCheckedChildren={t('common.off')} />
                </Form.Item>
              );
            }
            return null;
          }}
        </Form.Item>

        <Form.Item
          name="selfConfigEnable"
          label={t('provision.selfConfig')}
          valuePropName="checked"
        >
          <Switch checkedChildren={t('common.on')} unCheckedChildren={t('common.off')} />
        </Form.Item>

        {isView && policy && (
          <>
            <Divider>{t('provision.otherInfo')}</Divider>
            <Form.Item label={t('provision.createTime')}>
              <Typography.Text>{policy.createTime || '-'}</Typography.Text>
            </Form.Item>
            <Form.Item label={t('provision.updateTime')}>
              <Typography.Text>{policy.updateTime || '-'}</Typography.Text>
            </Form.Item>
          </>
        )}
      </Form>
    </Drawer>
  );
}
