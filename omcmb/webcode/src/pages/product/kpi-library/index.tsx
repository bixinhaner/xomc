/**
 * KpiLibraryPage — T-0180 P4 KPI 指标库主页面重设计。
 *
 * 用户决策(PRD §8 D1-D5 锁定):
 *   D1 不设保留名 custom 可覆盖内置同名
 *   D2 强制 platform 必填 + deviceType 与 ?tech= 一致
 *   D3 reload 简化(沿用 destructive 语义,Popconfirm danger 红按钮 + 不可撤销)
 *   D4 单位定义抽屉 width=720
 *   D5 详情平表 + 来源列(留给 XMLFilesModal 实现"按文件管理")
 *
 * UI 拓扑:
 *   一级 SummaryTab — 三制式 (ENB/GSM/GNB) 行,点击 drill-down
 *   二级 IndicatorsByTech — 指标 CRUD + 启用切换;toolbar 显[返回 + 制式名 + 管理 XML 文件]
 *   抽屉 UnitsDrawer — 单位定义(跨制式共享)
 *   弹窗 UploadXmlModal / XMLFilesModal
 *
 * URL 同步: ?tech=enb/gsm/gnb 让 F5 刷新仍停在二级,显式"返回"按钮才回一级。
 * 8 个 icon 对齐 param-model 风格(ArrowLeft / Inbox / CloudUpload / CloudDownload /
 *   Reload / Delete / Eye / Plus + AppstoreOutlined 管理 XML)。
 */
import { useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { Card, Button, Space, Popconfirm, message, Typography } from 'antd';
import {
  ArrowLeftOutlined,
  AppstoreOutlined,
  CloudDownloadOutlined,
  InboxOutlined,
  ReloadOutlined,
  UnorderedListOutlined,
} from '@ant-design/icons';
import {
  useIndicatorImportDirectory,
  useIndicatorCacheRefresh,
} from '@core/hooks/api/useIndicatorsLibrary';
import type { DeviceType, TechLower } from '@core/types/indicatorLibrary';
import { techToDeviceType } from '@core/types/indicatorLibrary';
import SummaryTab from './SummaryTab';
import IndicatorsByTech from './IndicatorsByTech';
import XMLFilesModal from './XMLFilesModal';
import UploadXmlModal from './UploadXmlModal';
import UnitsDrawer from './UnitsDrawer';

const { Text } = Typography;

const TECH_LABEL: Record<TechLower, string> = {
  enb: 'ENB (LTE)',
  gsm: 'GSM',
  gnb: 'GNB (5G NR)',
};

const VALID_TECHS: TechLower[] = ['enb', 'gsm', 'gnb'];

function parseTech(raw: string | null): TechLower | undefined {
  if (!raw) return undefined;
  const lower = raw.toLowerCase();
  return VALID_TECHS.includes(lower as TechLower) ? (lower as TechLower) : undefined;
}

export default function KpiLibraryPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const selectedTech = parseTech(searchParams.get('tech'));
  const inDetail = Boolean(selectedTech);
  const selectedDeviceType: DeviceType | undefined = selectedTech
    ? techToDeviceType(selectedTech)
    : undefined;

  const setTech = (next: TechLower | undefined) => {
    const params = new URLSearchParams(searchParams);
    if (next) {
      params.set('tech', next);
    } else {
      params.delete('tech');
    }
    setSearchParams(params, { replace: false });
  };

  // ── 公共状态 ────────────────────────────────────────────────
  const [unitsOpen, setUnitsOpen] = useState(false);
  const [uploadOpen, setUploadOpen] = useState(false);
  const [filesModalOpen, setFilesModalOpen] = useState(false);

  // 后端 mutations
  const importMut = useIndicatorImportDirectory();
  const cacheMut = useIndicatorCacheRefresh();

  return (
    <div style={{ padding: 16 }}>
      {/* 顶部 toolbar:列表态 vs 详情态 */}
      <Card size="small" style={{ marginBottom: 12 }}>
        <Space style={{ width: '100%', justifyContent: 'space-between' }} wrap>
          {inDetail && selectedTech ? (
            <Space wrap>
              <Button icon={<ArrowLeftOutlined />} onClick={() => setTech(undefined)}>
                返回
              </Button>
              <Text strong>{TECH_LABEL[selectedTech]} 的指标列表</Text>
              <Button
                icon={<UnorderedListOutlined />}
                onClick={() => setFilesModalOpen(true)}
              >
                管理 XML 文件
              </Button>
            </Space>
          ) : (
            <Space wrap>
              <Text strong>三制式聚合</Text>
            </Space>
          )}
          <Space wrap>
            {/* 单位定义按钮(列表态显示;详情态隐藏避免视觉噪声) */}
            {!inDetail && (
              <Button icon={<AppstoreOutlined />} onClick={() => setUnitsOpen(true)}>
                单位定义
              </Button>
            )}
            <Button icon={<InboxOutlined />} onClick={() => setUploadOpen(true)}>
              上传 XML
            </Button>
            {/* 导入 XML — 加法 UPSERT(mode=import) */}
            <Popconfirm
              title="确认导入 XML?"
              description={
                <div style={{ maxWidth: 320 }}>
                  从 <code>datamodels/</code> <b>加法 UPSERT</b> 当前 XML 文件:
                  <br />· 新增指标 → 插入
                  <br />· 已存在指标 → 更新
                  <br />· DB 中已无 XML 对应的孤儿 → <b>保留不删</b>
                </div>
              }
              okText="确认导入"
              cancelText="取消"
              placement="bottomRight"
              onConfirm={() => {
                importMut
                  .mutateAsync('import')
                  .then((r) => message.success(`已导入:${r.reloaded}`))
                  .catch((e) => message.error((e as Error).message));
              }}
            >
              <Button loading={importMut.isPending && importMut.variables === 'import'}>
                导入 XML
              </Button>
            </Popconfirm>
            {/* 重载 XML — destructive(mode=reload) */}
            <Popconfirm
              title="确认重载 XML?"
              description={
                <div style={{ maxWidth: 360 }}>
                  从 <code>datamodels/</code> <b>destructive 全量重载</b>:
                  <br />· 当前 XML 中的指标 → UPSERT(覆盖 UI 编辑)
                  <br />· DB 中已无 XML 对应的孤儿 → <b>删除</b>
                  <br />· 关联的公式 / 启用记录级联清理
                  <br />操作不可撤销!
                </div>
              }
              okText="确认重载"
              cancelText="取消"
              okButtonProps={{ danger: true }}
              placement="bottomRight"
              onConfirm={() => {
                importMut
                  .mutateAsync('reload')
                  .then((r) => {
                    const orphans = r.orphans;
                    const total = orphans
                      ? (orphans.enb || 0) + (orphans.gsm || 0) + (orphans.gnb || 0)
                      : 0;
                    message.success(
                      total > 0
                        ? `已重载,清理 ${total} 个孤儿(enb=${orphans?.enb ?? 0} gsm=${orphans?.gsm ?? 0} gnb=${orphans?.gnb ?? 0})`
                        : '已重载,无孤儿'
                    );
                  })
                  .catch((e) => message.error((e as Error).message));
              }}
            >
              <Button
                icon={<CloudDownloadOutlined />}
                loading={importMut.isPending && importMut.variables === 'reload'}
                danger
              >
                重载 XML
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
        </Space>
      </Card>

      {/* 主区域:列表态(SummaryTab)↔ 详情态(IndicatorsByTech) */}
      {inDetail && selectedDeviceType ? (
        <IndicatorsByTech deviceType={selectedDeviceType} />
      ) : (
        <SummaryTab onSelect={setTech} />
      )}

      {/* 抽屉与弹窗 */}
      <UnitsDrawer open={unitsOpen} onClose={() => setUnitsOpen(false)} />
      <UploadXmlModal open={uploadOpen} onClose={() => setUploadOpen(false)} />
      <XMLFilesModal
        open={filesModalOpen}
        tech={selectedTech}
        onClose={() => setFilesModalOpen(false)}
      />
    </div>
  );
}
