import { useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Button,
  Card,
  Col,
  Form,
  Input,
  Modal,
  Row,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  Typography,
  message,
} from 'antd';
import { DeleteOutlined, EditOutlined, PlusOutlined, SaveOutlined } from '@ant-design/icons';
import {
  useAlarmEmailSetting,
  useAlarmEmailSubscriptions,
  useCreateAlarmEmailSubscription,
  useDeleteAlarmEmailSubscription,
  useUpdateAlarmEmailSetting,
  useUpdateAlarmEmailSubscription,
} from '@core/hooks/api/useAlarms';
import { useAllAlarmDefinitions } from '@core/hooks/api/useAlarmDefinitions';
import { useDeviceGroups, useDeviceList, useProductClasses } from '@core/hooks/api/useDevices';
import type {
  AlarmEmailSubscription,
  AlarmEmailSubscriptionInput,
} from '@core/services/api/alarmApi';
import { useT } from '@/hooks/useT';
import { useAppStore } from '@core/store/appStore';
import { useUserStore } from '@core/store/userStore';
import { buildDeviceGroupPathName } from '@core/utils/deviceGroupDisplay';
import { ALARM_EMAIL_EVENT_TYPES, buildAlarmSourceOptions } from './options';

const { Text } = Typography;
const minuteOptions = [0, 10, 30, 60] as const;

interface SubscriptionFormValues extends Omit<AlarmEmailSubscriptionInput, 'recipients'> {
  recipients_text: string;
}

function parseRecipients(value: string): string[] {
  return Array.from(new Set(value.split(/[;,\n]/).map((item) => item.trim()).filter(Boolean)));
}

function isEmailAddress(value: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value);
}

function toFormValues(item?: AlarmEmailSubscription): SubscriptionFormValues {
  return {
    name: item?.name ?? '',
    description: item?.description ?? '',
    enabled: item?.enabled ?? false,
    interval_minutes: item?.interval_minutes ?? 10,
    tolerance_minutes: item?.tolerance_minutes ?? 0,
    recipients_text: item?.recipients.join('; ') ?? '',
    include_default_recipients: item?.include_default_recipients ?? true,
    alarm_identifiers: item?.alarm_identifiers ?? [],
    severities: item?.severities ?? [],
    alarm_sources: item?.alarm_sources ?? [],
    event_types: item?.event_types ?? [],
    device_ids: item?.device_ids ?? [],
    device_group_ids: item?.device_group_ids ?? [],
  };
}

