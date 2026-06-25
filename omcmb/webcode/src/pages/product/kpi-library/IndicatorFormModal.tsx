/**
 * IndicatorFormModal — KPI 指标新建/编辑 UI (Issue #535 + #640, v1 webcode/Antd5)。
 *
 * 指标全生命周期管理统一进 产品中心→KPI指标库。本 Modal 承载「新建/编辑指标」:
 *   · 字段:中文名(必填) / 英文名(必填) / 归属分组(必填) / 单位 / 统计类型(statisType) /
 *     级别 / 指标类型(indicatorType：直接采集/公式计算) / 描述。
 *   · 公式维护区「PlatformFormulasSection」的可见性按「本次开弹是否为新建」区分：
 *     · 新建态（入场 propIndicator=null）+ kpi 类型：公式区立刻出现（local 模式）。
 *       用户在 drafts 里配 1~N 条平台公式 → 点保存 → 后端在同一事务里把指标 + 全部公式
 *       原子写入（model.CreateIndicatorRequest.Formulas + service.BatchCreate）→ Modal 关闭。
 *       校验：kpi 类型至少配 1 条公式（issue #640）。
 *     · 编辑态（入场 propIndicator 有值，由 IndicatorDrawer 右上「编辑」打开）：始终不显示公式区，保存后
 *       自动关闭 Modal 回到 Drawer；Drawer 已有 PlatformFormulasSection（server 模式），避免两处重复。
 *     区分使用 useRef 在 open rising edge 时锁定，本次打开期间不变。
 *   · 指标类型替代旧「计数器」Switch：Radio 二选一，语义更贴近用户认知（「数据从哪来」而非
 *     「是不是计数器」）。后端映射 isCounter='1'(counter) / '0'(kpi)。
 *   · 统计类型对齐老 OMC perf_indicators.statis_type 业务：sum/avg/max/min/pct 下拉选择，
 *     驱动后端 G5 cron 聚合（见 omcgo/internal/pm/kpi/calculator.go::AggregateByStatisType）。
 *   · 归属分组 Select 选项来自 useIndicatorGroups(把分组树拍平),必填。
 *   · create 模式:id 由 crypto.randomUUID().replace(/-/g,'') 生成,默认是自定义指标,可选组。
 *   · edit 模式且 indicator.isBuildIn 为真 → 归属分组 Select disabled + Tooltip 提示
 *     (XML 真相源会覆盖);仅自定义指标可改组。
 *
 * 契约(omcgo/internal/pm/indicator/model.go CreateIndicatorRequest):
 *   必填 en_name / cn_name / group_id(device_type 由 query 注入,前端无需传);
 *   没有 name 字段 → 建/改用 enName + cnName + groupId(不要只传 name)。
 *   statis_type 后端接受为可选、传受控枚举值，本 UI 限定为 STATIS_TYPE_VALUES。
 *   arithmetic 不再由本表单维护 — 公式统一走 perf_formulas_<dt> 的多平台 CRUD。
 *   formulas（issue #640）：仅 kpi 类型新建时下发，FormulaInput[]，后端事务原子写入。
 */
import { useEffect, useRef, useState } from 'react';
import { Modal, Form, Input, Select, Radio, Tooltip, Divider, Button, message } from 'antd';
import type { AxiosError } from 'axios';
import {
  useCreateIndicator,
  useUpdateIndicator,
} from '@core/hooks/api/useIndicatorsLibrary';
import type {
  DeviceType,
  FormulaDraft,
  IndicatorInfo,
  IndicatorTypeValue,
} from '@core/types/indicatorLibrary';
import {
  STATIS_TYPE_VALUES,
  INDICATOR_UNIT_OPTIONS,
  INDICATOR_LEVEL_OPTIONS,
  INDICATOR_TYPE_OPTIONS,
} from '@core/types/indicatorLibrary';
import GroupTreeSelect from './GroupTreeSelect';
import PlatformFormulasSection from './PlatformFormulasSection';
import { useT } from '@/hooks/useT';

interface Props {
  open: boolean;
  onClose: () => void;
  deviceType: DeviceType;
  operatorCode?: string;
  // 详情态（URL ?platform= 锁定）时从父级透传；新建时会随 payload 下发给后端，后端
  // 同事务内写一行占位公式 → 避免详情列表 platform_name EXISTS 过滤掉刚建的指标。
  // 列表态（主页全局新建、IndicatorDrawer 里的“编辑基本信息”）不传 → 向后兼容不写公式。
  platform?: string;
  // null/undefined → 新建模式;有值 → 编辑模式(预填该行)。
  indicator?: IndicatorInfo | null;
}

