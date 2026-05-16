import { useState, useMemo, useCallback } from 'react';
import {
  Button,
  Space,
  Tag,
  Typography,
  Modal,
  message,
  Row,
  Col,
  Card,
  Statistic,
} from 'antd';
import {
  ExportOutlined,
  BarChartOutlined,
} from '@ant-design/icons';
import type { Dayjs } from 'dayjs';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useT } from '@/hooks/useT';

// 事件级别类型
type EventLevel = 'info' | 'warning' | 'error' | 'success';

interface EventLog {
  id: string;
  neCode: string;
  neIpAddress: string;
  eventName: string;
  eventReason: string;
  time: string;
  eventLevel: EventLevel;
}

// Mock 数据
const mockEventLogs: EventLog[] = [
  { id: '1', neCode: 'ENB00001', neIpAddress: '10.1.0.101', eventName: '配置变更', eventReason: '修改了小区参数配置', time: '2026-03-26 10:00:00', eventLevel: 'info' },
  { id: '2', neCode: 'GNB00002', neIpAddress: '10.1.1.102', eventName: '告警产生', eventReason: '温度过高告警', time: '2026-03-26 09:30:00', eventLevel: 'warning' },
  { id: '3', neCode: 'ENB00003', neIpAddress: '10.2.0.103', eventName: '连接断开', eventReason: '网络连接中断', time: '2026-03-26 09:00:00', eventLevel: 'error' },
  { id: '4', neCode: 'ENB00001', neIpAddress: '10.1.0.101', eventName: '软件升级', eventReason: '软件版本从V1.2.0升级到V1.3.0', time: '2026-03-25 16:00:00', eventLevel: 'success' },
  { id: '5', neCode: 'GNB00004', neIpAddress: '10.1.1.104', eventName: '告警清除', eventReason: '温度过高告警已清除', time: '2026-03-25 14:00:00', eventLevel: 'info' },
  { id: '6', neCode: 'ENB00005', neIpAddress: '10.2.0.105', eventName: '重启', eventReason: '设备重启完成', time: '2026-03-25 12:00:00', eventLevel: 'info' },
  { id: '7', neCode: 'GNB00002', neIpAddress: '10.1.1.102', eventName: '连接建立', eventReason: '设备与OMC建立连接', time: '2026-03-25 10:00:00', eventLevel: 'success' },
  { id: '8', neCode: 'ENB00006', neIpAddress: '10.2.0.106', eventName: '状态变化', eventReason: '设备状态从离线变为在线', time: '2026-03-25 08:00:00', eventLevel: 'info' },
];

