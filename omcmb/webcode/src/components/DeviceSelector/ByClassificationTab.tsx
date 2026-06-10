import React, { useState } from 'react';
import { Button, Col, Row, Tree, Typography } from 'antd';
import type { DataNode } from 'antd/es/tree';
import type { Device } from '@core/types/device';
import { useThemeToken } from '@/hooks/useThemeToken';
import { useT } from '@/hooks/useT';

interface Dimension {
  key: string;
  /** i18n key for the dimension label, resolved at render via useT. */
  labelKey: string;
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

// Translator type for tree-node titles whose labels are enumerable (status dimensions).
type Translator = (id: string) => string;

const DIMENSIONS: Dimension[] = [
  {
    key: 'region',
    labelKey: 'deviceSelector.dim.region',
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
    labelKey: 'deviceSelector.dim.subnet',
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
    labelKey: 'deviceSelector.dim.networkType',
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
    key: 'productClass',
    labelKey: 'deviceSelector.dim.productClass',
    getTree: (devices) =>
      buildTree(
        devices.map((d) => d.productClass),
        'productClass'
      ),
    filterDevices: (devices, keys) => {
      const types = keys.map((k) => k.replace('productClass__', ''));
      return devices.filter((d) => types.includes(d.productClass));
    },
  },
  {
    key: 'vendor',
    labelKey: 'deviceSelector.dim.vendor',
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
    labelKey: 'deviceSelector.dim.connStatus',
    // getTree resolved with translator inside component (STATUS_TREE_BUILDERS).
    getTree: () => [],
    filterDevices: (devices, keys) => {
      const statuses = keys.map((k) => k.replace('connStatus__', ''));
      return devices.filter((d) => statuses.includes(d.connStatus));
    },
  },
  {
    key: 'engStatus',
    labelKey: 'deviceSelector.dim.engStatus',
    // getTree resolved with translator inside component (STATUS_TREE_BUILDERS).
    getTree: () => [],
    filterDevices: (devices, keys) => {
      const statuses = keys.map((k) => k.replace('engStatus__', ''));
      return devices.filter((d) => statuses.includes(d.engStatus));
    },
  },
  {
    key: 'site',
    labelKey: 'deviceSelector.dim.site',
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

// Status dimensions have a fixed enum of leaves whose titles need translation.
const STATUS_TREE_BUILDERS: Record<string, (t: Translator) => DataNode[]> = {
  connStatus: (t) => [
    { key: 'connStatus__online', title: t('device.online'), isLeaf: true },
    { key: 'connStatus__offline', title: t('device.offline'), isLeaf: true },
  ],
  engStatus: (t) => [
    { key: 'engStatus__commissioned', title: t('device.engStatus.commissioned'), isLeaf: true },
    { key: 'engStatus__uncommissioned', title: t('device.engStatus.uncommissioned'), isLeaf: true },
    { key: 'engStatus__decommissioned', title: t('device.engStatus.decommissioned'), isLeaf: true },
  ],
};

const ByClassificationTab: React.FC<ByClassificationTabProps> = ({
  devices,
  selectedSns,
  onSelectionChange,
}) => {
  const t = useT();
  const [activeDimension, setActiveDimension] = useState<string>('region');
  const [checkedKeys, setCheckedKeys] = useState<string[]>([]);
  const token = useThemeToken();

  const dimension = DIMENSIONS.find((d) => d.key === activeDimension) ?? DIMENSIONS[0];
  const statusTreeBuilder = STATUS_TREE_BUILDERS[dimension.key];
  const treeData = statusTreeBuilder ? statusTreeBuilder(t) : dimension.getTree(devices);
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
              {t(dim.labelKey)}
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
            {t('deviceSelector.matchedDevices')}
          </Typography.Text>
          <Typography.Text strong style={{ fontSize: 32, color: token.colorPrimary }}>
            {filteredDevices.length}
          </Typography.Text>
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            {t('deviceSelector.unit')}
          </Typography.Text>
          <Button
            type="primary"
            size="small"
            disabled={filteredDevices.length === 0}
            onClick={handleApply}
          >
            {t('deviceSelector.addToSelected')}
          </Button>
        </div>
      </Col>
    </Row>
  );
};

export default ByClassificationTab;