interface IndicatorFormValues {
  cnName: string;
  enName: string;
  groupId: string;
  unit?: string;
  statisType?: string;
  indicatorLevel?: string;
  // 指标类型（替代旧 isCounter Switch）：initialValue='kpi'。新建态选 'kpi' 保存后转编辑态会出现 PlatformFormulasSection。
  indicatorType: IndicatorTypeValue;
  description?: string;
}


function errMsg(e: unknown): string {
  const ax = e as AxiosError<{ msg?: string; message?: string }>;
  return (
    ax.response?.data?.msg ??
    ax.response?.data?.message ??
    (e instanceof Error ? e.message : String(e))
  );
}

export default function IndicatorFormModal({
  open,
  onClose,
  deviceType,
  operatorCode,
  platform,
  indicator: propIndicator,
}: Props) {
  const t = useT();
  const [form] = Form.useForm<IndicatorFormValues>();

  // currentIndicator 本地态：新建态起点 null，create 成功后注入返回值转「编辑态」，避免弹窗关闭后要去列表再点一下才能加公式。
  const [currentIndicator, setCurrentIndicator] = useState<IndicatorInfo | null>(
    propIndicator ?? null,
  );

  // drafts — 新建态 + kpi 类型时的本地公式草稿(issue #640 C 方案)。父组件持有,
  // PlatformFormulasSection 以 mode='local' 渲染、增删改回写。提交时随 createIndicator
  // 一并下发,后端在同一事务里把指标 + 全部公式原子写入。
  const [drafts, setDrafts] = useState<FormulaDraft[]>([]);

  // openedInCreate：本次「开弹」是否以新建态入场（propIndicator == null）。
  // ref 锁定本次打开期间不变 — create 成功后 currentIndicator 转编辑态时仍让公式区可见；
  // 编辑态（从 IndicatorDrawer 点「编辑」进来）始终为 false → 公式区不出现，避免与 Drawer 重复。
  const openedInCreateRef = useRef<boolean>(propIndicator == null);

  const isEdit = Boolean(currentIndicator);
  // 内置指标(is_build_in==='1')编辑时归属分组只读 — XML 真相源会覆盖,改了也无效。
  const builtinGroupReadonly = isEdit && Boolean(currentIndicator?.isBuildIn);

  // 归属分组数据源由 GroupTreeSelect 内部 useIndicatorGroups 获取(不传 platform → 全量)。

  const createMut = useCreateIndicator();
  const updateMut = useUpdateIndicator();

  // 跟踪 indicatorType，决定公式区是否显示（counter 类型不需公式）。
  const watchedType = Form.useWatch('indicatorType', form);

  // open 切换 / 目标指标变化时同步本地态 + 表单初值。
  // 同时在 open 上升沿记录本次「是不是新建态」（锁定本次打开生命周期）。
  useEffect(() => {
    if (!open) return;
    openedInCreateRef.current = propIndicator == null;
    setCurrentIndicator(propIndicator ?? null);
    if (propIndicator) {
      form.setFieldsValue({
        cnName: propIndicator.cnName ?? '',
        enName: propIndicator.enName ?? '',
        groupId: propIndicator.groupId ?? '',
        unit: propIndicator.unit,
        // 编辑预填统计类型，后端 BackendIndicator.statis_type 原始透传。
        statisType: propIndicator.statisType,
        indicatorLevel: propIndicator.indicatorLevel,
        // 指标类型按后端 isCounter 字段反推：true → counter、false → kpi。
        indicatorType: propIndicator.isCounter ? 'counter' : 'kpi',
        description: propIndicator.description,
      });
    } else {
      // 新建默认 kpi（公式计算）—老 OMC 自定义指标几乎都是派生 KPI，Counter 需与设备硬件埋点对齐不可随意新建。
      form.resetFields();
      form.setFieldsValue({ indicatorType: 'kpi' });
      // 新建态每次开弹清空 drafts(避免上次未提交的草稿污染本次)。
      setDrafts([]);
    }
  }, [open, propIndicator, form]);

  const handleSubmit = async () => {
    const values = await form.validateFields();
    const cnName = values.cnName.trim();
    const enName = values.enName.trim();
    // 指标类型 → isCounter 映射（counter='1' / kpi='0'）。arithmetic 不再由本表单携带 —
    // 新建态下公式由下方 PlatformFormulasSection 维护；编辑态公式由背后 Drawer 维护。
    const indicatorType: IndicatorTypeValue = values.indicatorType ?? 'kpi';
    const isCounterFlag = indicatorType === 'counter' ? '1' : '0';
    try {
      if (currentIndicator) {
        const updated = await updateMut.mutateAsync({
          deviceType,
          id: currentIndicator.id,
          input: {
            cnName,
            enName,
            // 内置指标归属由 XML 决定,不下发 group_id(避免无效写)。
            ...(builtinGroupReadonly ? {} : { groupId: values.groupId }),
            unit: values.unit?.trim() || undefined,
            statisType: values.statisType || undefined,
            indicatorLevel: values.indicatorLevel?.trim() || undefined,
            isCounter: isCounterFlag,
            cnDescription: values.description?.trim() || undefined,
            enDescription: values.description?.trim() || undefined,
          },
        });
        // updated 某些字段会被后端转换（如 isCounter 自动修正），同步本地态。
        setCurrentIndicator(updated);
        message.success(t('product.kpi.indicator.updateSuccess'));
        // 编辑态（从 Drawer 进来）：保存后自动关闭 Modal 回到 Drawer；Drawer 会重拉公式。
        // 新建态「转编辑」后的后续保存不关弹，让用户继续配公式。
        if (!openedInCreateRef.current) {
          onClose();
        }
      } else {
        // C 方案(issue #640):kpi 类型新建必须至少配 1 条公式 — 避免「直接采集」之外
        // 的派生 KPI 落库时无任何公式可算,后续在详情页才发现需要回头补。
        if (indicatorType === 'kpi' && drafts.length === 0) {
          message.error(t('product.kpi.indicator.formulaAtLeastOne'));
          return;
        }
        const created = await createMut.mutateAsync({
          deviceType,
          input: {
            id: crypto.randomUUID().replace(/-/g, ''),
            // 后端 payload 映射不发 name(只认 en_name/cn_name/group_id);
            // name 仅为满足前端类型契约,取中文名占位。
            name: cnName,
            cnName,
            enName,
            groupId: values.groupId,
            unit: values.unit?.trim() || undefined,
            statisType: values.statisType || undefined,
            indicatorLevel: values.indicatorLevel?.trim() || undefined,
            isCounter: isCounterFlag,
            cnDescription: values.description?.trim() || undefined,
            enDescription: values.description?.trim() || undefined,
            operatorCode,
            // kpi 类型走 formulas(真实公式集合,事务原子写入);counter 类型保留旧的
            // platform 占位逻辑 — 详情态进入时 URL ?platform= 已锁定,占位让该平台下
            // 详情列表的 platform_name EXISTS 过滤能查到刚建的 counter 指标。
            ...(indicatorType === 'kpi' && drafts.length > 0
              ? { formulas: drafts }
              : { platform: platform || undefined }),
          },
        });
        // 创建成功 → 通知父级刷新 + 关弹。drafts 已经在同一事务里落库,无需再转编辑态补。
        setCurrentIndicator(created);
        message.success(t('product.kpi.indicator.createSuccess'));
        onClose();
      }
    } catch (e) {
      message.error(errMsg(e));
    }
  };

  const submitting = createMut.isPending || updateMut.isPending;
  // 公式区可见性(issue #640 C 方案):
  //   · 新建态(openedInCreateRef.current=true) + kpi 类型 → 立刻渲染 local 模式公式区,
  //     用户随建随配,提交时随 createIndicator 一并原子写入。
  //   · 编辑态(openedInCreateRef.current=false) → 始终不渲染 — 避免与 Drawer 背后的
  //     PlatformFormulasSection(server 模式)重复。
  //   · counter 类型不需公式,隐藏。
  const showFormulasLocal = openedInCreateRef.current && watchedType === 'kpi';

  return (
    <Modal
      title={
        isEdit
          ? t('product.kpi.indicator.editTitle')
          : t('product.kpi.indicator.createTitle')
      }
      open={open}
      onCancel={onClose}
      destroyOnHidden
      width={620}
      footer={[
        <Button key="close" onClick={onClose} disabled={submitting}>
          {currentIndicator ? t('common.close') : t('common.cancel')}
        </Button>,
        <Button
          key="ok"
          type="primary"
          loading={submitting}
          onClick={() => void handleSubmit()}
        >
          {t('common.save')}
        </Button>,
      ]}
    >
      <Form form={form} layout="vertical">
        <Form.Item
          name="cnName"
          label={t('product.kpi.indicator.cnNameLabel')}
          rules={[{ required: true, message: t('product.kpi.indicator.cnNameRequired') }]}
        >
          <Input maxLength={128} />
        </Form.Item>
        <Form.Item
          name="enName"
          label={t('product.kpi.indicator.enNameLabel')}
          rules={[{ required: true, message: t('product.kpi.indicator.enNameRequired') }]}
        >
          <Input maxLength={128} />
        </Form.Item>
        <Form.Item
          name="groupId"
          label={t('product.kpi.indicator.groupLabel')}
          rules={[{ required: true, message: t('product.kpi.indicator.groupRequired') }]}
          extra={builtinGroupReadonly ? t('product.kpi.indicator.builtinGroupReadonly') : undefined}
        >
          {/* GroupTreeSelect 是 Antd TreeSelect 的薄包装(value/onChange 可省);
              作为 Form.Item 直接子节点,Form.Item 经 cloneElement 注入 value/onChange
              接管表单受控。内置只读提示走 extra,不再包 Tooltip。 */}
          <GroupTreeSelect
            deviceType={deviceType}
            operatorCode={operatorCode}
            disabled={builtinGroupReadonly}
            placeholder={t('product.kpi.indicator.groupRequired')}
            style={{ width: '100%' }}
          />
        </Form.Item>
        <Form.Item name="unit" label={t('product.kpi.indicator.unitLabel')}>
          {/* Unit 下拉 — 选项集 = 老 OMC indicator_unit 字典 + 现有 XML unitId 全集（见
              frontend-core INDICATOR_UNIT_OPTIONS）。showSearch 支持输入过滤；编辑预填值
              不在枚举时不强制清空（antd Select 容忍受控外值）。 */}
          <Select
            allowClear
            showSearch
            placeholder={t('product.kpi.indicator.unitRequired')}
            options={INDICATOR_UNIT_OPTIONS.map((v) => ({ value: v, label: v }))}
          />
        </Form.Item>
        <Form.Item name="statisType" label={t('product.kpi.indicator.statisTypeLabel')}>
          <Select
            allowClear
            placeholder={t('product.kpi.indicator.statisTypeRequired')}
            options={STATIS_TYPE_VALUES.map((v) => ({
              value: v,
              label: t(`product.kpi.indicator.statisType.${v}`),
            }))}
          />
        </Form.Item>
        {deviceType !== 'GNB' && (
          <Form.Item name="indicatorLevel" label={t('product.kpi.indicator.levelLabel')}>
            {/* Level 下拉 — 对外只暴露 Device / PLMN 两项；老数据存量值 'both' 通过
                allowClear 允许保留显示（Select 容忍受控外值），但下拉不作为可选项。 */}
            <Select
              allowClear
              placeholder={t('product.kpi.indicator.levelRequired')}
              options={INDICATOR_LEVEL_OPTIONS.map((o) => ({ value: o.value, label: o.label }))}
            />
          </Form.Item>
        )}
        <Form.Item
          name="indicatorType"
          label={t('product.kpi.indicator.typeLabel')}
          rules={[{ required: true }]}
          initialValue="kpi"
        >
          {/* 指标类型二选一（替代旧「计数器」Switch）。
              Tooltip 包裹文本节点，鼠标悬停出现说明，不影响 Radio 点击热区。 */}
          <Radio.Group>
            {INDICATOR_TYPE_OPTIONS.map((o) => (
              <Radio key={o.value} value={o.value}>
                <Tooltip title={t(`product.kpi.indicator.type${o.value === 'counter' ? 'Counter' : 'Kpi'}Hint`)}>
                  <span>{t(`product.kpi.indicator.type${o.value === 'counter' ? 'Counter' : 'Kpi'}`)}</span>
                </Tooltip>
              </Radio>
            ))}
          </Radio.Group>
        </Form.Item>
        <Form.Item name="description" label={t('product.kpi.indicator.descLabel')}>
          <Input.TextArea rows={3} maxLength={512} />
        </Form.Item>
      </Form>

      {/* 公式维护区(issue #640 C 方案) — 新建态 + kpi 类型时以 local 模式立刻出现,
          drafts 由本组件 state 持有,PlatformFormulasSection 内部增删改回写;提交时
          createIndicator 会把 drafts 序列化到 formulas 字段,后端事务原子写入。
          编辑态(IndicatorDrawer 进入)始终不渲染本区 — 编辑路径的公式由 Drawer 背后的
          PlatformFormulasSection(server 模式)维护,避免重复。 */}
      {showFormulasLocal ? (
        <>
          <Divider style={{ margin: '12px 0' }} />
          <PlatformFormulasSection
            mode="local"
            deviceType={deviceType}
            value={drafts}
            onChange={setDrafts}
          />
        </>
      ) : null}
    </Modal>
  );
}
