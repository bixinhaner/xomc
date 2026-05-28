import { useState } from 'react';
import { Tabs, Card, Button, Space, message, Popconfirm } from 'antd';
import { CloudDownloadOutlined, ReloadOutlined } from '@ant-design/icons';
import {
  useIndicatorImportDirectory,
  useIndicatorCacheRefresh,
} from '@core/hooks/api/useIndicatorsLibrary';
import IndicatorTab from './IndicatorTab';
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
        extra={
          <Space>
            <Popconfirm
              title="确认重载 XML？"
              description={
                <div style={{ maxWidth: 320 }}>
                  将从 <code>datamodels/</code> 重新加载所有指标 XML。
                  <br />
                  UI 中对指标定义 / 启用状态 / 单位的编辑将被 XML 值覆盖；操作不可撤销。
                </div>
              }
              okText="确认重载"
              cancelText="取消"
              okButtonProps={{ danger: true }}
              placement="bottomRight"
              onConfirm={() => {
                // 不返回 Promise — 让 Popconfirm 立即关闭；loading 反馈交给触发按钮
                importMut
                  .mutateAsync()
                  .then((r) => message.success(`已重载：${r.reloaded}`))
                  .catch((e) => message.error((e as Error).message));
              }}
            >
              <Button icon={<CloudDownloadOutlined />} loading={importMut.isPending} danger>
                XML 导入 / 重载
              </Button>
            </Popconfirm>
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
          { key: 'units', label: '单位定义', children: <IndicatorUnitsTab /> },
        ]}
      />
    </div>
  );
}
