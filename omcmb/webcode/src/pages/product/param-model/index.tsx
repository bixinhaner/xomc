import { useState } from 'react';
import { Tabs, Card, Button, Space, message, Popconfirm } from 'antd';
import { CloudDownloadOutlined, ReloadOutlined } from '@ant-design/icons';
import {
  useParamModelImportDirectory,
  useParamModelCacheRefresh,
} from '@core/hooks/api/useParamModels';
import ModelsTab from './ModelsTab';
import MappingsTab from './MappingsTab';
import StandardParamsTab from './StandardParamsTab';

export default function ParamModelPage() {
  const [tab, setTab] = useState('models');
  const [selectedModelName, setSelectedModelName] = useState<string | undefined>();
  const importMut = useParamModelImportDirectory();
  const cacheMut = useParamModelCacheRefresh();

  return (
    <div style={{ padding: 16 }}>
      <Card
        size="small"
        style={{ marginBottom: 12 }}
        title="参数模型浏览器 / Parameter Models"
        extra={
          <Space>
            <Popconfirm
              title="确认重载 XML？"
              description={
                <div style={{ maxWidth: 320 }}>
                  将从 <code>datamodels/</code> 重新加载所有参数模型 XML。
                  <br />
                  UI 中对参数模型 / 映射的编辑将被 XML 值覆盖；操作不可撤销。
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
                  .then(() => message.success('已刷新参数模型缓存'))
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
          {
            key: 'models',
            label: '参数模型清单',
            children: (
              <ModelsTab
                selectedName={selectedModelName}
                onSelect={(name) => {
                  setSelectedModelName(name);
                  setTab('mappings');
                }}
              />
            ),
          },
          {
            key: 'mappings',
            label: '默认映射',
            children: <MappingsTab selectedName={selectedModelName} />,
          },
          {
            key: 'standard',
            label: '标准参数树',
            children: <StandardParamsTab />,
          },
        ]}
      />
    </div>
  );
}
