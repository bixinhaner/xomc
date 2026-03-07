import React, { useCallback, useMemo, useState } from 'react';
import { Button, Drawer, Space, Tag, Typography } from 'antd';
import { EyeOutlined, SearchOutlined } from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useT } from '@/hooks/useT';

const { Text, Paragraph, Title } = Typography;

interface AlarmLibraryEntry {
  id: string;
  alarmCode: string;
  alarmName: string;
  severity: 'critical' | 'major' | 'minor' | 'warning';
  possibleCauses: string;
  handlingSuggestions: string;
  updateTime: string;
  neType: string;
  vendor: string;
}

const MOCK_LIBRARY: AlarmLibraryEntry[] = [
  {
    id: '1', alarmCode: 'ALM-0001', alarmName: 'CPU占用率超阈值',
    severity: 'major', neType: 'eNB', vendor: '华为',
    possibleCauses: '1. 系统负载过高\n2. 进程异常死循环\n3. 内存泄漏导致CPU占用上升',
    handlingSuggestions: '1. 检查系统进程，终止异常进程\n2. 重启相关服务\n3. 如持续发生，考虑升级硬件或优化配置',
    updateTime: '2024-02-01',
  },
  {
    id: '2', alarmCode: 'ALM-0002', alarmName: '设备断连告警',
    severity: 'critical', neType: 'gNB', vendor: '华为',
    possibleCauses: '1. 网络链路中断\n2. 设备断电\n3. 配置错误导致连接失败',
    handlingSuggestions: '1. 检查网络连接和电源\n2. 检查防火墙规则\n3. 验证管理IP配置是否正确',
    updateTime: '2024-02-05',
  },
  {
    id: '3', alarmCode: 'ALM-0003', alarmName: '温度过高告警',
    severity: 'major', neType: 'CPE', vendor: '中兴',
    possibleCauses: '1. 环境温度过高\n2. 散热风扇故障\n3. 设备长期高负载运行',
    handlingSuggestions: '1. 检查机房温度，开启空调降温\n2. 检查并更换散热风扇\n3. 降低设备工作负载',
    updateTime: '2024-02-08',
  },
  {
    id: '4', alarmCode: 'ALM-0004', alarmName: '光模块接收功率低',
    severity: 'minor', neType: 'eNB', vendor: '爱立信',
    possibleCauses: '1. 光纤弯折损耗\n2. 光模块老化\n3. 光纤连接器污染',
    handlingSuggestions: '1. 检查光纤路由，避免弯折\n2. 清洁光纤连接器\n3. 如光模块老化严重，及时更换',
    updateTime: '2024-02-10',
  },
  {
    id: '5', alarmCode: 'ALM-0005', alarmName: '磁盘空间不足',
    severity: 'warning', neType: 'eGW', vendor: '华为',
    possibleCauses: '1. 日志文件未定期清理\n2. 核心转储文件积累\n3. 业务数据增长过快',
    handlingSuggestions: '1. 定期清理日志文件\n2. 配置日志自动归档策略\n3. 扩容存储空间',
    updateTime: '2024-02-12',
  },
  {
    id: '6', alarmCode: 'ALM-0006', alarmName: '链路丢包率异常',
    severity: 'major', neType: 'gNB', vendor: '中兴',
    possibleCauses: '1. 网络拥塞\n2. 物理链路质量差\n3. QoS配置不当',
    handlingSuggestions: '1. 检查网络流量，优化路由\n2. 检查物理链路，替换损坏线缆\n3. 调整QoS配置',
    updateTime: '2024-02-15',
  },
  {
    id: '7', alarmCode: 'ALM-0007', alarmName: '软件版本不匹配',
    severity: 'warning', neType: 'eNB', vendor: '大唐',
    possibleCauses: '1. 软件升级后版本兼容性问题\n2. 配置文件版本不一致',
    handlingSuggestions: '1. 检查软件版本兼容性列表\n2. 升级或回退至兼容版本\n3. 重新下发配置',
    updateTime: '2024-02-18',
  },
  {
    id: '8', alarmCode: 'ALM-0008', alarmName: '内存使用率超阈值',
    severity: 'major', neType: 'eGW', vendor: '京信',
    possibleCauses: '1. 内存泄漏\n2. 业务并发量过大\n3. 缓存配置不合理',
    handlingSuggestions: '1. 重启相关服务释放内存\n2. 检查内存泄漏点并修复\n3. 优化缓存配置',
    updateTime: '2024-02-20',
  },
];

