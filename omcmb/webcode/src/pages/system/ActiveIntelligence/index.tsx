import { useCallback, useEffect, useMemo, useState } from 'react';
import { App, Button, Drawer, Empty, Input, Modal, Spin, Tag, Typography } from 'antd';
import {
  BellOutlined, CalendarOutlined, CheckCircleOutlined, DeploymentUnitOutlined,
  EyeOutlined, QuestionCircleOutlined, RobotOutlined, SafetyCertificateOutlined,
  SearchOutlined, SendOutlined, SettingOutlined, UserOutlined, WifiOutlined,
} from '@ant-design/icons';
import { proactiveAgentApi } from '@core/services/api/proactiveAgentApi';
import type { ProactiveOverview, ProactiveScenario } from '@core/types/proactiveAgent';
import { useT } from '@/hooks/useT';
import styles from './index.module.css';

type BuilderStep = 'severity' | 'delivery' | 'ready';

const fallbackScenario = (key: string, eventType: string, tools: string[], surfaces: string[], status: 'ACTIVE' | 'DISABLED' = 'ACTIVE'): ProactiveScenario => ({
  key, name: key, description: key, status, rolloutMode: 'SHADOW', rolloutPercentage: status === 'ACTIVE' ? 100 : 0,
  version: 1, eventTypes: [eventType], allowedOperations: tools, deliverySurfaces: surfaces,
  dedupeWindowSeconds: key === 'daily-operations-summary' ? 82800 : 1800,
  rateLimitPerHour: key === 'daily-operations-summary' ? 24 : 200, timeoutSeconds: 120,
  stats: { matched: 0, successRate: 0, avgDurationSeconds: 0 },
});

const disconnectedOverview: ProactiveOverview = {
  scenarios: [
    fallbackScenario('task-failure-analysis', 'omc.task.failed.v1', ['get.devices.by_id'], ['attention']),
    fallbackScenario('access-review-assistant', 'omc.device.access-review-required.v1', ['get.device_access.candidates'], ['attention']),
    fallbackScenario('severe-alarm-explanation', 'omc.alarm.severe-raised.v1', ['get.alarms.active'], ['attention']),
    fallbackScenario('daily-operations-summary', 'omc.daily-operations-summary-requested.v1', ['get.devices'], ['dashboard'], 'DISABLED'),
  ],
  packages: [], runs: [], connectorHealth: { status: 'UNKNOWN', queueDepth: 0 }, stats: {},
};

