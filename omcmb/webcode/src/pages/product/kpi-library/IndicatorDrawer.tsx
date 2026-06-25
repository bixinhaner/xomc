import { useState } from 'react';
import {
  Drawer,
  Descriptions,
  Tag,
  Button,
  Space,
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
            <Descriptions.Item label={t('product.kpi.indicator.enabledTag')} span={2}>
              {enabled ? <Tag color="success">{t('product.kpi.indicator.enabledTag')}</Tag> : <Tag>{t('product.kpi.indicator.disabledTag')}</Tag>}
            </Descriptions.Item>
            {/* PM-P3:界面公式展示用编号版 arithmetic(对运维编号才是工作语言);
                标准名版 formula 退为工程内部物,见下方"全平台公式"表(保留 CRUD)。 */}
            <Descriptions.Item label={t('product.kpi.indicator.arithmetic')} span={2}>
              {indicator.arithmetic ? <code style={{ fontSize: 12 }}>{indicator.arithmetic}</code> : '—'}
            </Descriptions.Item>
            <Descriptions.Item label={t('common.description')} span={2}>
              {indicator.description || '—'}
            </Descriptions.Item>
          </Descriptions>

          <Space style={{ marginBottom: 12, justifyContent: 'space-between', width: '100%' }}>
            <strong>{t('product.kpi.indicator.formulasTitle')}</strong>
          </Space>

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