export default function AlarmEmailSubscriptions() {
  const t = useT();
  const locale = useAppStore((state) => state.locale);
  const isSuperAdmin = useUserStore((state) => state.currentUser?.isSuperAdmin === true);
  const [settingForm] = Form.useForm<{ enabled: boolean; default_recipients_text: string }>();
  const [subscriptionForm] = Form.useForm<SubscriptionFormValues>();
  const [editing, setEditing] = useState<AlarmEmailSubscription>();
  const [modalOpen, setModalOpen] = useState(false);
  const [deviceSearch, setDeviceSearch] = useState('');

  const settingQuery = useAlarmEmailSetting();
  const subscriptionsQuery = useAlarmEmailSubscriptions();
  const definitionsQuery = useAllAlarmDefinitions({ pageSize: 500 });
  const devicesQuery = useDeviceList({
    page: 1,
    pageSize: 100,
    searchText: deviceSearch || undefined,
  });
  const groupsQuery = useDeviceGroups();
  const productClassesQuery = useProductClasses();
  const updateSetting = useUpdateAlarmEmailSetting();
  const createSubscription = useCreateAlarmEmailSubscription();
  const updateSubscription = useUpdateAlarmEmailSubscription();
  const deleteSubscription = useDeleteAlarmEmailSubscription();
  const settingReady = settingQuery.isSuccess && !settingQuery.isFetching;

  useEffect(() => {
    if (!settingQuery.data) return;
    settingForm.setFieldsValue({
      enabled: settingQuery.data.enabled,
      default_recipients_text: settingQuery.data.default_recipients.join('; '),
    });
  }, [settingForm, settingQuery.data]);

  const intervalOptions = useMemo(
    () => minuteOptions.map((value) => ({
      value,
      label: value === 0 ? t('alarm.email.realTime') : t('alarm.email.minutes', { count: value }),
    })),
    [t],
  );
  const alarmOptions = useMemo(
    () => (definitionsQuery.data?.items ?? []).map((item) => ({
      value: item.identifier,
      label: `${item.identifier} - ${item.cnName || item.enName || item.identifier}`,
    })),
    [definitionsQuery.data?.items],
  );
  const eventTypeOptions = useMemo(
    () => ALARM_EMAIL_EVENT_TYPES.map((item) => ({
      value: item.value,
      label: t(item.labelKey),
    })),
    [t],
  );
  const alarmSourceOptions = useMemo(
    () => buildAlarmSourceOptions(productClassesQuery.data, subscriptionsQuery.data),
    [productClassesQuery.data, subscriptionsQuery.data],
  );
  const deviceOptions = useMemo(
    () => (devicesQuery.data?.items ?? []).map((item) => ({
      value: item.id,
      label: `${item.sn}${item.name ? ` - ${item.name}` : ''}`,
    })),
    [devicesQuery.data?.items],
  );
  const groupOptions = useMemo(
    () => {
      const groups = groupsQuery.data?.groups ?? [];
      return groups.map((item) => ({
        value: item.id,
        label: buildDeviceGroupPathName(item, groups, locale),
      }));
    },
    [groupsQuery.data, locale],
  );

  const openCreate = () => {
    setDeviceSearch('');
    setEditing(undefined);
    setModalOpen(true);
  };

  const openEdit = (item: AlarmEmailSubscription) => {
    setDeviceSearch('');
    setEditing(item);
    setModalOpen(true);
  };

  const saveSetting = async () => {
    if (!settingReady) return;
    const values = await settingForm.validateFields();
    const recipients = parseRecipients(values.default_recipients_text);
    if (recipients.some((recipient) => !isEmailAddress(recipient))) {
      void message.error(t('alarm.email.invalidRecipient'));
      return;
    }
    await updateSetting.mutateAsync({
      enabled: values.enabled,
      default_recipients: recipients,
    });
    void message.success(t('common.saveSuccess'));
  };

  const saveSubscription = async () => {
    const values = await subscriptionForm.validateFields();
    const { recipients_text, ...rest } = values;
    const recipients = parseRecipients(recipients_text);
    if (recipients.some((recipient) => !isEmailAddress(recipient))) {
      void message.error(t('alarm.email.invalidRecipient'));
      return;
    }
    const defaultRecipients = parseRecipients(settingForm.getFieldValue('default_recipients_text') ?? '');
    if (values.enabled && recipients.length === 0 && (!values.include_default_recipients || defaultRecipients.length === 0)) {
      void message.error(t('alarm.email.noRecipients'));
      return;
    }
    const input: AlarmEmailSubscriptionInput = {
      ...rest,
      recipients,
    };
    if (editing) {
      await updateSubscription.mutateAsync({ id: editing.id, input });
    } else {
      await createSubscription.mutateAsync(input);
    }
    setModalOpen(false);
    void message.success(t('common.saveSuccess'));
  };

  const confirmDelete = (item: AlarmEmailSubscription) => {
    Modal.confirm({
      title: t('alarm.email.deleteTitle'),
      content: t('alarm.email.deleteConfirm', { name: item.name }),
      okButtonProps: { danger: true },
      onOk: async () => {
        await deleteSubscription.mutateAsync(item.id);
        void message.success(t('common.deleteSuccess'));
      },
    });
  };

  return (
    <Space orientation="vertical" size={16} style={{ width: '100%' }}>
      <Card title={t('alarm.email.pageTitle')}>
        <Alert
          showIcon
          type="info"
          title={t('alarm.email.businessHint')}
          style={{ marginBottom: 16 }}
        />
        {settingQuery.isError && (
          <Alert
            showIcon
            type="error"
            title={t('empty.loadFailed')}
            description={t('empty.loadFailedDesc')}
            action={<Button onClick={() => void settingQuery.refetch()}>{t('common.retry')}</Button>}
            style={{ marginBottom: 16 }}
          />
        )}
        <Form form={settingForm} layout="vertical" initialValues={{ enabled: false, default_recipients_text: '' }}>
          <Form.Item name="enabled" label={t('alarm.email.globalEnabled')} valuePropName="checked">
            <Switch disabled={!isSuperAdmin || !settingReady} />
          </Form.Item>
          <Form.Item
            name="default_recipients_text"
            label={t('alarm.email.defaultRecipients')}
            extra={t('alarm.email.recipientHint')}
          >
            <Input.TextArea rows={2} disabled={!isSuperAdmin || !settingReady} />
          </Form.Item>
          <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
            <Button
              type="primary"
              icon={<SaveOutlined />}
              loading={updateSetting.isPending}
              disabled={!isSuperAdmin || !settingReady}
              onClick={() => void saveSetting()}
            >
              {t('common.save')}
            </Button>
          </div>
        </Form>
      </Card>

      <Card
        title={t('alarm.email.subscriptionTitle')}
        extra={<Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>{t('common.add')}</Button>}
      >
        {subscriptionsQuery.isError && (
          <Alert
            showIcon
            type="error"
            title={t('empty.loadFailed')}
            description={t('empty.loadFailedDesc')}
            action={<Button onClick={() => void subscriptionsQuery.refetch()}>{t('common.retry')}</Button>}
            style={{ marginBottom: 16 }}
          />
        )}
        <Table<AlarmEmailSubscription>
          rowKey="id"
          loading={subscriptionsQuery.isLoading}
          dataSource={subscriptionsQuery.data ?? []}
          pagination={false}
          scroll={{ x: 900 }}
          columns={[
            { title: t('common.name'), dataIndex: 'name', width: 200 },
            {
              title: t('common.status'),
              dataIndex: 'enabled',
              width: 100,
              render: (enabled: boolean) => enabled
                ? <Tag color="success">{t('common.enabled')}</Tag>
                : <Tag>{t('common.disabled')}</Tag>,
            },
            {
              title: t('alarm.email.interval'),
              width: 120,
              render: (_, item) => item.interval_minutes === 0
                ? t('alarm.email.realTime')
                : t('alarm.email.minutes', { count: item.interval_minutes }),
            },
            {
              title: t('alarm.email.tolerance'),
              width: 120,
              render: (_, item) => item.tolerance_minutes === 0
                ? t('alarm.email.realTime')
                : t('alarm.email.minutes', { count: item.tolerance_minutes }),
            },
            {
              title: t('alarm.email.recipients'),
              width: 240,
              render: (_, item) => (
                <Text ellipsis={{ tooltip: item.recipients.join('; ') }} style={{ maxWidth: 260 }}>
                  {item.recipients.join('; ') || (item.include_default_recipients ? t('alarm.email.useDefaults') : '—')}
                </Text>
              ),
            },
            {
              title: t('common.actions'),
              width: 120,
              render: (_, item) => (
                <Space>
                  <Button type="text" icon={<EditOutlined />} onClick={() => openEdit(item)} />
                  <Button danger type="text" icon={<DeleteOutlined />} onClick={() => confirmDelete(item)} />
                </Space>
              ),
            },
          ]}
        />
      </Card>

      <Modal
        open={modalOpen}
        title={editing ? t('alarm.email.editSubscription') : t('alarm.email.addSubscription')}
        width={880}
        confirmLoading={createSubscription.isPending || updateSubscription.isPending}
        onCancel={() => setModalOpen(false)}
        onOk={() => void saveSubscription()}
        afterOpenChange={(open) => {
          if (open) subscriptionForm.setFieldsValue(toFormValues(editing));
        }}
        destroyOnHidden
      >
        <Form form={subscriptionForm} layout="vertical" initialValues={toFormValues()}>
          <Row gutter={[24, 0]}>
            <Col xs={24} md={18}>
              <Form.Item name="name" label={t('common.name')} rules={[{ required: true }]}>
                <Input />
              </Form.Item>
            </Col>
            <Col xs={24} md={6}>
              <Form.Item name="enabled" label={t('common.enabled')} valuePropName="checked">
                <Switch />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name="description" label={t('common.description')}>
            <Input.TextArea rows={2} />
          </Form.Item>
          <Row gutter={[24, 0]}>
            <Col xs={24} sm={12} md={8}>
              <Form.Item name="interval_minutes" label={t('alarm.email.interval')} rules={[{ required: true }]}>
                <Select options={intervalOptions} />
              </Form.Item>
            </Col>
            <Col xs={24} sm={12} md={8}>
              <Form.Item name="tolerance_minutes" label={t('alarm.email.tolerance')} rules={[{ required: true }]}>
                <Select options={intervalOptions} />
              </Form.Item>
            </Col>
            <Col xs={24} sm={12} md={8}>
              <Form.Item name="include_default_recipients" label={t('alarm.email.includeDefaults')} valuePropName="checked">
                <Switch />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name="recipients_text" label={t('alarm.email.recipients')} extra={t('alarm.email.recipientHint')}>
            <Input.TextArea rows={2} />
          </Form.Item>
          <Form.Item name="alarm_identifiers" label={t('alarm.email.alarmIdentifiers')}>
            <Select mode="multiple" showSearch optionFilterProp="label" options={alarmOptions} />
          </Form.Item>
          <Row gutter={[24, 0]}>
            <Col xs={24} md={8}>
              <Form.Item name="severities" label={t('alarm.email.severities')}>
                <Select mode="multiple" options={[
                  { value: 1, label: t('alarm.severity.critical') },
                  { value: 2, label: t('alarm.severity.major') },
                  { value: 3, label: t('alarm.severity.minor') },
                  { value: 4, label: t('alarm.severity.warning') },
                ]} />
              </Form.Item>
            </Col>
            <Col xs={24} md={8}>
              <Form.Item
                name="event_types"
                label={t('alarm.email.eventTypes')}
                extra={t('alarm.email.eventTypeHint')}
              >
                <Select mode="tags" options={eventTypeOptions} />
              </Form.Item>
            </Col>
            <Col xs={24} md={8}>
              <Form.Item
                name="alarm_sources"
                label={t('alarm.email.alarmSources')}
                extra={t('alarm.email.alarmSourceHint')}
              >
                <Select
                  mode="tags"
                  loading={productClassesQuery.isLoading}
                  options={alarmSourceOptions}
                />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item
            name="device_group_ids"
            label={t('alarm.email.deviceGroups')}
            extra={t('alarm.email.deviceGroupHint')}
          >
            <Select mode="multiple" showSearch optionFilterProp="label" options={groupOptions} />
          </Form.Item>
          <Form.Item name="device_ids" label={t('alarm.email.devices')} extra={t('alarm.email.deviceLimitHint')}>
            <Select
              mode="multiple"
              showSearch
              filterOption={false}
              onSearch={(value) => setDeviceSearch(value.trim())}
              loading={devicesQuery.isFetching}
              options={deviceOptions}
            />
          </Form.Item>
        </Form>
      </Modal>
    </Space>
  );
}
