import { useState, useMemo } from 'react';
import { Badge, Input, List, Space, Tag, Tree, Typography } from 'antd';
import { SearchOutlined, EnvironmentOutlined } from '@ant-design/icons';
import type { DataNode } from 'antd/es/tree';
import MapPageLayout from '@/components/Layout/MapPageLayout';
import GISMap from '@/components/GISMap';
import type { MapDevice } from '@/components/GISMap';
import { useSites, useDomainTree } from '@/hooks/api/useTopology';
import type { Site } from '@/types/topology';
import { useT } from '@/hooks/useT';

const MOCK_SITES: Site[] = [
  { id: '1', name: '北京朝阳站点01', domainId: 'bj-cy', address: '北京市朝阳区建国路88号', longitude: 116.46, latitude: 39.92, deviceCount: 3, status: 'active' },
  { id: '2', name: '北京海淀站点01', domainId: 'bj-hd', address: '北京市海淀区中关村大街1号', longitude: 116.31, latitude: 39.98, deviceCount: 2, status: 'active' },
  { id: '3', name: '北京朝阳站点02', domainId: 'bj-cy', address: '北京市朝阳区望京街道', longitude: 116.49, latitude: 40.00, deviceCount: 1, status: 'maintenance' },
  { id: '4', name: '上海浦东站点01', domainId: 'sh-pd', address: '上海市浦东新区张江高科技园区', longitude: 121.60, latitude: 31.21, deviceCount: 4, status: 'active' },
  { id: '5', name: '上海静安站点01', domainId: 'sh-ja', address: '上海市静安区南京西路1882号', longitude: 121.45, latitude: 31.23, deviceCount: 2, status: 'active' },
  { id: '6', name: '广州天河站点01', domainId: 'gz-th', address: '广州市天河区珠江新城', longitude: 113.33, latitude: 23.12, deviceCount: 3, status: 'active' },
  { id: '7', name: '深圳南山站点01', domainId: 'sz-ns', address: '深圳市南山区科技园', longitude: 113.93, latitude: 22.53, deviceCount: 2, status: 'active' },
];

const SITE_STATUS_MAP_KEYS: Record<string, { color: string; key: string }> = {
  active: { color: 'success', key: 'status.online' },
  inactive: { color: 'default', key: 'status.disabled' },
  maintenance: { color: 'warning', key: 'status.pending' },
};

const MOCK_DOMAIN_TREE: DataNode[] = [
  {
    title: '全国',
    key: 'cn',
    children: [
      {
        title: '北京',
        key: 'bj',
        children: [
          { title: '朝阳区', key: 'bj-cy' },
          { title: '海淀区', key: 'bj-hd' },
        ],
      },
      {
        title: '上海',
        key: 'sh',
        children: [
          { title: '浦东新区', key: 'sh-pd' },
          { title: '静安区', key: 'sh-ja' },
        ],
      },
      {
        title: '广东',
        key: 'gd',
        children: [
          { title: '广州市', key: 'gz-th' },
          { title: '深圳市', key: 'sz-ns' },
        ],
      },
    ],
  },
];

const STATUS_DEVICE_COUNT = {
  online: 12,
  offline: 2,
  warning: 3,
  error: 1,
};

