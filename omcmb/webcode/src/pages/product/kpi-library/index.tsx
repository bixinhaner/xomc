/**
 * KpiLibraryPage — T-0180 KPI 指标库主页面。
 *
 * 2026-05-29 用户调整(P4 后续):
 *   1. 删除"三制式聚合"文字标签
 *   2. "上传 XML" 与 "导入 XML" 合并 — "导入 XML" 触发自定义 XML 文件上传(原 Upload 行为);
 *      旧"导入 XML" mode=import(纯 UPSERT 不删孤儿) UI 入口下线;
 *      "重载 XML" 继续 mode=reload(系统内置 + custom XML 全量 + 删孤儿)
 *   3. SummaryTab 删除"操作"列(制式名 button.link 已可点击进入详情)
 *   4. 制式行加业务说明文字
 *   5. 详情页:
 *      a) 顶部 toolbar 增加 分组筛选 + 平台筛选 + 关键字搜索,与"返回"同一行
 *      b) 详情页表格加"平台"列 + "分组"列
 *      c) "详情"列改名为"操作",EyeOutlined → EditOutlined
 *      d) 详情态隐藏 [导入 XML][重载 XML][刷新缓存] — 这些是全局操作,仅在列表态显示
 *      e) 单位定义按钮仍在列表态可用
 */
import { useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { Card, Button, Space, Popconfirm, message, Typography, Select, Input } from 'antd';
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
  usePlatformList,
  useIndicatorGroups,
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

const OPERATOR_CODE = 'default';

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

  // ── 详情态 filter 状态(上提到 toolbar,与返回同一行) ──────────────────────
  const [detailKeyword, setDetailKeyword] = useState('');
  const [detailPlatform, setDetailPlatform] = useState<string | undefined>(undefined);
  const [detailGroupId, setDetailGroupId] = useState<string | undefined>(undefined);

  // 切换制式时重置详情过滤,避免上一制式的 group/platform 漏到下一制式
  const onSwitchTech = (next: TechLower | undefined) => {
    setDetailKeyword('');
    setDetailPlatform(undefined);
    setDetailGroupId(undefined);
    setTech(next);
  };

  // ── 公共状态 ────────────────────────────────────────────────
  const [unitsOpen, setUnitsOpen] = useState(false);
  const [uploadOpen, setUploadOpen] = useState(false);
  const [filesModalOpen, setFilesModalOpen] = useState(false);

  // 后端 mutations
  const importMut = useIndicatorImportDirectory();
  const cacheMut = useIndicatorCacheRefresh();

  // 详情态可用的过滤数据源(平台列表 + 分组树)
  const { data: platformData } = usePlatformList(selectedDeviceType ?? 'ENB');
  const platformOptions = useMemo(
    () =>
      (selectedDeviceType ? platformData?.items || [] : []).map((p) => ({ label: p, value: p })),
    [platformData, selectedDeviceType]
  );

  const { data: groupData } = useIndicatorGroups(selectedDeviceType ?? 'ENB', OPERATOR_CODE);
  const groupOptions = useMemo(() => {
    if (!selectedDeviceType) return [];
    // 扁平化分组树(本期只支持一级筛选,后续可加层级 indent)
    const flat: { label: string; value: string }[] = [];
    const walk = (nodes: { id: string; name: string; children?: typeof nodes }[] | undefined) => {
      (nodes || []).forEach((n) => {
        flat.push({ label: n.name || n.id, value: n.id });
        if (n.children) walk(n.children);
      });
    };
    walk(groupData?.items as never);
    return flat;
  }, [groupData, selectedDeviceType]);

  return (
    <div style={{ padding: 16 }}>
      {/* 顶部 toolbar:列表态 vs 详情态;详情态把过滤器与返回放在同一行 */}
      <Card size="small" style={{ marginBottom: 12 }}>
        <Space style={{ width: '100%', justifyContent: 'space-between' }} wrap>
          {inDetail && selectedTech ? (
            <Space wrap>
              <Button icon={<ArrowLeftOutlined />} onClick={() => onSwitchTech(undefined)}>
                返回
              </Button>
              <Text strong>{TECH_LABEL[selectedTech]} 的指标列表</Text>
              <Button
                icon={<UnorderedListOutlined />}
                onClick={() => setFilesModalOpen(true)}
              >
                管理 XML 文件
              </Button>
              <Select
                placeholder="按分组筛选"
                allowClear
                value={detailGroupId}
                onChange={(v) => setDetailGroupId(v)}
                options={groupOptions}
                style={{ width: 180 }}
                showSearch
                optionFilterProp="label"
              />
              <Select
                placeholder="按平台筛选"
                allowClear
                value={detailPlatform}
                onChange={(v) => setDetailPlatform(v)}
                options={platformOptions}
                style={{ width: 180 }}
                showSearch
                optionFilterProp="label"
              />
              <Input.Search
                placeholder="搜索 ID / 名称"
                allowClear
                value={detailKeyword}
                onChange={(e) => setDetailKeyword(e.target.value)}
                style={{ width: 240 }}
              />
            </Space>
          ) : (
            // 列表态不再显"三制式聚合"标签(用户决策);左侧仅留单位定义入口
            <Space wrap>
              <Button icon={<AppstoreOutlined />} onClick={() => setUnitsOpen(true)}>
                单位定义
              </Button>
            </Space>
          )}
          {/* 详情态隐藏全局操作按钮(导入/重载/刷新)— 仅列表态显示 */}
          {!inDetail && (
            <Space wrap>
              {/* 用户决策:删除独立"上传 XML"按钮,"导入 XML" 即触发自定义 XML 上传 */}
              <Button icon={<InboxOutlined />} onClick={() => setUploadOpen(true)}>
                导入 XML
              </Button>
              {/* 重载 XML — 系统内置 + custom 全量重新加载 + 删孤儿(destructive mode=reload) */}
              <Popconfirm
                title="确认重载 XML?"
                description={
                  <div style={{ maxWidth: 360 }}>
                    重新加载 <b>系统内置</b> + <b>自定义</b> 所有 XML:
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
          )}
        </Space>
      </Card>

      {/* 主区域:列表态 vs 详情态二选一 */}
      {inDetail && selectedDeviceType ? (
        <IndicatorsByTech
          deviceType={selectedDeviceType}
          filter={{
            keyword: detailKeyword,
            platform: detailPlatform,
            groupId: detailGroupId,
          }}
        />
      ) : (
        <SummaryTab onSelect={onSwitchTech} />
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
