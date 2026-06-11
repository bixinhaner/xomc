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
  // 2026-06-03:启用状态由父组件按 default 启用桶(enabledSet)传入,
  // 与列表开关同源;不再用 indicator.isEnabled(未反映 default 桶)。
  enabled: boolean;
  onClose: () => void;
}

interface FormulaFormValues {
  platform: string;
  formula: string;
}

export default function IndicatorDrawer({ open, deviceType, indicator, enabled, onClose }: Props) {
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
      title: t('common.action'),
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
            title={t('product.kpi.confirmDeletePlatformFormula', { name: row.platformName })}
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
        if (depth < 0) throw new Error(t('product.kpi.formulaBracketMismatch'));
      }
      if (depth !== 0) throw new Error(t('product.kpi.formulaBracketMismatch'));
      await upsertMut.mutateAsync({
        deviceType,
        indicatorId: indicator.id,
        platform: v.platform.trim(),
        formula: v.formula.trim(),
      });
      message.success(editing ? t('common.updated') : t('common.created'));
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
      title={indicator ? t('product.kpi.indicatorDetailFull', { id: indicator.id, name: indicator.cnName || indicator.name }) : t('product.kpi.indicatorDetail')}
      placement="right"
      size={760}
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
              {enabled ? <Tag color="success">{t('product.kpi.indicator.enabledTag')}</Tag> : <Tag>{t('product.kpi.indicator.disabledTag')}</Tag>}
            </Descriptions.Item>
            {/* PM-P3:界面公式展示用编号版 arithmetic(对运维编号才是工作语言);
                标准名版 formula 退为工程内部物,见下方"全平台公式"表(保留 CRUD)。 */}
            <Descriptions.Item label={t('product.kpi.indicator.arithmetic')} span={2}>
              {indicator.arithmetic ? <code style={{ fontSize: 12 }}>{indicator.arithmetic}</code> : '—'}
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
              {t('product.kpi.newFormula')}
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
            title={editing ? t('product.kpi.editFormulaTitle', { name: editing.platformName }) : t('product.kpi.newPlatformFormula')}
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
                rules={[{ required: true, message: t('common.required') }]}
                extra={t('product.kpi.platformExtra')}
              >
                <Input disabled={Boolean(editing)} />
              </Form.Item>
              <Form.Item
                name="formula"
                label={t('product.kpi.indicator.formulaLabel')}
                rules={[{ required: true, message: t('common.required') }]}
                extra={t('product.kpi.formulaExtra')}
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
