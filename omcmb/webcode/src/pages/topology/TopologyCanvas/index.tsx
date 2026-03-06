import { useState, useMemo } from 'react';
import { Button, Input, List, Select, Space, Tag, Tooltip, Typography, message } from 'antd';
import {
  SearchOutlined,
  ZoomInOutlined,
  ZoomOutOutlined,
  AimOutlined,
  FilterOutlined,
  ReloadOutlined,
  PartitionOutlined,
  TableOutlined,
} from '@ant-design/icons';
import MapPageLayout from '@/components/Layout/MapPageLayout';
import TopologyCanvas from '@/components/TopologyCanvas';
import { useTopoGraph } from '@/hooks/api/useTopology';
import type { TopoNode } from '@/types/topology';
import { useT } from '@/hooks/useT';

const LAYOUT_OPTIONS_KEYS = [
  { labelKey: 'topology.layout.force', value: 'force' },
  { labelKey: 'topology.layout.tree', value: 'tree' },
  { labelKey: 'topology.layout.circular', value: 'circular' },
  { labelKey: 'topology.layout.hierarchy', value: 'hierarchy' },
];

const NODE_STATUS_MAP_KEYS: Record<string, { color: string; key: string }> = {
  online: { color: 'success', key: 'status.online' },
  offline: { color: 'default', key: 'status.offline' },
  alarm: { color: 'error', key: 'status.failed' },
  maintenance: { color: 'warning', key: 'status.pending' },
};

const NODE_TYPE_COLORS: Record<string, string> = {
  eNB: 'blue',
  gNB: 'purple',
  CPE: 'cyan',
  eGW: 'orange',
  domain: 'green',
  router: 'geekblue',
  switch: 'volcano',
};

