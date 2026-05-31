import { useEffect } from 'react';
import { Modal, Form, Select, message, Alert } from 'antd';
import { useProductList, useBindOrphan } from '@core/hooks/api/useProducts';
import type { OrphanDevice } from '@core/types/product';
import { useT } from '@/hooks/useT';

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
  const t = useT();
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
        message.success(t('product.orphan.bind.success', { success, total: devices.length }));
      } else {
        message.warning(t('product.orphan.bind.partial', { success, failed }));
      }
      onDone();
      onClose();
    } catch (e) {
      const msg = (e as Error).message;
      if (msg) message.error(msg);
    }
  };

  // 2026-05-28: 标题 / Alert 文案明确"用户已选数量"语义,避免与分页 pageSize
  // 混淆。devices.length 始终等于父组件传入的 selectedRows.length(已选行)。
  const count = devices.length;

  return (
    <Modal
      title={
        count === 1
          ? t('product.orphan.bind.titleSingle')
          : t('product.orphan.bind.titleMulti', { count })
      }
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
          count === 1
            ? t('product.orphan.bind.descSingle', { sn: devices[0].serialNumber, pc: devices[0].productClass || '—' })
            : t('product.orphan.bind.descMulti', { count })
        }
      />
      <Form<FormValues> form={form} layout="vertical">
        <Form.Item
          name="productId"
          label={t('product.orphan.targetProduct')}
          rules={[{ required: true, message: t('product.orphan.bind.targetProductRequired') }]}
        >
          <Select
            showSearch
            placeholder={t('product.orphan.bindPh')}
            optionFilterProp="label"
            options={(productData?.items || []).map((p) => ({
              label: `${p.name}(${p.vendor || '—'} · ${p.tech.toUpperCase()})`,
              value: p.id,
            }))}
          />
        </Form.Item>
      </Form>
    </Modal>
  );
}