const SEVERITY_TAG_COLOR: Record<string, string> = {
  critical: 'red', major: 'orange', minor: 'gold', warning: 'blue',
};

export default function AlarmSupportLibrary() {
  const t = useT();
  const [filterParams, setFilterParams] = useState<Record<string, unknown>>({});
  const [currentPage, setCurrentPage] = useState(1);
  const [drawerEntry, setDrawerEntry] = useState<AlarmLibraryEntry | null>(null);

  const SEVERITY_LABEL: Record<string, string> = useMemo(() => ({
    critical: t('alarm.severity.critical'),
    major: t('alarm.severity.major'),
    minor: t('alarm.severity.minor'),
    warning: t('alarm.severity.warning'),
  }), [t]);

  const FILTER_FIELDS: FilterField[] = useMemo(() => [
    { name: 'alarmCode', label: t('alarm.code'), type: 'input' },
    { name: 'alarmName', label: t('alarm.name'), type: 'input' },
    {
      name: 'severity',
      label: t('alarm.severity'),
      type: 'select',
      options: [
        { label: t('alarm.severity.critical'), value: 'critical' },
        { label: t('alarm.severity.major'), value: 'major' },
        { label: t('alarm.severity.minor'), value: 'minor' },
        { label: t('alarm.severity.warning'), value: 'warning' },
      ],
    },
    {
      name: 'neType',
      label: t('alarm.neType'),
      type: 'select',
      options: [
        { label: 'eNB', value: 'eNB' },
        { label: 'gNB', value: 'gNB' },
        { label: 'CPE', value: 'CPE' },
        { label: 'eGW', value: 'eGW' },
      ],
    },
    {
      name: 'vendor',
      label: t('device.vendor'),
      type: 'select',
      options: [
        { label: '华为', value: '华为' },
        { label: '中兴', value: '中兴' },
        { label: '爱立信', value: '爱立信' },
        { label: '大唐', value: '大唐' },
        { label: '京信', value: '京信' },
      ],
    },
  ], [t]);

  const filteredData = useMemo(() => {
    return MOCK_LIBRARY.filter((entry) => {
      if (filterParams.alarmCode && !entry.alarmCode.toLowerCase().includes(String(filterParams.alarmCode).toLowerCase())) return false;
      if (filterParams.alarmName && !entry.alarmName.toLowerCase().includes(String(filterParams.alarmName).toLowerCase())) return false;
      if (filterParams.severity && entry.severity !== filterParams.severity) return false;
      if (filterParams.neType && entry.neType !== filterParams.neType) return false;
      if (filterParams.vendor && entry.vendor !== filterParams.vendor) return false;
      return true;
    });
  }, [filterParams]);

  const handleSearch = useCallback((values: Record<string, unknown>) => {
    setFilterParams(values);
    setCurrentPage(1);
  }, []);

  const handleReset = useCallback(() => {
    setFilterParams({});
    setCurrentPage(1);
  }, []);

  const columns = useMemo(
    (): DataTableColumn<AlarmLibraryEntry>[] => [
      {
        key: 'alarmCode',
        title: t('alarm.code'),
        dataIndex: 'alarmCode',
        width: 110,
        mono: true,
        render: (v) => <Text style={{ fontFamily: 'monospace', fontSize: 12, fontWeight: 600 }}>{String(v)}</Text>,
      },
      { key: 'alarmName', title: t('alarm.name'), dataIndex: 'alarmName', width: 180, ellipsis: true },
      {
        key: 'severity',
        title: t('alarm.severity'),
        dataIndex: 'severity',
        width: 80,
        render: (_val, record) => (
          <Tag color={SEVERITY_TAG_COLOR[record.severity]}>{SEVERITY_LABEL[record.severity]}</Tag>
        ),
      },
      { key: 'neType', title: t('alarm.neType'), dataIndex: 'neType', width: 90 },
      { key: 'vendor', title: t('device.vendor'), dataIndex: 'vendor', width: 90 },
      {
        key: 'possibleCauses',
        title: t('alarm.content'),
        dataIndex: 'possibleCauses',
        width: 240,
        ellipsis: true,
        render: (v) => (
          <Text type="secondary" title={String(v)} style={{ fontSize: 12 }}>
            {String(v).split('\n')[0]}...
          </Text>
        ),
      },
      {
        key: 'handlingSuggestions',
        title: t('table.description'),
        dataIndex: 'handlingSuggestions',
        width: 240,
        ellipsis: true,
        render: (v) => (
          <Text type="secondary" title={String(v)} style={{ fontSize: 12 }}>
            {String(v).split('\n')[0]}...
          </Text>
        ),
      },
      { key: 'updateTime', title: t('table.updateTime'), dataIndex: 'updateTime', width: 120 },
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
            icon={<EyeOutlined />}
            onClick={() => setDrawerEntry(record)}
          >
            {t('common.detail')}
          </Button>
        ),
      },
    ],
    [t, SEVERITY_LABEL]
  );

  return (
    <>
      <ListPageLayout
        title={t('nav.alarm.library')}
      >
        <FilterBar
          filterId="alarm-support-library"
          fields={FILTER_FIELDS}
          onSearch={handleSearch}
          onReset={handleReset}
          collapsedRows={1}
        />

        <DataTable<AlarmLibraryEntry>
          tableId="alarm-support-library-table"
          columns={columns}
          dataSource={filteredData}
          loading={false}
          rowKey="id"
          total={filteredData.length}
          pageSize={20}
          currentPage={currentPage}
          onPageChange={(p) => setCurrentPage(p)}
          defaultDensity="compact"
        />
      </ListPageLayout>

      {/* Detail Drawer */}
      <Drawer
        title={
          <Space>
            <Text style={{ fontFamily: 'monospace' }}>{drawerEntry?.alarmCode}</Text>
            <span>{drawerEntry?.alarmName}</span>
            {drawerEntry && (
              <Tag color={SEVERITY_TAG_COLOR[drawerEntry.severity]}>
                {SEVERITY_LABEL[drawerEntry.severity]}
              </Tag>
            )}
          </Space>
        }
        open={Boolean(drawerEntry)}
        onClose={() => setDrawerEntry(null)}
        width={520}
      >
        {drawerEntry && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 20 }}>
            <div>
              <Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 4 }}>
                {t('common.detail')}
              </Text>
              <table style={{ width: '100%', fontSize: 14 }}>
                <tbody>
                  {[
                    { label: t('alarm.code'), value: drawerEntry.alarmCode },
                    { label: t('alarm.name'), value: drawerEntry.alarmName },
                    { label: t('alarm.severity'), value: SEVERITY_LABEL[drawerEntry.severity] },
                    { label: t('alarm.neType'), value: drawerEntry.neType },
                    { label: t('device.vendor'), value: drawerEntry.vendor },
                    { label: t('table.updateTime'), value: drawerEntry.updateTime },
                  ].map(({ label, value }) => (
                    <tr key={label} style={{ borderBottom: '1px solid #f5f5f5' }}>
                      <td style={{ padding: '8px 0', color: '#8c8c8c', width: 100 }}>{label}</td>
                      <td style={{ padding: '8px 0', fontWeight: 500 }}>{value}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            <div>
              <Title level={5} style={{ fontSize: 14, marginBottom: 8, color: '#FA8C16' }}>
                {t('alarm.content')}
              </Title>
              <div
                style={{
                  background: '#fff7e6',
                  border: '1px solid #ffd591',
                  borderRadius: 6,
                  padding: '12px 16px',
                }}
              >
                {drawerEntry.possibleCauses.split('\n').map((line, idx) => (
                  <Paragraph key={idx} style={{ margin: '2px 0', fontSize: 13 }}>
                    {line}
                  </Paragraph>
                ))}
              </div>
            </div>

            <div>
              <Title level={5} style={{ fontSize: 14, marginBottom: 8, color: '#52C41A' }}>
                {t('table.description')}
              </Title>
              <div
                style={{
                  background: '#f6ffed',
                  border: '1px solid #b7eb8f',
                  borderRadius: 6,
                  padding: '12px 16px',
                }}
              >
                {drawerEntry.handlingSuggestions.split('\n').map((line, idx) => (
                  <Paragraph key={idx} style={{ margin: '2px 0', fontSize: 13 }}>
                    {line}
                  </Paragraph>
                ))}
              </div>
            </div>
          </div>
        )}
      </Drawer>
    </>
  );
}
