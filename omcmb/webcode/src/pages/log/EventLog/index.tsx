import { useState, useMemo } from 'react';
import {
  Button,
  Space,
  Tag,
  Typography,
  Modal,
  message,
  Drawer,
  Row,
  Col,
  Card,
  Statistic,
} from 'antd';
import {
  ExportOutlined,
  BarChartOutlined,
  EyeOutlined,
} from '@ant-design/icons';
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

// 事件级别配置
const eventLevelConfig: Record<EventLevel, { text: string; color: string }> = {
  info: { text: '信息', color: 'blue' },
  warning: { text: '警告', color: 'orange' },
  error: { text: '错误', color: 'red' },
  success: { text: '成功', color: 'green' },
};

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

// 事件类型列表
const eventTypes = [
  { label: '配置变更', value: '配置变更' },
  { label: '状态变化', value: '状态变化' },
  { label: '告警产生', value: '告警产生' },
  { label: '告警清除', value: '告警清除' },
  { label: '连接建立', value: '连接建立' },
  { label: '连接断开', value: '连接断开' },
  { label: '软件升级', value: '软件升级' },
  { label: '重启', value: '重启' },
];

export default function EventLog() {
  const t = useT();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [statisticVisible, setStatisticVisible] = useState(false);
  const [detailVisible, setDetailVisible] = useState(false);
  const [selectedEvent, setSelectedEvent] = useState<EventLog | null>(null);

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

  // 筛选字段
  const filterFields: FilterField[] = useMemo(() => [
    { name: 'keyword', label: '设备标识', type: 'input', placeholder: '请输入设备唯一标识' },
    {
      name: 'eventName',
      label: '事件类型',
      type: 'select',
      options: eventTypes,
    },
  ], []);

  // 导出
  const handleExport = () => {
    void message.success('正在导出事件日志数据...');
  };

  // 查看详情
  const handleViewDetail = (record: EventLog) => {
    setSelectedEvent(record);
    setDetailVisible(true);
  };

  // 统计
  const handleStatistic = () => {
    setStatisticVisible(true);
  };

  // 表格列
  const columns: DataTableColumn<EventLog>[] = useMemo(() => [
    {
      key: 'id',
      title: 'ID',
      dataIndex: 'id',
      width: 60,
    },
    {
      key: 'neCode',
      title: '设备唯一标识',
      dataIndex: 'neCode',
      width: 200,
      render: (val: string) => <Typography.Text style={{ fontFamily: 'monospace' }}>{val}</Typography.Text>,
    },
    {
      key: 'neIpAddress',
      title: '基站IP',
      dataIndex: 'neIpAddress',
      width: 150,
      render: (val: string) => <Typography.Text style={{ fontFamily: 'monospace' }}>{val}</Typography.Text>,
    },
    {
      key: 'eventName',
      title: '事件类型',
      dataIndex: 'eventName',
      width: 120,
      render: (val: string) => <Tag color="blue">{val}</Tag>,
    },
    {
      key: 'eventReason',
      title: '原因',
      dataIndex: 'eventReason',
      ellipsis: true,
    },
    {
      key: 'eventLevel',
      title: '级别',
      dataIndex: 'eventLevel',
      width: 80,
      render: (val: EventLevel) => {
        const cfg = eventLevelConfig[val];
        return <Tag color={cfg.color}>{cfg.text}</Tag>;
      },
    },
    {
      key: 'time',
      title: '时间',
      dataIndex: 'time',
      width: 160,
    },
    {
      key: 'actions',
      title: '操作',
      width: 80,
      fixed: 'right',
      render: (_: unknown, record: EventLog) => (
        <Button
          type="link"
          size="small"
          icon={<EyeOutlined />}
          onClick={() => handleViewDetail(record)}
        >
          详情
        </Button>
      ),
    },
  ], []);

  return (
    <ListPageLayout
      title="事件日志"
      extra={
        <Space>
          <Button icon={<BarChartOutlined />} onClick={handleStatistic}>
            统计
          </Button>
          <Button icon={<ExportOutlined />} onClick={handleExport}>
            导出
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

      <DataTable<EventLog>
        tableId="event-log-list"
        columns={columns}
        dataSource={filteredData}
        rowKey="id"
        total={filteredData.length}
        currentPage={1}
        pageSize={20}
        onPageChange={() => {}}
        scroll={{ x: 1100 }}
      />

      {/* 统计弹窗 */}
      <Modal
        title="事件统计"
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
                  title={stat.name}
                  value={stat.count}
                  valueStyle={{ color: '#1890ff' }}
                />
              </Card>
            </Col>
          ))}
        </Row>
        <div style={{ marginTop: 24 }}>
          <h4>事件分布</h4>
          <div style={{ background: '#f5f5f5', padding: 16, borderRadius: 4 }}>
            {statistics.map((stat) => (
              <div key={stat.name} style={{ marginBottom: 8, display: 'flex', alignItems: 'center' }}>
                <span style={{ width: 100 }}>{stat.name}:</span>
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

      {/* 详情抽屉 */}
      <Drawer
        title="事件详情"
        placement="right"
        width={500}
        open={detailVisible}
        onClose={() => setDetailVisible(false)}
      >
        {selectedEvent && (
          <div>
            <p><strong>ID:</strong> {selectedEvent.id}</p>
            <p><strong>设备唯一标识:</strong> {selectedEvent.neCode}</p>
            <p><strong>基站IP:</strong> {selectedEvent.neIpAddress}</p>
            <p>
              <strong>事件类型:</strong>{' '}
              <Tag color="blue">{selectedEvent.eventName}</Tag>
            </p>
            <p>
              <strong>事件级别:</strong>{' '}
              <Tag color={eventLevelConfig[selectedEvent.eventLevel].color}>
                {eventLevelConfig[selectedEvent.eventLevel].text}
              </Tag>
            </p>
            <p><strong>原因:</strong> {selectedEvent.eventReason}</p>
            <p><strong>时间:</strong> {selectedEvent.time}</p>
          </div>
        )}
      </Drawer>
    </ListPageLayout>
  );
}
