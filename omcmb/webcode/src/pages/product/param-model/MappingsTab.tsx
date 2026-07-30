import { useMemo, useState } from 'react';
import {
  Card,
  Table,
  Button,
  Space,
  Select,
  Input,
  InputNumber,
  Modal,
  Form,
  message,
  Popconfirm,
  Tooltip,
  Tag,
  Empty,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  ArrowLeftOutlined,
  ExclamationCircleFilled,
  PlusOutlined,
  DeleteOutlined,
  EditOutlined,
} from '@ant-design/icons';
import {
  useParamMappings,
  useStandardParams,
  useCreateMapping,
  useUpdateMapping,
  useDeleteMapping,
} from '@core/hooks/api/useParamModels';
import {
  STANDARD_DATA_TYPES,
  STANDARD_CHANGE_APPLIES,
  dataTypeRangeKind,
  isUnsignedDataType,
} from '@core/types/paramModel';
import type { ParamMapping, CreateMappingInput, UpdateMappingInput } from '@core/types/paramModel';
import { makeSeqColumn } from '@/components/Table/seqColumn';
import {
  useResizableColumns,
  ResizableColumnsStyle,
} from '@/components/Table/resizableColumns';
import SearchInput from '@/components/SearchInput';
import { useT } from '@/hooks/useT';

