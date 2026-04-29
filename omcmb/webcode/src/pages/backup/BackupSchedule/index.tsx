// T-0070 / R-102 followup: BackupSchedule UI 重设计
//
// 历史: 此页面（路由 /backup/schedule，菜单"备份调度"）原本是"配置文件 import/export"
// demo placeholder（460 行 mock + Blob 客户端拼接 XML），与同名的 BackupSchedule 后端
// 完全错位。T-0016 audit 后判定为未上线生产的 dev artifact，按 PRD §1 wholesale replace。
//
// 现在: 消费 frontend-core 4 schedule hooks，做真实的 cron 调度任务管理。
//
// 非目标（PRD §5）：cron 解析依赖 / 下次执行时间预测 / 调度执行历史 / clone / template

import { useCallback, useMemo, useState } from 'react';
import {
  Button,
  Form,
  Input,
  Modal,
  Popconfirm,
  Radio,
  Select,
  Space,
  Switch,
  Tag,
  message,
} from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import {
  useBackupSchedules,
  useCreateBackupSchedule,
  useUpdateBackupSchedule,
  useDeleteBackupSchedules,
} from '@core/hooks/api/useBackup';
import { useAllDeviceGroups } from '@core/hooks/api/useSystem';
import type { BackupSchedule } from '@core/mock/data/backup';
import type { DeviceGroup } from '@core/types/device';
import { useT } from '@/hooks/useT';

// ---------------------------------------------------------------------------
// Cron presets (in-house, no third-party dependency per PRD §5)
// ---------------------------------------------------------------------------

interface CronPreset {
  key: string;
  cron: string;
  labelKey: string;
}

const CRON_PRESETS: readonly CronPreset[] = [
  { key: 'daily-midnight', cron: '0 0 * * *', labelKey: 'backup.cronDaily00' },
  { key: 'weekly-mon', cron: '0 0 * * 1', labelKey: 'backup.cronWeeklyMon' },
  { key: 'monthly-1st', cron: '0 0 1 * *', labelKey: 'backup.cronMonthly1st' },
  { key: 'every-6h', cron: '0 */6 * * *', labelKey: 'backup.cronEvery6h' },
  { key: 'custom', cron: '', labelKey: 'backup.cronCustom' },
] as const;

function isValidCron(s: string): boolean {
  // Minimal 5-field shape check: server is the source of truth for semantic validation.
  const parts = s.trim().split(/\s+/);
  return parts.length === 5 && parts.every((p) => p.length > 0);
}

function presetForCron(cron: string): string {
  const hit = CRON_PRESETS.find((p) => p.cron === cron);
  return hit ? hit.key : 'custom';
}

// ---------------------------------------------------------------------------
// UI types
// ---------------------------------------------------------------------------

interface ScheduleRow extends Record<string, unknown> {
  id: string;
  scheduleName: string;
  cronExpression: string;
  backupType: BackupSchedule['backupType'];
  deviceGroups: string[]; // group IDs
  enabled: boolean;
  createTime: string;
}

function toRow(s: BackupSchedule): ScheduleRow {
  return {
    id: s.id,
    scheduleName: s.scheduleName,
    cronExpression: s.cronExpression,
    backupType: s.backupType,
    deviceGroups: s.deviceGroups ?? [],
    enabled: s.enabled,
    createTime: s.createTime,
  };
}

interface ScheduleFormValues {
  scheduleName: string;
  cronPreset: string;
  cronExpression: string;
  backupType: BackupSchedule['backupType'];
  deviceGroups: string[];
  enabled: boolean;
}

function getErrMsg(e: unknown): string {
  if (e instanceof Error) return e.message;
  if (typeof e === 'object' && e && 'message' in e) {
    return String((e as { message: unknown }).message);
  }
  return '';
}

const BACKUP_TYPE_TAG_COLOR: Record<BackupSchedule['backupType'], string> = {
  full: 'blue',
  incremental: 'cyan',
  'config-only': 'purple',
};

