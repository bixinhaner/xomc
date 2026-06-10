import { useState, useMemo } from 'react';
import { Button, Card, message, Modal, Tooltip, Badge, Space } from 'antd';
import { PlusOutlined, StopOutlined, DeleteOutlined, EyeOutlined } from '@ant-design/icons';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useT } from '@/hooks/useT';
import {
  useMRTasks,
  useStopMRTask,
  useDeleteMRTask,
} from '@core/hooks/api/useMrTasks';
import type {
  MRTask,
  MRTaskStatus,
} from '@core/types/mrTask';

import CreateDrawer from './CreateDrawer';
import DetailDrawer from './DetailDrawer';

/**
 * MRTasksPanel — MR 任务管理面板（无 ListPageLayout 页面 chrome）。
 *
 * 用户决策（2026-05-25）："MR 只活在文件传输模块"，因此该面板设计为可被
 * FileTransferCenter 内联嵌入；不再单独占用 /mr/tasks 路由 + sidebar 菜单。
 *
 * **受控/非受控模式**（2026-05-26）：
 *   - 非受控（standalone 占位用法）：面板自己渲染"标题 + 新建按钮"
 *   - 受控（嵌入 FileTransferCenter）：parent 传 createOpen/onCreateOpenChange，
 *     面板隐藏内部触发按钮，让 parent 的顶部 "New Task" 按钮接管 — 与 UFTE 一致
 *
 * 流程：
 *   - 列表：分页 + 状态/关键字筛选 + 行操作（查看/停止/删除）
 *   - 新建：抽屉表单（task_name + MR 参数 + 起止时间 + 目标设备）
 *   - 详情：抽屉（任务概要 + cell 进度表，10s 轮询）
 *
 * F05 PRD docs/project/prd/F05-mr-task-management.md
 */
interface MRTasksPanelProps {
  /** parent 控制的"新建抽屉是否打开"。给值就是受控模式 — 面板内部不再渲染新建按钮 */
  createOpen?: boolean;
  /** 配合 createOpen — close 时由 parent setState(false) */
  onCreateOpenChange?: (open: boolean) => void;
}

// 状态 → Antd Badge 颜色映射
const statusBadgeMap: Record<MRTaskStatus, 'default' | 'processing' | 'success' | 'warning' | 'error'> = {
  waitting: 'default',
  on: 'processing',
  off: 'success',
  suspend: 'warning',
  termination: 'warning',
};

