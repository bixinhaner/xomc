import { Card, Tag, Timeline, Statistic, Row, Col, Badge } from 'antd';
import {
  ExclamationCircleOutlined,
  UserOutlined,
  ApiOutlined,
  DatabaseOutlined,
  CloudServerOutlined,
} from '@ant-design/icons';
import GaugeChart from '@/components/Charts/GaugeChart';
import StatusIndicator from '@/components/StatusIndicator';
import { useSystemInfo } from '@core/hooks/api/useSystem';
import { useT } from '@/hooks/useT';

const serviceStatuses = [
  { name: '设备接入服务', status: 'online' as const, uptime: '15天3小时', port: 8080 },
  { name: '告警处理服务', status: 'online' as const, uptime: '15天3小时', port: 8081 },
  { name: '性能采集服务', status: 'online' as const, uptime: '15天3小时', port: 8082 },
  { name: '文件管理服务', status: 'online' as const, uptime: '14天22小时', port: 8083 },
  { name: '任务调度服务', status: 'online' as const, uptime: '15天3小时', port: 8084 },
  { name: 'MML执行服务', status: 'warning' as const, uptime: '2天1小时', port: 8085 },
  { name: '数据库主节点', status: 'online' as const, uptime: '30天0小时', port: 5432 },
  { name: '数据库从节点', status: 'online' as const, uptime: '30天0小时', port: 5433 },
  { name: 'Redis缓存', status: 'online' as const, uptime: '30天0小时', port: 6379 },
  { name: 'Kafka消息队列', status: 'offline' as const, uptime: '—', port: 9092 },
];

const recentOperations = [
  { time: '5分钟前', user: 'admin', action: '创建升级计划 "批量eNB升级"', type: 'create' },
  { time: '12分钟前', user: 'operator1', action: '确认告警 ALM-001 (ENB00001)', type: 'confirm' },
  { time: '28分钟前', user: 'admin', action: '重置用户 operator2 的密码', type: 'update' },
  { time: '45分钟前', user: 'operator2', action: '导出性能数据报表', type: 'export' },
  { time: '1小时前', user: 'admin', action: '同步设备 GNB00001 配置', type: 'sync' },
  { time: '2小时前', user: 'system', action: '自动清理过期日志文件 (释放 12GB)', type: 'cleanup' },
];

const typeColors: Record<string, string> = {
  create: 'blue', confirm: 'green', update: 'cyan', export: 'purple', sync: 'orange', cleanup: 'default',
};

export default function SystemDashboard() {
  const t = useT();
  const { data: systemInfo } = useSystemInfo();

  const cpuUsage = (systemInfo as Record<string, unknown>)?.cpuUsage as number ?? 42;
  const memUsage = (systemInfo as Record<string, unknown>)?.memUsage as number ?? 67;
  const diskUsage = (systemInfo as Record<string, unknown>)?.diskUsage as number ?? 58;
  const activeSessions = (systemInfo as Record<string, unknown>)?.activeSessions as number ?? 8;
  const onlineDevices = (systemInfo as Record<string, unknown>)?.onlineDevices as number ?? 189;
  const totalDevices = (systemInfo as Record<string, unknown>)?.totalDevices as number ?? 215;
  // 活动告警统计：用真实数据；未就绪时回退 0，不再展示硬编码占位数字（避免刷新闪现假数。issue #370）
  const activeAlarms = (systemInfo as Record<string, unknown>)?.activeAlarms as number ?? 0;

  const offlineServices = serviceStatuses.filter((s) => s.status === 'offline').length;
  const warningServices = serviceStatuses.filter((s) => s.status === 'warning').length;

  return (
    <div style={{ padding: 16, display: 'flex', flexDirection: 'column', gap: 16 }}>
      <Row gutter={16}>
        <Col span={6}>
          <Card size="small">
            <Statistic title={t('status.online')} value={onlineDevices} suffix={`/ ${totalDevices}`} prefix={<CloudServerOutlined />} valueStyle={{ color: '#52c41a' }} />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic title={t('alarm.name')} value={activeAlarms} prefix={<ExclamationCircleOutlined />} valueStyle={{ color: activeAlarms > 10 ? '#ff4d4f' : '#faad14' }} />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic title={t('user.lastLogin')} value={activeSessions} prefix={<UserOutlined />} />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title={t('table.status')}
              value={serviceStatuses.length - offlineServices - warningServices}
              suffix={`/ ${serviceStatuses.length}`}
              prefix={<ApiOutlined />}
              valueStyle={{ color: offlineServices > 0 ? '#ff4d4f' : '#52c41a' }}
            />
          </Card>
        </Col>
      </Row>

      <Row gutter={16}>
        <Col span={8}>
          <Card title={t('system.dashboard.cpu')} size="small" style={{ textAlign: 'center' }}>
            <GaugeChart value={cpuUsage} max={100} unit="%" />
            <div style={{ marginTop: 8, color: cpuUsage > 80 ? '#ff4d4f' : cpuUsage > 60 ? '#faad14' : '#52c41a', fontWeight: 500 }}>
              {cpuUsage}%
            </div>
          </Card>
        </Col>
        <Col span={8}>
          <Card title={t('system.dashboard.memory')} size="small" style={{ textAlign: 'center' }}>
            <GaugeChart value={memUsage} max={100} unit="%" />
            <div style={{ marginTop: 8, color: memUsage > 85 ? '#ff4d4f' : memUsage > 70 ? '#faad14' : '#52c41a', fontWeight: 500 }}>
              {memUsage}%
            </div>
          </Card>
        </Col>
        <Col span={8}>
          <Card title={t('system.dashboard.disk')} size="small" style={{ textAlign: 'center' }}>
            <GaugeChart value={diskUsage} max={100} unit="%" />
            <div style={{ marginTop: 8, color: diskUsage > 85 ? '#ff4d4f' : diskUsage > 70 ? '#faad14' : '#52c41a', fontWeight: 500 }}>
              {diskUsage}%
            </div>
          </Card>
        </Col>
      </Row>

      <Row gutter={16}>
        <Col span={14}>
          <Card title={
            <span>
              {t('table.status')}
              {offlineServices > 0 && <Badge count={offlineServices} style={{ marginLeft: 8 }} />}
            </span>
          } size="small">
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: 8 }}>
              {serviceStatuses.map((svc) => (
                <div key={svc.name} style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '4px 0' }}>
                  <StatusIndicator status={svc.status} />
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ fontSize: 13, fontWeight: 500, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{svc.name}</div>
                    <div style={{ fontSize: 11, color: '#999' }}>:{svc.port} · {svc.uptime}</div>
                  </div>
                </div>
              ))}
            </div>
          </Card>
        </Col>
        <Col span={10}>
          <Card title={t('nav.system.operationLog')} size="small" style={{ height: '100%' }}>
            <Timeline
              items={recentOperations.map((op) => ({
                dot: <DatabaseOutlined style={{ fontSize: 12 }} />,
                children: (
                  <div style={{ fontSize: 12 }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 2 }}>
                      <Tag color={typeColors[op.type] ?? 'default'} style={{ fontSize: 11, margin: 0 }}>{op.user}</Tag>
                      <span style={{ color: '#999' }}>{op.time}</span>
                    </div>
                    <div style={{ color: 'var(--color-neutral-600)' }}>{op.action}</div>
                  </div>
                ),
              }))}
              style={{ fontSize: 12 }}
            />
          </Card>
        </Col>
      </Row>
    </div>
  );
}
