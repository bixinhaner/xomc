import { useState } from 'react';
import { Card, Button, Tag, Space, Typography, message, Descriptions } from 'antd';
import { ReloadOutlined, CheckCircleOutlined } from '@ant-design/icons';
import { useIntl } from 'react-intl';

import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useReloadDictLoader } from '@core/hooks/api/useDictLoader';
import {
  KNOWN_DICT_LOADERS,
  type DictLoaderName,
  type DictLoaderReloadResult,
} from '@core/services/api/dictLoaderApi';

const { Text } = Typography;

/**
 * T-0183 — 字典 Loader 热重载页。
 *
 * 列出 5 个 dictload Loader,每条提供"重载"按钮。完成后展示 rows / files /
 * elapsed / cache_keys_cleared 报告。
 *
 * 主要场景:
 *   - operator 手工编辑 paramModel XML 后,点 "参数模型字典" 重载 → 后端自动同步
 *     param_mappings + 清 Redis L2(无需 redis-cli DEL)
 *   - omcctl device sweep-paths --apply 用户已自动完成此动作,无需重复操作
 *
 * 鉴权:仅 super_admin 可见(后端 superAdminGroup 限制)。
 */
export default function DictLoaderPage(): JSX.Element {
  const intl = useIntl();
  const lang: 'zh-CN' | 'en-US' = intl.locale === 'en-US' ? 'en-US' : 'zh-CN';

  const reloadMutation = useReloadDictLoader();
  const [lastResult, setLastResult] = useState<
    Record<string, DictLoaderReloadResult | null>
  >({});

  const handleReload = async (name: DictLoaderName): Promise<void> => {
    try {
      const result = await reloadMutation.mutateAsync(name);
      setLastResult((prev) => ({ ...prev, [name]: result }));
      message.success(
        lang === 'en-US'
          ? `${name} reloaded: ${result.files_loaded} files, ${result.rows_affected} rows`
          : `${name} 已重载:${result.files_loaded} 个文件,${result.rows_affected} 行`,
      );
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e);
      message.error(`${name} reload failed: ${msg}`);
    }
  };

  return (
    <ListPageLayout title={lang === 'en-US' ? 'Dictionary Loader Reload' : '字典 Loader 重载'}>
      <Text type="secondary" style={{ display: 'block', marginBottom: 16 }}>
        {lang === 'en-US'
          ? 'After hand-editing dictionary XML (e.g. paramModel), click the corresponding button to trigger backend reload + auto cache invalidation. omcctl device sweep-paths --apply already does this automatically; no need to trigger here for sweep flow.'
          : '手工编辑字典 XML(如 paramModel)后,点击对应按钮触发后端重读 + 自动清缓存。omcctl device sweep-paths --apply 已自动完成此动作,无需在此重复操作。'}
      </Text>

      <Space direction="vertical" size={16} style={{ width: '100%' }}>
        {KNOWN_DICT_LOADERS.map((loader) => {
          const result = lastResult[loader.name];
          const isLoading =
            reloadMutation.isPending && reloadMutation.variables === loader.name;
          return (
            <Card
              key={loader.name}
              size="small"
              title={
                <Space>
                  <Text strong>{loader.labelI18n[lang]}</Text>
                  <Tag color="default">{loader.name}</Tag>
                </Space>
              }
              extra={
                <Button
                  type="primary"
                  size="small"
                  icon={<ReloadOutlined />}
                  loading={isLoading}
                  onClick={() => handleReload(loader.name)}
                >
                  {lang === 'en-US' ? 'Reload' : '重新加载'}
                </Button>
              }
            >
              <Text type="secondary">{loader.descriptionI18n[lang]}</Text>

              {result && (
                <Descriptions
                  size="small"
                  column={3}
                  style={{ marginTop: 12 }}
                  items={[
                    {
                      key: 'files',
                      label: lang === 'en-US' ? 'Files' : '文件',
                      children: `${result.files_loaded} loaded / ${result.files_skipped} skipped`,
                    },
                    {
                      key: 'rows',
                      label: lang === 'en-US' ? 'Rows' : '行',
                      children: result.rows_affected,
                    },
                    {
                      key: 'elapsed',
                      label: lang === 'en-US' ? 'Elapsed' : '耗时',
                      children: `${result.elapsed_ms} ms`,
                    },
                    ...(result.cache_keys_cleared !== undefined
                      ? [
                          {
                            key: 'cache',
                            label: lang === 'en-US' ? 'Cache' : '缓存',
                            children: `${result.cache_keys_cleared} keys cleared, v${result.cache_version_after}`,
                          },
                        ]
                      : []),
                    ...(result.errors && result.errors.length > 0
                      ? [
                          {
                            key: 'errors',
                            label: lang === 'en-US' ? 'Errors' : '错误',
                            children: <Text type="danger">{result.errors.join('; ')}</Text>,
                          },
                        ]
                      : [
                          {
                            key: 'ok',
                            label: lang === 'en-US' ? 'Status' : '状态',
                            children: (
                              <Text type="success">
                                <CheckCircleOutlined /> OK
                              </Text>
                            ),
                          },
                        ]),
                  ]}
                />
              )}
            </Card>
          );
        })}
      </Space>
    </ListPageLayout>
  );
}
