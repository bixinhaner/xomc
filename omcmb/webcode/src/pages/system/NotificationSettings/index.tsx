import { useState } from 'react';
import {
  Button,
  Card,
  Form,
  Input,
  Select,
  Switch,
  Tabs,
  Table,
  Tag,
  Space,
  Modal,
  message,
  Checkbox,
} from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, SaveOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useT } from '@/hooks/useT';

interface NotificationRule {
  id: string;
  name: string;
  triggerEvent: string;
  channels: string[];
  recipients: string;
  enabled: boolean;
}

interface RecipientGroup {
  id: string;
  name: string;
  emails: string;
  phones: string;
  memberCount: number;
}

const mockRules: NotificationRule[] = [
  { id: 'nr-001', name: '紧急告警通知', triggerEvent: 'alarm_critical', channels: ['email', 'sms'], recipients: '运维一组', enabled: true },
  { id: 'nr-002', name: '重要告警通知', triggerEvent: 'alarm_major', channels: ['email'], recipients: '运维一组,运维二组', enabled: true },
  { id: 'nr-003', name: '设备离线通知', triggerEvent: 'device_offline', channels: ['email'], recipients: '运维一组', enabled: true },
  { id: 'nr-004', name: '升级任务完成通知', triggerEvent: 'task_complete', channels: ['email'], recipients: '管理员', enabled: false },
  { id: 'nr-005', name: '系统存储告警', triggerEvent: 'storage_warning', channels: ['email', 'sms'], recipients: '管理员', enabled: true },
];

const mockGroups: RecipientGroup[] = [
  { id: 'rg-001', name: '管理员', emails: 'admin@example.com', phones: '13800000001', memberCount: 2 },
  { id: 'rg-002', name: '运维一组', emails: 'ops1@example.com,ops2@example.com', phones: '13800000002,13800000003', memberCount: 5 },
  { id: 'rg-003', name: '运维二组', emails: 'ops3@example.com', phones: '13800000004', memberCount: 3 },
];

const triggerEventOptions = [
  { label: '紧急告警产生', value: 'alarm_critical' },
  { label: '重要告警产生', value: 'alarm_major' },
  { label: '设备离线', value: 'device_offline' },
  { label: '设备恢复在线', value: 'device_online' },
  { label: '升级任务完成', value: 'task_complete' },
  { label: '升级任务失败', value: 'task_failed' },
  { label: '存储空间告警', value: 'storage_warning' },
  { label: '系统服务异常', value: 'service_error' },
  { label: '用户登录失败超限', value: 'login_failed' },
];

const triggerEventLabelMap: Record<string, string> = Object.fromEntries(triggerEventOptions.map((o) => [o.value, o.label]));

