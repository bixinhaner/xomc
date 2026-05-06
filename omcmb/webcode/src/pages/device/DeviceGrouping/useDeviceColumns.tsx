import { useCallback, useMemo, useState } from 'react';
import { Button, Input, Tag, Tooltip } from 'antd';
import { CheckOutlined, CloseOutlined, EditOutlined } from '@ant-design/icons';
import StatusIndicator from '@/components/StatusIndicator';
import type { DataTableColumn } from '@/components/DataTable';
import type { Device, EngStatus } from '@core/types/device';

export interface UseDeviceColumnsOptions {
  onEditDevice: (device: Device) => void;
  t: (id: string, values?: Record<string, unknown>) => string;
}

const REMARK_LABEL_STORAGE_KEY = 'omc_grouping_remark_label';

const calculateOfflineDays = (lastOnlineTime: string): number => {
  if (!lastOnlineTime) return 0;
  const lastOnline = new Date(lastOnlineTime);
  const now = new Date();
  const diffMs = now.getTime() - lastOnline.getTime();
  return Math.max(0, Math.floor(diffMs / (1000 * 60 * 60 * 24)));
};

/**
 * 构建设备列表的列定义，并维护 Remark 表头自定义标签的本地状态。
 * 拆分自 DeviceListPanel 以保持单文件 ≤ 400 行（W2.C.2 / T-0054）。
 */
export function useDeviceColumns({ onEditDevice, t }: UseDeviceColumnsOptions) {
  // Remark 列头自定义标签
  const [remarkLabel, setRemarkLabel] = useState(() => {
    return localStorage.getItem(REMARK_LABEL_STORAGE_KEY) || t('device.remark');
  });
  const [editingRemark, setEditingRemark] = useState(false);
  const [remarkInput, setRemarkInput] = useState('');

  const handleRemarkLabelSave = useCallback(() => {
    const val = remarkInput.trim();
    if (!val) return;
    setRemarkLabel(val);
    setEditingRemark(false);
    localStorage.setItem(REMARK_LABEL_STORAGE_KEY, val);
  }, [remarkInput]);

  const handleRemarkLabelCancel = useCallback(() => {
    setEditingRemark(false);
  }, []);

  const remarkHeaderRender = useMemo(() => {
    if (editingRemark) {
      return (
        <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }} onClick={(e) => e.stopPropagation()}>
          <Input
            size="small"
            value={remarkInput}
            onChange={(e) => setRemarkInput(e.target.value)}
            onPressEnter={handleRemarkLabelSave}
            style={{ width: 100 }}
            maxLength={30}
            autoFocus
          />
          <CheckOutlined
            style={{ fontSize: 12, color: '#52c41a', cursor: 'pointer' }}
            onClick={handleRemarkLabelSave}
          />
          <CloseOutlined
            style={{ fontSize: 12, color: '#ff4d4f', cursor: 'pointer' }}
            onClick={handleRemarkLabelCancel}
          />
        </span>
      );
    }
    return (
      <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}>
        <Tooltip title={remarkLabel}>
          <span style={{ maxWidth: 80, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {remarkLabel}
          </span>
        </Tooltip>
        <EditOutlined
          style={{ fontSize: 12, color: '#8c8c8c', cursor: 'pointer' }}
          onClick={(e) => {
            e.stopPropagation();
            setRemarkInput(remarkLabel);
            setEditingRemark(true);
          }}
        />
      </span>
    );
  }, [editingRemark, remarkInput, remarkLabel, handleRemarkLabelSave, handleRemarkLabelCancel]);

  const columns = useMemo<DataTableColumn<Device>[]>(
    () => [
      {
        key: 'actions',
        title: t('table.operation'),
        dataIndex: 'id',
        width: 80,
        fixed: 'right',
        render: (_val, record) => (
          <Button
            type="link"
            size="small"
            onClick={() => onEditDevice(record)}
          >
            {t('common.edit')}
          </Button>
        ),
      },
      {
        key: 'connStatus',
        title: t('device.connStatus'),
        dataIndex: 'connStatus',
        width: 90,
        render: (val) => (
          <StatusIndicator
            status={val === 'online' ? 'online' : 'offline'}
            text={val === 'online' ? t('status.online') : t('status.offline')}
          />
        ),
      },
      {
        key: 'engStatus',
        title: t('device.installStatus'),
        dataIndex: 'engStatus',
        width: 100,
        render: (val) => {
          const engStatus = val as EngStatus;
          const statusMap: Record<EngStatus, { label: string; color: string }> = {
            commissioned: { label: t('device.engStatus.commissioned'), color: 'green' },
            uncommissioned: { label: t('device.engStatus.uncommissioned'), color: 'orange' },
            decommissioned: { label: t('device.engStatus.decommissioned'), color: 'red' },
          };
          const { label, color } = statusMap[engStatus] || { label: String(val), color: 'default' };
          return <Tag color={color}>{label}</Tag>;
        },
      },
      {
        key: 'sn',
        title: t('device.serialNumber'),
        dataIndex: 'sn',
        width: 150,
        mono: true,
      },
      { key: 'name', title: t('device.stationName'), dataIndex: 'name', width: 160, ellipsis: true },
      { key: 'macAddress', title: t('device.macAddress'), dataIndex: 'macAddress', width: 150, mono: true },
      { key: 'groupName', title: t('device.groupName'), dataIndex: 'groupName', width: 140, ellipsis: true },
      {
        // T-0027 D9：分组归属来源列（PRD §12.6）
        // 后端 device_group_members.source_type → camelCase Device.sourceType
        // 'manual' / 'rule' / undefined（旧数据未填）；undefined 视作 'manual'
        // 与 schema DEFAULT 一致（migration 000051）
        key: 'sourceType',
        title: t('device.sourceType'),
        dataIndex: 'sourceType',
        width: 100,
        render: (_val, record) => {
          const src = record.sourceType ?? 'manual';
          const color = src === 'rule' ? 'blue' : 'default';
          const label = t(`device.sourceType.${src}`);
          return <Tag color={color}>{label}</Tag>;
        },
      },
      { key: 'longitude', title: t('device.longitude'), dataIndex: 'longitude', width: 100 },
      { key: 'latitude', title: t('device.latitude'), dataIndex: 'latitude', width: 100 },
      { key: 'gpsHeight', title: t('device.height'), dataIndex: 'gpsHeight', width: 80 },
      {
        key: 'offlineDays',
        title: t('device.offlineDays'),
        width: 100,
        render: (_val, record) => {
          if (record.connStatus === 'online') return '-';
          return calculateOfflineDays(record.lastOnlineTime);
        },
      },
      {
        key: 'remark',
        title: remarkLabel,
        dataIndex: 'remark',
        width: 180,
        headerRender: remarkHeaderRender,
        ellipsis: true,
      },
    ],
    [t, onEditDevice, remarkLabel, remarkHeaderRender]
  );

  return columns;
}
