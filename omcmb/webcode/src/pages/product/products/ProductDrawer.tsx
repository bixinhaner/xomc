import { useEffect, useMemo, useState } from 'react';
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
  Tooltip,
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
} from '@core/hooks/api/useProducts';
// #241：三个下拉(参数模型/KPI平台/告警neType)改字典数据源绑定(T-0182),由字典机制统一刷新,
// 不再各下拉各搞一套 query key。与设备列表 network_type/product_class 同范式。
import { useDictionary } from '@core/hooks/api/useSystem';
import { useSystemLicense } from '@core/hooks/api/useSystemLicense';
import { useTechnologyDictionary } from '@core/hooks/api/useTechnologyDictionary';
import {
  filterDeviceStandardOptionsByLicense,
  isDeviceStandardVisibleByLicense,
} from '@core/utils/licenseFeatures';
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
  // #241：参数模型下拉改字典驱动后,提交值 = param_models.name(后端按名反查 id);字段随之改名。
  paramModelName?: string;
  indicatorPlatform: string;
  alarmNeType: string;
}

// #241：把字典明细映射为 antd Select options(label/value 同名值)。与设备列表 toOptions 同款。
function toDictOptions(
  dict: { sysDictionaryDetails?: { label: string; value: string }[] } | null | undefined,
): { label: string; value: string }[] {
  return (dict?.sysDictionaryDetails ?? []).map((d) => ({ label: d.label, value: d.value }));
}

const FALLBACK_TECH_OPTIONS = [
  { label: 'eNB (LTE)', value: 'lte' },
  { label: 'gNB (NR)', value: 'nr' },
  { label: 'GSM', value: 'gsm' },
];