export default function NotificationSettings() {
  const t = useT();
  const [rules, setRules] = useState<NotificationRule[]>(mockRules);
  const [groups, setGroups] = useState<RecipientGroup[]>(mockGroups);
  const [ruleModalVisible, setRuleModalVisible] = useState(false);
  const [groupModalVisible, setGroupModalVisible] = useState(false);
  const [editRule, setEditRule] = useState<NotificationRule | null>(null);
  const [editGroup, setEditGroup] = useState<RecipientGroup | null>(null);
  const [ruleForm] = Form.useForm();
  const [groupForm] = Form.useForm();
  const [emailTemplateForm] = Form.useForm();

  const handleSaveRule = () => {
    ruleForm.validateFields().then((vals) => {
      if (editRule) {
        setRules((prev) => prev.map((r) => r.id === editRule.id ? { ...r, ...vals } : r));
        void message.success(t('common.save'));
      } else {
        setRules((prev) => [...prev, { id: `nr-${Date.now()}`, ...vals, enabled: true }]);
        void message.success(t('common.save'));
      }
      setRuleModalVisible(false);
      ruleForm.resetFields();
      setEditRule(null);
    });
  };

  const handleSaveGroup = () => {
    groupForm.validateFields().then((vals) => {
      if (editGroup) {
        const emailList = String(vals.emails).split(/[,\n]/).filter(Boolean);
        setGroups((prev) => prev.map((g) => g.id === editGroup.id ? { ...g, ...vals, memberCount: emailList.length } : g));
        void message.success(t('common.save'));
      } else {
        const emailList = String(vals.emails).split(/[,\n]/).filter(Boolean);
        setGroups((prev) => [...prev, { id: `rg-${Date.now()}`, ...vals, memberCount: emailList.length }]);
        void message.success(t('common.save'));
      }
      setGroupModalVisible(false);
      groupForm.resetFields();
      setEditGroup(null);
    });
  };

  const ruleColumns = [
    { title: t('table.name'), dataIndex: 'name', key: 'name', ellipsis: true },
    { title: t('table.type'), dataIndex: 'triggerEvent', key: 'triggerEvent', width: 160, render: (val: string) => triggerEventLabelMap[val] ?? val },
    {
      title: t('table.type'), dataIndex: 'channels', key: 'channels', width: 140,
      render: (val: string[]) => val.map((c) => <Tag key={c} color={c === 'email' ? 'blue' : 'green'}>{c === 'email' ? 'Email' : 'SMS'}</Tag>),
    },
    { title: t('table.name'), dataIndex: 'recipients', key: 'recipients', width: 160 },
    {
      title: t('table.status'), dataIndex: 'enabled', key: 'enabled', width: 90,
      render: (val: boolean, record: NotificationRule) => (
        <Switch checked={val} size="small"
          onChange={(checked) => setRules((prev) => prev.map((r) => r.id === record.id ? { ...r, enabled: checked } : r))} />
      ),
    },
    {
      title: t('table.operation'), key: 'actions', width: 120, fixed: 'right',
      render: (_: unknown, record: NotificationRule) => (
        <Space size={4}>
          <Button type="link" size="small" icon={<EditOutlined />}
            onClick={() => { setEditRule(record); ruleForm.setFieldsValue(record); setRuleModalVisible(true); }}>
            {t('common.edit')}
          </Button>
          <Button type="link" size="small" danger icon={<DeleteOutlined />}
            onClick={() => { setRules((prev) => prev.filter((r) => r.id !== record.id)); void message.success(t('common.deleteSuccess')); }}>
            {t('common.delete')}
          </Button>
        </Space>
      ),
    },
  ];

  const groupColumns = [
    { title: t('table.name'), dataIndex: 'name', key: 'name', width: 130 },
    { title: t('user.email'), dataIndex: 'emails', key: 'emails', ellipsis: true },
    { title: t('user.phone'), dataIndex: 'phones', key: 'phones', ellipsis: true },
    { title: t('table.total'), dataIndex: 'memberCount', key: 'memberCount', width: 80, render: (val: number) => `${val}` },
    {
      title: t('table.operation'), key: 'actions', width: 120, fixed: 'right',
      render: (_: unknown, record: RecipientGroup) => (
        <Space size={4}>
          <Button type="link" size="small" icon={<EditOutlined />}
            onClick={() => { setEditGroup(record); groupForm.setFieldsValue(record); setGroupModalVisible(true); }}>
            {t('common.edit')}
          </Button>
          <Button type="link" size="small" danger icon={<DeleteOutlined />}
            onClick={() => { setGroups((prev) => prev.filter((g) => g.id !== record.id)); void message.success(t('common.deleteSuccess')); }}>
            {t('common.delete')}
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <ListPageLayout title={t('nav.system.notifications')}>
      <Tabs
        items={[
          {
            key: 'rules',
            label: t('nav.system.notifications'),
            children: (
              <Card
                extra={
                  <Button type="primary" icon={<PlusOutlined />}
                    onClick={() => { setEditRule(null); ruleForm.resetFields(); setRuleModalVisible(true); }}>
                    {t('common.add')}
                  </Button>
                }
              >
                <Table dataSource={rules} columns={ruleColumns} rowKey="id" size="small" pagination={{ pageSize: 10 }} />
              </Card>
            ),
          },
          {
            key: 'groups',
            label: t('table.name'),
            children: (
              <Card
                extra={
                  <Button type="primary" icon={<PlusOutlined />}
                    onClick={() => { setEditGroup(null); groupForm.resetFields(); setGroupModalVisible(true); }}>
                    {t('common.add')}
                  </Button>
                }
              >
                <Table dataSource={groups} columns={groupColumns} rowKey="id" size="small" pagination={{ pageSize: 10 }} />
              </Card>
            ),
          },
          {
            key: 'template',
            label: t('nav.ops.templates'),
            children: (
              <Card extra={<Button type="primary" icon={<SaveOutlined />} onClick={() => void message.success(t('common.save'))}>{t('common.save')}</Button>}>
                <Form form={emailTemplateForm} layout="vertical" style={{ maxWidth: 600 }}
                  initialValues={{ subject: '【OMC告警】{severity} - {alarmName} - {deviceSn}', body: '尊敬的管理员：\n\n系统检测到以下事件，请及时处理：\n\n事件类型：{triggerEvent}\n设备SN：{deviceSn}\n告警时间：{timestamp}\n告警级别：{severity}\n告警内容：{content}\n\n请登录OMC平台处理：http://omc.example.com\n\n此邮件由系统自动发送，请勿回复。' }}>
                  <Form.Item name="subject" label={t('table.name')}>
                    <Input placeholder="{severity}, {alarmName}, {deviceSn}, {timestamp}" />
                  </Form.Item>
                  <Form.Item name="body" label={t('table.description')}>
                    <Input.TextArea rows={10} placeholder="{triggerEvent}, {deviceSn}, {timestamp}, {severity}, {content}" />
                  </Form.Item>
                </Form>
              </Card>
            ),
          },
        ]}
      />

      <Modal
        title={editRule ? t('common.edit') : t('common.add')}
        open={ruleModalVisible}
        onOk={handleSaveRule}
        onCancel={() => { setRuleModalVisible(false); ruleForm.resetFields(); setEditRule(null); }}
        width={520}
      >
        <Form form={ruleForm} layout="vertical">
          <Form.Item name="name" label={t('table.name')} rules={[{ required: true }]}>
            <Input placeholder={t('table.name')} />
          </Form.Item>
          <Form.Item name="triggerEvent" label={t('table.type')} rules={[{ required: true }]}>
            <Select options={triggerEventOptions} placeholder={t('common.pleaseSelect')} />
          </Form.Item>
          <Form.Item name="channels" label={t('table.type')} rules={[{ required: true }]}>
            <Checkbox.Group options={[{ label: 'Email', value: 'email' }, { label: 'SMS', value: 'sms' }]} />
          </Form.Item>
          <Form.Item name="recipients" label={t('table.name')} rules={[{ required: true }]}>
            <Select
              mode="multiple"
              placeholder={t('common.pleaseSelect')}
              options={groups.map((g) => ({ label: `${g.name} (${g.memberCount})`, value: g.name }))}
            />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={editGroup ? t('common.edit') : t('common.add')}
        open={groupModalVisible}
        onOk={handleSaveGroup}
        onCancel={() => { setGroupModalVisible(false); groupForm.resetFields(); setEditGroup(null); }}
        width={520}
      >
        <Form form={groupForm} layout="vertical">
          <Form.Item name="name" label={t('table.name')} rules={[{ required: true }]}>
            <Input placeholder={t('table.name')} />
          </Form.Item>
          <Form.Item name="emails" label={t('user.email')} rules={[{ required: true }]}>
            <Input.TextArea rows={3} placeholder={t('user.email')} />
          </Form.Item>
          <Form.Item name="phones" label={t('user.phone')}>
            <Input.TextArea rows={2} placeholder={t('user.phone')} />
          </Form.Item>
        </Form>
      </Modal>
    </ListPageLayout>
  );
}
