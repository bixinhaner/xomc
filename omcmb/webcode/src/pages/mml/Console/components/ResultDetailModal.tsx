import { Alert, Button, Descriptions, message, Modal, Popover, Space, Table, Tag, Tooltip, Typography } from 'antd';
import { DownloadOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { useT } from '@/hooks/useT';
import type { ExecMeta, ExecStatus, PathTask, ResultColumn, ResultRow, VerifyItem } from '../types';
import { STATUS_META, UNVERIFIED_REASON_TEXT, opColor, opLabelI18nKey } from '../constants';
import { exportOne, saveBlob } from '../download';
import { useExportTaskDeviceCSV } from '@core/hooks/api/useMmlConsole';
import { usePermission } from '@core/hooks/usePermission';
import {
  DISPATCH_FAILED_FALLBACK,
  PARTIAL_PATH_FAILED_FALLBACK,
  PATH_TASK_FAILED_FALLBACK,
  READBACK_FAILED_FALLBACK,
} from '../adapters';

const { Text } = Typography;

function localizedFallback(value: string | undefined, t: (id: string) => string): string | undefined {
  switch (value) {
    case PATH_TASK_FAILED_FALLBACK:
      return t('mml.consoleV2.result.pathFailed');
    case PARTIAL_PATH_FAILED_FALLBACK:
      return t('mml.consoleV2.result.partialPathFailed');
    case DISPATCH_FAILED_FALLBACK:
      return t('mml.consoleV2.detail.dispatchFailedFallback');
    case READBACK_FAILED_FALLBACK:
      return t('mml.consoleV2.detail.readbackFailedFallback');
    default:
      return value;
  }
}

interface ResultDetailModalProps {
  open: boolean;
  row: ResultRow | null;
  execMeta: ExecMeta | null;
  /** 命令 ID（= mml_tasks.id，每次批量执行全局唯一） */
  commandId: string | null;
  columns: ResultColumn[];
  onClose: () => void;
}

interface ParsedParam {
  key: string;
  path: string;
  label: string;
  value: string;
}

/**
 * 单设备执行详情（设计 §3.11）—— 任务信息（命令 ID / 设备任务 ID / 下发·响应时间）+ 执行状态 +
 * 读类参数值表 / 写类「读后核实」对比（预期 vs 读回）+ 逐 PATH 子任务 + 格式化结果报文。
 * 当前为 mock 数据。
 */
export default function ResultDetailModal({
  open,
  row,
  execMeta,
  commandId,
  columns,
  onClose,
}: ResultDetailModalProps) {
  const t = useT();
  // 「下载结果」统一走后端单设备 CSV（与操作列下载同一路径 buildDeviceCSVMulti 竖表），
  // 不再用客户端 exportOne 的横表格式；仅无真实任务 ID 的兜底场景退回 exportOne。
  const exportDeviceCsv = useExportTaskDeviceCSV();
  const canExportPerm = usePermission('mml:console:export');

  const handleDownloadResult = (): void => {
    if (!row) return;
    if (!commandId) {
      exportOne(columns, row);
      return;
    }
    const cmd = execMeta?.commandName ?? execMeta?.label ?? 'mml-result';
    exportDeviceCsv.mutate(
      { taskId: commandId, deviceSn: row.deviceSn },
      {
        onSuccess: (blob) => {
          saveBlob(blob, `${cmd}_${row.deviceSn}.csv`);
          void message.success(t('mml.consoleV2.result.deviceDownloaded', { sn: row.deviceSn }));
        },
        onError: (e) =>
          void message.error(e instanceof Error ? e.message : t('mml.consoleV2.result.exportFailed')),
      },
    );
  };

  const read = execMeta?.read ?? true;
  const status = row?.status;
  const opText = execMeta ? t(opLabelI18nKey(execMeta.operationType) ?? '') : '';

  const parsedParams: ParsedParam[] = row
    ? columns.map((c) => ({
        key: c.key,
        path: c.path,
        label: c.label,
        value: row.cells[c.path] ?? '',
      }))
    : [];

  const paramColumns: ColumnsType<ParsedParam> = [
    {
      title: t('mml.consoleV2.detail.col.paramPath'),
      dataIndex: 'path',
      key: 'path',
      render: (v: string, r) => (
        <span>
          <Text>{r.label}</Text>
          <br />
          <Text code style={{ fontSize: 11, wordBreak: 'break-all' }}>
            {v}
          </Text>
        </span>
      ),
    },
    {
      title: t('mml.consoleV2.detail.col.value'),
      dataIndex: 'value',
      key: 'value',
      width: 220,
      render: (v: string) =>
        v ? (
          <Text style={{ fontFamily: 'monospace', fontSize: 12, wordBreak: 'break-all' }}>{v}</Text>
        ) : (
          <Text type="secondary">{t('mml.consoleV2.detail.emptyParen')}</Text>
        ),
    },
  ];

  // 读后核实对比表（写类，§3.11.2）
  const verifyColumns: ColumnsType<VerifyItem> = [
    {
      title: t('mml.consoleV2.detail.col.param'),
      dataIndex: 'label',
      key: 'label',
      render: (_v, r) => (
        <span>
          <Text>{r.label}</Text>
          <br />
          <Text code style={{ fontSize: 11, wordBreak: 'break-all' }}>
            {r.path}
          </Text>
        </span>
      ),
    },
    {
      title: t('mml.consoleV2.detail.col.expected'),
      dataIndex: 'expected',
      key: 'expected',
      width: 150,
      render: (v: string) => (
        <Text style={{ fontFamily: 'monospace', fontSize: 12 }}>{v || '-'}</Text>
      ),
    },
    {
      title: t('mml.consoleV2.detail.col.actual'),
      dataIndex: 'actual',
      key: 'actual',
      width: 150,
      render: (v: string, r) =>
        v ? (
          <Text type={r.matched ? undefined : 'danger'} style={{ fontFamily: 'monospace', fontSize: 12 }}>
            {v}
          </Text>
        ) : (
          <Text type="secondary">{t('mml.consoleV2.detail.notReadBack')}</Text>
        ),
    },
    {
      title: t('mml.consoleV2.detail.col.verify'),
      dataIndex: 'matched',
      key: 'matched',
      width: 90,
      render: (m: boolean) => (m ? <Tag color="success">{t('mml.consoleV2.detail.matched')}</Tag> : <Tag color="error">{t('mml.consoleV2.detail.mismatched')}</Tag>),
    },
  ];

  // PATH 列表（#196：MOD 下发 + LST 回读前后对比）。列序：类型 | 子任务 ID | 状态 | PATH | PATH 值。
  const pathTaskColumns: ColumnsType<PathTask> = [
    {
      title: t('mml.consoleV2.detail.col.type'),
      dataIndex: 'opType',
      key: 'opType',
      width: 96,
      render: (v?: string) =>
        v === 'MOD' ? (
          <Tag color="blue">{t('mml.consoleV2.detail.modDispatch')}</Tag>
        ) : v === 'LST' ? (
          <Tag color="green">{t('mml.consoleV2.detail.lstReadback')}</Tag>
        ) : (
          <Text type="secondary">-</Text>
        ),
    },
    {
      title: t('mml.consoleV2.detail.col.subTaskId'),
      dataIndex: 'subTaskId',
      key: 'subTaskId',
      width: 88,
      align: 'center',
      // #196：不直接显示冗长 ID，仅提供「复制」按钮（点击复制完整 device_task id）；无 ID 时占位 -。
      render: (v: string) =>
        v ? (
          <Text copyable={{ text: v, tooltips: [t('mml.consoleV2.detail.copySubTaskId'), t('mml.consoleV2.detail.copied')] }} style={{ fontSize: 12 }}>
            {t('mml.consoleV2.detail.copy')}
          </Text>
        ) : (
          <Text type="secondary" style={{ fontSize: 11 }}>
            -
          </Text>
        ),
    },
    {
      title: t('mml.consoleV2.detail.col.status'),
      dataIndex: 'status',
      key: 'status',
      width: 80,
      render: (s: ExecStatus) => <Tag color={STATUS_META[s].color}>{t(STATUS_META[s].textKey)}</Tag>,
    },
    {
      title: 'PATH',
      dataIndex: 'path',
      key: 'path',
      render: (v: string) => (
        <Text code style={{ fontSize: 11, wordBreak: 'break-all' }}>
          {v}
        </Text>
      ),
    },
    {
      // MOD 行显下发值、LST 行显回读值；失败时显故障原因。值可能很长 → 截断 + 悬停看全 + 复制。
      title: t('mml.consoleV2.detail.col.pathValue'),
      dataIndex: 'value',
      key: 'value',
      width: 220,
      render: (v: string) => {
        const text = localizedFallback(v, t);
        return text ? (
          <Text
            copyable={{ text, tooltips: [t('mml.consoleV2.detail.copy'), t('mml.consoleV2.detail.copied')] }}
            ellipsis={{ tooltip: text }}
            style={{ fontSize: 11, maxWidth: 180 }}
          >
            {text}
          </Text>
        ) : (
          <Text type="secondary">-</Text>
        );
      },
    },
  ];

  return (
    <Modal
      title={row ? t('mml.consoleV2.detail.titleWithSn', { sn: row.deviceSn }) : t('mml.consoleV2.detail.title')}
      open={open}
      onCancel={onClose}
      width={780}
      destroyOnHidden
      footer={
        <Space>
          <Tooltip title={canExportPerm ? undefined : '无导出权限'}>
            <Button
              icon={<DownloadOutlined />}
              disabled={!row || !canExportPerm}
              loading={exportDeviceCsv.isPending}
              onClick={handleDownloadResult}
            >
              {t('mml.consoleV2.detail.downloadResult')}
            </Button>
          </Tooltip>
          <Button onClick={onClose}>{t('common.close')}</Button>
        </Space>
      }
    >
      {row && execMeta && (
        <Space orientation="vertical" size={14} style={{ width: '100%' }}>
          {/* 任务信息 */}
          <div style={{ padding: 12, background: '#fafafa', border: '1px solid #f0f0f0', borderRadius: 6 }}>
            <Descriptions
              size="small"
              column={1}
              styles={{ label: { width: 96, color: '#595959' } }}
              items={[
                {
                  key: 'taskId',
                  label: t('mml.consoleV2.detail.taskId'),
                  children: commandId ? (
                    <Text code copyable={{ text: commandId }} style={{ fontSize: 12 }}>
                      {commandId}
                    </Text>
                  ) : (
                    <Text type="secondary">-</Text>
                  ),
                },
                {
                  key: 'cmd',
                  label: t('mml.consoleV2.detail.execCommand'),
                  children: (
                    <Space size={6} wrap>
                      {/* BUG-10 修复：若 commandName 已以操作类型中文开头（如「查询 设备基本信息」），
                          Tag 只显示操作类型码（如 LST），避免「LST 查询 查询 设备基本信息」叠词 */}
                      <Tag color={opColor(execMeta.operationType)} style={{ marginInlineEnd: 0 }}>
                        {execMeta.operationType}
                        {opText && !(execMeta.commandName ?? execMeta.label)?.startsWith(opText) && ` ${opText}`}
                      </Tag>
                      <Text strong>{execMeta.commandName ?? execMeta.label}</Text>
                      {columns.length > 0 && (
                        <Popover
                          trigger="hover"
                          placement="bottomLeft"
                          title={t('mml.consoleV2.detail.execPathTitle', { count: columns.length })}
                          content={
                            // MOD 与 LST 同一样式：仅展示标签 + PATH（MOD 下发值在下方「PATH 列表」展示，不在此重复）。
                            <div style={{ maxHeight: 320, overflow: 'auto', maxWidth: 460 }}>
                              {columns.map((c) => (
                                <div key={c.key} style={{ marginBottom: 6, lineHeight: 1.4 }}>
                                  <Text style={{ fontSize: 12 }}>{c.label}</Text>
                                  <br />
                                  <Text code style={{ fontSize: 11, wordBreak: 'break-all' }}>
                                    {c.path}
                                  </Text>
                                </div>
                              ))}
                            </div>
                          }
                        >
                          <Text
                            style={{
                              color: '#1677ff',
                              cursor: 'pointer',
                              borderBottom: '1px dashed #1677ff',
                              fontSize: 12,
                            }}
                          >
                            PATH ({columns.length})
                          </Text>
                        </Popover>
                      )}
                    </Space>
                  ),
                },
                {
                  key: 'device',
                  label: t('mml.consoleV2.detail.deviceSn'),
                  children: <Text>{row.deviceSn}</Text>,
                },
                {
                  key: 'status',
                  label: t('mml.consoleV2.detail.execStatus'),
                  children: (
                    <Space size={12} wrap>
                      <Tag color={STATUS_META[row.status].color}>{t(STATUS_META[row.status].textKey)}</Tag>
                      <Text type="secondary">{t('mml.consoleV2.detail.elapsed', { sec: (row.elapsedMs / 1000).toFixed(1) })}</Text>
                      <Text type="secondary">
                        {t('mml.consoleV2.detail.dispatchResponse', { dispatched: row.dispatchedAt ?? '-', responded: row.respondedAt ?? '-' })}
                      </Text>
                    </Space>
                  ),
                },
              ]}
            />
          </div>

          {/* 状态提示：失败原因 / 未生效 / 未核实 */}
          {status === 'failed' && row.faultCode && (
            <Alert
              type="error"
              showIcon
              title={t('mml.consoleV2.detail.alert.dispatchFailed')}
              description={localizedFallback(row.faultCode, t)}
            />
          )}
          {status === 'mismatch' && (
            <Alert
              type="error"
              showIcon
              title={t('mml.consoleV2.detail.alert.mismatchTitle')}
              description={t('mml.consoleV2.detail.alert.mismatchDesc')}
            />
          )}
          {status === 'unverified' && (
            <Alert
              type="warning"
              showIcon
              title={t('mml.consoleV2.detail.alert.unverifiedTitle', { reason: row.unverifiedReason ? t(UNVERIFIED_REASON_TEXT[row.unverifiedReason]) : t('mml.consoleV2.detail.reasonUnknown') })}
              description={t('mml.consoleV2.detail.alert.unverifiedDesc')}
            />
          )}

          {/* 读后核实对比（写类，含 verify 时） */}
          {!read && row.verify && row.verify.length > 0 && (
            <div>
              <Text strong style={{ fontSize: 13 }}>
                {t('mml.consoleV2.detail.verifySection')}
              </Text>
              <div style={{ marginTop: 6 }}>
                <Table<VerifyItem>
                  size="small"
                  rowKey="path"
                  columns={verifyColumns}
                  dataSource={row.verify}
                  pagination={false}
                  scroll={{ y: 220 }}
                />
              </div>
            </div>
          )}

          {/* 读类：参数值解析 */}
          {read && (
            <div>
              <Text strong style={{ fontSize: 13 }}>
                {t('mml.consoleV2.detail.execResultSection')}
              </Text>
              <div style={{ marginTop: 6 }}>
                {status === 'success' ? (
                  <Table<ParsedParam>
                    size="small"
                    rowKey="key"
                    columns={paramColumns}
                    dataSource={parsedParams}
                    pagination={false}
                    scroll={{ y: 240 }}
                  />
                ) : (
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    {t('mml.consoleV2.detail.failedNoResult')}
                  </Text>
                )}
              </div>
            </div>
          )}

          {/* PATH 列表（#196：MOD 下发 + LST 回读前后对比） */}
          {row.pathTasks && row.pathTasks.length > 0 && (
            <div>
              <Text strong style={{ fontSize: 13 }}>
                {t('mml.consoleV2.detail.pathListSection')}
              </Text>
              <div style={{ marginTop: 6 }}>
                <Table<PathTask>
                  size="small"
                  rowKey={(t) => `${t.opType ?? ''}-${t.pathIndex}-${t.path}`}
                  columns={pathTaskColumns}
                  dataSource={row.pathTasks}
                  pagination={false}
                  scroll={{ y: 200 }}
                />
              </div>
            </div>
          )}
        </Space>
      )}
    </Modal>
  );
}