export default function GISMapView() {
  const t = useT();
  const [searchValue, setSearchValue] = useState('');
  const [selectedDomain, setSelectedDomain] = useState<string>('');
  const [selectedSite, setSelectedSite] = useState<Site | null>(null);

  const { data: sitesData } = useSites({ domainId: selectedDomain || undefined });
  const { data: domainTree } = useDomainTree();
  void domainTree;

  const sites = (sitesData ?? MOCK_SITES) as Site[];

  const filteredSites = sites.filter((s) => {
    const matchDomain = !selectedDomain || selectedDomain === 'cn' || s.domainId === selectedDomain || s.domainId.startsWith(selectedDomain);
    const matchSearch = !searchValue || s.name.includes(searchValue) || s.address.includes(searchValue);
    return matchDomain && matchSearch;
  });

  const mapDevices: MapDevice[] = filteredSites.map((site) => ({
    lat: site.latitude,
    lng: site.longitude,
    name: site.name,
    status: site.status === 'active' ? 'online' : site.status === 'maintenance' ? 'warning' : 'offline',
    sn: site.id,
  }));

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
      <div style={{ padding: '0 12px 8px' }}>
        <Typography.Text style={{ fontSize: 12, color: '#8c8c8c' }}>
          {t('table.total')}: {filteredSites.length}
        </Typography.Text>
      </div>

      {/* Domain tree */}
      <div style={{ padding: '0 8px 4px', borderBottom: '1px solid #f0f0f0' }}>
        <Tree
          treeData={MOCK_DOMAIN_TREE}
          defaultExpandAll
          showLine
          onSelect={(keys) => {
            const key = keys[0] as string;
            setSelectedDomain(key ?? '');
          }}
          style={{ fontSize: 12 }}
        />
      </div>

      {/* Site list */}
      <div style={{ flex: 1, overflow: 'auto' }}>
        <List
          size="small"
          dataSource={filteredSites}
          renderItem={(site) => (
            <List.Item
              style={{
                cursor: 'pointer',
                background: selectedSite?.id === site.id ? '#e6f4ff' : 'transparent',
                padding: '6px 12px',
              }}
              onClick={() => setSelectedSite(site)}
            >
              <div style={{ width: '100%' }}>
                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                  <Space size={4}>
                    <EnvironmentOutlined style={{ color: 'var(--color-primary-600)', fontSize: 12 }} />
                    <Typography.Text style={{ fontSize: 12, fontWeight: 500 }}>{site.name}</Typography.Text>
                  </Space>
                  <Tag color={SITE_STATUS_MAP_KEYS[site.status]?.color} style={{ fontSize: 10, margin: 0 }}>
                    {t(SITE_STATUS_MAP_KEYS[site.status]?.key)}
                  </Tag>
                </div>
                <div style={{ fontSize: 11, color: '#8c8c8c', marginTop: 2 }}>{site.address}</div>
                <div style={{ fontSize: 11, color: '#8c8c8c' }}>{t('table.total')}: {site.deviceCount}</div>
              </div>
            </List.Item>
          )}
        />
      </div>
    </div>
  );

  return (
    <MapPageLayout panel={leftPanel} defaultPanelWidth={300}>
      <div style={{ position: 'relative', width: '100%', height: '100%' }}>
        <GISMap
          devices={mapDevices}
          height="100%"
          onDeviceClick={(device) => {
            const site = filteredSites.find((s) => s.id === device.sn);
            if (site) setSelectedSite(site);
          }}
        />

        {/* Floating stats panel */}
        <div
          style={{
            position: 'absolute',
            bottom: 24,
            right: 24,
            background: 'rgba(255,255,255,0.95)',
            backdropFilter: 'blur(8px)',
            borderRadius: 8,
            padding: '12px 16px',
            boxShadow: '0 4px 12px rgba(0,0,0,0.12)',
            minWidth: 200,
            border: '1px solid #f0f0f0',
            zIndex: 10,
          }}
        >
          <Typography.Text strong style={{ fontSize: 12, display: 'block', marginBottom: 8 }}>
            {t('table.status')}
          </Typography.Text>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 12 }}>
              <Badge status="success" text={t('status.online')} />
              <Typography.Text strong style={{ color: '#52c41a' }}>{STATUS_DEVICE_COUNT.online}</Typography.Text>
            </div>
            <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 12 }}>
              <Badge status="warning" text={t('status.running')} />
              <Typography.Text strong style={{ color: '#fa8c16' }}>{STATUS_DEVICE_COUNT.warning}</Typography.Text>
            </div>
            <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 12 }}>
              <Badge status="error" text={t('status.failed')} />
              <Typography.Text strong style={{ color: '#f5222d' }}>{STATUS_DEVICE_COUNT.error}</Typography.Text>
            </div>
            <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 12 }}>
              <Badge status="default" text={t('status.offline')} />
              <Typography.Text strong style={{ color: '#8c8c8c' }}>{STATUS_DEVICE_COUNT.offline}</Typography.Text>
            </div>
          </div>
          <div style={{ borderTop: '1px solid #f0f0f0', marginTop: 8, paddingTop: 6 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 12 }}>
              <Typography.Text type="secondary">{t('table.total')}</Typography.Text>
              <Typography.Text strong>
                {Object.values(STATUS_DEVICE_COUNT).reduce((a, b) => a + b, 0)}
              </Typography.Text>
            </div>
          </div>
        </div>

        {/* Selected site info card */}
        {selectedSite && (
          <div
            style={{
              position: 'absolute',
              top: 16,
              right: 16,
              background: 'rgba(255,255,255,0.97)',
              borderRadius: 8,
              padding: '12px 16px',
              boxShadow: '0 4px 12px rgba(0,0,0,0.12)',
              minWidth: 220,
              border: '1px solid #f0f0f0',
              zIndex: 10,
            }}
          >
            <Typography.Text strong style={{ fontSize: 13, display: 'block', marginBottom: 6 }}>
              {selectedSite.name}
            </Typography.Text>
            <div style={{ fontSize: 12, color: '#595959', marginBottom: 4 }}>{selectedSite.address}</div>
            <div style={{ fontSize: 12 }}>
              <span style={{ color: '#8c8c8c' }}>{t('table.site')}: </span>
              {selectedSite.latitude.toFixed(4)}, {selectedSite.longitude.toFixed(4)}
            </div>
            <div style={{ fontSize: 12 }}>
              <span style={{ color: '#8c8c8c' }}>{t('table.total')}: </span>
              {selectedSite.deviceCount}
            </div>
            <Tag color={SITE_STATUS_MAP_KEYS[selectedSite.status]?.color} style={{ marginTop: 6 }}>
              {t(SITE_STATUS_MAP_KEYS[selectedSite.status]?.key)}
            </Tag>
          </div>
        )}
      </div>
    </MapPageLayout>
  );
}
