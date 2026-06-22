/**
 * IndicatorFormModal — KPI 指标新建/编辑 UI (Issue #535, v1 webcode/Antd5)。
 *
 * 指标全生命周期管理统一进 产品中心→KPI指标库。本 Modal 承载「新建/编辑指标」:
 *   · 字段:中文名(必填) / 英文名(必填) / 归属分组(必填) / 单位 / 数据类型 /
 *     级别 / 计数器(Switch) / 描述。
 *   · 归属分组 Select 选项来自 useIndicatorGroups(把分组树拍平),必填。
 *   · create 模式:id 由 crypto.randomUUID().replace(/-/g,'') 生成,默认是自定义指标,可选组。
 *   · edit 模式且 indicator.isBuildIn 为真 → 归属分组 Select disabled + Tooltip 提示
 *     (XML 真相源会覆盖);仅自定义指标可改组。
 *
 * 契约(omcgo/internal/pm/indicator/model.go CreateIndicatorRequest):
 *   必填 en_name / cn_name / group_id(device_type 由 query 注入,前端无需传);
 *   没有 name 字段 → 建/改用 enName + cnName + groupId(不要只传 name)。
 */
import { useEffect } from 'react';
import { Modal, Form, Input, Switch, message } from 'antd';
import type { AxiosError } from 'axios';
import {
  useCreateIndicator,
  useUpdateIndicator,
} from '@core/hooks/api/useIndicatorsLibrary';
import type { DeviceType, IndicatorInfo } from '@core/types/indicatorLibrary';
import GroupTreeSelect from './GroupTreeSelect';
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
  dataType?: string;
  indicatorLevel?: string;
  isCounter?: boolean;
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
  indicator,
}: Props) {
  const t = useT();
  const [form] = Form.useForm<IndicatorFormValues>();

  const isEdit = Boolean(indicator);
  // 内置指标(is_build_in==='1')编辑时归属分组只读 — XML 真相源会覆盖,改了也无效。
  const builtinGroupReadonly = isEdit && Boolean(indicator?.isBuildIn);

  // 归属分组数据源由 GroupTreeSelect 内部 useIndicatorGroups 获取(不传 platform → 全量)。

  const createMut = useCreateIndicator();
  const updateMut = useUpdateIndicator();

  // open 切换 / 目标指标变化时同步表单初值(编辑预填,新建清空)。
  useEffect(() => {
    if (!open) return;
    if (indicator) {
      form.setFieldsValue({
        cnName: indicator.cnName ?? '',
        enName: indicator.enName ?? '',
        groupId: indicator.groupId ?? '',
        unit: indicator.unit,
        // 编辑预填数据类型用列表行的 counterType(后端 data_type_label/data_type)。
        dataType: indicator.counterType,
        indicatorLevel: indicator.indicatorLevel,
        isCounter: Boolean(indicator.isCounter),
        description: indicator.description,
      });
    } else {
      form.resetFields();
    }
  }, [open, indicator, form]);

  const handleSubmit = async () => {
    const values = await form.validateFields();
    const cnName = values.cnName.trim();
    const enName = values.enName.trim();
    try {
      if (indicator) {
        await updateMut.mutateAsync({
          deviceType,
          id: indicator.id,
          input: {
            cnName,
            enName,
            // 内置指标归属由 XML 决定,不下发 group_id(避免无效写)。
            ...(builtinGroupReadonly ? {} : { groupId: values.groupId }),
            unit: values.unit?.trim() || undefined,
            dataType: values.dataType?.trim() || undefined,
            indicatorLevel: values.indicatorLevel?.trim() || undefined,
            isCounter: values.isCounter ? '1' : '0',
            cnDescription: values.description?.trim() || undefined,
            enDescription: values.description?.trim() || undefined,
          },
        });
        message.success(t('product.kpi.indicator.updateSuccess'));
      } else {
        await createMut.mutateAsync({
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
            dataType: values.dataType?.trim() || undefined,
            indicatorLevel: values.indicatorLevel?.trim() || undefined,
            isCounter: values.isCounter ? '1' : '0',
            cnDescription: values.description?.trim() || undefined,
            enDescription: values.description?.trim() || undefined,
            operatorCode,
            // 详情态新建 → 下发 platform，后端同事务写占位 formula（避免“保存后查不到” bug）。
            platform: platform || undefined,
          },
        });
        message.success(t('product.kpi.indicator.createSuccess'));
      }
      onClose();
    } catch (e) {
      message.error(errMsg(e));
    }
  };

  return (
    <Modal
      title={
        isEdit
          ? t('product.kpi.indicator.editTitle')
          : t('product.kpi.indicator.createTitle')
      }
      open={open}
      onCancel={onClose}
      onOk={() => void handleSubmit()}
      okText={t('common.save')}
      cancelText={t('common.cancel')}
      confirmLoading={createMut.isPending || updateMut.isPending}
      destroyOnHidden
      width={560}
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
          <Input maxLength={64} />
        </Form.Item>
        <Form.Item name="dataType" label={t('product.kpi.indicator.dataTypeLabel')}>
          <Input maxLength={64} />
        </Form.Item>
        {deviceType !== 'GNB' && (
          <Form.Item name="indicatorLevel" label={t('product.kpi.indicator.levelLabel')}>
            <Input maxLength={64} />
          </Form.Item>
        )}
        <Form.Item
          name="isCounter"
          label={t('product.kpi.indicator.isCounterLabel')}
          valuePropName="checked"
        >
          <Switch />
        </Form.Item>
        <Form.Item name="description" label={t('product.kpi.indicator.descLabel')}>
          <Input.TextArea rows={3} maxLength={512} />
        </Form.Item>
      </Form>
    </Modal>
  );
}
