/**
 * StandardParamsPage — 标准参数树 / Standard Params 独立菜单页面。
 *
 * 2026-05-28 用户决策:
 *   - 从 product/param-model 的 Tab 中拆出,作为产品中心一级菜单项独立路由
 *   - 路由 /product/standard-params,菜单 product:standard-params,
 *     RBAC 通过 menus + role_menus 控制(默认绑定 admin / super_admin)
 *
 * 内容与原 StandardParamsTab 等价(列表 + 搜索 + 类型过滤 + 新增/编辑/删除)。
 */
import { useState } from 'react';
import {
  Card,
  Table,
  Input,
  Select,
  Space,
  Button,
  Modal,
  Form,
  Popconfirm,
  message,
} from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import {
  useStandardParams,
  useUpsertStandard,
  useDeleteStandard,
} from '@core/hooks/api/useParamModels';
import type { StandardParam, UpsertStandardInput } from '@core/types/paramModel';
import { makeSeqColumn } from '@/components/Table/seqColumn';
import { useT } from '@/hooks/useT';

export default function StandardParamsPage() {
  const t = useT();
  const ENTRY_OPTIONS = [
    { label: t('common.all'), value: '' },
    { label: 'parameter', value: 'parameter' },
    { label: 'object', value: 'object' },
  ];
  const [keyword, setKeyword] = useState('');
  const [entryType, setEntryType] = useState('');
  const { data, isLoading } = useStandardParams({
    keyword: keyword || undefined,
    entryType: entryType || undefined,
  });
  const upsertMut = useUpsertStandard();
  const deleteMut = useDeleteStandard();

  const [editing, setEditing] = useState<StandardParam | null>(null);
  const [creating, setCreating] = useState(false);
  const [form] = Form.useForm<UpsertStandardInput>();

  const items = data?.items || [];

  const columns = [
    { title: t('product.standardParams.col.standardPath'), dataIndex: 'standardPath', ellipsis: true },
    { title: t('product.standardParams.col.entryType'), dataIndex: 'entryType', width: 100 },
    { title: t('product.standardParams.col.access'), dataIndex: 'access', width: 100 },
    { title: t('product.standardParams.col.dataType'), dataIndex: 'dataType', width: 100 },
    { title: t('product.standardParams.col.changeApplies'), dataIndex: 'changeApplies', width: 130 },
    { title: t('product.standardParams.col.min'), dataIndex: 'minValue', width: 80 },
    { title: t('product.standardParams.col.max'), dataIndex: 'maxValue', width: 80 },
    {
      title: t('common.action'),
      width: 120,
      render: (_: unknown, row: StandardParam) => (
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
            title={t('common.confirmDeleteName', { name: row.standardPath })}
            onConfirm={() =>
              deleteMut
                .mutateAsync(row.standardPath)
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
      await upsertMut.mutateAsync({ input: v, path: editing?.standardPath });
      message.success(editing ? t('common.saved') : t('common.created'));
      setEditing(null);
      setCreating(false);
      form.resetFields();
    } catch (e) {
      const msg = (e as Error).message;
      if (msg) message.error(msg);
    }
  };

  return (
    <div style={{ padding: 16 }}>
      <Card size="small">
        {/* 2026-05-28 用户决策:筛选栏靠左,「新增」按钮靠右 */}
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            gap: 12,
            flexWrap: 'wrap',
            marginBottom: 12,
          }}
        >
          <Space>
            <Input.Search
              placeholder={t('product.standardParams.searchPh')}
              allowClear
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
              style={{ width: 240 }}
            />
            <Select
              value={entryType}
              onChange={(v) => setEntryType(v)}
              options={ENTRY_OPTIONS}
              style={{ width: 120 }}
            />
          </Space>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => {
              setCreating(true);
              setEditing(null);
              form.resetFields();
              form.setFieldsValue({
                entryType: 'parameter',
                access: 'readWrite',
                dataType: 'string',
                changeApplies: 'reload',
              });
            }}
          >
            {t('common.create')}
          </Button>
        </div>
        <Table<StandardParam>
          rowKey="standardPath"
          loading={isLoading}
          columns={[makeSeqColumn<StandardParam>({ title: t('table.rowNumber'), dataSource: items }), ...columns]}
          dataSource={items}
          size="small"
          pagination={{ pageSize: 50, showSizeChanger: true, showTotal: (total) => t('common.totalCount', { count: total }) }}
        />
        <Modal
          title={editing ? t('product.standardParams.editTitle') : t('product.standardParams.newTitle')}
          open={Boolean(editing) || creating}
          onOk={() => void handleSave()}
          onCancel={() => {
            setEditing(null);
            setCreating(false);
            form.resetFields();
          }}
          confirmLoading={upsertMut.isPending}
          width={620}
          destroyOnHidden
        >
          <Form form={form} layout="vertical">
            <Form.Item
              name="standardPath"
              label="standard_path"
              rules={[{ required: true, message: t('common.required') }]}
            >
              <Input disabled={Boolean(editing)} />
            </Form.Item>
            <Space wrap>
              <Form.Item name="entryType" label="entry_type" rules={[{ required: true }]}>
                <Select
                  options={[
                    { label: 'parameter', value: 'parameter' },
                    { label: 'object', value: 'object' },
                  ]}
                  style={{ width: 140 }}
                />
              </Form.Item>
              <Form.Item name="access" label="access" rules={[{ required: true }]}>
                <Select
                  options={[
                    { label: 'readWrite', value: 'readWrite' },
                    { label: 'readOnly', value: 'readOnly' },
                  ]}
                  style={{ width: 140 }}
                />
              </Form.Item>
              <Form.Item name="dataType" label="data_type" rules={[{ required: true }]}>
                <Input style={{ width: 140 }} />
              </Form.Item>
              <Form.Item name="changeApplies" label="change_applies">
                <Input style={{ width: 140 }} />
              </Form.Item>
              <Form.Item name="minValue" label="min">
                <Input style={{ width: 140 }} />
              </Form.Item>
              <Form.Item name="maxValue" label="max">
                <Input style={{ width: 140 }} />
              </Form.Item>
            </Space>
          </Form>
        </Modal>
      </Card>
    </div>
  );
}
