import { useEffect } from 'react';
import { Modal, Form, Select, message, Alert } from 'antd';
import { useProductList, useBindOrphan } from '@core/hooks/api/useProducts';
import type { OrphanDevice } from '@core/types/product';

interface Props {
  open: boolean;
  devices: OrphanDevice[];   // 单台或多台
  onClose: () => void;
  onDone: () => void;
}

interface FormValues {
  productId: string;
}

export default function BindProductModal({ open, devices, onClose, onDone }: Props) {
  const { data: productData } = useProductList();
  const bindMut = useBindOrphan();
  const [form] = Form.useForm<FormValues>();

  useEffect(() => {
    if (!open) form.resetFields();
  }, [open, form]);

  const handleSubmit = async () => {
    try {
      const v = await form.validateFields();
      let success = 0;
      let failed = 0;
      // 串行调用，避免后端短时间内大量并发
      for (const d of devices) {
        try {
          await bindMut.mutateAsync({ deviceId: d.id, productId: v.productId });
          success += 1;
        } catch {
          failed += 1;
        }
      }
      if (failed === 0) {
        message.success(`已绑定 ${success}/${devices.length} 台`);
      } else {
        message.warning(`成功 ${success} / 失败 ${failed}`);
      }
      onDone();
      onClose();
    } catch (e) {
      const msg = (e as Error).message;
      if (msg) message.error(msg);
    }
  };

  return (
    <Modal
      title={`绑定产品 — ${devices.length} 台设备`}
      open={open}
      onOk={() => void handleSubmit()}
      onCancel={onClose}
      confirmLoading={bindMut.isPending}
      destroyOnHidden
      width={520}
    >
      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 12 }}
        message={
          devices.length === 1
            ? `设备：${devices[0].serialNumber}（productClass=${devices[0].productClass}）`
            : `批量绑定 ${devices.length} 台；将串行调用单台 bind 接口`
        }
      />
      <Form<FormValues> form={form} layout="vertical">
        <Form.Item
          name="productId"
          label="目标产品"
          rules={[{ required: true, message: '请选择产品' }]}
        >
          <Select
            showSearch
            placeholder="选择要绑定的产品"
            optionFilterProp="label"
            options={(productData?.items || []).map((p) => ({
              label: `${p.name}（${p.vendor || '—'} · ${p.tech.toUpperCase()}）`,
              value: p.id,
            }))}
          />
        </Form.Item>
      </Form>
    </Modal>
  );
}
