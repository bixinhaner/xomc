import {
  Drawer,
  Descriptions,
  Tag,
  Table,
  Badge,
  Empty,
  Skeleton,
} from 'antd';
import { useT } from '@/hooks/useT';
import { useMRTask } from '@core/hooks/api/useMrTasks';
import type { MRTaskStatus } from '@core/types/mrTask';

interface DetailDrawerProps {
  taskId?: string;
  onClose: () => void;
}

const taskStatusBadgeMap: Record<MRTaskStatus, 'default' | 'processing' | 'success' | 'warning' | 'error'> = {
  waitting: 'default',
  on: 'processing',
  off: 'success',
  suspend: 'warning',
  termination: 'warning',
};

/**
 * MR 任务详情抽屉。
 *
 * 简化版（2026-05-26）：
 *   - 不再展示 task_id（用户视角无意义的 UUID）
 *   - 不再展示 cell-level 进度（cell 维度对用户太细），改为"本次任务的目标设备"列表
 */
export default function DetailDrawer({ taskId, onClose }: DetailDrawerProps) {
  const t = useT();

  const { data: task, isLoading: taskLoading } = useMRTask(taskId);

  const renderSummary = () => {
    if (taskLoading) return <Skeleton active />;
    if (!task) return <Empty />;
    return (
      <Descriptions size="small" column={2} bordered>
        <Descriptions.Item label={t('mrTask.field.taskName')}>{task.taskName}</Descriptions.Item>
        <Descriptions.Item label={t('mrTask.field.status')}>
          <Badge
            status={taskStatusBadgeMap[task.taskStatus] ?? 'default'}
            text={t(`mrTask.status.${task.taskStatus}`)}
          />
        </Descriptions.Item>
        <Descriptions.Item label={t('mrTask.field.measureType')} span={2}>
          {task.mrType.split(',').map((s) => s.trim()).join(' / ')}
        </Descriptions.Item>
        <Descriptions.Item label={t('mrTask.field.statisPeriod')}>
          {task.statisPeriod}
        </Descriptions.Item>
        <Descriptions.Item label={t('mrTask.field.reportPeriod')}>
          {task.reportPeriod} {t('mrTask.field.reportPeriodSuffix')}
        </Descriptions.Item>
        <Descriptions.Item label={t('mrTask.field.startTime')}>
          {new Date(task.startTime).toLocaleString()}
        </Descriptions.Item>
        <Descriptions.Item label={t('mrTask.field.endTime')}>
          {task.endTime ? (
            new Date(task.endTime).toLocaleString()
          ) : (
            <span style={{ color: 'rgba(0,0,0,0.45)' }}>
              {t('mrTask.field.endTimeUnlimited')}
            </span>
          )}
        </Descriptions.Item>
        <Descriptions.Item label={t('mrTask.field.creator')} span={2}>{task.creator}</Descriptions.Item>
      </Descriptions>
    );
  };

  // 目标设备：直接渲染 task.targetDeviceSns（SN 字符串数组）
  const deviceRows = (task?.targetDeviceSns ?? []).map((sn, idx) => ({ key: idx, sn }));

  return (
    <Drawer
      title={t('mrTask.detail.title')}
      width={760}
      open={Boolean(taskId)}
      onClose={onClose}
      destroyOnHidden
    >
      <div style={{ marginBottom: 16 }}>
        <h4 style={{ marginBottom: 8 }}>{t('mrTask.detail.section.summary')}</h4>
        {renderSummary()}
      </div>
      <div>
        <h4 style={{ marginBottom: 8 }}>
          {t('mrTask.detail.section.targetDevices')}
          <Tag style={{ marginLeft: 8 }}>{deviceRows.length}</Tag>
        </h4>
        <Table
          size="small"
          rowKey="key"
          dataSource={deviceRows}
          columns={[
            {
              title: t('mrTask.field.deviceSn'),
              dataIndex: 'sn',
              key: 'sn',
              render: (val: string) => (
                <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{val}</span>
              ),
            },
          ]}
          locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={null} /> }}
          pagination={deviceRows.length > 10 ? { pageSize: 10 } : false}
        />
      </div>
    </Drawer>
  );
}
