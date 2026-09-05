import { useEffect, useState } from 'react';
import { Alert, Checkbox, Form, Input, InputNumber, Modal, Radio, Select, Spin } from 'antd';
import { useQuery } from '@tanstack/react-query';
import { useUserStore } from '@core/store/userStore';
import { deviceApi } from '@core/services/api/deviceApi';
import type { AssistantDefinition } from '@core/types/assistant';
import { useT } from '@/hooks/useT';
import styles from './index.module.css';

interface Props { definition: AssistantDefinition; open: boolean; busy: boolean; onClose(): void; onSave(value: AssistantDefinition): Promise<void> }
export default function AssistantEditor({ definition, open, busy, onClose, onSave }: Props) {
  const t = useT(); const owner = useUserStore((state) => state.currentUser?.id ?? 'current');
  const [form] = Form.useForm<AssistantDefinition>();
  const [search, setSearch] = useState('');
  const [query, setQuery] = useState('');
  useEffect(() => { const timer = setTimeout(() => setQuery(search), 300); return () => clearTimeout(timer); }, [search]);
  useEffect(() => { if (open) form.setFieldsValue(structuredClone(definition)); }, [open, definition, form]);
  const kind = Form.useWatch(['trigger', 'kind'], form) ?? definition.trigger.kind;
  const scopeKind = Form.useWatch(['scope', 'kind'], form) ?? definition.scope.kind;
  const devices = useQuery({ queryKey: ['assistant-device-search', owner, query], queryFn: () => deviceApi.getList({ page: 1, pageSize: 20, searchText: query }), enabled: open && scopeKind === 'device' });
  const options = (devices.data?.items ?? []).map((device) => ({ value: device.id, label: `${device.name || device.sn} · ${device.sn}` }));
  if (definition.scope.deviceId && !options.some((o) => o.value === definition.scope.deviceId)) options.unshift({ value: definition.scope.deviceId, label: definition.scope.label || definition.scope.deviceId });
  const required = [{ required: true, message: t('assistant.fieldRequired') }];
  const submit = async () => {
    try {
      const fields = await form.validateFields();
      const next: AssistantDefinition = {
        ...definition, ...fields, scope: { ...definition.scope, ...fields.scope },
        trigger: { ...definition.trigger, ...fields.trigger, conditions: definition.trigger.conditions ?? [] },
      };
      if (next.scope.kind === 'visible') next.scope = { kind: 'visible' };
      else next.scope.label = options.find((o) => o.value === next.scope.deviceId)?.label || next.scope.deviceId;
      // Do not keep event predicates when the user explicitly chooses another trigger.
      if (next.trigger.kind !== 'event' || next.trigger.eventType !== definition.trigger.eventType) next.trigger.conditions = [];
      await onSave(next);
    } catch (error) {
      if (error && typeof error === 'object' && 'errorFields' in error) form.scrollToField((error as { errorFields: Array<{ name: string[] }> }).errorFields[0]?.name ?? []);
    }
  };
  return <Modal open={open} title={t('assistant.edit')} width={620} onCancel={onClose} onOk={() => void submit()} confirmLoading={busy} okText={t('common.save')} cancelText={t('common.cancel')} maskClosable={!busy} closable={!busy}>
    <Alert type="info" showIcon title={t('assistant.editNotice')} className={styles.editorNotice} />
    <Form form={form} layout="vertical" initialValues={definition} disabled={busy} preserve>
      <Form.Item name="name" label={t('assistant.name')} rules={required}><Input maxLength={100} /></Form.Item>
      <Form.Item name="goal" label={t('assistant.goal')} rules={required}><Input.TextArea autoSize={{ minRows: 3, maxRows: 8 }} maxLength={8000} /></Form.Item>
      <Form.Item name={['scope', 'kind']} label={t('assistant.scope')}><Radio.Group options={[{ value: 'visible', label: t('assistant.scope.visible') }, { value: 'device', label: t('assistant.scope.device') }]} /></Form.Item>
      {scopeKind === 'device' && <>
        <Form.Item name={['scope', 'deviceId']} rules={required} label={t('assistant.scope.device')}>
          <Select showSearch={{ filterOption: false, onSearch: setSearch }} placeholder={t('assistant.devicePlaceholder')} options={options} loading={devices.isFetching} notFoundContent={devices.isFetching ? <Spin size="small" /> : t('common.noData')} />
        </Form.Item>
        {devices.isError && <Alert type="error" showIcon title={t('assistant.deviceSearchError')} />}
        <p className={styles.muted}>{t('assistant.deviceScopeNotice')}</p>
      </>}
      <Form.Item name={['trigger', 'kind']} label={t('assistant.trigger')} rules={required}>
        <Select options={['manual', 'interval', 'schedule', 'event'].map((value) => ({ value, label: t(value === 'manual' ? 'assistant.trigger.manual' : `assistant.triggerOption.${value}`) }))} />
      </Form.Item>
      {kind === 'interval' && <Form.Item name={['trigger', 'intervalMinutes']} label={t('assistant.interval')} rules={required}><InputNumber min={5} max={10080} style={{ width: '100%' }} /></Form.Item>}
      {kind === 'schedule' && <>
        <div className={styles.formRow}>
          <Form.Item name={['trigger', 'time']} label={t('assistant.time')} rules={required}><Input type="time" /></Form.Item>
          <Form.Item name={['trigger', 'timezone']} label={t('assistant.timezone')} rules={required}><Input placeholder={Intl.DateTimeFormat().resolvedOptions().timeZone} /></Form.Item>
        </div>
        <Form.Item name={['trigger', 'weekdays']} label={t('assistant.weekdays')} rules={required}><Checkbox.Group options={[1, 2, 3, 4, 5, 6, 0].map((value) => ({ value, label: t(`assistant.days.${value}`) }))} /></Form.Item>
      </>}
      {kind === 'event' && <Form.Item name={['trigger', 'eventType']} label={t('assistant.trigger.event')} rules={required}>
        <Select options={['omc.alarm.severe-raised.v1', 'omc.task.failed.v1'].map((value) => ({ value, label: t(`assistant.event.${value}`) }))} />
      </Form.Item>}
      <Form.Item name="notify" label={t('assistant.notify')}><Radio.Group options={['findings', 'always'].map((value) => ({ value, label: t(`assistant.notify.${value}`) }))} /></Form.Item>
      <Form.Item name="cooldownMinutes" label={t('assistant.cooldown', { minutes: definition.cooldownMinutes })}><InputNumber min={0} max={10080} /></Form.Item>
    </Form>
  </Modal>;
}
