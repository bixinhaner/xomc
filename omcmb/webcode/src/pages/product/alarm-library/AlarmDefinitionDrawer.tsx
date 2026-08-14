import { useEffect } from 'react';
import { Drawer, Form, Input, Select, Switch, Space, Button, message, Tag } from 'antd';
import {
  useCreateAlarmDefinition,
  useUpdateAlarmDefinition,
  useAlarmSeverityLevels,
} from '@core/hooks/api/useAlarmDefinitions';
import type {
  AlarmDefinition,
  CreateAlarmDefinitionInput,
  UpdateAlarmDefinitionInput,
} from '@core/types/alarmDefinition';
import { useT } from '@/hooks/useT';

const EVENT_TYPE_OPTIONS = [
  { value: 30000, labelKey: 'alarm.eventType.communication' },
  { value: 30001, labelKey: 'alarm.eventType.qualityOfService' },
  { value: 30002, labelKey: 'alarm.eventType.processingError' },
  { value: 30003, labelKey: 'alarm.eventType.device' },
  { value: 30004, labelKey: 'alarm.eventType.environment' },
  { value: 30006, labelKey: 'alarm.eventType.performance' },
] as const;

function normalizeEventType(eventType: AlarmDefinition['eventType']): number | undefined {
  if (eventType === undefined || eventType === null || eventType === '') {
    return undefined;
  }

  const value = String(eventType).trim();
  switch (value) {
    case 'communication':
      return 30000;
    case 'qualityOfService':
      return 30001;
    case 'processingError':
      return 30002;
    case 'device':
    case 'equipment':
      return 30003;
    case 'environment':
      return 30004;
    case 'performance':
    case 'service':
      return 30006;
    default: {
      const numeric = Number(value);
      return Number.isFinite(numeric) ? numeric : undefined;
    }
  }
}

interface Props {
  open: boolean;
  definition: AlarmDefinition | null;
  /** 新增模式下的默认网元类型(从二级页面 drill-down 上下文带入)。 */
  defaultNeType?: string;
  /** 新增模式下是否锁定 ne_type 输入框(由二级页面预填后禁止改写)。 */
  lockNeType?: boolean;
  onClose: () => void;
}

interface FormValues {
  identifier: string;
  neType: string;
  cnName: string;
  enName: string;
  severityCode: number;
  eventType?: number;
  cnProbableCause?: string;
  enProbableCause?: string;
  cnSuggestion?: string;
  enSuggestion?: string;
  description?: string;
  isShow: boolean;
}