export default function ActiveIntelligence() {
  const t = useT();
  const { message } = App.useApp();
  const [data, setData] = useState<ProactiveOverview>();
  const [loading, setLoading] = useState(true);
  const [connectionError, setConnectionError] = useState('');
  const [builderOpen, setBuilderOpen] = useState(true);
  const [professionalOpen, setProfessionalOpen] = useState(false);
  const [testOpen, setTestOpen] = useState(false);
  const [step, setStep] = useState<BuilderStep>('severity');
  const [request, setRequest] = useState(() => t('activeIntelligence.beginner.defaultRequest'));
  const [severity, setSeverity] = useState<'both' | 'critical' | 'unsure'>();
  const [privateOnly, setPrivateOnly] = useState(true);
  const [saving, setSaving] = useState(false);

  const load = useCallback(async () => {
    setLoading(true); setConnectionError('');
    try { setData(await proactiveAgentApi.overview()); }
    catch (error) {
      setConnectionError(error instanceof Error ? error.message : t('activeIntelligence.loadFailed'));
      setData((current) => current ?? disconnectedOverview);
    } finally { setLoading(false); }
  }, [t]);
  useEffect(() => { void load(); }, [load]);

  const selectedScenario = useMemo(() => data?.scenarios.find((item) => item.key === 'severe-alarm-explanation'), [data]);
  const severityLabel = severity === 'critical' ? t('activeIntelligence.beginner.criticalOnly') : t('activeIntelligence.beginner.severeAndImportant');

  const resetBuilder = () => {
    setRequest(t('activeIntelligence.beginner.defaultRequest')); setSeverity(undefined); setPrivateOnly(true); setStep('severity'); setBuilderOpen(true);
  };
  const chooseSeverity = (value: 'both' | 'critical' | 'unsure') => { setSeverity(value); setStep('delivery'); };
  const finishDelivery = (value: boolean) => { setPrivateOnly(value); setStep('ready'); };
  const startTrial = async () => {
    if (!selectedScenario || connectionError) return;
    setSaving(true);
    try {
      await proactiveAgentApi.updateScenario(selectedScenario.key, { status: 'ACTIVE', rolloutMode: 'SHADOW', rolloutPercentage: 100 });
      message.success(t('activeIntelligence.beginner.started')); setTestOpen(false); setBuilderOpen(false); await load();
    } catch (error) { message.error(error instanceof Error ? error.message : t('activeIntelligence.saveFailed')); }
    finally { setSaving(false); }
  };

  if (loading && !data) return <div className={styles.center}><Spin /></div>;

  return <div className={styles.page}>
    <div className={styles.breadcrumb}>
      <span>{t('nav.system')}</span><b>/</b><span>{t('activeIntelligence.title')}</span>
      {builderOpen && <><b>/</b><strong>{t('activeIntelligence.beginner.create')}</strong></>}
    </div>

    {builderOpen ? <section className={styles.builder}>
      <header className={styles.builderHeader}>
        <div><Typography.Title level={2}>{t('activeIntelligence.beginner.create')}</Typography.Title><Typography.Text>{t('activeIntelligence.beginner.createSubtitle')}</Typography.Text></div>
        <Button type="text" onClick={() => setBuilderOpen(false)}>{t('activeIntelligence.beginner.later')}</Button>
      </header>
      <div className={styles.builderGrid}>
        <main className={styles.conversation}>
          <div className={styles.messages}>
            <div className={`${styles.messageRow} ${styles.userRow}`}><div className={`${styles.bubble} ${styles.userBubble}`}>{request}</div><span className={styles.userAvatar}><UserOutlined /></span></div>
            <div className={styles.messageRow}>
              <span className={styles.botAvatar}><RobotOutlined /></span>
              <div className={`${styles.bubble} ${styles.botBubble}`}>
                <p>{t('activeIntelligence.beginner.understood')}</p>
                {step === 'severity' && <><p>{t('activeIntelligence.beginner.askSeverity')}</p><strong>{t('activeIntelligence.beginner.importantQuestion')}</strong></>}
                {step === 'delivery' && <><p>{t('activeIntelligence.beginner.severityAnswered', { severity: severityLabel })}</p><strong>{t('activeIntelligence.beginner.askDelivery')}</strong></>}
                {step === 'ready' && <><p>{t('activeIntelligence.beginner.readyIntro')}</p><strong>{t('activeIntelligence.beginner.readyQuestion')}</strong></>}
              </div>
            </div>
            {step === 'severity' && <div className={styles.quickChoices}>
              <button type="button" onClick={() => chooseSeverity('both')}><CheckCircleOutlined />{t('activeIntelligence.beginner.together')}</button>
              <button type="button" onClick={() => chooseSeverity('critical')}><EyeOutlined />{t('activeIntelligence.beginner.criticalOnly')}</button>
              <button type="button" onClick={() => chooseSeverity('unsure')}><QuestionCircleOutlined />{t('activeIntelligence.beginner.unsure')}</button>
            </div>}
            {step === 'delivery' && <div className={styles.quickChoices}>
              <button type="button" onClick={() => finishDelivery(true)}><UserOutlined />{t('activeIntelligence.beginner.onlyMe')}</button>
              <button type="button" onClick={() => finishDelivery(false)}><BellOutlined />{t('activeIntelligence.beginner.operationsTeam')}</button>
            </div>}
            {step === 'ready' && <div className={styles.readyActions}>
              <Button type="primary" size="large" onClick={() => setTestOpen(true)}>{t('activeIntelligence.beginner.tryFirst')}</Button>
              <Button size="large" onClick={() => setStep('severity')}>{t('activeIntelligence.beginner.changeAnswer')}</Button>
            </div>}
          </div>
          <div className={styles.composerArea}>
            <div className={styles.examples}>
              <button type="button" onClick={() => { setRequest(t('activeIntelligence.beginner.exampleOffline')); setStep('severity'); }}><WifiOutlined />{t('activeIntelligence.beginner.exampleOffline')}</button>
              <button type="button" onClick={() => { setRequest(t('activeIntelligence.beginner.exampleDaily')); setStep('severity'); }}><CalendarOutlined />{t('activeIntelligence.beginner.exampleDaily')}</button>
              <button type="button" onClick={() => { setRequest(t('activeIntelligence.beginner.exampleReview')); setStep('severity'); }}><BellOutlined />{t('activeIntelligence.beginner.exampleReview')}</button>
            </div>
            <div className={styles.composer}><Input.TextArea autoSize={{ minRows: 1, maxRows: 3 }} value={request} placeholder={t('activeIntelligence.beginner.inputPlaceholder')} onChange={(event) => setRequest(event.target.value)} /><Button type="primary" icon={<SendOutlined />} aria-label={t('activeIntelligence.beginner.send')} onClick={() => setStep('severity')} /></div>
          </div>
        </main>

        <aside className={styles.summary}>
          <div className={styles.summaryHeader}><Typography.Title level={3}>{t('activeIntelligence.beginner.yourHelper')}</Typography.Title><Typography.Text type="secondary">{t('activeIntelligence.beginner.summaryHint')}</Typography.Text></div>
          <div className={styles.summaryItem}><span className={styles.summaryIcon}><CalendarOutlined /></span><div><small>{t('activeIntelligence.beginner.when')}</small><strong>{severity ? severityLabel : t('activeIntelligence.beginner.severeAndImportant')}</strong><p>{t('activeIntelligence.beginner.whenHint')}</p></div></div>
          <div className={styles.summaryItem}><span className={styles.summaryIcon}><SearchOutlined /></span><div><small>{t('activeIntelligence.beginner.what')}</small><strong>{t('activeIntelligence.beginner.whatValue')}</strong><p>{t('activeIntelligence.beginner.whatHint')}</p></div></div>
          <div className={styles.summaryItem}><span className={`${styles.summaryIcon} ${styles.summaryIconGreen}`}><UserOutlined /></span><div><small>{t('activeIntelligence.beginner.who')}</small><strong>{privateOnly ? t('activeIntelligence.beginner.whoMe') : t('activeIntelligence.beginner.whoTeam')}</strong><p>{t('activeIntelligence.beginner.whoHint')}</p></div></div>
          <div className={styles.safety}><SafetyCertificateOutlined /><div><strong>{t('activeIntelligence.beginner.readOnly')}</strong><span>{t('activeIntelligence.beginner.readOnlyHint')}</span></div></div>
          {connectionError && <div className={styles.connectionNotice}>{t('activeIntelligence.beginner.connectionWaiting')}</div>}
          <div className={styles.summaryFooter}><Button type="link" onClick={() => setBuilderOpen(false)}>{t('activeIntelligence.beginner.later')}</Button><Button type="primary" disabled={step !== 'ready' || Boolean(connectionError)} onClick={() => setTestOpen(true)}>{step === 'ready' ? t('activeIntelligence.beginner.tryFirst') : t('activeIntelligence.beginner.answerToContinue')}</Button></div>
          <button className={styles.professionalLink} type="button" onClick={() => setProfessionalOpen(true)}><SettingOutlined />{t('activeIntelligence.beginner.professional')}</button>
        </aside>
      </div>
    </section> : <section className={styles.helperList}>
      <header className={styles.listHeader}><div><Typography.Title level={2}>{t('activeIntelligence.beginner.myHelpers')}</Typography.Title><Typography.Text type="secondary">{t('activeIntelligence.beginner.myHelpersHint')}</Typography.Text></div><Button type="primary" size="large" icon={<RobotOutlined />} onClick={resetBuilder}>{t('activeIntelligence.beginner.newNeed')}</Button></header>
      <div className={styles.listSurface}>{data?.scenarios.map((item) => <button key={item.key} type="button" className={styles.helperRow} onClick={resetBuilder}>
        <span className={styles.helperIcon}><DeploymentUnitOutlined /></span><span><strong>{t(`activeIntelligence.scenario.${item.key}.beginnerName`)}</strong><small>{t(`activeIntelligence.scenario.${item.key}.beginnerDescription`)}</small></span>
        <Tag color={item.status === 'DISABLED' ? 'default' : item.rolloutMode === 'SHADOW' ? 'gold' : 'green'}>{item.status === 'DISABLED' ? t('activeIntelligence.beginner.paused') : item.rolloutMode === 'SHADOW' ? t('activeIntelligence.beginner.observing') : t('activeIntelligence.beginner.helping')}</Tag>
      </button>)}</div>
    </section>}

    <Modal open={testOpen} title={t('activeIntelligence.beginner.previewTitle')} okText={t('activeIntelligence.beginner.startTrial')} cancelText={t('activeIntelligence.beginner.keepTalking')} confirmLoading={saving} okButtonProps={{ disabled: Boolean(connectionError) }} onOk={() => void startTrial()} onCancel={() => setTestOpen(false)}>
      <div className={styles.previewNotice}>{t('activeIntelligence.beginner.previewNotice')}</div>
      <div className={styles.previewFinding}><Tag color="orange">{t('activeIntelligence.beginner.important')}</Tag><strong>{t('activeIntelligence.beginner.previewFindingTitle')}</strong><p>{t('activeIntelligence.beginner.previewFindingBody')}</p><span>{t('activeIntelligence.beginner.previewSuggestion')}</span></div>
      <div className={styles.previewSafety}><SafetyCertificateOutlined />{t('activeIntelligence.beginner.previewSafety')}</div>
    </Modal>
    <Drawer open={professionalOpen} size={560} title={t('activeIntelligence.beginner.professional')} onClose={() => setProfessionalOpen(false)}>
      {selectedScenario ? <div className={styles.professionalContent}><Typography.Paragraph type="secondary">{t('activeIntelligence.beginner.professionalHint')}</Typography.Paragraph><dl><dt>{t('activeIntelligence.events')}</dt><dd>{selectedScenario.eventTypes.join(', ')}</dd><dt>{t('activeIntelligence.allowedTools')}</dt><dd>{selectedScenario.allowedOperations.join(', ')}</dd><dt>{t('activeIntelligence.dedupe')}</dt><dd>{selectedScenario.dedupeWindowSeconds}s</dd><dt>{t('activeIntelligence.rateLimit')}</dt><dd>{selectedScenario.rateLimitPerHour}/h</dd></dl></div> : <Empty />}
    </Drawer>
  </div>;
}