// 长 TR069 PATH 单元格（#214）：列宽放不下时——悬浮 tooltip 看完整路径 + 一键复制；
// 配合可拖拽列宽（useResizableColumns），三管齐下确保完整路径可见/可取。
function PathCell({ value, prefix }: { value: string; prefix?: React.ReactNode }) {
  const t = useT();
  if (!value) return <>-</>;
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 4, minWidth: 0 }}>
      {prefix}
      <Tooltip title={value} placement="topLeft">
        <span
          style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', flex: 1, minWidth: 0 }}
        >
          {value}
        </span>
      </Tooltip>
      <Typography.Text
        copyable={{ text: value, tooltips: [t('common.copy'), t('table.copied')] }}
        style={{ flex: 'none' }}
      />
    </div>
  );
}

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
  const [editing, setEditing] = useState<ParamMapping | null>(null);

  // 当前模型已存在的标准 PATH 集合——用于新增态去重(同一标准 PATH 不能重复添加)。
  const usedStdPaths = useMemo(
    () => new Set((data?.items ?? []).map((m) => m.standardPath)),
    [data],
  );

  // 标准 PATH 下拉数据源(与 product/standard-params 同一接口,全量 <500 条,客户端搜索)。
  const { data: stdData } = useStandardParams();
  // 新增态把已添加的标准 PATH 置灰并标注「已添加」,从源头阻断重复;编辑态整个 Select 已 disabled,不受影响。
  const stdOptions = useMemo(
    () =>
      (stdData?.items ?? []).map((s) => {
        const used = !editing && usedStdPaths.has(s.standardPath);
        return {
          label: used ? `${s.standardPath}（${t('product.paramModel.mappings.stdAdded')}）` : s.standardPath,
          value: s.standardPath,
          disabled: used,
        };
      }),
    [stdData, usedStdPaths, editing, t],
  );
  // 选中标准 PATH 时,用其元属性回填表单(entryType/access/dataType/changeApplies/min/max)。
  const onPickStandard = (path: string) => {
    const sp = (stdData?.items ?? []).find((s) => s.standardPath === path);
    if (!sp) return;
    form.setFieldsValue({
      entryType: sp.entryType || 'parameter',
      access: sp.access || 'readWrite',
      dataType: sp.dataType || 'string',
      changeApplies: sp.changeApplies || undefined,
      minValue: sp.minValue,
      maxValue: sp.maxValue,
    });
  };

  // 条目类型筛选(对齐 standard-params 页面):'' = 全部
  const [entryFilter, setEntryFilter] = useState('');
  // 手动搜索:仅在 onSearch(回车/点击搜索)时应用 keyword。
  const [keyword, setKeyword] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(50);

  // 过滤条件变化时回到第一页——渲染期重置,避免 set-state-in-effect。
  const filterKey = `${keyword}|${entryFilter}|${selectedName ?? ''}`;
  const [prevFilterKey, setPrevFilterKey] = useState(filterKey);
  if (filterKey !== prevFilterKey) {
    setPrevFilterKey(filterKey);
    setPage(1);
  }
  const [creating, setCreating] = useState(false);
  const [form] = Form.useForm<CreateMappingInput | UpdateMappingInput>();

  // 当前表单内的 dataType / changeApplies —— 用于动态 label 与历史值兼容（对齐 standard-params 页面）。
  const watchedDataType = Form.useWatch('dataType', form);
  const watchedChangeApplies = Form.useWatch('changeApplies', form);

  // dataType 下拉：枚举集 + 若当前编辑/回填值不在枚举内则并入（历史遗留如 STRING/bool 不被静默清空）。
  const dataTypeOptions = (() => {
    const opts: { label: string; value: string }[] = STANDARD_DATA_TYPES.map((v) => ({
      label: v,
      value: v,
    }));
    if (watchedDataType && !STANDARD_DATA_TYPES.includes(watchedDataType as never)) {
      opts.push({ label: watchedDataType, value: watchedDataType });
    }
    return opts;
  })();

  // changeApplies 下拉：枚举集（显示英文枚举值 Immediate/OnReboot）+ 历史遗留值（如 reload/immediate）并入。
  const changeAppliesOptions = (() => {
    const opts: { label: string; value: string }[] = STANDARD_CHANGE_APPLIES.map((v) => ({
      label: v,
      value: v,
    }));
    if (
      watchedChangeApplies &&
      !STANDARD_CHANGE_APPLIES.includes(watchedChangeApplies as never)
    ) {
      opts.push({ label: watchedChangeApplies, value: watchedChangeApplies });
    }
    return opts;
  })();

  // min/max 按 dataType 分三档：string→长度，int/unsignedInt→数值，boolean/dateTime→无范围（禁用）。
  const rangeKind = dataTypeRangeKind(watchedDataType);
  const rangeNone = rangeKind === 'none';
  const minLabel =
    rangeKind === 'length'
      ? t('product.standardParams.col.minLength')
      : t('product.standardParams.col.min');
  const maxLabel =
    rangeKind === 'length'
      ? t('product.standardParams.col.maxLength')
      : t('product.standardParams.col.max');
  // unsignedInt 下界 0；无范围类型禁用并清空。
  const minBound = isUnsignedDataType(watchedDataType) ? 0 : undefined;
  const intPlaceholder = rangeNone
    ? t('product.standardParams.noRange')
    : t('product.standardParams.intPlaceholder');

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

  const columns: ColumnsType<ParamMapping> = [
    {
      title: t('product.paramModel.mappings.col.standardPath'),
      dataIndex: 'standardPath',
      key: 'standardPath',
      width: 280, // #214：长 TR069 路径——给默认宽 + 可拖宽（useResizableColumns）+ PathCell tooltip/复制
      render: (v: string, row: ParamMapping) => {
        const stdCount = countPlaceholder(v);
        const privCount = countPlaceholder(row.privatePath);
        const mismatch = privCount > stdCount;
        return (
          <PathCell
            value={v}
            prefix={
              mismatch ? (
                <Tooltip title={t('product.paramModel.mappings.placeholderMismatch', { std: stdCount, priv: privCount })}>
                  <ExclamationCircleFilled style={{ color: '#ff4d4f', flex: 'none' }} />
                </Tooltip>
              ) : undefined
            }
          />
        );
      },
    },
    {
      title: t('product.paramModel.mappings.col.privatePath'),
      dataIndex: 'privatePath',
      key: 'privatePath',
      width: 280, // #214：同上
      render: (v: string) => <PathCell value={v} />,
    },
    { title: t('product.paramModel.mappings.col.entryType'), dataIndex: 'entryType', width: 90 },
    { title: t('product.paramModel.mappings.col.access'), dataIndex: 'access', width: 110 },
    { title: t('product.paramModel.mappings.col.dataType'), dataIndex: 'dataType', width: 100 },
    { title: t('product.paramModel.mappings.col.changeApplies'), dataIndex: 'changeApplies', width: 110 },
    { title: t('product.paramModel.mappings.col.min'), dataIndex: 'minValue', width: 80 },
    { title: t('product.paramModel.mappings.col.max'), dataIndex: 'maxValue', width: 80 },
    {
      title: t('product.paramModel.mappings.col.source'),
      dataIndex: 'source',
      width: 90,
      render: (s: string) =>
        s === 'custom' ? (
          <Tag color="blue">{t('product.paramModel.mappings.source.custom')}</Tag>
        ) : (
          <Tag>{t('product.paramModel.mappings.source.builtin')}</Tag>
        ),
    },
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
          {/* T-PMSRC：仅 source='custom' 可删;内置(来自 XML)置灰 + Tooltip。 */}
          {row.deletable ? (
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
          ) : (
            <Tooltip title={t('product.paramModel.mappings.builtinNotDeletable')}>
              <Button size="small" danger icon={<DeleteOutlined />} disabled />
            </Tooltip>
          )}
        </Space>
      ),
    },
  ];

  // #214：把全部列接入「可拖拽列宽」（带 width 的列可拖；标准/私有 PATH 默认 280，可拖宽看全长路径，
  // 列宽按浏览器持久化）。必须在任何条件 return 之前调用（Hooks 规则）。
  const allColumns: ColumnsType<ParamMapping> = [
    makeSeqColumn<ParamMapping>({ title: t('table.rowNumber'), dataSource: filtered }),
    ...columns,
  ];
  const {
    columns: resizableColumns,
    components: resizableComponents,
    tableClassName,
  } = useResizableColumns<ParamMapping>(allColumns, { storageKey: 'parammodel-mappings-colwidth' });

  const handleSave = async () => {
    if (!selectedName) return;
    try {
      const v = await form.validateFields();
      // InputNumber 产出 number | undefined / null；后端契约 minValue/maxValue 为字符串，
      // 这里规整：空(undefined/null/'') → undefined(留空=不校验)，数值 → 字符串。
      const norm = (n: unknown): string | undefined =>
        n === undefined || n === null || n === '' ? undefined : String(n);
      const input: CreateMappingInput = {
        standardPath: v.standardPath as string,
        privatePath: v.privatePath as string,
        entryType: (v.entryType as string) || 'parameter',
        access: (v.access as string) || 'readWrite',
        dataType: (v.dataType as string) || 'string',
        changeApplies: v.changeApplies,
        minValue: norm(v.minValue),
        maxValue: norm(v.maxValue),
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
    <>
      {/* 2026-06-03:返回+筛选行对齐 kpi-library 详情页 — 独立 toolbar Card + 两端对齐 Space,
          返回默认尺寸 + 模型名(fontWeight 600);右侧 搜索 / 条目筛选 / 新增映射。 */}
      <Card size="small" style={{ marginBottom: 12 }}>
        <Space style={{ width: '100%', justifyContent: 'space-between' }} wrap>
          <Space wrap>
            {onBack && (
              <Button icon={<ArrowLeftOutlined />} onClick={onBack}>
                {t('common.back')}
              </Button>
            )}
            <span style={{ fontWeight: 600 }}>{selectedName}</span>
          </Space>
          <Space wrap>
            <SearchInput
              placeholder={t('product.paramModel.pathSearchPh')}
              allowClear
              onSearch={(v) => setKeyword(v.trim())}
              style={{ width: 320 }}
              enterButton
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
                form.setFieldsValue({ entryType: 'parameter', access: 'readWrite', dataType: 'string', changeApplies: 'OnReboot' });
              }}
            >
              {t('common.create')}
            </Button>
          </Space>
        </Space>
      </Card>
      <Card size="small">
        <ResizableColumnsStyle scope={tableClassName} />
        <Table<ParamMapping>
          rowKey="id"
          className={tableClassName}
          loading={isLoading}
          columns={resizableColumns}
          components={resizableComponents}
          tableLayout="fixed"
          scroll={{ x: 'max-content' }}
          dataSource={filtered}
          size="small"
          pagination={{
            current: page,
            pageSize,
            total: filtered.length,
            showSizeChanger: true,
            pageSizeOptions: ['10', '20', '50', '100'],
            showTotal: (n) => t('common.totalCount', { count: n }),
            onChange: (p, ps) => {
              setPage(p);
              if (ps !== pageSize) setPageSize(ps);
            },
          }}
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
          {/* 标准 PATH:可搜索下拉,选项来自 standard-params;选中回填元属性。
              编辑态锁定(后端不允许改 standard_path)。 */}
          <Form.Item
            name="standardPath"
            label={t('product.paramModel.mappings.col.standardPath')}
            rules={[
              { required: true, message: t('common.required') },
              {
                // 新增态:同一标准 PATH 不能重复添加(编辑态 standard_path 锁定,跳过)。
                validator: (_, value: string) => {
                  if (editing || !value || !usedStdPaths.has(value)) return Promise.resolve();
                  return Promise.reject(new Error(t('product.paramModel.mappings.standardDuplicate')));
                },
              },
            ]}
            extra={
              editing
                ? t('product.paramModel.mappings.standardLockedHint')
                : t('product.paramModel.mappings.privatePathExtra')
            }
          >
            <Select
              showSearch
              placeholder={t('product.paramModel.mappings.standardPickPh')}
              options={stdOptions}
              optionFilterProp="value"
              disabled={Boolean(editing)}
              onChange={(v: string) => onPickStandard(v)}
            />
          </Form.Item>
          {/* 私有 PATH:编辑态允许更新(后端 UpdateMapping 支持改 private_path,并把 source 自动转为 custom)。
              受 (param_model_id, private_path) 唯一约束保护,改成已存在的私有 PATH 后端会返回错误。 */}
          <Form.Item
            name="privatePath"
            label={t('product.paramModel.mappings.col.privatePath')}
            rules={[{ required: true, message: t('common.required') }]}
            extra={editing ? t('product.paramModel.mappings.privateEditHint') : undefined}
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
              <Select
                options={dataTypeOptions}
                style={{ width: 140 }}
                onChange={(val) => {
                  // 切到无取值范围类型（boolean/dateTime）时清空 min/max，避免提交残留值。
                  if (dataTypeRangeKind(val as string) === 'none') {
                    form.setFieldsValue({ minValue: undefined, maxValue: undefined });
                  }
                }}
              />
            </Form.Item>
          </Space>
          <Space wrap>
            <Form.Item name="changeApplies" label={t('product.paramModel.mappings.col.changeApplies')}>
              <Select options={changeAppliesOptions} style={{ width: 140 }} />
            </Form.Item>
            <Form.Item name="minValue" label={minLabel}>
              <InputNumber
                precision={0}
                min={minBound}
                disabled={rangeNone}
                placeholder={intPlaceholder}
                style={{ width: 140 }}
              />
            </Form.Item>
            <Form.Item name="maxValue" label={maxLabel}>
              <InputNumber
                precision={0}
                min={minBound}
                disabled={rangeNone}
                placeholder={intPlaceholder}
                style={{ width: 140 }}
              />
            </Form.Item>
          </Space>
        </Form>
      </Modal>
      </Card>
    </>
  );
}