export default function TopologyCanvasPage() {
  const t = useT();
  const [searchValue, setSearchValue] = useState('');
  const [selectedNode, setSelectedNode] = useState<TopoNode | null>(null);
  const [layoutType, setLayoutType] = useState('force');
  const [nodeTypeFilter, setNodeTypeFilter] = useState<string>('');
  const [statusFilter, setStatusFilter] = useState<string>('');
  const [showLabels, setShowLabels] = useState(true);
  const [zoomLevel, setZoomLevel] = useState(1);

  const { data: graphData, refetch } = useTopoGraph();

  const nodes = graphData?.nodes ?? [];
  const edges = graphData?.edges ?? [];

  const filteredNodes = nodes.filter((n) => {
    const matchSearch = !searchValue || n.label.includes(searchValue) || (n.deviceSn ?? '').includes(searchValue);
    const matchType = !nodeTypeFilter || n.type === nodeTypeFilter;
    const matchStatus = !statusFilter || n.status === statusFilter;
    return matchSearch && matchType && matchStatus;
  });

  const filteredEdges = edges.filter((e) =>
    filteredNodes.some((n) => n.id === e.source) && filteredNodes.some((n) => n.id === e.target),
  );

  const nodeTypes = [...new Set(nodes.map((n) => n.type))];

  const leftPanel = (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <div style={{ padding: '12px 12px 8px', borderBottom: '1px solid #f0f0f0' }}>
        <Typography.Text strong style={{ fontSize: 13 }}>{t('common.search')}</Typography.Text>
      </div>
      <div style={{ padding: '8px 12px' }}>
        <Input
          size="small"
          placeholder={t('common.search')}
          prefix={<SearchOutlined />}
          value={searchValue}
          onChange={(e) => setSearchValue(e.target.value)}
          allowClear
        />
      </div>

      {/* Filters */}
      <div style={{ padding: '0 12px 8px' }}>
        <Select
          size="small"
          style={{ width: '100%', marginBottom: 6 }}
          placeholder={t('table.type')}
          allowClear
          value={nodeTypeFilter || undefined}
          onChange={(val) => setNodeTypeFilter(val ?? '')}
          options={nodeTypes.map((t) => ({ label: t, value: t }))}
        />
        <Select
          size="small"
          style={{ width: '100%' }}
          placeholder={t('table.status')}
          allowClear
          value={statusFilter || undefined}
          onChange={(val) => setStatusFilter(val ?? '')}
          options={Object.entries(NODE_STATUS_MAP_KEYS).map(([k, v]) => ({ label: t(v.key), value: k }))}
        />
      </div>

      {/* Stats */}
      <div style={{ padding: '4px 12px 8px', borderBottom: '1px solid #f0f0f0' }}>
        <Typography.Text style={{ fontSize: 11, color: '#8c8c8c' }}>
          {t('table.total')}: {filteredNodes.length} / {filteredEdges.length}
        </Typography.Text>
      </div>

      {/* Node list */}
      <div style={{ flex: 1, overflow: 'auto' }}>
        <List
          size="small"
          dataSource={filteredNodes}
          renderItem={(node) => (
            <List.Item
              style={{
                cursor: 'pointer',
                background: selectedNode?.id === node.id ? '#e6f4ff' : 'transparent',
                padding: '5px 12px',
              }}
              onClick={() => setSelectedNode(node)}
            >
              <div style={{ width: '100%' }}>
                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                  <Typography.Text style={{ fontSize: 12 }} ellipsis>{node.label}</Typography.Text>
                  <Tag color={NODE_STATUS_MAP_KEYS[node.status]?.color} style={{ fontSize: 10, margin: 0 }}>
                    {t(NODE_STATUS_MAP_KEYS[node.status]?.key)}
                  </Tag>
                </div>
                <div style={{ display: 'flex', gap: 4, marginTop: 2 }}>
                  <Tag color={NODE_TYPE_COLORS[node.type] ?? 'default'} style={{ fontSize: 10 }}>{node.type}</Tag>
                  {node.deviceSn && <span style={{ fontSize: 10, color: '#8c8c8c', fontFamily: 'monospace' }}>{node.deviceSn}</span>}
                </div>
              </div>
            </List.Item>
          )}
        />
      </div>
    </div>
  );

  return (
    <MapPageLayout panel={leftPanel} defaultPanelWidth={280}>
      <div style={{ position: 'relative', width: '100%', height: '100%' }}>
        <TopologyCanvas
          nodes={filteredNodes}
          edges={filteredEdges}
          height="100%"
          onNodeClick={(node) => setSelectedNode(node)}
        />

        {/* Floating toolbar */}
        <div
          style={{
            position: 'absolute',
            top: 16,
            right: 16,
            background: 'rgba(255,255,255,0.95)',
            borderRadius: 8,
            padding: 8,
            boxShadow: '0 2px 8px rgba(0,0,0,0.12)',
            display: 'flex',
            flexDirection: 'column',
            gap: 4,
            zIndex: 10,
          }}
        >
          <Tooltip title={t('common.view')} placement="left">
            <Button size="small" icon={<ZoomInOutlined />} onClick={() => setZoomLevel((v) => Math.min(v + 0.2, 3))} />
          </Tooltip>
          <Tooltip title={t('common.view')} placement="left">
            <Button size="small" icon={<ZoomOutOutlined />} onClick={() => setZoomLevel((v) => Math.max(v - 0.2, 0.3))} />
          </Tooltip>
          <Tooltip title={t('common.reset')} placement="left">
            <Button size="small" icon={<AimOutlined />} onClick={() => setZoomLevel(1)} />
          </Tooltip>
          <div style={{ borderTop: '1px solid #f0f0f0', margin: '2px 0' }} />
          <Tooltip title={t('common.view')} placement="left">
            <Button
              size="small"
              type={showLabels ? 'primary' : 'default'}
              icon={<TableOutlined />}
              onClick={() => setShowLabels((v) => !v)}
            />
          </Tooltip>
          <Tooltip title={t('common.search')} placement="left">
            <Button size="small" icon={<FilterOutlined />} onClick={() => void message.info(t('common.search'))} />
          </Tooltip>
          <div style={{ borderTop: '1px solid #f0f0f0', margin: '2px 0' }} />
          <Tooltip title={t('common.refresh')} placement="left">
            <Button size="small" icon={<ReloadOutlined />} onClick={() => void refetch()} />
          </Tooltip>
        </div>

        {/* Layout selector */}
        <div
          style={{
            position: 'absolute',
            bottom: 16,
            right: 16,
            background: 'rgba(255,255,255,0.95)',
            borderRadius: 8,
            padding: '8px 12px',
            boxShadow: '0 2px 8px rgba(0,0,0,0.12)',
            display: 'flex',
            alignItems: 'center',
            gap: 8,
            zIndex: 10,
          }}
        >
          <PartitionOutlined style={{ fontSize: 12, color: '#8c8c8c' }} />
          <Typography.Text style={{ fontSize: 12 }}>{t('table.type')}:</Typography.Text>
          <Space size={4}>
            {LAYOUT_OPTIONS_KEYS.map((opt) => (
              <Button
                key={opt.value}
                size="small"
                type={layoutType === opt.value ? 'primary' : 'default'}
                onClick={() => {
                  setLayoutType(opt.value);
                }}
                style={{ fontSize: 11 }}
              >
                {opt.value}
              </Button>
            ))}
          </Space>
        </div>

        {/* Selected node detail */}
        {selectedNode && (
          <div
            style={{
              position: 'absolute',
              top: 16,
              left: 16,
              background: 'rgba(255,255,255,0.97)',
              borderRadius: 8,
              padding: '10px 14px',
              boxShadow: '0 2px 8px rgba(0,0,0,0.12)',
              minWidth: 180,
              zIndex: 10,
            }}
          >
            <Typography.Text strong style={{ fontSize: 12, display: 'block', marginBottom: 4 }}>
              {selectedNode.label}
            </Typography.Text>
            <div style={{ fontSize: 11, color: '#595959' }}>
              <div>{t('table.type')}: <Tag color={NODE_TYPE_COLORS[selectedNode.type]}>{selectedNode.type}</Tag></div>
              <div>{t('table.status')}: <Tag color={NODE_STATUS_MAP_KEYS[selectedNode.status]?.color}>{t(NODE_STATUS_MAP_KEYS[selectedNode.status]?.key)}</Tag></div>
              {selectedNode.deviceSn && <div style={{ fontFamily: 'monospace', marginTop: 2 }}>SN: {selectedNode.deviceSn}</div>}
            </div>
            <Button
              size="small"
              style={{ marginTop: 6, fontSize: 11 }}
              onClick={() => setSelectedNode(null)}
            >
              {t('common.close')}
            </Button>
          </div>
        )}
      </div>
    </MapPageLayout>
  );
}
