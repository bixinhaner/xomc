import { Alert, Button, Descriptions, Modal, Space, Table, Tag, Typography } from 'antd';
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

  // 逐 PATH 子任务表（§3.11.3：父任务 = 设备任务 ID，每 path 一子任务）
  const pathTaskColumns: ColumnsType<PathTask> = [
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
      title: '子任务 ID',
      dataIndex: 'subTaskId',
      key: 'subTaskId',
      width: 200,
      render: (v: string) => (
        <Text code copyable={{ text: v }} style={{ fontSize: 11 }}>
          {v.slice(0, 8)}…
        </Text>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 110,
      render: (s: ExecStatus) => <Tag color={STATUS_META[s].color}>{STATUS_META[s].text}</Tag>,
    },
    { title: '下发', dataIndex: 'dispatchedAt', key: 'dispatchedAt', width: 88 },
    { title: '响应', dataIndex: 'respondedAt', key: 'respondedAt', width: 88 },
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
            下载该设备结果
          </Button>
          <Button onClick={onClose}>关闭</Button>
        </Space>
      }
    >
      {row && execMeta && (
        <Space direction="vertical" size={14} style={{ width: '100%' }}>
          {/* 任务信息 */}
          <div style={{ padding: 12, background: '#fafafa', border: '1px solid #f0f0f0', borderRadius: 6 }}>
            <Descriptions
              size="small"
              column={1}
              styles={{ label: { width: 96, color: '#595959' } }}
              items={[
                {
                  key: 'cmd',
                  label: '执行命令',
                  children: (
                    <Space size={6} wrap>
                      <Tag color={opColor(execMeta.operationType)} style={{ marginInlineEnd: 0 }}>
                        {execMeta.operationType} {opLabel(execMeta.operationType)}
                      </Tag>
                      {execMeta.commandName && <Text strong>{execMeta.commandName}</Text>}
                      <Text code style={{ fontSize: 12 }}>
                        {execMeta.label}
                      </Text>
                    </Space>
                  ),
                },
                {
                  key: 'commandId',
                  label: '命令 ID',
                  children: commandId ? (
                    <Text code copyable={{ text: commandId }} style={{ fontSize: 12 }}>
                      {commandId}
                    </Text>
                  ) : (
                    <Text type="secondary">-</Text>
                  ),
                },
                {
                  key: 'deviceTaskId',
                  label: '设备任务 ID',
                  children: (
                    <Text code copyable={{ text: row.deviceTaskId }} style={{ fontSize: 12 }}>
                      {row.deviceTaskId}
                    </Text>
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
                    <Space size={12}>
                      <Tag color={STATUS_META[row.status].color}>{STATUS_META[row.status].text}</Tag>
                      <Text type="secondary">用时 {(row.elapsedMs / 1000).toFixed(1)} s</Text>
                    </Space>
                  ),
                },
                {
                  key: 'time',
                  label: '下发 / 响应',
                  children: (
                    <Text type="secondary">
                      {row.dispatchedAt ?? '-'} → {row.respondedAt ?? '-'}
                    </Text>
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

          {/* 逐 PATH 子任务（§3.11.3：父任务 = 设备任务 ID） */}
          {row.pathTasks && row.pathTasks.length > 0 && (
            <div>
              <Text strong style={{ fontSize: 13 }}>
                逐 PATH 子任务（父任务 = 设备任务 ID）
              </Text>
              <div style={{ marginTop: 6 }}>
                <Table<PathTask>
                  size="small"
                  rowKey="subTaskId"
                  columns={pathTaskColumns}
                  dataSource={row.pathTasks}
                  pagination={false}
                  scroll={{ y: 200 }}
                />
              </div>
            </div>
          )}

          {/* 结果报文：格式化 XML */}
          <div>
            <Text strong style={{ fontSize: 13 }}>
              结果报文（格式化 XML）
            </Text>
            <div style={{ marginTop: 6 }}>
              <XmlViewer xml={row.raw} maxHeight={300} />
            </div>
          </div>
        </Space>
      )}
    </Modal>
  );
}