export default function MRTasksPanel(props: MRTasksPanelProps = {}) {
  const t = useT();
  const isControlled = props.createOpen !== undefined;

  // 列表筛选 / 分页
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [selectedKeys, setSelectedKeys] = useState<React.Key[]>([]);

  // 抽屉
  const [internalCreateOpen, setInternalCreateOpen] = useState(false);
  const createOpen = isControlled ? Boolean(props.createOpen) : internalCreateOpen;
  const setCreateOpen = (next: boolean) => {
    if (isControlled) {
      props.onCreateOpenChange?.(next);
    } else {
      setInternalCreateOpen(next);
    }
  };
  const [detailTaskId, setDetailTaskId] = useState<string | undefined>(undefined);

  // 数据
  const listFilter = useMemo(
    () => ({
      page,
      pageSize,
      keyword: filters.keyword ? String(filters.keyword) : undefined,
      status: filters.status ? (filters.status as MRTaskStatus) : undefined,
    }),
    [page, pageSize, filters],
  );
  const { data, isLoading, refetch } = useMRTasks(listFilter);

  const stopMutation = useStopMRTask();
  const deleteMutation = useDeleteMRTask();

  // 筛选栏
  const filterFields: FilterField[] = useMemo(
    () => [
      {
        name: 'keyword',
        label: t('mrTask.field.taskName'),
        type: 'input',
        placeholder: t('mrTask.field.taskNamePlaceholder'),
      },
      {
        name: 'status',
        label: t('mrTask.field.status'),
        type: 'select',
        options: [
          { label: t('mrTask.status.waitting'), value: 'waitting' },
          { label: t('mrTask.status.on'), value: 'on' },
          { label: t('mrTask.status.off'), value: 'off' },
          { label: t('mrTask.status.termination'), value: 'termination' },
        ],
      },
    ],
    [t],
  );

  // 操作 — 停止
  const handleStop = (record: MRTask) => {
    Modal.confirm({
      title: t('mrTask.confirm.stop', { name: record.taskName }),
      content: t('mrTask.confirm.stopHint'),
      okType: 'danger',
      onOk: async () => {
        try {
          await stopMutation.mutateAsync(record.taskId);
          void message.success(t('mrTask.toast.stopSuccess'));
        } catch (e) {
          const msg = e instanceof Error ? e.message : String(e);
          void message.error(t('mrTask.toast.stopFailed', { msg }));
        }
      },
    });
  };

  // 操作 — 删除
  const handleDelete = (record: MRTask) => {
    Modal.confirm({
      title: t('mrTask.confirm.delete', { name: record.taskName }),
      content: t('mrTask.confirm.deleteHint'),
      okType: 'danger',
      onOk: async () => {
        try {
          await deleteMutation.mutateAsync(record.taskId);
          void message.success(t('mrTask.toast.deleteSuccess'));
        } catch (e) {
          const msg = e instanceof Error ? e.message : String(e);
          void message.error(t('mrTask.toast.deleteFailed', { msg }));
        }
      },
    });
  };

  const columns: DataTableColumn<MRTask & Record<string, unknown>>[] = useMemo(
    () => [
      {
        key: 'taskName',
        title: t('mrTask.field.taskName'),
        dataIndex: 'taskName',
        width: 260,
        ellipsis: true,
      },
      {
        key: 'taskStatus',
        title: t('mrTask.field.status'),
        dataIndex: 'taskStatus',
        width: 100,
        render: (val) => {
          const s = val as MRTaskStatus;
          return <Badge status={statusBadgeMap[s] ?? 'default'} text={t(`mrTask.status.${s}`)} />;
        },
      },
      {
        // 平铺成逗号文本，避免 3 个 Tag 在窄列里换行
        key: 'mrType',
        title: t('mrTask.field.measureType'),
        dataIndex: 'mrType',
        width: 160,
        ellipsis: true,
        render: (val) => (
          <span style={{ whiteSpace: 'nowrap' }}>
            {String(val).split(',').map((s) => s.trim()).join(' / ')}
          </span>
        ),
      },
      {
        key: 'reportPeriod',
        title: t('mrTask.field.reportPeriod'),
        dataIndex: 'reportPeriod',
        width: 100,
        render: (val) => `${String(val)} ${t('mrTask.field.reportPeriodSuffix')}`,
      },
      {
        key: 'startTime',
        title: t('mrTask.field.startTime'),
        dataIndex: 'startTime',
        width: 170,
        render: (val) => new Date(String(val)).toLocaleString(),
      },
      {
        key: 'endTime',
        title: t('mrTask.field.endTime'),
        dataIndex: 'endTime',
        width: 170,
        render: (val) =>
          val ? (
            new Date(String(val)).toLocaleString()
          ) : (
            <span style={{ color: 'rgba(0,0,0,0.45)' }}>
              {t('mrTask.field.endTimeUnlimited')}
            </span>
          ),
      },
      {
        key: 'actions',
        title: t('table.operation'),
        dataIndex: 'taskId',
        width: 220,
        fixed: 'right',
        render: (_, record) => {
          const r = record as MRTask;
          const stoppable = r.taskStatus === 'waitting' || r.taskStatus === 'on';
          const deletable = r.taskStatus === 'off' || r.taskStatus === 'termination';
          return (
            <>
              <Button
                type="link"
                size="small"
                icon={<EyeOutlined />}
                onClick={() => setDetailTaskId(r.taskId)}
              >
                {t('mrTask.action.viewDetail')}
              </Button>
              <Tooltip title={stoppable ? undefined : t('mrTask.confirm.stopHint')}>
                <Button
                  type="link"
                  size="small"
                  icon={<StopOutlined />}
                  disabled={!stoppable}
                  onClick={() => handleStop(r)}
                >
                  {t('mrTask.action.stop')}
                </Button>
              </Tooltip>
              <Tooltip title={deletable ? undefined : t('mrTask.confirm.deleteHint')}>
                <Button
                  type="link"
                  size="small"
                  danger
                  icon={<DeleteOutlined />}
                  disabled={!deletable}
                  onClick={() => handleDelete(r)}
                >
                  {t('mrTask.action.delete')}
                </Button>
              </Tooltip>
            </>
          );
        },
      },
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [t],
  );

  const items = (data?.items ?? []) as (MRTask & Record<string, unknown>)[];
  const total = data?.total ?? 0;

  return (
    <Space orientation="vertical" size={12} style={{ width: '100%' }}>
      {/* 非受控模式（standalone）显示自己的标题 + 新建按钮；
          受控模式（嵌入 FileTransferCenter）由 parent 的顶部 New Task 按钮接管。 */}
      {!isControlled && (
        <Space style={{ justifyContent: 'space-between', width: '100%' }}>
          <span style={{ fontWeight: 600, fontSize: 16 }}>{t('mrTask.page.title')}</span>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => setCreateOpen(true)}
          >
            {t('mrTask.action.create')}
          </Button>
        </Space>
      )}
      <FilterBar
        filterId="mr-tasks-filter"
        fields={filterFields}
        onSearch={(vals) => {
          setFilters(vals);
          setPage(1);
        }}
        onReset={() => {
          setFilters({});
          setPage(1);
        }}
      />
      <Card
        size="small"
        variant="outlined"
        styles={{ body: { padding: 0 } }}
      >
        <DataTable
          tableId="mr-tasks-list"
          columns={columns}
          dataSource={items}
          loading={isLoading}
          rowKey="taskId"
          total={total}
          pageSize={pageSize}
          currentPage={page}
          onPageChange={(p, s) => {
            setPage(p);
            setPageSize(s);
          }}
          selectable
          selectedRowKeys={selectedKeys}
          onSelectionChange={(keys) => setSelectedKeys(keys)}
          scroll={{ x: 1500 }}
        />
      </Card>

      <CreateDrawer
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        onCreated={() => {
          setCreateOpen(false);
          void refetch();
        }}
      />

      <DetailDrawer
        taskId={detailTaskId}
        onClose={() => setDetailTaskId(undefined)}
      />
    </Space>
  );
}
