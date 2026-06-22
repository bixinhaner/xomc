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
import { useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { Card, Button, Space } from 'antd';
import {
  ArrowLeftOutlined,
  InboxOutlined,
  PlusOutlined,
} from '@ant-design/icons';
import type { DeviceType, TechLower } from '@core/types/indicatorLibrary';
import { techToDeviceType } from '@core/types/indicatorLibrary';
import SummaryTab from './SummaryTab';
import IndicatorsByTech from './IndicatorsByTech';
import UploadXmlModal from './UploadXmlModal';
import GroupsManageModal from './GroupsManageModal';
import IndicatorFormModal from './IndicatorFormModal';
import GroupTreeSelect from './GroupTreeSelect';
import SearchInput from '@/components/SearchInput';
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
  const [uploadOpen, setUploadOpen] = useState(false);
  // Issue #525:指标分组(功能集)维护 Modal(仅详情态工具栏可开)
  const [groupsOpen, setGroupsOpen] = useState(false);
  // Issue #535:指标新建 Modal(详情态工具栏「新建指标」打开 create 模式)
  const [createOpen, setCreateOpen] = useState(false);
  // 2026-06-03:一级 SummaryTab 搜索上提到 toolbar(与「导入 XML」同行)
  const [summaryQuery, setSummaryQuery] = useState('');

  // 2026-06-22:分组筛选改用 GroupTreeSelect(真·树形,可展开收起),
  // 数据源 useIndicatorGroups 内移到组件里,platform 过滤(只显示当前 platform 涉及
  // 的分组,避免下拉里 23 个 group 中 ~16 个选了返空)经 GroupTreeSelect props 透传。

  // 全局操作按钮(列表态 toolbar 右侧)。
  // 2026-06-03 用户决策:合并「导入 XML / 重载 XML / 刷新缓存」为单个「导入 XML」—
  // 后端上传端点内部已自动做 destructive 重载(删孤儿)+ 刷新缓存,前端无需再单独调。
  const globalActions = (
    <Space wrap>
      <Button icon={<InboxOutlined />} onClick={() => setUploadOpen(true)}>
        {t('common.importXml')}
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
                <SearchInput
                  placeholder={t('product.kpi.idNameSearchPh')}
                  allowClear
                  onSearch={(v) => setDetailKeyword(v.trim())}
                  style={{ width: 320 }}
                  enterButton
                />
                {selectedDeviceType ? (
                  <GroupTreeSelect
                    deviceType={selectedDeviceType}
                    operatorCode={OPERATOR_CODE}
                    platform={selectedPlatform || undefined}
                    value={detailGroupId}
                    onChange={(v) => setDetailGroupId(v)}
                    placeholder={t('product.kpi.groupFilterPh')}
                    allowClear
                    style={{ width: 220 }}
                  />
                ) : null}
                <Button onClick={() => setGroupsOpen(true)}>
                  {t('product.kpi.group.manage')}
                </Button>
                <Button
                  type="primary"
                  icon={<PlusOutlined />}
                  onClick={() => setCreateOpen(true)}
                >
                  {t('product.kpi.indicator.newIndicator')}
                </Button>
              </Space>
            </>
          ) : (
            <>
              {/* 列表态左侧:一级平台搜索框(与右侧「导入 XML」等操作同一行) */}
              <SearchInput
                placeholder={t('product.kpi.summary.searchPh')}
                allowClear
                onSearch={(v) => setSummaryQuery(v.trim())}
                style={{ width: 320 }}
                enterButton
              />
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
        <SummaryTab onSelect={onSelectPlatform} query={summaryQuery} />
      )}

      {/* 抽屉与弹窗 */}
      <UploadXmlModal open={uploadOpen} onClose={() => setUploadOpen(false)} />
      {selectedDeviceType && (
        <GroupsManageModal
          open={groupsOpen}
          onClose={() => setGroupsOpen(false)}
          deviceType={selectedDeviceType}
          operatorCode={OPERATOR_CODE}
        />
      )}
      {selectedDeviceType && (
        <IndicatorFormModal
          open={createOpen}
          onClose={() => setCreateOpen(false)}
          deviceType={selectedDeviceType}
          operatorCode={OPERATOR_CODE}
          // 详情态(selectedPlatform 非空)时透传→后端同事务写占位 formula；
          // 列表态不传（不写公式，保持向后兼容行为）。
          platform={selectedPlatform || undefined}
          indicator={null}
        />
      )}
    </div>
  );
}
