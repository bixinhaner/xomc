import { useState } from 'react';
import { Card, Table, Tag, Button, Space, Modal, Form, Input, Switch, message, Popconfirm } from 'antd';
import { EditOutlined, DeleteOutlined } from '@ant-design/icons';
import {
  useParamModelList,
  useUpdateParamModel,
  useDeleteParamModel,
} from '@core/hooks/api/useParamModels';
import type { ParamModel, UpdateParamModelInput } from '@core/types/paramModel';

interface Props {
  selectedName?: string;
  onSelect: (name: string) => void;
}

export default function ModelsTab({ selectedName, onSelect }: Props) {
  const { data, isLoading } = useParamModelList();
  const updateMut = useUpdateParamModel();
  const deleteMut = useDeleteParamModel();

  const [editing, setEditing] = useState<ParamModel | null>(null);
  const [form] = Form.useForm<UpdateParamModelInput>();

  const items = data?.items || [];

  const columns = [
    {
      title: '名称',
      dataIndex: 'name',
      width: 240,
      render: (v: string, row: ParamModel) => (
        <Button
          type="link"
          size="small"
          onClick={() => onSelect(row.name)}
          style={{ padding: 0, fontWeight: row.name === selectedName ? 600 : 400 }}
        >
          {v}
        </Button>
      ),
    },
    { title: '总条目', dataIndex: 'totalEntries', width: 90 },
    { title: '对象数', dataIndex: 'totalObjects', width: 90 },
    { title: '参数数', dataIndex: 'totalParams', width: 90 },
    { title: '加载源', dataIndex: 'loadedFrom', width: 240 },
    {
      title: '激活',
      dataIndex: 'isActive',
      width: 80,
      render: (v: boolean) => (v ? <Tag color="success">激活</Tag> : <Tag>禁用</Tag>),
    },
    { title: '描述', dataIndex: 'description', ellipsis: true },
    {
      title: '操作',
      width: 160,
      render: (_: unknown, row: ParamModel) => (
        <Space>
          <Button
            size="small"
            icon={<EditOutlined />}
            onClick={() => {
              setEditing(row);
              form.setFieldsValue({ description: row.description, isActive: row.isActive });
            }}
          />
          <Popconfirm
            title={`确认删除参数模型「${row.name}」？关联映射会一并删除`}
            onConfirm={() =>
              deleteMut
                .mutateAsync(row.name)
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
    if (!editing) return;
    try {
      const v = await form.validateFields();
      await updateMut.mutateAsync({ name: editing.name, input: v });
      message.success('已保存');
      setEditing(null);
    } catch (e) {
      const msg = (e as Error).message;
      if (msg) message.error(msg);
    }
  };

  return (
    <Card size="small" title="参数模型清单 / Param Models">
      <Table<ParamModel>
        rowKey="id"
        loading={isLoading}
        columns={columns}
        dataSource={items}
        size="small"
        pagination={{ pageSize: 20 }}
      />
      <Modal
        title={`编辑参数模型：${editing?.name}`}
        open={Boolean(editing)}
        onOk={() => void handleSave()}
        onCancel={() => setEditing(null)}
        confirmLoading={updateMut.isPending}
        destroyOnClose
      >
        <Form form={form} layout="vertical">
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={3} />
          </Form.Item>
          <Form.Item name="isActive" label="激活" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}
