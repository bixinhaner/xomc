/**
 * ParamModelPage — 参数模型主页面。
 *
 * 2026-05-28 用户决策(布局调整):
 *   1. "标准参数树"已拆为独立菜单页 product/standard-params,本页不再渲染
 *   2. drill-down 主从导航:
 *      - 默认显示参数模型清单(ModelsTab); 点击模型名 → 进入该模型的参数列表
 *      - 进入参数列表后,toolbar 左侧显示"返回"按钮 + 当前模型名;
 *        点击返回 → 清空 selectedModelName → 回到清单
 *   3. 顶部 toolbar 单行:
 *      - 列表态: [搜索 + 重载/刷新按钮]
 *      - 详情态: [返回 + 模型名 + 重载/刷新按钮]
 *
 * 2026-05-29 用户决策(语义对齐):
 *   - 原"上传 XML"和"导入 XML"两个按钮语义重叠,合并为单个"导入 XML"按钮;
 *     "导入 XML" = 用户选本地 XML 文件 → host /opt/omc/data/param-mappings-custom/
 *     上传链路(端点 POST /param-models/upload-xml,multipart);
 *     "重载 XML" = 后端扫 datamodels/ 全量 destructive 重载(行为不变,与旧版一致)。
 *   - 后端 import-directory?mode=import 端点物理保留(API contract 不破坏),
 *     UI 不再调用;import-directory?mode=reload 由"重载 XML"继续使用。
 *
 * 2026-05-29 用户决策(详情态收敛):
 *   - 详情态删除"导入 XML / 重载 XML / 刷新缓存"按钮 — 全局动作,只放列表态
 *   - "返回"按钮 + 模型名下沉到 MappingsTab Card 标题区,与 search/filter/新增映射
 *     合并为单行(避免上下两条 toolbar)
 *   - 详情态彻底不渲染顶部 toolbar Card
 */
