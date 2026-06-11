import { useState, useEffect, useMemo } from 'react';
import { Button, Card, Col, Flex, Input, Pagination, Row, Select, Space, Statistic, Tag, Tooltip, Typography, message } from 'antd';
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
import { useTopoGraph } from '@core/hooks/api/useTopology';
import type { TopoNode, NodeType, NodeStatus } from '@core/types/topology';
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
  const [highlightedNodeId, setHighlightedNodeId] = useState<string | null>(null);
  const [layoutType, setLayoutType] = useState('force');
  const [nodeTypeFilter, setNodeTypeFilter] = useState<string>('');
  const [statusFilter, setStatusFilter] = useState<string>('');
  const [showLabels, setShowLabels] = useState(true);
  const [, setZoomLevel] = useState(1);
  const [limit, setLimit] = useState(500); // 默认 500 个节点

  // 使用服务端筛选：将 nodeType、status 和 limit 传递给 API
  const { data: graphData, refetch } = useTopoGraph({
    layoutType,
    nodeType: nodeTypeFilter as NodeType,
    status: statusFilter as NodeStatus,
    limit,
  });

  const nodes = graphData?.nodes ?? [];
  const edges = graphData?.edges ?? [];
  const statistics = graphData?.statistics;

  // 搜索仍在客户端进行（支持按标签或设备序列号搜索）
  const filteredNodes = useMemo(
    () => nodes.filter((n) => {
      const matchSearch = !searchValue || n.label.includes(searchValue) || (n.deviceSn ?? '').includes(searchValue);
      return matchSearch;
    }),
    [nodes, searchValue],
  );

  const filteredEdges = useMemo(
    () => edges.filter((e) =>
      filteredNodes.some((n) => n.id === e.source) && filteredNodes.some((n) => n.id === e.target),
    ),
    [edges, filteredNodes],
  );

  // 侧边栏分页：antd6 List 已废弃，改用受控的 Pagination + 手动切片复刻原 List
  // 的内置分页（pageSize 50 / 不显示 size 切换 / showTotal）。原先靠 key 重置
  // List 内部页码，现把「当前筛选键 + 页码」放进同一份 state，筛选键变化时在渲染中
  // 同步归 1（React 推荐的"渲染期按依赖调整 state"模式，避免 effect 内 setState）。
  const SIDEBAR_PAGE_SIZE = 50;
  const sidebarListKey = useMemo(
    () => `sidebar-${searchValue}-${nodeTypeFilter}-${statusFilter}`,
    [searchValue, nodeTypeFilter, statusFilter],
  );
  const [sidebarPaging, setSidebarPaging] = useState({ key: sidebarListKey, page: 1 });
  const sidebarPage = sidebarPaging.key === sidebarListKey ? sidebarPaging.page : 1;
  if (sidebarPaging.key !== sidebarListKey) {
    setSidebarPaging({ key: sidebarListKey, page: 1 });
  }
  const pagedNodes = useMemo(
    () => filteredNodes.slice((sidebarPage - 1) * SIDEBAR_PAGE_SIZE, sidebarPage * SIDEBAR_PAGE_SIZE),
    [filteredNodes, sidebarPage],
  );

  // 获取所有节点类型（用于筛选器选项）
  const nodeTypes = [...new Set(nodes.map((n) => n.type))];

  // 数据量警告提示
  useEffect(() => {
    if (nodes.length >= limit) {
      void message.warning(
        t('topology.largeDataWarning') ||
        `已加载 ${nodes.length} 个节点（达到上限），请使用筛选功能或增加限制来查看更多`
      );
    }
  }, [nodes.length, limit, t]);

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
          options={nodeTypes.map((type) => ({ label: type, value: type }))}
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

      {/* Node list —— antd6 List 已废弃：用 Flex 纵向容器 + map 复刻条目（条目自带
          padding，仅补 List split 默认的底部分隔线），分页改受控 Pagination。 */}
      <div style={{ flex: 1, overflow: 'auto' }}>
        <Flex vertical>
          {pagedNodes.map((node, idx) => (
            <div
              key={node.id}
              style={{
                cursor: 'pointer',
                background: highlightedNodeId === node.id ? '#e6f4ff' : 'transparent',
                padding: '5px 12px',
                borderBottom: idx === pagedNodes.length - 1 ? 'none' : '1px solid #f0f0f0',
              }}
              onClick={() => {
                const targetNode = nodes.find(n => n.id === node.id);
                if (targetNode) {
                  setSelectedNode(targetNode);
                  setHighlightedNodeId(node.id);
                }
              }}
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
            </div>
          ))}
        </Flex>
        {filteredNodes.length > 0 && (
          <div style={{ display: 'flex', justifyContent: 'flex-end', padding: '10px 12px' }}>
            <Pagination
              size="small"
              current={sidebarPage}
              pageSize={SIDEBAR_PAGE_SIZE}
              total={filteredNodes.length}
              showSizeChanger={false}
              showTotal={(total) => `${t('table.total')}: ${total}`}
              onChange={(page) => setSidebarPaging({ key: sidebarListKey, page })}
            />
          </div>
        )}
      </div>
    </div>
  );

  return (
    <MapPageLayout panel={leftPanel} defaultPanelWidth={280}>
      <div style={{ position: 'relative', width: '100%', height: '100%', display: 'flex', flexDirection: 'column' }}>
        {/* Statistics Panel */}
        {statistics && (
          <Card
            size="small"
            style={{
              margin: '0 0 12px 0',
              borderRadius: 8,
              boxShadow: '0 1px 4px rgba(0,0,0,0.08)',
            }}
            bodyStyle={{ padding: '12px 16px' }}
          >
            <Row gutter={16} align="middle">
              <Col span={4}>
                <Statistic
                  title={<span style={{ fontSize: 11, color: '#8c8c8c' }}>已加载/限制</span>}
                  value={`${nodes.length} / ${limit}`}
                  valueStyle={{ fontSize: 16, fontWeight: 600, color: nodes.length >= limit ? '#faad14' : '#1890ff' }}
                />
              </Col>
              <Col span={5}>
                <Statistic
                  title={<span style={{ fontSize: 11, color: '#8c8c8c' }}>{t('topology.stats.totalNodes')}</span>}
                  value={statistics.totalNodes}
                  valueStyle={{ fontSize: 16, fontWeight: 600, color: '#1890ff' }}
                />
              </Col>
              <Col span={5}>
                <Statistic
                  title={<span style={{ fontSize: 11, color: '#8c8c8c' }}>{t('topology.stats.onlineNodes')}</span>}
                  value={statistics.onlineNodes}
                  valueStyle={{ fontSize: 16, fontWeight: 600, color: '#52c41a' }}
                />
              </Col>
              <Col span={5}>
                <Statistic
                  title={<span style={{ fontSize: 11, color: '#8c8c8c' }}>{t('topology.stats.totalEdges')}</span>}
                  value={statistics.totalEdges}
                  valueStyle={{ fontSize: 16, fontWeight: 600, color: '#722ed1' }}
                />
              </Col>
              <Col span={5}>
                <Space size="small" style={{ display: 'flex', alignItems: 'center', height: '100%' }}>
                  <Select
                    size="small"
                    value={limit}
                    onChange={(val) => setLimit(val)}
                    style={{ width: 100 }}
                    options={[
                      { label: '100 节点', value: 100 },
                      { label: '500 节点', value: 500 },
                      { label: '1000 节点', value: 1000 },
                      { label: '2000 节点', value: 2000 },
                    ]}
                  />
                  <Typography.Text style={{ fontSize: 11, color: '#8c8c8c' }}>节点限制</Typography.Text>
                </Space>
              </Col>
            </Row>
            {/* Node type breakdown */}
            {statistics.nodeTypeCounts && Object.keys(statistics.nodeTypeCounts).length > 0 && (
              <div style={{ marginTop: 8, paddingTop: 8, borderTop: '1px solid #f0f0f0' }}>
                <Typography.Text style={{ fontSize: 11, color: '#8c8c8c', marginRight: 8 }}>
                  {t('topology.stats.nodeTypeBreakdown')}:
                </Typography.Text>
                {Object.entries(statistics.nodeTypeCounts).map(([type, count]) => (
                  <Tag key={type} color={NODE_TYPE_COLORS[type] ?? 'default'} style={{ fontSize: 10, marginBottom: 2 }}>
                    {type}: {count}
                  </Tag>
                ))}
              </div>
            )}
          </Card>
        )}
        <div style={{ flex: 1, position: 'relative' }}>
          <TopologyCanvas
          nodes={filteredNodes}
          edges={filteredEdges}
          height="100%"
          highlightedNodeId={highlightedNodeId}
          selectedNode={selectedNode}
          onNodeClick={(node) => {
            setSelectedNode(node);
            setHighlightedNodeId(node.id);
          }}
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
          <Typography.Text style={{ fontSize: 12 }}>{t('topology.settings.layoutAlgorithm')}:</Typography.Text>
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
                {t(opt.labelKey)}
              </Button>
            ))}
          </Space>
        </div>
        </div>
      </div>
    </MapPageLayout>
  );
}
