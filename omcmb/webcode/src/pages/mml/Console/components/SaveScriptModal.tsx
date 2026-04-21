import { useEffect, useState } from 'react';
import { Modal, Form, Input, Select, Button, message } from 'antd';
import { useCreateMMLScript } from '@/hooks/api/useMML';
import { useDictionary } from '@/hooks/api/useSystem';
import { useT } from '@/hooks/useT';

interface SaveScriptModalProps {
  open: boolean;
  defaultName: string;
  defaultContent: string;
  onClose: () => void;
}

export default function SaveScriptModal({ open, defaultName, defaultContent, onClose }: SaveScriptModalProps) {
  const t = useT();
  const [form] = Form.useForm();
  const createMutation = useCreateMMLScript();
  const { data: productTypeDict } = useDictionary('product_type');

  const productTypeOptions = (productTypeDict?.sysDictionaryDetails ?? []).map((d) => ({
    label: d.label,
    value: d.value,
  }));

  const [content, setContent] = useState(defaultContent);

  useEffect(() => {
    if (open) {
      form.resetFields();
      form.setFieldsValue({
        scriptName: defaultName,
        description: '',
        deviceType: undefined,
        tags: [],
      });
      setContent(defaultContent);
    }
  }, [defaultContent, defaultName, form, open]);

  const handleSubmit = async () => {
    let values;
    try {
      values = await form.validateFields();
    } catch {
      return;
    }
    try {
      await createMutation.mutateAsync({
        scriptName: values.scriptName,
        description: values.description || '',
        content,
        deviceType: values.deviceType || '',
        creator: '',
        tags: values.tags || [],
      });
      message.success(t('mml.scriptSaved'));
      onClose();
    } catch (err) {
      message.error(t('mml.scriptSaveFailed', { error: err instanceof Error ? err.message : 'Unknown' }));
    }
  };

  return (
    <Modal
      title={t('mml.console.saveScript')}
      open={open}
      onCancel={onClose}
      width={560}
      destroyOnClose
      footer={
        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
          <Button onClick={onClose}>{t('common.cancel')}</Button>
          <Button type="primary" onClick={() => void handleSubmit()} loading={createMutation.isPending}>
            {t('common.confirm')}
          </Button>
        </div>
      }
    >
      <Form form={form} layout="vertical" size="small">
        <Form.Item
          name="scriptName"
          label={t('mml.scriptName')}
          rules={[{ required: true, message: t('mml.inputScriptName') }]}
        >
          <Input placeholder={t('mml.inputScriptName')} maxLength={200} />
        </Form.Item>

        <Form.Item name="deviceType" label={t('mml.productType')}>
          <Select placeholder={t('mml.selectProductType')} allowClear options={productTypeOptions} />
        </Form.Item>

        <Form.Item name="tags" label={t('mml.tags')}>
          <Select mode="tags" placeholder={t('mml.inputTags')} />
        </Form.Item>

        <Form.Item name="description" label={t('common.description')}>
          <Input.TextArea rows={2} placeholder={t('mml.scriptDescription')} maxLength={500} />
        </Form.Item>

        <Form.Item label={t('mml.scriptContent')}>
          <Input.TextArea
            rows={6}
            value={content}
            onChange={(e) => setContent(e.target.value)}
            style={{ fontFamily: "'SFMono-Regular', Consolas, monospace" }}
          />
        </Form.Item>
      </Form>
    </Modal>
  );
}