import { useState } from 'react';
import { Card, Input, Button, Space, Popconfirm, message, Upload, Modal } from 'antd';
import type { UploadProps } from 'antd';
import {
  CloudDownloadOutlined,
  InboxOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import {
  useParamModelReloadDirectory,
  useParamModelCacheRefresh,
  useUploadParamModelXML,
} from '@core/hooks/api/useParamModels';
import type { AxiosError } from 'axios';
import ModelsTab from './ModelsTab';
import MappingsTab from './MappingsTab';
import { useT } from '@/hooks/useT';

export default function ParamModelPage() {
  const t = useT();
  const [selectedModelName, setSelectedModelName] = useState<string | undefined>();
  const [keyword, setKeyword] = useState('');
  const reloadMut = useParamModelReloadDirectory();
  const cacheMut = useParamModelCacheRefresh();
  const uploadMut = useUploadParamModelXML();

  const inDetail = Boolean(selectedModelName);

  // 2026-05-29:"导入 XML" 按钮的 Upload customRequest — 用 antd Upload 触发
  // multipart 上传到 POST /param-models/upload-xml(host bind mount
  // /opt/omc/data/param-mappings-custom/,升级不丢);409 同名 → Modal 确认 →
  // force=true 覆盖(旧文件备份 .bak.<ts>)。文案 / 提示统一走 antd message。
  const uploadProps: UploadProps = {
    accept: '.xml',
    maxCount: 1,
    showUploadList: false,
    customRequest: ({ file, onSuccess, onError }) => {
      const realFile = file as File;
      const runUpload = (force: boolean): Promise<void> =>
        uploadMut
          .mutateAsync({ file: realFile, force })
          .then((r) => {
            message.success(
              r.overwrite && r.backup
                ? t('product.paramModel.importOverride', { file: r.filename, backup: r.backup ?? '' })
                : t('product.paramModel.importSuccess', { file: r.filename }),
            );
            onSuccess?.(r);
          })
          .catch((e: unknown) => {
            const ax = e as AxiosError<{ message?: string }>;
            // 409 → 同名冲突,弹二次确认走 force=true
            if (ax.response?.status === 409 && !force) {
              Modal.confirm({
                title: t('product.paramModel.overrideTitle'),
                content: (
                  <div style={{ maxWidth: 360 }}>
                    {t('product.paramModel.overrideContentPre')}<code>{realFile.name}</code>{t('product.paramModel.overrideContentPost')}
                    <br />{t('product.paramModel.reloadHint')}
                  </div>
                ),
                okText: t('common.override'),
                okButtonProps: { danger: true },
                onOk: () => runUpload(true),
              });
              onError?.(ax);
              return;
            }
            const msg =
              ax.response?.data?.message ??
              (e instanceof Error ? e.message : String(e));
            message.error(msg);
            onError?.(ax);
          });
      void runUpload(false);
    },
  };

  return (
    <div style={{ padding: 16 }}>
      {/* 顶部 toolbar:仅列表态。详情态由 MappingsTab 自带头部(返回 + 模型名 + 筛选/新增 同行) */}
      {!inDetail && (
        <Card size="small" style={{ marginBottom: 12 }}>
          <Space style={{ width: '100%', justifyContent: 'space-between' }}>
            <Input.Search
              placeholder={t('product.paramModel.searchPh')}
              allowClear
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
              style={{ width: 320 }}
            />
            <Space>
              {/* "导入 XML" = 用户选本地 XML 上传,落 host /opt/omc/data/param-mappings-custom/ */}
              <Upload {...uploadProps}>
                <Button icon={<InboxOutlined />} loading={uploadMut.isPending}>
                  {t('common.importXml')}
                </Button>
              </Upload>
              <Popconfirm
                title={t('product.paramModel.reloadTitle')}
                description={
                  <div style={{ maxWidth: 360 }}>
                    {t('product.paramModel.reloadHint1Pre')}<code>datamodels/</code><b>{t('product.paramModel.reloadHint1Post')}</b>
                    <br />{t('product.paramModel.bullet.upsert')}
                    <br />{t('product.paramModel.bullet.orphan1')}<b>{t('product.paramModel.bullet.orphan2')}</b>
                    <br />{t('product.paramModel.bullet.cascade1')}<code>param_mappings</code>{t('product.paramModel.bullet.cascade2')}
                    <br />{t('product.paramModel.bullet.setnull1')}<code>products.param_model_id</code>{t('product.paramModel.bullet.setnull2')}
                    <br />{t('common.actionUndoable')}
                  </div>
                }
                okText={t('product.products.reloadOk')}
                cancelText={t('common.cancel')}
                okButtonProps={{ danger: true }}
                placement="bottomRight"
                onConfirm={() => {
                  reloadMut
                    .mutateAsync()
                    .then((raw) => {
                      const r = raw as { reloaded: number; orphans_deleted: number };
                      return message.success(
                        r.orphans_deleted > 0
                          ? t('product.paramModel.reloadOrphans', { count: r.reloaded, orphans: r.orphans_deleted })
                          : t('product.paramModel.reloadSuccess', { count: r.reloaded }),
                      );
                    })
                    .catch((e) => message.error((e as Error).message));
                }}
              >
                <Button icon={<CloudDownloadOutlined />} loading={reloadMut.isPending} danger>
                  {t('common.reloadXml')}
                </Button>
              </Popconfirm>
              <Button
                icon={<ReloadOutlined />}
                loading={cacheMut.isPending}
                onClick={() =>
                  cacheMut
                    .mutateAsync()
                    .then(() => message.success(t('product.paramModel.cacheRefreshed')))
                    .catch((e) => message.error((e as Error).message))
                }
              >
                {t('common.refreshCache')}
              </Button>
            </Space>
          </Space>
        </Card>
      )}

      {/* drill-down 主体:列表态 ↔ 详情态 二选一 */}
      {inDetail ? (
        <MappingsTab
          selectedName={selectedModelName}
          onBack={() => setSelectedModelName(undefined)}
        />
      ) : (
        <ModelsTab
          selectedName={selectedModelName}
          onSelect={setSelectedModelName}
          keyword={keyword}
        />
      )}
    </div>
  );
}
