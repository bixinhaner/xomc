import { useMemo, useState } from 'react';
import {
  Card,
  Table,
  Tag,
  Button,
  Space,
  Select,
  Input,
  Modal,
  Form,
  Switch,
  message,
  Popconfirm,
  Tooltip,
  Typography,
  Empty,
} from 'antd';
import {
  ArrowLeftOutlined,
  ExclamationCircleFilled,
  PlusOutlined,
  DeleteOutlined,
  EditOutlined,
} from '@ant-design/icons';
import {
  useParamMappings,
  useCreateMapping,
  useUpdateMapping,
  useDeleteMapping,
} from '@core/hooks/api/useParamModels';
import type { ParamMapping, CreateMappingInput, UpdateMappingInput } from '@core/types/paramModel';

const { Text } = Typography;

interface Props {
  selectedName?: string;
  // 2026-05-29 用户决策:返回 button 下沉到本组件,与 search/filter/新增映射 合并为单行
  // (代替旧的"顶部 toolbar + Card 标题"双行结构,且 Card 不再带"默认映射 / Mappings — X" 标题)。
  onBack?: () => void;
}

const STORABLE_OPTIONS = [
  { label: '全部', value: 'all' },
  { label: '仅可存储', value: 'true' },
  { label: '仅不可存储', value: 'false' },
];

const ENTRY_OPTIONS = [
  { label: 'parameter', value: 'parameter' },
  { label: 'object', value: 'object' },
];

const ACCESS_OPTIONS = [
  { label: 'readWrite', value: 'readWrite' },
  { label: 'readOnly', value: 'readOnly' },
  { label: 'writeOnly', value: 'writeOnly' },
];

function countPlaceholder(p: string): number {
  const m = p.match(/\{i\}/g);
  return m ? m.length : 0;
}

