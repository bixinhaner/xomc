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
  Checkbox,
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
} from '@core/hooks/api/useProducts';
import { useParamModelList } from '@core/hooks/api/useParamModels';
import type {
  Product,
  ProductPattern,
  CreateProductInput,
  UpdateProductInput,
  DeviceAttrsOverride,
} from '@core/types/product';

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
  radioModes: string;
  description: string;
  paramModelId?: string;
  indicatorDeviceType: string;
  indicatorPlatform: string;
  alarmNeType: string;
  enableFiletype11: boolean;
  enableUnknownAlarm: boolean;
  override_data_type: boolean;
  override_access: boolean;
  override_min_value: boolean;
  override_max_value: boolean;
  override_change_applies: boolean;
}

const DEVICE_TYPE_OPTIONS = [
  { label: 'ENB (LTE)', value: 'ENB' },
  { label: 'GNB (5G NR)', value: 'GNB' },
  { label: 'GSM', value: 'GSM' },
];

const TECH_OPTIONS = [
  { label: 'LTE (4G)', value: 'lte' },
  { label: 'NR (5G)', value: 'nr' },
  { label: 'GSM', value: 'gsm' },
];

export default function ProductDrawer({ open, product, onClose }: Props) {
  const isEdit = Boolean(product);
  const [form] = Form.useForm<FormValues>();
  const [activeTab, setActiveTab] = useState('basic');
  const [newPattern, setNewPattern] = useState('');

  const { data: detail } = useProductDetail(isEdit ? product?.id : undefined);
  const { data: paramModels } = useParamModelList();

  const createMut = useCreateProduct();
  const updateMut = useUpdateProduct();
  const addPatMut = useCreatePattern();
  const updPatMut = useUpdatePattern();
  const delPatMut = useDeletePattern();
  const movPatMut = useMovePattern();

  useEffect(() => {
    if (!open) return;
    if (product) {
      const o = product.deviceAttrsOverride || {};
      form.setFieldsValue({
        name: product.name,
        vendor: product.vendor,
        tech: product.tech,
        radioModes: product.radioModes,
        description: product.description,
        paramModelId: product.paramModelId,
        indicatorDeviceType: product.indicatorDeviceType,
        indicatorPlatform: product.indicatorPlatform,
        alarmNeType: product.alarmNeType,
        enableFiletype11: product.enableFiletype11,
        enableUnknownAlarm: product.enableUnknownAlarm,
        override_data_type: Boolean(o.data_type),
        override_access: Boolean(o.access),
        override_min_value: Boolean(o.min_value),
        override_max_value: Boolean(o.max_value),
        override_change_applies: Boolean(o.change_applies),
      });
    } else {
      form.resetFields();
      form.setFieldsValue({
        enableFiletype11: true,
        enableUnknownAlarm: false,
        override_data_type: false,
        override_access: true,
        override_min_value: false,
        override_max_value: false,
        override_change_applies: true,
      });
    }
    setActiveTab('basic');
    setNewPattern('');
  }, [open, product, form]);

  const buildOverride = (v: FormValues): DeviceAttrsOverride => ({
    data_type: v.override_data_type,
    access: v.override_access,
    min_value: v.override_min_value,
    max_value: v.override_max_value,
    change_applies: v.override_change_applies,
  });

  const handleSubmit = async () => {
    try {
      const v = await form.validateFields();
      if (isEdit && product) {
        const input: UpdateProductInput = {
          name: v.name,
          vendor: v.vendor,
          tech: v.tech,
          radioModes: v.radioModes,
          description: v.description,
          indicatorDeviceType: v.indicatorDeviceType,
          indicatorPlatform: v.indicatorPlatform,
          alarmNeType: v.alarmNeType,
          enableFiletype11: v.enableFiletype11,
          enableUnknownAlarm: v.enableUnknownAlarm,
          deviceAttrsOverride: buildOverride(v),
          paramModelId: v.paramModelId,
          clearParamModel: !v.paramModelId,
        };
        await updateMut.mutateAsync({ id: product.id, input });
        message.success('已保存');
      } else {
        const input: CreateProductInput = {
          name: v.name,
          vendor: v.vendor,
          tech: v.tech,
          radioModes: v.radioModes,
          description: v.description,
          indicatorDeviceType: v.indicatorDeviceType,
          indicatorPlatform: v.indicatorPlatform,
          alarmNeType: v.alarmNeType,
          enableFiletype11: v.enableFiletype11,
          enableUnknownAlarm: v.enableUnknownAlarm,
          deviceAttrsOverride: buildOverride(v),
          paramModelId: v.paramModelId,
        };
        await createMut.mutateAsync(input);
        message.success('已创建');
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
      title: '正则规则',
      dataIndex: 'productClass',
      render: (v: string, row: ProductPattern) => (
        <Input
          defaultValue={v}
          onBlur={(e) => {
            const next = e.target.value.trim();
            if (next && next !== v && product) {
              updPatMut
                .mutateAsync({ productId: product.id, patternId: row.id, productClass: next })
                .then(() => message.success('已更新'))
                .catch((er) => message.error((er as Error).message));
            }
          }}
        />
      ),
    },
    {
      title: '操作',
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
            title="确认删除该正则？"
            onConfirm={() =>
              product &&
              delPatMut
                .mutateAsync({ productId: product.id, patternId: row.id })
                .then(() => message.success('已删除'))
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
      title={isEdit ? `编辑产品：${product?.name}` : '新增产品'}
      placement="right"
      width={720}
      open={open}
      onClose={onClose}
      destroyOnClose
      footer={
        <Space style={{ float: 'right' }}>
          <Button onClick={onClose}>取消</Button>
          <Button
            type="primary"
            loading={createMut.isPending || updateMut.isPending}
            onClick={() => void handleSubmit()}
          >
            保存
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
              label: '基本信息',
              children: (
                <>
                  <Form.Item name="name" label="产品名" rules={[{ required: true, message: '产品名必填' }]}>
                    <Input placeholder="如 PicoCell-LTE-V2" />
                  </Form.Item>
                  <Form.Item name="description" label="描述">
                    <Input.TextArea rows={2} />
                  </Form.Item>
                  <Form.Item name="vendor" label="厂商">
                    <Input placeholder="Comba / Baicells / ..." />
                  </Form.Item>
                  <Form.Item name="tech" label="制式" rules={[{ required: true }]}>
                    <Select options={TECH_OPTIONS} />
                  </Form.Item>
                  <Form.Item name="radioModes" label="Radio Modes">
                    <Input placeholder="fdd / tdd / fdd-tdd" />
                  </Form.Item>
                  <Form.Item
                    name="paramModelId"
                    label="参数模型"
                    extra="选择该产品默认参数模型；P4-04 可在参数模型浏览器维护映射"
                  >
                    <Select
                      allowClear
                      placeholder="选择参数模型"
                      options={(paramModels?.items || []).map((m) => ({
                        label: `${m.name}（${m.totalParams} 参数）`,
                        value: m.id,
                      }))}
                      showSearch
                      optionFilterProp="label"
                    />
                  </Form.Item>
                  <Form.Item
                    name="indicatorDeviceType"
                    label="指标设备类型"
                    rules={[{ required: true }]}
                    extra="决定 KPI 库 5 Tabs 中加载哪一类计数器/公式"
                  >
                    <Select options={DEVICE_TYPE_OPTIONS} />
                  </Form.Item>
                  <Form.Item
                    name="indicatorPlatform"
                    label="指标平台名"
                    rules={[{ required: true }]}
                    extra="对应 KPI 库公式的 platform_name（如 enb-comba / gnb-default）"
                  >
                    <Input placeholder="enb-default / enb-comba / ..." />
                  </Form.Item>
                  <Form.Item name="alarmNeType" label="告警网元类型" rules={[{ required: true }]}>
                    <Input placeholder="eNodeB / gNodeB / BTS / ..." />
                  </Form.Item>
                </>
              ),
            },
            {
              key: 'refs',
              label: '字典引用',
              children: (
                <Space direction="vertical" style={{ width: '100%' }}>
                  <Text>
                    参数模型：{' '}
                    {form.getFieldValue('paramModelId')
                      ? (paramModels?.items || []).find((m) => m.id === form.getFieldValue('paramModelId'))
                          ?.name || '—'
                      : '未指定'}
                  </Text>
                  <Text>指标设备类型 / 平台：{form.getFieldValue('indicatorDeviceType')} / {form.getFieldValue('indicatorPlatform')}</Text>
                  <Text>告警网元类型：{form.getFieldValue('alarmNeType')}</Text>
                  <Text type="secondary">
                    引用条目数 / 指标数 / 告警数将在 P4-04 / P4-05 / P4-06 实现统计聚合后可见。
                  </Text>
                </Space>
              ),
            },
            {
              key: 'upload',
              label: '上传策略',
              children: (
                <>
                  <Form.Item
                    name="enableFiletype11"
                    label="启用参数文件上传 (Upload FileType=11)"
                    valuePropName="checked"
                    extra="false 时跳过 Upload 流程；true 时设备不支持 SOAP Fault 触发降级到默认映射"
                  >
                    <Switch />
                  </Form.Item>
                  <Form.Item
                    label="设备属性覆盖（device_attrs_override）"
                    extra="勾选后 Intersect 时该属性以设备实际上传值为准；未勾选用默认参数模型属性"
                  >
                    <Space wrap>
                      <Form.Item name="override_access" valuePropName="checked" noStyle>
                        <Checkbox>access</Checkbox>
                      </Form.Item>
                      <Form.Item name="override_change_applies" valuePropName="checked" noStyle>
                        <Checkbox>change_applies</Checkbox>
                      </Form.Item>
                      <Form.Item name="override_min_value" valuePropName="checked" noStyle>
                        <Checkbox>min_value</Checkbox>
                      </Form.Item>
                      <Form.Item name="override_max_value" valuePropName="checked" noStyle>
                        <Checkbox>max_value</Checkbox>
                      </Form.Item>
                      <Form.Item name="override_data_type" valuePropName="checked" noStyle>
                        <Checkbox disabled>data_type（禁止覆盖）</Checkbox>
                      </Form.Item>
                    </Space>
                  </Form.Item>
                  <Form.Item
                    name="enableUnknownAlarm"
                    label="接纳未识别告警"
                    valuePropName="checked"
                    extra="true 时未匹配 alarm_definitions 的告警写 fallback (severity=Warning, is_unknown=true)；false 直接丢弃"
                  >
                    <Switch />
                  </Form.Item>
                </>
              ),
            },
            {
              key: 'patterns',
              label: '正则模式',
              disabled: !isEdit,
              children: (
                <>
                  {!isEdit ? (
                    <Text type="secondary">先保存产品后再添加正则规则</Text>
                  ) : (
                    <>
                      <Space style={{ marginBottom: 12 }}>
                        <Input
                          placeholder="新规则（如 PicoCell-LTE-.+）"
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
                              message.success('已添加');
                            } catch (er) {
                              message.error((er as Error).message);
                            }
                          }}
                        >
                          添加
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
                  )}
                </>
              ),
            },
          ]}
        />
      </Form>
    </Drawer>
  );
}
