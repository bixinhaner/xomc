import { Alert, App, Button, Card, Select, Skeleton, Space, Tag } from 'antd';
import { useQuery, useQueryClient, useMutation } from '@tanstack/react-query';
import { proactiveAgentApi } from '@core/services/api/proactiveAgentApi';
import { useT } from '@/hooks/useT';
import type { ProactiveScenario } from '@core/types/proactiveAgent';
import styles from './index.module.css';
export default function LegacyScenarios() {
  const t = useT(); const { modal, message } = App.useApp(); const cache = useQueryClient();
  const query = useQuery({ queryKey: ['legacy-proactive-overview'], queryFn: () => proactiveAgentApi.overview(), retry: false });
  const change = useMutation({ mutationFn: ({ key, mode }: { key: string; mode: string }) => proactiveAgentApi.updateScenario(key, { status: mode === 'disabled' ? 'DISABLED' : 'ACTIVE', rolloutMode: mode === 'shadow' ? 'SHADOW' : 'FULL', rolloutPercentage: mode === 'disabled' ? 0 : 100 }), onSuccess: () => cache.invalidateQueries({ queryKey: ['legacy-proactive-overview'] }), onError: () => message.error(t('assistant.error.generic')) });
  const update = (scenario: ProactiveScenario, mode: string) => modal.confirm({ title: t('assistant.legacy.changeTitle'), content: t('assistant.legacy.changeBody'), okText: t('common.confirm'), cancelText: t('common.cancel'), onOk: () => change.mutateAsync({ key: scenario.key, mode }) });
  return <div className={styles.legacy}>
    <Alert type="info" showIcon title={t('assistant.legacyHint')} />
    {query.isPending && <Skeleton active />}
    {query.isError && <Alert type="warning" showIcon title={t('assistant.loadError')} action={<Button onClick={() => void query.refetch()}>{t('assistant.retry')}</Button>} />}
    {query.data?.scenarios.map((scenario) => <Card key={scenario.key} title={scenario.name} extra={<Select aria-label={`${scenario.name} ${t('assistant.configure')}`} style={{ minWidth: 160 }} disabled={change.isPending} value={scenario.status === 'DISABLED' ? 'disabled' : scenario.rolloutMode === 'SHADOW' ? 'shadow' : 'active'} options={['disabled', 'shadow', 'active'].map((value) => ({ value, label: t(`assistant.legacy.${value}`) }))} onChange={(value) => update(scenario, value)} />}>
      <p>{scenario.description}</p><Space wrap>{scenario.eventTypes.map((event) => <Tag key={event}>{event}</Tag>)}</Space>
    </Card>)}
  </div>;
}
