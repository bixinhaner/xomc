import { useState } from 'react';
import { Alert, Button, Space, Tabs, message } from 'antd';
import { CloseCircleOutlined } from '@ant-design/icons';
import { useSearchParams } from 'react-router-dom';
import ConfigSnapshotLibrary from '@/pages/backup/ConfigSnapshotLibrary';
import DeviceLicenseLibrary from '@/pages/backup/DeviceLicenseLibrary';
import FirmwareUpload from '@/pages/software/FirmwareUpload';
import MRFilesPage from '@/pages/mr/Files';
import PMFilesPage from '@/pages/pm/Files';
import KpiExportLibrary from '@/pages/transfer/KpiExport/KpiExportLibrary';
import ImsParamLibrary from '@/pages/transfer/ImsParamLibrary';
import { useT } from '@/hooks/useT';

const VALID_TABS = new Set(['version', 'config', 'license', 'imsParam', 'mr', 'pm', 'kpiExport']);

const PANE_STYLE: React.CSSProperties = { paddingTop: 8 };

export default function FileManagementPage() {
  const t = useT();
  const [searchParams, setSearchParams] = useSearchParams();
  // 支持 deep link：?tab=license 直接定位到 License 文件 tab
  const queryTab = searchParams.get('tab') ?? '';
  const initialKey = VALID_TABS.has(queryTab) ? queryTab : 'version';
  const [activeKey, setActiveKey] = useState(initialKey);
  // ?return=ufte 表示这是从「任务创建」抽屉点"打开 XXX 文件管理"按钮 window.open
  // 的新 tab —— 用户完成上传后应**关闭本 tab**返回原任务创建抽屉（状态不丢）。
  // 与 software/FirmwareUpload 的 fromUFTE 同款行为。
  const fromUFTE = searchParams.get('return') === 'ufte';

  const handleTabChange = (k: string) => {
    setActiveKey(k);
    const next = new URLSearchParams(searchParams);
    next.set('tab', k);
    setSearchParams(next, { replace: true });
  };

  const handleCloseTab = () => {
    // window.open 打开的同源 tab，可以 window.close() 自闭
    window.close();
    // 兜底：浏览器拒绝关闭时（极少见，比如脚本之外打开的 tab）给个提示
    setTimeout(() => {
      void message.info(t('ufte.fileManagement.autoCloseBlocked'));
    }, 300);
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 8, flex: 1, minHeight: 0 }}>
      {fromUFTE && (
        <Alert
          type="info"
          showIcon
          message={t('ufte.fileManagement.fromUfte.title')}
          description={t('ufte.fileManagement.fromUfte.desc')}
          action={(
            <Space>
              <Button
                size="small"
                type="primary"
                icon={<CloseCircleOutlined />}
                onClick={handleCloseTab}
              >
                {t('ufte.fileManagement.close')}
              </Button>
            </Space>
          )}
          style={{ marginBottom: 4 }}
        />
      )}
      <Tabs
        activeKey={activeKey}
        onChange={handleTabChange}
        destroyOnHidden
        tabBarStyle={{ marginBottom: 16 }}
        // fromUFTE 时只显示当前 tab —— 用户是从某个具体任务类型（升级 / 配置恢复 /
        // license 升级）的"打开 XX 文件管理"按钮跳过来的，其它 tab 露出反而会
        // 让用户跑题走错。直链进入（菜单点 / 收藏夹）则保留三 tab 全展示。
        items={[
          {
            key: 'version',
            label: t('ufte.fileManagement.tab.version'),
            children: <div style={PANE_STYLE}><FirmwareUpload embedded /></div>,
          },
          {
            key: 'config',
            label: t('ufte.fileManagement.tab.config'),
            children: <div style={PANE_STYLE}><ConfigSnapshotLibrary /></div>,
          },
          {
            key: 'license',
            label: t('ufte.fileManagement.tab.license'),
            children: <div style={PANE_STYLE}><DeviceLicenseLibrary /></div>,
          },
          {
            key: 'imsParam',
            label: t('ufte.fileManagement.tab.imsParam'),
            children: <div style={PANE_STYLE}><ImsParamLibrary /></div>,
          },
          {
            key: 'mr',
            label: t('ufte.fileManagement.tab.mr'),
            children: <div style={PANE_STYLE}><MRFilesPage embedded /></div>,
          },
          {
            key: 'pm',
            label: t('ufte.fileManagement.tab.pm'),
            children: <div style={PANE_STYLE}><PMFilesPage embedded /></div>,
          },
          {
            key: 'kpiExport',
            label: t('ufte.fileManagement.tab.kpiExport'),
            children: <div style={PANE_STYLE}><KpiExportLibrary /></div>,
          },
        ].filter((item) => !fromUFTE || item.key === activeKey)}
      />
    </div>
  );
}
