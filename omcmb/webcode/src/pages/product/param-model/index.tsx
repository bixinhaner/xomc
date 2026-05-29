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
 */
import { useState } from 'react';
import { Card, Input, Button, Space, Popconfirm, message, Typography, Upload, Modal } from 'antd';
import type { UploadProps } from 'antd';
import {
  ArrowLeftOutlined,
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

const { Text } = Typography;

export default function ParamModelPage() {
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
                ? `已覆盖导入:${r.filename}(旧版备份 ${r.backup})`
                : `已导入:${r.filename}`,
            );
            onSuccess?.(r);
          })
          .catch((e: unknown) => {
            const ax = e as AxiosError<{ message?: string }>;
            // 409 → 同名冲突,弹二次确认走 force=true
            if (ax.response?.status === 409 && !force) {
              Modal.confirm({
                title: '同名 XML 已存在',
                content: (
                  <div style={{ maxWidth: 360 }}>
                    检测到 <code>{realFile.name}</code> 已存在,确认覆盖?
                    <br />旧文件会自动备份为 <code>.bak.&lt;ts&gt;</code>。
                  </div>
                ),
                okText: '覆盖',
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
      {/* 顶部 toolbar:列表态展示搜索;详情态展示返回 + 当前模型名 */}
      <Card size="small" style={{ marginBottom: 12 }}>
        <Space style={{ width: '100%', justifyContent: 'space-between' }}>
          {inDetail ? (
            <Space>
              <Button
                icon={<ArrowLeftOutlined />}
                onClick={() => setSelectedModelName(undefined)}
              >
                返回
              </Button>
              <Text strong>{selectedModelName} 的参数列表</Text>
            </Space>
          ) : (
            <Input.Search
              placeholder="搜索参数模型名称 / 描述"
              allowClear
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
              style={{ width: 280 }}
            />
          )}
          <Space>
            {/* 2026-05-29 用户决策:"导入 XML" = 用户选本地 XML 上传,落 host
                /opt/omc/data/param-mappings-custom/(升级不丢),作为自定义参数模型。
                旧的"上传 XML"按钮已合并进来 — 同一个 multipart upload 链路。 */}
            <Upload {...uploadProps}>
              <Button icon={<InboxOutlined />} loading={uploadMut.isPending}>
                导入 XML
              </Button>
            </Upload>
            <Popconfirm
              title="确认重载 XML?"
              description={
                <div style={{ maxWidth: 360 }}>
                  从 <code>datamodels/</code> <b>destructive 全量重载</b>:
                  <br />· 当前 XML 中的模型 → UPSERT (覆盖 UI 编辑)
                  <br />· DB 中已无 XML 对应的孤儿模型 → <b>删除</b>
                  <br />· 关联的 <code>param_mappings</code> 级联删除
                  <br />· 关联的 <code>products.param_model_id</code> 被置空 (SET NULL)
                  <br />操作不可撤销!
                </div>
              }
              okText="确认重载"
              cancelText="取消"
              okButtonProps={{ danger: true }}
              placement="bottomRight"
              onConfirm={() => {
                reloadMut
                  .mutateAsync()
                  .then((r) =>
                    message.success(
                      `已重载:${r.reloaded}${r.orphans_deleted > 0 ? ` (清理 ${r.orphans_deleted} 个孤儿模型)` : ''}`,
                    ),
                  )
                  .catch((e) => message.error((e as Error).message));
              }}
            >
              <Button icon={<CloudDownloadOutlined />} loading={reloadMut.isPending} danger>
                重载 XML
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
        </Space>
      </Card>

      {/* drill-down 主体:列表态 ↔ 详情态 二选一 */}
      {inDetail ? (
        <MappingsTab selectedName={selectedModelName} />
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
