import { useState } from 'react';
import { Alert, App, Button, Collapse, Empty, Spin, Tag } from 'antd';
import { CheckCircleOutlined, CopyOutlined, ExperimentOutlined, HistoryOutlined, LoadingOutlined } from '@ant-design/icons';
import { useIntl } from 'react-intl';
import type { AssistantCapability, AssistantRun } from '@core/types/assistant';
import { useT } from '@/hooks/useT';
import { dateText, errorText, isRunning } from './helpers';
import styles from './index.module.css';

interface Props { runs: AssistantRun[]; capabilities: AssistantCapability[]; busy: boolean; onCancel(id: string): Promise<void>; selectedRunId?: string }
export default function AssistantResults({ runs, capabilities, busy, onCancel, selectedRunId }: Props) {
  const t = useT(); const { locale } = useIntl(); const { message } = App.useApp();
  const [selected, setSelected] = useState<string>();
  const run = runs.find((item) => item.id === (selected ?? selectedRunId)) ?? runs[0];
  if (!run) return <div className={styles.emptyResults}><Empty image={<HistoryOutlined className={styles.emptyIcon} />} description={<><h3>{t('assistant.noResults')}</h3><p>{t('assistant.noResultsBody')}</p></>} /></div>;
  const title = (id: string) => capabilities.find((item) => item.operationId === id)?.title || id;
  const copy = async () => {
    if (!run.output) return;
    try { await navigator.clipboard.writeText([run.output.title, run.output.summary, ...run.output.facts.map((f) => f.text), ...run.output.nextSteps].join('\n\n')); message.success(t('assistant.copied')); }
    catch { message.error(t('common.copyFailed')); }
  };
  return <div className={styles.resultsLayout}>
    <div className={styles.historyList} aria-label={t('assistant.results')}>
      {runs.map((item) => <button type="button" key={item.id} onClick={() => setSelected(item.id)} className={`${styles.historyItem} ${run.id === item.id ? styles.selectedHistory : ''}`} aria-pressed={run.id === item.id}>
        <span><strong>{t(`assistant.kind.${item.kind}`)}</strong><Tag color={item.status === 'FAILED' ? 'error' : item.status === 'COMPLETED' ? 'success' : 'processing'}>{t(`assistant.run.${item.status}`)}</Tag></span>
        <small>{dateText(item.createdAt, locale)}</small>
        <span className={styles.historyTitle}>{item.output?.title || t(`assistant.run.${item.status}`)}</span>
      </button>)}
    </div>
    <section className={styles.resultPanel} aria-label={t('assistant.results')}>
      <header className={styles.resultHeader}><div><Tag>{t('assistant.runRevision', { revision: run.revision })}</Tag><span className={styles.muted}>{t('assistant.createdAt', { time: dateText(run.createdAt, locale) })}</span></div>{run.output && <Button icon={<CopyOutlined />} onClick={() => void copy()}>{t('assistant.copy')}</Button>}</header>
      {isRunning(run) && <div className={styles.runningPanel} role="status" aria-live="polite">
        <Spin indicator={<LoadingOutlined spin />} size="large" />
        <h3>{run.status === 'CANCELLING' ? t('assistant.run.CANCELLING') : t('assistant.running')}</h3>
        <p>{run.status === 'QUEUED' ? t('assistant.queued') : run.status === 'CANCELLING' ? t('assistant.cancelPending') : t('assistant.runningHint')}</p>
        <Button disabled={busy || run.status === 'CANCELLING'} onClick={() => void onCancel(run.id)}>{t('assistant.cancelRun')}</Button>
      </div>}
      {run.status === 'FAILED' && <Alert type="error" showIcon title={t('assistant.runFailed')} description={<><p>{t('assistant.runFailedHint')}</p>{run.errorCode && <code>{run.errorCode}</code>}</>} />}
      {run.errorCode === 'ASSISTANT_RESULT_ACCESS_REVOKED' && <Alert type="warning" showIcon title={errorText(run.errorCode, t)} />}
      {run.status === 'CANCELLED' && <Alert type="info" showIcon title={t('assistant.run.CANCELLED')} />}
      {run.output && <>
        <div className={styles.resultHero}>
          <Tag icon={run.output.outcome === 'no_change' ? <CheckCircleOutlined /> : <ExperimentOutlined />} color={run.output.outcome === 'no_change' ? 'success' : run.output.outcome === 'insufficient_data' ? 'warning' : 'processing'}>{t(`assistant.outcome.${run.output.outcome}`)}</Tag>
          <h2>{run.output.title}</h2><p>{run.output.summary}</p>
        </div>
        {run.output.facts.length > 0 && <section className={styles.resultSection}><h3>{t('assistant.facts')}</h3>{run.output.facts.map((fact, i) => <div key={i} className={styles.fact}><CheckCircleOutlined /><div><p>{fact.text}</p><small>{t('assistant.evidence', { sources: fact.evidenceRefs.map((ref) => title(ref.replace(/^tool:/, ''))).join(' / ') })}</small></div></div>)}</section>}
        {run.output.hypotheses.length > 0 && <section className={`${styles.resultSection} ${styles.hypotheses}`}><h3>{t('assistant.hypotheses')}</h3>{run.output.hypotheses.map((text, i) => <p key={i}>{text}</p>)}</section>}
        {run.output.nextSteps.length > 0 && <section className={styles.resultSection}><h3>{t('assistant.nextSteps')}</h3>{run.output.nextSteps.map((text, i) => <p key={i}>{i + 1}. {text}</p>)}</section>}
      </>}
      {run.tools.length > 0 && <Collapse ghost items={[{ key: 'queries', label: t('assistant.toolProgress'), children: <div className={styles.toolList}>{run.tools.map((tool) => <div key={tool.id}><span>{title(tool.operationId)}</span><code>{tool.status}</code></div>)}</div> }]} />}
      {run.completedAt && <p className={styles.muted}>{t('assistant.completedAt', { time: dateText(run.completedAt, locale) })}</p>}
    </section>
  </div>;
}
