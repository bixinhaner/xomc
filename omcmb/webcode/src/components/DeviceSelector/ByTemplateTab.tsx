import React, { useMemo } from 'react';
import { Button, Table, Tag, Typography } from 'antd';
import type { TableProps } from 'antd';
import { useT } from '@/hooks/useT';

interface DeviceTemplate {
  id: string;
  name: string;
  description: string;
  deviceType: string;
  count: number;
  sns: string[];
}

interface ByTemplateTabProps {
  selectedSns: string[];
  onSelectionChange: (sns: string[]) => void;
}

// Predefined template definitions (in production these would come from API).
// Display labels resolved via i18n inside the component; deviceType holds either
// a literal product label (eNB/gNB) or an i18n key for the generic "All" type.
interface PresetTemplateDef {
  id: string;
  nameKey: string;
  descKey: string;
  /** When set, deviceType is resolved via this i18n key; otherwise deviceTypeLiteral is shown. */
  deviceTypeKey?: string;
  deviceTypeLiteral?: string;
}

const PRESET_TEMPLATE_DEFS: PresetTemplateDef[] = [
  {
    id: 'tpl_all_online',
    nameKey: 'deviceSelector.tpl.allOnline',
    descKey: 'deviceSelector.tpl.allOnlineDesc',
    deviceTypeKey: 'deviceSelector.tpl.typeAll',
  },
  {
    id: 'tpl_all_enb',
    nameKey: 'deviceSelector.tpl.allEnb',
    descKey: 'deviceSelector.tpl.allEnbDesc',
    deviceTypeLiteral: 'eNB',
  },
  {
    id: 'tpl_all_gnb',
    nameKey: 'deviceSelector.tpl.allGnb',
    descKey: 'deviceSelector.tpl.allGnbDesc',
    deviceTypeLiteral: 'gNB',
  },
  {
    id: 'tpl_alarm_devices',
    nameKey: 'deviceSelector.tpl.alarmDevices',
    descKey: 'deviceSelector.tpl.alarmDevicesDesc',
    deviceTypeKey: 'deviceSelector.tpl.typeAll',
  },
  {
    id: 'tpl_unmanaged',
    nameKey: 'deviceSelector.tpl.unmanaged',
    descKey: 'deviceSelector.tpl.unmanagedDesc',
    deviceTypeKey: 'deviceSelector.tpl.typeAll',
  },
];

const ByTemplateTab: React.FC<ByTemplateTabProps> = ({
  selectedSns,
  onSelectionChange,
}) => {
  const t = useT();
  const [selectedTemplateId, setSelectedTemplateId] = React.useState<string | null>(null);

  const presetTemplates: DeviceTemplate[] = useMemo(
    () =>
      PRESET_TEMPLATE_DEFS.map((def) => ({
        id: def.id,
        name: t(def.nameKey),
        description: t(def.descKey),
        deviceType: def.deviceTypeKey ? t(def.deviceTypeKey) : def.deviceTypeLiteral ?? '',
        count: 0,
        sns: [],
      })),
    [t],
  );

  const handleApply = (template: DeviceTemplate) => {
    setSelectedTemplateId(template.id);
    // In production, this would trigger an API call to fetch SNs for the template
    // For now merge with existing selection using template's sns array
    const merged = [...new Set([...selectedSns, ...template.sns])];
    onSelectionChange(merged);
  };

  const columns: TableProps<DeviceTemplate>['columns'] = [
    {
      title: t('deviceSelector.tpl.colName'),
      dataIndex: 'name',
      key: 'name',
      render: (v: string, record) => (
        <span>
          <Typography.Text strong>{v}</Typography.Text>
          {selectedTemplateId === record.id && (
            <Tag color="blue" style={{ marginLeft: 8 }}>
              {t('deviceSelector.tpl.selectedTag')}
            </Tag>
          )}
        </span>
      ),
    },
    {
      title: t('common.description'),
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
    {
      title: t('deviceSelector.tpl.colDeviceType'),
      dataIndex: 'deviceType',
      key: 'deviceType',
      width: 100,
    },
    {
      title: t('common.operation'),
      key: 'action',
      width: 100,
      render: (_: unknown, record) => (
        <Button
          type="link"
          size="small"
          onClick={() => handleApply(record)}
        >
          {t('deviceSelector.tpl.apply')}
        </Button>
      ),
    },
  ];

  return (
    <div>
      <Typography.Text type="secondary" style={{ display: 'block', marginBottom: 12, fontSize: 13 }}>
        {t('deviceSelector.tpl.hint')}
      </Typography.Text>
      <Table<DeviceTemplate>
        rowKey="id"
        columns={columns}
        dataSource={presetTemplates}
        size="small"
        pagination={false}
        onRow={(record) => ({
          style: {
            background: selectedTemplateId === record.id ? '#e6f7ff' : undefined,
            cursor: 'pointer',
          },
          onClick: () => handleApply(record),
        })}
      />
    </div>
  );
};

export default ByTemplateTab;
