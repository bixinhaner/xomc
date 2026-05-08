import { useState } from 'react';
import { Tabs, Card, Button, Space, message, Tooltip } from 'antd';
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
            <Tooltip title="POST /param-models/import-directory；从后端 datamodels/ 重新加载所有 XML 模型">
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
