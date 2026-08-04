import { useMemo, useState } from 'react';
import {
  Button,
  Card,
  Empty,
  Input,
  message,
  Segmented,
  Space,
  Table,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { DownloadOutlined, ProfileOutlined, RedoOutlined } from '@ant-design/icons';
import { useExportTaskCSV, useExportTaskDeviceCSV } from '@core/hooks/api/useMmlConsole';
import { useT } from '@/hooks/useT';
import type {
  ExecMeta,
  ExecStatus,
  ResultColumn,
  ResultRow,
  UnverifiedReason,
} from '../types';
import { STATUS_META, UNVERIFIED_REASON_TEXT } from '../constants';
import { exportAll, exportOne, saveBlob } from '../download';
import { expandObjectPathColumns, PATH_FAILED_CELL } from '../adapters';
import ResultDetailModal from './ResultDetailModal';
import { usePermission } from '@core/hooks/usePermission';

const { Text } = Typography;

interface ResultTableProps {
  execMeta: ExecMeta | null;
  /** 当前记录的命令 ID（= mml_tasks.id，传给详情页展示/深链） */
  commandId: string | null;
  columns: ResultColumn[];
  rows: ResultRow[];
  running: boolean;
  hasExecuted: boolean;
  /** 基于设备的「重新执行」回调（仅 console 传入；任务记录-查看为只读不传）。 */
  onReexecute?: (deviceSn: string) => void;
}

type StatusFilter = 'all' | 'success' | 'failed';

/** 状态 Tag；unverified 悬浮显示原因（只写/重启生效/查询失败，设计 §3.11.2）。 */
function StatusTag({
  status,
  reason,
  t,
}: {
  status: ExecStatus;
  reason?: UnverifiedReason;
  t: (id: string) => string;
}) {
  const meta = STATUS_META[status];
  const tag = <Tag color={meta.color}>{t(meta.textKey)}</Tag>;
  if (status === 'unverified' && reason) {
    return <Tooltip title={t(UNVERIFIED_REASON_TEXT[reason])}>{tag}</Tooltip>;
  }
  return tag;
}

/**
 * 右侧执行结果表格（设计 §3.4，本次改版核心）—— 行=设备、列=参数路径（读）/状态+故障（写）。
 * 顶部汇总条（总数/成功/失败/用时 + 状态过滤 + 设备过滤 + 下载全部），失败行可展开看原始报文，
 * 行尾单设备下载，底部折叠保留全量原始报文。当前为 mock 数据驱动。
 */
export default function ResultTable({
  execMeta,
  commandId,
  columns,
  rows,
  running,
  hasExecuted,
  onReexecute,
}: ResultTableProps) {
  const t = useT();
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all');
  const [snKeyword, setSnKeyword] = useState('');
  const [viewingRow, setViewingRow] = useState<ResultRow | null>(null);

  // CSV 导出走后端（落 MinIO + 记入 mml_tasks），拿预签名 URL 触发浏览器下载。
  const exportCsv = useExportTaskCSV();
  const exportDeviceCsv = useExportTaskDeviceCSV();
  // issue #409：导出 / 重新执行的按钮级权限（无权限禁用 + Tooltip，不隐藏）。
  const canExportPerm = usePermission('mml:console:export');
  const canExecutePerm = usePermission('mml:console:execute');
  const displayColumns = useMemo(
    () => expandObjectPathColumns(columns, rows),
    [columns, rows],
  );

  const handleExportAllCsv = (): void => {
    if (!commandId) {
      // 无真实任务 ID（理论不达），回退客户端导出。
      exportAll('csv', displayColumns, rows, execMeta?.label ?? 'result');
      return;
    }
    const name = t('mml.consoleV2.result.summaryFileName', { name: execMeta?.commandName ?? execMeta?.label ?? 'mml-result' });
    exportCsv.mutate(commandId, {
      onSuccess: (blob) => {
        saveBlob(blob, name);
        void message.success(t('mml.consoleV2.result.summaryDownloaded'));
      },
      onError: (e) => void message.error(e instanceof Error ? e.message : t('mml.consoleV2.result.exportFailed')),
    });
  };

  const handleExportDeviceCsv = (deviceSn: string): void => {
    if (!commandId) {
      exportOne(displayColumns, rows.find((r) => r.deviceSn === deviceSn)!);
      return;
    }
    // 文件命名：命令名称 + 设备SN（与「下载全部」的命令名前缀口径一致）。
    const cmd = execMeta?.commandName ?? execMeta?.label ?? 'mml-result';
    exportDeviceCsv.mutate(
      { taskId: commandId, deviceSn },
      {
        onSuccess: (blob) => {
          saveBlob(blob, `${cmd}_${deviceSn}.csv`);
          void message.success(t('mml.consoleV2.result.deviceDownloaded', { sn: deviceSn }));
        },
        onError: (e) => void message.error(e instanceof Error ? e.message : t('mml.consoleV2.result.exportFailed')),
      },
    );
  };

  const stats = useMemo(() => {
    const total = rows.length;
    // 「执行中」= 待执行 + 执行中两态的设备行数（§3.11.7 需求②）
    const inProgress = rows.filter((r) => r.status === 'pending' || r.status === 'running').length;
    const success = rows.filter((r) => r.status === 'success').length;
    const unverified = rows.filter((r) => r.status === 'unverified').length;
    const mismatch = rows.filter((r) => r.status === 'mismatch').length;
    const failed = rows.filter((r) => r.status === 'failed').length;
    // 「问题行」= RPC 失败 + 核实未生效（mismatch）
    const problem = failed + mismatch;
    return { total, inProgress, success, unverified, mismatch, failed, problem };
  }, [rows]);

  // 写类（读后核实）才展示「未核实/未生效」统计；读类只有成功/失败。
  const isWrite = execMeta ? !execMeta.read : false;

  const filteredRows = useMemo(() => {
    const kw = snKeyword.trim().toLowerCase();
    return rows.filter((r) => {
      if (statusFilter === 'success' && r.status !== 'success') return false;
      if (statusFilter === 'failed' && r.status !== 'failed') return false;
      if (kw && !r.deviceSn.toLowerCase().includes(kw)) return false;
      return true;
    });
  }, [rows, statusFilter, snKeyword]);

  const tableColumns: ColumnsType<ResultRow> = useMemo(() => {
    const base: ColumnsType<ResultRow> = [
      {
        title: t('mml.consoleV2.result.col.deviceSn'),
        dataIndex: 'deviceSn',
        key: 'deviceSn',
        width: 190,
        fixed: 'left',
        ellipsis: true,
      },
      {
        title: t('mml.consoleV2.result.col.action'),
        key: 'action',
        width: 110,
        fixed: 'left',
        align: 'center',
        render: (_v, r) => (
          <Space size={0}>
            <Tooltip title={t('mml.consoleV2.result.view')}>
              <Button
                type="text"
                size="small"
                icon={<ProfileOutlined />}
                onClick={() => setViewingRow(r)}
              />
            </Tooltip>
            <Tooltip title={canExportPerm ? t('mml.consoleV2.result.downloadDevice') : t('common.noPermission')}>
              <Button
                type="text"
                size="small"
                icon={<DownloadOutlined />}
                disabled={!canExportPerm}
                loading={exportDeviceCsv.isPending && exportDeviceCsv.variables?.deviceSn === r.deviceSn}
                onClick={() => handleExportDeviceCsv(r.deviceSn)}
              />
            </Tooltip>
            {onReexecute && (
              <Tooltip title={canExecutePerm ? t('mml.consoleV2.result.reexecute') : t('common.noPermission')}>
                <Button
                  type="text"
                  size="small"
                  icon={<RedoOutlined />}
                  disabled={running || !canExecutePerm}
                  onClick={() => onReexecute(r.deviceSn)}
                />
              </Tooltip>
            )}
          </Space>
        ),
      },
      {
        title: t('mml.consoleV2.result.col.status'),
        dataIndex: 'status',
        key: 'status',
        width: 120,
        render: (s: ExecStatus, r) => <StatusTag status={s} reason={r.unverifiedReason} t={t} />,
      },
      {
        title: t('mml.planCommand'),
        key: 'planCommand',
        width: 150,
        ellipsis: true,
        render: (_v: unknown, r: ResultRow) => (
          <Text style={{ fontSize: 12 }} ellipsis={{ tooltip: r.commandName || r.commandCode || '-' }}>
            {r.commandName || r.commandCode || '-'}
          </Text>
        ),
      },
    ];

    const dynamic: ColumnsType<ResultRow> = displayColumns.map((c) => ({
      title: c.label,
      key: c.key,
      width: 140,
      ellipsis: true,
      render: (_v, r) => {
        const val = r.cells[c.path];
        if (r.status === 'failed') {
          // 逐 PATH：失败行仍含成功 path 的读回值与失败 path 的失败占位符，逐格呈现；
          // 整体下发失败（无单格值）回退「-」。
          if (val === PATH_FAILED_CELL) return <Text type="danger">{t('mml.consoleV2.result.pathFailed')}</Text>;
          return val ? <Text>{val}</Text> : <Text type="secondary">-</Text>;
        }
        // 未核实（只写/重启生效）：无读回值，灰显占位
        if (r.status === 'unverified') {
          return (
            <Tooltip title={t('mml.consoleV2.result.unverifiedCellTip')}>
              <Text type="secondary">—</Text>
            </Tooltip>
          );
        }
        // 未生效（写成功但读回不符）：读回值标红
        if (r.status === 'mismatch') {
          return val ? <Text type="danger">{val}</Text> : <Text type="secondary">-</Text>;
        }
        if (val === '✓') return <Tag color="success">✓</Tag>;
        // 成功但读回值为空（设备返回空串/无值）：显示「空」（多语言），区别于「-」（无结果）。
        if (val === '' || val == null) {
          return <Text type="secondary">{t('mml.console.emptyValue')}</Text>;
        }
        return <Text>{val}</Text>;
      },
    }));

    const tail: ColumnsType<ResultRow> = [
      {
        title: t('mml.consoleV2.result.col.dispatchedAt'),
        dataIndex: 'dispatchedAt',
        key: 'dispatchedAt',
        width: 104,
        render: (v?: string) =>
          v ? <Text style={{ fontSize: 12 }}>{v}</Text> : <Text type="secondary">-</Text>,
      },
      {
        title: t('mml.consoleV2.result.col.respondedAt'),
        dataIndex: 'respondedAt',
        key: 'respondedAt',
        width: 104,
        render: (v?: string) =>
          v ? <Text style={{ fontSize: 12 }}>{v}</Text> : <Text type="secondary">-</Text>,
      },
    ];

    return [...base, ...dynamic, ...tail];
    // commandId / 导出 mutation / 重新执行回调 / running 进依赖：切任务或对应状态变化时刷新「操作」列。
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [displayColumns, commandId, exportDeviceCsv.isPending, exportDeviceCsv.variables, onReexecute, running, t, canExportPerm, canExecutePerm]);

  return (
    <Card
      title={
        // 「执行结果」标题 + 当前命令名 + 汇总统计同一行（设计 §3.11.1 / §3.11.7 修订）
        <Space size={20} wrap style={{ rowGap: 4 }}>
          <Space size={6}>
            <span>{t('mml.consoleV2.result.title')}</span>
            {hasExecuted && execMeta && (
              <Text type="secondary" style={{ fontWeight: 400, fontSize: 14 }}>
                · {execMeta.commandName ?? execMeta.label}
              </Text>
            )}
          </Space>
          {hasExecuted && (
            <Space size={14} wrap style={{ fontWeight: 400, fontSize: 13 }}>
              <span>
                {t('mml.consoleV2.result.statTotal')} <b>{stats.total}</b>
              </span>
              <span style={{ color: '#1677ff' }}>{t('mml.consoleV2.result.statInProgress', { count: stats.inProgress })}</span>
              <span style={{ color: '#52c41a' }}>{t('mml.consoleV2.result.statSuccess', { count: stats.success })}</span>
              {isWrite && <span style={{ color: '#faad14' }}>{t('mml.consoleV2.result.statUnverified', { count: stats.unverified })}</span>}
              {isWrite && <span style={{ color: '#ff4d4f' }}>{t('mml.consoleV2.result.statMismatch', { count: stats.mismatch })}</span>}
              <span style={{ color: '#ff4d4f' }}>{t('mml.consoleV2.result.statFailed', { count: stats.failed })}</span>
            </Space>
          )}
        </Space>
      }
      variant="borderless"
      style={{ height: '100%', display: 'flex', flexDirection: 'column' }}
      styles={{ body: { padding: 16, flex: 1, minHeight: 0, overflow: 'auto' } }}
      extra={
        <Tooltip title={canExportPerm ? undefined : t('common.noPermission')}>
          <Button
            icon={<DownloadOutlined />}
            disabled={rows.length === 0 || !canExportPerm}
            loading={exportCsv.isPending}
            onClick={handleExportAllCsv}
          >
            {t('mml.consoleV2.result.downloadAll')}
          </Button>
        </Tooltip>
      }
    >
      {!hasExecuted ? (
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description={t('mml.consoleV2.result.emptyHint')}
          style={{ marginTop: 80 }}
        />
      ) : (
        <Space orientation="vertical" size={12} style={{ width: '100%' }}>
          <Space wrap style={{ justifyContent: 'space-between', width: '100%' }}>
            <Segmented<StatusFilter>
              value={statusFilter}
              onChange={(v) => setStatusFilter(v as StatusFilter)}
              options={[
                { label: t('mml.consoleV2.result.filterAll', { count: stats.total }), value: 'all' },
                { label: t('mml.consoleV2.result.filterSuccess', { count: stats.success }), value: 'success' },
                { label: t('mml.consoleV2.result.filterFailed', { count: stats.failed }), value: 'failed' },
              ]}
            />
            <Input.Search
              allowClear
              placeholder={t('mml.consoleV2.result.snFilterPlaceholder')}
              style={{ width: 240 }}
              value={snKeyword}
              onChange={(e) => setSnKeyword(e.target.value)}
            />
          </Space>

          <Table<ResultRow>
            rowKey="deviceSn"
            size="small"
            loading={running}
            columns={tableColumns}
            dataSource={filteredRows}
            scroll={{ x: 'max-content' }}
            sticky
            pagination={{ pageSize: 20, size: 'small', showTotal: (count) => t('mml.consoleV2.result.totalRows', { count }) }}
          />
        </Space>
      )}

      <ResultDetailModal
        open={!!viewingRow}
        row={viewingRow}
        execMeta={execMeta}
        commandId={commandId}
        columns={displayColumns}
        onClose={() => setViewingRow(null)}
      />
    </Card>
  );
}
