import { Tabs } from 'antd';
import { useTaskStore } from '@core/store/taskStore';
import { useT } from '@/hooks/useT';
import TaskPanelHeader from './TaskPanelHeader';
import SingleTaskTab from './SingleTaskTab';
import BatchTaskTab from './BatchTaskTab';
import ExportTaskTab from './ExportTaskTab';
import styles from './TaskPanel.module.css';

export default function TaskPanel() {
  const panelExpanded = useTaskStore((s) => s.panelExpanded);
  const activeTab = useTaskStore((s) => s.activeTab);
  const setActiveTab = useTaskStore((s) => s.setActiveTab);
  const t = useT();

  const tabItems = [
    {
      key: 'single',
      label: t('task.single'),
      children: <SingleTaskTab />,
    },
    {
      key: 'batch',
      label: t('task.batch'),
      children: <BatchTaskTab />,
    },
    {
      key: 'export',
      label: t('task.export'),
      children: <ExportTaskTab />,
    },
  ];

  return (
    <div
      className={`${styles.panel} ${panelExpanded ? styles.panelExpanded : styles.panelCollapsed}`}
      aria-label={t('task.panel.title')}
    >
      <TaskPanelHeader />
      {panelExpanded && (
        <div className={styles.panelBody}>
          <Tabs
            activeKey={activeTab}
            onChange={(key) => setActiveTab(key as 'single' | 'batch' | 'export')}
            size="small"
            items={tabItems}
            style={{ flex: 1 }}
            tabBarStyle={{
              margin: '0 0 0 0',
              padding: '0 16px',
              height: 36,
              borderBottom: '1px solid #f0f0f0',
              background: '#fafafa',
            }}
          />
        </div>
      )}
    </div>
  );
}
