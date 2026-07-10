import { useMemo, useState } from 'react';
import { Button, Card, Col, Row, Space, Table, Tag, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import type { MMLScriptImportValidation, MMLScriptIssue, MMLTaskPlanItem } from '@core/types/mml';

export interface ScriptImportPreviewProps {
  validation?: MMLScriptImportValidation | null;
  readOnly?: boolean;
}

type IssueFilter = 'all' | 'error' | 'warning';

function issueForLine(issues: MMLScriptIssue[], lineNo: number): MMLScriptIssue[] {
  return issues.filter((issue) => issue.lineNo === lineNo);
}

/** Server-authoritative TXT summary and read-only plan preview. */
export default function ScriptImportPreview({ validation, readOnly = true }: ScriptImportPreviewProps) {
  const [filter, setFilter] = useState<IssueFilter>('all');
  const summary = validation?.summary;
  const issues = validation?.issues ?? [];
  const rows = useMemo<MMLTaskPlanItem[]>(() => {
    const plans = validation?.planItems ?? [];
    if (filter === 'all') return plans;
    const lineNumbers = new Set(issues.filter((issue) => issue.severity === filter).map((issue) => issue.lineNo));
    return plans.filter((plan) => lineNumbers.has(plan.lineNo));
  }, [filter, issues, validation?.planItems]);

  const downloadReport = () => {
    const report = issues.map((issue) => `${issue.lineNo ?? '-'}\t${issue.severity}\t${issue.code}\t${issue.message ?? ''}`).join('\n');
    const url = URL.createObjectURL(new Blob([report], { type: 'text/plain;charset=utf-8' }));
    const anchor = document.createElement('a');
    anchor.href = url;
    anchor.download = 'mml-script-errors.txt';
    anchor.click();
    URL.revokeObjectURL(url);
  };

  if (!validation) return null;

  const columns: ColumnsType<MMLTaskPlanItem> = [
    { title: '行号', dataIndex: 'lineNo', width: 72 },
    { title: '设备 SN', dataIndex: 'deviceSn', width: 150, ellipsis: true },
    { title: '顺序', dataIndex: 'order', width: 72 },
    { title: '命令', key: 'command', render: (_, row) => <Typography.Text code>{row.command.commandCode}</Typography.Text> },
    { title: '参数摘要', key: 'parameters', render: (_, row) => row.command.parameters ? JSON.stringify(row.command.parameters) : '-' },
    {
      title: '校验结果', key: 'issues', render: (_, row) => {
        const lineIssues = issueForLine(issues, row.lineNo);
        if (!lineIssues.length) return <Tag color="success">通过</Tag>;
        return <Space wrap>{lineIssues.map((issue, index) => <Tag key={`${issue.code}-${index}`} color={issue.severity === 'error' ? 'error' : 'warning'}>{issue.code}</Tag>)}</Space>;
      },
    },
  ];

  return (
    <Space direction="vertical" size={14} style={{ width: '100%' }}>
      <Typography.Text type="secondary">{readOnly ? 'TXT 内容由导入文件生成，只读。' : ''}</Typography.Text>
      {validation.originalFilename ? <Typography.Text>文件：<span>{validation.originalFilename}</span></Typography.Text> : null}
      <Card size="small" title="校验摘要">
        <Row gutter={12}>
          <Col span={6}><Typography.Text>有效命令行</Typography.Text><div><Typography.Title level={4}>{summary?.validLines ?? 0}</Typography.Title></div></Col>
          <Col span={6}><Typography.Text>设备数</Typography.Text><div><Typography.Title level={4}>{summary?.deviceCount ?? 0}</Typography.Title></div></Col>
          <Col span={6}><Typography.Text>错误数</Typography.Text><div><Typography.Title level={4} type="danger">{summary?.errorCount ?? 0}</Typography.Title></div></Col>
          <Col span={6}><Typography.Text>警告数</Typography.Text><div><Typography.Title level={4} type="warning">{summary?.warningCount ?? 0}</Typography.Title></div></Col>
        </Row>
      </Card>
      <Space wrap>
        <Button type={filter === 'all' ? 'primary' : 'default'} onClick={() => setFilter('all')}>全部</Button>
        <Button type={filter === 'error' ? 'primary' : 'default'} onClick={() => setFilter('error')}>仅看错误</Button>
        <Button type={filter === 'warning' ? 'primary' : 'default'} onClick={() => setFilter('warning')}>仅看警告</Button>
        <Button onClick={downloadReport} disabled={!issues.length}>下载错误报告</Button>
      </Space>
      {issues.length ? (
        <Card size="small" title="逐行问题">
          <Space direction="vertical" style={{ width: '100%' }}>
            {issues.map((issue, index) => (
              <Typography.Text key={`${issue.code}-${issue.lineNo ?? 'x'}-${index}`} type={issue.severity === 'error' ? 'danger' : 'warning'}>
                {issue.lineNo ? `第 ${issue.lineNo} 行：` : ''}{issue.code}{issue.message ? ` — ${issue.message}` : ''}
              </Typography.Text>
            ))}
          </Space>
        </Card>
      ) : null}
      <Table<MMLTaskPlanItem> rowKey={(row) => `${row.lineNo}-${row.deviceSn}-${row.order}`} columns={columns} dataSource={rows} size="small" pagination={{ pageSize: 20 }} scroll={{ x: 780 }} />
    </Space>
  );
}
