import { useState } from 'react';
import {
  Drawer,
  Descriptions,
  Tag,
  Button,
  Empty,
} from 'antd';
import { EditOutlined } from '@ant-design/icons';
import type { DeviceType, IndicatorInfo } from '@core/types/indicatorLibrary';
import { useT } from '@/hooks/useT';
import IndicatorFormModal from './IndicatorFormModal';
import PlatformFormulasSection from './PlatformFormulasSection';

interface Props {
  open: boolean;
  deviceType: DeviceType;
  indicator: IndicatorInfo | null;
  // 2026-06-03:启用状态由父组件按 default 启用桶(enabledSet)传入,
  // 与列表开关同源;不再用 indicator.isEnabled(未反映 default 桶)。
  enabled: boolean;
  onClose: () => void;
}

export default function IndicatorDrawer({ open, deviceType, indicator, enabled, onClose }: Props) {
  const t = useT();
  // Issue #535 收口:把「编辑基础信息」并进详情抽屉,列表操作列不再有第二个编辑按钮。
  const [editBasicOpen, setEditBasicOpen] = useState(false);

  return (
    <>
    <Drawer
      title={indicator ? t('product.kpi.indicatorDetailFull', { id: indicator.id, name: indicator.cnName || indicator.name }) : t('product.kpi.indicatorDetail')}
      placement="right"
      size={760}
      open={open}
      onClose={onClose}
      destroyOnHidden
      extra={
        indicator ? (
          <Button icon={<EditOutlined />} onClick={() => setEditBasicOpen(true)}>
            {t('common.edit')}
          </Button>
        ) : null
      }
    >
      {!indicator ? (
        <Empty />
      ) : (
        <>
          <Descriptions column={2} size="small" bordered style={{ marginBottom: 16 }}>
            <Descriptions.Item label="ID">{indicator.id}</Descriptions.Item>
            <Descriptions.Item label={t('product.kpi.indicator.deviceType')}>{indicator.deviceType}</Descriptions.Item>
            <Descriptions.Item label={t('common.cnName')}>{indicator.cnName || '—'}</Descriptions.Item>
            <Descriptions.Item label={t('common.enName')}>{indicator.enName || '—'}</Descriptions.Item>
            <Descriptions.Item label={t('product.kpi.indicator.group')}>{indicator.groupName || indicator.groupId || '—'}</Descriptions.Item>
            <Descriptions.Item label={t('common.unit')}>{indicator.unit || '—'}</Descriptions.Item>
            {/* #193:计数器类型只对原始计数器(isCounter)有意义 — 派生 KPI 在 XML 无
                dataType,落库 data_type=NULL,恒"—"会误导,故对 KPI 型隐藏该字段。 */}
            {indicator.isCounter && (
              <Descriptions.Item label={t('product.kpi.indicator.counterType')}>{indicator.counterType || '—'}</Descriptions.Item>
            )}
            <Descriptions.Item label={t('product.kpi.indicator.level')}>{indicator.indicatorLevel || '—'}</Descriptions.Item>
            {/* 2026-06-25:补齐与新建表单同源字段 — 指标类型(direct/formula) + 统计类型,
                之前详情态缺这两项导致用户「看不到自己刚建时填的内容」(user feedback #4)。
                指标类型按 isCounter 反推:true=直接采集 / false=公式计算,与表单 INDICATOR_TYPE_OPTIONS 一致。
                统计类型 i18n key 走 product.kpi.indicator.statisType.<value>;未配则 — 。 */}
            <Descriptions.Item label={t('product.kpi.indicator.typeLabel')}>
              {indicator.isCounter
                ? t('product.kpi.indicator.typeCounter')
                : t('product.kpi.indicator.typeKpi')}
            </Descriptions.Item>
            <Descriptions.Item label={t('product.kpi.indicator.statisTypeLabel')}>
              {indicator.statisType
                ? t(`product.kpi.indicator.statisType.${indicator.statisType}`)
                : '—'}
            </Descriptions.Item>
            <Descriptions.Item label={t('product.kpi.indicator.enabledTag')} span={2}>
              {enabled ? <Tag color="success">{t('product.kpi.indicator.enabledTag')}</Tag> : <Tag>{t('product.kpi.indicator.disabledTag')}</Tag>}
            </Descriptions.Item>
            {/* 2026-06-25:删除「编号公式 (arithmetic)」Descriptions.Item — 与下方
                「全平台公式」表区域语义重复(arithmetic 即 perf_formulas.formula 的同源数据,
                只是字段名不同),用户反馈 #4 多了这条。完整公式 CRUD 仍在下方 PlatformFormulasSection。 */}
            <Descriptions.Item label={t('common.description')} span={2}>
              {indicator.description || '—'}
            </Descriptions.Item>
          </Descriptions>

          {/* 2026-06-25:删除外层「全平台公式」strong 标题 —
              PlatformFormulasSection 内部第 215 行已自带同名 strong + 「新增公式」按钮,
              此处再放一个会出现两个完全一样的标题(用户反馈 #1)。 */}
          <PlatformFormulasSection deviceType={deviceType} indicatorId={indicator.id} />
        </>
      )}
    </Drawer>
    <IndicatorFormModal
      open={editBasicOpen}
      deviceType={deviceType}
      indicator={indicator}
      onClose={() => setEditBasicOpen(false)}
    />
    </>
  );
}
