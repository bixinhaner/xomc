/**
 * PlatformFormulasSection — KPI 指标「全平台公式」CRUD 区（v1 webcode/Antd5）。
 *
 * 共享组件 — 同时供 IndicatorDrawer（详情抽屉）与 IndicatorFormModal（新建/编辑弹窗）使用，
 * 保证新建态与编辑态公式维护方式完全一致：按 platform 维度的多条公式 CRUD（不是单条
 * arithmetic textarea）。
 *
 * 数据契约：
 *   · 列表/CRUD 数据源 = rela_platform_indicator_formula_<dt>（按 platform 多行）
 *   · useFormulas(deviceType, indicatorId) → { items: PlatformFormula[] }
 *   · useUpsertFormula() → 按 platform 主键 upsert（同 platform 第二次保存即覆盖）
 *   · useDeleteFormula() → 按 platform 删除
 *
 * 前置条件：必须有 indicatorId。新建模式调用方应在 createIndicator 成功后把返回的
 *   indicator 注入本地 state，再渲染本组件（弹窗不关闭、转入「编辑态」）。
 */
import { useMemo, useState } from 'react';
import {
  Table,
  Button,
  Space,
  Input,
  Select,
  Modal,
  Form,
  Popconfirm,
  message,
  Tag,
} from 'antd';
import { PlusOutlined, DeleteOutlined, EditOutlined } from '@ant-design/icons';
import {
  useFormulas,
  useUpsertFormula,
  useDeleteFormula,
  usePlatformList,
} from '@core/hooks/api/useIndicatorsLibrary';
import type { DeviceType, PlatformFormula } from '@core/types/indicatorLibrary';
import { useT } from '@/hooks/useT';

// 「所有平台共用」约定值，与后端 indicator.PlatformAll 常量保持一致（XML 字典里
// enb/ALL.xml 同义；KPIRoute 装配按 (具体平台, ALL) 并查，具体平台优先）。
const PLATFORM_ALL = 'ALL';

interface FormulaFormValues {
  platform: string;
  formula: string;
}

interface Props {
  deviceType: DeviceType;
  indicatorId: string;
}

export default function PlatformFormulasSection({ deviceType, indicatorId }: Props) {
  const t = useT();
  const { data: formulasData } = useFormulas(deviceType, indicatorId);
  // 平台下拉数据源：后端 GET /api/v1/indicators/platforms 返回该 deviceType 下公式表里
  // distinct 出来的全部 platform_name（不含未使用过的产品 platform）。
  const { data: platformsData } = usePlatformList(deviceType);
  const upsertMut = useUpsertFormula();
  const deleteMut = useDeleteFormula();

  const [editing, setEditing] = useState<PlatformFormula | null>(null);
  const [creating, setCreating] = useState(false);
  const [form] = Form.useForm<FormulaFormValues>();

  const formulas = formulasData?.items || [];

  // 平台选项：始终把 ALL 放在第一位（即使数据库里还没人选过 ALL，也要可选）；
  // 其余按后端返回的 distinct platform_name 列出。Set 兜底去重。
  const platformOptions = useMemo(() => {
    const fromApi = platformsData?.items ?? [];
    const ordered = [PLATFORM_ALL, ...fromApi.filter((p) => p !== PLATFORM_ALL)];
    const seen = new Set<string>();
    return ordered
      .filter((p) => {
        if (seen.has(p)) return false;
        seen.add(p);
        return true;
      })
      .map((value) => ({
        value,
        label:
          value === PLATFORM_ALL
            ? `${value}${t('product.kpi.platformAllSuffix')}`
            : value,
      }));
  }, [platformsData, t]);

  const formulaColumns = [
    {
      title: t('product.kpi.indicator.platformName'),
      dataIndex: 'platformName',
      width: 180,
      render: (v: string) => <Tag color="cyan">{v}</Tag>,
    },
    {
      title: t('product.kpi.indicator.formulaLabel'),
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
              deleteMut
                .mutateAsync({ deviceType, indicatorId, platform: row.platformName })
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
      // 基础校验：括号匹配（后端 service 还会做完整语法校验）。
      let depth = 0;
      for (const ch of v.formula) {
        if (ch === '(') depth++;
        else if (ch === ')') depth--;
        if (depth < 0) throw new Error(t('product.kpi.formulaBracketMismatch'));
      }
      if (depth !== 0) throw new Error(t('product.kpi.formulaBracketMismatch'));
      await upsertMut.mutateAsync({
        deviceType,
        indicatorId,
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
    <>
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
        title={
          editing
            ? t('product.kpi.editFormulaTitle', { name: editing.platformName })
            : t('product.kpi.newPlatformFormula')
        }
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
            {/* 平台下拉：选 ALL = 对所有平台共用（与后端 indicator.PlatformAll 对齐，
                同指标在具体平台另有公式时具体平台优先）。编辑现有公式时 platform 是主键
                不可改 → disabled；同时 showSearch 支持键盘筛选。 */}
            <Select
              showSearch
              placeholder={t('product.kpi.indicator.platformName')}
              options={platformOptions}
              disabled={Boolean(editing)}
              optionFilterProp="label"
            />
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
  );
}
