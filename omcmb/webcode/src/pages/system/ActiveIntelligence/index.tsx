import { useEffect, useRef, useState } from 'react';
import { Alert, App, Badge, Button, Collapse, Input, Skeleton, Tabs, Tag, Tooltip } from 'antd';
import type { TextAreaRef } from 'antd/es/input/TextArea';
import { ArrowLeftOutlined, ArrowRightOutlined, BellOutlined, CalendarOutlined, CheckCircleOutlined, EditOutlined, ExperimentOutlined, LockOutlined, PauseCircleOutlined, PlayCircleOutlined, PlusOutlined, RobotOutlined, SafetyCertificateOutlined, SendOutlined, SettingOutlined, ThunderboltOutlined } from '@ant-design/icons';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useIntl } from 'react-intl';
import { useSearchParams } from 'react-router-dom';
import { assistantApi } from '@core/services/api/assistantApi';
import { useUserStore } from '@core/store/userStore';
import type { Assistant, AssistantDefinition, AssistantRun } from '@core/types/assistant';
import { useT } from '@/hooks/useT';
import AssistantEditor from './AssistantEditor';
import AssistantResults from './AssistantResults';
import LegacyScenarios from './LegacyScenarios';
import { canPublish, dateText, errorText, isRunning, trialFor, triggerText } from './helpers';
import styles from './index.module.css';

const examples = [{ key: 'daily', icon: <CalendarOutlined aria-hidden="true" /> }, { key: 'alarm', icon: <BellOutlined aria-hidden="true" /> }, { key: 'manual', icon: <ExperimentOutlined aria-hidden="true" /> }];
const stateColor = { draft: 'default', active: 'success', paused: 'default', blocked: 'warning' } as const;

