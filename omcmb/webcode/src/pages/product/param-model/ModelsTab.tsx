import { useMemo, useState } from 'react';
import { Card, Table, Tag, Button, Space, Modal, Form, Input, Switch, message, Popconfirm, Tooltip } from 'antd';
import { EditOutlined, DeleteOutlined } from '@ant-design/icons';
import {
  useParamModelList,
  useUpdateParamModel,
  useDeleteParamModel,
} from '@core/hooks/api/useParamModels';
import type { ParamModel, ParamModelSource, UpdateParamModelInput } from '@core/types/paramModel';

interface Props {
  selectedName?: string;
  onSelect: (name: string) => void;
  /** 2026-05-28: 由 param-model index 顶部统一 toolbar 提供的关键字过滤
   *  (取消 Tabs 后,搜索框上提到容器外,client-side 过滤 name / description)。 */
  keyword?: string;
}

export default function ModelsTab({ selectedName, onSelect, keyword }: Props) {
  const { data, isLoading } = useParamModelList();
  const updateMut = useUpdateParamModel();
  const deleteMut = useDeleteParamModel();

  const [editing, setEditing] = useState<ParamModel | null>(null);
  const [form] = Form.useForm<UpdateParamModelInput>();

  const items = useMemo(() => {
    const raw = data?.items || [];
    const k = keyword?.trim().toLowerCase();
    if (!k) return raw;
    return raw.filter((m) =>
      (m.name + ' ' + (m.description || '')).toLowerCase().includes(k),
    );
  }, [data, keyword]);

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
    {
      // T-0178: 来源列 — 后端 source.go::ClassifySource 派生,前端只渲染
      title: '来源',
      dataIndex: 'source',
      width: 90,
      filters: [
        { text: '内置', value: 'builtin' as ParamModelSource },
        { text: '自定义', value: 'custom' as ParamModelSource },
        { text: '未知', value: 'unknown' as ParamModelSource },
      ],
      onFilter: (val: boolean | React.Key, row: ParamModel) => row.source === val,
      render: (s: ParamModelSource | undefined, row: ParamModel) => {
        // 防御:老缓存可能无 source 字段,fallback unknown
        const src = s ?? 'unknown';
        const tag =
          src === 'custom' ? (
            <Tag color="blue">自定义</Tag>
          ) : src === 'builtin' ? (
            <Tag>内置</Tag>
          ) : (
            <Tag color="warning">未知</Tag>
          );
        return <Tooltip title={row.loadedFrom}>{tag}</Tooltip>;
      },
    },
    { title: '加载源', dataIndex: 'loadedFrom', width: 260, ellipsis: true },
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
          {/* T-0178: 仅 deletable=true(custom)行可点击删除;内置/未知置灰 + Tooltip */}
          {row.deletable ? (
            <Popconfirm
              title={`确认删除自定义参数模型「${row.name}」?`}
              description={
                <div style={{ maxWidth: 320 }}>
                  · 物理文件将 rename 为 <code>.deleted.&lt;ts&gt;</code> 备份
                  <br />· 关联 <code>param_mappings</code> 级联删除
                  <br />· 若存在同名内置 XML,删除后将自动回退到内置版本
                </div>
              }
              okButtonProps={{ danger: true }}
              okText="确认删除"
              onConfirm={() =>
                deleteMut
                  .mutateAsync(row.name)
                  .then(() => message.success('已删除'))
                  .catch((e) => message.error((e as Error).message))
              }
            >
              <Button size="small" danger icon={<DeleteOutlined />} />
            </Popconfirm>
          ) : (
            <Tooltip
              title={
                <div style={{ maxWidth: 240 }}>
                  内置参数模型不可在线删除。如需移除,请在下一版镜像的{' '}
                  <code>data/param-mappings/</code> 中删掉对应 XML,重新构建并发布。
                </div>
              }
              placement="topRight"
            >
              <Button
                size="small"
                icon={<DeleteOutlined />}
                disabled
                aria-label="builtin XML not deletable"
              />
            </Tooltip>
          )}
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
    <Card size="small">
      <Table<ParamModel>
        rowKey="id"
        loading={isLoading}
        columns={columns}
        dataSource={items}
        size="small"
        pagination={{ pageSize: 20, showTotal: (t) => `共 ${t} 条` }}
      />
      <Modal
        title={`编辑参数模型：${editing?.name}`}
        open={Boolean(editing)}
        onOk={() => void handleSave()}
        onCancel={() => setEditing(null)}
        confirmLoading={updateMut.isPending}
        destroyOnHidden
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
