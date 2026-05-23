import { useState } from 'react';
import { Tabs } from 'antd';
import ConfigSnapshotLibrary from '@/pages/backup/ConfigSnapshotLibrary';
import FirmwareUpload from '@/pages/software/FirmwareUpload';

// tabBar 下面留 8px 给内容呼吸；ConfigSnapshotLibrary 自带 Card 已有内边距，
// 但 FirmwareUpload 内容贴 tabBar 显得拥挤——统一在父层给 16px 间距。
const PANE_STYLE: React.CSSProperties = { paddingTop: 8 };

export default function FileManagementPage() {
  const [activeKey, setActiveKey] = useState('version');
  return (
    <Tabs
      activeKey={activeKey}
      onChange={setActiveKey}
      destroyInactiveTabPane
      tabBarStyle={{ marginBottom: 16 }}
      items={[
        {
          key: 'version',
          label: '版本文件',
          children: <div style={PANE_STYLE}><FirmwareUpload embedded /></div>,
        },
        {
          key: 'config',
          label: '配置文件',
          children: <div style={PANE_STYLE}><ConfigSnapshotLibrary /></div>,
        },
      ]}
    />
  );
}
