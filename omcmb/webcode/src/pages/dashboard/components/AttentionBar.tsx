import { useMemo, useState } from 'react';
import {
  Alert,
  Badge,
  Button,
  Card,
  Drawer,
  Empty,
  Grid,
  Pagination,
  Skeleton,
  Space,
  Tag,
  Tooltip,
} from 'antd';
import { AlertOutlined, CheckSquareOutlined, RightOutlined } from '@ant-design/icons';
import { useLocation, useNavigate } from 'react-router-dom';
import { useAttentionPage, useAttentionSummary } from '@core/hooks/api/useAttention';
import type {
  AttentionAction,
  AttentionItem,
  AttentionSection,
  AttentionSectionKey,
} from '@core/types/attention';
import { formatSystemTime } from '@core/utils/systemTime';
import { useT } from '@/hooks/useT';
import AgentFindingDrawer from '@/components/AgentFindingDrawer/AgentFindingDrawer';
import styles from './AttentionBar.module.css';

const ACTION_PRIORITY: AttentionAction[] = [
 'view_assistant_result',
  'view_agent_finding',
  'review_device_candidate',
  'view_alarm',
];

const ACTION_I18N: Record<AttentionAction, string> = {
 view_assistant_result: 'assistant.results',
  view_agent_finding: 'dashboard.attention.action.viewFinding',
  review_device_candidate: 'dashboard.attention.action.reviewCandidate',
  view_alarm: 'dashboard.attention.action.viewAlarm',
};

const PREVIEW_ITEM_LIMIT = 1;

function itemAction(item: AttentionItem): AttentionAction | undefined {
  return ACTION_PRIORITY.find((action) => item.allowedActions.includes(action));
}

function itemTime(item: AttentionItem): string | undefined {
  return item.occurredAt ?? item.createdAt ?? item.updatedAt;
}

function itemTag(item: AttentionItem): { color: string; key: string } | undefined {
  if (item.kind === 'active_alarm' && item.severity) {
    return {
      color: item.severity === 'critical' ? 'red' : 'orange',
      key: `dashboard.attention.level.${item.severity}`,
    };
  }
  if (item.kind === 'device_access_review') {
    return { color: 'blue', key: 'dashboard.attention.type.deviceReview' };
  }
  if (item.kind === 'assistant_result') return { color: 'blue', key: 'assistant.eyebrow' };
  if (item.kind === 'agent_finding') {
    return { color: item.severity === 'critical' ? 'red' : 'purple', key: 'dashboard.attention.type.agentFinding' };
  }
  return undefined;
}

function AttentionItemButton({ item, onOpen, drawer = false }: {
  item: AttentionItem;
  onOpen: (item: AttentionItem) => void;
  drawer?: boolean;
}) {
  const t = useT();
  const action = itemAction(item);
  const timestamp = itemTime(item);
  const tag = itemTag(item);
  return (
    <button
      type="button"
      className={`${styles.itemButton} ${drawer ? styles.drawerItem : ''}`}
      onClick={() => onOpen(item)}
      aria-label={`${item.title} · ${action ? t(ACTION_I18N[action]) : t('common.view')}`}
    >
      <span className={styles.itemHeader}>
        <Tooltip title={item.title}>
          <span className={styles.itemTitle}>{item.title}</span>
        </Tooltip>
        <Space size={4}>
          {tag && (
            <Tag color={tag.color}>
              {t(tag.key)}
            </Tag>
          )}
          <RightOutlined style={{ fontSize: 10 }} />
        </Space>
      </span>
      <span className={styles.itemSummary}>{item.summary || '—'}</span>
      <span className={styles.itemMeta}>
        {timestamp && <span>{formatSystemTime(timestamp)}</span>}
        {action && <span>{t(ACTION_I18N[action])}</span>}
      </span>
    </button>
  );
}

