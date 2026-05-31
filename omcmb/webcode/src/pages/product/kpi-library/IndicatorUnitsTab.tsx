import { useState } from 'react';
import { Card, Table, Button, Space, Modal, Form, Input, Popconfirm, message } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import {
  useIndicatorUnits,
  useUpsertUnit,
  useUpdateUnit,
  useDeleteUnit,
} from '@core/hooks/api/useIndicatorsLibrary';
import type { IndicatorUnit, UnitInput } from '@core/types/indicatorLibrary';
import { useT } from '@/hooks/useT';

export default function IndicatorUnitsTab() {
  const t = useT();
  const { data, isLoading } = useIndicatorUnits();
  const upsertMut = useUpsertUnit();
  const updateMut = useUpdateUnit();
  const deleteMut = useDeleteUnit();

  const [editing, setEditing] = useState<IndicatorUnit | null>(null);
  const [creating, setCreating] = useState(false);
  const [form] = Form.useForm<UnitInput>();

  const items = data?.items || [];

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 200 },
    { title: t('common.enName'), dataIndex: 'enName', width: 240 },
    { title: t('common.cnName'), dataIndex: 'cnName' },
    {
      title: t('common.action'),
      width: 130,
      render: (_: unknown, row: IndicatorUnit) => (
        <Space>
          <Button
            size="small"
            icon={<EditOutlined />}
            onClick={() => {
              setEditing(row);
              form.setFieldsValue(row);
            }}
          />
          <Popconfirm
            title={t('product.kpi.confirmDeleteUnit', { id: row.id })}
            onConfirm={() =>
              deleteMut
                .mutateAsync(row.id)
                .then(() => message.success(t('common.deleted')))
                .catch((e) => message.error((e as Error).message))
            }
          >
            <Button size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const handleSave = async () => {
    try {
      const v = await form.validateFields();
      if (editing) {
        await updateMut.mutateAsync({ id: editing.id, input: v });
        message.success(t('common.updated'));
      } else {
        await upsertMut.mutateAsync(v);
        message.success(t('common.created'));
      }
      setEditing(null);
      setCreating(false);
      form.resetFields();
    } catch (e) {
      const msg = (e as Error).message;
      if (msg) message.error(msg);
    }
  };

  return (
    <Card
      size="small"
      title={t('product.kpi.units.title')}
      extra={
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => {
            setCreating(true);
            setEditing(null);
            form.resetFields();
          }}
        >
          {t('product.kpi.newUnit')}
        </Button>
      }
    >
      <Table<IndicatorUnit>
        rowKey="id"
        loading={isLoading}
        columns={columns}
        dataSource={items}
        size="small"
        pagination={false}
      />
      <Modal
        title={editing ? t('product.kpi.editUnitTitle', { id: editing.id }) : t('product.kpi.newUnit')}
        open={Boolean(editing) || creating}
        onOk={() => void handleSave()}
        onCancel={() => {
          setEditing(null);
          setCreating(false);
          form.resetFields();
        }}
        confirmLoading={upsertMut.isPending || updateMut.isPending}
        destroyOnHidden
      >
        <Form form={form} layout="vertical">
          <Form.Item name="id" label="ID" rules={[{ required: true }]}>
            <Input disabled={Boolean(editing)} placeholder={t('product.kpi.units.codePh')} />
          </Form.Item>
          <Form.Item name="enName" label={t('common.enName')} rules={[{ required: true }]}>
            <Input placeholder={t('product.kpi.units.enPh')} />
          </Form.Item>
          <Form.Item name="cnName" label={t('common.cnName')} rules={[{ required: true }]}>
            <Input placeholder={t('product.kpi.units.cnPh')} />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}