export default function MappingsTab({ selectedName, onBack }: Props) {
  const { data, isLoading } = useParamMappings(selectedName);
  const createMut = useCreateMapping();
  const updateMut = useUpdateMapping();
  const deleteMut = useDeleteMapping();

  const [storableFilter, setStorableFilter] = useState<'all' | 'true' | 'false'>('all');
  const [keyword, setKeyword] = useState('');
  const [editing, setEditing] = useState<ParamMapping | null>(null);
  const [creating, setCreating] = useState(false);
  const [form] = Form.useForm<CreateMappingInput | UpdateMappingInput>();

  const filtered = useMemo(() => {
    let list = data?.items || [];
    if (storableFilter === 'true') list = list.filter((m) => m.isStorable);
    if (storableFilter === 'false') list = list.filter((m) => !m.isStorable);
    if (keyword.trim()) {
      const k = keyword.trim().toLowerCase();
      list = list.filter((m) => (m.standardPath + ' ' + m.privatePath).toLowerCase().includes(k));
    }
    return list;
  }, [data, storableFilter, keyword]);

  const columns = [
    {
      title: 'standard_path',
      dataIndex: 'standardPath',
      ellipsis: true,
      render: (v: string, row: ParamMapping) => {
        const stdCount = countPlaceholder(v);
        const privCount = countPlaceholder(row.privatePath);
        const mismatch = stdCount !== privCount;
        return (
          <Space>
            {mismatch && (
              <Tooltip title={`占位符 {i} 数量不匹配：standard=${stdCount} private=${privCount}`}>
                <ExclamationCircleFilled style={{ color: '#ff4d4f' }} />
              </Tooltip>
            )}
            <span>{v}</span>
          </Space>
        );
      },
    },
    { title: 'private_path', dataIndex: 'privatePath', ellipsis: true },
    { title: 'entry', dataIndex: 'entryType', width: 90 },
    { title: 'access', dataIndex: 'access', width: 100 },
    { title: 'data_type', dataIndex: 'dataType', width: 100 },
    {
      title: 'storable',
      dataIndex: 'isStorable',
      width: 90,
      render: (v: boolean) => (v ? <Tag color="success">是</Tag> : <Tag>否</Tag>),
    },
    { title: 'sw', dataIndex: 'softwareVersion', width: 80 },
    {
      title: '操作',
      width: 120,
      render: (_: unknown, row: ParamMapping) => (
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
            title="确认删除该映射？"
            onConfirm={() =>
              selectedName &&
              deleteMut
                .mutateAsync({ name: selectedName, id: row.id })
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
    if (!selectedName) return;
    try {
      const v = await form.validateFields();
      const input: CreateMappingInput = {
        standardPath: v.standardPath as string,
        privatePath: v.privatePath as string,
        entryType: (v.entryType as string) || 'parameter',
        access: (v.access as string) || 'readWrite',
        dataType: (v.dataType as string) || 'string',
        changeApplies: v.changeApplies,
        minValue: v.minValue,
        maxValue: v.maxValue,
        isStorable: v.isStorable ?? true,
        isActive: v.isActive ?? true,
        softwareVersion: v.softwareVersion,
      };
      if (editing) {
        await updateMut.mutateAsync({ name: selectedName, id: editing.id, input });
        message.success('已保存');
      } else {
        await createMut.mutateAsync({ name: selectedName, input });
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

  if (!selectedName) {
    return (
      <Card size="small">
        <Empty description="请先在上方「参数模型清单」中选择一个模型" />
      </Card>
    );
  }

  return (
    <Card
      size="small"
      // 2026-05-29 用户决策:旧"默认映射 / Mappings — X"标题删除;改用 返回 + 模型名
      // 占左,合并原顶部 toolbar 行 + 原 Card extra 筛选行为单行布局。
      title={
        <Space>
          {onBack && (
            <Button icon={<ArrowLeftOutlined />} onClick={onBack} size="small">
              返回
            </Button>
          )}
          <Text strong>{selectedName}</Text>
        </Space>
      }
      extra={
        <Space>
          <Input.Search
            placeholder="搜索 path"
            allowClear
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            style={{ width: 220 }}
          />
          <Select
            value={storableFilter}
            onChange={(v) => setStorableFilter(v)}
            options={STORABLE_OPTIONS}
            style={{ width: 130 }}
          />
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => {
              setCreating(true);
              setEditing(null);
              form.resetFields();
              form.setFieldsValue({ entryType: 'parameter', access: 'readWrite', isStorable: true, isActive: true });
            }}
          >
            新增映射
          </Button>
        </Space>
      }
    >
      <Table<ParamMapping>
        rowKey="id"
        loading={isLoading}
        columns={columns}
        dataSource={filtered}
        size="small"
        pagination={{ pageSize: 50, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
      />
      <Modal
        title={editing ? '编辑映射' : '新增映射'}
        open={Boolean(editing) || creating}
        onOk={() => void handleSave()}
        onCancel={() => {
          setEditing(null);
          setCreating(false);
          form.resetFields();
        }}
        confirmLoading={createMut.isPending || updateMut.isPending}
        width={680}
        destroyOnHidden
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="standardPath"
            label="standard_path"
            rules={[{ required: true, message: '必填' }]}
            extra="支持 {i} 占位符；与 private_path 的 {i} 数量必须一致"
          >
            <Input placeholder="Device.Cellular.AccessPoint.{i}.PLMN" />
          </Form.Item>
          <Form.Item
            name="privatePath"
            label="private_path"
            rules={[{ required: true, message: '必填' }]}
          >
            <Input placeholder="X_VENDOR_AccessPoint.{i}.PLMNID" />
          </Form.Item>
          <Space style={{ width: '100%' }} size="middle" wrap>
            <Form.Item name="entryType" label="entry_type">
              <Select options={ENTRY_OPTIONS} style={{ width: 140 }} />
            </Form.Item>
            <Form.Item name="access" label="access">
              <Select options={ACCESS_OPTIONS} style={{ width: 140 }} />
            </Form.Item>
            <Form.Item name="dataType" label="data_type">
              <Input style={{ width: 140 }} placeholder="string / int / bool" />
            </Form.Item>
            <Form.Item name="changeApplies" label="change_applies">
              <Input style={{ width: 140 }} placeholder="reload / immediate" />
            </Form.Item>
            <Form.Item name="softwareVersion" label="sw_version">
              <Input style={{ width: 120 }} />
            </Form.Item>
          </Space>
          <Space wrap>
            <Form.Item name="minValue" label="min_value">
              <Input style={{ width: 140 }} />
            </Form.Item>
            <Form.Item name="maxValue" label="max_value">
              <Input style={{ width: 140 }} />
            </Form.Item>
            <Form.Item name="isStorable" label="storable" valuePropName="checked">
              <Switch />
            </Form.Item>
            <Form.Item name="isActive" label="激活" valuePropName="checked">
              <Switch />
            </Form.Item>
          </Space>
        </Form>
      </Modal>
    </Card>
  );
}
