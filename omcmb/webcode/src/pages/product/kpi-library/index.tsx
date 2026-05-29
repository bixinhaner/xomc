/**
 * KpiLibraryPage — T-0180 KPI 指标库主页面(2026-05-29 第三轮调整)。
 *
 * 用户决策(本轮):
 *   1. "单位定义"靠右对齐(与全局操作按钮一起)
 *   2. 一级列表以 (平台, 制式) 唯一作为一行(粒度调整);点击平台 → 进入该平台详情;
 *      删除"管理 XML 文件"按钮;取消"按平台筛选"
 *   3. SummaryTab "XML 文件"列改为 XML 文件名(后端返 loaded_from,前端展 basename)
 *   4. "说明"列在最后,根据 (制式 + 平台) 简洁拼接
 *   5. (留空)
 *
 * URL 状态:?tech=enb&platform=ALL — F5 留二级 + 平台粒度
 */
import { useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { Card, Button, Space, Popconfirm, message, Select, Input } from 'antd';
import {
  ArrowLeftOutlined,
  AppstoreOutlined,
  CloudDownloadOutlined,
  InboxOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import {
  useIndicatorImportDirectory,
  useIndicatorCacheRefresh,
  useIndicatorGroups,
} from '@core/hooks/api/useIndicatorsLibrary';
import type { DeviceType, TechLower } from '@core/types/indicatorLibrary';
import { techToDeviceType } from '@core/types/indicatorLibrary';
import SummaryTab from './SummaryTab';
import IndicatorsByTech from './IndicatorsByTech';
import UploadXmlModal from './UploadXmlModal';
import UnitsDrawer from './UnitsDrawer';

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
  const selectedPlatform = searchParams.get('platform') || undefined;
  const inDetail = Boolean(selectedTech && selectedPlatform);
  const selectedDeviceType: DeviceType | undefined = selectedTech
    ? techToDeviceType(selectedTech)
    : undefined;

  const setTechPlatform = (tech: TechLower | undefined, platform?: string) => {
    const params = new URLSearchParams(searchParams);
    if (tech) {
      params.set('tech', tech);
    } else {
      params.delete('tech');
    }
    if (platform) {
      params.set('platform', platform);
    } else {
      params.delete('platform');
    }
    setSearchParams(params, { replace: false });
  };

  // ── 详情态 filter 状态(无 platform — 已由 URL 锁定) ───────────────────────
  const [detailKeyword, setDetailKeyword] = useState('');
  const [detailGroupId, setDetailGroupId] = useState<string | undefined>(undefined);

  const onSelectPlatform = (tech: TechLower, platform: string) => {
    setDetailKeyword('');
    setDetailGroupId(undefined);
    setTechPlatform(tech, platform);
  };

  const onBack = () => {
    setDetailKeyword('');
    setDetailGroupId(undefined);
    setTechPlatform(undefined, undefined);
  };

  // ── 公共状态 ────────────────────────────────────────────────
  const [unitsOpen, setUnitsOpen] = useState(false);
  const [uploadOpen, setUploadOpen] = useState(false);

  const importMut = useIndicatorImportDirectory();
  const cacheMut = useIndicatorCacheRefresh();

  // 详情态分组筛选数据源
  const { data: groupData } = useIndicatorGroups(selectedDeviceType ?? 'ENB', OPERATOR_CODE);
  const groupOptions = useMemo(() => {
    if (!selectedDeviceType) return [];
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

  // 全局操作按钮(列表态 toolbar 右侧)
  const globalActions = (
    <Space wrap>
      <Button icon={<AppstoreOutlined />} onClick={() => setUnitsOpen(true)}>
        单位定义
      </Button>
      <Button icon={<InboxOutlined />} onClick={() => setUploadOpen(true)}>
        导入 XML
      </Button>
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
  );

  return (
    <div style={{ padding: 16 }}>
      {/* 顶部 toolbar:列表态左空右动作;详情态左[返回+制式·平台名]右[分组+搜索] */}
      <Card size="small" style={{ marginBottom: 12 }}>
        <Space style={{ width: '100%', justifyContent: 'space-between' }} wrap>
          {inDetail && selectedTech && selectedPlatform ? (
            <>
              <Space wrap>
                <Button icon={<ArrowLeftOutlined />} onClick={onBack}>
                  返回
                </Button>
                <span style={{ fontWeight: 600 }}>
                  {TECH_LABEL[selectedTech]} · {selectedPlatform}
                </span>
              </Space>
              <Space wrap>
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
                <Input.Search
                  placeholder="搜索 ID / 名称"
                  allowClear
                  value={detailKeyword}
                  onChange={(e) => setDetailKeyword(e.target.value)}
                  style={{ width: 240 }}
                />
              </Space>
            </>
          ) : (
            <>
              {/* 列表态左侧留空 */}
              <span />
              {globalActions}
            </>
          )}
        </Space>
      </Card>

      {/* 主区域 */}
      {inDetail && selectedDeviceType && selectedPlatform ? (
        <IndicatorsByTech
          deviceType={selectedDeviceType}
          filter={{
            keyword: detailKeyword,
            platform: selectedPlatform,
            groupId: detailGroupId,
          }}
        />
      ) : (
        <SummaryTab onSelect={onSelectPlatform} />
      )}

      {/* 抽屉与弹窗 */}
      <UnitsDrawer open={unitsOpen} onClose={() => setUnitsOpen(false)} />
      <UploadXmlModal open={uploadOpen} onClose={() => setUploadOpen(false)} />
    </div>
  );
}
