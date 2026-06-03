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
 *      - 列表态: [搜索 + 导入 XML 按钮]
 *      - 详情态: 由 MappingsTab 自带头部
 *
 * 2026-06-03 用户决策(三页统一):
 *   - 合并「导入 XML / 重载 XML / 刷新缓存」为单个「导入 XML」按钮(用户选本地 XML
 *     上传到 host /opt/omc/data/param-mappings-custom/,端点 POST /param-models/upload-xml)。
 *   - 后端上传端点内部已自动 destructive 重载(删孤儿)+ 刷新缓存,前端无需再单独调;
 *     旧的 import-directory / cache/refresh 端点已下线,对应 reload/cache 按钮删除。
 *   - 上传前查重:命中已存在文件名 → 弹覆盖确认 → force=true 上传(409 仍作兜底)。
 *
 * 2026-05-29 用户决策(详情态收敛):
 *   - 详情态删除"导入 XML / 重载 XML / 刷新缓存"按钮 — 全局动作,只放列表态
 *   - "返回"按钮 + 模型名下沉到 MappingsTab Card 标题区,与 search/filter/新增映射
 *     合并为单行(避免上下两条 toolbar)
 *   - 详情态彻底不渲染顶部 toolbar Card
 */
import { useState } from 'react';
import { Card, Input, Button, Space, message, Upload, Modal } from 'antd';
import type { UploadProps } from 'antd';
import { InboxOutlined } from '@ant-design/icons';
import {
  useParamModelList,
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
  const uploadMut = useUploadParamModelXML();
  // 上传前查重数据源:已存在的参数模型清单(按 loadedFrom basename 比对文件名)。
  const { data: modelListData } = useParamModelList();

  const inDetail = Boolean(selectedModelName);

  // 文件名查重:与已存在模型的 loadedFrom basename 大小写不敏感比对。
  const isDuplicateFile = (fileName: string): boolean => {
    const target = fileName.toLowerCase();
    return (modelListData?.items || []).some((m) => {
      const base = (m.loadedFrom || '').split('/').pop()?.toLowerCase() ?? '';
      return base === target;
    });
  };

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
      // 弹覆盖确认(查重命中 / 后端 409 兜底共用)。
      const confirmOverride = () =>
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
            // 409 → 同名冲突(兜底),弹二次确认走 force=true
            if (ax.response?.status === 409 && !force) {
              confirmOverride();
              onError?.(ax);
              return;
            }
            const msg =
              ax.response?.data?.message ??
              (e instanceof Error ? e.message : String(e));
            message.error(msg);
            onError?.(ax);
          });
      // 上传前查重:文件名已存在 → 先弹覆盖确认;否则直接上传(409 仍作兜底)。
      if (isDuplicateFile(realFile.name)) {
        confirmOverride();
        return;
      }
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
              {/* 「导入 XML」= 用户选本地 XML 上传,落 host /opt/omc/data/param-mappings-custom/;
                  后端上传端点内部已自动重载 + 刷新缓存。 */}
              <Upload {...uploadProps}>
                <Button icon={<InboxOutlined />} loading={uploadMut.isPending}>
                  {t('common.importXml')}
                </Button>
              </Upload>
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
