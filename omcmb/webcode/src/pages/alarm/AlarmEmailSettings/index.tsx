import { useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Button,
  Checkbox,
  Drawer,
  Form,
  Input,
  Modal,
  Popconfirm,
  Radio,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  Typography,
  message,
} from 'antd';
import { DeleteOutlined, EditOutlined, PlusOutlined } from '@ant-design/icons';
import { useDebounce } from 'ahooks';
import type { ColumnsType } from 'antd/es/table';
import {
  useAlarmEmailSettings,
  useAlarmEmailDefaults,
  useArchiveAlarmEmailSetting,
  useCreateAlarmEmailSetting,
  useUpdateAlarmEmailSetting,
  useUpdateAlarmEmailDefaults,
} from '@core/hooks/api/useNotifications';
import { useAllAlarmDefinitions } from '@core/hooks/api/useAlarmDefinitions';
import { useDeviceGroups, useDeviceList, useDevicesByIds } from '@core/hooks/api/useDevices';
import { useUserStore } from '@core/store/userStore';
import { isNotificationRevisionConflict } from '@core/services/api/notificationApi';
import type { AlarmEmailSetting, AlarmEmailSettingPayload } from '@core/types/notification';
import { formatSystemTime } from '@core/utils/systemTime';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useT } from '@/hooks/useT';

type ScopeMode = 'device' | 'group';

interface FormValues {
  name: string;
  enabled: boolean;
  alarmIdentifiers: string[];
  severities: number[];
  technologies: string[];
  scopeMode: ScopeMode;
  deviceIds: string[];
  deviceGroupIds: string[];
  intervalMinutes: 0 | 10 | 30 | 60;
  toleranceDurationMinutes: 0 | 10 | 30 | 60;
  recipients: string;
  includeDefaultRecipients: boolean;
}

const minuteValues = [0, 10, 30, 60] as const;
const severityValues = [1, 2, 3, 4] as const;

