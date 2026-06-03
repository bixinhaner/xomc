import { useMemo, useState } from 'react';
import {
  Card,
  Table,
  Button,
  Space,
  Select,
  Input,
  Modal,
  Form,
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
import { makeSeqColumn } from '@/components/Table/seqColumn';
import { useT } from '@/hooks/useT';

const { Text } = Typography;

interface Props {
  selectedName?: string;
  // 2026-05-29 用户决策:返回 button 下沉到本组件,与 search/filter/新增映射 合并为单行
  // (代替旧的"顶部 toolbar + Card 标题"双行结构,且 Card 不再带"默认映射 / Mappings — X" 标题)。
  onBack?: () => void;
}

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
  const t = useT();
  const { data, isLoading } = useParamMappings(selectedName);
  const createMut = useCreateMapping();
  const updateMut = useUpdateMapping();
  const deleteMut = useDeleteMapping();

  // 条目类型筛选(对齐 standard-params 页面):'' = 全部
  const [entryFilter, setEntryFilter] = useState('');
  const [keyword, setKeyword] = useState('');
  const [editing, setEditing] = useState<ParamMapping | null>(null);
  const [creating, setCreating] = useState(false);
  const [form] = Form.useForm<CreateMappingInput | UpdateMappingInput>();

  // 条目类型下拉(对齐 standard-params 的 ENTRY_OPTIONS:全部 / parameter / object)
  const ENTRY_FILTER_OPTIONS = [
    { label: t('common.all'), value: '' },
    { label: 'parameter', value: 'parameter' },
    { label: 'object', value: 'object' },
  ];

  const filtered = useMemo(() => {
    let list = data?.items || [];
    if (entryFilter) list = list.filter((m) => m.entryType === entryFilter);
    if (keyword.trim()) {
      const k = keyword.trim().toLowerCase();
      list = list.filter((m) => (m.standardPath + ' ' + m.privatePath).toLowerCase().includes(k));
    }
    return list;
  }, [data, entryFilter, keyword]);

  const columns = [
    {
      title: t('product.paramModel.mappings.col.standardPath'),
      dataIndex: 'standardPath',
      ellipsis: true,
      render: (v: string, row: ParamMapping) => {
        const stdCount = countPlaceholder(v);
        const privCount = countPlaceholder(row.privatePath);
        const mismatch = stdCount !== privCount;
        return (
          <Space>
            {mismatch && (
              <Tooltip title={t('product.paramModel.mappings.placeholderMismatch', { std: stdCount, priv: privCount })}>
                <ExclamationCircleFilled style={{ color: '#ff4d4f' }} />
              </Tooltip>
            )}
            <span>{v}</span>
          </Space>
        );
      },
    },
    { title: t('product.paramModel.mappings.col.privatePath'), dataIndex: 'privatePath', ellipsis: true },
    { title: t('product.paramModel.mappings.col.entryType'), dataIndex: 'entryType', width: 90 },
    { title: t('product.paramModel.mappings.col.access'), dataIndex: 'access', width: 110 },
    { title: t('product.paramModel.mappings.col.dataType'), dataIndex: 'dataType', width: 100 },
    { title: t('product.paramModel.mappings.col.changeApplies'), dataIndex: 'changeApplies', width: 110 },
    { title: t('product.paramModel.mappings.col.min'), dataIndex: 'minValue', width: 80 },
    { title: t('product.paramModel.mappings.col.max'), dataIndex: 'maxValue', width: 80 },
    {
      title: t('common.action'),
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
            title={t('product.paramModel.mappingDelTitle')}
            onConfirm={() =>
              selectedName &&
              deleteMut
                .mutateAsync({ name: selectedName, id: row.id })
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
        message.success(t('common.saved'));
      } else {
        await createMut.mutateAsync({ name: selectedName, input });
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

  if (!selectedName) {
    return (
      <Card size="small">
        <Empty description={t('product.paramModel.selectModelHint')} />
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
              {t('common.back')}
            </Button>
          )}
          <Text strong>{selectedName}</Text>
        </Space>
      }
      extra={
        <Space>
          <Input.Search
            placeholder={t('product.paramModel.pathSearchPh')}
            allowClear
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            style={{ width: 220 }}
          />
          <Select
            value={entryFilter}
            onChange={(v) => setEntryFilter(v)}
            options={ENTRY_FILTER_OPTIONS}
            style={{ width: 120 }}
          />
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => {
              setCreating(true);
              setEditing(null);
              form.resetFields();
              form.setFieldsValue({ entryType: 'parameter', access: 'readWrite', dataType: 'string', changeApplies: 'reload' });
            }}
          >
            {t('product.paramModel.mappings.newBtn')}
          </Button>
        </Space>
      }
    >
      <Table<ParamMapping>
        rowKey="id"
        loading={isLoading}
        columns={[makeSeqColumn<ParamMapping>({ title: t('table.rowNumber'), dataSource: filtered }), ...columns]}
        dataSource={filtered}
        size="small"
        pagination={{ pageSize: 50, showSizeChanger: true, showTotal: (total) => t('common.totalCount', { count: total }) }}
      />
      <Modal
        title={editing ? t('product.paramModel.mappings.editTitle') : t('product.paramModel.mappings.newTitle')}
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
            label={t('product.paramModel.mappings.col.standardPath')}
            rules={[{ required: true, message: t('common.required') }]}
            extra={t('product.paramModel.mappings.privatePathExtra')}
          >
            <Input placeholder="Device.Cellular.AccessPoint.{i}.PLMN" />
          </Form.Item>
          <Form.Item
            name="privatePath"
            label={t('product.paramModel.mappings.col.privatePath')}
            rules={[{ required: true, message: t('common.required') }]}
          >
            <Input placeholder="X_VENDOR_AccessPoint.{i}.PLMNID" />
          </Form.Item>
          <Space style={{ width: '100%' }} size="middle" wrap>
            <Form.Item name="entryType" label={t('product.paramModel.mappings.col.entryType')} rules={[{ required: true }]}>
              <Select options={ENTRY_OPTIONS} style={{ width: 140 }} />
            </Form.Item>
            <Form.Item name="access" label={t('product.paramModel.mappings.col.access')} rules={[{ required: true }]}>
              <Select options={ACCESS_OPTIONS} style={{ width: 140 }} />
            </Form.Item>
            <Form.Item name="dataType" label={t('product.paramModel.mappings.col.dataType')} rules={[{ required: true }]}>
              <Input style={{ width: 140 }} placeholder="string / int / bool" />
            </Form.Item>
          </Space>
          <Space wrap>
            <Form.Item name="changeApplies" label={t('product.paramModel.mappings.col.changeApplies')}>
              <Input style={{ width: 140 }} placeholder="reload / immediate" />
            </Form.Item>
            <Form.Item name="minValue" label={t('product.paramModel.mappings.col.min')}>
              <Input style={{ width: 140 }} />
            </Form.Item>
            <Form.Item name="maxValue" label={t('product.paramModel.mappings.col.max')}>
              <Input style={{ width: 140 }} />
            </Form.Item>
          </Space>
        </Form>
      </Modal>
    </Card>
  );
}
