import { useCallback, useMemo, useState } from 'react';
import { Input, Tag, Tooltip } from 'antd';
import { CheckOutlined, CloseOutlined, EditOutlined } from '@ant-design/icons';
import StatusIndicator from '@/components/StatusIndicator';
import type { DataTableColumn } from '@/components/DataTable';
import type { Device } from '@core/types/device';

export interface UseDeviceColumnsOptions {
  t: (id: string, values?: Record<string, string | number>) => string;
}

const REMARK_LABEL_STORAGE_KEY = 'omc_grouping_remark_label';

/**
 * 构建设备列表的列定义，并维护 Remark 表头自定义标签的本地状态。
 * 拆分自 DeviceListPanel 以保持单文件 ≤ 400 行（W2.C.2 / T-0054）。
 *
 * 列裁剪（按产品要求）：去掉「操作（编辑）」「安装状态」「经度」「纬度」「高度」
 * 「离线天数」列；保留 连接状态 / 基站编码 / 基站名称 / MAC / 设备分组 / 归属来源 / 备注。
 */
export function useDeviceColumns({ t }: UseDeviceColumnsOptions) {
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
        // 与「设备列表」页保持一致：表头用 'SN'（而非「基站编码」）。
        key: 'sn',
        title: 'SN',
        dataIndex: 'sn',
        width: 150,
        mono: true,
      },
      { key: 'name', title: t('device.stationName'), dataIndex: 'name', width: 160, ellipsis: true },
      {
        // 连接状态列放到「设备名称」之后（产品要求）。
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
      { key: 'macAddress', title: t('device.macAddress'), dataIndex: 'macAddress', width: 150, mono: true },
      { key: 'groupName', title: t('device.groupName'), dataIndex: 'groupName', width: 140, ellipsis: true },
      {
        // T-0027 D9：分组归属来源列（PRD §12.6）
        // 后端 device_group_members.source_type → camelCase Device.sourceType
        // 'manual' / 'rule' / 'auto' / undefined（旧接口未填）；undefined 不伪装成手工来源。
        key: 'sourceType',
        title: t('device.sourceType'),
        dataIndex: 'sourceType',
        width: 100,
        render: (_val, record) => {
          const src = record.sourceType;
          if (!src) return <Tag>-</Tag>;
          const color = src === 'rule' ? 'blue' : src === 'auto' ? 'green' : 'default';
          const label = t(`device.sourceType.${src}`);
          return <Tag color={color}>{label}</Tag>;
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
    [t, remarkLabel, remarkHeaderRender]
  );

  return columns;
}
