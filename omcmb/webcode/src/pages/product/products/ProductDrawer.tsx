import { useEffect, useState } from 'react';
import {
  Drawer,
  Form,
  Input,
  Switch,
  Select,
  Tabs,
  Button,
  Space,
  Table,
  Popconfirm,
  message,
  Typography,
  Tag,
} from 'antd';
import { ArrowUpOutlined, ArrowDownOutlined, DeleteOutlined } from '@ant-design/icons';
import {
  useProductDetail,
  useCreateProduct,
  useUpdateProduct,
  useCreatePattern,
  useUpdatePattern,
  useDeletePattern,
  useMovePattern,
  useIndicatorPlatforms,
  useAlarmNeTypes,
} from '@core/hooks/api/useProducts';
import { useParamModelList } from '@core/hooks/api/useParamModels';
import type {
  Product,
  ProductPattern,
  CreateProductInput,
  UpdateProductInput,
  DeviceAttrsOverride,
} from '@core/types/product';
import { useT } from '@/hooks/useT';

const { Text } = Typography;

interface Props {
  open: boolean;
  product?: Product | null;
  onClose: () => void;
}

interface FormValues {
  name: string;
  vendor: string;
  tech: string;
  description: string;
  paramModelId?: string;
  indicatorPlatform: string;
  alarmNeType: string;
}

const TECH_OPTIONS = [
  { label: 'LTE (4G)', value: 'lte' },
  { label: 'NR (5G)', value: 'nr' },
  { label: 'GSM', value: 'gsm' },
];

// 2026-06-03 用户决策:取消"指标设备类型"的修改/新增 —— 由制式(tech)派生,不再单独编辑。
// lte→enb / nr→gnb / gsm→gsm(两者一一对应,原本冗余)。
const TECH_TO_DEVTYPE: Record<string, string> = { lte: 'enb', nr: 'gnb', gsm: 'gsm' };

// 上传策略 / 属性覆盖默认值：原"上传策略"页签已下线，新增时套用默认；
// 编辑时沿用产品已有值（避免保存时把不可见字段清零）。
const DEFAULT_OVERRIDE: DeviceAttrsOverride = {
  data_type: false,
  access: true,
  min_value: false,
  max_value: false,
  change_applies: true,
};