function SectionState({ sectionKey, section, loading, failed, onRetry, onOpen }: {
  sectionKey: AttentionSectionKey;
  section?: AttentionSection;
  loading: boolean;
  failed: boolean;
  onRetry: () => void;
  onOpen: (item: AttentionItem) => void;
}) {
  const t = useT();
  if (loading) return <Skeleton active paragraph={{ rows: 2 }} title={false} />;
  if (failed || section?.status === 'error') {
    return <Alert type="error" showIcon title={t('dashboard.attention.loadFailed')} action={<Button size="small" onClick={onRetry}>{t('dashboard.attention.retry')}</Button>} />;
  }
  if (!section?.items.length) {
    return <div className={styles.compactEmpty}>{t(sectionKey === 'abnormalities' ? 'dashboard.attention.emptyAbnormalities' : 'dashboard.attention.emptyTodos')}</div>;
  }
  return <>{section.items.slice(0, PREVIEW_ITEM_LIMIT).map((item) => <AttentionItemButton key={item.id} item={item} onOpen={onOpen} />)}</>;
}

function CompactSection({ sectionKey, section, loading, failed, onRetry, onViewAll, onOpen }: {
  sectionKey: AttentionSectionKey;
  section?: AttentionSection;
  loading: boolean;
  failed: boolean;
  onRetry: () => void;
  onViewAll: () => void;
  onOpen: (item: AttentionItem) => void;
}) {
  const t = useT();
  const isAbnormal = sectionKey === 'abnormalities';
  return (
    <section
      className={styles.sectionPanel}
      aria-label={t(isAbnormal ? 'dashboard.attention.abnormalities' : 'dashboard.attention.todos')}
    >
      <div className={styles.sectionHeader}>
        <Space size={8} className={styles.sectionTitle}>
          {isAbnormal ? <AlertOutlined /> : <CheckSquareOutlined />}
          <span>{t(isAbnormal ? 'dashboard.attention.abnormalities' : 'dashboard.attention.todos')}</span>
          <Badge count={section?.total ?? 0} showZero overflowCount={999} />
        </Space>
        <Space size={8} className={styles.sectionActions}>
          {section?.status === 'partial' && <Tag color="orange">{t('dashboard.attention.partial')}</Tag>}
          <Button type="link" size="small" onClick={onViewAll}>{t('dashboard.attention.viewAll')}</Button>
        </Space>
      </div>
      <div className={styles.sectionBody}>
        <SectionState sectionKey={sectionKey} section={section} loading={loading} failed={failed} onRetry={onRetry} onOpen={onOpen} />
      </div>
    </section>
  );
}