// 2026-06-03 用户决策:取消"指标设备类型"的修改/新增 —— 由制式(tech)派生,不再单独编辑。
// lte→enb / nr→gnb / gsm→gsm(两者一一对应,原本冗余)。
const TECH_TO_DEVTYPE: Record<string, string> = { lte: 'enb', nr: 'gnb', gsm: 'gsm', ups: '' };

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

  // #241：三个下拉数据源改字典(T-0182 绑定来源字段,字典机制统一刷新 — 三库导入后自动刷新 + 每日 cron 兜底)。
  const { data: paramModelDict } = useDictionary('param_model_name');
  const { data: kpiPlatformDict } = useDictionary('kpi_platform_enb');
  const { data: alarmNeTypeDict } = useDictionary('alarm_ne_type');
  const { options: technologyOptions, isLoading: technologyOptionsLoading } = useTechnologyDictionary();
  const { data: systemLicense, isLoading: systemLicenseLoading } = useSystemLicense();
  const showUPSOptions = isDeviceStandardVisibleByLicense(systemLicense, systemLicenseLoading, 'UPS');
  const techOptions = useMemo(() => {
    const base = (technologyOptions.length > 0 ? technologyOptions : FALLBACK_TECH_OPTIONS)
      .map((opt) => ({ label: opt.label, value: opt.value }));
    if (showUPSOptions && !base.some((opt) => opt.value.toLowerCase() === 'ups')) {
      base.push({ label: 'UPS', value: 'ups' });
    }
    return filterDeviceStandardOptionsByLicense(base, systemLicense, systemLicenseLoading);
  }, [systemLicense, systemLicenseLoading, technologyOptions, showUPSOptions]);
  const paramModelOptions = useMemo(
    () => filterDeviceStandardOptionsByLicense(toDictOptions(paramModelDict), systemLicense, systemLicenseLoading),
    [paramModelDict, systemLicense, systemLicenseLoading],
  );
  const alarmNeTypeOptions = useMemo(
    () => filterDeviceStandardOptionsByLicense(toDictOptions(alarmNeTypeDict), systemLicense, systemLicenseLoading),
    [alarmNeTypeDict, systemLicense, systemLicenseLoading],
  );

  // 指标设备类型由制式派生(不再单独编辑);ENB(lte) 才需要选指标平台
  const tech = Form.useWatch('tech', form);
  const indicatorDeviceType = TECH_TO_DEVTYPE[tech || ''] || '';
  const isENB = indicatorDeviceType === 'enb';

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
        // #241：编辑回填用后端反查的 paramModelName(字典 value=name);后端再按名反查 id。
        paramModelName: product.paramModelName,
        indicatorPlatform: product.indicatorPlatform,
        alarmNeType: product.alarmNeType,
      });
    } else {
      form.resetFields();
      // 新增默认制式=LTE(最常见的 ENB 基站),使「KPI指标名称」字段默认可见可填,与编辑态对齐;
      // 该字段仅 ENB/LTE 适用,切到 GSM/NR 会自动隐藏(保留 2026-06-03「平台仅 ENB」决策)。
      // 核心网等非无线产品不区分制式,可清空 tech(allowClear)→ indicatorDeviceType 派生为 ''。
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
          // 指标设备类型由制式派生(取消单独编辑);制式为空(核心网等非无线产品)→ 派生为 ''
          indicatorDeviceType: TECH_TO_DEVTYPE[v.tech] ?? '',
          indicatorPlatform: v.indicatorPlatform,
          alarmNeType: v.alarmNeType,
          enableFiletype11: product.enableFiletype11,
          enableUnknownAlarm: product.enableUnknownAlarm,
          deviceAttrsOverride: product.deviceAttrsOverride ?? DEFAULT_OVERRIDE,
          // #241：提交字典 value(param_models.name);后端按名反查 param_model_id。未选则清空软引用。
          paramModelName: v.paramModelName,
          clearParamModel: !v.paramModelName,
        };
        await updateMut.mutateAsync({ id: product.id, input });
        message.success(t('common.saved'));
      } else {
        const input: CreateProductInput = {
          name: v.name,
          vendor: v.vendor,
          tech: v.tech,
          description: v.description,
          // 指标设备类型由制式派生(取消单独编辑);制式为空(核心网等非无线产品)→ 派生为 ''
          indicatorDeviceType: TECH_TO_DEVTYPE[v.tech] ?? '',
          indicatorPlatform: v.indicatorPlatform,
          alarmNeType: v.alarmNeType,
          enableFiletype11: true,
          enableUnknownAlarm: false,
          deviceAttrsOverride: DEFAULT_OVERRIDE,
          // #241：提交字典 value(param_models.name);后端按名反查 param_model_id。
          paramModelName: v.paramModelName,
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

  // 内置正则（source='builtin'，来自 products.xml）UI 只读：编辑/启停/移动/删除一律置灰。
  // 仅 custom（UI 新增、重灌保留）可改。后端 guardPatternEditable 同步硬拒（403）。
  const builtinTip = t('product.products.builtinReadonly');
  const patternColumns = [
    {
      title: 'sort',
      dataIndex: 'sortOrder',
      width: 70,
      render: (v: number) => <Tag color="blue">{v}</Tag>,
    },
    {
      title: t('product.products.col.source'),
      dataIndex: 'source',
      width: 80,
      render: (v: string) =>
        v === 'custom' ? (
          <Tag color="green">{t('product.products.sourceCustom')}</Tag>
        ) : (
          <Tag>{t('product.products.sourceBuiltin')}</Tag>
        ),
    },
    {
      title: t('product.products.col.regex'),
      dataIndex: 'productClass',
      render: (v: string, row: ProductPattern) =>
        row.deletable ? (
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
        ) : (
          <Tooltip title={builtinTip}>
            <Input value={v} readOnly disabled />
          </Tooltip>
        ),
    },
    {
      title: t('common.enable'),
      dataIndex: 'isActive',
      width: 70,
      render: (v: boolean, row: ProductPattern) => (
        <Tooltip title={row.deletable ? '' : builtinTip}>
          <Switch
            size="small"
            checked={v}
            disabled={!row.deletable}
            onChange={(checked) => {
              if (!product) return;
              updPatMut
                .mutateAsync({ productId: product.id, patternId: row.id, isActive: checked })
                .then(() => message.success(checked ? t('common.enabled') : t('common.disabled')))
                .catch((er) => message.error((er as Error).message));
            }}
          />
        </Tooltip>
      ),
    },
    {
      title: t('common.action'),
      width: 200,
      render: (_: unknown, row: ProductPattern) => {
        const locked = !product || !row.deletable;
        return (
          <Tooltip title={!row.deletable ? builtinTip : ''}>
            <Space>
              <Button
                size="small"
                icon={<ArrowUpOutlined />}
                disabled={locked}
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
                disabled={locked}
                onClick={() =>
                  product &&
                  movPatMut
                    .mutateAsync({ productId: product.id, patternId: row.id, direction: 'down' })
                    .catch((er) => message.error((er as Error).message))
                }
              />
              <Popconfirm
                title={t('product.products.delRegexTitle')}
                disabled={locked}
                onConfirm={() =>
                  product &&
                  delPatMut
                    .mutateAsync({ productId: product.id, patternId: row.id })
                    .then(() => message.success(t('common.deleted')))
                    .catch((er) => message.error((er as Error).message))
                }
              >
                <Button size="small" danger icon={<DeleteOutlined />} disabled={locked} />
              </Popconfirm>
            </Space>
          </Tooltip>
        );
      },
    },
  ];

  return (
    <Drawer
      title={isEdit ? t('product.product.drawer.editTitle', { name: product?.name ?? '' }) : t('product.product.drawer.createTitle')}
      placement="right"
      size={720}
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
                  <Form.Item name="tech" label={t('product.products.tech')} tooltip={t('product.products.techOptionalTip')}>
                    <Select
                      options={techOptions}
                      loading={technologyOptionsLoading}
                      allowClear
                      onChange={(val) => {
                        // 制式变更后:非 ENB(lte) 清空指标平台(平台仅 ENB 适用)。
                        // 核心网等非无线产品不区分制式,清空 tech → indicatorDeviceType 派生为 ''。
                        if ((TECH_TO_DEVTYPE[val as string] || '') !== 'enb') {
                          form.setFieldValue('indicatorPlatform', undefined);
                        }
                      }}
                    />
                  </Form.Item>
                  <Form.Item
                    name="paramModelName"
                    label={t('product.products.paramModel')}
                    rules={[{ required: true, message: t('common.pleaseSelect') }]}
                  >
                    <Select
                      placeholder={t('product.products.paramModelPh')}
                      options={paramModelOptions}
                      showSearch
                      optionFilterProp="label"
                      notFoundContent={paramModelDict ? t('product.product.drawer.notFoundParamModels') : t('common.loading')}
                    />
                  </Form.Item>
                  {isENB && (
                    <Form.Item
                      name="indicatorPlatform"
                      label={t('product.products.indicatorPlatform')}
                      rules={[{ required: true, message: t('product.product.drawer.indicatorPlatformRequired') }]}
                    >
                      <Select
                        placeholder={t('product.products.indicatorPlatformPh')}
                        options={toDictOptions(kpiPlatformDict)}
                        showSearch
                        optionFilterProp="label"
                        notFoundContent={kpiPlatformDict ? t('product.product.drawer.notFoundPlatforms') : t('common.loading')}
                      />
                    </Form.Item>
                  )}
                  <Form.Item name="alarmNeType" label={t('product.products.alarmNeType')} rules={[{ required: true }]}>
                    <Select
                      placeholder={t('product.products.alarmNeTypePh')}
                      options={alarmNeTypeOptions}
                      showSearch
                      optionFilterProp="label"
                      notFoundContent={alarmNeTypeDict ? t('product.product.drawer.notFoundAlarms') : t('common.loading')}
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
