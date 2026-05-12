import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { DatePicker, Modal, Spin, Tree, Typography } from 'antd';
import type { Dayjs } from 'dayjs';
import dayjs from 'dayjs';
import { FolderOutlined } from '@ant-design/icons';
import type { DataNode, TreeProps } from 'antd/es/tree';
import { useT } from '@/hooks/useT';
import { useDeviceGroups } from '@core/hooks/api/useDevices';
import type { DeviceGroup } from '@core/types/device';

const { RangePicker } = DatePicker;
const { Text } = Typography;

export interface ExportParams {
  deviceGroupIds: string[];
  timeRange?: [string, string];
}

interface ExportModalProps {
  open: boolean;
  onClose: () => void;
  onConfirm: (params: ExportParams) => void;
  confirmLoading?: boolean;
}

// 构建树形数据，支持 checkable
function buildTreeData(
  groups: DeviceGroup[],
  checkedKeys: string[]
): DataNode[] {
  // 找出根节点：允许 null / undefined / 空字符串，或父节点不在当前可见集合内。
  // 后一种情况用于兼容权限裁剪后的残缺树，避免整棵树空白。
  const groupIdSet = new Set(groups.map((g) => g.id));
  const rootGroups = groups.filter((g) => !g.parentId || !groupIdSet.has(g.parentId));

  function buildNode(group: DeviceGroup): DataNode {
    const children = groups.filter((g) => g.parentId === group.id);
    const _isChecked = checkedKeys.includes(group.id);

    return {
      key: group.id,
      title: (
        <span>
          <FolderOutlined style={{ marginRight: 6, color: '#FA8C16' }} />
          {group.name}
          <Text type="secondary" style={{ fontSize: 11, marginLeft: 6 }}>
            ({group.deviceCount})
          </Text>
        </span>
      ),
      icon: null,
      children: children.length > 0 ? children.map(buildNode) : undefined,
    };
  }

  return rootGroups.map(buildNode);
}

// 获取所有节点的 key
function getAllGroupIds(groups: DeviceGroup[]): string[] {
  return groups.map((g) => g.id);
}

export default function ExportModal({ open, onClose, onConfirm, confirmLoading }: ExportModalProps) {
  const t = useT();
  const { data: groupsData, isLoading: groupsLoading } = useDeviceGroups();
  const deviceGroups = groupsData?.groups ?? [];
  const [checkedKeys, setCheckedKeys] = useState<string[]>([]);
  const [timeRange, setTimeRange] = useState<[Dayjs, Dayjs] | null>(null);

  // 重置状态
  useEffect(() => {
    if (!open) {
      setCheckedKeys([]);
      setTimeRange(null);
    }
  }, [open]);

  const handleCheck: TreeProps['onCheck'] = useCallback((checked, _info) => {
    // checked 可能是字符串数组或 { checked: string[], halfChecked: string[] }
    if (Array.isArray(checked)) {
      setCheckedKeys(checked as string[]);
    } else {
      setCheckedKeys((checked as { checked: string[] }).checked);
    }
  }, []);

  const handleOk = useCallback(() => {
    onConfirm({
      deviceGroupIds: checkedKeys,
      timeRange: timeRange ? [timeRange[0].toISOString(), timeRange[1].toISOString()] : undefined,
    });
  }, [onConfirm, checkedKeys, timeRange]);

  const handleCancel = useCallback(() => {
    setCheckedKeys([]);
    setTimeRange(null);
    onClose();
  }, [onClose]);

  const treeData = useMemo(
    () => buildTreeData(deviceGroups, checkedKeys),
    [deviceGroups, checkedKeys]
  );

  // 全选/取消全选
  const allGroupIds = useMemo(() => getAllGroupIds(deviceGroups), [deviceGroups]);
  const isAllChecked = deviceGroups.length > 0 && checkedKeys.length === allGroupIds.length;

  const handleCheckAll = useCallback((checked: boolean) => {
    if (checked) {
      setCheckedKeys(allGroupIds);
    } else {
      setCheckedKeys([]);
    }
  }, [allGroupIds]);

  return (
    <Modal
      title={t('export.title')}
      open={open}
      onOk={handleOk}
      onCancel={handleCancel}
      okText={t('export.startExport')}
      cancelText={t('common.cancel')}
      confirmLoading={confirmLoading}
      okButtonProps={{ disabled: checkedKeys.length === 0 || groupsLoading }}
      width={600}
      destroyOnClose
    >
      {/* 1. 设备组选择 - 树形结构 */}
      <div style={{ marginBottom: 16 }}>
        <div
          style={{
            marginBottom: 8,
            fontWeight: 'bold',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
          }}
        >
          <span>{t('export.selectDeviceGroup')}</span>
          <span
            style={{
              fontSize: 12,
              color: '#1677ff',
              cursor: 'pointer',
              fontWeight: 'normal',
            }}
            onClick={() => handleCheckAll(!isAllChecked)}
          >
            {isAllChecked ? t('common.unselectAll') : t('common.selectAll')}
          </span>
        </div>
        <Spin spinning={groupsLoading}>
          <div
            style={{
              border: '1px solid #d9d9d9',
              borderRadius: 6,
              padding: 8,
              maxHeight: 280,
              overflowY: 'auto',
              background: '#fafafa',
            }}
          >
            {deviceGroups.length === 0 && !groupsLoading ? (
              <div style={{ color: '#999', textAlign: 'center', padding: 20 }}>
                {t('common.noData')}
              </div>
            ) : (
              <Tree
                checkable
                checkedKeys={checkedKeys}
                onCheck={handleCheck}
                treeData={treeData}
                defaultExpandAll
                style={{ fontSize: 13, background: 'transparent' }}
              />
            )}
          </div>
        </Spin>
        <div style={{ marginTop: 4, color: '#666', fontSize: 12 }}>
          {t('export.selectedCount', { count: checkedKeys.length })}
        </div>
      </div>

      {/* 2. 故障时间段 */}
      <div>
        <div style={{ marginBottom: 8, fontWeight: 'bold' }}>{t('export.timeRange')}</div>
        <RangePicker
          showTime
          value={timeRange}
          onChange={(dates) => setTimeRange(dates as [Dayjs, Dayjs] | null)}
          style={{ width: '100%' }}
          placeholder={[t('export.startTime'), t('export.endTime')]}
          ranges={{
            [t('export.today')]: [dayjs().startOf('day'), dayjs().endOf('day')],
            [t('export.thisWeek')]: [dayjs().startOf('week'), dayjs().endOf('week')],
            [t('export.thisMonth')]: [dayjs().startOf('month'), dayjs().endOf('month')],
            [t('export.last7Days')]: [dayjs().subtract(7, 'days').startOf('day'), dayjs().endOf('day')],
            [t('export.last30Days')]: [dayjs().subtract(30, 'days').startOf('day'), dayjs().endOf('day')],
          }}
        />
      </div>
    </Modal>
  );
}
