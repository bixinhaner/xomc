import { useCallback, useMemo } from 'react';
import { Button, Checkbox, Input, List, Pagination, Select, Space, Tag, Typography } from 'antd';
import { SearchOutlined, UserAddOutlined, DeleteOutlined } from '@ant-design/icons';
import type { ConsoleDevice } from '../types';
import { STATUS_COLORS, PRODUCT_TYPE_OPTIONS } from '../types';
import { useThemeToken } from '@/hooks/useThemeToken';
import { useT } from '@/hooks/useT';

interface DeviceTreeProps {
  selectedDevices: ConsoleDevice[];
  filteredDevices: ConsoleDevice[];
  paginatedDevices: ConsoleDevice[];
  searchText: string;
  productTypeFilter: string;
  currentPage: number;
  isAllSelected: boolean;
  isIndeterminate: boolean;
  totalFiltered: number;
  totalPages: number;
  onSearchChange: (text: string) => void;
  onFilterChange: (filter: string) => void;
  onPageChange: (page: number) => void;
  onToggleDevice: (device: ConsoleDevice, selected: boolean) => void;
  onToggleSelectAll: (checked: boolean) => void;
  onRemoveDevice: (sn: string) => void;
  onClearSelection: () => void;
  onBatchInput: () => void;
}

export default function DeviceTree({
  selectedDevices,
  paginatedDevices,
  searchText,
  productTypeFilter,
  currentPage,
  isAllSelected,
  isIndeterminate,
  totalFiltered,
  totalPages,
  onSearchChange,
  onFilterChange,
  onPageChange,
  onToggleDevice,
  onToggleSelectAll,
  onRemoveDevice,
  onClearSelection,
  onBatchInput,
}: DeviceTreeProps) {
  const t = useT();
  const token = useThemeToken();

  // 按类型分组设备
  const devicesByType = useMemo(() => {
    const map = new Map<string, ConsoleDevice[]>();
    paginatedDevices.forEach((device) => {
      const devices = map.get(device.type) || [];
      devices.push(device);
      map.set(device.type, devices);
    });
    return map;
  }, [paginatedDevices]);

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      {/* 头部 */}
      <div style={{
        padding: '8px 12px',
        borderBottom: `1px solid ${token.colorBorderSecondary}`,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
      }}>
        <Space size={4}>
          <Typography.Text strong style={{ fontSize: 12 }}>{t('device.name')}</Typography.Text>
          <Tag style={{ fontSize: 10, margin: 0 }}>{selectedDevices.length}</Tag>
        </Space>
        <Button
          size="small"
          icon={<UserAddOutlined />}
          onClick={onBatchInput}
          style={{ fontSize: 12 }}
        >
          批量输入
        </Button>
      </div>

      {/* 搜索和筛选 */}
      <div style={{ padding: '8px 12px' }}>
        <Space wrap size="small" style={{ width: '100%' }}>
          <Input
            size="small"
            style={{ width: 100 }}
            placeholder={t('common.search')}
            prefix={<SearchOutlined />}
            value={searchText}
            onChange={(e) => onSearchChange(e.target.value)}
            allowClear
          />
          <Select
            size="small"
            style={{ width: 80 }}
            placeholder="类型"
            allowClear
            options={PRODUCT_TYPE_OPTIONS}
            value={productTypeFilter || undefined}
            onChange={(val) => onFilterChange(val ?? '')}
          />
        </Space>
      </div>

      {/* 全选 */}
      <div style={{
        padding: '4px 8px',
        borderBottom: `1px solid ${token.colorBorderSecondary}`,
        display: 'flex',
        alignItems: 'center',
      }}>
        <Checkbox
          checked={isAllSelected}
          indeterminate={isIndeterminate}
          onChange={(e) => onToggleSelectAll(e.target.checked)}
          disabled={totalFiltered === 0}
        >
          <span style={{ fontSize: 11 }}>
            {t('common.selectAll')} ({selectedDevices.filter((d) => paginatedDevices.some((p) => p.sn === d.sn)).length}/{totalFiltered})
          </span>
        </Checkbox>
      </div>

      {/* 设备列表 */}
      <div className="no-scrollbar" style={{ flex: 1, minHeight: 80, overflow: 'auto' }}>
        <List
          size="small"
          dataSource={paginatedDevices}
          renderItem={(device) => {
            const isSelected = selectedDevices.some((d) => d.sn === device.sn);
            return (
              <List.Item
                style={{
                  cursor: 'pointer',
                  background: isSelected ? token.colorPrimaryBg : 'transparent',
                  padding: '4px 12px',
                }}
                onClick={() => onToggleDevice(device, !isSelected)}
              >
                <div style={{ display: 'flex', alignItems: 'center', gap: 4, width: '100%' }}>
                  <Checkbox
                    checked={isSelected}
                    onChange={(e) => onToggleDevice(device, e.target.checked)}
                    onClick={(e) => e.stopPropagation()}
                  />
                  <div
                    style={{
                      width: 6,
                      height: 6,
                      borderRadius: '50%',
                      background: STATUS_COLORS[device.status] ?? '#d9d9d9',
                    }}
                  />
                  <div style={{ flex: 1, minWidth: 0, fontSize: 11 }}>
                    <div style={{ fontWeight: 500 }}>{device.sn}</div>
                    <div style={{
                      fontSize: 10,
                      color: '#8c8c8c',
                      overflow: 'hidden',
                      textOverflow: 'ellipsis',
                      whiteSpace: 'nowrap',
                    }}>
                      {device.name}
                    </div>
                  </div>
                  <Tag style={{ fontSize: 9, margin: 0 }}>{device.type}</Tag>
                </div>
              </List.Item>
            );
          }}
        />
      </div>

      {/* 分页 */}
      {totalPages > 1 && (
        <div style={{ padding: '4px 8px', textAlign: 'center' }}>
          <Pagination
            size="small"
            current={currentPage}
            pageSize={8}
            total={totalFiltered}
            onChange={onPageChange}
            showSizeChanger={false}
            simple
          />
        </div>
      )}

      {/* 已选设备 */}
      {selectedDevices.length > 0 && (
        <div style={{ borderTop: `1px solid ${token.colorBorderSecondary}` }}>
          <div style={{
            padding: '4px 8px',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            background: '#fafafa',
          }}>
            <Typography.Text strong style={{ fontSize: 11 }}>
              已选 ({selectedDevices.length})
            </Typography.Text>
            <Button
              size="small"
              danger
              type="text"
              icon={<DeleteOutlined />}
              onClick={onClearSelection}
              style={{ fontSize: 10 }}
            />
          </div>
          <div
            className="no-scrollbar"
            style={{ maxHeight: 60, overflow: 'auto', padding: '4px 8px' }}
          >
            {selectedDevices.map((d) => (
              <Tag
                key={d.sn}
                closable
                onClose={() => onRemoveDevice(d.sn)}
                style={{ margin: '2px 4px', fontSize: 10 }}
              >
                {d.type}-{d.sn}
              </Tag>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
