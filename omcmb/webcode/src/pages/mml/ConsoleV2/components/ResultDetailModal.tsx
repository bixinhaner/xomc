import { Alert, Button, Descriptions, Modal, Space, Table, Tag, Typography } from 'antd';
import { DownloadOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import XmlViewer from '@/components/XmlViewer';
import type { ExecMeta, ResultColumn, ResultRow } from '../types';
import { STATUS_META, opColor, opLabel } from '../constants';
import { exportOne } from '../download';

const { Text } = Typography;

interface ResultDetailModalProps {
  open: boolean;
  row: ResultRow | null;
  execMeta: ExecMeta | null;
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
 * 单设备执行详情（设计 §3.4「失败行可展开看原始报文」+ 用户需求：参考任务记录「查看」）。
 *
 * 复刻 V1 任务记录 TaskRecord 的 per-device 详情：任务信息（命令/操作类型/涉及路径）+
 * 执行状态 + 失败原因 + 执行结果解析（读=参数值表，写=状态行）+ 原始报文。当前为 mock 数据。
 */
export default function ResultDetailModal({
  open,
  row,
  execMeta,
  columns,
  onClose,
}: ResultDetailModalProps) {
  const read = execMeta?.read ?? true;
  const success = row?.status === 'success';

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

  return (
    <Modal
      title={row ? `执行详情 - ${row.deviceSn}` : '执行详情'}
      open={open}
      onCancel={onClose}
      width={760}
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
              styles={{ label: { width: 90, color: '#595959' } }}
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
                      <Text code style={{ fontSize: 12 }}>{execMeta.label}</Text>
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
                    <Space size={12}>
                      <Tag color={STATUS_META[row.status].color}>{STATUS_META[row.status].text}</Tag>
                      <Text type="secondary">用时 {(row.elapsedMs / 1000).toFixed(1)} s</Text>
                    </Space>
                  ),
                },
                {
                  key: 'paths',
                  label: '涉及路径',
                  children: <Text type="secondary">{columns.length} 项</Text>,
                },
              ]}
            />
          </div>

          {/* 失败原因 */}
          {!success && row.faultCode && (
            <Alert type="error" showIcon message="失败原因" description={row.faultCode} />
          )}

          {/* 执行结果解析：每个 path 的指定结果（读类）/ 下发状态（写类） */}
          <div>
            <Text strong style={{ fontSize: 13 }}>
              执行结果
            </Text>
            <div style={{ marginTop: 6 }}>
              {!success ? (
                <Text type="secondary" style={{ fontSize: 12 }}>
                  执行失败，无可解析结果。
                </Text>
              ) : read ? (
                <Table<ParsedParam>
                  size="small"
                  rowKey="key"
                  columns={paramColumns}
                  dataSource={parsedParams}
                  pagination={false}
                  scroll={{ y: 240 }}
                />
              ) : (
                <Tag color="success">下发成功 · 立即生效</Tag>
              )}
            </div>
          </div>

          {/* 结果报文：直接结果的 XML，格式化后显示 */}
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
