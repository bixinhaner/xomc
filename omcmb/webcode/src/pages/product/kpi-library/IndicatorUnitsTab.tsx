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

export default function IndicatorUnitsTab() {
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
    { title: '英文名', dataIndex: 'enName', width: 240 },
    { title: '中文名', dataIndex: 'cnName' },
    {
      title: '操作',
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
            title={`确认删除单位「${row.id}」？`}
            onConfirm={() =>
              deleteMut
                .mutateAsync(row.id)
                .then(() => message.success('已删除'))
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
        message.success('已更新');
      } else {
        await upsertMut.mutateAsync(v);
        message.success('已创建');
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
      title="单位定义 / Indicator Units"
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
          新增单位
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
        title={editing ? `编辑单位：${editing.id}` : '新增单位'}
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
            <Input disabled={Boolean(editing)} placeholder="如 percent / count / mbps" />
          </Form.Item>
          <Form.Item name="enName" label="英文名" rules={[{ required: true }]}>
            <Input placeholder="如 % / Count / Mbps" />
          </Form.Item>
          <Form.Item name="cnName" label="中文名" rules={[{ required: true }]}>
            <Input placeholder="如 百分比 / 次数 / 兆比特每秒" />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}