export default function ActiveIntelligence() {
  const t = useT(); const { locale } = useIntl(); const { message, modal } = App.useApp(); const cache = useQueryClient();
  const owner = useUserStore((state) => state.currentUser?.id ?? 'current');
  const [params, setParams] = useSearchParams(); const selectedId = params.get('assistant') ?? '';
  const [tab, setTab] = useState('configure'); const [legacy, setLegacy] = useState(false);
  const [text, setText] = useState(''); const [pendingText, setPendingText] = useState(''); const [busy, setBusy] = useState('');
  const [error, setError] = useState<unknown>(); const [editOpen, setEditOpen] = useState(false);
  const inputRef = useRef<TextAreaRef>(null); const messagesEnd = useRef<HTMLDivElement>(null);
  const listKey = ['assistants', owner]; const detailKey = ['assistant', owner, selectedId]; const runsKey = ['assistant-runs', owner, selectedId];
  const list = useQuery({ queryKey: listKey, queryFn: ({ signal }) => assistantApi.list(signal), refetchInterval: 15000 });
  const catalog = useQuery({ queryKey: ['assistant-catalog', owner], queryFn: ({ signal }) => assistantApi.catalog(signal), retry: false, staleTime: 15000 });
  const detail = useQuery({ queryKey: detailKey, queryFn: ({ signal }) => assistantApi.get(selectedId, signal), enabled: !!selectedId, refetchInterval: busy ? false : 15000 });
  const history = useQuery({ queryKey: runsKey, queryFn: ({ signal }) => assistantApi.runs(selectedId, signal), enabled: !!selectedId, refetchInterval: (query) => query.state.data?.some(isRunning) ? 1500 : 15000 });
  const assistant = detail.data; const runs = history.data ?? []; const capabilities = catalog.data?.capabilities ?? [];
  const connected = catalog.data?.connected === true; const running = runs.some(isRunning);
  const trial = assistant ? trialFor(assistant, runs) : undefined;
  const publishReady = !!assistant && canPublish(assistant, runs) && connected;
  const plan = assistant?.definition;
  useEffect(() => { if (params.get('run')) setTab('results'); }, [params]);
  useEffect(() => { if (pendingText || assistant?.messages.length) messagesEnd.current?.scrollIntoView?.({ block: 'nearest', behavior: 'smooth' }); }, [pendingText, assistant?.messages.length]);
  useEffect(() => {
    if (!selectedId) return;
    const id = params.get('run');
    if (id && runs.some((run) => run.id === id && !run.readAt)) void assistantApi.read(id).then(() => cache.invalidateQueries({ queryKey: runsKey })).catch(() => undefined);
  // Mark only the explicitly opened notification, not every result in the list.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedId, params.get('run'), runs.find((run) => run.id === params.get('run'))?.readAt]);
  const updateAssistant = async (value: Assistant) => { cache.setQueryData(['assistant', owner, value.id], value); await cache.invalidateQueries({ queryKey: listKey }); };
  const open = (id: string, nextTab = 'configure') => {
    const next = new URLSearchParams(params); if (id) next.set('assistant', id); else next.delete('assistant'); next.delete('run');
    setParams(next); setTab(nextTab); setError(undefined); setText(''); setLegacy(false);
  };
  const act = async (name: string, fn: () => Promise<void>) => {
    if (busy) return; setBusy(name); setError(undefined);
    try { await fn(); }
    catch (cause) { setError(cause); if (String(cause).includes('REVISION_CONFLICT')) await cache.invalidateQueries({ queryKey: detailKey }); }
    finally { setBusy(''); }
  };
  const create = (example?: string) => void act('create', async () => {
    const value = await assistantApi.create(crypto.randomUUID(), locale, Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC');
    await updateAssistant(value); open(value.id); if (example) setText(example);
    setTimeout(() => inputRef.current?.focus(), 0);
  });
  const send = () => {
    if (!assistant || !text.trim() || busy || !connected) return;
    const input = text.trim(); setPendingText(input);
    void act('plan', async () => {
      const value = await assistantApi.plan(assistant.id, assistant.revision, input);
      await updateAssistant(value); setText('');
    }).finally(() => { setPendingText(''); inputRef.current?.focus(); });
  };
  const start = (kind: 'trial' | 'manual') => void act('run', async () => {
    if (!assistant) return;
    const run = await assistantApi.start(assistant.id, assistant.revision, kind, crypto.randomUUID());
    cache.setQueryData<AssistantRun[]>(runsKey, (old) => [run, ...(old ?? []).filter((item) => item.id !== run.id)]);
    setTab('results'); await cache.invalidateQueries({ queryKey: listKey });
  });
  const publish = () => {
    if (!assistant || !publishReady) return;
    modal.confirm({ title: t('assistant.publishTitle'), content: t(plan?.trigger.kind === 'manual' ? 'assistant.publishManualBody' : 'assistant.publishBody'), okText: t('assistant.publish'), cancelText: t('common.cancel'), onOk: async () => {
      await act('publish', async () => { await updateAssistant(await assistantApi.publish(assistant.id, assistant.revision)); message.success(t('assistant.published')); });
    } });
  };
  const state = (next: 'active' | 'paused') => {
    if (!assistant) return;
    const change = () => act('state', async () => { await updateAssistant(await assistantApi.state(assistant.id, next)); });
    if (next === 'paused') modal.confirm({ title: t('assistant.pauseTitle'), content: t('assistant.pauseBody'), okText: t('assistant.pause'), cancelText: t('common.cancel'), onOk: change });
    else void change();
  };
  const save = async (definition: AssistantDefinition) => {
    if (!assistant) return;
    await act('save', async () => { await updateAssistant(await assistantApi.save(assistant.id, assistant.revision, definition)); setEditOpen(false); message.success(t('assistant.saved')); });
  };
  const cancel = (id: string) => act('cancel', async () => { await assistantApi.cancel(id); await cache.invalidateQueries({ queryKey: runsKey }); });
  const loading = selectedId ? detail.isPending : list.isPending;
  const loadError = selectedId ? detail.error : list.error;
  return <div className={styles.page}>
    <header className={styles.pageHeader}>
      <div>
        {selectedId || legacy ? <Button type="text" icon={<ArrowLeftOutlined aria-hidden="true" />} className={styles.back} disabled={!!busy} onClick={() => open('')}>{t('assistant.back')}</Button> : <div className={styles.eyebrow}><RobotOutlined aria-hidden="true" /> {t('assistant.eyebrow')}</div>}
        <h1>{legacy ? t('assistant.systemScenarios') : selectedId ? plan?.name || t('assistant.untitled') : t('assistant.title')}</h1>
        <p className={styles.subtitle}>{selectedId ? t('assistant.safety') : t('assistant.subtitle')}</p>
      </div>
      <div className={styles.headerActions}>
        {!selectedId && !legacy && <><Button type="text" icon={<SettingOutlined aria-hidden="true" />} onClick={() => setLegacy(true)}>{t('assistant.systemScenarios')}</Button><Button type="primary" size="large" icon={<PlusOutlined aria-hidden="true" />} loading={busy === 'create'} disabled={!!busy} onClick={() => create()}>{t('assistant.new')}</Button></>}
        {assistant && <>
          <Tag color={stateColor[assistant.state]}>{t(`assistant.state.${assistant.state}`)}</Tag>
          {assistant.publishedRevision && assistant.state === 'active' && <Button icon={<PauseCircleOutlined aria-hidden="true" />} disabled={!!busy} onClick={() => state('paused')}>{t('assistant.pause')}</Button>}
          {assistant.publishedRevision && assistant.state !== 'active' && <Button icon={<PlayCircleOutlined aria-hidden="true" />} disabled={!!busy || !connected} onClick={() => state('active')}>{t('assistant.resume')}</Button>}
          {assistant.state === 'active' && <Button icon={<PlayCircleOutlined aria-hidden="true" />} disabled={!!busy || running || !connected} onClick={() => start('manual')}>{t('assistant.runNow')}</Button>}
          {publishReady && <Button type="primary" icon={<CheckCircleOutlined aria-hidden="true" />} disabled={!!busy} onClick={publish}>{t(assistant.publishedRevision ? 'assistant.publishChanges' : 'assistant.publish')}</Button>}
        </>}
      </div>
    </header>
    {error !== undefined && <Alert type="error" showIcon closable onClose={() => setError(undefined)} title={errorText(error, t)} className={styles.notice} />}
    {!legacy && !catalog.isPending && !catalog.isError && !connected && <Alert type="warning" showIcon title={t('assistant.notConnected')} description={t('assistant.connectionHint')} action={<Button size="small" onClick={() => void catalog.refetch()}>{t('assistant.retry')}</Button>} className={styles.notice} />}
    {!legacy && catalog.isError && <Alert type="error" showIcon title={errorText(catalog.error, t)} action={<Button size="small" onClick={() => void catalog.refetch()}>{t('assistant.retry')}</Button>} className={styles.notice} />}
    {!legacy && connected && !capabilities.length && <Alert type="warning" showIcon title={t('assistant.noCapabilities')} className={styles.notice} />}
    {legacy ? <LegacyScenarios /> : loading ? <div className={styles.loading}><Skeleton active paragraph={{ rows: 8 }} /></div> : loadError ? <Alert type="error" showIcon title={t('assistant.loadError')} action={<Button onClick={() => void (selectedId ? detail.refetch() : list.refetch())}>{t('assistant.retry')}</Button>} /> : !selectedId ? <>
      <div className={styles.stats}>
        {(['all', 'active', 'draft'] as const).map((key) => <div className={styles.stat} key={key}><span>{t(`assistant.stat.${key}`)}</span><strong>{key === 'all' ? list.data?.length ?? 0 : (list.data ?? []).filter((item) => key === 'active' ? item.state === 'active' : !item.publishedRevision).length}</strong></div>)}
        <div className={styles.privateNote}><SafetyCertificateOutlined aria-hidden="true" /><div><strong>{t('assistant.readOnly')}</strong><span>{t('assistant.private')}</span></div></div>
      </div>
      {!list.data?.length ? <section className={styles.welcome}>
        <div className={styles.welcomeIcon}><RobotOutlined aria-hidden="true" /></div><h2>{t('assistant.emptyTitle')}</h2><p>{t('assistant.emptyBody')}</p>
        <div className={styles.examples}>{examples.map((example) => <button type="button" key={example.key} disabled={!!busy} onClick={() => create(t(`assistant.example.${example.key}`))}>{example.icon}<strong>{t(`assistant.exampleLabel.${example.key}`)}</strong><span>{t(`assistant.example.${example.key}`)}</span><ArrowRightOutlined aria-hidden="true" /></button>)}</div>
      </section> : <div className={styles.assistantGrid}>
        {list.data.map((item) => <button type="button" key={item.id} className={styles.assistantCard} disabled={!!busy} onClick={() => open(item.id, item.publishedRevision ? 'results' : 'configure')}>
          <div className={styles.cardTop}><span className={styles.cardIcon}><RobotOutlined aria-hidden="true" /></span><Tag color={stateColor[item.state]}>{t(`assistant.state.${item.state}`)}</Tag></div>
          <h2>{item.definition?.name || t('assistant.untitled')}</h2><p className={styles.cardGoal}>{item.definition?.goal || t('assistant.startConversationHint')}</p>
          <div className={styles.cardMeta}><CalendarOutlined aria-hidden="true" /><span>{triggerText(item.publishedDefinition ?? item.definition, t)}</span></div>
          <div className={styles.cardFooter}><span><LockOutlined aria-hidden="true" /> {t('assistant.private')}</span><ArrowRightOutlined aria-hidden="true" /></div>
          <small>{t('assistant.lastRun')} · {item.lastRunAt ? dateText(item.lastRunAt, locale) : t('assistant.notRun')}</small>
        </button>)}
        <button type="button" className={styles.newCard} disabled={!!busy} onClick={() => create()}><PlusOutlined aria-hidden="true" /><strong>{t('assistant.new')}</strong><span>{t('assistant.startConversationHint')}</span></button>
      </div>}
    </> : assistant && <>
      {assistant.publishedRevision && assistant.publishedRevision !== assistant.revision && <Alert type="info" showIcon title={t('assistant.unpublished', { revision: assistant.publishedRevision })} className={styles.notice} />}
      {assistant.state === 'blocked' && <Alert type="warning" showIcon title={errorText(assistant.lastError, t)} className={styles.notice} />}
      <Tabs activeKey={tab} onChange={setTab} items={[
        { key: 'configure', label: <span><EditOutlined aria-hidden="true" /> {t('assistant.configure')}</span>, children: <div className={styles.workspace}>
          <section className={styles.conversation} aria-label={t('assistant.conversation')}>
            <div className={styles.conversationHeader}><RobotOutlined aria-hidden="true" /><strong>{t('assistant.conversation')}</strong><span className={styles.muted}>{t('assistant.version', { revision: assistant.revision })}</span></div>
            <div className={styles.messages} role="log" aria-label={t('assistant.conversation')} aria-live="polite">
              {!assistant.messages.length && <div className={styles.conversationWelcome}><span className={styles.welcomeIcon}><ThunderboltOutlined aria-hidden="true" /></span><h2>{t('assistant.startConversation')}</h2><p>{t('assistant.startConversationHint')}</p></div>}
              {assistant.messages.map((item, i) => <div className={`${styles.message} ${item.role === 'user' ? styles.userMessage : styles.botMessage}`} key={`${assistant.id}-${i}`}><span className={styles.messageLabel}>{item.role === 'assistant' ? <RobotOutlined aria-hidden="true" /> : <LockOutlined aria-hidden="true" />}</span><p>{item.text}</p></div>)}
              {pendingText && <><div className={`${styles.message} ${styles.userMessage}`}><p>{pendingText}</p></div><div className={styles.planning} role="status"><Badge status="processing" /><span>{t('assistant.planning')}</span></div></>}
              <div ref={messagesEnd} />
            </div>
            {assistant.questions.length > 0 && !pendingText && <div className={styles.questions}>{assistant.questions.map((question, i) => <p key={i}><span>{i + 1}</span>{question}</p>)}</div>}
            <div className={styles.composer}>
              {!assistant.messages.length && <div className={styles.exampleChips}>{examples.map((example) => <Button key={example.key} size="small" icon={example.icon} onClick={() => { setText(t(`assistant.example.${example.key}`)); inputRef.current?.focus(); }}>{t(`assistant.exampleLabel.${example.key}`)}</Button>)}</div>}
              <div className={styles.inputRow}><Input.TextArea ref={inputRef} value={text} onChange={(event) => setText(event.target.value)} aria-label={t('assistant.send')} placeholder={t(assistant.messages.length ? 'assistant.adjustPlaceholder' : 'assistant.placeholder')} autoSize={{ minRows: 2, maxRows: 6 }} maxLength={8000} disabled={busy === 'plan'} onKeyDown={(event) => { if (event.key === 'Enter' && !event.shiftKey && !event.nativeEvent.isComposing && event.keyCode !== 229) { event.preventDefault(); send(); } }} /><Tooltip title={t('assistant.send')}><Button type="primary" size="large" shape="circle" icon={<SendOutlined aria-hidden="true" />} aria-label={t('assistant.send')} loading={busy === 'plan'} disabled={!text.trim() || !!busy || !connected || !capabilities.length} onClick={send} /></Tooltip></div>
              <div className={styles.composerFooter}><span>{t('assistant.composerHint')}</span><span><LockOutlined aria-hidden="true" /> {t('assistant.private')}</span></div>
            </div>
          </section>
          <aside className={styles.planCard} aria-label={t('assistant.planCard')}>
            <div className={styles.planHeader}><span className={styles.eyebrow}>{t(assistant.readiness === 'ready' ? 'assistant.planReady' : assistant.readiness === 'unsupported' ? 'assistant.unsupported' : 'assistant.needsInput')}</span><h2>{t('assistant.planCard')}</h2><p>{t('assistant.planHint')}</p></div>
            {plan ? <>
              <dl className={styles.planDetails}>
                <div><dt><ExperimentOutlined aria-hidden="true" /> {t('assistant.goal')}</dt><dd>{plan.goal}</dd></div>
                <div><dt><SafetyCertificateOutlined aria-hidden="true" /> {t('assistant.scope')}</dt><dd>{plan.scope.kind === 'visible' ? t('assistant.scope.visible') : plan.scope.label || plan.scope.deviceId}</dd></div>
                <div><dt><CalendarOutlined aria-hidden="true" /> {t('assistant.trigger')}</dt><dd>{triggerText(plan, t)}</dd>{plan.trigger.kind === 'event' && plan.trigger.conditions.length > 0 && <small>{plan.trigger.conditions.map((item) => `${item.field} ${item.op} ${JSON.stringify(item.value)}`).join(' · ')}</small>}</div>
                <div><dt><BellOutlined aria-hidden="true" /> {t('assistant.notify')}</dt><dd>{t(`assistant.notify.${plan.notify}`)}</dd><small>{t('assistant.cooldown', { minutes: plan.cooldownMinutes })}</small></div>
              </dl>
              <Button type="text" icon={<EditOutlined aria-hidden="true" />} disabled={!!busy} onClick={() => setEditOpen(true)}>{t('assistant.edit')}</Button>
              <Collapse ghost items={[{ key: 'capabilities', label: t('assistant.technical'), children: <div className={styles.capabilities}>{plan.operations.map((operation) => <p key={operation}>{capabilities.find((item) => item.operationId === operation)?.title || operation}</p>)}</div> }]} />
            </> : <div className={styles.planEmpty}><ExperimentOutlined aria-hidden="true" /><p>{t('assistant.noPlan')}</p></div>}
            {assistant.missingCapabilities.length > 0 && <Alert type="warning" showIcon title={t('assistant.missing')} description={assistant.missingCapabilities.join('；')} className={styles.planNotice} />}
            <div className={styles.planActions}>
              <p><SafetyCertificateOutlined aria-hidden="true" /> {t('assistant.readOnly')}</p>
              <Button block type="primary" size="large" icon={<ExperimentOutlined aria-hidden="true" />} loading={busy === 'run'} disabled={!plan || assistant.readiness !== 'ready' || !!busy || !connected || running} onClick={() => start('trial')}>{t('assistant.trial')}</Button>
              <Button block size="large" disabled={!publishReady || !!busy} onClick={publish}>{t(assistant.publishedRevision ? 'assistant.publishChanges' : 'assistant.publish')}</Button>
              <small>{t(trial?.output?.outcome === 'insufficient_data' || trial?.status === 'FAILED' ? 'assistant.trialFailedGate' : 'assistant.trialRequired')}</small>
            </div>
          </aside>
        </div> },
        { key: 'results', label: <span><ExperimentOutlined aria-hidden="true" /> {t('assistant.results')} {running && <Badge status="processing" />}</span>, children: history.isPending ? <Skeleton active /> : history.isError ? <Alert type="error" title={t('assistant.loadError')} action={<Button onClick={() => void history.refetch()}>{t('assistant.retry')}</Button>} /> : <AssistantResults runs={runs} capabilities={capabilities} busy={!!busy} onCancel={cancel} selectedRunId={params.get('run') ?? undefined} /> },
      ]} />
      <div className={styles.detailFooter}><span><LockOutlined aria-hidden="true" /> {t('assistant.private')}</span><span>{assistant.nextRunAt ? `${t('assistant.nextRun')} · ${dateText(assistant.nextRunAt, locale)}` : assistant.state === 'active' ? t(assistant.publishedDefinition?.trigger.kind === 'event' ? 'assistant.listening' : 'assistant.manualReady') : t('assistant.notScheduled')}</span></div>
      {plan && <AssistantEditor definition={plan} open={editOpen} busy={busy === 'save'} onClose={() => setEditOpen(false)} onSave={save} />}
    </>}
  </div>;
}