function AttentionDrawer({ open, section, onClose, onOpen }: {
  open: boolean;
  section: AttentionSectionKey;
  onClose: () => void;
  onOpen: (item: AttentionItem) => void;
}) {
  const t = useT();
  const screens = Grid.useBreakpoint();
  const [pages, setPages] = useState<Record<AttentionSectionKey, number>>({ abnormalities: 1, todos: 1 });
  const [pageSize, setPageSize] = useState(20);
  const currentPage = pages[section];
  const query = useAttentionPage(section, currentPage, pageSize, open);
  const data = query.data;
  const emptyKey = section === 'abnormalities' ? 'dashboard.attention.emptyAbnormalities' : 'dashboard.attention.emptyTodos';
  const drawerTitle = section === 'abnormalities'
    ? t('dashboard.attention.abnormalities')
    : t('dashboard.attention.todos');
  const drawerHint = section === 'abnormalities'
    ? t('dashboard.attention.drawerHintAbnormalities')
    : t('dashboard.attention.drawerHintTodos');
  const drawerIcon = section === 'abnormalities' ? <AlertOutlined /> : <CheckSquareOutlined />;
  const content = useMemo(() => {
    if (query.isPending) return <Skeleton active paragraph={{ rows: 6 }} title={false} />;
    if (query.isError || data?.status === 'error') return <Alert type="error" showIcon title={t('dashboard.attention.loadFailed')} action={<Button size="small" onClick={() => void query.refetch()}>{t('dashboard.attention.retry')}</Button>} />;
    if (!data?.items.length) return <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t(emptyKey)} />;
    return <>{data.status === 'partial' && <Alert type="warning" showIcon title={t('dashboard.attention.partial')} style={{ marginBottom: 8 }} />}{data.items.map((item) => <AttentionItemButton key={item.id} item={item} drawer onOpen={onOpen} />)}</>;
  }, [data, emptyKey, onOpen, query, t]);

  return (
    <Drawer
      title={(
        <Space size={10} className={styles.drawerTitle}>
          {drawerIcon}
          <span>{drawerTitle}</span>
          {typeof data?.total === 'number' && <Badge count={data.total} showZero overflowCount={999} />}
        </Space>
      )}
      open={open}
      onClose={onClose}
      size={screens.md ? 520 : '100%'}
      className={styles.attentionDrawer}
    >
      <div className={styles.drawerContent}>
        <div className={styles.drawerIntro}>{drawerHint}</div>
        <div className={styles.drawerList} aria-busy={query.isFetching}>{content}</div>
        {(data?.total ?? 0) > 0 && (
          <div className={styles.drawerFooter}>
            <Pagination
              current={currentPage}
              pageSize={pageSize}
              total={data?.total ?? 0}
              showSizeChanger
              pageSizeOptions={[10, 20, 50]}
              onChange={(page, nextPageSize) => {
                if (nextPageSize !== pageSize) {
                  setPageSize(nextPageSize);
                  setPages({ abnormalities: 1, todos: 1 });
                  return;
                }
                setPages((current) => ({ ...current, [section]: page }));
              }}
            />
          </div>
        )}
      </div>
    </Drawer>
  );
}

export default function AttentionBar() {
  const t = useT();
  const navigate = useNavigate();
  const location = useLocation();
  const summary = useAttentionSummary();
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [drawerSection, setDrawerSection] = useState<AttentionSectionKey>('abnormalities');
  const initialFindingId = new URLSearchParams(location.search).get('agentFinding') ?? undefined;
  const [findingId, setFindingId] = useState<string | undefined>(initialFindingId);
  const openItem = (item: AttentionItem) => {
    if (item.kind === 'assistant_result') return { color: 'blue', key: 'assistant.eyebrow' };
  if (item.kind === 'agent_finding') {
      setFindingId(item.sourceId);
      const params = new URLSearchParams(location.search);
      params.set('agentFinding', item.sourceId);
      void navigate(`${location.pathname}?${params.toString()}`, { replace: true });
      return;
    }
    void navigate(item.detailRoute);
  };
  const closeFinding = () => {
    setFindingId(undefined);
    const params = new URLSearchParams(location.search);
    params.delete('agentFinding');
    const query = params.toString();
    void navigate(`${location.pathname}${query ? `?${query}` : ''}`, { replace: true });
  };
  const showAll = (section: AttentionSectionKey) => {
    setDrawerSection(section);
    setDrawerOpen(true);
  };

  return (
    <>
      <Card
        size="small"
        className={styles.rootCard}
        title={t('dashboard.attention.drawerTitle')}
        data-testid="dashboard-attention-bar"
      >
        <div className={styles.sectionsGrid}>
          <CompactSection sectionKey="abnormalities" section={summary.data?.abnormalities} loading={summary.isPending} failed={summary.isError} onRetry={() => void summary.refetch()} onViewAll={() => showAll('abnormalities')} onOpen={openItem} />
          <CompactSection sectionKey="todos" section={summary.data?.todos} loading={summary.isPending} failed={summary.isError} onRetry={() => void summary.refetch()} onViewAll={() => showAll('todos')} onOpen={openItem} />
        </div>
      </Card>
      <AttentionDrawer open={drawerOpen} section={drawerSection} onClose={() => setDrawerOpen(false)} onOpen={openItem} />
      <AgentFindingDrawer findingId={findingId} open={Boolean(findingId)} onOpenChange={(open) => { if (!open) closeFinding(); }} />
    </>
  );
}
