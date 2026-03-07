import React, { useState } from 'react';
import { Button, Col, Row, Tree, Typography } from 'antd';
import type { DataNode } from 'antd/es/tree';
import type { Device } from '@/types/device';
import { useThemeToken } from '@/hooks/useThemeToken';

interface Dimension {
  key: string;
  label: string;
  getTree: (devices: Device[]) => DataNode[];
  filterDevices: (devices: Device[], selectedKeys: string[]) => Device[];
}

interface ByClassificationTabProps {
  devices: Device[];
  selectedSns: string[];
  onSelectionChange: (sns: string[]) => void;
}

const buildTree = (
  items: string[],
  parentKey: string
): DataNode[] =>
  [...new Set(items)].filter(Boolean).map((item) => ({
    key: `${parentKey}__${item}`,
    title: item,
    isLeaf: true,
  }));

const DIMENSIONS: Dimension[] = [
  {
    key: 'region',
    label: '行政区域',
    getTree: (devices) =>
      buildTree(
        devices.map((d) => d.region),
        'region'
      ),
    filterDevices: (devices, keys) => {
      const regions = keys.map((k) => k.replace('region__', ''));
      return devices.filter((d) => regions.includes(d.region));
    },
  },
  {
    key: 'subnet',
    label: '子网',
    getTree: (devices) =>
      buildTree(
        devices.map((d) => d.subnet),
        'subnet'
      ),
    filterDevices: (devices, keys) => {
      const subnets = keys.map((k) => k.replace('subnet__', ''));
      return devices.filter((d) => subnets.includes(d.subnet));
    },
  },
  {
    key: 'networkType',
    label: '网络类型',
    getTree: (devices) =>
      buildTree(
        devices.map((d) => d.networkType),
        'networkType'
      ),
    filterDevices: (devices, keys) => {
      const types = keys.map((k) => k.replace('networkType__', ''));
      return devices.filter((d) => types.includes(d.networkType));
    },
  },
  {
    key: 'productType',
    label: '产品类型',
    getTree: (devices) =>
      buildTree(
        devices.map((d) => d.productType),
        'productType'
      ),
    filterDevices: (devices, keys) => {
      const types = keys.map((k) => k.replace('productType__', ''));
      return devices.filter((d) => types.includes(d.productType));
    },
  },
  {
    key: 'vendor',
    label: '厂商',
    getTree: (devices) =>
      buildTree(
        devices.map((d) => d.vendor),
        'vendor'
      ),
    filterDevices: (devices, keys) => {
      const vendors = keys.map((k) => k.replace('vendor__', ''));
      return devices.filter((d) => vendors.includes(d.vendor));
    },
  },
  {
    key: 'connStatus',
    label: '连接状态',
    getTree: () => [
      { key: 'connStatus__online', title: '在线', isLeaf: true },
      { key: 'connStatus__offline', title: '离线', isLeaf: true },
    ],
    filterDevices: (devices, keys) => {
      const statuses = keys.map((k) => k.replace('connStatus__', ''));
      return devices.filter((d) => statuses.includes(d.connStatus));
    },
  },
  {
    key: 'engStatus',
    label: '工程状态',
    getTree: () => [
      { key: 'engStatus__commissioned', title: '已开通', isLeaf: true },
      { key: 'engStatus__uncommissioned', title: '未开通', isLeaf: true },
      { key: 'engStatus__decommissioned', title: '已退网', isLeaf: true },
    ],
    filterDevices: (devices, keys) => {
      const statuses = keys.map((k) => k.replace('engStatus__', ''));
      return devices.filter((d) => statuses.includes(d.engStatus));
    },
  },
  {
    key: 'site',
    label: '站点',
    getTree: (devices) =>
      buildTree(
        devices.map((d) => d.site),
        'site'
      ),
    filterDevices: (devices, keys) => {
      const sites = keys.map((k) => k.replace('site__', ''));
      return devices.filter((d) => sites.includes(d.site));
    },
  },
];

const ByClassificationTab: React.FC<ByClassificationTabProps> = ({
  devices,
  selectedSns,
  onSelectionChange,
}) => {
  const [activeDimension, setActiveDimension] = useState<string>('region');
  const [checkedKeys, setCheckedKeys] = useState<string[]>([]);
  const token = useThemeToken();

  const dimension = DIMENSIONS.find((d) => d.key === activeDimension) ?? DIMENSIONS[0];
  const treeData = dimension.getTree(devices);
  const filteredDevices = dimension.filterDevices(devices, checkedKeys);

  const handleApply = () => {
    const sns = filteredDevices.map((d) => d.sn);
    const merged = [...new Set([...selectedSns, ...sns])];
    onSelectionChange(merged);
  };

  return (
    <Row gutter={12} style={{ height: 400 }}>
      {/* Dimension buttons */}
      <Col span={6}>
        <div
          style={{
            border: `1px solid ${token.colorBorderSecondary}`,
          }}
        >
          {DIMENSIONS.map((dim) => (
            <button
              key={dim.key}
              onClick={() => {
                setActiveDimension(dim.key);
                setCheckedKeys([]);
              }}
              style={{
                display: 'block',
                width: '100%',
                textAlign: 'left',
                padding: '10px 12px',
                border: 'none',
                borderBottom: `1px solid ${token.colorBorderSecondary}`,
                background: activeDimension === dim.key ? token.controlItemBgActive : token.colorBgContainer,
                color: activeDimension === dim.key ? token.colorPrimary : token.colorText,
                fontWeight: activeDimension === dim.key ? 500 : 400,
                cursor: 'pointer',
                fontSize: 13,
              }}
            >
              {dim.label}
            </button>
          ))}
        </div>
      </Col>

      {/* Tree */}
      <Col span={12}>
        <div
          style={{
            border: `1px solid ${token.colorBorderSecondary}`,
            borderRadius: 6,
            padding: 8,
            height: '100%',
            overflow: 'auto',
          }}
        >
          <Tree
            checkable
            treeData={treeData}
            checkedKeys={checkedKeys}
            onCheck={(keys) => {
              setCheckedKeys(
                Array.isArray(keys) ? (keys as string[]) : (keys.checked as string[])
              );
            }}
          />
        </div>
      </Col>

      {/* Result count */}
      <Col span={6}>
        <div
          style={{
            border: `1px solid ${token.colorBorderSecondary}`,
            borderRadius: 6,
            padding: 12,
            height: '100%',
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            justifyContent: 'center',
            gap: 12,
          }}
        >
          <Typography.Text type="secondary" style={{ fontSize: 13 }}>
            符合条件的设备
          </Typography.Text>
          <Typography.Text strong style={{ fontSize: 32, color: token.colorPrimary }}>
            {filteredDevices.length}
          </Typography.Text>
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            台
          </Typography.Text>
          <Button
            type="primary"
            size="small"
            disabled={filteredDevices.length === 0}
            onClick={handleApply}
          >
            添加到已选
          </Button>
        </div>
      </Col>
    </Row>
  );
};

export default ByClassificationTab;
