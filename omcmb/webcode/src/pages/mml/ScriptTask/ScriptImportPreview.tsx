import { useMemo, useState } from 'react';
import { Button, Card, Col, Row, Space, Table, Tag, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import type { MMLScriptImportValidation, MMLScriptIssue, MMLTaskPlanItem } from '@core/types/mml';
import { useT } from '@/hooks/useT';
import MmlCommandDisplay from '../components/MmlCommandDisplay';

export interface ScriptImportPreviewProps {
  validation?: MMLScriptImportValidation | null;
  readOnly?: boolean;
}

type IssueFilter = 'all' | 'error' | 'warning';

function issueForLine(issues: MMLScriptIssue[], lineNo: number): MMLScriptIssue[] {
  return issues.filter((issue) => issue.lineNo === lineNo);
}

function issueDescription(issue: MMLScriptIssue, t: ReturnType<typeof useT>): string {
  return issue.displayMessage || issue.message || t('mml.scriptImport.issueFallback');
}

function issueLineText(issue: MMLScriptIssue, t: ReturnType<typeof useT>): string {
  const prefix = issue.lineNo ? t('mml.scriptImport.linePrefix', { line: issue.lineNo }) : '';
  return `${prefix}${issueDescription(issue, t)}`;
}

function stripDeviceSuffix(rawLine: string | undefined, deviceSn: string): string {
  const line = (rawLine ?? '').trim();
  if (!line) return '';
  const separator = line.lastIndexOf(';');
  if (separator >= 0) {
    const commandPart = line.slice(0, separator).trim();
    const devicePart = line.slice(separator + 1).trim();
    const devices = devicePart.split(',').map((sn) => sn.trim()).filter(Boolean);
    if (!deviceSn || devices.includes(deviceSn.trim())) {
      return commandPart;
    }
  }
  return line.replace(/;+$/, '').trim();
}

function normalizeOperation(value: unknown): string {
  return typeof value === 'string' ? value.trim().toUpperCase() : '';
}

function formatParamValue(value: unknown): string {
  if (value === null || value === undefined) return '';
  return String(value);
}

function formatParameters(parameters: Record<string, unknown> | undefined): string {
  if (!parameters) return '';
  return Object.keys(parameters)
    .filter((key) => key.trim())
    .sort()
    .map((key) => `${key}=${formatParamValue(parameters[key])}`)
    .join(',');
}

function planItemCommandText(row: MMLTaskPlanItem): string {
  const fromRawLine = stripDeviceSuffix(row.rawLine, row.deviceSn);
  if (fromRawLine) return fromRawLine;

  const code = row.command.commandCode || '';
  const operation = normalizeOperation(row.command.operationType) || (code.startsWith('RAW ') ? code.slice(4).trim() : '');
  const target = row.command.paramPaths?.length ? row.command.paramPaths.join(',') : '';
  const params = formatParameters(row.command.parameters);
  if (code.startsWith('RAW ') && operation) {
    return `${operation}${target ? ` ${target}` : ''}${params ? `:${params}` : ''}`.trim();
  }
  return `${code}${params ? `:${params}` : ''}`.trim();
}

/** Server-authoritative TXT summary and read-only plan preview. */
export default function ScriptImportPreview({ validation, readOnly = true }: ScriptImportPreviewProps) {
  const t = useT();
  const [filter, setFilter] = useState<IssueFilter>('all');
  const summary = validation?.summary;
  const issues = validation?.issues ?? [];
  const visibleIssues = filter === 'all' ? issues : issues.filter((issue) => issue.severity === filter);
  const rows = useMemo<MMLTaskPlanItem[]>(() => {
    const plans = validation?.planItems ?? [];
    if (filter === 'all') return plans;
    const lineNumbers = new Set(issues.filter((issue) => issue.severity === filter).map((issue) => issue.lineNo));
    return plans.filter((plan) => lineNumbers.has(plan.lineNo));
  }, [filter, issues, validation?.planItems]);

  const downloadReport = () => {
    const report = visibleIssues.map((issue) => issueLineText(issue, t)).join('\n');
    const url = URL.createObjectURL(new Blob([report], { type: 'text/plain;charset=utf-8' }));
    const anchor = document.createElement('a');
    anchor.href = url;
    anchor.download = 'mml-script-errors.txt';
    anchor.click();
    URL.revokeObjectURL(url);
  };

  if (!validation) return null;

  const columns: ColumnsType<MMLTaskPlanItem> = [
    { title: t('mml.scriptImport.lineNo'), dataIndex: 'lineNo', width: 72 },
    {
      title: t('mml.scriptImport.deviceSn'),
      dataIndex: 'deviceSn',
      width: 240,
      onCell: () => ({ style: { minWidth: 240, whiteSpace: 'nowrap' } }),
    },
    { title: t('mml.scriptImport.order'), dataIndex: 'order', width: 72 },
    {
      title: t('mml.scriptImport.command'),
      key: 'command',
      width: 360,
      render: (_, row) => {
        const command = planItemCommandText(row);
        return command ? <MmlCommandDisplay command={command} maxTargetWidth={260} /> : '-';
      },
    },
    {
      title: t('mml.scriptImport.validationResult'), key: 'issues', render: (_, row) => {
        const lineIssues = issueForLine(issues, row.lineNo);
        if (!lineIssues.length) return <Tag color="success">{t('mml.scriptImport.passed')}</Tag>;
        return <Space wrap>{lineIssues.map((issue, index) => <Tag key={`${issue.code}-${index}`} color={issue.severity === 'error' ? 'error' : 'warning'}>{issueDescription(issue, t)}</Tag>)}</Space>;
      },
    },
  ];

  return (
    <Space direction="vertical" size={14} style={{ width: '100%' }}>
      <Typography.Text type="secondary">{readOnly ? t('mml.scriptImport.readOnlyContent') : ''}</Typography.Text>
      {validation.originalFilename ? <Typography.Text>{t('mml.scriptImport.fileLabel')}：<span>{validation.originalFilename}</span></Typography.Text> : null}
      <Card size="small" title={t('mml.scriptImport.validationSummary')}>
        <Row gutter={12}>
          <Col span={6}><Typography.Text>{t('mml.scriptImport.validLines')}</Typography.Text><div><Typography.Title level={4}>{summary?.validLines ?? 0}</Typography.Title></div></Col>
          <Col span={6}><Typography.Text>{t('mml.scriptImport.deviceCount')}</Typography.Text><div><Typography.Title level={4}>{summary?.deviceCount ?? 0}</Typography.Title></div></Col>
          <Col span={6}><Typography.Text>{t('mml.scriptImport.errorCount')}</Typography.Text><div><Typography.Title level={4} type="danger">{summary?.errorCount ?? 0}</Typography.Title></div></Col>
          <Col span={6}><Typography.Text>{t('mml.scriptImport.warningCount')}</Typography.Text><div><Typography.Title level={4} type="warning">{summary?.warningCount ?? 0}</Typography.Title></div></Col>
        </Row>
      </Card>
      <Space wrap>
        <Button type={filter === 'all' ? 'primary' : 'default'} onClick={() => setFilter('all')}>{t('mml.scriptImport.all')}</Button>
        <Button type={filter === 'error' ? 'primary' : 'default'} onClick={() => setFilter('error')}>{t('mml.scriptImport.onlyErrors')}</Button>
        <Button type={filter === 'warning' ? 'primary' : 'default'} onClick={() => setFilter('warning')}>{t('mml.scriptImport.onlyWarnings')}</Button>
        <Button onClick={downloadReport} disabled={!visibleIssues.length}>{t('mml.scriptImport.downloadErrorReport')}</Button>
      </Space>
      {visibleIssues.length ? (
        <Card size="small" title={t('mml.scriptImport.issues')}>
          <Space direction="vertical" style={{ width: '100%' }}>
            {visibleIssues.map((issue, index) => (
              <Typography.Text key={`${issue.code}-${issue.lineNo ?? 'x'}-${index}`} type={issue.severity === 'error' ? 'danger' : 'warning'}>
                {issueLineText(issue, t)}
              </Typography.Text>
            ))}
          </Space>
        </Card>
      ) : null}
      <Table<MMLTaskPlanItem> rowKey={(row) => `${row.lineNo}-${row.deviceSn}-${row.order}`} columns={columns} dataSource={rows} size="small" pagination={{ pageSize: 20 }} scroll={{ x: 840 }} />
    </Space>
  );
}
