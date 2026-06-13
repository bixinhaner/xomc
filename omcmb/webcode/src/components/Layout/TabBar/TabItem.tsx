import { useSortable } from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import type { TabItem as TabItemType } from '@core/store/tabStore';
import { useT } from '@/hooks/useT';
import TabContextMenu from './TabContextMenu';
import styles from './TabBar.module.css';

interface Props {
  tab: TabItemType;
  /** 已按当前语言解析好的标题（由 TabBar 经 useTabLabelResolver 统一计算）。 */
  label: string;
  isActive: boolean;
  onClose: (key: string, e: React.MouseEvent) => void;
  onClick: (tab: TabItemType) => void;
}

export default function TabItem({ tab, label, isActive, onClose, onClick }: Props) {
  const t = useT();
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({
    id: tab.key,
    disabled: tab.key === 'dashboard',
  });

  const style: React.CSSProperties = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  };

  const handleMiddleClick = (e: React.MouseEvent) => {
    if (e.button === 1 && tab.closable !== false) {
      e.preventDefault();
      onClose(tab.key, e);
    }
  };

  const classNames = [
    styles.tab,
    isActive ? styles.active : '',
    tab.key === 'dashboard' ? styles.dashboard : '',
    isDragging ? styles.dragging : '',
  ]
    .filter(Boolean)
    .join(' ');

  const resolvedLabel = label;

  return (
    <TabContextMenu tab={tab}>
      <div
        ref={setNodeRef}
        style={style}
        className={classNames}
        onClick={() => onClick(tab)}
        onMouseDown={handleMiddleClick}
        {...attributes}
        {...listeners}
        role="tab"
        aria-selected={isActive}
        tabIndex={isActive ? 0 : -1}
      >
        <span className={styles.tabLabel} title={resolvedLabel}>
          {resolvedLabel}
        </span>
        {tab.closable !== false && (
          <button
            className={styles.tabClose}
            onClick={(e) => onClose(tab.key, e)}
            aria-label={`${t('common.close')} ${resolvedLabel}`}
            title={t('common.close')}
            type="button"
          >
            ×
          </button>
        )}
      </div>
    </TabContextMenu>
  );
}
