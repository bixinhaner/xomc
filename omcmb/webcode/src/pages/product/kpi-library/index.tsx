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
import { useT } from '@/hooks/useT';

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
  const t = useT();
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

  // 2026-05-29 详情态分组筛选数据源 — 平台过滤,只显示当前 platform 涉及的分组,
  // 避免下拉里 23 个 group 中 ~16 个选了返空。selectedPlatform 不存在(列表态)时
  // 传 undefined → 后端返全量 23 个,本组件也不会用到(只渲在详情态)。
  const { data: groupData } = useIndicatorGroups(
    selectedDeviceType ?? 'ENB',
    OPERATOR_CODE,
    selectedPlatform || undefined,
  );
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
        {t('product.kpi.unitsDefinition')}
      </Button>
      <Button icon={<InboxOutlined />} onClick={() => setUploadOpen(true)}>
        {t('common.importXml')}
      </Button>
      <Popconfirm
        title={t('product.paramModel.reloadTitle')}
        description={
          <div style={{ maxWidth: 360 }}>
            {t('product.kpi.reloadDesc')}
            <br />{t('product.kpi.bullet.reloadUpsert')}
            
            <br />{t('product.kpi.bullet.reloadCascade')}
            <br />{t('common.actionUndoable')}
          </div>
        }
        okText={t('product.products.reloadOk')}
        cancelText={t('common.cancel')}
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
                  ? t('product.kpi.reloadOrphansSummary', { total, enb: orphans?.enb ?? 0, gsm: orphans?.gsm ?? 0, gnb: orphans?.gnb ?? 0 })
                  : t('product.kpi.reloadNoOrphans')
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
          {t('common.reloadXml')}
        </Button>
      </Popconfirm>
      <Button
        icon={<ReloadOutlined />}
        loading={cacheMut.isPending}
        onClick={() =>
          cacheMut
            .mutateAsync()
            .then((r) => message.success(r.note ? r.note : t('common.cacheRefreshed')))
            .catch((e) => message.error((e as Error).message))
        }
      >
        {t('common.refreshCache')}
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
                  {t('common.back')}
                </Button>
                <span style={{ fontWeight: 600 }}>
                  {TECH_LABEL[selectedTech]} · {selectedPlatform}
                </span>
              </Space>
              <Space wrap>
                {/* 2026-05-29 用户决策:搜索框放在分组筛选前面(主要操作前置) */}
                <Input.Search
                  placeholder={t('product.kpi.idNameSearchPh')}
                  allowClear
                  value={detailKeyword}
                  onChange={(e) => setDetailKeyword(e.target.value)}
                  style={{ width: 240 }}
                />
                <Select
                  placeholder={t('product.kpi.groupFilterPh')}
                  allowClear
                  value={detailGroupId}
                  onChange={(v) => setDetailGroupId(v)}
                  options={groupOptions}
                  style={{ width: 180 }}
                  showSearch
                  optionFilterProp="label"
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
