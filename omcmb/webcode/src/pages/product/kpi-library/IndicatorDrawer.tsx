import { useState } from 'react';
import {
  Drawer,
  Descriptions,
  Tag,
  Table,
  Button,
  Space,
  Input,
  Modal,
  Form,
  Popconfirm,
  message,
  Empty,
} from 'antd';
import { PlusOutlined, DeleteOutlined, EditOutlined } from '@ant-design/icons';
import {
  useFormulas,
  useUpsertFormula,
  useDeleteFormula,
} from '@core/hooks/api/useIndicatorsLibrary';
import type { DeviceType, IndicatorInfo, PlatformFormula } from '@core/types/indicatorLibrary';
import { useT } from '@/hooks/useT';

interface Props {
  open: boolean;
  deviceType: DeviceType;
  indicator: IndicatorInfo | null;
  onClose: () => void;
}

interface FormulaFormValues {
  platform: string;
  formula: string;
}

export default function IndicatorDrawer({ open, deviceType, indicator, onClose }: Props) {
  const t = useT();
  const { data: formulasData } = useFormulas(deviceType, indicator?.id);
  const upsertMut = useUpsertFormula();
  const deleteMut = useDeleteFormula();

  const [editing, setEditing] = useState<PlatformFormula | null>(null);
  const [creating, setCreating] = useState(false);
  const [form] = Form.useForm<FormulaFormValues>();

  const formulas = formulasData?.items || [];

  const formulaColumns = [
    {
      title: 'platform',
      dataIndex: 'platformName',
      width: 180,
      render: (v: string) => <Tag color="cyan">{v}</Tag>,
    },
    {
      title: 'formula',
      dataIndex: 'formula',
      ellipsis: true,
      render: (v: string) => <code style={{ fontSize: 12 }}>{v}</code>,
    },
    {
      title: '操作',
      width: 120,
      render: (_: unknown, row: PlatformFormula) => (
        <Space>
          <Button
            size="small"
            icon={<EditOutlined />}
            onClick={() => {
              setEditing(row);
              form.setFieldsValue({ platform: row.platformName, formula: row.formula });
            }}
          />
          <Popconfirm
            title={`确认删除「${row.platformName}」公式？`}
            onConfirm={() =>
              indicator &&
              deleteMut
                .mutateAsync({ deviceType, indicatorId: indicator.id, platform: row.platformName })
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
    if (!indicator) return;
    try {
      const v = await form.validateFields();
      // 基础校验：括号匹配
      let depth = 0;
      for (const ch of v.formula) {
        if (ch === '(') depth++;
        else if (ch === ')') depth--;
        if (depth < 0) throw new Error('公式括号不匹配');
      }
      if (depth !== 0) throw new Error('公式括号不匹配');
      await upsertMut.mutateAsync({
        deviceType,
        indicatorId: indicator.id,
        platform: v.platform.trim(),
        formula: v.formula.trim(),
      });
      message.success(editing ? '已更新' : '已创建');
      setEditing(null);
      setCreating(false);
      form.resetFields();
    } catch (e) {
      const msg = (e as Error).message;
      if (msg) message.error(msg);
    }
  };

  return (
    <Drawer
      title={indicator ? `指标详情：${indicator.id} ${indicator.cnName || indicator.name}` : '指标详情'}
      placement="right"
      width={760}
      open={open}
      onClose={onClose}
      destroyOnHidden
    >
      {!indicator ? (
        <Empty />
      ) : (
        <>
          <Descriptions column={2} size="small" bordered style={{ marginBottom: 16 }}>
            <Descriptions.Item label="ID">{indicator.id}</Descriptions.Item>
            <Descriptions.Item label={t('product.kpi.indicator.deviceType')}>{indicator.deviceType}</Descriptions.Item>
            <Descriptions.Item label={t('common.cnName')}>{indicator.cnName || '—'}</Descriptions.Item>
            <Descriptions.Item label={t('common.enName')}>{indicator.enName || '—'}</Descriptions.Item>
            <Descriptions.Item label={t('product.kpi.indicator.group')}>{indicator.groupName || indicator.groupId || '—'}</Descriptions.Item>
            <Descriptions.Item label={t('common.unit')}>{indicator.unit || '—'}</Descriptions.Item>
            <Descriptions.Item label={t('product.kpi.indicator.counterType')}>{indicator.counterType || '—'}</Descriptions.Item>
            <Descriptions.Item label={t('product.kpi.indicator.level')}>{indicator.indicatorLevel || '—'}</Descriptions.Item>
            <Descriptions.Item label={t('product.kpi.indicator.enabledTag')} span={2}>
              {indicator.isEnabled ? <Tag color="success">{t('product.kpi.indicator.enabledTag')}</Tag> : <Tag>{t('product.kpi.indicator.disabledTag')}</Tag>}
            </Descriptions.Item>
            <Descriptions.Item label={t('common.description')} span={2}>
              {indicator.description || '—'}
            </Descriptions.Item>
          </Descriptions>

          <Space style={{ marginBottom: 12, justifyContent: 'space-between', width: '100%' }}>
            <strong>{t('product.kpi.indicator.formulasTitle')}</strong>
            <Button
              size="small"
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => {
                setCreating(true);
                setEditing(null);
                form.resetFields();
              }}
            >
              新增公式
            </Button>
          </Space>

          <Table<PlatformFormula>
            rowKey="platformName"
            columns={formulaColumns}
            dataSource={formulas}
            size="small"
            pagination={false}
          />

          <Modal
            title={editing ? `编辑公式：${editing.platformName}` : '新增平台公式'}
            open={Boolean(editing) || creating}
            onOk={() => void handleSave()}
            onCancel={() => {
              setEditing(null);
              setCreating(false);
              form.resetFields();
            }}
            confirmLoading={upsertMut.isPending}
            destroyOnHidden
            width={620}
          >
            <Form form={form} layout="vertical">
              <Form.Item
                name="platform"
                label={t('product.kpi.indicator.platformName')}
                rules={[{ required: true, message: '必填' }]}
                extra="对应 product 的 indicator_platform 字段；如 enb-default / gnb-comba"
              >
                <Input disabled={Boolean(editing)} />
              </Form.Item>
              <Form.Item
                name="formula"
                label={t('product.kpi.indicator.formulaLabel')}
                rules={[{ required: true, message: '必填' }]}
                extra="支持基础四则、括号、计数器名；前端做括号匹配校验，后端做完整语法验证"
              >
                <Input.TextArea rows={4} placeholder={t('product.kpi.indicator.formulaPh')} />
              </Form.Item>
            </Form>
          </Modal>
        </>
      )}
    </Drawer>
  );
}
