import { useEffect } from 'react';
import { Alert, Button, Drawer, Empty, Grid, Progress, Skeleton, Space, Tag } from 'antd';
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  ExperimentOutlined,
  LinkOutlined,
  RobotOutlined,
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import {
  useAgentFinding,
  useContinueAgentFinding,
  useDismissAgentFinding,
  useMarkAgentFindingRead,
} from '@core/hooks/api/useAgentFindings';
import type { AgentFindingResourceRef } from '@core/types/agentFinding';
import { formatSystemTime } from '@core/utils/systemTime';
import { useT } from '@/hooks/useT';
import styles from './AgentFindingDrawer.module.css';

interface Props {
  findingId?: string;
  open: boolean;
  onOpenChange(open: boolean): void;
}

const severityColor: Record<string, string> = {
  critical: 'red', high: 'volcano', medium: 'orange', low: 'blue', info: 'default',
};

function resourceRoute(resource: AgentFindingResourceRef): string | undefined {
  if (resource.type === 'device' && resource.label) {
    return `/device/detail/${encodeURIComponent(resource.label)}`;
  }
  if (resource.type === 'alarm') {
    return `/alarm/current?alarmId=${encodeURIComponent(resource.id)}`;
  }
  return undefined;
}

export default function AgentFindingDrawer({ findingId, open, onOpenChange }: Props) {
  const t = useT();
  const navigate = useNavigate();
  const screens = Grid.useBreakpoint();
  const query = useAgentFinding(findingId, open);
  const read = useMarkAgentFindingRead();
  const dismiss = useDismissAgentFinding();
  const continuation = useContinueAgentFinding();
  const finding = query.data;

  useEffect(() => {
    if (!open || !findingId || !finding || finding.read || read.isPending) return;
    read.mutate(findingId);
  }, [finding, findingId, open, read]);

  const continueAgent = async () => {
    if (!findingId) return;
    const launch = await continuation.mutateAsync(findingId);
    window.dispatchEvent(new CustomEvent('xomc:open-agent-finding', { detail: launch }));
    onOpenChange(false);
  };

  const dismissFinding = async () => {
    if (!findingId) return;
    await dismiss.mutateAsync(findingId);
    onOpenChange(false);
  };

  return (
    <Drawer
      title={<Space><RobotOutlined /><span>{t('agentFinding.title')}</span></Space>}
      open={open}
      onClose={() => onOpenChange(false)}
      size={screens.md ? 640 : '100%'}
      className={styles.drawer}
      extra={finding && <Tag color={severityColor[finding.severity]}>{t(`agentFinding.severity.${finding.severity}`)}</Tag>}
      footer={finding ? (
        <div className={styles.footer}>
          <Button icon={<CloseCircleOutlined />} loading={dismiss.isPending} onClick={() => void dismissFinding()}>
            {t('agentFinding.dismiss')}
          </Button>
          <Button type="primary" icon={<RobotOutlined />} loading={continuation.isPending} onClick={() => void continueAgent()}>
            {t('agentFinding.continue')}
          </Button>
        </div>
      ) : null}
    >
      {query.isPending && <Skeleton active paragraph={{ rows: 10 }} />}
      {query.isError && <Alert type="error" showIcon title={t('agentFinding.loadFailed')} action={<Button size="small" onClick={() => void query.refetch()}>{t('common.retry')}</Button>} />}
      {!query.isPending && !query.isError && !finding && <Empty description={t('agentFinding.notFound')} />}
      {finding && (
        <div className={styles.content}>
          <section className={styles.hero}>
            <div className={styles.eyebrow}>{t('agentFinding.generatedAt', { time: formatSystemTime(finding.createdAt) })}</div>
            <h2>{finding.title}</h2>
            <p>{finding.summary}</p>
            <div className={styles.confidence}>
              <span>{t('agentFinding.confidence')}</span>
              <Progress percent={Math.round(finding.confidence * 100)} size="small" status="normal" />
            </div>
          </section>

          {finding.resources.length > 0 && (
            <section className={styles.section}>
              <h3><LinkOutlined /> {t('agentFinding.resources')}</h3>
              <div className={styles.resourceList}>
                {finding.resources.map((resource) => {
                  const route = resourceRoute(resource);
                  return <Button key={`${resource.type}:${resource.id}`} disabled={!route} onClick={() => route && navigate(route)}>{resource.label || resource.id}</Button>;
                })}
              </div>
            </section>
          )}

          <section className={styles.section}>
            <h3><CheckCircleOutlined /> {t('agentFinding.facts')}</h3>
            {finding.facts.length ? <ol className={styles.statementList}>{finding.facts.map((fact) => <li key={fact.id}><span>{fact.text}</span><small>{t('agentFinding.evidenceCount', { count: fact.evidenceRefs.length })}</small></li>)}</ol> : <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('agentFinding.noFacts')} />}
          </section>

          {finding.hypotheses.length > 0 && (
            <section className={`${styles.section} ${styles.hypotheses}`}>
              <h3><ExperimentOutlined /> {t('agentFinding.hypotheses')}</h3>
              <ol className={styles.statementList}>{finding.hypotheses.map((hypothesis) => <li key={hypothesis.id}><span>{hypothesis.text}</span><small>{t('agentFinding.hypothesisConfidence', { value: Math.round(hypothesis.confidence * 100) })}</small></li>)}</ol>
            </section>
          )}

          {Object.keys(finding.details).length > 0 && (
            <section className={styles.section}>
              <h3>{t('agentFinding.details')}</h3>
              <dl className={styles.details}>{Object.entries(finding.details).map(([key, value]) => <div key={key}><dt>{key}</dt><dd>{typeof value === 'object' ? JSON.stringify(value) : String(value)}</dd></div>)}</dl>
            </section>
          )}
        </div>
      )}
    </Drawer>
  );
}
