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
      destroyOnClose
    >
      {!indicator ? (
        <Empty />
      ) : (
        <>
          <Descriptions column={2} size="small" bordered style={{ marginBottom: 16 }}>
            <Descriptions.Item label="ID">{indicator.id}</Descriptions.Item>
            <Descriptions.Item label="设备类型">{indicator.deviceType}</Descriptions.Item>
            <Descriptions.Item label="中文名">{indicator.cnName || '—'}</Descriptions.Item>
            <Descriptions.Item label="英文名">{indicator.enName || '—'}</Descriptions.Item>
            <Descriptions.Item label="分组">{indicator.groupName || indicator.groupId || '—'}</Descriptions.Item>
            <Descriptions.Item label="单位">{indicator.unit || '—'}</Descriptions.Item>
            <Descriptions.Item label="计数器类型">{indicator.counterType || '—'}</Descriptions.Item>
            <Descriptions.Item label="级别">{indicator.indicatorLevel || '—'}</Descriptions.Item>
            <Descriptions.Item label="启用" span={2}>
              {indicator.isEnabled ? <Tag color="success">已启用</Tag> : <Tag>未启用</Tag>}
            </Descriptions.Item>
            <Descriptions.Item label="描述" span={2}>
              {indicator.description || '—'}
            </Descriptions.Item>
          </Descriptions>

          <Space style={{ marginBottom: 12, justifyContent: 'space-between', width: '100%' }}>
            <strong>全平台公式 / Per-Platform Formulas</strong>
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
            destroyOnClose
            width={620}
          >
            <Form form={form} layout="vertical">
              <Form.Item
                name="platform"
                label="平台名 (platform_name)"
                rules={[{ required: true, message: '必填' }]}
                extra="对应 product 的 indicator_platform 字段；如 enb-default / gnb-comba"
              >
                <Input disabled={Boolean(editing)} />
              </Form.Item>
              <Form.Item
                name="formula"
                label="公式 (formula)"
                rules={[{ required: true, message: '必填' }]}
                extra="支持基础四则、括号、计数器名；前端做括号匹配校验，后端做完整语法验证"
              >
                <Input.TextArea rows={4} placeholder="如 C1 / (C1 + C2) * 100" />
              </Form.Item>
            </Form>
          </Modal>
        </>
      )}
    </Drawer>
  );
}