export default function ProductDrawer({ open, product, onClose }: Props) {
  const t = useT();
  const isEdit = Boolean(product);
  const [form] = Form.useForm<FormValues>();
  const [activeTab, setActiveTab] = useState('basic');
  const [newPattern, setNewPattern] = useState('');
  // 新增模式下暂存的正则数组；提交时随 Create 接口一并落库（后端事务原子）
  const [pendingPatterns, setPendingPatterns] = useState<string[]>([]);

  const { data: detail } = useProductDetail(isEdit ? product?.id : undefined);
  const { data: paramModels } = useParamModelList();

  // 指标设备类型由制式派生(不再单独编辑);ENB(lte) 才需要选指标平台
  const tech = Form.useWatch('tech', form);
  const indicatorDeviceType = TECH_TO_DEVTYPE[tech || ''] || '';
  const isENB = indicatorDeviceType === 'enb';
  const { data: indicatorPlatforms } = useIndicatorPlatforms(isENB ? indicatorDeviceType : undefined);
  const { data: alarmNeTypes } = useAlarmNeTypes();

  const createMut = useCreateProduct();
  const updateMut = useUpdateProduct();
  const addPatMut = useCreatePattern();
  const updPatMut = useUpdatePattern();
  const delPatMut = useDeletePattern();
  const movPatMut = useMovePattern();

  useEffect(() => {
    if (!open) return;
    if (product) {
      form.setFieldsValue({
        name: product.name,
        vendor: product.vendor,
        tech: product.tech,
        description: product.description,
        paramModelId: product.paramModelId,
        indicatorPlatform: product.indicatorPlatform,
        alarmNeType: product.alarmNeType,
      });
    } else {
      form.resetFields();
      // 新增默认制式=LTE(最常见的 ENB 基站),使「KPI指标名称」字段默认可见可填,与编辑态对齐;
      // 该字段仅 ENB/LTE 适用,切到 GSM/NR 会自动隐藏(保留 2026-06-03「平台仅 ENB」决策)。
      form.setFieldsValue({ tech: 'lte' });
    }
    setActiveTab('basic');
    setNewPattern('');
    setPendingPatterns([]);
  }, [open, product, form]);

  const handleSubmit = async () => {
    try {
      const v = await form.validateFields();
      if (isEdit && product) {
        const input: UpdateProductInput = {
          name: v.name,
          vendor: v.vendor,
          tech: v.tech,
          description: v.description,
          // 指标设备类型由制式派生(取消单独编辑)
          indicatorDeviceType: TECH_TO_DEVTYPE[v.tech] ?? product.indicatorDeviceType,
          indicatorPlatform: v.indicatorPlatform,
          alarmNeType: v.alarmNeType,
          enableFiletype11: product.enableFiletype11,
          enableUnknownAlarm: product.enableUnknownAlarm,
          deviceAttrsOverride: product.deviceAttrsOverride ?? DEFAULT_OVERRIDE,
          paramModelId: v.paramModelId,
          clearParamModel: !v.paramModelId,
        };
        await updateMut.mutateAsync({ id: product.id, input });
        message.success(t('common.saved'));
      } else {
        const input: CreateProductInput = {
          name: v.name,
          vendor: v.vendor,
          tech: v.tech,
          description: v.description,
          // 指标设备类型由制式派生(取消单独编辑)
          indicatorDeviceType: TECH_TO_DEVTYPE[v.tech] ?? '',
          indicatorPlatform: v.indicatorPlatform,
          alarmNeType: v.alarmNeType,
          enableFiletype11: true,
          enableUnknownAlarm: false,
          deviceAttrsOverride: DEFAULT_OVERRIDE,
          paramModelId: v.paramModelId,
          patterns: pendingPatterns.length > 0 ? pendingPatterns : undefined,
        };
        await createMut.mutateAsync(input);
        message.success(t('common.created'));
      }
      onClose();
    } catch (err) {
      const msg = (err as Error).message;
      if (msg) message.error(msg);
    }
  };

  const patterns: ProductPattern[] = (detail?.patterns || []).slice().sort((a, b) => a.sortOrder - b.sortOrder);

  const patternColumns = [
    {
      title: 'sort',
      dataIndex: 'sortOrder',
      width: 70,
      render: (v: number) => <Tag color="blue">{v}</Tag>,
    },
    {
      title: t('product.products.col.regex'),
      dataIndex: 'productClass',
      render: (v: string, row: ProductPattern) => (
        <Input
          defaultValue={v}
          onBlur={(e) => {
            const next = e.target.value.trim();
            if (next && next !== v && product) {
              updPatMut
                .mutateAsync({ productId: product.id, patternId: row.id, productClass: next })
                .then(() => message.success(t('common.updated')))
                .catch((er) => message.error((er as Error).message));
            }
          }}
        />
      ),
    },
    {
      title: t('common.enable'),
      dataIndex: 'isActive',
      width: 70,
      render: (v: boolean, row: ProductPattern) => (
        <Switch
          size="small"
          checked={v}
          onChange={(checked) => {
            if (!product) return;
            updPatMut
              .mutateAsync({ productId: product.id, patternId: row.id, isActive: checked })
              .then(() => message.success(checked ? t('common.enabled') : t('common.disabled')))
              .catch((er) => message.error((er as Error).message));
          }}
        />
      ),
    },
    {
      title: t('common.action'),
      width: 200,
      render: (_: unknown, row: ProductPattern) => (
        <Space>
          <Button
            size="small"
            icon={<ArrowUpOutlined />}
            disabled={!product}
            onClick={() =>
              product &&
              movPatMut
                .mutateAsync({ productId: product.id, patternId: row.id, direction: 'up' })
                .catch((er) => message.error((er as Error).message))
            }
          />
          <Button
            size="small"
            icon={<ArrowDownOutlined />}
            disabled={!product}
            onClick={() =>
              product &&
              movPatMut
                .mutateAsync({ productId: product.id, patternId: row.id, direction: 'down' })
                .catch((er) => message.error((er as Error).message))
            }
          />
          <Popconfirm
            title={t('product.products.delRegexTitle')}
            onConfirm={() =>
              product &&
              delPatMut
                .mutateAsync({ productId: product.id, patternId: row.id })
                .then(() => message.success(t('common.deleted')))
                .catch((er) => message.error((er as Error).message))
            }
          >
            <Button size="small" danger icon={<DeleteOutlined />} disabled={!product} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <Drawer
      title={isEdit ? t('product.product.drawer.editTitle', { name: product?.name ?? '' }) : t('product.product.drawer.createTitle')}
      placement="right"
      width={720}
      open={open}
      onClose={onClose}
      destroyOnHidden
      footer={
        <Space style={{ float: 'right' }}>
          <Button onClick={onClose}>{t('common.cancel')}</Button>
          <Button
            type="primary"
            loading={createMut.isPending || updateMut.isPending}
            onClick={() => void handleSubmit()}
          >
            {t('common.save')}
          </Button>
        </Space>
      }
    >
      <Form<FormValues> form={form} layout="vertical">
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          items={[
            {
              key: 'basic',
              label: t('product.product.drawer.tabBasic'),
              children: (
                <>
                  <Form.Item name="name" label={t('product.products.name')} rules={[{ required: true, message: t('product.products.nameRequired') }]}>
                    <Input placeholder={t('product.products.namePh')} />
                  </Form.Item>
                  <Form.Item name="description" label={t('common.description')}>
                    <Input.TextArea rows={2} />
                  </Form.Item>
                  <Form.Item name="vendor" label={t('common.vendor')}>
                    <Input placeholder="Comba / Baicells / ..." />
                  </Form.Item>
                  <Form.Item name="tech" label={t('product.products.tech')} rules={[{ required: true }]}>
                    <Select
                      options={TECH_OPTIONS}
                      onChange={(val) => {
                        // 制式变更后:非 ENB(lte) 清空指标平台(平台仅 ENB 适用)
                        if ((TECH_TO_DEVTYPE[val] || '') !== 'enb') {
                          form.setFieldValue('indicatorPlatform', undefined);
                        }
                      }}
                    />
                  </Form.Item>
                  <Form.Item
                    name="paramModelId"
                    label={t('product.products.paramModel')}
                    rules={[{ required: true, message: t('common.pleaseSelect') }]}
                    extra={t('product.product.drawer.paramModelExtra')}
                  >
                    <Select
                      placeholder={t('product.products.paramModelPh')}
                      options={(paramModels?.items || []).map((m) => ({
                        label: t('product.product.drawer.paramModelOption', { name: m.name, count: m.totalParams }),
                        value: m.id,
                      }))}
                      showSearch
                      optionFilterProp="label"
                    />
                  </Form.Item>
                  {isENB && (
                    <Form.Item
                      name="indicatorPlatform"
                      label={t('product.products.indicatorPlatform')}
                      rules={[{ required: true, message: t('product.product.drawer.indicatorPlatformRequired') }]}
                      extra={t('product.product.drawer.indicatorPlatformExtraLong')}
                    >
                      <Select
                        placeholder={t('product.products.indicatorPlatformPh')}
                        options={(indicatorPlatforms || []).map((p) => ({ label: p, value: p }))}
                        showSearch
                        optionFilterProp="label"
                        notFoundContent={indicatorPlatforms ? t('product.product.drawer.notFoundPlatforms') : t('common.loading')}
                      />
                    </Form.Item>
                  )}
                  <Form.Item name="alarmNeType" label={t('product.products.alarmNeType')} rules={[{ required: true }]}>
                    <Select
                      placeholder={t('product.products.alarmNeTypePh')}
                      options={(alarmNeTypes || []).map((n) => ({ label: n, value: n }))}
                      showSearch
                      optionFilterProp="label"
                      notFoundContent={alarmNeTypes ? t('product.product.drawer.notFoundAlarms') : t('common.loading')}
                    />
                  </Form.Item>
                </>
              ),
            },
            {
              key: 'patterns',
              label: t('product.product.drawer.tabPatterns'),
              children: !isEdit ? (
                <>
                  <Text type="secondary" style={{ display: 'block', marginBottom: 12 }}>
                    {t('product.product.drawer.patternsHint')}
                  </Text>
                  <Space style={{ marginBottom: 12 }}>
                    <Input
                      placeholder={t('product.products.newRegexPh')}
                      value={newPattern}
                      onChange={(e) => setNewPattern(e.target.value)}
                      onPressEnter={() => {
                        const v = newPattern.trim();
                        if (!v) return;
                        if (pendingPatterns.includes(v)) {
                          message.warning(t('common.regexExists'));
                          return;
                        }
                        setPendingPatterns([...pendingPatterns, v]);
                        setNewPattern('');
                      }}
                      style={{ width: 320 }}
                    />
                    <Button
                      type="primary"
                      disabled={!newPattern.trim()}
                      onClick={() => {
                        const v = newPattern.trim();
                        if (!v) return;
                        if (pendingPatterns.includes(v)) {
                          message.warning(t('common.regexExists'));
                          return;
                        }
                        setPendingPatterns([...pendingPatterns, v]);
                        setNewPattern('');
                      }}
                    >
                      {t('common.add')}
                    </Button>
                  </Space>
                  <Table<{ key: number; productClass: string }>
                    rowKey="key"
                    size="small"
                    columns={[
                      {
                        title: t('common.sortOrder'),
                        dataIndex: 'key',
                        width: 70,
                        render: (v: number) => <Tag color="blue">{v + 1}</Tag>,
                      },
                      { title: t('product.products.col.regex'), dataIndex: 'productClass' },
                      {
                        title: t('common.action'),
                        width: 80,
                        render: (_: unknown, _row, idx: number) => (
                          <Button
                            size="small"
                            danger
                            icon={<DeleteOutlined />}
                            onClick={() => {
                              setPendingPatterns(pendingPatterns.filter((_, i) => i !== idx));
                            }}
                          />
                        ),
                      },
                    ]}
                    dataSource={pendingPatterns.map((pc, idx) => ({ key: idx, productClass: pc }))}
                    pagination={false}
                    locale={{ emptyText: t('product.product.drawer.emptyPatterns') }}
                  />
                </>
              ) : (
                <>
                  <Space style={{ marginBottom: 12 }}>
                    <Input
                      placeholder={t('product.products.newRegexPh')}
                      value={newPattern}
                      onChange={(e) => setNewPattern(e.target.value)}
                      style={{ width: 320 }}
                    />
                    <Button
                      type="primary"
                      loading={addPatMut.isPending}
                      disabled={!newPattern.trim() || !product}
                      onClick={async () => {
                        if (!product) return;
                        try {
                          await addPatMut.mutateAsync({
                            productId: product.id,
                            productClass: newPattern.trim(),
                          });
                          setNewPattern('');
                          message.success(t('common.added'));
                        } catch (er) {
                          message.error((er as Error).message);
                        }
                      }}
                    >
                      {t('common.add')}
                    </Button>
                  </Space>
                  <Table<ProductPattern>
                    rowKey="id"
                    size="small"
                    columns={patternColumns}
                    dataSource={patterns}
                    pagination={false}
                  />
                </>
              ),
            },
          ]}
        />
      </Form>
    </Drawer>
  );
}