const BACKUP_TYPE_LABEL_KEY: Record<BackupSchedule['backupType'], string> = {
  full: 'backup.fullBackup',
  incremental: 'backup.incrementalBackup',
  'config-only': 'backup.configBackup',
};

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function BackupSchedulePage(): JSX.Element {
  const t = useT();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingRow, setEditingRow] = useState<ScheduleRow | null>(null);
  const [form] = Form.useForm<ScheduleFormValues>();
  const cronPreset = Form.useWatch('cronPreset', form);

  const { data, isLoading, refetch } = useBackupSchedules({ page, pageSize });
  const { data: groupsData } = useAllDeviceGroups();
  const createSchedule = useCreateBackupSchedule();
  const updateSchedule = useUpdateBackupSchedule();
  const deleteSchedules = useDeleteBackupSchedules();

  const tableSource: ScheduleRow[] = useMemo(
    () => (data?.items ?? []).map(toRow),
    [data?.items],
  );

  const groupOptions = useMemo(
    () =>
      (groupsData ?? []).map((g: DeviceGroup) => ({
        label: g.name,
        value: g.id,
      })),
    [groupsData],
  );

  const groupNameById = useMemo(() => {
    const m = new Map<string, string>();
    for (const g of groupsData ?? []) m.set(g.id, g.name);
    return m;
  }, [groupsData]);

  const openCreate = (): void => {
    setEditingRow(null);
    form.resetFields();
    form.setFieldsValue({
      scheduleName: '',
      cronPreset: 'daily-midnight',
      cronExpression: '0 0 * * *',
      backupType: 'full',
      deviceGroups: [],
      enabled: true,
    });
    setModalVisible(true);
  };

  const openEdit = useCallback(
    (row: ScheduleRow): void => {
      setEditingRow(row);
      form.resetFields();
      form.setFieldsValue({
        scheduleName: row.scheduleName,
        cronPreset: presetForCron(row.cronExpression),
        cronExpression: row.cronExpression,
        backupType: row.backupType,
        deviceGroups: row.deviceGroups,
        enabled: row.enabled,
      });
      setModalVisible(true);
    },
    [form],
  );

  const closeModal = (): void => {
    setModalVisible(false);
    setEditingRow(null);
    form.resetFields();
  };

  const handlePresetChange = (key: string): void => {
    const preset = CRON_PRESETS.find((p) => p.key === key);
    if (preset && preset.cron) {
      form.setFieldValue('cronExpression', preset.cron);
    }
  };

  const handleSave = async (): Promise<void> => {
    let vals: ScheduleFormValues;
    try {
      vals = await form.validateFields();
    } catch {
      return; // antd surfaces field errors
    }

    const payload: Omit<BackupSchedule, 'id' | 'createTime'> = {
      scheduleName: vals.scheduleName,
      cronExpression: vals.cronExpression,
      cronDescription: '', // backend does not store; reserved for UI-derived text
      enabled: vals.enabled,
      backupType: vals.backupType,
      deviceGroups: vals.deviceGroups,
      retentionDays: 0, // not yet wired
      storageLocation: '',
      nextRunTime: '',
      creator: '',
    };

    if (editingRow) {
      updateSchedule.mutate(
        { id: editingRow.id, data: payload },
        {
          onSuccess: () => {
            void message.success(t('backup.scheduleUpdateSuccess'));
            closeModal();
          },
          onError: (e: unknown) => {
            void message.error(t('backup.scheduleUpdateFailed', { error: getErrMsg(e) }));
          },
        },
      );
    } else {
      createSchedule.mutate(payload, {
        onSuccess: () => {
          void message.success(t('backup.scheduleCreateSuccess'));
          closeModal();
        },
        onError: (e: unknown) => {
          void message.error(t('backup.scheduleCreateFailed', { error: getErrMsg(e) }));
        },
      });
    }
  };

  const handleToggleEnabled = useCallback(
    (row: ScheduleRow, enabled: boolean): void => {
      updateSchedule.mutate(
        { id: row.id, data: { enabled } },
        {
          onError: (e: unknown) => {
            void message.error(t('backup.scheduleUpdateFailed', { error: getErrMsg(e) }));
          },
        },
      );
    },
    [updateSchedule, t],
  );

  const handleDelete = useCallback(
    (row: ScheduleRow): void => {
      deleteSchedules.mutate([row.id], {
        onSuccess: () => {
          void message.success(t('backup.scheduleDeleteSuccess'));
        },
        onError: (e: unknown) => {
          void message.error(t('backup.scheduleDeleteFailed', { error: getErrMsg(e) }));
        },
      });
    },
    [deleteSchedules, t],
  );

  const columns: DataTableColumn<ScheduleRow>[] = useMemo(
    () => [
      { key: 'scheduleName', title: t('table.name'), dataIndex: 'scheduleName', width: 200 },
      {
        key: 'cronExpression',
        title: t('backup.cronExpression'),
        dataIndex: 'cronExpression',
        width: 180,
        mono: true,
      },
      {
        key: 'backupType',
        title: t('backup.backupType'),
        dataIndex: 'backupType',
        width: 130,
        render: (val) => {
          const v = val as BackupSchedule['backupType'];
          return <Tag color={BACKUP_TYPE_TAG_COLOR[v]}>{t(BACKUP_TYPE_LABEL_KEY[v])}</Tag>;
        },
      },
      {
        key: 'deviceGroups',
        title: t('backup.deviceGroups'),
        dataIndex: 'deviceGroups',
        width: 220,
        render: (val) => {
          const ids = val as string[];
          if (!ids.length) return '-';
          const names = ids.map((id) => groupNameById.get(id) ?? id).join(', ');
          return (
            <span title={names}>
              {t('backup.deviceGroupsCount', { count: ids.length })}
            </span>
          );
        },
      },
      {
        key: 'enabled',
        title: t('table.status'),
        dataIndex: 'enabled',
        width: 100,
        render: (val, record) => (
          <Switch
            size="small"
            checked={val as boolean}
            checkedChildren={t('common.enable')}
            unCheckedChildren={t('common.disable')}
            onChange={(checked) => handleToggleEnabled(record, checked)}
          />
        ),
      },
      { key: 'createTime', title: t('table.createTime'), dataIndex: 'createTime', width: 170 },
      {
        key: 'operation',
        title: t('table.operation'),
        width: 140,
        fixed: 'right',
        render: (_val, record) => (
          <Space size="small">
            <Button type="link" size="small" onClick={() => openEdit(record)}>
              {t('common.edit')}
            </Button>
            <Popconfirm
              title={t('backup.confirmDeleteSchedule', { name: record.scheduleName })}
              onConfirm={() => handleDelete(record)}
              okText={t('common.confirm')}
              cancelText={t('common.cancel')}
              okButtonProps={{ danger: true }}
            >
              <Button type="link" size="small" danger>
                {t('common.delete')}
              </Button>
            </Popconfirm>
          </Space>
        ),
      },
    ],
    [t, groupNameById, openEdit, handleToggleEnabled, handleDelete],
  );

  return (
    <ListPageLayout
      title={t('backup.scheduleListTitle')}
      extra={
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
          {t('backup.newSchedule')}
        </Button>
      }
    >
      <DataTable<ScheduleRow>
        tableId="backup-schedules"
        columns={columns}
        dataSource={tableSource}
        loading={isLoading}
        rowKey="id"
        total={data?.total ?? tableSource.length}
        currentPage={page}
        pageSize={pageSize}
        onPageChange={(p, s) => {
          setPage(p);
          setPageSize(s);
        }}
        onRefresh={() => void refetch()}
        scroll={{ x: 1200 }}
      />

      <Modal
        title={editingRow ? t('backup.editSchedule') : t('backup.newSchedule')}
        open={modalVisible}
        onOk={() => void handleSave()}
        onCancel={closeModal}
        okText={t('common.save')}
        cancelText={t('common.cancel')}
        width={620}
        confirmLoading={createSchedule.isPending || updateSchedule.isPending}
        destroyOnHidden
      >
        <Form<ScheduleFormValues> form={form} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item
            label={t('table.name')}
            name="scheduleName"
            rules={[{ required: true, message: t('backup.scheduleNameRequired') }]}
          >
            <Input maxLength={128} />
          </Form.Item>

          <Form.Item label={t('backup.cronPreset')} name="cronPreset">
            <Radio.Group onChange={(e) => handlePresetChange(e.target.value as string)}>
              {CRON_PRESETS.map((p) => (
                <Radio key={p.key} value={p.key}>
                  {t(p.labelKey)}
                </Radio>
              ))}
            </Radio.Group>
          </Form.Item>

          <Form.Item
            label={t('backup.cronExpression')}
            name="cronExpression"
            rules={[
              { required: true, message: t('backup.cronExprInvalid') },
              {
                validator: (_rule, value: string) =>
                  isValidCron(value) ? Promise.resolve() : Promise.reject(new Error(t('backup.cronExprInvalid'))),
              },
            ]}
            extra={cronPreset === 'custom' ? t('backup.cronCustomHint') : undefined}
          >
            <Input
              placeholder="* * * * *"
              disabled={cronPreset !== 'custom'}
              style={{ fontFamily: 'monospace' }}
            />
          </Form.Item>

          <Form.Item
            label={t('backup.backupType')}
            name="backupType"
            rules={[{ required: true }]}
          >
            <Radio.Group>
              <Radio value="full">{t('backup.fullBackup')}</Radio>
              <Radio value="incremental">{t('backup.incrementalBackup')}</Radio>
              <Radio value="config-only">{t('backup.configBackup')}</Radio>
            </Radio.Group>
          </Form.Item>

          <Form.Item
            label={t('backup.deviceGroups')}
            name="deviceGroups"
            rules={[{ required: true, type: 'array', min: 1 }]}
          >
            <Select
              mode="multiple"
              placeholder={t('common.placeholder')}
              options={groupOptions}
              optionFilterProp="label"
              showSearch
            />
          </Form.Item>

          <Form.Item
            label={t('common.enable')}
            name="enabled"
            valuePropName="checked"
          >
            {/* enabled default 在 openCreate/openEdit 通过 form.setFieldsValue 注入；不再走 initialValue */}
            <Switch checkedChildren={t('status.enabled')} unCheckedChildren={t('status.disabled')} />
          </Form.Item>
        </Form>
      </Modal>
    </ListPageLayout>
  );
}
