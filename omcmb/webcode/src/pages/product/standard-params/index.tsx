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
  InputNumber,
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
import {
  STANDARD_DATA_TYPES,
  STANDARD_CHANGE_APPLIES,
  dataTypeRangeKind,
  isUnsignedDataType,
} from '@core/types/paramModel';
import type { StandardParam, UpsertStandardInput } from '@core/types/paramModel';
import { makeSeqColumn } from '@/components/Table/seqColumn';
import SearchInput from '@/components/SearchInput';
import { formatSystemTime } from '@core/utils/systemTime';
import { useT } from '@/hooks/useT';
import { getI18nKeyByBizCode } from '@core/i18n/bizCodeMessages';
import { nextPageOnPaginationChange } from './pagination';

export default function StandardParamsPage() {
  const t = useT();
  const ENTRY_OPTIONS = [
    { label: t('common.all'), value: '' },
    { label: 'parameter', value: 'parameter' },
    { label: 'object', value: 'object' },
  ];
  // 手动搜索:keyword 仅在 onSearch(回车/点击搜索)时应用。
  const [keyword, setKeyword] = useState('');
  const [entryType, setEntryType] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(50);
  const { data, isLoading } = useStandardParams({
    keyword: keyword || undefined,
    entryType: entryType || undefined,
  });

  // 过滤条件变化时回到第一页——渲染期重置,避免 set-state-in-effect。
  const filterKey = `${keyword}|${entryType}`;
  const [prevFilterKey, setPrevFilterKey] = useState(filterKey);
  if (filterKey !== prevFilterKey) {
    setPrevFilterKey(filterKey);
    setPage(1);
  }
  const upsertMut = useUpsertStandard();
  const deleteMut = useDeleteStandard();

  const [editing, setEditing] = useState<StandardParam | null>(null);
  const [creating, setCreating] = useState(false);
  const [form] = Form.useForm<UpsertStandardInput>();

  // 当前表单内的 dataType / changeApplies —— 用于动态 label 与历史值兼容。
  const watchedDataType = Form.useWatch('dataType', form);
  const watchedChangeApplies = Form.useWatch('changeApplies', form);

  // dataType 下拉：枚举集 + 若当前编辑值不在枚举内则并入（历史遗留如 STRING/U_INT 不被静默清空）。
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

  // changeApplies 下拉：枚举集（显示英文枚举值，与 dataType 一致）+ 历史遗留值（如 reload/immediate 小写）并入。
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

  const items = data?.items || [];

  const columns = [
    { title: t('product.standardParams.col.standardPath'), dataIndex: 'standardPath', ellipsis: true },
    { title: t('product.standardParams.col.entryType'), dataIndex: 'entryType', width: 100 },
    { title: t('product.standardParams.col.access'), dataIndex: 'access', width: 130 },
    { title: t('product.standardParams.col.dataType'), dataIndex: 'dataType', width: 100 },
    { title: t('product.standardParams.col.changeApplies'), dataIndex: 'changeApplies', width: 130 },
    { title: t('product.standardParams.col.min'), dataIndex: 'minValue', width: 80 },
    { title: t('product.standardParams.col.max'), dataIndex: 'maxValue', width: 80 },
    {
      title: t('common.updateTime'),
      dataIndex: 'updatedAt',
      width: 170,
      render: (value: string) => formatSystemTime(value),
    },
    {
      title: t('product.standardParams.col.updatedFields'),
      dataIndex: 'updatedFields',
      width: 220,
      ellipsis: true,
      render: (fields?: string[]) =>
        fields && fields.length > 0 ? fields.join(', ') : '-',
    },
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
      // InputNumber 产出 number | undefined / null；后端契约 minValue/maxValue 为字符串，
      // 这里规整：空(undefined/null) → undefined(留空=不校验)，数值 → 字符串。
      const norm = (n: unknown): string | undefined =>
        n === undefined || n === null || n === '' ? undefined : String(n);
      const input: UpsertStandardInput = {
        ...v,
        minValue: norm(v.minValue),
        maxValue: norm(v.maxValue),
      };
      await upsertMut.mutateAsync({ input, path: editing?.standardPath });
      message.success(editing ? t('common.saved') : t('common.created'));
      setEditing(null);
      setCreating(false);
      form.resetFields();
    } catch (e) {
      const error = e as Error & {
        bizCode?: number;
        response?: { data?: { biz_code?: number; code?: number } };
      };
      const bizCode = error.bizCode ?? error.response?.data?.biz_code ?? error.response?.data?.code;
      const i18nKey = getI18nKeyByBizCode(bizCode);
      if (i18nKey === 'product.standardParams.pathExists') {
        message.error(t(i18nKey, { path: form.getFieldValue('standardPath') }));
      } else if (error.message) {
        message.error(error.message);
      }
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
            <SearchInput
              placeholder={t('product.standardParams.searchPh')}
              allowClear
              onSearch={(v) => setKeyword(v.trim())}
              style={{ width: 320 }}
              enterButton
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
                changeApplies: 'OnReboot',
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
          pagination={{
            current: page,
            pageSize,
            total: items.length,
            showSizeChanger: true,
            pageSizeOptions: ['10', '20', '50', '100'],
            showTotal: (n) => t('common.totalCount', { count: n }),
            onChange: (p, ps) => {
              // 改每页条数时回到第 1 页重新切片(issue #190:约 2000 条标准参数树,
              // 改 pageSize 后若停留在原页码会出现切片不刷新)。仅翻页时按目标页码走。
              const next = nextPageOnPaginationChange(p, ps, pageSize);
              if (next.sizeChanged) setPageSize(next.pageSize);
              setPage(next.page);
            },
          }}
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
              label={t('product.standardParams.col.standardPath')}
              rules={[{ required: true, message: t('common.required') }]}
            >
              <Input disabled={Boolean(editing)} />
            </Form.Item>
            <Space wrap>
              <Form.Item name="entryType" label={t('product.standardParams.col.entryType')} rules={[{ required: true }]}>
                <Select
                  options={[
                    { label: 'parameter', value: 'parameter' },
                    { label: 'object', value: 'object' },
                  ]}
                  style={{ width: 140 }}
                />
              </Form.Item>
              <Form.Item name="access" label={t('product.standardParams.col.access')} rules={[{ required: true }]}>
                <Select
                  options={[
                    { label: 'readWrite', value: 'readWrite' },
                    { label: 'readOnly', value: 'readOnly' },
                  ]}
                  style={{ width: 140 }}
                />
              </Form.Item>
              <Form.Item name="dataType" label={t('product.standardParams.col.dataType')} rules={[{ required: true }]}>
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
              <Form.Item name="changeApplies" label={t('product.standardParams.col.changeApplies')}>
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
    </div>
  );
}
