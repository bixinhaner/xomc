import { Alert, Button, Descriptions, Modal, Popover, Space, Table, Tabs, Tag, Typography } from 'antd';
import { DownloadOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import XmlViewer from '@/components/XmlViewer';
import type { ExecMeta, ExecStatus, PathTask, ResultColumn, ResultRow, VerifyItem } from '../types';
import { STATUS_META, UNVERIFIED_REASON_TEXT, opColor, opLabel } from '../constants';
import { exportOne } from '../download';

const { Text } = Typography;

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
  const read = execMeta?.read ?? true;
  const status = row?.status;

  const parsedParams: ParsedParam[] = row
    ? columns.map((c) => ({
        key: c.key,
        path: c.path,
        label: c.label,
        value: row.cells[c.path] ?? '',
      }))
    : [];

  // #196：MOD 下发值（path→value）取自 PATH 列表的 MOD 行，供「执行 PATH」弹层显示下发值。
  const modSetValues = new Map<string, string>(
    (row?.pathTasks ?? [])
      .filter((t) => t.opType === 'MOD' && t.value)
      .map((t) => [t.path, t.value]),
  );

  const paramColumns: ColumnsType<ParsedParam> = [
    {
      title: '参数路径',
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
      title: '值',
      dataIndex: 'value',
      key: 'value',
      width: 220,
      render: (v: string) =>
        v ? (
          <Text style={{ fontFamily: 'monospace', fontSize: 12, wordBreak: 'break-all' }}>{v}</Text>
        ) : (
          <Text type="secondary">(空)</Text>
        ),
    },
  ];

  // 读后核实对比表（写类，§3.11.2）
  const verifyColumns: ColumnsType<VerifyItem> = [
    {
      title: '参数',
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
      title: '预期下发值',
      dataIndex: 'expected',
      key: 'expected',
      width: 150,
      render: (v: string) => (
        <Text style={{ fontFamily: 'monospace', fontSize: 12 }}>{v || '-'}</Text>
      ),
    },
    {
      title: '读回值',
      dataIndex: 'actual',
      key: 'actual',
      width: 150,
      render: (v: string, r) =>
        v ? (
          <Text type={r.matched ? undefined : 'danger'} style={{ fontFamily: 'monospace', fontSize: 12 }}>
            {v}
          </Text>
        ) : (
          <Text type="secondary">(未读回)</Text>
        ),
    },
    {
      title: '核实',
      dataIndex: 'matched',
      key: 'matched',
      width: 90,
      render: (m: boolean) => (m ? <Tag color="success">一致</Tag> : <Tag color="error">不一致</Tag>),
    },
  ];

  // PATH 列表（#196：MOD 下发 + LST 回读前后对比）。列序：类型 | 子任务 ID | 状态 | PATH | PATH 值。
  const pathTaskColumns: ColumnsType<PathTask> = [
    {
      title: '类型',
      dataIndex: 'opType',
      key: 'opType',
      width: 96,
      render: (v?: string) =>
        v === 'MOD' ? (
          <Tag color="blue">MOD 下发</Tag>
        ) : v === 'LST' ? (
          <Tag color="green">LST 回读</Tag>
        ) : (
          <Text type="secondary">-</Text>
        ),
    },
    {
      title: '子任务 ID',
      dataIndex: 'subTaskId',
      key: 'subTaskId',
      width: 88,
      align: 'center',
      // #196：不直接显示冗长 ID，仅提供「复制」按钮（点击复制完整 device_task id）；无 ID 时占位 -。
      render: (v: string) =>
        v ? (
          <Text copyable={{ text: v, tooltips: ['复制子任务 ID', '已复制'] }} style={{ fontSize: 12 }}>
            复制
          </Text>
        ) : (
          <Text type="secondary" style={{ fontSize: 11 }}>
            -
          </Text>
        ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 80,
      render: (s: ExecStatus) => <Tag color={STATUS_META[s].color}>{STATUS_META[s].text}</Tag>,
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
      title: 'PATH 值',
      dataIndex: 'value',
      key: 'value',
      width: 220,
      render: (v: string) =>
        v ? (
          <Text
            copyable={{ text: v, tooltips: ['复制', '已复制'] }}
            ellipsis={{ tooltip: v }}
            style={{ fontSize: 11, maxWidth: 180 }}
          >
            {v}
          </Text>
        ) : (
          <Text type="secondary">-</Text>
        ),
    },
  ];

  return (
    <Modal
      title={row ? `执行详情 - ${row.deviceSn}` : '执行详情'}
      open={open}
      onCancel={onClose}
      width={780}
      destroyOnHidden
      footer={
        <Space>
          <Button
            icon={<DownloadOutlined />}
            disabled={!row}
            onClick={() => row && exportOne(columns, row)}
          >
            下载执行结果
          </Button>
          <Button onClick={onClose}>关闭</Button>
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
                  label: '任务 ID',
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
                  label: '执行命令',
                  children: (
                    <Space size={6} wrap>
                      <Tag color={opColor(execMeta.operationType)} style={{ marginInlineEnd: 0 }}>
                        {execMeta.operationType} {opLabel(execMeta.operationType)}
                      </Tag>
                      <Text strong>{execMeta.commandName ?? execMeta.label}</Text>
                      {columns.length > 0 && (
                        <Popover
                          trigger="hover"
                          placement="bottomLeft"
                          title={`执行 PATH（${columns.length}）`}
                          content={
                            <div style={{ maxHeight: 320, overflow: 'auto', maxWidth: 460 }}>
                              {columns.map((c) => {
                                // #196：MOD 显示该 path 的下发值（取自 PATH 列表 MOD 行）
                                const setVal = modSetValues.get(c.path);
                                return (
                                  <div key={c.key} style={{ marginBottom: 6, lineHeight: 1.4 }}>
                                    <Text style={{ fontSize: 12 }}>{c.label}</Text>
                                    <br />
                                    <Text code style={{ fontSize: 11, wordBreak: 'break-all' }}>
                                      {c.path}
                                    </Text>
                                    {setVal != null && setVal !== '' && (
                                      <>
                                        <br />
                                        <Text type="secondary" style={{ fontSize: 11 }}>
                                          下发值：
                                        </Text>
                                        <Text style={{ fontFamily: 'monospace', fontSize: 11, wordBreak: 'break-all' }}>
                                          {setVal}
                                        </Text>
                                      </>
                                    )}
                                  </div>
                                );
                              })}
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
                  label: '设备 SN',
                  children: <Text>{row.deviceSn}</Text>,
                },
                {
                  key: 'status',
                  label: '执行状态',
                  children: (
                    <Space size={12} wrap>
                      <Tag color={STATUS_META[row.status].color}>{STATUS_META[row.status].text}</Tag>
                      <Text type="secondary">用时 {(row.elapsedMs / 1000).toFixed(1)} s</Text>
                      <Text type="secondary">
                        下发 / 响应：{row.dispatchedAt ?? '-'} → {row.respondedAt ?? '-'}
                      </Text>
                    </Space>
                  ),
                },
              ]}
            />
          </div>

          {/* 状态提示：失败原因 / 未生效 / 未核实 */}
          {status === 'failed' && row.faultCode && (
            <Alert type="error" showIcon message="下发失败" description={row.faultCode} />
          )}
          {status === 'mismatch' && (
            <Alert
              type="error"
              showIcon
              message="核实不一致：参数未生效"
              description="写 RPC 响应成功，但读回值与预期下发值不符——基站可能未真正应用该变更。"
            />
          )}
          {status === 'unverified' && (
            <Alert
              type="warning"
              showIcon
              message={`已下发·未核实（${row.unverifiedReason ? UNVERIFIED_REASON_TEXT[row.unverifiedReason] : '原因未知'}）`}
              description="写 RPC 响应成功，但该参数未做读回核实，不计为失败；请知悉「未核实 ≠ 已确认生效」。"
            />
          )}

          {/* 读后核实对比（写类，含 verify 时） */}
          {!read && row.verify && row.verify.length > 0 && (
            <div>
              <Text strong style={{ fontSize: 13 }}>
                读后核实对比（预期 vs 读回）
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
                执行结果
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
                    执行失败，无可解析结果。
                  </Text>
                )}
              </div>
            </div>
          )}

          {/* PATH 列表（#196：MOD 下发 + LST 回读前后对比） */}
          {row.pathTasks && row.pathTasks.length > 0 && (
            <div>
              <Text strong style={{ fontSize: 13 }}>
                PATH 列表
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

          {/* 结果报文：格式化 XML。#196：MOD 回读复合时分「MOD 响应 / 回读 LST 响应」两个页签。 */}
          <div>
            <Text strong style={{ fontSize: 13 }}>
              结果报文（格式化 XML）
            </Text>
            <div style={{ marginTop: 6 }}>
              {row.readbackRaw ? (
                <Tabs
                  size="small"
                  items={[
                    { key: 'mod', label: 'MOD 响应', children: <XmlViewer xml={row.raw} maxHeight={280} /> },
                    {
                      key: 'lst',
                      label: '回读 LST 响应',
                      children: <XmlViewer xml={row.readbackRaw} maxHeight={280} />,
                    },
                  ]}
                />
              ) : (
                <XmlViewer xml={row.raw} maxHeight={300} />
              )}
            </div>
          </div>
        </Space>
      )}
    </Modal>
  );
}
