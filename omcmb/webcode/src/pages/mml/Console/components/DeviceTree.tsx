import { useCallback, useMemo } from 'react';
import { Button, Checkbox, Input, List, Pagination, Select, Space, Tag, Typography } from 'antd';
import { SearchOutlined, UserAddOutlined, DeleteOutlined } from '@ant-design/icons';
import type { ConsoleDevice } from '../types';
import { STATUS_COLORS } from '../types';
import { useThemeToken } from '@/hooks/useThemeToken';
import { useT } from '@/hooks/useT';
import { useDictionary } from '@/hooks/api/useSystem';

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
  const { data: productTypeDict } = useDictionary('product_type');

  // 产品类型选项（字典驱动）
  const productTypeOptions = useMemo(
    () => (productTypeDict?.sysDictionaryDetails ?? []).map((d) => ({ label: d.label, value: d.value })),
    [productTypeDict]
  );

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
      <div
        style={{
          padding: '10px 12px',
          borderBottom: `1px solid ${token.colorBorderSecondary}`,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          background: token.colorBgLayout,
        }}
      >
        <Space size={6}>
          <Typography.Text strong style={{ fontSize: 13 }}>{t('device.name')}</Typography.Text>
          <Tag
            style={{
              fontSize: 10,
              margin: 0,
              background: token.colorPrimaryBg,
              border: `1px solid ${token.colorPrimaryBorder}`,
              color: token.colorPrimary,
              borderRadius: 10,
              padding: '0 6px',
            }}
          >
            {selectedDevices.length}
          </Tag>
        </Space>
        <Button
          size="small"
          type="primary"
          ghost
          icon={<UserAddOutlined />}
          onClick={onBatchInput}
          style={{ fontSize: 11, borderRadius: 4 }}
        >
          批量输入
        </Button>
      </div>

      {/* 搜索和筛选 */}
      <div style={{ padding: '10px 12px', background: token.colorBgContainer }}>
        <Input
          size="small"
          style={{ marginBottom: 8, borderRadius: 4 }}
          placeholder="设备SN,设备名称"
          prefix={<SearchOutlined style={{ color: '#bfbfbf' }} />}
          value={searchText}
          onChange={(e) => onSearchChange(e.target.value)}
          allowClear
        />
        <Select
          size="small"
          style={{ width: '100%', borderRadius: 4 }}
          placeholder="产品类型"
          allowClear
          options={productTypeOptions}
          value={productTypeFilter || undefined}
          onChange={(val) => onFilterChange(val ?? '')}
        />
      </div>

      {/* 全选 */}
      <div
        style={{
          padding: '6px 12px',
          borderBottom: `1px solid ${token.colorBorderSecondary}`,
          display: 'flex',
          alignItems: 'center',
          background: token.colorBgContainer,
        }}
      >
        <Checkbox
          checked={isAllSelected}
          indeterminate={isIndeterminate}
          onChange={(e) => onToggleSelectAll(e.target.checked)}
          disabled={totalFiltered === 0}
        >
          <span style={{ fontSize: 11, color: token.colorTextSecondary }}>
            {t('common.selectAll')} ({selectedDevices.filter((d) => paginatedDevices.some((p) => p.sn === d.sn)).length}/{totalFiltered})
          </span>
        </Checkbox>
      </div>

      {/* 设备列表 */}
      <div
        className="no-scrollbar"
        style={{
          flex: 1,
          minHeight: 0,
          overflow: 'auto',
          background: token.colorBgContainer,
        }}
      >
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
                  padding: '6px 12px',
                  borderLeft: isSelected ? `3px solid ${token.colorPrimary}` : '3px solid transparent',
                  transition: 'all 0.15s ease',
                }}
                onClick={() => onToggleDevice(device, !isSelected)}
              >
                <div style={{ display: 'flex', alignItems: 'center', gap: 6, width: '100%' }}>
                  <Checkbox
                    checked={isSelected}
                    onChange={(e) => onToggleDevice(device, e.target.checked)}
                    onClick={(e) => e.stopPropagation()}
                  />
                  <div
                    style={{
                      width: 8,
                      height: 8,
                      borderRadius: '50%',
                      background: STATUS_COLORS[device.status] ?? '#d9d9d9',
                      boxShadow: `0 0 4px ${STATUS_COLORS[device.status] ?? '#d9d9d9'}`,
                    }}
                  />
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ fontWeight: 500, fontSize: 12 }}>{device.sn}</div>
                    <div
                      style={{
                        fontSize: 10,
                        color: token.colorTextSecondary,
                        overflow: 'hidden',
                        textOverflow: 'ellipsis',
                        whiteSpace: 'nowrap',
                      }}
                    >
                      {device.name}
                    </div>
                  </div>
                  <Tag
                    style={{
                      fontSize: 9,
                      margin: 0,
                      background: token.colorBgLayout,
                      border: 'none',
                      borderRadius: 4,
                    }}
                  >
                    {device.type}
                  </Tag>
                </div>
              </List.Item>
            );
          }}
        />
      </div>

      {/* 分页 */}
      {totalPages > 1 && (
        <div
          style={{
            padding: '8px 12px',
            textAlign: 'center',
            borderTop: `1px solid ${token.colorBorderSecondary}`,
            background: token.colorBgContainer,
          }}
        >
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
        <div
          style={{
            borderTop: `1px solid ${token.colorBorderSecondary}`,
            background: token.colorPrimaryBg,
          }}
        >
          <div
            style={{
              padding: '6px 12px',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
            }}
          >
            <Typography.Text
              strong
              style={{ fontSize: 11, color: token.colorPrimary }}
            >
              已选设备 ({selectedDevices.length})
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
            style={{
              maxHeight: 70,
              overflow: 'auto',
              padding: '4px 12px 8px',
            }}
          >
            {selectedDevices.map((d) => (
              <Tag
                key={d.sn}
                closable
                onClose={() => onRemoveDevice(d.sn)}
                style={{
                  margin: '2px 4px',
                  fontSize: 10,
                  background: token.colorBgContainer,
                  border: `1px solid ${token.colorBorder}`,
                  borderRadius: 4,
                }}
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
