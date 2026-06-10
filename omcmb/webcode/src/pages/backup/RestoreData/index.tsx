import { useMemo, useState } from 'react';
import {
  App,
  Button,
  Drawer,
  Form,
  Input,
  Radio,
  Progress,
  Space,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';

import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useT } from '@/hooks/useT';

import type { RestoreTask, RestoreStatus } from '@core/mock/data/backup';
import {
  useBackupRestoreTasks,
  useCreateBackupRestore,
  useCreateBackupRestoreByTaskID,
} from '@core/hooks/api/useBackup';

// T-0078: wholesale rewrite of the prior 1123-line mock page. The new page
// consumes the T-0072 backend endpoints (POST /backup/restore + GET
// /backup/restore-tasks). Path browsing is deferred to T-0079 (which fills
// backup_tasks.file_path and unlocks "select from history") — for now the
// operator pastes a known {bucket, object_path} pair.

const STATUS_TAG: Record<RestoreStatus, { color: string; key: string }> = {
  pending: { color: 'default', key: 'backup.restore.statusPending' },
  running: { color: 'processing', key: 'backup.restore.statusRunning' },
  completed: { color: 'success', key: 'backup.restore.statusCompleted' },
  failed: { color: 'error', key: 'backup.restore.statusFailed' },
  cancelled: { color: 'warning', key: 'backup.restore.statusCancelled' },
};

interface RestoreFormValues {
  backupTaskId: string;
  bucket: string;
  objectPath: string;
  targetDeviceSns: string;
}

function formatDate(iso?: string): string {
  if (!iso) return '-';
  return dayjs(iso).format('YYYY-MM-DD HH:mm:ss');
}

function parseSnList(raw: string): string[] {
  return raw
    .split(/[\r\n]+/)
    .map((s) => s.trim())
    .filter(Boolean);
}

