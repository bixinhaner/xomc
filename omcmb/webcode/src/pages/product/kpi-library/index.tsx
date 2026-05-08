import { useState } from 'react';
import { Tabs, Card, Button, Space, message, Tooltip } from 'antd';
import { CloudDownloadOutlined, ReloadOutlined } from '@ant-design/icons';
import {
  useIndicatorImportDirectory,
  useIndicatorCacheRefresh,
} from '@core/hooks/api/useIndicatorsLibrary';
import IndicatorTab from './IndicatorTab';
import EnabledIndicatorsTab from './EnabledIndicatorsTab';
import IndicatorUnitsTab from './IndicatorUnitsTab';

export default function KpiLibraryPage() {
  const [tab, setTab] = useState('ENB');
  const importMut = useIndicatorImportDirectory();
  const cacheMut = useIndicatorCacheRefresh();

  return (
    <div style={{ padding: 16 }}>
      <Card
        size="small"
        style={{ marginBottom: 12 }}
        title="KPI 指标库 / KPI Library"
        extra={
          <Space>
            <Tooltip title="POST /indicators/import-directory；从后端 datamodels/ 重新加载所有指标 XML">
              <Button
                icon={<CloudDownloadOutlined />}
                loading={importMut.isPending}
                onClick={() =>
                  importMut
                    .mutateAsync()
                    .then((r) => message.success(`已重载：${r.reloaded}`))
                    .catch((e) => message.error((e as Error).message))
                }
              >
                XML 导入 / 重载
              </Button>
            </Tooltip>
            <Button
              icon={<ReloadOutlined />}
              loading={cacheMut.isPending}
              onClick={() =>
                cacheMut
                  .mutateAsync()
                  .then((r) => message.success(r.note ? `${r.note}` : '已刷新缓存'))
                  .catch((e) => message.error((e as Error).message))
              }
            >
              刷新缓存
            </Button>
          </Space>
        }
      />

      <Tabs
        activeKey={tab}
        onChange={setTab}
        items={[
          { key: 'ENB', label: 'ENB (LTE)', children: <IndicatorTab deviceType="ENB" /> },
          { key: 'GSM', label: 'GSM', children: <IndicatorTab deviceType="GSM" /> },
          { key: 'GNB', label: 'GNB (5G NR)', children: <IndicatorTab deviceType="GNB" /> },
          { key: 'enabled', label: '启用指标', children: <EnabledIndicatorsTab /> },
          { key: 'units', label: '单位定义', children: <IndicatorUnitsTab /> },
        ]}
      />
    </div>
  );
}