export default function AlarmEmailSettings() {
  const t = useT();
  const isSuperAdmin = useUserStore((state) => state.currentUser?.isSuperAdmin === true);
  const settings = useAlarmEmailSettings();
  const defaults = useAlarmEmailDefaults(isSuperAdmin);
  const createMutation = useCreateAlarmEmailSetting();
  const updateMutation = useUpdateAlarmEmailSetting();
  const updateDefaultsMutation = useUpdateAlarmEmailDefaults();
  const archiveMutation = useArchiveAlarmEmailSetting();
  const alarmDefinitions = useAllAlarmDefinitions();
  const [deviceSearch, setDeviceSearch] = useState('');
  const debouncedDeviceSearch = useDebounce(deviceSearch, { wait: 300 });
  const devices = useDeviceList({
    page: 1,
    pageSize: 50,
    searchText: debouncedDeviceSearch.trim() || undefined,
  });
  const deviceGroups = useDeviceGroups();
  const [editing, setEditing] = useState<AlarmEmailSetting | null | undefined>(undefined);
  const [defaultsOpen, setDefaultsOpen] = useState(false);
  const [defaultRecipients, setDefaultRecipients] = useState('');
  const [form] = Form.useForm<FormValues>();
  const scopeMode = Form.useWatch('scopeMode', form);
  const watchedDeviceIds = Form.useWatch('deviceIds', form);
  const selectedDeviceIds = useMemo(() => watchedDeviceIds ?? [], [watchedDeviceIds]);
  const selectedDeviceQueries = useDevicesByIds(selectedDeviceIds);

  useEffect(() => {
    if (editing === undefined) return;
    form.setFieldsValue({
      name: editing?.name ?? '',
      enabled: editing?.enabled ?? false,
      alarmIdentifiers: editing?.alarmIdentifiers ?? [],
      severities: editing?.severities ?? severityValues.slice(),
      technologies: editing?.technologies ?? [],
      scopeMode: editing?.deviceGroupIds.length ? 'group' : 'device',
      deviceIds: editing?.deviceIds ?? [],
      deviceGroupIds: editing?.deviceGroupIds ?? [],
      intervalMinutes: editing?.intervalMinutes ?? 0,
      toleranceDurationMinutes: editing?.toleranceDurationMinutes ?? 0,
      recipients: editing?.recipients.join(';') ?? '',
      includeDefaultRecipients: editing?.includeDefaultRecipients ?? false,
    });
  }, [editing, form]);

  const intervalOptions = minuteValues.map((value) => ({
    value,
    label: value === 0 ? t('alarm.emailSettings.realtime') : t('alarm.emailSettings.minutes', { value }),
  }));
  const severityOptions = severityValues.map((value) => ({
    value,
    label: t(`alarm.emailSettings.severity.${value}`),
  }));
  const alarmOptions = useMemo(
    () => (alarmDefinitions.data?.items ?? []).map((item) => ({
      value: item.identifier,
      label: `${item.identifier} · ${item.cnName || item.enName || item.identifier}`,
    })),
    [alarmDefinitions.data?.items],
  );
  const deviceOptions = useMemo(() => {
    const byID = new Map(
      (devices.data?.items ?? []).map((item) => [item.id, { value: item.id, label: `${item.sn} · ${item.name || item.sn}` }]),
    );
    selectedDeviceQueries.forEach((query, index) => {
      const item = query.data;
      const id = selectedDeviceIds[index];
      if (item && id && !byID.has(id)) {
        byID.set(id, { value: id, label: `${item.sn} · ${item.name || item.sn}` });
      }
    });
    return Array.from(byID.values());
  }, [devices.data?.items, selectedDeviceIds, selectedDeviceQueries]);
  const groupOptions = useMemo(
    () => (deviceGroups.data?.groups ?? []).map((item) => ({ value: item.id, label: item.name })),
    [deviceGroups.data?.groups],
  );

  const showError = (error: unknown) => {
    void message.error(
      isNotificationRevisionConflict(error)
        ? t('notification.concurrentConflict')
        : t('notification.operationFailed', { error: error instanceof Error ? error.message : String(error) }),
    );
  };

  const openDefaults = () => {
    setDefaultRecipients((defaults.data?.recipients ?? []).join(';'));
    setDefaultsOpen(true);
  };

  const saveDefaults = async () => {
    if (!defaults.data) return;
    try {
      const recipients = defaultRecipients
        .split(/[;,\n]/)
        .map((value) => value.trim())
        .filter(Boolean);
      await updateDefaultsMutation.mutateAsync({ revision: defaults.data.revision, recipients });
      void message.success(t('notification.saveSuccess'));
      setDefaultsOpen(false);
    } catch (error) {
      showError(error);
    }
  };

  const save = async () => {
    try {
      const values = await form.validateFields();
      const recipients = values.recipients
        .split(/[;,\n]/)
        .map((value) => value.trim())
        .filter(Boolean);
      if (recipients.length === 0 && !values.includeDefaultRecipients) {
        form.setFields([{ name: 'recipients', errors: [t('alarm.emailSettings.recipientRequired')] }]);
        return;
      }
      const payload: AlarmEmailSettingPayload = {
        name: values.name.trim(),
        enabled: values.enabled,
        alarmIdentifiers: values.alarmIdentifiers ?? [],
        severities: values.severities ?? [],
        technologies: values.technologies ?? [],
        deviceIds: values.scopeMode === 'device' ? (values.deviceIds ?? []) : [],
        deviceGroupIds: values.scopeMode === 'group' ? (values.deviceGroupIds ?? []) : [],
        intervalMinutes: values.intervalMinutes,
        toleranceDurationMinutes: values.toleranceDurationMinutes,
        recipients,
        includeDefaultRecipients: values.includeDefaultRecipients,
      };
      if (editing) {
        await updateMutation.mutateAsync({ id: editing.id, revision: editing.revision, payload });
      } else {
        await createMutation.mutateAsync(payload);
      }
      void message.success(t('notification.saveSuccess'));
      setEditing(undefined);
    } catch (error) {
      if (error && typeof error === 'object' && 'errorFields' in error) return;
      showError(error);
    }
  };

  const columns: ColumnsType<AlarmEmailSetting> = [
    { title: t('alarm.emailSettings.name'), dataIndex: 'name', key: 'name', ellipsis: true },
    {
      title: t('alarm.emailSettings.status'), dataIndex: 'enabled', key: 'enabled', width: 100,
      render: (enabled: boolean) => enabled
        ? <Tag color="green">{t('common.enabled')}</Tag>
        : <Tag>{t('common.disabled')}</Tag>,
    },
    {
      title: t('alarm.emailSettings.interval'), dataIndex: 'intervalMinutes', key: 'intervalMinutes', width: 130,
      render: (value: number) => value === 0 ? t('alarm.emailSettings.realtime') : t('alarm.emailSettings.minutes', { value }),
    },
    {
      title: t('alarm.emailSettings.tolerance'), dataIndex: 'toleranceDurationMinutes', key: 'tolerance', width: 140,
      render: (value: number) => value === 0 ? t('alarm.emailSettings.realtime') : t('alarm.emailSettings.minutes', { value }),
    },
    {
      title: t('alarm.emailSettings.recipients'), key: 'recipients', ellipsis: true,
      render: (_value, record) => [
        ...record.recipients,
        ...(record.includeDefaultRecipients ? [t('alarm.emailSettings.defaultRecipients')] : []),
      ].join('; '),
    },
    {
      title: t('alarm.emailSettings.updatedAt'), dataIndex: 'updatedAt', key: 'updatedAt', width: 180,
      render: (value: string) => formatSystemTime(value),
    },
    {
      title: t('common.operation'), key: 'actions', width: 150, fixed: 'right',
      render: (_value, record) => (
        <Space size={4}>
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => setEditing(record)}>
            {t('common.edit')}
          </Button>
          <Popconfirm
            title={t('alarm.emailSettings.deleteConfirm')}
            onConfirm={async () => {
              try {
                await archiveMutation.mutateAsync({ id: record.id, revision: record.revision });
                void message.success(t('common.deleteSuccess'));
              } catch (error) {
                showError(error);
              }
            }}
          >
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>{t('common.delete')}</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const submitting = createMutation.isPending || updateMutation.isPending;

  return (
    <ListPageLayout title={t('alarm.emailSettings.title')}>
      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
        title={t('alarm.emailSettings.fixedBodyTitle')}
        description={t('alarm.emailSettings.fixedBodyDescription')}
      />
      <Space style={{ marginBottom: 16 }}>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setEditing(null)}>
          {t('alarm.emailSettings.create')}
        </Button>
        <Button onClick={() => void settings.refetch()}>{t('common.refresh')}</Button>
        {isSuperAdmin && (
          <Button loading={defaults.isLoading} onClick={openDefaults}>
            {t('alarm.emailSettings.manageDefaultRecipients')}
          </Button>
        )}
      </Space>
      <Table
        rowKey="id"
        size="small"
        columns={columns}
        dataSource={settings.data ?? []}
        loading={settings.isLoading}
        pagination={false}
        scroll={{ x: 1100 }}
      />

      <Drawer
        title={editing ? t('alarm.emailSettings.edit') : t('alarm.emailSettings.create')}
        open={editing !== undefined}
        onClose={() => setEditing(undefined)}
        width={720}
        destroyOnHidden
        footer={(
          <div style={{ textAlign: 'right' }}>
            <Space>
              <Button onClick={() => setEditing(undefined)}>{t('common.cancel')}</Button>
              <Button type="primary" loading={submitting} onClick={() => void save()}>{t('common.save')}</Button>
            </Space>
          </div>
        )}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label={t('alarm.emailSettings.name')} rules={[{ required: true, max: 128 }]}>
            <Input maxLength={128} />
          </Form.Item>
          <Form.Item name="enabled" label={t('alarm.emailSettings.enable')} valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item name="alarmIdentifiers" label={t('alarm.emailSettings.alarmScope')}>
            <Select mode="multiple" showSearch options={alarmOptions} loading={alarmDefinitions.isLoading} optionFilterProp="label" />
          </Form.Item>
          <Form.Item name="severities" label={t('alarm.emailSettings.severity')}>
            <Checkbox.Group options={severityOptions} />
          </Form.Item>
          <Form.Item name="technologies" label={t('alarm.emailSettings.technology')}>
            <Checkbox.Group options={[
              { label: '2G', value: 'gsm' },
              { label: '4G', value: 'lte' },
              { label: '5G', value: 'nr' },
            ]} />
          </Form.Item>
          <Form.Item name="scopeMode" label={t('alarm.emailSettings.neScope')} rules={[{ required: true }]}>
            <Radio.Group options={[
              { value: 'device', label: t('alarm.emailSettings.device') },
              { value: 'group', label: t('alarm.emailSettings.deviceGroup') },
            ]} />
          </Form.Item>
          {scopeMode === 'group' ? (
            <Form.Item name="deviceGroupIds" label={t('alarm.emailSettings.deviceGroup')} rules={[{ required: true }]}>
              <Select mode="multiple" showSearch optionFilterProp="label" options={groupOptions} loading={deviceGroups.isLoading} />
            </Form.Item>
          ) : (
            <Form.Item name="deviceIds" label={t('alarm.emailSettings.device')} rules={[{ required: true }]}>
              <Select
                mode="multiple"
                showSearch
                filterOption={false}
                onSearch={setDeviceSearch}
                options={deviceOptions}
                loading={devices.isFetching}
              />
            </Form.Item>
          )}
          <Space size="large" align="start">
            <Form.Item name="intervalMinutes" label={t('alarm.emailSettings.interval')} rules={[{ required: true }]}>
              <Select style={{ width: 180 }} options={intervalOptions} />
            </Form.Item>
            <Form.Item name="toleranceDurationMinutes" label={t('alarm.emailSettings.tolerance')} rules={[{ required: true }]}>
              <Select style={{ width: 180 }} options={intervalOptions} />
            </Form.Item>
          </Space>
          <Form.Item name="recipients" label={t('alarm.emailSettings.recipients')}>
            <Input.TextArea rows={4} placeholder={t('alarm.emailSettings.recipientsHint')} />
          </Form.Item>
          <Form.Item name="includeDefaultRecipients" valuePropName="checked">
            <Checkbox>{t('alarm.emailSettings.includeDefaultRecipients')}</Checkbox>
          </Form.Item>
          <Typography.Text type="secondary">{t('alarm.emailSettings.bodyFields')}</Typography.Text>
        </Form>
      </Drawer>
      <Modal
        title={t('alarm.emailSettings.manageDefaultRecipients')}
        open={defaultsOpen}
        onCancel={() => setDefaultsOpen(false)}
        onOk={() => void saveDefaults()}
        confirmLoading={updateDefaultsMutation.isPending}
        okButtonProps={{ disabled: !defaults.data }}
        destroyOnHidden
      >
        <Alert
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
          title={t('alarm.emailSettings.defaultRecipientsDescription')}
        />
        <Input.TextArea
          rows={5}
          value={defaultRecipients}
          onChange={(event) => setDefaultRecipients(event.target.value)}
          placeholder={t('alarm.emailSettings.recipientsHint')}
        />
      </Modal>
    </ListPageLayout>
  );
}