export default function EventLog() {
  const t = useT();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [statisticVisible, setStatisticVisible] = useState(false);

  // 事件类型列表（依赖 t）
  const eventTypes = useMemo(() => [
    { label: t('log.event.configChange'), value: t('log.event.configChange') },
    { label: t('log.event.statusChange'), value: t('log.event.statusChange') },
    { label: t('log.event.alarmGenerated'), value: t('log.event.alarmGenerated') },
    { label: t('log.event.alarmCleared'), value: t('log.event.alarmCleared') },
    { label: t('log.event.connectionEstablished'), value: t('log.event.connectionEstablished') },
    { label: t('log.event.connectionLost'), value: t('log.event.connectionLost') },
    { label: t('log.event.softwareUpgrade'), value: t('log.event.softwareUpgrade') },
    { label: t('log.event.reboot'), value: t('log.event.reboot') },
  ], [t]);

  // 事件名称映射到 i18n 键（用于统计显示）
  const eventNameToI18nKey = useMemo(() => ({
    '配置变更': 'log.event.configChange',
    '状态变化': 'log.event.statusChange',
    '告警产生': 'log.event.alarmGenerated',
    '告警清除': 'log.event.alarmCleared',
    '连接建立': 'log.event.connectionEstablished',
    '连接断开': 'log.event.connectionLost',
    '软件升级': 'log.event.softwareUpgrade',
    '重启': 'log.event.reboot',
  } as const), []);

  // 获取事件名称的国际化文本
  const getEventNameLabel = useCallback((name: string) => {
    return eventNameToI18nKey[name as keyof typeof eventNameToI18nKey] ? t(eventNameToI18nKey[name as keyof typeof eventNameToI18nKey]) : name;
  }, [t, eventNameToI18nKey]);

  // 过滤数据
  const filteredData = useMemo(() => {
    return mockEventLogs.filter((row) => {
      if (filters.keyword && typeof filters.keyword === 'string') {
        const keyword = filters.keyword.toLowerCase();
        if (!row.neCode.toLowerCase().includes(keyword)) {
          return false;
        }
      }
      if (filters.eventName && typeof filters.eventName === 'string') {
        if (row.eventName !== filters.eventName) {
          return false;
        }
      }
      // 时间范围过滤
      if (filters.timeRange && Array.isArray(filters.timeRange)) {
        const [startTime, endTime] = filters.timeRange as [Dayjs, Dayjs];
        const rowTime = row.time;
        if (startTime && endTime) {
          const rowDate = new Date(rowTime);
          if (rowDate < startTime.toDate() || rowDate > endTime.toDate()) {
            return false;
          }
        }
      }
      return true;
    });
  }, [filters]);

  // 统计数据
  const statistics = useMemo(() => {
    const counts: Record<string, number> = {};
    filteredData.forEach((log) => {
      counts[log.eventName] = (counts[log.eventName] || 0) + 1;
    });
    return Object.entries(counts)
      .map(([name, count]) => ({ name, count }))
      .sort((a, b) => b.count - a.count);
  }, [filteredData]);

  // 筛选字段 - 增加时间搜索
  const filterFields: FilterField[] = useMemo(() => [
    { name: 'keyword', label: t('log.deviceCode'), type: 'input', placeholder: t('log.inputDeviceCode') },
    {
      name: 'eventName',
      label: t('log.event.eventType'),
      type: 'select',
      options: eventTypes,
    },
    { name: 'timeRange', label: t('log.timeRange'), type: 'date-range', span: 2 },
  ], [t, eventTypes]);

  // 导出
  const handleExport = () => {
    void message.success(t('log.event.exporting'));
  };

  // 统计
  const handleStatistic = () => {
    setStatisticVisible(true);
  };

  // 表格列 - 去掉级别列，去掉操作项
  const columns: DataTableColumn<EventLog>[] = useMemo(() => [
    {
      key: 'id',
      title: 'ID',
      dataIndex: 'id',
      width: 80,
      render: (val: unknown) => <Typography.Text style={{ fontFamily: 'monospace' }}>{val as string}</Typography.Text>,
    },
    {
      key: 'neCode',
      title: t('log.deviceCode'),
      dataIndex: 'neCode',
      width: 140,
      render: (val: unknown) => <Typography.Text style={{ fontFamily: 'monospace' }}>{val as string}</Typography.Text>,
    },
    {
      key: 'neIpAddress',
      title: t('log.exception.column.baseIp'),
      dataIndex: 'neIpAddress',
      width: 140,
      render: (val: unknown) => <Typography.Text style={{ fontFamily: 'monospace' }}>{val as string}</Typography.Text>,
    },
    {
      key: 'eventName',
      title: t('log.event.eventType'),
      dataIndex: 'eventName',
      width: 120,
      render: (val: unknown) => <Tag color="blue">{val as string}</Tag>,
    },
    {
      key: 'eventReason',
      title: t('log.event.column.reason'),
      dataIndex: 'eventReason',
      ellipsis: true,
    },
    {
      key: 'time',
      title: t('log.event.column.time'),
      dataIndex: 'time',
      width: 160,
    },
  ], [t]);

  return (
    <ListPageLayout
      title={t('log.eventLog')}
      extra={
        <Space>
          <Button icon={<BarChartOutlined />} onClick={handleStatistic}>
            {t('log.event.statistics')}
          </Button>
          <Button type="primary" icon={<ExportOutlined />} onClick={handleExport}>
            {t('log.event.export')}
          </Button>
        </Space>
      }
    >
      <FilterBar
        filterId="event-log-filter"
        fields={filterFields}
        onSearch={setFilters}
        onReset={() => setFilters({})}
      />

      <Card
        size="small"
        bordered
        style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}
        styles={{ body: { padding: 0, display: 'flex', flexDirection: 'column', flex: 1, overflow: 'hidden' } }}
      >
        <DataTable<EventLog>
          tableId="event-log-list"
          columns={columns}
          dataSource={filteredData}
          rowKey="id"
          total={filteredData.length}
          currentPage={1}
          pageSize={20}
          onPageChange={() => {}}
          scroll={{ x: 'max-content', y: 'calc(100vh - 400px)' }}
          showRowNumber
          rowNumberTitle={t('log.event.seq')}
        />
      </Card>

      {/* 统计弹窗 */}
      <Modal
        title={t('log.event.statModal.title')}
        open={statisticVisible}
        onCancel={() => setStatisticVisible(false)}
        footer={null}
        width={700}
      >
        <Row gutter={[16, 16]}>
          {statistics.map((stat) => (
            <Col span={6} key={stat.name}>
              <Card>
                <Statistic
                  title={getEventNameLabel(stat.name)}
                  value={stat.count}
                  valueStyle={{ color: '#1890ff' }}
                />
              </Card>
            </Col>
          ))}
        </Row>
        <div style={{ marginTop: 24 }}>
          <h4>{t('log.event.stat.eventDistribution')}</h4>
          <div style={{ background: '#f5f5f5', padding: 16, borderRadius: 4 }}>
            {statistics.map((stat) => (
              <div key={stat.name} style={{ marginBottom: 8, display: 'flex', alignItems: 'center' }}>
                <span style={{ width: 100 }}>{getEventNameLabel(stat.name)}:</span>
                <div style={{
                  flex: 1,
                  height: 20,
                  background: '#e0e0e0',
                  borderRadius: 4,
                  overflow: 'hidden',
                }}>
                  <div style={{
                    width: `${(stat.count / Math.max(...statistics.map(s => s.count))) * 100}%`,
                    height: '100%',
                    background: '#1890ff',
                  }} />
                </div>
                <span style={{ width: 40, textAlign: 'right' }}>{stat.count}</span>
              </div>
            ))}
          </div>
        </div>
      </Modal>
    </ListPageLayout>
  );
}
