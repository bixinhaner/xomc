import { Badge } from 'antd';
import { UpOutlined, DownOutlined } from '@ant-design/icons';
import { useTaskStore } from '@/store/taskStore';
import { useT } from '@/hooks/useT';
import styles from './TaskPanel.module.css';

export default function TaskPanelHeader() {
  const panelExpanded = useTaskStore((s) => s.panelExpanded);
  const togglePanel = useTaskStore((s) => s.togglePanel);
  const singleTasks = useTaskStore((s) => s.singleTasks);
  const batchTasks = useTaskStore((s) => s.batchTasks);
  const exportTasks = useTaskStore((s) => s.exportTasks);
  const t = useT();

  const runningCount =
    singleTasks.filter((tk) => tk.status === 'running').length +
    batchTasks.filter((tk) => tk.status === 'running').length +
    exportTasks.filter((tk) => tk.status === 'running').length;

  const totalCount = singleTasks.length + batchTasks.length + exportTasks.length;

  return (
    <div
      className={styles.panelHeader}
      onClick={togglePanel}
      role="button"
      tabIndex={0}
      aria-expanded={panelExpanded}
      aria-label={t('task.panel.title')}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault();
          togglePanel();
        }
      }}
    >
      <div className={styles.panelHeaderLeft}>
        <span className={styles.panelTitle}>{t('task.panel.title')}</span>
        {totalCount > 0 && (
          <Badge
            count={runningCount > 0 ? runningCount : totalCount}
            size="small"
            style={{
              backgroundColor: runningCount > 0 ? 'var(--color-primary-600)' : '#8c8c8c',
              fontSize: 10,
            }}
          />
        )}
      </div>
      <button
        className={styles.panelExpandBtn}
        type="button"
        onClick={(e) => {
          e.stopPropagation();
          togglePanel();
        }}
      >
        {panelExpanded ? <UpOutlined /> : <DownOutlined />}
      </button>
    </div>
  );
}