export default function AlarmDefinitionDrawer({
  open,
  definition,
  defaultNeType,
  lockNeType,
  onClose,
}: Props) {
  const t = useT();
  const isEdit = Boolean(definition);
  const [form] = Form.useForm<FormValues>();
  const { data: sevData } = useAlarmSeverityLevels();
  const createMut = useCreateAlarmDefinition();
  const updateMut = useUpdateAlarmDefinition();
  const severityOptions = (sevData?.items || []).map((s) => {
    const code = Number(s.code);
    const cn = s.cnName ?? s.name ?? String(code);
    const en = s.enName ?? s.name ?? '';
    const label = en && en !== cn ? `${code} - ${cn} / ${en}` : `${code} - ${cn}`;
    return { label, value: code };
  });
  const defaultSeverityCode = severityOptions[severityOptions.length - 1]?.value ?? 4;

  useEffect(() => {
    if (!open) return;
    if (definition) {
      form.setFieldsValue({
        identifier: definition.identifier,
        neType: definition.neType,
        cnName: definition.cnName,
        enName: definition.enName,
        severityCode: Number(definition.severityCode),
        eventType: normalizeEventType(definition.eventType),
        cnProbableCause: definition.cnProbableCause,
        enProbableCause: definition.enProbableCause,
        cnSuggestion: definition.cnSuggestion,
        enSuggestion: definition.enSuggestion,
        description: definition.description,
        isShow: definition.isShow,
      });
    } else {
      form.resetFields();
      form.setFieldsValue({
        isShow: true,
        severityCode: defaultSeverityCode,
        eventType: undefined,
        neType: defaultNeType ?? '',
      });
    }
  }, [open, definition, defaultNeType, defaultSeverityCode, form, sevData]);

  const handleSave = async () => {
    try {
      const v = await form.validateFields();
      if (isEdit && definition) {
        const input: UpdateAlarmDefinitionInput = {
          neType: v.neType,
          cnName: v.cnName,
          enName: v.enName,
          severityCode: v.severityCode,
          eventType: v.eventType,
          cnProbableCause: v.cnProbableCause,
          enProbableCause: v.enProbableCause,
          cnSuggestion: v.cnSuggestion,
          enSuggestion: v.enSuggestion,
          description: v.description,
          isShow: v.isShow,
        };
        await updateMut.mutateAsync({ identifier: definition.identifier, input });
        message.success(t('common.updated'));
      } else {
        const input: CreateAlarmDefinitionInput = {
          identifier: v.identifier,
          neType: v.neType,
          cnName: v.cnName,
          enName: v.enName,
          severityCode: v.severityCode,
          eventType: v.eventType,
          cnProbableCause: v.cnProbableCause,
          enProbableCause: v.enProbableCause,
          cnSuggestion: v.cnSuggestion,
          enSuggestion: v.enSuggestion,
          description: v.description,
          isShow: v.isShow,
        };
        await createMut.mutateAsync(input);
        message.success(t('common.created'));
      }
      onClose();
    } catch (e) {
      const msg = (e as Error).message;
      if (msg) message.error(msg);
    }
  };

  return (
    <Drawer
      title={
        isEdit ? (
          <Space>
            {t('product.alarm.def.detailPrefix')}<Tag color="blue">{definition?.identifier}</Tag>
            {definition?.isUnknown && <Tag color="warning">{t('product.alarm.def.unknownTag')}</Tag>}
          </Space>
        ) : (
          t('product.alarm.def.titleCreate')
        )
      }
      placement="right"
      width={680}
      open={open}
      onClose={onClose}
      destroyOnHidden
      footer={
        <Space style={{ float: 'right' }}>
          <Button onClick={onClose}>{t('common.cancel')}</Button>
          <Button
            type="primary"
            loading={createMut.isPending || updateMut.isPending}
            onClick={() => void handleSave()}
          >
            {t('common.save')}
          </Button>
        </Space>
      }
    >
      <Form<FormValues> form={form} layout="vertical">
        <Form.Item name="identifier" label={t('product.alarm.def.identifier')} rules={[{ required: true }]}>
          <Input disabled={isEdit} placeholder={t('product.alarm.def.identifierPh')} />
        </Form.Item>
        <Form.Item name="neType" label={t('product.alarm.def.neType')} rules={[{ required: true }]}>
          <Input placeholder="eNodeB / gNodeB / BTS / ..." disabled={!isEdit && Boolean(lockNeType)} />
        </Form.Item>
        <Form.Item name="cnName" label={t('common.cnName')} rules={[{ required: true }]}>
          <Input />
        </Form.Item>
        <Form.Item name="enName" label={t('common.enName')} rules={[{ required: true }]}>
          <Input />
        </Form.Item>
        <Form.Item name="severityCode" label={t('product.alarm.severityLabel')} rules={[{ required: true }]}>
          <Select
            key={severityOptions.map((option) => option.value).join(',') || 'severity-loading'}
            options={severityOptions}
          />
        </Form.Item>
        <Form.Item name="eventType" label={t('product.alarm.def.eventType')}>
          <Select
            allowClear
            placeholder={t('product.alarm.def.eventType')}
            options={EVENT_TYPE_OPTIONS.map((option) => ({
              label: t(option.labelKey),
              value: option.value,
            }))}
          />
        </Form.Item>
        <Form.Item name="cnProbableCause" label={t('product.alarm.def.cnCause')}>
          <Input.TextArea rows={2} />
        </Form.Item>
        <Form.Item name="enProbableCause" label={t('product.alarm.def.enCause')}>
          <Input.TextArea rows={2} />
        </Form.Item>
        <Form.Item name="cnSuggestion" label={t('product.alarm.def.cnSuggestion')}>
          <Input.TextArea rows={3} />
        </Form.Item>
        <Form.Item name="enSuggestion" label={t('product.alarm.def.enSuggestion')}>
          <Input.TextArea rows={3} />
        </Form.Item>
        <Form.Item name="description" label={t('table.description')}>
          <Input.TextArea rows={2} placeholder={t('table.description')} />
        </Form.Item>
        <Form.Item name="isShow" label={t('product.alarm.def.uiShow')} valuePropName="checked">
          <Switch />
        </Form.Item>
      </Form>
    </Drawer>
  );
}
