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
    try {
      const values = await form.validateFields();
      await createMutation.mutateAsync({
        scriptName: values.scriptName,
        description: values.description || '',
        content,
        deviceType: values.deviceType || '',
        creator: '',
        tags: values.tags || [],
      });
      void message.success(t('mml.scriptSaved'));
      onClose();
    } catch {
      // validation errors shown inline
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
          label={t('mml.scriptName') || '脚本名称'}
          rules={[{ required: true, message: t('mml.inputScriptName') || '请输入脚本名称' }]}
        >
          <Input placeholder={t('mml.inputScriptName') || '请输入脚本名称'} maxLength={200} />
        </Form.Item>

        <Form.Item name="deviceType" label={t('mml.productType') || '产品类型'}>
          <Select placeholder={t('mml.selectProductType') || '选择产品类型'} allowClear options={productTypeOptions} />
        </Form.Item>

        <Form.Item name="tags" label={t('mml.tags') || '标签'}>
          <Select mode="tags" placeholder={t('mml.inputTags') || '输入标签'} />
        </Form.Item>

        <Form.Item name="description" label={t('common.description') || '描述'}>
          <Input.TextArea rows={2} placeholder={t('mml.scriptDescription') || '脚本描述'} maxLength={500} />
        </Form.Item>

        <Form.Item label={t('mml.scriptContent') || '脚本内容'}>
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