export default function RestoreData() {
  const t = useT();
  const { message } = App.useApp();

  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [restoreMode, setRestoreMode] = useState<'path' | 'task'>('path');
  const [form] = Form.useForm<RestoreFormValues>();

  const { data, isLoading, refetch } = useBackupRestoreTasks({
    page,
    pageSize,
  });
  const createMutation = useCreateBackupRestore();
  const createByTaskIdMutation = useCreateBackupRestoreByTaskID();

  const items: RestoreTask[] = data?.items ?? [];

  const columns: DataTableColumn<RestoreTask>[] = useMemo(
    () => [
      {
        key: 'sourcePath',
        title: t('backup.restore.sourcePath'),
        dataIndex: 'sourceObjectPath',
        ellipsis: true,
        render: (_: unknown, r: RestoreTask) => (
          <Tooltip title={`${r.sourceBucket}/${r.sourceObjectPath}`}>
            <Typography.Text code>{r.sourceObjectPath}</Typography.Text>
          </Tooltip>
        ),
      },
      {
        key: 'targetCount',
        title: t('backup.restore.targetCount'),
        width: 110,
        render: (_: unknown, r: RestoreTask) => r.targetDeviceSns.length,
      },
      {
        key: 'status',
        title: t('common.status'),
        width: 110,
        render: (_: unknown, r: RestoreTask) => {
          const cfg = STATUS_TAG[r.status];
          return <Tag color={cfg.color}>{t(cfg.key)}</Tag>;
        },
      },
      {
        key: 'progress',
        title: t('common.progress'),
        width: 160,
        render: (_: unknown, r: RestoreTask) => (
          <Progress
            percent={r.progress}
            size="small"
            status={
              r.status === 'failed'
                ? 'exception'
                : r.status === 'completed'
                  ? 'success'
                  : 'active'
            }
          />
        ),
      },
      {
        key: 'errorMessage',
        title: t('common.error'),
        width: 80,
        render: (_: unknown, r: RestoreTask) =>
          r.errorMessage ? (
            <Tooltip title={r.errorMessage}>
              <Tag color="error">!</Tag>
            </Tooltip>
          ) : null,
      },
      {
        key: 'startedAt',
        title: t('backup.restore.startedAt'),
        width: 180,
        render: (_: unknown, r: RestoreTask) => formatDate(r.startedAt),
      },
      {
        key: 'completedAt',
        title: t('backup.restore.completedAt'),
        width: 180,
        render: (_: unknown, r: RestoreTask) => formatDate(r.completedAt),
      },
      {
        key: 'createdBy',
        title: t('backup.restore.createdBy'),
        width: 140,
        render: (_: unknown, r: RestoreTask) => r.createdBy ?? '-',
      },
    ],
    [t]
  );

  const handleCreate = async () => {
    try {
      const values = await form.validateFields();
      const sns = parseSnList(values.targetDeviceSns);
      // SN list non-emptiness now enforced by the Form.Item validator below;
      // this guard is defense-in-depth in case validateFields is bypassed.
      if (sns.length === 0) return;
      if (restoreMode === 'task') {
        await createByTaskIdMutation.mutateAsync({
          backupTaskId: values.backupTaskId,
          targetDeviceSns: sns,
        });
      } else {
        await createMutation.mutateAsync({
          bucket: values.bucket,
          objectPath: values.objectPath,
          targetDeviceSns: sns,
        });
      }
      message.success(t('backup.restore.submitSuccess'));
      setDrawerOpen(false);
      form.resetFields();
    } catch (err: unknown) {
      // Validation rejection arrives as ValidateErrorEntity — let antd surface
      // field-level errors and only toast on real submission failures.
      if (err && typeof err === 'object' && 'errorFields' in err) return;
      const detail =
        err instanceof Error ? err.message : String(err ?? 'unknown');
      message.error(
        t('backup.restore.submitFailed', { error: detail })
      );
    }
  };

  return (
    <ListPageLayout
      title={t('backup.restore.title')}
      extra={
        <Space>
          <Button
            icon={<ReloadOutlined />}
            onClick={() => {
              void refetch();
            }}
          >
            {t('common.refresh')}
          </Button>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => setDrawerOpen(true)}
          >
            {t('backup.restore.create')}
          </Button>
        </Space>
      }
    >
      <DataTable
        tableId="backup-restore-tasks"
        columns={columns}
        dataSource={items}
        rowKey="id"
        loading={isLoading}
        total={data?.total ?? 0}
        currentPage={page}
        pageSize={pageSize}
        onPageChange={(p, ps) => {
          setPage(p);
          setPageSize(ps);
        }}
        onRefresh={() => {
          void refetch();
        }}
      />

      <Drawer
        title={t('backup.restore.create')}
        placement="right"
        size={520}
        open={drawerOpen}
        onClose={() => {
          setDrawerOpen(false);
          setRestoreMode('path');
        }}
        destroyOnHidden
        extra={
          <Space>
            <Button onClick={() => setDrawerOpen(false)}>
              {t('common.cancel')}
            </Button>
            <Button
              type="primary"
              loading={createMutation.isPending}
              onClick={() => {
                void handleCreate();
              }}
            >
              {t('backup.restore.submit')}
            </Button>
          </Space>
        }
      >
        <Form
          form={form}
          layout="vertical"
          initialValues={{ bucket: 'config_backup' }}
        >
          <Form.Item label={t('backup.restore.mode')}>
            <Radio.Group
              value={restoreMode}
              onChange={(e) => {
                setRestoreMode(e.target.value);
                form.resetFields(['backupTaskId', 'bucket', 'objectPath']);
                form.setFieldsValue({ bucket: 'config_backup', targetDeviceSns: '' });
              }}
            >
              <Radio.Button value="path">{t('backup.restore.modePath')}</Radio.Button>
              <Radio.Button value="task">{t('backup.restore.modeTask')}</Radio.Button>
            </Radio.Group>
          </Form.Item>

          {restoreMode === 'task' ? (
            <Form.Item
              name="backupTaskId"
              label={t('backup.restore.backupTaskId')}
              extra={t('backup.restore.backupTaskIdHint')}
              rules={[{ required: true }]}
            >
              <Input />
            </Form.Item>
          ) : null}

          {restoreMode === 'path' ? (
            <>
              <Form.Item
                name="bucket"
                label={t('backup.restore.bucket')}
                extra={t('backup.restore.bucketHint')}
                rules={[
                  { required: true },
                  {
                    pattern: /^config_backup$/,
                    message: t('backup.restore.bucketHint'),
                  },
                ]}
              >
                <Input />
              </Form.Item>

              <Form.Item
                name="objectPath"
                label={t('backup.restore.objectPath')}
                extra={t('backup.restore.objectPathHint')}
                rules={[
                  { required: true },
                  {
                    validator: (_, value: string) => {
                      if (!value) return Promise.resolve();
                      if (value.includes('..') || value.startsWith('/')) {
                        return Promise.reject(
                          new Error(t('backup.restore.pathTraversal'))
                        );
                      }
                      return Promise.resolve();
                    },
                  },
                ]}
              >
                <Input
                  placeholder={t('backup.restore.objectPathPlaceholder')}
                />
              </Form.Item>
            </>
          ) : null}

          <Form.Item
            name="targetDeviceSns"
            label={t('backup.restore.targetDevices')}
            extra={t('backup.restore.targetDevicesHint')}
            rules={[
              { required: true },
              {
                // Reject all-whitespace input so the error renders inline as a
                // field-level message (review M2 fix — avoids inconsistent UX
                // where bucket/objectPath errors are inline but SN errors
                // surfaced as a top toast).
                validator: (_, value: string) => {
                  if (!value) return Promise.resolve();
                  const sns = parseSnList(value);
                  return sns.length === 0
                    ? Promise.reject(
                        new Error(t('backup.restore.targetDevicesHint'))
                      )
                    : Promise.resolve();
                },
              },
            ]}
          >
            <Input.TextArea
              rows={6}
              placeholder={t('backup.restore.targetDevicesPlaceholder')}
            />
          </Form.Item>
        </Form>
      </Drawer>
    </ListPageLayout>
  );
}
